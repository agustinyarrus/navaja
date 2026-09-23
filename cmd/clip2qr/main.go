// clip2qr dibuja un código QR del portapapeles (o de un texto) en la terminal.
package main

import (
	"os"

	"github.com/agustinyarrus/navaja/internal/tools/clip2qr"
	"github.com/agustinyarrus/navaja/internal/tui"
	"github.com/agustinyarrus/navaja/internal/version"
)

func main() {
	t := tui.Open()
	defer t.Close()
	os.Exit(clip2qr.Main(t, version.String(), os.Args[1:]))
}
