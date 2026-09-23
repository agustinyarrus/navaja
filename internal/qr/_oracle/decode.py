# Decodifica con zxing-cpp cada PNG del manifiesto y compara con el texto original.
import json, sys, os
import zxingcpp
from PIL import Image

out = sys.argv[1]
manifest = json.load(open(os.path.join(out, "manifest.json"), encoding="utf-8"))
ok = bad = 0
fails = []
for e in manifest:
    img = Image.open(os.path.join(out, e["file"]))
    results = zxingcpp.read_barcodes(img)
    got = results[0].text if results else None
    # zxing puede devolver el texto en latin-1/utf-8; normalizamos comparando bytes
    if got == e["text"]:
        ok += 1
    else:
        bad += 1
        fails.append((e["file"], e["level"], f"v{e['version']}", f"mask{e['mask']}",
                      repr(e["text"][:30]), repr(got[:30]) if got else "None"))

print(f"OK={ok}  FALLIDOS={bad}  de {len(manifest)}")
for f in fails:
    print("  FALLO", f)
sys.exit(0 if bad == 0 else 1)
