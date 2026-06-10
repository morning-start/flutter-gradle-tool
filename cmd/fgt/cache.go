package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/cache"
	"flutter-gradle-tool/internal/errors"
)

func formatSize(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(size)/(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(size)/(1<<20))
	default:
		return fmt.Sprintf("%.1f KiB", float64(size)/(1<<10))
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
				return errors.Wrap(errors.ExitProjectNotFound, "cache inspection failed", err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", info.Root)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "size: %s\n", formatSize(info.TotalSize))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "caches:")
			for _, v := range info.CachesBreakdown {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %s\n", v.Name, formatSize(v.Size))
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "wrapper dists:")
			for _, v := range info.WrapperBreakdown {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %s\n", v.Name, formatSize(v.Size))
			}
			return nil
		},
	}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean Gradle cache directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cleanAll {
				return errors.New(errors.ExitUnknownCommand, "use --all to remove cache directories")
			}
			removed, err := cache.CleanAll()
			if err != nil {
				return errors.Wrap(errors.ExitProjectNotFound, "cache clean failed", err)
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