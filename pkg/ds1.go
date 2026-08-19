package pkg

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gravestench/bitstream"
	"github.com/gravestench/mathlib"
)

const (
	maxActNumber    = 5
	maxMapDimension = 4096
	maxRecordCount  = 1 << 20
)

// DS1 represents the "stamp" data that is used to build up maps.
type DS1 struct {
	Files                      []string            // FilePtr table of file string pointers
	Objects                    []Object            // Objects
	NPCPathOrder               []int               // Object indexes in serialized NPC-path record order
	Tiles                      [][]TileRecord      // The tile data for the DS1
	SubstitutionGroups         []SubstitutionGroup // Substitution groups for the DS1
	HeaderUnknown              [2]uint32           // Preserved header dwords used by versions 9 through 13
	SubstitutionUnknown        uint32              // Preserved dword before substitution groups in version 18
	Version                                        // The version of the DS1
	Width                      int32               // Width of map, in # of tiles
	Height                     int32               // Height of map, in # of tiles
	Act                        int32               // Act, from 1 to 5. This tells which act table to use for the Objects list
	SubstitutionType           int32               // SubstitutionType (layer type): 0 if no layer, else type 1 or type 2
	NumberOfWalls              int32               // WallNum number of wall & orientation layers used
	NumberOfFloors             int32               // number of floor layers used
	NumberOfShadowLayers       int32               // ShadowNum number of shadow layer used
	NumberOfSubstitutionLayers int32               // SubstitutionNum number of substitution layer used
	SubstitutionGroupsNum      int32               // SubstitutionGroupsNum number of substitution groups, datas between objects & NPC paths
}

// FromBytes loads the specified DS1 file
func FromBytes(fileData []byte) (ds1 *DS1, err error) {
	return FromReader(bytes.NewReader(fileData))
}

// FromReader incrementally decodes a forward-only DS1 stream.
func FromReader(source io.Reader) (ds1 *DS1, err error) {
	if source == nil {
		return nil, fmt.Errorf("ds1: nil reader")
	}
	stream := bitstream.NewStreamReader(source)

	ds1 = &DS1{
		Act:                        1,
		NumberOfFloors:             0,
		NumberOfWalls:              0,
		NumberOfShadowLayers:       1,
		NumberOfSubstitutionLayers: 0,
	}

	if v, err := stream.Next(4).Bytes().AsInt32(); err != nil {
		return nil, fmt.Errorf("ds1: version: %w", err)
	} else {
		ds1.Version = Version(v)
	}
	if !ds1.Version.Supported() {
		return nil, fmt.Errorf("ds1: unsupported version %d", ds1.Version)
	}

	if ds1.Width, err = stream.Next(4).Bytes().AsInt32(); err != nil {
		return nil, err
	}

	if ds1.Height, err = stream.Next(4).Bytes().AsInt32(); err != nil {
		return nil, err
	}

	// minimum of 1
	ds1.Width++
	ds1.Height++
	if ds1.Width <= 0 || ds1.Height <= 0 || ds1.Width > maxMapDimension || ds1.Height > maxMapDimension ||
		int64(ds1.Width)*int64(ds1.Height) > maxRecordCount {
		return nil, fmt.Errorf("invalid map dimensions %dx%d", ds1.Width, ds1.Height)
	}

	if ds1.Version.EncodesAct() {
		storedAct, readErr := stream.Next(4).Bytes().AsInt32()
		if readErr != nil {
			return nil, readErr
		}
		ds1.Act = storedAct + 1
		if ds1.Act < 1 || ds1.Act > maxActNumber {
			return nil, fmt.Errorf("invalid act %d", ds1.Act)
		}
	}

	if ds1.Version.EncodesSubstitutionLayers() {
		if ds1.SubstitutionType, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return nil, err
		}

		if ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2 {
			ds1.NumberOfSubstitutionLayers = 1
		}
	}

	if ds1.Version.EncodesFiles() { //nolint:gomnd // Version number
		// These files reference things that don't exist anymore :-?
		numberOfFiles, err := stream.Next(4).Bytes().AsInt32()
		if err != nil {
			return nil, err
		}
		if numberOfFiles < 0 || numberOfFiles > maxRecordCount {
			return nil, fmt.Errorf("invalid file count %d", numberOfFiles)
		}

		ds1.Files = make([]string, numberOfFiles)

		for i := 0; i < int(numberOfFiles); i++ {
			ds1.Files[i] = ""

			for {
				ch, err := stream.Next(1).Bytes().AsByte()
				if err != nil {
					return nil, err
				}

				if ch == 0 {
					break
				}

				ds1.Files[i] += string(ch)
			}
		}
	}

	if ds1.Version.HasUnknownBytes1() {
		for index := range ds1.HeaderUnknown {
			if ds1.HeaderUnknown[index], err = stream.Next(4).Bytes().AsUInt32(); err != nil {
				return nil, fmt.Errorf("header unknown dword %d: %w", index, err)
			}
		}
	}

	if ds1.Version.EncodesFloorLayers() {
		if ds1.NumberOfWalls, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return nil, err
		}

		if ds1.Version.EncodesWallLayers() {
			if ds1.NumberOfFloors, err = stream.Next(4).Bytes().AsInt32(); err != nil {
				return nil, err
			}
		} else {
			ds1.NumberOfFloors = 1
		}
	}
	if ds1.Version.EncodesSimpleLayers() {
		ds1.NumberOfWalls = 1
		ds1.NumberOfFloors = 1
		ds1.NumberOfSubstitutionLayers = 1
	}
	if ds1.NumberOfWalls < 0 || ds1.NumberOfWalls > 4 || ds1.NumberOfFloors < 0 || ds1.NumberOfFloors > 2 {
		return nil, fmt.Errorf("invalid layer counts: %d walls, %d floors", ds1.NumberOfWalls, ds1.NumberOfFloors)
	}

	layerStream := ds1.setupStreamLayerTypes()

	ds1.Tiles = make([][]TileRecord, ds1.Height)

	for y := range ds1.Tiles {
		ds1.Tiles[y] = make([]TileRecord, ds1.Width)
		for x := 0; x < int(ds1.Width); x++ {
			ds1.Tiles[y][x].Walls = make([]WallRecord, ds1.NumberOfWalls)
			ds1.Tiles[y][x].Floors = make([]FloorShadowRecord, ds1.NumberOfFloors)
			ds1.Tiles[y][x].Shadows = make([]FloorShadowRecord, ds1.NumberOfShadowLayers)
			ds1.Tiles[y][x].Substitutions = make([]SubstitutionRecord, ds1.NumberOfSubstitutionLayers)
		}
	}

	if err = ds1.loadLayerStreams(stream, layerStream); err != nil {
		return nil, err
	}

	if err = ds1.loadObjects(stream); err != nil {
		return nil, err
	}

	if err = ds1.loadSubstitutions(stream); err != nil {
		return nil, err
	}

	if err = ds1.loadNPCs(stream); err != nil {
		return nil, err
	}

	return ds1, nil
}

func (ds1 *DS1) loadObjects(br *bitstream.StreamReader) error {
	if ds1.Version >= 2 { //nolint:gomnd // Version number
		numberOfObjects, err := br.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}
		if numberOfObjects < 0 || numberOfObjects > maxRecordCount {
			return fmt.Errorf("invalid object count %d", numberOfObjects)
		}

		ds1.Objects = make([]Object, numberOfObjects)

		for objIdx := 0; objIdx < int(numberOfObjects); objIdx++ {
			newObject := Object{}
			if newObject.Type, err = br.Next(4).Bytes().AsInt32(); err != nil {
				return fmt.Errorf("object %d type: %w", objIdx, err)
			}
			if newObject.ID, err = br.Next(4).Bytes().AsInt32(); err != nil {
				return fmt.Errorf("object %d ID: %w", objIdx, err)
			}
			if newObject.X, err = br.Next(4).Bytes().AsInt32(); err != nil {
				return fmt.Errorf("object %d X: %w", objIdx, err)
			}
			if newObject.Y, err = br.Next(4).Bytes().AsInt32(); err != nil {
				return fmt.Errorf("object %d Y: %w", objIdx, err)
			}
			if ds1.Version.EncodesObjectFlags() {
				if newObject.Flags, err = br.Next(4).Bytes().AsInt32(); err != nil {
					return fmt.Errorf("object %d flags: %w", objIdx, err)
				}
			}

			ds1.Objects[objIdx] = newObject
		}
	} else {
		ds1.Objects = make([]Object, 0)
	}

	return nil
}

func (ds1 *DS1) loadSubstitutions(stream *bitstream.StreamReader) (err error) {
	ds1.SubstitutionGroups = make([]SubstitutionGroup, 0)

	hasSubType := ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2
	hasEncodedSubGroups := ds1.Version.EncodesSubstitutionGroups() && hasSubType

	if !hasEncodedSubGroups {
		return nil
	}

	if ds1.Version.HasUnknownBytes2() {
		if ds1.SubstitutionUnknown, err = stream.Next(4).Bytes().AsUInt32(); err != nil {
			return fmt.Errorf("substitution preamble: %w", err)
		}
	}

	numberOfSubGroups, err := stream.Next(4).Bytes().AsInt32()
	if err != nil {
		return err
	}
	if numberOfSubGroups < 0 || numberOfSubGroups > maxRecordCount {
		return fmt.Errorf("invalid substitution group count %d", numberOfSubGroups)
	}

	ds1.SubstitutionGroups = make([]SubstitutionGroup, numberOfSubGroups)
	ds1.SubstitutionGroupsNum = numberOfSubGroups

	for subIdx := 0; subIdx < int(numberOfSubGroups); subIdx++ {
		newSub := SubstitutionGroup{}
		if newSub.TileX, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return err
		}
		if newSub.TileY, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return err
		}
		if newSub.WidthInTiles, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return err
		}
		if newSub.HeightInTiles, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return err
		}
		if ds1.Version.EncodesSubstitutionGroupExtra() {
			if newSub.Unknown, err = stream.Next(4).Bytes().AsInt32(); err != nil {
				return err
			}
		}

		ds1.SubstitutionGroups[subIdx] = newSub
	}

	return nil
}

func (ds1 *DS1) setupStreamLayerTypes() []LayerStreamType {
	if ds1.Version.EncodesSimpleLayers() { //nolint:gomnd // Version number
		return []LayerStreamType{
			LayerStreamWall1,
			LayerStreamFloor1,
			LayerStreamOrientation1,
			LayerStreamSubstitute,
			LayerStreamShadow,
		}
	}

	// iirc, there is a layer that specifies orientations for the tiles, it is always the same as the number of walls.
	var numDirections = ds1.NumberOfWalls

	numLayers := ds1.NumberOfWalls +
		numDirections +
		ds1.NumberOfFloors +
		ds1.NumberOfShadowLayers +
		ds1.NumberOfSubstitutionLayers

	layerStream := make([]LayerStreamType, numLayers)

	layerIdx := 0
	for i := 0; i < int(ds1.NumberOfWalls); i++ {
		layerStream[layerIdx] = LayerStreamType(int(LayerStreamWall1) + i)

		// again, this is for the orientation
		layerStream[layerIdx+1] = LayerStreamType(int(LayerStreamOrientation1) + i)

		layerIdx++
		layerIdx++
	}

	for i := 0; i < int(ds1.NumberOfFloors); i++ {
		layerStream[layerIdx] = LayerStreamType(int(LayerStreamFloor1) + i)
		layerIdx++
	}

	if ds1.NumberOfShadowLayers > 0 {
		layerStream[layerIdx] = LayerStreamShadow
		layerIdx++
	}

	if ds1.NumberOfSubstitutionLayers > 0 {
		layerStream[layerIdx] = LayerStreamSubstitute
	}

	return layerStream
}

func (ds1 *DS1) loadNPCs(stream *bitstream.StreamReader) (err error) {
	if !ds1.Version.EncodesNPCs() {
		return nil
	}

	numberOfNpcs, err := stream.Next(4).Bytes().AsInt32()
	if err != nil {
		return err
	}
	if numberOfNpcs < 0 || numberOfNpcs > maxRecordCount {
		return fmt.Errorf("invalid NPC count %d", numberOfNpcs)
	}

	for npcIdx := 0; npcIdx < int(numberOfNpcs); npcIdx++ {
		numPaths, err := stream.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}
		if numPaths < 0 || numPaths > maxRecordCount {
			return fmt.Errorf("invalid NPC path count %d", numPaths)
		}

		npcX, err := stream.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}

		npcY, err := stream.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}

		objIdx := -1

		for idx, ds1Obj := range ds1.Objects {
			if ds1Obj.X == npcX && ds1Obj.Y == npcY {
				objIdx = idx
				break
			}
		}

		if objIdx > -1 {
			ds1.NPCPathOrder = append(ds1.NPCPathOrder, objIdx)
			if err = ds1.loadNpcPaths(stream, objIdx, int(numPaths)); err != nil {
				return err
			}

			continue
		}

		const dwordBytes = 4
		dwordsPerPath := 2
		if ds1.Version.EncodesNPCExtraData() {
			dwordsPerPath++
		}
		if _, err := stream.Next(int(numPaths) * dwordsPerPath * dwordBytes).Bytes().AsBytes(); err != nil {
			return fmt.Errorf("skip %d paths for unmatched NPC at (%d,%d): %w", numPaths, npcX, npcY, err)
		}
	}

	return nil
}

func (ds1 *DS1) loadNpcPaths(br *bitstream.StreamReader, objIdx, numPaths int) (err error) {
	if ds1.Objects[objIdx].Paths != nil {
		return fmt.Errorf("duplicate NPC path record for object %d", objIdx)
	}
	ds1.Objects[objIdx].Paths = make([]Path, numPaths)

	for pathIdx := 0; pathIdx < numPaths; pathIdx++ {
		newPath := Path{}
		x, err := br.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}

		y, err := br.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}

		newPath.Position = *mathlib.NewVector2(float64(x), float64(y))

		if ds1.Version.EncodesNPCExtraData() {
			action, err := br.Next(4).Bytes().AsInt32()
			if err != nil {
				return err
			}

			newPath.Action = int(action)
		}

		ds1.Objects[objIdx].Paths[pathIdx] = newPath
	}

	return nil
}

func (ds1 *DS1) loadLayerStreams(stream *bitstream.StreamReader, layerStream []LayerStreamType) (err error) {
	for lIdx := range layerStream {
		layerStreamType := layerStream[lIdx]

		for y := 0; y < int(ds1.Height); y++ {
			for x := 0; x < int(ds1.Width); x++ {
				bits, err := stream.Next(4).Bytes().AsUInt32()
				if err != nil {
					return err
				}

				switch layerStreamType {
				case LayerStreamWall1, LayerStreamWall2, LayerStreamWall3, LayerStreamWall4:
					wallIndex := int(layerStreamType) - int(LayerStreamWall1)
					ds1.Tiles[y][x].Walls[wallIndex].SetPacked(bits)
				case LayerStreamOrientation1, LayerStreamOrientation2,
					LayerStreamOrientation3, LayerStreamOrientation4:
					wallIndex := int(layerStreamType) - int(LayerStreamOrientation1)
					wall := &ds1.Tiles[y][x].Walls[wallIndex]
					rawOrientation := byte(bits)
					orientation := rawOrientation

					if !ds1.Version.EncodesDirectOrientations() {
						if int(rawOrientation) < len(legacyOrientationLookup) {
							orientation = legacyOrientationLookup[rawOrientation]
						}
					}

					wall.Type = TileType(orientation)
					wall.RawOrientation = rawOrientation
					wall.OrientationUnknown = bits & 0xffffff00
					wall.Zero = byte(bits >> 8)
				case LayerStreamFloor1, LayerStreamFloor2:
					floorIndex := int(layerStreamType) - int(LayerStreamFloor1)
					ds1.Tiles[y][x].Floors[floorIndex].SetPacked(bits)
				case LayerStreamShadow:
					ds1.Tiles[y][x].Shadows[0].SetPacked(bits)
				case LayerStreamSubstitute:
					ds1.Tiles[y][x].Substitutions[0].Unknown = bits
				}
			}
		}
	}

	return nil
}
