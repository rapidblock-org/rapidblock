package blockapply

import (
	"bytes"
	"fmt"
	"io"
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
		fmt.Fprintf(&buf, "%s: added %d new block(s)\n", serverName, stats.InsertCount)
	}
	if stats.UpdateCount > 0 {
		fmt.Fprintf(&buf, "%s: modified %d existing block(s)\n", serverName, stats.UpdateCount)
	}
	if stats.DeleteCount > 0 {
		fmt.Fprintf(&buf, "%s: deleted %d existing block(s) that are now remediated\n", serverName, stats.DeleteCount)
	}
	return w.Write(buf.Bytes())
}
