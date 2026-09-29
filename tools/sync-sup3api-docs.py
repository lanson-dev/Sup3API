"""Publish the canonical asset schema with the static customer documentation."""
import argparse
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--check', action='store_true')
args = parser.parse_args()
source = root / 'docs/sup3/openapi.json'
target = root / 'frontend/public/docs/assets.openapi.json'
data = source.read_bytes()
json.loads(data)
if args.check:
    if not target.exists() or target.read_bytes() != data:
        raise SystemExit('Sup3API schema is stale: run python tools/sync-sup3api-docs.py')
    print('Sup3API schema is synchronized.')
else:
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(data)
    print('Published Sup3API asset schema.')

home = (root / 'frontend/src/content/sup3api-home.html').read_bytes()
standalone = root / 'frontend/public/sup3api.html'
if args.check:
    if not standalone.exists() or standalone.read_bytes() != home:
        raise SystemExit('Sup3API standalone homepage is stale: run python tools/sync-sup3api-docs.py')
else:
    standalone.write_bytes(home)

# Legacy standalone URL remains an alias of the current site.
legacy = root / "frontend/public/agraphs.html"
if args.check:
    if legacy.read_bytes() != home:
        raise SystemExit("Legacy homepage alias is stale")
else:
    legacy.write_bytes(home)
