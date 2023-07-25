package ds1

import (
	"github.com/gravestench/ds1/pkg"
)

type (
	DS1                = pkg.DS1
	Object             = pkg.Object
	Version            = pkg.Version
	Path               = pkg.Path
	FloorShadowRecord  = pkg.FloorShadowRecord
	LayerStreamType    = pkg.LayerStreamType
	SubstitutionRecord = pkg.SubstitutionRecord
	SubstitutionGroup  = pkg.SubstitutionGroup
	TileType           = pkg.TileType
	TileRecord         = pkg.TileRecord
	WallRecord         = pkg.WallRecord
)

func FromBytes(data []byte) (*DS1, error) {
	return pkg.FromBytes(data)
}
