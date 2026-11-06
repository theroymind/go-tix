// Package starter provides the vault tix init sets up by default: its profile and the note templates that profile
// names.
package starter

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

//go:embed templates/*.md
var templates embed.FS

const (
	templateDirectory = "templates"
	featureNoteType   = "feature"
	taskNoteType      = "task"
	ideaNoteType      = "idea"
	taskDirectory     = "tasks"
	ideaDirectory     = "ideas"
	filesHeading      = "Files"
	noteStatusTodo    = "todo"
	noteStatusActive  = "active"
	noteStatusDone    = "done"
	noteStatusOpen    = "open"
	propertyDependsOn = "depends_on"
	propertyFeature   = "feature"
	templateExtension = ".md"
)

// ErrTemplateExists reports a template file already in the vault, which tix init won't overwrite.
var ErrTemplateExists = errors.New("already exists")

// Profile returns the default vault profile rooted at root: features and tasks as tickets, ideas as docs, each
// started from its template, with tasks and ideas in their own directories.
func Profile(root string) backend.Profile {
	statuses := map[string]ticket.Status{
		noteStatusTodo:   ticket.StatusOpen,
		noteStatusActive: ticket.StatusInProgress,
		noteStatusDone:   ticket.StatusCompleted,
	}
	feature := backend.TicketMapping{TicketType: ticket.TypeEpic, Statuses: statuses}
	task := backend.TicketMapping{
		TicketType: ticket.TypeTask,
		Statuses:   statuses,
		Fields:     map[string]string{propertyDependsOn: ticket.FieldDependencies, propertyFeature: ticket.FieldParent},
	}
	idea := backend.DocMapping{
		Statuses:  []string{noteStatusOpen},
		Template:  templatePath(ideaNoteType),
		Directory: ideaDirectory,
	}
	return backend.Profile{
		Root:            root,
		TypeProperty:    backend.DefaultTypeProperty,
		Tickets:         map[string]backend.TicketMapping{featureNoteType: feature, taskNoteType: task},
		Docs:            map[string]backend.DocMapping{ideaNoteType: idea},
		CriteriaHeading: backend.DefaultCriteriaHeading,
		TicketTemplates: map[string]string{
			featureNoteType: templatePath(featureNoteType),
			taskNoteType:    templatePath(taskNoteType),
		},
		TicketDirectories: map[string]string{taskNoteType: taskDirectory},
		FilesHeading:      filesHeading,
		RequiredSections: map[string][]string{
			featureNoteType: {backend.DefaultDescriptionHeading, backend.DefaultCriteriaHeading},
			taskNoteType:    {backend.DefaultDescriptionHeading, filesHeading, backend.DefaultCriteriaHeading},
		},
	}
}

// WriteTemplates writes every template the default profile names under vaultDirectory, refusing before writing
// anything when one is already there.
func WriteTemplates(vaultDirectory string) error {
	entries, err := templates.ReadDir(templateDirectory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(vaultDirectory, templateDirectory, entry.Name())
		if _, err := os.Stat(path); err == nil {
			return &fs.PathError{Op: "write template", Path: path, Err: ErrTemplateExists}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(vaultDirectory, templateDirectory), backend.DirectoryPermission); err != nil {
		return err
	}
	for _, entry := range entries {
		contents, err := templates.ReadFile(templateDirectory + "/" + entry.Name())
		if err != nil {
			return err
		}
		path := filepath.Join(vaultDirectory, templateDirectory, entry.Name())
		if err := os.WriteFile(path, contents, backend.FilePermission); err != nil {
			return err
		}
	}
	return nil
}

func templatePath(noteType string) string {
	return templateDirectory + "/" + noteType + templateExtension
}
