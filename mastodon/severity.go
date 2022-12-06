package mastodon

import (
	"bytes"
	"encoding"
	"fmt"
	"strings"
)

type Severity byte

const (
	SeveritySilence Severity = 0
	SeveritySuspend Severity = 1
	SeverityNoOp    Severity = 2
	NumSeverities   Severity = 3
)

type SeverityData struct {
	GoName string
	Name   string
}

var severityDataArray = [NumSeverities]SeverityData{
	{"mastodon.SeveritySilence", "silence"},
	{"mastodon.SeveritySuspend", "suspend"},
	{"mastodon.SeverityNoOp", "noop"},
}

func (severity Severity) Data() SeverityData {
	if severity < NumSeverities {
		return severityDataArray[severity]
	}
	goName := fmt.Sprintf("mastodon.Severity(%d)", uint(severity))
	name := fmt.Sprintf("%d", uint(severity))
	return SeverityData{goName, name}
}

func (severity Severity) GoString() string {
	return severity.Data().GoName
}

func (severity Severity) String() string {
	return severity.Data().Name
}

func (severity Severity) MarshalText() ([]byte, error) {
	str := severity.String()
	return []byte(str), nil
}

func (severity *Severity) UnmarshalText(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	str := string(raw)
	for enum := Severity(0); enum < NumSeverities; enum++ {
		data := severityDataArray[enum]
		if str == data.GoName || strings.EqualFold(str, data.Name) {
			*severity = enum
			return nil
		}
	}
	*severity = 0
	return fmt.Errorf("unknown mastodon.Severity value %q", str)
}

var (
	_ fmt.GoStringer           = Severity(0)
	_ fmt.Stringer             = Severity(0)
	_ encoding.TextMarshaler   = Severity(0)
	_ encoding.TextUnmarshaler = (*Severity)(nil)
)
