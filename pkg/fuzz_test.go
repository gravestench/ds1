package pkg

import "testing"

func FuzzFromBytes(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 16))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = FromBytes(data) })
}
