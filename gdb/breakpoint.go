package gdb

import "github.com/alexandre-lavoie/debugger-go/core"

const (
	SoftwareBreakpoint core.BreakpointType = iota
	HardwareBreakpoint
	WriteWatchpoint
	ReadWatchpoint
	AccessWatchpoint
	InterruptBreakpoint = 0xFF
)
