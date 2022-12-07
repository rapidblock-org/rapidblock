package groupsio

import (
	"encoding"
	"fmt"
	"strings"
)

type Column byte

const (
	IgnoreColumn Column = iota
	DomainColumn
	DateRequestedColumn
	DateDecidedColumn
	RequesterColumn
	ReceiptsColumn
	IsBlockedColumn
	ReasonColumn
	TagsColumn
	NumColumns
)

type ColumnData struct {
	GoName  string
	Name    string
	Aliases []string
}

var columnDataArray = [...]ColumnData{
	{"groupsio.IgnoreColumn", "ignore", nil},
	{"groupsio.DomainColumn", "domain", []string{"instance"}},
	{"groupsio.DateRequestedColumn", "date_requested", nil},
	{"groupsio.DateDecidedColumn", "date_decided", nil},
	{"groupsio.RequesterColumn", "requester", nil},
	{"groupsio.ReceiptsColumn", "receipts", nil},
	{"groupsio.IsBlockedColumn", "is_blocked", nil},
	{"groupsio.ReasonColumn", "reason", nil},
	{"groupsio.TagsColumn", "tags", nil},
}

func (col Column) Data() ColumnData {
	if col < NumColumns {
		return columnDataArray[col]
	}
	goName := fmt.Sprintf("groupsio.Column(%d)", uint(col))
	name := fmt.Sprintf("column-id-%d", uint(col))
	return ColumnData{goName, name, nil}
}

func (col Column) GoString() string {
	return col.Data().GoName
}

func (col Column) String() string {
	return col.Data().Name
}

func (col Column) MarshalText() ([]byte, error) {
	str := col.String()
	return []byte(str), nil
}

func (col *Column) UnmarshalText(raw []byte) error {
	str := string(raw)
	for enum := Column(0); enum < NumColumns; enum++ {
		data := columnDataArray[enum]
		if str == data.GoName || strings.EqualFold(str, data.Name) {
			*col = enum
			return nil
		}
		for _, alias := range data.Aliases {
			if strings.EqualFold(str, alias) {
				*col = enum
				return nil
			}
		}
	}
	*col = 0
	return fmt.Errorf("unknown Column enum value %q", str)
}

var (
	_ encoding.TextMarshaler   = Column(0)
	_ encoding.TextUnmarshaler = (*Column)(nil)
)
