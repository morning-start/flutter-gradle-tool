package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAndFormat(t *testing.T) {
	projectDir := t.TempDir()
	writeDoctorProjectFiles(t, projectDir, "aliyun")

	report, err := Check(projectDir)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("Check() issues = %v, want none", report.Issues)
	}

	output := Format(report)
	if !strings.Contains(output, "status: ok") {
		t.Fatalf("Format() output missing status ok:\n%s", output)
	}
	if !strings.Contains(output, "wrapper: aliyun") {
		t.Fatalf("Format() output missing wrapper source:\n%s", output)
	}
}

func TestCheckReportsMismatch(t *testing.T) {
	projectDir := t.TempDir()
	writeDoctorProjectFiles(t, projectDir, "official")

	// Simulate config drift.
	if err := os.WriteFile(filepath.Join(projectDir, ".fgt-config"), []byte(`{"source":"aliyun"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, err := Check(projectDir)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(report.Issues) == 0 {
		t.Fatalf("Check() issues = none, want mismatch")
	}
}

func writeDoctorProjectFiles(t *testing.T, projectDir, source string) {
	t.Helper()

	wrapperDir := filepath.Join(projectDir, "android", "gradle", "wrapper")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	wrapperURL := "https://services.gradle.org/distributions/gradle-8.5-all.zip"
	if source == "aliyun" {
		wrapperURL = "https://mirrors.aliyun.com/maven/gradle/gradle-8.5-all.zip"
	}

	if err := os.WriteFile(filepath.Join(wrapperDir, "gradle-wrapper.properties"), []byte("distributionUrl="+strings.ReplaceAll(wrapperURL, "://", `\://`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() wrapper error = %v", err)
	}

	buildGradleContent := `
buildscript {
    repositories {
        // Added by fgt
        maven { url 'https://maven.aliyun.com/repository/public' }
        google()
        mavenCentral()
    }
}
`
	if source == "official" {
		buildGradleContent = `
buildscript {
    repositories {
        google()
        mavenCentral()
    }
}
`
	}
	if err := os.WriteFile(filepath.Join(projectDir, "android", "build.gradle"), []byte(buildGradleContent), 0o644); err != nil {
		t.Fatalf("WriteFile() build.gradle error = %v", err)
	}

	if source == "aliyun" {
		if err := os.WriteFile(filepath.Join(projectDir, ".fgt-config"), []byte(`{"source":"aliyun"}`), 0o644); err != nil {
			t.Fatalf("WriteFile() config error = %v", err)
		}
	}
}
