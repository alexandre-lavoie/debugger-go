package gdb

func PayloadChecksum(data []byte) uint8 {
	checksum := uint8(0)

	for _, c := range data {
		checksum += c
	}

	return checksum
}
