#!/usr/bin/env python3
"""Fresh independent verification of already-retained proof; no source gate."""
import hashlib,json,pathlib,sys,tarfile
HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
BASE=ROOT/'.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation'
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
 return h.hexdigest()
def verify():
 results={}
 for name,sub,key in [('required','standalone-0547321','records'),('named','standalone-named-0547321/retained-gates','objects')]:
  p=BASE/sub;manifest=json.loads((p/'recovery-manifest.json').read_text());archives=json.loads((p/'archive-manifests.json').read_text());seen=set();count=0
  for r in manifest[key]:
   obj=pathlib.Path(r['recovery_object'])
   if str(obj) not in seen:
    assert obj.is_file() and obj.stat().st_size==r['size'] and sha(obj)==r['sha256'];seen.add(str(obj))
  for archive in archives.values():
   with tarfile.open(archive['object']['recovery_object'],'r:gz') as saved:
    actual=saved.getmembers();expected=archive['verified_members'];assert len(actual)==len(expected)
    for a,e in zip(actual,expected):
     assert a.isfile() and a.name==e['path'] and a.size==e['size'] and oct(a.mode)==e['mode'];h=hashlib.sha256();f=saved.extractfile(a)
     for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
     assert h.hexdigest()==e['sha256'];count+=1
  if name=='named':
   for leaf,h in manifest['original_completed_receipt_log_seals_unchanged'].items():assert sha(p.parent/leaf)==h
  results[name]={'unique_objects':len(seen),'archive_members':count,'result':'pass'}
 assert results['required']['unique_objects']==158 and results['required']['archive_members']==7979
 assert results['named']['unique_objects']==142 and results['named']['archive_members']==768
 return results
if __name__=='__main__':
 result={'classification':'deferred-cache housekeeping recovery check only','proofs':verify(),'original_targets_read':False,'no_original_hashed_evidence_rewritten':True}
 destination=HERE/(sys.argv[1] if len(sys.argv)>1 else 'recovery-before.json');destination.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
