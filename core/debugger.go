package core

import "context"

type StaticDebugger interface {
	AddSoftwareBreakpoint(ctx context.Context, address uint, handler BreakpointHandler) (*Breakpoint, error)
	AddHardwareBreakpoint(ctx context.Context, address uint, handler BreakpointHandler) (*Breakpoint, error)
	AddInterruptBreakpoint(ctx context.Context, handler BreakpointHandler) (*Breakpoint, error)

	AddReadWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error)
	AddWriteWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error)
	AddAccessWatchpoint(ctx context.Context, address uint, length uint, handler BreakpointHandler) (*Breakpoint, error)

	RemoveBreakpoint(ctx context.Context, b *Breakpoint) error

	GetRegister(name string) (*Register, error)

	ReadRegister(ctx context.Context, reg *Register) (RegisterValue, error)
	ReadRegisters(ctx context.Context) ([]RegisterValue, error)

	WriteRegister(ctx context.Context, regVal RegisterValue) error
	WriteRegisters(ctx context.Context, regVals []RegisterValue) error

	ReadMemory(ctx context.Context, address uint, length uint) ([]byte, error)
	WriteMemory(ctx context.Context, address uint, data []byte) error
}

type Debugger interface {
	StaticDebugger

	Close()

	Interrupt(ctx context.Context) error

	Step(ctx context.Context) error
	StepAt(ctx context.Context, address uint) error

	Continue(ctx context.Context) error
	ContinueAt(ctx context.Context, address uint) error

	Run(ctx context.Context) error

	WaitForReply(ctx context.Context) (ReplyType, error)
}
