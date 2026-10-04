#!/usr/bin/env python3
"""Capture bounded artifact command evidence without altering source."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
CONFIG = ROOT / '.github/.golangci.yml'
ENV = dict(os.environ)
ENV.update(GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='')
ENV['PATH'] = str(Path(GO).parent) + ':' + ENV['PATH']
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(key, None)

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    return {str(path.relative_to(ROOT)): digest(path)
            for path in sorted((ROOT / 'tools/gomad3/artifact').glob('*.go'))}

name, kind = sys.argv[1:3]
cwd = ROOT / 'tools/gomad3'
if kind == 'report':
    base = 'e98c7a3e2ca845b92edcdef185f5c9ae67be4a3a'
    names = ['baseline-package', 'baseline-focused', 'baseline-lint', 'baseline-errortype',
             'characterization-baseline', 'final-package', 'final-focused', 'final-lint',
             'final-errortype', 'final-boundary', 'final-validation', 'final-static']
    receipts = {item: json.loads((OUT / (item + '.json')).read_text()) for item in names}
    def findings(item):
        pattern = r'^(.*\.go):(\d+):(\d+): (.*) \((\w+)\)$'
        return [dict(path=path, line=int(line), column=int(column), message=message, linter=linter)
                for path, line, column, message, linter in re.findall(pattern, (OUT / (item + '.log')).read_text(), re.MULTILINE)]
    before, after = findings('baseline-lint'), findings('final-lint')
    def key(finding):
        return finding['path'], finding['message'], finding['linter']
    mapped_lines = (333, 346, 352, 356, 402, 406, 410)
    resolved = [finding for finding in before if finding['path'] == 'tools/gomad3/artifact/store.go'
                and finding['line'] in mapped_lines and finding['linter'] == 'errcheck']
    residual_before = [finding for finding in before if finding not in resolved]
    assert sorted(map(key, residual_before)) == sorted(map(key, after))
    protected_before = json.loads((OUT / 'protected-before.json').read_text())
    protected_after = json.loads((OUT / 'protected-after.json').read_text())
    admission = json.loads((OUT / 'source-admission.json').read_text())
    current = snapshot()
    baseline = receipts['baseline-package']['source_before']
    def base_source(path):
        return subprocess.check_output(['git', 'show', base + ':' + path], cwd=ROOT)
    baseline_matches_base = all(value == hashlib.sha256(base_source(path)).hexdigest() for path, value in baseline.items())
    def omit_private(data):
        for start, end in ((b'func copyPayload(', b'func syncPayloadDirectories('),
                           (b'func writePayload(', b'func copyWithContext(')):
            prefix, rest = data.split(start, 1)
            _, suffix = rest.split(end, 1)
            data = prefix + end + suffix
        return data
    store = 'tools/gomad3/artifact/store.go'
    tests = 'tools/gomad3/artifact/store_test.go'
    store_outside_unchanged = omit_private(base_source(store)) == omit_private((ROOT / store).read_bytes())
    marker = b'func TestPublishFailsBeforePublicationWhenByteCapacityIsExceeded('
    original_tests_unchanged = base_source(tests).split(marker, 1)[1] == (ROOT / tests).read_bytes().split(marker, 1)[1]
    final_names = [item for item in names if item.startswith('final-')]
    changed = subprocess.check_output(['git', 'diff', base, '--name-only', '--', 'tools/gomad3',
        'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    checks = dict(baseline_matches_base=baseline_matches_base,
        all_gate_sources_stable=all(item['stable'] for item in receipts.values()),
        final_gate_sources_match_current=all(receipts[item]['source_after'] == current for item in final_names),
        protected_inputs_stable=protected_before['aggregate_sha256'] == protected_after['aggregate_sha256'] == admission['protected_aggregate_sha256'],
        protected_files=protected_after['files'], protected_sha256=protected_after['aggregate_sha256'],
        store_outside_private_helpers_unchanged=store_outside_unchanged, original_store_test_bodies_unchanged=original_tests_unchanged,
        changed_product_paths=changed)
    assert all(checks[key] for key in ('baseline_matches_base', 'all_gate_sources_stable', 'final_gate_sources_match_current',
        'protected_inputs_stable', 'store_outside_private_helpers_unchanged', 'original_store_test_bodies_unchanged'))
    assert changed == [store, tests] and len(before) == 28 and len(after) == 21 and len(resolved) == 7
    assert all(receipts[item]['exit'] == (1 if item.endswith('-lint') else 0) for item in names)
    assert not (OUT / 'final-static.log').read_text()
    result = dict(status='SOURCE_PROGRESS_ONLY', task='fn-109-gomad-deepen-modules-and-tool-interfaces.29',
        base_commit=base, branch='gomad', commits=[], commit_range='',
        requested_model='gpt-6.1-sol', requested_effort='high', tier='session (jev-unavailable(no_key))',
        actual_model=None, actual_model_metadata_available=False, platform=protected_after['platform'],
        source_checks=checks, source_final={path: current[path] for path in changed},
        gates={item: dict(receipt=item + '.json', log=receipt['log'], exit=receipt['exit'],
            elapsed_seconds=receipt['elapsed_seconds'], stable=receipt['stable']) for item, receipt in receipts.items()},
        pinned_tools=receipts['baseline-lint']['tools'], lint_config_sha256=receipts['baseline-lint']['config_sha256'],
        lint=dict(total_before=len(before), total_after=len(after), mapped_before=7, mapped_after=0,
            before=before, after=after, resolved=resolved, introduced=[],
            cleanup_mapping=[dict(baseline_line=line, checked_cleanup_line=336 if line == 333 else 355 if line < 400 else 418)
                             for line in mapped_lines],
            residual='All 21 residual path/message/linter identities match baseline, including directory.Close shifted 470 to 490.',
            broader_scope='Historical 419 is not a fresh whole-scope count.'),
        test_first=dict(policy_red='baseline-lint.log reproduces all seven actual pinned unchecked-close findings before production edits',
            characterization='characterization-baseline.log passes five real-file controls before production edits; these characterize primary/lifetime behavior, not first-Close faults'),
        first_close=dict(inspected_sources=protected_after['pinned_file_close_source'],
            genuine_first_close_fault_exercised=False,
            limitation='No deterministic legitimate first os.File.Close failure is reproduced. Conditional dual-error ordering is source-inspected only; success controls reject observing a second input Close.'),
        generator=dict(inputs_inspected='tools/gomad3/Makefile VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS and validate recipes',
            source_is_generator_input=False, disposition='make validate passes without regeneration; protected generated/runtime/protocol/pin/config/dependency inputs are unchanged'),
        boundaries=dict(historical_test='TestRecordAndArtifactHaveSeparateOwners is absent in current source; TestPackageArchitecture verifies distinct record/artifact owners',
            executed=['TestPackageArchitecture', 'TestPublicPackagesDoNotExportTypeAliases', 'TestArchitecturePublicSignatureFixtures',
                      'TestRunnerRequestsCompileInExternalModule', 'TestRunnerExternalConsumerCompiles']),
        review=dict(owner='root', status='pending fresh independent source review', formal='conductor-deferred; no formal SHIP'),
        remaining=['Original R13/R18/R19, task12/predecessors and task21 acceptance',
            'Matched first-baseline fixed identities and complete/full/formal qualification',
            'Patched-runtime native darwin/arm64 and linux/amd64 and affected integration/qualification gates',
            'Genuine first-Close fault coverage', '21 residual artifact diagnostics and broader historical lint qualification'],
        handover=str(OUT / 'handover.md'), historical_references=['../../../tmp/artifact-cleanup-source-mapping.md',
            '../task-21/preservation-disclosure-2026-10-04.md', '../task-28/handover.md'],
        owned_delegates=0, live_command_handles=0, pending_commands=0, worker_git_writes=0, worker_flow_lifecycle_writes=0)
    (OUT / (name + '.json')).write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(dict(status=result['status'], source_checks=checks, lint_before=len(before), lint_after=len(after))))
    sys.exit(0)
if kind == 'protect':
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim',
        'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {path: digest(ROOT / path) for path in paths
              if path not in ('tools/gomad3/artifact/store.go', 'tools/gomad3/artifact/store_test.go')}
    receipt = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest(),
        captured=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        selection='tracked Gomad modules, root go.mod/go.sum and lint config excluding only artifact/store.go and artifact/store_test.go',
        pinned_file_close_source={str(Path(GO).parents[1] / 'src/os' / path): digest(Path(GO).parents[1] / 'src/os' / path)
                                  for path in ('file_posix.go', 'file_unix.go')},
        platform=subprocess.check_output([GO, 'env', 'GOOS', 'GOARCH'], env=ENV, text=True).splitlines(),
        patched_go_exists=(cwd / '.toolchain/bin/go').exists())
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps(receipt))
    sys.exit(0)
if kind == 'test':
    command = [GO, 'test', '-count=1', '-tags', 'test_dep', *sys.argv[3:]]
elif kind == 'lint':
    command = [LINT, 'run', '--config', str(CONFIG), '--build-tags', 'test_dep', '--fix=false', './artifact']
elif kind == 'errortype':
    command = [ERROR, '-tags', 'test_dep', './artifact']
elif kind == 'validate':
    command = ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1']
elif kind == 'static':
    command = ['bash', '-c', 'git diff --check -- artifact/store.go artifact/store_test.go && ' + str(Path(GO).parent / 'gofmt') + ' -l artifact/store.go artifact/store_test.go']
else:
    raise ValueError(kind)
before = snapshot()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with (OUT / (name + '.log')).open('w') as log:
    result = subprocess.run(command, cwd=cwd, env=ENV, stdout=log, stderr=subprocess.STDOUT)
after = snapshot()
receipt = dict(command=command, cwd=str(cwd), environment={key: ENV.get(key) for key in
    ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')},
    started=started, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    elapsed_seconds=time.monotonic() - clock, exit=result.returncode, source_before=before, source_after=after,
    stable=before == after, tools={path: digest(path) for path in (GO, LINT, ERROR)},
    config_sha256=digest(CONFIG), log=name + '.log', log_sha256=digest(OUT / (name + '.log')))
(OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({key: receipt[key] for key in ('command', 'exit', 'elapsed_seconds', 'stable', 'log')}))
