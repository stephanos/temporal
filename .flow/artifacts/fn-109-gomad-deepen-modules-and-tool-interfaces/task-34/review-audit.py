import ast
import collections
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
ADMISSION = json.loads((PROOF / 'root-admission.json').read_text())
BASE = '608df98bdbf1e079e6db8a87849330797ba34439'
FINAL = 'cd34cd65054126f2d1311db44395354355b45766abdd707210eb8b8ef6dd7c02'
EXPECTED = {'private-mode': 1, 'portable-cli': 34, 'boundaries': 5}
WRITER_NAMES = ['root-admission.json', 'root-scope-gate.py', 'run_checks.py', 'handover.md', 'evidence.json', 'source-check.json', 'lint-delta.json', 'base_commit'] + [f'{p}-{n}.{s}' for p in ['baseline', 'final'] for n in ['private-mode', 'portable-cli', 'boundaries', 'lint', 'errortype', 'gofmt'] for s in ['log', 'receipt.json']]

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

bindings = {}
for node in ast.parse((PROOF / 'run_checks.py').read_text()).body:
    if isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name):
        name = node.targets[0].id
        if name in ['GO_BIN', 'ENV_UNSET']:
            bindings[name] = ast.literal_eval(node.value)
GO_BIN = bindings['GO_BIN']
TOOLS = {GO_BIN + '/go': '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0': 'acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc', '/tmp/fn109-lint-tools.ZdNe1t50/errortype': 'db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc', str(ROOT / '.github/.golangci.yml'): '2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43'}
ENV_SET = {'PATH': GO_BIN + ':/usr/local/bin:/usr/bin:/bin', 'GOENV': 'off', 'GOFLAGS': '', 'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOMAXPROCS': '2'}
ENV_UNSET = bindings['ENV_UNSET']

def freeze():
    protected = {p: sha(ROOT / p) for p in ADMISSION['protected_files']}
    assert len(protected) == 1044
    assert protected == ADMISSION['protected_files']
    assert {p: sha(p) for p in TOOLS} == TOOLS
    assert sha(ROOT / '.flow/tmp/cli-remaining-diagnostics-owner-plan.md') == ADMISSION['owner_plan_sha256']
    assert subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT).decode().strip() == BASE
    assert sha(ROOT / ADMISSION['source_path']) == FINAL
    return {'protected_count': len(protected), 'protected_manifest_sha256': hashlib.sha256(json.dumps(protected, sort_keys=True).encode()).hexdigest(), 'fixture_sha256': FINAL, 'tools': TOOLS, 'head': BASE, 'admission_sha256': sha(PROOF / 'root-admission.json')}

def lint_blocks(path):
    lines = Path(path).read_text().splitlines(keepends=True)
    result = {}
    for i, line in enumerate(lines):
        if re.match(r'^tools/.*\.go:\d+:\d+:', line):
            assert line not in result
            result[line] = ''.join(lines[i:i+3])
    return result

def audit():
    before = freeze()
    old = subprocess.check_output(['git', 'show', BASE + ':' + ADMISSION['source_path']], cwd=ROOT)
    assert hashlib.sha256(old).hexdigest() == ADMISSION['source_before_sha256']
    needle = b'\t\tos.Stdin = original\n\t\treader.Close()\n'
    replacement = b'\t\tos.Stdin = original\n\t\tif err := reader.Close(); err != nil {\n\t\t\tt.Errorf("close coordinator request reader: %v", err)\n\t\t}\n'
    assert old.count(needle) == 1
    assert old.replace(needle, replacement) == (ROOT / ADMISSION['source_path']).read_bytes()
    evidence = json.loads((PROOF / 'evidence.json').read_text())
    source_check = json.loads((PROOF / 'source-check.json').read_text())
    checks = []
    for phase in ['baseline', 'final']:
        for name in ['private-mode', 'portable-cli', 'boundaries', 'lint', 'errortype', 'gofmt']:
            receipt_path = PROOF / f'{phase}-{name}.receipt.json'
            r = json.loads(receipt_path.read_text())
            log = PROOF / f'{phase}-{name}.log'
            assert r['log'] == str(log.relative_to(ROOT)) and r['log_sha256'] == sha(log)
            assert r['cwd'] == str(ROOT / 'tools/gomad3')
            assert r['environment'] == {'set': ENV_SET, 'unset': ENV_UNSET}
            assert r['exit_code'] == (1 if name == 'lint' else 0) and r['signal'] is None
            assert r['stability'] and r['source_before'] == r['source_after']
            src = r['source_before']
            assert src['admission_sha256'] == before['admission_sha256'] and src['protected_count'] == 1044 and src['protected_unchanged']
            assert src['tools'] == TOOLS and src['fixture_sha256'] == (ADMISSION['source_before_sha256'] if phase == 'baseline' else FINAL)
            elapsed = (datetime.datetime.fromisoformat(r['end']) - datetime.datetime.fromisoformat(r['start'])).total_seconds()
            assert elapsed >= 0 and abs(elapsed - r['elapsed_seconds']) < 1
            passes = len(re.findall(r'^--- PASS: ', log.read_text(), re.M)) if name in EXPECTED else None
            if name in EXPECTED:
                assert passes == EXPECTED[name] and log.read_text().endswith('\n') and '\nPASS\n' in log.read_text()
                assert not re.search(r'^(--- FAIL:|--- SKIP:|FAIL\b)', log.read_text(), re.M)
            if name == 'gofmt':
                assert log.stat().st_size == 0
            summary = {'phase': phase, 'check': name, 'receipt': str(receipt_path.relative_to(ROOT)), 'exit_code': r['exit_code'], 'elapsed_seconds': r['elapsed_seconds'], 'top_level_passes': passes}
            assert summary in evidence['commands'] and summary in source_check['checks']
            assert r['command'] == evidence['tests'][len(checks)]
            if phase == 'final':
                prior = json.loads((PROOF / f'baseline-{name}.receipt.json').read_text())
                assert r['command'] == prior['command']
            checks.append(summary)
    a, b = lint_blocks(PROOF / 'baseline-lint.log'), lint_blocks(PROOF / 'final-lint.log')
    resolved = set(a) - set(b)
    assert len(a) == 54 and len(b) == 53 and not set(b) - set(a)
    assert resolved == {'tools/gomad3/cmd/gomad/internal/cli/characterization_test.go:92:15: Error return value of `reader.Close` is not checked (errcheck)\n'}
    assert all(a[k] == b[k] for k in b)
    kinds = collections.Counter(re.search(r'\((\w+)\)\n$', k).group(1) for k in b)
    assert kinds == {'errcheck': 52, 'staticcheck': 1}
    delta = json.loads((PROOF / 'lint-delta.json').read_text())
    assert delta['resolved'] == [x.rstrip('\n') for x in resolved] and delta['introduced'] == [] and delta['remaining'] == dict(kinds)
    after = freeze()
    assert before == after
    return {'before': before, 'after': after, 'exact_fixture_reconstruction': True, 'every_other_byte_preserved': True, 'writer_checks': checks, 'lint': {'baseline': 54, 'final': 53, 'resolved': [x.rstrip('\n') for x in resolved], 'introduced': [], 'retained_blocks_byte_identical': True, 'remaining': dict(kinds)}, 'writer_proof_hashes': {n: sha(PROOF / n) for n in WRITER_NAMES}}

if __name__ == '__main__':
    result = audit()
    if len(sys.argv) > 1 and sys.argv[1] == 'gates':
        result['fresh_checks'] = []
        environment = dict(os.environ)
        for key in ENV_UNSET:
            environment.pop(key, None)
        environment.update(ENV_SET)
        for name in ['private-mode', 'portable-cli', 'boundaries', 'lint', 'errortype', 'gofmt']:
            command = json.loads((PROOF / f'final-{name}.receipt.json').read_text())['command']
            before = freeze()
            log = PROOF / f'review-final-{name}.log'
            assert not log.exists()
            start = datetime.datetime.now(datetime.timezone.utc).isoformat()
            tick = time.monotonic()
            with log.open('xb') as output:
                run = subprocess.run(shlex.split(command), cwd=ROOT / 'tools/gomad3', env=environment, stdout=output, stderr=subprocess.STDOUT, timeout=300)
            end = datetime.datetime.now(datetime.timezone.utc).isoformat()
            after = freeze()
            r = {'command': command, 'cwd': str(ROOT / 'tools/gomad3'), 'environment': {'set': ENV_SET, 'unset': ENV_UNSET}, 'source_before': before, 'source_after': after, 'start': start, 'end': end, 'elapsed_seconds': round(time.monotonic()-tick, 3), 'exit_code': run.returncode, 'log': str(log.relative_to(ROOT)), 'log_sha256': sha(log), 'top_level_passes': len(re.findall(r'^--- PASS: ', log.read_text(), re.M)) if name in EXPECTED else None}
            receipt_path = PROOF / f'review-final-{name}.receipt.json'
            with receipt_path.open('x') as output:
                output.write(json.dumps(r, indent=2) + '\n')
            assert before == after and run.returncode == (1 if name == 'lint' else 0)
            if name in EXPECTED:
                assert r['top_level_passes'] == EXPECTED[name] and '\nPASS\n' in log.read_text()
            if name == 'lint':
                assert lint_blocks(log) == lint_blocks(PROOF / 'final-lint.log')
            if name == 'gofmt':
                assert log.stat().st_size == 0
            result['fresh_checks'].append(r)
            print(name, run.returncode, r['top_level_passes'], r['elapsed_seconds'], flush=True)
        destination = PROOF / 'review-audit-evidence.json'
        assert not destination.exists()
        result['after_gates'] = freeze()
        result['audit_script_sha256'] = sha(__file__)
        with destination.open('x') as output:
            output.write(json.dumps(result, indent=2) + '\n')
    else:
        print(json.dumps({'read_only_audit': 'PASS', 'source_sha256': FINAL, 'protected_count': 1044, 'writer_checks': len(result['writer_checks']), 'lint': result['lint']}, indent=2))
