package checksum

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

const bufferSize = 1 << 20 // 1 MiB

func File(filePath string, isText bool) ([]byte, error) {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %q: %w", filePath, err)
	}

	needClose := true
	defer func() {
		if needClose {
			_ = f.Close()
		}
	}()

	var srcBuf [bufferSize]byte
	var dstBuf [bufferSize]byte
	h := sha256.New()
	for {
		n, err := f.Read(srcBuf[:])
		isEOF := false
		switch {
		case err == nil:
			// pass
		case err == io.EOF:
			isEOF = true
		default:
			return nil, fmt.Errorf("I/O error while reading file: %q: %w", filePath, err)
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

	needClose = false
	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close file: %q: %w", filePath, err)
	}

	checksum := h.Sum(nil)
	return checksum, nil
}
