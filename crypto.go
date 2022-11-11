package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func signFile(privKey ed25519.PrivateKey, pubKey ed25519.PublicKey, checksum []byte, sigFileName string) {
	signature := ed25519.Sign(privKey, checksum)
	if !ed25519.Verify(pubKey, checksum, signature) {
		fmt.Fprintf(os.Stderr, "fatal: failed to verify signature after creation!\n")
		os.Exit(1)
	}
	WriteKeySigFile(sigFileName, signature, false)
}

func verifyFile(pubKey ed25519.PublicKey, checksum []byte, sigFileName string) {
	signature := ReadKeySigFile(sigFileName, ed25519.SignatureSize)
	if !ed25519.Verify(pubKey, checksum, signature) {
		str0 := base64.StdEncoding.EncodeToString(checksum)
		str1 := base64.StdEncoding.EncodeToString(pubKey)
		str2 := base64.StdEncoding.EncodeToString(signature)
		fmt.Fprintf(os.Stderr, "fatal: signature verification failed!\n\tSHA-256 checksum: %s\n\tEd25519 public key: %s\n\tEd25519 signature: %s\n", str0, str1, str2)
		switch {
		case flagText:
			fmt.Fprintf(os.Stderr, "\tmaybe try again without --text?\n")
		default:
			fmt.Fprintf(os.Stderr, "\tmaybe try again with --text?\n")
		}
		os.Exit(1)
	}
}
