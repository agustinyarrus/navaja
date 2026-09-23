package qr

import (
	"bytes"
	"testing"
)

// TestReedSolomonVector fija el vector conocido de "HELLO" v1-M contra los
// bytes de corrección correctos (verificados con un decodificador externo). Es
// la guardia contra la regresión del orden del polinomio generador, que hacía
// que ningún QR se pudiera leer aunque los datos fueran correctos.
func TestReedSolomonVector(t *testing.T) {
	data := []byte{0x20, 0x2B, 0x0B, 0x78, 0xCC, 0x00, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11}
	want := []byte{0x4D, 0xDD, 0x85, 0x13, 0xDB, 0x5F, 0x47, 0x73, 0x9A, 0x65}
	if got := rsEncode(data, 10); !bytes.Equal(got, want) {
		t.Errorf("rsEncode(HELLO)\n got  % X\n want % X", got, want)
	}
}

// TestModeDetection: el detector elige el modo más compacto.
func TestModeDetection(t *testing.T) {
	casos := []struct {
		in   string
		want mode
	}{
		{"12345", modeNumeric},
		{"HELLO WORLD", modeAlphanumeric},
		{"HTTPS://EXAMPLE.COM", modeAlphanumeric},
		{"hola mundo", modeByte}, // minúsculas no están en el juego alfanumérico
		{"café", modeByte},       // UTF-8
	}
	for _, c := range casos {
		if got := detectMode([]byte(c.in)); got != c.want {
			t.Errorf("detectMode(%q) = %v, esperaba %v", c.in, got, c.want)
		}
	}
}

// TestVersionSelection: se elige la versión más chica que entra, y crece con el
// tamaño del dato.
func TestVersionSelection(t *testing.T) {
	corto, err := chooseVersion(modeByte, 10, M, 1)
	if err != nil || corto != 1 {
		t.Errorf("dato corto → versión %d (err %v), esperaba 1", corto, err)
	}
	largo, err := chooseVersion(modeByte, 400, M, 1)
	if err != nil {
		t.Fatal(err)
	}
	if largo <= corto {
		t.Errorf("un dato más largo debería pedir una versión mayor (%d vs %d)", largo, corto)
	}
}

// TestDemasiadoLargo: un dato imposible da ErrTooLong, no un pánico.
func TestDemasiadoLargo(t *testing.T) {
	big := make([]byte, 8000)
	for i := range big {
		big[i] = 'A'
	}
	if _, err := Encode(big, DefaultOptions()); err == nil {
		t.Fatal("esperaba ErrTooLong con 8000 bytes")
	}
}

// TestTamañoMatriz: el lado de la matriz sigue la fórmula 17+4v.
func TestTamañoMatriz(t *testing.T) {
	m, err := Encode([]byte("test"), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 17+4*m.Version {
		t.Errorf("lado = %d, esperaba %d para v%d", m.Size, 17+4*m.Version, m.Version)
	}
}

// TestMascaraElegida: sin forzar, la máscara elegida está en 0..7.
func TestMascaraElegida(t *testing.T) {
	m, err := Encode([]byte("elegir mascara"), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if m.Mask < 0 || m.Mask > 7 {
		t.Errorf("máscara = %d, fuera de 0..7", m.Mask)
	}
}

// TestFinderPatterns: los tres buscadores están donde deben (esquina = oscuro,
// anillo interior claro).
func TestFinderPatterns(t *testing.T) {
	m, err := Encode([]byte("x"), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	esquinas := [][2]int{{0, 0}, {m.Size - 7, 0}, {0, m.Size - 7}}
	for _, e := range esquinas {
		if !m.At(e[0], e[1]) {
			t.Errorf("buscador en (%d,%d): esquina debería ser oscura", e[0], e[1])
		}
		if m.At(e[0]+1, e[1]+1) {
			t.Errorf("buscador en (%d,%d): el anillo interior debería ser claro", e[0], e[1])
		}
		if !m.At(e[0]+3, e[1]+3) {
			t.Errorf("buscador en (%d,%d): el centro debería ser oscuro", e[0], e[1])
		}
	}
}

// TestModuloOscuro: el módulo oscuro fijo siempre está presente.
func TestModuloOscuro(t *testing.T) {
	m, err := Encode([]byte("dark"), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !m.At(8, m.Size-8) {
		t.Error("falta el módulo oscuro obligatorio en (8, size-8)")
	}
}
