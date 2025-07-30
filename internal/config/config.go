package config

import (
	"encoding/json"
	"os"
	"syscall"
)

type Config struct {
	DbUrl           string
	CurrentUserName string
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	err := write(*c)
	if err != nil {
		return err
	}

	return nil
}

const configFileName = ".gatorconfig.json"

func write(c Config) error {
	filepath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	configData, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath, configData, syscall.O_RDWR)
	if err != nil {
		return err
	}
	return nil
}

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
