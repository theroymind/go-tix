package tix_test

import (
	"strings"
	"testing"

	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/railyard-cli"
)

func TestHelp(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		mentions         []string
		absent           []string
	}{
		{"-h alone prints the full usage", []string{"-h"}, cli.ExitCodeSuccess, []string{tix.Program.Usage}, nil},
		{"--help alone prints the full usage", []string{"--help"}, cli.ExitCodeSuccess, []string{tix.Program.Usage}, nil},
		{"help -h prints the full usage", []string{"help", "-h"}, cli.ExitCodeSuccess, []string{tix.Program.Usage}, nil},
		{"add --help prints only add's usage", []string{"add", "--help"}, cli.ExitCodeSuccess,
			[]string{"usage:\n  add --type TYPE", "(prints the new id"}, []string{"  show REF"}},
		{"-h anywhere in a command's options prints its usage without running it", []string{"done", "nope", "-h"}, cli.ExitCodeSuccess,
			[]string{"usage:\n  done [--force] REF..."}, []string{"no ticket"}},
		{"help COMMAND prints only that command's usage", []string{"help", "tick"}, cli.ExitCodeSuccess,
			[]string{"usage:\n  tick REF (N... | --all)\n", "(N is a criterion number"}, []string{"untick"}},
		{"ac's usage holds each of its subcommands", []string{"ac", "--help"}, cli.ExitCodeSuccess,
			[]string{"  ac ls [--open] REF", "  ac set REF N TEXT", "  ac add REF TEXT"}, []string{"  tick"}},
		{"acceptance-criteria shares ac's usage", []string{"acceptance-criteria", "-h"}, cli.ExitCodeSuccess,
			[]string{"  ac ls [--open] REF", "  ac add REF TEXT"}, nil},
		{"list shares ls's usage", []string{"list", "-h"}, cli.ExitCodeSuccess,
			[]string{"usage:\n  ls [--status S]", "list is an alias"}, nil},
		{"help for an unknown command says it is unknown", []string{"help", "shwo"}, cli.ExitCodeUsage,
			[]string{`unknown command "shwo"`}, nil},
		{"--help after -- is an argument, not a request for help", []string{"tick", "x", "--", "--help"}, cli.ExitCodeUsage,
			[]string{`"--help" is not a criterion number`}, nil},
		{"A usage error shows the command's usage, not the full one", []string{"tick", "x"}, cli.ExitCodeUsage,
			[]string{"criterion number", "usage:\n  tick REF"}, []string{"  show REF", "commands:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, t.TempDir(), tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			for _, absent := range tt.absent {
				if strings.Contains(output, absent) {
					t.Errorf("output %q mentions %q", output, absent)
				}
			}
		})
	}
}
