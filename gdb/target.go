package gdb

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"

	"github.com/alexandre-lavoie/debugger-go/core"
)

type Target struct {
	Name string
	Path string
	Arch Architecture
}

type Architecture struct {
	Registers []*core.Register
	PC        *core.Register
}

func (arch *Architecture) PointerBitSize() uint {
	if len(arch.Registers) == 0 {
		return 64
	}

	if arch.PC != nil {
		return arch.PC.BitSize
	} else {
		return arch.Registers[0].BitSize
	}
}

func (arch *Architecture) FormatAddress(addr uint) string {
	ps := arch.PointerBitSize()

	format := fmt.Sprintf("%%0%dx", ps/4)
	return fmt.Sprintf(format, addr)
}

func ReadTarget(r io.Reader) (Target, error) {
	target := Target{}

	decoder := xml.NewDecoder(r)

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}

			return target, err
		}

		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "architecture":
				name, err := parseArchitectureName(decoder)
				if err != nil {
					return target, err
				}

				target.Name = name
			case "include":
				if el.Name.Space != "xi" {
					continue
				}

				path, err := parseInclude(el)
				if err != nil {
					return target, err
				}

				target.Path = path
			case "reg":
				reg, err := parseRegister(el)
				if err != nil {
					return target, nil
				}

				rptr := &reg

				reg.Index = uint(len(target.Arch.Registers))
				target.Arch.Registers = append(target.Arch.Registers, rptr)

				if reg.Type == "code_ptr" {
					target.Arch.PC = rptr
				}
			}
		}
	}

	return target, nil
}

func ReadArchitecture(r io.Reader) (Architecture, error) {
	arch := Architecture{}

	decoder := xml.NewDecoder(r)

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}

			return arch, err
		}

		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "reg":
				reg, err := parseRegister(el)
				if err != nil {
					return arch, nil
				}

				rptr := &reg

				reg.Index = uint(len(arch.Registers))
				arch.Registers = append(arch.Registers, rptr)

				if reg.Type == "code_ptr" {
					arch.PC = rptr
				}
			}
		}
	}

	return arch, nil
}

func parseInclude(el xml.StartElement) (string, error) {
	for _, attr := range el.Attr {
		switch attr.Name.Local {
		case "href":
			return attr.Value, nil
		}
	}

	return "", fmt.Errorf("invalid include")
}

func parseArchitectureName(decoder *xml.Decoder) (string, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", err
		}

		switch body := tok.(type) {
		case xml.CharData:
			return string(body[:]), nil
		}
	}
}

func parseRegister(el xml.StartElement) (core.Register, error) {
	reg := core.Register{}

	for _, attr := range el.Attr {
		switch attr.Name.Local {
		case "name":
			reg.Name = attr.Value
		case "bitsize":
			v, err := strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return reg, err
			}

			reg.BitSize = uint(v)
		case "type":
			reg.Type = attr.Value
		}
	}

	return reg, nil
}
