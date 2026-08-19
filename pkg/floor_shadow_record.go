package pkg

// FloorShadowRecord represents a floor or shadow record in a DS1 file.
type FloorShadowRecord struct {
	Prop1       byte
	Sequence    byte
	Unknown1    byte
	Style       byte
	Unknown2    byte
	Hidden      bool
	RandomIndex byte
	Animated    bool
	YAdjust     int
}

// Packed returns the four serialized property bytes as one little-endian
// dword. Runtime-only rendering fields are intentionally excluded.
func (record FloorShadowRecord) Packed() uint32 {
	return packCell(record.Prop1, record.Sequence, record.Unknown1, record.Style, record.Unknown2, record.Hidden)
}

// SetPacked replaces the serialized properties from a little-endian dword.
func (record *FloorShadowRecord) SetPacked(value uint32) {
	record.Prop1, record.Sequence, record.Unknown1, record.Style, record.Unknown2, record.Hidden = unpackCell(value)
}
