package verifycommand

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/commands/command"
	"github.com/chronos-tachyon/rapidblock/internal/checksum"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

var Factory command.FactoryFunc = func() command.Command {
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

	return command.Command{
		Name:        "verify",
		Description: "Verifies an Ed25519 cryptographic signature.",
		Options:     options,
		Main: func() int {
			if publicKeyFile == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -p / --public-key-file\n")
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
			return Main(isText, publicKeyFile, dataFile, signatureFile)
		},
	}
}

func Main(isText bool, publicKeyFile string, dataFile string, signatureFile string) int {
	raw, err := iohelpers.ReadBase64File(publicKeyFile, ed25519.PublicKeySize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	pubKey := ed25519.PublicKey(raw)

	signature, err := iohelpers.ReadBase64File(signatureFile, ed25519.SignatureSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	checksum, err := checksum.File(dataFile, isText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	if !ed25519.Verify(pubKey, checksum, signature) {
		str0 := base64.StdEncoding.EncodeToString(checksum)
		str1 := base64.StdEncoding.EncodeToString(pubKey)
		str2 := base64.StdEncoding.EncodeToString(signature)
		str3 := "with"
		if isText {
			str3 = "without"
		}
		fmt.Fprintf(os.Stderr, "fatal: signature verification failed!\n\tSHA-256 checksum: %s\n\tEd25519 public key: %s\n\tEd25519 signature: %s\n\tmaybe try again %s --text?\n", str0, str1, str2, str3)
		return 1
	}

	fmt.Println("OK")
	return 0
}
