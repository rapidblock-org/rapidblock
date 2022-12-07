package helpcommand

import (
	"context"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/command"
)

type helpFactory struct {
	command.BaseFactory
}

func (helpFactory) Name() string {
	return "help"
}

func (helpFactory) Description() string {
	return "Print program usage information."
}

func (helpFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	options := getopt.New()
	options.SetParameters("[<subcommand>]")

	return options, func(ctx context.Context) int {
		logger := zerolog.Ctx(ctx)

		n := options.NArgs()
		switch {
		case n <= 0:
			_, err := dispatcher.PrintHelp(false)
			if err != nil {
				logger.Error().Err(err).Send()
				return 1
			}
			return 0

		case n == 1:
			commandName := options.Arg(0)
			factory, found := dispatcher.Lookup(commandName)
			if !found {
				logger.Error().
					Str("commandName", commandName).
					Msg("unknown subcommand")
				return 1
			}
			_, err := dispatcher.PrintHelpForCommand(false, commandName, factory)
			if err != nil {
				logger.Error().Err(err).Send()
				return 1
			}
			return 0

		default:
			logger.Error().
				Msgf("found %d unexpected positional arguments\n", n-1)
			return 1
		}
	}
}

var Factory command.Factory = helpFactory{}
