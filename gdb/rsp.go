package gdb

import (
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

	Target *Target
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
		arch, err := gdb.QueryTarget(ctx)
		if err != nil {
			return err
		}

		gdb.Target = &arch
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
	command := fmt.Sprintf("c%s", gdb.Target.Arch.FormatAddress(address))

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

func (gdb *GDBRSP) QueryTargetPartial(ctx context.Context) (Target, error) {
	res, err := gdb.ReadFeature(ctx, "target.xml")
	if err != nil {
		return Target{}, nil
	}

	target, err := ReadTarget(bytes.NewReader(res))
	if err != nil {
		return target, err
	}

	if target.Name == "" {
		return target, errors.New("no name")
	}

	return target, nil
}

func (gdb *GDBRSP) QueryTarget(ctx context.Context) (Target, error) {
	target, err := gdb.QueryTargetPartial(ctx)
	if err != nil {
		return Target{}, err
	}

	if target.Path == "" {
		return target, nil
	}

	raw, err := gdb.ReadFeature(ctx, target.Path)
	if err != nil {
		return target, err
	}

	arch, err := ReadArchitecture(bytes.NewReader(raw))
	target.Arch = arch

	return target, err
}

func (gdb *GDBRSP) ReadFeature(ctx context.Context, path string) ([]byte, error) {
	buffer := new(bytes.Buffer)

	size := 0x700
	for offset := 0; ; offset += size {
		command := fmt.Sprintf("qXfer:features:read:%s:%x,%x", path, offset, size)

		if err := gdb.SendCommand(ctx, command); err != nil {
			return nil, err
		}

		res, err := gdb.RecvResponse(ctx)
		if err != nil {
			return nil, err
		}

		// TODO: Not sure what is the extra bit.
		buffer.Write(res[1:])

		if len(res) < size {
			break
		}
	}

	return buffer.Bytes(), nil
}

func (gdb *GDBRSP) ReadRegisterIndex(ctx context.Context, index uint) (RegisterValue, error) {
	command := fmt.Sprintf("p%x", index)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return RegisterValue{}, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return RegisterValue{}, err
	}

	val, err := hex.DecodeString(string(res))
	if err != nil {
		return RegisterValue{}, err
	}

	if gdb.Target == nil || uint(len(gdb.Target.Arch.Registers)) <= index {
		return RegisterValue{Value: val}, nil
	}

	return RegisterValue{Definition: &gdb.Target.Arch.Registers[index], Value: val}, nil
}

func (gdb *GDBRSP) ReadRegisters(ctx context.Context) ([]RegisterValue, error) {
	raw, err := gdb.ReadRegistersRaw(ctx)
	if err != nil {
		return nil, err
	}

	return ReadRegisters(bytes.NewReader(raw), &gdb.Target.Arch)
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

func (gdb *GDBRSP) WriteRegisterIndex(ctx context.Context, index uint, data []byte) error {
	x := hex.EncodeToString(data)

	command := fmt.Sprintf("P%x=%s", index, x)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return err
	}

	resd := string(res)
	if resd == "OK" {
		return nil
	} else {
		return errors.New(fmt.Sprintf("failed: %s", resd[1:]))
	}
}

func (gdb *GDBRSP) WriteRegisters(ctx context.Context, regs []RegisterValue) error {
	data := new(bytes.Buffer)

	if err := WriteRegisters(data, regs); err != nil {
		return err
	}

	x := hex.EncodeToString(data.Bytes())

	command := fmt.Sprintf("G%s", x)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return err
	}

	if string(res) == "OK" {
		return nil
	} else {
		return fmt.Errorf("error %s", res[1:])
	}
}

func (gdb *GDBRSP) ReadMemory(ctx context.Context, address uint, length uint) ([]byte, error) {
	command := fmt.Sprintf("m%s,%x", gdb.Target.Arch.FormatAddress(address), length)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return nil, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return nil, err
	}

	if len(res) > 0 && res[0] == 'E' {
		return nil, fmt.Errorf("error %s", res[1:])
	}

	raw, err := hex.DecodeString(string(res[:]))
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (gdb *GDBRSP) WriteMemory(ctx context.Context, address uint, data []byte) error {
	x := hex.EncodeToString(data)

	command := fmt.Sprintf("M%s,%x:%s", gdb.Target.Arch.FormatAddress(address), len(data), x)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return err
	}

	if string(res) == "OK" {
		return nil
	} else {
		return fmt.Errorf("error %s", res[1:])
	}
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

	for {
		if err := WritePacket(gdb.Conn, data); err != nil {
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

	payload, err := ReadPacket(gdb.Conn)
	if err != nil {
		return nil, err
	}

	if _, err := gdb.Conn.Write([]byte{'+'}); err != nil {
		return nil, err
	}

	return payload, nil
}
