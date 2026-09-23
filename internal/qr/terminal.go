package qr

import "strings"

// terminal.go dibuja el QR en una consola. Como una celda de texto es casi el
// doble de alta que de ancha, se usan medios bloques (▀ ▄ █) para meter DOS
// módulos verticales en una sola fila de caracteres: así el código sale
// cuadrado y escaneable, en la mitad de líneas.

// Los caracteres representan, de arriba abajo, la pareja (superior, inferior):
// ambos claros = espacio; solo arriba oscuro = ▀; solo abajo = ▄; ambos = █.
const (
	bothLight = ' '
	topDark   = '▀'
	botDark   = '▄'
	bothDark  = '█'
)

// TerminalString devuelve el QR como texto para la consola, con una zona de
// silencio de `quiet` módulos. dark/light son verdadero/falso; cuando la
// terminal pinta texto claro sobre fondo oscuro conviene invertir (ver Invert).
func (m *Matrix) TerminalString(quiet int) string {
	return m.terminal(quiet, false)
}

// TerminalStringInverted invierte oscuro y claro: útil si el fondo del código
// tiene que ser el color del texto (por ejemplo, terminal de fondo claro).
func (m *Matrix) TerminalStringInverted(quiet int) string {
	return m.terminal(quiet, true)
}

func (m *Matrix) terminal(quiet int, invert bool) string {
	if quiet < 0 {
		quiet = 0
	}
	side := m.Size + 2*quiet
	dark := func(x, y int) bool {
		mx, my := x-quiet, y-quiet
		on := m.At(mx, my)
		if invert {
			return !on
		}
		return on
	}

	var b strings.Builder
	b.Grow(side * (side/2 + 1) * 4)
	for y := 0; y < side; y += 2 {
		for x := 0; x < side; x++ {
			top := dark(x, y)
			bot := y+1 < side && dark(x, y+1)
			switch {
			case top && bot:
				b.WriteRune(bothDark)
			case top:
				b.WriteRune(topDark)
			case bot:
				b.WriteRune(botDark)
			default:
				b.WriteRune(bothLight)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// TerminalLines es como TerminalString pero devuelve las líneas sueltas, para
// que la herramienta les ponga el margen y las centre.
func (m *Matrix) TerminalLines(quiet int, invert bool) []string {
	s := m.terminal(quiet, invert)
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}
