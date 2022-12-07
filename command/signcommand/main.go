package signcommand

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

type signFactory struct {
	command.BaseFactory
}

func (signFactory) Name() string {
	return "sign"
}

func (signFactory) Description() string {
	return "Signs a file using an Ed25519 private key."
}

func (signFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		isText         bool
		publicKeyFile  string
		privateKeyFile string
		dataFile       string
		signatureFile  string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&isText, "text", 't', "perform newline canonicalization, under the assumption that -d / --data-file is text")
	options.FlagLong(&publicKeyFile, "public-key-file", 'p', "path to the public key file to verify the signature with")
	options.FlagLong(&privateKeyFile, "private-key-file", 'k', "path to the private key file to sign with")
	options.FlagLong(&dataFile, "data-file", 'd', "path to the data file to sign")
	options.FlagLong(&signatureFile, "signature-file", 's', "path to the signature file to create")

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
		return Main(ctx, isText, publicKeyFile, privateKeyFile, dataFile, signatureFile)
	}
}

var Factory command.Factory = signFactory{}

func Main(ctx context.Context, isText bool, publicKeyFile string, privateKeyFile string, dataFile string, signatureFile string) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("publicKeyFile", publicKeyFile).
		Str("privateKeyFile", privateKeyFile).
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

	raw, err = iohelpers.ReadBase64File(privateKeyFile, ed25519.SeedSize)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	privKey := ed25519.NewKeyFromSeed(raw)

	computedPubKey := privKey.Public().(ed25519.PublicKey)
	if !pubKey.Equal(computedPubKey) {
		str0 := base64.StdEncoding.EncodeToString(computedPubKey[:])
		str1 := base64.StdEncoding.EncodeToString(pubKey[:])
		logger.Error().
			Str("computed", str0).
			Str("provided", str1).
			Msg("private key does not match public key!")
		return 1
	}

	checksum, err := checksum.File(dataFile, isText)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	signature := ed25519.Sign(privKey, checksum)
	if !ed25519.Verify(pubKey, checksum, signature) {
		str0 := base64.StdEncoding.EncodeToString(checksum)
		str1 := base64.StdEncoding.EncodeToString(signature)
		logger.Error().
			Str("checksum", str0).
			Str("signature", str1).
			Msg("failed to verify signature after creation!")
		return 1
	}

	err = iohelpers.WriteBase64File(signatureFile, signature, false)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	return 0
}
