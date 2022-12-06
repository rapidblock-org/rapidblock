package appversion

import (
	"bytes"
	"fmt"
	"io"
)

const userAgentFormat = "RapidBlock/%s (+https://github.com/chronos-tachyon/rapidblock/)"

var (
	Version    = "devel"
	Commit     = ""
	CommitDate = ""
	TreeState  = ""
)

func Print(w io.Writer) (int, error) {
	var buf bytes.Buffer
	buf.Grow(64)
	buf.WriteString(Version)
	buf.WriteByte('\n')
	if Commit != "" {
		buf.WriteString("commit=")
		buf.WriteString(Commit)
		buf.WriteByte('\n')
	}
	if CommitDate != "" {
		buf.WriteString("commitDate=")
		buf.WriteString(CommitDate)
		buf.WriteByte('\n')
	}
	if TreeState != "" {
		buf.WriteString("treeState=")
		buf.WriteString(TreeState)
		buf.WriteByte('\n')
	}
	return w.Write(buf.Bytes())
}

func UserAgent() string {
	return fmt.Sprintf(userAgentFormat, Version)
}
