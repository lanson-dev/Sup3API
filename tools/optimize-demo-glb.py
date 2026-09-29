"""Shrink a Tripo demo's color atlas without changing geometry, skin or materials.

Usage: python tools/optimize-demo-glb.py input.glb output.glb
Requires Pillow. Only supports Tripo's single-image, first-buffer-view exports.
"""
import io
import json
import struct
import sys
from pathlib import Path

from PIL import Image

source, destination = map(Path, sys.argv[1:3])
raw = source.read_bytes()
assert struct.unpack_from('<4sII', raw) == (b'glTF', 2, len(raw))
json_length, chunk = struct.unpack_from('<II', raw, 12)
assert chunk == 0x4E4F534A
doc = json.loads(raw[20:20 + json_length])
binary_length, chunk = struct.unpack_from('<II', raw, 20 + json_length)
assert chunk == 0x004E4942
binary = raw[28 + json_length:]
assert len(binary) == binary_length
assert len(doc['images']) == 1 and doc['images'][0]['bufferView'] == 0
view = doc['bufferViews'][0]
assert view['buffer'] == 0 and view.get('byteOffset', 0) == 0
old_end = (view['byteLength'] + 3) // 4 * 4
atlas = Image.open(io.BytesIO(binary[:view['byteLength']])).convert('RGB')
atlas.thumbnail((1024, 1024), Image.Resampling.LANCZOS)
output = io.BytesIO()
atlas.save(output, format='JPEG', quality=82, optimize=True)
image = output.getvalue()
new_image = image + b'\0' * (-len(image) % 4)
shift = len(new_image) - old_end
for other in doc['bufferViews'][1:]:
    # The fallback buffer is virtual; only physical buffer 0 offsets move.
    for item in [other, other.get('extensions', {}).get('EXT_meshopt_compression', {})]:
        if item.get('buffer') == 0:
            offset = item.get('byteOffset', 0)
            assert offset >= old_end, 'Overlapping image/mesh data is unsupported'
            item['byteOffset'] = offset + shift
view['byteLength'] = len(image)
doc['images'][0]['mimeType'] = 'image/jpeg'
doc['buffers'][0]['byteLength'] += shift
packed = new_image + binary[old_end:]
assert packed[len(new_image):] == binary[old_end:]  # Mesh + bind data unchanged.
metadata = json.dumps(doc, separators=(',', ':')).encode()
metadata += b' ' * (-len(metadata) % 4)
result = (struct.pack('<4sII', b'glTF', 2, 28 + len(metadata) + len(packed))
          + struct.pack('<II', len(metadata), 0x4E4F534A) + metadata
          + struct.pack('<II', len(packed), 0x004E4942) + packed)
assert len(result) < len(raw), 'No size improvement; keep the original'
destination.parent.mkdir(parents=True, exist_ok=True)
destination.write_bytes(result)
print(f'{source.name}: {len(raw):,} -> {len(result):,} bytes; atlas {atlas.size}')
