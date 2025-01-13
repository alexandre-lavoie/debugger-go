package gdb_test

import (
	"context"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/core"
	"github.com/alexandre-lavoie/debugger-go/gdb"
)

const TestFullTargetPacket = `x<?xml version="1.0"?>
	<target version="1.0">
		<architecture>test</architecture>
		<feature>
			<reg name="int32" bitsize="32" type="int32" regnum="0" />
		</feature>
	</target>
`

const TestPartialTargetPacket = `x<?xml version="1.0"?>
	<target version="1.0">
		<architecture>test</architecture>
		<xi:include href="arch.xml" />
	</target>
`

const TestArchPacket = `x<?xml version="1.0"?>
	<!DOCTYPE target SYSTEM "gdb-target.dtd">
	<feature>
		<reg name="int32" bitsize="32" type="int32" regnum="0" />
	</feature>
`

func TestQueryTargetFull(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte(TestFullTargetPacket)),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))

	// Act
	target, err := g.QueryTarget(context.Background())

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if target.Name != "test" {
		t.Errorf("invalid name")
		return
	}

	if len(target.Arch.Registers.List) != 1 {
		t.Errorf("invalid register count")
		return
	}

	r := core.Register{Name: "int32", BitSize: 32, Type: "int32"}
	if *target.Arch.Registers.GetIndexU(0) != r {
		t.Errorf("invalid registers")
		return
	}
}

func TestQueryTargetPartial(t *testing.T) {
	// Arrange
	output := [][]byte{
		[]byte("+"),
		gdb.BuildPacket([]byte(TestPartialTargetPacket)),
		[]byte("+"),
		gdb.BuildPacket([]byte(TestArchPacket)),
	}

	g := gdb.NewGDBRSP(NewTestConnection(output))

	// Act
	target, err := g.QueryTarget(context.Background())

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if target.Name != "test" {
		t.Errorf("invalid name")
		return
	}

	if len(target.Arch.Registers.List) != 1 {
		t.Errorf("invalid register count")
		return
	}

	r := core.Register{Name: "int32", BitSize: 32, Type: "int32"}
	if *target.Arch.Registers.GetIndexU(0) != r {
		t.Errorf("invalid registers")
		return
	}
}
