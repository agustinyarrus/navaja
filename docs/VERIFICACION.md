# Verificación

La regla: cada herramienta se contrasta con algo **independiente**. Un parser que lee lo que escribió su propio escritor no prueba nada; qpdf, PDFium, Pillow o zxing sí.

## Tests de Go

```powershell
go test ./...
```

64 tests en `cli`, `fsx`, `textdist`, `tui`, `imaging`, `qr`, `pdf` y `squash`. Los de `imaging` y `pdf` leen fixtures de `%TEMP%\navaja-fx` y `%TEMP%\navaja-pdf`; si no están se saltean (no fallan). Para generarlos:

```powershell
.\scripts\fixtures.ps1     # necesita ffmpeg y Python con pillow, pikepdf y reportlab
```

## img

```powershell
python internal\imaging\_oracle\pixel_check.py origen.webp convertida.png
```

Compara la salida contra el origen decodificado por Pillow (libwebp, libjpeg…), píxel a píxel.

| Origen | Resultado |
|---|---|
| WebP sin pérdida | diferencia 0 en RGB y alfa |
| WebP con alfa | diferencia 0 |
| GIF | diferencia 0 |
| WebP con pérdida | diferencia 0 |

```powershell
python internal\imaging\_oracle\webp_stress.py <img.exe> <carpeta>
```

El estrés de las WebP con pérdida genera 18 casos con Pillow y los compara contra libwebp; los 18 dan idénticos:

- tamaños mínimos e impares, de 1×1 a 333×201, que es donde se complican los bordes del sobremuestreo;
- ruido de color puro, donde el croma cambia en cada píxel;
- los mismos casos con alfa.

**Lección:** las WebP con pérdida llegaron a diferir hasta 20 niveles en todos los píxeles. `x/image/webp` entrega YCbCr y la conversión de Go lo trataba como un JPEG: rango completo y el croma del píxel más cercano. VP8 usa BT.601 en rango limitado, y libwebp interpola el croma con un filtro 9-3-3-1. `webpcolor.go` replica esa aritmética.

También se verificó:

- el aplanado del alfa sobre un fondo (teal al 50 % sobre `#0b0b0f` da `(77, 112, 108)`, la cuenta exacta);
- los 8 cuadros de un GIF animado salen compuestos a tamaño completo;
- una tanda de 60 archivos en paralelo.

## clip2qr

```powershell
go run .\internal\qr\_oracle\gen.go <carpeta>          # 60 PNG de QR variados + manifiesto
python .\internal\qr\_oracle\decode.py <carpeta>       # zxing-cpp los decodifica y compara
```

Resultado: 60 de 60. Cubre los modos numérico, alfanumérico y byte (UTF-8, hebreo, japonés), los cuatro niveles de corrección y versiones hasta la 40.

**Lección:** al principio ningún código se podía leer. Comparando la matriz módulo a módulo con la de `segno` se vio que los datos coincidían y la corrección no. El polinomio generador de Reed–Solomon estaba en orden inverso. `TestReedSolomonVector` fija ahora los bytes correctos de un vector conocido.

## pdf-merge

```powershell
.\internal\pdf\_oracle\all.ps1 [-Corpus lista.txt]
```

Corre todo y termina con una tarjeta de resumen. El corpus es un archivo de texto con una ruta de PDF por línea, por ejemplo:

```powershell
Get-ChildItem $HOME\Documents -Recurse -Filter *.pdf | ForEach-Object FullName > $env:TEMP\navaja-sweep-list.txt
```

| Etapa | Script | Qué comprueba |
|---|---|---|
| tests de Go | `go test ./internal/pdf/` | parser, merge, deduplicación, Tarjan |
| fixtures | `verify.py` | rangos, reverso, object streams |
| marcadores y enlaces | `outline_cases.ps1` + `outline_check.py` | 6 casos, incluidos destinos con nombre en todas sus formas |
| cifrado | `crypt_check.py` | 13 casos: R2–R6, contraseña de usuario y de propietario, object streams cifrados |
| formularios | `list_forms.py` + `form_check.py` | campos únicos, sin widgets huérfanos, cada uno con su valor original |
| barrido | `sweep.py` | cada PDF del corpus combinado solo |
| estrés | `bigmerge.py` | todo el corpus en un único PDF |

`verify.py` es el oráculo triple que usan las demás etapas:

- **qpdf** (vía pikepdf): chequeo estructural estricto. Solo cuentan las advertencias nuevas, no las que ya traía el origen.
- **pypdf**: la cantidad de páginas y el texto de cada una, en orden.
- **PDFium** (el motor de Chrome, vía pypdfium2): cada página combinada renderizada y comparada píxel a píxel con la original.

Con un corpus de 56 PDF reales (generados por Chrome en varias versiones, iTextSharp, SAP NetWeaver, reportlab y otros), los 56 pasan. Combinados todos juntos dan 352 páginas en unos 2 s, y la deduplicación baja de 94,6 MB a 21,8 MB.

**Lecciones:** los oráculos encontraron seis defectos que las pruebas internas no veían.

- Los números reales se redondeaban a 6 decimales. La `/FontMatrix` de las fuentes Type 3 que genera Chrome (1/2048) quedaba en 0.000488 y la negrita se dibujaba un 0,06 % más chica. Ahora se usa la representación más corta que vuelve exacta al mismo float64.
- Los xref streams llevan el predictor PNG "Up" y hay que revertirlo antes de leerlos.
- Cuando `/Length` es una referencia indirecta, hay que resolverla para cortar el stream exacto.
- Puede venir basura antes del `%PDF-` (el estándar la tolera en el primer KB).
- En las revisiones 2 a 4, las contraseñas van en PDFDocEncoding: una "ñ" es el byte 0xF1, no los dos bytes de UTF-8.
- Advertencias que ya traía el PDF de origen aparecían como si fueran de la combinación. El oráculo ahora las compara contra las del origen.

`streamdiff.py` y `dictdiff.py` son herramientas de diagnóstico: comparan streams y diccionarios de origen y salida para encontrar dónde difieren.

## vidsquash

Los 7 tests del planificador (`go test ./internal/squash/`) comprueban:

- el plan nunca promete más bytes que el objetivo;
- los videos verticales se escalan por el lado corto;
- nunca se agranda un video;
- el recorte se respeta;
- las reglas del audio.

La prueba de punta a punta con videos reales está pendiente.
