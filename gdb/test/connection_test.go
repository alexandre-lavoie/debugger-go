package gdb_test

import (
	"bytes"
	"io"
	"time"

	"github.com/alexandre-lavoie/debugger-go/gdb"
)

type TestConnection struct {
	Input  io.Reader
	Output io.ReadWriter
}

var _ gdb.GDBConnection = &TestConnection{}

func NewTestConnection(output [][]byte) *TestConnection {
	readers := make([]io.Reader, len(output))
	for i, o := range output {
		readers[i] = bytes.NewReader(o)
	}

	return &TestConnection{
		Input:  io.MultiReader(readers...),
		Output: new(bytes.Buffer),
	}
}

func (conn *TestConnection) SetDeadline(t time.Time) error {
	return nil
}

func (conn *TestConnection) Close() error {
	return nil
}

func (conn *TestConnection) Read(p []byte) (int, error) {
	n, err := conn.Input.Read(p)

	return n, err
}

func (conn *TestConnection) Write(p []byte) (int, error) {
	n, err := conn.Output.Write(p)

	return n, err
}
