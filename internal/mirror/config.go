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
	return nil
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

		distributionURL := extractDistributionURL(string(data))
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

func extractDistributionURL(content string) string {
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
