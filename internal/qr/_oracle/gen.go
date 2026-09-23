//go:build ignore

// gen.go emite un lote de PNGs de QR variados (modos, niveles, tamaños) a una
// carpeta, con un manifiesto de qué texto lleva cada uno, para que un
// decodificador externo (zxing) verifique que se leen y coinciden.
package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/agustinyarrus/navaja/internal/qr"
)

func main() {
	out := os.Args[1]
	os.MkdirAll(out, 0o755)

	casos := []string{
		"https://github.com/agustinyarrus/navaja",
		"12345678901234567890",
		"HELLO WORLD 123 $%*+-./:",
		"café con leche y medialunas — ñandú",
		"WIFI:S:MiRed;T:WPA;P:clave-secreta-123;;",
		strings.Repeat("A", 300),
		strings.Repeat("dato variado 0123 ", 40),
		"x",
		"BEGIN:VCARD\nFN:Agustín\nEND:VCARD",
		"מבחן unicode 日本語 test",
		strings.Repeat("9", 500),           // numérico largo → versión alta
		strings.Repeat("ABCDE12345 ", 120), // alfanumérico enorme
		strings.Repeat("λ", 400),           // bytes multibyte, versión grande
		"tel:+54 9 11 5555-5555",
		"mailto:test@example.com?subject=hola",
	}
	niveles := []qr.Level{qr.L, qr.M, qr.Q, qr.H}

	type entry struct {
		File  string `json:"file"`
		Text  string `json:"text"`
		Level string `json:"level"`
		Ver   int    `json:"version"`
		Mask  int    `json:"mask"`
	}
	var manifest []entry

	idx := 0
	for _, texto := range casos {
		for _, lvl := range niveles {
			m, err := qr.Encode([]byte(texto), qr.Options{Level: lvl, Mask: -1})
			if err != nil {
				fmt.Fprintf(os.Stderr, "encode falló (%q, %s): %v\n", texto, lvl, err)
				continue
			}
			name := fmt.Sprintf("qr_%03d.png", idx)
			f, _ := os.Create(filepath.Join(out, name))
			png.Encode(f, m.Image(6, 4))
			f.Close()
			manifest = append(manifest, entry{name, texto, lvl.String(), m.Version, m.Mask})
			idx++
		}
	}
	mf, _ := os.Create(filepath.Join(out, "manifest.json"))
	json.NewEncoder(mf).Encode(manifest)
	mf.Close()
	fmt.Printf("%d PNGs de QR en %s\n", idx, out)
}
