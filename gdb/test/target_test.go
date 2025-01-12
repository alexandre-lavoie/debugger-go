package gdb_test

import (
	"context"
	"testing"

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

	conn := NewTestConnection(output)

	g := gdb.GDBRSP{
		Conn: conn,
	}

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

	if len(target.Arch.Registers) != 1 {
		t.Errorf("invalid register count")
		return
	}

	if [1]gdb.Register(target.Arch.Registers) != [1]gdb.Register{{Name: "int32", BitSize: 32, Type: "int32"}} {
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

	conn := NewTestConnection(output)

	g := gdb.GDBRSP{
		Conn: conn,
	}

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

	if len(target.Arch.Registers) != 1 {
		t.Errorf("invalid register count")
		return
	}

	if [1]gdb.Register(target.Arch.Registers) != [1]gdb.Register{{Name: "int32", BitSize: 32, Type: "int32"}} {
		t.Errorf("invalid registers")
		return
	}
}
