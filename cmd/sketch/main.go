// Command sketch lists the plain .sketch.md notes that plan a repo's files, each one beside the file it plans.
//
// Run sketch ls for a table of notes and their targets, sketch ls --json for the same as JSON, and sketch help for
// the note format.
package main

import (
	"os"

	"github.com/theroymind/go-tix/internal/sketch/command"
)

func main() {
	os.Exit(int(command.Program.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
