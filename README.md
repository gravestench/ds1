<!-- PROJECT LOGO -->
<h1 align="center">DS1</h1>
<p align="center">
  Package for transcoding DS1 tileset files found 
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
archives. This package is designed to efficiently work with DS1 tileset files 
and includes support for both reading and decoding them.

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

#### Load DS1 File
To load a DS1 file from a byte slice, use the `FromBytes` function:

```golang
fileData := // Load your DS1 file data here as a byte slice
ds1, err := ds1.FromBytes(fileData)
if err != nil {
    // Handle error
}
// Use the ds1 object to access the DS1 file data
```

### Features
The DS1 transcoder package offers the following features:
- Efficiently read and parse DS1 image files.
- Extract information about the DS1's version, width, height, act, and layer types.
- Access objects, tiles, substitution groups, and other relevant data from the DS1 file.

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