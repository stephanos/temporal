import json
from pathlib import Path
import re
import sys
sys.dont_write_bytecode = True
import capture_v2 as r

prior_root = r.ROOT.parent/'fn-109-74-retention-candidate'
prior = prior_root/'.flow/tmp/fn10974-evidence'
historical = r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
assert r.sha(historical) == '2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
binding = json.loads((prior/'validate-binding.json').read_text())
receipt = json.loads((prior/'validate.json').read_text())
assert r.sha(prior/'validate-binding.json') == receipt['binding_sha256']
assert r.sha(prior/'validate.log') == receipt['log_sha256']
assert receipt['exit_code'] == 0 and not receipt['timed_out'] and not receipt['remaining_group_members']
assert binding['source_manifest'] == receipt['source_after']
assert binding['tools_manifest'] == receipt['tools_after']
old = {str(Path(p).relative_to(prior_root)):d for p,d in binding['source_manifest'].items() if p.startswith(str(prior_root)+'/') and '/.flow/' not in p}
now = {str(Path(p).relative_to(r.ROOT)):d for p,d in r.sources().items() if p.startswith(str(r.ROOT)+'/') and '/.flow/' not in p}
assert old.keys() == now.keys()
changed = {p:[old[p],now[p]] for p in old if old[p] != now[p]}
assert set(changed) == set(r.SELECTED)
current_tools = r.tools()
assert current_tools.keys() == binding['tools_manifest'].keys()
for path,record in binding['tools_manifest'].items():
    if path == 'make_exported_lookup':
        expected = dict(record)
        expected['PATH'] = expected['PATH'].replace(str(prior_root),str(r.ROOT))
        assert current_tools[path] == expected
    else:
        assert current_tools[path] == record
retained = {}
for line in historical.read_text().splitlines():
    parts = line.split('  ',1)
    if len(parts) == 2 and parts[1].startswith('tests/'):
        retained[parts[1]] = parts[0]
assert len(retained) == 132
top_tests = sorted(p for p in retained if Path(p).parent == Path('tests') and p.endswith('_test.go'))
discovered = sorted(str(p.relative_to(r.ROOT)) for p in (r.ROOT/'tests').glob('*_test.go') if p.is_file())
assert len(top_tests) == 113 and top_tests == discovered
assert all((r.ROOT/p).is_file() and r.sha(r.ROOT/p) == digest == binding['source_manifest'][str(prior_root/p)] for p,digest in retained.items())
current_binding = json.loads((r.OUTPUT/'host-vet-binding.json').read_text())
old_settings,now_settings = dict(binding['actual_go_settings']),dict(current_binding['actual_go_settings'])
for value in (old_settings,now_settings):
    value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)','go-build<VOLATILE>',value['GOGCCFLAGS'])
settings_differences = {key:[old_settings.get(key),now_settings.get(key)] for key in old_settings.keys()|now_settings.keys() if old_settings.get(key)!=now_settings.get(key)}
assert set(settings_differences) <= {'GOCACHE','GOMOD'}
assert old_settings['GOCACHE'] == str(prior_root/'tools/gomad3/.toolchain/generator-cache')
assert now_settings['GOCACHE'] == r.environment()['GOCACHE']
assert old_settings['GOMOD'] == str(prior_root/'tools/gomad3/go.mod')
assert now_settings['GOMOD'] == str(r.ROOT/'tools/gomad3/go.mod')
print(json.dumps({'validation_execution':'REUSED: task74 numeric exit0; no current candidate validation execution','go_settings_probe':'NOT_EXECUTED: consumed existing host-vet settings capture','prior_receipt_sha256':r.sha(prior/'validate.json'),'prior_binding_sha256':r.sha(prior/'validate-binding.json'),'prior_log_sha256':r.sha(prior/'validate.log'),'prior_argv':receipt['argv'],'prior_cwd':receipt['cwd'],'reconciled_product_identity_count':len(old),'unchanged_product_path_membership':True,'excluded_individually_unconsumed_runner_test_bodies':changed,'current_selected_file_sha256':{p:r.sha(r.ROOT/p) for p in r.SELECTED},'materialized_domain_files':132,'discovered_top_level_tests':113,'all_domain_hashes_match':True,'domain_files':retained,'tools_reconciled':True,'settings_differences':settings_differences,'source_basis':'Makefile validate routes version/protocol/boundary/patch/scripts/compatibility/qualification. Protocol inputs/outputs exclude these three Runner tests; scripts reads only shell/Perl; compatibility compiles its own package tests and production dependencies; qualification discovers ./tests only. All other product identities and path membership remain unchanged.','limitations':'Honors historical execution under its original worktree-local generator cache. Current host-vet settings confirm other effective Go values; GOCACHE differs deliberately and GOMOD relocates. No current generator-cache execution or hermetic cache claim.'},sort_keys=True,indent=2))
