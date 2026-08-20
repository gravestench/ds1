package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

type encodeLayout struct {
	layers            []LayerStreamType
	pathObjectIndexes []int
}

// Encode returns a DS1 representation of the model. The selected Version is
// preserved; use LatestVersion for the canonical DS1Edit-compatible layout.
func (ds1 *DS1) Encode() ([]byte, error) {
	var output bytes.Buffer
	if err := ds1.EncodeTo(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// EncodeTo writes a DS1 representation of the model to destination.
func (ds1 *DS1) EncodeTo(destination io.Writer) error {
	if destination == nil {
		return fmt.Errorf("ds1: nil destination")
	}
	layout, err := ds1.validateForEncoding()
	if err != nil {
		return err
	}

	encoder := ds1Encoder{destination: destination}
	if err := ds1.writeTo(&encoder, layout); err != nil {
		return fmt.Errorf("ds1: write: %w", err)
	}
	return nil
}

func (ds1 *DS1) validateForEncoding() (*encodeLayout, error) {
	if ds1 == nil {
		return nil, fmt.Errorf("ds1: nil model")
	}
	if !ds1.Version.Supported() {
		return nil, fmt.Errorf("ds1: unsupported version %d", ds1.Version)
	}
	if ds1.Width <= 0 || ds1.Height <= 0 || ds1.Width > maxMapDimension || ds1.Height > maxMapDimension ||
		int64(ds1.Width)*int64(ds1.Height) > maxRecordCount {
		return nil, fmt.Errorf("ds1: invalid map dimensions %dx%d", ds1.Width, ds1.Height)
	}
	if ds1.Act < 1 || ds1.Act > maxActNumber {
		return nil, fmt.Errorf("ds1: invalid act %d", ds1.Act)
	}
	if !ds1.Version.EncodesAct() && ds1.Act != 1 {
		return nil, fmt.Errorf("ds1: version %d cannot encode act %d", ds1.Version, ds1.Act)
	}

	if ds1.SubstitutionType < 0 || ds1.SubstitutionType > 2 {
		return nil, fmt.Errorf("ds1: invalid substitution type %d", ds1.SubstitutionType)
	}
	if !ds1.Version.EncodesSubstitutionLayers() && ds1.SubstitutionType != 0 {
		return nil, fmt.Errorf("ds1: version %d cannot encode substitution type", ds1.Version)
	}

	walls, floors, substitutions := ds1.NumberOfWalls, ds1.NumberOfFloors, ds1.NumberOfSubstitutionLayers
	if ds1.Version.EncodesSimpleLayers() {
		if walls != 1 || floors != 1 || ds1.NumberOfShadowLayers != 1 || substitutions != 1 {
			return nil, fmt.Errorf("ds1: version %d requires 1 wall, 1 floor, 1 shadow, and 1 substitution layer", ds1.Version)
		}
	} else {
		if walls < 0 || walls > 4 {
			return nil, fmt.Errorf("ds1: invalid wall layer count %d", walls)
		}
		if ds1.Version.EncodesWallLayers() {
			if floors < 0 || floors > 2 {
				return nil, fmt.Errorf("ds1: invalid floor layer count %d", floors)
			}
		} else if floors != 1 {
			return nil, fmt.Errorf("ds1: version %d requires 1 floor layer", ds1.Version)
		}
		if ds1.NumberOfShadowLayers != 1 {
			return nil, fmt.Errorf("ds1: version %d requires 1 shadow layer", ds1.Version)
		}
		expectedSubstitutions := int32(0)
		if ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2 {
			expectedSubstitutions = 1
		}
		if substitutions != expectedSubstitutions {
			return nil, fmt.Errorf("ds1: substitution layer count %d does not match type %d", substitutions, ds1.SubstitutionType)
		}
	}

	if !ds1.Version.EncodesFiles() && len(ds1.Files) != 0 {
		return nil, fmt.Errorf("ds1: version %d cannot encode file references", ds1.Version)
	}
	if len(ds1.Files) > maxRecordCount {
		return nil, fmt.Errorf("ds1: invalid file count %d", len(ds1.Files))
	}
	for index, name := range ds1.Files {
		if strings.IndexByte(name, 0) >= 0 {
			return nil, fmt.Errorf("ds1: file reference %d contains a NUL byte", index)
		}
	}

	if !ds1.Version.HasUnknownBytes1() && ds1.HeaderUnknown != [2]uint32{} {
		return nil, fmt.Errorf("ds1: version %d cannot encode preserved header dwords", ds1.Version)
	}
	groupsEnabled := ds1.Version.EncodesSubstitutionGroups() &&
		(ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2)
	if !groupsEnabled && len(ds1.SubstitutionGroups) != 0 {
		return nil, fmt.Errorf("ds1: version %d and substitution type %d cannot encode groups", ds1.Version, ds1.SubstitutionType)
	}
	if len(ds1.SubstitutionGroups) > maxRecordCount {
		return nil, fmt.Errorf("ds1: invalid substitution group count %d", len(ds1.SubstitutionGroups))
	}
	if (!groupsEnabled || !ds1.Version.HasUnknownBytes2()) && ds1.SubstitutionUnknown != 0 {
		return nil, fmt.Errorf("ds1: version %d cannot encode the substitution preamble", ds1.Version)
	}
	if !ds1.Version.EncodesSubstitutionGroupExtra() {
		for index, group := range ds1.SubstitutionGroups {
			if group.Unknown != 0 {
				return nil, fmt.Errorf("ds1: substitution group %d has extra data not supported by version %d", index, ds1.Version)
			}
		}
	}

	if len(ds1.Tiles) != int(ds1.Height) {
		return nil, fmt.Errorf("ds1: tile row count %d, want %d", len(ds1.Tiles), ds1.Height)
	}
	for y, row := range ds1.Tiles {
		if len(row) != int(ds1.Width) {
			return nil, fmt.Errorf("ds1: tile row %d has width %d, want %d", y, len(row), ds1.Width)
		}
		for x := range row {
			tile := &row[x]
			if len(tile.Walls) != int(walls) || len(tile.Floors) != int(floors) ||
				len(tile.Shadows) != int(ds1.NumberOfShadowLayers) || len(tile.Substitutions) != int(substitutions) {
				return nil, fmt.Errorf(
					"ds1: tile (%d,%d) layer shape is %d/%d/%d/%d, want %d/%d/%d/%d",
					x, y, len(tile.Walls), len(tile.Floors), len(tile.Shadows), len(tile.Substitutions),
					walls, floors, ds1.NumberOfShadowLayers, substitutions,
				)
			}
			for layer := range tile.Walls {
				if err := validateWallRecord(tile.Walls[layer], ds1.Version); err != nil {
					return nil, fmt.Errorf("ds1: tile (%d,%d) wall %d: %w", x, y, layer, err)
				}
			}
			for layer := range tile.Floors {
				if err := validateCellRecord(tile.Floors[layer]); err != nil {
					return nil, fmt.Errorf("ds1: tile (%d,%d) floor %d: %w", x, y, layer, err)
				}
			}
			for layer := range tile.Shadows {
				if err := validateCellRecord(tile.Shadows[layer]); err != nil {
					return nil, fmt.Errorf("ds1: tile (%d,%d) shadow %d: %w", x, y, layer, err)
				}
			}
		}
	}

	if !ds1.Version.EncodesObjectFlags() {
		for index, object := range ds1.Objects {
			if object.Flags != 0 {
				return nil, fmt.Errorf("ds1: object %d has flags not supported by version %d", index, ds1.Version)
			}
		}
	}
	if ds1.Version < 2 && len(ds1.Objects) != 0 {
		return nil, fmt.Errorf("ds1: version %d cannot encode objects", ds1.Version)
	}
	if len(ds1.Objects) > maxRecordCount {
		return nil, fmt.Errorf("ds1: invalid object count %d", len(ds1.Objects))
	}

	layout := &encodeLayout{layers: ds1.setupStreamLayerTypes()}
	firstObjectAt := make(map[[2]int32]int, len(ds1.Objects))
	for index := range ds1.Objects {
		object := &ds1.Objects[index]
		coordinate := [2]int32{object.X, object.Y}
		firstIndex, exists := firstObjectAt[coordinate]
		if !exists {
			firstObjectAt[coordinate] = index
			firstIndex = index
		}
		if len(object.Paths) == 0 {
			continue
		}
		if !ds1.Version.EncodesNPCs() {
			return nil, fmt.Errorf("ds1: object %d has paths not supported by version %d", index, ds1.Version)
		}
		if firstIndex != index {
			return nil, fmt.Errorf("ds1: object %d paths are ambiguous with object %d at (%d,%d)", index, firstIndex, object.X, object.Y)
		}
		if len(object.Paths) > maxRecordCount {
			return nil, fmt.Errorf("ds1: object %d has invalid path count %d", index, len(object.Paths))
		}
		for pathIndex, path := range object.Paths {
			if _, err := pathCoordinate(path.Position.X); err != nil {
				return nil, fmt.Errorf("ds1: object %d path %d X: %w", index, pathIndex, err)
			}
			if _, err := pathCoordinate(path.Position.Y); err != nil {
				return nil, fmt.Errorf("ds1: object %d path %d Y: %w", index, pathIndex, err)
			}
			// Version 14 does not serialize action. DS1Edit reads its paths as
			// action 1, while a newly constructed model may still carry zero.
			// Both represent the same v14 bytes; every other action would be
			// silently lost and remains invalid.
			if !ds1.Version.EncodesNPCExtraData() && path.Action != 0 && path.Action != 1 {
				return nil, fmt.Errorf("ds1: object %d path %d has an action not supported by version %d", index, pathIndex, ds1.Version)
			}
			if int64(path.Action) < math.MinInt32 || int64(path.Action) > math.MaxInt32 {
				return nil, fmt.Errorf("ds1: object %d path %d action %d exceeds int32", index, pathIndex, path.Action)
			}
		}
		layout.pathObjectIndexes = append(layout.pathObjectIndexes, index)
	}
	if len(layout.pathObjectIndexes) > maxRecordCount {
		return nil, fmt.Errorf("ds1: invalid NPC path record count %d", len(layout.pathObjectIndexes))
	}
	if len(ds1.NPCPathOrder) != 0 {
		if !ds1.Version.EncodesNPCs() {
			return nil, fmt.Errorf("ds1: version %d cannot encode NPC path order", ds1.Version)
		}
		if len(ds1.NPCPathOrder) != len(layout.pathObjectIndexes) {
			return nil, fmt.Errorf("ds1: NPC path order contains %d objects, want %d", len(ds1.NPCPathOrder), len(layout.pathObjectIndexes))
		}
		seen := make(map[int]bool, len(ds1.NPCPathOrder))
		for position, objectIndex := range ds1.NPCPathOrder {
			if objectIndex < 0 || objectIndex >= len(ds1.Objects) || len(ds1.Objects[objectIndex].Paths) == 0 {
				return nil, fmt.Errorf("ds1: NPC path order entry %d refers to invalid object %d", position, objectIndex)
			}
			if seen[objectIndex] {
				return nil, fmt.Errorf("ds1: NPC path order repeats object %d", objectIndex)
			}
			seen[objectIndex] = true
		}
		layout.pathObjectIndexes = append(layout.pathObjectIndexes[:0], ds1.NPCPathOrder...)
	}
	return layout, nil
}

func validateWallRecord(record WallRecord, version Version) error {
	if err := validateCellValues(record.Sequence, record.Unknown1, record.Style, record.Unknown2); err != nil {
		return err
	}
	if record.OrientationUnknown&0xff != 0 {
		return fmt.Errorf("orientation unknown bits overlap the orientation byte")
	}
	_, err := serializedOrientation(record, version)
	return err
}

func validateCellRecord(record FloorShadowRecord) error {
	return validateCellValues(record.Sequence, record.Unknown1, record.Style, record.Unknown2)
}

func validateCellValues(sequence, unknown1, style, unknown2 byte) error {
	if sequence > 0x3f || unknown1 > 0x3f || style > 0x3f || unknown2 > 0x1f {
		return fmt.Errorf("property fields exceed their serialized bit widths")
	}
	return nil
}

func serializedOrientation(record WallRecord, version Version) (uint32, error) {
	raw := byte(record.Type)
	if !version.EncodesDirectOrientations() {
		raw = record.RawOrientation
		if decodedLegacyOrientation(raw) != byte(record.Type) {
			var found bool
			raw, found = encodeLegacyOrientation(byte(record.Type))
			if !found {
				return 0, fmt.Errorf("orientation %d cannot be encoded by version %d", record.Type, version)
			}
		}
	}
	unknown := record.OrientationUnknown
	if unknown == 0 && record.Zero != 0 {
		unknown = uint32(record.Zero) << 8
	}
	return unknown | uint32(raw), nil
}

func decodedLegacyOrientation(raw byte) byte {
	if int(raw) < len(legacyOrientationLookup) {
		return legacyOrientationLookup[raw]
	}
	return raw
}

func encodeLegacyOrientation(orientation byte) (byte, bool) {
	for raw, decoded := range legacyOrientationLookup {
		if decoded == orientation {
			return byte(raw), true
		}
	}
	if int(orientation) >= len(legacyOrientationLookup) {
		return orientation, true
	}
	return 0, false
}

func pathCoordinate(value float64) (int32, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) || value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("coordinate %v is not an int32", value)
	}
	return int32(value), nil
}

func (ds1 *DS1) writeTo(encoder *ds1Encoder, layout *encodeLayout) error {
	if err := encoder.int32(int32(ds1.Version)); err != nil {
		return err
	}
	if err := encoder.int32(ds1.Width - 1); err != nil {
		return err
	}
	if err := encoder.int32(ds1.Height - 1); err != nil {
		return err
	}
	if ds1.Version.EncodesAct() {
		if err := encoder.int32(ds1.Act - 1); err != nil {
			return err
		}
	}
	if ds1.Version.EncodesSubstitutionLayers() {
		if err := encoder.int32(ds1.SubstitutionType); err != nil {
			return err
		}
	}
	if ds1.Version.EncodesFiles() {
		if err := encoder.int32(int32(len(ds1.Files))); err != nil {
			return err
		}
		for _, name := range ds1.Files {
			if err := encoder.bytes([]byte(name)); err != nil {
				return err
			}
			if err := encoder.bytes([]byte{0}); err != nil {
				return err
			}
		}
	}
	if ds1.Version.HasUnknownBytes1() {
		for _, value := range ds1.HeaderUnknown {
			if err := encoder.uint32(value); err != nil {
				return err
			}
		}
	}
	if ds1.Version.EncodesFloorLayers() {
		if err := encoder.int32(ds1.NumberOfWalls); err != nil {
			return err
		}
		if ds1.Version.EncodesWallLayers() {
			if err := encoder.int32(ds1.NumberOfFloors); err != nil {
				return err
			}
		}
	}
	if err := ds1.writeLayerStreams(encoder, layout.layers); err != nil {
		return err
	}
	if ds1.Version >= 2 {
		if err := encoder.int32(int32(len(ds1.Objects))); err != nil {
			return err
		}
		for _, object := range ds1.Objects {
			for _, value := range [...]int32{object.Type, object.ID, object.X, object.Y} {
				if err := encoder.int32(value); err != nil {
					return err
				}
			}
			if ds1.Version.EncodesObjectFlags() {
				if err := encoder.int32(object.Flags); err != nil {
					return err
				}
			}
		}
	}
	groupsEnabled := ds1.Version.EncodesSubstitutionGroups() &&
		(ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2)
	if groupsEnabled {
		if ds1.Version.HasUnknownBytes2() {
			if err := encoder.uint32(ds1.SubstitutionUnknown); err != nil {
				return err
			}
		}
		if err := encoder.int32(int32(len(ds1.SubstitutionGroups))); err != nil {
			return err
		}
		for _, group := range ds1.SubstitutionGroups {
			for _, value := range [...]int32{group.TileX, group.TileY, group.WidthInTiles, group.HeightInTiles} {
				if err := encoder.int32(value); err != nil {
					return err
				}
			}
			if ds1.Version.EncodesSubstitutionGroupExtra() {
				if err := encoder.int32(group.Unknown); err != nil {
					return err
				}
			}
		}
	}
	if ds1.Version.EncodesNPCs() {
		if err := encoder.int32(int32(len(layout.pathObjectIndexes))); err != nil {
			return err
		}
		for _, objectIndex := range layout.pathObjectIndexes {
			object := &ds1.Objects[objectIndex]
			for _, value := range [...]int32{int32(len(object.Paths)), object.X, object.Y} {
				if err := encoder.int32(value); err != nil {
					return err
				}
			}
			for _, path := range object.Paths {
				x, _ := pathCoordinate(path.Position.X)
				y, _ := pathCoordinate(path.Position.Y)
				if err := encoder.int32(x); err != nil {
					return err
				}
				if err := encoder.int32(y); err != nil {
					return err
				}
				if ds1.Version.EncodesNPCExtraData() {
					if err := encoder.int32(int32(path.Action)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (ds1 *DS1) writeLayerStreams(encoder *ds1Encoder, layers []LayerStreamType) error {
	for _, layer := range layers {
		for y := range ds1.Tiles {
			for x := range ds1.Tiles[y] {
				tile := &ds1.Tiles[y][x]
				var value uint32
				switch layer {
				case LayerStreamWall1, LayerStreamWall2, LayerStreamWall3, LayerStreamWall4:
					value = tile.Walls[int(layer)-int(LayerStreamWall1)].Packed()
				case LayerStreamOrientation1, LayerStreamOrientation2, LayerStreamOrientation3, LayerStreamOrientation4:
					value, _ = serializedOrientation(tile.Walls[int(layer)-int(LayerStreamOrientation1)], ds1.Version)
				case LayerStreamFloor1, LayerStreamFloor2:
					value = tile.Floors[int(layer)-int(LayerStreamFloor1)].Packed()
				case LayerStreamShadow:
					value = tile.Shadows[0].Packed()
				case LayerStreamSubstitute:
					value = tile.Substitutions[0].Unknown
				default:
					return fmt.Errorf("unknown layer stream %d", layer)
				}
				if err := encoder.uint32(value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type ds1Encoder struct {
	destination io.Writer
	word        [4]byte
}

func (encoder *ds1Encoder) int32(value int32) error {
	return encoder.uint32(uint32(value))
}

func (encoder *ds1Encoder) uint32(value uint32) error {
	binary.LittleEndian.PutUint32(encoder.word[:], value)
	return encoder.bytes(encoder.word[:])
}

func (encoder *ds1Encoder) bytes(value []byte) error {
	for len(value) != 0 {
		written, err := encoder.destination.Write(value)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(value) {
			return io.ErrShortWrite
		}
		value = value[written:]
	}
	return nil
}
