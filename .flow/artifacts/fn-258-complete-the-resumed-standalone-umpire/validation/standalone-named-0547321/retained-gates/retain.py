#!/usr/bin/env python3
"""Retain completed named default gates only; never opens the active release target/log."""
import collections
import gzip
import hashlib
import json
import os
import pathlib
import shutil
import tarfile
import tempfile

HERE=pathlib.Path(__file__).resolve().parent
NAMED=HERE.parent
TEMPORAL=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
PARENT=pathlib.Path('/home/agent/.cache/codex-build/fn258-standalone-0547321')
STORE=TEMPORAL.parent/'.recovery/fn258-standalone-proof/objects'
REV='05473210b48c3db862f22051eb7154ce98258f98'
TREE='46829511b6d582b40f3fa5095f42bdf980c7d466'
RUNS={'make-check':'gate-3896264-1793197198117320136','make-time':'gate-3917126-1793197470946652849'}
records=[]

def sha(path):
 h=hashlib.sha256()
 with path.open('rb') as stream:
  for data in iter(lambda:stream.read(1024*1024),b''):h.update(data)
 return h.hexdigest()

def read(path):return json.loads(path.read_bytes())
def write(name,data):(HERE/name).write_text(json.dumps(data,indent=2)+'\n')
def retain(path,name,role,readable=False):
 path=pathlib.Path(path)
 assert path.is_file() and not path.is_symlink(),str(path)
 digest=sha(path);dest=STORE/digest
 if dest.exists():assert sha(dest)==digest
 else:
  with tempfile.NamedTemporaryFile(dir=STORE,delete=False) as temp:
   temporary=pathlib.Path(temp.name)
   with path.open('rb') as source:shutil.copyfileobj(source,temp)
  assert sha(temporary)==digest;temporary.replace(dest)
 record={'source':str(path),'destination':name,'role':role,'sha256':digest,'size':path.stat().st_size,'mode':oct(path.stat().st_mode&0o777),'recovery_object':str(dest)};records.append(record)
 if readable:
  target=HERE/name;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(path,target);assert sha(target)==digest
 return record

def archive(run,name):
 files=[p for p in sorted(run.rglob('*')) if p.is_file()]
 with tempfile.NamedTemporaryFile(dir=STORE.parent,suffix='.tar.gz',delete=False) as temp:path=pathlib.Path(temp.name)
 members=[]
 with path.open('wb') as raw:
  with gzip.GzipFile(filename='',fileobj=raw,mode='wb',mtime=0) as compressed:
   with tarfile.open(fileobj=compressed,mode='w|') as output:
    for file in files:
     info=output.gettarinfo(str(file),arcname=str(file.relative_to(run)));assert info.isfile();info.mode&=0o7777;info.uid=info.gid=info.mtime=0;info.uname=info.gname=''
     with file.open('rb') as stream:output.addfile(info,stream)
     members.append({'path':info.name,'size':info.size,'mode':oct(info.mode),'sha256':sha(file)})
 record=retain(path,name+'/stage-authority.tar.gz','complete command-associated native stage authority, claims, observations and logs');path.unlink()
 with tarfile.open(record['recovery_object'],'r:gz') as saved:
  actual=saved.getmembers();assert [r.name for r in actual]==[r['path'] for r in members]
  for a,b in zip(actual,members):assert a.isfile() and a.size==b['size'] and oct(a.mode)==b['mode'] and hashlib.sha256(saved.extractfile(a).read()).hexdigest()==b['sha256']
 return {'object':record,'verified_members':members}

def subsequence(needle,haystack):
 remaining=iter(haystack)
 return all(any(line==candidate for candidate in remaining) for line in needle)

seals={name+suffix:sha(NAMED/(name+suffix)) for name in RUNS for suffix in ['.json','.log']}
commands={};archives={}
for name,run_id in RUNS.items():
 receipt=read(NAMED/(name+'.json'));log=(NAMED/(name+'.log')).read_bytes();run=PARENT/'gate-receipts'/run_id
 assert receipt['source']==REV and receipt['source_tree']==TREE and receipt['exit_code']==0 and receipt['source_status_after']==''
 assert hashlib.sha256(log).hexdigest()==receipt['log_sha256']
 assert receipt['argv']==['make','check' if name=='make-check' else 'time']
 rows=[json.loads(line) for line in (run/'stages.jsonl').read_text().splitlines()]
 assert len(rows)==18 and len({r['stage'] for r in rows})==18
 for row in rows:
  assert row['exit']==0 and row['count_error'] is None and row['count_authority']=='exclusive-stage'
  assert read(run/'framework'/(row['stage']+'.json'))==row
 assert all(row['state']=='success' for row in rows[:-1])
 assert rows[-1]['stage']=='standalone' and rows[-1]['state']=='skipped' and rows[-1]['reason']=='standalone runs only with TLC required'
 assert not any(r['child_count_receipts'] for r in rows)
 raw={k:(run/(k+'.log')).read_bytes() for k in ['stdout','stderr']}
 matching={k:subsequence(v.splitlines(),log.splitlines()) for k,v in raw.items()};assert all(matching.values()),matching
 events=[];begins=set();finishes={};tlc=collections.Counter()
 for event in sorted((run/'work/framework/tests').glob('*.event'),key=lambda p:int(p.stem)):
  fields=event.read_text().strip().split('\t',2);assert len(fields)==3 and fields[0]=='1'
  kind,payload=fields[1:];events.append({'path':event.name,'kind':kind,'payload':payload,'sha256':sha(event)})
  parts=payload.split('\t')
  if kind=='tests-begin':assert parts[0] not in begins;begins.add(parts[0])
  elif kind=='tests-finish':
   assert parts[0] not in finishes and len(parts)==5;finishes[parts[0]]=dict(zip(['discovered','executed','ignored','skipped'],map(int,parts[1:])))
  elif kind.startswith('tlc-'):tlc[kind]+=1
 assert begins==finishes.keys() and len(finishes)==2
 counts={k:sum(r[k] for r in finishes.values()) for k in ['discovered','executed','ignored','skipped']}
 tests=next(row for row in rows if row['stage']=='tests');assert all(tests['counts'][k]==v for k,v in counts.items())
 assert counts=={'discovered':915,'executed':912,'ignored':3,'skipped':0}
 for suffix in ['.json','.log']:retain(NAMED/(name+suffix),name+'/command'+suffix,'original completed command receipt/log',readable=True)
 for leaf in ['admission.json','stages.jsonl','stdout.log','stderr.log']:retain(run/leaf,name+'/'+leaf,'matching native stage authority/admission/raw channel',readable=True)
 for leaf in sorted((run/'framework').glob('*.json')):retain(leaf,name+'/stages/'+leaf.name,'individual native stage receipt',readable=True)
 write(name+'-observations.json',{'events':events,'test_producers':finishes,'aggregate':counts,'raw_tlc_event_totals':dict(tlc),'published_tlc_counts':{k:tests['counts'][k] for k in ['tlc_launches','tlc_configurations']},'note':'Raw TLC events may be explicit backend controls; default model-TLC coverage is skipped. Published TLC totals stay unknown/null.'})
 archives[name]=archive(run,name)
 commands[name]={'receipt':name+'/command.json','native_run':str(run),'run_id':run_id,'runner_seconds':receipt['seconds'],'stage_count':18,'successful_stages':17,'skipped_stages':['standalone'],'counts':counts,'test_producers':finishes,'backend_mode':receipt['backend_mode'],'target_existed_before':receipt['target_existed_before'],'native_channel_ordered_subsequence_matches':matching,'binding_method':'Each complete native stdout/stderr channel is an exact ordered subsequence of its matching sealed Make log; invocation order and measured durations corroborate the association. Original command JSON did not record a native gate ID, so this is independently matched association, not an original explicit run-ID seal.','source_cleanliness':'Runner source_status_after was empty at completion; source HEAD/tree in receipt match054. No active release source or output was inspected.'}
old=read(NAMED.parent/'standalone-0547321/recovery-manifest.json')
seen=set()
for row in old['records']:
 if not row['destination'].startswith('parent/named/'):continue
 path=pathlib.Path(row['source'])
 if not str(path).startswith(str(PARENT)+'/') or str(path) in seen:continue
 seen.add(str(path));record=retain(path,'post-time/'+str(path.relative_to(PARENT)),'current post-time named helper/wasm/native parity payload')
 record['prior_required_sha256']=row['sha256'];record['matches_prior_required']=record['sha256']==row['sha256']
for dirname in ['guest-cache','guest-wrappers']:
 for path in sorted((PARENT/dirname).rglob('*')):
  if path.is_file() and (path.name.endswith('.umpire-id') or path.name.startswith('umpire-lock-') and path.suffix=='.json' or dirname=='guest-wrappers'):
   retain(path,'post-time/identity/'+str(path.relative_to(PARENT)),'current post-time guest compile identity or compiler wrapper')
assert all(sha(NAMED/name)==digest for name,digest in seals.items())
for row in records:assert sha(pathlib.Path(row['recovery_object']))==row['sha256']
write('archive-manifests.json',archives)
write('recovery-manifest.json',{'source_commit':REV,'source_tree':TREE,'objects':records,'content_store':str(STORE),'original_completed_receipt_log_seals_unchanged':seals,'artifact_mapping_limit':'Named parent outputs were sealed after make-time. No original per-check artifact hash snapshot was supplied; do not infer an intermediate per-check artifact identity from the current files. Command stage successes, source stamps and exact log matches are separate retained observations.','active_release_read':False,'result':'pass'})
summary={'source_commit':REV,'source_tree':TREE,'commands':commands,'time_measurements':{'make_check_wall_seconds':272.948165505004,'make_time_wall_seconds':112.5601406779897,'make_time_gate_seconds':102.122,'make_time_inner_seconds':10.343,'legacy_gate_soft_seconds':60,'legacy_inner_soft_seconds':10,'gate_soft_target_missed':True,'inner_soft_target_missed':True,'inner_loop_test_count':'not separately observed in these receipts; do not reuse the full tests-stage count for the inner loop'},'temperature':'Existing owned parent cache reused. First named check rebuilt some crates; subsequent time is warmer. Neither is the independently isolated cold required proof.','required_cold_proof':'../standalone-0547321: distinct earlier054 make validate evidence, already retained and verified; no new required-TLC or cold-child claim here.','release':'Active separate HOST release target/log intentionally not read or modified; no release result included.','retained_named_records':len(records),'unique_objects':len({r['sha256'] for r in records}),'archive_members':sum(len(a['verified_members']) for a in archives.values()),'source_or_parenttarget_modified':False,'original_logs_and_json_bytes_preserved':True,'result':'pass'}
write('summary.json',summary)
print(json.dumps({'result':'pass','records':len(records),'commands':{k:{x:v[x] for x in ['successful_stages','skipped_stages','counts','native_channel_ordered_subsequence_matches']} for k,v in commands.items()}},indent=2))
