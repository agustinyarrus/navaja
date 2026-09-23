package qr

import (
	"image"
	"image/color"
)

// Image renderiza la matriz como imagen en escala de grises: cada módulo es un
// cuadrado de `scale` píxeles, con un borde claro (zona de silencio) de `quiet`
// módulos, obligatorio para que un lector encuentre el código.
func (m *Matrix) Image(scale, quiet int) *image.Gray {
	if scale < 1 {
		scale = 1
	}
	if quiet < 0 {
		quiet = 0
	}
	side := (m.Size + 2*quiet) * scale
	img := image.NewGray(image.Rect(0, 0, side, side))
	// Fondo claro.
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; x++ {
			if !m.At(x, y) {
				continue
			}
			px0 := (x + quiet) * scale
			py0 := (y + quiet) * scale
			for dy := 0; dy < scale; dy++ {
				row := (py0 + dy) * img.Stride
				for dx := 0; dx < scale; dx++ {
					img.Pix[row+px0+dx] = 0x00
				}
			}
		}
	}
	return img
}

var _ = color.Gray{} // image/color queda disponible para quien extienda el render
