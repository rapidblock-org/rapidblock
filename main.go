package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	getopt "github.com/pborman/getopt/v2"
)

const bufferSize = 1 << 20 // 1 MiB

var reSpace = regexp.MustCompile(`\s+`)

var (
	flagGenerate       bool
	flagSign           bool
	flagText           bool
	flagDataFile       string
	flagSigFile        string
	flagPublicKeyFile  string
	flagPrivateKeyFile string
)

func init() {
	getopt.SetParameters("")
	getopt.FlagLong(&flagGenerate, "generate-key", 'g', "generate an Ed25519 keypair, writing it to --public-key and --private-key")
	getopt.FlagLong(&flagSign, "sign", 'S', "sign --data, writing the Ed25519 signature to --signature\ndefault is to verify --data against --signature")
	getopt.FlagLong(&flagText, "text", 't', "perform newline canonicalization, under the assumption that --data is text")
	getopt.FlagLong(&flagDataFile, "data", 'd', "path to the payload file to sign or to verify")
	getopt.FlagLong(&flagSigFile, "signature", 's', "path to the base-64 Ed25519 signature file to create or to verify")
	getopt.FlagLong(&flagPublicKeyFile, "public-key", 'p', "path to the base-64 Ed25519 public key file to verify with")
	getopt.FlagLong(&flagPrivateKeyFile, "private-key", 'k', "path to the base-64 Ed25519 private key file to sign with")
}

func loadFile(filePath string, fileType string, expectedSize int) []byte {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read %s from file %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}

	raw = reSpace.ReplaceAllLiteral(raw, nil)

	data := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
	dataSize, err := base64.StdEncoding.Decode(data, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to decode base-64 %s: %v\n", fileType, err)
		os.Exit(1)
	}

	if dataSize != expectedSize {
		fmt.Fprintf(os.Stderr, "error: %s has wrong length: expected %d bytes, got %d bytes\n", fileType, expectedSize, dataSize)
		os.Exit(1)
	}

	return data[:dataSize]
}

func storeFile(filePath string, fileType string, isPrivate bool, data []byte) {
	encodedLen := base64.StdEncoding.EncodedLen(len(data))
	encoded := make([]byte, encodedLen, encodedLen+2)
	base64.StdEncoding.Encode(encoded, data)
	encoded = append(encoded, '\r', '\n')

	mode := os.FileMode(0o666)
	if isPrivate {
		mode = os.FileMode(0o600)
	}

	dirPath := filepath.Dir(filePath)
	dir, err := os.OpenFile(dirPath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open directory containing %s file %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}
	defer dir.Close()

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create %s file %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}

	_, err = file.Write(encoded)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: I/O error while writing %s to %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}

	err = file.Sync()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: I/O error while writing %s to %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}

	err = dir.Sync()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: I/O error while writing %s to %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}

	err = file.Close()
	if err != nil {
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: failed to close %s file %q: %v\n", fileType, filePath, err)
		os.Exit(1)
	}
}

func signFile(privKey ed25519.PrivateKey, pubKey ed25519.PublicKey, checksum []byte, sigFileName string) {
	signature := ed25519.Sign(privKey, checksum)
	if !ed25519.Verify(pubKey, checksum, signature) {
		fmt.Fprintf(os.Stderr, "error: failed to verify signature after creation!\n")
		os.Exit(1)
	}
	storeFile(sigFileName, "signature", false, signature)
}

func verifyFile(pubKey ed25519.PublicKey, checksum []byte, sigFileName string) {
	signature := loadFile(sigFileName, "signature", ed25519.SignatureSize)
	if !ed25519.Verify(pubKey, checksum, signature) {
		str0 := base64.StdEncoding.EncodeToString(checksum)
		str1 := base64.StdEncoding.EncodeToString(pubKey)
		str2 := base64.StdEncoding.EncodeToString(signature)
		fmt.Fprintf(os.Stderr, "error: signature verification failed!\n\tSHA-256 checksum: %s\n\tEd25519 public key: %s\n\tEd25519 signature: %s\n", str0, str1, str2)
		os.Exit(1)
	}
}

type mode int

const (
	modeVerify mode = iota
	modeSign
	modeGenerate
)

func main() {
	getopt.Parse()

	mode := modeVerify
	switch {
	case flagGenerate && flagSign:
		fmt.Fprintf(os.Stderr, "error: --generate and --sign are mutually exclusive\n")
		os.Exit(1)

	case flagGenerate:
		mode = modeGenerate

	case flagSign:
		mode = modeSign
	}

	switch {
	case flagPublicKeyFile == "":
		fmt.Fprintf(os.Stderr, "error: missing required flag -p / --public-key\n")
		os.Exit(1)

	case flagPrivateKeyFile == "" && mode != modeVerify:
		fmt.Fprintf(os.Stderr, "error: missing required flag -k / --private-key\n")
		os.Exit(1)

	case flagDataFile == "" && mode != modeGenerate:
		fmt.Fprintf(os.Stderr, "error: missing required flag -d / --data\n")
		os.Exit(1)

	case flagSigFile == "" && mode != modeGenerate:
		fmt.Fprintf(os.Stderr, "error: missing required flag -s / --signature\n")
		os.Exit(1)
	}

	var pubKey ed25519.PublicKey
	var privKey ed25519.PrivateKey

	if flagGenerate {
		var err error
		pubKey, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to generate key: %v\n", err)
			os.Exit(1)
		}
		seed := privKey.Seed()
		storeFile(flagPublicKeyFile, "public key", false, pubKey[:])
		storeFile(flagPrivateKeyFile, "private key", true, seed[:])
		return
	}

	pubKey = ed25519.PublicKey(loadFile(flagPublicKeyFile, "public key", ed25519.PublicKeySize))

	if flagSign {
		privKey = ed25519.NewKeyFromSeed(loadFile(flagPrivateKeyFile, "private key", ed25519.SeedSize))
		computedPubKey := privKey.Public().(ed25519.PublicKey)
		if !pubKey.Equal(computedPubKey) {
			str0 := base64.StdEncoding.EncodeToString(computedPubKey[:])
			str1 := base64.StdEncoding.EncodeToString(pubKey[:])
			fmt.Fprintf(os.Stderr, "error: private key does not match public key!\n\tEd25519 public key calculated from private key: %s\n\tEd25519 public key provided: %s\n", str0, str1)
		}
	}

	dataFile, err := os.OpenFile(flagDataFile, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open %q: %v\n", flagDataFile, err)
		os.Exit(1)
	}

	var srcBuf [bufferSize]byte
	var dstBuf [bufferSize]byte
	h := sha256.New()
	for {
		n, err := dataFile.Read(srcBuf[:])
		isEOF := false
		switch {
		case err == nil:
			// pass
		case err == io.EOF:
			isEOF = true
		default:
			_ = dataFile.Close()
			fmt.Fprintf(os.Stderr, "error: I/O error while reading %q: %v\n", flagDataFile, err)
			os.Exit(1)
		}

		var data []byte
		if flagText {
			data = dstBuf[:0]
			for _, ch := range srcBuf[:n] {
				if ch == '\r' {
					continue
				}
				data = append(data, ch)
			}
		} else {
			data = srcBuf[:n]
		}

		h.Write(data)
		if isEOF {
			break
		}
	}
	checksum := h.Sum(nil)

	err = dataFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to close %q: %v\n", flagDataFile, err)
		os.Exit(1)
	}

	switch {
	case flagSign:
		signFile(privKey, pubKey, checksum, flagSigFile)
	default:
		verifyFile(pubKey, checksum, flagSigFile)
		fmt.Println("OK")
	}
}
