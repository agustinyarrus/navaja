package qr

// bch.go calcula la información de formato (15 bits) y de versión (18 bits) con
// sus códigos BCH, tal como los fija el estándar. Son datos que van en zonas
// fijas de la matriz y llevan su propia corrección de errores.

// formatInfo devuelve los 15 bits de información de formato para un nivel y una
// máscara, ya enmascarados con el patrón fijo 0x5412 del estándar.
func formatInfo(level Level, mask int) uint32 {
	// Los 2 bits de nivel en el orden del estándar (no es L=0,M=1,…): 01,00,11,10.
	levelBits := map[Level]uint32{L: 0b01, M: 0b00, Q: 0b11, H: 0b10}[level]
	data := levelBits<<3 | uint32(mask)
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1)
		if rem&(1<<10) != 0 {
			rem ^= 0x537 // generador BCH(15,5): x^10+x^8+x^5+x^4+x^2+x+1
		}
	}
	return ((data<<10 | rem) ^ 0x5412) & 0x7fff
}

// versionInfo devuelve los 18 bits de información de versión (solo v≥7).
func versionInfo(version int) uint32 {
	data := uint32(version)
	rem := data
	for i := 0; i < 12; i++ {
		rem = (rem << 1)
		if rem&(1<<12) != 0 {
			rem ^= 0x1f25 // generador BCH(18,6)
		}
	}
	return data<<12 | rem
}
