package main

import (
	"errors"
	"os"

	apperrors "flutter-gradle-tool/internal/errors"
)

func main() {
	if err := execute(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return apperrors.ExitUnknownCommand
}
