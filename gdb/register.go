package gdb

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/alexandre-lavoie/debugger-go/core"
)

func ReadRegisters(r io.Reader, architecture *Architecture) ([]core.RegisterValue, error) {
	if architecture == nil {
		return nil, errors.New("nil architecture")
	}

	registers := make([]core.RegisterValue, len(architecture.Registers))

	for i, def := range architecture.Registers {
		reg, err := ReadRegister(r, def)
		if err != nil {
			return nil, err
		}

		registers[i] = reg
	}

	return registers, nil
}

func ReadRegister(r io.Reader, definition *core.Register) (core.RegisterValue, error) {
	if definition == nil {
		return core.RegisterValue{}, errors.New("nil definition")
	}

	reg := core.RegisterValue{Definition: definition}

	reg.Value = make([]byte, reg.Definition.BitSize/8)

	if err := binary.Read(r, binary.LittleEndian, &reg.Value); err != nil {
		return reg, err
	}

	return reg, nil
}

func WriteRegisters(w io.Writer, regs []core.RegisterValue) error {
	for _, reg := range regs {
		if err := WriteRegister(w, &reg); err != nil {
			return err
		}
	}

	return nil
}

func WriteRegister(w io.Writer, reg *core.RegisterValue) error {
	if _, err := w.Write(reg.Value); err != nil {
		return err
	}

	return nil
}
