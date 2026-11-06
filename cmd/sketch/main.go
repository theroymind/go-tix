// Command sketch roughs out a feature's files, types and functions as a sketch.yml beside the feature's notes,
// one short command at a time.
//
// Run sketch -C FEATURE/TASK/FILE fn NAME to add an item, sketch -C FEATURE/TASK/FILE:ITEM in TEXT to add a line
// to it, and sketch show FEATURE to print the sketch, with --json for a program or --md for a reader.
package main

import (
	"os"

	"github.com/theroymind/go-tix/internal/sketch/command"
)

func main() {
	os.Exit(int(command.Program.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
