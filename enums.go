package main

import (
	"encoding"
	"fmt"
	"strings"
)

type EnumData struct {
	GoName  string
	Name    string
	Aliases []string
}

type ColumnID byte

const (
	IgnoreID ColumnID = iota
	DomainID
	DateRequestedID
	DateDecidedID
	RequesterID
	ReceiptsID
	IsBlockedID
	ReasonID
	TagsID
)

var columnIDDataArray = [...]EnumData{
	{"IgnoreID", "ignore", nil},
	{"DomainID", "domain", []string{"instance"}},
	{"DateRequestedID", "date_requested", nil},
	{"DateDecidedID", "date_decided", nil},
	{"RequesterID", "requester", nil},
	{"ReceiptsID", "receipts", nil},
	{"IsBlockedID", "is_blocked", nil},
	{"ReasonID", "reason", nil},
	{"TagsID", "tags", nil},
}

func (enum ColumnID) Data() EnumData {
	i := uint(enum)
	j := uint(len(columnIDDataArray))
	if i < j {
		return columnIDDataArray[i]
	}
	goName := fmt.Sprintf("ColumnID(%d)", i)
	name := fmt.Sprintf("column-id-%d", i)
	return EnumData{goName, name, nil}
}

func (enum ColumnID) GoString() string {
	return enum.Data().GoName
}

func (enum ColumnID) String() string {
	return enum.Data().Name
}

func (enum ColumnID) MarshalText() ([]byte, error) {
	str := enum.String()
	return []byte(str), nil
}

func (enum *ColumnID) UnmarshalText(raw []byte) error {
	str := string(raw)
	for index := uint(0); index < uint(len(columnIDDataArray)); index++ {
		data := columnIDDataArray[index]
		if strings.EqualFold(str, data.Name) {
			*enum = ColumnID(index)
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*enum = ColumnID(index)
				return nil
			}
		}
	}
	*enum = 0
	return fmt.Errorf("unknown ColumnID enum value %q", str)
}

var (
	_ encoding.TextMarshaler   = ColumnID(0)
	_ encoding.TextUnmarshaler = (*ColumnID)(nil)
)

type GIOType byte

const (
	UnknownType GIOType = iota
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

var gioColumnTypeDataArray = [...]EnumData{
	{"UnknownType", "unknown", nil},
	{"TextType", "text", nil},
	{"ParagraphType", "paragraph", nil},
	{"HTMLParagraphType", "html_paragraph", nil},
	{"CheckboxType", "checkbox", nil},
	{"MultipleChoiceType", "multiple_choice", []string{"multi_choice"}},
	{"NumberType", "number", nil},
	{"DateType", "date", nil},
	{"TimeType", "time", nil},
	{"LinkType", "link", nil},
	{"ImageType", "image", nil},
	{"AddressType", "address", nil},
}

func (enum GIOType) Data() EnumData {
	i := uint(enum)
	j := uint(len(gioColumnTypeDataArray))
	if i < j {
		return gioColumnTypeDataArray[i]
	}
	goName := fmt.Sprintf("GIOType(%d)", i)
	name := fmt.Sprintf("type-%d", i)
	return EnumData{goName, name, nil}
}

func (enum GIOType) GoString() string {
	return enum.Data().GoName
}

func (enum GIOType) String() string {
	return enum.Data().Name
}

func (enum GIOType) MarshalText() ([]byte, error) {
	str := enum.String()
	return []byte(str), nil
}

func (enum *GIOType) UnmarshalText(raw []byte) error {
	str := string(raw)
	for index := uint(0); index < uint(len(gioColumnTypeDataArray)); index++ {
		data := gioColumnTypeDataArray[index]
		if strings.EqualFold(str, data.Name) {
			*enum = GIOType(index)
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*enum = GIOType(index)
				return nil
			}
		}
	}
	*enum = 0
	return fmt.Errorf("unknown GIOType enum value %q", str)
}

var (
	_ encoding.TextMarshaler   = GIOType(0)
	_ encoding.TextUnmarshaler = (*GIOType)(nil)
)
