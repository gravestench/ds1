package ds1

import (
	"io"

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

const LatestVersion = pkg.LatestVersion

func FromBytes(data []byte) (*DS1, error) {
	return pkg.FromBytes(data)
}

func FromReader(source io.Reader) (*DS1, error) { return pkg.FromReader(source) }

func Encode(value *DS1) ([]byte, error) { return value.Encode() }

func EncodeTo(destination io.Writer, value *DS1) error { return value.EncodeTo(destination) }
