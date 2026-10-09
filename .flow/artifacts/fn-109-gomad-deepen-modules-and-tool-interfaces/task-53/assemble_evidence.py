import collections
import hashlib
import json
import pathlib

OUT = pathlib.Path(__file__).resolve().parent
names = ['preservation-before', 'preservation-before-literals-corrected', 'preservation-before-expanded',
         'preservation-before-final', 'preservation-after', 'full-ordinary-cli', 'affected-vet',
         'standalone-errortype', 'affected-configured-lint', 'architecture-source-sets', 'generated-validation',
         'make-fast-task-base', 'make-gomad-original-base', 'format-check', 'tool-identity', 'runner-ownership']
receipts = []
for name in names:
    record = json.loads((OUT / (name + '.json')).read_text())
    assert record['source_unchanged'] and record['terminal'], name
    for suffix in ['stdout', 'stderr']:
        assert hashlib.sha256((OUT / (name + '.' + suffix)).read_bytes()).hexdigest() == record[suffix + '_sha256'], name
    receipts.append({'receipt': name + '.json', **record})
def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    tests = [event for event in events if event.get('Test') and event['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(event['Action'] for event in tests)),
            'top_level_counts': dict(collections.Counter(event['Action'] for event in tests if '/' not in event['Test'])),
            'failed_tests': [event['Test'] for event in tests if event['Action'] == 'fail'],
            'skipped_tests': [event['Test'] for event in tests if event['Action'] == 'skip'],
            'package_results': [{key: event[key] for key in ['Package', 'Action', 'Elapsed'] if key in event}
                                for event in events if not event.get('Test') and event['Action'] in ['pass', 'fail', 'skip']]}
proof = json.loads((OUT / 'source-proof.json').read_text())
baseline = json.loads((OUT / 'preservation-before-final.json').read_text())
candidate = json.loads((OUT / 'preservation-after.json').read_text())
ordinary = observations('full-ordinary-cli')
evidence = {
    'task': 'fn-109-gomad-deepen-modules-and-tool-interfaces.53',
    'status': 'in_progress', 'branch': 'gomad', 'base_commit': (OUT / 'base_commit').read_text().strip(),
    'commits': [], 'prs': [], 'commit_owner': 'root; worker does not stage or commit',
    'tests': [row['command'] for row in receipts], 'gates': receipts,
    'baseline': {'preservation': 'green before product edits', 'receipt': 'preservation-before-final.json',
                 'source_sha256': baseline['source_before_sha256'],
                 'meaningful_defect': '../task-52/make-gomad-original-base.json and .stdout (source-identical CLI RED145; unchanged red not rerun)',
                 'initial_fixture_failures': 'preservation-before.json has two new diagnostic literal mistakes; preservation-before-expanded.json has one new replay stdout literal mistake. Corrected on unchanged product before the final baseline. These are fixture development failures, not the lint defect.'},
    'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'],
    'source_fingerprint_scope': 'Tracked tools/gomad3, tools/gomad3integration, gomad3.yml, lint configuration, Makefile, AGENTS.md, MILESTONES.md, lintcode plus additive terminal_diagnostics_test.go',
    'execution_platform': 'Stock Go1.27.1 developmental linux/arm64; fresh private overlayfs /tmp TMPDIR; no supported native qualification',
    'routing': {'tier_line': 'Tier: session (jev-unavailable(no_key))', 'requested_model': 'gpt-6.1-sol at high', 'actual_execution_metadata': 'not exposed'},
    'source_proof': 'source-proof.json', 'lint': proof['lint'],
    'preservation': {'before': observations('preservation-before-final'), 'after': observations('preservation-after'),
                     'real_read_only_file_EBADF': True, 'exact_attempted_bytes': True, 'later_secondary_attempts': True,
                     'public_failure_files_unchanged': True, 'completed_private_callback_marker_files_retained': True,
                     'native_artifact_publication_proved': False, 'original_tests_unchanged': True,
                     'all_nonadmitted_production_bytes_restored': True, 'individual_runtime_gap_count': 14},
    'full_ordinary_cli': ordinary,
    'ordinary_failures': [
        {'test': 'TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput',
         'observation': 'Expected invalid-input status2; actual3 because default patched .toolchain/bin/go is absent before read-only module analysis.',
         'ownership': 'Retained affected ordinary source gate remains open. This ordinary source failure is not waived by native transfer; no filtered pass or stock-toolchain substitution.'},
        {'test': 'TestRunDoctorReportsAvailableContractAsJSON',
         'observation': 'Expected available/status0; actual1 because genuine host linux/arm64 is unsupported (existing fixture toolchain metadata says go1.26.4).',
         'ownership': 'Affected ordinary source observation remains red; native available-contract proof awaits supported execution under fn149/fn128. Fixture/host policy unchanged.'},
        {'test': 'TestCheckReportsAvailableContract',
         'observation': 'Available-contract unit fixture rejected genuine linux/arm64 host.',
         'ownership': 'Affected ordinary source observation remains red; native available-contract proof awaits supported execution under fn149/fn128. No synthetic host success.'},
        {'package': 'go.temporal.io/server/tools/gomad3/cmd/gomad',
         'observation': 'TestMain in e2e_test.go failed to build CLI with absent .toolchain/bin/go; no package tests collected.',
         'ownership': 'Native CLI target execution and aggregate host gate belong to deferred fn149/fn128. The actual full command still failed and is never recorded as an ordinary or native pass.'},
    ],
    'remaining_source_gates': 'Unfiltered affected configured lint exit1 with5 excluded findings; original-base integrated lint exit2 with99 residuals; full ordinary affected command exit1. Formal review/completion remain open.',
    'integrated_errortype': 'Unreached after original-base configured lint fails; standalone and task-base fast passes do not replace it.',
    'excluded_output_findings': 'Doctor stdout3 and replay verified stdout1 are unchanged and need separate admitted output owners; application ST1005 also unchanged.',
    'architecture_source_sets': 'architecture-source-sets.json passes all six required selections, including TestHostPackageVet/darwin/arm64, linux/amd64 and developmental linux/arm64 inventories. runner-ownership.json separately passes private execution injection boundary.',
    'review': 'Fresh independent source-progress review pending with root; worker issued no verdict or formal review.',
    'native': 'fn149/fn128 remain deferred and unverified. No native pass, qualification bound, publication, PR, push or CI authority.',
    'protected_turbo_sha256': proof['protected_turbo_sha256'],
}
destination = OUT / 'evidence.json'
if destination.exists():
    raise SystemExit('Refusing to overwrite retained evidence')
destination.write_text(json.dumps(evidence, indent=2) + '\n')
print(json.dumps({'ordinary': ordinary, 'lint': proof['lint'], 'candidate': evidence['frozen_source_sha256']}))
