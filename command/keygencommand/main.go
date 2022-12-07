package keygencommand

import (
	"context"
	"crypto/ed25519"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

type keygenFactory struct {
	command.BaseFactory
}

func (keygenFactory) Name() string {
	return "keygen"
}

func (keygenFactory) Aliases() []string {
	return []string{"genkey", "generate-key"}
}

func (keygenFactory) Description() string {
	return "Generates an Ed25519 cryptographic key pair for signing files."
}

func (keygenFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		publicKeyFile  string
		privateKeyFile string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&publicKeyFile, "public-key-file", 'p', "path to the public key file to create")
	options.FlagLong(&privateKeyFile, "private-key-file", 'k', "path to the private key file to create")

	return options, func(ctx context.Context) int {
		if publicKeyFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -p / --public-key-file")
			return 1
		}
		if privateKeyFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -k / --private-key-file")
			return 1
		}
		return Main(ctx, publicKeyFile, privateKeyFile)
	}
}

var Factory command.Factory = keygenFactory{}

func Main(ctx context.Context, publicKeyFile string, privateKeyFile string) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("publicKeyFile", publicKeyFile).
		Str("privateKeyFile", privateKeyFile).
		Logger()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate key")
		return 1
	}

	seed := privKey.Seed()
	if err := iohelpers.WriteBase64File(publicKeyFile, pubKey[:], false); err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	if err := iohelpers.WriteBase64File(privateKeyFile, seed[:], true); err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	return 0
}
