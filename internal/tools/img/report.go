package img

import (
	"time"

	"github.com/agustinyarrus/navaja/internal/batch"
	"github.com/agustinyarrus/navaja/internal/cli"
	"github.com/agustinyarrus/navaja/internal/imaging"
	"github.com/agustinyarrus/navaja/internal/tui"
)

// resumenPlan muestra, antes de arrancar, cuántas imágenes se van a convertir y
// a qué formato: da contexto sin ruido.
func resumenPlan(t *tui.Term, planes []plan, destino imaging.Format, o opciones) []string {
	convertibles := 0
	for _, p := range planes {
		if p.saltar == "" {
			convertibles++
		}
	}
	chips := []tui.Chip{
		{Text: tui.Count(int64(len(planes)), "imagen encontrada", "imágenes encontradas"), Dot: tui.Teal},
		{Text: tui.Count(int64(convertibles), "por convertir", "por convertir"), Dot: tui.Lavender},
		{Text: "destino ." + destino.String(), Dot: tui.Peach},
	}
	if o.frames {
		chips = append(chips, tui.Chip{Text: "cuadro por archivo", Dot: tui.Sky})
	}
	return []string{"", t.Chips(chips...), ""}
}

// informe arma la tarjeta final con totales, bytes movidos y velocidad.
func informe(t *tui.Term, resultados []batch.Result, destino imaging.Format, elapsed time.Duration) int {
	tl := batch.Summarize(resultados, elapsed)
	var totalIn, totalOut int64
	var totalCuadros int
	for _, r := range resultados {
		if r.Status != batch.OK {
			continue
		}
		if res, ok := r.Data.(imaging.Result); ok {
			totalIn += res.InBytes
			totalOut += res.OutBytes
			totalCuadros += res.FrameCount
		}
	}

	items := []tui.Item{
		{Label: "Convertidas", Value: tui.Int(int64(tl.OK)), Dot: tui.Sage},
	}
	if tl.Skip > 0 {
		items = append(items, tui.Item{Label: "Salteadas", Value: tui.Int(int64(tl.Skip)), Dot: tui.Cream})
	}
	if tl.Fail > 0 {
		items = append(items, tui.Item{Label: "Con error", Value: tui.Int(int64(tl.Fail)), Dot: tui.Rose})
	}
	if totalCuadros > tl.OK {
		items = append(items, tui.Item{Label: "Archivos escritos", Value: tui.Int(int64(totalCuadros)), Dot: tui.Sky})
	}
	items = append(items,
		tui.Item{Label: "Tamaño final", Value: tui.Bytes(totalOut), Dot: tui.Teal},
		tui.Item{Label: "Cambio de peso", Value: porcentajeTamano(totalIn, totalOut), Dot: tui.Lavender},
		tui.Item{Label: "Tiempo", Value: tui.Duration(elapsed), Dot: tui.Peach},
	)
	t.Lines(t.Card("img · "+destino.String(), items))

	switch {
	case tl.Fail > 0 && tl.OK > 0:
		return cli.ExitPartial
	case tl.OK == 0 && tl.Fail > 0:
		return cli.ExitFailure
	default:
		return cli.ExitOK
	}
}
