import datetime
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
ARCH = ROOT/'tools/gomad3/internal/gomadtool/architecture'
def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

repair = json.loads((OUT/'repair-evidence.json').read_text())
subprocess.run(['python3',str(OUT/'repair-audit.py')],cwd=ROOT,check=True)
final = json.loads((OUT/'repair-final.json').read_text())['source']
assert {str(p.relative_to(ROOT)):digest(p) for p in sorted(ARCH.glob('*.go'))} == final
assert (ARCH/'standard.go').read_bytes() == (OUT/'sources/writer-stage/standard.go').read_bytes()
assert (ARCH/'error_provenance_test.go').read_bytes().startswith((OUT/'sources/writer-stage/error_provenance_test.go').read_bytes())
names = ['package','focused','boundaries','broader','consumer','lint','errortype','validate','static','map-candidate','map-baseline','worker-audit','worker-audit-stable']
references = []
logs = {}
tools = json.loads((OUT/'final-package.json').read_text())['tools']
tools['/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt'] = '83ecb88aa19246f24d28a91a107c6774af21b416e7ab69609612cf44107e99f3'
for suffix in names:
    name = 'review-repair-'+suffix
    path = OUT/(name+'.json')
    receipt = json.loads(path.read_text())
    expected_exit = 1 if suffix in ('lint','map-candidate','worker-audit') else 0
    assert receipt['exit'] == expected_exit and not receipt['timed_out'] and receipt['timeout_seconds'] is None
    assert receipt['stable'] and receipt['source_before'] == receipt['source_after'] == final
    assert receipt['protected_before'] == receipt['protected_after'] == {'files':1042,'aggregate_sha256':'b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61'}
    assert receipt['tools'] == tools
    for tool,expected in tools.items():
        assert digest(tool) == expected
    assert receipt['config_sha256'] == digest(ROOT/'.github/.golangci.yml') == '2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43'
    assert receipt['runner_sha256'] == digest(OUT/'review-repair-runner.py')
    assert receipt['cwd'] == str(ROOT if suffix.startswith('worker-audit') else ROOT/'tools/gomad3')
    env = receipt['environment']
    assert [env[k] for k in ('GOWORK','GOTOOLCHAIN','GOPROXY','GOFLAGS')] == ['off','local','off','']
    assert env['GOMADSEED'] is None and env['GOMAD3_CHILD_SEED'] is None
    assert env['PATH'].split(':')[0] == '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin'
    start,end = [datetime.datetime.fromisoformat(receipt[k]) for k in ('started','ended')]
    assert start.tzinfo and end.tzinfo and receipt['elapsed_seconds'] >= 0
    assert abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
    assert digest(OUT/receipt['log']) == receipt['log_sha256']
    logs[suffix] = (OUT/receipt['log']).read_text()
    references.append(dict(path=path.name,sha256=digest(path),exit=receipt['exit'],elapsed_seconds=receipt['elapsed_seconds'],log=receipt['log'],log_sha256=receipt['log_sha256'],command=receipt['command']))
    if suffix not in ('focused','map-candidate','map-baseline','worker-audit','worker-audit-stable','boundaries','broader'):
        mapped = 'repair-'+suffix
        reference = next(r for r in repair['command_receipts'] if r['path']==mapped+'.json')
        assert receipt['command'] == shlex.split(reference['command'])
    if suffix in ('boundaries','broader'):
        reference = next(r for r in repair['command_receipts'] if r['path']=='repair-root-'+suffix+'.json')
        assert receipt['command'] == shlex.split(reference['command'])
    if suffix == 'focused':
        assert receipt['command'] == ['/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go','test','-count=1','-tags','test_dep','-v','./internal/gomadtool/architecture','-run','^(TestErrorProvenance|TestEffectCallbackContextsAndUnwrapReturns|TestDependencyInitialization|TestThirdPartyInitialization|TestStandardStartupIdentity|TestMemorySummarySourceIdentity|TestCallbackContainerMutations|TestRangeAssignmentSlots|TestImplicitCallbackPrecedence|TestPureMemoryFormattingAndJSON)']
    if suffix.startswith('worker-audit'):
        assert receipt['command'] == ['python3',str(OUT/'repair-audit.py')]
    if suffix in ('map-candidate','map-baseline'):
        assert receipt['command'] == ['python3',str(OUT/'review-repair-map-probe.py'),suffix.removeprefix('map-')]
        assert receipt['baseline_production_sha256'] == {name:digest(OUT/'sources/baseline'/name) for name in ('effects.go','standard.go')}
for suffix in ('package','focused'):
    log = logs[suffix]
    assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M)) == 35
    assert log.count('stock-host causal fixture:') == 35 and log.count('package edges=0 effects=') == 70
    assert not re.search(r'^(--- FAIL|FAIL)',log,re.M)
    bodies = dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]+/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S))
    assert len(bodies) == 35
    for case,body in bodies.items():
        count = int(re.search(r'actual callbacks=(\d+)',body).group(1))
        observations = re.findall(r'metadata \{[^}]+\} package edges=0 effects=(.*)',body)
        assert len(observations) == 2 and '--- PASS: TestBehavior' in body
        for effect in observations:
            if count:
                assert all(v in effect for v in ('host-effect','record.Check','time.Now')) and 'unresolved-effect' not in effect
            elif case.endswith(('/unknown-return','/unknown-interface-elements')):
                assert 'unresolved-effect' in effect and 'unknown dynamic callback receiver' in effect
            else:
                assert effect == '[]'
    for case in ('dirty','clean','concrete-zero-element','clean-concrete-zero-element','nil-interface-elements','unknown-interface-elements'):
        assert 'TestErrorProvenanceEmptySliceAlias/'+case in bodies
    for name in ('TestEffectCallbackContextsAndUnwrapReturns','TestDependencyInitialization','TestThirdPartyInitialization','TestStandardStartupIdentity','TestMemorySummarySourceIdentity','TestCallbackContainerMutations','TestRangeAssignmentSlots','TestImplicitCallbackPrecedence','TestPureMemoryFormattingAndJSON'):
        assert re.search(r'^--- PASS: '+name+r' ',log,re.M)
for suffix,names_required in (('boundaries',repair['required_boundaries']),('broader',['TestPureModulesHaveNoHostEffects','TestExactModuleEdges','TestHostPackageVet'])):
    for name in names_required:
        assert len(re.findall(r'^=== RUN   '+re.escape(name)+r'$',logs[suffix],re.M)) == 1
        assert re.search(r'^--- PASS: '+re.escape(name)+r' ',logs[suffix],re.M)
for platform in ('darwin/arm64','linux/amd64','linux/arm64'):
    assert re.search(r'^    --- PASS: TestHostPackageVet/'+platform+r' ',logs['broader'],re.M)
assert len(re.findall(r'^--- PASS:',logs['consumer'],re.M)) == 30
assert logs['lint'] == (OUT/'baseline-lint.log').read_text() == (OUT/'repair-lint.log').read_text()
assert logs['static'] == logs['errortype'] == ''
assert 'frozen_artifacts ==' in logs['worker-audit'] and 'AssertionError' in logs['worker-audit']
assert json.loads(logs['worker-audit-stable'])['writes'] == 0
for suffix in ('map-baseline','map-candidate'):
    receipt = json.loads((OUT/('review-repair-'+suffix+'.json')).read_text())
    assert receipt['probe_sha256'] == digest(OUT/'review-repair-map-probe.py')
    first = json.loads(logs[suffix].splitlines()[0])
    assert first['stage'] == suffix.removeprefix('map-')
    substitutions = first['overlay']
    test_key = str(ARCH/'error_provenance_test.go')
    expected_keys = {test_key}
    if suffix == 'map-baseline':
        for name in ('effects.go','standard.go'):
            key = str(ARCH/name)
            expected_keys.add(key)
            assert substitutions[key] == str(OUT/'sources/baseline'/name)
            git_bytes = subprocess.check_output(['git','show',repair['base_commit']+':'+str((ARCH/name).relative_to(ROOT))],cwd=ROOT)
            assert hashlib.sha256(git_bytes).hexdigest() == receipt['baseline_production_sha256'][name]
    assert set(substitutions) == expected_keys
    temporary_test = Path(substitutions[test_key])
    assert temporary_test.is_absolute() and temporary_test.name == 'error_provenance_test.go'
    assert temporary_test.parent.name.startswith('task32-repair-map-review-')
    assert not temporary_test.exists()
    source = (ARCH/'error_provenance_test.go').read_text()+first['extra_fixture_source']
    assert hashlib.sha256(source.encode()).hexdigest() == first['test_source_sha256']
    assert logs[suffix].count('actual callbacks=1 writer count=0') == 1 and logs[suffix].count('actual callbacks=0 writer count=0') == 1
    bodies = dict(re.findall(r'^=== RUN   (TestRepairReviewMapAlias/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',logs[suffix],re.M|re.S))
    assert len(bodies) == 2
    for case,body in bodies.items():
        assert '--- PASS: TestBehavior' in body
        effects = re.findall(r'metadata \{[^}]+\} package edges=0 effects=(.*)',body)
        assert len(effects) == 2
        for effect in effects:
            if suffix == 'map-candidate':
                assert 'unresolved-effect' in effect and 'unresolved callback' in effect and 'time.Now' not in effect
            elif case.endswith('/clean'):
                assert effect == '[]'
            else:
                assert all(v in effect for v in ('host-effect','record.Check','canonicaljson.Dirty','time.Now'))
assert len(re.findall(r'^    --- FAIL: TestRepairReviewMapAlias/',logs['map-candidate'],re.M)) == 2
assert len(re.findall(r'^    --- PASS: TestRepairReviewMapAlias/',logs['map-baseline'],re.M)) == 2
assert json.loads(logs['map-candidate'].splitlines()[0])['test_source_sha256'] == json.loads(logs['map-baseline'].splitlines()[0])['test_source_sha256']
assert json.loads(logs['map-candidate'].splitlines()[0])['extra_fixture_source'] == json.loads(logs['map-baseline'].splitlines()[0])['extra_fixture_source']
assert not (ROOT/'tools/gomad3/.toolchain/bin/go').exists()
result = dict(task=repair['task'],base_commit=repair['base_commit'],verdict='NEEDS_WORK',closed_findings=['Initial Important converted fresh slice first-element alias finding'],actionable_findings=[dict(severity='Important',file='tools/gomad3/internal/gomadtool/architecture/effects.go',line=750,issue='Converted fresh maps lose first inserted concrete callback and reject clean callback; slice-only allocation repair leaves map alias regression')],historical_worker_receipts=17,historical_review_receipts=13,historical_files=89,writer_repair_receipts=14,writer_inconclusive_zero_selections=2,new_review_receipts=references,source=final,protected_files=1042,protected_aggregate_sha256='b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61',architecture_top_level_tests=len(re.findall(r'^--- PASS:',logs['package'],re.M)),focused_top_level_tests=len(re.findall(r'^--- PASS:',logs['focused'],re.M)),consumer_top_level_tests=30,causal_fixtures=35,metadata_observations=70,required_boundary_tests=5,broader_boundary_tests=3,map_probe=dict(baseline_exit=0,candidate_exit=1,candidate_failures=2,source_sha256=json.loads(logs['map-baseline'].splitlines()[0])['test_source_sha256']),lint=dict(inherited=4,introduced=0,resolved=0),errortype=0,audit_retry='First attempt failed whole-artifact equality because report was written concurrently; immutable stable retry passes after writes ended',qualified_native=False,formal_review=False,live_handles=0,report_sha256=digest(OUT/'repair-source-review.md'),review_audit_sha256=digest(__file__),review_runner_sha256=digest(OUT/'review-repair-runner.py'),review_probe_sha256=digest(OUT/'review-repair-map-probe.py'))
print(json.dumps(result,indent=2))
