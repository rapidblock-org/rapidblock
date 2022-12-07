package applycommand

import (
	"context"
	"io"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/blockapply"
	"github.com/chronos-tachyon/rapidblock/blockfile"
	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

type applyFactory struct {
	command.BaseFactory
}

func (applyFactory) Name() string {
	return "apply"
}

func (applyFactory) Description() string {
	return "Applies the given RapidBlock blocklist file to configured instances."
}

func (applyFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		configFile string
		dataFile   string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&configFile, "config-file", 'c', "JSON/YAML config that lists the Fediverse servers to connect to")
	options.FlagLong(&dataFile, "data-file", 'd', "path to the JSON file to apply")

	return options, func(ctx context.Context) int {
		if configFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -c / --config-file")
			return 1
		}
		if dataFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -d / --data-file")
			return 1
		}
		return Main(ctx, configFile, dataFile, dispatcher.Stdout())
	}
}

var Factory command.Factory = applyFactory{}

func Main(ctx context.Context, configFile string, dataFile string, stdout io.Writer) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("configFile", configFile).
		Str("dataFile", dataFile).
		Logger()

	var config blockapply.Config
	err := iohelpers.Load(&config, configFile, true)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	var file blockfile.BlockFile
	err = iohelpers.Load(&file, dataFile, true)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	for _, server := range config.Servers {
		fn := server.Mode.Func()
		stats, err := fn(ctx, server, file)
		if err != nil {
			logger.Error().Str("serverName", server.Name).Err(err).Send()
			return 1
		}
		_, _ = stats.WriteTo(server.Name, stdout)
	}
	return 0
}
