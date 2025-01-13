package gdb

import (
	"context"
	"fmt"
	"net"
)

func Remote(ctx context.Context, host string, port uint16) (*GDBRSP, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	gdb := NewGDBRSP(conn)

	if err := gdb.init(ctx); err != nil {
		return nil, err
	}

	return gdb, nil
}
