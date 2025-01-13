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
	"sync"
	"time"

	"github.com/alexandre-lavoie/debugger-go/core"
)

type GDBConnection interface {
	io.ReadWriteCloser

	SetDeadline(t time.Time) error
}

type GDBRSP struct {
	Conn GDBConnection

	Target *Target

	ThreadID    uint32
	Breakpoints core.Breakpoints

	Paused     bool
	Stopped    bool
	ExitCode   uint
	SignalCode uint

	connMutex sync.Mutex
}

var _ core.Debugger = &GDBRSP{}

func NewGDBRSP(conn GDBConnection) *GDBRSP {
	return &GDBRSP{
		Conn:        conn,
		Breakpoints: core.NewBreakpoints(),
	}
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

		gdb.Target = arch
	}

	return nil
}

func (gdb *GDBRSP) recvConnect(ctx context.Context) (core.ReplyType, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(500)*time.Millisecond)
	defer cancel()

	r, _ := gdb.WaitForReply(ctx)

	return r, nil
}

func (gdb *GDBRSP) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(500)*time.Millisecond)
	defer cancel()

	gdb.SendCommandAsync(ctx, "D")

	gdb.Conn.Close()
}

func (gdb *GDBRSP) Interrupt(ctx context.Context) error {
	gdb.connMutex.Lock()
	defer gdb.connMutex.Unlock()

	gdb.Paused = true

	if _, err := gdb.Conn.Write([]byte{0x3}); err != nil {
		return err
	}

	return nil
}

func (gdb *GDBRSP) Step(ctx context.Context) error {
	command := "s"

	res, err := gdb.SendCommandResponseAsync(ctx, command)
	if err != nil {
		return err
	}

	r, err := gdb.ProcessReply(ctx, res)
	if err != nil {
		return err
	}

	return gdb.Update(ctx, r)
}

func (gdb *GDBRSP) StepAt(ctx context.Context, address uint) error {
	command := fmt.Sprintf("s%s", gdb.Target.Arch.FormatAddress(address))

	res, err := gdb.SendCommandResponseAsync(ctx, command)
	if err != nil {
		return err
	}

	r, err := gdb.ProcessReply(ctx, res)
	if err != nil {
		return err
	}

	return gdb.Update(ctx, r)
}

func (gdb *GDBRSP) Continue(ctx context.Context) error {
	command := "c"

	res, err := gdb.SendCommandResponseAsync(ctx, command)
	if err != nil {
		return err
	}

	r, err := gdb.ProcessReply(ctx, res)
	if err != nil {
		return err
	}

	return gdb.Update(ctx, r)
}

func (gdb *GDBRSP) ContinueAt(ctx context.Context, address uint) error {
	command := fmt.Sprintf("c%s", gdb.Target.Arch.FormatAddress(address))

	res, err := gdb.SendCommandResponseAsync(ctx, command)
	if err != nil {
		return err
	}

	r, err := gdb.ProcessReply(ctx, res)
	if err != nil {
		return err
	}

	return gdb.Update(ctx, r)
}

func (gdb *GDBRSP) Run(ctx context.Context) error {
	for !gdb.Stopped {
		if err := gdb.Continue(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (gdb *GDBRSP) Update(ctx context.Context, r core.ReplyType) error {
	defer func() { gdb.Paused = false }()

	var oerr error

	switch r {
	case SignalReply:
		fallthrough
	case StatusReply:
		if gdb.Paused {
			for _, b := range gdb.Breakpoints.List {
				if b == nil {
					continue
				}

				if b.Type != InterruptBreakpoint {
					continue
				}

				if err := b.Handler(ctx, gdb); err != nil {
					oerr = errors.Join(oerr, err)
				}
			}
		} else if gdb.Target.Arch.PC != nil {
			val, err := gdb.ReadRegister(ctx, gdb.Target.Arch.PC)
			if err != nil {
				return err
			}

			ptr := val.ToUint()

			step := false

			for _, b := range gdb.Breakpoints.List {
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
					oerr = errors.Join(oerr, err)
				}
			}

			// Skip over breakpoint to prevent debugger from getting stuck
			if step {
				if _, err := gdb.SendCommandResponseSync(ctx, "s"); err != nil {
					return err
				}
			}
		}
	default:
	}

	return oerr
}

func (gdb *GDBRSP) AddSoftwareBreakpoint(ctx context.Context, address uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, SoftwareBreakpoint, address, 1, handler)
}

func (gdb *GDBRSP) AddHardwareBreakpoint(ctx context.Context, address uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, HardwareBreakpoint, address, 1, handler)
}

func (gdb *GDBRSP) AddInterruptBreakpoint(ctx context.Context, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, InterruptBreakpoint, 0, 0, handler)
}

func (gdb *GDBRSP) AddReadWatchpoint(ctx context.Context, address uint, length uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, ReadWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddWriteWatchpoint(ctx context.Context, address uint, length uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, WriteWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddAccessWatchpoint(ctx context.Context, address uint, length uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	return gdb.AddBreakpoint(ctx, AccessWatchpoint, address, length, handler)
}

func (gdb *GDBRSP) AddBreakpoint(ctx context.Context, t core.BreakpointType, address uint, byteLength uint, handler core.BreakpointHandler) (*core.Breakpoint, error) {
	b, update, err := gdb.Breakpoints.Add(t, address, byteLength, handler)
	if err != nil {
		return b, err
	}

	if update && b.Type != InterruptBreakpoint {
		command := fmt.Sprintf("Z%x,%s,%x", t, gdb.Target.Arch.FormatAddress(address), byteLength)

		res, err := gdb.SendCommandResponseSync(ctx, command)
		if err != nil {
			return b, err
		}

		if string(res[:2]) == "OK" {
			return b, nil
		} else if res[0] == 'E' {
			return b, fmt.Errorf("error %s", res[1:])
		} else {
			return b, fmt.Errorf("unimplemented")
		}
	}

	return b, nil
}

func (gdb *GDBRSP) RemoveBreakpoint(ctx context.Context, b *core.Breakpoint) error {
	update, err := gdb.Breakpoints.Remove(b)
	if err != nil {
		return err
	}

	if update && b.Type != InterruptBreakpoint {
		command := fmt.Sprintf("z%x,%s,%x", b.Type, gdb.Target.Arch.FormatAddress(b.Address), b.Length)

		res, err := gdb.SendCommandResponseSync(ctx, command)
		if err != nil {
			return err
		}

		if string(res[:2]) == "OK" {
			return nil
		} else if res[0] == 'E' {
			return fmt.Errorf("error %s", res[1:])
		} else {
			return fmt.Errorf("unimplemented")
		}
	}

	return nil
}

func (gdb *GDBRSP) WaitForReply(ctx context.Context) (core.ReplyType, error) {
	p, err := gdb.RecvResponse(ctx)
	if err != nil {
		return InvalidReply, err
	}

	return gdb.ProcessReply(ctx, p)
}

func (gdb *GDBRSP) ProcessReply(ctx context.Context, p []byte) (core.ReplyType, error) {
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

	return core.ReplyType(p[0]), nil
}

func (gdb *GDBRSP) QuerySupported(ctx context.Context) (GDBFeatures, error) {
	command := "qSupported"

	res, err := gdb.SendCommandResponseSync(ctx, command)
	if err != nil {
		return nil, err
	}

	features, err := ParseFeatures(string(res[:]))
	if err != nil {
		return nil, err
	}

	return features, nil
}

func (gdb *GDBRSP) QueryTargetPartial(ctx context.Context) (*Target, error) {
	res, err := gdb.ReadFeature(ctx, "target.xml")
	if err != nil {
		return nil, nil
	}

	target, err := ReadTarget(bytes.NewReader(res))
	if err != nil {
		return nil, err
	}

	if target.Name == "" {
		return nil, errors.New("no name")
	}

	return target, nil
}

func (gdb *GDBRSP) QueryTarget(ctx context.Context) (*Target, error) {
	target, err := gdb.QueryTargetPartial(ctx)
	if err != nil {
		return nil, err
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

		res, err := gdb.SendCommandResponseSync(ctx, command)
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

func (gdb *GDBRSP) GetRegister(name string) (*core.Register, error) {
	if reg, ok := gdb.Target.Arch.Registers.GetName(name); ok {
		return reg, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (gdb *GDBRSP) ReadRegister(ctx context.Context, reg *core.Register) (core.RegisterValue, error) {
	if r, ok := gdb.Target.Arch.Registers.GetIndex(reg.Index); ok {
		if r != reg {
			return core.RegisterValue{}, errors.New("register not in architecture")
		}
	} else {
		return core.RegisterValue{}, errors.New("register out of bounds")
	}

	command := fmt.Sprintf("p%x", reg.Index)

	res, err := gdb.SendCommandResponseSync(ctx, command)
	if err != nil {
		return core.RegisterValue{}, err
	}

	val, err := hex.DecodeString(string(res))
	if err != nil {
		return core.RegisterValue{}, err
	}

	return core.RegisterValue{Definition: reg, Value: val}, nil
}

func (gdb *GDBRSP) ReadRegisterIndex(ctx context.Context, index uint) (core.RegisterValue, error) {
	command := fmt.Sprintf("p%x", index)

	res, err := gdb.SendCommandResponseSync(ctx, command)
	if err != nil {
		return core.RegisterValue{}, err
	}

	val, err := hex.DecodeString(string(res))
	if err != nil {
		return core.RegisterValue{}, err
	}

	if gdb.Target == nil {
		return core.RegisterValue{Value: val}, nil
	}

	if reg, ok := gdb.Target.Arch.Registers.GetIndex(index); ok {
		return core.RegisterValue{Definition: reg, Value: val}, nil
	} else {
		return core.RegisterValue{Value: val}, nil
	}
}

func (gdb *GDBRSP) ReadRegisters(ctx context.Context) ([]core.RegisterValue, error) {
	raw, err := gdb.ReadRegistersRaw(ctx)
	if err != nil {
		return nil, err
	}

	return ReadRegisters(bytes.NewReader(raw), gdb.Target.Arch)
}

func (gdb *GDBRSP) ReadRegistersRaw(ctx context.Context) ([]byte, error) {
	command := "g"

	res, err := gdb.SendCommandResponseSync(ctx, command)
	if err != nil {
		return nil, err
	}

	raw, err := hex.DecodeString(string(res[:]))
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (gdb *GDBRSP) WriteRegister(ctx context.Context, regVal core.RegisterValue) error {
	if reg, ok := gdb.Target.Arch.Registers.GetIndex(regVal.Definition.Index); ok {
		if reg != regVal.Definition {
			return errors.New("register not in architecture")
		}
	} else {
		return errors.New("register out of bounds")
	}

	x := hex.EncodeToString(regVal.Value)

	command := fmt.Sprintf("P%x=%s", regVal.Definition.Index, x)

	res, err := gdb.SendCommandResponseSync(ctx, command)
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

func (gdb *GDBRSP) WriteRegisterIndex(ctx context.Context, index uint, data []byte) error {
	x := hex.EncodeToString(data)

	command := fmt.Sprintf("P%x=%s", index, x)

	res, err := gdb.SendCommandResponseSync(ctx, command)
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

func (gdb *GDBRSP) WriteRegisters(ctx context.Context, regs []core.RegisterValue) error {
	data := new(bytes.Buffer)

	if err := WriteRegisters(data, regs); err != nil {
		return err
	}

	x := hex.EncodeToString(data.Bytes())

	command := fmt.Sprintf("G%s", x)

	res, err := gdb.SendCommandResponseSync(ctx, command)
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

	res, err := gdb.SendCommandResponseSync(ctx, command)
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

	res, err := gdb.SendCommandResponseSync(ctx, command)
	if err != nil {
		return err
	}

	if string(res) == "OK" {
		return nil
	} else {
		return fmt.Errorf("error %s", res[1:])
	}
}

func (gdb *GDBRSP) SendCommandAsync(ctx context.Context, data string) error {
	gdb.connMutex.Lock()
	defer gdb.connMutex.Unlock()

	return gdb.SendCommandUnsafe(ctx, []byte(data))
}

func (gdb *GDBRSP) SendCommandResponseSync(ctx context.Context, data string) ([]byte, error) {
	gdb.connMutex.Lock()
	defer gdb.connMutex.Unlock()

	if err := gdb.SendCommandUnsafe(ctx, []byte(data)); err != nil {
		return nil, err
	}

	return gdb.RecvResponse(ctx)
}

func (gdb *GDBRSP) SendCommandResponseAsync(ctx context.Context, data string) ([]byte, error) {
	if err := gdb.SendCommandAsync(ctx, data); err != nil {
		return nil, err
	}

	return gdb.RecvResponse(ctx)
}

func (gdb *GDBRSP) SendCommandUnsafe(ctx context.Context, data []byte) error {
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
