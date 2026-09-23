package qr

import (
	"errors"
	"fmt"
)

// MaxVersion es la versión más grande del estándar.
const MaxVersion = 40

// ErrTooLong se devuelve cuando el dato no entra ni en la versión 40.
var ErrTooLong = errors.New("el texto es demasiado largo para un código QR")

// Options controla la codificación.
type Options struct {
	Level      Level // nivel de corrección; por defecto M
	MinVersion int   // versión mínima a usar (1 si es 0)
	Mask       int   // máscara forzada 0..7; -1 (o <0) elige la mejor
}

// DefaultOptions es la configuración recomendada: nivel M, versión automática,
// máscara elegida por penalización.
func DefaultOptions() Options { return Options{Level: M, MinVersion: 1, Mask: -1} }

// Encode convierte data en una matriz de código QR lista para dibujar. Elige el
// modo más compacto, la versión más chica que entra y la mejor máscara.
//
// Complejidad: la elige la evaluación de máscaras, O(8 · n²) con n el lado de la
// matriz; para el resto domina Reed–Solomon, O(datos · ecPorBloque).
func Encode(data []byte, opt Options) (*Matrix, error) {
	if opt.MinVersion < 1 {
		opt.MinVersion = 1
	}
	m := detectMode(data)
	version, err := chooseVersion(m, len(data), opt.Level, opt.MinVersion)
	if err != nil {
		return nil, err
	}

	codewords := buildCodewords(data, m, version, opt.Level)
	final := interleave(codewords, version, opt.Level)

	if opt.Mask >= 0 && opt.Mask <= 7 {
		return build(version, opt.Level, opt.Mask, final), nil
	}
	return bestMask(version, opt.Level, final), nil
}

// chooseVersion busca la versión más chica (desde MinVersion) cuyo espacio de
// datos alcanza para el contenido más los indicadores. Búsqueda lineal sobre 40
// valores: no vale la pena algo más fino.
func chooseVersion(m mode, n int, level Level, minVersion int) (int, error) {
	for v := minVersion; v <= MaxVersion; v++ {
		need := 4 + charCountBits(m, v) + segmentBits(m, n)
		if need <= dataCapacityBits(v, level) {
			return v, nil
		}
	}
	return 0, fmt.Errorf("%w (%d caracteres, nivel %s)", ErrTooLong, n, level)
}

// build arma una matriz completa con una máscara concreta.
func build(version int, level Level, mask int, codewords []byte) *Matrix {
	m := newMatrix(version)
	m.Level = level
	m.Mask = mask
	m.placeFunctionPatterns()
	m.placeData(codewords)
	m.applyMask(mask)
	m.writeFormat()
	m.writeVersion()
	return m
}

// bestMask prueba las 8 máscaras y se queda con la de menor penalización. Se
// arma la matriz base una vez y se clona por máscara para no repetir el trabajo
// de colocar patrones y datos.
func bestMask(version int, level Level, codewords []byte) *Matrix {
	base := newMatrix(version)
	base.Level = level
	base.placeFunctionPatterns()
	base.placeData(codewords)

	var best *Matrix
	bestScore := 1 << 30
	for k := 0; k < 8; k++ {
		cand := base.clone()
		cand.Mask = k
		cand.applyMask(k)
		cand.writeFormat()
		cand.writeVersion()
		if s := cand.penalty(); s < bestScore {
			bestScore, best = s, cand
		}
	}
	return best
}

func (m *Matrix) clone() *Matrix {
	c := &Matrix{Size: m.Size, Version: m.Version, Level: m.Level}
	c.modules = append([]bool(nil), m.modules...)
	c.fixed = append([]bool(nil), m.fixed...)
	return c
}
