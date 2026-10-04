"""Reaudit frozen review receipts without reinterpreting its generated output."""
import hashlib
from pathlib import Path

OUT = Path(__file__).resolve().parent
original = (OUT / 'review-verify.py').read_bytes()
assert hashlib.sha256(original).hexdigest() == '1db3d76358eebe631fe72cdc0198e55a4df70e85bed1208ea6bad97f2d29222d'
assert hashlib.sha256((OUT / 'review-final-audit.json').read_bytes()).hexdigest() == '42f43b1aa527618d30c03fc863e0b10382130410039ca3823c06771545168e18'
needle = "for path in OUT.glob('review-*.json'):\n"
replacement = "for path in (p for p in OUT.glob('review-*.json') if p.name != 'review-final-audit.json'):\n"
code = original.decode()
assert code.count(needle) == 1
code = code.replace(needle, replacement)
exec(compile(code, str(OUT / 'review-verify.py'), 'exec'),
     {'__name__': '__main__', '__file__': str(OUT / 'review-verify.py')})
