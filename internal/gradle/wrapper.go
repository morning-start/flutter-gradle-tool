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

func ParseWrapperDistributionURL(content string) (string, string, error) {
	matches := wrapperPattern.FindStringSubmatch(normalizeWrapperContent(content))
	if len(matches) != 4 {
		return "", "", apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}
	return matches[2], matches[3], nil
}

func RewriteWrapperProperties(content string, source mirror.Source) (string, bool, error) {
	normalized := normalizeWrapperContent(content)
	matches := wrapperPattern.FindStringSubmatch(normalized)
	if len(matches) != 4 {
		return "", false, apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}

	version := matches[2]
	distType := matches[3]
	targetURL := buildDistributionURL(source.GradleURL, version, distType)
	currentURL := normalizeDistributionURL(matches[1])
	if currentURL == targetURL {
		return content, false, nil
	}

	replacement := "distributionUrl=" + escapeDistributionURL(targetURL)
	updated := wrapperPattern.ReplaceAllString(normalized, replacement)
	return restoreOriginalLineEndings(content, updated), true, nil
}

func buildDistributionURL(baseURL, version, distType string) string {
	base := strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/gradle-%s-%s.zip", base, version, distType)
}

func normalizeWrapperContent(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func normalizeDistributionURL(url string) string {
	return strings.ReplaceAll(strings.TrimSpace(url), `\://`, "://")
}

func escapeDistributionURL(url string) string {
	return strings.ReplaceAll(url, "://", `\://`)
}

func restoreOriginalLineEndings(original, updated string) string {
	if strings.Contains(original, "\r\n") {
		return strings.ReplaceAll(updated, "\n", "\r\n")
	}
	return updated
}

// RewriteWrapperPropertiesToLocal rewrites the distributionUrl in
// gradle-wrapper.properties to point to a local file:// URL. This is
// used when mise manages a local Gradle distribution. The distType
// parameter should match the existing type (e.g., "all" or "bin").
func RewriteWrapperPropertiesToLocal(content, localZipPath, distType string) (string, bool, error) {
	normalized := normalizeWrapperContent(content)
	matches := wrapperPattern.FindStringSubmatch(normalized)
	if len(matches) != 4 {
		return "", false, apperrors.New(apperrors.ExitWrapperParse, "distributionUrl is invalid")
	}

	version := matches[2]
	// Build a file:// URL pointing to the local zip.
	targetURL := buildLocalDistributionURL(localZipPath, version, distType)
	currentURL := normalizeDistributionURL(matches[1])
	if currentURL == targetURL {
		return content, false, nil
	}

	replacement := "distributionUrl=" + escapeDistributionURL(targetURL)
	updated := wrapperPattern.ReplaceAllString(normalized, replacement)
	return restoreOriginalLineEndings(content, updated), true, nil
}

func buildLocalDistributionURL(zipPath, version, distType string) string {
	// Convert Windows backslashes to forward slashes for the URL.
	slashPath := strings.ReplaceAll(filepath.ToSlash(zipPath), `\`, "/")
	// Normalize: ensure exactly one leading slash after file://
	// On Unix, the path already starts with /, so file:///path works.
	// On Windows, the path might be C:/..., so we need file:///C:/...
	var url string
	if strings.HasPrefix(slashPath, "/") {
		url = "file://" + slashPath
	} else {
		url = "file:///" + slashPath
	}
	// If the zip path already contains the version and type, use it directly.
	if strings.Contains(filepath.Base(zipPath), "gradle-"+version+"-"+distType) {
		return url
	}
	// Otherwise append the expected filename.
	return fmt.Sprintf("%s/gradle-%s-%s.zip", strings.TrimRight(url, "/"), version, distType)
}
