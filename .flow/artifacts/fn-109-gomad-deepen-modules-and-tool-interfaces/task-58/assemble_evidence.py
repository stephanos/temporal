import collections
import difflib
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/tmp/gomad-fn109-parallel.pmgezCtg/task-58').resolve()
OUT = pathlib.Path(__file__).resolve().parent
BASE = (ROOT / '.flow/tmp/base_commit').read_text().strip()
PRODUCTION = ['tools/gomad3/runner/inspect.go', 'tools/gomad3/internal/gomadtool/conformance/driver.go']
TESTS = ['tools/gomad3/runner/inspect_cleanup_test.go', 'tools/gomad3/internal/gomadtool/conformance/driver_cleanup_test.go']


def digest(data):
    return hashlib.sha256(data).hexdigest()


def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)


def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    tests = [row for row in events if row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(row['Action'] for row in tests)),
            'top_level_counts': dict(collections.Counter(row['Action'] for row in tests if '/' not in row['Test'])),
            'failed_tests': [row['Test'] for row in tests if row['Action'] == 'fail'],
            'skipped_tests': [row['Test'] for row in tests if row['Action'] == 'skip'],
            'package_results': [{key: row[key] for key in ['Package', 'Action', 'Elapsed'] if key in row} for row in events if not row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]}


before = {path: original(path) for path in PRODUCTION}
after = {path: (ROOT / path).read_bytes() for path in PRODUCTION}
signature = b'func Inspect(path string, options InspectOptions) (result Inspection, retErr error) {'
deferred = b'\t\tdefer func(opened *artifact.Opened) {\n\t\t\tif err := opened.Close(); err != nil {\n\t\t\t\tresult = Inspection{}\n\t\t\t\tif retErr == nil {\n\t\t\t\t\tretErr = err\n\t\t\t\t} else {\n\t\t\t\t\tretErr = errors.Join(retErr, err)\n\t\t\t\t}\n\t\t\t}\n\t\t}(opened)\n'
assert after[PRODUCTION[0]].count(signature) == 1 and after[PRODUCTION[0]].count(deferred) == 1
assert after[PRODUCTION[0]].replace(signature, b'func Inspect(path string, options InspectOptions) (Inspection, error) {').replace(deferred, b'\t\tdefer opened.Close()\n') == before[PRODUCTION[0]]
checked = b'\t\t\tcleanupErr := cleanup()\n\t\t\terr := fmt.Errorf("negative compiler fixture package is invalid: %s", compilerCase.Package)\n\t\t\tif cleanupErr != nil {\n\t\t\t\terr = errors.Join(err, cleanupErr)\n\t\t\t}\n\t\t\treturn nil, nil, err\n'
unchecked = b'\t\t\tcleanup()\n\t\t\treturn nil, nil, fmt.Errorf("negative compiler fixture package is invalid: %s", compilerCase.Package)\n'
assert after[PRODUCTION[1]].count(checked) == 1
assert after[PRODUCTION[1]].replace(checked, unchecked) == before[PRODUCTION[1]]
preserved = {}
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0')
for value in paths:
    if value and value.decode() not in PRODUCTION:
        path = value.decode()
        data = (ROOT / path).read_bytes()
        assert data == original(path), path
        preserved[path] = digest(data)
assert not subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT).strip()
pattern = r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)'
old_rows = re.findall(pattern, (OUT.parent / 'task-56/corrected-make-gomad-original-base.stdout').read_text())
new_rows = re.findall(pattern, (OUT / 'make-gomad-original-base.stdout').read_text())
keys = lambda rows: collections.Counter((path, column, message, statement, caret) for path, _, column, message, statement, caret in rows)
removed, added = keys(old_rows) - keys(new_rows), keys(new_rows) - keys(old_rows)
assert len(old_rows) == 80
assert sum(removed.values()) == 2 and sum(added.values()) == 0
assert {row[0] for row in removed} == set(PRODUCTION)
assert all('(errcheck)' in row[2] for row in removed)
expected_residual = []
for row in old_rows:
    path, line, column, message, statement, caret = row
    if (path, column, message, statement, caret) in removed:
        continue
    new_line = int(line)
    if path in PRODUCTION:
        mapping = {}
        for block in difflib.SequenceMatcher(None, before[path].splitlines(), after[path].splitlines(), autojunk=False).get_matching_blocks():
            mapping.update({block.a + offset + 1: block.b + offset + 1 for offset in range(block.size)})
        new_line = mapping[int(line)]
    expected_residual.append((path, str(new_line), column, message, statement, caret))
assert collections.Counter(new_rows) == collections.Counter(expected_residual)
scoped = re.findall(pattern, (OUT / 'affected-configured-lint.stdout').read_text())
old_scoped = [row for row in old_rows if row[0].rsplit('/', 1)[0] in ['tools/gomad3/runner', 'tools/gomad3/internal/gomadtool/conformance']]
assert keys(scoped) == keys(old_scoped) - removed
lint = {'original_base': {'before': len(old_rows), 'after': len(new_rows), 'removed': sum(removed.values()), 'introduced': sum(added.values()),
                         'baseline_receipt': '../task-56/corrected-make-gomad-original-base.json', 'candidate_receipt': 'make-gomad-original-base.json'},
        'affected': {'before': len(old_scoped), 'after': len(scoped), 'removed': sum(removed.values()), 'introduced': 0,
                     'baseline': 'Extracted affected-package subset of retained original-base RED80; no separate scoped baseline run.', 'candidate_receipt': 'affected-configured-lint.json'},
        'removed_diagnostics': [list(row) for row in old_rows if (row[0], row[2], row[3], row[4], row[5]) in removed],
        'residual_path_column_message_statement_caret_multiset_preserved': True,
        'residual_lines_match_exact_unchanged_source_line_mapping': True,
        'integrated_errortype': 'Original-base Make recipe exits at configured lint and does not reach its errortype stage. Standalone affected errortype is separate evidence.'}
names = ['focused-after', 'full-ordinary-conformance', 'affected-vet', 'standalone-errortype', 'architecture-source-sets',
         'runner-ownership', 'generated-validation', 'format-check', 'affected-configured-lint', 'make-fast-task-base', 'make-gomad-original-base']
receipts = {}
for name in ['focused-before', 'corrected-focused-before'] + names:
    row = json.loads((OUT / (name + '.json')).read_text())
    assert row['terminal'] and row['source_unchanged'] and not row['timed_out'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / (name + '.' + suffix)).read_bytes()) == row[suffix + '_sha256'], name
    assert digest((OUT / (name + '-sources.json')).read_bytes()) == row['source_manifest_sha256']
    receipts[name] = row
candidate = receipts['focused-after']
assert all(receipts[name]['source_before_sha256'] == candidate['source_before_sha256'] for name in names)
assert all(row['tools'] == candidate['tools'] for row in receipts.values())
baseline_manifest = json.loads((OUT / 'corrected-focused-before-sources.json').read_text())
candidate_manifest = json.loads((OUT / 'focused-after-sources.json').read_text())
assert {path for path in candidate_manifest if candidate_manifest[path] != baseline_manifest[path]} == set(PRODUCTION)
for path in TESTS:
    assert digest((ROOT / path).read_bytes()) == baseline_manifest[path]
focused, ordinary = observations('focused-after'), observations('full-ordinary-conformance')
assert focused['counts'].get('fail', 0) == 0 and focused['counts']['pass'] > 0
assert ordinary['counts']['pass'] > 0
proof = {'base_commit': BASE, 'workspace': str(ROOT), 'production': {path: {'base_sha256': digest(before[path]), 'candidate_sha256': digest(after[path]), 'complete_preimage_reconstructed': True} for path in PRODUCTION},
         'tests': {path: digest((ROOT / path).read_bytes()) for path in TESTS},
         'corrected_focused_baseline_identical_final_tests': True, 'initial_to_corrected_test_delta': 'initial-to-corrected-proof.json', 'other_tracked_sources_and_old_tests_preserved_count': len(preserved),
         'preserved_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()),
         'source_order': {'Inspect': 'Same receiver evaluated at original defer registration. Project manifest, target sharing and optional choices retain their ordering. Close runs after all original work. Nil Close leaves complete result and primary object unchanged. Sole error direct; primary and genuine cleanup error join primary-first; every genuine Close failure clears Inspection.',
                          'interceptionFixtures': 'One immediate cleanup captured before exact original primary fmt.Errorf. No defer, retry, ownership transfer or additional operation. Nil cleanup returns original nil/nil/error tuple and object; nonnil cleanup joins primary-first.'},
         'runtime_gaps': ['Inspect genuine sole Close failure and simultaneous original primary/Close failure unexecuted. Pinned qualified Unix os.Root.Close always returns nil; no fault seam, descriptor theft or race introduced.',
                          'Invalid compiler-fixture workspace RemoveAll failure and primary-plus-cleanup failure unexecuted; no existing deterministic failure hook.',
                          'Fixture preparation controls run no native compiler or interception execution. Mock runWith requests only GOROOT before invalid fixture rejection.'],
         'unix_root_close_source': {'path': '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/os/root_openat.go',
                                    'sha256': digest(pathlib.Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/os/root_openat.go').read_bytes()), 'lines': '32-40'},
         'lint': lint}
evidence = {'task': 'fn-109.58', 'status': 'in_progress', 'workspace': str(ROOT), 'base_commit': BASE,
            'branch': subprocess.check_output(['git', 'branch', '--show-current'], cwd=ROOT, text=True).strip(), 'commits': [], 'prs': [],
            'commit_owner': 'Root granted one local source-progress checkpoint after independent review and exact candidate/authoring-reconciliation verification. Root owns integration and target commits.',
            'checkpoint_authorization': {'scope': 'Four owned product/test files and complete task58 packet only.',
                                         'frozen_source_sha256': candidate['source_before_sha256'],
                                         'commit_id_record': 'Worker return and root integration record. This packet contains no self-referential checkpoint SHA.',
                                         'formal_acceptance': 'in_progress; required red source gates and combined ordinary runner obligation remain open.'},
            'baseline': {'status': 'red', 'analyzer_receipt_reused': '../task-56/corrected-make-gomad-original-base.json',
                         'focused': 'corrected-focused-before.json', 'initial_fixture_failure': 'focused-before.json. Newly authored expected package list included reviewed-only os/exec. Corrected from independently read manifest.intercepts before production edits; raw failure retained. Both new test files also gained stronger direct-error checks, and the driver cleanup fallback was guarded. See initial-to-corrected-proof.json for the complete delta and exact initial preimages.',
                         'initial_to_corrected_test_delta': 'initial-to-corrected-proof.json',
                         'authoring_diff_encoding': 'Root authorized lossless JSON encoding of both reviewed unified diffs. Decoded bytes retain their reviewed SHA256; exact initial preimages and all gate receipts are unchanged.'},
            'tests': [receipts[name]['command'] for name in names],
            'gates': [{'receipt': name + '.json', **{key: receipts[name][key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_before_sha256', 'terminal', 'source_unchanged']}} for name in names],
            'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'], 'source_proof': 'source-proof.json', 'gate_runner_sha256': digest((OUT / 'run_gate.py').read_bytes()),
            'focused': focused, 'ordinary_conformance': ordinary, 'lint': lint, 'runtime_gaps': proof['runtime_gaps'],
            'full_ordinary_runner': 'Root reserved one ordinary runner gate on integrated tasks57-59 batch. No worker receipt or reused pass claimed.',
            'routing': {'tier': 'Tier: session (jev-unavailable(no_key))', 'implementer_preference': 'gpt-6.1-sol at high', 'actual_execution_model': 'unobserved'},
            'review': 'Root reported fresh independent bounded SOURCE-PROGRESS acceptance with Minor resolved, verified the exact candidate and authoring reconciliation, and granted one local checkpoint. Worker issued no review verdict. Formal acceptance remains open.',
            'native': 'Real source execution is Linux/arm64. fn149/fn128 remain deferred/unverified; ordinary artifact inspection and fixture preparation supply no native compiler or full native test-host pass.',
            'initial_gate_runner_sha256': digest((OUT / 'initial-run_gate.py').read_bytes()),
            'acceptance': 'Required affected and original-base lint remain red. Ordinary conformance fails the unchanged TestRuntimeOwnedControlProbe on missing patched .toolchain/bin/go before runtime-owned build. No ordinary/native pass claimed. Original first-baseline/fixed-identity/R18/R19 and formal acceptance stay open wherever unproved. No Done, formal SHIP, full-spec acceptance, native revival, PR, push or CI authority.',
            'all_handles_terminal': True}
for name, value in [('source-proof.json', proof), ('evidence.json', evidence)]:
    path = OUT / name
    if path.exists():
        raise SystemExit('Refusing to overwrite retained evidence')
    text = json.dumps(value, indent=2) + '\n'
    patch = '*** Begin Patch\n*** Add File: ' + str(path.relative_to(ROOT)) + '\n' + ''.join('+' + line + '\n' for line in text.splitlines()) + '*** End Patch\n'
    subprocess.run(['apply_patch'], input=patch, text=True, cwd=ROOT, check=True, env={key: value for key, value in __import__('os').environ.items() if key != 'BASH_ENV'})
print(json.dumps({'focused': focused, 'ordinary': ordinary, 'lint': lint, 'source': evidence['frozen_source_sha256']}))
