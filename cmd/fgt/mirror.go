package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/mirror"
)

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

			source, err := resolveSource(cmd, sourceName, interactive, false)
			if err != nil {
				return err
			}

			return mirror.SaveConfig(projectDir, source.Name)
		},
	}

	cmd.Flags().StringVar(&sourceName, "source", "", "Mirror source name")
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
			results := mirror.TestSources(cmd.Context(), mirror.BuiltinSources, 5*time.Second, 4)
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "NAME\tSTATUS\tDURATION\tURL")
			for _, result := range results {
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", result.Source.Name, result.Status, result.Duration, result.TestedURL)
			}
			return nil
		},
	}
}