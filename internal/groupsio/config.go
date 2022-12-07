package groupsio

import (
	"bytes"
)

type AccountConfig struct {
	DatabaseID uint64               `json:"databaseID"`
	Cookies    map[string]string    `json:"cookies"`
	Columns    map[int]ColumnConfig `json:"columns"`
}

type ColumnConfig struct {
	ID      Column         `json:"id"`
	Choices map[int]string `json:"choices"`
}

func (config AccountConfig) CookieString() (string, bool) {
	if len(config.Cookies) <= 0 {
		return "", false
	}

	var buf bytes.Buffer
	isNext := false
	for name, value := range config.Cookies {
		if isNext {
			buf.WriteString("; ")
		}
		buf.WriteString(name)
		buf.WriteByte('=')
		buf.WriteString(value)
		isNext = true
	}
	return buf.String(), true
}
