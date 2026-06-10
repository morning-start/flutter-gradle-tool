package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apperrors "flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/gradle"
	"flutter-gradle-tool/internal/mirror"
)

type Report struct {
	ProjectDir         string
	WrapperPath        string
	WrapperSource      string
	BuildGradlePath    string
	BuildGradleMirrors bool
	ConfigPath         string
	ConfigSource       string
	Issues             []string
}

func Check(projectDir string) (Report, error) {
	if _, err := os.Stat(projectDir); err != nil {
		return Report{}, apperrors.Wrap(apperrors.ExitProjectNotFound, "project dir not found", err)
	}

	report := Report{ProjectDir: projectDir}

	if source, err := mirror.LoadConfig(projectDir); err != nil {
		return Report{}, err
	} else {
		report.ConfigPath = mirror.ConfigPath(projectDir)
		report.ConfigSource = source
	}

	wrapperPath, wrapperContent, err := loadFirstExisting(
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	)
	if err == nil {
		report.WrapperPath = wrapperPath
		version, distType, parseErr := gradle.ParseWrapperDistributionURL(wrapperContent)
		if parseErr != nil {
			report.Issues = append(report.Issues, "wrapper distributionUrl is invalid")
		} else if source := mirror.SourceFromDistributionURL(extractURL(wrapperContent)); source != nil {
			report.WrapperSource = source.Name
			_ = version
			_ = distType
		} else {
			report.Issues = append(report.Issues, "wrapper mirror source is unknown")
		}
	} else {
		report.Issues = append(report.Issues, "gradle wrapper file is missing")
	}

	buildGradlePath, buildGradleContent, err := loadFirstExisting(
		filepath.Join(projectDir, "android", "build.gradle"),
		filepath.Join(projectDir, "build.gradle"),
	)
	if err == nil {
		report.BuildGradlePath = buildGradlePath
		report.BuildGradleMirrors = gradle.BuildGradleHasMirror(buildGradleContent)
	} else {
		report.Issues = append(report.Issues, "build.gradle file is missing")
	}

	if report.ConfigSource != "" && report.WrapperSource != "" && report.ConfigSource != report.WrapperSource {
		report.Issues = append(report.Issues, fmt.Sprintf("config source %s does not match wrapper source %s", report.ConfigSource, report.WrapperSource))
	}

	if report.ConfigSource == "official" && report.BuildGradleMirrors {
		report.Issues = append(report.Issues, "official source should not keep Maven mirror blocks")
	}

	if report.ConfigSource != "" && report.ConfigSource != "official" && !report.BuildGradleMirrors {
		report.Issues = append(report.Issues, "Maven mirror blocks are missing")
	}

	return report, nil
}

func Format(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "project: %s\n", report.ProjectDir)
	if report.ConfigSource == "" {
		b.WriteString("config: not set\n")
	} else {
		fmt.Fprintf(&b, "config: %s\n", report.ConfigSource)
	}
	if report.WrapperSource == "" {
		b.WriteString("wrapper: unknown\n")
	} else {
		fmt.Fprintf(&b, "wrapper: %s\n", report.WrapperSource)
	}
	if report.BuildGradlePath == "" {
		b.WriteString("build.gradle: missing\n")
	} else if report.BuildGradleMirrors {
		b.WriteString("build.gradle: mirrors present\n")
	} else {
		b.WriteString("build.gradle: mirrors absent\n")
	}
	if len(report.Issues) == 0 {
		b.WriteString("status: ok\n")
	} else {
		b.WriteString("status: issues\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(&b, "- %s\n", issue)
		}
	}
	return b.String()
}

func loadFirstExisting(paths ...string) (string, string, error) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return path, string(data), nil
		}
	}
	return "", "", apperrors.New(apperrors.ExitProjectNotFound, "file not found")
}

func extractURL(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "distributionUrl=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "distributionUrl="))
		}
	}
	return ""
}
