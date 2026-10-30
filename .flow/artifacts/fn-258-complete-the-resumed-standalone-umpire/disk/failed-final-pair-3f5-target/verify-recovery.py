#!/usr/bin/env python3
"""Verify old paired guest/failure proof using only retained manifests and CAS."""
import hashlib,json,pathlib,tarfile
HERE=pathlib.Path(__file__).resolve().parent
def digest(path):
 h=hashlib.sha256()
 with path.open('rb') as f:
  for data in iter(lambda:f.read(1024*1024),b''):h.update(data)
 return h.hexdigest()
def verify(manifest,archive):
 for row in manifest['objects']:
  path=pathlib.Path(row['recovery_object']);assert path.is_file() and path.stat().st_size==row['size'] and digest(path)==row['sha256'],row['destination']
 obj=archive['object'];assert obj in manifest['objects']
 with tarfile.open(obj['recovery_object'],'r:gz') as saved:
  actual=saved.getmembers();assert len(actual)==len(archive['members'])
  for member,expected in zip(actual,archive['members']):
   assert member.isfile() and member.name==expected['path'] and member.size==expected['size'] and oct(member.mode)==expected['mode']
   assert not pathlib.PurePosixPath(member.name).is_absolute() and '..' not in pathlib.PurePosixPath(member.name).parts
   h=hashlib.sha256()
   stream=saved.extractfile(member)
   for chunk in iter(lambda:stream.read(1024*1024),b''):h.update(chunk)
   assert h.hexdigest()==expected['sha256'],member.name
 return {'objects':len(manifest['objects']),'unique_objects':len({r['sha256'] for r in manifest['objects']}),'archive_members':len(actual)}
def main():
 manifest=json.loads((HERE/'recovery-manifest.json').read_text());archive=json.loads((HERE/'archive-manifest.json').read_text());result=verify(manifest,archive)
 # In-memory expected digest/missing member corruptions must refuse recovery.
 import copy
 bad=copy.deepcopy(manifest);bad['objects'][0]['sha256']='0'*64
 try:verify(bad,archive)
 except AssertionError:pass
 else:raise AssertionError('corrupt digest accepted')
 bad=copy.deepcopy(archive);bad['members']=bad['members'][:-1]
 try:verify(manifest,bad)
 except AssertionError:pass
 else:raise AssertionError('missing member accepted')
 result.update(result='pass',original_cache_or_source_reads=False,controls={'corrupt_object_digest_refused':True,'missing_archive_member_refused':True})
 (HERE/'independent-recovery-verification.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
if __name__=='__main__':main()
