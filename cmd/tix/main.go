// Command tix tracks tickets as YAML files in a .tix directory committed with the project.
//
// Run tix init --prefix PREFIX once, then tix add, tix show and tix done.
package main

import (
	"os"

	"github.com/theroymind/railyard/internal/tix"
)

func main() {
	os.Exit(int(tix.Program.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
