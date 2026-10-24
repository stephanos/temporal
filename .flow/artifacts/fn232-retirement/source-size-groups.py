"""Aggregate source-size.json into the fn-232.13 Go-vs-Rust closeout groups."""
import json, sys
G = 'tools/gomad/'
HOST = ['cmd/gomad/', 'internal/runner/', 'internal/evidence/artifact/', 'internal/evidence/record/',
        'internal/evidence/qualification/', 'internal/host/', 'internal/backends/target/', 'internal/backends/testfixture/',
        'internal/backends/wasm/backend/', 'internal/backends/wasm/cmd/', 'internal/backends/wasm/command/',
        'internal/backends/wasm/installation/', 'internal/backends/wasm/qualification/',
        'internal/backends/wasm/internal/measure/', 'internal/backends/wasm/internal/measurement/',
        'internal/backends/wasm/internal/moduleinfo/', 'internal/backends/wasm/internal/sweep/',
        'internal/backends/wasm/internal/temporalinput/']
def group(path):
    if not path.startswith(G):
        return 'repository-other'
    rel = path[len(G):]
    if rel.startswith('internal/evidence/qualification/corpus/'):
        return 'retained-go-model-oracle-toolchain-support'
    if rel.startswith('internal/backends/native/'):
        return 'go-native-backend (fn-223)'
    if any(rel.startswith(h) for h in HOST) or (rel.rsplit('/', 1)[0] in {'internal/backends', 'internal/backends/wasm'} and rel.endswith('.go')):
        return 'go-host (retired by fn-232)'
    if rel.startswith('internal/backends/wasm/wasi/') or rel.startswith('internal/model/wasi/'):
        return 'go-wasi-reference (moved to internal/model/wasi)'
    if rel.startswith('internal/backends/wasm/wasmhost/'):
        return 'rust-helper (pre-rewrite location)'
    if rel.startswith('rust/wasmhost/'):
        return 'rust-linked-engine'
    if rel.startswith('rust/'):
        return 'rust-host (gomad crate, driver, tools)'
    if rel.startswith('internal/'):
        return 'retained-go-model-oracle-toolchain-support'
    return 'gomad-other (docs, scripts, parity testdata)'
report = json.load(open(sys.argv[1]))
out = {}
for side in ('baseline', 'candidate'):
    agg = {}
    for row in report[side]['files']:
        if 'language' not in row:
            continue
        key = (group(row['path']), row['language'], row['category'])
        a = agg.setdefault(key, [0, 0, 0])
        a[0] += 1; a[1] += row['bytes']; a[2] += row['code_lines']
    out[side] = agg
keys = sorted(set(out['baseline']) | set(out['candidate']))
print('| group | language | category | base files | base bytes | base code | cand files | cand bytes | cand code |')
print('| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |')
for k in keys:
    if k[0] == 'repository-other':
        continue
    b = out['baseline'].get(k, [0, 0, 0]); c = out['candidate'].get(k, [0, 0, 0])
    if max(b + c) == 0:
        continue
    print('| %s | %s | %s | %d | %d | %d | %d | %d | %d |' % (k + tuple(b) + tuple(c)))
print()
print('| group (all languages/categories) | base files | base bytes | base code | cand files | cand bytes | cand code |')
print('| --- | ---: | ---: | ---: | ---: | ---: | ---: |')
for g in sorted({k[0] for k in keys}):
    b = [sum(v[i] for k, v in out['baseline'].items() if k[0] == g) for i in range(3)]
    c = [sum(v[i] for k, v in out['candidate'].items() if k[0] == g) for i in range(3)]
    print('| %s | %d | %d | %d | %d | %d | %d |' % ((g,) + tuple(b) + tuple(c)))
print()
for side in ('baseline', 'candidate'):
    r = report[side]
    print(side, r['revision'], 'repository files', r['repository_files'], 'bytes', r['repository_bytes'], 'paths_sha256', r['paths_sha256'])
print('counter', report['schema'], report['counter_sha256'], report['counter_git_revision'])
