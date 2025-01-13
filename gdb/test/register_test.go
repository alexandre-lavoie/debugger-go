package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/core"
	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestRegisterToUint(t *testing.T) {
	// Arrange
	def := &core.Register{
		Name:    "int32",
		Index:   0,
		BitSize: 32,
		Type:    "int32",
	}

	reg := core.RegisterValue{
		Definition: def,
		Value:      []byte{0x1, 0x2, 0x3, 0x4},
	}

	// Act
	v := reg.ToUint()

	// Assert
	if v != 0x04030201 {
		t.Error("invalid value")
		return
	}
}

func TestReadRegister(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("deadbeef")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					Index:   0,
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	reg, err := g.ReadRegister(context.Background(), target.Arch.Registers[0])
	if err != nil {
		t.Error(err)
		return
	}

	// Assert
	if reg.Definition != target.Arch.Registers[0] {
		t.Errorf("invalid register")
		return
	}

	if reg.Value == nil || len(reg.Value) != 4 {
		t.Errorf("invalid value")
		return
	}

	if [4]byte(reg.Value) != [4]byte{0xDE, 0xAD, 0xBE, 0xEF} {
		t.Errorf("invalid value")
		return
	}
}

func TestReadRegisterIndex(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("deadbeef")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	reg, err := g.ReadRegisterIndex(context.Background(), 0)
	if err != nil {
		t.Error(err)
		return
	}

	// Assert
	if reg.Definition != target.Arch.Registers[0] {
		t.Errorf("invalid register")
		return
	}

	if reg.Value == nil || len(reg.Value) != 4 {
		t.Errorf("invalid value")
		return
	}

	if [4]byte(reg.Value) != [4]byte{0xDE, 0xAD, 0xBE, 0xEF} {
		t.Errorf("invalid value")
		return
	}
}

func TestReadRegisters(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("deadbeefcafebabe")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	regs, err := g.ReadRegisters(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	// Assert
	{
		reg := regs[0]

		if reg.Definition != target.Arch.Registers[0] {
			t.Errorf("invalid register")
			return
		}

		if reg.Value == nil || len(reg.Value) != 4 {
			t.Errorf("invalid value")
			return
		}

		if [4]byte(reg.Value) != [4]byte{0xDE, 0xAD, 0xBE, 0xEF} {
			t.Errorf("invalid value")
			return
		}
	}

	{
		reg := regs[1]

		if reg.Definition != target.Arch.Registers[1] {
			t.Errorf("invalid register")
			return
		}

		if reg.Value == nil || len(reg.Value) != 4 {
			t.Errorf("invalid value")
			return
		}

		if [4]byte(reg.Value) != [4]byte{0xCA, 0xFE, 0xBA, 0xBE} {
			t.Errorf("invalid value")
			return
		}
	}
}

func TestWriteRegister(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	err := g.WriteRegister(context.Background(), core.RegisterValue{Definition: target.Arch.Registers[0], Value: []byte("\xDE\xAD\xBE\xEF")})

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

	if string(data) != "P0=deadbeef" {
		t.Errorf("invalid message")
		return
	}
}

func TestWriteRegisterIndex(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	// Act
	err := g.WriteRegisterIndex(context.Background(), 0, []byte("\xDE\xAD\xBE\xEF"))

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

	if string(data) != "P0=deadbeef" {
		t.Errorf("invalid message")
		return
	}
}

func TestWriteRegisterIndexFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	reg := []byte("\xDE\xAD\xBE\xEF")

	// Act
	err := g.WriteRegisterIndex(context.Background(), 0, reg)

	// Assert
	if err == nil {
		t.Error("no error")
		return
	}
}

func TestWriteRegisters(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("OK")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	regs := []core.RegisterValue{
		{
			Definition: target.Arch.Registers[0],
			Value:      []byte("\xDE\xAD\xBE\xEF"),
		},
		{
			Definition: target.Arch.Registers[1],
			Value:      []byte("\xCA\xFE\xBA\xBE"),
		},
	}

	// Act
	err := g.WriteRegisters(context.Background(), regs)

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

	if string(data) != "Gdeadbeefcafebabe" {
		t.Errorf("invalid request")
		return
	}
}

func TestWriteRegistersFail(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte("E1")),
	}

	conn := NewTestConnection(output)

	target := &gdb.Target{
		Name: "test",
		Arch: gdb.Architecture{
			Registers: []*core.Register{
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
				{
					Name:    "int32",
					BitSize: 32,
					Type:    "int32",
				},
			},
		},
	}

	g := gdb.GDBRSP{
		Conn:   conn,
		Target: target,
	}

	regs := []core.RegisterValue{
		{
			Definition: target.Arch.Registers[0],
			Value:      []byte("\xDE\xAD\xBE\xEF"),
		},
		{
			Definition: target.Arch.Registers[1],
			Value:      []byte("\xCA\xFE\xBA\xBE"),
		},
	}

	// Act
	err := g.WriteRegisters(context.Background(), regs)

	// Assert
	if err == nil {
		t.Errorf("no error")
		return
	}
}
