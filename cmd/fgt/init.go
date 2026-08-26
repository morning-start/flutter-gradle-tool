package main

import (
	"archive/zip"
	"fmt"
	"io"
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
			if miseMode {
				return runMiseInit(cmd, projectDir, wrapperOnly, mavenOnly)
			}
			return runMirrorInit(cmd, projectDir, sourceName, interactive, wrapperOnly, mavenOnly)
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

// --- mirror mode ---

func runMirrorInit(cmd *cobra.Command, projectDir, sourceName string, interactive, wrapperOnly, mavenOnly bool) error {
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
}

// --- mise mode ---

func runMiseInit(cmd *cobra.Command, projectDir string, wrapperOnly, mavenOnly bool) error {
	out := cmd.ErrOrStderr()

	if !mise.IsMiseInstalled() {
		return errors.New(errors.ExitUnknownSource, "mise is not installed or not on PATH")
	}

	info, err := mise.DetectGradle(projectDir)
	if err != nil {
		return fmt.Errorf("detect mise gradle: %w", err)
	}
	if info == nil {
		return errors.New(errors.ExitUnknownSource, "mise does not manage a Gradle version (run 'mise use gradle@<version>' first)")
	}
	if !info.Available {
		return errors.New(errors.ExitWrapperParse, fmt.Sprintf(
			"mise manages Gradle %s but no usable installation found", info.Version))
	}

	_, _ = fmt.Fprintf(out, "detected mise-managed Gradle %s (source: %s)\n", info.Version, info.Source)

	if !wrapperOnly {
		if err := rewriteWrapperForMise(projectDir, info); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "gradle-wrapper.properties updated to use local mise distribution")
	}

	if !mavenOnly && !wrapperOnly {
		_, _ = fmt.Fprintln(out, "tip: use 'fgt init --source <mirror>' separately to also configure Maven mirrors")
	}
	return nil
}

// rewriteWrapperForMise sets distributionUrl to a file:// URL pointing to a
// local zip created from the mise Gradle home directory. The zip is cached
// and reused on subsequent runs.
func rewriteWrapperForMise(projectDir string, info *mise.GradleInfo) error {
	if info.GradleHome == "" {
		return errors.New(errors.ExitWrapperParse, "no extracted Gradle directory found from mise")
	}

	wrapperPath, err := findWrapperPath(projectDir)
	if err != nil {
		return err
	}

	wrapperContent, err := os.ReadFile(wrapperPath)
	if err != nil {
		return fmt.Errorf("read wrapper file: %w", err)
	}

	// Ensure a local zip exists.
	zipPath := info.ZipPath
	if zipPath == "" {
		_, distType, _ := gradle.ParseWrapperDistributionURL(string(wrapperContent))
		if distType == "" {
			distType = "all"
		}
		zipPath, err = ensureLocalZip(info.GradleHome, info.Version, distType)
		if err != nil {
			return err
		}
	}

	updated, changed, err := gradle.RewriteWrapperPropertiesToLocal(string(wrapperContent), zipPath)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(wrapperPath, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write wrapper file", err)
		}
	}
	return nil
}

// ensureLocalZip creates a zip from the extracted Gradle home if one
// doesn't already exist. Uses Go's native archive/zip for portability.
func ensureLocalZip(gradleHome, version, distType string) (string, error) {
	zipPath := filepath.Join(filepath.Dir(gradleHome), fmt.Sprintf("gradle-%s-%s.zip", version, distType))
	if _, err := os.Stat(zipPath); err == nil {
		return zipPath, nil
	}

	fmt.Fprintf(os.Stderr, "creating local gradle zip from %s ...\n", gradleHome)
	if err := createZipFromDir(gradleHome, zipPath); err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	return zipPath, nil
}

// createZipFromDir creates a zip archive from a directory. The directory
// itself becomes the top-level entry (e.g., "gradle-8.14.5/").
func createZipFromDir(sourceDir, zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	baseDir := filepath.Base(sourceDir)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(baseDir, relPath))

		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(writer, src)
		return err
	})
}

// --- shared helpers ---

func findWrapperPath(projectDir string) (string, error) {
	return firstExistingPath(
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	)
}

func rewriteWrapperFile(projectDir string, source mirror.Source) error {
	path, err := findWrapperPath(projectDir)
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
		return nil // build.gradle is optional
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read build.gradle: %w", err)
	}

	var updated string
	var changed bool
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
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("required file not found")
}
