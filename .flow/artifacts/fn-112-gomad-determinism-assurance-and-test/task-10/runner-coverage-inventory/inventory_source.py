import argparse
import collections
import hashlib
import json
import pathlib
import re
import subprocess


ROOT = pathlib.Path(__file__).resolve().parents[5]
OUT = pathlib.Path(__file__).resolve().parent
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62'
BASE = '1af406663d1304453a70b370c22fef92778808c9'
FINGERPRINT = 'fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def source(path):
    physical = ROOT / path
    if physical.is_file():
        return physical.read_bytes(), 'physical'
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT), 'committed-blob'


def functions(path):
    data = path.read_bytes()
    text = data.decode()
    tokens = re.compile(r'//[^\n]*|/\*[\s\S]*?\*/|`[^`]*`|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|[{}]')
    result = {}
    for match in re.finditer(r'^func (\w+)\([^\n]*', text, re.M):
        brace = text.index('{', match.start())
        depth = 0
        end = None
        for token in tokens.finditer(text, brace):
            if token.group() == '{':
                depth += 1
            elif token.group() == '}':
                depth -= 1
                if depth == 0:
                    end = token.end()
                    break
        assert end is not None, path
        first = text.count('\n', 0, match.start()) + 1
        last = text.count('\n', 0, end) + 1
        result[match.group(1)] = {
            'file': str(path.relative_to(ROOT)), 'line': first, 'end_line': last,
            'function_sha256': sha(text[match.start():end].encode()),
            'file_sha256': sha(data),
            'assertion_sites': [first + offset for offset, line in
                                enumerate(text[match.start():end].splitlines())
                                if re.search(r'\bt\.(?:Fatalf?|Errorf?)\(', line)],
        }
    return result


def derive():
    subprocess.run(['git', 'merge-base', '--is-ancestor', BASE, 'HEAD'], cwd=ROOT, check=True)
    changed = subprocess.check_output(['git', 'log', '--format=', '--name-only', BASE + '..HEAD'], cwd=ROOT).decode().splitlines()
    assert all(not path or path.startswith('.flow/') for path in changed), 'Post-base executable source changes require a new source admission'
    receipt_bytes = (PACKET / 'full-ordinary-runner.json').read_bytes()
    receipt = json.loads(receipt_bytes)
    raw = (PACKET / 'full-ordinary-runner.stdout').read_bytes()
    assert sha(raw) == receipt['stdout_sha256']
    assert receipt['exit_code'] == 1 and receipt['terminal'] and not receipt['timed_out']
    manifest_bytes = (PACKET / receipt['source_manifest']).read_bytes()
    manifest = json.loads(manifest_bytes)
    assert sha(manifest_bytes) == receipt['source_manifest_sha256']
    assert sha(json.dumps(manifest, sort_keys=True).encode()) == FINGERPRINT
    availability = collections.Counter()
    sparse = []
    for path, expected in manifest.items():
        data, kind = source(path)
        assert sha(data) == expected, path
        availability[kind] += 1
        if kind != 'physical':
            sparse.append(path)
    events = [json.loads(line) for line in raw.splitlines()]
    counts = collections.Counter()
    failed = []
    for event in events:
        if event.get('Test') and event['Action'] in ('pass', 'fail', 'skip'):
            counts['all_' + event['Action']] += 1
            if '/' not in event['Test']:
                counts['top_' + event['Action']] += 1
                if event['Action'] == 'fail':
                    failed.append(event['Test'])
    assert counts == {'all_pass': 345, 'all_fail': 288, 'all_skip': 12,
                      'top_pass': 82, 'top_fail': 123, 'top_skip': 12}
    definitions = {}
    for path in sorted((ROOT / 'tools/gomad3/runner').glob('*_test.go')):
        for name, record in functions(path).items():
            assert name not in definitions, name
            definitions[name] = record
    rows = []
    classes = collections.Counter()
    for index, name in enumerate(failed):
        outputs = [(line, event) for line, event in enumerate(events, 1)
                   if event['Action'] == 'output' and event.get('Test', '').split('/')[0] == name]
        first_line, first = next((line, event) for line, event in outputs
                                if re.match(r'\s+\w+_test.go:\d+:', event.get('Output', '')))
        all_output = ''.join(event.get('Output', '') for _, event in outputs)
        if 'deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64' in all_output:
            kind = 'explicit-unsupported-host'
        elif '.toolchain/bin/go: no such file or directory' in all_output:
            kind = 'explicit-missing-pinned-go'
        elif 'preparation.stageError{stage:"validation"' in all_output:
            kind = 'validation-stage-with-unprinted-cause'
        elif 'crash helper failed: exit status 1' in all_output:
            kind = 'crash-helper-exit-1'
        else:
            kind = 'preparation-failed-campaign-not-resumable'
        classes[kind] += 1
        rows.append({
            'index': index, 'test': name, 'source': definitions[name],
            'first_observed_failure': {
                'stdout_json_line': first_line, 'test_or_subtest': first['Test'],
                'output': first['Output'], 'presentation_class': kind,
                'cause_claim': 'None. Presentation is retained output; inferred setup causes are separate.',
            },
            'recorded_failed_test_paths': [event['Test'] for event in events
                                           if event['Action'] == 'fail' and event.get('Test', '').split('/')[0] == name],
        })
    bindings = {}
    paths = ['AGENTS.md', 'MILESTONES.md', 'tools/gomad3/README.md',
             '.flow/artifacts/native-scope-transfer-2026-10-07.md',
             '.flow/artifacts/linux-scope-transfer-2026-10-04.md',
             '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.63.md',
             '.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md',
        '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md',
             '.flow/specs/fn-112-gomad-determinism-assurance-and-test.md',
             str((OUT / 'admission.md').relative_to(ROOT))]
    for path in paths:
        data, kind = source(path)
        bindings[path] = {'sha256': sha(data), 'availability': kind}
    return {
        'base_commit': BASE, 'execution_head': receipt['head'],
        'observation': {'command': receipt['command'], 'exit_code': 1,
                        'elapsed_seconds': receipt['elapsed_seconds'], 'new_execution': False,
                        'counts': dict(counts), 'failure_presentations': dict(classes)},
        'raw_binding': {'receipt_sha256': sha(receipt_bytes), 'stdout_sha256': sha(raw),
                        'receipt': str((PACKET / 'full-ordinary-runner.json').relative_to(ROOT)),
                        'stdout': str((PACKET / 'full-ordinary-runner.stdout').relative_to(ROOT))},
        'source_binding': {'manifest_sha256': sha(manifest_bytes), 'fingerprint': FINGERPRINT,
                           'manifest': str((PACKET / receipt['source_manifest']).relative_to(ROOT)),
                           'inputs': len(manifest), 'availability': dict(availability),
                           'committed_only_paths': sparse},
        'consumed_contracts': bindings, 'tests': rows,
    }


parser = argparse.ArgumentParser()
parser.add_argument('--verify', action='store_true')
args = parser.parse_args()
mechanical = derive()
if args.verify:
    inventory = json.loads((OUT / 'inventory.json').read_bytes())
    for field in mechanical:
        if field != 'tests':
            assert inventory[field] == mechanical[field], field
    assert len(inventory['tests']) == 123
    expected_native = {8, 9, 10, 16, 17, 24, 51, 53, 62, 63, 110, 121}
    for observed, row in zip(mechanical['tests'], inventory['tests']):
        for field in observed:
            assert row[field] == observed[field], (row['test'], field)
        for field in ('operations', 'portable_assertions_blocked', 'native_execution_slice', 'ownership'):
            assert row[field], (row['test'], field)
        assert row['native_execution_slice']['seeded_target_intended'] == (row['index'] in expected_native), row['test']
        for rule in row['ownership'].values():
            assert rule in inventory['ownership_rules'], (row['test'], rule)
        if 'InjectionCharacterization' in row['test']:
            assert row['assertion_separation'], row['test']
    print(json.dumps({'verified_tests': 123, 'base_commit': BASE,
                      'bound_inputs': mechanical['source_binding'],
                      'counts': mechanical['observation']['counts'],
                      'inventory_sha256': sha((OUT / 'inventory.json').read_bytes())}, indent=2))
else:
    print(json.dumps(mechanical, indent=2))
