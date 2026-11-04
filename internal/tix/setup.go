package tix

import (
	"fmt"
	"net/url"
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
	embeddings, err := askEmbeddings(setup)
	if err != nil {
		return backend.Config{}, err
	}
	config.Embeddings = embeddings
	return config, nil
}

func askProfile(setup prompter) (backend.Profile, error) {
	root, err := setup.ask(questionDocsDirectory, backend.DefaultVaultRoot, checkDocsDirectory)
	if err != nil {
		return backend.Profile{}, err
	}
	typeProperty, err := setup.ask(questionTypeProperty, backend.DefaultTypeProperty, checkRequired)
	if err != nil {
		return backend.Profile{}, err
	}
	criteriaHeading, err := setup.ask(questionCriteriaHeading, backend.DefaultCriteriaHeading, checkRequired)
	if err != nil {
		return backend.Profile{}, err
	}
	ticketNoteTypes, err := setup.askList(questionTicketNoteTypes, defaultTicketNoteTypes, checkTicketNoteTypes)
	if err != nil {
		return backend.Profile{}, err
	}
	tickets := map[string]backend.TicketMapping{}
	for _, noteType := range ticketNoteTypes {
		mapping, err := askTicketMapping(setup, noteType)
		if err != nil {
			return backend.Profile{}, err
		}
		tickets[noteType] = mapping
	}
	checkDocTypes := func(docNoteTypes []string) error {
		return checkDocNoteTypes(docNoteTypes, tickets)
	}
	docNoteTypes, err := setup.askList(questionDocNoteTypes, "", checkDocTypes)
	if err != nil {
		return backend.Profile{}, err
	}
	profile := backend.Profile{Root: root, TypeProperty: typeProperty, Tickets: tickets, CriteriaHeading: criteriaHeading}
	if len(docNoteTypes) > 0 {
		profile.Docs = map[string]backend.DocMapping{}
	}
	for _, noteType := range docNoteTypes {
		statusesQuestion := fmt.Sprintf(questionDocStatuses, noteType)
		statuses, err := setup.askList(statusesQuestion, "", acceptAnyList)
		if err != nil {
			return backend.Profile{}, err
		}
		profile.Docs[noteType] = backend.DocMapping{Statuses: statuses}
	}
	return profile, nil
}

func askTicketMapping(setup prompter, noteType string) (backend.TicketMapping, error) {
	fallbackType := ""
	if ticket.Type(noteType).IsValid() {
		fallbackType = noteType
	}
	typeQuestion := fmt.Sprintf(questionTicketType, noteType)
	typeAnswer, err := setup.ask(typeQuestion, fallbackType, checkTicketType)
	if err != nil {
		return backend.TicketMapping{}, err
	}
	mapping := backend.TicketMapping{TicketType: ticket.Type(typeAnswer)}
	checkStatuses := func(answer string) error {
		statuses, err := parseStatuses(answer)
		if err != nil {
			return err
		}
		candidate := backend.TicketMapping{TicketType: mapping.TicketType, Statuses: statuses}
		if err := candidate.Validate(); err != nil {
			return err
		}
		mapping = candidate
		return nil
	}
	statusesQuestion := fmt.Sprintf(questionTicketStatuses, noteType)
	if _, err := setup.ask(statusesQuestion, defaultTicketStatuses, checkStatuses); err != nil {
		return backend.TicketMapping{}, err
	}
	return mapping, nil
}

func askEmbeddings(setup prompter) (*backend.Embeddings, error) {
	endpoint, err := setup.ask(questionEmbeddingsEndpoint, "", checkEndpoint)
	if err != nil {
		return nil, err
	}
	if endpoint == "" {
		return nil, nil
	}
	model, err := setup.ask(questionEmbeddingsModel, "", checkRequired)
	if err != nil {
		return nil, err
	}
	return &backend.Embeddings{Endpoint: endpoint, Model: model}, nil
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

func checkTicketType(answer string) error {
	if !ticket.Type(answer).IsValid() {
		options := validation.InvalidOption(ticket.Types)
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

func checkRequired(answer string) error {
	if answer == "" {
		return ErrAnswerRequired
	}
	return nil
}

func checkEndpoint(answer string) error {
	if answer == "" {
		return nil
	}
	parsed, err := url.Parse(answer)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("%q %w", answer, ErrEndpointNotURL)
	}
	return nil
}

func checkTicketNoteTypes(noteTypes []string) error {
	if len(noteTypes) == 0 {
		return ErrExpectedTicketNoteType
	}
	return nil
}

func checkDocNoteTypes(noteTypes []string, tickets map[string]backend.TicketMapping) error {
	for _, noteType := range noteTypes {
		if _, isTicket := tickets[noteType]; isTicket {
			return fmt.Errorf("%q %w", noteType, ErrTicketAndDocType)
		}
	}
	return nil
}

func acceptAnyList([]string) error {
	return nil
}
