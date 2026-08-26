package gradle

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	apperrors "flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/mirror"
)

var wrapperPattern = regexp.MustCompile(`(?m)^distributionUrl\s*=\s*(.+/gradle-(.+)-(all|bin|src)\.zip)\s*$`)

// ParseWrapperDistributionURL extracts the Gradle version and distribution
// type from gradle-wrapper.properties content.
func ParseWrapperDistributionURL(content string) (version, distType string, err error) {
	matches := wrapperPattern.FindStringSubmatch(normalizeContent(content))
	if len(matches) != 4 {
		return "", "", apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}
	return matches[2], matches[3], nil
}

// RewriteWrapperProperties replaces the distributionUrl with a mirror URL
// from the given source, preserving the version and distribution type.
func RewriteWrapperProperties(content string, source mirror.Source) (string, bool, error) {
	normalized := normalizeContent(content)
	matches := wrapperPattern.FindStringSubmatch(normalized)
	if len(matches) != 4 {
		return "", false, apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}

	targetURL := buildMirrorURL(source.GradleURL, matches[2], matches[3])
	if normalizeURL(matches[1]) == targetURL {
		return content, false, nil
	}

	updated := wrapperPattern.ReplaceAllString(normalized, "distributionUrl="+escapeURL(targetURL))
	return restoreLineEndings(content, updated), true, nil
}

// RewriteWrapperPropertiesToLocal replaces the distributionUrl with a
// file:// URL pointing to a local zip file.
func RewriteWrapperPropertiesToLocal(content, localZipPath string) (string, bool, error) {
	normalized := normalizeContent(content)
	matches := wrapperPattern.FindStringSubmatch(normalized)
	if len(matches) < 2 {
		return "", false, apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}

	targetURL := buildLocalURL(localZipPath)
	if normalizeURL(matches[1]) == targetURL {
		return content, false, nil
	}

	updated := wrapperPattern.ReplaceAllString(normalized, "distributionUrl="+escapeURL(targetURL))
	return restoreLineEndings(content, updated), true, nil
}

// --- internal helpers ---

func buildMirrorURL(baseURL, version, distType string) string {
	return fmt.Sprintf("%s/gradle-%s-%s.zip", strings.TrimRight(baseURL, "/"), version, distType)
}

func buildLocalURL(zipPath string) string {
	slash := strings.ReplaceAll(filepath.ToSlash(zipPath), `\`, "/")
	if strings.HasPrefix(slash, "/") {
		return "file://" + slash
	}
	return "file:///" + slash
}

func normalizeContent(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func normalizeURL(u string) string { return strings.ReplaceAll(strings.TrimSpace(u), `\://`, "://") }

func escapeURL(u string) string { return strings.ReplaceAll(u, "://", `\://`) }

func restoreLineEndings(orig, updated string) string {
	if strings.Contains(orig, "\r\n") {
		return strings.ReplaceAll(updated, "\n", "\r\n")
	}
	return updated
}
