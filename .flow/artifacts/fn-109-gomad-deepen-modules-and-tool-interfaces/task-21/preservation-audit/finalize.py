"""Bind retained R18 outputs and reusable historical receipts to fresh hashes."""
import datetime
import hashlib
import json
from pathlib import Path

ROOT=Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT=ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit'
inventory=json.loads((OUT/'inventory.json').read_text())
source=[]
for role,path in [('baseline',Path(inventory['baseline_root'])),('current',Path(inventory['current_root']))]:
    manifest=inventory['source_after'][role]
    mismatches=[k for k,h in manifest.items() if not (path/k).is_file() or hashlib.sha256((path/k).read_bytes()).hexdigest()!=h]
    assert not mismatches,(role,mismatches)
    source.append({'role':role,'root':str(path),'files':len(manifest),'canonical_inventory_sha256':hashlib.sha256(json.dumps(manifest,sort_keys=True,separators=(',',':')).encode()).hexdigest(),'mismatches':mismatches})

checks=[]
for name in ['task-19/round4-command-logs.sha256','task-21/baseline-reconstruction/input.sha256']:
    manifest=ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces'/name
    rows=[]
    for line in manifest.read_text().splitlines():
        h,k=line.split(None,1);k=k.lstrip('*');f=ROOT/k
        actual=hashlib.sha256(f.read_bytes()).hexdigest() if f.is_file() else None
        if actual!=h:rows.append({'path':k,'expected':h,'current':actual})
    checks.append({'manifest':str(manifest.relative_to(ROOT)),'manifest_sha256':hashlib.sha256(manifest.read_bytes()).hexdigest(),'entries':len(manifest.read_text().splitlines()),'mismatches':rows})
    allowed={'.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md'} if 'input.sha256' in name else set()
    assert all(r['path'] in allowed for r in rows),(name,rows)

def delta(a,b,p=''):
    if type(a)!=type(b):return [{'path':p,'baseline':a,'current':b}]
    if isinstance(a,dict):return sum((delta(a.get(k),b.get(k),p+'/'+k) for k in sorted(a.keys()|b.keys())),[])
    if isinstance(a,list):
        if len(a)!=len(b):return [{'path':p+'/length','baseline':len(a),'current':len(b)}]
        return sum((delta(x,y,p+'/'+str(i)) for i,(x,y) in enumerate(zip(a,b))),[])
    return [] if a==b else [{'path':p,'baseline':a,'current':b}]
pack_changes=[]
for section,rows in inventory['boundary_pack_diffs'].items():
    for row in rows:
        if row['baseline'] and row['current']:
            pack_changes.append({'path':row['path'],'leaf_changes':delta(json.loads((Path(inventory['baseline_root'])/row['path']).read_text()),json.loads((Path(inventory['current_root'])/row['path']).read_text()))})

result={'verified_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source_inventories':source,'retained_manifests':checks,'pack_semantic_diffs':pack_changes,'patched_toolchain_launcher_exists':(ROOT/'tools/gomad3/.toolchain/bin/go').exists(),'native_qualification':False}
(OUT/'final-binding.json').write_text(json.dumps(result,indent=2)+'\n')
entries={str(p.relative_to(OUT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(OUT.rglob('*')) if p.is_file() and p.name!='outputs.sha256'}
(OUT/'outputs.sha256').write_text(''.join(h+'  '+str((OUT/k).relative_to(ROOT))+'\n' for k,h in entries.items()))
print(json.dumps({'source':source,'retained_manifest_mismatches':[{'manifest':x['manifest'],'mismatches':x['mismatches']} for x in checks],'output_files':len(entries),'outputs_manifest_sha256':hashlib.sha256((OUT/'outputs.sha256').read_bytes()).hexdigest()},indent=2))
