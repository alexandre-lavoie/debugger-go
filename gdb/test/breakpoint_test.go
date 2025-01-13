package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/core"
	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestAddBreakpoint(t *testing.T) {
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
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if b != g.Breakpoints[0] {
		t.Errorf("bad breakpoint")
		return
	}
}

func TestAddBreakpointFail(t *testing.T) {
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

	// Act
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if b != nil || err == nil {
		t.Errorf("no error")
		return
	}
}

func TestAddBreakpointUnimplemented(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("`'")),
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
	b, err := g.AddBreakpoint(context.Background(), gdb.SoftwareBreakpoint, 0, 1, nil)

	// Assert
	if b != nil || err == nil {
		t.Errorf("no error")
		return
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
	}

	b := &core.Breakpoint{}

	g := gdb.GDBRSP{
		Conn:        conn,
		Target:      target,
		Breakpoints: []*core.Breakpoint{b},
	}

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if g.Breakpoints[0] != nil {
		t.Errorf("breakpoint not removed")
		return
	}
}

func TestRemoveBreakpointFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
	}

	b := &core.Breakpoint{}

	g := gdb.GDBRSP{
		Conn:        conn,
		Target:      target,
		Breakpoints: []*core.Breakpoint{b},
	}

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}

	if g.Breakpoints[0] == nil {
		t.Errorf("breakpoint removed")
		return
	}
}

func TestRemoveBreakpointUnimplemented(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("`'")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
	}

	b := &core.Breakpoint{}

	g := gdb.GDBRSP{
		Conn:        conn,
		Target:      target,
		Breakpoints: []*core.Breakpoint{b},
	}

	// Act
	err := g.RemoveBreakpoint(context.Background(), b)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}

	if g.Breakpoints[0] == nil {
		t.Errorf("breakpoint removed")
		return
	}
}
