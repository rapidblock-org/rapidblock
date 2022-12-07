package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/command/applycommand"
	"github.com/chronos-tachyon/rapidblock/command/exportcommand"
	"github.com/chronos-tachyon/rapidblock/command/helpcommand"
	"github.com/chronos-tachyon/rapidblock/command/keygencommand"
	"github.com/chronos-tachyon/rapidblock/command/preparecommand"
	"github.com/chronos-tachyon/rapidblock/command/registercommand"
	"github.com/chronos-tachyon/rapidblock/command/signcommand"
	"github.com/chronos-tachyon/rapidblock/command/verifycommand"
	"github.com/chronos-tachyon/rapidblock/command/versioncommand"
	"github.com/chronos-tachyon/rapidblock/internal/appversion"

	_ "github.com/chronos-tachyon/rapidblock/mastodon"
)

var (
	Version    = "devel"
	Commit     = ""
	CommitDate = ""
	TreeState  = ""
)

func main() {
	appversion.Version = Version
	appversion.Commit = Commit
	appversion.CommitDate = CommitDate
	appversion.TreeState = TreeState

	dispatcher := command.MakeDispatcher(
		"rapidblock",
		helpcommand.Factory,
		versioncommand.Factory,
		keygencommand.Factory,
		preparecommand.Factory,
		exportcommand.Factory,
		signcommand.Factory,
		verifycommand.Factory,
		registercommand.Factory,
		applycommand.Factory,
	)

	var logWriter io.Writer

	switch logOutput := os.Getenv("LOG_OUTPUT"); logOutput {
	case "":
		fallthrough
	case "stderr":
		logWriter = os.Stderr
	case "stdout":
		logWriter = os.Stdout
	default:
		f, err := os.OpenFile(logOutput, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
		if err != nil {
			panic(fmt.Errorf("failed to open log file: %q: %w", logOutput, err))
		}
		defer func() {
			_ = f.Sync()
			_ = f.Close()
		}()
		logWriter = f
	}

	switch logFormat := os.Getenv("LOG_FORMAT"); logFormat {
	case "":
		fallthrough
	case "console":
		logWriter = zerolog.ConsoleWriter{
			Out:        logWriter,
			TimeFormat: "2006-01-02T15:04:05.999Z07:00",
		}
	case "console-plain":
		logWriter = zerolog.ConsoleWriter{
			Out:        logWriter,
			TimeFormat: "2006-01-02T15:04:05.999Z07:00",
			NoColor:    true,
		}
	case "raw":
		// pass
	case "cbor":
		// pass
	case "json":
		// pass
	default:
		panic(fmt.Errorf("unknown log format %q, must be one of \"console\" or \"json\"", logFormat))
	}

	switch logLevel := os.Getenv("LOG_LEVEL"); logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "":
		fallthrough
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		fallthrough
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "err":
		fallthrough
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		panic(fmt.Errorf("unknown log level %q, must be one of \"debug\", \"info\", \"warn\", \"error\"", logLevel))
	}

	log.Logger = zerolog.New(logWriter).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.DurationFieldUnit = time.Second
	zerolog.DurationFieldInteger = false
	zerolog.DefaultContextLogger = &log.Logger

	ctx := context.Background()
	os.Exit(dispatcher.Main(ctx, os.Args))
}
