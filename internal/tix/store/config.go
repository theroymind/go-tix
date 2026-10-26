package store

import (
	"os"
	"path/filepath"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

const (
	configFieldPrefix     = "prefix"
	problemPrefixMismatch = "prefix must match "
)

type Config struct {
	Prefix string `yaml:"prefix"`
}

func (config Config) Validate() error {
	var problems validation.Errors
	if !ticket.IsValidPrefix(config.Prefix) {
		problems.Add(configFieldPrefix, problemPrefixMismatch+ticket.PrefixPattern)
	}
	return problems.Err()
}

func ConfigPath(tixDirectory string) string {
	return filepath.Join(tixDirectory, ticket.ConfigFileName)
}

func ReadConfig(tixDirectory string) (Config, error) {
	path := ConfigPath(tixDirectory)
	text, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	document, err := parseYAML(text, path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := decodeYAML(document, &config, path); err != nil {
		return Config{}, err
	}
	return config, Validate(config, path)
}

func WriteConfig(tixDirectory string, config Config) error {
	path := ConfigPath(tixDirectory)
	if err := Validate(config, path); err != nil {
		return err
	}
	text, err := encodeYAML(config)
	if err != nil {
		return err
	}
	return writeAtomically(path, text)
}
