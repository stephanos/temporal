#!/usr/bin/env python3
"""Authorized exact idle parent cache cleanup for milestone disk admission only."""
import datetime,importlib.util,json,os,pathlib,shutil,stat,subprocess
HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
ART=ROOT/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire'
TARGET=pathlib.Path('/home/agent/.cache/codex-build/fn258-standalone-0547321')
LITERAL='/home/agent/.cache/codex-build/fn258-standalone-0547321'
SOURCE=pathlib.Path('/home/agent/.cache/codex-build/tmp/fn258-standalone-0547321/checkout')
FLOOR=50*1024**3
spec=importlib.util.spec_from_file_location('audit',ART/'disk/failed-final-pair-3f5-target/retain-audit.py');audit=importlib.util.module_from_spec(spec);spec.loader.exec_module(audit);audit.TARGET=TARGET
spec=importlib.util.spec_from_file_location('proof',HERE/'verify-proof.py');proof=importlib.util.module_from_spec(spec);spec.loader.exec_module(proof)
def save(n,x):(HERE/n).write_text(json.dumps(x,indent=2)+'\n')
def allocated():return int(subprocess.check_output(['du','-s','-B1',LITERAL]).split()[0])
def free():
 s=os.statvfs(TARGET.parent);return s.f_bavail*s.f_frsize
def guard_floor(label):
 x={'available_bytes':free(),'target_allocated_bytes':allocated(),'required_bytes':FLOOR};x['projected_available']=x['available_bytes']+x['target_allocated_bytes'];save(label,x)
 if x['projected_available']<FLOOR:raise RuntimeError('STOP: projected private free below50GiB; need additional exact idle cache admission')
 return x
def idle(label):
 x=audit.owners();save(label,x);assert not x['matches'];return x
def identity(p):
 s=p.stat();return {'path':str(p),'inode':s.st_ino,'device':s.st_dev,'uid':s.st_uid}
assert str(TARGET)==LITERAL and TARGET.resolve()==TARGET and not TARGET.is_symlink()
initial=guard_floor('admission-before.json');assert json.loads((HERE/'recovery-before.json').read_text())['proofs']['named']['result']=='pass'
source={'head':subprocess.check_output(['git','-C',str(SOURCE),'rev-parse','HEAD'],text=True).strip(),'tree':subprocess.check_output(['git','-C',str(SOURCE),'rev-parse','HEAD^{tree}'],text=True).strip(),'status':subprocess.check_output(['git','-C',str(SOURCE),'status','--porcelain'],text=True)}
assert source['head']=='05473210b48c3db862f22051eb7154ce98258f98' and not source['status']
owner=idle('ownership-before.json');inventory=audit.inventory();save('inventory-before.json',{'target':LITERAL,'records':inventory})
assert all(r['uid']==os.getuid() and r['kind']!='special' and not r.get('mountpoint') for r in inventory)
# Verify every retained record sourced from this exact target, including current payloads,
# identities, old/current stage receipts, native helpers, wasm, report/row native parity data.
coverage=[];covered=set()
for sub,key in [('standalone-0547321','records'),('standalone-named-0547321/retained-gates','objects')]:
 manifest=json.loads((ART/'validation'/sub/'recovery-manifest.json').read_text())
 for r in manifest[key]:
  if not r['source'].startswith(LITERAL+'/'):continue
  p=pathlib.Path(r['source']);assert p.is_file() and p.stat().st_size==r['size'] and audit.sha(p)==r['sha256'],str(p)
  if 'mode' in r:assert oct(stat.S_IMODE(p.stat().st_mode))==r['mode']
  coverage.append({'manifest':sub,'source':str(p),'sha256':r['sha256'],'role':r.get('role'),'recovery_object':r['recovery_object']});covered.add(str(p))
assert len(coverage)==322
save('current-required-payload-coverage.json',{'checked_records':len(coverage),'unique_target_paths':len(covered),'all_current_required_payloads_match_retained':True,'records':coverage,'scope':'Existing complete required and named authority archives and322explicit target source records; no new target archive, no acceptance gate.'})
# Hash current manifests/logs/receipts (outside deletion boundary); protect source/index identities.
protected_files=[]
for sub in ['standalone-0547321','standalone-named-0547321']:
 for f in (ART/'validation'/sub).rglob('*'):
  if f.is_file():protected_files.append({'path':str(f),'sha256':audit.sha(f),'size':f.stat().st_size})
for f in [SOURCE/'.git/index',SOURCE/'.git/HEAD']:
 if f.is_file():protected_files.append({'path':str(f),'sha256':audit.sha(f),'size':f.stat().st_size})
save('protected-files-before.json',protected_files)
protected_roots=[ROOT/'.flow/tmp/fn258-standalone-release-0547321',pathlib.Path('/home/agent/.cache/codex-build/fn258-final-pair-cde-054'),pathlib.Path('/home/agent/.cache/codex-build/tmp/fn258-final-pair-cde-054'),SOURCE,pathlib.Path('/home/agent/.cache/codex-build/tmp/umpire-standalone-Bnt8GY'),audit.CAS,ROOT.parent/'.recovery/fn258-guest-proof/objects']
root_ids=[identity(p) for p in protected_roots if p.is_dir()];save('protected-root-identities-before.json',root_ids)
# Full-inode kernel lease check, not mere presence of Cargo lock files.
keys={f"{os.major(r['device']):02x}:{os.minor(r['device']):02x}:{r['inode']}":r['path'] for r in inventory};locks=[]
for line in pathlib.Path('/proc/locks').read_text().splitlines():
 for token in line.split():
  if token in keys:locks.append({'path':keys[token],'record':line})
assert not locks
save('lease-state.json',{'held_target_locks':locks,'unreadable_system_fields':owner['unreadable'],'note':'File presence is not held lease. Full process cwd/argv/env/fd/maps checked before mutation.'})
tags=[]
for r in inventory:
 if r['kind']=='file' and pathlib.PurePosixPath(r['path']).name=='CACHEDIR.TAG':
  p=TARGET/r['path']
  if p.read_bytes().startswith(b'Signature: 8a477f597d28d172789f06886806bc55'):tags.append(p.parent)
tags=[p for p in tags if not any(p!=q and q in p.parents for q in tags)];save('real-cargo-tags.json',[str(p) for p in tags]);assert tags
idle('ownership-immediate.json');guard_floor('admission-immediate.json')
commands=[]
for child in tags:
 assert child==TARGET or TARGET in child.parents
 argv=['mise','exec','--','cargo','clean','--manifest-path',str(SOURCE/'Cargo.toml'),'--target-dir',str(child)]
 env=os.environ.copy();env.update(MISE_TRUSTED_CONFIG_PATHS=str(SOURCE),MISE_AUTO_INSTALL='false',MISE_EXEC_AUTO_INSTALL='false',MISE_NOT_FOUND_AUTO_INSTALL='false')
 result=subprocess.run(argv,cwd=SOURCE,env=env,capture_output=True,text=True);commands.append({'argv':argv,'exit':result.returncode,'stdout':result.stdout,'stderr':result.stderr});save('cargo-clean-results.json',commands);assert result.returncode==0
if TARGET.exists():
 remainder=audit.inventory();save('owned-remainder.json',{'records':remainder,'metadata_inventory_sha256':audit.sha(HERE/'inventory-before.json')})
 assert all(r['uid']==os.getuid() and r['kind']!='special' and not r.get('mountpoint') for r in remainder)
 idle('ownership-before-remainder.json');guard_floor('admission-before-remainder.json')
 assert str(TARGET)==LITERAL and TARGET.resolve()==TARGET and not TARGET.is_symlink()
 shutil.rmtree(TARGET)
assert not TARGET.exists() and not TARGET.is_symlink()
for r in protected_files:
 p=pathlib.Path(r['path']);assert p.is_file() and p.stat().st_size==r['size'] and audit.sha(p)==r['sha256'],str(p)
for r in root_ids:assert identity(pathlib.Path(r['path']))==r
assert subprocess.check_output(['git','-C',str(SOURCE),'rev-parse','HEAD'],text=True).strip()==source['head'] and subprocess.check_output(['git','-C',str(SOURCE),'status','--porcelain'],text=True)==source['status']
recovered=proof.verify();save('recovery-after.json',{'proofs':recovered,'original_targets_read':False})
available=free();receipt={'observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'purpose':'Deferred migration cache housekeeping to admit MILESTONESfn254.16; no migration acceptance work.','authorized_removed_target':LITERAL,'target_absent':True,'allocated_bytes_before':initial['target_allocated_bytes'],'allocated_bytes_after':0,'cargo_clean_tagged_targets':len(commands),'guarded_remainder_reason':'Orchestration root has no valid Cargo tag; real tagged children cleaned first, then literal-path/uid/type/live-ownership guarded inventoried remainder. No fabricated tag.','required_named_recovery_after':recovered,'all322current_required_payload_records_recoverable':True,'protected_source_index_receipts_recovery_hostrelease_cde_roots_unchanged':True,'available_private_bytes':available,'available_private_gib':available/1024**3,'required_floor_bytes':FLOOR,'meets50gib_now':available>=FLOOR,'df':subprocess.check_output(['df','-B1',str(TARGET.parent),str(pathlib.Path(os.environ.get('TMPDIR',str(TARGET.parent/'tmp')))),str(ROOT)],text=True),'unreadable_system_fields':owner['unreadable'],'no_other_cache_source_ref_flow_main_or_shared_mutation':True,'no_build_gate_review_or_new_full_target_archive':True,'admission_limit':'Current measured floor only; root must remeasure and admitfn254 workload reservation immediately before build.'}
save('cleanup-receipt.json',receipt);print(json.dumps({'target_absent':True,'released_bytes':initial['target_allocated_bytes'],'available_gib':available/1024**3,'meets50gib':available>=FLOOR,'proofs':recovered}))
