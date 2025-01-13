package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestAddBreakpoint(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

	// Act
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if b != g.Breakpoints.List[0] {
		t.Errorf("bad breakpoint")
		return
	}

	data, err := gdb.ReadPacket(conn.Output)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != "Z0,0000000000000000,1" {
		t.Errorf("invalid request")
		return
	}
}

func TestAddBreakpointRepeat(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

	// Act 0
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert 0
	if err != nil {
		t.Error(err)
		return
	}

	if b != g.Breakpoints.List[0] {
		t.Errorf("bad breakpoint")
		return
	}

	data, err := gdb.ReadPacket(conn.Output)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != "Z0,0000000000000000,1" {
		t.Errorf("invalid request")
		return
	}

	// Act 1
	b, err = g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Asset 1
	if err != nil {
		t.Error(err)
		return
	}

	if b != g.Breakpoints.List[1] {
		t.Errorf("bad breakpoint")
		return
	}

	data, err = gdb.ReadPacket(conn.Output)
	if err.Error() != "EOF" {
		t.Error(err)
		return
	}
}

func TestAddBreakpointFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

	// Act
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}

	// TODO: Handle breakpoint remove on fail?
	if b != nil {
		// t.Errorf("breakpoint added")
	}
}

func TestAddBreakpointUnimplemented(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("`'")),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

	// Act
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if err == nil {
		t.Errorf("no error")
	}

	// TODO: Handle breakpoint remove on fail?
	if b != nil {
		// t.Errorf("breakpoint added")
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

	b, _, _ := g.Breakpoints.Add(gdb.SoftwareBreakpoint, 0, 1, nil)

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if g.Breakpoints.List[0] != nil {
		t.Errorf("breakpoint not removed")
		return
	}

	data, err := gdb.ReadPacket(conn.Output)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != "z0,0000000000000000,1" {
		t.Errorf("invalid request")
		return
	}
}

func TestRemoveBreakpointRepeat(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	g := gdb.NewGDBRSP(conn)
	g.Target = gdb.NewTarget()

	b0, _, err := g.Breakpoints.Add(gdb.SoftwareBreakpoint, 0, 1, nil)
	b1, _, err := g.Breakpoints.Add(gdb.SoftwareBreakpoint, 0, 1, nil)

	// Act 0
	err = g.RemoveBreakpoint(context.Background(), b0)

	// Assert 0
	if err != nil {
		t.Error(err)
		return
	}

	if g.Breakpoints.List[0] != nil {
		t.Errorf("not removed")
		return
	}

	data, err := gdb.ReadPacket(conn.Output)
	if err.Error() != "EOF" {
		t.Error(err)
		return
	}

	// Act 1
	err = g.RemoveBreakpoint(context.Background(), b1)

	// Asset 1
	if err != nil {
		t.Error(err)
		return
	}

	if g.Breakpoints.List[1] != nil {
		t.Errorf("bad breakpoint")
		return
	}

	data, err = gdb.ReadPacket(conn.Output)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != "z0,0000000000000000,1" {
		t.Errorf("invalid request")
		return
	}
}

func TestRemoveBreakpointFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

	b, _, _ := g.Breakpoints.Add(gdb.SoftwareBreakpoint, 0, 1, nil)

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err == nil {
		t.Errorf("no error")
	}

	// TODO: Handle breakpoint re-add on fail?
	if g.Breakpoints.List[0] == nil {
		// t.Errorf("breakpoint removed")
	}
}

func TestRemoveBreakpointUnimplemented(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("`'")),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))
	g.Target = gdb.NewTarget()

	b, _, _ := g.Breakpoints.Add(gdb.SoftwareBreakpoint, 0, 1, nil)

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err == nil {
		t.Errorf("no error")
	}

	// TODO: Handle breakpoint re-add on fail?
	if g.Breakpoints.List[0] == nil {
		// t.Errorf("breakpoint removed")
	}
}
