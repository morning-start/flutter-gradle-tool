package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flutter-gradle-tool/internal/errors"
	"flutter-gradle-tool/internal/gradle"
	"flutter-gradle-tool/internal/mirror"
	"flutter-gradle-tool/internal/mise"
)

func newInitCommand() *cobra.Command {
	var (
		sourceName  string
		wrapperOnly bool
		mavenOnly   bool
		ciMode      bool
		interactive bool
		miseMode    bool
		restoreMode bool
		cleanZip    bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Gradle mirror settings for a Flutter project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if restoreMode {
				return runRestore(cmd, projectDir, cleanZip)
			}
			if ciMode && sourceName == "" {
				return errors.New(errors.ExitCIRequiresSource, "--ci requires --source")
			}
			if miseMode {
				return runMiseInit(cmd, projectDir, wrapperOnly, mavenOnly)
			}
			return runMirrorInit(cmd, projectDir, sourceName, interactive, wrapperOnly, mavenOnly)
		},
	}

	cmd.Flags().StringVarP(&sourceName, "source", "s", "", "Mirror source name")
	cmd.Flags().BoolVarP(&wrapperOnly, "wrapper-only", "w", false, "Only change wrapper mirror")
	cmd.Flags().BoolVarP(&mavenOnly, "maven-only", "m", false, "Only change Maven mirror")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Non-interactive mode")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive selection")
	cmd.Flags().BoolVar(&miseMode, "mise", false, "Use mise-managed local Gradle distribution")
	cmd.Flags().BoolVar(&restoreMode, "restore", false, "Restore original Gradle settings (undo fgt changes)")
	cmd.Flags().BoolVar(&cleanZip, "clean-zip", false, "Also remove mise-generated local zip files")

	return cmd
}

// --- restore mode ---

func runRestore(cmd *cobra.Command, projectDir string, cleanZip bool) error {
	out := cmd.ErrOrStderr()
	var restored int

	// 1. Restore gradle-wrapper.properties from git.
	if path, err := findWrapperPath(projectDir); err == nil {
		if gitContent, err := gitShowHEAD(path); err == nil {
			current, _ := os.ReadFile(path)
			if string(current) != gitContent {
				if err := os.WriteFile(path, []byte(gitContent), 0o644); err != nil {
					return errors.Wrap(errors.ExitPermission, "restore wrapper file", err)
				}
				_, _ = fmt.Fprintf(out, "restored: %s\n", path)
				restored++
			}
		} else {
			_, _ = fmt.Fprintf(out, "skip wrapper restore (not in git or no git): %v\n", err)
		}
	}

	// 2. Remove Maven mirror blocks from build.gradle.
	if path, err := findBuildGradlePath(projectDir); err == nil {
		content, err := os.ReadFile(path)
		if err == nil {
			official := mirror.Source{Name: "official"}
			var updated string
			var changed bool
			if strings.HasSuffix(path, ".kts") {
				updated, changed, err = gradle.RewriteBuildGradleKTS(string(content), official)
			} else {
				updated, changed, err = gradle.RewriteBuildGradle(string(content), official)
			}
			if err == nil && changed {
				if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
					return errors.Wrap(errors.ExitPermission, "restore build.gradle", err)
				}
				_, _ = fmt.Fprintf(out, "restored: %s\n", path)
				restored++
			}
		}
	}

	// 3. Delete .fgt-config.
	configPath := mirror.ConfigPath(projectDir)
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Remove(configPath); err != nil {
			return errors.Wrap(errors.ExitPermission, "remove .fgt-config", err)
		}
		_, _ = fmt.Fprintf(out, "removed: %s\n", configPath)
		restored++
	}

	// 4. Optionally clean mise-generated zip files.
	if cleanZip {
		if n, err := cleanMiseZips(projectDir); err != nil {
			_, _ = fmt.Fprintf(out, "warning: clean zip: %v\n", err)
		} else if n > 0 {
			_, _ = fmt.Fprintf(out, "removed %d local zip file(s)\n", n)
			restored += n
		}
	}

	if restored == 0 {
		_, _ = fmt.Fprintln(out, "nothing to restore — project is clean")
	} else {
		_, _ = fmt.Fprintf(out, "restored %d item(s)\n", restored)
	}
	return nil
}

func gitShowHEAD(absPath string) (string, error) {
	dir := filepath.Dir(absPath)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}

	relPath, err := filepath.Rel(dir, absPath)
	if err != nil {
		return "", err
	}
	relPath = filepath.ToSlash(relPath)

	out, err := execCommand(dir, "git", "show", "HEAD:"+relPath)
	if err != nil {
		return "", fmt.Errorf("git show HEAD:%s: %w", relPath, err)
	}
	return out, nil
}

func execCommand(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func cleanMiseZips(projectDir string) (int, error) {
	if !mise.IsMiseInstalled() {
		return 0, nil
	}
	info, err := mise.DetectGradle(projectDir)
	if err != nil || info == nil || info.InstallDir == "" {
		return 0, nil
	}
	var removed int
	for _, e := range mustReadDir(info.InstallDir) {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "gradle-") && strings.HasSuffix(name, ".zip") {
			if os.Remove(filepath.Join(info.InstallDir, name)) == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// --- mirror mode ---

func runMirrorInit(cmd *cobra.Command, projectDir, sourceName string, interactive, wrapperOnly, mavenOnly bool) error {
	source, err := resolveSource(cmd, sourceName, interactive, true)
	if err != nil {
		return err
	}

	if source.Name == "official" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "warning: official source uses overseas servers, download speed may be slow")
	}

	if !wrapperOnly {
		if err := rewriteWrapperFile(projectDir, *source); err != nil {
			return err
		}
	}
	if !mavenOnly {
		if err := rewriteBuildGradleFile(projectDir, *source); err != nil {
			return err
		}
	}
	if err := mirror.SaveConfig(projectDir, source.Name); err != nil {
		return err
	}
	return nil
}

// --- mise mode ---

func runMiseInit(cmd *cobra.Command, projectDir string, wrapperOnly, mavenOnly bool) error {
	out := cmd.ErrOrStderr()

	if !mise.IsMiseInstalled() {
		return errors.New(errors.ExitUnknownSource, "mise is not installed or not on PATH")
	}

	info, err := mise.DetectGradle(projectDir)
	if err != nil {
		return fmt.Errorf("detect mise gradle: %w", err)
	}
	if info == nil {
		return errors.New(errors.ExitUnknownSource, "mise does not manage a Gradle version (run 'mise use gradle@<version>' first)")
	}
	if !info.Available {
		return errors.New(errors.ExitWrapperParse, fmt.Sprintf(
			"mise manages Gradle %s but no usable installation found", info.Version))
	}

	_, _ = fmt.Fprintf(out, "detected mise-managed Gradle %s (source: %s)\n", info.Version, info.Source)

	if !wrapperOnly {
		if err := rewriteWrapperForMise(projectDir, info); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "gradle-wrapper.properties updated to use local mise distribution")
	}

	if !mavenOnly && !wrapperOnly {
		_, _ = fmt.Fprintln(out, "tip: use 'fgt init --source <mirror>' separately to also configure Maven mirrors")
	}
	return nil
}

func rewriteWrapperForMise(projectDir string, info *mise.GradleInfo) error {
	if info.GradleHome == "" {
		return errors.New(errors.ExitWrapperParse, "no extracted Gradle directory found from mise")
	}

	wrapperPath, err := findWrapperPath(projectDir)
	if err != nil {
		return err
	}

	wrapperContent, err := os.ReadFile(wrapperPath)
	if err != nil {
		return fmt.Errorf("read wrapper file: %w", err)
	}

	zipPath := info.ZipPath
	if zipPath == "" {
		_, distType, _ := gradle.ParseWrapperDistributionURL(string(wrapperContent))
		if distType == "" {
			distType = "all"
		}
		zipPath, err = ensureLocalZip(info.GradleHome, info.Version, distType)
		if err != nil {
			return err
		}
	}

	updated, changed, err := gradle.RewriteWrapperPropertiesToLocal(string(wrapperContent), zipPath)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(wrapperPath, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write wrapper file", err)
		}
	}
	return nil
}

func ensureLocalZip(gradleHome, version, distType string) (string, error) {
	zipPath := filepath.Join(filepath.Dir(gradleHome), fmt.Sprintf("gradle-%s-%s.zip", version, distType))
	if _, err := os.Stat(zipPath); err == nil {
		return zipPath, nil
	}
	fmt.Fprintf(os.Stderr, "creating local gradle zip from %s ...\n", gradleHome)
	if err := createZipFromDir(gradleHome, zipPath); err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	return zipPath, nil
}

func createZipFromDir(sourceDir, zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	baseDir := filepath.Base(sourceDir)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(baseDir, relPath))

		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(writer, src)
		return err
	})
}

// --- shared helpers ---

func findWrapperPath(projectDir string) (string, error) {
	return firstExistingPath(
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	)
}

func findBuildGradlePath(projectDir string) (string, error) {
	return firstExistingPath(
		filepath.Join(projectDir, "android", "build.gradle.kts"),
		filepath.Join(projectDir, "build.gradle.kts"),
		filepath.Join(projectDir, "android", "build.gradle"),
		filepath.Join(projectDir, "build.gradle"),
	)
}

func findGitRoot(dir string) string {
	for i := 0; i < 20; i++ {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

func rewriteWrapperFile(projectDir string, source mirror.Source) error {
	path, err := findWrapperPath(projectDir)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read wrapper file: %w", err)
	}
	updated, changed, err := gradle.RewriteWrapperProperties(string(content), source)
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write wrapper file", err)
		}
	}
	return nil
}

func rewriteBuildGradleFile(projectDir string, source mirror.Source) error {
	path, err := findBuildGradlePath(projectDir)
	if err != nil {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read build.gradle: %w", err)
	}
	var updated string
	var changed bool
	if strings.HasSuffix(path, ".kts") {
		updated, changed, err = gradle.RewriteBuildGradleKTS(string(content), source)
	} else {
		updated, changed, err = gradle.RewriteBuildGradle(string(content), source)
	}
	if err != nil {
		return err
	}
	if changed {
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return errors.Wrap(errors.ExitPermission, "write build.gradle", err)
		}
	}
	return nil
}

func firstExistingPath(paths ...string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("required file not found")
}

func mustReadDir(dir string) []os.DirEntry {
	entries, _ := os.ReadDir(dir)
	return entries
}
