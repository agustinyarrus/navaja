// Package qr codifica datos en un código QR (ISO/IEC 18004) en Go puro, sin
// dependencias: detección de modo, corrección Reed–Solomon, ensamblado de la
// matriz y elección de la mejor de las 8 máscaras por penalización. La salida
// es una matriz de módulos que la herramienta pinta en la terminal.
package qr

// galois.go implementa la aritmética del campo de Galois GF(256) con el
// polinomio generador 0x11d (x⁸+x⁴+x³+x²+1), el que fija el estándar QR, y sobre
// ella la corrección de errores Reed–Solomon.

// Tablas de exponenciales y logaritmos en GF(256): convierten multiplicación en
// suma de logaritmos, así el producto es O(1) con dos lookups. Se llenan una
// sola vez al cargar el paquete.
var (
	expTable [512]byte // expTable[i] = α^i (duplicada a 512 para no hacer i%255)
	logTable [256]byte // logTable[α^i] = i
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		expTable[i] = byte(x)
		logTable[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 { // desbordó el grado 8: reducir módulo 0x11d
			x ^= 0x11d
		}
	}
	for i := 255; i < 512; i++ {
		expTable[i] = expTable[i-255]
	}
}

// gfMul multiplica dos elementos de GF(256). El cero se maneja aparte porque no
// tiene logaritmo.
func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[int(logTable[a])+int(logTable[b])]
}

// rsGenerator devuelve el polinomio generador de grado `degree` para
// Reed–Solomon: producto de (x - α^i) para i en [0, degree). Los coeficientes
// quedan con el LÍDER primero (g[0] = coeficiente de x^degree = 1), que es el
// orden que espera la división de rsEncode. Se memoiza por grado.
var rsGenCache = map[int][]byte{}

func rsGenerator(degree int) []byte {
	if g, ok := rsGenCache[degree]; ok {
		return g
	}
	// Se construye con el término constante primero y al final se invierte, para
	// dejar el coeficiente líder en g[0].
	g := []byte{1}
	for i := 0; i < degree; i++ {
		// Multiplicar g(x) por (x - α^i) = (x + α^i) en GF(2).
		next := make([]byte, len(g)+1)
		factor := expTable[i]
		for j := 0; j < len(g); j++ {
			next[j] ^= gfMul(g[j], factor)
			next[j+1] ^= g[j]
		}
		g = next
	}
	for l, r := 0, len(g)-1; l < r; l, r = l+1, r-1 {
		g[l], g[r] = g[r], g[l]
	}
	rsGenCache[degree] = g
	return g
}

// rsEncode calcula los `ecLen` bytes de corrección para un bloque de datos.
// Es la división polinómica de data·x^ecLen entre el generador; el resto son
// los códigos de corrección. O(len(data) · ecLen), el costo intrínseco.
func rsEncode(data []byte, ecLen int) []byte {
	gen := rsGenerator(ecLen)
	res := make([]byte, len(data)+ecLen)
	copy(res, data)
	for i := 0; i < len(data); i++ {
		coef := res[i]
		if coef == 0 {
			continue
		}
		lg := int(logTable[coef])
		for j := 0; j < len(gen); j++ {
			res[i+j] ^= expTable[int(logTable[gen[j]])+lg]
		}
	}
	return res[len(data):] // el resto de la división
}
