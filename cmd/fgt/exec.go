package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/gradle"
)

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