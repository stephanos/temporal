import ast,hashlib,json,os,shutil,stat,subprocess,tarfile
from pathlib import Path
ROOT=Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
RECOVERY=ROOT.parent/'.recovery/fn25417-disk-20261027T170723Z'
ARCHIVE=RECOVERY/'retained-proof-output.tar.gz'
manifest=json.loads((RECOVERY/'manifest.json').read_text())
TARGETS=[Path(p) for p in manifest['targets']]
source_before=manifest['source_seals']; entries=manifest['entries']
removed_bytes=manifest['cache_allocated_bytes'];free_before=manifest['free_before_bytes']
sha=lambda p:hashlib.file_digest(open(p,'rb'),'sha256').hexdigest()
tree=ast.parse((ROOT/'.flow/tmp/disk-cleanup-20261027/reclaim.py').read_text())
helpers=[n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name in ('audit_live','source_seals')]
exec(compile(ast.Module(body=helpers,type_ignores=[]),'cleanup-helpers','exec'))
audit_live();assert source_before==source_seals();assert all(t.is_dir() for t in TARGETS)
expected={row['archive']:row for row in entries};verified={}
with tarfile.open(ARCHIVE,'r|gz') as tar:
 for member in tar:
  row=expected[member.name]
  assert member.name not in verified and stat.S_IMODE(member.mode)==row['mode']
  if row['kind']=='symlink':assert member.issym() and member.linkname==row['link']
  elif member.islnk():
   assert member.linkname in verified and verified[member.linkname].get('sha256')==row['sha256']
  else:
   assert member.isfile()
   stream=tar.extractfile(member)
   assert stream is not None and hashlib.file_digest(stream,'sha256').hexdigest()==row['sha256']
  verified[member.name]=row
assert set(verified)==set(expected)
archive_sha=sha(ARCHIVE)
print(json.dumps({'phase':'verified','files':len(verified),'archive_bytes':ARCHIVE.stat().st_size,'archive_sha256':archive_sha}),flush=True)
assert source_before==source_seals();audit_live();results=[]
env=os.environ.copy();env.pop('BASH_ENV',None)
for t in TARGETS:
 audit_live()
 result=subprocess.run(['mise','exec','--','cargo','clean','--target-dir',str(t)],cwd=ROOT/'model/umpire',env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 row={'target':str(t),'cargo_exit':result.returncode,'cargo_output':result.stdout}
 if t.exists():
  audit_live();shutil.rmtree(t);row['exact_owned_cache_removal']=True
 assert not t.exists();results.append(row)
 print(json.dumps({'phase':'removed','target':str(t),'cargo_exit':result.returncode}),flush=True)
assert source_before==source_seals()
protected=[ROOT/'.git',ROOT/'.worktrees/fn25417-work-counts',Path('/home/agent/.cache/codex-build/cargo-target'),Path('/home/agent/.cache/codex-build/fn25417-work-counts')]
assert all(p.exists() for p in protected)
summary={'recovery':str(RECOVERY),'archive':str(ARCHIVE),'archive_sha256':archive_sha,'manifest_sha256':sha(RECOVERY/'manifest.json'),'retained_files_verified':len(entries),'cache_allocated_bytes_removed':removed_bytes,'archive_bytes':ARCHIVE.stat().st_size,'net_reclaimed_estimate_bytes':removed_bytes-ARCHIVE.stat().st_size,'free_before_bytes':free_before,'free_after_bytes':shutil.disk_usage(ROOT).free,'framework_sources_unchanged':True,'protected_shared_and_next_task_caches_present':True,'results':results}
(RECOVERY/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
(ROOT/'.flow/tmp/disk-cleanup-20261027/result.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps(summary),flush=True)
