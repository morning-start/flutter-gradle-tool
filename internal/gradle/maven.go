package gradle

import (
	"fmt"
	"regexp"
	"strings"

	"flutter-gradle-tool/internal/mirror"
)

const mavenMarker = "// Added by fgt"

var repositoriesLinePattern = regexp.MustCompile(`^(\s*)repositories\s*\{\s*$`)
var markerLinePattern = regexp.MustCompile(`^\s*// Added by fgt\s*$`)
var mavenLinePattern = regexp.MustCompile(`^\s*maven\s*\{\s*url\s+['"][^'"]+['"]\s*\}\s*$`)
var mavenKTSLinePattern = regexp.MustCompile(`^\s*maven\s*\(\s*url\s*=\s*uri\(\s*["'][^"']+["']\s*\)\s*\)\s*$`)

func RewriteBuildGradle(content string, source mirror.Source) (string, bool, error) {
	return rewriteBuildGradle(content, source, false)
}

func RewriteBuildGradleKTS(content string, source mirror.Source) (string, bool, error) {
	return rewriteBuildGradle(content, source, true)
}

func rewriteBuildGradle(content string, source mirror.Source, kotlin bool) (string, bool, error) {
	lines, lineEnding := splitLines(content)
	strippedLines, stripped := stripMavenBlocks(lines, kotlin)

	if source.MavenURL == "" {
		updated := joinLines(strippedLines, lineEnding)
		return updated, stripped, nil
	}

	injectedLines := injectMavenBlocks(strippedLines, source.MavenURL, kotlin)
	updated := joinLines(injectedLines, lineEnding)
	return updated, updated != content, nil
}

func stripMavenBlocks(lines []string, kotlin bool) ([]string, bool) {
	var out []string
	changed := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !markerLinePattern.MatchString(strings.TrimRight(line, "\r")) {
			out = append(out, line)
			continue
		}

		changed = true
		if i+1 < len(lines) && mavenLinePatternFor(kotlin).MatchString(strings.TrimRight(lines[i+1], "\r")) {
			i++
		}
	}

	return out, changed
}

func injectMavenBlocks(lines []string, mavenURL string, kotlin bool) []string {
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
		out = append(out, indent+"    "+mavenSnippet(mavenURL, kotlin))
	}

	return out
}

func mavenLinePatternFor(kotlin bool) *regexp.Regexp {
	if kotlin {
		return mavenKTSLinePattern
	}
	return mavenLinePattern
}

func mavenSnippet(mavenURL string, kotlin bool) string {
	if kotlin {
		return fmt.Sprintf(`maven(url = uri("%s"))`, mavenURL)
	}
	return fmt.Sprintf("maven { url '%s' }", mavenURL)
}

func splitLines(content string) ([]string, string) {
	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}
	return strings.Split(normalizeContent(content), "\n"), lineEnding
}

func joinLines(lines []string, lineEnding string) string {
	return strings.Join(lines, lineEnding)
}

func BuildGradleHasMirror(content string) bool {
	return strings.Contains(content, mavenMarker)
}
