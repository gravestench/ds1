// Command ds1-info prints DS1 metadata as JSON.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gravestench/ds1"
)

type summary struct {
	Version            int32    `json:"version"`
	Width              int32    `json:"width"`
	Height             int32    `json:"height"`
	Act                int32    `json:"act"`
	Walls              int32    `json:"wall_layers"`
	Floors             int32    `json:"floor_layers"`
	Objects            int      `json:"objects"`
	Files              []string `json:"files"`
	SubstitutionGroups int      `json:"substitution_groups"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s FILE.ds1\n", os.Args[0])
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	d, err := ds1.FromBytes(data)
	if err != nil {
		fail(err)
	}
	s := summary{int32(d.Version), d.Width, d.Height, d.Act, d.NumberOfWalls, d.NumberOfFloors, len(d.Objects), d.Files, len(d.SubstitutionGroups)}
	if err := json.NewEncoder(os.Stdout).Encode(s); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintf(os.Stderr, "ds1-info: %v\n", err); os.Exit(1) }
