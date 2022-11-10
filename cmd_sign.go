package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func cmdSign() {
	switch {
	case flagPublicKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -p / --public-key-file\n")
		os.Exit(1)

	case flagPrivateKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -k / --private-key-file\n")
		os.Exit(1)

	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -d / --data-file\n")
		os.Exit(1)

	case flagSigFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -s / --signature-file\n")
		os.Exit(1)
	}

	pubKey := ed25519.PublicKey(ReadKeySigFile(flagPublicKeyFile, ed25519.PublicKeySize))
	privKey := ed25519.NewKeyFromSeed(ReadKeySigFile(flagPrivateKeyFile, ed25519.SeedSize))
	computedPubKey := privKey.Public().(ed25519.PublicKey)
	if !pubKey.Equal(computedPubKey) {
		str0 := base64.StdEncoding.EncodeToString(computedPubKey[:])
		str1 := base64.StdEncoding.EncodeToString(pubKey[:])
		fmt.Fprintf(os.Stderr, "error: private key does not match public key!\n\tEd25519 public key calculated from private key: %s\n\tEd25519 public key provided: %s\n", str0, str1)
	}

	checksum := checksumFile(flagDataFile, flagText)
	signFile(privKey, pubKey, checksum, flagSigFile)
}
