package versioncommand

import (
	"context"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/command"
)

type versionFactory struct {
	command.BaseFactory
}

func (versionFactory) Name() string {
	return "version"
}

func (versionFactory) Description() string {
	return "Print program version information."
}

func (versionFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	options := getopt.New()
	options.SetParameters("")

	return options, func(ctx context.Context) int {
		logger := zerolog.Ctx(ctx)
		_, err := dispatcher.PrintVersion()
		if err != nil {
			logger.Error().Err(err).Send()
			return 1
		}
		return 0
	}
}

var Factory command.Factory = versionFactory{}
