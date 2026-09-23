# Estrés de la conversión de WebP con pérdida: genera casos con Pillow (tamaños
# impares y mínimos, ruido de color puro, con y sin alfa), los convierte con
# img.exe y compara contra lo que decodifica libwebp (vía Pillow), píxel a píxel.
# Uso: python webp_stress.py <img.exe> <carpeta de trabajo>
import os
import random
import subprocess
import sys

import numpy as np
from PIL import Image

exe, work = sys.argv[1], sys.argv[2]
os.makedirs(work, exist_ok=True)
rng = np.random.default_rng(20260923)

def noise(w, h, alpha):
    arr = rng.integers(0, 256, size=(h, w, 4 if alpha else 3), dtype=np.uint8)
    return Image.fromarray(arr, "RGBA" if alpha else "RGB")

def blocks(w, h):
    img = Image.new("RGB", (w, h))
    px = img.load()
    for _ in range(40):
        x0, y0 = random.randrange(w), random.randrange(h)
        c = tuple(random.randrange(256) for _ in range(3))
        for y in range(y0, min(h, y0 + random.randrange(1, h + 1))):
            for x in range(x0, min(w, x0 + random.randrange(1, w + 1))):
                px[x, y] = c
    return img

random.seed(7)
cases = []
for w, h in [(1, 1), (1, 2), (2, 1), (2, 2), (3, 3), (17, 13), (64, 48), (333, 201)]:
    cases.append((f"ruido_{w}x{h}", noise(w, h, False)))
    cases.append((f"ruido_alfa_{w}x{h}", noise(w, h, True)))
cases.append(("bloques_320x240", blocks(320, 240)))
cases.append(("bloques_impar_161x99", blocks(161, 99)))

fails = 0
for name, img in cases:
    src = os.path.join(work, name + ".webp")
    img.save(src, "WEBP", quality=80, method=4)  # CON pérdida (VP8, + ALPH si hay alfa)
    out = os.path.join(work, name + ".png")
    subprocess.run([exe, src, "--out-dir", work, "--force", "--no-color"], capture_output=True)
    ref = np.asarray(Image.open(src).convert("RGBA"), dtype=np.int16)
    got = np.asarray(Image.open(out).convert("RGBA"), dtype=np.int16)
    if ref.shape != got.shape:
        fails += 1
        print(f"  ✗ {name}: tamaño {got.shape} ≠ {ref.shape}")
        continue
    d = np.abs(ref - got)
    if d.max() == 0:
        print(f"  ✓ {name}: idéntico a libwebp")
    else:
        fails += 1
        print(f"  ✗ {name}: diferencia máx RGB {int(d[..., :3].max())} alfa {int(d[..., 3].max())}, "
              f"{int((d.sum(axis=2) > 0).sum())} píxeles distintos")

print()
print(f"  resultado: {len(cases) - fails} de {len(cases)} idénticos a libwebp")
sys.exit(0 if fails == 0 else 1)
