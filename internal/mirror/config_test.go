package mirror_test

import (
	"os"
	"path/filepath"
	"testing"

	"flutter-gradle-tool/internal/mirror"
)

func TestSaveAndLoadConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := mirror.SaveConfig(projectDir, "aliyun"); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	got, err := mirror.LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got != "aliyun" {
		t.Fatalf("LoadConfig() = %q, want %q", got, "aliyun")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	got, err := mirror.LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got != "" {
		t.Fatalf("LoadConfig() = %q, want empty string", got)
	}
}

func TestReverseInferSourceFromWrapper(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wrapperDir := filepath.Join(projectDir, "android", "gradle", "wrapper")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	content := "distributionUrl=https\\://mirrors.aliyun.com/maven/gradle/gradle-8.5-all.zip\n"
	if err := os.WriteFile(filepath.Join(wrapperDir, "gradle-wrapper.properties"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := mirror.ReverseInferSource(projectDir)
	if err != nil {
		t.Fatalf("ReverseInferSource() error = %v", err)
	}
	if got != "aliyun" {
		t.Fatalf("ReverseInferSource() = %q, want %q", got, "aliyun")
	}
}
