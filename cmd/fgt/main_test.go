package main

import (
	"errors"
	"testing"

	apperrors "flutter-gradle-tool/internal/errors"
)

func TestExitCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "app error", err: apperrors.New(apperrors.ExitUnknownSource, "bad source"), want: apperrors.ExitUnknownSource},
		{name: "wrapped app error", err: apperrors.Wrap(apperrors.ExitCIRequiresSource, "init failed", errors.New("missing source")), want: apperrors.ExitCIRequiresSource},
		{name: "plain error", err: errors.New("plain"), want: apperrors.ExitUnknownCommand},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := exitCode(tc.err); got != tc.want {
				t.Fatalf("exitCode() = %d, want %d", got, tc.want)
			}
		})
	}
}
