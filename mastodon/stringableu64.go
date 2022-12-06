package mastodon

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type StringableU64 uint64

func (su64 StringableU64) GoString() string {
	out := make([]byte, 0, 24)
	out = append(out, "mastodon.StringableU64("...)
	out = strconv.AppendUint(out, uint64(su64), 10)
	out = append(out, ")"...)
	return string(out)
}

func (su64 StringableU64) String() string {
	return strconv.FormatUint(uint64(su64), 10)
}

func (su64 StringableU64) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, 20)
	out = append(out, '"')
	out = strconv.AppendUint(out, uint64(su64), 10)
	out = append(out, '"')
	return out, nil
}

func (su64 *StringableU64) UnmarshalJSON(raw []byte) error {
	var u64 uint64
	var err error

	str := string(raw)
	if err == nil && len(raw) > 0 && raw[0] == '"' {
		err = json.Unmarshal(raw, &str)
	}
	if err == nil {
		u64, err = strconv.ParseUint(str, 0, 64)
	}
	if err == nil {
		*su64 = StringableU64(u64)
		return nil
	}
	return fmt.Errorf("failed to parse %q as string-wrapped uint64: %w", str, err)
}

var (
	_ fmt.GoStringer   = StringableU64(0)
	_ fmt.Stringer     = StringableU64(0)
	_ json.Marshaler   = StringableU64(0)
	_ json.Unmarshaler = (*StringableU64)(nil)
)
