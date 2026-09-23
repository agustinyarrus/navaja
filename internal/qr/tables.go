package qr

// tables.go tiene los datos del estándar QR. En vez de tabular a mano las 160
// combinaciones de bloques (versión × nivel), se guardan solo tres arreglos
// base y el reparto en grupos se DERIVA con la regla del estándar. Menos
// superficie para un error de transcripción.

// Level es el nivel de corrección de errores.
type Level uint8

const (
	L Level = iota // ~7 % recuperable
	M              // ~15 %
	Q              // ~25 %
	H              // ~30 %
)

var levelNames = map[Level]string{L: "L", M: "M", Q: "Q", H: "H"}

func (l Level) String() string { return levelNames[l] }

// ParseLevel interpreta el nivel desde una letra.
func ParseLevel(s string) (Level, bool) {
	switch s {
	case "l", "L":
		return L, true
	case "m", "M":
		return M, true
	case "q", "Q":
		return Q, true
	case "h", "H":
		return H, true
	}
	return 0, false
}

// totalCodewords[v] = cantidad total de codewords de 8 bits de la versión v
// (datos + corrección). Índice por versión-1 (v1..v40).
var totalCodewords = [40]int{
	26, 44, 70, 100, 134, 172, 196, 242, 292, 346,
	404, 466, 532, 581, 655, 733, 815, 901, 991, 1085,
	1156, 1258, 1364, 1474, 1588, 1706, 1828, 1921, 2051, 2185,
	2323, 2465, 2611, 2761, 2876, 3034, 3196, 3362, 3532, 3706,
}

// ecPerBlock[v-1][nivel] = codewords de corrección por bloque.
var ecPerBlock = [40][4]int{
	{7, 10, 13, 17}, {10, 16, 22, 28}, {15, 26, 18, 22}, {20, 18, 26, 16}, {26, 24, 18, 22},
	{18, 16, 24, 28}, {20, 18, 18, 26}, {24, 22, 22, 26}, {30, 22, 20, 24}, {18, 26, 24, 28},
	{20, 30, 28, 24}, {24, 22, 26, 28}, {26, 22, 24, 22}, {30, 24, 20, 24}, {22, 24, 30, 24},
	{24, 28, 24, 30}, {28, 28, 28, 28}, {30, 26, 28, 28}, {28, 26, 26, 26}, {28, 26, 30, 28},
	{28, 26, 28, 30}, {28, 28, 30, 24}, {30, 28, 30, 30}, {30, 28, 30, 30}, {26, 28, 30, 30},
	{28, 28, 28, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30},
	{30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30},
	{30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30}, {30, 28, 30, 30},
}

// numBlocks[v-1][nivel] = cantidad de bloques de Reed–Solomon.
var numBlocks = [40][4]int{
	{1, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 2, 2}, {1, 2, 2, 4}, {1, 2, 4, 4},
	{2, 4, 4, 4}, {2, 4, 6, 5}, {2, 4, 6, 6}, {2, 5, 8, 8}, {4, 5, 8, 8},
	{4, 5, 8, 11}, {4, 8, 10, 11}, {4, 9, 12, 16}, {4, 9, 16, 16}, {6, 10, 12, 18},
	{6, 10, 17, 16}, {6, 11, 16, 19}, {6, 13, 18, 21}, {7, 14, 21, 25}, {8, 16, 20, 25},
	{8, 17, 23, 25}, {9, 17, 23, 34}, {9, 18, 25, 30}, {10, 20, 27, 32}, {12, 21, 29, 35},
	{12, 23, 34, 37}, {12, 25, 34, 40}, {13, 26, 35, 42}, {14, 28, 38, 45}, {15, 29, 40, 48},
	{16, 31, 43, 51}, {17, 33, 45, 54}, {18, 35, 48, 57}, {19, 37, 51, 60}, {19, 38, 53, 63},
	{20, 40, 56, 66}, {21, 43, 59, 70}, {22, 45, 62, 74}, {24, 47, 65, 77}, {25, 49, 68, 81},
}

// blockLayout describe cómo se parten los datos en bloques para una versión y
// un nivel. group2 tiene un codeword de datos más que group1.
type blockLayout struct {
	ecPerBlock  int
	numG1, dcG1 int // bloques del grupo 1 y sus codewords de datos
	numG2, dcG2 int
	totalData   int
}

// layoutFor deriva el reparto en bloques con la regla del estándar: los datos
// se reparten lo más parejo posible entre los bloques, y los que sobran (resto)
// se ubican en el grupo 2 con un codeword más cada uno.
func layoutFor(version int, level Level) blockLayout {
	ec := ecPerBlock[version-1][level]
	blocks := numBlocks[version-1][level]
	totalData := totalCodewords[version-1] - ec*blocks
	dcG1 := totalData / blocks
	numG2 := totalData % blocks
	numG1 := blocks - numG2
	return blockLayout{
		ecPerBlock: ec,
		numG1:      numG1, dcG1: dcG1,
		numG2: numG2, dcG2: dcG1 + 1,
		totalData: totalData,
	}
}

// DataCapacityBits devuelve cuántos bits de datos entran en una versión/nivel
// (los codewords de datos por 8).
func dataCapacityBits(version int, level Level) int {
	return (totalCodewords[version-1] - ecPerBlock[version-1][level]*numBlocks[version-1][level]) * 8
}

// alignmentPositions[v-1] son las coordenadas centrales de los patrones de
// alineación de la versión (producto cartesiano, menos las esquinas ocupadas
// por los buscadores). La versión 1 no tiene.
var alignmentPositions = [40][]int{
	{}, {6, 18}, {6, 22}, {6, 26}, {6, 30}, {6, 34}, {6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50},
	{6, 30, 54}, {6, 32, 58}, {6, 34, 62}, {6, 26, 46, 66}, {6, 26, 48, 70}, {6, 26, 50, 74}, {6, 30, 54, 78}, {6, 30, 56, 82}, {6, 30, 58, 86}, {6, 34, 62, 90},
	{6, 28, 50, 72, 94}, {6, 26, 50, 74, 98}, {6, 30, 54, 78, 102}, {6, 28, 54, 80, 106}, {6, 32, 58, 84, 110}, {6, 30, 58, 86, 114}, {6, 34, 62, 90, 118}, {6, 26, 50, 74, 98, 122}, {6, 30, 54, 78, 102, 126}, {6, 26, 52, 78, 104, 130},
	{6, 30, 56, 82, 108, 134}, {6, 34, 60, 86, 112, 138}, {6, 30, 58, 86, 114, 142}, {6, 34, 62, 90, 118, 146}, {6, 30, 54, 78, 102, 126, 150}, {6, 24, 50, 76, 102, 128, 154}, {6, 28, 54, 80, 106, 132, 158}, {6, 32, 58, 84, 110, 136, 162}, {6, 26, 54, 82, 110, 138, 166}, {6, 30, 58, 86, 114, 142, 170},
}
