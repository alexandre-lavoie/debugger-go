package gdb

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

type Target struct {
	Name string
	Path string
}

type Architecture struct {
	Name      string
	Registers []Register
}

func (arch *Architecture) PointerBitSize() uint {
	if len(arch.Registers) == 0 {
		return 64
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
			case "target":
				for {
					tok, err := decoder.Token()
					if err != nil {
						return target, err
					}

					found := false

					switch body := tok.(type) {
					case xml.CharData:
						target.Name = string(body[:])
						found = true
					}

					if found {
						break
					}
				}
			case "include":
				if el.Name.Space != "xi" {
					continue
				}

				for _, attr := range el.Attr {
					switch attr.Name.Local {
					case "href":
						target.Path = attr.Value
					}
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
				reg := Register{}

				for _, attr := range el.Attr {
					switch attr.Name.Local {
					case "name":
						reg.Name = attr.Value
					case "bitsize":
						v, err := strconv.ParseUint(attr.Value, 10, 64)
						if err != nil {
							return arch, err
						}

						reg.BitSize = uint(v)
					case "type":
						reg.Type = attr.Value
					}
				}

				arch.Registers = append(arch.Registers, reg)
			}
		}
	}

	return arch, nil
}
