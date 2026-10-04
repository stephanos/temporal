import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
admission = json.loads((PROOF / 'root-admission.json').read_text())
base = admission['base_commit']
production, tests = admission['source_paths']


def original(path):
    return subprocess.check_output(['git', 'show', base + ':' + path], cwd=ROOT)


old = b'''\t\t\tswitch next.Config.FailurePolicy {
\t\t\tcase PolicyFirst:
\t\t\t\tnext.StopReason = StopFirstFailure
\t\t\t\tpolicyStopped = true
\t\t\tcase PolicyBudget:
\t\t\t\tif uint64(len(next.FailureSignatures)) >= next.Config.FailureBudget {
\t\t\t\t\tnext.StopReason = StopFailureBudget
\t\t\t\t\tpolicyStopped = true
\t\t\t\t}
\t\t\t}'''
new = b'''\t\t\tif next.Config.FailurePolicy == PolicyFirst {
\t\t\t\tnext.StopReason = StopFirstFailure
\t\t\t\tpolicyStopped = true
\t\t\t} else if next.Config.FailurePolicy == PolicyBudget && uint64(len(next.FailureSignatures)) >= next.Config.FailureBudget {
\t\t\t\tnext.StopReason = StopFailureBudget
\t\t\t\tpolicyStopped = true
\t\t\t}'''
before = original(production)
after = (ROOT / production).read_bytes()
assert before.count(old) == 1 and after.count(new) == 1
assert after.replace(new, old, 1) == before
assert (ROOT / tests).read_bytes().startswith(original(tests))
assert all(hashlib.sha256((ROOT / p).read_bytes()).hexdigest() == h for p, h in admission['protected_files'].items())
controls = json.loads((PROOF / 'controls-focused.receipt.json').read_text())
final = json.loads((PROOF / 'final-package.receipt.json').read_text())
assert controls['source_before']['sources'][production] == admission['source_before_sha256'][production]
assert controls['source_before']['sources'][tests] == final['source_before']['sources'][tests]
assert controls['exit_code'] == final['exit_code'] == 0
baseline_diagnostics = (PROOF / 'baseline-lint.log').read_text()
final_diagnostics = (PROOF / 'final-lint.log').read_text()
assert '* exhaustive: 1' in baseline_diagnostics
assert '0 issues.' in final_diagnostics
counts = {}
for phase in ('baseline', 'controls', 'final'):
    for gate in ('package', 'focused', 'boundaries'):
        log = PROOF / f'{phase}-{gate}.log'
        if log.exists():
            lines = log.read_text().splitlines()
            counts[f'{phase}-{gate}'] = sum(line.startswith('--- PASS: Test') for line in lines)
            assert counts[f'{phase}-{gate}'] > 0
report = {
    'base_commit': base,
    'production_reconstruction_exact': True,
    'original_test_file_byte_prefix': True,
    'protected_count': len(admission['protected_files']),
    'protected_unchanged': True,
    'literal_controls_passed_on_original_production': True,
    'literal_tests_unchanged_at_final_package': True,
    'test_counts_top_level': counts,
    'diagnostic_delta': {'baseline': 1, 'final': 0, 'resolved': 1, 'introduced': 0, 'residual': 0, 'rule': 'exhaustive'},
    'sources': {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in admission['source_paths']},
    'generator_inputs_affected': False,
    'generator_inspection': ['tools/gomad3/Makefile:6-9', 'tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:594-615'],
}
(PROOF / 'source-preservation.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps(report, indent=2))
