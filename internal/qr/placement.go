package qr

// placement.go entrelaza los bloques de datos y corrección según el estándar y
// coloca el flujo resultante en la matriz siguiendo el recorrido en zigzag.

// interleave arma la secuencia final de codewords: primero los de datos,
// entrelazados columna por columna entre los bloques; después los de
// corrección, también entrelazados. Es lo que reparte una ráfaga de daño entre
// varios bloques y permite recuperarla.
func interleave(data []byte, version int, level Level) []byte {
	lay := layoutFor(version, level)
	blocks := make([][]byte, 0, lay.numG1+lay.numG2)
	ecBlocks := make([][]byte, 0, lay.numG1+lay.numG2)

	pos := 0
	take := func(n int) []byte {
		b := data[pos : pos+n]
		pos += n
		return b
	}
	for i := 0; i < lay.numG1; i++ {
		blk := take(lay.dcG1)
		blocks = append(blocks, blk)
		ecBlocks = append(ecBlocks, rsEncode(blk, lay.ecPerBlock))
	}
	for i := 0; i < lay.numG2; i++ {
		blk := take(lay.dcG2)
		blocks = append(blocks, blk)
		ecBlocks = append(ecBlocks, rsEncode(blk, lay.ecPerBlock))
	}

	var out []byte
	maxData := lay.dcG2 // el más grande de los dos grupos
	if lay.numG2 == 0 {
		maxData = lay.dcG1
	}
	for c := 0; c < maxData; c++ {
		for _, blk := range blocks {
			if c < len(blk) {
				out = append(out, blk[c])
			}
		}
	}
	for c := 0; c < lay.ecPerBlock; c++ {
		for _, ec := range ecBlocks {
			out = append(out, ec[c])
		}
	}
	return out
}

// placeData recorre la matriz en zigzag desde abajo a la derecha, en franjas de
// dos columnas hacia arriba y hacia abajo alternadamente, y escribe los bits de
// los codewords en cada módulo libre. Salta la columna 6 (timing vertical).
func (m *Matrix) placeData(codewords []byte) {
	bitAt := func(i int) bool {
		return codewords[i/8]&(byte(1)<<(7-uint(i%8))) != 0
	}
	bit := 0
	total := len(codewords) * 8
	upward := true
	for col := m.Size - 1; col > 0; col -= 2 {
		if col == 6 { // la columna de timing corre toda la altura
			col--
		}
		for i := 0; i < m.Size; i++ {
			y := i
			if upward {
				y = m.Size - 1 - i
			}
			for dx := 0; dx < 2; dx++ {
				x := col - dx
				if m.isFixed(x, y) {
					continue
				}
				dark := false
				if bit < total {
					dark = bitAt(bit)
				}
				m.set(x, y, dark) // los bits sobrantes quedan en 0 (módulos de resto)
				bit++
			}
		}
		upward = !upward
	}
}
