package gdb

type ReplyType uint8

const (
	SignalReply    ReplyType = 'S'
	StatusReply    ReplyType = 'T'
	ExitReply      ReplyType = 'W'
	TerminateReply ReplyType = 'X'
	DataReply      ReplyType = 'O'
	InvalidReply   ReplyType = '!'
)
