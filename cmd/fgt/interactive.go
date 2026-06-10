package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/mirror"
)

// resolveSource resolves a mirror source from a name or interactive selection.
// If allowDetect is true, it falls back to detecting the current source from the project.
func resolveSource(cmd *cobra.Command, sourceName string, interactive, allowDetect bool) (*mirror.Source, error) {
	if sourceName != "" {
		source := mirror.FindByName(sourceName)
		if source == nil {
			return nil, errors.New(errors.ExitUnknownSource, fmt.Sprintf("unknown mirror source: %s", sourceName))
		}
		return source, nil
	}

	if interactive {
		return chooseMirrorSourceInteractively(cmd)
	}

	if allowDetect {
		if current, err := mirror.CurrentSource(projectDir); err != nil {
			return nil, err
		} else if current != "" {
			source := mirror.FindByName(current)
			if source != nil {
				return source, nil
			}
		}
	}

	return nil, fmt.Errorf("--source is required")
}

func chooseMirrorSourceInteractively(cmd *cobra.Command) (*mirror.Source, error) {
	current, _ := mirror.CurrentSource(projectDir)
	out := cmd.OutOrStdout()
	in := bufio.NewReader(cmd.InOrStdin())

	fmt.Fprintln(out, "Available mirror sources:")
	for i, source := range mirror.BuiltinSources {
		marker := " "
		if source.Name == current {
			marker = "*"
		}
		fmt.Fprintf(out, "%s %d) %s - %s\n", marker, i+1, source.Name, source.DisplayName)
	}
	fmt.Fprint(out, "Select mirror source by number: ")

	choiceLine, err := in.ReadString('\n')
	if err != nil {
		return nil, err
	}
	choiceLine = strings.TrimSpace(choiceLine)
	if choiceLine == "" {
		return nil, errors.New(errors.ExitUnknownCommand, "selection is required")
	}

	index, err := strconv.Atoi(choiceLine)
	if err != nil || index < 1 || index > len(mirror.BuiltinSources) {
		return nil, errors.New(errors.ExitUnknownSource, fmt.Sprintf("invalid selection: %s", choiceLine))
	}

	selected := mirror.BuiltinSources[index-1]
	fmt.Fprintf(out, "Apply %s? [y/N]: ", selected.Name)
	confirmLine, err := in.ReadString('\n')
	if err != nil {
		return nil, err
	}
	confirmLine = strings.TrimSpace(strings.ToLower(confirmLine))
	if confirmLine != "y" && confirmLine != "yes" {
		return nil, errors.New(errors.ExitUnknownCommand, "selection cancelled")
	}

	return &selected, nil
}