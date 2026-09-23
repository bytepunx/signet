package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type cliConfig struct {
	Server    string `yaml:"server"`
	TokenFile string `yaml:"token_file"`
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "signet", "config.yaml"), nil
}

func readCliConfig() (cliConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return cliConfig{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cliConfig{}, nil
	}
	if err != nil {
		return cliConfig{}, fmt.Errorf("read config: %w", err)
	}
	var cfg cliConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cliConfig{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func writeCliConfig(cfg cliConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
