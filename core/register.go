package core

import (
	"fmt"
	"sync"
)

type Register struct {
	Name    string
	Index   uint
	BitSize uint
	Type    string
}

type Registers struct {
	List []*Register
	Map  map[string]*Register

	addMutex sync.Mutex
}

func NewRegisters() Registers {
	return Registers{
		Map: make(map[string]*Register),
	}
}

func (r *Registers) Add(name string, bitSize uint, t string) (*Register, error) {
	r.addMutex.Lock()
	defer r.addMutex.Unlock()

	reg := &Register{
		Name:    name,
		Index:   uint(len(r.List)),
		BitSize: bitSize,
		Type:    t,
	}

	r.List = append(r.List, reg)
	r.Map[reg.Name] = reg

	return reg, nil
}

func (r *Registers) GetName(name string) (*Register, bool) {
	v, ok := r.Map[name]

	return v, ok
}

func (r *Registers) GetIndex(idx uint) (*Register, bool) {
	if idx >= uint(len(r.List)) {
		return nil, false
	}

	return r.List[idx], true
}

func (r *Registers) GetIndexU(idx uint) *Register {
	return r.List[idx]
}

type RegisterValue struct {
	Definition *Register
	Value      []byte
}

func (r *RegisterValue) ToUint() uint {
	o := uint(0)

	for i := len(r.Value) - 1; i >= 0; i-- {
		o <<= 8
		o |= uint(r.Value[i])
	}

	return o
}

func (r *RegisterValue) ToString() string {
	switch r.Definition.Type {
	case "int32":
		fallthrough
	case "int":
		fallthrough
	case "code_ptr":
		fallthrough
	case "data_ptr":
		return fmt.Sprintf("%x", r.ToUint())
	default:
		return fmt.Sprintf("%v", r.Value)
	}
}
