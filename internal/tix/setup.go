package tix

import (
	"fmt"
	"path/filepath"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/starter"
)

func askConfig(setup prompter) (backend.Config, error) {
	prefix, err := setup.ask(questionPrefix, "", checkPrefix)
	if err != nil {
		return backend.Config{}, err
	}
	kindAnswer, err := setup.ask(questionBackend, string(backend.KindVault), checkKind)
	if err != nil {
		return backend.Config{}, err
	}
	kind := backend.Kind(kindAnswer)
	if kind != backend.KindVault {
		return configFor(prefix, kind, ""), nil
	}
	root, err := setup.ask(questionDocsDirectory, backend.DefaultVaultRoot, checkDocsDirectory)
	if err != nil {
		return backend.Config{}, err
	}
	return configFor(prefix, kind, root), nil
}

// configFor returns the config tix init writes: the prefix alone for the YAML backend, as tix v1 wrote it, or the
// starter vault rooted at root.
func configFor(prefix string, kind backend.Kind, root string) backend.Config {
	if kind != backend.KindVault {
		return backend.Config{Prefix: prefix}
	}
	profile := starter.Profile(root)
	return backend.Config{Prefix: prefix, Backend: kind, Vault: &profile}
}

func checkPrefix(answer string) error {
	if !ticket.IsValidPrefix(answer) {
		return fmt.Errorf("%q %w %s", answer, ErrPrefixMismatch, ticket.PrefixPattern)
	}
	return nil
}

func checkKind(answer string) error {
	if !backend.Kind(answer).IsKnown() {
		options := validation.InvalidOption(backend.Kinds)
		return fmt.Errorf("%q %w: %s", answer, ErrInvalidOption, options)
	}
	return nil
}

func checkDocsDirectory(answer string) error {
	if !filepath.IsLocal(answer) {
		return fmt.Errorf("%q %w", answer, ErrDocsDirectoryNotLocal)
	}
	return nil
}
