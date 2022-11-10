package main

import (
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"
)

var Version = "devel"

const (
	PrepareData        = "prepare-data"
	GenerateKey        = "generate-key"
	Sign               = "sign"
	Verify             = "verify"
	AllModes           = PrepareData + ", " + GenerateKey + ", " + Sign + ", " + Verify
	PrepareSignVerify  = PrepareData + ", " + Sign + ", " + Verify
	GenerateSignVerify = GenerateKey + ", " + Sign + ", " + Verify
	GenerateSign       = GenerateKey + ", " + Sign
	SignVerify         = Sign + ", " + Verify
)

var (
	flagVersion         bool
	flagText            bool
	flagMode            string
	flagCredentialsFile string
	flagSheetID         string
	flagSheetName       string
	flagDataFile        string
	flagSigFile         string
	flagPublicKeyFile   string
	flagPrivateKeyFile  string
)

func init() {
	getopt.SetParameters("")
	getopt.FlagLong(&flagVersion, "version", 'V', "show version information and exit")
	getopt.FlagLong(&flagText, "text", 't', "["+SignVerify+"] perform newline canonicalization, under the assumption that --data-file is text")
	getopt.FlagLong(&flagMode, "mode", 'm', "select mode of operation: "+AllModes)
	getopt.FlagLong(&flagCredentialsFile, "credentials-file", 'K', "["+PrepareData+"] path to the JWT service account credentials")
	getopt.FlagLong(&flagSheetID, "sheet-id", 'H', "["+PrepareData+"] ID of the Google Sheet spreadsheet to pull data from")
	getopt.FlagLong(&flagSheetName, "sheet-name", 'N', "["+PrepareData+"] Name of the Google Sheet sheet to pull data from")
	getopt.FlagLong(&flagDataFile, "data-file", 'd', "["+PrepareSignVerify+"] path to the payload file to create, sign, or verify")
	getopt.FlagLong(&flagSigFile, "signature-file", 's', "["+SignVerify+"] path to the base-64 Ed25519 signature file to create or verify")
	getopt.FlagLong(&flagPublicKeyFile, "public-key-file", 'p', "["+GenerateSignVerify+"] path to the base-64 Ed25519 public key file to verify with")
	getopt.FlagLong(&flagPrivateKeyFile, "private-key-file", 'k', "["+GenerateSign+"] path to the base-64 Ed25519 private key file to sign with")
}

func main() {
	getopt.Parse()

	if flagVersion {
		fmt.Println(Version)
		return
	}

	switch flagMode {
	case PrepareData:
		cmdPrepareData()
	case GenerateKey:
		cmdGenerateKey()
	case Sign:
		cmdSign()
	case Verify:
		cmdVerify()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown value %q for -m / --mode flag, expected one of: %s\n", flagMode, AllModes)
		os.Exit(1)
	}
}
