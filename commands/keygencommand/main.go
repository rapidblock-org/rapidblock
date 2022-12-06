package keygencommand

import (
	"crypto/ed25519"
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/commands/command"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

var Factory command.FactoryFunc = func() command.Command {
	var (
		publicKeyFile  string
		privateKeyFile string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&publicKeyFile, "public-key-file", 'p', "path to the public key file to create")
	options.FlagLong(&privateKeyFile, "private-key-file", 'k', "path to the private key file to create")

	return command.Command{
		Name:        "keygen",
		Aliases:     []string{"genkey", "generate-key"},
		Description: "Generates an Ed25519 cryptographic key pair for signing files.",
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
			return Main(publicKeyFile, privateKeyFile)
		},
	}
}

func Main(publicKeyFile string, privateKeyFile string) int {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to generate key: %v\n", err)
		return 1
	}

	seed := privKey.Seed()
	if err := iohelpers.WriteBase64File(publicKeyFile, pubKey[:], false); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	if err := iohelpers.WriteBase64File(privateKeyFile, seed[:], true); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	return 0
}
