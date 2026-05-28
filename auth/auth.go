package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type AuthConfig struct {
	Token string `json:"token"`
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "claudenelson", "config.json"), nil
}

// LoadToken reads the config file and extracts the saved token string
func LoadToken() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", errors.New("you are not logged in. Please run 'claudenelson auth login' first")
	}

	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AuthConfig
	if err := json.Unmarshal(fileBytes, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse config file JSON: %w", err)
	}

	if cfg.Token == "" {
		return "", errors.New("saved token is empty. Please run 'claudenelson auth login' again")
	}

	return cfg.Token, nil
}