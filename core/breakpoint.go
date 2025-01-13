package core

import (
	"context"
	"errors"
	"sync"
)

type BreakpointType uint8

type BreakpointSeenEntry struct {
	Type    BreakpointType
	Address uint
	Length  uint
}

type BreakpointHandler func(ctx context.Context, dbg StaticDebugger) error

type Breakpoint struct {
	Type    BreakpointType
	Index   uint
	Address uint
	Length  uint
	Handler BreakpointHandler
}

type Breakpoints struct {
	List []*Breakpoint

	seen        map[BreakpointSeenEntry]uint
	addMutex    sync.Mutex
	removeMutex sync.Mutex
}

func NewBreakpoints() Breakpoints {
	return Breakpoints{
		seen: make(map[BreakpointSeenEntry]uint),
	}
}

func (b *Breakpoints) Add(t BreakpointType, address uint, length uint, handler BreakpointHandler) (*Breakpoint, bool, error) {
	b.addMutex.Lock()
	defer b.addMutex.Unlock()

	br := &Breakpoint{
		Type:    t,
		Index:   uint(len(b.List)),
		Address: address,
		Length:  length,
		Handler: handler,
	}

	b.List = append(b.List, br)

	e := BreakpointSeenEntry{
		Type:    br.Type,
		Address: br.Address,
		Length:  br.Length,
	}

	if count, ok := b.seen[e]; ok {
		b.seen[e] += 1

		return br, count == 0, nil
	} else {
		b.seen[e] = 1

		return br, true, nil
	}
}

func (b *Breakpoints) Remove(br *Breakpoint) (bool, error) {
	b.removeMutex.Lock()
	defer b.removeMutex.Unlock()

	if br.Index >= uint(len(b.List)) {
		return false, errors.New("out of bounds")
	}

	if br != b.List[br.Index] {
		return false, errors.New("invalid breakpoint")
	}

	e := BreakpointSeenEntry{
		Type:    br.Type,
		Address: br.Address,
		Length:  br.Length,
	}

	b.List[br.Index] = nil

	if count, ok := b.seen[e]; ok {
		b.seen[e] -= 1

		return count <= 1, nil
	} else {
		panic("unreachable")
	}
}

func (b *Breakpoints) GetIndex(idx uint) (*Breakpoint, bool) {
	if idx >= uint(len(b.List)) {
		return nil, false
	}

	return b.List[idx], true
}

func (b *Breakpoints) GetIndexU(idx uint) *Breakpoint {
	return b.List[idx]
}
