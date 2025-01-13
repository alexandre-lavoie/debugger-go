package core

import "context"

type BreakpointType uint8

type Breakpoint struct {
	Type    BreakpointType
	Address uint
	Length  uint
	Handler BreakpointHandler
}

type BreakpointHandler func(ctx context.Context, dbg Debugger) error
