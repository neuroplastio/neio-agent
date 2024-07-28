package hiddesc

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"
)

func TestMoonlander(t *testing.T) {
	bb, err := os.ReadFile("../../testdata/keyboards/zsa-moonlander/desc/01-boot-kb.desc")
	if err != nil {
		t.Fatal(err)
	}
	parser := NewDescriptorDecoder(bytes.NewBuffer(bb))

	desc, err := parser.Decode()
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)
	encoder := NewDescriptorEncoder(buf, desc)
	err = encoder.Encode()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hex.Dump(buf.Bytes()))
}

func TestWacom(t *testing.T) {
	bb, err := os.ReadFile("../../testdata/wacom.desc")
	if err != nil {
		t.Fatal(err)
	}
	parser := NewDescriptorDecoder(bytes.NewReader(bb))

	desc, err := parser.Decode()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(desc)

	buf := bytes.NewBuffer(nil)
	encoder := NewDescriptorEncoder(buf, desc)
	err = encoder.Encode()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hex.Dump(buf.Bytes()))

	t.Fatal("")
}
