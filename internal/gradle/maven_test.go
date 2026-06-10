package gradle

import (
	"strings"
	"testing"

	"flutter-gradle-tool/internal/mirror"
)

const sampleBuildGradle = `
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

func TestRewriteBuildGradleInjectsMirror(t *testing.T) {
	t.Parallel()

	got, changed, err := RewriteBuildGradle(sampleBuildGradle, mirror.Source{
		Name:     "aliyun",
		MavenURL: "https://maven.aliyun.com/repository/public",
	})
	if err != nil {
		t.Fatalf("RewriteBuildGradle() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteBuildGradle() changed = false, want true")
	}

	if strings.Count(got, "Added by fgt") != 2 {
		t.Fatalf("RewriteBuildGradle() marker count = %d, want 2", strings.Count(got, "Added by fgt"))
	}
	if !strings.Contains(got, "maven { url 'https://maven.aliyun.com/repository/public' }") {
		t.Fatalf("RewriteBuildGradle() missing maven mirror block:\n%s", got)
	}
}

func TestRewriteBuildGradleIdempotent(t *testing.T) {
	t.Parallel()

	withMirror, _, err := RewriteBuildGradle(sampleBuildGradle, mirror.Source{
		Name:     "aliyun",
		MavenURL: "https://maven.aliyun.com/repository/public",
	})
	if err != nil {
		t.Fatalf("initial RewriteBuildGradle() error = %v", err)
	}

	got, changed, err := RewriteBuildGradle(withMirror, mirror.Source{
		Name:     "aliyun",
		MavenURL: "https://maven.aliyun.com/repository/public",
	})
	if err != nil {
		t.Fatalf("second RewriteBuildGradle() error = %v", err)
	}
	if changed {
		t.Fatalf("RewriteBuildGradle() changed = true, want false")
	}
	if got != withMirror {
		t.Fatalf("RewriteBuildGradle() not idempotent")
	}
}

func TestRewriteBuildGradleRemovesMirror(t *testing.T) {
	t.Parallel()

	withMirror, _, err := RewriteBuildGradle(sampleBuildGradle, mirror.Source{
		Name:     "aliyun",
		MavenURL: "https://maven.aliyun.com/repository/public",
	})
	if err != nil {
		t.Fatalf("initial RewriteBuildGradle() error = %v", err)
	}

	got, changed, err := RewriteBuildGradle(withMirror, mirror.Source{Name: "official"})
	if err != nil {
		t.Fatalf("RewriteBuildGradle() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteBuildGradle() changed = false, want true")
	}
	if strings.Contains(got, "Added by fgt") {
		t.Fatalf("RewriteBuildGradle() still contains marker:\n%s", got)
	}
}
