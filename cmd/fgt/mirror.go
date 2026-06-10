package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/errors"
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

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(tw, "\tNAME\tGRADLE URL")
			fmt.Fprintln(tw, "\t----\t----------")
			for _, source := range mirror.BuiltinSources {
				marker := " "
				if source.Name == current {
					marker = "*"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", marker, source.Name, source.GradleURL)
			}
			return tw.Flush()
		},
	}
}

func newMirrorSetCommand() *cobra.Command {
	var (
		sourceName  string
		ciMode      bool
		interactive bool
		wrapperOnly bool
		mavenOnly   bool
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
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Non-interactive mode")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive selection")
	cmd.Flags().BoolVarP(&wrapperOnly, "wrapper-only", "w", false, "Only change wrapper mirror")
	cmd.Flags().BoolVarP(&mavenOnly, "maven-only", "m", false, "Only change Maven mirror")

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
			concurrency := 4
			if env := os.Getenv("FGT_TEST_CONCURRENCY"); env != "" {
				if v, err := strconv.Atoi(env); err == nil && v > 0 {
					concurrency = v
				}
			}
			results := mirror.TestSources(cmd.Context(), mirror.BuiltinSources, 5*time.Second, concurrency)
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(tw, "NAME\tSTATUS\tDURATION\tURL")
			fmt.Fprintln(tw, "----\t------\t--------\t---")
			for _, result := range results {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", result.Source.Name, result.Status, result.Duration, result.TestedURL)
			}
			if err := tw.Flush(); err != nil {
				return err
			}

			allFailed := true
			for _, r := range results {
				if r.OK {
					allFailed = false
					break
				}
			}
			if allFailed {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "all mirror sources are unreachable, check your network")
				return errors.New(errors.ExitNetwork, "all mirror sources unreachable")
			}
			return nil
		},
	}
}