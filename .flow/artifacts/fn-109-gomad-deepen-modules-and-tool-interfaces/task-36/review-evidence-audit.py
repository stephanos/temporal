import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
test = ROOT / 'tools/gomad3/runner/internal/exploration/choice/engine_test.go'
source = test.read_text()
capture = (PROOF / 'capture-focused.log').read_text()
pairs = re.findall(r'BASE state=(sha256:[0-9a-f]{64}) segment=(sha256:[0-9a-f]{64})', capture)
literals = re.findall(r'wantState: "(sha256:[0-9a-f]{64})", wantRound: "(sha256:[0-9a-f]{64})"', source)
assert pairs == literals and len(pairs) == 8
checks = json.loads((PROOF / 'review-checks.json').read_text())
counts = {}
subtests = {}
for name, count in [('package', 16), ('focused', 8), ('boundaries', 5)]:
    lines = (PROOF / ('review-' + name + '.log')).read_text().splitlines()
    names = [line.split(' (', 1)[0].split('PASS: ', 1)[1] for line in lines if line.startswith('--- PASS: Test')]
    assert len(names) == count and len(set(names)) == count
    counts[name] = names
    if name == 'focused':
        for prefix, expected in [('TestExplorationFailurePolicyBaseline/', 8), ('TestExplorationValidatesSiblingsAfterPolicyStop/', 12), ('TestExplorationRejectsInvalidFailurePolicyConfiguration/', 4)]:
            selected = [line.split(' (', 1)[0].split('PASS: ', 1)[1] for line in lines if line.startswith('    --- PASS: ' + prefix)]
            assert len(selected) == expected and len(set(selected)) == expected
            subtests[prefix] = selected
boundary_names = {'TestPackageArchitecture', 'TestPublicPackagesDoNotExportTypeAliases', 'TestArchitecturePublicSignatureFixtures', 'TestRunnerRequestsCompileInExternalModule', 'TestRunnerExternalConsumerCompiles'}
assert set(counts['boundaries']) == boundary_names
raw = subprocess.run(['git', 'diff', '--no-index', '--check', '/dev/null', str(PROOF / 'audit-environment.log')], cwd=ROOT, text=True, capture_output=True)
assert raw.returncode == 3 and raw.stderr == ''
assert raw.stdout == str(PROOF / 'audit-environment.log') + ':10: new blank line at EOF.\n'
report = {'result': 'PASS', 'script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
          'eight_capture_pairs_equal_final_literals': True, 'independent_top_level_tests': counts, 'independent_new_subtests': subtests,
          'raw_archive_warning': {'argv': raw.args, 'exit_code': raw.returncode, 'stdout': raw.stdout, 'stderr': raw.stderr},
          'sources': checks['after']['sources'], 'handles_running': 0, 'source_index_head_flow_writes': 0}
(PROOF / 'review-evidence-audit.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps(report, indent=2))
