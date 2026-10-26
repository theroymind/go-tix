package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

var expectedID = testPrefix + ticket.IDSeparator + ticket.IDCharacters[len(ticket.IDCharacters)-ticket.IDSuffixLength:]

func openTestStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	tixDirectory := filepath.Join(root, ticket.DirectoryName)
	makeDirectory(t, tixDirectory)
	if err := WriteConfig(tixDirectory, Config{Prefix: testPrefix}); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store
}

func TestReadTicketFileReadsATicketWhoseIDMatchesItsFileName(t *testing.T) {
	store := openTestStore(t)
	expected := parseFixture(t)
	expected.ID = expectedID
	writeFile(t, store.TicketPath(expectedID), serialize(t, expected))

	file, err := store.ReadTicketFile(expectedID)
	if err != nil {
		t.Fatalf("ReadTicketFile() error = %v", err)
	}
	if !reflect.DeepEqual(file.Ticket, expected) {
		t.Errorf("ReadTicketFile() = %+v, want %+v", file.Ticket, expected)
	}
}

func TestStoreRejectsAFileWhoseIDDiffersFromItsFileName(t *testing.T) {
	store := openTestStore(t)
	fixture := readFixture(t)
	path := store.TicketPath(expectedID)
	writeFile(t, path, fixture)
	mismatch := idMismatch(path, parseFixture(t).ID, expectedID)

	tests := []struct {
		name      string
		operation func() error
	}{
		{"Reading", func() error {
			_, err := store.ReadTicketFile(expectedID)
			return err
		}},
		{"Updating", func() error {
			return store.UpdateTicket(expectedID, []Change{{Field: ticket.FieldStatus, Value: ticket.StatusCompleted}})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if !errors.Is(err, ErrIDMismatch) || err.Error() != mismatch.Error() {
				t.Errorf("error = %v, want %v", err, mismatch)
			}
			if got := readFile(t, path); !bytes.Equal(got, fixture) {
				t.Errorf("file changed to\n%s", got)
			}
		})
	}
}

func TestReadTicketsReadsEveryTicketFile(t *testing.T) {
	tests := []struct {
		name       string
		isInvalid  bool
		expectedOK bool
	}{
		{"Every ticket is read, and the config is skipped", false, true},
		{"An invalid ticket file fails the read", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTestStore(t)
			expected := parseFixture(t)
			expected.ID = expectedID
			if err := store.CreateTicket(expected); err != nil {
				t.Fatalf("CreateTicket() error = %v", err)
			}
			if tt.isInvalid {
				writeFile(t, store.TicketPath(testPrefix+ticket.IDSeparator+"0000"), []byte("title: [\n"))
			}

			tickets, err := store.ReadTickets()

			if (err == nil) != tt.expectedOK {
				t.Fatalf("ReadTickets() error = %v, want success %v", err, tt.expectedOK)
			}
			if tt.expectedOK && !reflect.DeepEqual(tickets, []ticket.Ticket{expected}) {
				t.Errorf("ReadTickets() = %+v, want %+v", tickets, []ticket.Ticket{expected})
			}
		})
	}
}
