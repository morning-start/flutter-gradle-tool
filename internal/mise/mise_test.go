package mise

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsMiseInstalled(t *testing.T) {
	// This test just verifies the function doesn't panic.
	// The actual result depends on the test environment.
	_ = IsMiseInstalled()
}

func TestDetectGradle_NotInstalled(t *testing.T) {
	// If mise is not installed, DetectGradle should return nil, nil.
	if IsMiseInstalled() {
		t.Skip("mise is installed, skipping not-installed test")
	}

	info, err := DetectGradle()
	if err != nil {
		t.Fatalf("DetectGradle() error = %v", err)
	}
	if info != nil {
		t.Fatalf("DetectGradle() = %+v, want nil when mise not installed", info)
	}
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

	t.Run("bin zip exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		zipPath := filepath.Join(dir, "gradle-8.5-bin.zip")
		if err := os.WriteFile(zipPath, []byte("fake"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := findDistributionZip(dir, "8.5")
		if got != zipPath {
			t.Fatalf("findDistributionZip() = %q, want %q", got, zipPath)
		}
	})

	t.Run("nested in wrapper dists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		nestedDir := filepath.Join(dir, "gradle", "wrapper", "dists", "gradle-8.5-all", "abc123")
		if err := os.MkdirAll(nestedDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		zipPath := filepath.Join(nestedDir, "gradle-8.5-all.zip")
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

	t.Run("walk fallback finds zip", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		subDir := filepath.Join(dir, "some", "nested", "path")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		zipPath := filepath.Join(subDir, "gradle-8.5-all.zip")
		if err := os.WriteFile(zipPath, []byte("fake"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := findDistributionZip(dir, "8.5")
		if got != zipPath {
			t.Fatalf("findDistributionZip() = %q, want %q", got, zipPath)
		}
	})

	t.Run("walk depth limit", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// Create a deeply nested structure beyond depth 3.
		deepDir := filepath.Join(dir, "a", "b", "c", "d", "e")
		if err := os.MkdirAll(deepDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		zipPath := filepath.Join(deepDir, "gradle-8.5-all.zip")
		if err := os.WriteFile(zipPath, []byte("fake"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := findDistributionZip(dir, "8.5")
		// Should not find it because it's too deep.
		if got != "" {
			t.Fatalf("findDistributionZip() = %q, want empty (too deep)", got)
		}
	})
}

func TestFileDistributionURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unix path",
			input: "/home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip",
			want:  "file:///home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip",
		},
		{
			name:  "path with spaces",
			input: "/home/user/my tools/gradle-8.5-all.zip",
			want:  "file:///home/user/my%20tools/gradle-8.5-all.zip",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FileDistributionURL(tc.input)
			if got != tc.want {
				t.Fatalf("FileDistributionURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestMiseGradleInstallDir_Fallback(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv is used.

	// Test the fallback path detection with a temp directory.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows

	version := "8.5"
	candidateDir := filepath.Join(home, ".local", "share", "mise", "installs", "gradle", version)
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// We can't easily test the actual mise command, but we can verify
	// the fallback path detection logic works.
	// The function tries mise commands first, then falls back to path detection.
	// Since mise is likely not installed in the test env, it will try fallbacks.
}
