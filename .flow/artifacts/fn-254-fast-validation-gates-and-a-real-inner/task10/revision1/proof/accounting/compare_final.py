import collections,json,pathlib
P=pathlib.Path(__file__).parent;b=json.loads((P/'baseline.json').read_text());f=json.loads((P/'final-inventory.json').read_text());issues=[]
def key(x):return tuple(x[k] for k in ['package','target_kind','target','test','ignored'])
old=collections.Counter(key(x['old']) for x in b['rust']);actual=collections.Counter(map(key,f['rust']));added=collections.Counter(map(key,f['new_rust']))
if actual!=old+added:issues.append('Rust full-coordinate multisets differ')
if any(v!=1 for v in actual.values()):issues.append('Duplicate Rust identity')
if set(old)&set(added):issues.append('New Rust control duplicates baseline')
if len({x['old'] for x in f['stages']})!=23 or {x['old'] for x in f['stages']}!={x['old'] for x in b['stages']}:issues.append('Old stage missing/duplicate/unmatched')
parts=[(p['owner'],p['new']) for x in f['stages'] for p in x['partitions']]
if len(parts)!=len(set(parts)):issues.append('Duplicate stage ownership partition')
expected={x['old'] for x in b['shell'] if x['granularity']!='aggregate'}
if len(f['shell'])!=len(expected) or {x['old'] for x in f['shell']}!=expected:issues.append('Old shell leaf missing/duplicate/unmatched')
for x in f['shell']:
 if x['status']!='source mapped' or not x['source']['lines']:issues.append({'shell':x['old'],'status':x['status']})
for x in f['rust_source_proof']['changed_existing_files']:
 if x['removed_function_names']:issues.append({'removed_functions':x})
report={'exact_accounting_pass':not issues,'issues':issues,'rust_baseline':sum(old.values()),'rust_added_log_identities':sum(added.values()),'ignored_baseline':sum(x['old']['ignored'] for x in b['rust']),'stages_baseline':23,'stage_ownership_partitions':len(parts),'shell_leaf_baseline':len(expected),'execution_claim':False,'source_commit':f['source_commit'],'unlogged_new_declarations':f['new_declared_framework_integration_cases_not_in_current_log'],'limitation':'Final source freeze hash renewal and execution results remain conductor-owned. No full gate or930-test execution inferred.'}
(P/'comparison.json').write_text(json.dumps(report,indent=2,sort_keys=True)+'\n');print(json.dumps(report))
