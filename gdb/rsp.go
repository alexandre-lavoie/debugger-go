package gdb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

type GDBRSP struct {
	Conn GDBConnection

	Arch *Architecture
}

type GDBConnection interface {
	io.ReadWriteCloser

	SetDeadline(t time.Time) error
}

func (gdb *GDBRSP) init(ctx context.Context) error {
	res, err := gdb.recvConnect(ctx)
	if err == nil {
		// TODO: Handle initial data
		_ = res
	}

	{
		arch, err := gdb.QueryArchitecture(ctx)
		if err != nil {
			return err
		}

		gdb.Arch = &arch
	}

	return nil
}

func (gdb *GDBRSP) recvConnect(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(500)*time.Millisecond)
	defer cancel()

	return gdb.RecvResponse(ctx)
}

func (gdb *GDBRSP) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(500)*time.Millisecond)
	defer cancel()

	gdb.SendCommand(ctx, "D")

	gdb.Conn.Close()
}

func (gdb *GDBRSP) Interrupt(ctx context.Context) error {
	command := "\x03"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) Continue(ctx context.Context) error {
	command := "c"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) ContinueAt(ctx context.Context, address uint) error {
	command := fmt.Sprintf("c%s", gdb.Arch.FormatAddress(address))

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) QuerySupported(ctx context.Context) (GDBFeatures, error) {
	command := "qSupported"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return nil, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return nil, err
	}

	features, err := ParseFeatures(string(res[:]))
	if err != nil {
		return nil, err
	}

	return features, nil
}

func (gdb *GDBRSP) QueryTarget(ctx context.Context) (Target, error) {
	command := "qXfer:features:read:target.xml:0,fff"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return Target{}, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return Target{}, err
	}

	return ReadTarget(bytes.NewReader(res))
}

func (gdb *GDBRSP) QueryArchitecture(ctx context.Context) (Architecture, error) {
	target, err := gdb.QueryTarget(ctx)
	if err != nil {
		return Architecture{}, err
	}

	if target.Path == "" {
		return Architecture{}, errors.New("no architecture file path")
	}

	buffer := new(bytes.Buffer)

	size := 0x700
	for offset := 0; ; offset += size {
		command := fmt.Sprintf("qXfer:features:read:%s:%x,%x", target.Path, offset, size)

		if err := gdb.SendCommand(ctx, command); err != nil {
			return Architecture{}, err
		}

		res, err := gdb.RecvResponse(ctx)
		if err != nil {
			return Architecture{}, err
		}

		// TODO: Is there always an extra byte at the start?
		buffer.Write(res[1:])

		if len(res) < size {
			break
		}
	}

	raw := buffer.Bytes()

	arch, err := ReadArchitecture(bytes.NewReader(raw))

	arch.Name = target.Name

	return arch, err
}

func (gdb *GDBRSP) ReadRegistersRaw(ctx context.Context) ([]byte, error) {
	command := "g"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return nil, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := hex.DecodeString(string(res[:]))
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (gdb *GDBRSP) ReadRegisters(ctx context.Context) ([]RegisterValue, error) {
	raw, err := gdb.ReadRegistersRaw(ctx)
	if err != nil {
		return nil, err
	}

	return ReadRegisters(bytes.NewReader(raw), gdb.Arch)
}

func (gdb *GDBRSP) ReadMemory(ctx context.Context, address uint, length uint) ([]byte, error) {
	command := fmt.Sprintf("m%s,%x", gdb.Arch.FormatAddress(address), length)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return nil, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := hex.DecodeString(string(res[:]))
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (gdb *GDBRSP) Wait(ctx context.Context) error {
	_, err := gdb.RecvResponse(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) SendCommand(ctx context.Context, data string) error {
	return gdb.SendRawCommand(ctx, []byte(data))
}

func (gdb *GDBRSP) SendRawCommand(ctx context.Context, data []byte) error {
	if t, ok := ctx.Deadline(); ok {
		gdb.Conn.SetDeadline(t)
	} else {
		gdb.Conn.SetDeadline(time.Time{})
	}

	checksum := PayloadChecksum(data)

	buffer := new(bytes.Buffer)

	buffer.Write([]byte("$"))
	buffer.Write(data)
	buffer.Write([]byte(fmt.Sprintf("#%02x", checksum)))

	packet := buffer.Bytes()

	for {
		if _, err := gdb.Conn.Write(packet); err != nil {
			return err
		}

		var res byte
		if err := binary.Read(gdb.Conn, binary.LittleEndian, &res); err != nil {
			return err
		}

		if res == '+' {
			break
		}

		if res != '-' {
			return errors.New("bad response")
		}
	}

	return nil
}

func (gdb *GDBRSP) RecvResponse(ctx context.Context) ([]byte, error) {
	if t, ok := ctx.Deadline(); ok {
		gdb.Conn.SetDeadline(t)
	} else {
		gdb.Conn.SetDeadline(time.Time{})
	}

	r := bufio.NewReader(gdb.Conn)

	if _, err := r.ReadBytes('$'); err != nil {
		return nil, err
	}

	payload, err := r.ReadBytes('#')
	if err != nil {
		return nil, err
	}
	payload = payload[:len(payload)-1]

	{
		c0, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		c1, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		buffer := []byte{c0, c1}

		var checksum uint
		fmt.Sscanf(string(buffer[:]), "%x", &checksum)

		actual := PayloadChecksum(payload)

		if uint8(checksum) != actual {
			fmt.Printf("%v != %v\n", checksum, actual)

			return nil, errors.New("invalid checksum")
		}
	}

	if _, err := gdb.Conn.Write([]byte{'+'}); err != nil {
		return nil, err
	}

	return payload, nil
}
