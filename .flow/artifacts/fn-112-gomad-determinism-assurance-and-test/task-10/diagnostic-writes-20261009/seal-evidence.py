import collections
import datetime
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BASE = (OUT / 'base-commit').read_text().strip()
PRIOR = OUT.parent / 'source-acceptance-20261009'
OWNED = ['tools/gomad3/cmd/gomadtool/soak.go', 'tools/gomad3/cmd/gomadtool/soak_test.go']

def digest(contents):
    return hashlib.sha256(contents).hexdigest()

def git(*arguments):
    return subprocess.check_output(['git', *arguments], cwd=ROOT)

def issues(name):
    return re.findall(r'^(.+\.go):(\d+):(\d+): (.+) \(([^)]+)\)$', (OUT / (name + '.stdout')).read_text(), re.M)

before_issues = issues('baseline-configured-lint')
after_issues = issues('final-configured-lint')
removed = sorted(set(before_issues) - set(after_issues))
introduced = sorted(set(after_issues) - set(before_issues))
assert len(before_issues) == 63 and len(after_issues) == 59
assert len(removed) == 4 and not introduced
assert all(row[0] == OWNED[0] and row[4] == 'errcheck' for row in removed)

tracked = [path for path in git('ls-files', 'tools/gomad3', 'tools/gomad3integration', '.github',
                              'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode').decode().splitlines()]
changed = git('diff', '--name-only', BASE, '--', *tracked).decode().splitlines()
assert sorted(changed) == sorted(OWNED), changed
source_changes = {}
for path in OWNED:
    before = git('show', BASE + ':' + path)
    after = (ROOT / path).read_bytes()
    source_changes[path] = {'before_sha256': digest(before), 'after_sha256': digest(after)}
    if path.endswith('/soak.go'):
        calls = lambda value: re.findall(rb'fmt\.F(?:println|printf)\(stderr, [^\n]+?\)', value)
        assert len(calls(before)) == 4 and calls(before) == calls(after)
        source_changes[path]['ordered_diagnostic_calls_unchanged'] = True
    else:
        old_body = before.split(b'func TestSoakRejectsInvalidInvocationsAsInvalidInput', 1)[1]
        old_in_after = after.split(b'func TestSoakRejectsInvalidInvocationsAsInvalidInput', 1)[1].split(b'\nfunc TestSoakDiagnosticWritesPreserveStatus', 1)[0]
        assert old_body.rstrip() == old_in_after.rstrip()
        source_changes[path]['original_test_body_unchanged'] = True

protected_before = {
    '.turbo/plans/gomad3-glossary-update.md': '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188',
    '.turbo/technical-debt.md': 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287',
}
protected = {path: {'before_sha256_reported_by_root': value, 'current_sha256_observed_by_worker': digest((ROOT / path).read_bytes())}
             for path, value in protected_before.items()}
assert all(row['before_sha256_reported_by_root'] == row['current_sha256_observed_by_worker'] for row in protected.values())
assert not git('diff', '--cached', '--name-only'), 'Unexpected staged changes'
gates = {}
for path in sorted(OUT.glob('*.json')):
    data = json.loads(path.read_text())
    if 'command' in data and 'exit_code' in data:
        assert data['terminal'] and data['source_unchanged'], path.name
        for stream in ['stdout', 'stderr']:
            assert digest((OUT / (path.stem + '.' + stream)).read_bytes()) == data[stream + '_sha256'], path.name
        gates[path.stem] = data
counts = {}
for name, gate in gates.items():
    events = gate['test_events']
    if events:
        counts[name] = {
            'top_level_pass': sum(row['Action'] == 'pass' and 'Test' in row and '/' not in row['Test'] for row in events),
            'all_test_pass': sum(row['Action'] == 'pass' and 'Test' in row for row in events),
            'failed_tests': [row['Test'] for row in events if row['Action'] == 'fail' and 'Test' in row],
            'skipped_tests': [row['Test'] for row in events if row['Action'] == 'skip' and 'Test' in row],
        }
        if gate['exit_code'] == 0:
            assert counts[name]['all_test_pass'] > 0 and not counts[name]['failed_tests'] and not counts[name]['skipped_tests']
original_issues = issues('make-gomad-original-base')
assert len(original_issues) == 204, len(original_issues)
oracle = pathlib.Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/os/file_unix.go')
assert 'return &LinkError{"rename", oldname, newname, syscall.EEXIST}' in oracle.read_text()
head = git('rev-parse', 'HEAD').decode().strip()
result = {
    'task': 'fn-112-gomad-determinism-assurance-and-test.10', 'status': 'in_progress', 'acceptance_verdict': None,
    'base_commit': BASE, 'head_commit': head, 'source_range': BASE + '..' + head,
    'commits': git('rev-list', '--reverse', BASE + '..HEAD').decode().splitlines(),
    'tests': [data['command'] for data in gates.values()], 'prs': [], 'gates': gates, 'test_counts': counts,
    'source_changes': source_changes, 'unrelated_product_paths_unchanged_from_base': len(tracked) - len(OWNED),
    'generator_input_inspection': 'tools/gomad3/Makefile:5-9,23-33,158-173; admitted files change no declared generator inputs, pins or outputs; check-only validate passes',
    'preservation': 'ordered four diagnostic calls and original invalid-input test body byte-preserved; every other tracked product input unchanged from base; existing first-baseline/fixed-identity obligations and historical evidence retain their meaning',
    'protected_user_files': protected, 'index_empty': True,
    'lint': {'baseline_scoped': 63, 'final_scoped': 59, 'removed': removed, 'introduced': introduced,
             'residual_by_file': dict(collections.Counter(row[0] for row in after_issues)),
             'original_base': '951c5516e9e7b3066e7e069adda9565cfd68844c', 'original_base_final_diagnostics': len(original_issues),
             'make_admission_result': 'exit0, 55 host packages, diff-filtered zero issues; errortype reached',
             'make_original_result': 'exit2, 55 host packages, 204 configured diagnostics; integrated errortype unreached'},
    'baseline': 'full ordinary cmd package green on changed private overlayfs TMPDIR, 195 tests/subtests; validate green; configured lint red63 pre-edit',
    'failed_fixture_construction': 'preservation-controls-before-fix failed on incorrectly assumed EISDIR and infrastructure_failure literals; retained, corrected from pinned Go/soak source before production edit; this is not the meaningful errcheck reproduction',
    'prior_evidence': {str(PRIOR.relative_to(ROOT) / name): digest((PRIOR / name).read_bytes()) for name in ['handover.md', 'evidence.json', 'configured-lint.log']},
    'prior_failure': 'task51 usage-status-20261009 ordinary-package-final Git-fixture failure on FUSE retained unchanged; root admitted one changed-input ordinary baseline; causality unproved',
    'stdlib_rename_oracle': {'path': str(oracle), 'sha256': digest(oracle.read_bytes()), 'lines': '26-44', 'contract': 'Unix os.rename returns EEXIST for file-over-directory before calling syscall.Rename'},
    'skips': ['native make -C tools/gomad3 test and full test-host/soak remain deferred under fn-149/fn-128; this stock linux/arm64 execution is portable source evidence only',
              'formal impl-review deferred to root; required aggregate source lint remains red; no Flow lifecycle mutation or commit'],
    'native_qualification_claim': False, 'native_bound': None, 'live_command_handles': [],
    'tier': 'session (jev-unavailable(no_key))', 'requested_implementer': 'gpt-6.1-sol at high',
    'review': 'not dispatched; root owns review/lifecycle/commits; aggregate source lint remains red',
    'packet_scripts': {name: digest((OUT / name).read_bytes()) for name in ['run_gate.py', 'seal-evidence.py']},
    'sealed_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
}
with (OUT / 'evidence.json').open('x') as output:
    json.dump(result, output, indent=2)
    output.write('\n')
print(json.dumps({'gates': len(gates), 'scoped_lint': len(after_issues), 'original_lint': len(original_issues), 'head': head, 'changed_product_paths': changed, 'all_handles_terminal': True}))
