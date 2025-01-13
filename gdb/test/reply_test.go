package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestWaitReplyStatus(t *testing.T) {
	// Arrange
	output := [][]byte{
		gdb.BuildPacket([]byte("T05thread:11;")),
	}

	conn := NewTestConnection(output)

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

	// Act
	r, err := g.WaitForReply(context.Background())

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if r != gdb.StatusReply {
		t.Errorf("bad reply")
		return
	}

	if g.ThreadID != 0x11 {
		t.Errorf("bad thread")
		return
	}
}
