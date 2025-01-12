package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestReadMemory(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("deadbeef")),
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
	mem, err := g.ReadMemory(context.Background(), 0, 4)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if [4]byte(mem) != [4]byte{0xDE, 0xAD, 0xBE, 0xEF} {
		t.Errorf("invalid memory")
		return
	}
}

func TestReadMemoryFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
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
	_, err := g.ReadMemory(context.Background(), 0, 4)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}
}

func TestWriteMemory(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	mem := []byte("\xDE\xAD\xBE\xEF")

	// Act
	err := g.WriteMemory(context.Background(), 0, mem)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	data, err := gdb.ReadPacket(conn.Output)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != "M0000000000000000,4:deadbeef" {
		t.Errorf("invalid request")
		return
	}
}

func TestWriteMemoryFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	mem := []byte("\xDE\xAD\xBE\xEF")

	// Act
	err := g.WriteMemory(context.Background(), 0, mem)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}
}
