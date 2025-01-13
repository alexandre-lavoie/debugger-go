package gdb

import "github.com/alexandre-lavoie/debugger-go/core"

const (
	SignalReply    core.ReplyType = 'S'
	StatusReply    core.ReplyType = 'T'
	ExitReply      core.ReplyType = 'W'
	TerminateReply core.ReplyType = 'X'
	DataReply      core.ReplyType = 'O'
	InvalidReply   core.ReplyType = '!'
)
