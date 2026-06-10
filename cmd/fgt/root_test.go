package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMirrorSetAndCurrent(t *testing.T) {
	projectDir := t.TempDir()

	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "mirror", "set", "--source", "aliyun"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mirror set returned error: %v", err)
	}

	configPath := filepath.Join(projectDir, ".fgt-config")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file missing: %v", err)
	}

	out.Reset()
	cmd = newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "mirror", "current"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mirror current returned error: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "aliyun" {
		t.Fatalf("mirror current output = %q, want %q", got, "aliyun")
	}
}

func TestMirrorListMarksCurrent(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".fgt-config"), []byte(`{"source":"tencent"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "mirror", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mirror list returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "*\ttencent") && !strings.Contains(output, "* tencent") {
		t.Fatalf("mirror list output missing current marker for tencent:\n%s", output)
	}
}
