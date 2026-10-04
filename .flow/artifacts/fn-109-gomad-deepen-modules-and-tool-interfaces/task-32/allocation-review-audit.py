import datetime
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def inventory():
    return {str(p.relative_to(OUT)): digest(p) for p in sorted(OUT.rglob('*')) if p.is_file()}

frozen = inventory()
worker = json.loads((OUT / 'allocation-repair-evidence.json').read_text())
worker_result = subprocess.run(['python3', str(OUT / 'review-allocation-worker-audit.py')], cwd=ROOT, check=True, capture_output=True, text=True)
historical = json.loads(worker_result.stdout)
assert historical['writes'] == 0
assert [historical[k] for k in ('historical_files', 'historical_receipts', 'archived_scout_receipts', 'allocation_receipts')] == [234, 80, 4, 14]
assert [historical[k] for k in ('introduced_alias_red', 'supplementary_inherited_red', 'origin_remaining_value_copy_red')] == [12, 7, 2]
final = json.loads((OUT / 'allocation-repair-final.json').read_text())['source']
assert len(final) == 15
assert {str(p.relative_to(ROOT)): digest(p) for p in sorted(ARCH.glob('*.go'))} == final
assert final[str((ARCH / 'effects.go').relative_to(ROOT))] == '42828ad3f181b87f9e857db067a5474c3c947721bdb40cdb751fb1f3f0c7844b'
assert final[str((ARCH / 'standard.go').relative_to(ROOT))] == 'efad9411c3876b452d4d2f4905a4ba1cb92b43a0428ea2f039c94b3e5d9fd88d'
assert final[str((ARCH / 'error_provenance_test.go').relative_to(ROOT))] == '6280351c2184961fdb24be4c482bb64d8350750883373f6245ace742e5608598'
assert (ARCH / 'standard.go').read_bytes() == (OUT / 'sources/writer-stage/standard.go').read_bytes()
assert (ARCH / 'error_provenance_test.go').read_bytes().startswith((OUT / 'sources/map-repair-final/error_provenance_test.go').read_bytes())
admission = json.loads((OUT / 'source-admission.json').read_text())
for path, expected in admission['protected_original_documents'].items():
    if path == 'AGENTS.md':
        historical_doc = subprocess.check_output(['git', 'show', worker['base_commit'] + ':' + path], cwd=ROOT)
        assert hashlib.sha256(historical_doc).hexdigest() == expected
    else:
        assert digest(ROOT / path) == expected, path
assert subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() == worker['base_commit']
assert subprocess.check_output(['git', 'branch', '--show-current'], cwd=ROOT, text=True).strip() == 'gomad'
assert subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT, text=True) == 'AGENTS.md\n'
historical_doc = subprocess.check_output(['git', 'show', worker['base_commit'] + ':AGENTS.md'], cwd=ROOT)
current_doc = (ROOT / 'AGENTS.md').read_bytes()
expected_doc = historical_doc.decode()
for before, after in (
    ('thinking scout: claude-opus-5-5 at high\n\n### Codex', 'thinking scout: claude-opus-5-5 at high\n\nresearch: claude-fable-5-1 at high\n\n### Codex'),
    ('thinking scout: gpt-6.1-sol at high\n\nDemanding tasks', 'thinking scout: gpt-6.1-sol at high\n\nresearch: gpt-6-astra at high\n\n`research` includes codebase surveys, audits, tool and literature evaluations, and\nweb research, including investigations supporting implementation or design. Always\nuse the matching research model above for research tasks. Spec writing, task\nbreakdown, and design decisions use the thinking scout.\n\nDemanding tasks'),
):
    assert expected_doc.count(before) == 1
    expected_doc = expected_doc.replace(before, after)
assert current_doc == expected_doc.encode()
assert digest(ROOT / 'AGENTS.md') == '3ce85bf0fd1be8f06398b71eeae1959f0ee7081c885a431bb7764c79c39d602f'
assert subprocess.check_output(['git', 'show', ':AGENTS.md'], cwd=ROOT) == current_doc
subprocess.run(['git', 'diff', '--cached', '--check'], cwd=ROOT, check=True)
authorized_document_delta = dict(path='AGENTS.md', historical_sha256=hashlib.sha256(historical_doc).hexdigest(), current_and_index_sha256=digest(ROOT / 'AGENTS.md'), exact_added_lines=9, staged_paths=['AGENTS.md'], historical_audit_substitutions=1, archived_audits_unchanged=True)
protected = worker['protected']
assert protected == dict(files=1042, aggregate_sha256='b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61')
tools = json.loads((OUT / 'allocation-repair-package.json').read_text())['tools']
commands = {Path(r['path']).stem.removeprefix('allocation-repair-'): shlex.split(r['command']) for r in worker['command_receipts']}
environment = json.loads((OUT / 'review-allocation-static.json').read_text())['environment']
assert [environment[k] for k in ('GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS')] == ['off', 'local', 'off', '']
assert environment['GOMADSEED'] is None and environment['GOMAD3_CHILD_SEED'] is None
assert environment['PATH'].split(':')[0] == '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin'
logs, receipts = {}, []
suffixes = ['package', 'focused', 'boundaries', 'broader', 'consumer', 'lint', 'errortype', 'validate', 'static', 'worker-audit']
for suffix in suffixes:
    path = OUT / ('review-allocation-' + suffix + '.json')
    receipt = json.loads(path.read_text())
    assert receipt['exit'] == (1 if suffix == 'lint' else 0)
    assert receipt['stable'] and not receipt['timed_out'] and receipt['timeout_seconds'] is None
    assert receipt['source_before'] == receipt['source_after'] == final
    assert receipt['protected_before'] == receipt['protected_after'] == protected
    assert receipt['tools'] == tools
    for tool, expected in tools.items():
        assert digest(tool) == expected
    assert receipt['config_sha256'] == digest(ROOT / '.github/.golangci.yml') == '2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43'
    runner = 'review-allocation-audit-runner.py' if suffix == 'worker-audit' else 'review-allocation-runner.py'
    assert receipt['runner_sha256'] == digest(OUT / runner)
    assert receipt['cwd'] == str(ROOT if suffix == 'worker-audit' else ROOT / 'tools/gomad3')
    assert receipt['environment'] == environment
    expected_command = ['python3', str(OUT / 'review-allocation-worker-audit.py')] if suffix == 'worker-audit' else commands[suffix]
    assert receipt['command'] == expected_command
    start, end = [datetime.datetime.fromisoformat(receipt[k]) for k in ('started', 'ended')]
    assert start.tzinfo and end.tzinfo and receipt['elapsed_seconds'] >= 0
    assert abs((end-start).total_seconds() - receipt['elapsed_seconds']) < 1
    assert receipt['log'] == 'review-allocation-' + suffix + '.log'
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    logs[suffix] = (OUT / receipt['log']).read_text()
    receipts.append(dict(path=path.name, sha256=digest(path), command=receipt['command'], cwd=receipt['cwd'], environment=receipt['environment'], started=receipt['started'], ended=receipt['ended'], elapsed_seconds=receipt['elapsed_seconds'], timeout_seconds=None, exit=receipt['exit'], tools=receipt['tools'], config_sha256=receipt['config_sha256'], log=receipt['log'], log_sha256=receipt['log_sha256']))

def bodies(log):
    return dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]*/[^\n]+)\n(.*?)(?=^=== RUN|\Z)', log, re.M | re.S))

reference = bodies((OUT / 'allocation-repair-causal-green.log').read_text())
assert len(reference) == 67
aliases = {'TestErrorProvenanceAllocationAliases/' + family + '/' + mode for family in ('new-array', 'new-function', 'new-map', 'new-slice', 'map-literal', 'array-address-literal') for mode in ('dirty', 'clean')}
assert len(aliases) == 12 and aliases <= set(reference)
unknown = {'TestErrorProvenanceWriter/unknown-return', 'TestErrorProvenanceEmptySliceAlias/unknown-interface-elements', 'TestErrorProvenanceEmptyMapAlias/unknown-interface-elements', 'TestErrorProvenanceAllocationControls/unknown-interface-field'}
dirty_count = 0
for body in reference.values():
    dirty_count += 'actual callbacks=1 ' in body
assert dirty_count == 32
for suffix, count in (('package', 26), ('focused', 16)):
    log = logs[suffix]
    assert len(re.findall(r'^--- PASS:', log, re.M)) == count
    assert len(re.findall(r'^=== RUN   [^/\n]+$', log, re.M)) == count
    assert not re.search(r'^(--- FAIL|FAIL)', log, re.M)
    cases = bodies(log)
    assert set(cases) == set(reference)
    assert log.count('stock-host causal fixture:') == 67
    assert log.count('package edges=0 effects=') == 134
    assert len(re.findall(r'^    --- PASS: TestErrorProvenance', log, re.M)) == 67
    for case, body in cases.items():
        assert '--- PASS: TestBehavior' in body
        counters = re.findall(r'actual callbacks=(\d+) writer count=(\d+)', body)
        assert counters == re.findall(r'actual callbacks=(\d+) writer count=(\d+)', reference[case])
        assert len(counters) == 1 and counters[0][0] in ('0', '1')
        for platform in ('linux amd64', 'darwin arm64'):
            pattern = r'metadata \{' + platform + r'\} package edges=0 effects=(.*)'
            effects = re.findall(pattern, body)
            expected = re.findall(pattern, reference[case])
            assert len(effects) == len(expected) == 1
            effect = effects[0]
            if counters[0][0] == '1':
                callback = re.search(r'canonicaljson\.[\w.]+|dependency\.Leaf\.Error', expected[0]).group()
                assert all(part in effect for part in ('host-effect', 'record.Check', callback, 'time.Now'))
                assert 'unresolved-effect' not in effect
            elif case in unknown:
                assert all(part in effect for part in ('unresolved-effect', 'record.Check', 'unknown dynamic callback receiver'))
            else:
                assert effect == '[]', (suffix, case, effect)
    for name in ('TestEffectCallbackContextsAndUnwrapReturns', 'TestDependencyInitialization', 'TestThirdPartyInitialization', 'TestStandardStartupIdentity', 'TestMemorySummarySourceIdentity', 'TestCallbackContainerMutations', 'TestRangeAssignmentSlots', 'TestImplicitCallbackPrecedence', 'TestPureMemoryFormattingAndJSON'):
        assert re.search(r'^--- PASS: ' + name + r' ', log, re.M)
for suffix, names in (('boundaries', worker['required_boundaries']), ('broader', ['TestPureModulesHaveNoHostEffects', 'TestExactModuleEdges', 'TestHostPackageVet'])):
    assert len(re.findall(r'^--- PASS:', logs[suffix], re.M)) == len(names)
    assert len(re.findall(r'^=== RUN   [^/\n]+$', logs[suffix], re.M)) == len(names)
    for name in names:
        assert len(re.findall(r'^=== RUN   ' + name + r'$', logs[suffix], re.M)) == 1
        assert re.search(r'^--- PASS: ' + name + r' ', logs[suffix], re.M)
for platform in ('darwin/arm64', 'linux/amd64', 'linux/arm64'):
    assert re.search(r'^    --- PASS: TestHostPackageVet/' + platform + r' ', logs['broader'], re.M)
assert len(re.findall(r'^--- PASS:', logs['consumer'], re.M)) == 30
assert len(re.findall(r'^=== RUN   [^/\n]+$', logs['consumer'], re.M)) == 30
assert logs['lint'] == (OUT / 'baseline-lint.log').read_text() == (OUT / 'allocation-repair-lint.log').read_text()
assert logs['errortype'] == logs['static'] == ''
assert json.loads(logs['worker-audit']) == historical
assert not (ROOT / 'tools/gomad3/.toolchain/bin/go').exists()
assert inventory() == frozen
assert (ROOT / 'AGENTS.md').read_bytes() == subprocess.check_output(['git', 'show', ':AGENTS.md'], cwd=ROOT) == current_doc
assert subprocess.check_output(['git', 'diff', '--cached', '--name-only'], cwd=ROOT, text=True) == 'AGENTS.md\n'
result = dict(task=worker['task'], base_commit=worker['base_commit'], verdict='SOURCE_PROGRESS_COMMIT_ONLY', actionable_findings=[], closed_findings=['fresh make slice alias', 'fresh make map alias', 'empty map literal alias', 'six admitted fresh allocation alias families'], historical=historical, new_review_receipts=receipts, source=final, protected=protected, architecture_top_level_tests=26, focused_top_level_tests=16, consumer_top_level_tests=30, causal_fixtures=67, metadata_observations=134, required_boundary_tests=5, broader_boundary_tests=3, dirty_causal_cases=32, unknown_fail_closed_cases=4, lint=dict(inherited=4, introduced=0, resolved=0), errortype=0, qualified_native=False, formal_review=False, original_acceptance='R8/R18/R19/task19/fn105D4/predecessors/task21/first-baseline/full/completion/formal/both-native/affected-consumer open', requested_reviewer='gpt-6.1-sol/high', actual_execution_model=None, same_configured_codex_family=True, tier='session (jev-unavailable(no_key))', writes=0, live_handles=0, delegates=0, report_sha256=digest(OUT / 'allocation-source-review.md'), review_audit_sha256=digest(__file__), review_runner_sha256=digest(OUT / 'review-allocation-runner.py'))
result['authorized_document_delta'] = authorized_document_delta
result['worker_audit_wrapper_sha256'] = digest(OUT / 'review-allocation-worker-audit.py')
result['review_audit_runner_sha256'] = digest(OUT / 'review-allocation-audit-runner.py')
print(json.dumps(result, indent=2))
