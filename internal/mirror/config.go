package mirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProjectConfig struct {
	Source string `json:"source"`
}

func ConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".fgt-config")
}

func SaveConfig(projectDir, source string) error {
	data, err := json.Marshal(ProjectConfig{Source: normalizeName(source)})
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(projectDir), data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return ensureGitIgnore(projectDir)
}

func ensureGitIgnore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	entry := ".fgt-config"

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read .gitignore: %w", err)
		}
		return os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644)
	}

	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("append .gitignore: %w", err)
	}
	defer f.Close()

	if !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("write .gitignore: %w", err)
		}
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

func LoadConfig(projectDir string) (string, error) {
	data, err := os.ReadFile(ConfigPath(projectDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}
	return normalizeName(cfg.Source), nil
}

func CurrentSource(projectDir string) (string, error) {
	if source, err := LoadConfig(projectDir); err != nil {
		return "", err
	} else if source != "" {
		return source, nil
	}

	return ReverseInferSource(projectDir)
}

func ReverseInferSource(projectDir string) (string, error) {
	for _, candidate := range wrapperCandidates(projectDir) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("read wrapper config: %w", err)
		}

		distributionURL := ExtractDistributionURL(string(data))
		if distributionURL == "" {
			continue
		}

		if source := SourceFromDistributionURL(distributionURL); source != nil {
			return source.Name, nil
		}
	}

	return "", nil
}

func wrapperCandidates(projectDir string) []string {
	return []string{
		filepath.Join(projectDir, "android", "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "gradle", "wrapper", "gradle-wrapper.properties"),
		filepath.Join(projectDir, "android", "gradle-wrapper.properties"),
	}
}

func ExtractDistributionURL(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "distributionUrl=") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, "distributionUrl="))
	}
	return ""
}
