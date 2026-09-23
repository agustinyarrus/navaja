package qr

// matrix.go construye la cuadrícula de módulos y coloca los patrones de función
// (buscadores, separadores, timing, alineación, módulo oscuro) y las zonas
// reservadas para la información de formato y versión.

// Matrix es la cuadrícula final de módulos del QR. true = oscuro.
type Matrix struct {
	Size    int
	Version int
	Level   Level
	Mask    int
	modules []bool
	fixed   []bool // true donde hay patrón de función: no se enmascara ni se escribe
}

func newMatrix(version int) *Matrix {
	size := 17 + 4*version
	return &Matrix{
		Size:    size,
		Version: version,
		modules: make([]bool, size*size),
		fixed:   make([]bool, size*size),
	}
}

// At devuelve el módulo en (x, y); fuera de rango es claro.
func (m *Matrix) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Size || y >= m.Size {
		return false
	}
	return m.modules[y*m.Size+x]
}

func (m *Matrix) set(x, y int, dark bool) { m.modules[y*m.Size+x] = dark }

func (m *Matrix) isFixed(x, y int) bool { return m.fixed[y*m.Size+x] }

func (m *Matrix) setFixed(x, y int, dark bool) {
	m.modules[y*m.Size+x] = dark
	m.fixed[y*m.Size+x] = true
}

// placeFunctionPatterns pone todo lo que no son datos: los tres buscadores con
// sus separadores, los patrones de timing, los de alineación, el módulo oscuro
// y reserva las áreas de formato y versión.
func (m *Matrix) placeFunctionPatterns() {
	m.placeFinder(0, 0)
	m.placeFinder(m.Size-7, 0)
	m.placeFinder(0, m.Size-7)
	m.placeSeparators()
	m.placeTiming()
	m.placeAlignment()
	// Módulo oscuro, siempre presente, junto al buscador inferior izquierdo.
	m.setFixed(8, m.Size-8, true)
	m.reserveFormat()
	if m.Version >= 7 {
		m.reserveVersion()
	}
}

// placeFinder dibuja un patrón buscador 7×7 con su ojo en la esquina (x0, y0).
func (m *Matrix) placeFinder(x0, y0 int) {
	for dy := 0; dy < 7; dy++ {
		for dx := 0; dx < 7; dx++ {
			ring := dx == 0 || dx == 6 || dy == 0 || dy == 6
			core := dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4
			m.setFixed(x0+dx, y0+dy, ring || core)
		}
	}
}

// placeSeparators pone la banda clara de un módulo que rodea cada buscador.
func (m *Matrix) placeSeparators() {
	corners := [][2]int{{0, 0}, {m.Size - 7, 0}, {0, m.Size - 7}}
	for _, c := range corners {
		for i := -1; i <= 7; i++ {
			m.markLightIfInside(c[0]+i, c[1]-1)
			m.markLightIfInside(c[0]+i, c[1]+7)
			m.markLightIfInside(c[0]-1, c[1]+i)
			m.markLightIfInside(c[0]+7, c[1]+i)
		}
	}
}

func (m *Matrix) markLightIfInside(x, y int) {
	if x >= 0 && y >= 0 && x < m.Size && y < m.Size && !m.isFixed(x, y) {
		m.setFixed(x, y, false)
	}
}

// placeTiming dibuja las dos líneas de timing (alternadas) en la fila y la
// columna 6, entre los buscadores.
func (m *Matrix) placeTiming() {
	for i := 8; i < m.Size-8; i++ {
		dark := i%2 == 0
		if !m.isFixed(i, 6) {
			m.setFixed(i, 6, dark)
		}
		if !m.isFixed(6, i) {
			m.setFixed(6, i, dark)
		}
	}
}

// placeAlignment dibuja los patrones de alineación 5×5 en las intersecciones de
// las coordenadas de la versión, salvo donde chocan con un buscador.
func (m *Matrix) placeAlignment() {
	pos := alignmentPositions[m.Version-1]
	for _, cy := range pos {
		for _, cx := range pos {
			if m.overlapsFinder(cx, cy) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					ring := dx == -2 || dx == 2 || dy == -2 || dy == 2
					center := dx == 0 && dy == 0
					m.setFixed(cx+dx, cy+dy, ring || center)
				}
			}
		}
	}
}

// overlapsFinder dice si un patrón de alineación centrado en (cx, cy) pisaría
// alguno de los tres buscadores.
func (m *Matrix) overlapsFinder(cx, cy int) bool {
	near := func(x, y, ox, oy int) bool { return abs(x-ox) <= 4 && abs(y-oy) <= 4 }
	return near(cx, cy, 3, 3) || near(cx, cy, m.Size-4, 3) || near(cx, cy, 3, m.Size-4)
}

// reserveFormat marca como fijas (sin escribir aún) las 2×15 celdas de la
// información de formato, alrededor de los buscadores superiores.
func (m *Matrix) reserveFormat() {
	for i := 0; i <= 8; i++ {
		m.reserve(i, 8)
		m.reserve(8, i)
	}
	for i := 0; i < 8; i++ {
		m.reserve(8, m.Size-1-i)
		m.reserve(m.Size-1-i, 8)
	}
}

func (m *Matrix) reserve(x, y int) {
	if !m.isFixed(x, y) {
		m.fixed[y*m.Size+x] = true
	}
}

// reserveVersion marca las dos zonas 3×6 de información de versión (v≥7).
func (m *Matrix) reserveVersion() {
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			m.reserve(i, m.Size-11+j)
			m.reserve(m.Size-11+j, i)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
