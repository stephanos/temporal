import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/tmp/gomad-fn109-parallel.pmgezCtg/task-59').resolve()
OUT = pathlib.Path(__file__).resolve().parent
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
BASE = (ROOT / '.flow/tmp/base_commit').read_text().strip()
INSERTIONS = {
    'tools/gomad3/runner/choice_exploration_divergence_test.go': b'\t\t\t\tcase choice.DivergenceAlternativeSet, choice.DivergenceTapeExhausted, choice.DivergenceIdentityMissing, choice.DivergenceIdentityDuplicate, choice.DivergenceAlternativeCapacity, choice.DivergenceObservation:\n',
    'tools/gomad3/runner/completion_characterization_test.go': b'\tconfig.WorldTransitionLimit = 1 << 20\n\tswitch strategy {\n\tcase StrategySeed:\n',
}

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

files, preserved = {}, {}
for path, insertion in INSERTIONS.items():
    before, after = original(path), (ROOT / path).read_bytes()
    assert after.count(insertion) == 1
    replacement = b'' if 'divergence' in path else insertion.replace(b'\tcase StrategySeed:\n', b'')
    assert after.replace(insertion, replacement, 1) == before, path
    files[path] = {'base_sha256': digest(before), 'candidate_sha256': digest(after), 'complete_original_bytes_reconstructed': True}
for value in subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0'):
    if value and value.decode() not in INSERTIONS:
        path = value.decode()
        data = (ROOT / path).read_bytes()
        assert data == original(path), path
        preserved[path] = digest(data)
assert not subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT).strip()
assert set(subprocess.check_output(['git', 'diff', '--name-only'], cwd=ROOT, text=True).splitlines()) == set(INSERTIONS)

def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    outcomes = [row for row in events if row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(row['Action'] for row in outcomes)),
            'top_level_counts': dict(collections.Counter(row['Action'] for row in outcomes if '/' not in row['Test'])),
            'failed_top_level_tests': [row['Test'] for row in outcomes if row['Action'] == 'fail' and '/' not in row['Test']],
            'failed_test_names_sha256': digest(json.dumps([row['Test'] for row in outcomes if row['Action'] == 'fail']).encode()),
            'skipped_tests': [row['Test'] for row in outcomes if row['Action'] == 'skip']}

names = ['focused-before', 'focused-after', 'affected-vet', 'standalone-errortype', 'architecture-source-sets', 'runner-ownership',
         'generated-validation', 'format-check', 'affected-configured-lint', 'make-fast-task-base', 'make-gomad-original-base']
receipts = {}
for name in names:
    path = OUT / (name + '.json')
    if not path.exists():
        continue
    record = json.loads(path.read_text())
    assert record['terminal'] and record['source_unchanged'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / (name + '.' + suffix)).read_bytes()) == record[suffix + '_sha256'], name
    receipts[name] = record
baseline = receipts['focused-before']
candidate = receipts.get('focused-after')
if candidate:
    assert all(row['source_before_sha256'] == candidate['source_before_sha256'] for name, row in receipts.items() if name != 'focused-before')
    assert observations('focused-before') == observations('focused-after')
assert all(row['tools'] == baseline['tools'] for row in receipts.values())
binding_paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
                                        '.github/workflows/gomad3.yml', '.github/.golangci.yml',
                                        'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
binding_files = {value.decode(): digest((ROOT / value.decode()).read_bytes()) for value in binding_paths if value}
if candidate:
    assert digest(json.dumps(binding_files, sort_keys=True).encode()) == candidate['source_before_sha256']
for path in INSERTIONS:
    binding_files[path] = digest(original(path))
assert digest(json.dumps(binding_files, sort_keys=True).encode()) == baseline['source_before_sha256']
pattern = r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)'
old_rows = re.findall(pattern, (OUT.parent / 'task-56/corrected-make-gomad-original-base.stdout').read_text())
lint = {'baseline_receipt': '../task-56/corrected-make-gomad-original-base.json', 'baseline_findings': len(old_rows)}
if 'make-gomad-original-base' in receipts:
    new_rows = re.findall(pattern, (OUT / 'make-gomad-original-base.stdout').read_text())
    removed = collections.Counter(old_rows) - collections.Counter(new_rows)
    added = collections.Counter(new_rows) - collections.Counter(old_rows)
    assert (len(old_rows), len(new_rows), sum(removed.values()), sum(added.values())) == (80, 78, 2, 0)
    assert all(row[0] in INSERTIONS and '(exhaustive)' in row[3] for row in removed)
    lint.update({'candidate_receipt': 'make-gomad-original-base.json', 'candidate_findings': 78,
                 'removed': 2, 'added': 0, 'removed_diagnostics': list(removed),
                 'residual_full_diagnostic_blocks_including_line_numbers_preserved': True})
    lint['integrated_errortype'] = 'Unreached after configured golangci failure; standalone affected observation retained separately.'
    scoped = re.findall(pattern, (OUT / 'affected-configured-lint.stdout').read_text())
    old_scoped = [row for row in old_rows if row[0].startswith('tools/gomad3/runner/') and '/runner/internal/' not in row[0]]
    assert (len(old_scoped), len(scoped)) == (17, 15)
    assert collections.Counter(old_scoped) - collections.Counter(scoped) == removed
    assert not collections.Counter(scoped) - collections.Counter(old_scoped)
    lint['affected'] = {'before': 17, 'after': 15, 'removed': 2, 'added': 0,
                        'baseline': 'Retained actual original-base RED80 root Runner subset; no separate affected baseline run.',
                        'candidate_receipt': 'affected-configured-lint.json', 'all_residual_full_blocks_preserved': True}
    lint['fast'] = 'Task-base fast exit0 filters1837 analyzer reports to0 and completes its configured silent errortype recipe; it supplies no unfiltered-green claim.'
proof = {'base_commit': BASE, 'files': files, 'other_tracked_sources_and_old_tests_preserved_count': len(preserved),
         'preserved_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()),
         'all_original_assertions_tables_mutations_comments_and_defaults_preserved': True,
         'baseline_fingerprint_reconstructed': baseline['source_before_sha256'],
         'runtime_gap': 'Only runner-domain fallback executes successfully; host validation prevents the seven campaign tests from reaching their intended assertions.',
         'lint': lint}
evidence = {'task': 'fn-109.59', 'status': 'in_progress', 'workspace': str(ROOT), 'base_commit': BASE,
            'commits': subprocess.check_output(['git', 'rev-list', '--reverse', BASE + '..HEAD'], cwd=ROOT, text=True).splitlines(),
            'prs': [], 'tests': [row['command'] for row in receipts.values()],
            'baseline': 'red. focused-before exit1; actual Linux/arm64 adapter rejection before campaign assertions; retained original-base RED80.',
            'gates': [{'receipt': name + '.json', 'receipt_sha256': digest((OUT / (name + '.json')).read_bytes()),
                       **{key: row[key] for key in ['exit_code', 'elapsed_seconds', 'source_before_sha256', 'terminal', 'source_unchanged', 'timed_out']}} for name, row in receipts.items()],
            'focused_before': observations('focused-before'),
            'source_proof': 'source-proof.json', 'tools': baseline['tools'], 'lint': lint,
            'gate_runner_sha256': digest((OUT / 'run_gate.py').read_bytes()),
            'routing': 'Tier: session (jev-unavailable(no_key)); actual execution-model telemetry unavailable.',
            'review': 'Conductor owns fresh independent review; worker issued no verdict.',
            'commit_owner': 'Root; changes unstaged/uncommitted until explicit post-verification checkpoint grant.',
            'acceptance': 'Open. Required focused campaign controls fail before assertions; required configured/original-base source gates remain red where observed. Original first-baseline/fixed-identity/R18/R19 and deferred native requirements remain.',
            'native': 'fn149/fn128 deferred and unverified. Linux/arm64 stock-source observations supply no patched-runtime/native pass.',
            'ordinary_runner': 'Not run locally under conductor grant. Root owns one full ordinary ./runner gate on the integrated tasks57-59 candidate; requirement remains open.',
            'all_handles_terminal': True}
if candidate:
    evidence['focused_after'] = observations('focused-after')
for name, value in [('source-proof.json', proof), ('evidence.json', evidence)]:
    path = OUT / name
    if path.exists():
        raise SystemExit('Refusing to overwrite evidence')
    path.write_text(json.dumps(value, indent=2) + '\n')
print(json.dumps({'preserved_count': len(preserved), 'receipts': list(receipts), 'lint': lint}))
