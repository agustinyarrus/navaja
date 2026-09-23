//go:build windows

// Package clip2qr implementa `clip2qr`: toma el texto del portapapeles (o de un
// argumento / la entrada estándar) y dibuja un código QR en la terminal, con la
// opción de guardarlo como PNG. Pensado para pasar una URL del escritorio al
// teléfono sin subir nada a ningún lado.
package clip2qr

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/agustinyarrus/navaja/internal/cli"
	"github.com/agustinyarrus/navaja/internal/fsx"
	"github.com/agustinyarrus/navaja/internal/qr"
	"github.com/agustinyarrus/navaja/internal/tui"
	"github.com/agustinyarrus/navaja/internal/win"
)

type opciones struct {
	nivel    string
	quiet    int
	invertir bool
	png      string
	escala   int
	textoArg string
	stdin    bool
}

// Main es el punto de entrada del subcomando.
func Main(t *tui.Term, version string, args []string) int {
	o := opciones{nivel: "m", quiet: 2, escala: 8}
	app := cli.New("clip2qr", version, "convierte el portapapeles en un QR en la terminal")
	app.Usage = []string{
		"clip2qr                 QR de lo que haya en el portapapeles",
		"clip2qr \"texto o URL\"    QR de un texto puntual",
		"echo hola | clip2qr -    QR de la entrada estándar",
	}
	app.Section("contenido")
	app.String(&o.textoArg, "text", 0, "TEXTO", "usar este texto en vez del portapapeles")
	app.Bool(&o.stdin, "stdin", 0, "leer el texto de la entrada estándar")
	app.Section("código")
	app.Enum(&o.nivel, "level", 'e', "corrección de errores (más alto = más robusto y más denso)", "l", "m", "q", "h")
	app.Int(&o.quiet, "quiet", 'q', "N", "módulos de borde claro alrededor", 0, 10)
	app.Bool(&o.invertir, "invert", 'i', "invertir claro/oscuro (para terminales de fondo claro)")
	app.Section("guardar")
	app.String(&o.png, "png", 'o', "ARCHIVO", "además, guardar el QR como PNG")
	app.Int(&o.escala, "scale", 's', "N", "píxeles por módulo en el PNG", 1, 64)
	app.Examples = []cli.Example{
		{Cmd: "clip2qr", Desc: "el caso típico: copiás una URL y la escaneás con el teléfono"},
		{Cmd: "clip2qr -e h -o wifi.png", Desc: "QR robusto y guardado como imagen"},
		{Cmd: "clip2qr --text \"https://…\"", Desc: "sin tocar el portapapeles"},
	}
	app.Notes = []string{
		"Todo local: el texto no sale de la máquina.",
		"El QR se dibuja con medios bloques para que quede cuadrado; si tu terminal no los muestra bien, guardá el PNG.",
	}

	pos, done, code := app.Start(t, args)
	if done {
		return code
	}

	texto, fuente, err := leerTexto(o, pos)
	if err != nil {
		return fallo(t, err)
	}
	if strings.TrimSpace(texto) == "" {
		return fallo(t, fmt.Errorf("no hay texto para codificar (%s)", fuente))
	}

	nivel, _ := qr.ParseLevel(o.nivel)
	m, err := qr.Encode([]byte(texto), qr.Options{Level: nivel, Mask: -1})
	if err != nil {
		return fallo(t, err)
	}

	t.Lines(t.Header("clip2qr", "convierte el portapapeles en un QR", version))
	dibujar(t, m, o.quiet, o.invertir)

	guardado := ""
	if o.png != "" {
		if err := guardarPNG(o.png, m, o.escala, o.quiet); err != nil {
			return fallo(t, err)
		}
		guardado = fsx.Display(o.png)
	}

	t.Lines(tarjeta(t, m, texto, fuente, guardado))
	return cli.ExitOK
}

// leerTexto decide de dónde sale el contenido: argumento explícito, --text,
// --stdin / "-", o el portapapeles por defecto.
func leerTexto(o opciones, pos []string) (texto, fuente string, err error) {
	switch {
	case o.textoArg != "":
		return o.textoArg, "argumento --text", nil
	case o.stdin || (len(pos) == 1 && pos[0] == "-"):
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", "", fmt.Errorf("no pude leer la entrada estándar: %w", err)
		}
		return strings.TrimRight(string(b), "\r\n"), "entrada estándar", nil
	case len(pos) > 0:
		return strings.Join(pos, " "), "argumento", nil
	default:
		txt, err := win.ClipboardText()
		if err != nil {
			return "", "portapapeles", err
		}
		return txt, "portapapeles", nil
	}
}

// dibujar centra el QR en la consola y lo pinta.
func dibujar(t *tui.Term, m *qr.Matrix, quiet int, invert bool) {
	// En una consola de fondo oscuro, los módulos "oscuros" del QR deben verse
	// como el color de fondo y los "claros" como tinta: por eso se invierte por
	// defecto (salvo que el usuario pida lo contrario).
	lines := m.TerminalLines(quiet, !invert)
	ancho := t.Width()
	pad := 0
	if len(lines) > 0 {
		pad = max(0, (ancho-len([]rune(lines[0])))/2)
	}
	margen := strings.Repeat(" ", pad)
	out := make([]string, 0, len(lines)+2)
	out = append(out, "")
	for _, l := range lines {
		out = append(out, margen+t.Paint(tui.Text, l))
	}
	out = append(out, "")
	t.Lines(out)
}

func guardarPNG(path string, m *qr.Matrix, escala, quiet int) error {
	return fsx.WriteAtomic(path, func(w io.Writer) error {
		return png.Encode(w, m.Image(escala, max(quiet, 4)))
	})
}

func tarjeta(t *tui.Term, m *qr.Matrix, texto, fuente, guardado string) []string {
	preview := texto
	if len([]rune(preview)) > 48 {
		preview = string([]rune(preview)[:47]) + "…"
	}
	preview = strings.ReplaceAll(strings.ReplaceAll(preview, "\n", "⏎"), "\r", "")
	items := []tui.Item{
		{Label: "Contenido", Value: preview, Dot: tui.Teal},
		{Label: "Longitud", Value: tui.Count(int64(len([]rune(texto))), "carácter", "caracteres"), Dot: tui.Lavender},
		{Label: "Fuente", Value: fuente, Dot: tui.Sky},
		{Label: "Versión QR", Value: fmt.Sprintf("%d (%d×%d)", m.Version, m.Size, m.Size), Dot: tui.Peach},
		{Label: "Corrección", Value: "nivel " + m.Level.String(), Dot: tui.Pink},
	}
	if guardado != "" {
		items = append(items, tui.Item{Label: "PNG", Value: guardado, Dot: tui.Sage})
	}
	return t.Card("clip2qr", items)
}

func fallo(t *tui.Term, err error) int {
	t.Blank()
	t.Line(t.Paint(tui.Rose, "✗ ") + t.Paint(tui.Text, err.Error()))
	t.Blank()
	return cli.ExitFailure
}
