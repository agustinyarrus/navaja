// pdf-merge combina varios PDF en uno, local, con selección de páginas.
package main

import (
	"os"

	"github.com/agustinyarrus/navaja/internal/tools/pdfmerge"
	"github.com/agustinyarrus/navaja/internal/tui"
	"github.com/agustinyarrus/navaja/internal/version"
)

func main() {
	t := tui.Open()
	defer t.Close()
	os.Exit(pdfmerge.Main(t, version.String(), os.Args[1:]))
}
