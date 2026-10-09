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

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

before, after = original(CLI).decode(), (ROOT / CLI).read_text()
pattern = r'(?m)^(\t*)if _, err := (fmt\.Fprintf\(stdout,.*\)); err != nil \{\n\1\treturn 3\n\1}\n'
sites = []
restored = after
for expression in [
    'fmt.Fprintf(stdout, "%s\\n", encoded)',
    'fmt.Fprintf(stdout, "gomad doctor: available=%t host=%s go=%s toolchain=%s runner=%s boundary=%s\\n", report.Available, report.Host, report.GoVersion, report.ToolchainBuild, report.RunnerBuild, report.BoundaryManifestVersion)',
    'fmt.Fprintf(stdout, "%-10s %-5s %s\\n", check.Name, check.Status, check.Detail)',
]:
    start, end = restored.index('func (app application) runDoctor('), restored.index('type exploreDependencies struct')
    matches = [match for match in re.finditer(pattern, restored) if match[2] == expression and start < match.start() < end]
    assert len(matches) == 1, expression
    match = matches[0]
    restored = restored[:match.start()] + match[1] + match[2] + '\n' + restored[match.end():]
    sites.append({'expression': expression, 'failed_write_status': 3})
assert restored == before
predecessor_proof = json.loads((OUT.parent / 'task-53/source-proof.json').read_text())
assert next(row['candidate_sha256'] for row in predecessor_proof['production_files'] if row['path'] == CLI) == digest(before.encode())
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration'], cwd=ROOT).split(b'\0')
preserved = {}
for value in paths:
    if not value:
        continue
    path = value.decode()
    if path != CLI:
        expected, actual = original(path), (ROOT / path).read_bytes()
        assert expected == actual, path
        preserved[path] = digest(actual)
turbo = {path: digest((ROOT / path).read_bytes()) for path in ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md']}
assert turbo == predecessor_proof['protected_turbo_sha256']

def findings(output):
    return re.findall(r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)', output)

def lint_delta(name, expected_before, expected_after):
    old = findings((OUT.parent / ('task-53/' + name + '.stdout')).read_text())
    new = findings((OUT / (name + '.stdout')).read_text())
    old_keys = collections.Counter((path, message, statement) for path, _, _, message, statement, _ in old)
    new_keys = collections.Counter((path, message, statement) for path, _, _, message, statement, _ in new)
    removed, added = old_keys - new_keys, new_keys - old_keys
    assert (len(old), len(new), sum(removed.values()), sum(added.values())) == (expected_before, expected_after, 3, 0)
    assert all(path == CLI and 'not checked' in message and any(expression in statement for expression in [row['expression'] for row in sites]) for path, message, statement in removed)
    return {'before': len(old), 'after': len(new), 'removed': sum(removed.values()), 'added': sum(added.values()),
            'residual_message_statement_multiset_preserved': True,
            'baseline_receipt': '../task-53/' + name + '.json', 'candidate_receipt': name + '.json'}

def observations(name):
    events = [json.loads(line) for line in (OUT / (name + '.stdout')).read_text().splitlines() if line.startswith('{')]
    tests = [event for event in events if event.get('Test') and event['Action'] in ['pass', 'fail', 'skip']]
    return {'counts': dict(collections.Counter(event['Action'] for event in tests)),
            'failed_tests': [event['Test'] for event in tests if event['Action'] == 'fail'],
            'skipped_tests': [event['Test'] for event in tests if event['Action'] == 'skip'],
            'package_results': [{key: event[key] for key in ['Package', 'Action', 'Elapsed'] if key in event}
                                for event in events if not event.get('Test') and event['Action'] in ['pass', 'fail', 'skip']]}

names = ['focused-before', 'focused-before-final', 'focused-after', 'full-ordinary-cli', 'affected-vet',
         'standalone-errortype', 'affected-configured-lint', 'architecture-source-sets', 'runner-ownership',
         'generated-validation', 'format-check', 'make-fast-task-base', 'make-gomad-original-base', 'tool-identity']
receipts = []
for name in names:
    record = json.loads((OUT / (name + '.json')).read_text())
    assert record['source_unchanged'] and record['terminal'], name
    for suffix in ['stdout', 'stderr']:
        assert digest((OUT / (name + '.' + suffix)).read_bytes()) == record[suffix + '_sha256'], name
    receipts.append({'receipt': name + '.json', **record})
candidate = next(row for row in receipts if row['receipt'] == 'focused-after.json')
assert all(row['source_before_sha256'] == candidate['source_before_sha256'] for row in receipts[2:])
proof = {'base_commit': BASE, 'production_path': CLI, 'base_sha256': digest(before.encode()),
         'candidate_sha256': digest(after.encode()), 'three_checks': sites,
         'complete_original_bytes_reconstructed': True, 'old_tests_and_other_source_preserved_count': len(preserved),
         'preserved_source_manifest_sha256': digest(json.dumps(preserved, sort_keys=True).encode()),
         'protected_turbo_sha256': turbo,
         'lint': {'affected': lint_delta('affected-configured-lint', 5, 2),
                  'original_base': lint_delta('make-gomad-original-base', 99, 96)}}
ordinary = observations('full-ordinary-cli')
evidence = {
    'task': 'fn-109.54', 'status': 'in_progress', 'branch': 'gomad', 'base_commit': BASE, 'commits': [], 'prs': [],
    'commit_owner': 'root; worker never stages or commits',
    'tests': [row['command'] for row in receipts], 'gates': receipts,
    'baseline': {'status': 'red', 'focused_receipt': 'focused-before-final.json',
                 'meaningful_failures': observations('focused-before-final'),
                 'inherited_required_red': '../task-53/make-gomad-original-base.json and affected-configured-lint.json (exact CLI source reused, not rerun)',
                 'initial_fixture_error': 'focused-before.json retains the corrected root-error literal missing CLI; this is separate from the five meaningful write failures.'},
    'source_proof': proof, 'frozen_source_sha256': candidate['source_before_sha256'], 'tools': candidate['tools'],
    'execution_platform': 'stock Go1.27.1 developmental linux/arm64; private /tmp overlayfs; no native qualification',
    'routing': {'tier_line': 'Tier: session (jev-unavailable(no_key))', 'implementer_preference': 'gpt-6.1-sol at high', 'actual_execution_metadata': 'unavailable'},
    'focused_after': observations('focused-after'), 'full_ordinary_cli': ordinary,
    'healthy_controls': 'Exported Run; independent complete literal text and JSON skeleton/field ordering/adapter pins. Expected dynamic inputs: real host, absolute fixture paths, independent SHA256 of actual os.Executable bytes, public deterministicio profile identity. Expectations never invoke Check.',
    'runtime_controls': 'Genuine read-only os.File EBADF on selected attempts1(JSON/headline),2(first row),11(interior row),20(final row),21(existing footer); exact attempts/counts, successful prefix, empty stderr and probe cleanup. Flag/argument/root errors retain status2, zero stdout attempts and unprobed artifacts.',
    'ordinary_failures': [
        {'test': 'TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput', 'cause': 'Absent patched .toolchain/bin/go produces status3 before expected read-only module status2; ordinary source gate remains open.'},
        {'test': 'TestRunDoctorReportsAvailableContractAsJSON', 'cause': 'Real linux/arm64 unsupported; old fixture go1.26.4 does not prove current supported-host status0.'},
        {'test': 'TestCheckReportsAvailableContract', 'cause': 'Real linux/arm64 unsupported; available contract requires supported host.'},
        {'package': 'cmd/gomad', 'cause': 'Native e2e TestMain CLI build fails before collecting tests because patched .toolchain/bin/go is absent; deferred native owner remains unchanged.'},
    ],
    'remaining_source_gates': 'Unfiltered affected configured lint exit1 (replay stdout and application ST1005); original-base integrated lint exit2 (96 findings); full ordinary affected tests exit1. Formal acceptance remains open.',
    'integrated_errortype': 'Unreached after original-base configured lint failure; standalone/task-base fast passes do not replace it.',
    'coverage_gaps': 'Healthy availability status0 and native Runner/toolchain execution are not proved. Probe directory was created by healthy control and checked empty throughout; failure cases do not individually prove creation from absence. No simulated supported host, native pass or universal report-failure claim.',
    'review': 'Root owns fresh independent source-progress review; worker issued no verdict and no formal implementation review.',
    'native': 'fn149/fn128 deferred and unverified; no PR, push, CI, native revival or publication authority.',
    'predecessor': '../task-53/handover.md and independent-review.md',
}
for name, record in [('source-proof.json', proof), ('evidence.json', evidence)]:
    destination = OUT / name
    if destination.exists():
        raise SystemExit('Refusing to overwrite retained evidence')
    destination.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({'ordinary': ordinary, 'lint': proof['lint'], 'source': evidence['frozen_source_sha256']}))
