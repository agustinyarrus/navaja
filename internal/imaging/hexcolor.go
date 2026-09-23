package imaging

import (
	"fmt"
	"image/color"
	"strings"
)

// ParseHexColor entiende un color en hex: #rgb, #rrggbb, #rrggbbaa (con o sin
// #), o los nombres "white"/"black"/"blanco"/"negro". Devuelve NRGBA.
func ParseHexColor(s string) (color.Color, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "white", "blanco", "":
		return color.NRGBA{255, 255, 255, 255}, nil
	case "black", "negro":
		return color.NRGBA{0, 0, 0, 255}, nil
	case "none", "transparente", "transparent":
		return color.NRGBA{0, 0, 0, 0}, nil
	}
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	expand := func(b byte) uint8 {
		v := hexNibble(b)
		return v<<4 | v
	}
	switch len(h) {
	case 3: // rgb
		if !isHex(h) {
			break
		}
		return color.NRGBA{expand(h[0]), expand(h[1]), expand(h[2]), 255}, nil
	case 6: // rrggbb
		if !isHex(h) {
			break
		}
		return color.NRGBA{byteAt(h, 0), byteAt(h, 2), byteAt(h, 4), 255}, nil
	case 8: // rrggbbaa
		if !isHex(h) {
			break
		}
		return color.NRGBA{byteAt(h, 0), byteAt(h, 2), byteAt(h, 4), byteAt(h, 6)}, nil
	}
	return nil, fmt.Errorf("color %q inválido (usá #rrggbb, #rgb o \"blanco\"/\"negro\")", s)
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func hexNibble(b byte) uint8 {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}

func byteAt(s string, i int) uint8 { return hexNibble(s[i])<<4 | hexNibble(s[i+1]) }
