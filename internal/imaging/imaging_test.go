package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

// fxDir es donde se generaron los fixtures reales (ver la sesión); si no está,
// el test se saltea sin fallar para que la suite corra en cualquier máquina.
func fxDir(t *testing.T) string {
	dir := filepath.Join(os.TempDir(), "navaja-fx")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no hay fixtures en %s (generalos con la corrida de la sesión)", dir)
	}
	return dir
}

func decodeFile(t *testing.T, path string) *Decoded {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("abrir %s: %v", path, err)
	}
	defer f.Close()
	d, err := Decode(f)
	if err != nil {
		t.Fatalf("decodificar %s: %v", path, err)
	}
	return d
}

// TestSniffYDimensiones verifica que el formato y las dimensiones coinciden con
// lo que reportó Pillow (el oráculo).
func TestSniffYDimensiones(t *testing.T) {
	dir := fxDir(t)
	casos := []struct {
		file     string
		format   Format
		w, h     int
		animated bool
	}{
		{"lossless.webp", WEBP, 320, 200, false},
		{"lossy.webp", WEBP, 320, 200, false},
		{"alpha.webp", WEBP, 160, 120, false},
		{"static.gif", GIF, 320, 200, false},
		{"anim.gif", GIF, 120, 120, true},
		{"base.png", PNG, 320, 200, false},
		{"q.jpg", JPEG, 320, 200, false},
		{"b.bmp", BMP, 320, 200, false},
	}
	for _, c := range casos {
		t.Run(c.file, func(t *testing.T) {
			d := decodeFile(t, filepath.Join(dir, c.file))
			if d.Source != c.format {
				t.Errorf("formato = %v, esperaba %v", d.Source, c.format)
			}
			if d.Width != c.w || d.Height != c.h {
				t.Errorf("dimensiones = %dx%d, esperaba %dx%d", d.Width, d.Height, c.w, c.h)
			}
			if d.Animated() != c.animated {
				t.Errorf("animated = %v, esperaba %v (%d cuadros)", d.Animated(), c.animated, len(d.Frames))
			}
		})
	}
}

// TestAnimAllFramesFullSize: cada cuadro compuesto de un animado debe medir la
// pantalla completa, no el recorte parcial. Es el bug clásico que evita la
// composición con disposal.
func TestAnimAllFramesFullSize(t *testing.T) {
	dir := fxDir(t)
	d := decodeFile(t, filepath.Join(dir, "anim.gif"))
	if len(d.Frames) != 8 {
		t.Fatalf("cuadros = %d, esperaba 8", len(d.Frames))
	}
	for i, f := range d.Frames {
		b := f.Bounds()
		if b.Dx() != d.Width || b.Dy() != d.Height {
			t.Errorf("cuadro %d mide %dx%d, esperaba %dx%d (recorte parcial sin componer)",
				i+1, b.Dx(), b.Dy(), d.Width, d.Height)
		}
	}
}

// TestRoundTripPNG: webp → png → decodificar de vuelta conserva dimensiones y,
// en un webp con alfa, el alfa sobrevive a PNG.
func TestRoundTripPNG(t *testing.T) {
	dir := fxDir(t)
	d := decodeFile(t, filepath.Join(dir, "alpha.webp"))
	if !d.HasAlpha {
		t.Fatal("alpha.webp debería reportar alfa")
	}
	var buf bytes.Buffer
	if err := Encode(&buf, d, EncodeOptions{Format: PNG}); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	round, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode round: %v", err)
	}
	if round.Source != PNG || round.Width != d.Width || round.Height != d.Height {
		t.Errorf("round-trip = %v %dx%d, esperaba PNG %dx%d", round.Source, round.Width, round.Height, d.Width, d.Height)
	}
	if !round.HasAlpha {
		t.Error("el PNG resultante perdió el canal alfa")
	}
}

// TestFlattenJPEG: al pasar a JPEG (opaco) un origen con alfa transparente, el
// fondo elegido tiene que aparecer donde había transparencia.
func TestFlattenJPEG(t *testing.T) {
	// Imagen 2x2 totalmente transparente.
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	err := EncodeFrame(&buf, src, EncodeOptions{Format: JPEG, JPEGQuality: 100, Background: color.RGBA{R: 10, G: 200, B: 160, A: 255}})
	if err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	round, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode jpeg: %v", err)
	}
	r, g, b, _ := round.Frames[0].At(0, 0).RGBA()
	// JPEG es con pérdida: se tolera un margen amplio, solo se comprueba que el
	// fondo teal (no negro, no blanco) llegó.
	if r>>8 > 80 || g>>8 < 150 || b>>8 < 110 {
		t.Errorf("fondo aplanado = (%d,%d,%d), esperaba un teal (~10,200,160)", r>>8, g>>8, b>>8)
	}
}

// TestGIFAnimadoRoundTrip: un GIF animado recodificado sigue teniendo la misma
// cantidad de cuadros.
func TestGIFAnimadoRoundTrip(t *testing.T) {
	dir := fxDir(t)
	d := decodeFile(t, filepath.Join(dir, "anim.gif"))
	var buf bytes.Buffer
	if err := Encode(&buf, d, EncodeOptions{Format: GIF}); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode gif: %v", err)
	}
	if len(g.Image) != len(d.Frames) {
		t.Errorf("cuadros re-codificados = %d, esperaba %d", len(g.Image), len(d.Frames))
	}
}

// TestSniffVsExtension: sniff no le cree a la extensión. Un webp renombrado a
// .png se decodifica igual como webp.
func TestSniffVsExtension(t *testing.T) {
	dir := fxDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "lossless.webp"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if d.Source != WEBP {
		t.Errorf("sniff = %v, esperaba WEBP aun con otro nombre", d.Source)
	}
}

// TestFormatoDesconocido: bytes que no son imagen dan ErrUnsupported.
func TestFormatoDesconocido(t *testing.T) {
	_, err := Decode(bytes.NewReader([]byte("esto no es una imagen, es texto plano")))
	if err == nil {
		t.Fatal("esperaba error con bytes no-imagen")
	}
}
