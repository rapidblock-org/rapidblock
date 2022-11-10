package main

import (
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"
)

var Version = "devel"

var (
	flagVersion        bool
	flagPull           bool
	flagGenerate       bool
	flagSign           bool
	flagText           bool
	flagCredentials    string
	flagSheetID        string
	flagSheetName      string
	flagDataFile       string
	flagSigFile        string
	flagPublicKeyFile  string
	flagPrivateKeyFile string
)

func init() {
	getopt.SetParameters("")
	getopt.FlagLong(&flagVersion, "version", 'V', "show version information and exit")
	getopt.FlagLong(&flagPull, "pull", 0, "[pull mode] pull data from Google Sheets and create JSON")
	getopt.FlagLong(&flagGenerate, "generate-key", 'g', "[generate mode] generate an Ed25519 keypair, writing it to --public-key-file and --private-key-file")
	getopt.FlagLong(&flagSign, "sign", 'S', "[sign mode] sign --data-file, writing the Ed25519 signature to --signature-file\ndefault is to verify --data-file against --signature-file")
	getopt.FlagLong(&flagText, "text", 't', "[sign, verify] perform newline canonicalization, under the assumption that --data-file is text")
	getopt.FlagLong(&flagCredentials, "credentials-file", 0, "[pull] path to the JWT service account credentials")
	getopt.FlagLong(&flagSheetID, "sheet-id", 0, "[pull] ID of the Google Sheet spreadsheet to pull data from")
	getopt.FlagLong(&flagSheetName, "sheet-name", 0, "[pull] Name of the Google Sheet sheet to pull data from")
	getopt.FlagLong(&flagDataFile, "data-file", 'd', "[pull, sign, verify] path to the payload file to create, sign, or verify")
	getopt.FlagLong(&flagSigFile, "signature-file", 's', "[sign, verify] path to the base-64 Ed25519 signature file to create or verify")
	getopt.FlagLong(&flagPublicKeyFile, "public-key-file", 'p', "[generate, sign, verify] path to the base-64 Ed25519 public key file to verify with")
	getopt.FlagLong(&flagPrivateKeyFile, "private-key-file", 'k', "[generate, sign] path to the base-64 Ed25519 private key file to sign with")
}

func main() {
	getopt.Parse()

	if flagVersion {
		fmt.Println(Version)
		return
	}

	count := 0
	if flagPull {
		count++
	}
	if flagGenerate {
		count++
	}
	if flagSign {
		count++
	}

	switch {
	case count > 1:
		fmt.Fprintf(os.Stderr, "error: --pull, --generate, and --sign are mutually exclusive\n")
		os.Exit(1)

	case flagPull:
		cmdPull()

	case flagGenerate:
		cmdGenerate()

	case flagSign:
		cmdSign()

	default:
		cmdVerify()
	}
}
