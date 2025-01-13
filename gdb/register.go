package gdb

import (
	"encoding/binary"
	"errors"
	"io"
)

type Register struct {
	Name    string
	Index   uint
	BitSize uint
	Type    string
}

type RegisterValue struct {
	Definition *Register
	Value      []byte
}

func (r *RegisterValue) ToUint() uint {
	o := uint(0)

	for i := len(r.Value) - 1; i >= 0; i-- {
		o <<= 8
		o &= uint(r.Value[i])
	}

	return o
}

func ReadRegisters(r io.Reader, architecture *Architecture) ([]RegisterValue, error) {
	if architecture == nil {
		return nil, errors.New("nil architecture")
	}

	registers := make([]RegisterValue, len(architecture.Registers))

	for i, def := range architecture.Registers {
		reg, err := ReadRegister(r, def)
		if err != nil {
			return nil, err
		}

		registers[i] = reg
	}

	return registers, nil
}

func ReadRegister(r io.Reader, definition *Register) (RegisterValue, error) {
	if definition == nil {
		return RegisterValue{}, errors.New("nil definition")
	}

	reg := RegisterValue{Definition: definition}

	reg.Value = make([]byte, reg.Definition.BitSize/8)

	if err := binary.Read(r, binary.LittleEndian, &reg.Value); err != nil {
		return reg, err
	}

	return reg, nil
}

func WriteRegisters(w io.Writer, regs []RegisterValue) error {
	for _, reg := range regs {
		if err := WriteRegister(w, &reg); err != nil {
			return err
		}
	}

	return nil
}

func WriteRegister(w io.Writer, reg *RegisterValue) error {
	if _, err := w.Write(reg.Value); err != nil {
		return err
	}

	return nil
}
