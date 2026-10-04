#!/usr/bin/env python3
"""Capture bounded public artifact copy evidence without altering source."""
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
    base = '08096389f252e35ff2cd898ca5e381f9b878f46c'
    names = ['baseline-package', 'baseline-lint', 'baseline-errortype', 'characterization-baseline',
             'final-package', 'final-focused', 'final-retained-private', 'final-lint',
             'final-errortype', 'final-boundary', 'final-validation', 'final-static']
    receipts = {item: json.loads((OUT / (item + '.json')).read_text()) for item in names}
    def findings(item):
        pattern = r'^(.*\.go):(\d+):(\d+): (.*) \((\w+)\)$'
        return [dict(path=path, line=int(line), column=int(column), message=message, linter=linter)
                for path, line, column, message, linter in re.findall(pattern, (OUT / (item + '.log')).read_text(), re.MULTILINE)]
    before, after = findings('baseline-lint'), findings('final-lint')
    mapping = {'tools/gomad3/artifact/open.go': (270, 276, 283, 288, 293, 297),
               'tools/gomad3/artifact/opened_test.go': (41, 61, 145, 256, 323)}
    resolved = [finding for finding in before if finding['linter'] == 'errcheck'
                and finding['line'] in mapping.get(finding['path'], ())]
    def key(finding):
        return finding['path'], finding['message'], finding['linter']
    assert sorted(map(key, [finding for finding in before if finding not in resolved])) == sorted(map(key, after))
    protected_before = json.loads((OUT / 'protected-before.json').read_text())
    protected_after = json.loads((OUT / 'protected-after.json').read_text())
    admission = json.loads((OUT / 'source-admission.json').read_text())
    current = snapshot()
    def base_source(path):
        return subprocess.check_output(['git', 'show', base + ':' + path], cwd=ROOT).decode()
    baseline = receipts['baseline-package']['source_before']
    baseline_matches_base = all(value == hashlib.sha256(base_source(path).encode()).hexdigest() for path, value in baseline.items())
    source = 'tools/gomad3/artifact/open.go'
    tests = 'tools/gomad3/artifact/opened_test.go'
    original_source, final_source = base_source(source), (ROOT / source).read_text()
    start, end = 'func (opened *Opened) CopyPayload(', '\nfunc listedFile('
    def outside_copy(data):
        prefix, body = data.split(start, 1)
        return prefix + end + body.split(end, 1)[1]
    def copy_operation(data):
        body = data.split(start, 1)[1].split(end, 1)[0]
        body = body.replace('(retErr error)', 'error')
        body = body.replace('\tdefer source.Close()\n', '')
        for handle in ('source', 'destinationFile'):
            cleanup = ('\tdefer func() {\n\t\tif closeErr := ' + handle + '.Close(); closeErr != nil {\n'
                '\t\t\tif retErr == nil {\n\t\t\t\tretErr = closeErr\n\t\t\t} else {\n'
                '\t\t\t\tretErr = errors.Join(retErr, closeErr)\n\t\t\t}\n\t\t}\n\t}()\n')
            body = body.replace(cleanup, '')
        body = re.sub(r'^\t+destinationFile.Close\(\)\n', '', body, flags=re.MULTILINE)
        return body.replace('\treturn destinationFile.Close()\n', '\treturn nil\n')
    original_tests = base_source(tests)
    restored_tests = (ROOT / tests).read_text().replace('\t"errors"\n', '')
    prefix, remainder = restored_tests.split('func TestOpenedArtifactCopyDestinationErrorsPreserveHandle(', 1)
    restored_tests = prefix + 'func TestClosedArtifactRejectsPayloadAccess(' + remainder.split('func TestClosedArtifactRejectsPayloadAccess(', 1)[1]
    for indent in ('\t', '\t\t\t'):
        cleanup = (indent + 'defer func() {\n' + indent + '\tif err := opened.Close(); err != nil {\n'
            + indent + '\t\tt.Error(err)\n' + indent + '\t}\n' + indent + '}()\n')
        restored_tests = restored_tests.replace(cleanup, indent + 'defer opened.Close()\n')
    prefix, pinned = restored_tests.split('func TestOpenedArtifactReadsPinnedDirectoryAfterReplacement(', 1)
    first, addition = pinned.split('\tcopied := filepath.Join(t.TempDir(), "stdout")\n', 1)
    restored_tests = prefix + 'func TestOpenedArtifactReadsPinnedDirectoryAfterReplacement(' + first + '\tif opened.Manifest().' + addition.split('\tif opened.Manifest().', 1)[1]
    prefix, matrix = restored_tests.split('\t\t\tif test.name == "over bound" {\n', 1)
    restored_tests = prefix + '\t\t})\n' + matrix.split('\t\t})\n', 1)[1]
    changed = subprocess.check_output(['git', 'diff', base, '--name-only', '--', 'tools/gomad3',
        'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    final_names = [item for item in names if item.startswith('final-')]
    checks = dict(baseline_matches_base=baseline_matches_base,
        all_gate_sources_stable=all(item['stable'] for item in receipts.values()),
        final_gate_sources_match_current=all(receipts[item]['source_after'] == current for item in final_names),
        protected_inputs_stable=protected_before['aggregate_sha256'] == protected_after['aggregate_sha256'] == admission['protected_aggregate_sha256'],
        protected_files=protected_after['files'], protected_sha256=protected_after['aggregate_sha256'],
        source_outside_CopyPayload_unchanged=outside_copy(original_source) == outside_copy(final_source),
        copy_operation_unchanged_except_result_and_cleanup=copy_operation(original_source) == copy_operation(final_source),
        original_test_bytes_recovered=restored_tests == original_tests,
        changed_product_paths=changed,
        all_original_documents_unchanged=all(digest(ROOT / path) == value for path, value in admission['protected_original_documents'].items()))
    assert all(checks[key] for key in ('baseline_matches_base', 'all_gate_sources_stable', 'final_gate_sources_match_current',
        'protected_inputs_stable', 'source_outside_CopyPayload_unchanged', 'copy_operation_unchanged_except_result_and_cleanup',
        'original_test_bytes_recovered', 'all_original_documents_unchanged')), checks
    assert changed == [source, tests] and len(before) == 21 and len(after) == 10 and len(resolved) == 11
    assert all(not receipts[item]['timed_out'] and receipts[item]['exit'] == (1 if item.endswith('-lint') else 0) for item in names)
    assert not (OUT / 'final-static.log').read_text()
    test_counts = {item: len(re.findall(r'^=== RUN   ', (OUT / (item + '.log')).read_text(), re.MULTILINE))
                   for item in names if 'test' in receipts[item]['command']}
    assert all(test_counts.values())
    mutation_names = ['mutation-nil-close-wrap', 'mutation-destination-second-close', 'mutation-restored']
    mutations = {item: json.loads((OUT / (item + '.json')).read_text()) for item in mutation_names}
    assert all(mutations[item]['stable'] and not mutations[item]['timed_out'] for item in mutation_names)
    assert mutations['mutation-nil-close-wrap']['exit'] == mutations['mutation-destination-second-close']['exit'] == 1
    assert mutations['mutation-restored']['exit'] == 0 and mutations['mutation-restored']['source_after'] == current
    assert '*errors.joinError' in (OUT / 'mutation-nil-close-wrap.log').read_text()
    assert 'file already closed' in (OUT / 'mutation-destination-second-close.log').read_text()
    result = dict(status='SOURCE_PROGRESS_ONLY', task='fn-109-gomad-deepen-modules-and-tool-interfaces.30',
        flow_status='in_progress', base_commit=base, branch='gomad', commits=[], commit_range='',
        requested_model='gpt-6.1-sol', requested_effort='high', tier='session (jev-unavailable(no_key))',
        actual_model_metadata_available=False, platform=protected_after['platform'], source_checks=checks,
        source_final={path: current[path] for path in changed},
        gates={item: dict(receipt=item + '.json', log=receipt['log'], exit=receipt['exit'],
            elapsed_seconds=receipt['elapsed_seconds'], stable=receipt['stable']) for item, receipt in receipts.items()},
        test_run_entries=test_counts, tests=[' '.join(receipts[item]['command']) for item in names], prs=[],
        pinned_tools=receipts['baseline-lint']['tools'], lint_config_sha256=receipts['baseline-lint']['config_sha256'],
        lint=dict(total_before=len(before), total_after=len(after), mapped_before=11, mapped_after=0,
            before=before, after=after, resolved=resolved, introduced=[],
            cleanup_mapping=[dict(path=path, baseline_line=line) for path, lines in mapping.items() for line in lines],
            residual='All 10 residual path/message/linter identities match baseline; the two unchanged reflection switches shift lines.',
            broader_scope='Historical 419 is not a fresh whole-scope count.'),
        test_first=dict(policy_red='Actual baseline-lint.log reproduces all eleven admitted unchecked-close findings before production changes',
            characterization='Real-file destination errors, validation precedence, handle reuse and pinned stdout copy pass before production changes',
            runtime_close_regression_proof=False),
        test_mutations={item: dict(receipt=item + '.json', log=receipt['log'], exit=receipt['exit'],
            elapsed_seconds=receipt['elapsed_seconds'], stable=receipt['stable']) for item, receipt in mutations.items()},
        first_close=dict(inspected_sources=protected_after['pinned_file_close_source'], genuine_first_close_fault_exercised=False,
            limitation='No legitimate deterministic first-Close or simultaneous operation/destination/source-close fault was reproduced; conditional error identity and traversal order are source-inspected.'),
        behavior=dict(nil_cleanup='Preserves exact existing returned object/type/unwrap shape',
            single_close='Adopts the exact cleanup error when the existing result is nil; destination success-Close remains direct',
            genuine_cleanup_failure='Formerly ignored source/branch destination cleanup failures become visible; joins preserve primary, destination, source traversal order and add cleanup text'),
        generator=dict(inputs_inspected='Makefile VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS and validate recipes',
            source_is_generator_input=False, disposition='make validate passes without regeneration; protected generator/runtime/protocol/pin/config/dependency inputs unchanged'),
        boundaries=dict(historical_test='TestRecordAndArtifactHaveSeparateOwners is absent; TestPackageArchitecture enforces separate record/artifact owners',
            executed=['TestPackageArchitecture', 'TestPublicPackagesDoNotExportTypeAliases', 'TestArchitecturePublicSignatureFixtures',
                      'TestRunnerRequestsCompileInExternalModule', 'TestRunnerExternalConsumerCompiles']),
        review=dict(owner='root', status='pending fresh independent source review', formal='conductor-deferred; no formal SHIP'),
        remaining=['Original R13/R18/R19, task12/predecessors and task21 acceptance',
            'Matched original first-baseline fixed identities and full/formal qualification',
            'Patched-runtime native darwin/arm64 and linux/amd64 and affected consumer/integration/qualification gates',
            'Genuine first-Close and simultaneous operation/destination/source Close faults',
            'Post-validation growth/hash, partial destination writes, Chmod and Sync fault execution',
            '10 residual artifact diagnostics and broader lint qualification'],
        handover=str(OUT / 'handover.md'), owned_delegates=0, live_command_handles=0, pending_commands=0,
        worker_git_writes=0, worker_flow_lifecycle_writes=0)
    (OUT / (name + '.json')).write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(dict(status=result['status'], source_checks=checks, source_final=result['source_final'],
        lint_before=len(before), lint_after=len(after), test_run_entries=test_counts)))
    sys.exit(0)
if kind == 'protect':
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim',
        'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {path: digest(ROOT / path) for path in paths
              if path not in ('tools/gomad3/artifact/open.go', 'tools/gomad3/artifact/opened_test.go')}
    receipt = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest(),
        captured=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        selection='tracked Gomad modules, root go.mod/go.sum and lint config excluding only artifact/open.go and artifact/opened_test.go',
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
    command = ['bash', '-c', 'git diff --check -- artifact/open.go artifact/opened_test.go && ' + str(Path(GO).parent / 'gofmt') + ' -l artifact/open.go artifact/opened_test.go']
else:
    raise ValueError(kind)
before = snapshot()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
with (OUT / (name + '.log')).open('w') as log:
    try:
        result = subprocess.run(command, cwd=cwd, env=ENV, stdout=log, stderr=subprocess.STDOUT, timeout=600)
        child_exit, timed_out = result.returncode, False
    except subprocess.TimeoutExpired:
        child_exit, timed_out = None, True
after = snapshot()
receipt = dict(command=command, cwd=str(cwd), environment={key: ENV.get(key) for key in
    ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')},
    started=started, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    elapsed_seconds=time.monotonic() - clock, exit=child_exit, timed_out=timed_out,
    source_before=before, source_after=after, stable=before == after,
    tools={path: digest(path) for path in (GO, LINT, ERROR)}, config_sha256=digest(CONFIG),
    log=name + '.log', log_sha256=digest(OUT / (name + '.log')))
(OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({key: receipt[key] for key in ('command', 'exit', 'timed_out', 'elapsed_seconds', 'stable', 'log')}))
