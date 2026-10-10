import hashlib
import json
import pathlib
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-61')
OUT = pathlib.Path(__file__).resolve().parent
SOURCE = 'tools/gomad3/toolchain/build_test.go'
BASE = (ROOT / '.flow/tmp/base_commit').read_text().strip()
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
old = subprocess.check_output(['git', 'show', BASE + ':' + SOURCE], cwd=ROOT, text=True)
new = (ROOT / SOURCE).read_text()
start = 'func TestBuildSerializesConcurrentSameKey(t *testing.T) {'
end = '\nfunc TestBuildInjectedFailuresLeaveNoTemporaryState'
helper = '''type buildLockWaitContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (ctx *buildLockWaitContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.entered) })
	return ctx.Context.Done()
}

'''
assert new.count(helper) == 1
new_without_helper = new.replace(helper, '', 1)
assert old[:old.index(start)] == new_without_helper[:new_without_helper.index(start)]
assert old[old.index(end):] == new_without_helper[new_without_helper.index(end):]
old_body = old[old.index(start):old.index(end)]
new_body = new_without_helper[new_without_helper.index(start):new_without_helper.index(end)]
assertion_start = '\tif first.err != nil || second.err != nil {'
assert old_body[old_body.index(assertion_start):] == new_body[new_body.index(assertion_start):]
for unchanged in [
    '\troot := writeBuildFixture(t)\n\tvar builds atomic.Int64\n\tstarted := make(chan struct{})\n\trelease := make(chan struct{})\n',
    '\tdependencies := fakeDependencies(t, &builds)\n\tdependencies.run = fakeRunner(t, "darwin", "arm64", &builds, func() {\n\t\tclose(started)\n',
    '\tconfig := testConfig(root)\n\ttype outcome struct {\n\t\tresult BuildResult\n\t\terr    error\n\t}\n\tresults := make(chan outcome, 2)\n',
]:
    assert old_body.count(unchanged) == new_body.count(unchanged) == 1
changed = subprocess.check_output(['git', 'diff', '--name-only', BASE, '--', 'tools/gomad3'], cwd=ROOT, text=True).splitlines()
assert changed == [SOURCE], changed
for path in ['tools/gomad3/toolchain/build.go', 'tools/gomad3/toolchain/lock.go', 'tools/gomad3/internal/hostfs/lock_unix.go']:
    assert subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT) == (ROOT / path).read_bytes()
baseline_runner = (OUT / 'run_gate.py').read_bytes()
assert hashlib.sha256(baseline_runner).hexdigest() == 'b9b9f127b6c94cfeec460d93a41601c765eb4aae8c75316a9afb98fc610886c6'
for name in ['baseline-builds', 'baseline-lint', 'baseline-vet', 'baseline-format', 'baseline-original-base']:
    receipt = json.loads((OUT / (name + '.json')).read_text())
    assert receipt['terminal'] and receipt['source_unchanged'] and not receipt['timed_out']
    assert receipt['gate_runner_sha256'] == hashlib.sha256(baseline_runner).hexdigest()
    for suffix in ['stdout', 'stderr']:
        assert receipt[suffix + '_sha256'] == hashlib.sha256((OUT / (name + '.' + suffix)).read_bytes()).hexdigest()
result = {
    'base_commit': BASE,
    'baseline_source_sha256': hashlib.sha256(old.encode()).hexdigest(),
    'candidate_source_sha256': hashlib.sha256(new.encode()).hexdigest(),
    'only_product_path_changed': SOURCE,
    'all_other_test_bytes_equal': True,
    'all_original_outcome_assertions_equal': True,
    'fixture_inputs_equal': True,
    'production_lock_build_and_hostfs_equal': True,
    'baseline_runner_and_raw_hashes_valid': True,
}
print(json.dumps(result, indent=2))
