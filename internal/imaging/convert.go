package imaging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/agustinyarrus/navaja/internal/fsx"
)

// Job es una conversión pedida: de dónde a dónde y con qué opciones.
type Job struct {
	Input  string
	Output string // destino base; en modo Frames se le inserta el número
	Enc    EncodeOptions
	Frames bool // exportar cada cuadro de un animado a un archivo aparte
}

// Result es lo que pasó con un Job.
type Result struct {
	Job        Job
	Source     Format
	Width      int
	Height     int
	FrameCount int   // cuántos archivos se escribieron
	OutBytes   int64 // suma de bytes escritos
	InBytes    int64
	Outputs    []string
}

// Convert ejecuta una conversión: decodifica, y escribe uno o varios archivos
// de forma atómica. No decide nombres ni sobrescrituras: eso ya viene resuelto
// en el Job (frontera validada una sola vez).
func Convert(j Job) (Result, error) {
	in, err := os.Open(j.Input)
	if err != nil {
		return Result{}, err
	}
	defer in.Close()
	if st, err := in.Stat(); err == nil {
		if st.IsDir() {
			return Result{}, fmt.Errorf("es una carpeta")
		}
	}

	dec, err := Decode(in)
	if err != nil {
		return Result{}, err
	}
	res := Result{Job: j, Source: dec.Source, Width: dec.Width, Height: dec.Height}
	if fi, err := os.Stat(j.Input); err == nil {
		res.InBytes = fi.Size()
	}

	if j.Frames && dec.Animated() {
		return writeFrames(j, dec, res)
	}

	err = fsx.WriteAtomic(j.Output, func(w io.Writer) error {
		return Encode(w, dec, j.Enc)
	})
	if err != nil {
		return Result{}, err
	}
	res.FrameCount = 1
	res.Outputs = []string{j.Output}
	res.OutBytes = fileSize(j.Output)
	return res, nil
}

// writeFrames explota un animado: base-0001.png, base-0002.png, … con el ancho
// de dígitos justo para la cantidad de cuadros.
func writeFrames(j Job, dec *Decoded, res Result) (Result, error) {
	ext := filepath.Ext(j.Output)
	base := j.Output[:len(j.Output)-len(ext)]
	width := digits(len(dec.Frames))
	for i, frame := range dec.Frames {
		name := fmt.Sprintf("%s-%0*d%s", base, width, i+1, ext)
		var buf bytes.Buffer
		if err := EncodeFrame(&buf, frame, j.Enc); err != nil {
			return Result{}, fmt.Errorf("cuadro %d: %w", i+1, err)
		}
		if err := fsx.WriteAtomic(name, func(w io.Writer) error {
			_, e := w.Write(buf.Bytes())
			return e
		}); err != nil {
			return Result{}, err
		}
		res.Outputs = append(res.Outputs, name)
		res.OutBytes += int64(buf.Len())
	}
	res.FrameCount = len(dec.Frames)
	return res, nil
}

func digits(n int) int {
	d := 1
	for n >= 10 {
		n /= 10
		d++
	}
	return max(d, 3) // mínimo 3: -001, se ordena bien en el explorador
}

func fileSize(path string) int64 {
	if fi, err := os.Stat(path); err == nil {
		return fi.Size()
	}
	return 0
}

// ImageExtensions son las extensiones que la herramienta considera imágenes al
// recorrer una carpeta (para no intentar convertir un .txt que quedó ahí).
var ImageExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true, ".tif": true, ".tiff": true,
}

// IsImageName dice si el nombre tiene una extensión de imagen conocida.
func IsImageName(name string) bool {
	return ImageExtensions[toLowerExt(name)]
}

func toLowerExt(name string) string {
	ext := filepath.Ext(name)
	buf := make([]byte, len(ext))
	for i := 0; i < len(ext); i++ {
		c := ext[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[i] = c
	}
	return string(buf)
}
