#!/usr/bin/env python3
"""Inventory exactly the released standalone child target; never performs cleanup."""
import datetime
import hashlib
import json
import os
import pathlib
import stat
import subprocess

HERE=pathlib.Path(__file__).resolve().parent
TEMPORAL=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
CHILD=pathlib.Path('/home/agent/.cache/codex-build/tmp/umpire-standalone-Bnt8GY')
TARGET=CHILD/'target'
PARENT=pathlib.Path('/home/agent/.cache/codex-build/fn258-standalone-0547321')
PROOF=TEMPORAL/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/standalone-0547321'

def digest(path):
 h=hashlib.sha256()
 with path.open('rb') as stream:
  for chunk in iter(lambda:stream.read(1024*1024),b''):h.update(chunk)
 return h.hexdigest()

def save(name,data):(HERE/name).write_text(json.dumps(data,indent=2)+'\n')

def scan(root,exclude=None,hash_content=True):
 result=[]
 for directory,dirs,files in os.walk(root,followlinks=False):
  directory=pathlib.Path(directory)
  if exclude:
   dirs[:]=[name for name in dirs if directory/name != exclude]
  for name in sorted(dirs+files):
   p=directory/name
   if exclude and p==exclude:continue
   st=p.lstat();row={'path':str(p.relative_to(root)),'mode':oct(stat.S_IMODE(st.st_mode)),'size':st.st_size,'inode':st.st_ino,'device':st.st_dev,'mtime_ns':st.st_mtime_ns}
   if p.is_symlink():row.update(kind='symlink',link=os.readlink(p))
   elif p.is_dir():row.update(kind='directory',mountpoint=os.path.ismount(p))
   elif p.is_file():
    row['kind']='file'
    if hash_content:row['sha256']=digest(p)
   else:row['kind']='special'
   result.append(row)
 return sorted(result,key=lambda x:x['path'])

def owners():
 hits=[];unknown=[]
 for p in pathlib.Path('/proc').iterdir():
  if not p.name.isdigit():continue
  try:cwd=os.readlink(p/'cwd');args=(p/'cmdline').read_bytes().split(b'\0');env=(p/'environ').read_bytes().split(b'\0')
  except PermissionError:unknown.append(int(p.name));continue
  except (FileNotFoundError,ProcessLookupError):continue
  within=lambda text: text==str(TARGET) or text.startswith(str(TARGET)+'/')
  r={'pid':int(p.name),'cwd':cwd if within(cwd) else None,'arguments':[],'environment':{},'fds':[]}
  for arg in args:
   value=arg.decode(errors='replace')
   if value.startswith('/') and within(value):r['arguments'].append(value)
  for raw in env:
   if b'=' not in raw:continue
   k,v=raw.split(b'=',1);k=k.decode(errors='replace');v=v.decode(errors='replace')
   if k in {'CARGO_TARGET_DIR','TMPDIR','GOCACHE','GOMODCACHE','PWD','UMPIRE_GATE_RECEIPTS','UMPIRE_STANDALONE_OUTPUT'} and within(v):r['environment'][k]=v
  try:
   for fd in (p/'fd').iterdir():
    try:
     name=os.readlink(fd)
     if within(name):r['fds'].append({'fd':fd.name,'path':name,'lease_locks':[line for line in (p/'fdinfo'/fd.name).read_text().splitlines() if line.startswith('lock:')]})
    except (PermissionError,FileNotFoundError,ProcessLookupError):continue
  except (PermissionError,FileNotFoundError,ProcessLookupError):pass
  if r['cwd'] or r['arguments'] or r['environment'] or r['fds']:hits.append(r)
 return {'matches':hits,'unreadable_system_pids':unknown}

def df(path):return subprocess.check_output(['df','-B1',str(path)]).decode()

def main():
 assert TARGET.parent==CHILD and TARGET.name=='target' and not TARGET.is_symlink() and TARGET.resolve()==TARGET
 assert json.loads((CHILD/'inputs.json').read_text())['target']==str(TARGET)
 assert json.loads((CHILD/'result.json').read_text())['complete'] is True
 assert subprocess.check_output(['git','-C',str(TEMPORAL),'rev-parse','HEAD']).decode().strip()=='c258547ed5cc7d2ddc2247fce956dc5bb8504d32'
 expected_top={'doc','guest-wrappers','debug','gate-receipts','panic-native','current-contract-native','go-build','.rustc_info.json','wasm32-wasip1','xtask-bootstrap','tmp','guest','tests','.rustdoc_fingerprint.json','guest-cache'}
 assert {p.name for p in TARGET.iterdir()}==expected_top
 inventory=scan(TARGET)
 assert not any(r['kind']=='special' or r.get('mountpoint') for r in inventory)
 save('target-inventory-before.json',{'target':str(TARGET),'records':inventory,'regular_payload_bytes':sum(r['size'] for r in inventory if r['kind']=='file'),'record_count':len(inventory)})
 protected={'child_outside_target':scan(CHILD,exclude=TARGET),'parent_target_metadata':scan(PARENT,hash_content=False),'required_retained_source_files':[]}
 manifest=json.loads((PROOF/'recovery-manifest.json').read_text())
 for row in manifest['records']:
  p=pathlib.Path(row['source'])
  if str(p).startswith(str(TARGET)+'/'):continue
  if p.is_file():
   actual=digest(p);assert actual==row['sha256'],str(p)
   protected['required_retained_source_files'].append({'path':str(p),'sha256':actual})
 save('protected-before.json',protected)
 live=owners();assert not live['matches'],live
 valid_tags=[]
 for row in inventory:
  if row['kind']=='file' and pathlib.PurePosixPath(row['path']).name=='CACHEDIR.TAG':
   path=TARGET/row['path']
   if path.read_bytes().startswith(b'Signature: 8a477f597d28d172789f06886806bc55'):
    valid_tags.append(str(path.parent))
 valid_tags=[p for p in valid_tags if not any(p.startswith(q+'/') for q in valid_tags if p!=q)]
 receipt={'observed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'authorized_exact_target':str(TARGET),'task2_closure_commit':'c258547ed5cc7d2ddc2247fce956dc5bb8504d32','target_identity':{'device':TARGET.stat().st_dev,'inode':TARGET.stat().st_ino,'resolved':str(TARGET.resolve())},'allocated_bytes_before':int(subprocess.check_output(['du','-s','-B1',str(TARGET)]).split()[0]),'df_target_before':df(TARGET),'df_host_before':df(TEMPORAL),'ownership':live,'real_tagged_child_targets':sorted(valid_tags),'target_inventory_sha256':digest(HERE/'target-inventory-before.json'),'protected_before_sha256':digest(HERE/'protected-before.json'),'recovery_verification':json.loads((PROOF/'recovery-verification.json').read_text()),'status':'preflight passed; no cleanup performed by this script'}
 save('preflight.json',receipt)
 print(json.dumps({'records':len(inventory),'allocated_bytes':receipt['allocated_bytes_before'],'owners':live,'tagged_children':len(valid_tags)},indent=2))

if __name__ == "__main__":
 main()
