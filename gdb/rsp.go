package gdb

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type GDBRSP struct {
	Conn GDBConnection

	Target *Target

	ThreadID    uint32
	Breakpoints []*Breakpoint

	Stopped    bool
	ExitCode   uint
	SignalCode uint
}

type GDBConnection interface {
	io.ReadWriteCloser

	SetDeadline(t time.Time) error
}

func (gdb *GDBRSP) init(ctx context.Context) error {
	_, err := gdb.recvConnect(ctx)
	if err != nil {
		return err
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

func (gdb *GDBRSP) recvConnect(ctx context.Context) (ReplyType, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(500)*time.Millisecond)
	defer cancel()

	r, _ := gdb.WaitReply(ctx)

	return r, nil
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

func (gdb *GDBRSP) Step(ctx context.Context) error {
	command := "s"

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) StepAt(ctx context.Context, address uint) error {
	command := fmt.Sprintf("s%s", gdb.Target.Arch.FormatAddress(address))

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

func (gdb *GDBRSP) Run(ctx context.Context) error {
	for !gdb.Stopped {
		r, err := gdb.WaitReply(ctx)
		if err != nil {
			return err
		}

		switch r {
		case SignalReply:
			fallthrough
		case StatusReply:
			if gdb.Target.Arch.PC != nil {
				val, err := gdb.ReadRegister(ctx, gdb.Target.Arch.PC)
				if err != nil {
					return err
				}

				ptr := val.ToUint()

				for _, b := range gdb.Breakpoints {
					if b == nil {
						continue
					}

					if b.Handler == nil {
						continue
					}

					if ptr < b.Address || ptr >= b.Address+b.Length {
						continue
					}

					if err := b.Handler(ctx, gdb); err != nil {
						return err
					}

					// Skip over breakpoint to prevent debugger from getting stuck
					switch b.Type {
					case SoftwareBreakpoint:
					case HardwareBreakpoint:
						if err := gdb.Step(ctx); err != nil {
							return err
						}

						if _, err := gdb.WaitReply(ctx); err != nil {
							return err
						}
					}

					break
				}
			}
		default:
		}

		if err := gdb.Continue(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (gdb *GDBRSP) AddSoftwareBreakpoint(ctx context.Context, address uint, handler BreakpointHandler) (*Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, SoftwareBreakpoint, address, 1, handler)
}

func (gdb *GDBRSP) AddHardwareBreakpoint(ctx context.Context, address uint, handler BreakpointHandler) (*Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, HardwareBreakpoint, address, 1, handler)
}

func (gdb *GDBRSP) AddReadWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, ReadWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddWriteWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, WriteWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddAccessWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, AccessWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddBreakpoint(ctx context.Context, t BreakpointType, address uint, byteLength uint, handler BreakpointHandler) (*Breakpoint, error) {
	command := fmt.Sprintf("Z%x,%s,%x", t, gdb.Target.Arch.FormatAddress(address), byteLength)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return nil, err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return nil, err
	}

	b := &Breakpoint{
		Type:    t,
		Address: address,
		Length:  byteLength,
		Handler: handler,
	}

	if string(res[:2]) == "OK" {
		gdb.Breakpoints = append(gdb.Breakpoints, b)

		return b, nil
	} else if res[0] == 'E' {
		return nil, fmt.Errorf("error %s", res[1:])
	} else {
		return nil, fmt.Errorf("unimplemented")
	}
}

func (gdb *GDBRSP) RemoveBreakpoint(ctx context.Context, b *Breakpoint) error {
	idx := 0
	found := false

	for i, bn := range gdb.Breakpoints {
		if bn == b {
			idx = i
			break
		}
	}

	if found {
		return errors.New("not found")
	}

	command := fmt.Sprintf("z%x,%s,%x", b.Type, gdb.Target.Arch.FormatAddress(b.Address), b.Length)

	if err := gdb.SendCommand(ctx, command); err != nil {
		return err
	}

	res, err := gdb.RecvResponse(ctx)
	if err != nil {
		return err
	}

	if string(res[:2]) == "OK" {
		gdb.Breakpoints[idx] = nil

		return nil
	} else if res[0] == 'E' {
		return fmt.Errorf("error %s", res[1:])
	} else {
		return fmt.Errorf("unimplemented")
	}
}

func (gdb *GDBRSP) WaitReply(ctx context.Context) (ReplyType, error) {
	p, err := gdb.RecvResponse(ctx)
	if err != nil {
		return InvalidReply, err
	}

	return gdb.ProcessReply(ctx, p)
}

func (gdb *GDBRSP) ProcessReply(ctx context.Context, p []byte) (ReplyType, error) {
	if len(p) == 0 {
		return InvalidReply, errors.New("invalid length")
	}

	switch p[0] {
	case byte(SignalReply):
		if _, err := fmt.Sscanf(string(p[1:3]), "%x", &gdb.SignalCode); err != nil {
			return InvalidReply, err
		}
	case byte(StatusReply):
		if _, err := fmt.Sscanf(string(p[1:3]), "%x", &gdb.SignalCode); err != nil {
			return InvalidReply, err
		}

		parts := strings.Split(string(p[3:]), ";")

		for _, part := range parts {
			if !strings.Contains(part, ":") {
				continue
			}

			parts := strings.Split(part, ":")

			key, hv := parts[0], parts[1]

			if key == "thread" {
				if _, err := fmt.Sscanf(hv, "%x", &gdb.ThreadID); err != nil {
					return InvalidReply, err
				}
			} else {
				// TODO: Handle registers
			}
		}
	case byte(ExitReply):
		gdb.Stopped = true

		if _, err := fmt.Sscanf(string(p[1:3]), "%x", &gdb.ExitCode); err != nil {
			return InvalidReply, err
		}
	case byte(TerminateReply):
		gdb.Stopped = true

		if _, err := fmt.Sscanf(string(p[1:3]), "%x", &gdb.SignalCode); err != nil {
			return InvalidReply, err
		}
	case byte(DataReply):
		// TODO: Is this required?
	default:
		return InvalidReply, errors.New("unhandled reply")
	}

	return ReplyType(p[0]), nil
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

func (gdb *GDBRSP) ReadRegister(ctx context.Context, reg *Register) (RegisterValue, error) {
	if reg.Index >= uint(len(gdb.Target.Arch.Registers)) {
		return RegisterValue{}, errors.New("register out of bounds")
	}

	if gdb.Target.Arch.Registers[reg.Index] != reg {
		return RegisterValue{}, errors.New("register not in architecture")
	}

	command := fmt.Sprintf("p%x", reg.Index)

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

	return RegisterValue{Definition: reg, Value: val}, nil
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

	return RegisterValue{Definition: gdb.Target.Arch.Registers[index], Value: val}, nil
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
