// Package mise provides integration with mise (formerly rtx) for managing
// local Gradle distributions. When mise is installed and manages a Gradle
// version, fgt can configure gradle-wrapper.properties to use the local
// distribution zip via file:// protocol, eliminating network downloads.
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
	// Version is the Gradle version managed by mise (e.g., "8.5").
	Version string
	// InstallDir is the root directory where mise installed this Gradle version.
	InstallDir string
	// ZipPath is the path to the local distribution zip file, if it exists.
	ZipPath string
	// Available indicates whether a usable local distribution was found.
	Available bool
}

// IsMiseInstalled checks if mise is available on the system PATH.
func IsMiseInstalled() bool {
	_, err := exec.LookPath("mise")
	return err == nil
}

// DetectGradle checks if mise manages a Gradle installation and returns
// information about it. It returns nil if mise is not installed or does
// not manage Gradle.
func DetectGradle() (*GradleInfo, error) {
	if !IsMiseInstalled() {
		return nil, nil
	}

	version, err := miseCurrentGradleVersion()
	if err != nil {
		// mise is installed but doesn't manage Gradle — not an error.
		return nil, nil
	}

	info := &GradleInfo{
		Version: version,
	}

	installDir, err := miseGradleInstallDir(version)
	if err == nil && installDir != "" {
		info.InstallDir = installDir
		info.ZipPath = findDistributionZip(installDir, version)
		info.Available = info.ZipPath != ""
	}

	return info, nil
}

// miseCurrentGradleVersion runs `mise current gradle` to get the active
// Gradle version managed by mise.
func miseCurrentGradleVersion() (string, error) {
	cmd := exec.Command("mise", "current", "gradle")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("mise current gradle: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" || version == "missing" {
		return "", nil
	}
	return version, nil
}

// miseGradleInstallDir resolves the installation directory for a given
// Gradle version managed by mise.
func miseGradleInstallDir(version string) (string, error) {
	// Try `mise where gradle <version>` first.
	cmd := exec.Command("mise", "where", "gradle", version)
	output, err := cmd.Output()
	if err == nil {
		dir := strings.TrimSpace(string(output))
		if dir != "" {
			return dir, nil
		}
	}

	// Fallback: check common mise installation paths.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(home, ".local", "share", "mise", "installs", "gradle", version),
		filepath.Join(home, ".local", "share", "rtx", "installs", "gradle", version),
		filepath.Join(home, ".asdf", "installs", "gradle", version),
	}

	// On Windows, also check common paths.
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			candidates = append(candidates,
				filepath.Join(localAppData, "mise", "installs", "gradle", version),
			)
		}
		programData := os.Getenv("PROGRAMDATA")
		if programData != "" {
			candidates = append(candidates,
				filepath.Join(programData, "mise", "installs", "gradle", version),
			)
		}
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}

	return "", nil
}

// findDistributionZip searches for a Gradle distribution zip file in the
// installation directory. The zip is typically located in a subdirectory
// named after the distribution type (e.g., gradle-8.5-all.zip).
func findDistributionZip(installDir, version string) string {
	// Common locations for the distribution zip within the install dir.
	patterns := []string{
		filepath.Join(installDir, "gradle-"+version+"-all.zip"),
		filepath.Join(installDir, "gradle-"+version+"-bin.zip"),
		filepath.Join(installDir, "gradle-"+version+"-src.zip"),
		filepath.Join(installDir, "gradle", "wrapper", "dists",
			"gradle-"+version+"-all", "*", "gradle-"+version+"-all.zip"),
		filepath.Join(installDir, "gradle", "wrapper", "dists",
			"gradle-"+version+"-bin", "*", "gradle-"+version+"-bin.zip"),
	}

	for _, pattern := range patterns {
		if strings.Contains(pattern, "*") {
			matches, _ := filepath.Glob(pattern)
			if len(matches) > 0 {
				return matches[0]
			}
		} else {
			if _, err := os.Stat(pattern); err == nil {
				return pattern
			}
		}
	}

	// Walk the install dir looking for a matching zip (depth-limited).
	var found string
	_ = filepath.Walk(installDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if found != "" {
			return filepath.SkipDir
		}
		name := info.Name()
		if !info.IsDir() && strings.HasPrefix(name, "gradle-"+version+"-") &&
			strings.HasSuffix(name, ".zip") {
			found = path
			return filepath.SkipDir
		}
		// Limit search depth to avoid traversing too deep.
		depth := strings.Count(strings.TrimPrefix(path, installDir), string(os.PathSeparator))
		if depth > 3 {
			return filepath.SkipDir
		}
		return nil
	})

	return found
}

// FileDistributionURL returns a file:// URL suitable for use as a
// gradle-wrapper.properties distributionUrl. The path is converted to
// forward slashes and properly escaped.
func FileDistributionURL(zipPath string) string {
	// Convert Windows backslashes to forward slashes for the URL.
	slashPath := strings.ReplaceAll(filepath.ToSlash(zipPath), `\`, "/")
	// Normalize: ensure exactly one leading slash after file://
	if strings.HasPrefix(slashPath, "/") {
		return "file://" + strings.ReplaceAll(slashPath, " ", "%20")
	}
	return "file:///" + strings.ReplaceAll(slashPath, " ", "%20")
}
