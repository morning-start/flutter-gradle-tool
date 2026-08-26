package gradle

import (
	"strings"
	"testing"

	"flutter-gradle-tool/internal/mirror"
)

func TestRewriteWrapperDistributionURL(t *testing.T) {
	t.Parallel()

	input := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\n"

	got, changed, err := RewriteWrapperProperties(input, mirror.Source{
		Name:      "aliyun",
		GradleURL: "https://mirrors.aliyun.com/maven/gradle",
	})
	if err != nil {
		t.Fatalf("RewriteWrapperProperties() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteWrapperProperties() changed = false, want true")
	}

	want := "distributionUrl=https\\://mirrors.aliyun.com/maven/gradle/gradle-8.5-all.zip"
	if !strings.Contains(got, want) {
		t.Fatalf("RewriteWrapperProperties() = %q, want to contain %q", got, want)
	}
}

func TestRewriteWrapperPropertiesIdempotent(t *testing.T) {
	t.Parallel()

	input := "distributionUrl=https\\://mirrors.aliyun.com/maven/gradle/gradle-8.5-all.zip\n"

	got, changed, err := RewriteWrapperProperties(input, mirror.Source{
		Name:      "aliyun",
		GradleURL: "https://mirrors.aliyun.com/maven/gradle",
	})
	if err != nil {
		t.Fatalf("RewriteWrapperProperties() error = %v", err)
	}
	if changed {
		t.Fatalf("RewriteWrapperProperties() changed = true, want false")
	}
	if got != input {
		t.Fatalf("RewriteWrapperProperties() = %q, want %q", got, input)
	}
}

func TestParseWrapperDistributionURL(t *testing.T) {
	t.Parallel()

	version, distType, err := ParseWrapperDistributionURL("distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5.1-bin.zip\n")
	if err != nil {
		t.Fatalf("ParseWrapperDistributionURL() error = %v", err)
	}
	if version != "8.5.1" {
		t.Fatalf("version = %q, want %q", version, "8.5.1")
	}
	if distType != "bin" {
		t.Fatalf("distribution type = %q, want %q", distType, "bin")
	}
}

func TestParseWrapperDistributionURLError(t *testing.T) {
	t.Parallel()

	_, _, err := ParseWrapperDistributionURL("distributionUrl=https\\://example.com/gradle.zip\n")
	if err == nil {
		t.Fatalf("ParseWrapperDistributionURL() error = nil, want error")
	}
}

func TestRewriteWrapperPropertiesToLocal(t *testing.T) {
	t.Parallel()

	input := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\n"
	zipPath := "/home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip"

	got, changed, err := RewriteWrapperPropertiesToLocal(input, zipPath, "all")
	if err != nil {
		t.Fatalf("RewriteWrapperPropertiesToLocal() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteWrapperPropertiesToLocal() changed = false, want true")
	}

	want := "distributionUrl=file\\:///home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip"
	if !strings.Contains(got, want) {
		t.Fatalf("RewriteWrapperPropertiesToLocal() = %q, want to contain %q", got, want)
	}
}

func TestRewriteWrapperPropertiesToLocalIdempotent(t *testing.T) {
	t.Parallel()

	zipPath := "/home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip"
	input := "distributionUrl=file\\:///home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip\n"

	got, changed, err := RewriteWrapperPropertiesToLocal(input, zipPath, "all")
	if err != nil {
		t.Fatalf("RewriteWrapperPropertiesToLocal() error = %v", err)
	}
	if changed {
		t.Fatalf("RewriteWrapperPropertiesToLocal() changed = true, want false")
	}
	if got != input {
		t.Fatalf("RewriteWrapperPropertiesToLocal() = %q, want %q", got, input)
	}
}

func TestRewriteWrapperPropertiesToLocalBin(t *testing.T) {
	t.Parallel()

	input := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-bin.zip\n"
	zipPath := "/home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-bin.zip"

	got, changed, err := RewriteWrapperPropertiesToLocal(input, zipPath, "bin")
	if err != nil {
		t.Fatalf("RewriteWrapperPropertiesToLocal() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteWrapperPropertiesToLocal() changed = false, want true")
	}

	want := "distributionUrl=file\\:///home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-bin.zip"
	if !strings.Contains(got, want) {
		t.Fatalf("RewriteWrapperPropertiesToLocal() = %q, want to contain %q", got, want)
	}
}

func TestRewriteWrapperPropertiesToLocalCRLF(t *testing.T) {
	t.Parallel()

	input := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\r\n"
	zipPath := "/home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip"

	got, changed, err := RewriteWrapperPropertiesToLocal(input, zipPath, "all")
	if err != nil {
		t.Fatalf("RewriteWrapperPropertiesToLocal() error = %v", err)
	}
	if !changed {
		t.Fatalf("RewriteWrapperPropertiesToLocal() changed = false, want true")
	}

	// The output uses escaped :// as \:// in the properties file format.
	want := "distributionUrl=file\\:///home/user/.local/share/mise/installs/gradle/8.5/gradle-8.5-all.zip"
	if !strings.Contains(got, want) {
		t.Fatalf("RewriteWrapperPropertiesToLocal() = %q, want to contain %q", got, want)
	}
}
