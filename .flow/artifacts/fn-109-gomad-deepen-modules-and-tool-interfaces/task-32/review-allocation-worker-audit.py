import contextlib
import hashlib
import io
import json
from pathlib import Path
import runpy
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
BASE = 'f931c9879e3017b346562667b9f6fbcc4db458ec'
original_path = OUT / 'audit.py'
original_read_text = Path.read_text
source = original_read_text(original_path)
before = '    assert digest(ROOT / path) == expected, path'
after = "    assert (hashlib.sha256(subprocess.check_output(['git', 'show', admission['base_commit'] + ':' + path], cwd=ROOT)).hexdigest() if path == 'AGENTS.md' else digest(ROOT / path)) == expected, path"
assert source.count(before) == 1
assert hashlib.sha256(original_path.read_bytes()).hexdigest() == '99383e78b8b0bdc4c5e8c3909a20123c9ca6253cfac73183286c0efc9a32d202'
historical = subprocess.check_output(['git', 'show', BASE + ':AGENTS.md'], cwd=ROOT)
assert hashlib.sha256(historical).hexdigest() == '1d1647deacf876fd634c64c2760194e210f67198fd31194fd4e464be6bc80f0d'

def historical_read_text(path, *args, **kwargs):
    text = original_read_text(path, *args, **kwargs)
    if path.resolve() == original_path:
        assert text == source
        return text.replace(before, after)
    return text

Path.read_text = historical_read_text
try:
    with contextlib.redirect_stdout(io.StringIO()) as output:
        runpy.run_path(str(OUT / 'allocation-repair-audit.py'))
finally:
    Path.read_text = original_read_text
result = json.loads(output.getvalue())
result['authorized_document_historical_view'] = dict(path='AGENTS.md', git_base=BASE, sha256=hashlib.sha256(historical).hexdigest(), source_substitutions=1, archived_audit_unchanged=True)
print(json.dumps(result, sort_keys=True))
