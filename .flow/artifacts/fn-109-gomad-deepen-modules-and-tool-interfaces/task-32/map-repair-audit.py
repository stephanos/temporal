import contextlib
import datetime
import hashlib
import io
import json
from pathlib import Path
import re
import shlex
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def inventory():
    return {str(path.relative_to(OUT)):digest(path) for path in sorted(OUT.rglob('*')) if path.is_file()}

evidence = json.loads((OUT / 'map-repair-evidence.json').read_text())
frozen_artifacts = inventory()
for path, expected in evidence['historical_files'].items():
    assert digest(OUT / path) == expected, path
assert len(evidence['historical_files']) == 167
for reference in evidence['handover_audit_refs']:
    assert digest(OUT / reference['path']) == reference['sha256']
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip() == evidence['base_commit']
assert subprocess.check_output(['git','branch','--show-current'],cwd=ROOT,text=True).strip() == 'gomad'

# Execute immutable historical auditors with only their current-source reads
# redirected to the exact saved slice-stage files. Their original assertions,
# tool/env/log/receipt checks and artifact-equality guard remain intact.
def substitute(source, before, after):
    assert source.count(before) == 1, before
    return source.replace(before, after)

saved_source = json.loads((OUT / 'repair-final.json').read_text())['source']
slice_audit = (OUT / 'repair-audit.py').read_text()
slice_audit = substitute(slice_audit,
    "\nactual = {str(path.relative_to(ROOT)): digest(path) for path in sorted(ARCH.glob('*.go'))}\n",
    "\nactual = stages['repair-final']\n")
slice_audit = substitute(slice_audit,
    "new_effects = (ARCH / 'effects.go').read_text()",
    "new_effects = (OUT / 'sources/repair-final/effects.go').read_text()")
slice_audit = substitute(slice_audit,
    "final_test = (ARCH / 'error_provenance_test.go').read_text()",
    "final_test = (OUT / 'sources/repair-final/error_provenance_test.go').read_text()")
def historical_slice_audit():
    namespace = {'__file__':str(OUT / 'repair-audit.py')}
    with contextlib.redirect_stdout(io.StringIO()) as output:
        exec(compile(slice_audit,str(OUT / 'repair-audit.py'),'exec'),namespace)
    result = json.loads(output.getvalue())
    assert [result[key] for key in ('historical_worker_receipts','historical_review_receipts','repair_receipts','causal_fixtures','writes')] == [17,13,14,35,0]
    return namespace

history = historical_slice_audit()
review_audit = (OUT / 'repair-review-audit.py').read_text()
review_audit = substitute(review_audit,
    "subprocess.run(['python3',str(OUT/'repair-audit.py')],cwd=ROOT,check=True)",
    'historical_slice_audit()')
review_audit = substitute(review_audit,
    "assert {str(p.relative_to(ROOT)):digest(p) for p in sorted(ARCH.glob('*.go'))} == final",
    'assert saved_source == final')
review_audit = substitute(review_audit,
    "(ARCH/'error_provenance_test.go').read_bytes().startswith",
    "(OUT/'sources/repair-final/error_provenance_test.go').read_bytes().startswith")
review_audit = substitute(review_audit,
    "source = (ARCH/'error_provenance_test.go').read_text()+first['extra_fixture_source']",
    "source = (OUT/'sources/repair-final/error_provenance_test.go').read_text()+first['extra_fixture_source']")
namespace = {'__file__':str(OUT / 'repair-review-audit.py'), 'historical_slice_audit':historical_slice_audit, 'saved_source':saved_source}
with contextlib.redirect_stdout(io.StringIO()) as output:
    exec(compile(review_audit,str(OUT / 'repair-review-audit.py'),'exec'),namespace)
review = json.loads(output.getvalue())
assert review == json.loads((OUT / 'repair-source-review-checks.json').read_text())
assert len(review['new_review_receipts']) == 13 and review['map_probe']['candidate_failures'] == 2
assert review['map_probe']['candidate_exit'] == 1 and review['map_probe']['baseline_exit'] == 0
assert 'whole-artifact equality' in review['audit_retry']

admission = history['old']['admission']
protected = {'files':1042, 'aggregate_sha256':admission['protected_aggregate_sha256']}
assert protected['aggregate_sha256'] == 'b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61'
stages = {}
for reference in evidence['snapshots']:
    assert digest(OUT / reference['path']) == reference['sha256']
    name = Path(reference['path']).stem
    stage = json.loads((OUT / reference['path']).read_text())
    stages[name] = stage['source']
    assert stage['protected'] == protected and stage['base_commit'] == evidence['base_commit']
    assert len(stage['source']) == 15
    for item in ('effects.go','standard.go','error_provenance_test.go'):
        key = str((ARCH / item).relative_to(ROOT))
        assert digest(OUT / 'sources' / name / item) == stage['source'][key]
assert stages['map-repair-pre'] == saved_source
actual = {str(path.relative_to(ROOT)):digest(path) for path in sorted(ARCH.glob('*.go'))}
assert actual == stages['map-repair-final']
for path, expected in history['old']['baseline'].items():
    if path not in admission['protected_exclusions']:
        assert actual[path] == expected
old_effects = (OUT / 'sources/map-repair-pre/effects.go').read_text()
insert = '\t\t\t\t} else if mapping, ok := pkg.Info.TypeOf(call).Underlying().(*types.Map); ok {\n\t\t\t\t\tv.elements = valueOf(mapping.Elem())\n\t\t\t\t\tv.elements.unknown = false\n'
anchor = '\t\t\t\t\tv.elements = valueOf(slice.Elem())\n\t\t\t\t\tv.elements.unknown = false\n'
assert old_effects.count(anchor) == 1
assert (ARCH / 'effects.go').read_text() == old_effects.replace(anchor,anchor+insert)
assert (ARCH / 'standard.go').read_bytes() == (OUT / 'sources/map-repair-pre/standard.go').read_bytes() == (OUT / 'sources/writer-stage/standard.go').read_bytes()
old_test = (OUT / 'sources/map-repair-pre/error_provenance_test.go').read_bytes()
assert old_test == (OUT / 'sources/repair-final/error_provenance_test.go').read_bytes()
for name in ('map-repair-test-first','map-repair-final'):
    test = (OUT / 'sources' / name / 'error_provenance_test.go').read_bytes()
    assert test.startswith(old_test)
    appended = test[len(old_test):].decode()
    assert appended.count('func TestErrorProvenanceEmptyMapAlias') == 1 and appended.count('{name:') == 4
assert (ARCH / 'error_provenance_test.go').read_bytes() == (OUT / 'sources/map-repair-test-first/error_provenance_test.go').read_bytes()
for item in ('effects.go','standard.go'):
    assert (OUT / 'sources/map-repair-test-first' / item).read_bytes() == (OUT / 'sources/map-repair-pre' / item).read_bytes()

logs = {}
reference_commands = {Path(ref['path']).stem.removeprefix('review-repair-'):ref['command'] for ref in review['new_review_receipts']}
prior = json.loads((OUT / 'repair-package.json').read_text())
for reference in evidence['command_receipts']:
    assert digest(OUT / reference['path']) == reference['sha256']
    receipt = json.loads((OUT / reference['path']).read_text())
    name = Path(reference['path']).stem
    stage = 'map-repair-test-first' if name == 'map-repair-causal-red' else 'map-repair-final'
    assert receipt['source_before'] == receipt['source_after'] == stages[stage] and receipt['stable']
    assert receipt['protected_before'] == receipt['protected_after'] == protected
    assert receipt['exit'] == (1 if name in ('map-repair-causal-red','map-repair-lint') else 0) == reference['expected_exit']
    assert not receipt['timed_out'] and receipt['timeout_seconds'] == 600
    assert receipt['cwd'] == str(ROOT / 'tools/gomad3') and receipt['environment'] == prior['environment']
    assert receipt['tools'] == prior['tools']
    for tool, expected in receipt['tools'].items():
        assert digest(tool) == expected
    assert receipt['config_sha256'] == digest(ROOT / '.github/.golangci.yml')
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    assert receipt['command'] == shlex.split(reference['command'])
    if name.startswith('map-repair-causal-'):
        assert receipt['command'] == [next(iter(prior['tools'])),'test','-count=1','-tags','test_dep','-v','./internal/gomadtool/architecture','-run','^TestErrorProvenanceEmptyMapAlias$']
    else:
        assert receipt['command'] == reference_commands[name.removeprefix('map-repair-')]
    start, end = (datetime.datetime.fromisoformat(receipt[key]) for key in ('started','ended'))
    assert start.tzinfo and end.tzinfo and 0 <= receipt['elapsed_seconds'] < 600
    assert abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
    logs[name] = (OUT / receipt['log']).read_text()
assert len(logs) == 11
red = logs['map-repair-causal-red']
assert set(re.findall(r'^    --- FAIL: (TestErrorProvenance\S+) ',red,re.M)) == {'TestErrorProvenanceEmptyMapAlias/'+case for case in ('dirty','clean','nil-interface-elements')}
dirty = history['dirty'] | {'TestErrorProvenanceEmptyMapAlias/dirty'}
for name, expected in (('map-repair-causal-red',4),('map-repair-causal-green',4),('map-repair-package',39),('map-repair-focused',39)):
    log = logs[name]
    bodies = dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]+/[^ \n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S))
    assert len(bodies) == expected and log.count('stock-host causal fixture:') == expected
    if name != 'map-repair-causal-red':
        assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M)) == expected
    for case, body in bodies.items():
        calls = 1 if case in dirty else 0
        count = 7 if case in {'TestErrorProvenanceWriter/fprint','TestErrorProvenanceWriter/fprintf','TestErrorProvenanceWriter/fprintln','TestErrorProvenanceWriter/nil-return'} else 0
        assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)',body) == [(str(calls),str(count))]
        assert '--- PASS: TestBehavior' in body
        for platform in ('linux amd64','darwin arm64'):
            effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)',body).group(1)
            if name == 'map-repair-causal-red':
                assert 'unresolved-effect' in effect and 'time.Now' not in effect
                if case.endswith(('/dirty','/clean')):
                    assert 'unresolved callback' in effect
                    assert ('returned callback escaped {' if calls else 'pure callback rejected {')+platform+'}' in body
                else:
                    assert 'unknown dynamic callback receiver' in effect
                    if case.endswith('/nil-interface-elements'):
                        assert 'pure callback rejected {'+platform+'}' in body
            elif calls:
                assert all(item in effect for item in ('host-effect','record.Check','time.Now')) and 'unresolved-effect' not in effect
                if case.startswith(('TestErrorProvenanceEmptyMapAlias/','TestErrorProvenanceEmptySliceAlias/')):
                    callback = 'canonicaljson.Leaf.String' if case.endswith('/concrete-zero-element') else 'canonicaljson.Dirty'
                elif case.startswith('TestErrorProvenanceUnwrap/'):
                    callback = 'canonicaljson.Leaf.As' if case.endswith('/as-named-slice') else 'canonicaljson.Leaf.Is'
                elif case.endswith('/external-typed-source'):
                    callback = 'dependency.Leaf.Error'
                elif case.endswith(('/function-payload','/slice-alias','/pointer-alias')):
                    callback = 'canonicaljson.Dirty'
                else:
                    callback = 'canonicaljson.Leaf.Is' if case.endswith('/wrapped-is') else 'canonicaljson.Leaf.Error'
                assert callback in effect
            elif case.endswith(('/unknown-return','/unknown-interface-elements')):
                assert 'unresolved-effect' in effect and 'unknown dynamic callback receiver' in effect and 'record.Check' in effect
            else:
                assert effect == '[]'
for suffix, names in (('boundaries',evidence['required_boundaries']),('broader',['TestPureModulesHaveNoHostEffects','TestExactModuleEdges','TestHostPackageVet'])):
    log = logs['map-repair-'+suffix]
    for name in names:
        assert len(re.findall(r'^=== RUN   '+name+r'$',log,re.M)) == 1
        assert re.search(r'^--- PASS: '+name+r' ',log,re.M)
    assert len(re.findall(r'^--- PASS:',log,re.M)) == len(names)
for platform in ('darwin/arm64','linux/amd64','linux/arm64'):
    assert re.search(r'^    --- PASS: TestHostPackageVet/'+platform+r' ',logs['map-repair-broader'],re.M)
for name, count in (('map-repair-package',24),('map-repair-focused',14),('map-repair-consumer',30)):
    assert len(re.findall(r'^--- PASS:',logs[name],re.M)) == count
for name in ('TestEffectCallbackContextsAndUnwrapReturns','TestDependencyInitialization','TestThirdPartyInitialization','TestStandardStartupIdentity','TestMemorySummarySourceIdentity','TestCallbackContainerMutations','TestRangeAssignmentSlots','TestImplicitCallbackPrecedence','TestPureMemoryFormattingAndJSON'):
    assert re.search(r'^--- PASS: '+name+r' ',logs['map-repair-focused'],re.M)
assert logs['map-repair-lint'] == (OUT / 'baseline-lint.log').read_text()
assert logs['map-repair-errortype'] == logs['map-repair-static'] == ''
assert not (ROOT / 'tools/gomad3/.toolchain/bin/go').exists()
assert inventory() == frozen_artifacts
print(json.dumps(dict(historical_files=167,historical_receipts=57,map_repair_receipts=11,protected_files=1042,architecture_sources=15,causal_fixtures=39,metadata_observations=78,map_alias_red=2,fresh_nil_map_red=1,required_boundaries=5,broader_boundaries=3,prior_inconclusive_zero_selections=2,prior_artifact_concurrency_failure_retained=1,inherited_lint=4,introduced_lint=0,resolved_lint=0,errortype=0,writes=0,qualification='developmental stock linux/arm64; metadata linux/amd64 + darwin/arm64'),sort_keys=True))
