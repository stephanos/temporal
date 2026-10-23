from pathlib import Path
import json,hashlib,re,subprocess,tomllib
root=Path.cwd();base=Path('.flow/tmp/base_commit').read_text().strip();prior='fcbed683496e760acab514788f0e4be26dd55d2c'
art=Path('.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task1');art.mkdir(parents=True,exist_ok=True)
manifest=json.loads(Path('.flow/artifacts/fn-227-parameterized-models-declared-worlds/task6-production-hashes.json').read_text())
changed_manifest=[p for p,v in manifest.items() if not Path(p).is_file() or hashlib.sha256(Path(p).read_bytes()).hexdigest()!=v['sha256']]
allowed=['model/umpire/backends/tlc/src/lib.rs','model/umpire/backends/tlc/src/runner.rs','model/umpire/lift/src/emit.rs','model/umpire/lift/src/lib.rs','model/umpire/lift/src/names.rs','model/umpire/testing/src/backend.rs']
assert set(changed_manifest)==set(allowed),changed_manifest
meta=json.loads(Path('.flow/tmp/fn-211.1/cargo-metadata.json').read_text());packages={p['id']:p for p in meta['packages']};nodes={p['id']:p for p in meta['resolve']['nodes']}
seed=next(p['id'] for p in meta['packages'] if p['name']=='umpire-temporal-guest');closure=set();todo=[seed]
while todo:
 p=todo.pop()
 if p in closure:continue
 closure.add(p)
 for d in nodes[p]['deps']:
  if any(k['kind']!='dev' for k in d['dep_kinds']):todo.append(d['pkg'])
roots=sorted({str(Path(packages[p]['manifest_path']).parent.relative_to(root)) for p in closure if packages[p]['source'] is None})
extra=['model/temporal/feature/activity','model/temporal/hosts/go','model/umpire/hosts/go','model/temporal/proto','model/umpire/proto','tests/testpilot_model_slice_test.go','common/testing/testpilot','service','tools/umpire','tools/canary']
paths=subprocess.check_output(['git','ls-files',*roots,*extra]).decode().splitlines();paths=sorted({p for p in paths if '/expected/' not in p and '/tests/' not in p and '/snapshots/' not in p and not p.endswith(('.md','.txt'))})
hashes={}
for p in paths:
 current=Path(p).read_bytes();old=subprocess.check_output(['git','show',f'{prior}:{p}'])
 assert current==old,p
 hashes[p]=hashlib.sha256(current).hexdigest()
oldlock=tomllib.loads(subprocess.check_output(['git','show',f'{base}:model/Cargo.lock']).decode());newlock=tomllib.loads(Path('model/Cargo.lock').read_text())
oldpkgs={p['name']+'@'+p['version']:p for p in oldlock['package']};newpkgs={p['name']+'@'+p['version']:p for p in newlock['package']}
assert all(newpkgs.get(n)==v for n,v in oldpkgs.items())
assert set(newpkgs)-set(oldpkgs)=={'umpire-apalache@0.0.1'}
inventory=json.loads(Path('.flow/tmp/fn-211.1/baseline-inventory.json').read_text())
changed=[p for p,h in inventory['artifacts'].items() if hashlib.sha256(Path(p).read_bytes()).hexdigest()!=h]
assert all(p.endswith('.tla') for p in changed),changed
# Declaration grouping changes, comments and whitespace do not change expressions or symbols.
def normalized(text):
 text=re.sub(r'\(\*.*?\*\)','',text,flags=re.S);text=re.sub(r'\\\*[^\n]*','',text)
 declarations=[]
 text=re.sub(r'\b(VARIABLES?|CONSTANTS?)\s+([A-Za-z_][A-Za-z_0-9]*(?:\s*,\s*[A-Za-z_][A-Za-z_0-9]*)*)',lambda m: declarations.extend((m[1].rstrip('S'),n) for n in re.split(r'\s*,\s*',m[2])) or '',text)
 return re.sub(r'\s+','',text),sorted(declarations)
for p in changed:
 old=subprocess.check_output(['git','show',f'{base}:{p}']).decode()
 assert normalized(old)==normalized(Path(p).read_text()),p
proof={'base':base,'prior_live_source':prior,'original_manifest_entries':len(manifest),'original_manifest_unchanged':len(manifest)-len(changed_manifest),'changed_non_runtime_manifest_entries':changed_manifest,'guest_normal_build_dependency_roots':roots,'runtime_and_live_closure_entries':len(hashes),'runtime_and_live_closure_sha256':hashes,'existing_lock_packages_unchanged':True,'only_added_lock_package':'umpire-apalache@0.0.1','baseline_artifacts':len(inventory['artifacts']),'changed_artifacts':changed,'all_other_artifacts_byte_identical':True,'tlc_executable_expressions_and_declared_symbols_unchanged':True,'guest_sha256':hashlib.sha256(Path('model/temporal/hosts/go/umpire_temporal_guest.wasm').read_bytes()).hexdigest(),'reused_live_evidence':'.flow/artifacts/fn-227-parameterized-models-declared-worlds/task6-live-evidence-reuse.json','scope':'Exact unchanged runtime/guest and live-source closure permits reuse of task5 slice/gaps/privacy evidence; no new live execution, gaps receipt or cross-OS claim.'}
(art/'compatibility.json').write_text(json.dumps(proof,indent=2)+'\n')
print('runtime/live closure entries',len(hashes),'annotations-only artifacts',len(changed))
