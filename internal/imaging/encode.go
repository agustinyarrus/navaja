package imaging

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// EncodeOptions controla la codificación de salida.
type EncodeOptions struct {
	Format      Format
	JPEGQuality int         // 1..100 (solo JPEG)
	PNGFast     bool        // compresión rápida en vez de la mejor
	Background  color.Color // fondo al aplanar alfa hacia un formato opaco (JPEG/BMP)
}

// DefaultBackground es el fondo al aplanar transparencia: blanco, lo que
// espera la mayoría (un logo con alfa sobre blanco, no sobre negro).
var DefaultBackground = color.White

// Encode escribe la imagen decodificada en el formato pedido. Para un destino
// que no soporta animación se usa el primer cuadro; el GIF de salida conserva
// todos los cuadros si el origen era animado.
func Encode(w io.Writer, d *Decoded, opt EncodeOptions) error {
	if opt.Format == GIF {
		return encodeGIF(w, d)
	}
	return EncodeFrame(w, d.Frames[0], opt)
}

// EncodeFrame escribe un único cuadro (lo usa también el modo --frames).
func EncodeFrame(w io.Writer, img image.Image, opt EncodeOptions) error {
	switch opt.Format {
	case PNG:
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if opt.PNGFast {
			enc.CompressionLevel = png.BestSpeed
		}
		return enc.Encode(w, img)
	case JPEG:
		q := opt.JPEGQuality
		if q <= 0 {
			q = 90
		}
		return jpeg.Encode(w, flatten(img, opt.Background), &jpeg.Options{Quality: q})
	case BMP:
		return bmp.Encode(w, flatten(img, opt.Background))
	case TIFF:
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate})
	case GIF:
		return gif.Encode(w, img, &gif.Options{NumColors: 256})
	}
	return ErrUnsupported
}

// encodeGIF escribe un GIF; si el origen era animado, conserva cuadros, tiempos
// y bucle. Los cuadros compuestos se recuantizan a paleta con el codificador
// estándar (que aplica dithering de Floyd–Steinberg).
func encodeGIF(w io.Writer, d *Decoded) error {
	if !d.Animated() {
		return gif.Encode(w, d.Frames[0], &gif.Options{NumColors: 256})
	}
	out := &gif.GIF{LoopCount: d.LoopCount}
	for i, frame := range d.Frames {
		pal := image.NewPaletted(frame.Bounds(), paletteFor(frame))
		draw.FloydSteinberg.Draw(pal, frame.Bounds(), frame, frame.Bounds().Min)
		out.Image = append(out.Image, pal)
		delay := 10 // 100 ms por defecto si el origen no traía tiempos
		if i < len(d.Delays) && d.Delays[i] > 0 {
			delay = d.Delays[i]
		}
		out.Delay = append(out.Delay, delay)
		disposal := byte(gif.DisposalNone)
		if i < len(d.Disposal) {
			disposal = d.Disposal[i]
		}
		out.Disposal = append(out.Disposal, disposal)
	}
	return gif.EncodeAll(w, out)
}

// paletteFor arma una paleta de 256 para un cuadro. Usa la paleta del Plan 9
// (la que trae la stdlib), buena de propósito general; el dithering lo pone el
// que llama con draw.FloydSteinberg.
func paletteFor(img image.Image) color.Palette {
	pal := make(color.Palette, len(palette.Plan9))
	copy(pal, palette.Plan9)
	return pal
}

// flatten devuelve una imagen opaca: si img tiene alfa, la compone sobre bg;
// si ya es opaca, la devuelve tal cual (sin copiar).
func flatten(img image.Image, bg color.Color) image.Image {
	if !hasAlpha(img) {
		return img
	}
	if bg == nil {
		bg = DefaultBackground
	}
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Over)
	return out
}
