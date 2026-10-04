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

evidence = json.loads((OUT / 'repair-evidence.json').read_text())
frozen_artifacts = {str(path.relative_to(OUT)):digest(path) for path in sorted(OUT.rglob('*')) if path.is_file()}
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip() == evidence['base_commit']
for reference in evidence['handover_audit_refs']:
    assert digest(OUT / reference['path']) == reference['sha256']
for path, expected in evidence['historical_files'].items():
    assert digest(OUT / path) == expected, path
assert digest(OUT / 'independent-source-review.md') == 'ac11435f56e93932cb47063af3f70963a6b47bec25ba50486b1cd0fab0ab0fa2'
assert digest(OUT / 'independent-source-review-checks.json') == '67a30b4514884051cba5e275c85507ba628786bd3bcc1c2f50feee156b0194fd'
assert digest(OUT / 'review-audit.py') == 'd9973b65f27035ae699da520b6598c23b5cf14a011d062cec20140260cf0c9cf'

# Replay preserved assertions on their saved historical source view, never on
# the repaired tree. The only substitutions are the current-source read and
# the review auditor's invocation of that same historical worker audit.
historical_worker = (OUT / 'audit.py').read_text()
source_read = "actual = {str(path.relative_to(ROOT)): digest(path) for path in sorted(ARCH.glob('*.go'))}"
assert historical_worker.count(source_read) == 1
historical_worker = historical_worker.replace(source_read, 'actual = final')
def historical_worker_audit():
    namespace = {'__file__': str(OUT / 'audit.py')}
    with contextlib.redirect_stdout(io.StringIO()) as output:
        exec(compile(historical_worker, str(OUT / 'audit.py'), 'exec'), namespace)
    result = json.loads(output.getvalue())
    assert result['causal_red_cases'] == 14 and result['causal_fixtures'] == 29
    return namespace

old = historical_worker_audit()
historical_review = (OUT / 'review-audit.py').read_text()
invocation = "subprocess.run(['python3', str(OUT / 'audit.py')], cwd=ROOT, check=True)"
assert historical_review.count(invocation) == 1
historical_review = historical_review.replace(invocation, 'historical_worker_audit()')
namespace = {'__file__': str(OUT / 'review-audit.py'), 'historical_worker_audit': historical_worker_audit}
with contextlib.redirect_stdout(io.StringIO()) as output:
    exec(compile(historical_review, str(OUT / 'review-audit.py'), 'exec'), namespace)
review = json.loads(output.getvalue())
assert review == json.loads((OUT / 'independent-source-review-checks.json').read_text())
assert review['worker_receipts_verified'] == 17 and len(review['review_receipts']) == 13
assert review['review_probe']['candidate_failures'] == 2 and review['review_probe']['baseline_failures'] == 0
assert review['protected_files'] == 1042
for name in ('review-alias-full-baseline', 'review-alias-full-candidate'):
    receipt = json.loads((OUT / (name + '.json')).read_text())
    first = json.loads((OUT / receipt['log']).read_text().splitlines()[0])
    source = (OUT / 'sources/writer-stage/error_provenance_test.go').read_text() + first['extra_fixture_source']
    assert hashlib.sha256(source.encode()).hexdigest() == first['test_source_sha256']
    bodies = dict(re.findall(r'^=== RUN   (TestReviewAliasProvenance/[^\n]+)\n(.*?)(?=^=== RUN|\Z)', (OUT / receipt['log']).read_text(), re.M|re.S))
    for case in ('zero-slice-alias','zero-pointer-alias','zero-slice-clean'):
        body = bodies['TestReviewAliasProvenance/'+case]
        for platform in ('linux amd64','darwin arm64'):
            effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)', body).group(1)
            if name.endswith('candidate') and case != 'zero-pointer-alias':
                assert 'unresolved-effect' in effect and 'unresolved callback' in effect and 'time.Now' not in effect
            elif case == 'zero-slice-clean':
                assert effect == '[]'
            else:
                assert all(item in effect for item in ('host-effect','record.Check','canonicaljson.Dirty','time.Now'))
for reference in review['review_receipts']:
    receipt = json.loads((OUT / reference['path']).read_text())
    assert receipt['protected_before'] == receipt['protected_after'] == {'files':1042, 'aggregate_sha256':old['admission']['protected_aggregate_sha256']}
    assert receipt['cwd'] == str(ROOT if reference['path'] == 'review-worker-audit.json' else ROOT / 'tools/gomad3')

stages = {}
for reference in evidence['snapshots']:
    assert digest(OUT / reference['path']) == reference['sha256']
    name = Path(reference['path']).stem
    stage = json.loads((OUT / reference['path']).read_text())
    stages[name] = stage['source']
    assert len(stage['source']) == 15 and stage['base_commit'] == evidence['base_commit']
    assert stage['protected'] == {'files': 1042, 'aggregate_sha256': old['admission']['protected_aggregate_sha256']}
    for item in ('effects.go', 'standard.go', 'error_provenance_test.go'):
        key = str((ARCH / item).relative_to(ROOT))
        assert digest(OUT / 'sources' / name / item) == stage['source'][key]
assert stages['repair-baseline'] == old['final']
actual = {str(path.relative_to(ROOT)): digest(path) for path in sorted(ARCH.glob('*.go'))}
assert actual == stages['repair-final']
for path, expected in old['baseline'].items():
    if path not in old['admission']['protected_exclusions']:
        assert actual[path] == expected
old_effects = (OUT / 'sources/writer-stage/effects.go').read_text()
new_effects = (ARCH / 'effects.go').read_text()
make = '\t\t\tcase "make":\n\t\t\t\tv := valueOf(pkg.Info.TypeOf(call))\n\t\t\t\tif slice, ok := pkg.Info.TypeOf(call).Underlying().(*types.Slice); ok {\n\t\t\t\t\tv.elements = valueOf(slice.Elem())\n\t\t\t\t\tv.elements.unknown = false\n\t\t\t\t}\n\t\t\t\treturn v\n'
old_make = '\t\t\tcase "make":\n\t\t\t\treturn valueOf(pkg.Info.TypeOf(call))\n'
assert old_effects.count(old_make) == 1
assert new_effects == old_effects.replace(old_make, make)
assert (ARCH / 'standard.go').read_bytes() == (OUT / 'sources/writer-stage/standard.go').read_bytes()
old_test = (OUT / 'sources/writer-stage/error_provenance_test.go').read_text()
for name in stages:
    test = (OUT / 'sources' / name / 'error_provenance_test.go').read_text()
    assert test.startswith(old_test), name
    assert test[len(old_test):].count('func TestErrorProvenanceEmptySliceAlias') == (0 if name == 'repair-baseline' else 1)
assert (OUT / 'sources/repair-candidate/error_provenance_test.go').read_bytes() == (OUT / 'sources/repair-test-first/error_provenance_test.go').read_bytes()
controls = ('concrete-zero-element','clean-concrete-zero-element','nil-interface-elements','unknown-interface-elements')
final_test = (ARCH / 'error_provenance_test.go').read_text()
removed = [line for line in final_test.splitlines(keepends=True) if any('name: "'+name+'"' in line for name in controls)]
assert len(removed) == 4
assert ''.join(line for line in final_test.splitlines(keepends=True) if line not in removed) == (OUT / 'sources/repair-test-first/error_provenance_test.go').read_text()
for item in ('effects.go', 'standard.go'):
    assert (OUT / 'sources/repair-test-first' / item).read_bytes() == (OUT / 'sources/repair-baseline' / item).read_bytes()
    assert (OUT / 'sources/repair-candidate' / item).read_bytes() == (OUT / 'sources/repair-final' / item).read_bytes()

logs = {}
for reference in evidence['command_receipts']:
    assert digest(OUT / reference['path']) == reference['sha256']
    receipt = json.loads((OUT / reference['path']).read_text())
    assert receipt['exit'] == reference['expected_exit'] and not receipt['timed_out']
    assert receipt['stable'] and receipt['source_before'] == receipt['source_after'] == stages[reference['stage']]
    assert receipt['protected_before'] == receipt['protected_after'] == {'files':1042, 'aggregate_sha256':old['admission']['protected_aggregate_sha256']}
    assert receipt['command'] == shlex.split(reference['command'])
    assert receipt['cwd'] == str(ROOT / 'tools/gomad3') and receipt['timeout_seconds'] == 600
    start, end = (datetime.datetime.fromisoformat(receipt[key]) for key in ('started', 'ended'))
    assert start.tzinfo and end.tzinfo and 0 <= receipt['elapsed_seconds'] < 600
    assert abs((end-start).total_seconds()-receipt['elapsed_seconds']) < 1
    assert receipt['environment'] == old['json'].loads((OUT / 'final-package.json').read_text())['environment']
    tools = dict(json.loads((OUT / 'final-package.json').read_text())['tools'])
    tools['/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt'] = '83ecb88aa19246f24d28a91a107c6774af21b416e7ab69609612cf44107e99f3'
    assert receipt['tools'] == tools
    for path, expected in receipt['tools'].items():
        assert digest(path) == expected
    assert receipt['config_sha256'] == digest(ROOT / '.github/.golangci.yml')
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    logs[Path(reference['path']).stem] = (OUT / receipt['log']).read_text()
red = logs['repair-causal-red']
assert set(re.findall(r'^    --- FAIL: (TestErrorProvenance\S+) ', red, re.M)) == {'TestErrorProvenanceEmptySliceAlias/dirty','TestErrorProvenanceEmptySliceAlias/clean'}
for case, calls in (('dirty',1),('clean',0)):
    body = re.search(r'^=== RUN   TestErrorProvenanceEmptySliceAlias/'+case+r'\n(.*?)(?=^=== RUN|\Z)', red, re.M|re.S).group(1)
    assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)',body) == [(str(calls),'0')]
    assert '--- PASS: TestBehavior' in body
    for platform in ('linux amd64','darwin arm64'):
        effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)',body).group(1)
        assert 'unresolved-effect' in effect and 'unresolved callback' in effect and 'time.Now' not in effect
        assert ('returned callback escaped {' if calls else 'pure callback rejected {')+platform+'}' in body
assert logs['repair-lint'] == old['logs']['baseline-lint']
assert logs['repair-errortype'] == logs['repair-static'] == ''
dirty = old['dirty'] | {'TestErrorProvenanceEmptySliceAlias/dirty','TestErrorProvenanceEmptySliceAlias/concrete-zero-element'}
for name, expected in (('repair-alias-green',2),('repair-controls',6),('repair-package',35),('repair-focused',35)):
    log = logs[name]
    bodies = dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]+/[^ \n]+)\n(.*?)(?=^=== RUN|\Z)',log,re.M|re.S))
    assert len(bodies) == expected and log.count('stock-host causal fixture:') == expected
    assert len(re.findall(r'^    --- PASS: TestErrorProvenance',log,re.M)) == expected
    for case, body in bodies.items():
        count = 7 if case in {'TestErrorProvenanceWriter/fprint','TestErrorProvenanceWriter/fprintf','TestErrorProvenanceWriter/fprintln','TestErrorProvenanceWriter/nil-return'} else 0
        calls = 1 if case in dirty else 0
        assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)',body) == [(str(calls),str(count))]
        assert '--- PASS: TestBehavior' in body
        for platform in ('linux amd64','darwin arm64'):
            effect = re.search(r'metadata \{'+platform+r'\} package edges=0 effects=(.*)',body).group(1)
            if calls:
                assert 'host-effect' in effect and 'record.Check' in effect and 'time.Now' in effect and 'unresolved-effect' not in effect
                if case.startswith('TestErrorProvenanceEmptySliceAlias/'):
                    assert ('canonicaljson.Dirty' if case.endswith('/dirty') else 'canonicaljson.Leaf.String') in effect
            elif case.endswith(('/unknown-return','/unknown-interface-elements')):
                assert 'unresolved-effect' in effect and 'unknown dynamic callback receiver' in effect
            else:
                assert effect == '[]'
for name in evidence['required_boundaries']:
    assert len(re.findall(r'^=== RUN   '+name+r'$',logs['repair-root-boundaries'],re.M)) == 1
    assert re.search(r'^--- PASS: '+name+r' ',logs['repair-root-boundaries'],re.M)
for name in ('TestPureModulesHaveNoHostEffects','TestExactModuleEdges','TestHostPackageVet'):
    assert len(re.findall(r'^=== RUN   '+name+r'$',logs['repair-root-broader'],re.M)) == 1
    assert re.search(r'^--- PASS: '+name+r' ',logs['repair-root-broader'],re.M)
for name in ('repair-boundaries','repair-broader'):
    assert 'no tests to run' in logs[name] and not re.search(r'^=== RUN',logs[name],re.M)
assert len(re.findall(r'^--- PASS:',logs['repair-consumer'],re.M)) > 0
for name in ('TestEffectCallbackContextsAndUnwrapReturns','TestDependencyInitialization','TestThirdPartyInitialization','TestStandardStartupIdentity','TestMemorySummarySourceIdentity','TestCallbackContainerMutations','TestRangeAssignmentSlots','TestImplicitCallbackPrecedence','TestPureMemoryFormattingAndJSON'):
    assert re.search(r'^--- PASS: '+name+r' ',logs['repair-focused'],re.M)
assert not (ROOT / 'tools/gomad3/.toolchain/bin/go').exists()
assert frozen_artifacts == {str(path.relative_to(OUT)):digest(path) for path in sorted(OUT.rglob('*')) if path.is_file()}
print(json.dumps(dict(historical_worker_receipts=17,historical_review_receipts=13,historical_files=len(evidence['historical_files']),repair_receipts=len(logs),protected_files=1042,architecture_sources=15,causal_fixtures=35,metadata_observations=70,permanent_alias_red=2,required_boundaries=5,broader_boundaries=3,inconclusive_zero_selections=2,inherited_lint=4,introduced_lint=0,resolved_lint=0,errortype=0,qualification='developmental stock linux/arm64; metadata linux/amd64 + darwin/arm64',writes=0),sort_keys=True))
