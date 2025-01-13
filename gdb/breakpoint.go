package gdb

import "context"

type BreakpointType uint8

const (
	SoftwareBreakpoint = iota
	HardwareBreakpoint
	WriteWatchpoint
	ReadWatchpoint
	AccessWatchpoint
)

type BreakpointHandler func(ctx context.Context, g *GDBRSP) error

type Breakpoint struct {
	Type    BreakpointType
	Address uint
	Length  uint
	Handler BreakpointHandler
}
