package blockapply

import (
	"context"
	"fmt"

	"github.com/chronos-tachyon/rapidblock/blockfile"
)

type Func func(ctx context.Context, server Server, file blockfile.BlockFile) (Stats, error)

func funcNoOp(ctx context.Context, server Server, file blockfile.BlockFile) (Stats, error) {
	var zero Stats
	return zero, nil
}

func funcFail(ctx context.Context, server Server, file blockfile.BlockFile) (Stats, error) {
	var zero Stats
	return zero, fmt.Errorf("Mode %q not supported", server.Mode.String())
}
