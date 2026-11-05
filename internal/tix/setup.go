package tix

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

func askConfig(setup prompter) (backend.Config, error) {
	prefix, err := setup.ask(questionPrefix, "", checkPrefix)
	if err != nil {
		return backend.Config{}, err
	}
	config := backend.Config{Prefix: prefix}
	kindAnswer, err := setup.ask(questionBackend, string(backend.KindYAML), checkKind)
	if err != nil {
		return backend.Config{}, err
	}
	kind := backend.Kind(kindAnswer)
	if kind == backend.KindVault {
		profile, err := askProfile(setup)
		if err != nil {
			return backend.Config{}, err
		}
		config.Backend = kind
		config.Vault = &profile
	}
	return config, nil
}

func askProfile(setup prompter) (backend.Profile, error) {
	root, err := setup.ask(questionDocsDirectory, backend.DefaultVaultRoot, checkDocsDirectory)
	if err != nil {
		return backend.Profile{}, err
	}
	statuses, err := parseStatuses(defaultTicketStatuses)
	if err != nil {
		return backend.Profile{}, err
	}
	taskMapping := backend.TicketMapping{TicketType: ticket.TypeTask, Statuses: statuses}
	tickets := map[string]backend.TicketMapping{defaultTicketNoteType: taskMapping}
	profile := backend.Profile{
		Root:            root,
		TypeProperty:    backend.DefaultTypeProperty,
		Tickets:         tickets,
		CriteriaHeading: backend.DefaultCriteriaHeading,
	}
	return profile, nil
}

func parseStatuses(answer string) (map[string]ticket.Status, error) {
	statuses := map[string]ticket.Status{}
	for _, pair := range splitList(answer) {
		noteStatus, ticketStatus, found := strings.Cut(pair, pairSeparator)
		noteStatus = strings.TrimSpace(noteStatus)
		ticketStatus = strings.TrimSpace(ticketStatus)
		if !found || noteStatus == "" || ticketStatus == "" {
			return nil, fmt.Errorf("%q %w", pair, ErrExpectedStatusPair)
		}
		statuses[noteStatus] = ticket.Status(ticketStatus)
	}
	return statuses, nil
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
