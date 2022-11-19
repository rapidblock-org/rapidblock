package main

import (
	"encoding"
	"fmt"
	"strings"
)

type EnumData[T any] struct {
	Value   T
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

var columnIDDataArray = [...]EnumData[ColumnID]{
	{IgnoreID, "IgnoreID", "ignore", nil},
	{DomainID, "DomainID", "domain", []string{"instance"}},
	{DateRequestedID, "DateRequestedID", "date_requested", nil},
	{DateDecidedID, "DateDecidedID", "date_decided", nil},
	{RequesterID, "RequesterID", "requester", nil},
	{ReceiptsID, "ReceiptsID", "receipts", nil},
	{IsBlockedID, "IsBlockedID", "is_blocked", nil},
	{ReasonID, "ReasonID", "reason", nil},
	{TagsID, "TagsID", "tags", nil},
}

func (enum ColumnID) Data() EnumData[ColumnID] {
	i := uint(enum)
	j := uint(len(columnIDDataArray))
	if i < j {
		return columnIDDataArray[i]
	}
	goName := fmt.Sprintf("ColumnID(%d)", i)
	name := fmt.Sprintf("column-id-%d", i)
	return EnumData[ColumnID]{enum, goName, name, nil}
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
	for _, data := range columnIDDataArray {
		if strings.EqualFold(str, data.Name) {
			*enum = data.Value
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*enum = data.Value
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

var gioColumnTypeDataArray = [...]EnumData[GIOType]{
	{UnknownType, "UnknownType", "unknown", nil},
	{TextType, "TextType", "text", nil},
	{ParagraphType, "ParagraphType", "paragraph", nil},
	{HTMLParagraphType, "HTMLParagraphType", "html_paragraph", nil},
	{CheckboxType, "CheckboxType", "checkbox", nil},
	{MultipleChoiceType, "MultipleChoiceType", "multiple_choice", []string{"multi_choice"}},
	{NumberType, "NumberType", "number", nil},
	{DateType, "DateType", "date", nil},
	{TimeType, "TimeType", "time", nil},
	{LinkType, "LinkType", "link", nil},
	{ImageType, "ImageType", "image", nil},
	{AddressType, "AddressType", "address", nil},
}

func (enum GIOType) Data() EnumData[GIOType] {
	i := uint(enum)
	j := uint(len(gioColumnTypeDataArray))
	if i < j {
		return gioColumnTypeDataArray[i]
	}
	goName := fmt.Sprintf("GIOType(%d)", i)
	name := fmt.Sprintf("type-%d", i)
	return EnumData[GIOType]{enum, goName, name, nil}
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
	for _, data := range gioColumnTypeDataArray {
		if strings.EqualFold(str, data.Name) {
			*enum = data.Value
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*enum = data.Value
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
