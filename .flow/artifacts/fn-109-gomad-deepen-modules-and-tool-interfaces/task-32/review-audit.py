import datetime
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess

OUT = Path(__file__).resolve().parent
ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

worker = json.loads((OUT / 'evidence.json').read_text())
subprocess.run(['python3', str(OUT / 'audit.py')], cwd=ROOT, check=True)
expected_tools = {
 '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go':'1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64',
 '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0':'acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc',
 '/tmp/fn109-lint-tools.ZdNe1t50/errortype':'db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc',
}
final = json.loads((OUT / 'writer-stage.json').read_text())['source']
names = ['review-package','review-focused','review-boundaries','review-consumer','review-lint','review-errortype','review-validate','review-static','review-worker-audit','review-alias-candidate','review-alias-baseline','review-alias-full-candidate','review-alias-full-baseline']
references = []
for path in [OUT / r['path'] for r in worker['command_receipts']] + [OUT / (n+'.json') for n in names]:
    receipt = json.loads(path.read_text())
    assert receipt['stable'] and not receipt['timed_out']
    assert receipt['source_before'] == receipt['source_after']
    assert receipt['protected_before'] == receipt['protected_after']
    assert receipt['config_sha256'] == '2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43'
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    for tool, expected in expected_tools.items():
        assert receipt['tools'][tool] == digest(tool) == expected
    for tool, expected in receipt['tools'].items():
        assert digest(tool) == expected
    start = datetime.datetime.fromisoformat(receipt['started'])
    end = datetime.datetime.fromisoformat(receipt['ended'])
    assert start.tzinfo is not None and end.tzinfo is not None
    assert 0 <= receipt['elapsed_seconds'] and abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
    env = receipt['environment']
    assert [env[k] for k in ('GOWORK','GOTOOLCHAIN','GOPROXY','GOFLAGS')] == ['off','local','off','']
    assert env['GOMADSEED'] is None and env['GOMAD3_CHILD_SEED'] is None
    assert env['PATH'].split(':')[0] == str(Path(next(iter(expected_tools))).parent)
    if path.stem in names:
        assert receipt['source_before'] == final
        expected_exit = 1 if path.stem in ('review-lint','review-alias-candidate','review-alias-full-candidate') else 0
        assert receipt['exit'] == expected_exit
        assert receipt['timeout_seconds'] is None
        references.append(dict(path=path.name,sha256=digest(path),exit=receipt['exit'],elapsed_seconds=receipt['elapsed_seconds'],log=receipt['log'],log_sha256=receipt['log_sha256']))
    else:
        assert receipt['timeout_seconds'] == 600
        assert receipt['elapsed_seconds'] < receipt['timeout_seconds']
        assert receipt['cwd'] == str(ROOT / 'tools/gomad3')
        index = next(i for i,r in enumerate(worker['command_receipts']) if r['path']==path.name)
        assert receipt['command'] == shlex.split(worker['tests'][index])
for stage in ('test-first','unwrap-stage','conversion-stage','writer-stage'):
    source = (OUT / 'sources' / stage / 'error_provenance_test.go').read_text()
    assert source.replace('\t\tt.Logf("metadata %s package edges=0 effects=%v", platform, findings)\n','') == (OUT / 'sources/test-first/error_provenance_test.go').read_text()
logs = {name:(OUT / (name+'.log')).read_text() for name in names}
for name in ('review-package','review-focused'):
    log = logs[name]
    assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M)) == 29
    assert log.count('stock-host causal fixture:') == 29
    assert log.count('package edges=0 effects=') == 58
    assert not re.search(r'^(?:--- FAIL|FAIL)',log,re.M)
    for match in re.finditer(r'^=== RUN   (TestErrorProvenance[^\n/]+/[^\n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S):
        case, body = match.groups()
        count = int(re.search(r'actual callbacks=(\d+)',body).group(1))
        for observation in re.findall(r'metadata \{[^}]+\} package edges=0 effects=(.*)',body):
            if count:
                assert 'host-effect' in observation and 'record.Check' in observation and 'time.Now' in observation
                assert 'unresolved-effect' not in observation
            elif case.endswith('/unknown-return'):
                assert 'unresolved-effect' in observation and 'unknown dynamic callback receiver' in observation
            else:
                assert observation == '[]'
boundaries = worker['required_boundaries']
for name in boundaries:
    assert len(re.findall(r'^=== RUN   '+re.escape(name)+r'$',logs['review-boundaries'],re.M)) == 1
    assert re.search(r'^--- PASS: '+re.escape(name)+r' ',logs['review-boundaries'],re.M)
assert logs['review-lint'] == (OUT/'baseline-lint.log').read_text() == (OUT/'final-lint.log').read_text()
assert logs['review-errortype'] == logs['review-static'] == ''
baseline = logs['review-alias-full-baseline']
candidate = logs['review-alias-full-candidate']
for log in (baseline,candidate):
    assert log.count('actual callbacks=1 writer count=0') == 2
    assert log.count('actual callbacks=0 writer count=0') == 1
    assert log.count('--- PASS: TestBehavior') == 3
assert len(re.findall(r'^    --- FAIL: TestReviewAliasProvenance/',candidate,re.M)) == 2
assert len(re.findall(r'^    --- PASS: TestReviewAliasProvenance/',baseline,re.M)) == 3
assert baseline.count('effects=[]') == 2
assert candidate.count('pure callback rejected') == 2
assert candidate.count('returned callback escaped') == 2
assert candidate.count('unresolved callback') == 8
first = json.loads(baseline.splitlines()[0])
second = json.loads(candidate.splitlines()[0])
assert first['test_source_sha256'] == second['test_source_sha256']
assert first['extra_fixture_source'] == second['extra_fixture_source']
result = dict(task=worker['task'], verdict='NEEDS_WORK', actionable_important=1,worker_receipts_verified=17,review_receipts=references,source=final,protected_files=1042,protected_aggregate_sha256='b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61',new_causal_fixtures=29,metadata_observations=58,architecture_top_level_tests=len(re.findall(r'^--- PASS:',logs['review-package'],re.M)),focused_top_level_tests=len(re.findall(r'^--- PASS:',logs['review-focused'],re.M)),consumer_top_level_tests=len(re.findall(r'^--- PASS:',logs['review-consumer'],re.M)),boundary_tests=boundaries,lint=dict(inherited=4,introduced=0,resolved=0),errortype=0,review_probe=dict(fixtures=3,candidate_failures=2,baseline_failures=0,test_source_sha256=first['test_source_sha256']),qualified_native=False,formal_review=False,live_handles=0)
result['finding'] = dict(severity='Important',file='tools/gomad3/internal/gomadtool/architecture/effects.go',line=750,issue='Conversion shallow-copy detaches first slice-element mutation from caller; dirty path lost and clean alias falsely rejected',required_repair='Preserve shared element provenance through converted slices with initially nil elements and add dirty/clean regressions in admitted new test')
result['new_test_stage_delta'] = 'Exactly one metadata t.Logf insertion; fixture recipes/counters/assertions unchanged, new test file bytes differ'
result['qualification_open'] = ['R8','R18','R19','task19','fn105D4','predecessors','task21','first-baseline fixed identities','full','completion','formal','darwin/arm64 patched-native','linux/amd64 patched-native','affected-consumer native']
result['report_sha256'] = digest(OUT / 'independent-source-review.md')
result['review_audit_sha256'] = digest(__file__)
result['review_runner_sha256'] = digest(OUT / 'review-runner.py')
result['review_probe_sha256'] = digest(OUT / 'review-alias-probe.py')
print(json.dumps(result,indent=2))
