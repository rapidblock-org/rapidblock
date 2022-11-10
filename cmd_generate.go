package main

import (
	"crypto/ed25519"
	"fmt"
	"os"
)

func cmdGenerate() {
	switch {
	case flagPublicKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -p / --public-key-file\n")
		os.Exit(1)

	case flagPrivateKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -k / --private-key-file\n")
		os.Exit(1)
	}

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to generate key: %v\n", err)
		os.Exit(1)
	}

	seed := privKey.Seed()
	WriteKeySigFile(flagPublicKeyFile, pubKey[:], false)
	WriteKeySigFile(flagPrivateKeyFile, seed[:], true)
}
