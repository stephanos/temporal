import collections
import difflib
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[4]
OUT = pathlib.Path(__file__).resolve().parent
HEAD = '95354f677162df9cd76383569b11fe1f902870cd'
FINGERPRINT = 'fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297'
GATES = {
    'focused-combined': 0, 'full-ordinary-runner': 1, 'affected-vet': 0,
    'standalone-errortype': 0, 'architecture-source-sets': 0,
    'runner-ownership': 0, 'generated-validation': 0, 'format-check': 0,
    'affected-configured-lint': 1, 'make-fast-admission-base': 0,
    'make-gomad-original-base': 2,
}
PRODUCTS = {
    'runner/runner_test.go': '7357c8cb3074ffa2acb312c59522519cacdcf92b4585930ec3aba17bce533827',
    'runner/progress_start_test.go': '947e02fca321e403671e07465bae24ba80de7ed2b2f600fc03ecda9bd728e971',
    'runner/internal/execution/process_test.go': '4a94563b09e82dec418088309cc4ae01f49f7899389f4c0d184ecb0e6579bb59',
    'runner/internal/execution/process_fixture_output_test.go': '94b7fb2073c75a46b0835cc2620fc3c522c7d945bf7b6e5d1686c84a58e16572',
    'toolchain/build_test.go': 'ba64e85358a6993e669a5a1925ab0cee8ece625fb43df987b9bdeca437b2cd15',
}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def check(condition, message):
    if not condition:
        raise SystemExit(message)


def blob(path):
    return subprocess.check_output(['git', 'show', HEAD + ':' + path], cwd=ROOT)


def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines()]
    counts = collections.Counter()
    for event in events:
        if event.get('Test') and event['Action'] in ['pass', 'fail', 'skip']:
            counts['all_' + event['Action']] += 1
            if '/' not in event['Test']:
                counts['top_' + event['Action']] += 1
    return dict(counts), events


def lint_blocks(data):
    lines = data.decode().splitlines()
    blocks = {}
    pattern = re.compile(r'^(tools/gomad3/[^:]+):(\d+):(\d+): (.+)$')
    for index, line in enumerate(lines):
        match = pattern.match(line)
        if match:
            path, number, column, message = match.groups()
            key = (path, int(number), column, message, lines[index + 1], lines[index + 2])
            check(key not in blocks, 'Duplicate lint block')
            blocks[key] = int(number)
    return blocks


manifest_path = OUT / ('sources-' + FINGERPRINT + '.json')
manifest_bytes = manifest_path.read_bytes()
manifest = json.loads(manifest_bytes)
check(sha(json.dumps(manifest, sort_keys=True).encode()) == FINGERPRINT, 'Manifest fingerprint mismatch')
for path, expected in manifest.items():
    check(sha((ROOT / path).read_bytes()) == expected, 'Changed source ' + path)
for path, expected in PRODUCTS.items():
    check(sha((ROOT / 'tools/gomad3' / path).read_bytes()) == expected, 'Product mismatch ' + path)

receipts = {}
for name, expected_exit in GATES.items():
    record = json.loads((OUT / (name + '.json')).read_bytes())
    check(record['head'] == HEAD, 'Execution HEAD mismatch ' + name)
    check(record['terminal'] and not record['timed_out'], 'Incomplete command ' + name)
    check(record['exit_code'] == expected_exit, 'Unexpected exit ' + name)
    check(record['source_unchanged'] and record['orchestration_unchanged'], 'Changed gate inputs ' + name)
    check(record['source_before_sha256'] == record['source_after_sha256'] == FINGERPRINT, 'Source binding ' + name)
    check(record['source_manifest_sha256'] == sha(manifest_bytes), 'Manifest hash ' + name)
    check(record['source_manifest'] == manifest_path.name, 'Manifest name ' + name)
    check(record['orchestration_before'] == record['orchestration_after'], 'Orchestration binding ' + name)
    for path, expected in record['orchestration_before'].items():
        check(sha((OUT / path).read_bytes()) == expected, 'Changed orchestration ' + path)
    check(record['gate_runner_sha256'] == record['orchestration_before']['run_gate.py'], 'Runner binding ' + name)
    for path, expected in record['tools'].items():
        check(sha(pathlib.Path(path).read_bytes()) == expected, 'Tool mismatch ' + path)
    for stream in ['stdout', 'stderr']:
        check(sha((OUT / (name + '.' + stream)).read_bytes()) == record[stream + '_sha256'], 'Raw hash ' + name)
    receipts[name] = {'exit_code': record['exit_code'], 'elapsed_seconds': record['elapsed_seconds']}

baseline_root = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-60/'
baseline_record = json.loads(blob(baseline_root + 'baseline-original-lint.json'))
baseline_raw = blob(baseline_root + 'baseline-original-lint.stdout')
check(sha(baseline_raw) == baseline_record['stdout_sha256'], 'Baseline lint hash')
check(baseline_record['tools'] == json.loads((OUT / 'make-gomad-original-base.json').read_bytes())['tools'], 'Baseline tool identity')
baseline = lint_blocks(baseline_raw)
candidate = lint_blocks((OUT / 'make-gomad-original-base.stdout').read_bytes())
check(len(baseline) == 60 and len(candidate) == 53, 'Lint count')
mapped = {}
for path in {key[0] for key in candidate}:
    original = subprocess.check_output(['git', 'show', '7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2:' + path], cwd=ROOT).decode().splitlines()
    current = (ROOT / path).read_text().splitlines()
    positions = {}
    for match in difflib.SequenceMatcher(None, original, current, autojunk=False).get_matching_blocks():
        positions.update({match.b + offset + 1: match.a + offset + 1 for offset in range(match.size)})
    for key in candidate:
        if key[0] == path:
            check(key[1] in positions, 'Residual source line changed ' + path)
            mapped[key] = (path, positions[key[1]], *key[2:])
check(not set(mapped.values()) - baseline.keys(), 'Introduced lint block')
removed = baseline.keys() - set(mapped.values())
check(len(removed) == 7, 'Removal count')
check(sum(key[0] == 'tools/gomad3/runner/internal/execution/process_test.go' and key[3].endswith('(errcheck)') for key in removed) == 6, 'Child-output removals')
check(sum(key[0] == 'tools/gomad3/toolchain/build_test.go' and key[3].endswith('(forbidigo)') for key in removed) == 1, 'Sleep removal')

full, events = observations('full-ordinary-runner')
check(full == {'all_pass': 345, 'top_pass': 82, 'all_fail': 288, 'top_fail': 123, 'all_skip': 12, 'top_skip': 12}, 'Runner outcome counts')
check(events[-1]['Action'] == 'fail' and 'Test' not in events[-1], 'Missing terminal package failure')
check(not any('panic: test timed out' in event.get('Output', '') for event in events), 'Diagnostic test timeout')
focused, _ = observations('focused-combined')
check(focused == {'all_pass': 47, 'top_pass': 15}, 'Focused outcome counts')
check(len(lint_blocks((OUT / 'affected-configured-lint.stdout').read_bytes())) == 17, 'Affected lint count')
print(json.dumps({
    'execution_head': HEAD, 'source_fingerprint': FINGERPRINT,
    'source_file_count': len(manifest), 'receipts': receipts,
    'full_runner': full, 'focused': focused,
    'lint': {'baseline': 60, 'candidate': 53, 'removed': 7, 'introduced': 0, 'affected': 17,
             'line_mappings': [{'path': key[0], 'baseline_line': mapped[key][1], 'candidate_line': key[1]} for key in sorted(candidate) if mapped[key][1] != key[1]]},
}, indent=2))
