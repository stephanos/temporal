#!/usr/bin/env python3
"""Read-only bounded archive/local Scala-cache capture and later identity comparison."""
import argparse
import collections
import copy
import datetime
import hashlib
import json
import os
import pathlib
import stat
import subprocess

HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
ARCHIVE='model/archived'
FROZEN_TREE='cb5223b9a4839233876c98cf6158a46dc9d97a19'
CACHE_NAMES={'.scala-build','.bsp','.bloop','.metals','build','gen','target'}
# Explicit legacy locations from archive ignore rules, Scala directives and historical README.
EXPLICIT=[f'{base}/{name}' if base else name for base in ['', 'model', ARCHIVE]
          for name in ['.scala-build','.bsp','.bloop','.metals']]+[
    'model/archived/build','model/archived/gen','model/gen',
    'model/build/api-scalapb.jar','model/build/ir-scalapb.jar','model/build/reify.jar',
    'model/build/sample-check','model/build/history']


def git(*args,input=None,check=True):
 return subprocess.run(['git','-C',str(ROOT),*args],input=input,stdout=subprocess.PIPE,
                       stderr=subprocess.PIPE,check=check)

def sha(data):return hashlib.sha256(data).hexdigest()
def load(path):return json.loads(path.read_bytes())
def save(path,data):path.write_text(json.dumps(data,indent=2)+'\n')

def record(path):
 before=path.lstat();kind=('symlink' if stat.S_ISLNK(before.st_mode) else 'directory' if stat.S_ISDIR(before.st_mode) else 'file' if stat.S_ISREG(before.st_mode) else 'special')
 result={'path':str(path.relative_to(ROOT)),'kind':kind,'mode':oct(stat.S_IMODE(before.st_mode)),
         'size':before.st_size,'mtime_ns':before.st_mtime_ns}
 if kind=='file':
  digest=hashlib.sha256()
  with path.open('rb') as stream:
   for chunk in iter(lambda:stream.read(1024*1024),b''):digest.update(chunk)
  result['sha256']=digest.hexdigest()
 elif kind=='symlink':
  target=os.readlink(os.fsencode(path));result['link_target_hex']=target.hex();result['sha256']=sha(target)
 after=path.lstat();result['stable_read']=(before.st_ino,before.st_dev,before.st_mode,before.st_size,before.st_mtime_ns)==(after.st_ino,after.st_dev,after.st_mode,after.st_size,after.st_mtime_ns)
 return result

def tracked():
 result={}
 for row in git('ls-tree','-rz','HEAD','--',ARCHIVE).stdout.split(b'\0'):
  if row:
   meta,path=row.split(b'\t',1);mode,kind,blob=meta.decode().split()
   result[os.fsdecode(path)]={'git_mode':mode,'git_blob':blob}
 blobs=sorted({r['git_blob'] for r in result.values()})
 raw=git('cat-file','--batch',input=('\n'.join(blobs)+'\n').encode()).stdout
 position=0;hashes={}
 for blob in blobs:
  end=raw.index(b'\n',position);oid,kind,size=raw[position:end].decode().split();size=int(size)
  assert oid==blob and kind=='blob';data=raw[end+1:end+1+size]
  assert raw[end+1+size:end+2+size]==b'\n';position=end+2+size
  hashes[blob]={'sha256':sha(data),'size':size}
 for r in result.values():r.update(hashes[r['git_blob']])
 return result

def capture():
 head_before=git('rev-parse','HEAD').stdout.decode().strip()
 tree_result=git('rev-parse','HEAD:'+ARCHIVE,check=False)
 tree=tree_result.stdout.decode().strip() if tree_result.returncode==0 else None
 archive=ROOT/ARCHIVE
 paths={archive} if os.path.lexists(archive) else set();discovered=[]
 if archive.is_dir() and not archive.is_symlink():
  for directory,dirs,files in os.walk(archive,followlinks=False):
   directory=pathlib.Path(directory)
   for name in dirs+files:paths.add(directory/name)
   for name in dirs:
    if name in CACHE_NAMES:discovered.append(str((directory/name).relative_to(ROOT)))
 selection=sorted(set(EXPLICIT+discovered));roots=[]
 for relative in selection:
  path=ROOT/relative;exists=os.path.lexists(path)
  roots.append({'path':relative,'exists':exists,'selection':'discovered inside complete archive' if relative in discovered else 'explicit bounded legacy/local cache location','symlink':path.is_symlink()})
  if not exists:continue
  paths.add(path)
  if path.is_dir() and not path.is_symlink():
   for directory,dirs,files in os.walk(path,followlinks=False):
    for name in dirs+files:paths.add(pathlib.Path(directory)/name)
 records={str(path.relative_to(ROOT)):record(path) for path in sorted(paths)}
 git_records=tracked();untracked=[os.fsencode(p) for p,r in records.items() if r['kind']!='directory' and p not in git_records]
 ignored=git('check-ignore','--stdin','-z',input=b'\0'.join(untracked)+b'\0',check=False) if untracked else None
 assert ignored is None or ignored.returncode in [0,1],ignored.stderr.decode(errors='replace')
 ignored_paths={os.fsdecode(p) for p in ignored.stdout.split(b'\0') if p} if ignored else set()
 for path,row in records.items():
  row['git_class']='tracked-archive' if path in git_records else 'ignored-on-disk' if path in ignored_paths else 'untracked-or-directory'
  row['cache_related']=(not path.startswith(ARCHIVE+'/') or path.endswith('.semanticdb') or any(path==r or path.startswith(r+'/') for r in selection)) and path!=ARCHIVE
 differences=[]
 for path,expected in git_records.items():
  actual=records.get(path)
  if actual is None:differences.append({'path':path,'reason':'missing tracked archive file'});continue
  mode='120000' if actual['kind']=='symlink' else '100755' if int(actual['mode'],8)&0o111 else '100644'
  if actual.get('sha256')!=expected['sha256'] or mode!=expected['git_mode'] or actual['size']!=expected['size']:
   differences.append({'path':path,'expected':expected,'actual':actual})
 unstable=[p for p,r in records.items() if not r['stable_read']]
 head_after=git('rev-parse','HEAD').stdout.decode().strip()
 return {'schema_version':1,'observed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),
         'phase':'pre-cutover baseline; source cutover not performed','post_cutover_claim':False,
         'repository':str(ROOT),'head_before':head_before,'head_after':head_after,
         'frozen_git_archive_tree':FROZEN_TREE,'actual_git_archive_tree':tree,'git_archive_tree_error':tree_result.stderr.decode(errors='replace').strip() if tree_result.returncode else None,'git_archive_tree_matches_frozen':tree==FROZEN_TREE,
         'tracked_archive_files':len(git_records),'tracked_archive_disk_differences':differences,
         'selected_cache_roots':roots,'explicit_absent_paths':[r['path'] for r in roots if not r['exists']],
         'records':records,'unstable_reads':unstable,'consistent_capture':not unstable,
         'summary':{'records':len(records),'regular_files':sum(r['kind']=='file' for r in records.values()),
                    'regular_bytes':sum(r['size'] for r in records.values() if r['kind']=='file'),
                    'ignored_on_disk_files':sum(r['kind']!='directory' and r['git_class']=='ignored-on-disk' for r in records.values()),
                    'cache_related_files':sum(r['kind']!='directory' and r['cache_related'] for r in records.values()),
                    'symlinks':sum(r['kind']=='symlink' for r in records.values())},
         'classification_note':'Ignored paths are not assumed disposable runtime caches: source-like historical sample *.test files are included and protected alongside recognized Scala cache metadata.',
         'provenance':['model/archived/.gitignore: build/gen/.scala-build/.bsp and *.semanticdb ownership',
                       'model/archived/project.scala:8: former build/api-scalapb.jar input',
                       'model/archived/README.md:277,416,665,690,732,911: former model/build jars, sample-check, history and reify outputs',
                       '.gitignore:59-60: repository-root .bsp and .scala-build',
                       '.bsp/scala-cli.json and .scala-build/ide-inputs.json: root IDE-cache references; no referenced executable/global cache followed',
                       '.flow/artifacts/fn-257-extract-the-standalone-umpire/task2/reproduction-and-preservation.json: ignored cache bytes were explicitly not inventoried'],
         'boundaries':{'complete_archive':'All on-disk entries beneath model/archived, including hidden/ignored/untracked files, exact POSIX modes, symlink target bytes; symlinks never followed.',
                       'other_local_caches':'Only enumerated roots; repository/model root Scala IDE cache paths and explicitly named legacy build artifacts. No all-repo cache discovery.',
                       'excluded':'No global/shared/home toolchain/Coursier caches, worktree caches, current Rust/TLC/Apalache model/build subtrees or active release target were scanned.',
                       'outside_reference':'Root .bsp metadata refers to the installed Scala CLI outside the workspace and A.scala source; these references are not cache content coverage or authorization to inspect external locations.',
                       'historical_limit':'This is the first current ignored-cache-byte baseline for this migration. Earlier source preservation did not prove ignored cache identity; no historical before/after claim is made.',
                       'metadata_limit':'Full POSIX mode bits and raw symlink targets are retained. mtime is observed separately; atime/ctime, ACLs and xattrs are not identity guarantees.'}}

def identity(row):
 keys=['kind','mode']+(['sha256','size'] if row['kind']=='file' else ['link_target_hex','size'] if row['kind']=='symlink' else [])
 return {k:row.get(k) for k in keys}

def compare(before,after,phase):
 old,new=before['records'],after['records']
 added=sorted(new.keys()-old.keys());missing=sorted(old.keys()-new.keys())
 changed=[{'path':p,'before':identity(old[p]),'after':identity(new[p])} for p in sorted(old.keys()&new.keys()) if identity(old[p])!=identity(new[p])]
 roots_before={r['path']:r['exists'] for r in before['selected_cache_roots']};roots_after={r['path']:r['exists'] for r in after['selected_cache_roots']}
 root_changes=[{'path':p,'before':roots_before.get(p),'after':roots_after.get(p)} for p in sorted(roots_before.keys()|roots_after.keys()) if roots_before.get(p)!=roots_after.get(p)]
 passed=(before['consistent_capture'] and not added and not missing and not changed and not root_changes and before['actual_git_archive_tree']==after['actual_git_archive_tree'] and after['consistent_capture'])
 return {'phase':phase,'post_cutover_claim':phase=='post-cutover','result':'pass' if passed else 'fail',
         'before_observed_at':before['observed_at'],'after_observed_at':after['observed_at'],
         'git_archive_tree_before':before['actual_git_archive_tree'],'git_archive_tree_after':after['actual_git_archive_tree'],
         'added_paths':added,'missing_paths':missing,'byte_mode_or_symlink_changes':changed,'cache_root_presence_changes':root_changes,
         'mtime_only_changes':[p for p in sorted(old.keys()&new.keys()) if old[p]['mtime_ns']!=new[p]['mtime_ns'] and identity(old[p])==identity(new[p])],
         'unstable_reads':after['unstable_reads'],'scope':'Only bounded selected roots; no historical ignored-cache or unobserved external-cache identity claim.'}

def controls():
 sample={'actual_git_archive_tree':FROZEN_TREE,'observed_at':'synthetic','consistent_capture':True,'unstable_reads':[],
         'selected_cache_roots':[{'path':'model/.scala-build','exists':False}],
         'records':{'model/archived/source.scala':{'kind':'file','mode':'0o644','size':1,'sha256':'a','mtime_ns':0},
                    'model/archived/link':{'kind':'symlink','mode':'0o777','size':1,'link_target_hex':'61','mtime_ns':0}}}
 results=[]
 for name in ['file-bytes','file-mode','symlink-target','missing-file','cache-appears','archive-removed']:
  changed=copy.deepcopy(sample)
  if name=='file-bytes':changed['records']['model/archived/source.scala']['sha256']='b'
  if name=='file-mode':changed['records']['model/archived/source.scala']['mode']='0o755'
  if name=='symlink-target':changed['records']['model/archived/link']['link_target_hex']='62'
  if name=='missing-file':del changed['records']['model/archived/source.scala']
  if name=='cache-appears':changed['selected_cache_roots'][0]['exists']=True
  if name=='archive-removed':changed['records']={};changed['actual_git_archive_tree']=None
  results.append({'name':name,'refused':compare(sample,changed,'in-memory-control')['result']=='fail'})
 return {'result':'pass' if all(r['refused'] for r in results) else 'fail','controls':results,'source_mutations':False}

if __name__=='__main__':
 parser=argparse.ArgumentParser(description=__doc__);sub=parser.add_subparsers(dest='command',required=True)
 cap=sub.add_parser('capture');cap.add_argument('--output',type=pathlib.Path,required=True)
 cmp=sub.add_parser('compare');cmp.add_argument('--baseline',type=pathlib.Path,required=True);cmp.add_argument('--output',type=pathlib.Path,required=True);cmp.add_argument('--phase',required=True)
 con=sub.add_parser('controls');con.add_argument('--output',type=pathlib.Path,required=True)
 args=parser.parse_args()
 if args.command=='capture':
  data=capture();save(args.output,data);print(json.dumps({k:data[k] for k in ['actual_git_archive_tree','tracked_archive_files','summary','consistent_capture']}));raise SystemExit(0 if data['consistent_capture'] else 1)
 elif args.command=='compare':
  data=compare(load(args.baseline),capture(),args.phase);save(args.output,data);print(json.dumps(data,indent=2));raise SystemExit(0 if data['result']=='pass' else 1)
 else:
  data=controls();save(args.output,data);print(json.dumps(data,indent=2));raise SystemExit(0 if data['result']=='pass' else 1)
