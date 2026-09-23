// img convierte imágenes entre formatos en lote (webp/gif/jpg/bmp/tiff → png…).
package main

import (
	"os"

	"github.com/agustinyarrus/navaja/internal/tools/img"
	"github.com/agustinyarrus/navaja/internal/tui"
	"github.com/agustinyarrus/navaja/internal/version"
)

func main() {
	t := tui.Open()
	defer t.Close()
	os.Exit(img.Main(t, version.String(), os.Args[1:]))
}
