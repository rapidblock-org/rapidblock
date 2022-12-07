package blockapply

import (
	"bytes"
	"io"
	"strconv"
)

type Stats struct {
	InsertCount uint
	UpdateCount uint
	DeleteCount uint
}

func (stats Stats) WriteTo(serverName string, w io.Writer) (int, error) {
	var buf bytes.Buffer
	buf.Grow(256)
	if stats.InsertCount > 0 {
		buf.WriteString(serverName)
		buf.WriteString(": added ")
		buf.WriteString(strconv.FormatUint(uint64(stats.InsertCount), 10))
		buf.WriteString(" new block(s)\n")
	}
	if stats.UpdateCount > 0 {
		buf.WriteString(serverName)
		buf.WriteString(": modified ")
		buf.WriteString(strconv.FormatUint(uint64(stats.UpdateCount), 10))
		buf.WriteString(" existing block(s)\n")
	}
	if stats.DeleteCount > 0 {
		buf.WriteString(serverName)
		buf.WriteString(": deleted ")
		buf.WriteString(strconv.FormatUint(uint64(stats.DeleteCount), 10))
		buf.WriteString(" existing block(s)\n")
	}
	return w.Write(buf.Bytes())
}
