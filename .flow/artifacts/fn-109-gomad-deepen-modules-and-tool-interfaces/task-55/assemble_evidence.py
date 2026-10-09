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
TEST = 'tools/gomad3/cmd/gomad/internal/cli/replay_output_test.go'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

before, after = original(CLI), (ROOT / CLI).read_bytes()
old = b'\t\tfmt.Fprintf(stdout, "gomad: verified %s\\n", result.Artifact.Path)\n'
new = b'\t\tif _, err := fmt.Fprintf(stdout, "gomad: verified %s\\n", result.Artifact.Path); err != nil {\n\t\t\treturn 3\n\t\t}\n'
assert after.count(new) == 1 and after.replace(new, old) == before
prior = json.loads((OUT.parent / 'task-54/source-proof.json').read_text())
assert prior['candidate_sha256'] == digest(before)
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0')
preserved = {}
for value in paths:
    if value and value.decode() != CLI:
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
    return {'counts': dict(collections.Counter(row['Action'] for row in tests)),
            'failed_tests': [row['Test'] for row in tests if row['Action'] == 'fail'],
            'package_results': [{key: row[key] for key in ['Package', 'Action', 'Elapsed'] if key in row}
                                for row in events if not row.get('Test') and row['Action'] in ['pass', 'fail', 'skip']]}

def lint_delta(name, expected):
    pattern = r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)'
    old_rows = re.findall(pattern, (OUT.parent / ('task-54/' + name + '.stdout')).read_text())
    new_rows = re.findall(pattern, (OUT / (name + '.stdout')).read_text())
    keys = lambda rows: collections.Counter((path, column, message, statement, caret) for path, _, column, message, statement, caret in rows)
    old_keys, new_keys = keys(old_rows), keys(new_rows)
    removed, added = old_keys - new_keys, new_keys - old_keys
    assert (len(old_rows), len(new_rows), sum(removed.values()), sum(added.values())) == (*expected, 1, 0)
    site = next(iter(removed))
    assert site[0] == CLI and 'not checked' in site[2] and 'gomad: verified %s' in site[3]
    return {'before': len(old_rows), 'after': len(new_rows), 'removed': 1, 'added': 0,
            'removed_diagnostic': list(site), 'residual_path_column_message_statement_caret_multiset_preserved': True,
            'baseline_receipt': '../task-54/' + name + '.json', 'candidate_receipt': name + '.json'}

names = ['focused-before', 'focused-before-public', 'focused-before-final', 'public-fixture-diagnosis',
         'focused-red-final', 'focused-after', 'full-ordinary-cli', 'affected-vet', 'standalone-errortype',
         'affected-configured-lint', 'architecture-source-sets', 'runner-ownership', 'generated-validation',
         'format-check', 'make-fast-task-base', 'make-gomad-original-base', 'tool-identity']
receipts = {}
for name in names:
    receipt = json.loads((OUT / (name + '.json')).read_text())
    assert receipt['terminal'] and receipt['source_unchanged'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / (name + '.' + suffix)).read_bytes()) == receipt[suffix + '_sha256'], name
    receipts[name] = receipt
candidate = receipts['focused-after']
assert all(receipts[name]['source_before_sha256'] == candidate['source_before_sha256'] for name in names[names.index('focused-after'):])
assert observations('focused-after')['counts'].get('fail', 0) == 0
assert observations('focused-red-final')['failed_tests'] == ['TestReplayOutputVerifyOnly/EBADF', 'TestReplayOutputVerifyOnly']
proof = {
    'base_commit': BASE, 'production_path': CLI, 'base_sha256': digest(before), 'candidate_sha256': digest(after),
    'complete_original_bytes_reconstructed': True, 'single_check_failed_status': 3,
    'new_test_sha256': digest((ROOT / TEST).read_bytes()), 'old_tests_and_other_source_preserved_count': len(preserved),
    'preserved_source_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()), 'protected_turbo_sha256': turbo,
    'lint': {'affected': lint_delta('affected-configured-lint', (2, 1)), 'original_base': lint_delta('make-gomad-original-base', (96, 95))},
    'public_attempt_source_sha256': digest((OUT / 'public-fixture-attempt.go.txt').read_bytes()),
}
evidence = {
    'task': 'fn-109.55', 'status': 'in_progress', 'base_commit': BASE, 'branch': 'gomad', 'commits': [], 'prs': [],
    'commit_owner': 'root; all worker changes unstaged and uncommitted',
    'predecessor': '../task-54/handover.md, source-proof.json and independent-review.md',
    'baseline': {'status': 'red', 'inherited_actual_receipts_reused': ['../task-54/affected-configured-lint.json', '../task-54/make-gomad-original-base.json'],
                 'focused': 'focused-red-final.json', 'source_identical_before_cli_sha256': digest(before)},
    'tests': [receipts[name]['command'] for name in names],
    'gates': [{'receipt': name + '.json', **{key: receipts[name][key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_before_sha256', 'terminal', 'source_unchanged']}} for name in names],
    'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'], 'environment': candidate['environment'],
    'source_proof': 'source-proof.json', 'lint': proof['lint'],
    'focused_after': observations('focused-after'), 'full_ordinary_cli': observations('full-ordinary-cli'),
    'controls': 'Existing private fakeInstallation replayDependencies and real terminalDiagnostics read-only os.File EBADF; callback completed before output, one exact path-operand attempt, empty stderr, no fallback; exact installation/request/context fields, earlier installation3/preflight2/other replay3 and no success attempts. Existing ordinary replay characterization unchanged.',
    'public_path': {'receipt': 'public-fixture-diagnosis.json', 'source': 'public-fixture-attempt.go.txt',
                    'result': 'Real stock Go1.27.1 CLI test executable/buildinfo, actual linux/arm64, current public I/O profile, no World/choice/simulation state and stock-go-SHA256-bound private installation passed artifact publication/preflight, then VerifyAdapters refused linux/arm64 before success output.',
                    'diagnostic': 'incompatible replay artifact: verify replay adapters: deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64',
                    'gap': 'Exported Run success-write/EBADF path and patched-runtime/native replay are unproved. Attempted new permanent probe removed after real host refusal; no production seam, host spoof, native guard or arbitrary executable bytes.'},
    'retained_probe_failures': 'focused-before-public.json records initially missing canonical empty target slices. focused-before-final.json records opaque zero-success-attempt assertions; public-fixture-diagnosis.json retains exact real-host adapter refusal. These are separate from causal private EBADF RED.',
    'ordinary_inherited_failures': ['TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput', 'TestRunDoctorReportsAvailableContractAsJSON', 'TestCheckReportsAvailableContract'],
    'ordinary_new_fixture_conflict': 'TestCharacterizeOutputWriterFailures/replay_verification and its parent fail because the preserved old fixture still expects0 for failed reporting. Task55 explicitly admits3 but forbids old-test/comment edits; root owns any separate reconciliation admission.',
    'native_command_failure': 'cmd/gomad TestMain CLI build fails before collection because repository patched .toolchain/bin/go is absent; not relabeled as a portable pass.',
    'integrated_errortype': 'Unreached after original-base configured lint failure. Standalone and task-base fast passes do not substitute.',
    'acceptance': 'Open on affected configured lint1, integrated make lint2 and full ordinary test1; no formal review or Done.',
    'routing': {'tier_line': 'Tier: session (jev-unavailable(no_key))', 'implementer_preference': 'gpt-6.1-sol at high', 'actual_execution_metadata': 'unavailable'},
    'review': 'Root owns fresh independent source-progress review; worker issued no verdict.',
    'native': 'fn149/fn128 deferred and unverified; no native revival, PR, push, CI or publication.',
    'all_handles_terminal': True,
}
for name, value in [('source-proof.json', proof), ('evidence.json', evidence)]:
    path = OUT / name
    if path.exists():
        raise SystemExit('Refusing to overwrite retained evidence')
    path.write_text(json.dumps(value, indent=2) + '\n')
print(json.dumps({'focused': evidence['focused_after'], 'ordinary': evidence['full_ordinary_cli'], 'lint': proof['lint'], 'source': evidence['frozen_source_sha256']}))
