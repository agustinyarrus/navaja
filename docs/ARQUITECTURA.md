# Arquitectura

## Principios

- **Go puro, sin CGO.** Cada herramienta compila a un único `.exe` estático. Las llamadas a Windows van por `syscall`, sin bindings de C.
- **Una responsabilidad por paquete.** Lo que usan dos herramientas vive en un paquete compartido (`tui`, `cli`, `fsx`, `batch`); cada herramienta es solo su lógica y su interfaz.
- **Validar en la frontera.** Los flags se interpretan y validan una sola vez al parsear; adentro los valores ya son del tipo correcto.
- **Escritura atómica.** Toda salida va a un temporal en la misma carpeta y se renombra al final: un corte o un Ctrl+C nunca deja un archivo a medias ni pisa el anterior.
- **Salida visual que degrada.** Colores pastel sobre negro, progreso vivo y tarjeta de resumen en la consola; texto plano sin un solo escape cuando la salida es un pipe o un archivo.

## Núcleo compartido

### tui

- **Región viva**: el progreso se redibuja en el lugar a 20 cuadros por segundo y las líneas permanentes se imprimen por encima. Hay un orden de candados fijo para que no haya deadlock:
  - la función de dibujo corre sin el candado del terminal (puede tomar el del estado de la herramienta);
  - `Println` nunca llama a la función de dibujo: reusa el último cuadro.
- **Recorte seguro**: `ClipANSI` recorta una línea con escapes a N columnas visibles sin romper los colores. Las líneas vivas se cortan una columna antes del borde, para que el terminal no las parta y el conteo de líneas no se desfase.
- **Barra**: resolución de 1/8 de columna (bloques `▏▎▍▌▋▊▉█`) con degradé.
- **Tarjeta**: fondo apenas teñido, sin bordes.
- **Formato es-AR**: miles con punto, decimales con coma, bytes en unidades decimales (las que usan los servicios para sus límites).
- **Barra de tareas**: en Windows Terminal, el avance también se publica en el ícono y la pestaña (OSC 9;4).

### cli

- Flags al estilo GNU: `-o x`, `--out=x`, `-abc`, `--no-x`, `--`.
- Binders tipados con validación: enteros acotados, enumerados, tamaños (`25MB`, `10MiB`) y tiempos (`1:30`, `1m30s`).
- La ayuda se genera con el mismo lenguaje visual que el resto de la salida.
- Un flag mal escrito sugiere el más parecido ("¿quisiste decir --force?") por distancia de Damerau–Levenshtein. Es programación dinámica en O(n·m) con tres filas rodantes, y la transposición de dos letras cuenta como un solo error.

### fsx

- Patrones con `*`, `?`, `[]` y `**`, resueltos segmento a segmento. En Windows no distinguen mayúsculas, igual que el sistema de archivos. Ni cmd ni PowerShell expanden comodines para un exe nativo: lo hace la herramienta.
- Orden natural: `pag2` antes que `pag10`, comparando tramos de dígitos por valor y sin límite de largo.
- Si un archivo no existe, sugiere el más parecido de la misma carpeta.
- `WriteAtomic` reintenta el renombre con espera exponencial (un antivirus o un visor pueden tener el destino abierto un instante). No hace fsync a propósito: protege contra cortes del programa sin pagar un `FlushFileBuffers` por archivo.

### batch

- Pool de workers para tareas independientes.
- Los resultados se guardan por índice, así el resumen es determinista aunque terminen desordenados.
- Los contadores son atómicos.
- Un pánico en una tarea se convierte en un fallo de esa tarea, sin tirar abajo la tanda.

### win

- Las DLL del sistema se cargan por ruta absoluta de System32. Con el nombre pelado, `LoadLibrary` buscaría primero junto al exe, y un DLL plantado ahí ganaría.
- El portapapeles se lee y se escribe copiando con `RtlMoveMemory`, sin reinterpretar punteros nativos como punteros de Go.

## Herramientas

### imaging (img)

- El formato se reconoce por los primeros bytes (magic numbers), no por la extensión.
- **Composición de GIF**: los cuadros suelen guardar solo el rectángulo que cambió. Se reconstruye cada cuadro completo aplicando el método de disposición del anterior: nada, restaurar al fondo (transparente) o restaurar al lienzo previo. Cuesta O(cuadros × píxeles).
- **Salida GIF**: paleta de 256 colores con dithering de Floyd–Steinberg.
- **Formatos opacos**: la transparencia se aplana sobre el fondo elegido.

### qr (clip2qr)

- **GF(256)** con el polinomio `x⁸+x⁴+x³+x²+1` (0x11d) y tablas de exponenciales y logaritmos: una multiplicación son dos búsquedas.
- **Reed–Solomon**: el polinomio generador se memoiza por grado, con el coeficiente líder primero; ese orden fue el bug que impedía leer los códigos.
- **Tablas**: solo tres arreglos base (codewords por versión, corrección por bloque, cantidad de bloques). El reparto en grupos se deriva con la regla del estándar, así hay menos superficie para errores de transcripción.
- **Construcción**: información de formato y de versión con códigos BCH; colocación de datos en zigzag; las 8 máscaras evaluadas con las 4 reglas de penalización.

### pdf (pdf-merge)

**Lectura**

- **Lexer**: descendente recursivo sobre el archivo en memoria.
- **Números**: se escriben con la representación más corta que vuelve exacta al mismo float64. Un entero que desborda int64 se conserva como real.
- **Referencias cruzadas**:
  - tablas clásicas leídas por tokens (tolera filas de 19 o 21 bytes);
  - xref streams con `/W`, `/Index` y predictores PNG/TIFF;
  - object streams;
  - la cadena `/Prev` de la más nueva a la más vieja;
  - xref híbridos con `/XRefStm`.
- **Recuperación**:
  - si la tabla falla, se reconstruye barriendo el archivo por `N G obj`;
  - si un objeto no está donde dice la tabla (offsets corridos), la reconstrucción se hace una sola vez, al vuelo;
  - el trailer se busca en orden: el último `trailer`, después el xref stream más reciente, después el catálogo a mano.
- **Cifrado**: se configura antes de validar el catálogo, porque si el catálogo está en un object stream cifrado, sin clave ni se podría leer.
  - Las cadenas se descifran al cargar cada objeto directo.
  - Los streams se descifran al leer sus bytes, con memo.
  - Excepciones del estándar: el propio `/Encrypt`, los xref streams, los objetos dentro de un object stream (ya salen en claro) y los `/Metadata` con `EncryptMetadata false`.
  - Soporta RC4 de 40 y 128 bits, AES-128 y AES-256. Para la revisión 6 usa el Algorithm 2.B de ISO 32000-2.

**Combinación**

- **Un copiador por documento**: el mismo PDF pedido dos veces copia sus fuentes una sola vez.
- **Barreras del recorrido**: una referencia a una página no seleccionada pasa a null, y el árbol de páginas viejo y el catálogo de origen no se siguen. Así no se arrastran páginas huérfanas.
- **Dos pasadas**: primero se reservan los números de todas las páginas de salida y después se copia. Un enlace hacia adelante ya encuentra su destino.
- **Atributos heredados**: `MediaBox`, `CropBox`, `Resources` y `Rotate` se materializan en cada página.

**Deduplicación**

- **Hash de Merkle**: el hash de cada objeto se calcula sobre su forma canónica, con cada referencia reemplazada por el hash del objeto apuntado.
- **Orden**: los hijos quedan listos antes que los padres porque se recorren las componentes fuertemente conexas de Tarjan (versión iterativa), que salen en orden topológico inverso.
- **Objetos que conservan identidad**: los que están dentro de un ciclo, y las páginas, anotaciones, campos y capas.
- **Compactación**: al final se marca lo alcanzable desde el catálogo y se renumera. Todo es O(V + E), más el hashing de los bytes.

**Marcadores y enlaces**

- Se lee el árbol de marcadores de cada origen y se remapean sus destinos.
- Se podan los que no llevan a nada y se escribe un `/Count` correcto (positivo si está desplegado, negativo si está plegado).
- Los destinos con nombre se resuelven a destinos explícitos: tanto el `/Dests` de PDF 1.1 como el árbol `/Names` con sus `/Kids`.

**Formularios**

- Entran los campos que tienen algún widget en una página incluida, y se podan sus `/Kids`.
- Los nombres raíz que chocan entre archivos distintos se renombran.
- Se fusionan `/DR`, `/DA`, `/NeedAppearances` y `/SigFlags`. El `/XFA` se descarta, con aviso.

**Escritura**

- Tabla xref clásica.
- `/Length` siempre directo e igual a los bytes escritos.
- Claves ordenadas, para que la salida sea determinista.

**Diagnóstico**: `go run ./internal/pdf/_dbg/dbg.go archivo.pdf` muestra el xref y el trailer que ve el parser, usando `pdf.Debug`.

### ffx y squash (vidsquash)

**ffx**

- Busca ffmpeg junto al exe, después en el PATH, después en los accesos de winget.
- Lee el progreso de `-progress pipe:1` y se queda con el final de stderr para explicar un fallo.
- No abre ventanas propias.

**Planificador**

- **Presupuesto**: el objetivo menos un margen del 2 %, menos el overhead estimado del MP4 (4.096 B fijos, 12 B por cuadro de video y 6 B por cuadro de audio).
- **Audio**: baja de calidad antes que el video, pasa a mono por debajo de 64 kb/s y nunca supera al original.
- **Resolución y fps**:
  - La escalera se arma por el **lado corto**, para que un video vertical no se trate como uno de 1920 de alto.
  - Se elige la primera combinación de resolución y fps cuyos bits por píxel alcanzan: 0,050 para H.264, 0,035 para H.265.
  - A igual resolución prueba primero los fps originales y después la mitad, con un 10 % de tolerancia documentado a favor de conservar la resolución.

**Codificación**

- Dos pasadas. Si el archivo se pasa del objetivo o queda por debajo del 90 %, se repite solo la segunda pasada, porque las estadísticas de la primera sirven para cualquier bitrate.
- La corrección usa el método de la secante sobre tamaño(bitrate).
- Se queda con el intento más grande que entra en el objetivo.

**Calidad**: VMAF sobre tres ventanas de 2 s (al 20, 50 y 80 % del video) contra el original llevado a 1080p. Si ffmpeg no trae libvmaf, usa SSIM.
