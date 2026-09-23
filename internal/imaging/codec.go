// Package imaging decodifica y codifica imágenes entre los formatos que la web
// escupe todo el tiempo (webp, gif, png, jpeg, bmp, tiff) sin dependencias
// nativas: solo la stdlib y golang.org/x/image, todo Go puro.
package imaging

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"sort"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

// Format es un formato de imagen soportado.
type Format uint8

const (
	Unknown Format = iota
	PNG
	JPEG
	GIF
	WEBP
	BMP
	TIFF
)

var formatName = map[Format]string{
	PNG: "png", JPEG: "jpeg", GIF: "gif", WEBP: "webp", BMP: "bmp", TIFF: "tiff",
}

var formatExt = map[Format]string{
	PNG: ".png", JPEG: ".jpg", GIF: ".gif", WEBP: ".webp", BMP: ".bmp", TIFF: ".tiff",
}

// byName mapea nombres y alias de la línea de comandos a un formato.
var byName = map[string]Format{
	"png": PNG, "jpg": JPEG, "jpeg": JPEG, "gif": GIF,
	"webp": WEBP, "bmp": BMP, "tif": TIFF, "tiff": TIFF,
}

func (f Format) String() string { return formatName[f] }

// Ext devuelve la extensión canónica del formato (con el punto).
func (f Format) Ext() string { return formatExt[f] }

// CanEncode dice si sabemos ESCRIBIR ese formato. Se puede decodificar webp
// pero no codificarlo: no hay encoder de webp en Go puro.
func (f Format) CanEncode() bool { return f != WEBP && f != Unknown }

// ParseFormat interpreta el nombre de un formato de salida.
func ParseFormat(s string) (Format, error) {
	f, ok := byName[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(s), "."))]
	if !ok {
		return Unknown, fmt.Errorf("formato %q desconocido (png, jpg, gif, bmp, tiff)", s)
	}
	if !f.CanEncode() {
		return Unknown, fmt.Errorf("no puedo escribir %s (webp no tiene codificador en Go puro)", s)
	}
	return f, nil
}

// EncodableFormats lista los formatos de salida, ordenados, para la ayuda.
func EncodableFormats() []string {
	var out []string
	for f, n := range formatName {
		if f.CanEncode() {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// magic reconoce el formato por los primeros bytes: es lo confiable, la
// extensión miente (un .png que en realidad es un webp es clásico de la web).
func sniff(b []byte) Format {
	switch {
	case len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n":
		return PNG
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return JPEG
	case len(b) >= 6 && (string(b[:6]) == "GIF87a" || string(b[:6]) == "GIF89a"):
		return GIF
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return WEBP
	case len(b) >= 2 && b[0] == 'B' && b[1] == 'M':
		return BMP
	case len(b) >= 4 && (string(b[:4]) == "II*\x00" || string(b[:4]) == "MM\x00*"):
		return TIFF
	}
	return Unknown
}

// Decoded es una imagen ya decodificada, con sus cuadros compuestos (uno solo
// salvo en un GIF animado) y los metadatos que hacen falta para recodificar.
type Decoded struct {
	Source    Format
	Frames    []image.Image // siempre ≥1; ya compuestos a pantalla completa
	Delays    []int         // centésimas de segundo por cuadro (solo GIF)
	Disposal  []byte
	LoopCount int
	Width     int
	Height    int
	HasAlpha  bool
}

// Animated dice si la imagen tiene más de un cuadro.
func (d *Decoded) Animated() bool { return len(d.Frames) > 1 }

// ErrUnsupported se devuelve cuando los bytes no son de un formato conocido.
var ErrUnsupported = errors.New("no reconozco el formato de la imagen")

// Decode lee una imagen, componiendo los cuadros de un GIF animado.
func Decode(r io.Reader) (*Decoded, error) {
	br := bufio.NewReaderSize(r, 64<<10)
	head, err := br.Peek(16)
	if err != nil && err != io.EOF {
		return nil, err
	}
	format := sniff(head)
	if format == Unknown {
		return nil, ErrUnsupported
	}
	if format == GIF {
		return decodeGIF(br)
	}
	img, decodeErr := decodeStatic(format, br)
	if decodeErr != nil {
		return nil, fmt.Errorf("no pude decodificar el %s: %w", format, decodeErr)
	}
	alpha := hasAlpha(img) // antes de convertir: un YCbCr sin alfa sigue sin alfa
	if format == WEBP {
		img = convertWebP(img)
	}
	b := img.Bounds()
	return &Decoded{
		Source:   format,
		Frames:   []image.Image{img},
		Width:    b.Dx(),
		Height:   b.Dy(),
		HasAlpha: alpha,
	}, nil
}

func decodeStatic(format Format, r io.Reader) (image.Image, error) {
	switch format {
	case PNG:
		return png.Decode(r)
	case JPEG:
		return jpeg.Decode(r)
	case WEBP:
		return webp.Decode(r)
	case BMP:
		return bmp.Decode(r)
	case TIFF:
		return tiff.Decode(r)
	}
	return nil, ErrUnsupported
}

// hasAlpha dice si el modelo de color de la imagen puede llevar transparencia.
// Es una respuesta por TIPO, no por contenido: alcanza para decidir si al pasar
// a JPEG hay que aplanar sobre un fondo.
func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64, *image.NYCbCrA, *image.Paletted, *image.Alpha, *image.Alpha16:
		return true
	}
	return false
}
