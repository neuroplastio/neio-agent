package hiddesc

import (
	"bytes"
	"os"
	"testing"
)

func TestMoonlander(t *testing.T) {
	bb, err := os.ReadFile("./testdata/zsa-moonlander-a.bin")
	if err != nil {
		t.Fatal(err)
	}
	parser := NewDescriptorDecoder(bytes.NewBuffer(bb))

	_, err = parser.Decode()
	if err != nil {
		t.Fatal(err)
	}
}
