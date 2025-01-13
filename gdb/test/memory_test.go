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

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

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

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

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

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

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

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

	mem := []byte("\xDE\xAD\xBE\xEF")

	// Act
	err := g.WriteMemory(context.Background(), 0, mem)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}
}
