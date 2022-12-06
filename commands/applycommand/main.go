package applycommand

import (
	"context"
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/blockapply"
	"github.com/chronos-tachyon/rapidblock/blockfile"
	"github.com/chronos-tachyon/rapidblock/commands/command"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

var Factory command.FactoryFunc = func() command.Command {
	var (
		configFile string
		dataFile   string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&configFile, "config-file", 'c', "JSON/YAML config that lists the Fediverse servers to connect to")
	options.FlagLong(&dataFile, "data-file", 'd', "path to the JSON file to apply")

	return command.Command{
		Name:        "apply",
		Description: "Applies the given RapidBlock blocklist file to configured instances.",
		Options:     options,
		Main: func() int {
			if configFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -c / --config-file\n")
				return 1
			}
			if dataFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
				return 1
			}
			return Main(configFile, dataFile)
		},
	}
}

func Main(configFile string, dataFile string) int {
	var config blockapply.Config
	err := iohelpers.Load(&config, configFile, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	var file blockfile.BlockFile
	err = iohelpers.Load(&file, dataFile, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	ctx := context.Background()
	for _, server := range config.Servers {
		fn := server.Mode.Func()
		stats, err := fn(ctx, server, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", server.Name, err)
			return 1
		}
		stats.WriteTo(server.Name, os.Stdout)
	}
	return 0
}
