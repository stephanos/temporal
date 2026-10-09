import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BASE = (OUT / 'base_commit').read_text().strip()
PREFIX = 'tools/gomad3/cmd/gomad/internal/cli/'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

covered = {
    'application.go': {88: 'PrivateDispatch'},
    'campaign_shards.go': {36: 'PublicFailures/shard usage', 41: 'PublicFailures/shard syntax', 46: 'PublicFailures/shard installation', 63: 'CompletedOperations/execute-shard', 109: 'PublicFailures/merge usage', 117: 'PublicFailures/merge invalid plan'},
    'cli.go': {98: 'PublicFailures/usage', 136: 'PublicFailures/unknown', 364: 'PublicFailures/doctor usage', 442: 'CompletedOperations/explore', 611: 'PublicFailures/explore output', 619: 'PublicFailures/explore capability', 677: 'PublicFailures/explore strategy', 685: 'PublicFailures/explore seeds', 692: 'PublicFailures/explore regression', 700: 'PublicFailures/explore guidance', 708: 'PublicFailures/explore coverage', 722: 'PublicFailures/explore choices', 739: 'PublicFailures/explore target', 749: 'PublicFailures/explore working directory', 762: 'PublicFailures/explore installation', 999: 'PublicFailures/replay usage', 1004: 'PublicFailures/replay installation', 1013: 'ReplayClassifications/preflight', 1016: 'ReplayClassifications/host', 1025: 'CompletedOperations/replay'},
    'qualify.go': {181: 'CompletedOperations/qualify', 193: 'PublicFailures/qualify input and installation'},
    'resume.go': {44: 'PublicFailures/resume input', 52: 'PublicFailures/resume installation', 76: 'CompletedOperations/resume'},
}
gaps = {
    ('campaign_shards.go', 57): 'Classified operation error with reporter delivery failure: existing fake-dependency ordinary characterization; individual actual EBADF control not run. Real shard execution needs supported native Runner under fn149/fn128.',
    ('campaign_shards.go', 123): 'encoding/json failure for concrete CampaignMergeResult: no reachable malformed value is supplied by the validated public merge operation; no marshal seam added.',
    ('cli.go', 369): 'os.Executable failure: existing application dependency failure coverage, no individual actual EBADF control here.',
    ('cli.go', 374): 'Absolute executable resolution failure needs a relative executable plus genuine Getwd failure; no global cwd mutation or production seam added.',
    ('cli.go', 379): 'Absolute artifact resolution failure needs genuine Getwd failure after executable resolution; no global cwd mutation added.',
    ('cli.go', 396): 'encoding/json failure for concrete Doctor Report: normal validated detached report contains only supported values; no marshal seam added.',
    ('cli.go', 477): 'Missing plan output after successful installation: existing fake-dependency ordinary characterization covers failure delivery; individual actual EBADF control not run.',
    ('cli.go', 486): 'Classified plan operation error: existing fake-dependency ordinary characterization covers failure delivery; real portable preparation needs supported native toolchain under fn149/fn128.',
    ('cli.go', 494): 'encoding/json failure for concrete CampaignPlanResult: validated public plan supplies supported detached values; no marshal seam added.',
    ('cli.go', 599): 'Flag parse error plus failed reporter: existing ordinary characterization; no individual actual EBADF flag-error control run (flag Usage writes remain unchanged).',
    ('cli.go', 755): 'Ambient Getwd failure: no global cwd mutation or production seam added; existing private working-directory seam and ordinary tests unchanged.',
    ('qualify.go', 83): 'Flag parse error plus failed reporter: existing ordinary characterization; no individual actual EBADF flag-error control run (flag Usage writes remain unchanged).',
    ('resume.go', 32): 'Flag parse error plus failed reporter: existing ordinary characterization; no individual actual EBADF flag-error control run (flag Usage writes remain unchanged).',
    ('resume.go', 70): 'Classified Resume operation error plus failed reporter: existing fake-dependency ordinary characterization; actual interrupted native campaign belongs to fn149/fn128.',
}
proof = []
for name, expected in [('application.go', 1), ('campaign_shards.go', 8), ('cli.go', 29), ('qualify.go', 3), ('resume.go', 5)]:
    path = PREFIX + name
    before, after = original(path).decode(), (ROOT / path).read_text()
    pattern = r'(?m)^(\t*)if _, printErr := (fmt\.Fprint(?:ln|f)?\(stderr,.*\)); printErr != nil \{\n\1\t(return [^\n]+)\n\1}\n'
    old_lines, new_lines = before.splitlines(keepends=True), after.splitlines(keepends=True)
    cursor, checks = 0, []
    offsets = [0]
    for line in new_lines:
        offsets.append(offsets[-1] + len(line))
    for old_line in old_lines:
        if re.fullmatch(r'\t*fmt\.Fprint(?:ln|f)?\(stderr,.*\)\n', old_line):
            check = re.match(pattern, after[offsets[cursor]:])
            assert check is not None and check[2] == old_line.strip(), (name, old_line)
            checks.append((cursor + 1, check))
            cursor += 3
        else:
            assert old_line == new_lines[cursor], (name, cursor, old_line, new_lines[cursor])
            cursor += 1
    assert cursor == len(new_lines), name
    assert len(checks) == expected, (name, len(checks))
    sites = []
    original_sites = list(re.finditer(r'(?m)^\t*fmt\.Fprint(?:ln|f)?\(stderr,.*\)\n', before))
    assert len(original_sites) == expected
    for old, (candidate_line, new) in zip(original_sites, checks):
        line = before[:old.start()].count('\n') + 1
        expression = old[0].strip()
        assert expression == new[2]
        tail = before[old.end():].splitlines()[:3]
        primary = next(text.strip() for text in tail if text.strip().startswith('return '))
        assert new[3] == primary
        test = covered[name].get(line)
        sites.append({'original_line': line, 'candidate_line': candidate_line,
                      'expression': expression, 'failed_write_return': primary,
                      'runtime_control': 'TestTerminalDiagnostics' + test if test else None,
                      'runtime_gap': None if test else gaps[(name, line)]})
    proof.append({'path': path, 'base_sha256': digest(before.encode()), 'candidate_sha256': digest(after.encode()),
                  'admitted_checks': len(checks), 'complete_original_bytes_reconstructed': True,
                  'all_nonadmitted_bytes_including_comments_and_stdout_preserved': True, 'sites': sites})
tests = subprocess.check_output(['git', 'ls-files', '-z', PREFIX + '*test.go'], cwd=ROOT).split(b'\0')
protected_tests = {path.decode(): digest((ROOT / path.decode()).read_bytes()) for path in tests if path}
for path, current in protected_tests.items():
    assert digest(original(path)) == current, path
turbo = {path: digest((ROOT / path).read_bytes()) for path in ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md']}
assert turbo == {'.turbo/plans/gomad3-glossary-update.md': '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188', '.turbo/technical-debt.md': 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'}
old_output = (OUT.parent / 'task-52/make-gomad-original-base.stdout').read_text()
new_output = (OUT / 'make-gomad-original-base.stdout').read_text()
def findings(output):
    return re.findall(r'(?m)^([^\n]+\.go):([0-9]+):([0-9]+): ([^\n]+)\n([^\n]*)\n([^\n]*)', output)
old_findings, new_findings = findings(old_output), findings(new_output)
old_key = collections.Counter((path, message, statement) for path, _, _, message, statement, _ in old_findings)
new_key = collections.Counter((path, message, statement) for path, _, _, message, statement, _ in new_findings)
removed, added = old_key - new_key, new_key - old_key
assert len(old_findings) == 145 and len(new_findings) == 99
assert sum(removed.values()) == 46 and not added
assert all(path.startswith(PREFIX) and 'not checked' in message and 'stderr' in statement for path, message, statement in removed)
record = {'base_commit': BASE, 'production_files': proof, 'original_tests_unchanged': protected_tests,
          'protected_turbo_sha256': turbo,
          'lint': {'before': len(old_findings), 'after': len(new_findings), 'removed': sum(removed.values()), 'added': sum(added.values()),
                   'comparison': 'Exact original diagnostic path/message/statement multiset. Shifted original source lines mapped through admitted insertions; residual source statement/message bytes unchanged.'},
          'runtime_scope': 'Public failures use Run; private-mode dispatch uses existing application entry. Replay and completed-operation controls use existing private dependencies; completed marker files prove command reporting does not roll back those completed callback operations, not native artifact publication or target execution.'}
destination = OUT / 'source-proof.json'
if destination.exists():
    raise SystemExit('Refusing to overwrite retained evidence')
destination.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps({'checked': sum(row['admitted_checks'] for row in proof), 'runtime_gaps': len(gaps), 'lint': record['lint']}))
