package bits

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func NewScanner(data []byte) *Scanner {
	return &Scanner{
		bytes: data,
	}
}

type Scanner struct {
	bitOffset int
	bytes     []byte
}

func (s *Scanner) Next(bitSize int) Bits {
	byteStart := s.bitOffset / 8
	bitStart := s.bitOffset % 8
	byteSize := bitSize / 8
	missingBits := 8 - uint8(bitSize%8)
	if missingBits == 8 {
		missingBits = 0
	}
	if missingBits > 0 {
		byteSize++
	}
	s.bitOffset += bitSize
	if bitStart == 0 {
		return Bits{
			bytes:       s.bytes[byteStart : byteStart+byteSize],
			missingBits: missingBits,
		}
	}
	result := make([]byte, byteSize)
	for i := 0; i < byteSize; i++ {
		result[i] = s.bytes[byteStart+i] << bitStart
		result[i] |= s.bytes[byteStart+i+1] >> (8 - bitStart)
	}
	return Bits{
		bytes:       result,
		missingBits: missingBits,
	}
}

type Bits struct {
	missingBits uint8
	bytes       []byte
}

func (b Bits) String() string {
	result := ""
	for i, byte := range b.bytes {
		isLast := i == len(b.bytes)-1
		if isLast && b.missingBits > 0 {
			result += fmt.Sprintf("%08b", byte)[:8-b.missingBits]
			continue
		}
		result += fmt.Sprintf("%08b", byte)
		if !isLast {
			result += " "
		}
	}
	return result
}

func (b Bits) Equal(other Bits) bool {
	if b.missingBits != other.missingBits {
		return false
	}
	if len(b.bytes) != len(other.bytes) {
		return false
	}
	for i, byte := range b.bytes {
		if byte != other.bytes[i] {
			return false
		}
	}
	return true
}

func (b Bits) MissingBits() uint8 {
	return b.missingBits
}

func (b Bits) Bytes() []byte {
	return b.bytes
}

func (b Bits) Len() int {
	return len(b.bytes)*8 - int(b.missingBits)
}

func (b Bits) LenUint8() int {
	return b.Len() / 8
}

func (b Bits) LenUint16() int {
	return b.Len() / 16
}

func (b Bits) LenUint32() int {
	return b.Len() / 32
}

func (b Bits) IsSet(bit int) bool {
	if bit >= b.Len() {
		return false
	}
	byteOffset := bit / 8
	bitOffset := bit % 8
	return b.bytes[byteOffset]&(1<<bitOffset) != 0
}

func (b Bits) Set(bit int) bool {
	if bit >= b.Len() {
		return false
	}
	byteOffset := bit / 8
	bitOffset := bit % 8
	changed := b.bytes[byteOffset]&(1<<bitOffset) == 0
	b.bytes[byteOffset] |= 1 << bitOffset
	return changed
}

func (b Bits) Clear(bit int) bool {
	if bit >= b.Len() {
		return false
	}
	byteOffset := bit / 8
	bitOffset := bit % 8
	changed := b.bytes[byteOffset]&(1<<bitOffset) != 0
	b.bytes[byteOffset] &^= 1 << bitOffset
	return changed
}

func (b Bits) ClearAll() bool {
	changed := false
	for i := range b.bytes {
		if b.bytes[i] != 0 {
			changed = true
		}
		b.bytes[i] = 0
	}
	return changed
}

func (b Bits) IsEmpty() bool {
	for _, byte := range b.bytes {
		if byte != 0 {
			return false
		}
	}
	return true
}

func (b Bits) Each(f func(int, bool) bool) {
	i := 0
	size := b.Len()
	for _, b := range b.bytes {
		for j := 0; j < 8; j++ {
			if i >= size {
				return
			}
			if !f(i, b&(1<<j) != 0) {
				return
			}
			i++
		}
	}
}

func (b Bits) EachUint8(f func(int, uint8) bool) {
	for i, byte := range b.bytes {
		if !f(i, byte) {
			return
		}
	}
}

func (b Bits) EachUint16(f func(int, uint16) bool) {
	for i := 0; i < len(b.bytes); i += 2 {
		if !f(i, binary.LittleEndian.Uint16(b.bytes[i:i+2])) {
			return
		}
	}
}

func (b Bits) EachUint24(f func(int, uint16) bool) {
	for i := 0; i < len(b.bytes); i += 3 {
		if !f(i, binary.LittleEndian.Uint16(b.bytes[i:i+3])) {
			return
		}
	}
}

func (b Bits) EachUint32(f func(int, uint32) bool) {
	for i := 0; i < len(b.bytes); i += 4 {
		if !f(i, binary.LittleEndian.Uint32(b.bytes[i:i+4])) {
			return
		}
	}
}

// TODO: support variable bit sizes for integers

func (b Bits) SetUint7(index int, value uint8) {
	// TODO: this breaks with index > 0
	b.bytes[index] = value & 0x7F
}

func (b Bits) SetUint8(index int, value uint8) {
	b.bytes[index] = value
}

func (b Bits) SetUint16(index int, value uint16) {
	binary.LittleEndian.PutUint16(b.bytes[index*2:(index+1)*2], value)
}

func (b Bits) SetUint24(index int, value uint16) {
	// TODO: support 3 byte integers
	binary.LittleEndian.PutUint16(b.bytes[index*3:(index+1)*3], value)
}

func (b Bits) SetUint32(index int, value uint32) {
	binary.LittleEndian.PutUint32(b.bytes[index*4:(index+1)*4], value)
}

func (b Bits) Uint7(index int) uint8 {
	// TODO: this breaks with index > 0
	return b.bytes[index] & 0x7F
}

func (b Bits) Uint8(index int) uint8 {
	return b.bytes[index]
}

func (b Bits) Uint16(index int) uint16 {
	return binary.LittleEndian.Uint16(b.bytes[index*2 : (index+1)*2])
}

func (b Bits) Uint24(index int) uint16 {
	return binary.LittleEndian.Uint16(b.bytes[index*3 : (index+1)*3])
}

func (b Bits) Uint32(index int) uint32 {
	return binary.LittleEndian.Uint32(b.bytes[index*4 : (index+1)*4])
}

func (b Bits) Clone() Bits {
	bytes := make([]byte, len(b.bytes))
	copy(bytes, b.bytes)
	return Bits{
		bytes:       bytes,
		missingBits: b.missingBits,
	}
}

func NewBitSetFromString(s string) (Bits, error) {
	byteStrs := strings.Fields(s)
	b := Bits{
		bytes: make([]byte, len(byteStrs)),
	}
	for i, byteStr := range byteStrs {
		if len(byteStr) == 8 {
			byteVal, err := strconv.ParseUint(byteStr, 2, 8)
			if err != nil {
				return Bits{}, errors.New("invalid byte value")
			}
			b.bytes[i] = byte(byteVal)
		} else {
			if i != len(byteStrs)-1 {
				return Bits{}, errors.New("incomplete byte in the middle of the string")
			}
			b.missingBits = 8 - uint8(len(byteStr))
			byteStr = byteStr + strings.Repeat("0", 8-len(byteStr))
			byteVal, err := strconv.ParseUint(byteStr, 2, 8)
			if err != nil {
				return Bits{}, errors.New("invalid byte value")
			}
			b.bytes[i] = byte(byteVal)
		}
	}
	return b, nil
}

func NewZeros(bitSize int) Bits {
	byteSize := bitSize / 8
	missingBits := 8 - uint8(bitSize%8)
	if missingBits == 8 {
		missingBits = 0
	}
	if missingBits > 0 {
		byteSize++
	}
	return Bits{
		bytes:       make([]byte, byteSize),
		missingBits: missingBits,
	}
}

func New(data []byte, missingBits int) Bits {
	return Bits{
		bytes:       data,
		missingBits: uint8(missingBits),
	}
}

func ConcatBits(l, r Bits) Bits {
	if l.missingBits == 0 {
		return Bits{
			bytes:       append(l.bytes, r.bytes...),
			missingBits: r.missingBits,
		}
	}

	size := len(l.bytes) + len(r.bytes)
	if l.missingBits+r.missingBits >= 8 {
		size--
	}
	result := make([]byte, size)
	copy(result, l.bytes)
	i := len(l.bytes) - 1
	for ri := 0; ri < len(r.bytes); ri++ {
		result[i] |= r.bytes[ri] >> (8 - l.missingBits)
		if i == size-1 {
			break
		}
		result[i+1] = r.bytes[ri] << l.missingBits
		i++
	}
	return Bits{
		bytes:       result,
		missingBits: (l.missingBits + r.missingBits) % 8,
	}
}
