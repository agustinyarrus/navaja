// Package img implementa `img`: convierte imágenes entre formatos en lote
// (webp/gif/jpg/bmp/tiff → png y demás), en paralelo, con barra de progreso y
// tarjeta final. Nace de "webp2png / gif2png" pero sirve para cualquier par.
package img

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agustinyarrus/navaja/internal/batch"
	"github.com/agustinyarrus/navaja/internal/cli"
	"github.com/agustinyarrus/navaja/internal/fsx"
	"github.com/agustinyarrus/navaja/internal/imaging"
	"github.com/agustinyarrus/navaja/internal/tui"
)

type opciones struct {
	destino   string
	calidad   int
	carpeta   string
	fondo     string
	frames    bool
	recursivo bool
	forzar    bool
	rapido    bool
	trabajos  int
}

// Main es el punto de entrada del subcomando.
func Main(t *tui.Term, version string, args []string) int {
	o := opciones{destino: "png", calidad: 90, fondo: "blanco"}
	app := cli.New("img", version, "convierte imágenes entre formatos, en lote")
	app.Usage = []string{
		"img <archivos|carpetas|patrones…> [opciones]",
		"img *.webp                 cada .webp a .png, al lado del original",
		"img foto.gif --to jpg      un gif a jpg (primer cuadro)",
	}
	app.Section("qué producir")
	app.String(&o.destino, "to", 0, "png|jpg|gif|bmp|tiff", "formato de salida")
	app.Int(&o.calidad, "quality", 'q', "N", "calidad JPEG 1–100", 1, 100)
	app.Bool(&o.frames, "frames", 0, "de un animado, exportar cada cuadro a un archivo")
	app.String(&o.fondo, "background", 'b', "COLOR", "fondo al aplanar transparencia hacia JPEG/BMP")
	app.Bool(&o.rapido, "fast", 0, "PNG con compresión rápida en vez de la máxima")
	app.Section("de dónde y a dónde")
	app.String(&o.carpeta, "out-dir", 'o', "DIR", "carpeta de salida (por defecto, junto al original)")
	app.Bool(&o.recursivo, "recursive", 'r', "entrar en subcarpetas al pasar una carpeta")
	app.Bool(&o.forzar, "force", 'f', "sobrescribir si el destino ya existe")
	app.Int(&o.trabajos, "jobs", 'j', "N", "cuántas conversiones en paralelo (0 = una por CPU)", 0, 256)
	app.Examples = []cli.Example{
		{Cmd: "img *.webp", Desc: "el caso típico: toda la carpeta de webp a png"},
		{Cmd: "img fotos/ -r --to jpg -q 82", Desc: "recorrer una carpeta y pasar todo a jpg"},
		{Cmd: "img loader.gif --frames", Desc: "romper un gif animado en loader-001.png, …"},
		{Cmd: "img logo.webp --to jpg -b '#0b0b0f'", Desc: "aplanar el alfa sobre un fondo oscuro"},
	}
	app.Notes = []string{
		"El formato se detecta por el contenido, no por la extensión (un .png que en realidad es webp se convierte igual).",
		"Nunca se borra el original ni se sobrescribe sin --force.",
		"webp no se puede ESCRIBIR (no hay codificador en Go puro); sí leer.",
	}

	pos, done, code := app.Start(t, args)
	if done {
		return code
	}
	if len(pos) == 0 {
		t.Lines(app.Help(t))
		return cli.ExitUsage
	}

	destino, err := imaging.ParseFormat(o.destino)
	if err != nil {
		return fallo(t, err)
	}
	fondo, err := imaging.ParseHexColor(o.fondo)
	if err != nil {
		return fallo(t, err)
	}
	if o.carpeta != "" {
		if err := os.MkdirAll(o.carpeta, 0o755); err != nil {
			return fallo(t, fmt.Errorf("no puedo usar la carpeta de salida: %w", err))
		}
	}

	archivos, problemas := fsx.Expand(pos, fsx.ExpandOptions{
		Recursive: o.recursivo,
		Accept:    func(p string) bool { return imaging.IsImageName(p) },
	})

	t.Lines(t.Header("img", "convierte imágenes entre formatos, en lote", version))
	for _, pr := range problemas {
		t.Line(t.Status(tui.Warn, fsx.Display(pr.Pattern), pr.Err.Error()))
	}
	if len(archivos) == 0 {
		t.Blank()
		t.Line(t.Status(tui.Fail, "No hay imágenes para convertir", "revisá los patrones o la extensión"))
		return cli.ExitFailure
	}

	enc := imaging.EncodeOptions{
		Format:      destino,
		JPEGQuality: o.calidad,
		PNGFast:     o.rapido,
		Background:  fondo.(color.Color),
	}

	tareas, planes := construirTareas(archivos, destino, enc, o)
	t.Lines(resumenPlan(t, planes, destino, o))

	ctx, stop := cli.Interrupt(nil)
	defer stop()

	inicio := time.Now()
	resultados := batch.Run(ctx, t, tareas, batch.Options{
		Title:   "Convirtiendo a " + destino.String(),
		Workers: o.trabajos,
	})
	return informe(t, resultados, destino, time.Since(inicio))
}

// plan es la conversión resuelta de un archivo: su destino calculado y si se
// puede hacer (o el motivo por el que se saltea).
type plan struct {
	entrada string
	salida  string
	saltar  string // motivo del salteo; vacío si procede
}

// construirTareas resuelve el destino de cada archivo y arma las tareas.
// Detecta colisiones (dos entradas que caen en la misma salida) para no pisar
// resultados en silencio.
func construirTareas(archivos []string, destino imaging.Format, enc imaging.EncodeOptions, o opciones) ([]batch.Task, []plan) {
	tomadas := map[string]string{} // salida (normalizada) → entrada que la reclamó
	planes := make([]plan, 0, len(archivos))
	for _, in := range archivos {
		out := destinoDe(in, destino, o.carpeta)
		p := plan{entrada: in, salida: out}
		switch {
		case fsx.SameFile(in, out):
			p.saltar = "el destino es el mismo archivo"
		case !o.forzar && fsx.Exists(out) && !o.frames:
			p.saltar = "ya existe (usá --force)"
		default:
			if prev, choque := tomadas[strings.ToLower(out)]; choque && !o.frames {
				p.saltar = "chocaría con " + fsx.Display(prev)
			} else {
				tomadas[strings.ToLower(out)] = in
			}
		}
		planes = append(planes, p)
	}

	tareas := make([]batch.Task, 0, len(planes))
	for _, p := range planes {
		p := p
		tareas = append(tareas, batch.Task{
			Label: fsx.Display(p.entrada),
			Run: func(ctx context.Context) (batch.Status, string, any, error) {
				if p.saltar != "" {
					return batch.Skip, p.saltar, nil, nil
				}
				res, err := imaging.Convert(imaging.Job{
					Input:  p.entrada,
					Output: p.salida,
					Enc:    enc,
					Frames: o.frames,
				})
				if err != nil {
					return batch.Fail, err.Error(), nil, err
				}
				return batch.OK, detalleOK(res), res, nil
			},
		})
	}
	return tareas, planes
}

// destinoDe calcula la ruta de salida: cambia la extensión y, si hay --out-dir,
// muda la carpeta conservando el nombre.
func destinoDe(in string, destino imaging.Format, outDir string) string {
	base := fsx.ReplaceExt(filepath.Base(in), destino.Ext())
	if outDir != "" {
		return filepath.Join(outDir, base)
	}
	return filepath.Join(filepath.Dir(in), base)
}

func detalleOK(r imaging.Result) string {
	dims := fmt.Sprintf("%d×%d", r.Width, r.Height)
	if r.FrameCount > 1 {
		return fmt.Sprintf("%s · %s → %d cuadros · %s",
			r.Source, dims, r.FrameCount, tui.Bytes(r.OutBytes))
	}
	delta := ""
	if r.InBytes > 0 {
		delta = "  (" + porcentajeTamano(r.InBytes, r.OutBytes) + ")"
	}
	return fmt.Sprintf("%s → %s · %s · %s%s",
		r.Source, r.Job.Enc.Format, dims, tui.Bytes(r.OutBytes), delta)
}

func porcentajeTamano(in, out int64) string {
	if in == 0 {
		return "—"
	}
	d := float64(out-in) / float64(in)
	signo := ""
	if d > 0 {
		signo = "+"
	}
	return signo + tui.Percent(d, 0)
}

func fallo(t *tui.Term, err error) int {
	t.Blank()
	t.Line(t.Paint(tui.Rose, "✗ ") + t.Paint(tui.Text, err.Error()))
	t.Blank()
	return cli.ExitFailure
}
