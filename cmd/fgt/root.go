package main

import (
	"fmt"

	"github.com/spf13/cobra"

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
	cmd.AddCommand(newPlaceholderCommand("init"))
	cmd.AddCommand(newPlaceholderCommand("cache"))
	cmd.AddCommand(newPlaceholderCommand("doctor"))
	cmd.AddCommand(newPlaceholderCommand("exec"))

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
