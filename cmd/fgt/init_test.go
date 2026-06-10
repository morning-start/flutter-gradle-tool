package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitRewritesWrapperAndBuildGradle(t *testing.T) {
	projectDir := t.TempDir()
	writeFlutterProjectFiles(t, projectDir)

	out := &bytes.Buffer{}
	cmd := newRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--project-dir", projectDir, "init", "--source", "aliyun"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v\noutput:\n%s", err, out.String())
	}

	wrapper := readTestFile(t, filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"))
	if !strings.Contains(wrapper, "mirrors.aliyun.com/maven/gradle/gradle-8.5-all.zip") {
		t.Fatalf("wrapper not rewritten:\n%s", wrapper)
	}

	buildGradle := readTestFile(t, filepath.Join(projectDir, "android", "build.gradle"))
	if strings.Count(buildGradle, "Added by fgt") != 2 {
		t.Fatalf("build.gradle marker count = %d, want 2\n%s", strings.Count(buildGradle, "Added by fgt"), buildGradle)
	}
	if !strings.Contains(buildGradle, "maven { url 'https://maven.aliyun.com/repository/public' }") {
		t.Fatalf("build.gradle missing maven mirror block:\n%s", buildGradle)
	}

	config := readTestFile(t, filepath.Join(projectDir, ".fgt-config"))
	if !strings.Contains(config, `"source":"aliyun"`) {
		t.Fatalf("config not saved:\n%s", config)
	}
}

func TestInitOfficialRemovesMavenBlocks(t *testing.T) {
	projectDir := t.TempDir()
	writeFlutterProjectFiles(t, projectDir)

	cmd := newRootCommand()
	cmd.SetArgs([]string{"--project-dir", projectDir, "init", "--source", "aliyun"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first init returned error: %v", err)
	}

	cmd = newRootCommand()
	cmd.SetArgs([]string{"--project-dir", projectDir, "init", "--source", "official"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("second init returned error: %v", err)
	}

	buildGradle := readTestFile(t, filepath.Join(projectDir, "android", "build.gradle"))
	if strings.Contains(buildGradle, "Added by fgt") {
		t.Fatalf("build.gradle still contains mirror marker:\n%s", buildGradle)
	}

	wrapper := readTestFile(t, filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"))
	if !strings.Contains(wrapper, "services.gradle.org/distributions/gradle-8.5-all.zip") {
		t.Fatalf("wrapper not restored to official:\n%s", wrapper)
	}
}

func writeFlutterProjectFiles(t *testing.T, projectDir string) {
	t.Helper()

	wrapperDir := filepath.Join(projectDir, "android", "gradle", "wrapper")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	wrapperContent := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\n"
	if err := os.WriteFile(filepath.Join(wrapperDir, "gradle-wrapper.properties"), []byte(wrapperContent), 0o644); err != nil {
		t.Fatalf("WriteFile() wrapper error = %v", err)
	}

	buildGradleContent := `
buildscript {
    repositories {
        google()
        mavenCentral()
    }
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "android", "build.gradle"), []byte(buildGradleContent), 0o644); err != nil {
		t.Fatalf("WriteFile() build.gradle error = %v", err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
