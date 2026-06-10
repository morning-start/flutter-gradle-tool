package gradle

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	apperrors "flutter-gradle-tool/internal/errors"
)

func GradleUserHome() string {
	if env := os.Getenv("GRADLE_USER_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gradle"
	}
	return filepath.Join(home, ".gradle")
}

func RunGradle(projectDir string, tasks []string) (string, error) {
	if len(tasks) == 0 {
		return "", apperrors.New(apperrors.ExitUnknownCommand, "at least one gradle task is required")
	}

	wrapperDir, wrapperName, err := findWrapper(projectDir)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		args := append([]string{"/C", wrapperName}, tasks...)
		cmd = exec.Command("cmd", args...)
	} else {
		cmd = exec.Command("./"+wrapperName, tasks...)
	}
	cmd.Dir = wrapperDir
	cmd.Env = append(os.Environ(), "GRADLE_USER_HOME="+GradleUserHome())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), apperrors.Wrap(apperrors.ExitUnknownCommand, "gradle exec failed", err)
	}
	return string(output), nil
}

func findWrapper(projectDir string) (string, string, error) {
	candidates := []string{
		filepath.Join(projectDir, "android"),
		projectDir,
	}

	for _, base := range candidates {
		if runtime.GOOS == "windows" {
			path := filepath.Join(base, "gradlew.bat")
			if fileExists(path) {
				return base, "gradlew.bat", nil
			}
		} else {
			path := filepath.Join(base, "gradlew")
			if fileExists(path) {
				return base, "gradlew", nil
			}
		}
	}

	return "", "", apperrors.New(apperrors.ExitProjectNotFound, "gradle wrapper not found")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
