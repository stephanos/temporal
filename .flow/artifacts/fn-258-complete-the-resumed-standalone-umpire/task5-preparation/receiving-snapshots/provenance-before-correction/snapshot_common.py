"""Read-only source helpers and explicit custody policy; writes limited to isolated store."""
import functools
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import subprocess

HERE=Path(__file__).resolve().parent
TEMPORAL=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
STANDALONE=TEMPORAL.parent/'umpire'
STORE=TEMPORAL.parent/'.recovery/fn258-receiving-snapshots'
BARE=STORE/'prepared.git'
BLOBS=STORE/'blob-objects'
HISTORY=HERE.parents[1]/'history/history-map.json'
EXPORTER=HERE.parents[1]/'history/export-history.py'
ORIGINAL_VERIFIER=HERE.parents[1]/'history/verify-export.py'
PLAN_ORIGINAL=HERE.parent/'receiving-ref-creation-work.json'
OVERLAY_ORIGINAL=HERE.parent/'disposition-overlay.json'
PLAN=HERE/'receiving-ref-creation-work.input.json'
OVERLAY=HERE/'disposition-overlay.input.json'
LEDGER=HERE.parents[1]/'source/candidate-ledger.json'
CUSTODY_ORIGINAL=HERE.parent/'custody/custody-manifest.json'
CUSTODY=HERE/'custody-manifest.input.json'
EXCEPTION_HEAD='86f812692797a0e3451133f0d59f382818584312'
EXCEPTION_FILES={
 'model/dev/xtask/Cargo.toml':'197ea11f8d05d8d492d366586848ec97c8769ff1',
 'model/dev/xtask/README.md':'e51d320746c8e779225f9e54831b33e5b4ca2c71',
 'model/dev/xtask/tests/main.rs':'15feea01ea8f3a661f271c574b039646d69900a6',
 'model/dev/xtask/tests/retired_names.rs':'5b2ac37d480cf02b1a76c224c9f01a10dbc88f8b'}
ENV={k:v for k,v in os.environ.items()if not k.startswith('GIT_')}
ENV.update(GIT_OPTIONAL_LOCKS='0',BASH_ENV='')

def git(repo,*args,data=None,check=True):
 r=subprocess.run(['git','--no-optional-locks','--no-replace-objects','-c','core.fsmonitor=false','-c','core.hooksPath=/dev/null','-C',str(repo),*args],input=data,capture_output=True,env=ENV)
 if check and r.returncode:raise ValueError(str(args)+': '+r.stderr.decode(errors='replace'))
 return r.stdout if check else r.returncode

def sha(data):return hashlib.sha256(data).hexdigest()
def read(path):return json.loads(path.read_bytes())
def write(path,value):
 assert path.parent==HERE
 path.write_text(json.dumps(value,indent=2)+'\n')
def import_module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m

def inputs():
 pairs=[(p,p)for p in [HISTORY,EXPORTER,ORIGINAL_VERIFIER,LEDGER,HERE/'selection-policy.json']]
 pairs += [(PLAN_ORIGINAL,PLAN),(OVERLAY_ORIGINAL,OVERLAY),(CUSTODY_ORIGINAL,CUSTODY)]
 return {original.name:{'path':str(original),'sha256':sha(frozen.read_bytes())}for original,frozen in pairs}

@functools.lru_cache(maxsize=None)
def ancestor(name,revision):
 anchor=read(HISTORY)['boundaries'][name]['commit']
 return git(TEMPORAL,'merge-base','--is-ancestor',anchor,revision,check=False)==0

def rules(revision):
 r=[('model2/framework/',''),('model2/umpire/',''),('model2/hosts/go/','hosts/go/')]
 if ancestor('canonical_archive_move',revision)or ancestor('sibling_archive_move',revision):
  r.append(('model/umpire/',''))
  if not ancestor('host_move_inside_framework',revision):r.append(('model/hosts/go/','hosts/go/'))
  if not ancestor('native_tool_split',revision):r.append(('model/dev/xtask/','dev/xtask/'))
 return r

def policy_valid(policy):
 e=policy['candidate049_source_only_exception']
 if e['source_commit']!=EXCEPTION_HEAD or e['source_files']!={p:{'mode':'100644','blob':oid}for p,oid in EXCEPTION_FILES.items()}:
  raise ValueError('exception-identity-or-prefix-drift')
 if e['source_keeper_ref']!='refs/heads/work/fn25413-driver':raise ValueError('exception-keeper-drift')


def selected(revision,policy):
 policy_valid(policy)
 pairs=rules(revision);rows=[];excluded=[]
 selectors=[p.rstrip('/')for p,_ in pairs]+['LICENSE']
 for row in git(TEMPORAL,'ls-tree','-rz',revision,'--',*selectors).split(b'\0'):
  if not row:continue
  metadata,path=row.split(b'\t',1);mode,kind,oid=metadata.decode().split();source=path.decode()
  target='LICENSE'if source=='LICENSE'else None
  for prefix,dest in pairs:
   if source.startswith(prefix):target=dest+source[len(prefix):];break
  if target is None:continue
  if kind!='blob' or mode not in {'100644','100755','120000'}:raise ValueError('unsupported selected entry '+source)
  record={'source_path':source,'destination_path':target,'mode':mode,'blob':oid,'origin':'committed'}
  if revision==EXCEPTION_HEAD and source in EXCEPTION_FILES:
   if {'mode':mode,'blob':oid}!=policy['candidate049_source_only_exception']['source_files'][source]:raise ValueError('exception-blob-drift')
   excluded.append(record);continue
  rows.append(record)
 if revision==EXCEPTION_HEAD and {r['source_path']for r in excluded}!=set(EXCEPTION_FILES):raise ValueError('exception-prefix-coverage')
 return rows,excluded


def safe(path,data=None):
 parts=path.split('/')
 if not path or path.startswith('/')or any(p in {'','.','..','.git','.flow','.plans','archived'}for p in parts):raise ValueError('unsafe-or-tracker-path')
 if 'temporal'in path.lower()or path.endswith(('.scala','.sc','.class','.jar')):raise ValueError('system-or-Scala-path')
 if data is not None and (path.endswith(('Cargo.toml','.rs','.go','go.mod'))):
  text=data.decode(errors='replace')
  if re.search(r'(?m)^\s*(?:name\s*=\s*"temporal[-_]|use\s+umpire_temporal\b|umpire[-_]temporal(?:[-_][\w-]+)?\s*=|"github\.com/temporalio/temporal(?:/|"))',text):raise ValueError('system-import')


def normalize(rows):
 result={}
 for r in rows:
  target=r['destination_path'];safe(target)
  if target in result:raise ValueError('destination-collision')
  result[target]=r
 for path in result:
  if any('/'.join(path.split('/')[:i])in result for i in range(1,len(path.split('/')))):raise ValueError('file-directory-collision')
 return result


def custody_bytes(record):
 data=Path(record['object']).read_bytes()
 if sha(data)!=record['sha256']or len(data)!=record['size']:raise ValueError('custody-CAS-corruption')
 return data


def dirty_rows(committed,snap,revision,policy,source_bytes):
 result={r['source_path']:dict(r)for r in committed};changes=[];excluded=[]
 for raw in snap['raw_working_files']:
  source=raw['path'];target=None
  for prefix,dest in rules(revision):
   if source.startswith(prefix):target=dest+source[len(prefix):];break
  if target is None:
   excluded.append({'source_path':source,'reason':'outside original revision-qualified generic selection; preserved in source custody archive'});continue
  if raw['kind']=='missing':result.pop(source,None);changes.append({'source_path':source,'destination_path':target,'deleted':True});continue
  if raw['kind']not in {'file','symlink'}:raise ValueError('unsupported dirty file kind')
  data=custody_bytes(raw['content']);safe(target,data);oid=hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()
  source_bytes[oid]=data
  record={'source_path':source,'destination_path':target,'mode':raw['mode'],'blob':oid,'origin':'dirty-CAS','custody_sha256':raw['content']['sha256']}
  result[source]=record;changes.append(record)
 return list(result.values()),changes,excluded


def original_projection(revision):
 m=import_module('approved_export_'+revision,EXPORTER);m.SOURCE=revision
 h=read(HISTORY);h['support_provenance_allowlist']=[entry for entry in h['support_provenance_allowlist']if not entry['destination'].startswith('.history-support/')]
 rev=revision.encode();sets={'H':set(m.git(TEMPORAL,'rev-list',revision).splitlines()),
  'R':{rev}if ancestor('canonical_archive_move',revision)or ancestor('sibling_archive_move',revision)else set(),
  'D_host_move_inside_framework':{rev}if ancestor('host_move_inside_framework',revision)else set(),
  'D_native_tool_split':{rev}if ancestor('native_tool_split',revision)else set()}
 aliases=io.StringIO();projection=m.Projection(TEMPORAL,h,sets,aliases).snapshot(rev)
 if aliases.getvalue():raise ValueError('original-identical-alias-collision-requires-review')
 return {p.decode():{'mode':v[0].decode(),'blob':v[1].decode()}for p,v in projection.items()}


def canonical_state():
 tracked=git(STANDALONE,'ls-files','--stage','-z');paths={row.split(b'\t',1)[1]for row in tracked.split(b'\0')if row}
 paths.update(p for p in git(STANDALONE,'ls-files','--others','--exclude-standard','-z').split(b'\0')if p)
 digest=hashlib.sha256()
 for relative in sorted(paths):
  path=STANDALONE/os.fsdecode(relative);digest.update(relative+b'\0')
  if path.is_symlink():data=os.fsencode(os.readlink(path));mode=b'120000'
  else:data=path.read_bytes();mode=b'100755'if path.stat().st_mode&0o111 else b'100644'
  digest.update(mode+b' '+hashlib.sha256(data).digest())
 idx=Path(os.fsdecode(git(STANDALONE,'rev-parse','--path-format=absolute','--git-path','index').strip()))
 return {'head':git(STANDALONE,'rev-parse','HEAD').decode().strip(),'tree':git(STANDALONE,'rev-parse','HEAD^{tree}').decode().strip(),
 'refs_sha256':sha(git(STANDALONE,'for-each-ref','--format=%(refname) %(objectname)')),'index_entries_sha256':sha(tracked),
 'index_raw_sha256':sha(idx.read_bytes()),'working_bytes_modes_sha256':digest.hexdigest()}
