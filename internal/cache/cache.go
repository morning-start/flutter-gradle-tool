package cache

import (
	"fmt"
	"os"
	"path/filepath"

	apperrors "flutter-gradle-tool/internal/errors"
)

type Info struct {
	Root       string
	CachesDir  string
	WrapperDir string
	TotalSize  int64
	Exists     bool
}

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

func Inspect() (Info, error) {
	root := GradleUserHome()
	info := Info{
		Root:       root,
		CachesDir:  filepath.Join(root, "caches"),
		WrapperDir: filepath.Join(root, "wrapper"),
	}

	if _, err := os.Stat(root); err == nil {
		info.Exists = true
	}

	total, err := dirSize(root)
	if err != nil {
		return Info{}, err
	}
	info.TotalSize = total
	return info, nil
}

func CleanAll() ([]string, error) {
	root := GradleUserHome()
	targets := []string{
		filepath.Join(root, "caches"),
		filepath.Join(root, "wrapper", "dists"),
	}

	var removed []string
	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			if err := os.RemoveAll(target); err != nil {
				return removed, apperrors.Wrap(apperrors.ExitPermission, fmt.Sprintf("remove %s", target), err)
			}
			removed = append(removed, target)
		}
	}

	return removed, nil
}

func dirSize(root string) (int64, error) {
	var total int64
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
