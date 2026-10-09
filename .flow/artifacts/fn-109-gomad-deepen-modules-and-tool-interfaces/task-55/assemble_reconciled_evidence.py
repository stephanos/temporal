import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BASE = (OUT / 'base_commit').read_text().strip()
CLI = 'tools/gomad3/cmd/gomad/internal/cli/cli.go'
CHAR = 'tools/gomad3/cmd/gomad/internal/cli/characterization_test.go'
TEST = 'tools/gomad3/cmd/gomad/internal/cli/replay_output_test.go'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

before, after = original(CLI), (ROOT / CLI).read_bytes()
unchecked = b'\t\tfmt.Fprintf(stdout, "gomad: verified %s\\n", result.Artifact.Path)\n'
checked = b'\t\tif _, err := fmt.Fprintf(stdout, "gomad: verified %s\\n", result.Artifact.Path); err != nil {\n\t\t\treturn 3\n\t\t}\n'
assert after.count(checked) == 1 and after.replace(checked, unchecked) == before
old_char, new_char = original(CHAR), (ROOT / CHAR).read_bytes()
old_row = b'\t\t// Verification-only replay does not check its output write.\n\t\t{"replay verification", func(stdout, stderr io.Writer) int {\n\t\t\treturn runReplayWith([]string{"--verify-only", "/artifact"}, stdout, stderr, replay)\n\t\t}, 0},\n'
new_row = b'\t\t{"replay verification", func(stdout, stderr io.Writer) int {\n\t\t\treturn runReplayWith([]string{"--verify-only", "/artifact"}, stdout, stderr, replay)\n\t\t}, 3},\n'
assert new_char.count(new_row) == 1 and new_char.replace(new_row, old_row) == old_char
initial = json.loads((OUT / 'evidence.json').read_text())
prior = json.loads((OUT.parent / 'task-54/source-proof.json').read_text())
assert prior['candidate_sha256'] == digest(before)
preserved = {}
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0')
for value in paths:
    if value and value.decode() not in [CLI, CHAR]:
        path = value.decode()
        data = (ROOT / path).read_bytes()
        assert data == original(path), path
        preserved[path] = digest(data)
turbo = {path: digest((ROOT / path).read_bytes()) for path in prior['protected_turbo_sha256']}
assert turbo == prior['protected_turbo_sha256']
assert not subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT).strip()

def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    tests = [row for row in events if row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(row['Action'] for row in tests)), 'failed_tests': [row['Test'] for row in tests if row['Action'] == 'fail'],
            'package_results': [{key: row[key] for key in ['Package', 'Action', 'Elapsed'] if key in row} for row in events if not row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]}

def lint_delta(name, counts):
    pattern = r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)'
    old_rows = re.findall(pattern, (OUT.parent / ('task-54/' + name + '.stdout')).read_text())
    new_rows = re.findall(pattern, (OUT / ('reconciled-' + name + '.stdout')).read_text())
    keys = lambda rows: collections.Counter((path, column, message, statement, caret) for path, _, column, message, statement, caret in rows)
    old_keys, new_keys = keys(old_rows), keys(new_rows)
    removed, added = old_keys - new_keys, new_keys - old_keys
    assert (len(old_rows), len(new_rows), sum(removed.values()), sum(added.values())) == (*counts, 1, 0)
    site = next(iter(removed))
    assert site[0] == CLI and 'not checked' in site[2] and 'gomad: verified %s' in site[3]
    return {'before': len(old_rows), 'after': len(new_rows), 'removed': 1, 'added': 0, 'removed_diagnostic': list(site),
            'residual_path_column_message_statement_caret_multiset_preserved': True,
            'baseline_receipt': '../task-54/' + name + '.json', 'candidate_receipt': 'reconciled-' + name + '.json'}

names = ['focused', 'full-ordinary-cli', 'affected-vet', 'standalone-errortype', 'affected-configured-lint',
         'architecture-source-sets', 'runner-ownership', 'generated-validation', 'format-check', 'make-fast-task-base', 'make-gomad-original-base']
receipts = {}
for name in names:
    record = json.loads((OUT / ('reconciled-' + name + '.json')).read_text())
    assert record['terminal'] and record['source_unchanged'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / ('reconciled-' + name + '.' + suffix)).read_bytes()) == record[suffix + '_sha256'], name
    receipts[name] = record
candidate = receipts['focused']
assert all(row['source_before_sha256'] == candidate['source_before_sha256'] and row['tools'] == candidate['tools'] for row in receipts.values())
focused, ordinary = observations('reconciled-focused'), observations('reconciled-full-ordinary-cli')
assert focused['counts'].get('fail', 0) == 0 and focused['counts']['pass'] > 0
assert ordinary['counts'] == {'pass': 432, 'fail': 3}
assert ordinary['failed_tests'] == initial['ordinary_inherited_failures']
proof = {'base_commit': BASE, 'production_path': CLI, 'base_sha256': digest(before), 'candidate_sha256': digest(after),
         'complete_original_cli_bytes_reconstructed': True, 'single_check_failure_status': 3,
         'characterization_path': CHAR, 'characterization_base_sha256': digest(old_char), 'characterization_candidate_sha256': digest(new_char),
         'complete_original_characterization_bytes_reconstructed': True, 'only_fixture_delta': 'replay verification status0→3 and removal of its immediately owning obsolete comment',
         'new_test_sha256': digest((ROOT / TEST).read_bytes()), 'other_sources_and_old_tests_preserved_count': len(preserved),
         'preserved_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()), 'protected_turbo_sha256': turbo,
         'lint': {'affected': lint_delta('affected-configured-lint', (2, 1)), 'original_base': lint_delta('make-gomad-original-base', (96, 95))}}
evidence = {
    'task': 'fn-109.55', 'status': 'in_progress', 'base_commit': BASE, 'branch': 'gomad', 'commits': [], 'prs': [],
    'commit_owner': 'root; worker leaves changes unstaged/uncommitted',
    'initial_packet': {'evidence': 'evidence.json', 'evidence_sha256': digest((OUT / 'evidence.json').read_bytes()), 'proof': 'source-proof.json', 'proof_sha256': digest((OUT / 'source-proof.json').read_bytes()), 'source_sha256': initial['frozen_source_sha256'], 'ordinary_counts': initial['full_ordinary_cli']['counts']},
    'predecessor': '../task-54/handover.md, source-proof.json and independent-review.md',
    'baseline': initial['baseline'], 'tests': [row['command'] for row in receipts.values()],
    'gates': [{'receipt': 'reconciled-' + name + '.json', **{key: receipts[name][key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_before_sha256', 'terminal', 'source_unchanged']}} for name in names],
    'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'], 'environment': candidate['environment'],
    'gate_runner_sha256': digest((OUT / 'run_gate.py').read_bytes()), 'source_proof': 'reconciled-source-proof.json', 'lint': proof['lint'],
    'focused': focused, 'full_ordinary_cli': ordinary,
    'controls': initial['controls'], 'public_path': initial['public_path'],
    'retained_probe_failures': 'See immutable initial evidence.json and focused-before-public/focused-before-final/public-fixture-diagnosis receipts; public probe never supplied success-path/native coverage.',
    'ordinary_fixture_reconciliation': 'Root admission covers only expected datum0→3 and its immediately owning obsolete comment. Initial full ordinary receipt retained430passes/5fails; final432passes/3fails, zero skips. The direct mismatch and its parent vanish.',
    'ordinary_inherited_failures': initial['ordinary_inherited_failures'], 'native_command_failure': initial['native_command_failure'],
    'integrated_errortype': initial['integrated_errortype'], 'acceptance': initial['acceptance'],
    'routing': initial['routing'], 'review': initial['review'], 'native': initial['native'], 'all_handles_terminal': True,
}
for name, value in [('reconciled-source-proof.json', proof), ('reconciled-evidence.json', evidence)]:
    path = OUT / name
    if path.exists():
        raise SystemExit('Refusing to overwrite retained evidence')
    path.write_text(json.dumps(value, indent=2) + '\n')
print(json.dumps({'focused': focused, 'ordinary': ordinary, 'lint': proof['lint'], 'source': evidence['frozen_source_sha256']}))
