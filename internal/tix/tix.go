// Package tix provides the tix commands, which create, list, show, change and complete the tickets in a .tix
// directory, and list, show and change the docs a backend holds beside them.
package tix

import "github.com/theroymind/railyard-cli"

const usage = `usage: tix <command> [arguments]

commands:
  init [--prefix PREFIX] [--backend vault|yaml] [--docs DIR]
                         (a starter vault by default; without --prefix, asks for each setting)
  add --type TYPE [--parent REF] [--priority N] [--tag T]... [--dependency REF]... [--acceptance A]... [--description D] [--external-id ID] [--external-url URL] [--draft] TITLE
                         (prints the new id)
  show REF [--json]      (the ticket's own text with its criteria numbered, or a doc when no ticket matches)
  ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]
  ready [--type T] [--json]
  done [--force] REF     (refuses while a criterion is unticked)
  set REF KEY=VALUE...   (raw write to a ticket or doc; takes the values the file holds, not the ones ls shows)
  unset REF KEY...       (removes each key; refuses a required one, and a ticket's description, its body)
  dep REF DEPENDENCY_REF...
                         (refuses one that would form a cycle, naming its path)
  undep REF DEPENDENCY_REF...
  tick REF N...          (N is a criterion number from show)
  untick REF N...
  ac set REF N TEXT
  ac add REF TEXT
  docs ls [--type T] [--status S] [--json]
  doc show REF [--json]
  doc add --type T TITLE (prints the new id)
  backlinks REF [--json]
  context REF [--depth N] [--json]
                         (a card for each note within N hops, default 2, over parents, dependencies, wikilinks
                         and backlinks)
  supersede OLD NEW      (marks the OLD doc superseded by NEW)
  lint (REF... | --all)  (checks each note's required sections, summary and links; prints PATH: RULE: MESSAGE for
                         each problem and exits 1 when there is any)
  gate (REF... | --staged)
                         (lint on the staged ticket and doc notes, or the REFs; for a pre-commit hook)
  help

A set VALUE written as a list, such as [a, b], replaces the value with that list, and [] empties it; a wikilink
such as [[slug]] stays text. A ref is an id or a unique prefix of one. A TYPE is epic, story or task, or the project's own name for one, such
as phase. The table output is the one to read; --json is for a program.
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
		"unset":     runUnset,
		"dep":       runDep,
		"undep":     runUndep,
		"tick":      runTick,
		"untick":    runUntick,
		"ac":        runAc,
		"docs":      runDocs,
		"doc":       runDoc,
		"backlinks": runBacklinks,
		"context":   runContext,
		"supersede": runSupersede,
		"lint":      runLint,
		"gate":      runGate,
	},
}
