import collections
import difflib
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
sys.dont_write_bytecode = True
import capture as r

STATEMENT = 'configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)'
EXPECTED = dict(zip(r.SELECTED, ('885b3df456c376522b484cfbdaa70e5f70ffe4d77a9c8af1d825740443606b68','c86e26d2c016af55427faea8927c9b224a89778b4b5d70d81dc4257a8163f168','140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df')))

def events(name):
    return [json.loads(line) for line in (r.OUTPUT/(name+'.log')).read_text().splitlines()]

def observations(name):
    outcomes, diagnostics = {}, collections.defaultdict(list)
    for number,event in enumerate(events(name),1):
        test = event.get('Test')
        if test and event.get('Action') in ('pass','fail','skip'):
            assert test not in outcomes
            outcomes[test] = event['Action']
        if test and event.get('Action') == 'output' and event.get('OutputType') in ('error','error-continue'):
            diagnostics[test].append({'raw_line':number,'output_type':event['OutputType'],'output':event['Output']})
    assert outcomes
    return {'outcomes':outcomes,'counts':dict(collections.Counter(outcomes.values())),'diagnostics':dict(diagnostics),'lossless_diagnostics':{name:''.join(row['output'] for row in rows) for name,rows in diagnostics.items()}}

files, line_maps = {}, {}
for name in r.SELECTED:
    original = subprocess.check_output(['/usr/bin/git','show',r.BASE+':'+name],cwd=r.ROOT)
    assert hashlib.sha256(original).hexdigest() == EXPECTED[name]
    current = (r.ROOT/name).read_bytes()
    old, new = original.decode().splitlines(keepends=True), current.decode().splitlines(keepends=True)
    added, mapping, recovered = [], {}, []
    for tag,a,b,c,d in difflib.SequenceMatcher(None,old,new,autojunk=False).get_opcodes():
        assert tag in ('equal','insert'), (name,tag)
        if tag == 'equal':
            recovered.extend(new[c:d])
            mapping.update({j+1:i+1 for i,j in zip(range(a,b),range(c,d))})
        else:
            assert d-c == 1 and new[c].strip() == STATEMENT
            added.append({'final_line':c+1,'baseline_before_line':a,'previous_line':new[c-1],'next_line':new[d]})
    assert ''.join(recovered).encode() == original
    files[name] = {'base_sha256':EXPECTED[name],'candidate_sha256':r.sha(r.ROOT/name),'recovered_sha256':hashlib.sha256(''.join(recovered).encode()).hexdigest(),'insertions':added}
    line_maps[Path(name).name] = mapping

def normalize(text,after):
    if after:
        for filename,mapping in line_maps.items():
            text = re.sub('('+re.escape(filename)+r':)(\d+)(:)',lambda match:match[1]+str(mapping.get(int(match[2]),int(match[2])))+match[3],text)
    text = re.sub(r'(?<=\)\()0x[0-9a-f]+(?=\))','0x<POINTER>',text)
    text = re.sub(re.escape(r.environment()['TMPDIR'])+r'/(Test[^/\n"]*?)[0-9]+/([0-9]{3})(?=/)',lambda match:r.environment()['TMPDIR']+'/'+match[1]+'<TEMP>/'+match[2],text)
    return re.sub(r'campaign-[0-9]{8}T[0-9]{6}\.[0-9]+Z-[0-9a-f]{32}','campaign-<UTC-NONCE>',text)

records = {}
for path in sorted(r.OUTPUT.glob('*.json')):
    if path.name.endswith('-binding.json'):
        continue
    receipt = json.loads(path.read_text())
    if 'exit_code' not in receipt:
        continue
    binding_path = r.OUTPUT/(path.stem+'-binding.json')
    binding = json.loads(binding_path.read_text())
    assert r.sha(binding_path) == receipt['binding_sha256']
    assert r.sha(r.OUTPUT/(path.stem+'.log')) == receipt['log_sha256']
    assert binding['base'] == r.BASE
    assert binding['source_manifest'] == receipt['source_after']
    assert binding['tools_manifest'] == receipt['tools_after']
    assert receipt['source_before_after_equal'] and receipt['tools_before_after_equal']
    assert receipt['all_commands_terminal'] and not receipt['remaining_group_members']
    if receipt['go_settings_probe'] == 'EXECUTED':
        settings = [dict(binding['actual_go_settings']),dict(receipt['actual_go_settings_after'])]
        for value in settings:
            value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)','go-build<VOLATILE>',value['GOGCCFLAGS'])
        assert settings[0] == settings[1]
    else:
        assert receipt['go_settings_probe'].startswith('NOT_EXECUTED')
        assert binding['actual_go_settings'].startswith('NOT_EXECUTED')
        assert receipt['actual_go_settings_after'].startswith('NOT_EXECUTED')
    records[path.stem] = {'exit_code':receipt['exit_code'],'child_returncode':receipt['child_returncode'],'timed_out':receipt['timed_out'],'elapsed_seconds':receipt['elapsed_seconds'],'argv':receipt['argv'],'cwd':receipt['cwd'],'process_group':receipt['process_group'],'remaining_group_members':receipt['remaining_group_members'],'receipt_sha256':r.sha(path),'binding_sha256':r.sha(binding_path),'log_sha256':receipt['log_sha256']}

tests = {name:observations(name) for name in records if name.startswith(('before-','after-')) and 'lint' not in name}
comparison = {}
if 'after-ordinary' in tests:
    assert [len(files[name]['insertions']) for name in r.SELECTED] == [1,3,2]
    before,after = tests['before-ordinary'],tests['after-ordinary']
    names = sorted(before['outcomes'].keys()|after['outcomes'].keys())
    comparison['actual_name_union'] = {name:[before['outcomes'].get(name),after['outcomes'].get(name)] for name in names}
    comparison['changed_outcomes'] = {name:values for name,values in comparison['actual_name_union'].items() if values[0]!=values[1]}
    comparison['new_actual_names'] = [name for name in names if name not in before['outcomes']]
    comparison['missing_actual_names'] = [name for name in names if name not in after['outcomes']]
    comparison['changed_diagnostics'] = {}
    for name in sorted(before['lossless_diagnostics'].keys()|after['lossless_diagnostics'].keys()):
        left = normalize(before['lossless_diagnostics'].get(name,''),False)
        right = normalize(after['lossless_diagnostics'].get(name,''),True)
        if left != right:
            comparison['changed_diagnostics'][name] = {'before':left,'after':right}
    before_binding = json.loads((r.OUTPUT/'before-ordinary-binding.json').read_text())
    after_binding = json.loads((r.OUTPUT/'after-ordinary-binding.json').read_text())
    def product(binding):
        return {p:d for p,d in binding['source_manifest'].items() if p.startswith(str(r.ROOT)+'/') and '/.flow/' not in p}
    left,right = product(before_binding),product(after_binding)
    assert left.keys() == right.keys()
    assert sorted(p for p in left if left[p] != right[p]) == sorted(str(r.ROOT/p) for p in r.SELECTED)

lint = {}
for name in records:
    if 'lint' not in name:
        continue
    lines = (r.OUTPUT/(name+'.log')).read_text().splitlines()
    blocks = []
    for index,line in enumerate(lines):
        if re.match(r'^(tools/gomad3/|runner/)[^:]+:\d+:\d+: .+$',line):
            assert index+2 < len(lines) and '^' in lines[index+2]
            blocks.append('\n'.join(lines[index:index+3]))
    lint[name] = {'ordered_blocks':blocks,'count':len(blocks),'ordered_blocks_sha256':hashlib.sha256('\n'.join(blocks).encode()).hexdigest(),'integrated_errortype_command_observed':any('-vettool=' in line for line in lines),'numeric_exit':records[name]['exit_code']}
print(json.dumps({'product_acceptance_pass':False,'base_commit':r.BASE,'files':files,'tests':tests,'comparison':comparison,'lint':lint,'receipts':records,'diagnostic_normalization':'Lossless error/error-continue assembly before actual source-line mapping; typed-pointer addresses, TMPDIR/Test numeric suffixes and exact campaign UTC-nonce grammar only. Raw diagnostic fragments retained.'},sort_keys=True,indent=2))
