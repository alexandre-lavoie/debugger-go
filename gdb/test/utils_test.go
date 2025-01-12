package gdb_test

import (
	"bytes"
	"testing"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

func TestPacketChecksum(t *testing.T) {
	// Arrange
	data := []byte("test")

	// Act
	check := gdb.PacketChecksum(data)

	// Assert
	if check != 0xc0 {
		t.Errorf("invalid checksum")
		return
	}
}

func TestReadPacket(t *testing.T) {
	// Arrange
	data := []byte("$test#c0")

	r := bytes.NewReader(data)

	// Act
	raw, err := gdb.ReadPacket(r)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if string(raw) != "test" {
		t.Errorf("invalid data")
		return
	}
}

func TestWritePacket(t *testing.T) {
	// Arrange
	data := []byte("test")

	w := new(bytes.Buffer)

	// Act
	err := gdb.WritePacket(w, data)

	// Assert
	if err != nil {
		t.Error(err)
		return
	}

	if string(w.Bytes()) != "$test#c0" {
		t.Errorf("invalid packet")
		return
	}
}
