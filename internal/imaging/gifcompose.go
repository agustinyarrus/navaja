package imaging

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
)

// decodeGIF decodifica un GIF y COMPONE cada cuadro a pantalla completa
// respetando el método de disposición (disposal). Los cuadros de un GIF suelen
// guardar solo el rectángulo que cambió; la herramienta ingenua exporta esos
// recortes parciales. Acá se reconstruye el cuadro completo tal como se ve.
//
// El algoritmo es el estándar de composición GIF, O(cuadros × píxeles):
//   - antes de dibujar el cuadro i, se aplica la disposición del cuadro i-1:
//     0/1 (ninguna / no disponer): el lienzo queda como está;
//     2 (restaurar al fondo): se borra a transparente el rect del cuadro i-1;
//     3 (restaurar al previo): se vuelve al lienzo anterior a i-1.
//   - se dibuja el rect del cuadro i sobre el lienzo;
//   - se toma una copia del lienzo compuesto como cuadro i completo.
func decodeGIF(r io.Reader) (*Decoded, error) {
	g, err := gif.DecodeAll(r)
	if err != nil {
		return nil, err
	}
	if len(g.Image) == 0 {
		return nil, ErrUnsupported
	}

	// El lienzo lógico: si el GIF declara ancho/alto, se usan; si no, la unión
	// de los rectángulos de todos los cuadros.
	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	if bounds.Empty() {
		for _, f := range g.Image {
			bounds = bounds.Union(f.Bounds())
		}
	}

	canvas := image.NewRGBA(bounds)
	var previous *image.RGBA // instantánea para el disposal "restaurar al previo"
	frames := make([]image.Image, 0, len(g.Image))

	for i, frame := range g.Image {
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}

		if disposal == gif.DisposalPrevious {
			previous = cloneRGBA(canvas) // guardar ANTES de pintar este cuadro
		}

		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		// Copia del lienzo compuesto = cuadro i tal como se ve en pantalla.
		frames = append(frames, cloneRGBA(canvas))

		switch disposal {
		case gif.DisposalBackground:
			clearRect(canvas, frame.Bounds())
		case gif.DisposalPrevious:
			if previous != nil {
				draw.Draw(canvas, bounds, previous, bounds.Min, draw.Src)
			}
		}
	}

	return &Decoded{
		Source:    GIF,
		Frames:    frames,
		Delays:    append([]int(nil), g.Delay...),
		Disposal:  append([]byte(nil), g.Disposal...),
		LoopCount: g.LoopCount,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		HasAlpha:  gifHasAlpha(g),
	}, nil
}

// cloneRGBA copia un *image.RGBA (píxeles incluidos).
func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// clearRect pone transparente un rectángulo del lienzo.
func clearRect(canvas *image.RGBA, r image.Rectangle) {
	draw.Draw(canvas, r, image.NewUniform(color.RGBA{}), image.Point{}, draw.Src)
}

// gifHasAlpha dice si alguna paleta del GIF tiene un índice transparente.
func gifHasAlpha(g *gif.GIF) bool {
	for _, f := range g.Image {
		for _, c := range f.Palette {
			if _, _, _, a := c.RGBA(); a == 0 {
				return true
			}
		}
	}
	return false
}
