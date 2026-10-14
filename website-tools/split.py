# Split tall screenshots into readable chunks: python3 split.py <png> [chunk_height]
import sys
from PIL import Image
p = sys.argv[1]; ch = int(sys.argv[2]) if len(sys.argv) > 2 else 1600
im = Image.open(p); w, h = im.size
n = (h + ch - 1) // ch
for i in range(n):
    im.crop((0, i * ch, w, min(h, (i + 1) * ch))).save(p.replace('.png', '_%d.png' % i))
print(p, im.size, n)
