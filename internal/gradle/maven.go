package gradle

import (
	"regexp"
	"strings"

	apperrors "flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/mirror"
)

const mavenMarker = "// Added by fgt"

var repositoriesLinePattern = regexp.MustCompile(`^(\s*)repositories\s*\{\s*$`)
var markerLinePattern = regexp.MustCompile(`^\s*// Added by fgt\s*$`)
var mavenLinePattern = regexp.MustCompile(`^\s*maven\s*\{\s*url\s+['"][^'"]+['"]\s*\}\s*$`)

func RewriteBuildGradle(content string, source mirror.Source) (string, bool, error) {
	lines, lineEnding := splitLines(content)
	strippedLines, stripped := stripMavenBlocks(lines)

	if source.MavenURL == "" {
		updated := joinLines(strippedLines, lineEnding)
		return updated, stripped, nil
	}

	injectedLines := injectMavenBlocks(strippedLines, source.MavenURL)
	updated := joinLines(injectedLines, lineEnding)
	return updated, updated != content, nil
}

func stripMavenBlocks(lines []string) ([]string, bool) {
	var out []string
	changed := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !markerLinePattern.MatchString(strings.TrimRight(line, "\r")) {
			out = append(out, line)
			continue
		}

		changed = true
		if i+1 < len(lines) && mavenLinePattern.MatchString(strings.TrimRight(lines[i+1], "\r")) {
			i++
		}
	}

	return out, changed
}

func injectMavenBlocks(lines []string, mavenURL string) []string {
	var out []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		out = append(out, line)

		matches := repositoriesLinePattern.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if len(matches) != 2 {
			continue
		}

		indent := matches[1]
		out = append(out, indent+"    "+mavenMarker)
		out = append(out, indent+"    maven { url '"+mavenURL+"' }")
	}

	return out
}

func splitLines(content string) ([]string, string) {
	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}
	return strings.Split(normalizeWrapperContent(content), "\n"), lineEnding
}

func joinLines(lines []string, lineEnding string) string {
	return strings.Join(lines, lineEnding)
}

func BuildGradleHasMirror(content string) bool {
	return strings.Contains(content, mavenMarker)
}

func EnsureSupportedDSL(content string) error {
	if strings.Contains(content, "build.gradle.kts") {
		return apperrors.New(apperrors.ExitUnknownCommand, "kotlin dsl is not supported yet")
	}
	return nil
}
