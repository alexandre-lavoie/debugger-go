package gdb

import (
	"encoding/binary"
	"errors"
	"io"
)

type Register struct {
	Name    string
	BitSize uint
	Type    string
}

type RegisterValue struct {
	Definition *Register
	Value      []byte
}

func ReadRegisters(r io.Reader, architecture *Architecture) ([]RegisterValue, error) {
	if architecture == nil {
		return nil, errors.New("nil architecture")
	}

	registers := make([]RegisterValue, len(architecture.Registers))

	for i := range architecture.Registers {
		def := &architecture.Registers[i]

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
