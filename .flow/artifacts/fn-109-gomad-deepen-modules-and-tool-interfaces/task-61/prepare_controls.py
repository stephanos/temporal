import hashlib
import json
import pathlib
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-61')
OUT = pathlib.Path(__file__).resolve().parent
SCRATCH = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/tmp-61/controls')
SUFFIX = sys.argv[1] if len(sys.argv) > 1 else ''
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
SCRATCH.mkdir(exist_ok=True)
fixture_path = ROOT / 'tools/gomad3/toolchain/build_test.go'
lock_path = ROOT / 'tools/gomad3/toolchain/lock.go'
fixture, lock = fixture_path.read_text(), lock_path.read_text()

def replace_once(source, old, new):
    if source.count(old) != 1:
        raise SystemExit('Mutation did not have exactly one source match: ' + old)
    return source.replace(old, new, 1)

cleanup_source = '\t\t}\n\t}()\n\tpending++\n\tgo func() {'
instrumented = replace_once(fixture, cleanup_source, '\t\t}\n\t\tt.Log("controlled cleanup drained every launched builder")\n\t}()\n\tpending++\n\tgo func() {')
muted = replace_once(instrumented, '\tctx.once.Do(func() { close(ctx.entered) })', '\tctx.once.Do(func() {})')
sequential = replace_once(instrumented, '\tresults := make(chan outcome, 2)', '\tresults := make(chan outcome, 2)\n\tfirstCompleted := make(chan struct{})')
sequential = replace_once(sequential, '\t\tresult, err := buildWith(ctx, config, dependencies)\n\t\tresults <- outcome{result: result, err: err}', '\t\tdefer close(firstCompleted)\n\t\tresult, err := buildWith(ctx, config, dependencies)\n\t\tresults <- outcome{result: result, err: err}')
sequential = replace_once(sequential, '\t\tresult, err := buildWith(waiter, config, dependencies)', '\t\tselect {\n\t\tcase <-firstCompleted:\n\t\tcase <-ctx.Done():\n\t\t}\n\t\tresult, err := buildWith(waiter, config, dependencies)')
contended_select = '''	select {
	case <-waiter.entered:
	case result := <-results:
		pending--
		t.Fatalf("builder completed before lock contention: %+v", result)
	case <-ctx.Done():
		t.Fatal("second builder did not observe lock contention before timeout")
	}
'''
sequential = replace_once(sequential, contended_select, '')
mutations = {
    'muted-observer': {str(fixture_path): muted},
    'waited-false': {str(fixture_path): instrumented, str(lock_path): replace_once(lock, '\t\twaited = true', '\t\twaited = false')},
    'bypassed-wait-sequential': {str(fixture_path): sequential},
}
manifest = {'base_commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
            'candidate_sha256': hashlib.sha256(fixture.encode()).hexdigest(),
            'lock_sha256': hashlib.sha256(lock.encode()).hexdigest(), 'controls': {}}
for name, changes in mutations.items():
    replacements, hashes = {}, {}
    for source_path, contents in changes.items():
        target = SCRATCH / (name + SUFFIX + '-' + pathlib.Path(source_path).name)
        with target.open('x') as output:
            output.write(contents)
        replacements[source_path] = str(target)
        hashes[str(target)] = hashlib.sha256(target.read_bytes()).hexdigest()
    overlay = OUT / (name + SUFFIX + '-overlay.json')
    with overlay.open('x') as output:
        json.dump({'Replace': replacements}, output, indent=2)
        output.write('\n')
    hashes[str(overlay)] = hashlib.sha256(overlay.read_bytes()).hexdigest()
    manifest['controls'][name] = hashes
overlay = OUT / ('context-controls' + SUFFIX + '-overlay.json')
with overlay.open('x') as output:
    json.dump({'Replace': {str(ROOT / 'tools/gomad3/toolchain/task61_context_controls_test.go'): str(OUT / 'context_controls_test.go')}}, output, indent=2)
    output.write('\n')
manifest['controls']['context-controls'] = {str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in [overlay, OUT / 'context_controls_test.go']}
with (OUT / ('control-inputs' + SUFFIX + '.json')).open('x') as output:
    json.dump(manifest, output, indent=2)
    output.write('\n')
print(json.dumps(manifest, indent=2))
