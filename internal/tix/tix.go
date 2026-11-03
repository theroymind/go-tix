// Package tix provides the tix commands, which create, list, show, change and complete the tickets in a .tix
// directory, and list, show and change the docs a backend holds beside them.
package tix

import "github.com/theroymind/railyard/internal/cli"

const usage = `usage: tix <command> [arguments]

commands:
  init --prefix PREFIX
  add --type TYPE [--parent REF] [--priority N] [--tag T]... [--dependency REF]... [--acceptance A]... [--description D] [--draft] TITLE
  show REF [--json]      (a ticket, or a doc when no ticket matches)
  ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]
  ready [--type T] [--json]
  done [--force] REF
  set REF KEY=VALUE...   (raw write to a ticket or doc; does not check criteria, use done)
  tick REF N...          (N is a criterion number from show)
  untick REF N...
  ac set REF N TEXT
  ac add REF TEXT
  docs ls [--type T] [--status S] [--json]
  doc show REF [--json]
  doc add --type T TITLE
  backlinks REF [--json]
  supersede OLD NEW      (marks the OLD doc superseded by NEW)
  help
`

var Program = cli.Program{
	Name:  "tix",
	Usage: usage,
	Commands: map[string]cli.Command{
		"init":      runInit,
		"add":       runAdd,
		"show":      runShow,
		"ls":        runLs,
		"ready":     runReady,
		"done":      runDone,
		"set":       runSet,
		"tick":      runTick,
		"untick":    runUntick,
		"ac":        runAc,
		"docs":      runDocs,
		"doc":       runDoc,
		"backlinks": runBacklinks,
		"supersede": runSupersede,
	},
}
