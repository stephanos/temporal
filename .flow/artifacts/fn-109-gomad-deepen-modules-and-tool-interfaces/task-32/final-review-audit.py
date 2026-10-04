import datetime
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess

ROOT=Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT=Path(__file__).resolve().parent
ARCH=ROOT/'tools/gomad3/internal/gomadtool/architecture'
def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

worker=json.loads((OUT/'map-repair-evidence.json').read_text())
subprocess.run(['python3',str(OUT/'map-repair-audit.py')],cwd=ROOT,check=True)
final=json.loads((OUT/'map-repair-final.json').read_text())['source']
assert {str(p.relative_to(ROOT)):digest(p) for p in sorted(ARCH.glob('*.go'))}==final
assert (ARCH/'standard.go').read_bytes()==(OUT/'sources/writer-stage/standard.go').read_bytes()
assert (ARCH/'error_provenance_test.go').read_bytes().startswith((OUT/'sources/repair-final/error_provenance_test.go').read_bytes())
suffixes=['package','focused','boundaries','broader','consumer','lint','errortype','validate','static','literal-candidate','literal-baseline','worker-audit']
tools=json.loads((OUT/'map-repair-package.json').read_text())['tools']
expected_protected={'files':1042,'aggregate_sha256':'b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61'}
logs={}
receipts=[]
for suffix in suffixes:
    path=OUT/('review-final-'+suffix+'.json')
    receipt=json.loads(path.read_text())
    expected_exit=1 if suffix in ('lint','literal-candidate') else 0
    assert receipt['exit']==expected_exit and receipt['stable'] and not receipt['timed_out'] and receipt['timeout_seconds'] is None
    assert receipt['source_before']==receipt['source_after']==final
    assert receipt['protected_before']==receipt['protected_after']==expected_protected
    assert receipt['tools']==tools
    for tool,expected in tools.items():
        assert digest(tool)==expected
    assert receipt['config_sha256']==digest(ROOT/'.github/.golangci.yml')=='2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43'
    assert receipt['runner_sha256']==digest(OUT/'review-final-runner.py')
    assert receipt['cwd']==str(ROOT if suffix=='worker-audit' else ROOT/'tools/gomad3')
    env=receipt['environment']
    assert [env[k] for k in ('GOWORK','GOTOOLCHAIN','GOPROXY','GOFLAGS')]==['off','local','off','']
    assert env['GOMADSEED'] is None and env['GOMAD3_CHILD_SEED'] is None
    assert env['PATH'].split(':')[0]=='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin'
    start,end=[datetime.datetime.fromisoformat(receipt[k]) for k in ('started','ended')]
    assert start.tzinfo and end.tzinfo and receipt['elapsed_seconds']>=0
    assert abs((end-start).total_seconds()-receipt['elapsed_seconds'])<1
    assert digest(OUT/receipt['log'])==receipt['log_sha256']
    if suffix=='worker-audit':
        expected_command=['python3',str(OUT/'map-repair-audit.py')]
    elif suffix.startswith('literal-'):
        expected_command=['python3',str(OUT/'review-final-map-literal-probe.py'),suffix.removeprefix('literal-')]
        assert receipt['probe_sha256']==digest(OUT/'review-final-map-literal-probe.py')
        assert receipt['baseline_production_sha256']=={name:digest(OUT/'sources/baseline'/name) for name in ('effects.go','standard.go')}
    else:
        reference=next(r for r in worker['command_receipts'] if r['path']=='map-repair-'+suffix+'.json')
        expected_command=shlex.split(reference['command'])
    assert receipt['command']==expected_command
    logs[suffix]=(OUT/receipt['log']).read_text()
    receipts.append(dict(path=path.name,sha256=digest(path),command=receipt['command'],exit=receipt['exit'],elapsed_seconds=receipt['elapsed_seconds'],log=receipt['log'],log_sha256=receipt['log_sha256']))
for suffix,top in (('package',24),('focused',14)):
    log=logs[suffix]
    assert len(re.findall(r'^--- PASS:',log,re.M))==top
    assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M))==39
    assert log.count('stock-host causal fixture:')==39 and log.count('package edges=0 effects=')==78
    assert not re.search(r'^(--- FAIL|FAIL)',log,re.M)
    bodies=dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]+/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S))
    assert len(bodies)==39
    for case,body in bodies.items():
        count=int(re.search(r'actual callbacks=(\d+)',body).group(1))
        assert count in (0,1) and '--- PASS: TestBehavior' in body
        effects=re.findall(r'metadata \{[^}]+\} package edges=0 effects=(.*)',body)
        assert len(effects)==2
        if count:
            if case.startswith(('TestErrorProvenanceEmptySliceAlias/','TestErrorProvenanceEmptyMapAlias/')):
                callback='canonicaljson.Leaf.String' if case.endswith('/concrete-zero-element') else 'canonicaljson.Dirty'
            elif case.startswith('TestErrorProvenanceUnwrap/'):
                callback='canonicaljson.Leaf.As' if case.endswith('/as-named-slice') else 'canonicaljson.Leaf.Is'
            elif case.endswith('/external-typed-source'):
                callback='dependency.Leaf.Error'
            elif case.endswith(('/function-payload','/slice-alias','/pointer-alias')):
                callback='canonicaljson.Dirty'
            else:
                callback='canonicaljson.Leaf.Is' if case.endswith('/wrapped-is') else 'canonicaljson.Leaf.Error'
        for effect in effects:
            if count:
                assert all(v in effect for v in ('host-effect','record.Check','time.Now',callback)) and 'unresolved-effect' not in effect
            elif case.endswith(('/unknown-return','/unknown-interface-elements')):
                assert 'unresolved-effect' in effect and 'unknown dynamic callback receiver' in effect
            else:
                assert effect=='[]'
    for name in ('TestEffectCallbackContextsAndUnwrapReturns','TestDependencyInitialization','TestThirdPartyInitialization','TestStandardStartupIdentity','TestMemorySummarySourceIdentity','TestCallbackContainerMutations','TestRangeAssignmentSlots','TestImplicitCallbackPrecedence','TestPureMemoryFormattingAndJSON'):
        assert re.search(r'^--- PASS: '+name+r' ',log,re.M)
for suffix,names in (('boundaries',worker['required_boundaries']),('broader',['TestPureModulesHaveNoHostEffects','TestExactModuleEdges','TestHostPackageVet'])):
    assert len(re.findall(r'^--- PASS:',logs[suffix],re.M))==len(names)
    for name in names:
        assert len(re.findall(r'^=== RUN   '+re.escape(name)+r'$',logs[suffix],re.M))==1
        assert re.search(r'^--- PASS: '+re.escape(name)+r' ',logs[suffix],re.M)
for platform in ('darwin/arm64','linux/amd64','linux/arm64'):
    assert re.search(r'^    --- PASS: TestHostPackageVet/'+platform+r' ',logs['broader'],re.M)
assert len(re.findall(r'^--- PASS:',logs['consumer'],re.M))==30
assert logs['lint']==(OUT/'baseline-lint.log').read_text()==(OUT/'map-repair-lint.log').read_text()
assert logs['errortype']==logs['static']==''
assert json.loads(logs['worker-audit'])['writes']==0
for suffix in ('literal-baseline','literal-candidate'):
    receipt=json.loads((OUT/('review-final-'+suffix+'.json')).read_text())
    first=json.loads(logs[suffix].splitlines()[0])
    assert first['stage']==suffix.removeprefix('literal-')
    source=(ARCH/'error_provenance_test.go').read_text()+first['extra_fixture_source']
    assert hashlib.sha256(source.encode()).hexdigest()==first['test_source_sha256']
    expected_keys={str(ARCH/'error_provenance_test.go')}
    if suffix=='literal-baseline':
        for name in ('effects.go','standard.go'):
            key=str(ARCH/name)
            expected_keys.add(key)
            assert first['overlay'][key]==str(OUT/'sources/baseline'/name)
            git_bytes=subprocess.check_output(['git','show',worker['base_commit']+':'+str((ARCH/name).relative_to(ROOT))],cwd=ROOT)
            assert hashlib.sha256(git_bytes).hexdigest()==receipt['baseline_production_sha256'][name]
    assert set(first['overlay'])==expected_keys
    temporary_test=Path(first['overlay'][str(ARCH/'error_provenance_test.go')])
    assert temporary_test.is_absolute() and temporary_test.name=='error_provenance_test.go' and temporary_test.parent.name.startswith('task32-final-map-literal-review-') and not temporary_test.exists()
    assert logs[suffix].count('actual callbacks=1 writer count=0')==1 and logs[suffix].count('actual callbacks=0 writer count=0')==1
    bodies=dict(re.findall(r'^=== RUN   (TestFinalReviewMapLiteralAlias/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',logs[suffix],re.M|re.S))
    assert len(bodies)==2
    for case,body in bodies.items():
        assert '--- PASS: TestBehavior' in body
        effects=re.findall(r'metadata \{[^}]+\} package edges=0 effects=(.*)',body)
        assert len(effects)==2
        for effect in effects:
            if suffix=='literal-candidate':
                assert 'unresolved-effect' in effect and 'unresolved callback' in effect and 'time.Now' not in effect
            elif case.endswith('/clean'):
                assert effect=='[]'
            else:
                assert all(v in effect for v in ('host-effect','record.Check','canonicaljson.Dirty','time.Now'))
assert len(re.findall(r'^    --- FAIL: TestFinalReviewMapLiteralAlias/',logs['literal-candidate'],re.M))==2
assert len(re.findall(r'^    --- PASS: TestFinalReviewMapLiteralAlias/',logs['literal-baseline'],re.M))==2
candidate=json.loads(logs['literal-candidate'].splitlines()[0])
baseline=json.loads(logs['literal-baseline'].splitlines()[0])
assert candidate['test_source_sha256']==baseline['test_source_sha256'] and candidate['extra_fixture_source']==baseline['extra_fixture_source']
assert not (ROOT/'tools/gomad3/.toolchain/bin/go').exists()
result=dict(task=worker['task'],base_commit=worker['base_commit'],verdict='NEEDS_WORK',closed_reproductions=['make-allocated slice dirty/clean alias','make-allocated map dirty/clean alias'],actionable_findings=[dict(severity='Important',file='tools/gomad3/internal/gomadtool/architecture/effects.go',line=750,issue='Accepted map-alias finding persists for empty map literal, whose first inserted callback is detached by conversion header copy')],historical_files=167,historical_receipts=57,map_repair_receipts=11,new_review_receipts=receipts,source=final,protected=expected_protected,architecture_top_level_tests=24,focused_top_level_tests=14,consumer_top_level_tests=30,causal_fixtures=39,metadata_observations=78,required_boundary_tests=5,broader_boundary_tests=3,literal_probe=dict(baseline_exit=0,candidate_exit=1,candidate_failures=2,source_sha256=baseline['test_source_sha256']),lint=dict(inherited=4,introduced=0,resolved=0),errortype=0,qualified_native=False,formal_review=False,live_handles=0,report_sha256=digest(OUT/'final-source-review.md'),review_audit_sha256=digest(__file__),review_runner_sha256=digest(OUT/'review-final-runner.py'),review_probe_sha256=digest(OUT/'review-final-map-literal-probe.py'))
print(json.dumps(result,indent=2))
