package qr

// mask.go aplica las 8 máscaras del estándar sobre los módulos de datos y elige
// la mejor por la regla de penalización, que busca el patrón más difícil de
// confundir para un lector.

// maskCondition evalúa la fórmula de la máscara k en (x, y). Se invierte el
// módulo donde da true.
func maskCondition(k, x, y int) bool {
	switch k {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return (x*y)%2+(x*y)%3 == 0
	case 6:
		return ((x*y)%2+(x*y)%3)%2 == 0
	case 7:
		return ((x+y)%2+(x*y)%3)%2 == 0
	}
	return false
}

// applyMask invierte los módulos de datos donde la máscara k lo indica. No toca
// los módulos de función.
func (m *Matrix) applyMask(k int) {
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; x++ {
			if !m.isFixed(x, y) && maskCondition(k, x, y) {
				m.set(x, y, !m.At(x, y))
			}
		}
	}
}

// writeFormat escribe la información de formato (nivel + máscara) en sus dos
// copias. Se llama después de fijar la máscara.
func (m *Matrix) writeFormat() {
	bits := formatInfo(m.Level, m.Mask)
	// Copia 1: alrededor del buscador superior izquierdo.
	for i := 0; i <= 5; i++ {
		m.setFixed(8, i, bit(bits, i))
	}
	m.setFixed(8, 7, bit(bits, 6))
	m.setFixed(8, 8, bit(bits, 7))
	m.setFixed(7, 8, bit(bits, 8))
	for i := 9; i < 15; i++ {
		m.setFixed(14-i, 8, bit(bits, i))
	}
	// Copia 2: repartida junto a los otros dos buscadores.
	for i := 0; i < 8; i++ {
		m.setFixed(m.Size-1-i, 8, bit(bits, i))
	}
	for i := 8; i < 15; i++ {
		m.setFixed(8, m.Size-15+i, bit(bits, i))
	}
}

// writeVersion escribe la información de versión (solo v≥7) en sus dos zonas.
func (m *Matrix) writeVersion() {
	if m.Version < 7 {
		return
	}
	bits := versionInfo(m.Version)
	for i := 0; i < 18; i++ {
		b := bit(bits, i)
		r, c := i/3, i%3
		m.setFixed(r, m.Size-11+c, b)
		m.setFixed(m.Size-11+c, r, b)
	}
}

func bit(v uint32, i int) bool { return v&(1<<uint(i)) != 0 }

// penalty calcula la penalización total de la matriz según las cuatro reglas
// del estándar; menor es mejor.
func (m *Matrix) penalty() int {
	return m.penaltyRuns() + m.penaltyBlocks() + m.penaltyFinderLike() + m.penaltyBalance()
}

// Regla 1: rachas de 5+ módulos del mismo color en fila o columna.
func (m *Matrix) penaltyRuns() int {
	score := 0
	count := func(get func(i, j int) bool) {
		for i := 0; i < m.Size; i++ {
			run, prev := 1, get(i, 0)
			for j := 1; j < m.Size; j++ {
				cur := get(i, j)
				if cur == prev {
					run++
					continue
				}
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run, prev = 1, cur
			}
			if run >= 5 {
				score += 3 + (run - 5)
			}
		}
	}
	count(func(i, j int) bool { return m.At(j, i) }) // filas
	count(func(i, j int) bool { return m.At(i, j) }) // columnas
	return score
}

// Regla 2: bloques 2×2 del mismo color, 3 puntos cada uno.
func (m *Matrix) penaltyBlocks() int {
	score := 0
	for y := 0; y < m.Size-1; y++ {
		for x := 0; x < m.Size-1; x++ {
			c := m.At(x, y)
			if c == m.At(x+1, y) && c == m.At(x, y+1) && c == m.At(x+1, y+1) {
				score += 3
			}
		}
	}
	return score
}

// Regla 3: el patrón 1:1:3:1:1 rodeado de claro (parecido a un buscador), 40
// puntos cada aparición, en filas y columnas.
func (m *Matrix) penaltyFinderLike() int {
	pat1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	pat2 := []bool{false, false, false, false, true, false, true, true, true, false, true}
	score := 0
	matches := func(get func(i, j int) bool) {
		for i := 0; i < m.Size; i++ {
			for j := 0; j <= m.Size-11; j++ {
				if lineEquals(get, i, j, pat1) || lineEquals(get, i, j, pat2) {
					score += 40
				}
			}
		}
	}
	matches(func(i, j int) bool { return m.At(j, i) })
	matches(func(i, j int) bool { return m.At(i, j) })
	return score
}

func lineEquals(get func(i, j int) bool, i, j int, pat []bool) bool {
	for k, want := range pat {
		if get(i, j+k) != want {
			return false
		}
	}
	return true
}

// Regla 4: desequilibrio entre módulos oscuros y claros respecto del 50 %.
func (m *Matrix) penaltyBalance() int {
	dark := 0
	for _, v := range m.modules {
		if v {
			dark++
		}
	}
	total := m.Size * m.Size
	percent := dark * 100 / total
	// Distancia al múltiplo de 5 más cercano por debajo y por arriba, /5, ×10.
	lower := percent / 5 * 5
	upper := lower + 5
	d := min(abs(lower-50), abs(upper-50)) / 5
	return d * 10
}
