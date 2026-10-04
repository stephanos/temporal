import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
BASE = '39fc19c4618322b6a939f5b603d9ba69aab00b9b'

def sha(data):
    return hashlib.sha256(data).hexdigest()

def original(path):
    return subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)

def observing(handle, zero=False):
    return ('\tdefer func() {\n\t\tif closeErr := ' + handle + '.Close(); closeErr != nil {\n'
        + ('\t\t\tverified = record.File{}\n' if zero else '')
        + '\t\t\tif retErr == nil {\n\t\t\t\tretErr = closeErr\n\t\t\t} else {\n'
        + '\t\t\t\tretErr = errors.Join(retErr, closeErr)\n\t\t\t}\n\t\t}\n\t}()\n')

def test_cleanup(data):
    return data.replace('\tdefer func() {\n\t\tif err := opened.Close(); err != nil {\n'
        '\t\t\tt.Error(err)\n\t\t}\n\t}()\n', '\tdefer opened.Close()\n')

names = ['baseline-package', 'baseline-lint', 'baseline-errortype', 'characterization-baseline',
    'final-package', 'final-focused', 'final-retained-private-public', 'final-lint',
    'final-errortype', 'final-boundary', 'final-validation', 'final-static',
    'mutation-nil-close-wrap', 'mutation-file-second-close', 'mutation-restored']
receipts = {name: json.loads((OUT / (name + '.json')).read_text()) for name in names}
current = {str(path.relative_to(ROOT)): sha(path.read_bytes())
    for path in sorted((ROOT / 'tools/gomad3/artifact').glob('*.go'))}
assert len(current) == 23
assert all(receipt['stable'] and not receipt['timed_out'] for receipt in receipts.values())
assert all(len(receipt['source_before']) == 23 for receipt in receipts.values())
assert all(sha((OUT / receipt['log']).read_bytes()) == receipt['log_sha256'] for receipt in receipts.values())
for name, receipt in receipts.items():
    want = 1 if name.endswith('-lint') or name in ('mutation-nil-close-wrap', 'mutation-file-second-close') else 0
    assert receipt['exit'] == want, (name, receipt['exit'])
    assert receipt['source_before'] == receipt['source_after']
    if name.startswith('final-') or name == 'mutation-restored':
        assert receipt['source_after'] == current
baseline = receipts['baseline-package']['source_before']
assert all(value == sha(original(path)) for path, value in baseline.items())
assert all(receipts[name]['source_before'] == baseline for name in ('baseline-lint', 'baseline-errortype'))
for path in ('tools/gomad3/artifact/store.go', 'tools/gomad3/artifact/target_pool.go'):
    assert receipts['characterization-baseline']['source_before'][path] == baseline[path]

admission = json.loads((OUT / 'source-admission.json').read_text())
protected = {name: json.loads((OUT / (name + '.json')).read_text()) for name in ('protected-before', 'protected-after')}
assert all(item['files'] == 1039 and item['aggregate_sha256'] == admission['protected_aggregate_sha256'] for item in protected.values())
assert all(sha((ROOT / path).read_bytes()) == value for path, value in admission['protected_original_documents'].items())
assert all(sha(original(path)) == value for path, value in admission['source_plan_bounds'].items() if path.startswith('tools/'))

prefix = 'tools/gomad3/artifact/'
recovery = {}
store = (ROOT / (prefix + 'store.go')).read_text()
restored = store.replace('func syncDirectory(path string) (operationErr, closeErr error)', 'func syncDirectory(path string) error')
directory, tail = restored.split('func syncDirectory(path string) error {', 1)
body, rest = tail.split('\nfunc syncDirectoryContext(', 1)
body = body.replace('\t\treturn err, nil\n', '\t\treturn err\n').replace(
    '\tdefer func() {\n\t\tcloseErr = directory.Close()\n\t}()\n', '\tdefer directory.Close()\n').replace(
    '\treturn directory.Sync(), nil\n', '\treturn directory.Sync()\n')
restored = directory + 'func syncDirectory(path string) error {' + body + '\nfunc syncDirectoryContext(' + rest
new_context = ('\terr, closeErr := syncDirectory(path)\n\tif err == nil {\n\t\terr = ctx.Err()\n\t}\n'
    '\tif closeErr != nil {\n\t\tif err == nil {\n\t\t\treturn closeErr\n\t\t}\n'
    '\t\treturn errors.Join(err, closeErr)\n\t}\n\treturn err\n')
assert restored.count(new_context) == 1
restored = restored.replace(new_context, '\tif err := syncDirectory(path); err != nil {\n\t\treturn err\n\t}\n\treturn ctx.Err()\n')
assert restored.encode() == original(prefix + 'store.go')
recovery['store.go'] = sha(restored.encode())
pool = (ROOT / (prefix + 'target_pool.go')).read_text()
restored = pool.replace('(verified record.File, retErr error)', '(record.File, error)')
for handle in ('root', 'file'):
    assert restored.count(observing(handle, True)) == 1
    restored = restored.replace(observing(handle, True), '\tdefer ' + handle + '.Close()\n')
assert restored.encode() == original(prefix + 'target_pool.go')
recovery['target_pool.go'] = sha(restored.encode())
store_test = (ROOT / (prefix + 'store_test.go')).read_text()
before, addition = store_test.split('func TestSyncDirectoryContextPreservesPrimaryResults(', 1)
restored = before + 'func TestPrivatePayloadWritesLiteralMetadataAndBytes(' + addition.split('func TestPrivatePayloadWritesLiteralMetadataAndBytes(', 1)[1]
assert restored.encode() == original(prefix + 'store_test.go')
recovery['store_test.go'] = sha(restored.encode())
pool_test = test_cleanup((ROOT / (prefix + 'target_pool_test.go')).read_text()).replace('\t"context"\n', '')
before, addition = pool_test.split('func TestVerifySharedPayloadPreservesLiteralTarget(', 1)
restored = before + 'func TestPublishSharesOneTargetAcrossTheStoresOfOnePool(' + addition.split('func TestPublishSharesOneTargetAcrossTheStoresOfOnePool(', 1)[1]
restored = restored.replace('\t\t\toriginalEntry := poolEntries(t, store.TargetPool)[0]\n', '')
before, addition = restored.split('\t\t\tif test.poisoned {\n', 1)
restored = before + '\t\t})\n' + addition.split('\t\t})\n', 1)[1]
assert restored.encode() == original(prefix + 'target_pool_test.go')
recovery['target_pool_test.go'] = sha(restored.encode())
restored = test_cleanup((ROOT / (prefix + 'publication_test.go')).read_text())
assert restored.encode() == original(prefix + 'publication_test.go')
recovery['publication_test.go'] = sha(restored.encode())
changed = subprocess.check_output(['git', 'diff', BASE, '--name-only', '--', 'tools/gomad3',
    'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
assert sorted(changed) == sorted(admission['protected_exclusions'])

recipes = {
    'mutation-nil-close-wrap': (observing('root', True), '\tdefer func() {\n\t\tretErr = errors.Join(retErr, root.Close())\n\t}()\n'),
    'mutation-file-second-close': (observing('file', True), observing('file', True).replace('\t}()\n', '\t\tretErr = errors.Join(retErr, file.Close())\n\t}()\n')),
}
mutation_evidence = {}
for name, (search, replacement) in recipes.items():
    assert pool.count(search) == 1
    mutated = pool.replace(search, replacement)
    expected = dict(current)
    expected[prefix + 'target_pool.go'] = sha(mutated.encode())
    assert receipts[name]['source_before'] == expected
    mutation_evidence[name] = dict(path=prefix + 'target_pool.go', origin_sha256=current[prefix + 'target_pool.go'],
        search=search, replacement=replacement, occurrences=1, mutant_sha256=expected[prefix + 'target_pool.go'],
        receipt=name + '.json', limitation='Mutation sensitivity only; no first-Close OS-fault proof.')
assert '*errors.joinError' in (OUT / 'mutation-nil-close-wrap.log').read_text()
assert 'file already closed' in (OUT / 'mutation-file-second-close.log').read_text()

def findings(name):
    return [dict(path=path, line=int(line), column=int(column), message=message, linter=linter)
        for path, line, column, message, linter in re.findall(r'^(.*\.go):(\d+):(\d+): (.*) \((\w+)\)$',
            (OUT / (name + '.log')).read_text(), re.MULTILINE)]

before, after = findings('baseline-lint'), findings('final-lint')
mapping = {'store.go': (490,), 'target_pool.go': (216, 221), 'publication_test.go': (42, 132), 'target_pool_test.go': (262,)}
resolved = [item for item in before if item['linter'] == 'errcheck' and item['line'] in mapping.get(Path(item['path']).name, ())]
assert len(before) == 10 and len(after) == 4 and len(resolved) == 6
assert [item for item in before if item not in resolved] == after
counts = {name: len(re.findall(r'^=== RUN   ', (OUT / receipt['log']).read_text(), re.MULTILINE))
    for name, receipt in receipts.items() if 'test' in receipt['command']}
assert all(counts.values())
assert not (OUT / 'final-static.log').read_text()
assert not (OUT / 'final-errortype.log').read_text()
boundary_names = ['TestPackageArchitecture', 'TestPublicPackagesDoNotExportTypeAliases', 'TestArchitecturePublicSignatureFixtures',
    'TestRunnerRequestsCompileInExternalModule', 'TestRunnerExternalConsumerCompiles']
boundary_log = (OUT / 'final-boundary.log').read_text()
assert all('--- PASS: ' + name + ' (' in boundary_log for name in boundary_names)
result = dict(status='SOURCE_PROGRESS_ONLY', task=admission['task'], flow_status='in_progress',
    base_commit=BASE, branch='gomad', commits=[], commit_range='', prs=[],
    tier='Tier: session (jev-unavailable(no_key))', requested_model='gpt-6.1-sol', requested_effort='high',
    platform=protected['protected-after']['platform'],
    tests=[' '.join(receipt['command']) for receipt in receipts.values()],
    gates={name: dict(receipt=name + '.json', log=receipt['log'], exit=receipt['exit'],
        elapsed_seconds=receipt['elapsed_seconds'], log_sha256=receipt['log_sha256']) for name, receipt in receipts.items()},
    test_run_entries=counts, source_final=current,
    preservation_audit=dict(recipe='python3 ' + str(OUT / 'audit.py'), recovered_baseline_sha256=recovery,
        protected_files=1039, protected_aggregate_sha256=admission['protected_aggregate_sha256'],
        documents=admission['protected_original_documents'], changed_product_paths=changed),
    lint=dict(before=before, after=after, resolved=resolved, introduced=[], total_before=10, total_after=4,
        mapped_before=6, mapped_after=0, historical_scope='419 is historical, not a fresh whole-Gomad count'),
    baseline='package and errortype green; unfiltered lint red on 10 inherited findings, including six admitted policy violations',
    test_first='Real-file controls passed against original production; policy RED is the actual six baseline errcheck findings.',
    mutations=mutation_evidence, restored_receipt='mutation-restored.json',
    source_inspected=['directory operation/post-context selected before conditional cleanup merge; inner Close precedes post-context',
        'nil cleanup preserves primary identity; sole cleanup adopts raw error; dual cleanup joins primary first',
        'verifier file-before-root single release; cleanup failure clears metadata; caller keeps zero result/sharing and wrapper',
        'existing staging cleanup removes failed verifier link; renamed artifact and independently published pool winner remain owned'],
    unexecuted=['genuine first-Close OS fault', 'simultaneous operation/Close fault', 'verifier first-Close failure clearing metadata',
        'post-Sync cancellation timing and Close/post-context simultaneous failure'],
    generator='Makefile input lists and check recipes inspected; no artifact source is a generator input; make validate passed without regeneration',
    boundaries=dict(executed=boundary_names, absent_historical_test='TestRecordAndArtifactHaveSeparateOwners was not run'),
    pinned_tools=receipts['baseline-lint']['tools'], config_sha256=receipts['baseline-lint']['config_sha256'],
    pinned_sources=protected['protected-after']['pinned_close_join_sources'],
    review='Root owns fresh independent source review and commits; no formal review verdict claimed',
    remaining=['Original R13/R18/R19/task12/predecessors/task21 acceptance', 'matched original first-baseline fixed identities',
        'complete/full/formal and native darwin/arm64 plus linux/amd64 qualification', 'affected consumer/integration/qualification gates',
        'four inherited artifact lint findings; no new whole-Gomad count'],
    owned_delegates=0, live_command_handles=0, worker_git_writes=0, worker_flow_lifecycle_writes=0,
    handover=str(OUT / 'handover.md'))
(OUT / 'evidence.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(dict(status=result['status'], lint_before=10, lint_after=4, recovered_baseline=recovery,
    test_run_entries=counts, protected_files=1039, protected_sha256=admission['protected_aggregate_sha256'])))
