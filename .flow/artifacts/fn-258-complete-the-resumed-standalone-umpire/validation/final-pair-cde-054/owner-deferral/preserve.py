#!/usr/bin/env python3
"""Owner deferral shutdown evidence only; no migration gate or acceptance check."""
import datetime,hashlib,importlib.util,json,os,pathlib,shutil,subprocess
HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
ART=ROOT/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire'
PAIR=pathlib.Path('/home/agent/.cache/codex-build/tmp/fn258-final-pair-cde-054')
TARGET=pathlib.Path('/home/agent/.cache/codex-build/fn258-final-pair-cde-054')
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
 return h.hexdigest()
def save(n,x):(HERE/n).write_text(json.dumps(x,indent=2)+'\n')
# Reuse only read-only ownership helper; module is explicitly guarded.
spec=importlib.util.spec_from_file_location('audit',ART/'disk/failed-final-pair-3f5-target/retain-audit.py');audit=importlib.util.module_from_spec(spec);spec.loader.exec_module(audit)
live={}
for name,path in [('pair',PAIR),('target',TARGET)]:
 audit.TARGET=path;live[name]=audit.owners()
 assert not live[name]['matches'],live[name]
save('ownership.json',live)
records=[]
def copy(source,destination):
 assert source.is_file() and not source.is_symlink()
 dest=HERE/destination;dest.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,dest)
 digest=sha(source);assert sha(dest)==digest
 records.append({'source':str(source),'destination':destination,'sha256':digest,'size':source.stat().st_size,'mode':oct(source.stat().st_mode&0o7777)})
execution=next((PAIR/'execution-evidence').glob('paired-default-check-*'))
receipt=json.loads((execution/'00-paired-default-check.json').read_text());assert receipt['exit']==2 and sha(execution/'00-paired-default-check.log')==receipt['log_sha256']
for f in execution.rglob('*'):
 if f.is_file():copy(f,'execution/'+str(f.relative_to(execution)))
for f in (TARGET/'gate-receipts').rglob('*'):
 if f.is_file():copy(f,'partial-native-authority/'+str(f.relative_to(TARGET/'gate-receipts')))
for name in ['execution-state.json','launch.py','launch-plan.json']:
 copy(PAIR/name,'context/'+name)
lockfiles=[]
for f in [PAIR/'execution.lock']+list((TARGET/'gate-receipts').rglob('*lock*')):
 if f.is_file():lockfiles.append({'path':str(f),'size':f.stat().st_size,'sha256':sha(f),'device':f.stat().st_dev,'inode':f.stat().st_ino})
keys={f"{os.major(r['device']):02x}:{os.minor(r['device']):02x}:{r['inode']}":r['path'] for r in lockfiles};held=[]
for line in pathlib.Path('/proc/locks').read_text().splitlines():
 for token in line.split():
  if token in keys:held.append({'path':keys[token],'record':line})
assert not held
save('lease-state.json',{'observed_lock_files':lockfiles,'held_kernel_locks':held,'interpretation':'No readable process owns cwd/env/fd/maps under pair/target; no kernel execution-lock owner. Root systemPID unreadability is preserved in ownership.json.'})
save('manifest.json',{'classification':'OWNER_INTERRUPTED_MIGRATION_DEFERRED','records':records,'scope':'Raw terminal command and partial authority retained without final gate success/failure inference.'})
save('summary.json',{'observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'classification':'OWNER_INTERRUPTED_MIGRATION_DEFERRED','cause_authority':'User deferred migration and replaced goal with MILESTONES; root reports SIGINT sent to controller4052515, unifiedexec22416 terminal2.','source_pair':json.loads((PAIR/'execution-state.json').read_text())['sources'],'command':receipt['argv'],'exit':receipt['exit'],'seconds':receipt['ended_unix']-receipt['started_unix'],'functional_verdict':None,'retained_files':len(records),'terminal_log_sha256':receipt['log_sha256'],'native_final_stage_file_present':bool(list((TARGET/'gate-receipts').glob('*/stages.jsonl'))),'no_migration_gate_review_acceptance_check':True,'no_owned_readable_producer_or_held_execution_lease':True,'protected_original_sources_targets_and_caches_untouched':True,'remaining_unknowns':'Interrupted native run has no final stages.jsonl; partial stage files are observations, not complete stage accounting. Unreadable root systemPID fields are explicit.'})
print(json.dumps({'classification':'OWNER_INTERRUPTED','retained_files':len(records),'live_matches':{k:len(v['matches']) for k,v in live.items()},'held_locks':len(held)}))
