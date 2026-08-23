package config

import (
	"encoding/json"
	"fmt"
	"os"

	"bekadoux/gator/internal/common"
)

const defaultConfigFilename = ".gatorconfig.json"
const userNameCharLimit = 32

type Config struct {
	Filepath        string `json:"-"`
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func (c *Config) SetUser(newUserName string) error {
	if len(newUserName) > userNameCharLimit || newUserName == "" {
		return fmt.Errorf("invalid user name: %q", newUserName)
	}

	c.CurrentUserName = newUserName
	if err := c.write(); err != nil {
		return fmt.Errorf("could not write config to JSON: %w", err)
	}

	return nil
}

func (c *Config) write() (err error) {
	path, err := getDefaultConfigFilePath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not open file %q for writing: %w", path, err)
	}
	defer common.CloseWithError(&err, file, fmt.Sprintf("open file %q", path))

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("could not write JSON to %q: %w", path, err)
	}

	return nil
}

func Read(path string) (cfg Config, err error) {
	if path == "" {
		path, err = getDefaultConfigFilePath()
		if err != nil {
			return Config{}, fmt.Errorf("could not get default config file path: %w", err)
		}
	}
	cfg.Filepath = path

	file, err := os.Open(cfg.Filepath)
	if err != nil {
		return Config{}, fmt.Errorf("could not read gator config file at %q: %w", cfg.Filepath, err)
	}
	defer common.CloseWithError(&err, file, fmt.Sprintf("open file %q", cfg.Filepath))

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("could not decode JSON config file: %w", err)
	}

	return cfg, nil
}

func getDefaultConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user's home directory: %w", err)
	}

	path := fmt.Sprintf("%s/%s", homeDir, defaultConfigFilename)

	return path, nil
}
