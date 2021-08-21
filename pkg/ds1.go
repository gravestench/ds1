package pkg

import (
	"github.com/OpenDiablo2/bitstream"
	"github.com/gravestench/mathlib"
)

const maxActNumber = 5

// DS1 represents the "stamp" data that is used to build up maps.
type DS1 struct {
	Files                      []string            // FilePtr table of file string pointers
	Objects                    []Object            // Objects
	Tiles                      [][]TileRecord      // The tile data for the DS1
	SubstitutionGroups         []SubstitutionGroup // Substitution groups for the DS1
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
	stream := bitstream.NewReader().FromBytes(fileData...)

	ds1 = &DS1{
		Act:                        1,
		NumberOfFloors:             0,
		NumberOfWalls:              0,
		NumberOfShadowLayers:       1,
		NumberOfSubstitutionLayers: 0,
	}

	if v, err := stream.Next(4).Bytes().AsInt32(); err != nil {
		return nil, err
	} else {
		ds1.Version = Version(v)
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

	if ds1.Version.EncodesAct() {
		if ds1.Act, err = stream.Next(4).Bytes().AsInt32(); err != nil {
			return nil, err
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
		const unknownBytes1Length = 8
		stream.Next(unknownBytes1Length).Bytes() // skipping
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

func (ds1 *DS1) loadObjects(br *bitstream.Reader) error {
	if ds1.Version >= 2 { //nolint:gomnd // Version number
		numberOfObjects, err := br.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
		}

		ds1.Objects = make([]Object, numberOfObjects)

		for objIdx := 0; objIdx < int(numberOfObjects); objIdx++ {
			newObject := Object{}
			newObject.Type, err = br.Next(4).Bytes().AsInt32()
			newObject.ID, err = br.Next(4).Bytes().AsInt32()
			newObject.X, err = br.Next(4).Bytes().AsInt32()
			newObject.Y, err = br.Next(4).Bytes().AsInt32()
			newObject.Flags, err = br.Next(4).Bytes().AsInt32()

			ds1.Objects[objIdx] = newObject
		}
	} else {
		ds1.Objects = make([]Object, 0)
	}

	return nil
}

func (ds1 *DS1) loadSubstitutions(stream *bitstream.Reader) (err error) {
	ds1.SubstitutionGroups = make([]SubstitutionGroup, 0)

	hasSubType := ds1.SubstitutionType == 1 || ds1.SubstitutionType == 2
	hasEncodedSubGroups := ds1.Version.EncodesSubstitutionGroups() && hasSubType

	if !hasEncodedSubGroups {
		return nil
	}

	if ds1.Version.HasUnknownBytes2() {
		stream.Next(4).Bytes() // skip int32
	}

	numberOfSubGroups, err := stream.Next(4).Bytes().AsInt32()
	if err != nil {
		return err
	}

	ds1.SubstitutionGroups = make([]SubstitutionGroup, numberOfSubGroups)

	for subIdx := 0; subIdx < int(numberOfSubGroups); subIdx++ {
		newSub := SubstitutionGroup{}
		newSub.TileX, err = stream.Next(4).Bytes().AsInt32()
		newSub.TileY, err = stream.Next(4).Bytes().AsInt32()
		newSub.WidthInTiles, err = stream.Next(4).Bytes().AsInt32()
		newSub.HeightInTiles, err = stream.Next(4).Bytes().AsInt32()
		newSub.Unknown, err = stream.Next(4).Bytes().AsInt32()

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

func (ds1 *DS1) loadNPCs(stream *bitstream.Reader) (err error) {
	if !ds1.Version.EncodesNPCs() {
		return nil
	}

	numberOfNpcs, err := stream.Next(4).Bytes().AsInt32()
	if err != nil {
		return err
	}

	for npcIdx := 0; npcIdx < int(numberOfNpcs); npcIdx++ {
		numPaths, err := stream.Next(4).Bytes().AsInt32()
		if err != nil {
			return err
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
			if err = ds1.loadNpcPaths(stream, objIdx, int(numPaths)); err != nil {
				return err
			}

			continue
		}

		const normalBytesPerNpcPath = 2


		if ds1.Version.EncodesNPCExtraData() {
			stream.Next(normalBytesPerNpcPath + 1).Bytes()
		} else {
			stream.Next(normalBytesPerNpcPath).Bytes()
		}
	}

	return nil
}

func (ds1 *DS1) loadNpcPaths(br *bitstream.Reader, objIdx, numPaths int) (err error) {
	if ds1.Objects[objIdx].Paths == nil {
		ds1.Objects[objIdx].Paths = make([]Path, numPaths)
	}

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

func (ds1 *DS1) loadLayerStreams(stream *bitstream.Reader, layerStream []LayerStreamType) (err error) {
	var dirLookup = []int32{
		0x00, 0x01, 0x02, 0x01, 0x02, 0x03, 0x03, 0x05, 0x05, 0x06,
		0x06, 0x07, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E,
		0x0F, 0x10, 0x11, 0x12, 0x14,
	}

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
					ds1.Tiles[y][x].Walls[wallIndex].Prop1 = byte(bits & 0x000000FF)            //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Walls[wallIndex].Sequence = byte((bits & 0x00003F00) >> 8)  //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Walls[wallIndex].Unknown1 = byte((bits & 0x000FC000) >> 14) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Walls[wallIndex].Style = byte((bits & 0x03F00000) >> 20)    //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Walls[wallIndex].Unknown2 = byte((bits & 0x7C000000) >> 26) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Walls[wallIndex].Hidden = byte((bits&0x80000000)>>31) > 0   //nolint:gomnd // Bitmask
				case LayerStreamOrientation1, LayerStreamOrientation2,
					LayerStreamOrientation3, LayerStreamOrientation4:
					wallIndex := int(layerStreamType) - int(LayerStreamOrientation1)
					c := int32(bits & 0x000000FF) //nolint:gomnd // Bitmask

					if ds1.Version < 7 { //nolint:gomnd // Version number
						if c < int32(len(dirLookup)) {
							c = dirLookup[c]
						}
					}

					ds1.Tiles[y][x].Walls[wallIndex].Type = TileType(c)
					ds1.Tiles[y][x].Walls[wallIndex].Zero = byte((bits & 0xFFFFFF00) >> 8) //nolint:gomnd // Bitmask
				case LayerStreamFloor1, LayerStreamFloor2:
					floorIndex := int(layerStreamType) - int(LayerStreamFloor1)
					ds1.Tiles[y][x].Floors[floorIndex].Prop1 = byte(bits & 0x000000FF)            //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Floors[floorIndex].Sequence = byte((bits & 0x00003F00) >> 8)  //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Floors[floorIndex].Unknown1 = byte((bits & 0x000FC000) >> 14) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Floors[floorIndex].Style = byte((bits & 0x03F00000) >> 20)    //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Floors[floorIndex].Unknown2 = byte((bits & 0x7C000000) >> 26) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Floors[floorIndex].Hidden = byte((bits&0x80000000)>>31) > 0   //nolint:gomnd // Bitmask
				case LayerStreamShadow:
					ds1.Tiles[y][x].Shadows[0].Prop1 = byte(bits & 0x000000FF)            //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Shadows[0].Sequence = byte((bits & 0x00003F00) >> 8)  //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Shadows[0].Unknown1 = byte((bits & 0x000FC000) >> 14) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Shadows[0].Style = byte((bits & 0x03F00000) >> 20)    //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Shadows[0].Unknown2 = byte((bits & 0x7C000000) >> 26) //nolint:gomnd // Bitmask
					ds1.Tiles[y][x].Shadows[0].Hidden = byte((bits&0x80000000)>>31) > 0   //nolint:gomnd // Bitmask
				case LayerStreamSubstitute:
					ds1.Tiles[y][x].Substitutions[0].Unknown = bits
				}
			}
		}
	}

	return nil
}
