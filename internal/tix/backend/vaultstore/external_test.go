package vaultstore_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

const (
	externalID     = "INO-343"
	externalURL    = "https://tracker.example.com/issue/INO-343"
	mappedExternal = "      fields: {tracker_id: external_id, tracker_link: external_url}\n"
)

func externalVault(t *testing.T, fields, note string) (backend.Backend, string) {
	t.Helper()
	return writeVault(t, map[string]string{
		".tix/config.yml":   smallProfile(fields),
		"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
		"tasks/existing.md": note,
	})
}

func TestReadExternalFields(t *testing.T) {
	tests := []struct {
		name        string
		fields      string
		note        string
		expectedID  string
		expectedURL string
	}{
		{"A note without them has none", "", smallNote, "", ""},
		{"Unmapped, they are the properties named after the fields", "",
			"---\ntype: task\nstatus: todo\nexternal_id: INO-343\nexternal_url: " + externalURL + "\n---\n\n# Existing\n",
			externalID, externalURL},
		{"Mapped, they are the properties the profile names", mappedExternal,
			"---\ntype: task\nstatus: todo\ntracker_id: INO-343\ntracker_link: " + externalURL + "\n---\n\n# Existing\n",
			externalID, externalURL},
		{"Mapped, the properties named after the fields are ignored", mappedExternal,
			"---\ntype: task\nstatus: todo\nexternal_id: INO-343\n---\n\n# Existing\n", "", ""},
		{"Unmapped, a property named after the field but mapped to another field holds nothing", "      fields: {external_id: parent}\n",
			"---\ntype: task\nstatus: todo\nexternal_id: other\n---\n\n# Existing\n", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, _ := externalVault(t, tt.fields, tt.note)

			read, err := store.Read("existing")
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if read.ExternalID != tt.expectedID || read.ExternalURL != tt.expectedURL {
				t.Errorf("Read() external = %q, %q, expected %q, %q", read.ExternalID, read.ExternalURL, tt.expectedID, tt.expectedURL)
			}
		})
	}
}

func TestReadRejectsAnExternalURLThatIsNotAWebURL(t *testing.T) {
	store, _ := externalVault(t, "", "---\ntype: task\nstatus: todo\nexternal_url: INO-343\n---\n\n# Existing\n")

	_, err := store.Read("existing")

	var problems validation.Errors
	if !errors.As(err, &problems) || !strings.Contains(err.Error(), ticket.FieldExternalURL) {
		t.Errorf("Read() error = %v, expected a problem with %s", err, ticket.FieldExternalURL)
	}
}

func TestCreateWritesExternalFields(t *testing.T) {
	tests := []struct {
		name         string
		fields       string
		expectedText string
	}{
		{"Unmapped, they go in properties named after the fields", "",
			"---\ntype: task\nstatus: todo\nexternal_id: INO-343\nexternal_url: " + externalURL + "\n---\n\n# New\n"},
		{"Mapped, they go in the properties the profile names", mappedExternal,
			"---\ntype: task\nstatus: todo\ntracker_id: INO-343\ntracker_link: " + externalURL + "\n---\n\n# New\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := externalVault(t, tt.fields, smallNote)
			value := ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, ExternalID: externalID, ExternalURL: externalURL}

			slug, err := store.Create(value)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if text := vaultFiles(t, root)["tasks/new.md"]; text != tt.expectedText {
				t.Errorf("tasks/new.md = %q, expected %q", text, tt.expectedText)
			}
			read, err := store.Read(slug)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if read.ExternalID != externalID || read.ExternalURL != externalURL {
				t.Errorf("Read() external = %q, %q, expected %q, %q", read.ExternalID, read.ExternalURL, externalID, externalURL)
			}
		})
	}
}

func TestCreateRejectsAnExternalURLThatIsNotAWebURL(t *testing.T) {
	store, root := externalVault(t, "", smallNote)
	value := ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, ExternalURL: "INO-343"}

	_, err := store.Create(value)

	if !errors.Is(err, validation.ErrInvalid) {
		t.Errorf("Create() error = %v, expected %v", err, validation.ErrInvalid)
	}
	if _, written := vaultFiles(t, root)["tasks/new.md"]; written {
		t.Errorf("Create() wrote tasks/new.md for an invalid ticket")
	}
}

func TestApplyExternalFields(t *testing.T) {
	unmappedNote := "---\ntype: task\nstatus: todo\nexternal_id: INO-343\nexternal_url: " + externalURL + "\n---\n\n# Existing\n"
	mappedNote := "---\ntype: task\nstatus: todo\ntracker_id: INO-343\ntracker_link: " + externalURL + "\n---\n\n# Existing\n"
	tests := []struct {
		name          string
		fields        string
		note          string
		changes       []backend.Change
		expectedText  string
		expectedError error
	}{
		{"Setting them unmapped writes the properties named after the fields", "", smallNote,
			[]backend.Change{
				backend.SetField{Field: ticket.FieldExternalID, Value: externalID},
				backend.SetField{Field: ticket.FieldExternalURL, Value: externalURL},
			}, unmappedNote, nil},
		{"Setting them mapped writes the properties the profile names", mappedExternal, smallNote,
			[]backend.Change{
				backend.SetField{Field: ticket.FieldExternalID, Value: externalID},
				backend.SetField{Field: ticket.FieldExternalURL, Value: externalURL},
			}, mappedNote, nil},
		{"Removing them unmapped removes the properties named after the fields", "", unmappedNote,
			[]backend.Change{backend.RemoveField{Field: ticket.FieldExternalID}, backend.RemoveField{Field: ticket.FieldExternalURL}},
			smallNote, nil},
		{"Removing them mapped removes the properties the profile names", mappedExternal, mappedNote,
			[]backend.Change{backend.RemoveField{Field: ticket.FieldExternalID}, backend.RemoveField{Field: ticket.FieldExternalURL}},
			smallNote, nil},
		{"Removing one the note lacks is named", "", smallNote,
			[]backend.Change{backend.RemoveField{Field: ticket.FieldExternalID}}, smallNote, backend.ErrNoField},
		{"Setting a URL that is not a web URL is rejected", "", smallNote,
			[]backend.Change{backend.SetField{Field: ticket.FieldExternalURL, Value: "INO-343"}}, smallNote, validation.ErrInvalid},
		{"Setting a mapped property by its own name is unsupported", mappedExternal, smallNote,
			[]backend.Change{backend.SetField{Field: "tracker_id", Value: externalID}}, smallNote, backend.ErrUnsupported},
		{"Setting one whose default property is mapped to another field is unsupported", "      fields: {external_id: parent}\n", smallNote,
			[]backend.Change{backend.SetField{Field: ticket.FieldExternalID, Value: externalID}}, smallNote, backend.ErrUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := externalVault(t, tt.fields, tt.note)

			err := store.Apply("existing", tt.changes)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Apply() error = %v, expected %v", err, tt.expectedError)
			}
			if text := vaultFiles(t, root)["tasks/existing.md"]; text != tt.expectedText {
				t.Errorf("tasks/existing.md = %q, expected %q", text, tt.expectedText)
			}
		})
	}
}

func TestLintDoesNotReadExternalFieldsAsLinks(t *testing.T) {
	store, _ := externalVault(t, mappedExternal,
		"---\ntype: task\nstatus: todo\ntracker_id: INO-343\ntracker_link: "+externalURL+"\n---\n\n# Existing\n")

	notes, err := store.(backend.LintReader).LintNotes()
	if err != nil {
		t.Fatalf("LintNotes() error = %v", err)
	}
	for _, note := range notes {
		for _, link := range note.Links {
			if link == externalID || link == externalURL {
				t.Errorf("LintNotes() reads %q as a link", link)
			}
		}
	}
}
