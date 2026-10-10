import collections
import datetime
import difflib
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
WORKSPACE = PRIMARY / '.worktrees/fn-109-75-six-attachments-candidate'
EVIDENCE = WORKSPACE / '.flow/tmp/fn10975-evidence'
BASE = 'cc0c1c90fa5d2ac3f386c7a13fb51637d144c857'
CANDIDATE = '4bb2f25bd7c694aeea5a201def5ffa795e10cc08'
SELECTED = {
    'tools/gomad3/runner/diagnostics_test.go': ('885b3df456c376522b484cfbdaa70e5f70ffe4d77a9c8af1d825740443606b68', 1),
    'tools/gomad3/runner/retention_characterization_test.go': ('c86e26d2c016af55427faea8927c9b224a89778b4b5d70d81dc4257a8163f168', 3),
    'tools/gomad3/runner/inspect_test.go': ('140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df', 2),
}
ASSIGNMENT = 'configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)'
PARENTS = (
    'TestDiagnosticsRetainedWhenSuccessArtifactsAreDiscarded',
    'TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy',
    'TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs',
    'TestOpenReportsSimulationExplorationEvidence',
    'TestOpenReportsSimulationExplorationBoundsAndRemainingWork',
)
PHASE_GATES = ('ordinary', 'focused', 'controls', 'collateral', 'cold-bounds',
               'cold-capacity-seed', 'cold-capacity-choice-exploration',
               'cold-capacity-simulation-exploration', 'aggregate-lint')
GATES = [phase + '-' + gate for phase in ('before', 'after') for gate in PHASE_GATES]
GATES += ['gofmt', 'diff-check', 'host-vet', 'errortype', 'runner-lint', 'analysis-ordinary']
SEALS = {}
HASH_CACHE = {}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def read(path):
    path = Path(path)
    data = path.read_bytes()
    SEALS[str(path)] = digest(data)
    return data


def file_hash(path):
    path = str(path)
    if path not in HASH_CACHE:
        HASH_CACHE[path] = digest(Path(path).read_bytes())
    return HASH_CACHE[path]


def git(*args):
    return subprocess.check_output(['/usr/bin/git', *args], cwd=WORKSPACE)


def source_identity(path):
    path = Path(path)
    if path.is_symlink():
        return {'literal_link': os.readlink(path), 'resolved': str(path.resolve()),
                'target_sha256': file_hash(path) if path.is_file() else 'NON_FILE_TARGET'}
    if path.is_file():
        return file_hash(path)
    if path.is_dir():
        return 'NON_FILE_DIRECTORY_OR_GITLINK; not a consumed file'
    return 'ABSENT_UNMATERIALIZED'


def normalized_settings(settings):
    result = dict(settings)
    result['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)',
                                'go-build<VOLATILE>', result['GOGCCFLAGS'])
    return result


read(__file__)
assert set(git('diff', '--name-only', BASE, CANDIDATE).decode().splitlines()) == set(SELECTED)
reconstruction, line_maps, baseline_bytes = {}, {}, {}
for name, (expected, count) in SELECTED.items():
    before = git('show', BASE + ':' + name)
    after = read(WORKSPACE / name)
    assert digest(before) == expected
    assert after == git('show', CANDIDATE + ':' + name)
    baseline_bytes[str(WORKSPACE / name)] = before
    old, new = before.decode().splitlines(keepends=True), after.decode().splitlines(keepends=True)
    recovered, inserted, mapping = [], [], {}
    for tag, a, b, c, d in difflib.SequenceMatcher(None, old, new, autojunk=False).get_opcodes():
        if tag == 'equal':
            recovered.extend(new[c:d])
            mapping.update({j + 1: i + 1 for i, j in zip(range(a, b), range(c, d))})
        else:
            assert tag == 'insert' and d - c == 1 and new[c].strip() == ASSIGNMENT
            inserted.append(c + 1)
    assert len(inserted) == count and ''.join(recovered).encode() == before
    reconstruction[name] = {'base_sha256': expected, 'candidate_sha256': digest(after),
                            'reconstructed_sha256': digest(''.join(recovered).encode()),
                            'inserted_lines': inserted}
    line_maps[Path(name).name] = mapping

receipts, bindings, logs = {}, {}, {}
for name in GATES:
    receipt = json.loads(read(EVIDENCE / (name + '.json')))
    binding_data = read(EVIDENCE / (name + '-binding.json'))
    binding = json.loads(binding_data)
    log = read(EVIDENCE / (name + '.log'))
    assert digest(binding_data) == receipt['binding_sha256']
    assert digest(log) == receipt['log_sha256']
    assert binding['base'] == BASE and binding['workspace'] == str(WORKSPACE)
    assert receipt['argv'] == binding['argv'] and receipt['cwd'] == binding['cwd']
    assert binding['source_manifest'] == receipt['source_after']
    assert binding['tools_manifest'] == receipt['tools_after']
    assert receipt['source_before_after_equal'] and receipt['tools_before_after_equal']
    assert receipt['all_commands_terminal'] and not receipt['remaining_group_members']
    assert not receipt['timed_out'] and receipt['exit_code'] == receipt['child_returncode']
    assert datetime.datetime.fromisoformat(receipt['start_utc']) < datetime.datetime.fromisoformat(receipt['end_utc'])
    assert 0 <= receipt['elapsed_seconds'] < binding['wall_bound_seconds'] + 30
    wrapper = WORKSPACE / binding['wrapper_invocation_argv'][0]
    assert digest(read(wrapper)) == binding['source_manifest'][str(wrapper)]
    library = wrapper.parent / 'capture_v2.py' if name == 'analysis-ordinary' else wrapper
    assert digest(read(library)) == binding['wrapper_sha256']
    assert binding['source_manifest'][str(library)] == binding['wrapper_sha256']
    for path, identity in binding['source_manifest'].items():
        if name.startswith('before-') and path in baseline_bytes:
            assert identity == digest(baseline_bytes[path])
        else:
            assert source_identity(path) == identity, (name, path)
    for path, identity in binding['tools_manifest'].items():
        if path == 'make_exported_lookup':
            assert identity['SHELL'] == '/bin/sh'
            continue
        assert str(Path(path).resolve()) == identity['resolved']
        assert (os.readlink(path) if Path(path).is_symlink() else None) == identity['link']
        assert file_hash(path) == identity['sha256']
    if receipt['go_settings_probe'] == 'EXECUTED':
        assert normalized_settings(binding['actual_go_settings']) == normalized_settings(receipt['actual_go_settings_after'])
        assert receipt['go_settings_stable_except_numeric_GOGCCFLAGS'] is True
    else:
        assert binding['actual_go_settings'].startswith('NOT_EXECUTED')
        assert receipt['actual_go_settings_after'].startswith('NOT_EXECUTED')
    receipts[name], bindings[name], logs[name] = receipt, binding, log


def observations(name):
    outcomes, diagnostics, package_terminals = {}, collections.defaultdict(list), []
    for raw in logs[name].splitlines():
        event = json.loads(raw)
        key = (event.get('Package'), event.get('Test'))
        if event['Action'] in ('pass', 'fail', 'skip'):
            if key[1]:
                assert key not in outcomes
                outcomes[key] = event['Action']
            else:
                package_terminals.append((key[0], event['Action']))
        if key[1] and event['Action'] == 'output' and event.get('OutputType') in ('error', 'error-continue'):
            diagnostics[key].append(event['Output'])
    assert outcomes and package_terminals
    return outcomes, {key: ''.join(value) for key, value in diagnostics.items()}, package_terminals


def normalize(text, after):
    if after:
        for filename, mapping in line_maps.items():
            text = re.sub('(' + re.escape(filename) + r':)(\d+)(:)',
                          lambda match: match[1] + str(mapping.get(int(match[2]), int(match[2]))) + match[3], text)
    text = re.sub(r'(?<=\)\()0x[0-9a-f]+(?=\))', '0x<POINTER>', text)
    tmp = bindings['before-ordinary']['effective_selected_environment']['TMPDIR']
    text = re.sub(re.escape(tmp) + r'/(Test[^/\n"]*?)[0-9]+/([0-9]{3})(?=/)',
                  lambda match: tmp + '/' + match[1] + '<TEMP>/' + match[2], text)
    return re.sub(r'campaign-[0-9]{8}T[0-9]{6}\.[0-9]+Z-[0-9a-f]{32}', 'campaign-<UTC-NONCE>', text)


tests = {name: observations(name) for name in GATES if name.startswith(('before-', 'after-')) and 'lint' not in name}
left, right = tests['before-ordinary'], tests['after-ordinary']
assert bindings['before-ordinary']['head'] == BASE and bindings['after-ordinary']['head'] == CANDIDATE
for name in ('before-ordinary', 'after-ordinary'):
    assert receipts[name]['argv'] == ['go', 'test', '-tags', 'test_dep', '-count=1', '-json', './runner']
    assert receipts[name]['cwd'] == str(WORKSPACE / 'tools/gomad3') and receipts[name]['exit_code'] == 1
changed = {key: (left[0].get(key), right[0].get(key)) for key in left[0].keys() | right[0].keys() if left[0].get(key) != right[0].get(key)}
assert all(any(key[1] == parent or key[1].startswith(parent + '/') for parent in PARENTS) for key in changed)
assert collections.Counter(changed.values()) == {('fail', 'pass'): 14, (None, 'pass'): 6}
assert not left[0].keys() - right[0].keys()
new_names = {key[1] for key in right[0].keys() - left[0].keys()}
assert new_names == {PARENTS[2] + '/' + jobs + '/' + row for jobs in ('10_jobs', '100_jobs') for row in ('discard', 'success_count', 'success_bytes')}
unselected_diagnostic_changes = []
for key in left[1].keys() | right[1].keys():
    if key not in changed and normalize(left[1].get(key, ''), False) != normalize(right[1].get(key, ''), True):
        unselected_diagnostic_changes.append(key)
assert not unselected_diagnostic_changes
for gate in ('controls', 'collateral'):
    before, after = tests['before-' + gate], tests['after-' + gate]
    assert before[0] == after[0] and before[2] == after[2]
    assert all(normalize(before[1].get(key, ''), False) == normalize(after[1].get(key, ''), True) for key in before[1].keys() | after[1].keys())
for gate in ('focused', 'cold-bounds', 'cold-capacity-seed', 'cold-capacity-choice-exploration', 'cold-capacity-simulation-exploration', 'controls'):
    assert receipts['after-' + gate]['exit_code'] == 0
    assert set(tests['after-' + gate][0].values()) == {'pass'}
for gate in ('gofmt', 'diff-check', 'host-vet', 'errortype', 'analysis-ordinary'):
    assert receipts[gate]['exit_code'] == 0
assert logs['gofmt'] == logs['diff-check'] == b''

lint = {}
for gate in ('before-aggregate-lint', 'after-aggregate-lint', 'runner-lint'):
    lines = logs[gate].decode().splitlines()
    blocks = []
    for index, line in enumerate(lines):
        if re.match(r'^(tools/gomad3/|runner/)[^:]+:\d+:\d+: .+$', line):
            assert index + 2 < len(lines) and '^' in lines[index + 2]
            blocks.append('\n'.join(lines[index:index + 3]))
    lint[gate] = {'count': len(blocks), 'ordered_blocks_sha256': digest('\n'.join(blocks).encode()), 'numeric_exit': receipts[gate]['exit_code']}
    if gate != 'runner-lint':
        assert len(blocks) == 50 and lint[gate]['ordered_blocks_sha256'] == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
        assert 'GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c' in receipts[gate]['argv']
    else:
        assert len(blocks) == 6 and receipts[gate]['exit_code'] == 1

analysis = json.loads(logs['analysis-ordinary'])
for name, (outcomes, diagnostics, _) in tests.items():
    assert {key[1]: value for key, value in outcomes.items()} == analysis['tests'][name]['outcomes']
    assert {key[1]: value for key, value in diagnostics.items()} == analysis['tests'][name]['lossless_diagnostics']
for path, expected in SEALS.items():
    assert digest(Path(path).read_bytes()) == expected, path

print(json.dumps({
    'verification': 'PASS: named source reconstruction, terminal receipts and ordinary comparison only',
    'product_acceptance_pass': False,
    'candidate': CANDIDATE,
    'base': BASE,
    'reconstruction': reconstruction,
    'verified_receipt_count': len(receipts),
    'tests': {name: {'actual_names': len(value[0]), 'counts': dict(collections.Counter(value[0].values())), 'package_terminals': value[2]} for name, value in tests.items()},
    'ordinary_actual_name_union': len(left[0].keys() | right[0].keys()),
    'changed_outcomes': {key[1]: values for key, values in sorted(changed.items())},
    'missing_actual_names': [],
    'unselected_outcome_or_normalized_diagnostic_changes': [],
    'lint': lint,
    'input_seals': SEALS,
    'verifier_sha256': digest(Path(__file__).read_bytes()),
    'verifier_invocation_argv': sys.argv,
    'verifier_cwd': str(Path.cwd()),
    'verifier_python': {'literal': sys.executable, 'resolved': str(Path(sys.executable).resolve()), 'sha256': file_hash(sys.executable)},
    'limitations': 'No Go command or Go-env probe executed by this verifier. Shared caches, nonselected environment, libraries and full tool installations remain nonhermetic. wrapper_sha256 identifies the capture library, while wrapper_invocation_argv identifies the entry script; both are sealed separately. Captured shell literal is a template; non-Go post-helper caption is not proof of a Go-env probe. Receipt process-group emptiness is historical, not continuous global quiescence. Historical task74 deadline, remaining ordinary failures, RED50/RED6, pending standards and native acceptance remain open.',
}, sort_keys=True, indent=2))

