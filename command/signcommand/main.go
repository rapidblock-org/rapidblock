package signcommand

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/checksum"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

var Factory command.FactoryFunc = func() command.Command {
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

	return command.Command{
		Name:        "sign",
		Description: "Signs a file using an Ed25519 private key.",
		Options:     options,
		Main: func() int {
			if publicKeyFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -p / --public-key-file\n")
				return 1
			}
			if privateKeyFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -k / --private-key-file\n")
				return 1
			}
			if dataFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
				return 1
			}
			if signatureFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -s / --signature-file\n")
				return 1
			}
			return Main(isText, publicKeyFile, privateKeyFile, dataFile, signatureFile)
		},
	}
}

func Main(isText bool, publicKeyFile string, privateKeyFile string, dataFile string, signatureFile string) int {
	raw, err := iohelpers.ReadBase64File(publicKeyFile, ed25519.PublicKeySize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	pubKey := ed25519.PublicKey(raw)

	raw, err = iohelpers.ReadBase64File(privateKeyFile, ed25519.SeedSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	privKey := ed25519.NewKeyFromSeed(raw)

	computedPubKey := privKey.Public().(ed25519.PublicKey)
	if !pubKey.Equal(computedPubKey) {
		str0 := base64.StdEncoding.EncodeToString(computedPubKey[:])
		str1 := base64.StdEncoding.EncodeToString(pubKey[:])
		fmt.Fprintf(os.Stderr, "fatal: private key does not match public key!\n\tEd25519 public key calculated from private key: %s\n\tEd25519 public key provided: %s\n", str0, str1)
		return 1
	}

	checksum, err := checksum.File(dataFile, isText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	signature := ed25519.Sign(privKey, checksum)
	if !ed25519.Verify(pubKey, checksum, signature) {
		fmt.Fprintf(os.Stderr, "fatal: failed to verify signature after creation!\n")
		return 1
	}

	err = iohelpers.WriteBase64File(signatureFile, signature, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	return 0
}
