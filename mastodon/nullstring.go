package mastodon

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

var nullLiteral = []byte("null")

type NullString struct {
	IsValid     bool
	StringValue string
}

func MakeNullString(valid bool, str string) NullString {
	if valid {
		return NullString{IsValid: true, StringValue: str}
	}
	return NullString{}
}

func (ns NullString) GoString() string {
	if ns.IsValid {
		return strconv.Quote(ns.StringValue)
	}
	return "nil"
}

func (ns NullString) String() string {
	if ns.IsValid {
		return ns.StringValue
	}
	return "<nil>"
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if ns.IsValid {
		return json.Marshal(ns.StringValue)
	}
	return nullLiteral, nil
}

func (ns *NullString) UnmarshalJSON(raw []byte) error {
	*ns = NullString{}
	rawLen := uint(len(raw))
	if rawLen == 4 && bytes.Equal(raw, nullLiteral) {
		return nil
	}
	var str string
	err := json.Unmarshal(raw, &str)
	if err == nil {
		ns.IsValid = true
		ns.StringValue = str
		return nil
	}
	return err
}

func (ns NullString) Value() (driver.Value, error) {
	if ns.IsValid {
		return ns.StringValue, nil
	}
	return nil, nil
}

func (ns *NullString) Scan(value any) error {
	*ns = NullString{}
	if value == nil {
		return nil
	}
	switch x := value.(type) {
	case string:
		ns.IsValid = true
		ns.StringValue = x
		return nil
	case []byte:
		ns.IsValid = true
		ns.StringValue = string(x)
		return nil
	case time.Time:
		ns.IsValid = true
		ns.StringValue = x.Format(time.RFC3339Nano)
		return nil
	case float64:
		ns.IsValid = true
		ns.StringValue = strconv.FormatFloat(x, 'g', -1, 64)
		return nil
	case int64:
		ns.IsValid = true
		ns.StringValue = strconv.FormatInt(x, 10)
		return nil
	case bool:
		ns.IsValid = true
		ns.StringValue = strconv.FormatBool(x)
		return nil
	default:
		return fmt.Errorf("don't know how to convert value of type %T to string", value)
	}
}

var (
	_ json.Marshaler   = NullString{}
	_ json.Unmarshaler = (*NullString)(nil)
	_ driver.Valuer    = NullString{}
	_ sql.Scanner      = (*NullString)(nil)
)
