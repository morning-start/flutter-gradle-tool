// Package mise provides integration with mise (formerly rtx) for managing
// local Gradle distributions. When mise is installed and manages a Gradle
// version, fgt can configure gradle-wrapper.properties to use a local
// distribution via file:// protocol, eliminating network downloads.
//
// mise installs Gradle as an extracted directory (not a zip). fgt creates
// a zip from the extracted directory and sets distributionUrl to a file://
// URL pointing to that zip. The zip is cached alongside the mise install
// directory and reused on subsequent runs.
package mise

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GradleInfo holds information about a mise-managed Gradle installation.
type GradleInfo struct {
	// Version is the resolved Gradle version (e.g., "8.14.5").
	Version string
	// InstallDir is the root directory where mise installed this Gradle version.
	InstallDir string
	// GradleHome is the extracted Gradle distribution directory (contains bin/, lib/).
	GradleHome string
	// ZipPath is the path to a local distribution zip file, if one already exists.
	ZipPath string
	// Available indicates whether a usable local installation was found.
	Available bool
	// Source indicates where the version was determined from: "project" or "global".
	Source string
}

// IsMiseInstalled checks if mise is available on the system PATH.
func IsMiseInstalled() bool {
	_, err := exec.LookPath("mise")
	return err == nil
}

// DetectGradle checks if mise manages a Gradle installation and returns
// information about it. It returns nil if mise is not installed or does
// not manage Gradle. The projectDir is used to check for project-level
// mise.toml configuration (searching up to 3 parent directories).
func DetectGradle(projectDir string) (*GradleInfo, error) {
	if !IsMiseInstalled() {
		return nil, nil
	}

	version, source, err := detectGradleVersion(projectDir)
	if err != nil || version == "" {
		return nil, nil
	}

	info := &GradleInfo{
		Version: version,
		Source:  source,
	}

	installDir, err := findGradleInstallDir(version)
	if err == nil && installDir != "" {
		info.InstallDir = installDir
		info.GradleHome = findGradleHome(installDir, version)
		info.ZipPath = findDistributionZip(installDir, version)
		info.Available = info.GradleHome != "" || info.ZipPath != ""
	}

	return info, nil
}

// detectGradleVersion determines the Gradle version managed by mise,
// checking project-level config first, then falling back to `mise current`.
func detectGradleVersion(projectDir string) (version, source string, err error) {
	// 1. Check project-level mise.toml / .mise.toml / .tool-versions.
	if projectDir != "" {
		dir := projectDir
		for i := 0; i < 3; i++ {
			if v, ok := readMiseTomlVersion(dir, "gradle"); ok && v != "" && v != "missing" {
				if isSpecificVersion(v) {
					return v, "project", nil
				}
				// "latest", bare major, or range — fall through to mise current.
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 2. Use `mise current gradle` for resolved version.
	cmd := exec.Command("mise", "current", "gradle")
	output, cmdErr := cmd.Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("mise current gradle: %w", cmdErr)
	}

	v := strings.TrimSpace(string(output))
	if v == "" || v == "missing" {
		return "", "", nil
	}
	return v, "global", nil
}

// isSpecificVersion returns true if v is a concrete version like "8.14.5"
// (contains at least one dot), as opposed to "latest", bare "8", or ranges.
func isSpecificVersion(v string) bool {
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "latest", "lts", "stable", "missing":
		return false
	}
	if strings.ContainsAny(v, "^~>=<*!") {
		return false
	}
	if v[0] < '0' || v[0] > '9' {
		return false
	}
	return strings.Contains(v, ".")
}

// readMiseTomlVersion reads a tool version from mise.toml, .mise.toml,
// or .tool-versions in the given directory.
func readMiseTomlVersion(dir, tool string) (string, bool) {
	for _, name := range []string{"mise.toml", ".mise.toml", ".tool-versions"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if v := parseMiseTomlToolVersion(string(data), tool); v != "" {
			return v, true
		}
	}
	return "", false
}

// parseMiseTomlToolVersion extracts a tool version from TOML or
// .tool-versions content. Supports:
//   - [tools]\ngradle = "8.5"
//   - [tools]\ngradle = { version = "8.5" }
//   - gradle 8.5  (.tool-versions format)
func parseMiseTomlToolVersion(content, tool string) string {
	lines := strings.Split(content, "\n")
	inTools := false
	hasTomlSection := strings.Contains(content, "[")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Track TOML sections.
		if strings.HasPrefix(trimmed, "[") {
			inTools = trimmed == "[tools]"
			continue
		}

		if !hasTomlSection {
			// .tool-versions: "tool version"
			if parts := strings.Fields(trimmed); len(parts) >= 2 && parts[0] == tool {
				return strings.Trim(parts[1], `"'`)
			}
			continue
		}

		if !inTools {
			continue
		}

		// Simple: gradle = "8.5"
		key := tool + " ="
		if !strings.HasPrefix(trimmed, key) {
			key = tool + "="
			if !strings.HasPrefix(trimmed, key) {
				continue
			}
		}

		val := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[1])
		if !strings.Contains(val, "{") {
			return strings.Trim(val, `"'`)
		}

		// Complex: gradle = { version = "8.5" }
		if idx := strings.Index(val, "version"); idx >= 0 {
			rest := val[idx+len("version"):]
			rest = strings.TrimLeft(rest, `"' `)
			rest = strings.TrimPrefix(rest, `"version"`)
			rest = strings.TrimSpace(rest)
			rest = strings.TrimLeft(rest, "=")
			rest = strings.TrimSpace(rest)
			rest = strings.Trim(rest, `"'`)
			if end := strings.IndexAny(rest, `"',}`); end > 0 {
				return rest[:end]
			}
			return rest
		}
	}
	return ""
}

// findGradleInstallDir resolves the installation directory for a given
// Gradle version managed by mise. Tries `mise where` first, then checks
// common filesystem paths with prefix matching for partial versions.
func findGradleInstallDir(version string) (string, error) {
	cmd := exec.Command("mise", "where", "gradle", version)
	if output, err := cmd.Output(); err == nil {
		if dir := strings.TrimSpace(string(output)); dir != "" {
			return dir, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	paths := miseInstallPaths(home)
	if v := os.Getenv("LOCALAPPDATA"); runtime.GOOS == "windows" && v != "" {
		paths = append(paths, filepath.Join(v, "mise", "installs", "gradle", version))
	}
	if v := os.Getenv("PROGRAMDATA"); runtime.GOOS == "windows" && v != "" {
		paths = append(paths, filepath.Join(v, "mise", "installs", "gradle", version))
	}

	for _, dir := range paths {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}

	// Prefix match: "8.14" → "8.14.5".
	for _, dir := range paths {
		if best := bestVersionMatch(filepath.Dir(dir), version); best != "" {
			return best, nil
		}
	}
	return "", nil
}

func miseInstallPaths(home string) []string {
	return []string{
		filepath.Join(home, ".local", "share", "mise", "installs", "gradle"),
		filepath.Join(home, ".local", "share", "rtx", "installs", "gradle"),
		filepath.Join(home, ".asdf", "installs", "gradle"),
	}
}

// bestVersionMatch finds the longest directory name starting with prefix.
func bestVersionMatch(parent, prefix string) string {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		if e.Name() == prefix {
			return filepath.Join(parent, e.Name())
		}
		if best == "" || len(e.Name()) > len(filepath.Base(best)) {
			best = filepath.Join(parent, e.Name())
		}
	}
	return best
}

// findGradleHome looks for the extracted Gradle distribution (containing
// bin/gradle) within the mise install directory.
func findGradleHome(installDir, version string) string {
	home := filepath.Join(installDir, "gradle-"+version)
	if isGradleHome(home) {
		return home
	}
	for _, e := range mustReadDir(installDir) {
		if !e.IsDir() {
			continue
		}
		if c := filepath.Join(installDir, e.Name()); isGradleHome(c) {
			return c
		}
	}
	return ""
}

func isGradleHome(dir string) bool {
	name := "gradle"
	if runtime.GOOS == "windows" {
		name = "gradle.bat"
	}
	_, err := os.Stat(filepath.Join(dir, "bin", name))
	return err == nil
}

// findDistributionZip searches for an existing Gradle distribution zip
// in the install directory.
func findDistributionZip(installDir, version string) string {
	for _, suffix := range []string{"-all", "-bin", "-src"} {
		path := filepath.Join(installDir, "gradle-"+version+suffix+".zip")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func mustReadDir(dir string) []os.DirEntry {
	entries, _ := os.ReadDir(dir)
	return entries
}
