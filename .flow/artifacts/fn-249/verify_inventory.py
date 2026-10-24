"""Cross-check static coverage against independent rg symbol/file enumeration."""
import hashlib
import json
from pathlib import Path
import re
import subprocess

root = Path(__file__).resolve().parents[3]
out = root / '.flow/artifacts/fn-249'
x = json.loads((out / 'coverage.json').read_text())
files = subprocess.check_output(['rg', '--files', 'tools/gomad/internal/model'], cwd=root, text=True).splitlines()
packages = {str(Path(p).parent).removeprefix('tools/gomad/internal/model/') for p in files if p.endswith('.go')}
assert packages == {p['package'] for p in x['packages']}
result = subprocess.check_output(['rg','-n',r'^func Test\w+\(', 'tools/gomad/internal/model','tools/gomad/internal/backends/wasm/cluster','-g','*_test.go'],cwd=root,text=True)
actual = {(line.split(':',2)[0],re.search(r'func (Test\w+)\(',line)[1]) for line in result.splitlines()}
covered = {(t['file'],t['test']) for t in x['tests'] if t['file'].endswith('.go')}
assert actual == covered, (actual-covered, covered-actual)
rust_files = set(subprocess.check_output(['rg', '--files', 'tools/gomad/rust/wasmhost', '-g', '*.rs'], cwd=root, text=True).splitlines())
assert rust_files == {r['file'] for r in x['test_sources'] if r['file'].endswith('.rs')}, 'Missing Rust module/source outside selected inventory'
index = (out/'tests.md').read_text()
assert not index.endswith('\n\n')
for t in x['tests']:
    assert t['test'] in index and t['named_failure'] and t['reason']
    assert 'body_sha256' not in t and 'delegated_calls' not in t
    assert t['source_span']['end_line'] == len((root/t['file']).read_text().splitlines())
    if t['test'] == 'TestOutputCapacity':
        assert t['disposition'] == 'kept-Go' and t['role'] == 'guest fixture'
    if t['test'].endswith('corpus_headers_match_the_go_pin'):
        assert t['disposition'] == 'convert-to-Rust-corpus'
for src in x['test_sources']:
    text=(root/src['file']).read_text()
    assert hashlib.sha256((root/src['file']).read_bytes()).hexdigest()==src['sha256']
    if src['file'].endswith('.rs'):
        assert len(re.findall(r'(?m)^\s*#\[test\]',text))==len(src['tests']),src['file']
for fixture in x['fixtures']:
    assert hashlib.sha256((root/fixture['file']).read_bytes()).hexdigest()==fixture['sha256']
s=json.loads((out/'stages.json').read_text())
for path,sha in s['sources'].items():
    assert hashlib.sha256((root/path).read_bytes()).hexdigest()==sha
labels={r['label'] for r in s['stages']}
assert len(s['stages'])==len(labels)==96
for r in s['stages']:
    assert set(r['dependencies'])<=labels
assert all(p['disposition'] in {'kept-Go','convert-to-Rust-corpus','obsolete'} for p in x['packages'])
print(f'Independent rg coverage: {len(packages)} packages; {len(actual)} Go tests; all wasmhost Rust sources and their test attributes; {len(x["fixtures"])} fixture hashes; 96 unique stages and valid dependency targets')
