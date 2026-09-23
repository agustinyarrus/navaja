# navaja

Herramientas de consola para Windows, escritas en Go puro. Cada herramienta es un único `.exe` portable: sin instalador, sin dependencias en tiempo de ejecución, sin nube. Todo lo que procesan queda en la máquina.

Comparten un núcleo de consola propio (colores pastel sobre negro, progreso vivo, tarjeta de resumen, números en formato es-AR) y se verifican contra herramientas independientes, no contra sí mismas.

## Estado

| Herramienta | Qué hace | Estado |
|---|---|---|
| `img` | Convierte imágenes en lote: webp, gif, jpg, bmp, tiff → png y demás | ✅ lista (ver pendiente con webp con pérdida) |
| `clip2qr` | Dibuja en la terminal un QR del portapapeles, y opcionalmente lo guarda como PNG | ✅ lista |
| `pdf-merge` | Combina PDF con selección de páginas, marcadores, formularios, cifrado y deduplicación | ✅ lista |
| `vidsquash` | Comprime un video para que entre en un tamaño (Discord, WhatsApp, mail) | 🚧 motor escrito, falta el ejecutable |
| `killport` | Muestra qué proceso ocupa un puerto y lo libera | 🚧 pendiente |

Lo que falta, en orden, está en [docs/PENDIENTE.md](docs/PENDIENTE.md).

## Compilar

Requisitos: [Go](https://go.dev/dl) 1.24 o más nuevo (el `go.mod` pide 1.26 y Go descarga solo esa versión la primera vez).

```powershell
.\build.ps1          # compila cada herramienta a dist\<nombre>.exe
.\build.ps1 -Test    # antes corre go vet y todos los tests
```

Los `.exe` no necesitan nada más para funcionar, salvo `vidsquash`, que usa ffmpeg (`winget install Gyan.FFmpeg`).

Opciones comunes a todas: `-h/--help`, `-V/--version`, `--no-color` (también respeta la variable `NO_COLOR`). Si la salida no es una consola (pipe o archivo), no se emite ningún escape de color.

Códigos de salida: `0` todo bien · `1` algunos elementos fallaron · `2` línea de comandos inválida · `3` no se pudo hacer nada · `130` cancelado con Ctrl+C.

## img

Convierte imágenes entre formatos, en paralelo, con barra de progreso y tarjeta final.

```powershell
img *.webp                           # cada .webp a .png, al lado del original
img fotos\ -r --to jpg -q 82         # una carpeta entera, con subcarpetas
img loader.gif --frames              # un gif animado → loader-001.png, loader-002.png…
img logo.webp --to jpg -b '#0b0b0f'  # aplanar la transparencia sobre un fondo oscuro
```

- Lee webp (con y sin pérdida, con alfa), gif (estático y animado), png, jpg, bmp y tiff. Escribe png, jpg, gif, bmp y tiff (webp no tiene codificador en Go puro).
- Reconoce el formato por el contenido, no por la extensión: un `.png` que en realidad es webp se convierte igual.
- Los GIF animados se componen respetando el método de disposición de cada cuadro: `--frames` exporta cuadros completos, no los recortes parciales que guarda el archivo.
- El alfa se conserva hacia png y se aplana sobre `--background` hacia jpg y bmp.
- Nunca borra el original ni sobrescribe sin `--force`; detecta cuando dos entradas caerían en la misma salida. Escritura atómica: un corte no deja archivos a medias.

## clip2qr

Toma el texto del portapapeles (o de un argumento, o de la entrada estándar) y dibuja un código QR en la terminal. Pensado para pasar una URL de la PC al teléfono sin subir nada a ningún lado.

```powershell
clip2qr                          # QR de lo que haya en el portapapeles
clip2qr --text "https://…"       # de un texto puntual
echo hola | clip2qr -            # de la entrada estándar
clip2qr -e h -o wifi.png         # corrección alta y además guardado como PNG
```

- Codificador QR propio, sin dependencias: modos numérico, alfanumérico y byte (UTF-8), versiones 1 a 40, niveles L/M/Q/H, corrección Reed–Solomon sobre GF(256) y elección de la mejor de las 8 máscaras por las reglas de penalización del estándar.
- En la terminal usa medios bloques (`▀ ▄ █`): el código sale cuadrado y ocupa la mitad de líneas.

## pdf-merge

Combina varios PDF en uno, en el orden dado, con selección de páginas por archivo.

```powershell
pdf-merge a.pdf b.pdf c.pdf -o todo.pdf
pdf-merge contrato.pdf@1-2 firmas.pdf -o firmado.pdf
pdf-merge libro.pdf@impares -o impares.pdf     # también: pares, reverso, 8- (hasta el final)
pdf-merge escaneos\ -o legajo.pdf              # una carpeta entera, en orden natural
pdf-merge protegido.pdf --password clave       # PDF que pide contraseña
```

- **Parser propio y tolerante**: tablas xref clásicas y en stream, object streams, predictores PNG/TIFF, actualizaciones incrementales, basura antes del encabezado, offsets corridos (reconstruye la tabla barriendo el archivo) y `/Length` indirectos.
- **Deduplicación**: las imágenes y fuentes idénticas se guardan una vez aunque vengan de archivos distintos (hash de Merkle sobre el grafo de objetos, con las componentes fuertemente conexas de Tarjan). En una prueba con 56 PDF reales, la salida bajó de 94,6 MB a 21,8 MB.
- **Marcadores**: uno por archivo, con los marcadores originales anidados y apuntando a las páginas nuevas. Se podan los que apuntan a páginas no incluidas. `--bookmarks auto|files|keep|none`.
- **Enlaces internos**: los destinos con nombre se resuelven a destinos explícitos, así no chocan nombres entre archivos. Un enlace a una página que quedó afuera queda inerte, no roto.
- **Formularios**: los campos siguen siendo rellenables; si dos archivos tienen un campo con el mismo nombre, se renombra (`fecha` y `fecha_2`) para que no compartan valor. Avisa que una firma digital deja de validar al combinar.
- **Cifrado**: RC4 de 40 y 128 bits, AES-128 y AES-256 (revisiones 2 a 6 del manejador estándar). Prueba primero la contraseña vacía (los PDF "protegidos" que se abren sin pedir nada) y después las de `--password`, como de usuario o de propietario. La salida no va cifrada y lo avisa.

## vidsquash (en construcción)

Comprime un video para que entre en un tamaño máximo con la mejor calidad posible.

```powershell
vidsquash partida.mp4 --for discord                    # 10 MB
vidsquash viaje.mov -s 16MB                            # para WhatsApp
vidsquash charla.mp4 -s 25MB --from 2:10 --to 5:40     # solo un tramo
```

Ya está escrito (en `internal/`): el planificador, que reparte los bits entre audio y video y elige resolución y cuadros por segundo por bits por píxel, la codificación en dos pasadas con corrección del tamaño, la conversión HDR → SDR y la medición de calidad con VMAF. Falta el punto de entrada del ejecutable y probarla con videos reales: ver [docs/PENDIENTE.md](docs/PENDIENTE.md).

## killport (pendiente)

Para cuando un servidor de desarrollo queda colgado ocupando un puerto: muestra qué proceso escucha ahí (TCP/UDP, IPv4/IPv6, PID y nombre) y lo cierra, como `netstat -ano` más `taskkill` en un solo paso. No está implementada todavía.

## Cómo se verifica

Cada herramienta se contrasta con un decodificador o lector independiente, no consigo misma:

| Herramienta | Oráculo | Resultado |
|---|---|---|
| `img` | Pillow (libwebp, libjpeg…) píxel a píxel | webp sin pérdida, alfa, gif: diferencia 0 · webp con pérdida: pendiente |
| `clip2qr` | zxing-cpp decodifica los PNG | 60 de 60, hasta la versión 40 |
| `pdf-merge` | qpdf (estructura), pypdf (texto) y PDFium (render píxel a píxel) | 56 de 56 PDF reales; 13 de 13 casos de cifrado; 6 de 6 de marcadores; formularios con todos sus valores |

Además hay 64 tests de Go (`go test ./...`). El detalle, los fixtures y cómo correr todo: [docs/VERIFICACION.md](docs/VERIFICACION.md). La arquitectura y las decisiones de diseño: [docs/ARQUITECTURA.md](docs/ARQUITECTURA.md).

## Estructura

```
cmd/                  un main por herramienta (clip2qr, img, pdf-merge)
internal/
  tui/                consola: paleta, progreso vivo, tarjetas, tablas, formato es-AR
  cli/                flags estilo GNU, ayuda, sugerencias "¿quisiste decir…?"
  fsx/                patrones con ** , orden natural, escritura atómica
  batch/              tareas en paralelo con barra de progreso
  win/                llamadas a la API de Windows (consola, portapapeles)
  textdist/           distancia de edición para las sugerencias
  imaging/            decodificar, componer y codificar imágenes
  qr/                 codificador QR
  pdf/                parser, merge, deduplicación, marcadores, formularios, cifrado
  ffx/  squash/       ffmpeg y el planificador/codificador de vidsquash
  tools/<herramienta> la interfaz de línea de comandos de cada una
docs/                 arquitectura, verificación y pendientes
scripts/              generador de fixtures de prueba
```

## Seguir desde otra PC

1. El repositorio es privado: en la PC nueva, `gh auth login` con la cuenta `agustinyarrus`.
2. `gh repo clone agustinyarrus/navaja` (o `git clone https://github.com/agustinyarrus/navaja`).
3. Instalar Go y correr `.\build.ps1 -Test`.
4. Para las pruebas contra oráculos: Python 3.12+ con `pip install pillow numpy pikepdf pypdf pypdfium2 zxing-cpp segno reportlab`, ffmpeg, y `.\scripts\fixtures.ps1` para regenerar los archivos de prueba.
5. Seguir por [docs/PENDIENTE.md](docs/PENDIENTE.md).

## Licencia

MIT.
