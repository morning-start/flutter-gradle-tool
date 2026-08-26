package mise

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsMiseInstalled(t *testing.T) {
	// This test just verifies the function doesn't panic.
	_ = IsMiseInstalled()
}

func TestDetectGradle_NotInstalled(t *testing.T) {
	if IsMiseInstalled() {
		t.Skip("mise is installed, skipping not-installed test")
	}

	info, err := DetectGradle("")
	if err != nil {
		t.Fatalf("DetectGradle() error = %v", err)
	}
	if info != nil {
		t.Fatalf("DetectGradle() = %+v, want nil when mise not installed", info)
	}
}

func TestParseMiseTomlToolVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		tool    string
		want    string
	}{
		{
			name:    "simple TOML",
			content: "[tools]\ngradle = \"8.5\"\n",
			tool:    "gradle",
			want:    "8.5",
		},
		{
			name:    "TOML with latest",
			content: "[tools]\ngradle = \"latest\"\n",
			tool:    "gradle",
			want:    "latest",
		},
		{
			name:    "TOML with braces",
			content: "[tools]\ngradle = { version = \"8.5\" }\n",
			tool:    "gradle",
			want:    "8.5",
		},
		{
			name:    "tool-versions format",
			content: "gradle 8.5\nnode 20\n",
			tool:    "gradle",
			want:    "8.5",
		},
		{
			name:    "tool not found",
			content: "[tools]\nnode = \"20\"\n",
			tool:    "gradle",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			tool:    "gradle",
			want:    "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseMiseTomlToolVersion(tc.content, tc.tool)
			if got != tc.want {
				t.Fatalf("parseMiseTomlToolVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadMiseTomlVersion(t *testing.T) {
	t.Parallel()

	t.Run("mise.toml exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := "[tools]\ngradle = \"8.5\"\n"
		if err := os.WriteFile(filepath.Join(dir, "mise.toml"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		version, ok := readMiseTomlVersion(dir, "gradle")
		if !ok {
			t.Fatal("readMiseTomlVersion() ok = false, want true")
		}
		if version != "8.5" {
			t.Fatalf("readMiseTomlVersion() = %q, want %q", version, "8.5")
		}
	})

	t.Run(".mise.toml exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := "[tools]\ngradle = \"9.0\"\n"
		if err := os.WriteFile(filepath.Join(dir, ".mise.toml"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		version, ok := readMiseTomlVersion(dir, "gradle")
		if !ok {
			t.Fatal("readMiseTomlVersion() ok = false, want true")
		}
		if version != "9.0" {
			t.Fatalf("readMiseTomlVersion() = %q, want %q", version, "9.0")
		}
	})

	t.Run(".tool-versions exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := "gradle 8.5\nnode 20\n"
		if err := os.WriteFile(filepath.Join(dir, ".tool-versions"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		version, ok := readMiseTomlVersion(dir, "gradle")
		if !ok {
			t.Fatal("readMiseTomlVersion() ok = false, want true")
		}
		if version != "8.5" {
			t.Fatalf("readMiseTomlVersion() = %q, want %q", version, "8.5")
		}
	})

	t.Run("no config files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, ok := readMiseTomlVersion(dir, "gradle")
		if ok {
			t.Fatal("readMiseTomlVersion() ok = true, want false")
		}
	})
}

func TestFindDistributionZip(t *testing.T) {
	t.Parallel()

	t.Run("direct zip exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		zipPath := filepath.Join(dir, "gradle-8.5-all.zip")
		if err := os.WriteFile(zipPath, []byte("fake"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := findDistributionZip(dir, "8.5")
		if got != zipPath {
			t.Fatalf("findDistributionZip() = %q, want %q", got, zipPath)
		}
	})

	t.Run("no zip found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		got := findDistributionZip(dir, "8.5")
		if got != "" {
			t.Fatalf("findDistributionZip() = %q, want empty", got)
		}
	})
}

func TestFindGradleHome(t *testing.T) {
	t.Parallel()

	t.Run("standard layout", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		home := filepath.Join(dir, "gradle-8.5")
		binDir := filepath.Join(home, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		binName := "gradle"
		if runtime.GOOS == "windows" {
			binName = "gradle.bat"
		}
		if err := os.WriteFile(filepath.Join(binDir, binName), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := findGradleHome(dir, "8.5")
		if got != home {
			t.Fatalf("findGradleHome() = %q, want %q", got, home)
		}
	})

	t.Run("no gradle home", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		got := findGradleHome(dir, "8.5")
		if got != "" {
			t.Fatalf("findGradleHome() = %q, want empty", got)
		}
	})
}

func TestIsGradleHome(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		binDir := filepath.Join(dir, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		binName := "gradle"
		if runtime.GOOS == "windows" {
			binName = "gradle.bat"
		}
		if err := os.WriteFile(filepath.Join(binDir, binName), []byte(""), 0o755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		if !isGradleHome(dir) {
			t.Fatal("isGradleHome() = false, want true")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if isGradleHome(dir) {
			t.Fatal("isGradleHome() = true, want false")
		}
	})
}

func TestIsGradleHome_Invalid(t *testing.T) {
	t.Parallel()
	if isGradleHome(t.TempDir()) {
		t.Fatal("isGradleHome() = true, want false")
	}
}
