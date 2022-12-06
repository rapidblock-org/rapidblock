package fail

import (
	"bytes"
	"fmt"
	"os"
)

func Fail(format string, args ...any) {
	var buf bytes.Buffer
	buf.WriteString("fatal: ")
	fmt.Fprintf(&buf, format, args...)
	buf.WriteByte('\n')
	os.Stderr.Write(buf.Bytes())
	os.Exit(1)
	panic(nil)
}

func FailIf(cond bool, format string, args ...any) {
	if !cond {
		return
	}
	Fail(format, args...)
	panic(nil)
}

func Must(err error) {
	if err == nil {
		return
	}
	Fail("%v", err)
	panic(nil)
}

func Must1[T0 any](arg0 T0, err error) T0 {
	if err == nil {
		return arg0
	}
	Fail("%v", err)
	panic(nil)
}

func Must2[T0 any, T1 any](arg0 T0, arg1 T1, err error) (T0, T1) {
	if err == nil {
		return arg0, arg1
	}
	Fail("%v", err)
	panic(nil)
}
