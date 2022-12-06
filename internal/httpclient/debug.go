package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

func WriteDebug(w io.Writer, header http.Header, body []byte) {
	var buf bytes.Buffer
	buf.Grow(1024 + len(body))

	for name, values := range header {
		for _, value := range values {
			buf.WriteByte('\t')
			buf.WriteString(name)
			buf.WriteString(": ")
			buf.WriteString(value)
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')

	type state byte
	const (
		stateStartOfLine state = iota
		stateIndented
		stateEndOfLine
	)

	s := stateStartOfLine
	for _, ch := range body {
		switch {
		case ch == '\r' || ch == '\n':
			s = stateEndOfLine
		case s == stateEndOfLine:
			buf.WriteByte('\n')
			fallthrough
		case s == stateStartOfLine:
			buf.WriteByte('\t')
			fallthrough
		default:
			buf.WriteByte(ch)
			s = stateIndented
		}
	}
	if s != stateStartOfLine {
		buf.WriteByte('\n')
	}

	_, _ = w.Write(buf.Bytes())
}
