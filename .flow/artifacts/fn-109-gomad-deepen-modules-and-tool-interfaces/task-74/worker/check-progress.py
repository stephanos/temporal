import collections
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

sys.dont_write_bytecode = True
import run as r

HISTORICAL = r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
RESEARCH = r.ROOT/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retention-helper-next-slice.md'
PRIOR = r.ROOT.parent/'fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-aggregate-lint.log'
COMPLEXITY = r.ROOT.parent/'fn-109-23-lint-complexity-candidate/.flow/tmp/fn10923-complexity'

def domain():
    assert r.sha(HISTORICAL) == '2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
    retained = {}
    for line in HISTORICAL.read_text().splitlines():
        parts = line.split('  ',1)
        if len(parts) == 2 and parts[1].startswith('tests/'):
            retained[parts[1]] = parts[0]
    assert len(retained) == 132
    top_tests = [p for p in retained if Path(p).parent == Path('tests') and p.endswith('_test.go')]
    discovered = sorted(str(p.relative_to(r.ROOT)) for p in (r.ROOT/'tests').glob('*_test.go') if p.is_file())
    assert len(top_tests) == 113 and sorted(top_tests) == discovered
    actual = {p:r.sha(r.ROOT/p) for p in retained}
    assert retained == actual
    return {'materialized_domain_files':132, 'discovered_top_level_tests':113, 'all_historical_hashes_match':True, 'files':actual}

if len(sys.argv) > 1 and sys.argv[1] == 'domain':
    print(json.dumps(domain(),sort_keys=True,indent=2))
    sys.exit(0)

def events(name):
    return [json.loads(line) for line in (r.OUTPUT/(name+'.log')).read_text().splitlines()]

def outcomes(name):
    result = {}
    for event in events(name):
        if event.get('Test') and event.get('Action') in ('pass','fail','skip'):
            key = event['Test']
            assert key not in result
            result[key] = event['Action']
    return result

original = subprocess.check_output(['/usr/bin/git','show',r.BASE+':'+r.SELECTED],cwd=r.ROOT)
current = (r.ROOT/r.SELECTED).read_bytes()
statement = b'configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)\n'
addition1 = b'\t'+statement
addition2 = b'\t\t\t\t\t'+statement
assert current.count(statement) == 2
assert b'\texecutor.shape = rankProbes(t, probes...)\n'+addition1+b'\tsummary, err := exploreWith' in current
assert b'\t\t\t\t\texecutor.after = order\n'+addition2+b'\t\t\t\t\tsummary, err := exploreWith' in current
recovered = current.replace(addition2,b'',1).replace(addition1,b'',1)
assert recovered == original
assert hashlib.sha256(original).hexdigest() == '87253da780594e54139a2359ef5d2655ffbf4905b54eb1380c3700f3edaf8b4e'
paths = subprocess.check_output(['/usr/bin/git','diff',r.BASE,'--name-only','--',':(exclude).flow'],cwd=r.ROOT,text=True).splitlines()
assert paths == [r.SELECTED]

old, new = outcomes('before-ordinary'), outcomes('after-ordinary')
assert old.keys() == new.keys()
changed = {key:[old[key],new[key]] for key in old if old[key]!=new[key]}
selected = outcomes('after-table')
assert len(selected) == 19 and set(selected.values()) == {'pass'}
assert outcomes('before-table') == {key:'fail' for key in selected}
failed_leaf = r.TABLE.strip('^$')+'/failures/choice-exploration'
retained_failures = {r.TABLE.strip('^$'),failed_leaf}
assert changed == {key:['fail','pass'] for key in selected if key not in retained_failures}
assert len(changed) == 17
assert {key:new[key] for key in selected} == {key:('fail' if key in retained_failures else 'pass') for key in selected}
trace_outcomes = outcomes('diagnostic-ordinary-trace')
assert trace_outcomes.keys() == old.keys()
assert {key:[old[key],trace_outcomes[key]] for key in old if old[key]!=trace_outcomes[key]} == {key:['fail','pass'] for key in selected}
trace_output = json.loads((r.OUTPUT/'diagnostic-trace-output.json').read_text())
assert r.sha(trace_output['trace_path']) == trace_output['trace_sha256']
assert Path(trace_output['trace_path']).stat().st_size == trace_output['trace_size_bytes']
for strategy in ('seed','choice-exploration','simulation-exploration'):
    before = outcomes('before-cold-'+strategy)
    after = outcomes('after-cold-'+strategy)
    assert before.keys() == after.keys() and len(before) == 7
    assert set(before.values()) == {'fail'} and set(after.values()) == {'pass'}
    for event in events('before-cold-'+strategy):
        if event.get('Action') == 'output' and event.get('OutputType') == 'error':
            assert ('stageError{stage:"validation"' if strategy == 'seed' else 'host is linux/arm64') in event['Output']
    assert not any(e.get('OutputType') == 'error' for e in events('after-cold-'+strategy))
controls = outcomes('before-controls')
assert controls == outcomes('after-controls') and set(controls.values()) == {'pass'}
assert len(controls) > 7
shared = outcomes('before-shared')
assert shared == outcomes('after-shared') and set(shared.values()) == {'fail'}
historical = re.findall(r'^\d+ (Test\S+)$',RESEARCH.read_text(),re.MULTILINE)
assert len(historical) == len(set(historical)) == 50
assert all(old[name] == 'fail' for name in historical)
assert set(historical) == set(selected)|set(shared)

def diagnostics(name):
    result = collections.defaultdict(list)
    for line,event in enumerate(events(name),1):
        if event.get('Test') and event.get('Action') == 'output' and event.get('OutputType') == 'error':
            result[event['Test']].append({'raw_line':line,'output':event['Output']})
    return dict(result)

def normalize(text,after):
    if after:
        text = re.sub(r'(retention_characterization_test\.go:)(\d+)(:)',lambda m:m[1]+str(int(m[2])-(2 if int(m[2])>=490 else 1 if int(m[2])>=170 else 0))+m[3],text)
    text = re.sub(r'(?<=\)\()0x[0-9a-f]+(?=\))','0x<POINTER>',text)
    temp = re.escape(r.environment()['TMPDIR'])
    text = re.sub(temp+r'/(Test[^/\n"]*?)[0-9]+/([0-9]{3})(?=/)',lambda m:r.environment()['TMPDIR']+'/'+m[1]+'<TEMP>/'+m[2],text)
    text = re.sub(r'campaign-[0-9]{8}T[0-9]{6}\.[0-9]+Z-[0-9a-f]{32}','campaign-<UTC-NONCE>',text)
    return text

before_diag, after_diag = diagnostics('before-ordinary'), diagnostics('after-ordinary')
assert 'sync artifact store: context deadline exceeded' in ''.join(row['output'] for row in after_diag[failed_leaf])
assert 'Reason:"artifact_publication"' in ''.join(row['output'] for row in after_diag[failed_leaf])
diag_changes = {}
for name in sorted(before_diag.keys()|after_diag.keys()):
    before_text = [normalize(row['output'],False) for row in before_diag.get(name,[])]
    after_text = [normalize(row['output'],True) for row in after_diag.get(name,[])]
    if before_text != after_text:
        diag_changes[name] = {'before':before_diag.get(name,[]),'after':after_diag.get(name,[])}
collateral = [name for name in shared if '/' in name and name.endswith(('/choice-exploration','/simulation-exploration'))]
assert len(collateral) == 14
assert set(diag_changes) == (set(selected)-{r.TABLE.strip('^$')})|set(collateral), json.dumps(diag_changes,indent=2)
for name in collateral:
    assert 'host is linux/arm64' in ''.join(row['output'] for row in before_diag[name])
    assert 'stageError{stage:"validation"' in ''.join(row['output'] for row in after_diag[name])

def blocks(path,prefix='tools/gomad3/'):
    lines = path.read_text().splitlines()
    result = []
    for index,line in enumerate(lines):
        if re.match('^'+re.escape(prefix)+r'[^:]+:\d+:\d+: .+$',line):
            assert index+2 < len(lines) and '^' in lines[index+2]
            result.append('\n'.join(lines[index:index+3]))
    return result

lint_paths = [r.OUTPUT/'before-aggregate-lint.log',r.OUTPUT/'after-aggregate-lint.log',PRIOR,COMPLEXITY/'final-fast-lint.log',COMPLEXITY/'final-gomad-lint.log']
lint = blocks(lint_paths[0])
assert all(blocks(path) == lint for path in lint_paths)
assert len(lint) == 50
lint_sha = hashlib.sha256('\n'.join(lint).encode()).hexdigest()
assert lint_sha == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
assert collections.Counter(re.search(r'\(([^()]*)\)$',block.splitlines()[0])[1] for block in lint) == {'forbidigo':8,'staticcheck':42}
for name in ('before-aggregate-lint','after-aggregate-lint','fast-lint'):
    assert json.loads((r.OUTPUT/(name+'.json')).read_text())['exit_code'] == 2
assert json.loads((r.OUTPUT/'errortype.json').read_text())['exit_code'] == 0
fast_text = (r.OUTPUT/'fast-lint.log').read_text()
assert 'Killed\n' in fast_text and 'Error 137' in fast_text
assert not blocks(r.OUTPUT/'fast-lint.log')
assert '-vettool=' not in fast_text
assert '-vettool=' not in (r.OUTPUT/'after-aggregate-lint.log').read_text()
assert '-vettool=' in (r.ROOT/'Makefile').read_text()
runner_prior = r.ROOT.parent/'fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/runner-lint.log'
assert blocks(r.OUTPUT/'runner-lint.log') == blocks(runner_prior)
assert len(blocks(r.OUTPUT/'runner-lint.log')) == 6
assert (r.OUTPUT/'gofmt.log').read_bytes() == b''
architecture = outcomes('architecture')
assert architecture and set(architecture.values()) == {'pass'}
validation = domain()
binding = json.loads((r.OUTPUT/'validate-binding.json').read_text())
receipt = json.loads((r.OUTPUT/'validate.json').read_text())
assert receipt['exit_code'] == 0
assert all(binding['source_manifest'][str(r.ROOT/p)] == digest == receipt['source_after'][str(r.ROOT/p)] for p,digest in validation['files'].items())
before_binding = json.loads((r.OUTPUT/'before-ordinary-binding.json').read_text())
after_binding = json.loads((r.OUTPUT/'after-ordinary-binding.json').read_text())
def product_inputs(binding):
    return {p:d for p,d in binding['source_manifest'].items() if p.startswith(str(r.ROOT)+'/') and '/.flow/' not in p}
before_inputs, after_inputs = product_inputs(before_binding), product_inputs(after_binding)
assert before_inputs.keys() == after_inputs.keys()
assert [p for p in before_inputs if before_inputs[p] != after_inputs[p]] == [str(r.ROOT/r.SELECTED)]
receipts = {}
for path in sorted(r.OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    value = json.loads(path.read_text())
    if 'exit_code' not in value:
        continue
    input_binding = json.loads((r.OUTPUT/(path.stem+'-binding.json')).read_text())
    assert r.sha(r.OUTPUT/(path.stem+'-binding.json')) == value['binding_sha256']
    assert r.sha(r.OUTPUT/(path.stem+'.log')) == value['log_sha256']
    assert value['source_before_after_equal'] and value['tools_before_after_equal'] and value['go_settings_stable_except_numeric_GOGCCFLAGS'] and value['all_commands_terminal'] and not value['timed_out']
    assert input_binding['source_manifest'] == value['source_after']
    assert input_binding['tools_manifest'] == value['tools_after']
    assert not value['remaining_group_members']
    settings = [dict(input_binding['actual_go_settings']),dict(value['actual_go_settings_after'])]
    for settings_value in settings:
        settings_value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)','go-build<VOLATILE>',settings_value['GOGCCFLAGS'])
    assert settings[0] == settings[1]
    assert input_binding['base'] == r.BASE
    receipts[path.stem] = {'exit_code':value['exit_code'],'elapsed_seconds':value['elapsed_seconds'],'argv':value['argv'],'cwd':value['cwd'],'receipt_sha256':r.sha(path),'log_sha256':value['log_sha256']}
print(json.dumps({'product_acceptance_pass':False,'disposition':'RED progress only: authoritative ordinary newly reached artifact-store deadline and fast lint killed137 remain unresolved. Evidence audit is not acceptance.','retained_selected_failures':sorted(retained_failures),'instrumented_diagnostic_counts':dict(collections.Counter(trace_outcomes.values())),'all_instrumented_actual_names_and_outcomes':trace_outcomes,'trace_output':trace_output,'fast_lint':'INCONCLUSIVE: root linter killed137; nested RED50 and integrated errortype unreached on this actual route; no retry.','base_commit':r.BASE,'whole_file_reconstruction':True,'base_sha256':hashlib.sha256(original).hexdigest(),'candidate_sha256':r.sha(r.ROOT/r.SELECTED),'recovered_sha256':hashlib.sha256(recovered).hexdigest(),'ordinary_before_counts':dict(collections.Counter(old.values())),'ordinary_after_counts':dict(collections.Counter(new.values())),'named_domain':len(old),'all_actual_names_and_outcomes':{name:[old[name],new[name]] for name in old},'changed_outcomes':changed,'historical50':{name:[old[name],new[name]] for name in historical},'cold_processes':3,'controls':len(controls),'shared':len(shared),'source_attributed_collateral_names':collateral,'all_before_diagnostics':before_diag,'all_after_diagnostics':after_diag,'diagnostic_changes':diag_changes,'diagnostic_normalization':'Only retention line offsets caused by the two insertions, typed-pointer addresses, TMPDIR/Test numeric temp suffixes and exact campaign UTC-nonce grammar. Raw outputs retained.','lint_blocks':50,'lint_blocks_sha256':lint_sha,'aggregate_errortype':'Standalone rc0. Integrated errortype unreached in nested RED50 routes and in fast route killed137; no inferred pass.','architecture_outcomes':len(architecture),'validation_domain':validation,'receipts':receipts},sort_keys=True,indent=2))
