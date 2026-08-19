package pkg

var legacyOrientationLookup = [...]byte{
	0x00, 0x01, 0x02, 0x01, 0x02, 0x03, 0x03, 0x05, 0x05, 0x06,
	0x06, 0x07, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
	0x0f, 0x10, 0x11, 0x12, 0x14,
}

func packCell(prop1, sequence, unknown1, style, unknown2 byte, hidden bool) uint32 {
	value := uint32(prop1) |
		uint32(sequence)<<8 |
		uint32(unknown1)<<14 |
		uint32(style)<<20 |
		uint32(unknown2)<<26
	if hidden {
		value |= 1 << 31
	}
	return value
}

func unpackCell(value uint32) (prop1, sequence, unknown1, style, unknown2 byte, hidden bool) {
	return byte(value),
		byte(value >> 8 & 0x3f),
		byte(value >> 14 & 0x3f),
		byte(value >> 20 & 0x3f),
		byte(value >> 26 & 0x1f),
		value>>31 != 0
}
