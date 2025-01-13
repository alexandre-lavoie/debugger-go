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

	target := &gdb.Target{
		Name: "test",
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	r, err := g.WaitReply(context.Background())

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
