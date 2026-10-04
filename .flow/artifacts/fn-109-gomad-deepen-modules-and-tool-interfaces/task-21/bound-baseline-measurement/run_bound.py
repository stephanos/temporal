#!/usr/bin/env python3
"""Run the isolated baseline with bound settings and exact source path inventories."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import time

HERE = Path(__file__).resolve().parent
RESULT = HERE/'runs'
RESULT.mkdir(exist_ok=False)
SCRATCH = Path('/tmp/fn109-r19-bound-baseline.7hehYl3f/gomad3')
ORIGINAL = Path('/tmp/fn109-baseline-reconstruction.lDSSw8Gx')
ORIGINAL_MODULE = ORIGINAL/'tools/gomad3'
HISTORICAL = HERE.parent/'baseline-measurement'
HISTORICAL_SCRATCH = Path('/tmp/fn109-r19-baseline-measurement.Bq9eSbCF/gomad3')
RECONSTRUCTION = HERE.parent/'baseline-reconstruction'
REPO = HERE.parents[4]
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
BINDINGS = dict(GOENV='off', GOFLAGS='', GOEXPERIMENT='', GODEBUG='', GOGC='100', GOMEMLIMIT='off', GOMAXPROCS='2', CGO_ENABLED='1', GOOS='linux', GOARCH='arm64', GOARM64='v8.0', GOWORK='off', GOTOOLCHAIN='local', TZ='UTC')
SUPPLEMENTARY = {
    'r19_measurement_test.go': 'runner/r19_measurement_test.go',
    'r19_logical_policy_test.go': 'runner/r19_logical_policy_test.go',
    'r19_artifact_payload_test.go': 'artifact/r19_artifact_payload_test.go',
    'r19_transport_guard.go': 'runner/internal/execution/r19_transport_guard.go',
    'descriptor_dup_linux.go': 'runner/internal/execution/descriptor_dup_linux.go',
}
evidence = dict(commits=[], requested_model='gpt-6.1-sol/high', actual_model_metadata='unknown', tier='session (jev-unavailable(no_key))', current_comparison_run=False, qualification_admitted=False, scratch=str(SCRATCH), bindings=BINDINGS, predeclared_supplementary_paths=SUPPLEMENTARY, commands=[], inventory_checks=[], hashes={})

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def inventory(root):
    rows = []
    for path in sorted(root.rglob('*')):
        if path.is_symlink():
            raise ValueError(f'symlink in source/evidence inventory: {path}')
        if path.is_file():
            rows.append(dict(path=str(path.relative_to(root)), sha256=digest(path), mode=stat.S_IMODE(path.stat().st_mode)))
    return rows

def write_json(path, value):
    path.write_text(json.dumps(value, indent=2)+'\n')

def persist():
    write_json(RESULT/'execution-evidence.json', evidence)

def compare(label, before, after):
    left, right = {row['path']:row for row in before}, {row['path']:row for row in after}
    added, removed = sorted(right.keys()-left.keys()), sorted(left.keys()-right.keys())
    changed = sorted(key for key in left.keys()&right.keys() if left[key] != right[key])
    row = dict(label=label, file_count_before=len(before), file_count_after=len(after), added=added, removed=removed, changed=changed)
    evidence['inventory_checks'].append(row)
    persist()
    assert not added and not removed and not changed, row
    return row

def run(argv, cwd, output, extra=None):
    env = dict(os.environ, **BINDINGS)
    env.update(extra or {})
    assert all(env[name] == value for name, value in BINDINGS.items())
    child_subset = {name:env[name] for name in [*BINDINGS, *(extra or {})]}
    row = dict(argv=argv, cwd=str(cwd), child_environment_subset=child_subset, started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), output=output)
    evidence['commands'].append(row)
    persist()  # The exact child settings and argv are retained before launch.
    started = time.monotonic()
    child = subprocess.run(argv, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    row.update(exit_code=child.returncode, elapsed_seconds=time.monotonic()-started)
    (RESULT/output).write_bytes(child.stdout)
    persist()
    print(json.dumps(row), flush=True)
    assert child.returncode == 0, f'{output} exited {child.returncode}; retained output'
    return child.stdout

# Preserve complete old path sets and bytes, including every historical profile.
protected = {}
for label, path in [('original-baseline', ORIGINAL_MODULE), ('historical-artifacts', HISTORICAL), ('historical-scratch', HISTORICAL_SCRATCH)]:
    protected[label] = inventory(path)
    write_json(RESULT/(label+'-before.json'), protected[label])
assert len(protected['original-baseline']) == 670
write_json(RESULT/'bound-environment-before-launch.json', dict(bindings=BINDINGS, recorded_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), predeclared_supplementary_paths=SUPPLEMENTARY))
persist()
run(['sha256sum', '--check', '--quiet', str(RECONSTRUCTION/'source.sha256')], ORIGINAL, 'original-source-precheck.log')
drift = []
for line in (RECONSTRUCTION/'input.sha256').read_text().splitlines():
    old, relative = line.split('  ', 1)
    current = digest(REPO/relative)
    if old != current:
        drift.append(dict(path=relative, historical_sha256=old, current_sha256=current))
assert len(drift) == 1 and drift[0]['path'] == '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md', drift
evidence['historical_input_metadata_drift'] = drift
evidence['historical_reconstruction_source_inputs_unchanged'] = True
run([str(Path(GO).with_name('gofmt')), '-w', *(str(HERE/name) for name in SUPPLEMENTARY)], SCRATCH, 'gofmt-retained-fixture.log')
for retained, destination in SUPPLEMENTARY.items():
    shutil.copyfile(HERE/retained, SCRATCH/destination)
initial = inventory(SCRATCH)
assert len(initial) == 675
write_json(RESULT/'initial-source-inventory.json', initial)
(RESULT/'initial-source.sha256').write_text(''.join(row['sha256']+'  '+row['path']+'\n' for row in initial))
evidence['hashes']['initial-source-inventory.json'] = digest(RESULT/'initial-source-inventory.json')
evidence['hashes']['pinned-go-binary'] = digest(Path(GO))
run([GO, 'env', '-json', 'GOENV', 'GOFLAGS', 'GOEXPERIMENT', 'CGO_ENABLED', 'GOOS', 'GOARCH', 'GOARM64', 'GOWORK', 'GOTOOLCHAIN', 'GOVERSION'], SCRATCH, 'effective-go-env-before-build.json')
effective = json.loads((RESULT/'effective-go-env-before-build.json').read_text())
effective_expected = dict(BINDINGS, GOENV='')  # go env reports an empty path when GOENV=off.
assert all(effective[name] == effective_expected[name] for name in effective if name in effective_expected)
assert effective['GOVERSION'] == 'go1.27.1'
run([GO, 'version'], SCRATCH, 'pinned-go-version.log')
run(['uname', '-a'], SCRATCH, 'host.log')
for mode in ['discard', 'novel']:
    for jobs in [10, 100]:
        case = f'{mode}-{jobs}'
        directory = RESULT/case
        directory.mkdir()
        before = inventory(SCRATCH)
        write_json(RESULT/(case+'-source-before.json'), before)
        compare(case+' initial-to-before', initial, before)
        run([GO, 'test', '-tags', 'test_dep', '-count=1', '-timeout=6m', '-run', '^TestR19BoundedSeedCampaignMeasurement$', '-v', '-memprofilerate=1', '-memprofile='+str(directory/'process-end.pprof'), '-o', str(directory/'runner.test'), './runner'], SCRATCH, case+'.log', dict(R19_JOBS=str(jobs), R19_MODE=mode, R19_RESULT_DIR=str(directory)))
        after = inventory(SCRATCH)
        write_json(RESULT/(case+'-source-after.json'), after)
        compare(case+' before-to-after', before, after)
        run([GO, 'version', '-m', str(directory/'runner.test')], SCRATCH, case+'-binary-build-metadata.log')
        for stage in ['early-pair', 'late-pair', 'completed', 'process-end']:
            profile = directory/(stage+'.pprof')
            for sample in ['alloc_space', 'alloc_objects', 'inuse_space', 'inuse_objects']:
                run([GO, 'tool', 'pprof', '-top', '-nodecount=0', '-nodefraction=0', '-edgefraction=0', '-unit=bytes', '-sample_index='+sample, str(directory/'runner.test'), str(profile)], SCRATCH, case+'-'+stage+'-'+sample+'.log')
            run([GO, 'tool', 'pprof', '-raw', str(directory/'runner.test'), str(profile)], SCRATCH, case+'-'+stage+'-raw.log')
        for path in directory.rglob('*'):
            if path.is_file():
                evidence['hashes'][str(path.relative_to(RESULT))] = digest(path)
        persist()
run([GO, 'test', '-tags', 'test_dep', '-count=1', '-run', '^TestR19ArtifactInputAliasesProducedPayloads$', '-v', './runner'], SCRATCH, 'runner-constructor-alias.log')
run([GO, 'test', '-tags', 'test_dep', '-count=1', '-run', '^TestR19ArtifactDataPayloadAliasesInput$', '-v', './artifact'], SCRATCH, 'artifact-payload-alias.log')
run([GO, 'test', '-tags', 'test_dep', '-count=1', '-run', '^TestR19LogicalPolicyBothSeedSources$', '-v', './runner'], SCRATCH, 'logical-policy-both-roles.log', dict(R19_RESULT_DIR=str(RESULT)))
run([GO, 'test', '-tags', 'test_dep', '-count=1', '-run', '^TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs$', '-v', './runner'], SCRATCH, 'existing-capacity-controls.log')
run(['python3', str(HERE/'analyze_profiles.py')], SCRATCH, 'profile-analysis.log')
final = inventory(SCRATCH)
write_json(RESULT/'final-source-inventory.json', final)
compare('initial-to-final including controls', initial, final)
for label, path in [('original-baseline', ORIGINAL_MODULE), ('historical-artifacts', HISTORICAL), ('historical-scratch', HISTORICAL_SCRATCH)]:
    after = inventory(path)
    write_json(RESULT/(label+'-after.json'), after)
    compare(label+' preserved', protected[label], after)
for name in [*SUPPLEMENTARY, 'developmental-platform.patch', 'baseline-compilation-calls.patch', 'run_bound.py', 'analyze_profiles.py']:
    evidence['hashes']['retained-input/'+name] = digest(HERE/name)
persist()
(RESULT/'retained-output.sha256').write_text(''.join(digest(path)+'  '+str(path.relative_to(RESULT))+'\n' for path in sorted(RESULT.rglob('*')) if path.is_file() and path.name != 'retained-output.sha256'))
