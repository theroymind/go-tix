package tix

import (
	"io"
	"slices"
	"strings"

	"github.com/theroymind/railyard-cli"
)

// Run runs tix on the arguments, as Program.Run does, and answers a request for help first: -h or --help before the
// command, or help alone, prints the full usage; -h or --help among a command's arguments, before any --, or help
// COMMAND prints only that command's usage. Each succeeds without running the command. A usage error shows the
// command's usage, not the full one.
func Run(arguments []string, stdin io.Reader, stdout, stderr io.Writer) cli.ExitCode {
	if len(arguments) > 0 && isHelpFlag(arguments[0]) {
		_, _ = io.WriteString(stdout, usage)
		return cli.ExitCodeSuccess
	}
	if commandName, wantsHelp := helpTopic(arguments); wantsHelp {
		if isHelpFlag(commandName) {
			_, _ = io.WriteString(stdout, usage)
			return cli.ExitCodeSuccess
		}
		text, isKnown := commandUsage(commandName)
		if !isKnown {
			// Program says the command is unknown, pointing at help.
			return Program.Run([]string{commandName}, stdin, stdout, stderr)
		}
		_, _ = io.WriteString(stdout, text)
		return cli.ExitCodeSuccess
	}
	return runWithCommandUsage(arguments, stdin, stdout, stderr)
}

// runWithCommandUsage runs Program, and on a usage error follows the error with the command's own usage in place of
// the full one, so a mistyped command shows only the lines that fix it.
func runWithCommandUsage(arguments []string, stdin io.Reader, stdout, stderr io.Writer) cli.ExitCode {
	var message strings.Builder
	exitCode := Program.Run(arguments, stdin, stdout, &message)
	text := message.String()
	if exitCode == cli.ExitCodeUsage && len(arguments) > 0 && strings.HasSuffix(text, usage) {
		if commandText, isKnown := commandUsage(arguments[0]); isKnown {
			text = strings.TrimSuffix(text, usage) + commandText
		}
	}
	_, _ = io.WriteString(stderr, text)
	return exitCode
}

func isHelpFlag(argument string) bool {
	return argument == helpShortFlag || argument == helpLongFlag
}

// helpTopic returns the command whose usage the arguments ask for: COMMAND in help COMMAND, or the command whose
// arguments hold -h or --help before a --.
func helpTopic(arguments []string) (string, bool) {
	if len(arguments) < 2 {
		return "", false
	}
	if arguments[0] == helpCommand {
		return arguments[1], true
	}
	options := arguments[1:]
	if end := slices.Index(options, longOptionStart); end >= 0 {
		options = options[:end]
	}
	return arguments[0], slices.ContainsFunc(options, isHelpFlag)
}

// commandUsage returns the usage lines for one command, each entry with its wrapped lines, after the usage heading.
// An alias takes the lines of the command it stands for.
func commandUsage(commandName string) (string, bool) {
	if canonical, isAlias := commandAliases[commandName]; isAlias {
		commandName = canonical
	}
	var entries []string
	isInEntry := false
	for line := range strings.Lines(usage) {
		isEntryStart := strings.HasPrefix(line, usageIndent) && !strings.HasPrefix(line, usageIndent+" ")
		if isEntryStart {
			fields := strings.Fields(line)
			isInEntry = len(fields) > 0 && fields[0] == commandName
		} else if !strings.HasPrefix(line, usageIndent) {
			isInEntry = false
		}
		if isInEntry {
			entries = append(entries, line)
		}
	}
	if len(entries) == 0 {
		return "", false
	}
	return usageHeading + strings.Join(entries, ""), true
}
