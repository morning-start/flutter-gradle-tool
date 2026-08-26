package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/gradle"
	"flutter-gradle-tool/internal/mirror"
	"flutter-gradle-tool/internal/mise"
)

func newInitCommand() *cobra.Command {
	var (
		sourceName  string
		wrapperOnly bool
		mavenOnly   bool
		ciMode      bool
		interactive bool
		miseMode    bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Gradle mirror settings for a Flutter project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ciMode && sourceName == "" {
				return errors.New(errors.ExitCIRequiresSource, "--ci requires --source")
			}

			// --mise mode: use local mise-managed Gradle distribution.
			if miseMode {
				return runMiseInit(cmd, projectDir, wrapperOnly, mavenOnly)
			}

			source, err := resolveSource(cmd, sourceName, interactive, true)
			if err != nil {
				return err
			}

			if source.Name == "official" {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "warning: official source uses overseas servers, download speed may be slow")
			}

			if !wrapperOnly {
				if err := rewriteWrapperFile(projectDir, *source); err != nil {
					return err
				}
			}

			if !mavenOnly {
				if err := rewriteBuildGradleFile(projectDir, *source); err != nil {
					return err
				}
			}

			return mirror.SaveConfig(projectDir, source.Name)
		},
	}

	cmd.Flags().StringVarP(&sourceName, "source", "s", "", "Mirror source name")
	cmd.Flags().BoolVarP(&wrapperOnly, "wrapper-only", "w", false, "Only change wrapper mirror")
	cmd.Flags().BoolVarP(&mavenOnly, "maven-only", "m", false, "Only change Maven mirror")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Non-interactive mode")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive selection")
	cmd.Flags().BoolVar(&miseMode, "mise", false, "Use mise-managed local Gradle distribution")

	return cmd
}

func rewriteWrapperFile(projectDir string, source mirror.Source) error {
	path, err := firstExistingPath(
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read wrapper file: %w", err)
	}

	updated, changed, err := gradle.RewriteWrapperProperties(string(content), source)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write wrapper file", err)
		}
	}
	return nil
}

func rewriteBuildGradleFile(projectDir string, source mirror.Source) error {
	path, err := firstExistingPath(
		filepath.Join(projectDir, "android", "build.gradle.kts"),
		filepath.Join(projectDir, "build.gradle.kts"),
		filepath.Join(projectDir, "android", "build.gradle"),
		filepath.Join(projectDir, "build.gradle"),
	)
	if err != nil {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read build.gradle: %w", err)
	}

	var (
		updated string
		changed bool
	)
	if strings.HasSuffix(path, ".kts") {
		updated, changed, err = gradle.RewriteBuildGradleKTS(string(content), source)
	} else {
		updated, changed, err = gradle.RewriteBuildGradle(string(content), source)
	}
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write build.gradle", err)
		}
	}
	return nil
}

func firstExistingPath(paths ...string) (string, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("required file not found")
}

// runMiseInit configures the project to use a mise-managed local Gradle
// distribution. It detects mise, finds the local Gradle zip, and rewrites
// gradle-wrapper.properties to use a file:// URL.
func runMiseInit(cmd *cobra.Command, projectDir string, wrapperOnly, mavenOnly bool) error {
	out := cmd.ErrOrStderr()

	if !mise.IsMiseInstalled() {
		return errors.New(errors.ExitUnknownSource, "mise is not installed or not on PATH")
	}

	info, err := mise.DetectGradle()
	if err != nil {
		return fmt.Errorf("detect mise gradle: %w", err)
	}
	if info == nil {
		return errors.New(errors.ExitUnknownSource, "mise does not manage a Gradle version (run 'mise use gradle@<version>' first)")
	}
	if !info.Available {
		return errors.New(errors.ExitWrapperParse, fmt.Sprintf(
			"mise manages Gradle %s but no local distribution zip found in %s",
			info.Version, info.InstallDir))
	}

	_, _ = fmt.Fprintf(out, "detected mise-managed Gradle %s at %s\n", info.Version, info.ZipPath)

	if !wrapperOnly {
		if err := rewriteWrapperFileToLocal(projectDir, info.ZipPath); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "gradle-wrapper.properties updated to use local distribution")
	}

	// For mise mode, we don't touch Maven config by default since the
	// user's mirror settings are orthogonal to the local distribution.
	if !mavenOnly && !wrapperOnly {
		_, _ = fmt.Fprintln(out, "tip: use 'fgt init --source <mirror>' separately to also configure Maven mirrors")
	}

	return nil
}

// rewriteWrapperFileToLocal rewrites the distributionUrl to point to a
// local file:// URL for the given zip path.
func rewriteWrapperFileToLocal(projectDir, zipPath string) error {
	path, err := firstExistingPath(
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read wrapper file: %w", err)
	}

	// Detect existing dist type from the current distributionUrl.
	_, distType, parseErr := gradle.ParseWrapperDistributionURL(string(content))
	if parseErr != nil {
		// Default to "all" if we can't detect.
		distType = "all"
	}

	updated, changed, err := gradle.RewriteWrapperPropertiesToLocal(string(content), zipPath, distType)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write wrapper file", err)
		}
	}
	return nil
}