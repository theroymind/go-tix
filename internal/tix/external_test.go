package tix_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/railyard-cli"
)

const (
	cliExternalID  = "INO-343"
	cliExternalURL = "https://tracker.example.com/issue/INO-343"
)

type externalBackend struct {
	name     string
	setup    func(t *testing.T) string
	notePath func(root, id string) string
}

var externalBackends = []externalBackend{
	{"yaml",
		func(t *testing.T) string {
			root, _ := yamlRepository(t)
			return root
		},
		func(root, id string) string { return filepath.Join(root, ticket.DirectoryName, ticket.FileName(id)) }},
	{"vault",
		copyVault,
		func(root, id string) string { return filepath.Join(root, "docs", "tasks", id+".md") }},
}

func listJSON(t *testing.T, root string) []ticket.Ticket {
	t.Helper()
	exitCode, output := run(t, root, "ls", "--json")
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("ls exit code = %d, output %q", exitCode, output)
	}
	var listed []ticket.Ticket
	if err := json.Unmarshal([]byte(output), &listed); err != nil {
		t.Fatalf("ls --json printed %q: %v", output, err)
	}
	return listed
}

func TestExternalFieldsThroughTheCommands(t *testing.T) {
	for _, store := range externalBackends {
		t.Run(store.name, func(t *testing.T) {
			root := store.setup(t)

			exitCode, output := run(t, root, "add", "--type", "task", "--external-id", cliExternalID,
				"--external-url", cliExternalURL, "Tracked work")
			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("add exit code = %d, output %q", exitCode, output)
			}
			id := strings.TrimSpace(output)
			path := store.notePath(root, id)
			idLine := ticket.FieldExternalID + ": " + cliExternalID + "\n"
			urlLine := ticket.FieldExternalURL + ": " + cliExternalURL + "\n"
			if text := readFile(t, path); !strings.Contains(text, idLine+urlLine) {
				t.Errorf("%s does not hold %q:\n%s", path, idLine+urlLine, text)
			}

			exitCode, output = run(t, root, "show", id)
			if exitCode != cli.ExitCodeSuccess || !strings.Contains(output, idLine+urlLine) {
				t.Errorf("show exit code = %d, output does not hold %q:\n%s", exitCode, idLine+urlLine, output)
			}
			shown := showJSON(t, root, id)
			if shown.ExternalID != cliExternalID || shown.ExternalURL != cliExternalURL {
				t.Errorf("show --json external = %q, %q", shown.ExternalID, shown.ExternalURL)
			}
			found := false
			for _, listed := range listJSON(t, root) {
				if listed.ID == id {
					found = listed.ExternalID == cliExternalID && listed.ExternalURL == cliExternalURL
				}
			}
			if !found {
				t.Errorf("ls --json does not list %s with its external id and URL", id)
			}

			exitCode, output = run(t, root, "set", id, "external_id=INO-344", "external_url=http://tracker.example.com/1")
			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("set exit code = %d, output %q", exitCode, output)
			}
			if shown := showJSON(t, root, id); shown.ExternalID != "INO-344" || shown.ExternalURL != "http://tracker.example.com/1" {
				t.Errorf("after set, external = %q, %q", shown.ExternalID, shown.ExternalURL)
			}

			exitCode, output = run(t, root, "unset", id, "external_id", "external_url")
			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("unset exit code = %d, output %q", exitCode, output)
			}
			if text := readFile(t, path); strings.Contains(text, "external_") {
				t.Errorf("after unset, %s still holds an external field:\n%s", path, text)
			}
		})
	}
}

func TestExternalURLMustBeAWebURL(t *testing.T) {
	tests := []struct {
		name      string
		arguments func(id string) []string
	}{
		{"add refuses it", func(string) []string {
			return []string{"add", "--type", "task", "--external-url", cliExternalID, "Tracked work"}
		}},
		{"set refuses it", func(id string) []string { return []string{"set", id, "external_url=" + cliExternalID} }},
	}
	ids := map[string]string{"yaml": yamlID, "vault": "golem-guards"}
	for _, store := range externalBackends {
		for _, tt := range tests {
			t.Run(store.name+" "+tt.name, func(t *testing.T) {
				root := store.setup(t)
				before := vaultSnapshot(t, root)

				arguments := tt.arguments(ids[store.name])
				exitCode, output := run(t, root, arguments...)

				if exitCode != cli.ExitCodeFailure || !strings.Contains(output, ticket.FieldExternalURL) {
					t.Errorf("exit code = %d, output %q, expected a failure naming %s", exitCode, output, ticket.FieldExternalURL)
				}
				after := vaultSnapshot(t, root)
				if len(after) != len(before) {
					t.Errorf("files changed from %d to %d", len(before), len(after))
				}
				for path, text := range before {
					if after[path] != text {
						t.Errorf("%s changed", path)
					}
				}
			})
		}
	}
}
