<!-- PROJECT LOGO -->
<h1 align="center">DS1</h1>
<p align="center">
  Package for transcoding DS1 map-stamp files found 
  <br />
  inside Diablo 2 MPQ archives, representing tilesets.
  <br />
  <br />
  <a href="https://github.com/gravestench/ds1/issues">Report Bug</a>
  ·
  <a href="https://github.com/gravestench/ds1/issues">Request Feature</a>
</p>

<!-- ABOUT THE PROJECT -->
## About

The DS1 Transcoder package provides a Go implementation for handling DS1 files, 
which represent "stamp" data used to construct maps within Diablo 2 MPQ 
archives. This package is designed to efficiently work with DS1 map-stamp files.

## Project Structure
* `pkg/` - This directory contains the core DS1 transcoder library. This is the 
directory to import if you want to write new golang applications using this 
library. Aliases to this are made in `exports.go`
    ```golang
   import (
	   "github.com/gravestench/ds1"
  )
    ```
* `cmd/` - This directory contains command-line and graphical applications, each having their own sub-directory.
* `assets/` - This directory contains (or will contain...) files, like the images displayed in this README, or test dc6 file data.


## Getting Started

### Prerequisites
To use this DS1 transcoder package, ensure you have Go 1.16 or a later version 
installed, and your Go environment is set up correctly.

### Installation
To install the package, you can use Go's standard `go get` command:

```shell
go get -u github.com/gravestench/ds1
```

### Usage
Once you have installed the package, you can use it in your Go applications by 
importing it as follows:

```golang
import "github.com/gravestench/ds1"
```

#### Load and save a DS1 file
To load a DS1 file from a byte slice, use the `FromBytes` function:

```golang
fileData := // Load your DS1 file data here as a byte slice
stamp, err := ds1.FromBytes(fileData)
if err != nil {
    // Handle error
}

// Preserve the source version, including legacy layouts.
encoded, err := stamp.Encode()
if err != nil {
    // Handle error
}
```

For a newly constructed model, use `ds1.LatestVersion` to select the canonical
v18 layout emitted by DS1Edit. Layer counts and tile slices must match the
selected version; the encoder reports data that the target version cannot
represent instead of silently discarding it.

### Features
The DS1 transcoder package offers the following features:
- Incrementally read and validate DS1 files.
- Encode DS1 versions 1 through 18, including DS1Edit's canonical v18 layout.
- Preserve legacy orientation values, header dwords, substitution metadata, and NPC path ordering during round trips.
- Extract information about the DS1's version, width, height, act, and layer types.
- Access objects, tiles, substitution groups, and other relevant data from the DS1 file.

## Command-line tools

```shell
go install ./cmd/ds1-info
ds1-info path/to/map.ds1
```

`ds1-info` validates and decodes a map stamp, then writes its dimensions, layer counts, object count, substitutions, and referenced files as JSON.

<!-- CONTRIBUTING -->
## Contributing

Contributions to the DS1 transcoder package are welcome and encouraged. If you find any issues or have improvements to suggest, feel free to open an issue or submit a pull request.

To contribute to the project, follow these steps:

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<!-- MARKDOWN LINKS & IMAGES -->
[ds1]: https://github.com/gravestench/ds1
