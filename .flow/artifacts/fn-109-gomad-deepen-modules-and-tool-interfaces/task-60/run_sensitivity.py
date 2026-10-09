import hashlib
import json
import pathlib
import shlex
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-60').resolve()
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
OUT = pathlib.Path(__file__).resolve().parent
SCRATCH = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task60-tmp')
RUNNER = OUT / 'run_gate.py'
SOURCE = ROOT / 'tools/gomad3/runner/internal/execution/process_test.go'
original = SOURCE.read_text()
mutations = [
    ('output', '\t\tif stdoutErr != nil {\n\t\t\tos.Exit(7)\n\t\t}', '\t\tif stdoutErr != nil {\n\t\t\tos.Exit(99)\n\t\t}'),
    ('choice-marker', '\t\tif _, err := fmt.Fprintln(os.Stdout, "post-choice-marker"); err != nil {\n\t\t\tos.Exit(0)\n\t\t}', '\t\tif _, err := fmt.Fprintln(os.Stdout, "post-choice-marker"); err != nil {\n\t\t\tos.Exit(99)\n\t\t}'),
    ('choice-tape-readonly', '\t\tif _, err := fmt.Fprintln(os.Stdout, "choice tape read-only"); err != nil {\n\t\t\tos.Exit(0)\n\t\t}', '\t\tif _, err := fmt.Fprintln(os.Stdout, "choice tape read-only"); err != nil {\n\t\t\tos.Exit(99)\n\t\t}'),
    ('choice-reorder', '\tif _, err := fmt.Fprintln(os.Stdout, <-results+<-results); err != nil {\n\t\treturn\n\t}', '\tif _, err := fmt.Fprintln(os.Stdout, <-results+<-results); err != nil {\n\t\tos.Exit(99)\n\t}'),
    ('choice-select', 'func runChoiceSelectTarget() {\n\truntime.LockOSThread()\n\tdefer runtime.UnlockOSThread()\n\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\treturn\n\t}', 'func runChoiceSelectTarget() {\n\truntime.LockOSThread()\n\tdefer runtime.UnlockOSThread()\n\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\tos.Exit(99)\n\t}'),
    ('choice-prefix-rng', 'func runChoicePrefixRNGTarget() {\n\truntime.LockOSThread()\n\tdefer runtime.UnlockOSThread()\n\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\treturn\n\t}', 'func runChoicePrefixRNGTarget() {\n\truntime.LockOSThread()\n\tdefer runtime.UnlockOSThread()\n\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\tos.Exit(99)\n\t}'),
]
bindings = {'candidate_sha256': hashlib.sha256(SOURCE.read_bytes()).hexdigest(), 'mutations': []}
for name, before, after in mutations:
    if original.count(before) != 1:
        raise SystemExit('Mutation preimage is not unique ' + name)
    source = SCRATCH / ('mutant-' + name + '.go')
    overlay = SCRATCH / ('mutant-' + name + '.json')
    if source.exists() or overlay.exists():
        raise SystemExit('Refusing to overwrite scratch mutation ' + name)
    data = original.replace(before, after).encode()
    source.write_bytes(data)
    overlay.write_text(json.dumps({'Replace': {str(SOURCE): str(source)}}) + '\n')
    row = {'name': name, 'source': str(source), 'source_sha256': hashlib.sha256(data).hexdigest(), 'overlay': str(overlay), 'overlay_sha256': hashlib.sha256(overlay.read_bytes()).hexdigest()}
    command = 'go test -tags test_dep -count=1 -json -overlay=' + shlex.quote(str(overlay)) + ' -run ' + shlex.quote('^TestTargetFixtureOutputPreservesProcessOutcome$/^' + name + '$/^read-only-stdout$') + ' ./runner/internal/execution'
    result = subprocess.run([sys.executable, str(RUNNER), 'sensitivity-' + name, 'tools/gomad3', command], cwd=ROOT)
    row['exit_code'] = result.returncode
    row['inputs_unchanged'] = hashlib.sha256(source.read_bytes()).hexdigest() == row['source_sha256'] and hashlib.sha256(overlay.read_bytes()).hexdigest() == row['overlay_sha256']
    bindings['mutations'].append(row)
    (OUT / 'sensitivity-inputs.json').write_text(json.dumps(bindings, indent=2) + '\n')
    if result.returncode != 1 or not row['inputs_unchanged']:
        raise SystemExit('Unexpected sensitivity result ' + name)
    raw = (OUT / ('sensitivity-' + name + '.stdout')).read_text()
    if 'status/stderr = 99/' not in raw:
        raise SystemExit('Mutation failed for a different reason ' + name)
    print(name + ' rejected status99', flush=True)
if hashlib.sha256(SOURCE.read_bytes()).hexdigest() != bindings['candidate_sha256']:
    raise SystemExit('Candidate source changed')
