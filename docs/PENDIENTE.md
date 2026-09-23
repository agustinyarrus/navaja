# Pendiente

Lo que falta, en el orden en que conviene hacerlo. Cada punto dice qué hay hoy, qué falta y cómo se verifica.

## 1. vidsquash: terminarla y probarla

**Hoy está escrito:**

- `internal/ffx`: encontrar ffmpeg/ffprobe, leer las propiedades del video (incluida la rotación y si es HDR), correr ffmpeg leyendo el progreso de `-progress`.
- `internal/squash`: el planificador (7 tests), la codificación en dos pasadas con corrección del tamaño por el método de la secante (se rehace solo la segunda pasada), la conversión HDR → SDR y la medición de calidad con VMAF sobre tres muestras.
- `internal/tools/vidsquash`: la interfaz completa (flags, presets de servicios, progreso vivo, tarjeta final).

**Falta:**

1. `cmd/vidsquash/main.go`: el punto de entrada, con la misma forma que los otros `cmd/*/main.go` (abrir el terminal, llamar a `vidsquash.Main` con la versión, salir con su código).
2. Probarla de punta a punta:
   - un clip sintético 1080p60 (`ffmpeg -f lavfi -i testsrc2=s=1920x1080:r=60:d=20 -f lavfi -i sine=d=20 …`);
   - un video real de celular vertical;
   - un video HDR de iPhone (HLG) con rotación;
   - objetivos de 10, 16 y 25 MB. En todos, `ffprobe` tiene que confirmar que el tamaño es menor o igual al objetivo y que la duración y las pistas se conservan.
3. `--analyze`: elegir resolución y fps midiendo (codificar muestras de cada candidata y quedarse con la de mejor VMAF) en vez de solo por bits por píxel. Está diseñado, no escrito.

**Referencias de velocidad en la PC de desarrollo:** x264 `medium` a 720p30 ≈ 1,5 veces el tiempo real; VMAF sobre 20 s de 1080p60 ≈ 22 s. Por eso la calidad se mide en muestras y no en el video entero.

## 2. killport

No está implementada. La idea:

- leer las tablas TCP y UDP del sistema (IPv4 e IPv6) con el PID de cada socket, como `netstat -ano`;
- mostrar qué proceso ocupa cada puerto pedido (nombre, PID, dirección, estado);
- pedir confirmación y cerrarlo con `taskkill /PID` (`/T` para incluir los procesos hijos);
- verificar después que el puerto quedó libre.

Flags previstos: `--list` (solo mirar), `--yes`, `--tree`, `--tcp`/`--udp`, `--all-states`, y puertos como `3000`, `8080,8081` o `5000-5003`.

## 3. Mejoras

**img**
- WebP animadas: `x/image/webp` solo lee imágenes estáticas.
- Escribir WebP: no hay codificador en Go puro.

**pdf-merge**
- Salida con object streams y xref stream: la estructura ocuparía menos.
- Recomprimir los streams que vienen sin filtro.
- Conservar la numeración de páginas propia de cada archivo (`/PageLabels`) y fusionar las capas (`/OCProperties`).
- La estructura de accesibilidad (PDF etiquetado) hoy se descarta al combinar.

**Distribución**
- Releases en GitHub con los `.exe` que genera `build.ps1`, y opcionalmente un instalador Inno Setup como las otras apps.
- Integración continua: `go vet` y `go test` en `windows-latest` con GitHub Actions.
