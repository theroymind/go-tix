package backend

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Config is .tix/config.yml.
type Config struct {
	Prefix string `yaml:"prefix"`
	// Backend is the kind of backend that stores the tickets; empty means KindYAML.
	Backend Kind `yaml:"backend,omitempty"`
	// Vault is the vault backend's Profile, which that backend requires.
	Vault *Profile `yaml:"vault,omitempty"`
	// Embeddings names the endpoint and model tix index and tix search embed through. Without it they refuse to run.
	Embeddings *Embeddings `yaml:"embeddings,omitempty"`
}

// Embeddings is an OpenAI-compatible embeddings server: the base URL that /embeddings is appended to, such as
// http://localhost:11434/v1, and the model it serves.
type Embeddings struct {
	Endpoint string `yaml:"endpoint"`
	Model    string `yaml:"model"`
}

func (embeddings Embeddings) Validate() error {
	var problems validation.Errors
	parsed, err := url.Parse(embeddings.Endpoint)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		problems.Add(embeddingsFieldEndpoint, problemEndpointNotURL)
	}
	if strings.TrimSpace(embeddings.Model) == "" {
		problems.Add(embeddingsFieldModel, problemModelRequired)
	}
	return problems.Err()
}

func (config Config) Validate() error {
	var problems validation.Errors
	if !ticket.IsValidPrefix(config.Prefix) {
		problems.Add(configFieldPrefix, problemPrefixMismatch+ticket.PrefixPattern)
	}
	if config.Backend != "" && !config.Backend.IsKnown() {
		problems.Add(configFieldBackend, validation.InvalidOption(Kinds))
	}
	if config.Vault != nil {
		problems.AddNested(configFieldVault, config.Vault.Validate())
	} else if config.Backend == KindVault {
		problems.Add(configFieldVault, problemVaultRequired)
	}
	if config.Embeddings != nil {
		problems.AddNested(configFieldEmbeddings, config.Embeddings.Validate())
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
	if _, err := ParseYAML(text, path); err != nil {
		return Config{}, err
	}
	var config Config
	if err := DecodeKnownFields(text, &config, path); err != nil {
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
