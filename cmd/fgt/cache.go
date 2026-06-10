package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/cache"
	"flutter-gradle-tool/internal/errors"
)

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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "size: %d\n", info.TotalSize)
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