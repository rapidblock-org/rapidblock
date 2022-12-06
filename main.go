package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/commands/applycommand"
	"github.com/chronos-tachyon/rapidblock/commands/command"
	"github.com/chronos-tachyon/rapidblock/commands/exportcommand"
	"github.com/chronos-tachyon/rapidblock/commands/keygencommand"
	"github.com/chronos-tachyon/rapidblock/commands/preparecommand"
	"github.com/chronos-tachyon/rapidblock/commands/registercommand"
	"github.com/chronos-tachyon/rapidblock/commands/signcommand"
	"github.com/chronos-tachyon/rapidblock/commands/verifycommand"
	"github.com/chronos-tachyon/rapidblock/internal/appversion"

	_ "github.com/chronos-tachyon/rapidblock/mastodon"
)

var (
	Version    = "devel"
	Commit     = ""
	CommitDate = ""
	TreeState  = ""
)

var (
	ProgramName string
	CommandList []command.Command
	CommandMap  map[string]command.Command
)

func main() {
	appversion.Version = Version
	appversion.Commit = Commit
	appversion.CommitDate = CommitDate
	appversion.TreeState = TreeState

	ProgramName = "rapidblock"
	osArgs := os.Args
	n := len(osArgs)
	switch {
	case n <= 0:
		osArgs = make([]string, 1, 1)
		osArgs[0] = ProgramName
	default:
		ProgramName = filepath.Base(osArgs[0])
	}

	var (
		flagHelp       bool
		flagVersion    bool
		hasCommandName bool
		commandName    string
	)
	argv := make([]string, 0, n)
	argv = append(argv, ProgramName)

	for _, arg := range osArgs[1:] {
		switch {
		case arg == "-h" || arg == "--help":
			flagHelp = true
		case arg == "-V" || arg == "--version":
			flagVersion = true
		case hasCommandName:
			argv = append(argv, arg)
		case arg != "" && arg[0] == '-':
			argv = append(argv, arg)
		default:
			commandName = arg
			hasCommandName = true
		}
	}

	if flagVersion {
		hasCommandName = true
		commandName = "version"
		argv = argv[:1]
	}

	if flagHelp && !hasCommandName {
		printHelp(os.Stdout)
		return
	}

	if commandName == "" {
		printHelp(os.Stderr)
		os.Exit(1)
		return
	}

	c := LookupCommand(commandName)
	if flagHelp {
		printHelpForCommand(os.Stdout, commandName, c)
		return
	}
	argv[0] = fmt.Sprint(ProgramName, " ", commandName)
	c.Options.Parse(argv)
	os.Exit(c.Main())
}

func LookupCommand(commandName string) command.Command {
	c, found := CommandMap[commandName]
	if !found {
		fmt.Fprintf(os.Stderr, "fatal: unknown subcommand %q\n", commandName)
		os.Exit(1)
	}
	return c
}

var HelpFactory command.FactoryFunc = func() command.Command {
	options := getopt.New()
	options.SetParameters("[<subcommand>]")

	return command.Command{
		Name:        "help",
		Description: "Print program usage information.",
		Options:     options,
		Main: func() int {
			n := options.NArgs()
			switch {
			case n <= 0:
				printHelp(os.Stdout)
				return 0
			case n == 1:
				commandName := options.Arg(0)
				c := LookupCommand(commandName)
				printHelpForCommand(os.Stdout, commandName, c)
				return 0
			default:
				fmt.Fprintf(os.Stderr, "fatal: found %d unexpected positional arguments\n", n-1)
				return 1
			}
		},
	}
}

var VersionFactory command.FactoryFunc = func() command.Command {
	options := getopt.New()
	options.SetParameters("")

	return command.Command{
		Name:        "version",
		Description: "Print program version information.",
		Options:     options,
		Main: func() int {
			appversion.Print(os.Stdout)
			return 0
		},
	}
}

func printHelp(w io.Writer) (int, error) {
	var buf bytes.Buffer
	buf.Grow(256)

	var maxWidth uint
	for _, item := range CommandList {
		nameLen := uint(len(item.Name))
		if maxWidth < nameLen {
			maxWidth = nameLen
		}
	}
	maxWidth += (maxWidth & 1)

	buf.WriteString("Tool for managing syndicated domain blocks on Fediverse instances.\n")
	buf.WriteString("Usage: ")
	buf.WriteString(ProgramName)
	buf.WriteString(" <subcommand> [<flags>]\n")
	buf.WriteString("\n")
	buf.WriteString("Subcommands:\n")
	for _, item := range CommandList {
		name := item.Name
		desc := item.Description
		if desc == "" {
			desc = "(no description)"
		}

		buf.WriteString("  ")
		n := uint(len(name))
		for n < maxWidth {
			buf.WriteByte(' ')
			n++
		}
		buf.WriteString(name)
		buf.WriteString("  ")
		buf.WriteString(desc)
		buf.WriteByte('\n')
	}
	buf.WriteString("\n")
	buf.WriteString("Use \"rapidblock <subcommand> --help\" for help on individual subcommands.\n")
	return w.Write(buf.Bytes())
}

func printHelpForCommand(w io.Writer, commandName string, c command.Command) {
	c.Options.SetProgram(fmt.Sprint(ProgramName, " ", commandName))
	if c.Description != "" {
		fmt.Fprintf(w, "%s\n", c.Description)
	}
	c.Options.PrintUsage(w)
}

func init() {
	CommandList = []command.Command{
		HelpFactory.New(),
		VersionFactory.New(),
		keygencommand.Factory.New(),
		preparecommand.Factory.New(),
		exportcommand.Factory.New(),
		signcommand.Factory.New(),
		verifycommand.Factory.New(),
		registercommand.Factory.New(),
		applycommand.Factory.New(),
	}

	CommandMap = make(map[string]command.Command, len(CommandList))
	for _, item := range CommandList {
		CommandMap[item.Name] = item
		for _, alias := range item.Aliases {
			CommandMap[alias] = item
		}
	}
}
