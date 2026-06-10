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
)

func newInitCommand() *cobra.Command {
	var (
		sourceName  string
		wrapperOnly bool
		mavenOnly   bool
		ciMode      bool
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Gradle mirror settings for a Flutter project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ciMode && sourceName == "" {
				return errors.New(errors.ExitCIRequiresSource, "--ci requires --source")
			}

			source, err := resolveSource(cmd, sourceName, interactive, true)
			if err != nil {
				return err
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