package pkg

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestVersion12SubstitutionGroupHasFourDwords(t *testing.T) {
	data := new(bytes.Buffer)
	writeDwords(t, data,
		12, 0, 0, // version and stored dimensions
		1, 1, // act and substitution type
		0,    // file count
		0, 0, // v9-v13 unknown dwords
		1,             // wall count; floor count is implicit before v16
		0, 0, 0, 0, 0, // wall, orientation, floor, shadow, substitution cells
		0,          // object count
		1,          // substitution group count
		2, 3, 4, 5, // version 12 group: no trailing unknown dword
	)
	decoded, err := FromBytes(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.SubstitutionGroups) != 1 {
		t.Fatalf("group count = %d", len(decoded.SubstitutionGroups))
	}
	group := decoded.SubstitutionGroups[0]
	if group.TileX != 2 || group.TileY != 3 || group.WidthInTiles != 4 || group.HeightInTiles != 5 || group.Unknown != 0 {
		t.Fatalf("group = %+v", group)
	}
}

func TestUnmatchedNPCPathSkipUsesDwordsAndPathCount(t *testing.T) {
	data := new(bytes.Buffer)
	writeDwords(t, data,
		15, 0, 0, // version and stored dimensions
		1, 0, // act and no substitution layer
		0,          // file count
		1,          // wall count; floor count is implicit before v16
		0, 0, 0, 0, // wall, orientation, floor, shadow cells
		1,                // object count
		1, 99, 10, 10, 0, // object
		2,         // NPC path records
		2, 20, 20, // unmatched object, two paths
		21, 21, 7,
		22, 22, 8,
		1, 10, 10, // matched object, one path
		11, 12, 9,
	)
	decoded, err := FromBytes(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	paths := decoded.Objects[0].Paths
	if len(paths) != 1 || paths[0].Position.X != 11 || paths[0].Position.Y != 12 || paths[0].Action != 9 {
		t.Fatalf("matched paths = %+v", paths)
	}
}

func TestVersion14NPCPathsUseDS1EditDefaultAction(t *testing.T) {
	data := new(bytes.Buffer)
	writeDwords(t, data,
		14, 0, 0, // version and stored dimensions
		1, 0, // act and no substitution layer
		0,          // file count
		1,          // wall count; floor count is implicit before v16
		0, 0, 0, 0, // wall, orientation, floor, shadow cells
		1,                // object count
		1, 99, 10, 10, 0, // object
		1,         // NPC path records
		1, 10, 10, // path count and object position
		11, 12, // v14 path has no serialized action
	)
	decoded, err := FromBytes(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	paths := decoded.Objects[0].Paths
	if len(paths) != 1 || paths[0].Action != 1 {
		t.Fatalf("v14 paths = %+v, want one path with default action 1", paths)
	}
	encoded, err := decoded.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, data.Bytes()) {
		t.Fatal("v14 default action changed serialized bytes")
	}
}

func TestTruncatedObjectReturnsError(t *testing.T) {
	data := new(bytes.Buffer)
	writeDwords(t, data,
		15, 0, 0, 1, 0, 0, 1,
		0, 0, 0, 0,
		1, // object count, followed by only its type
		1,
	)
	if _, err := FromBytes(data.Bytes()); err == nil {
		t.Fatal("truncated object decoded without error")
	}
}

func writeDwords(t *testing.T, destination *bytes.Buffer, values ...int32) {
	t.Helper()
	for _, value := range values {
		if err := binary.Write(destination, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
}
