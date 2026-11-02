package backend

import (
	"os"
	"path/filepath"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Config is .tix/config.yml.
type Config struct {
	Prefix string `yaml:"prefix"`
	// Backend is the kind of backend that stores the tickets; empty means KindYAML.
	Backend Kind `yaml:"backend,omitempty"`
}

func (config Config) Validate() error {
	var problems validation.Errors
	if !ticket.IsValidPrefix(config.Prefix) {
		problems.Add(configFieldPrefix, problemPrefixMismatch+ticket.PrefixPattern)
	}
	if config.Backend != "" && !config.Backend.IsKnown() {
		problems.Add(configFieldBackend, validation.InvalidOption(Kinds))
	}
	return problems.Err()
}

// BackendKind is the configured backend, defaulting to KindYAML when the config names none.
func (config Config) BackendKind() Kind {
	if config.Backend == "" {
		return KindYAML
	}
	return config.Backend
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
	document, err := ParseYAML(text, path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := DecodeYAML(document, &config, path); err != nil {
		return Config{}, err
	}
	return config, Validate(config, path)
}

func WriteConfig(tixDirectory string, config Config) error {
	path := ConfigPath(tixDirectory)
	if err := Validate(config, path); err != nil {
		return err
	}
	text, err := EncodeYAML(config)
	if err != nil {
		return err
	}
	return WriteAtomically(path, text)
}
