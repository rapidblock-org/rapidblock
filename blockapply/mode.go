package blockapply

import (
	"bytes"
	"encoding"
	"fmt"
	"strings"
)

type Mode byte

const (
	ModeNoOp Mode = iota
	ModeMastodon3xSQL
	ModeMastodon4xSQL
	ModeMastodon4xREST
	NumModes
)

type ModeData struct {
	GoName  string
	Name    string
	Aliases []string
}

var modeDataArray = [NumModes]ModeData{
	{"blockapply.ModeNoOp", "noop", []string{""}},
	{"blockapply.ModeMastodon3xSQL", "mastodon-3.x-sql", []string{"mastodon-3.x"}},
	{"blockapply.ModeMastodon4xSQL", "mastodon-4.x-sql", nil},
	{"blockapply.ModeMastodon4xREST", "mastodon-4.x-rest", []string{"mastodon-4.x"}},
}

var modeFuncArray = [NumModes]Func{
	funcNoOp,
	funcFail,
	funcFail,
	funcFail,
}

func SetFunc(mode Mode, fn Func) {
	if uint(mode) >= uint(len(modeFuncArray)) {
		panic(fmt.Errorf("blockapply.Mode(%d) out of range", uint(mode)))
	}
	if fn == nil {
		fn = funcFail
	}
	modeFuncArray[mode] = fn
}

func (mode Mode) Data() ModeData {
	if mode < NumModes {
		return modeDataArray[mode]
	}
	goName := fmt.Sprintf("blockapply.Mode(%d)", uint(mode))
	name := fmt.Sprintf("#%d", uint(mode))
	return ModeData{goName, name, nil}
}

func (mode Mode) GoString() string {
	return mode.Data().GoName
}

func (mode Mode) String() string {
	return mode.Data().Name
}

func (mode Mode) Func() Func {
	if mode < NumModes {
		return modeFuncArray[mode]
	}
	return funcFail
}

func (mode Mode) MarshalText() ([]byte, error) {
	str := mode.String()
	return []byte(str), nil
}

func (mode *Mode) UnmarshalText(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	str := string(raw)
	for enum := Mode(0); enum < NumModes; enum++ {
		data := modeDataArray[enum]
		if str == data.GoName || strings.EqualFold(str, data.Name) {
			*mode = enum
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*mode = enum
				return nil
			}
		}
	}
	*mode = 0
	return fmt.Errorf("UnmarshalText: failed to parse %q as Mode", str)
}

var (
	_ fmt.GoStringer           = Mode(0)
	_ fmt.Stringer             = Mode(0)
	_ encoding.TextMarshaler   = Mode(0)
	_ encoding.TextUnmarshaler = (*Mode)(nil)
)
