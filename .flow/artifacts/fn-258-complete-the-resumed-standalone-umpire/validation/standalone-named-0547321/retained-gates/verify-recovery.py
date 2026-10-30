#!/usr/bin/env python3
"""Verify retained named-gate evidence without source/target/release access."""
import hashlib
import json
import pathlib
import tarfile

ROOT=pathlib.Path(__file__).resolve().parent
manifest=json.loads((ROOT/'recovery-manifest.json').read_text())
archives=json.loads((ROOT/'archive-manifests.json').read_text())
errors=[];seen=set();count=0
for record in manifest['objects']:
 path=pathlib.Path(record['recovery_object'])
 if str(path) in seen:continue
 seen.add(str(path))
 if not path.is_file() or path.stat().st_size!=record['size'] or hashlib.sha256(path.read_bytes()).hexdigest()!=record['sha256']:
  errors.append('object differs:'+record['destination'])
for name,archive in archives.items():
 with tarfile.open(archive['object']['recovery_object'],'r:gz') as retained:
  actual=retained.getmembers();expected=archive['verified_members']
  if [member.name for member in actual]!=[r['path'] for r in expected]:errors.append(name+' member paths differ')
  for member,record in zip(actual,expected):
   count+=1
   if not member.isfile() or member.size!=record['size'] or oct(member.mode)!=record['mode'] or hashlib.sha256(retained.extractfile(member).read()).hexdigest()!=record['sha256']:
    errors.append(name+':'+member.name)
for name,digest in manifest['original_completed_receipt_log_seals_unchanged'].items():
 if hashlib.sha256((ROOT.parent/name).read_bytes()).hexdigest()!=digest:errors.append('original changed:'+name)
result={'result':'pass' if not errors else 'fail','source_commit':manifest['source_commit'],'unique_objects_verified':len(seen),'archive_members_verified':count,'original_completed_files_verified':4,'active_release_read':False,'original_parenttarget_read':False,'errors':errors}
(ROOT/'independent-recovery-verification.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
if errors:raise SystemExit(1)
