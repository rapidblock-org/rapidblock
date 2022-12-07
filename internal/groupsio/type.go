package groupsio

import (
	"bytes"
	"encoding"
	"fmt"
	"strings"
)

type Type byte

const (
	UnknownType Type = iota
	TextType
	ParagraphType
	HTMLParagraphType
	CheckboxType
	MultipleChoiceType
	NumberType
	DateType
	TimeType
	LinkType
	ImageType
	AddressType
)

type TypeData struct {
	GoName  string
	Name    string
	Aliases []string
}

var typeDataArray = [...]TypeData{
	{
		GoName:  "groupsio.UnknownType",
		Name:    "unknown",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.TextType",
		Name:    "text",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.ParagraphType",
		Name:    "paragraph",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.HTMLParagraphType",
		Name:    "html_paragraph",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.CheckboxType",
		Name:    "checkbox",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.MultipleChoiceType",
		Name:    "multiple_choice",
		Aliases: []string{"multi_choice"},
	},
	{
		GoName:  "groupsio.NumberType",
		Name:    "number",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.DateType",
		Name:    "date",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.TimeType",
		Name:    "time",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.LinkType",
		Name:    "link",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.ImageType",
		Name:    "image",
		Aliases: nil,
	},
	{
		GoName:  "groupsio.AddressType",
		Name:    "address",
		Aliases: nil,
	},
}

func (enum Type) Data() TypeData {
	i := uint(enum)
	const arrayLen = uint(len(typeDataArray))
	if i < arrayLen {
		return typeDataArray[i]
	}
	goName := fmt.Sprintf("Type(%d)", i)
	name := fmt.Sprintf("type-%d", i)
	return TypeData{goName, name, nil}
}

func (enum Type) GoString() string {
	return enum.Data().GoName
}

func (enum Type) String() string {
	return enum.Data().Name
}

func (enum Type) MarshalText() ([]byte, error) {
	str := enum.String()
	return []byte(str), nil
}

func (enum *Type) UnmarshalText(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	str := string(raw)
	const arrayLen = uint(len(typeDataArray))
	for i := uint(0); i < arrayLen; i++ {
		data := typeDataArray[i]
		if str == data.GoName || strings.EqualFold(str, data.Name) {
			*enum = Type(i)
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*enum = Type(i)
				return nil
			}
		}
	}
	*enum = 0
	return fmt.Errorf("unknown groupsio.Type enum value %q", str)
}

var (
	_ encoding.TextMarshaler   = Type(0)
	_ encoding.TextUnmarshaler = (*Type)(nil)
)
