import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BASE = (OUT / 'base_commit').read_text().strip()
SOURCE = 'tools/gomad3/toolchain/build.go'
TEST = 'tools/gomad3/toolchain/build_cleanup_test.go'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

before, after = original(SOURCE), (ROOT / SOURCE).read_bytes()
ranges = [('func snapshotInputs(', 'func (snapshot inputSnapshot) remove('),
          ('func publishStable(', 'func temporaryFile('), ('func temporaryFile(', 'func runCommand(')]
masked_before, masked_after = before, after
for start, end in ranges:
    old = before[before.index(start.encode()):before.index(end.encode())]
    new = after[after.index(start.encode()):after.index(end.encode())]
    masked_before = masked_before.replace(old, start.encode())
    masked_after = masked_after.replace(new, start.encode())
assert masked_before == masked_after
assert before.count(b'\t\tpatch.Close()\n') == 1
assert before.count(b'\t\tfile.Close()\n') == 3
assert after.count(b'\t\tcloseErr := file.Close()\n') == 3
assert after.count(b'defer func(path string)') == 2
assert after.count(b'Published = true\n') == 2
assert after.index(b'stampPublished = true') < after.index(b'syncDirectory(toolchainRoot)', after.index(b'func publishStable('))
assert after.index(b'launcherPublished = true') < after.index(b'syncDirectory(binRoot)', after.index(b'func publishStable('))
preserved = {}
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0')
for value in paths:
    if value and value.decode() != SOURCE:
        path = value.decode()
        data = (ROOT / path).read_bytes()
        assert data == original(path), path
        preserved[path] = digest(data)
prior = json.loads((OUT.parent / 'task-55/reconciled-source-proof.json').read_text())
turbo = {path: digest((ROOT / path).read_bytes()) for path in prior['protected_turbo_sha256']}
assert turbo == prior['protected_turbo_sha256']
assert not subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT).strip()

def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    tests = [row for row in events if row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(row['Action'] for row in tests)),
            'top_level_counts': dict(collections.Counter(row['Action'] for row in tests if '/' not in row['Test'])),
            'failed_tests': [row['Test'] for row in tests if row['Action'] == 'fail'],
            'skipped_tests': [row['Test'] for row in tests if row['Action'] == 'skip'],
            'failure_output': [row['Output'] for row in events if row.get('Test') in {item['Test'] for item in tests if item['Action'] == 'fail'} and row.get('OutputType') == 'error'],
            'package_results': [{key: row[key] for key in ['Package', 'Action', 'Elapsed'] if key in row} for row in events if not row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]}

pattern = r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)'
old_rows = re.findall(pattern, (OUT.parent / 'task-55/reconciled-make-gomad-original-base.stdout').read_text())
new_rows = re.findall(pattern, (OUT / 'corrected-make-gomad-original-base.stdout').read_text())
keys = lambda rows: collections.Counter((path, column, message, statement, caret) for path, _, column, message, statement, caret in rows)
old_keys, new_keys = keys(old_rows), keys(new_rows)
removed, added = old_keys - new_keys, new_keys - old_keys
assert (len(old_rows), len(new_rows), sum(removed.values()), sum(added.values())) == (95, 80, 15, 0)
assert all(row[0] == SOURCE and '(errcheck)' in row[2] for row in removed)
assert collections.Counter(row for row in old_rows if row[0] != SOURCE) == collections.Counter(new_rows)
scoped = re.findall(pattern, (OUT / 'corrected-affected-configured-lint.stdout').read_text())
old_scoped = [row for row in old_rows if row[0].startswith('tools/gomad3/toolchain/')]
assert len(old_scoped) == 16 and len(scoped) == 1
assert keys(scoped) == keys(old_scoped) - removed
lint = {'original_base': {'before': 95, 'after': 80, 'removed': 15, 'added': 0,
                         'baseline_receipt': '../task-55/reconciled-make-gomad-original-base.json',
                         'candidate_receipt': 'corrected-make-gomad-original-base.json'},
        'affected': {'before': 16, 'after': 1, 'removed': 15, 'added': 0,
                     'baseline': 'Actual retained original-base RED95 contains 16 toolchain findings; affected baseline is this extracted subset, not a new scoped run.',
                     'candidate_receipt': 'corrected-affected-configured-lint.json'},
        'removed_diagnostics': [list(row) for row in old_rows if row[0] == SOURCE],
        'residual_path_column_message_statement_caret_multiset_preserved': True,
        'residual_full_diagnostic_blocks_including_line_numbers_preserved': True,
        'integrated_errortype': 'Original-base stage unreached after configured golangci exits 1 and recursive make exits 2. Corrected task-base fast stage reaches and passes its configured errortype recipe; standalone affected errortype passes separately.'}
initial_names = ['focused-after', 'full-ordinary-toolchain', 'affected-vet', 'standalone-errortype',
         'architecture-source-sets', 'runner-ownership', 'generated-validation', 'format-check',
         'affected-configured-lint', 'make-fast-task-base', 'make-gomad-original-base']
corrected_names = ['corrected-' + name for name in ['focused', 'affected-vet', 'standalone-errortype', 'architecture',
                   'format-check', 'affected-configured-lint', 'make-fast-task-base', 'make-gomad-original-base']]
names = initial_names + corrected_names
receipts = {}
for name in names:
    record = json.loads((OUT / (name + '.json')).read_text())
    assert record['terminal'] and record['source_unchanged'] and not record['timed_out'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / (name + '.' + suffix)).read_bytes()) == record[suffix + '_sha256'], name
    receipts[name] = record
candidate = receipts['corrected-focused']
initial_candidate = receipts['focused-after']
assert all(receipts[name]['source_before_sha256'] == initial_candidate['source_before_sha256'] for name in initial_names)
assert all(receipts[name]['source_before_sha256'] == candidate['source_before_sha256'] for name in corrected_names)
assert all(row['tools'] == candidate['tools'] for row in receipts.values())
final_test = (ROOT / TEST).read_bytes()
tagged = b'\t\t\tswitch failure {\n\t\t\tcase "stamp-rename":\n\t\t\t\tblocked = layout.BuildKeyFile()\n\t\t\tcase "launcher-rename":\n\t\t\t\tblocked = layout.GoCommand()\n\t\t\tdefault:\n\t\t\t\tconfig.Testing = true\n\t\t\t\tconfig.FailurePhase = failure\n\t\t\t}\n'
conditional = b'\t\t\tif failure == "stamp-rename" {\n\t\t\t\tblocked = layout.BuildKeyFile()\n\t\t\t} else if failure == "launcher-rename" {\n\t\t\t\tblocked = layout.GoCommand()\n\t\t\t} else {\n\t\t\t\tconfig.Testing = true\n\t\t\t\tconfig.FailurePhase = failure\n\t\t\t}\n'
assert final_test.count(tagged) == 1
initial_test = final_test.replace(tagged, conditional)
binding_paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
                                        '.github/workflows/gomad3.yml', '.github/.golangci.yml',
                                        'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
files = {value.decode(): digest((ROOT / value.decode()).read_bytes()) for value in binding_paths if value}
files[TEST] = digest(initial_test)
assert digest(json.dumps(files, sort_keys=True).encode()) == initial_candidate['source_before_sha256']
files[SOURCE] = digest(before)
baseline_candidate = json.loads((OUT / 'focused-before-corrected.json').read_text())
assert digest(json.dumps(files, sort_keys=True).encode()) == baseline_candidate['source_before_sha256']
focused, ordinary = observations('corrected-focused'), observations('full-ordinary-toolchain')
assert focused['counts'].get('fail', 0) == 0 and focused['counts']['pass'] > 0
proof = {'base_commit': BASE, 'production_path': SOURCE, 'base_sha256': digest(before), 'candidate_sha256': digest(after),
         'bytes_outside_three_admitted_functions_preserved': True,
         'new_test_sha256': digest((ROOT / TEST).read_bytes()), 'other_sources_and_old_tests_preserved_count': len(preserved),
         'review_fixture_correction': {'initial_additive_test_sha256': digest(initial_test), 'corrected_test_sha256': digest(final_test),
                                     'only_change': 'Equivalent tagged string switch retains exact assignment bodies/order; failure is an unchanged loop-local string. All four branches covered by corrected focused run.',
                                     'initial_candidate_reconstructed_fingerprint': initial_candidate['source_before_sha256'],
                                     'initial_BASE_additive_test_reconstructed_fingerprint': baseline_candidate['source_before_sha256'],
                                     'literal_identical_corrected_test_BASE_execution': 'Not run. Initial BASE used the prior conditional fixture; exact branch/body equivalence supplies bounded source-progress proof only.',
                                     'source_and_all_other_gated_inputs_unchanged_after_fixture_correction': True},
         'preserved_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()), 'protected_turbo_sha256': turbo,
         'source_order': {
             'snapshotInputs': 'Each early branch saves Close/Remove or Remove/RemoveAll results in original order before the original fmt.Errorf; joins only nonnil failures primary-first; zero snapshot retained.',
             'temporaryFile': 'Chmod/Write/Sync failure saves exactly one Close then Remove; final Close failure removes once; raw err unchanged if cleanup nil; empty path retained.',
             'publishStable': 'Inline defers register immediately after each successful temporaryFile at original sites, with arguments evaluated there. Stamp defer runs first. Independent rename flags set only after successful respective Rename before sync/hook. Each ignores only corresponding post-rename ENOENT; sole cleanup error direct, joins nonnil primary first.',
             'buildWith': 'Complete original bytes preserved, including existing buildFailure and already-checked cleanup boundaries.'},
         'runtime_gaps': [
             'snapshotInputs genuine patch.Close failure after copy failure, genuine patch Remove at each early path, overlay RemoveAll failure and simultaneous cleanup faults unexecuted.',
             'snapshotInputs final patch.Close failure and overlay MkdirTemp failure after patch close unexecuted.',
             'temporaryFile genuine Chmod/Write/Sync/final Close failure after CreateTemp, respective early Close/Remove failures and multiple faults unexecuted; ordinary rename/read/remove usability does not prove descriptor close by itself.',
             'publishStable genuine deferred Remove failure, unpublished missing-path ENOENT, unrelated post-rename cleanup error and multiple deferred cleanup failures unexecuted.',
             'publishStable root/bin directory Sync failures and stamp-temporary creation failure after launcher creation unexecuted.'
         ], 'lint': lint}
evidence = {'task': 'fn-109.56', 'status': 'in_progress', 'base_commit': BASE, 'branch': 'gomad', 'commits': [], 'prs': [],
            'commit_owner': 'root; worker leaves changes unstaged/uncommitted',
            'predecessor': '../task-55/reconciled-evidence.json, reconciled-source-proof.json and independent-review.md',
            'baseline': {'status': 'red', 'analyzer_receipt_reused': '../task-55/reconciled-make-gomad-original-base.json',
                         'focused': 'focused-before-corrected.json', 'initial_fixture_failure': 'focused-before.json. Existing debris helper labels intentional build-key destination directory temporary; corrected only the new fixture scan.'},
            'tests': [row['command'] for row in receipts.values()],
            'gates': [{'receipt': name + '.json', **{key: receipts[name][key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_before_sha256', 'terminal', 'source_unchanged']}} for name in names],
            'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'], 'environment': candidate['environment'],
            'initial_candidate_scope': {'fingerprint': initial_candidate['source_before_sha256'],
                                        'receipts': [name + '.json' for name in initial_names],
                                        'initial_reds': 'Affected2, fast1, original81. QF1003 in new conditional fixture was the only introduced diagnostic. Raw receipts retained.',
                                        'reuse': 'Initial full ordinary, complete static, private-injection and generated-validation receipts stay at initial fingerprint. Root-approved small-review-fix rule uses exact production/all-other-input preservation and fixture branch proof. No candidate-wide green rebinding.'},
            'corrected_candidate_scope': {'fingerprint': candidate['source_before_sha256'], 'receipts': [name + '.json' for name in corrected_names]},
            'gate_runner_sha256': digest((OUT / 'run_gate.py').read_bytes()), 'source_proof': 'source-proof.json', 'lint': lint,
            'focused': focused, 'full_ordinary_toolchain': ordinary,
            'controls': ['Literal patch/overlay bytes and 0600/0700 modes; independent snapshots after original mutation and removal.',
                         'Missing patch/overlay and nonregular symlink exact errors, zero snapshot and absence of temporary state.',
                         'Temporary literal NUL bytes and 0644/0755 modes; normal file usability; missing destination returns direct PathError and empty path.',
                         'Repeated stable publications bind literal launcher, stamp/key newline, 0755/0644 modes and no temporary state.',
                         'Existing fake dependencies preserve buildFailure/InjectedFailure phases, completed immutable build and stamp/launcher survival; genuine nonempty destination Rename fails with concrete LinkError and retained marker/old launcher.'],
            'runtime_gaps': proof['runtime_gaps'],
            'review_fixture_correction': proof['review_fixture_correction'],
            'acceptance': 'Open. Required affected and original-base lint red; full ordinary toolchain passed with five existing runtime inventory skips, supplying no full native pass. Original first-baseline/fixed-identity/R18/R19 and other source owners remain required.',
            'routing': {'tier': 'Tier: session (jev-unavailable(no_key))', 'implementer_preference': 'gpt-6.1-sol at high', 'actual_execution_model': 'unobserved'},
            'review': 'Root owns fresh independent source-progress review; worker issued no verdict or formal review.',
            'native': 'fn149/fn128 deferred and current candidate unverified. Ordinary helpers and existing fake dependencies run on developmental Linux/arm64; real Build rejects unsupported host. No native pass or publication authority.',
            'all_handles_terminal': True}
for name, value in [('source-proof.json', proof), ('evidence.json', evidence)]:
    path = OUT / name
    if path.exists():
        raise SystemExit('Refusing to overwrite retained evidence')
    path.write_text(json.dumps(value, indent=2) + '\n')
print(json.dumps({'focused': focused, 'ordinary': ordinary, 'lint': lint, 'source': evidence['frozen_source_sha256']}))
