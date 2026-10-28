#!/usr/bin/env python3
"""Retain exactly the terminal old pair target proof and audit eligibility; no cleanup."""
import collections,datetime,gzip,hashlib,json,os,pathlib,shutil,stat,subprocess,tarfile,tempfile
HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
ART=ROOT/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire'
PROOF=ART/'validation/final-pair-3f5-054'
TARGET=pathlib.Path('/home/agent/.cache/codex-build/fn258-final-pair-3f5-054')
PAIR=pathlib.Path('/home/agent/.cache/codex-build/tmp/fn258-final-pair-3f5-054')
CAS=ROOT.parent/'.recovery/fn258-standalone-proof/objects'
def save(n,x):(HERE/n).write_text(json.dumps(x,indent=2)+'\n')
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
 return h.hexdigest()
def within(s):return s==str(TARGET) or s.startswith(str(TARGET)+'/')
def inventory():
 rows=[]
 for parent,dirs,files in os.walk(TARGET,followlinks=False):
  for name in sorted(dirs+files):
   p=pathlib.Path(parent)/name;s=p.lstat();kind='symlink' if stat.S_ISLNK(s.st_mode) else 'directory' if stat.S_ISDIR(s.st_mode) else 'file' if stat.S_ISREG(s.st_mode) else 'special'
   r={'path':str(p.relative_to(TARGET)),'kind':kind,'size':s.st_size,'allocated_bytes':s.st_blocks*512,'mode':oct(stat.S_IMODE(s.st_mode)),'uid':s.st_uid,'gid':s.st_gid,'device':s.st_dev,'inode':s.st_ino,'nlink':s.st_nlink,'mtime_ns':s.st_mtime_ns}
   if kind=='symlink':r['link']=os.readlink(p)
   if kind=='directory':r['mountpoint']=os.path.ismount(p)
   rows.append(r)
 return sorted(rows,key=lambda r:r['path'])
def owners():
 matches=[];unknown=[];races=[]
 for p in pathlib.Path('/proc').iterdir():
  if not p.name.isdigit():continue
  r={'pid':int(p.name),'cwd':None,'argv':[],'environment':{},'fds':[],'mapped_files':[]}
  for role in ['cwd','cmdline','environ','fd','maps']:
   try:
    if role=='cwd':
     v=os.readlink(p/role)
     if within(v):r['cwd']=v
    elif role=='cmdline':
     r['argv']=[v.decode(errors='replace') for v in (p/role).read_bytes().split(b'\0') if v and (within(v.decode(errors='replace')) or v.startswith(('CARGO_TARGET_DIR='+str(TARGET)).encode()))]
    elif role=='environ':
     for v in (p/role).read_bytes().split(b'\0'):
      if b'=' not in v:continue
      k,value=v.split(b'=',1);value=value.decode(errors='replace')
      if within(value):r['environment'][k.decode(errors='replace')]=value
    elif role=='maps':
     r['mapped_files']=[line.split(maxsplit=5)[-1] for line in (p/role).read_text().splitlines() if len(line.split(maxsplit=5))==6 and within(line.split(maxsplit=5)[-1])]
    else:
     for fd in (p/'fd').iterdir():
      try:
       v=os.readlink(fd)
       if within(v):r['fds'].append({'fd':fd.name,'path':v,'locks':[l for l in (p/'fdinfo'/fd.name).read_text().splitlines() if l.startswith('lock:')]})
      except (FileNotFoundError,ProcessLookupError):pass
      except PermissionError:unknown.append({'pid':int(p.name),'role':'fd/'+fd.name})
   except (FileNotFoundError,ProcessLookupError):races.append({'pid':int(p.name),'role':role})
   except PermissionError:unknown.append({'pid':int(p.name),'role':role})
  if r['cwd'] or r['argv'] or r['environment'] or r['fds'] or r['mapped_files']:matches.append(r)
 return {'matches':matches,'unreadable':unknown,'exited_or_raced':races}
def main():
 assert not TARGET.is_symlink() and TARGET.resolve()==TARGET
 plan=json.loads((PROOF/'launch-plan.json').read_text());assert plan['target']==str(TARGET) and plan['pair']==str(PAIR)
 summary=json.loads((PROOF/'current-guest-identity/summary.json').read_text());assert len(summary['commands'])==5 and all(r['exit']==0 for r in summary['commands'])
 for r in summary['commands']:
  log=PROOF/'current-guest-identity'/pathlib.Path(r['log']).name;assert sha(log)==r['log_sha256']
 assert summary['after']['temporal']['head']==plan['expected']['temporal'] and summary['after']['umpire']['head']==plan['expected']['umpire']
 assert not summary['after']['temporal']['status'] and not summary['after']['umpire']['status']
 identity={}
 for repo in ['temporal','umpire']:
  p=PAIR/repo;identity[repo]={'head':subprocess.check_output(['git','-C',str(p),'rev-parse','HEAD'],text=True).strip(),'status':subprocess.check_output(['git','-C',str(p),'status','--porcelain'],text=True)}
  assert identity[repo]['head']==plan['expected'][repo] and not identity[repo]['status']
 before=inventory();save('target-inventory.json',{'target':str(TARGET),'records':before,'hash_scope':'metadata complete; proof-selected files hashed in archive-manifest. No hash claim for discarded reproducible intermediate bytes.'})
 first_owners=owners();save('ownership-before.json',first_owners)
 selected=set()
 for row in before:
  if row['kind']!='file':continue
  p=TARGET/row['path'];rel=pathlib.PurePosixPath(row['path'])
  if rel.parts[0] in {'proof','gate-receipts'} or p.suffix in {'.wasm','.umpire-id'} or p.name in {'.rustc_info.json','.rustdoc_fingerprint.json'} or (p.suffix=='.d' and any('guest' in part for part in rel.parts)) or (p.suffix=='.json' and any('guest' in part for part in rel.parts)):
   selected.add(p)
  if row['mode'] and int(row['mode'],8)&0o111 and (str(rel.parent) in {'debug/deps','debug','xtask-bootstrap/debug','guest','panic-native','current-contract-native'} or rel.parts[0]=='guest-wrappers') and not p.suffix:
   with p.open('rb') as f:
    if f.read(4)==b'\x7fELF':selected.add(p)
 # Retain all already preserved failure output and positive guest/generation receipt bytes.
 external=[]
 for sub in ['current-guest-identity','failed-default-3f5','resolution','pair-prerequisites-and-generators','framework-generation','temporal-generation']:
  external.extend(p for p in (PROOF/sub).rglob('*') if p.is_file() and not p.is_symlink())
 external += [PROOF/n for n in ['preparation.json','launch-plan.json','launch.py','preparation.log','light-generation-admission.json']]
 guest=TARGET/'proof/umpire_temporal_guest.wasm';canonical=PAIR/'temporal/model/temporal/hosts/go/umpire_temporal_guest.wasm'
 assert sha(guest)==sha(canonical)==plan['guest_sha256']
 external.append(canonical)
 # Verify pre-existing raw guest CAS when available; retain raw guest independently regardless.
 records=[]
 def retain(p,dest,role):
  h=sha(p);q=CAS/h
  existed=q.exists()
  if existed:assert sha(q)==h
  else:
   with tempfile.NamedTemporaryFile(dir=CAS,delete=False) as f:
    tmp=pathlib.Path(f.name)
    with p.open('rb') as source:shutil.copyfileobj(source,f)
   assert sha(tmp)==h;tmp.replace(q)
  r={'source':str(p),'destination':dest,'role':role,'sha256':h,'size':p.stat().st_size,'mode':oct(stat.S_IMODE(p.stat().st_mode)),'recovery_object':str(q),'object_preexisted':existed};records.append(r);return r
 rawguest=retain(guest,'proof/umpire_temporal_guest.wasm','fresh3f5Makefile guest equals committed canonical and terminal cmp/freshness commands')
 members=[]
 with tempfile.NamedTemporaryFile(dir=CAS.parent,suffix='.tar.gz',delete=False) as f:archive=pathlib.Path(f.name)
 with archive.open('wb') as raw:
  with gzip.GzipFile(filename='',fileobj=raw,mode='wb',mtime=0,compresslevel=1) as compressed:
   with tarfile.open(fileobj=compressed,mode='w|',dereference=True) as out:
    for p in sorted(selected)+sorted(set(external)):
     name='target/'+str(p.relative_to(TARGET)) if TARGET in p.parents else 'evidence/'+str(p.relative_to(PROOF)) if PROOF in p.parents else 'source-canonical/umpire_temporal_guest.wasm'
     info=out.gettarinfo(str(p),arcname=name);assert info.isfile();info.mode&=0o7777;info.uid=info.gid=info.mtime=0;info.uname=info.gname=''
     with p.open('rb') as data:out.addfile(info,data)
     members.append({'path':name,'sha256':sha(p),'size':info.size,'mode':oct(info.mode)})
 obj=retain(archive,'paired-proof.tar.gz','selected positiveguest/native producer inputs and complete failed-stage authority/evidence');archive.unlink()
 save('archive-manifest.json',{'object':obj,'members':members})
 save('recovery-manifest.json',{'objects':records,'source_pair':plan['expected'],'store':str(CAS),'scope':'Old pair terminal positive guest and failed broad gate only, not task4 success.'})
 after=inventory();assert before==after,'target changed during retention'
 last_owners=owners();save('ownership-after.json',last_owners)
 # Kernel locks bound to exact inventoried target inodes, including any Cargo/guest lock.
 key={f"{os.major(r['device']):02x}:{os.minor(r['device']):02x}:{r['inode']}":r['path'] for r in before}
 locks=[]
 for line in pathlib.Path('/proc/locks').read_text().splitlines():
  for token in line.split():
   if token in key:locks.append({'kernel_record':line,'target_path':key[token]})
 lockfiles=[dict(path=r['path'],size=r['size'],sha256=sha(TARGET/r['path'])) for r in before if r['kind']=='file' and ('lock' in pathlib.PurePosixPath(r['path']).name or 'lease' in pathlib.PurePosixPath(r['path']).name)]
 save('leases.json',{'kernel_locks_matching_target':locks,'lock_or_lease_files':lockfiles,'interpretation':'File presence alone is not held ownership; checked process fdinfo/maps/cwd/argv/environment and kernel locks. Recheck live ownership immediately before any later cleanup.'})
 unique={}
 for r in before:unique.setdefault((r['device'],r['inode']),r)
 allocated=int(subprocess.check_output(['du','-s','-B1',str(TARGET)]).split()[0])
 unknown=[r for r in before if r['kind']=='special' or r.get('mountpoint')]
 uidset=sorted({r['uid'] for r in before});matches=first_owners['matches']+last_owners['matches']
 save('summary.json',{'observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'target':str(TARGET),'identity':{'resolved':str(TARGET.resolve()),'device':TARGET.stat().st_dev,'inode':TARGET.stat().st_ino,'uid':TARGET.stat().st_uid},'allocated_bytes':allocated,'logical_regular_bytes':sum(r['size'] for r in unique.values() if r['kind']=='file'),'inventory_records':len(before),'uids':uidset,'df_target':subprocess.check_output(['df','-B1',str(TARGET)],text=True),'df_host':subprocess.check_output(['df','-B1',str(ROOT)],text=True),'source_identity':identity,'guest_sha256':rawguest['sha256'],'raw_guest_cas_preexisted':rawguest['object_preexisted'],'guest_commands_all5_exit0':True,'preserved_archive_members':len(members),'archive_compressed_bytes':obj['size'],'target_metadata_unchanged_during_retention':True,'live_target_references':matches,'unreadable_system_pid_fields':last_owners['unreadable'],'held_target_locks':locks,'unknown_special_or_mount_entries':unknown,'eligibility':'Conditional future exact-target cleanup only after independent recovery verification and root admission; no cleanup authorization exercised here. Preserve source clones, launchers, artifact evidence and every shared/other cache. Recheck process/lease state and metadata immediately beforehand. Unreadable systemPID fields remain explicit uncertainty.','eligible_on_observed_readable_processes':not matches and not locks and not unknown and uidset==[os.getuid()],'coverage_limit':'Broaddefault failed at Temporal timing-controls; completed framework tests915/912/3 and wasm build are partial proof. Go-host/Kettle-host and laterTemporal tests/guest/host stages not-started, not acceptance.','no_cleanup_build_source_ref_flow_or_shared_cache_mutation':True})
 print(json.dumps({'allocated_bytes':allocated,'archive_bytes':obj['size'],'members':len(members),'live_matches':len(matches),'unreadable_fields':len(last_owners['unreadable']),'unknown_entries':len(unknown)}))

if __name__=="__main__":main()
