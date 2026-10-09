import collections
import difflib
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/tmp/gomad-fn109-parallel.pmgezCtg/task-57')
OUT = pathlib.Path(__file__).resolve().parent
BASE = 'f830467fcb712a83ba504c7dd43e2beb7f2cf223'
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong worktree')


def sha(data):
    return hashlib.sha256(data).hexdigest()


def base_bytes(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)


changes = collections.defaultdict(list)


def deferred(path, receiver, kind, reporter='t', optional_closed=False, argument=None, operation='Close'):
    argument = argument or receiver
    condition = 'err != nil'
    if optional_closed:
        condition += ' && !errors.Is(err, os.ErrClosed)'
    old = '\tdefer ' + (receiver + '.' + operation + '()' if kind != 'int' else 'syscall.Close(' + argument + ')') + '\n'
    call = receiver + '.' + operation + '()' if kind != 'int' else 'syscall.Close(' + receiver + ')'
    new = '\tdefer func(' + receiver + ' ' + kind + ') {\n\t\tif err := ' + call + '; ' + condition + ' {\n\t\t\t' + reporter + '.Error(err)\n\t\t}\n\t}(' + argument + ')\n'
    changes[path].append((old, new))


deferred('tools/gomad3/choice/diagnostic_test.go', 'session', '*DiagnosticSession')
deferred('tools/gomad3/internal/hostfs/open_test.go', 'root', '*os.Root')
deferred('tools/gomad3/runner/choice_exploration_divergence_unix_test.go', 'input', 'io.WriteCloser', optional_closed=True)
deferred('tools/gomad3/runner/diagnostics_test.go', 'opened', '*artifact.Opened')
deferred('tools/gomad3/runner/inspect_test.go', 'journal', '*campaign.CampaignJournal')
deferred('tools/gomad3/runner/inspect_test.go', 'journal', '*campaign.CampaignJournal')
deferred('tools/gomad3/runner/retention_characterization_test.go', 'opened', '*artifact.Opened', reporter='replayer.t')
deferred('tools/gomad3/runner/runner_test.go', 'opened', '*artifact.Opened')
deferred('tools/gomad3/runner/internal/execution/descriptor_dup_linux_test.go', 'descriptor', 'int', argument='descriptors[0]')
deferred('tools/gomad3/runner/internal/execution/descriptor_dup_linux_test.go', 'descriptor', 'int', argument='descriptors[1]')
deferred('tools/gomad3/runner/internal/execution/process_test.go', 'stdoutHead', '*os.File')
deferred('tools/gomad3/runner/internal/execution/process_test.go', 'stderrHead', '*os.File')
deferred('tools/gomad3/runner/internal/execution/process_unix_test.go', 'resources', '*launchResources', operation='close')
deferred('tools/gomad3/runner/internal/minimizer/workspace_unix_test.go', 'stdin', 'io.WriteCloser', optional_closed=True)
changes['tools/gomad3/runner/runner_test.go'].extend([
    ('func (replayer *matchingReplayer) Replay(_ context.Context, config ReplaySpec) (ReplayResult, error) {',
     'func (replayer *matchingReplayer) Replay(_ context.Context, config ReplaySpec) (_ ReplayResult, retErr error) {'),
    ('\tdefer opened.Close()\n', '\tdefer func(opened *artifact.Opened) {\n\t\tif err := opened.Close(); err != nil {\n\t\t\tif retErr == nil {\n\t\t\t\tretErr = err\n\t\t\t} else {\n\t\t\t\tretErr = errors.Join(retErr, err)\n\t\t\t}\n\t\t}\n\t}(opened)\n'),
    ('\t\tfile.Close()\n', '\t\tif closeErr := file.Close(); closeErr != nil {\n\t\t\treturn execution.Result{}, errors.Join(err, closeErr)\n\t\t}\n'),
])
assert len(changes) == 11
assert sum(len(value) for value in changes.values()) == 17
proof = {'base_commit': BASE, 'admitted_cleanup_calls': 16, 'admitted_old_files': 11, 'files': {}, 'preserved': {}}
line_maps = {}
for path, replacements in changes.items():
    candidate = (ROOT / path).read_bytes()
    restored = candidate.decode()
    for old, new in replacements:
        assert new in restored, (path, new)
        restored = restored.replace(new, old, 1)
    original = base_bytes(path)
    assert restored.encode() == original, path
    proof['files'][path] = {'base_sha256': sha(original), 'candidate_sha256': sha(candidate), 'exact_base_reconstructed': True}
    mapping = {}
    for tag, a, b, c, d in difflib.SequenceMatcher(a=original.decode().splitlines(), b=candidate.decode().splitlines(), autojunk=False).get_opcodes():
        if tag == 'equal':
            for index in range(d - c):
                mapping[c + index + 1] = a + index + 1
    line_maps[path] = mapping
paths = subprocess.check_output(['git', 'ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration', '.github/workflows/gomad3.yml', '.github/.golangci.yml', 'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode'], cwd=ROOT).split(b'\0')
new_path = 'tools/gomad3/runner/cleanup_test.go'
for raw in paths:
    if not raw:
        continue
    path = raw.decode()
    if path not in changes and path != new_path:
        data = (ROOT / path).read_bytes()
        assert data == base_bytes(path), path
        proof['preserved'][path] = sha(data)
proof['preserved_count'] = len(proof['preserved'])
proof['preserved_manifest_sha256'] = sha(json.dumps(proof['preserved'], sort_keys=True).encode())
del proof['preserved']
proof['new_test_sha256'] = sha((ROOT / new_path).read_bytes())
proof['initial_additive_test_sha256'] = sha((ROOT / new_path).read_bytes().replace(b'\twant.Artifact.TargetSharing = ""\n', b''))
tracked_diff = set(subprocess.check_output(['git', 'diff', '--name-only', BASE, '--', 'tools/gomad3'], cwd=ROOT, text=True).splitlines())
assert set(changes) <= tracked_diff <= set(changes) | {new_path}, tracked_diff


def diagnostics(path, remap=False):
    lines = path.read_text().splitlines()
    found = []
    for index, line in enumerate(lines):
        match = re.match(r'(tools/gomad3/[^:]+):(\d+):(\d+): (.*)', line)
        if match:
            source, row, column, message = match.groups()
            row = int(row)
            if remap and source in line_maps:
                row = line_maps[source].get(row, row)
            found.append((source, row, column, message, *lines[index + 1:index + 3]))
    return found


baseline_path = OUT.parent / 'task-56/corrected-make-gomad-original-base.stdout'
baseline_receipt = json.loads((OUT.parent / 'task-56/corrected-make-gomad-original-base.json').read_text())
baseline_inputs = {}
for raw in paths:
    if raw:
        path = raw.decode()
        if path == new_path:
            continue
        data = base_bytes(path)
        if path == 'MILESTONES.md':
            data = subprocess.check_output(['git', 'show', '87733ae65f:' + path], cwd=ROOT)
        baseline_inputs[path] = sha(data)
baseline_fingerprint = sha(json.dumps(baseline_inputs, sort_keys=True).encode())
assert baseline_fingerprint == baseline_receipt['source_after_sha256'], baseline_fingerprint
assert baseline_receipt['source_unchanged'] and baseline_receipt['terminal']
assert sha(baseline_path.read_bytes()) == baseline_receipt['stdout_sha256']
proof['baseline_inventory_binding'] = {
    'receipt_source_sha256': baseline_fingerprint,
    'source_identical_except_committed_milestone_admission': True,
    'milestone_admission_commit': BASE,
    'comparison_commit': '87733ae65f',
    'baseline_stdout_sha256': baseline_receipt['stdout_sha256'],
    'no_green_baseline_or_candidate_rebinding': True,
}
baseline = diagnostics(baseline_path)
assert len(baseline) == 80
if (OUT / 'make-gomad-original-base.stdout').exists():
    final = diagnostics(OUT / 'make-gomad-original-base.stdout', remap=True)
    removed = list((collections.Counter(baseline) - collections.Counter(final)).elements())
    added = list((collections.Counter(final) - collections.Counter(baseline)).elements())
    proof['lint'] = {'before': len(baseline), 'after': len(final), 'removed': len(removed), 'added': len(added),
                     'removed_diagnostics': removed, 'added_diagnostics': added, 'residual_base_line_column_message_statement_caret_multiset_preserved': not added,
                     'baseline_receipt': '../task-56/corrected-make-gomad-original-base.json', 'candidate_receipt': 'make-gomad-original-base.json'}
print(json.dumps(proof, indent=2))
