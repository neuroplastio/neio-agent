package bits

import (
	"testing"
)

type bitTestCase struct {
	a             string
	aMissing      uint8
	aLen          int
	b             string
	bMissing      uint8
	bLen          int
	result        string
	resultMissing uint8
}

func TestBitSet(t *testing.T) {
	tests := []bitTestCase{
		{
			a:             "11111111 0011",
			b:             "11111111 0011",
			aLen:          2,
			bLen:          2,
			aMissing:      4,
			bMissing:      4,
			result:        "11111111 00111111 11110011",
			resultMissing: 0,
		},
		{
			a:             "10101010 111",
			b:             "10101010 111",
			aMissing:      5,
			bMissing:      5,
			aLen:          2,
			bLen:          2,
			result:        "10101010 11110101 010111",
			resultMissing: 2,
		},
		{
			a:             "11111111 0011111",
			b:             "00000000 1",
			aLen:          2,
			bLen:          2,
			aMissing:      1,
			bMissing:      7,
			result:        "11111111 00111110 00000001",
			resultMissing: 0,
		},
		{
			a:             "11111111 11111111",
			b:             "00000000 00000000",
			aLen:          2,
			bLen:          2,
			aMissing:      0,
			bMissing:      0,
			result:        "11111111 11111111 00000000 00000000",
			resultMissing: 0,
		},
		{
			a:             "11111111",
			b:             "00000000",
			aLen:          1,
			bLen:          1,
			aMissing:      0,
			bMissing:      0,
			result:        "11111111 00000000",
			resultMissing: 0,
		},
	}

	for i, test := range tests {
		a, err := NewBitSetFromString(test.a)
		if err != nil {
			t.Errorf("%d: %s", i, err)
		}
		if a.String() != test.a {
			t.Errorf("%d: %s != %s", i, a.String(), test.a)
		}
		if test.aMissing != a.missingBits {
			t.Errorf("%d: A lastByteShift: %d != %d", i, test.aMissing, a.missingBits)
		}
		if test.aLen != len(a.bytes) {
			t.Errorf("%d: A len: %d != %d", i, test.aLen, len(a.bytes))
		}
		b, err := NewBitSetFromString(test.b)
		if err != nil {
			t.Errorf("%d: %s", i, err)
		}
		if b.String() != test.b {
			t.Errorf("%d: %s != %s", i, b.String(), test.b)
		}
		if test.bMissing != b.missingBits {
			t.Errorf("%d: B lastByteShift: %d != %d", i, test.bMissing, b.missingBits)
		}
		if test.bLen != len(b.bytes) {
			t.Errorf("%d: B len: %d != %d", i, test.bLen, len(b.bytes))
		}
		concat := ConcatBits(a, b)
		if test.resultMissing != concat.missingBits {
			t.Errorf("%d: result lastByteShift: %d != %d", i, test.resultMissing, concat.missingBits)
		}
		if concat.String() != test.result {
			t.Errorf("%d: %s != %s", i, concat.String(), test.result)
		}
	}

}
