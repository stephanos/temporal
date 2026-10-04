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
    return {str(p.relative_to(OUT)):digest(p) for p in sorted(OUT.rglob('*')) if p.is_file()}

def substitute(source, before, after):
    assert source.count(before) == 1, before
    return source.replace(before, after)

def run_saved(source, path, additions=None):
    namespace = {'__file__':str(OUT / path), **(additions or {})}
    with contextlib.redirect_stdout(io.StringIO()) as output:
        exec(compile(source,str(OUT / path),'exec'),namespace)
    return namespace, json.loads(output.getvalue())

evidence = json.loads((OUT / 'allocation-repair-evidence.json').read_text())
frozen = inventory()
assert len(evidence['historical_files']) == 234
for path, expected in evidence['historical_files'].items():
    assert digest(OUT / path) == expected, path
for path, expected in evidence['proof_files'].items():
    assert digest(OUT / path) == expected, path
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip() == evidence['base_commit']
assert subprocess.check_output(['git','branch','--show-current'],cwd=ROOT,text=True).strip() == 'gomad'

# Three exact current-source substitutions replay the immutable map auditor.
# Its nested slice/original auditors and every historical assertion remain intact.
source = (OUT / 'map-repair-audit.py').read_text()
source = substitute(source,
    "\nactual = {str(path.relative_to(ROOT)):digest(path) for path in sorted(ARCH.glob('*.go'))}\n",
    "\nactual = stages['map-repair-final']\n")
source = substitute(source,
    "assert (ARCH / 'effects.go').read_text() == old_effects.replace(anchor,anchor+insert)",
    "assert (OUT / 'sources/map-repair-final/effects.go').read_text() == old_effects.replace(anchor,anchor+insert)")
source = substitute(source,
    "assert (ARCH / 'error_provenance_test.go').read_bytes() == (OUT / 'sources/map-repair-test-first/error_provenance_test.go').read_bytes()",
    "assert (OUT / 'sources/map-repair-final/error_provenance_test.go').read_bytes() == (OUT / 'sources/map-repair-test-first/error_provenance_test.go').read_bytes()")
def historical_map_audit():
    namespace, result = run_saved(source,'map-repair-audit.py')
    assert [result[k] for k in ('historical_receipts','map_repair_receipts','causal_fixtures','writes')] == [57,11,39,0]
    return namespace

history = historical_map_audit()
saved_source = json.loads((OUT / 'map-repair-final.json').read_text())['source']
review_source = (OUT / 'final-review-audit.py').read_text()
for before, after in (
    ("subprocess.run(['python3',str(OUT/'map-repair-audit.py')],cwd=ROOT,check=True)", "historical_map_audit()"),
    ("assert {str(p.relative_to(ROOT)):digest(p) for p in sorted(ARCH.glob('*.go'))}==final", "assert saved_source==final"),
    ("(ARCH/'error_provenance_test.go').read_bytes().startswith", "(OUT/'sources/map-repair-final/error_provenance_test.go').read_bytes().startswith"),
    ("source=(ARCH/'error_provenance_test.go').read_text()+first['extra_fixture_source']", "source=(OUT/'sources/map-repair-final/error_provenance_test.go').read_text()+first['extra_fixture_source']"),
):
    review_source = substitute(review_source,before,after)
_, review = run_saved(review_source,'final-review-audit.py',dict(historical_map_audit=historical_map_audit,saved_source=saved_source))
assert review == json.loads((OUT / 'final-source-review-checks.json').read_text())
assert len(review['new_review_receipts']) == 12 and review['literal_probe']['candidate_failures'] == 2

protected = {'files':1042,'aggregate_sha256':'b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61'}
stages = {}
for name in ('pre','test-first','valid-test-first','origin-stage','final'):
    name = 'allocation-repair-' + name
    stage = json.loads((OUT / (name + '.json')).read_text())
    assert stage['protected'] == protected and stage['base_commit'] == evidence['base_commit']
    assert len(stage['source']) == 15
    stages[name] = stage['source']
    for item in ('effects.go','standard.go','error_provenance_test.go'):
        assert digest(OUT / 'sources' / name / item) == stage['source'][str((ARCH / item).relative_to(ROOT))]
assert stages['allocation-repair-pre'] == saved_source
assert {str(p.relative_to(ROOT)):digest(p) for p in sorted(ARCH.glob('*.go'))} == stages['allocation-repair-final']
old_test = (OUT / 'sources/map-repair-final/error_provenance_test.go').read_bytes()
for name in stages:
    assert (OUT / 'sources' / name / 'error_provenance_test.go').read_bytes().startswith(old_test)
    assert (OUT / 'sources' / name / 'standard.go').read_bytes() == (OUT / 'sources/writer-stage/standard.go').read_bytes()
for item in ('effects.go','standard.go'):
    for name in ('test-first','valid-test-first'):
        assert (OUT / 'sources' / ('allocation-repair-'+name) / item).read_bytes() == (OUT / 'sources/allocation-repair-pre' / item).read_bytes()
invalid = (OUT / 'sources/allocation-repair-test-first/error_provenance_test.go').read_text()
before = '{name: "nil-new-interface", code: ' + chr(96) + 'v:=new(error);_ = fmt.Sprint(*v)' + chr(96) + '},'
after = '{name: "nil-new-interface", code: ' + chr(96) + 'v:=new(helper.Err);_ = fmt.Sprint(*v)' + chr(96) + ', helper: ' + chr(96) + 'type Err interface{Error()string}' + chr(96) + '},'
assert invalid.count(before) == 1
valid = (OUT / 'sources/allocation-repair-valid-test-first/error_provenance_test.go').read_text()
assert valid == invalid.replace(before,after)
for name in ('origin-stage','final'):
    assert (OUT / 'sources' / ('allocation-repair-'+name) / 'error_provenance_test.go').read_text() == valid
origin = (OUT / 'sources/allocation-repair-origin-stage/effects.go').read_text()
final = (OUT / 'sources/allocation-repair-final/effects.go').read_text()
copy_helper = final[final.index('func copyValue('):final.index('func addressValue(')]
assert origin.count('func addressValue(') == 1 and 'func copyValue(' not in origin
reconstructed = origin.replace('func addressValue(',copy_helper+'func addressValue(')
reconstructed = substitute(reconstructed,'converted := *args[0]','converted := copyValue(args[0], pkg.Info.TypeOf(call.Args[0]))')
reconstructed = substitute(reconstructed,'return &converted','return converted')
assert final == reconstructed
for item in ('effects.go','standard.go'):
    baseline_bytes = subprocess.check_output(['git','show',evidence['base_commit']+':'+str((ARCH/item).relative_to(ROOT))],cwd=ROOT)
    assert baseline_bytes == (OUT / 'sources/baseline' / item).read_bytes()
for path, expected in history['history']['old']['baseline'].items():
    if path not in history['history']['old']['admission']['protected_exclusions']:
        assert stages['allocation-repair-final'][path] == expected

prior = json.loads((OUT / 'repair-package.json').read_text())
GO = next(iter(prior['tools']))
logs = {}
reference_commands = {Path(r['path']).stem.removeprefix('map-repair-'):shlex.split(r['command']) for r in json.loads((OUT/'map-repair-evidence.json').read_text())['command_receipts']}
for reference in evidence['command_receipts']:
    name = Path(reference['path']).stem
    receipt = json.loads((OUT / reference['path']).read_text())
    assert digest(OUT / reference['path']) == reference['sha256']
    assert receipt['source_before'] == receipt['source_after'] == stages[reference['stage']] and receipt['stable']
    assert receipt['protected_before'] == receipt['protected_after'] == protected
    assert receipt['exit'] == reference['expected_exit'] and not receipt['timed_out'] and receipt['timeout_seconds'] == 600
    assert receipt['cwd'] == str(ROOT / 'tools/gomad3') and receipt['environment'] == prior['environment'] and receipt['tools'] == prior['tools']
    for tool, expected in receipt['tools'].items():
        assert digest(tool) == expected
    assert receipt['config_sha256'] == digest(ROOT / '.github/.golangci.yml')
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    assert receipt['command'] == shlex.split(reference['command'])
    suffix = name.removeprefix('allocation-repair-')
    if suffix in ('causal-red','causal-red-valid','origin-test','causal-green','baseline-controls'):
        pattern = '^TestErrorProvenance' if suffix == 'causal-green' else '^TestErrorProvenanceAllocation'
        command = [GO,'test','-count=1','-tags','test_dep','-v','./internal/gomadtool/architecture','-run',pattern]
        if suffix == 'baseline-controls':
            command[6:6] = ['-overlay',str(OUT / 'allocation-repair-baseline-overlay.json')]
            overlay = json.loads((OUT / 'allocation-repair-baseline-overlay.json').read_text())
            assert overlay == receipt['overlay'] == {'Replace':{str(ARCH/item):str(OUT/'sources/baseline'/item) for item in ('effects.go','standard.go')}}
            assert digest(OUT / 'allocation-repair-baseline-overlay.json') == receipt['overlay_sha256']
            effective = dict(stages[reference['stage']])
            for item in ('effects.go','standard.go'):
                effective[str((ARCH/item).relative_to(ROOT))] = digest(OUT/'sources/baseline'/item)
            assert effective == receipt['effective_source']
        assert receipt['command'] == command
    elif suffix == 'focused':
        assert receipt['command'] == [GO,'test','-count=1','-tags','test_dep','-v','./internal/gomadtool/architecture','-run','^(TestErrorProvenance|TestEffectCallbackContextsAndUnwrapReturns|TestDependencyInitialization|TestThirdPartyInitialization|TestStandardStartupIdentity|TestMemorySummarySourceIdentity|TestCallbackContainerMutations|TestRangeAssignmentSlots|TestImplicitCallbackPrecedence|TestPureMemoryFormattingAndJSON)']
    else:
        assert receipt['command'] == reference_commands[suffix]
    start, end = [datetime.datetime.fromisoformat(receipt[k]) for k in ('started','ended')]
    assert start.tzinfo and end.tzinfo and 0 <= receipt['elapsed_seconds'] < 600
    assert abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
    logs[suffix] = (OUT / receipt['log']).read_text()
assert len(logs) == 14

aliases = {'TestErrorProvenanceAllocationAliases/'+family+'/'+mode for family in ('new-array','new-function','new-map','new-slice','map-literal','array-address-literal') for mode in ('dirty','clean')}
copy_cases = {'TestErrorProvenanceAllocationControls/'+name for name in ('array-value-copy','struct-value-copy')}
empty_cases = {'TestErrorProvenanceAllocationControls/'+name for name in ('empty-make-map-values','empty-map-literal-values','empty-make-slice-values','empty-slice-literal-values','empty-array-literal-values')}
controls_dirty = {'TestErrorProvenanceAllocationControls/'+name for name in ('positive-array-zero','positive-new-array-zero','capacity-reslice-zero','nested-reference-copy','new-struct-dirty')}
dirty = history['dirty'] | {case for case in aliases if case.endswith('/dirty')} | controls_dirty
unknown = {'TestErrorProvenanceWriter/unknown-return','TestErrorProvenanceEmptySliceAlias/unknown-interface-elements','TestErrorProvenanceEmptyMapAlias/unknown-interface-elements','TestErrorProvenanceAllocationControls/unknown-interface-field'}
assert len(dirty) == 32 and len(unknown) == 4
def bodies(log,prefix='TestErrorProvenance'):
    return dict(re.findall(r'^=== RUN   ('+prefix+r'[^/\n]*/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S))
def failures(log,prefix='TestErrorProvenance'):
    return set(re.findall(r'^    --- FAIL: ('+prefix+r'\S+) ',log,re.M))
assert failures(logs['causal-red-valid']) == aliases | copy_cases | empty_cases
assert failures(logs['baseline-controls']) == copy_cases | empty_cases
assert failures(logs['origin-test']) == copy_cases
assert failures(logs['causal-red']) == aliases | copy_cases | empty_cases | {'TestErrorProvenanceAllocationControls/nil-new-interface'}
bad = bodies(logs['causal-red'])['TestErrorProvenanceAllocationControls/nil-new-interface']
assert 'imported as helper and not used' in bad and 'actual callbacks=' not in bad and 'metadata {' not in bad

def callback(case):
    if case.startswith('TestErrorProvenanceAllocation'):
        return 'canonicaljson.Leaf.String' if case.endswith(('/positive-array-zero','/positive-new-array-zero','/capacity-reslice-zero')) else 'canonicaljson.Dirty'
    if case.startswith(('TestErrorProvenanceEmptySliceAlias/','TestErrorProvenanceEmptyMapAlias/')):
        return 'canonicaljson.Leaf.String' if case.endswith('/concrete-zero-element') else 'canonicaljson.Dirty'
    if case.startswith('TestErrorProvenanceUnwrap/'):
        return 'canonicaljson.Leaf.As' if case.endswith('/as-named-slice') else 'canonicaljson.Leaf.Is'
    if case.endswith('/external-typed-source'):
        return 'dependency.Leaf.Error'
    if case.endswith(('/function-payload','/slice-alias','/pointer-alias')):
        return 'canonicaljson.Dirty'
    return 'canonicaljson.Leaf.Is' if case.endswith('/wrapped-is') else 'canonicaljson.Leaf.Error'

for suffix, count in (('causal-red-valid',28),('baseline-controls',28),('origin-test',28),('causal-green',67),('package',67),('focused',67)):
    log = logs[suffix]
    cases = bodies(log)
    assert len(cases) == count and log.count('stock-host causal fixture:') == count
    assert log.count('package edges=0 effects=') == 2*count
    if suffix in ('causal-green','package','focused'):
        assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M)) == count
        assert not failures(log)
    for case, body in cases.items():
        calls = 1 if case in dirty else 0
        writer_count = 7 if case in {'TestErrorProvenanceWriter/'+s for s in ('fprint','fprintf','fprintln','nil-return')} else 0
        assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)',body) == [(str(calls),str(writer_count))], (suffix,case)
        assert '--- PASS: TestBehavior' in body
        for platform in ('linux amd64','darwin arm64'):
            effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)',body).group(1)
            if suffix == 'causal-red-valid' and case in aliases:
                assert 'unresolved callback' in effect and 'unresolved-effect' in effect and 'time.Now' not in effect
                assert ('returned callback escaped {' if calls else 'pure callback rejected {')+platform+'}' in body
            elif suffix in ('causal-red-valid','baseline-controls','origin-test') and case in failures(log):
                inherited_callback = 'canonicaljson.Dirty' if case in copy_cases else 'canonicaljson.Leaf.String'
                assert all(part in effect for part in ('host-effect','record.Check',inherited_callback,'time.Now')) and calls == 0
                assert 'pure callback rejected {'+platform+'}' in body
            elif calls:
                assert all(part in effect for part in ('host-effect','record.Check',callback(case),'time.Now')) and 'unresolved-effect' not in effect
            elif case in unknown:
                assert 'unknown dynamic callback receiver' in effect and 'unresolved-effect' in effect and 'record.Check' in effect
            else:
                assert effect == '[]', (suffix,case,effect)
for suffix,names in (('boundaries',evidence['required_boundaries']),('broader',['TestPureModulesHaveNoHostEffects','TestExactModuleEdges','TestHostPackageVet'])):
    assert len(re.findall(r'^--- PASS:',logs[suffix],re.M)) == len(names)
    for name in names:
        assert len(re.findall(r'^=== RUN   '+name+r'$',logs[suffix],re.M)) == 1
        assert re.search(r'^--- PASS: '+name+r' ',logs[suffix],re.M)
for platform in ('darwin/arm64','linux/amd64','linux/arm64'):
    assert re.search(r'^    --- PASS: TestHostPackageVet/'+platform+r' ',logs['broader'],re.M)
for suffix,count in (('package',26),('focused',16),('consumer',30)):
    assert len(re.findall(r'^--- PASS:',logs[suffix],re.M)) == count
for name in ('TestEffectCallbackContextsAndUnwrapReturns','TestDependencyInitialization','TestThirdPartyInitialization','TestStandardStartupIdentity','TestMemorySummarySourceIdentity','TestCallbackContainerMutations','TestRangeAssignmentSlots','TestImplicitCallbackPrecedence','TestPureMemoryFormattingAndJSON'):
    assert re.search(r'^--- PASS: '+name+r' ',logs['focused'],re.M)
assert logs['lint'] == (OUT / 'baseline-lint.log').read_text()
assert logs['errortype'] == logs['static'] == ''

# Archived scout overlays are historical path data, not runtime dependencies.
# Reconstruct their three effective source bindings from retained stages + extra.
scout_inherited = {'TestAllocationAliasScout/'+s for s in ('array-address-zero-var/dirty','array-address-zero-var/clean','function-address-zero-var/clean')}
scout_aliases = {s.replace('TestErrorProvenanceAllocationAliases','TestAllocationAliasScout') for s in aliases}
for group, expected_count in (('main',24),('empty',5)):
    extra = (OUT / ('allocation-repair-scout-'+group+'-extra.go.txt')).read_bytes()
    for stage in ('baseline','candidate'):
        stem = 'allocation-repair-scout-'+group+'-'+stage
        receipt = json.loads((OUT / (stem+'.json')).read_text())
        log = (OUT / (stem+'.log')).read_text()
        assert receipt['stage'] == stage and receipt['base'] == evidence['base_commit'] and receipt['exit'] == 1
        assert receipt['source_before'] == receipt['source_after'] == stages['allocation-repair-pre']
        assert receipt['environment'] == prior['environment'] and receipt['cwd'] == prior['cwd']
        assert receipt['tool_sha256'] == digest(GO)
        assert receipt['probe_sha256'] == digest(OUT / ('allocation-repair-scout-'+group+'-script.txt'))
        assert receipt['extra_sha256'] == hashlib.sha256(extra).hexdigest()
        assert receipt['log_sha256'] == digest(OUT / (stem+'.log'))
        archived_overlay = OUT / (stem+'-overlay.json')
        assert receipt['overlay_sha256'] == digest(archived_overlay)
        assert json.loads(archived_overlay.read_text()) == {'Replace':receipt['overlay']}
        assert set(receipt['overlay']) == {str(ARCH / item) for item in ('effects.go','standard.go','error_provenance_test.go')}
        for item in ('effects.go','standard.go','error_provenance_test.go'):
            bound = (OUT / 'sources' / ('baseline' if stage == 'baseline' and item != 'error_provenance_test.go' else 'map-repair-final') / item).read_bytes()
            if item == 'error_provenance_test.go':
                bound += extra
            binding = receipt['source_bindings'][item]
            assert binding == dict(working_sha256=stages['allocation-repair-pre'][str((ARCH/item).relative_to(ROOT))],bound_sha256=hashlib.sha256(bound).hexdigest(),bytes=len(bound))
        test_name = 'TestAllocationAliasScout' + ('Empty' if group == 'empty' else '')
        assert receipt['command'] == [GO,'test','-count=1','-tags','test_dep','-v','-overlay',receipt['command'][7],'./internal/gomadtool/architecture','-run','^'+test_name+'$']
        start,end = [datetime.datetime.fromisoformat(receipt[k]) for k in ('started','ended')]
        assert start.tzinfo and end.tzinfo and 0 <= receipt['elapsed_seconds'] < 300
        assert abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
        cases = bodies(log,'TestAllocationAliasScout')
        assert len(cases) == expected_count
        assert log.count('stock-host causal fixture:') == expected_count
        assert log.count('package edges=0 effects=') == 2*expected_count
        assert failures(log,'TestAllocationAliasScout') == (set(cases) if group == 'empty' else scout_inherited | (scout_aliases if stage == 'candidate' else set()))
        for case,body in cases.items():
            count = 1 if group == 'main' and case.endswith('/dirty') else 0
            assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)',body) == [(str(count),'0')]
            assert '--- PASS: TestBehavior' in body
            for platform in ('linux amd64','darwin arm64'):
                effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)',body).group(1)
                if case in scout_aliases and stage == 'candidate':
                    assert 'unresolved callback' in effect and 'time.Now' not in effect
                elif group == 'empty':
                    assert count == 0 and all(part in effect for part in ('host-effect','canonicaljson.Leaf.String','time.Now'))
                elif case not in scout_inherited:
                    if count:
                        assert all(part in effect for part in ('host-effect','record.Check','canonicaljson.Dirty','time.Now'))
                    else:
                        assert effect == '[]'
assert digest(OUT / 'allocation-repair-scout-note.md') == 'f4669841f34dfcf540f0de873599c831e738ee711d4d1a9ec2dbfb48276c35e3'
assert not (ROOT / 'tools/gomad3/.toolchain/bin/go').exists()
assert inventory() == frozen
print(json.dumps(dict(historical_files=234,historical_receipts=80,archived_scout_receipts=4,allocation_receipts=14,protected_files=1042,architecture_sources=15,causal_fixtures=67,metadata_observations=134,introduced_alias_red=12,supplementary_inherited_red=7,origin_remaining_value_copy_red=2,inconclusive_unused_import=1,required_boundaries=5,broader_boundaries=3,consumer_tests=30,inherited_lint=4,introduced_lint=0,resolved_lint=0,errortype=0,writes=0,qualification='developmental stock linux/arm64; metadata linux/amd64 + darwin/arm64'),sort_keys=True))
