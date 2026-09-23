package qr

// bitstream.go arma el flujo de bits de datos: elige el modo más compacto para
// el texto, escribe el indicador de modo y de longitud, codifica el contenido y
// completa con relleno hasta llenar la capacidad de la versión.

// mode es el modo de codificación de datos.
type mode uint8

const (
	modeNumeric mode = iota
	modeAlphanumeric
	modeByte
)

// modeIndicator son los 4 bits que anuncian el modo.
var modeIndicator = map[mode]uint32{modeNumeric: 0b0001, modeAlphanumeric: 0b0010, modeByte: 0b0100}

// alnumChars es el juego de 45 caracteres del modo alfanumérico, en el orden
// que define sus valores (0..44).
const alnumChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

var alnumValue = func() map[byte]int {
	m := make(map[byte]int, len(alnumChars))
	for i := 0; i < len(alnumChars); i++ {
		m[alnumChars[i]] = i
	}
	return m
}()

// detectMode elige el modo más compacto que sirve para TODO el texto: numérico
// si son solo dígitos, alfanumérico si entra en el juego de 45, byte si no.
func detectMode(data []byte) mode {
	numeric, alnum := true, true
	for _, b := range data {
		if b < '0' || b > '9' {
			numeric = false
		}
		if _, ok := alnumValue[b]; !ok {
			alnum = false
		}
	}
	switch {
	case numeric:
		return modeNumeric
	case alnum:
		return modeAlphanumeric
	default:
		return modeByte
	}
}

// charCountBits devuelve cuántos bits ocupa el indicador de cantidad de
// caracteres, que depende del modo y del rango de versión.
func charCountBits(m mode, version int) int {
	switch {
	case version <= 9:
		return [3]int{10, 9, 8}[m]
	case version <= 26:
		return [3]int{12, 11, 16}[m]
	default:
		return [3]int{14, 13, 16}[m]
	}
}

// bitBuffer acumula bits de más a menos significativo.
type bitBuffer struct {
	bits []byte // un bit por byte para simpleza; se empaqueta al final
}

func (b *bitBuffer) append(value uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		b.bits = append(b.bits, byte((value>>i)&1))
	}
}

func (b *bitBuffer) len() int { return len(b.bits) }

// segmentBits calcula cuántos bits ocupa el contenido (sin contar indicadores)
// en un modo dado, para poder elegir la versión antes de codificar.
func segmentBits(m mode, n int) int {
	switch m {
	case modeNumeric:
		// grupos de 3 dígitos → 10 bits; resto de 2 → 7; de 1 → 4.
		return (n/3)*10 + map[int]int{0: 0, 1: 4, 2: 7}[n%3]
	case modeAlphanumeric:
		return (n/2)*11 + (n%2)*6
	default:
		return n * 8
	}
}

// encodeData escribe el segmento de datos (sin relleno) en el buffer.
func encodeData(b *bitBuffer, m mode, data []byte) {
	switch m {
	case modeNumeric:
		for i := 0; i < len(data); i += 3 {
			chunk := data[i:min(i+3, len(data))]
			val := 0
			for _, c := range chunk {
				val = val*10 + int(c-'0')
			}
			b.append(uint32(val), []int{0, 4, 7, 10}[len(chunk)])
		}
	case modeAlphanumeric:
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				b.append(uint32(alnumValue[data[i]]*45+alnumValue[data[i+1]]), 11)
			} else {
				b.append(uint32(alnumValue[data[i]]), 6)
			}
		}
	default:
		for _, c := range data {
			b.append(uint32(c), 8)
		}
	}
}

// buildCodewords produce los codewords de datos finales (con relleno) para una
// versión y nivel ya elegidos.
func buildCodewords(data []byte, m mode, version int, level Level) []byte {
	b := &bitBuffer{}
	b.append(modeIndicator[m], 4)
	b.append(uint32(len(data)), charCountBits(m, version))
	encodeData(b, m, data)

	capacity := dataCapacityBits(version, level)
	// Terminador: hasta 4 ceros, sin pasarse de la capacidad.
	for i := 0; i < 4 && b.len() < capacity; i++ {
		b.bits = append(b.bits, 0)
	}
	// Alinear a byte.
	for b.len()%8 != 0 {
		b.bits = append(b.bits, 0)
	}
	// Bytes de relleno alternados 0xEC / 0x11 hasta llenar.
	pad := []byte{0xEC, 0x11}
	for i := 0; b.len() < capacity; i++ {
		b.append(uint32(pad[i%2]), 8)
	}
	return packBits(b.bits)
}

func packBits(bits []byte) []byte {
	out := make([]byte, len(bits)/8)
	for i := range out {
		var v byte
		for j := 0; j < 8; j++ {
			v = v<<1 | bits[i*8+j]
		}
		out[i] = v
	}
	return out
}
