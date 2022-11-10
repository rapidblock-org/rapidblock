package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"regexp"
)

const bufferSize = 1 << 20 // 1 MiB

var reSpace = regexp.MustCompile(`\s+`)

func checksumFile(filePath string, isText bool) []byte {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open %q: %v\n", filePath, err)
		os.Exit(1)
	}

	var srcBuf [bufferSize]byte
	var dstBuf [bufferSize]byte
	h := sha256.New()
	for {
		n, err := file.Read(srcBuf[:])
		isEOF := false
		switch {
		case err == nil:
			// pass
		case err == io.EOF:
			isEOF = true
		default:
			_ = file.Close()
			fmt.Fprintf(os.Stderr, "error: I/O error while reading %q: %v\n", filePath, err)
			os.Exit(1)
		}

		var data []byte
		switch {
		case isText:
			data = dstBuf[:0]
			for _, ch := range srcBuf[:n] {
				if ch == '\r' {
					continue
				}
				data = append(data, ch)
			}
		default:
			data = srcBuf[:n]
		}

		h.Write(data)
		if isEOF {
			break
		}
	}
	checksum := h.Sum(nil)

	err = file.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to close %q: %v\n", filePath, err)
		os.Exit(1)
	}

	return checksum
}
