package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func ReadFile(filePath string) []byte {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %q: failed to read file: %v\n", filePath, err)
		os.Exit(1)
	}
	return raw
}

func WriteFile(filePath string, data []byte, isPrivate bool) {
	mode := os.FileMode(0o666)
	if isPrivate {
		mode = os.FileMode(0o600)
	}

	dirPath := filepath.Dir(filePath)
	dir, err := os.OpenFile(dirPath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %q: failed to open directory containing file: %v\n", filePath, err)
		os.Exit(1)
	}
	defer dir.Close()

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %q: failed to create file: %v\n", filePath, err)
		os.Exit(1)
	}

	_, err = file.Write(data)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: %q: I/O error: %v\n", filePath, err)
		os.Exit(1)
	}

	err = file.Sync()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: %q: I/O error: %v\n", filePath, err)
		os.Exit(1)
	}

	err = dir.Sync()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: %q: I/O error: %v\n", filePath, err)
		os.Exit(1)
	}

	err = file.Close()
	if err != nil {
		_ = os.Remove(filePath)
		fmt.Fprintf(os.Stderr, "error: %q: failed to close file: %v\n", filePath, err)
		os.Exit(1)
	}
}

func ReadKeySigFile(filePath string, expectedSize int) []byte {
	raw := ReadFile(filePath)
	raw = reSpace.ReplaceAllLiteral(raw, nil)
	data := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
	dataSize, err := base64.StdEncoding.Decode(data, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %q: failed to decode from base-64: %v\n", filePath, err)
		os.Exit(1)
	}
	if expectedSize >= 0 && dataSize != expectedSize {
		fmt.Fprintf(os.Stderr, "error: %q: data has wrong length: expected %d bytes, got %d bytes\n", filePath, expectedSize, dataSize)
		os.Exit(1)
	}
	return data[:dataSize]
}

func WriteKeySigFile(filePath string, data []byte, isPrivate bool) {
	encodedLen := base64.StdEncoding.EncodedLen(len(data))
	encoded := make([]byte, encodedLen, encodedLen+2)
	base64.StdEncoding.Encode(encoded, data)
	encoded = append(encoded, '\r', '\n')
	WriteFile(filePath, encoded, isPrivate)
}
