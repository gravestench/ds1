package pkg

const (
	versionEncodesFiles              = 3
	versionEncodesFloors             = 4
	versionSimpleLayersHigh          = 4 // up to v4, a ds1 only had 1 of each layer type
	versionEncodesWalls              = 16
	versionEncodesAct                = 8
	versionEncodesSubstitutionLayers = 10
	versionEncodesSubstitutionGroups = 12
	versionEncodesNpcs               = 14
	versionEncodesNpcExtraData       = 15
	versionHasUnknownBytes2          = 18

	versionUnknownBytes1Low  = 9
	versionUnknownBytes1High = 13
)

type Version int32

func (v Version) EncodesAct() bool {
	return v >= versionEncodesAct
}

func (v Version) EncodesSubstitutionLayers() bool {
	return v >= versionEncodesSubstitutionLayers
}

func (v Version) EncodesFiles() bool {
	return v >= versionEncodesFiles
}

func (v Version) HasUnknownBytes1() bool {
	return v >= versionUnknownBytes1Low && v <= versionUnknownBytes1High
}

func (v Version) EncodesFloorLayers() bool {
	return v >= versionEncodesFloors
}

func (v Version) EncodesWallLayers() bool {
	return v >= versionEncodesWalls
}

func (v Version) EncodesSubstitutionGroups() bool {
	return v >= versionEncodesSubstitutionGroups
}

func (v Version) HasUnknownBytes2() bool {
	return v >= versionHasUnknownBytes2
}

func (v Version) EncodesSimpleLayers() bool {
	return v < versionSimpleLayersHigh
}

func (v Version) EncodesNPCs() bool {
	return v >= versionEncodesNpcs
}

func (v Version) EncodesNPCExtraData() bool {
	return v >= versionEncodesNpcExtraData
}
