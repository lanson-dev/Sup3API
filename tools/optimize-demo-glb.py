"""Shrink Tripo demo textures without changing geometry, skin or materials.

Usage: python tools/optimize-demo-glb.py input.glb output.glb
Requires Pillow. Supports Tripo exports with image views before mesh data.
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
images = sorted(doc['images'], key=lambda image: image['bufferView'])
assert [image['bufferView'] for image in images] == list(range(len(images)))
color_images = {
    doc['textures'][material['pbrMetallicRoughness']['baseColorTexture']['index']]['source']
    for material in doc['materials']
    if 'baseColorTexture' in material.get('pbrMetallicRoughness', {})
}
old_end = 0
packed_images = bytearray()
for image in images:
    view = doc['bufferViews'][image['bufferView']]
    offset = view.get('byteOffset', 0)
    assert view['buffer'] == 0 and offset == old_end
    old_end = (offset + view['byteLength'] + 3) // 4 * 4
    atlas = Image.open(io.BytesIO(binary[offset:offset + view['byteLength']])).convert('RGB')
    color = doc['images'].index(image) in color_images
    limit = 1024 if color else 512
    atlas.thumbnail((limit, limit), Image.Resampling.LANCZOS)
    output = io.BytesIO()
    # Keep data-map channels lossless after resizing; JPEG only for color.
    atlas.save(output, format='JPEG' if color else 'PNG', **({'quality': 82, 'optimize': True} if color else {'optimize': True}))
    payload = output.getvalue()
    view.update(byteOffset=len(packed_images), byteLength=len(payload))
    image['mimeType'] = 'image/jpeg' if color else 'image/png'
    packed_images.extend(payload + b'\0' * (-len(payload) % 4))
shift = len(packed_images) - old_end
for other in doc['bufferViews'][len(images):]:
    # The fallback buffer is virtual; only physical buffer 0 offsets move.
    for item in [other, other.get('extensions', {}).get('EXT_meshopt_compression', {})]:
        if item.get('buffer') == 0:
            offset = item.get('byteOffset', 0)
            assert offset >= old_end, 'Overlapping image/mesh data is unsupported'
            item['byteOffset'] = offset + shift
doc['buffers'][0]['byteLength'] += shift
packed = packed_images + binary[old_end:]
assert packed[len(packed_images):] == binary[old_end:]  # Mesh + bind data unchanged.
metadata = json.dumps(doc, separators=(',', ':')).encode()
metadata += b' ' * (-len(metadata) % 4)
result = (struct.pack('<4sII', b'glTF', 2, 28 + len(metadata) + len(packed))
          + struct.pack('<II', len(metadata), 0x4E4F534A) + metadata
          + struct.pack('<II', len(packed), 0x004E4942) + packed)
assert len(result) < len(raw), 'No size improvement; keep the original'
destination.parent.mkdir(parents=True, exist_ok=True)
destination.write_bytes(result)
print(f'{source.name}: {len(raw):,} -> {len(result):,} bytes; {len(images)} textures, color 1024px / data 512px')
