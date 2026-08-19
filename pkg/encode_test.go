package pkg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gravestench/mathlib"
)

func TestEncodeMatchesDS1EditVersion18Layout(t *testing.T) {
	wall := WallRecord{Type: TileSpecialTile1}
	wall.SetPacked(0x44332211)
	floor := FloorShadowRecord{}
	floor.SetPacked(0x88776655)
	shadow := FloorShadowRecord{}
	shadow.SetPacked(0xccbbaa99)

	value := &DS1{
		Version:                    LatestVersion,
		Width:                      1,
		Height:                     1,
		Act:                        3,
		SubstitutionType:           1,
		Files:                      []string{"a.dt1"},
		NumberOfWalls:              1,
		NumberOfFloors:             1,
		NumberOfShadowLayers:       1,
		NumberOfSubstitutionLayers: 1,
		Tiles: [][]TileRecord{{{
			Walls:         []WallRecord{wall},
			Floors:        []FloorShadowRecord{floor},
			Shadows:       []FloorShadowRecord{shadow},
			Substitutions: []SubstitutionRecord{{Unknown: 0x12345678}},
		}}},
		Objects: []Object{{
			Type: 1, ID: 2, X: 15, Y: 20, Flags: 0x44,
			Paths: []Path{{Position: *mathlib.NewVector2(25, 30), Action: 7}},
		}},
		SubstitutionGroups: []SubstitutionGroup{{
			TileX: 4, TileY: 5, WidthInTiles: 6, HeightInTiles: 7, Unknown: 8,
		}},
	}

	encoded, err := value.Encode()
	if err != nil {
		t.Fatal(err)
	}

	want := new(bytes.Buffer)
	writeDwordsExact(t, want,
		18, 0, 0, // version and stored dimensions
		2, 1, // stored act and substitution type
		1, // file count
	)
	want.WriteString("a.dt1\x00")
	writeDwordsExact(t, want,
		1, 1, // wall and floor counts
		0x44332211, 10, // wall properties and orientation
		0x88776655,         // floor
		0xccbbaa99,         // shadow
		0x12345678,         // substitution cell
		1,                  // object count
		1, 2, 15, 20, 0x44, // object
		0, // DS1Edit's version 18 group preamble
		1, // group count
		4, 5, 6, 7, 8,
		1,         // NPC records
		1, 15, 20, // path count and owning object coordinates
		25, 30, 7, // path node
	)
	if !bytes.Equal(encoded, want.Bytes()) {
		t.Fatalf("encoded bytes differ\n got: %x\nwant: %x", encoded, want.Bytes())
	}
}

func TestEncodePreservesNPCPathRecordOrder(t *testing.T) {
	value := populatedVersionModel(LatestVersion)
	value.Objects = append(value.Objects, Object{
		Type: 2, ID: 3, X: 30, Y: 35, Flags: 0x5678,
		Paths: []Path{{Position: *mathlib.NewVector2(40, 45), Action: 10}},
	})
	value.NPCPathOrder = []int{1, 0}

	encoded, err := value.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := FromBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.NPCPathOrder) != 2 || decoded.NPCPathOrder[0] != 1 || decoded.NPCPathOrder[1] != 0 {
		t.Fatalf("NPC path order = %v, want [1 0]", decoded.NPCPathOrder)
	}
	reencoded, err := decoded.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatal("NPC path order changed during round trip")
	}
}

func TestEncodeRoundTripsEverySupportedVersion(t *testing.T) {
	for version := Version(1); version <= LatestVersion; version++ {
		version := version
		t.Run(stringVersion(version), func(t *testing.T) {
			value := populatedVersionModel(version)
			encoded, err := value.Encode()
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := FromBytes(encoded)
			if err != nil {
				t.Fatal(err)
			}
			reencoded, err := decoded.Encode()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(reencoded, encoded) {
				t.Fatalf("decode/encode changed bytes\nfirst:  %x\nsecond: %x", encoded, reencoded)
			}
			if decoded.Act != value.Act {
				t.Fatalf("act = %d, want %d", decoded.Act, value.Act)
			}
			if version < 7 && decoded.Tiles[0][0].Walls[0].RawOrientation != 5 {
				t.Fatalf("raw legacy orientation = %d, want 5", decoded.Tiles[0][0].Walls[0].RawOrientation)
			}
		})
	}
}

func TestEncodeRejectsUnrepresentableModelsBeforeWriting(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*DS1)
		match  string
	}{
		{"unsupported version", func(value *DS1) { value.Version = LatestVersion + 1 }, "unsupported version"},
		{"wrong tile width", func(value *DS1) { value.Tiles[0] = nil }, "tile row 0"},
		{"NUL file", func(value *DS1) { value.Files[0] = "bad\x00file" }, "NUL byte"},
		{"legacy flags", func(value *DS1) {
			value.Version = 5
			value.Act = 1
			value.NumberOfFloors = 1
			value.SubstitutionType = 0
			value.NumberOfSubstitutionLayers = 0
			value.SubstitutionGroups = nil
			value.SubstitutionUnknown = 0
			value.Objects[0].Paths = nil
			for x := range value.Tiles[0] {
				value.Tiles[0][x].Floors = value.Tiles[0][x].Floors[:1]
				value.Tiles[0][x].Substitutions = nil
			}
		}, "flags not supported"},
		{"version 14 action", func(value *DS1) {
			value.Version = 14
			value.NumberOfFloors = 1
			value.SubstitutionUnknown = 0
			for x := range value.Tiles[0] {
				value.Tiles[0][x].Floors = value.Tiles[0][x].Floors[:1]
			}
		}, "action not supported"},
		{"fractional path", func(value *DS1) { value.Objects[0].Paths[0].Position.X = 1.5 }, "not an int32"},
		{"ambiguous path owner", func(value *DS1) {
			value.Objects = append([]Object{{X: value.Objects[0].X, Y: value.Objects[0].Y}}, value.Objects...)
		}, "ambiguous"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := populatedVersionModel(LatestVersion)
			test.mutate(value)
			var destination bytes.Buffer
			destination.WriteString("sentinel")
			err := value.EncodeTo(&destination)
			if err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("error = %v, want text %q", err, test.match)
			}
			if destination.String() != "sentinel" {
				t.Fatalf("invalid model partially written: %x", destination.Bytes())
			}
		})
	}
}

func TestEncodeErrors(t *testing.T) {
	if _, err := (*DS1)(nil).Encode(); err == nil {
		t.Fatal("nil model encoded without error")
	}
	value := populatedVersionModel(LatestVersion)
	if err := value.EncodeTo(nil); err == nil {
		t.Fatal("nil destination accepted")
	}
	want := errors.New("write failed")
	if err := value.EncodeTo(errorWriter{want}); !errors.Is(err, want) {
		t.Fatalf("write error = %v, want %v", err, want)
	}
}

func TestDecodeRejectsUnsupportedVersion(t *testing.T) {
	var data bytes.Buffer
	writeDwordsExact(t, &data, 19)
	if _, err := FromBytes(data.Bytes()); err == nil || !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("error = %v", err)
	}
}

func populatedVersionModel(version Version) *DS1 {
	walls := int32(1)
	floors := int32(1)
	substitutions := int32(0)
	if version >= 4 {
		walls = 2
	}
	if version >= 16 {
		floors = 2
	}
	if version < 4 || version >= 10 {
		substitutions = 1
	}

	value := &DS1{
		Version:                    version,
		Width:                      2,
		Height:                     1,
		Act:                        1,
		NumberOfWalls:              walls,
		NumberOfFloors:             floors,
		NumberOfShadowLayers:       1,
		NumberOfSubstitutionLayers: substitutions,
		Tiles:                      make([][]TileRecord, 1),
	}
	if version >= 3 {
		value.Files = []string{"global\\tiles\\test.dt1"}
	}
	if version >= 6 {
		value.Objects = []Object{{Type: 1, ID: 2, X: 10, Y: 15, Flags: 0x1234}}
	} else if version >= 2 {
		value.Objects = []Object{{Type: 1, ID: 2, X: 10, Y: 15}}
	}
	if version >= 8 {
		value.Act = 5
	}
	if version >= 9 && version <= 13 {
		value.HeaderUnknown = [2]uint32{0x12345678, 0x90abcdef}
	}
	if version >= 10 {
		value.SubstitutionType = 2
	}
	if version >= 12 {
		group := SubstitutionGroup{TileX: 1, TileY: 2, WidthInTiles: 3, HeightInTiles: 4}
		if version >= 13 {
			group.Unknown = 5
		}
		value.SubstitutionGroups = []SubstitutionGroup{group}
	}
	if version >= 14 {
		path := Path{Position: *mathlib.NewVector2(20, 25)}
		if version >= 15 {
			path.Action = 9
		}
		value.Objects[0].Paths = []Path{path}
	}
	if version >= 18 {
		value.SubstitutionUnknown = 0xfedcba98
	}

	value.Tiles[0] = make([]TileRecord, value.Width)
	for x := range value.Tiles[0] {
		tile := &value.Tiles[0][x]
		tile.Walls = make([]WallRecord, walls)
		tile.Floors = make([]FloorShadowRecord, floors)
		tile.Shadows = make([]FloorShadowRecord, 1)
		tile.Substitutions = make([]SubstitutionRecord, substitutions)
		for layer := range tile.Walls {
			tile.Walls[layer].SetPacked(uint32(0x10203040 + x*0x100 + layer))
			tile.Walls[layer].OrientationUnknown = uint32(0x00a5b600 + x<<16 + layer<<8)
			if version < 7 {
				tile.Walls[layer].Type = TileType(3)
				tile.Walls[layer].RawOrientation = 5
			} else {
				tile.Walls[layer].Type = TileType(10 + layer)
			}
		}
		for layer := range tile.Floors {
			tile.Floors[layer].SetPacked(uint32(0x50607080 + x*0x100 + layer))
		}
		tile.Shadows[0].SetPacked(uint32(0x90a0b0c0 + x*0x100))
		if substitutions != 0 {
			tile.Substitutions[0].Unknown = uint32(0x0f0e0d0c + x)
		}
	}
	return value
}

func writeDwordsExact(t *testing.T, destination io.Writer, values ...uint32) {
	t.Helper()
	for _, value := range values {
		if err := binary.Write(destination, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
}

func stringVersion(version Version) string {
	return "v" + string(rune('0'+version/10)) + string(rune('0'+version%10))
}

type errorWriter struct{ err error }

func (writer errorWriter) Write([]byte) (int, error) { return 0, writer.err }
