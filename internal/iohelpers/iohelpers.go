package iohelpers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var reSpace = regexp.MustCompile(`\s+`)

func ReadFile(filePath string) ([]byte, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %q: %w", filePath, err)
	}
	return raw, nil
}

func WriteFile(filePath string, isPrivate bool, data []byte) error {
	mode := os.FileMode(0o666)
	if isPrivate {
		mode = os.FileMode(0o600)
	}

	dirPath := filepath.Dir(filePath)
	d, err := os.OpenFile(dirPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open directory: %q: %w", dirPath, err)
	}
	defer d.Close()

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("failed to create file: %q: %w", filePath, err)
	}

	needClose := true
	needRemove := true
	defer func() {
		if needClose {
			_ = f.Close()
		}
		if needRemove {
			_ = os.Remove(filePath)
		}
	}()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("I/O error: %q: %w", filePath, err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("I/O error: %q: %w", filePath, err)
	}

	err = d.Sync()
	if err != nil {
		return fmt.Errorf("I/O error: %q: %w", filePath, err)
	}

	needClose = false
	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to close file: %q: %w", filePath, err)
	}

	needRemove = false
	return nil
}

func ReadBase64File(filePath string, expectedSize int) ([]byte, error) {
	raw, err := ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	raw = reSpace.ReplaceAllLiteral(raw, nil)
	data := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
	dataSize, err := base64.StdEncoding.Decode(data, raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file contents from base-64: %q: %v", filePath, err)
	}
	if expectedSize >= 0 && dataSize != expectedSize {
		return nil, fmt.Errorf("file's contents have the wrong length: %q: expected %d bytes, got %d bytes", filePath, expectedSize, dataSize)
	}
	return data[:dataSize], nil
}

func WriteBase64File(filePath string, data []byte, isPrivate bool) error {
	encodedLen := base64.StdEncoding.EncodedLen(len(data))
	encoded := make([]byte, encodedLen, encodedLen+2)
	base64.StdEncoding.Encode(encoded, data)
	encoded = append(encoded, '\r', '\n')
	return WriteFile(filePath, isPrivate, encoded)
}

func Load[T any](out *T, filePath string, isStrict bool) error {
	switch {
	case strings.HasSuffix(filePath, ".json"):
		return LoadJSON(out, filePath, isStrict)
	case strings.HasSuffix(filePath, ".yaml"):
		return LoadYAML(out, filePath, isStrict)
	case strings.HasSuffix(filePath, ".yml"):
		return LoadYAML(out, filePath, isStrict)
	default:
		return LoadJSON(out, filePath, isStrict)
	}
}

func LoadJSON[T any](out *T, filePath string, isStrict bool) error {
	raw, err := ReadFile(filePath)
	if err != nil {
		return err
	}

	var tmp T
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if isStrict {
		d.DisallowUnknownFields()
	}
	err = d.Decode(&tmp)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %T: %q: %w", out, filePath, err)
	}

	*out = tmp
	return nil
}

func LoadYAML[T any](out *T, filePath string, isStrict bool) error {
	raw, err := ReadFile(filePath)
	if err != nil {
		return err
	}

	var tmp T
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(isStrict)
	err = d.Decode(&tmp)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal: %T: %q: %w", out, filePath, err)
	}

	*out = tmp
	return nil
}

func Store(filePath string, isPrivate bool, in any) error {
	switch {
	case strings.HasSuffix(filePath, ".json"):
		return StoreJSON(filePath, isPrivate, in)
	case strings.HasSuffix(filePath, ".yaml"):
		return StoreYAML(filePath, isPrivate, in)
	case strings.HasSuffix(filePath, ".yml"):
		return StoreYAML(filePath, isPrivate, in)
	default:
		return StoreJSON(filePath, isPrivate, in)
	}
}

func StoreJSON(filePath string, isPrivate bool, in any) error {
	var buf bytes.Buffer
	buf.Grow(1 << 16) // 64 KiB
	e := json.NewEncoder(&buf)
	e.SetIndent("", "  ")
	e.SetEscapeHTML(false)
	err := e.Encode(in)
	if err != nil {
		return fmt.Errorf("json.Marshal: %T: %q: %w", in, filePath, err)
	}
	raw := buf.Bytes()
	raw = bytes.TrimSpace(raw)
	raw = bytes.ReplaceAll(raw, []byte{'\n'}, []byte{'\r', '\n'})
	raw = append(raw, '\r', '\n')
	return WriteFile(filePath, isPrivate, raw)
}

func StoreYAML(filePath string, isPrivate bool, in any) error {
	var buf bytes.Buffer
	buf.Grow(1 << 16) // 64 KiB
	e := yaml.NewEncoder(&buf)
	e.SetIndent(2)
	err := e.Encode(in)
	if err != nil {
		return fmt.Errorf("yaml.Marshal: %T: %q: %w", in, filePath, err)
	}
	raw := buf.Bytes()
	raw = bytes.TrimSpace(raw)
	raw = append(raw, '\n')
	return WriteFile(filePath, isPrivate, raw)
}
