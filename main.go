package main

import (
	"fmt"
	"os"

	getopt "github.com/pborman/getopt/v2"
)

var (
	Version    = "devel"
	Commit     = "unknown"
	CommitDate = "unknown"
	TreeState  = "unknown"
)

const (
	PrepareData = "prepare-data"
	ExportCSV   = "export-csv"
	GenerateKey = "generate-key"
	Sign        = "sign"
	Verify      = "verify"
	Apply       = "apply"

	AllModes             = PrepareData + ", " + ExportCSV + "," + GenerateKey + ", " + Sign + ", " + Verify + ", " + Apply
	AllExceptGenerateKey = PrepareData + ", " + ExportCSV + "," + Sign + ", " + Verify + ", " + Apply
	GenerateSignVerify   = GenerateKey + ", " + Sign + ", " + Verify
	GenerateSign         = GenerateKey + ", " + Sign
	SignVerify           = Sign + ", " + Verify

	Mastodon3x = "mastodon-3.x"
	Mastodon4x = "mastodon-4.x"

	AllSoftware = Mastodon4x + ", " + Mastodon3x
)

var (
	flagVersion         bool
	flagText            bool
	flagMode            string
	flagSoftware        string
	flagAccountDataFile string
	flagSourceID        string
	flagCsvFile         string
	flagDataFile        string
	flagSigFile         string
	flagPublicKeyFile   string
	flagPrivateKeyFile  string
	flagDatabaseURL     string
)

func init() {
	getopt.SetParameters("")
	getopt.FlagLong(&flagVersion, "version", 'V', "show version information and exit")
	getopt.FlagLong(&flagText, "text", 't', "["+SignVerify+"] perform newline canonicalization, under the assumption that --data-file is text")
	getopt.FlagLong(&flagMode, "mode", 'm', "select mode of operation: "+AllModes)
	getopt.FlagLong(&flagSoftware, "software", 'x', "["+Apply+"] select which server software is in use: "+AllSoftware)
	getopt.FlagLong(&flagAccountDataFile, "account-data-file", 'A', "["+PrepareData+"] path to the groups.io cookies and database column mappings")
	getopt.FlagLong(&flagSourceID, "source-id", 'S', "["+PrepareData+"] ID of the Google Sheet spreadsheet to pull data from")
	getopt.FlagLong(&flagCsvFile, "csv-file", 'c', "["+ExportCSV+"] path to the CSV file to create")
	getopt.FlagLong(&flagDataFile, "data-file", 'd', "["+AllExceptGenerateKey+"] path to the JSON file to create, export from, sign, verify, or apply")
	getopt.FlagLong(&flagSigFile, "signature-file", 's', "["+SignVerify+"] path to the base-64 Ed25519 signature file to create or verify")
	getopt.FlagLong(&flagPublicKeyFile, "public-key-file", 'p', "["+GenerateSignVerify+"] path to the base-64 Ed25519 public key file to verify with")
	getopt.FlagLong(&flagPrivateKeyFile, "private-key-file", 'k', "["+GenerateSign+"] path to the base-64 Ed25519 private key file to sign with")
	getopt.FlagLong(&flagDatabaseURL, "database-url", 'D', "["+Apply+"] PostgreSQL database URL to connect to")
}

func main() {
	getopt.Parse()

	if flagVersion {
		fmt.Println(Version)
		fmt.Println(Commit)
		fmt.Println(CommitDate)
		fmt.Println(TreeState)
		return
	}

	switch flagMode {
	case PrepareData:
		cmdPrepareData()
	case ExportCSV:
		cmdExportCSV()
	case GenerateKey:
		cmdGenerateKey()
	case Sign:
		cmdSign()
	case Verify:
		cmdVerify()
	case Apply:
		cmdApply()
	default:
		fmt.Fprintf(os.Stderr, "fatal: unknown value %q for -m / --mode flag, expected one of: %s\n", flagMode, AllModes)
		os.Exit(1)
	}
}
