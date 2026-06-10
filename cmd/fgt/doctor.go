package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/doctor"
	"flutter-gradle-tool/internal/errors"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the current Flutter Gradle setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := doctor.Check(projectDir)
			if err != nil {
				return errors.Wrap(errors.ExitProjectNotFound, "doctor failed", err)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), doctor.Format(report))
			return nil
		},
	}
}