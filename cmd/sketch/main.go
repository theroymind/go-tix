// Command sketch explores a repo's tree with the plain .sketch.md notes that plan its files, each one beside the
// file it plans.
//
// Run sketch in a terminal to open the explorer, sketch ls for a table of notes and their targets, sketch ls --json
// for the same as JSON, and sketch help for the note format.
package main

import (
	"os"

	"github.com/theroymind/go-tix/internal/sketch/command"
)

func main() {
	os.Exit(int(command.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
