package main

import (
	"os"

	"github.com/spf13/cobra"

	apperrors "flutter-gradle-tool/internal/errors"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(projectDir); err != nil {
				return apperrors.Wrap(apperrors.ExitProjectNotFound, "project dir not found", err)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&projectDir, "project-dir", ".", "Path to Flutter project root")

	cmd.AddCommand(newMirrorCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newCacheCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newExecCommand())

	return cmd
}
