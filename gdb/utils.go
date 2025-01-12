package gdb

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

func PacketChecksum(data []byte) uint8 {
	checksum := uint8(0)

	for _, c := range data {
		checksum += c
	}

	return checksum
}

func BuildPacket(data []byte) []byte {
	checksum := PacketChecksum(data)

	buffer := new(bytes.Buffer)

	buffer.Write([]byte("$"))
	buffer.Write(data)
	buffer.Write([]byte(fmt.Sprintf("#%02x", checksum)))

	return buffer.Bytes()
}

func WritePacket(w io.Writer, data []byte) error {
	_, err := w.Write(BuildPacket(data))

	return err
}

func ReadPacket(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)

	if _, err := br.ReadBytes('$'); err != nil {
		return nil, err
	}

	payload, err := br.ReadBytes('#')
	if err != nil {
		return nil, err
	}
	payload = payload[:len(payload)-1]

	{
		c0, err := br.ReadByte()
		if err != nil {
			return nil, err
		}

		c1, err := br.ReadByte()
		if err != nil {
			return nil, err
		}

		buffer := []byte{c0, c1}

		var checksum uint
		fmt.Sscanf(string(buffer[:]), "%x", &checksum)

		actual := PacketChecksum(payload)

		if uint8(checksum) != actual {
			fmt.Printf("%v != %v\n", checksum, actual)

			return nil, errors.New("invalid checksum")
		}
	}

	return payload, nil
}
