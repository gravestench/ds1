package pkg

// WallRecord represents a wall record.
type WallRecord struct {
	Type        TileType
	Zero        byte
	Prop1       byte
	Sequence    byte
	Unknown1    byte
	Style       byte
	Unknown2    byte
	Hidden      bool
	RandomIndex byte
	YAdjust     int
}
