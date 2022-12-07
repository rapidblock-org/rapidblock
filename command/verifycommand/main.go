package verifycommand

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/checksum"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

type verifyFactory struct {
	command.BaseFactory
}

func (verifyFactory) Name() string {
	return "verify"
}

func (verifyFactory) Description() string {
	return "Verifies an Ed25519 cryptographic signature."
}

func (verifyFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		isText        bool
		publicKeyFile string
		dataFile      string
		signatureFile string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&isText, "text", 't', "perform newline canonicalization, under the assumption that --data-file is text")
	options.FlagLong(&publicKeyFile, "public-key-file", 'p', "path to the public key file to verify with")
	options.FlagLong(&dataFile, "data-file", 'd', "path to the signed data file to verify")
	options.FlagLong(&signatureFile, "signature-file", 's', "path to the detached signature file to verify")

	return options, func(ctx context.Context) int {
		if publicKeyFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -p / --public-key-file")
			return 1
		}
		if dataFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -d / --data-file")
			return 1
		}
		if signatureFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -s / --signature-file")
			return 1
		}
		return Main(ctx, isText, publicKeyFile, dataFile, signatureFile)
	}
}

var Factory command.Factory = verifyFactory{}

func Main(ctx context.Context, isText bool, publicKeyFile string, dataFile string, signatureFile string) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("publicKeyFile", publicKeyFile).
		Str("dataFile", dataFile).
		Str("signatureFile", signatureFile).
		Bool("isText", isText).
		Logger()

	raw, err := iohelpers.ReadBase64File(publicKeyFile, ed25519.PublicKeySize)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	pubKey := ed25519.PublicKey(raw)

	signature, err := iohelpers.ReadBase64File(signatureFile, ed25519.SignatureSize)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	checksum, err := checksum.File(dataFile, isText)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	if !ed25519.Verify(pubKey, checksum, signature) {
		str0 := base64.StdEncoding.EncodeToString(checksum)
		str1 := base64.StdEncoding.EncodeToString(pubKey)
		str2 := base64.StdEncoding.EncodeToString(signature)
		str3 := ""
		if isText {
			str3 = "out"
		}
		logger.Error().
			Str("checksum", str0).
			Str("publicKey", str1).
			Str("signature", str2).
			Msgf("signature verification failed! maybe try again with%s -t / --text?", str3)
		return 1
	}

	logger.Info().Msg("OK")
	return 0
}
