package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chronos-tachyon/rapidblock/internal/appversion"
	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"
)

type MainFunc func(context.Context) int

type Factory interface {
	New(Dispatcher) (*getopt.Set, MainFunc)
	Name() string
	Aliases() []string
	Description() string
}

type BaseFactory struct{}

func (BaseFactory) Aliases() []string {
	return nil
}

func (BaseFactory) Description() string {
	return ""
}

type Dispatcher struct {
	progName string
	list     []Factory
	byName   map[string]Factory
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func MakeDispatcher(progName string, list ...Factory) Dispatcher {
	listLen := uint(len(list))

	var dispatcher Dispatcher
	dispatcher.progName = progName
	dispatcher.list = make([]Factory, listLen)
	copy(dispatcher.list, list)
	dispatcher.byName = make(map[string]Factory, listLen)
	for _, factory := range dispatcher.list {
		dispatcher.byName[factory.Name()] = factory
		for _, alias := range factory.Aliases() {
			dispatcher.byName[alias] = factory
		}
	}
	dispatcher.stdin = os.Stdin
	dispatcher.stdout = os.Stdout
	dispatcher.stderr = os.Stderr
	return dispatcher
}

func (dispatcher Dispatcher) WithStdio(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) Dispatcher {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	dispatcher.stdin = stdin
	dispatcher.stdout = stdout
	dispatcher.stderr = stderr
	return dispatcher
}

func (dispatcher Dispatcher) Stdin() io.Reader {
	return dispatcher.stdin
}

func (dispatcher Dispatcher) Stdout() io.Writer {
	return dispatcher.stdout
}

func (dispatcher Dispatcher) Stderr() io.Writer {
	return dispatcher.stderr
}

func (dispatcher Dispatcher) NumCommands() uint {
	return uint(len(dispatcher.list))
}

func (dispatcher Dispatcher) Command(index uint) Factory {
	return dispatcher.list[index]
}

func (dispatcher Dispatcher) Lookup(commandName string) (Factory, bool) {
	factory, found := dispatcher.byName[commandName]
	return factory, found
}

func (dispatcher Dispatcher) Commands() []Factory {
	listLen := dispatcher.NumCommands()
	list := make([]Factory, listLen)
	copy(list, dispatcher.list)
	return list
}

func (dispatcher Dispatcher) Main(ctx context.Context, args []string) int {
	logger := zerolog.Ctx(ctx)

	n := len(args)
	switch {
	case n <= 0:
		args = make([]string, 1)
		args[0] = dispatcher.progName
	default:
		dispatcher.progName = filepath.Base(args[0])
	}

	var (
		wantHelp       bool
		wantVersion    bool
		hasCommandName bool
		commandName    string
		argv           []string
	)

	argv = make([]string, 0, n)
	argv = append(argv, dispatcher.progName)
	for _, arg := range args[1:] {
		switch {
		case arg == "-h" || arg == "--help":
			wantHelp = true
		case arg == "-V" || arg == "--version":
			wantVersion = true
		case hasCommandName:
			argv = append(argv, arg)
		case arg != "" && arg[0] == '-':
			argv = append(argv, arg)
		default:
			commandName = arg
			hasCommandName = true
		}
	}

	if wantVersion {
		hasCommandName = true
		commandName = "version"
		argv = argv[:1]
	}

	if wantHelp && !hasCommandName {
		_, err := dispatcher.PrintHelp(false)
		if err != nil {
			logger.Error().Err(err).Send()
			return 1
		}
		return 0
	}

	if commandName == "" {
		_, err := dispatcher.PrintHelp(true)
		if err != nil {
			logger.Error().Err(err).Send()
		}
		return 1
	}

	factory, found := dispatcher.Lookup(commandName)
	if !found {
		logger.Error().
			Str("commandName", commandName).
			Msg("unknown subcommand")
		return 1
	}
	if wantHelp {
		_, err := dispatcher.PrintHelpForCommand(false, commandName, factory)
		if err != nil {
			logger.Error().Err(err).Send()
			return 1
		}
		return 0
	}

	argv[0] = fmt.Sprint(dispatcher.progName, " ", commandName)
	opts, fn := factory.New(dispatcher)
	opts.Parse(argv)
	return fn(ctx)
}

func (dispatcher Dispatcher) PrintHelp(isError bool) (int, error) {
	var buf bytes.Buffer
	buf.Grow(256)

	var maxWidth uint
	for _, factory := range dispatcher.list {
		nameLen := uint(len(factory.Name()))
		if maxWidth < nameLen {
			maxWidth = nameLen
		}
	}
	maxWidth += (maxWidth & 1)

	buf.WriteString("Tool for managing syndicated domain blocks on Fediverse instances.\n")
	buf.WriteString("Usage: ")
	buf.WriteString(dispatcher.progName)
	buf.WriteString(" <subcommand> [<flags>]\n")
	buf.WriteString("\n")
	buf.WriteString("Subcommands:\n")
	for _, factory := range dispatcher.list {
		name := factory.Name()
		desc := factory.Description()
		if desc == "" {
			desc = "(no description provided)"
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

	w := dispatcher.Stdout()
	if isError {
		w = dispatcher.Stderr()
	}
	return w.Write(buf.Bytes())
}

func (dispatcher Dispatcher) PrintHelpForCommand(isError bool, commandName string, factory Factory) (int, error) {
	var buf bytes.Buffer
	buf.Grow(1 << 10) // 1 KiB
	desc := factory.Description()
	if desc != "" {
		buf.WriteString(desc)
		buf.WriteByte('\n')
	}
	opts, _ := factory.New(dispatcher)
	opts.SetProgram(fmt.Sprint(dispatcher.progName, " ", commandName))
	opts.PrintUsage(&buf)

	w := dispatcher.Stdout()
	if isError {
		w = dispatcher.Stderr()
	}
	return w.Write(buf.Bytes())
}

func (dispatcher Dispatcher) PrintVersion() (int, error) {
	return appversion.Print(dispatcher.Stdout())
}
