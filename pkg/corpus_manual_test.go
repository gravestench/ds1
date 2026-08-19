package pkg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManualDS1CorpusRoundTrip(t *testing.T) {
	root := os.Getenv("DS1_TEST_CORPUS")
	if root == "" {
		t.Skip("DS1_TEST_CORPUS is not set")
	}
	var count int
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".ds1") {
			return nil
		}
		count++
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: read: %v", path, err)
			return nil
		}
		value, err := FromBytes(data)
		if err != nil {
			t.Errorf("%s: decode: %v", path, err)
			return nil
		}
		encoded, err := value.Encode()
		if err != nil {
			t.Errorf("%s: encode: %v", path, err)
			return nil
		}
		if !bytes.Equal(encoded, data) {
			offset := 0
			for offset < len(data) && offset < len(encoded) && data[offset] == encoded[offset] {
				offset++
			}
			dataEnd := offset + 24
			if dataEnd > len(data) {
				dataEnd = len(data)
			}
			encodedEnd := offset + 24
			if encodedEnd > len(encoded) {
				encodedEnd = len(encoded)
			}
			t.Errorf("%s: encoded bytes differ (%d vs %d), first offset %d: %x -> %x", path, len(encoded), len(data), offset, data[offset:dataEnd], encoded[offset:encodedEnd])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no DS1 files found")
	}
}
