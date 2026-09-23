# Compara una imagen convertida por `img` contra el ORIGEN decodificado por
# Pillow (un decodificador independiente: libwebp, libjpeg, etc.).
#
#   python pixel_check.py origen.webp convertida.png [--tolerancia N]
#
# Con orígenes sin pérdida (webp lossless, png, gif, bmp) la salida tiene que ser
# idéntica: diferencia máxima 0 en RGB y en alfa. Con orígenes CON pérdida (webp
# lossy, jpg) dos decodificadores pueden diferir en unos pocos niveles por cómo
# reconstruyen el color (el submuestreo de croma no está normado al bit): para
# esos casos está --tolerancia.
import argparse
import sys

import numpy as np
from PIL import Image

p = argparse.ArgumentParser()
p.add_argument("origen")
p.add_argument("convertida")
p.add_argument("--tolerancia", type=int, default=0)
a = p.parse_args()

src = np.asarray(Image.open(a.origen).convert("RGBA"), dtype=np.int16)
out = np.asarray(Image.open(a.convertida).convert("RGBA"), dtype=np.int16)
if src.shape != out.shape:
    print(f"✗ tamaño distinto: origen {src.shape[1]}x{src.shape[0]}, salida {out.shape[1]}x{out.shape[0]}")
    sys.exit(1)

d = np.abs(src - out)
rgb, alpha = int(d[..., :3].max()), int(d[..., 3].max())
distintos = int((d.sum(axis=2) > 0).sum())
total = src.shape[0] * src.shape[1]
ok = rgb <= a.tolerancia and alpha <= a.tolerancia
print(f"{'✓' if ok else '✗'} diferencia máxima RGB {rgb} · alfa {alpha} · "
      f"píxeles distintos {distintos} de {total} (tolerancia {a.tolerancia})")
sys.exit(0 if ok else 1)
