package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	apperrors "flutter-gradle-tool/internal/errors"
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

func TestMirrorSetInteractive(t *testing.T) {
	projectDir := t.TempDir()

	in := strings.NewReader("3\ny\n")
	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "mirror", "set", "--interactive"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive mirror set returned error: %v\noutput:\n%s", err, out.String())
	}

	configPath := filepath.Join(projectDir, ".fgt-config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", configPath, err)
	}
	if !strings.Contains(string(data), `"source":"aliyun"`) {
		t.Fatalf("interactive mirror set saved wrong config:\n%s", string(data))
	}
}

func TestDoctorReportsOk(t *testing.T) {
	projectDir := t.TempDir()
	writeFlutterProjectFiles(t, projectDir)
	if err := os.WriteFile(filepath.Join(projectDir, ".fgt-config"), []byte(`{"source":"official"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "doctor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}

	if !strings.Contains(out.String(), "status: ok") {
		t.Fatalf("doctor output missing ok status:\n%s", out.String())
	}
}

func TestCacheCleanAndExec(t *testing.T) {
	projectDir := t.TempDir()
	writeFlutterProjectFiles(t, projectDir)
	if err := writeWrapperScript(projectDir); err != nil {
		t.Fatalf("writeWrapperScript() error = %v", err)
	}

	gradleHome := t.TempDir()
	t.Setenv("GRADLE_USER_HOME", gradleHome)
	if err := os.MkdirAll(filepath.Join(gradleHome, "caches"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(gradleHome, "wrapper", "dists"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "cache", "clean", "--all"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cache clean returned error: %v\noutput:\n%s", err, out.String())
	}
	if _, err := os.Stat(filepath.Join(gradleHome, "caches")); !os.IsNotExist(err) {
		t.Fatalf("caches directory still exists: %v", err)
	}

	out.Reset()
	cmd = newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "exec", "build"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("exec returned error: %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "wrapper:build") {
		t.Fatalf("exec output missing wrapper marker:\n%s", out.String())
	}
}

func TestInvalidProjectDirReturnsExitCode(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", filepath.Join(t.TempDir(), "missing"), "doctor"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing project dir")
	}

	if got := exitCode(err); got != apperrors.ExitProjectNotFound {
		t.Fatalf("exitCode() = %d, want %d", got, apperrors.ExitProjectNotFound)
	}
}

func writeWrapperScript(projectDir string) error {
	androidDir := filepath.Join(projectDir, "android")
	if err := os.MkdirAll(androidDir, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return os.WriteFile(filepath.Join(androidDir, "gradlew.bat"), []byte("@echo off\r\necho wrapper:%*\r\n"), 0o755)
	}

	return os.WriteFile(filepath.Join(androidDir, "gradlew"), []byte("#!/bin/sh\necho wrapper:$*\n"), 0o755)
}
