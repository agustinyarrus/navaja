package imaging

import "image"

// webpcolor.go convierte a RGB las WebP CON PÉRDIDA exactamente como libwebp,
// el decodificador de referencia (el de Chrome, Pillow y el propio Google).
//
// golang.org/x/image/webp entrega la imagen en YCbCr 4:2:0 y la conversión
// estándar de Go la trata como un JPEG: rango completo (JFIF) y el croma del
// píxel más cercano. VP8 usa BT.601 en rango LIMITADO, y libwebp además
// interpola el croma con un filtro 9-3-3-1 entre filas vecinas ("fancy
// upsampling"). Con la conversión de Go los colores salían corridos hasta 20
// niveles en cada píxel; con esta, la salida coincide con libwebp al bit.
//
// Es la aritmética de src/dsp/yuv.h y src/dsp/upsampling.c de libwebp. O(píxeles).

const (
	yuvFix2  = 6 // bits de precisión de la conversión en punto fijo
	yuvMask2 = (256 << yuvFix2) - 1
)

func multHi(v, coeff int) int { return (v * coeff) >> 8 }

// clip8 lleva el valor en punto fijo a 0–255, saturando.
func clip8(v int) uint8 {
	if v&^yuvMask2 == 0 {
		return uint8(v >> yuvFix2)
	}
	if v < 0 {
		return 0
	}
	return 255
}

func yuvToR(y, v int) uint8 { return clip8(multHi(y, 19077) + multHi(v, 26149) - 14234) }
func yuvToG(y, u, v int) uint8 {
	return clip8(multHi(y, 19077) - multHi(u, 6419) - multHi(v, 13320) + 8708)
}
func yuvToB(y, u int) uint8 { return clip8(multHi(y, 19077) + multHi(u, 33050) - 17685) }

// packUV empaqueta u y v en un solo entero (u abajo, v en el bit 16) para
// interpolar los dos canales con una sola suma, como hace libwebp.
func packUV(u, v byte) uint32 { return uint32(u) | uint32(v)<<16 }

// putRGB escribe un píxel RGBA opaco a partir de luma y croma empaquetado.
func putRGB(dst []byte, x int, y byte, uv uint32) {
	u, v, yy := int(uv&0xff), int(uv>>16), int(y)
	i := 4 * x
	dst[i] = yuvToR(yy, v)
	dst[i+1] = yuvToG(yy, u, v)
	dst[i+2] = yuvToB(yy, u)
	dst[i+3] = 0xff
}

// upsampleLinePair es UpsampleRgbaLinePair de libwebp: convierte dos filas de
// luma (bot puede faltar) interpolando el croma de las filas de croma top y cur
// con pesos 9-3-3-1 (los diagonales se precalculan una vez por par de píxeles).
func upsampleLinePair(topY, botY, topU, topV, curU, curV, topDst, botDst []byte, width int) {
	lastPair := (width - 1) >> 1
	tlUV := packUV(topU[0], topV[0]) // arriba a la izquierda
	lUV := packUV(curU[0], curV[0])  // izquierda
	putRGB(topDst, 0, topY[0], (3*tlUV+lUV+0x00020002)>>2)
	if botY != nil {
		putRGB(botDst, 0, botY[0], (3*lUV+tlUV+0x00020002)>>2)
	}
	for x := 1; x <= lastPair; x++ {
		tUV := packUV(topU[x], topV[x])
		uv := packUV(curU[x], curV[x])
		avg := tlUV + tUV + lUV + uv + 0x00080008
		diag12 := (avg + 2*(tUV+lUV)) >> 3
		diag03 := (avg + 2*(tlUV+uv)) >> 3
		putRGB(topDst, 2*x-1, topY[2*x-1], (diag12+tlUV)>>1)
		putRGB(topDst, 2*x, topY[2*x], (diag03+tUV)>>1)
		if botY != nil {
			putRGB(botDst, 2*x-1, botY[2*x-1], (diag03+lUV)>>1)
			putRGB(botDst, 2*x, botY[2*x], (diag12+uv)>>1)
		}
		tlUV, lUV = tUV, uv
	}
	if width&1 == 0 { // ancho par: el último píxel no tiene par a la derecha
		putRGB(topDst, width-1, topY[width-1], (3*tlUV+lUV+0x00020002)>>2)
		if botY != nil {
			putRGB(botDst, width-1, botY[width-1], (3*lUV+tlUV+0x00020002)>>2)
		}
	}
}

// webpToNRGBA convierte el YCbCr 4:2:0 de una WebP con pérdida (y su plano de
// alfa, si lo tiene) a NRGBA con el recorrido de filas de EmitFancyRGB:
//   - la fila 0 va sola, con su fila de croma como "arriba" y "actual";
//   - después, pares de filas (2k-1, 2k) con las filas de croma (k-1, k);
//   - si la altura es par, la última fila va sola con la última fila de croma.
func webpToNRGBA(m *image.YCbCr, alpha []byte, alphaStride int) *image.NRGBA {
	w, h := m.Rect.Dx(), m.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	if w == 0 || h == 0 {
		return out
	}
	uvw := (w + 1) / 2
	yRow := func(j int) []byte { return m.Y[j*m.YStride : j*m.YStride+w] }
	uRow := func(j int) []byte { return m.Cb[j*m.CStride : j*m.CStride+uvw] }
	vRow := func(j int) []byte { return m.Cr[j*m.CStride : j*m.CStride+uvw] }
	dst := func(j int) []byte { return out.Pix[j*out.Stride : j*out.Stride+4*w] }

	upsampleLinePair(yRow(0), nil, uRow(0), vRow(0), uRow(0), vRow(0), dst(0), nil, w)
	for k := 1; 2*k < h; k++ {
		upsampleLinePair(yRow(2*k-1), yRow(2*k), uRow(k-1), vRow(k-1), uRow(k), vRow(k), dst(2*k-1), dst(2*k), w)
	}
	if h%2 == 0 {
		last := h/2 - 1
		upsampleLinePair(yRow(h-1), nil, uRow(last), vRow(last), uRow(last), vRow(last), dst(h-1), nil, w)
	}

	if alpha != nil { // libwebp en modo RGBA: el alfa va tal cual, sin premultiplicar
		for j := 0; j < h; j++ {
			row := out.Pix[j*out.Stride:]
			src := alpha[j*alphaStride:]
			for i := 0; i < w; i++ {
				row[4*i+3] = src[i]
			}
		}
	}
	return out
}

// convertWebP aplica la conversión de libwebp a lo que devolvió x/image/webp:
// las WebP con pérdida llegan como YCbCr (o NYCbCrA si tienen alfa); las sin
// pérdida ya vienen en NRGBA y se devuelven intactas.
func convertWebP(img image.Image) image.Image {
	switch m := img.(type) {
	case *image.NYCbCrA:
		if m.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			return webpToNRGBA(&m.YCbCr, m.A, m.AStride)
		}
	case *image.YCbCr:
		if m.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			return webpToNRGBA(m, nil, 0)
		}
	}
	return img
}
