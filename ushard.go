package main

import "encoding/binary"

const USHARDS_TOTAL uint16 = 8

func uuid2ushard(uuid [16]byte) int16 {
	input := binary.BigEndian.Uint16(uuid[14:16])
	// TODO: Pass USHARDS_TOTAL dynamically
	remainder := input % USHARDS_TOTAL
	// Clamp to 0..int16
	remainder = remainder & 0x7FFF
	return int16(remainder)
}
