package pkg

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func writeStreamInt32(destination *bytes.Buffer, value int32) {
	_ = binary.Write(destination, binary.LittleEndian, value)
}

type singleByteReader struct{ *bytes.Reader }

func (r singleByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.Reader.Read(p)
}

func TestFromReaderAcceptsChunkedInput(t *testing.T) {
	var encoded bytes.Buffer
	writeStreamInt32(&encoded, 1)    // version
	writeStreamInt32(&encoded, 0)    // width becomes one
	writeStreamInt32(&encoded, 0)    // height becomes one
	encoded.Write(make([]byte, 5*4)) // wall, floor, orientation, substitution, shadow
	decoded, err := FromReader(singleByteReader{bytes.NewReader(encoded.Bytes())})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 1 || decoded.Height != 1 {
		t.Fatalf("dimensions = %dx%d", decoded.Width, decoded.Height)
	}
}
