package config

import (
	// "fmt"
	"encoding/json"
	"os"
)

type Config struct {
	DbUrl           string
	CurrentUserName string
}

// func (c *Config) SetUser(Config) error {
//
// }

const configFileName = ".gatorconfig.json"

func getConfigFilePath() (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filepath := homeDirectory + "/" + configFileName
	return filepath, nil
}

func Read() (*Config, error) {
	filepath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	fileContents, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	currentConfig := Config{}

	err = json.Unmarshal(fileContents, &currentConfig)
	if err != nil {
		return nil, err
	}

	return &currentConfig, nil
}
