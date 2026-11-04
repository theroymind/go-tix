// Command tix tracks tickets as YAML files in a .tix directory committed with the project, or reads them from
// an Obsidian vault when .tix/config.yml names the vault backend and its profile.
//
// Run tix init --prefix PREFIX once, then tix add, tix show, tix ls, tix ready, tix set, tix done,
// tix tick, tix untick and tix ac. On a vault, tix docs ls, tix doc show, tix backlinks and tix supersede work on
// its decisions, ideas and specs. tix index and tix search embed the notes through the embeddings endpoint
// .tix/config.yml names and rank them against a query.
package main

import (
	"os"

	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/backend/vaultstore"
	"github.com/theroymind/go-tix/internal/tix/backend/yamlstore"
)

func main() {
	backend.Register(backend.KindYAML, yamlstore.Open)
	backend.Register(backend.KindVault, vaultstore.Open)
	os.Exit(int(tix.Program.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
