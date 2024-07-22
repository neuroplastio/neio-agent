package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/hidapi"
	"github.com/neuroplastio/neuroplastio/hidapi/hidusage/usagepages"
)

type Char struct{}

func (a Char) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Char",
		Description: "Maps one ASCII character to a keyboard Usage. Only works for US layouts.",
		Signature:   "char(char: string, rightShift: boolean = false, modDelay: Duration = 1ms)",
	}
}

func (a Char) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	char := p.Args().String("char")
	if len(char) != 1 {
		return nil, fmt.Errorf("char must be a single character")
	}
	return NewCharActionHandler(p.Context(), rune(char[0]), p.Args().Boolean("rightShift"), p.Args().Duration("modDelay"))
}

func NewCharActionHandler(ctx context.Context, char rune, rightShift bool, modDelay time.Duration) (flowapi.ActionHandler, error) {
	key, shift, err := GetAsciiCharKey(char)
	if err != nil {
		return nil, err
	}
	usage := hidapi.NewUsage(usagepages.KeyboardKeypad, uint16(key))
	usageHandler := flowapi.NewActionUsageHandler(usage)
	if shift {
		shiftKey := usagepages.KeyLeftShift
		if rightShift {
			shiftKey = usagepages.KeyRightShift
		}
		shiftUsage := hidapi.NewUsage(usagepages.KeyboardKeypad, uint16(shiftKey))
		return NewModHandler(ctx, []hidapi.Usage{shiftUsage}, usageHandler, modDelay), nil
	}
	return usageHandler, nil
}

var asciiCharMap = map[rune]uint8{
	'-':  usagepages.KeyMinus,
	'=':  usagepages.KeyEqual,
	'[':  usagepages.KeyLeftBracket,
	']':  usagepages.KeyRightBracket,
	'\\': usagepages.KeyBackslash,
	';':  usagepages.KeySemicolon,
	'\'': usagepages.KeyCode34,
	',':  usagepages.KeyComma,
	'.':  usagepages.KeyPeriod,
	'/':  usagepages.KeySlash,
	'`':  usagepages.KeyGraveAccent,

	' ': usagepages.KeySpacebar,
}

var asciiCharMapShifted = map[rune]uint8{
	'_': usagepages.KeyMinus,
	'+': usagepages.KeyEqual,
	'{': usagepages.KeyLeftBracket,
	'}': usagepages.KeyRightBracket,
	'|': usagepages.KeyBackslash,
	':': usagepages.KeySemicolon,
	'"': usagepages.KeyCode34,
	'<': usagepages.KeyComma,
	'>': usagepages.KeyPeriod,
	'?': usagepages.KeySlash,
	'~': usagepages.KeyGraveAccent,

	'!': usagepages.Key1,
	'@': usagepages.Key2,
	'#': usagepages.Key3,
	'$': usagepages.Key4,
	'%': usagepages.Key5,
	'^': usagepages.Key6,
	'&': usagepages.Key7,
	'*': usagepages.Key8,
	'(': usagepages.Key9,
	')': usagepages.Key0,
}

func GetAsciiCharKey(r rune) (uint8, bool, error) {
	if r < ' ' || r > '~' {
		return 0, false, fmt.Errorf("char must be a printable ASCII character, got %c", r)
	}
	switch {
	case asciiCharMap[r] != 0:
		return asciiCharMap[r], false, nil
	case asciiCharMapShifted[r] != 0:
		return asciiCharMapShifted[r], true, nil
	case r >= 'a' && r <= 'z':
		return usagepages.KeyA + uint8(r-'a'), false, nil
	case r >= 'A' && r <= 'Z':
		return usagepages.KeyA + uint8(r-'A'), true, nil
	case r >= '0' && r <= '9':
		return usagepages.Key0 + uint8(r-'0'), false, nil
	default:
		return 0, false, fmt.Errorf("unsupported character: %c", r)
	}
}
