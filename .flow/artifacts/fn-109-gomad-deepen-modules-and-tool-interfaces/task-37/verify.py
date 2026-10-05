import hashlib
import json
from pathlib import Path
import re

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37'
PIN = json.loads((OUT / 'pin.json').read_text())
ADMISSION = json.loads((OUT / 'root-admission.json').read_text())

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def replace_once(text, before, after):
    assert text.count(before) == 1, repr(before)
    return text.replace(before, after, 1)

writer = (ROOT / ADMISSION['product_paths'][0]).read_text()
reconstructed = replace_once(writer,
    'func WriteQualificationReport(artifactRoot string, report QualificationReport) (reportPath string, retErr error) {',
    'func WriteQualificationReport(artifactRoot string, report QualificationReport) (string, error) {')
reconstructed = replace_once(reconstructed,
    '''defer func() {
		if removeErr := os.Remove(temporaryPath); removeErr != nil && (reportPath == "" || !errors.Is(removeErr, os.ErrNotExist)) {
			if retErr == nil {
				retErr = removeErr
			} else {
				retErr = errors.Join(retErr, removeErr)
			}
		}
	}()''', 'defer os.Remove(temporaryPath)')
for message in ('make qualification report private', 'write qualification report', 'sync qualification report'):
    reconstructed = replace_once(reconstructed,
        '\t\tif closeErr := temporary.Close(); closeErr != nil {\n\t\t\treturn "", errors.Join(fmt.Errorf("' + message + ': %w", err), closeErr)\n\t\t}',
        '\t\ttemporary.Close()')
reconstructed = replace_once(reconstructed,
    'func OpenQualificationReport(path string) (report QualificationReport, retErr error) {',
    'func OpenQualificationReport(path string) (QualificationReport, error) {')
reconstructed = replace_once(reconstructed,
    '''defer func() {
		if closeErr := file.Close(); closeErr != nil {
			report = QualificationReport{}
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()''', 'defer file.Close()')
assert reconstructed == (OUT / 'base-qualification.go').read_text(), 'original production bodies/comments changed'

tests = (ROOT / ADMISSION['product_paths'][1]).read_text()
assert tests == (OUT / 'controls-qualification_test.go').read_text(), 'final controls differ from pre-production controls'
start = tests.index('\nfunc TestQualificationReportStoragePreservation(')
end = tests.index('\nfunc successfulEvidence()', start)
original_tests = tests[:start] + tests[end:]
for added in ('bytes', 'errors', 'reflect', 'strings', 'syscall', 'go.temporal.io/server/tools/gomad3/internal/canonicaljson'):
    original_tests = replace_once(original_tests, '\t"' + added + '"\n', '')
assert original_tests == (OUT / 'base-qualification_test.go').read_text(), 'old test bodies/assertions/comments changed'
for name, path in zip(('base-qualification.go', 'base-qualification_test.go'), ADMISSION['product_paths']):
    assert sha(OUT / name) == ADMISSION['baseline_source_sha256'][path]

protected = {p: value for p, value in PIN['source'].items() if p not in ADMISSION['product_paths']}
for path, value in protected.items():
    assert sha(ROOT / path) == value, 'protected input changed: ' + path
for name, tool in ADMISSION['tools'].items():
    assert sha(Path(tool['path'])) == tool['sha256'], 'tool changed: ' + name
assert sha(OUT / 'root-admission.json') == '1884fa3f4a874020e582dd9fe858a5e5de53f539cb52992af4023cae5c31b84e'

def findings(log):
    return [{'path': m[1], 'line': int(m[2]), 'column': int(m[3]), 'message': m[4], 'linter': m[5]} for m in re.finditer(r'^(tools/gomad3/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$', log, re.M)]

baseline = findings((OUT / 'baseline-lint.log').read_text())
final = findings((OUT / 'final-lint-corrected.log').read_text())
assert len(baseline) == 6 and len(final) == 1
expected_lines = (248, 250, 254, 258, 289)
resolved = [entry for entry in baseline if entry['linter'] == 'errcheck']
assert tuple(entry['line'] for entry in resolved) == expected_lines
assert final == [entry for entry in baseline if entry['linter'] == 'gci']
checked_lines = [writer[:match.start()].count('\n') + 1 for match in re.finditer(r'if (?:removeErr := os.Remove\(temporaryPath\)|closeErr := temporary.Close\(\)|closeErr := file.Close\(\))', writer)]
assert len(checked_lines) == 5
repair_map = [{'original_line': entry['line'], 'checked_call_line': line, 'path': entry['path'], 'message': entry['message']} for entry, line in zip(resolved, checked_lines)]
assert (OUT / 'baseline-lint.json').read_text()
before = json.loads((OUT / 'baseline-lint.json').read_text())
after = json.loads((OUT / 'final-lint-corrected.json').read_text())
assert before['command'] == after['command'] and before['tools'] == after['tools'] and before['environment'] == after['environment']
assert before['exit_code'] == after['exit_code'] == 1

controls = {'TestQualificationReportStoragePreservation', 'TestQualificationReportWriteValidationPreservation', 'TestQualificationReportReadValidationPreservation', 'TestWriteReportRetainsCanonicalPrivateFile', 'TestWriteRejectsInconsistentDeterministicOutcome', 'TestBuildReportClassifiesExactChoiceReplayWithEvidenceDivergence', 'TestDiagnosticsRejectCorruptedSavedReportBindings', 'TestDiagnosticsOffReportFieldsRemainAbsent'}
boundaries = {'TestPackageArchitecture', 'TestPublicPackagesDoNotExportTypeAliases', 'TestArchitecturePublicSignatureFixtures', 'TestRunnerRequestsCompileInExternalModule', 'TestRunnerExternalConsumerCompiles'}
required = ('saved-base-controls', 'verified-controls', 'verified-package', 'verified-consumers', 'verified-errortype', 'verified-boundaries', 'verified-purity-edges', 'verified-gofmt')
counts = {}
final_product = {p: sha(ROOT / p) for p in ADMISSION['product_paths']}
for name in required:
    receipt = json.loads((OUT / (name + '.json')).read_text())
    log = (OUT / receipt['log']).read_text()
    assert receipt['exit_code'] == 0 and receipt['status'] == 'terminal' and receipt['source_stable']
    assert sha(OUT / receipt['log']) == receipt['log_sha256']
    if name.startswith('verified-'):
        assert receipt['source_before']['product'] == final_product, 'check not on final source: ' + name
    observed = set(re.findall(r'^--- PASS: (Test[^/ ]+) ', log, re.M))
    started = set(re.findall(r'^=== RUN   (Test[^/\s]+)$', log, re.M))
    if name in ('saved-base-controls', 'verified-controls'):
        assert observed == started == controls, 'wrong control surface: ' + name
    if name == 'verified-boundaries':
        assert observed == started == boundaries, 'wrong boundary surface'
    if name in ('verified-package', 'verified-consumers', 'verified-purity-edges'):
        assert len(observed) > 0 and observed == started, 'missing/skipped test surface: ' + name
    if name == 'verified-gofmt':
        assert log == ''
    counts[name] = len(observed)

if (OUT / 'freeze.json').exists():
    for path, value in json.loads((OUT / 'freeze.json').read_text())['files'].items():
        assert sha(OUT / path) == value, 'frozen proof changed: ' + path

print(json.dumps({'status': 'SOURCE_PROGRESS_ONLY', 'base_commit': PIN['base_commit'], 'source_sha256': final_product, 'original_production_reconstructed': True, 'original_tests_reconstructed': True, 'protected_inputs_unchanged': len(protected), 'saved_baseline_controls_equal_final': True, 'lint': {'baseline': baseline, 'resolved': resolved, 'final': final, 'introduced': [], 'repair_map': repair_map}, 'passed_top_level_tests': counts, 'required_acceptance_state': 'open; task remains in_progress pending root review and original full/formal/native qualification', 'live_worker_handles': 0}, indent=2, sort_keys=True))
