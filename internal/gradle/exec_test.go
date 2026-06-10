package gradle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunGradle(t *testing.T) {
	projectDir := t.TempDir()
	androidDir := filepath.Join(projectDir, "android")
	if err := os.MkdirAll(androidDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if runtime.GOOS == "windows" {
		script := "@echo off\r\necho wrapper:%*\r\n"
		if err := os.WriteFile(filepath.Join(androidDir, "gradlew.bat"), []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	} else {
		script := "#!/bin/sh\necho wrapper:$*\n"
		if err := os.WriteFile(filepath.Join(androidDir, "gradlew"), []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	output, err := RunGradle(projectDir, []string{"build"})
	if err != nil {
		t.Fatalf("RunGradle() error = %v; output=%s", err, output)
	}
	if !strings.Contains(output, "wrapper:build") {
		t.Fatalf("RunGradle() output = %q, want wrapper:build", output)
	}
}
