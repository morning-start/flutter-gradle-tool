package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/cache"
	"flutter-gradle-tool/internal/doctor"
	"flutter-gradle-tool/internal/gradle"
	"flutter-gradle-tool/internal/mirror"
)

var (
	version    = "dev"
	projectDir string
)

func execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "fgt",
		Short:         "Flutter Gradle Tool",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.PersistentFlags().StringVar(&projectDir, "project-dir", ".", "Path to Flutter project root")

	cmd.AddCommand(newMirrorCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newCacheCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newExecCommand())

	return cmd
}

func newPlaceholderCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s command placeholder", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s command is not implemented yet", name)
		},
	}
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the current Flutter Gradle setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := doctor.Check(projectDir)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), doctor.Format(report))
			return nil
		},
	}
}

func newCacheCommand() *cobra.Command {
	var cleanAll bool

	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clean Gradle cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := cache.Inspect()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", info.Root)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "size: %d\n", info.TotalSize)
			return nil
		},
	}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean Gradle cache directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cleanAll {
				return fmt.Errorf("use --all to remove cache directories")
			}
			removed, err := cache.CleanAll()
			if err != nil {
				return err
			}
			for _, target := range removed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "removed: %s\n", target)
			}
			return nil
		},
	}
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all known Gradle cache directories")
	cmd.AddCommand(cleanCmd)

	return cmd
}

func newExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "exec [gradle task...]",
		Short: "Run Gradle tasks through the project wrapper",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := gradle.RunGradle(projectDir, args)
			if output != "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			}
			return err
		},
	}
}

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
				return fmt.Errorf("--ci requires --source")
			}
			if interactive {
				return fmt.Errorf("interactive init is not implemented yet")
			}

			source, err := resolveInitSource(projectDir, sourceName)
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

	cmd.Flags().StringVar(&sourceName, "source", "", "Mirror source name")
	cmd.Flags().BoolVar(&wrapperOnly, "wrapper-only", false, "Only change wrapper mirror")
	cmd.Flags().BoolVar(&mavenOnly, "maven-only", false, "Only change Maven mirror")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Non-interactive mode")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive selection")

	return cmd
}

func resolveInitSource(projectDir, sourceName string) (*mirror.Source, error) {
	if sourceName != "" {
		source := mirror.FindByName(sourceName)
		if source == nil {
			return nil, fmt.Errorf("unknown mirror source: %s", sourceName)
		}
		return source, nil
	}

	if current, err := mirror.CurrentSource(projectDir); err != nil {
		return nil, err
	} else if current != "" {
		source := mirror.FindByName(current)
		if source != nil {
			return source, nil
		}
	}

	return nil, fmt.Errorf("--source is required")
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
			return fmt.Errorf("write wrapper file: %w", err)
		}
	}
	return nil
}

func rewriteBuildGradleFile(projectDir string, source mirror.Source) error {
	path, err := firstExistingPath(
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

	updated, changed, err := gradle.RewriteBuildGradle(string(content), source)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write build.gradle: %w", err)
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

func newMirrorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Manage Gradle mirrors",
	}

	cmd.AddCommand(newMirrorListCommand())
	cmd.AddCommand(newMirrorSetCommand())
	cmd.AddCommand(newMirrorCurrentCommand())
	cmd.AddCommand(newMirrorTestCommand())

	return cmd
}

func newMirrorListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in mirror sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := mirror.CurrentSource(projectDir)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "CURRENT\tNAME\tDISPLAY\tGRADLE_URL\tMAVEN_URL")
			for _, source := range mirror.BuiltinSources {
				marker := " "
				if source.Name == current {
					marker = "*"
				}
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", marker, source.Name, source.DisplayName, source.GradleURL, source.MavenURL)
			}
			return nil
		},
	}
}

func newMirrorSetCommand() *cobra.Command {
	var (
		sourceName  string
		wrapperOnly bool
		mavenOnly   bool
		ciMode      bool
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the active mirror source",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ciMode && sourceName == "" {
				return fmt.Errorf("--ci requires --source")
			}

			if interactive && sourceName == "" {
				sourceName = "aliyun"
			}

			if sourceName == "" {
				return fmt.Errorf("--source is required")
			}

			source := mirror.FindByName(sourceName)
			if source == nil {
				return fmt.Errorf("unknown mirror source: %s", sourceName)
			}

			if wrapperOnly || mavenOnly {
				// P1 scope only persists selection. File mutation will be added in P2.
			}

			return mirror.SaveConfig(projectDir, source.Name)
		},
	}

	cmd.Flags().StringVar(&sourceName, "source", "", "Mirror source name")
	cmd.Flags().BoolVar(&wrapperOnly, "wrapper-only", false, "Only change wrapper mirror")
	cmd.Flags().BoolVar(&mavenOnly, "maven-only", false, "Only change Maven mirror")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Non-interactive mode")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive selection")

	return cmd
}

func newMirrorCurrentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current mirror source",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := mirror.CurrentSource(projectDir)
			if err != nil {
				return err
			}
			if current == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "not configured")
				return nil
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), current)
			return nil
		},
	}
}

func newMirrorTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test mirror availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "mirror test is not implemented yet")
			return nil
		},
	}
}
