"""Static fn-249 census; no Go/Rust compilation or fixture execution."""
import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / '.flow/artifacts/fn-249'
MODULE = 'go.temporal.io/server/tools/gomad/'
MODEL = 'tools/gomad/internal/model/'
KEPT = {'capture', 'iowire', 'strictjson', 'runtimesource', 'cmd/protocolgen', 'generation/protocol', 'wasi/testdata/runtime_cost'}

def digest(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()

def line(text, offset):
    return text.count('\n', 0, offset) + 1

def collect():
    paths = sorted(subprocess.check_output(['rg', '--files', 'tools/gomad'], cwd=ROOT, text=True).splitlines())
    go = [p for p in paths if p.endswith('.go')]
    packages = sorted({str(Path(p).parent)[len(MODEL):] for p in go if p.startswith(MODEL)})
    imports = []
    for path in go:
        text = (ROOT / path).read_text()
        for block in re.finditer(r'(?m)^import\s+(?:\([\s\S]*?^\)|[^\n]+)', text):
            for item in re.finditer(r'"(go\.temporal\.io/server/tools/gomad/[^"\n]+)"', block.group()):
                imports.append({'file': path, 'line': line(text, block.start()+item.start()), 'package': item[1][len(MODULE):], 'kind': 'test' if path.endswith('_test.go') else 'fixture/tool' if '/testdata/' in path or '/cmd/' in path else 'non-test'})
    rows = []
    for package in packages:
        prefix = MODEL + package + '/'
        files = [p for p in go if str(Path(p).parent) == prefix[:-1]]
        consumers = [r for r in imports if r['package'] == 'internal/model/' + package]
        transitive = set()
        frontier = {str(Path(r['file']).parent) for r in consumers}
        while frontier:
            parent = frontier.pop()
            if parent in transitive:
                continue
            transitive.add(parent)
            relative = parent.removeprefix('tools/gomad/')
            frontier.update(str(Path(r['file']).parent) for r in imports if r['package'] == relative)
        disposition = 'kept-Go' if package in KEPT else 'convert-to-Rust-corpus'
        rows.append(dict(package=package, disposition=disposition, files=files, direct_import_consumers=consumers, transitive_import_consumer_packages=sorted(transitive), same_package_tests=[p for p in files if p.endswith('_test.go')]))
    selected = [p for p in paths if (p.startswith(MODEL) or p.startswith('tools/gomad/internal/backends/wasm/cluster/')) and p.endswith('_test.go')]
    selected += [p for p in paths if p.startswith('tools/gomad/rust/wasmhost/') and p.endswith('.rs') and ('corpus' in p or 'conformance' in p or re.search(r'(?:internal/model|go_pin|go_corpus|GOMAD_GO_GZIP_ORACLE|replay_plan)', (ROOT/p).read_text()))]
    tests, sources = [], []
    for path in sorted(set(selected)):
        text = (ROOT/path).read_text()
        isgo = path.endswith('.go')
        pattern = r'(?m)^func (Test\w+)\(t \*testing\.T\)' if isgo else r'(?m)^\s*#\[test\](?:(?!\bfn\b)[\s\S])*?\bfn\s+(\w+)\s*\('
        matches = list(re.finditer(pattern, text))
        sources.append({'file': path, 'sha256': digest(ROOT/path), 'tests': [m[1] for m in matches], 'role': 'test entrypoints' if matches else 'support/case construction; covered by caller entrypoints'})
        for m in matches:
            # Function termination at its opening indentation, avoids braces in string literals.
            fnpos = text.find('fn ', m.start()) if not isgo else m.start()
            startline = text.rfind('\n', 0, fnpos)+1
            indent = text[startline:fnpos]
            end = re.search(r'(?m)^'+re.escape(indent)+r'}\s*$', text[m.end():])
            finish = m.end()+end.end() if end else len(text)
            body = text[m.end():finish]
            evidence = []
            for i, value in enumerate(body.splitlines()):
                if re.search(r'\b(?:Fatalf?|Errorf?|Equal|NotEqual|NoError|ErrorContains|ErrorIs|True|False|Panics|NotPanics|Len|Empty|NotEmpty|Contains|assert\w*|panic)\s*[!(.]', value):
                    evidence.append({'line': line(text,m.end())+i, 'source': value.strip()})
            pkg = str(Path(path).parent).removeprefix(MODEL)
            disposition = 'kept-Go' if isgo and pkg in KEPT else 'convert-to-Rust-corpus'
            reason = 'Retains production capture/codec/runtime/generator invariants.' if disposition == 'kept-Go' else 'Preserve these assertions against Rust models/linked outcomes and reviewed inputs; conversion is not yet implemented.'
            if 'DriftPinIsCurrent' in m[1] or 'headers_match_the_go_pin' in m[1]:
                disposition = 'obsolete'
                reason = 'Live Go implementation fingerprint becomes obsolete after retirement; keep captured revision provenance and corpus integrity checks.'
            tests.append(dict(file=path, line=line(text,fnpos), test=m[1], disposition=disposition, reason=reason, named_failure='Violation of '+re.sub(r'(?<=[a-z0-9])(?=[A-Z])', ' ', m[1].removeprefix('Test')).replace('_', ' '), assertion_sites=evidence, body_sha256=hashlib.sha256(body.encode()).hexdigest(), delegated_calls=sorted(set(re.findall(r'([A-Za-z_][A-Za-z_0-9:]*)\(', body))) if not evidence else [], subtest_literals=re.findall(r't\.Run\("([^"\n]+)"',body)))
    fixture_paths = [p for p in paths if '/testdata/' in p and p.endswith('.json') and (p.startswith(MODEL) or p.startswith('tools/gomad/rust/wasmhost/') or p.startswith('tools/gomad/internal/backends/wasm/cluster/'))]
    fixtures=[]
    for path in fixture_paths:
        try:
            value=json.loads((ROOT/path).read_text())
        except (ValueError, UnicodeDecodeError):
            value=None
        fixtures.append({'file':path,'sha256':digest(ROOT/path),'case':value.get('name',Path(path).stem) if isinstance(value,dict) else Path(path).stem,'fields':sorted(value) if isinstance(value,dict) else [],'disposition':'convert-to-Rust-corpus','reason':'Retain reviewed input/expected behavior; raw stdio-only fields require explicit translation or obsolescence in task .2/.4.'})
    refs=[]
    for path in paths:
        if Path(path).suffix not in {'.rs','.py','.go','.toml','.md','.txt'} and Path(path).name != 'Makefile':
            continue
        for i,s in enumerate((ROOT/path).read_text(errors='replace').splitlines(),1):
            if re.search(r'internal/model/|GOMAD_WASM_(?:TEST|CONFORMANCE)_HELPER|GOMAD_GO_GZIP_ORACLE',s):
                refs.append({'file':path,'line':i,'source':s.strip()})
    return {'source_revision':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'scope':'Static lexical source census, all build tags; no execution or mutation proof. Transitive sets include test imports and are not production reach claims. Fixture rows include supporting inputs, not only oracle expectations.','packages':rows,'test_sources':sources,'tests':tests,'fixtures':fixtures,'path_and_helper_references':refs}

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--check',action='store_true')
    args=parser.parse_args()
    result=collect()
    path=OUT/'coverage.json'
    if path.exists():
        result['source_revision']=json.loads(path.read_text())['source_revision']
    if args.check:
        old=json.loads(path.read_text())
        result['source_revision']=old['source_revision']
        assert result==old, 'Source census changed: regenerate and review inventory'
        print('coverage matches:',len(result['packages']),'packages,',len(result['test_sources']),'test/support sources,',len(result['tests']),'test functions,',len(result['fixtures']),'JSON fixtures')
    else:
        parts=[]
        for key,value in result.items():
            encoded='[\n'+',\n'.join(json.dumps(row,separators=(',',':')) for row in value)+'\n]' if isinstance(value,list) else json.dumps(value)
            parts.append(json.dumps(key)+':'+encoded)
        path.write_text('{\n'+',\n'.join(parts)+'\n}\n')
        lines=['# Test entrypoint disposition index', '', 'Part of [inventory.md](inventory.md). Source: `'+result['source_revision']+'`. Named contracts are source assertions, not executed mutations. See [coverage.json](coverage.json) for exact assertion sites, body hashes, delegated calls and disposition reasons. Conversion is a future obligation, not completed coverage.', '']
        for src in result['test_sources']:
            lines += ['## '+src['file'], '']
            tests=[t for t in result['tests'] if t['file']==src['file']]
            if not tests:
                lines += ['Supporting case/assertion source; no standalone test entrypoints. Preserve caller coverage during conversion.', '']
                continue
            lines += ['| Test / source line | Disposition | Named failure contract |','| --- | --- | --- |']
            for t in tests:
                lines += ['| `'+t['test']+'`:'+str(t['line'])+' | '+t['disposition']+' | '+t['named_failure']+' |']
            lines += ['']
        (OUT/'tests.md').write_text('\n'.join(lines)+'\n')
        print('wrote',path)
if __name__=='__main__':
    main()
