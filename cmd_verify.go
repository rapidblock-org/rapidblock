package main

import (
	"crypto/ed25519"
	"fmt"
	"os"
)

func cmdVerify() {
	switch {
	case flagPublicKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -p / --public-key-file\n")
		os.Exit(1)

	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -d / --data-file\n")
		os.Exit(1)

	case flagSigFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -s / --signature-file\n")
		os.Exit(1)
	}

	pubKey := ed25519.PublicKey(ReadKeySigFile(flagPublicKeyFile, ed25519.PublicKeySize))
	checksum := checksumFile(flagDataFile, flagText)
	verifyFile(pubKey, checksum, flagSigFile)
	fmt.Println("OK")
}
