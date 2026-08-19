package pkg

// WallRecord represents a wall record.
type WallRecord struct {
	Type               TileType
	Zero               byte   // Legacy alias for the first unknown orientation byte
	RawOrientation     byte   // Serialized orientation before the pre-v7 lookup is applied
	OrientationUnknown uint32 // Upper 24 bits of the serialized orientation dword
	Prop1              byte
	Sequence           byte
	Unknown1           byte
	Style              byte
	Unknown2           byte
	Hidden             bool
	RandomIndex        byte
	YAdjust            int
}

// Packed returns the four serialized wall-property bytes as one little-endian
// dword. Orientation and runtime-only rendering fields are excluded.
func (record WallRecord) Packed() uint32 {
	return packCell(record.Prop1, record.Sequence, record.Unknown1, record.Style, record.Unknown2, record.Hidden)
}

// SetPacked replaces the serialized wall properties from a little-endian dword.
func (record *WallRecord) SetPacked(value uint32) {
	record.Prop1, record.Sequence, record.Unknown1, record.Style, record.Unknown2, record.Hidden = unpackCell(value)
}
