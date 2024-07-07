package main

import "encoding/binary"

func uuid2ushard(uuid [16]byte, TOTAL_USHARDS uint16) int16 {
	input := binary.BigEndian.Uint16(uuid[14:16])
	// TODO: Pass USHARDS_TOTAL dynamically
	remainder := input % TOTAL_USHARDS
	// Clamp to 0..int16
	remainder = remainder & 0x7FFF
	return int16(remainder)
}
