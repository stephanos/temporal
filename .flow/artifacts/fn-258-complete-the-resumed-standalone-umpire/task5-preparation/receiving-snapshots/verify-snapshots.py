#!/usr/bin/env python3
"""Independently project raw source Git trees and verify prepared/recovered custody objects."""
import collections
import copy
import functools
import hashlib
from pathlib import Path
import snapshot_common as c

RULES=[('model2/framework/',''),('model2/umpire/',''),('model2/hosts/go/','hosts/go/'),
       ('model/umpire/',''),('model/hosts/go/','hosts/go/'),('model/dev/xtask/','dev/xtask/')]


def main():
 report=c.read(c.HERE/'prepared-targets.json');policy=c.read(c.HERE/'selection-policy.json');c.policy_valid(policy)
 assert report['input_bindings']==c.inputs()
 plan=c.read(c.PLAN);ledger=c.read(c.LEDGER);custody=c.read(c.CUSTODY);history=c.read(c.HISTORY)
 raw=c.import_module('independent_raw_git',c.ORIGINAL_VERIFIER)
 source=raw.Repository(c.TEMPORAL);recovered=raw.Repository(Path(report['independent_recovery']))
 prepared=raw.Repository(Path(report['isolated_bare_store']))
 baseline=c.canonical_state();assert baseline==report['canonical_standalone_after']==report['canonical_standalone_before']
 before_dirty=c.import_module('custody_read_helpers',c.HERE.parent/'custody/capture-custody.py')
 dirty_fp={s['candidate']:before_dirty.fingerprint(Path(s['root']))for s in custody['snapshots']};assert dirty_fp==custody['originals_after']
 @functools.lru_cache(maxsize=None)
 def descends(name,revision):
  return c.git(c.TEMPORAL,'merge-base','--is-ancestor',history['boundaries'][name]['commit'],revision,check=False)==0
 @functools.lru_cache(maxsize=256)
 def project(revision):
  root,_,_=raw.commit_parts(source.object(revision,'commit'));entries=[];retained=[];all_selected=[]
  archived=descends('canonical_archive_move',revision)or descends('sibling_archive_move',revision)
  for prefix,destination in RULES:
   allowed=prefix.startswith('model2/')or (archived and(prefix=='model/umpire/'or(prefix=='model/hosts/go/'and not descends('host_move_inside_framework',revision))or(prefix=='model/dev/xtask/'and not descends('native_tool_split',revision))))
   if not allowed:continue
   subtree=source.lookup(root,prefix.rstrip('/').encode())
   if subtree is None:continue
   assert subtree[0]=='040000'
   for path,(mode,oid)in source.leaves(subtree[1]):
    original=prefix+path.decode();target=destination+path.decode();r={'source_path':original,'destination_path':target,'mode':mode,'blob':oid,'origin':'committed'};all_selected.append(r)
    if revision==c.EXCEPTION_HEAD and original in c.EXCEPTION_FILES:
     assert mode=='100644'and oid==c.EXCEPTION_FILES[original];retained.append(r)
    else:entries.append(r)
  license=source.lookup(root,b'LICENSE');assert license
  lic={'source_path':'LICENSE','destination_path':'LICENSE','mode':license[0],'blob':license[1],'origin':'committed'};entries.append(lic);all_selected.append(lic)
  assert len(all_selected)==len(entries)+len(retained)
  if revision==c.EXCEPTION_HEAD:assert len(retained)==4
  return entries,retained,all_selected
 @functools.lru_cache(maxsize=16384)
 def original_blob(oid):return source.object(oid,'blob')
 def payload_for(record):
  data=Path(record['blob_object']).read_bytes();assert Path(record['blob_object']).parent==c.BLOBS
  assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==record['blob']
  assert len(data)==record['size']and c.sha(data)==record['sha256']
  return data
 def same_records(expected,actual,dirty_payload=None):
  expected_map={r['destination_path']:r for r in expected};assert len(expected_map)==len(expected)
  actual_map={r['destination_path']:r for r in actual};assert len(actual_map)==len(actual)
  assert set(actual_map)==set(expected_map),'full-projection-path-coverage'
  for dest,r in actual_map.items():
   c.safe(dest);e=expected_map[dest]
   assert all(r[key]==e[key]for key in ['source_path','mode','blob','origin']),'source-mode-blob-coverage'
   data=payload_for(r);c.safe(dest,data)
   expected_bytes=(dirty_payload or{}).get(r['source_path']) if e['origin']=='dirty-CAS'else original_blob(e['blob'])
   assert data==expected_bytes,'source-byte-coverage'
  return actual_map
 def deletion_absent(record,after_paths):
  assert record['after_destination_absent']is True and record['destination_path']not in after_paths
 def raw_dirty(expected,snap,revision):
  result={r['source_path']:dict(r)for r in expected};payloads={};excluded=[];changes=[]
  archived=descends('canonical_archive_move',revision)or descends('sibling_archive_move',revision)
  for item in snap['raw_working_files']:
   original=item['path'];destination=None
   for prefix,target in RULES:
    allowed=prefix.startswith('model2/')or(archived and(prefix=='model/umpire/'or(prefix=='model/hosts/go/'and not descends('host_move_inside_framework',revision))or(prefix=='model/dev/xtask/'and not descends('native_tool_split',revision))))
    if allowed and original.startswith(prefix):destination=target+original[len(prefix):];break
   if destination is None:excluded.append(original);continue
   if item['kind']=='missing':result.pop(original,None);changes.append(original);continue
   data=c.custody_bytes(item['content']);oid=hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest();payloads[original]=data
   result[original]={'source_path':original,'destination_path':destination,'mode':item['mode'],'blob':oid,'origin':'dirty-CAS'};changes.append(original)
  return list(result.values()),payloads,excluded,changes
 variants={v['name']:v for v in report['variants']};assert len(variants)==50
 plans={w['receiver']['planned_ref']:w for w in plan['work']};candidates=report['candidates'];assert len(candidates)==48
 assert set(plans)=={r['proposed_receiver_ref']for r in candidates}
 commit_oids={v['commit']for v in variants.values()};assert len(commit_oids)==50
 def custody_metadata(data,expected_parents):
  _,parents,identity=raw.commit_parts(data)
  assert parents==expected_parents
  declared=policy['custody_commit_identity'];expected_identity=(f"{declared['name']} <{declared['email']}> {report['custody_commit_timestamp']} +0000").encode()
  assert identity[0]==(expected_identity,)and identity[1]==(expected_identity,)
 total_files=0;net_deletion_count=0;mapped_deletion_count=0;exception_count=0;raw_variants=0
 for row in candidates:
  p=plans[row['proposed_receiver_ref']];revision=p['source_r1_observed_head']or p['source_original_head'];entry=next(e for e in ledger['entries']if f"{e['repository']}:{e['ref']}"==p['candidate_id'])
  assert row['candidate_id']==p['candidate_id']and row['source_commit']==revision and row['source_ref']==entry['ref']
  assert row['comparison_base']==entry['merge_base_with_frozen_source']
  assert c.git(c.TEMPORAL,'rev-parse',row['source_ref']).decode().strip()==revision
  expected,excluded,all_selected=project(revision);base,_,_=project(row['comparison_base']);base_map={r['destination_path']:r for r in base};expected_map={r['destination_path']:r for r in expected}
  c.normalize(expected);c.normalize(base)
  assert row['source_only_exception']==excluded
  if excluded:
   exception_count+=1;assert revision==c.EXCEPTION_HEAD and len(excluded)==4
   assert len(expected)+len(excluded)==len(all_selected)
  committed=variants[row['committed_variant']];assert committed['kind']=='committed-custody-snapshot'and committed['source_commit']==revision and committed['parents']==[]
  same_records(expected,committed['records']);total_files+=len(expected)
  expected_net=set(base_map)-set(expected_map);actual_net={r['destination_path']for r in row['net_projected_deletions']}
  assert len(actual_net)==len(row['net_projected_deletions'])and expected_net==actual_net
  for deletion in row['net_projected_deletions']:
   e=base_map[deletion['destination_path']];assert deletion['source_path']==e['source_path']and deletion['mode']==e['mode']and deletion['blob']==e['blob']
   deletion_absent(deletion,expected_map)
   assert payload_for(deletion)==original_blob(e['blob'])
  before_source={r['source_path']:r for r in base};after_source={r['source_path']:r for r in expected}
  planned_D=[m for m in p['provisional_path_mappings']if any(o['kind']=='committed-delta'and o['status']=='D'for o in m['observations'])]
  assert len(planned_D)==len(row['mapped_source_deletion_beforeimages'])
  actual_D={r['source_path']:r for r in row['mapped_source_deletion_beforeimages']};assert set(actual_D)=={r['source_path']for r in planned_D}
  for m in planned_D:
   d=actual_D[m['source_path']];e=before_source[m['source_path']]
   assert d['destination_path']==m['destination_path']==e['destination_path']and d['blob']==e['blob']and d['mode']==e['mode']
   assert payload_for(d)==original_blob(e['blob'])
   assert d['source_path_after_absent']==(m['source_path']not in after_source)
   assert d['mapped_destination_after_absent']==(m['destination_path']not in expected_map)
  net_deletion_count+=len(expected_net);mapped_deletion_count+=len(planned_D)
  if row['candidate']in {'039','075'}:
   raw_variants+=1;snap=next(s for s in custody['snapshots']if s['candidate']==row['candidate']);target=variants[row['target_variant']]
   dirty,payloads,dirty_excluded,changes=raw_dirty(expected,snap,revision)
   same_records(dirty,target['records'],payloads);assert target['kind']=='raw-dirty-custody-variant'and target['parents']==[row['committed_variant']]
   assert target['custody_archive']==custody['archives'][row['candidate']]and target['raw_index']==snap['raw_index']and target['unmerged_stages']==snap['unmerged_stages']
   assert {r['source_path']for r in target['source_only_dirty_records']}==set(dirty_excluded)
   assert {r['source_path']for r in target['raw_changes']}==set(changes)
   assert target['buildable_or_feature_accepted']is False
   total_files+=len(dirty)
  else:assert row['target_variant']==row['committed_variant']
  assert row['target_oid']==variants[row['target_variant']]['commit']and row['committed_oid']==committed['commit']
  assert c.git(Path(report['isolated_bare_store']),'rev-parse',row['prepared_ref']).decode().strip()==row['target_oid']
  assert c.git(c.STANDALONE,'show-ref','--verify','--quiet',row['proposed_receiver_ref'],check=False)!=0
 for variant in variants.values():
  assert variant['buildable_or_feature_accepted']is False
  expected_tree={r['destination_path']:(r['mode'],r['blob'])for r in variant['records']}
  for repo in [prepared,recovered]:
   data=repo.object(variant['commit'],'commit');tree,parents,identity=raw.commit_parts(data)
   assert c.sha(data)==variant['raw_commit_sha256']and tree==variant['tree']
   assert parents==[variants[name]['commit']for name in variant['parents']]==variant['prepared_parent_oids']
   assert all(parent in commit_oids for parent in parents)
   custody_metadata(data,variant['prepared_parent_oids'])
   assert dict((p.decode(),v)for p,v in repo.leaves(tree))==expected_tree
   for r in variant['records']:assert repo.object(r['blob'],'blob')==payload_for(r)
 heads=[row['target_oid']for row in candidates];actual_graph=set(c.git(Path(report['independent_recovery']),'rev-list',*heads).decode().splitlines());assert actual_graph==commit_oids,'unmapped-or-original-history-import'
 assert exception_count==1 and raw_variants==2
 # Refusal controls exercise source/policy/tree manifests without changing real stores.
 controls=[]
 def refuses(name,expected,fn):
  try:fn();passed=False
  except (AssertionError,ValueError,KeyError):passed=True
  controls.append({'name':name,'expected_refusal':expected,'passed':passed})
 sample=next(v for v in variants.values()if v['kind']=='committed-custody-snapshot');sample_expected=project(sample['source_commit'])[0]
 def changed_record(field,value):
  records=copy.deepcopy(sample['records']);records[0][field]=value;same_records(sample_expected,records)
 refuses('missing-file','full projection coverage',lambda:same_records(sample_expected,sample['records'][1:]))
 refuses('unexpected-tracker-file','source path coverage',lambda:same_records(sample_expected,sample['records']+[dict(sample['records'][0],destination_path='.flow/tasks/fake.md')]))
 refuses('wrong-mode','exact mode identity',lambda:changed_record('mode','100755'if sample['records'][0]['mode']=='100644'else'100644'))
 refuses('wrong-blob','exact blob identity',lambda:changed_record('blob','0'*40))
 refuses('duplicate-destination','collision rejection',lambda:c.normalize(sample['records']+[dict(sample['records'][0],source_path='different/source.rs')]))
 refuses('system-import','no system dependencies',lambda:c.safe('Cargo.toml',b'[package]\nname = "temporal-xtask"\n'))
 for name,path,data in [
  ('Go-canonical-SDK','probe.go',b'package probe\nimport "go.temporal.io/sdk/client"\n'),
  ('Go-canonical-API','probe.go',b'package probe\nimport (\n api "go.temporal.io/api/workflow/v1"\n)\n'),
  ('Go-module-API','go.mod',b'module probe\nrequire go.temporal.io/api v1.0.0\n'),
  ('Rust-API-import','probe.rs',b'use temporal_api::workflow::Workflow;\n'),
  ('Rust-API-qualified','probe.rs',b'fn f() { temporal_api::run(); }\n'),
  ('Rust-API-dependency','Cargo.toml',b'[dependencies]\ntemporal-api = "1"\n'),
  ('Rust-renamed-API-dependency','Cargo.toml',b'[dependencies]\nother = { package = "umpire-temporal-api", version = "1" }\n')]:
  refuses(name,'actual system dependency form',lambda path=path,data=data:c.safe(path,data))
 c.safe('fixture.rs',b'const BAD: &str = r#"use temporal_api::Workflow;\nuse umpire_temporal::X;"#;\n')
 c.safe('fixture.go',b'package fixture\nconst bad = `import "go.temporal.io/api/workflow/v1"`\n')
 refuses('Scala-file','no Scala bytes',lambda:c.safe('legacy/Source.scala',b'object X'))
 for name,mutate in [('exception-OID-drift',lambda p:p['candidate049_source_only_exception'].update(source_commit='0'*40)),('exception-blob-drift',lambda p:p['candidate049_source_only_exception']['source_files']['model/dev/xtask/Cargo.toml'].update(blob='0'*40)),('exception-extra-prefix-file',lambda p:p['candidate049_source_only_exception']['source_files'].update({'model/dev/xtask/extra.rs':{'mode':'100644','blob':'0'*40}}))]:
  def action(mutate=mutate):p=copy.deepcopy(policy);mutate(p);c.policy_valid(p)
  refuses(name,'exact root-approved four-path policy',action)
 dirty=next(v for v in variants.values()if v['kind']=='raw-dirty-custody-variant');snap=next(s for s in custody['snapshots']if s['candidate']==dirty['candidate']);expected,payloads,_,_=raw_dirty(project(dirty['source_commit'])[0],snap,dirty['source_commit'])
 refuses('dirty-path-omitted','full raw custody coverage',lambda:same_records(expected,dirty['records'][:-1],payloads))
 deletion=next(r['net_projected_deletions'][0]for r in candidates if r['net_projected_deletions'])
 def corrupt_beforeimage():d=copy.deepcopy(deletion);d['sha256']='0'*64;payload_for(d)
 refuses('deletion-beforeimage-corrupt','raw beforeimage hash',corrupt_beforeimage)
 refuses('deletion-after-absence-false','exact after-absence',lambda:deletion_absent(dict(deletion,after_destination_absent=False),set()))
 refuses('deleted-path-resurrected','exact after-absence',lambda:deletion_absent(deletion,{deletion['destination_path']}))
 sample_raw=prepared.object(sample['commit'],'commit')
 foreign_parent=sample_raw.replace(b'\nauthor ',b'\nparent '+sample['source_commit'].encode()+b'\nauthor ',1)
 refuses('original-parent-invention','new custody parents only',lambda:custody_metadata(foreign_parent,[]))
 false_author=sample_raw.replace(policy['custody_commit_identity']['name'].encode(),b'Original Human',1)
 refuses('original-author-spoof','new agent authorship only',lambda:custody_metadata(false_author,[]))
 assert all(item['passed']for item in controls)
 bundle=Path(report['prepared_bundle']['path']);committed_bundle=c.HERE/'prepared-snapshots.bundle'
 def check_bundle(data):
  assert len(data)==report['prepared_bundle']['size']and c.sha(data)==report['prepared_bundle']['sha256']
 check_bundle(bundle.read_bytes());check_bundle(committed_bundle.read_bytes())
 refuses('committed-bundle-corrupt','exact committed bundle hash and size',lambda:check_bundle(committed_bundle.read_bytes()[:-1]))
 bundle_heads=dict(line.split()for line in c.git(c.STANDALONE,'bundle','list-heads',str(committed_bundle)).decode().splitlines())
 assert set(bundle_heads)==set(heads)
 # Reachability, not mere object-database existence, from the actual bundle heads.
 bundle_reachable=set(c.git(Path(report['independent_recovery']),'rev-list','--objects','--no-object-names',*bundle_heads).decode().splitlines())
 canonical='05473210b48c3db862f22051eb7154ce98258f98'
 canonical_reachable=set(c.git(c.STANDALONE,'rev-list','--objects','--no-object-names',canonical).decode().splitlines())
 support=c.read(c.HERE/'candidate-049-ownership-disposition.json')['source_only_provenance_support_records']
 support_root,_,_=raw.commit_parts(source.object(c.EXCEPTION_HEAD,'commit'))
 qualified_support=[]
 for qualification in history['support_provenance_allowlist']:
  if not qualification['destination'].startswith('.history-support/'):continue
  entry=source.lookup(support_root,qualification['path'].encode())
  if entry is not None:qualified_support.append((qualification['path'],entry[0],entry[1]))
 assert len(support)==len(qualified_support)==28
 assert {(row['source_path'],row['mode'],row['blob'])for row in support}==set(qualified_support)

 support_facts=[dict(row,computed_reachable_from_canonical054=row['blob']in canonical_reachable)for row in support]
 beforeimages=[dict(candidate=row['candidate'],kind=kind,**record)for row in candidates for kind in ['net_projected_deletions','mapped_source_deletion_beforeimages']for record in row[kind]]
 gaps=[dict(row,bundle_reachable=False,recovery_location=row['blob_object'],original_source_repository=str(c.TEMPORAL))for row in beforeimages if row['blob']not in bundle_reachable]
 for gap in gaps:payload_for(gap)
 support_gaps=[row for row in support_facts if not row['computed_reachable_from_canonical054']]
 for gap in support_gaps:assert original_blob(gap['blob'])
 # All planned changed paths are accounted by the complete proposed target projection
 # or an explicit mapped source-deletion beforeimage, independent of dirty variants.
 present=0;deleted=0
 for row in candidates:
  p=plans[row['proposed_receiver_ref']];paths={r['source_path']for r in variants[row['target_variant']]['records']};deletions={r['source_path']for r in row['mapped_source_deletion_beforeimages']}
  for mapping in p['provisional_path_mappings']:
   path=mapping['source_path'];assert (path in paths)!=(path in deletions)
   present+=path in paths;deleted+=path in deletions
 assert present+deleted==sum(len(p['provisional_path_mappings'])for p in plans.values())
 c.write(c.HERE/'computed-recovery-coverage.json',{'canonical_head':canonical,'canonical_support_records':support_facts,'canonical_support_reachable':len(support)-len(support_gaps),'canonical_support_gaps':support_gaps,'actual_bundle_sha256':c.sha(committed_bundle.read_bytes()),'bundle_heads':bundle_heads,'beforeimage_occurrences':len(beforeimages),'beforeimage_unique_blobs':len({r['blob']for r in beforeimages}),'beforeimage_blobs_not_bundle_reachable':gaps,'planned_occurrences':present+deleted,'receiver_present':present,'mapped_deletion_beforeimages':deleted,'unaccounted':0})
 assert all(item['passed']for item in controls)

 after=c.canonical_state();assert after==baseline
 assert {s['candidate']:before_dirty.fingerprint(Path(s['root']))for s in custody['snapshots']}==dirty_fp
 for ref,oid in report['source_refs_before'].items():assert c.git(c.TEMPORAL,'rev-parse',ref).decode().strip()==oid
 for repo in [source,prepared,recovered]:repo.close()
 transaction='start\n'+''.join('create '+row['proposed_receiver_ref']+' '+row['target_oid']+'\n'for row in candidates)+'prepare\ncommit\n'
 (c.HERE/'proposed-create-only-transaction.txt').write_text(transaction)
 test=c.STORE/'transaction-test.git'
 if not test.exists():
  c.git(c.STORE,'init','--bare',str(test));c.git(test,'bundle','unbundle',str(bundle));c.git(test,'update-ref','--stdin',data=transaction.encode())
 else:
  assert dict(line.split()for line in c.git(test,'for-each-ref','--format=%(refname) %(objectname)').decode().splitlines())=={row['proposed_receiver_ref']:row['target_oid']for row in candidates}
 assert c.git(test,'update-ref','--stdin',data=transaction.encode(),check=False)!=0
 for row in candidates:assert c.git(test,'rev-parse',row['proposed_receiver_ref']).decode().strip()==row['target_oid']
 receipt={'result':'pass','prepared_manifest_sha256':c.sha((c.HERE/'prepared-targets.json').read_bytes()),
  'verifier_sha256':c.sha(Path(__file__).read_bytes()),'builder_sha256':c.sha((c.HERE/'build-snapshots.py').read_bytes()),'common_helper_sha256':c.sha((c.HERE/'snapshot_common.py').read_bytes()),
  'frozen_preparation_input_copies':{str(c.PLAN_ORIGINAL):str(c.PLAN),str(c.OVERLAY_ORIGINAL):str(c.OVERLAY),str(c.CUSTODY_ORIGINAL):str(c.CUSTODY)},
  'verified_targets':48,'new_agent_custody_commits':50,'full_variant_file_records_verified':total_files,
  'net_projected_deletion_beforeimages':net_deletion_count,'mapped_source_deletion_beforeimages':mapped_deletion_count,
  'exact_candidate049_source_only_paths':4,'qualified_dirty_variants':2,'exception_and_selection_controls':controls,
  'computed_recovery_coverage_sha256':c.sha((c.HERE/'computed-recovery-coverage.json').read_bytes()),'committed_bundle_verified':{'path':str(committed_bundle),'sha256':c.sha(committed_bundle.read_bytes()),'size':committed_bundle.stat().st_size},'bundle_recovery_exact_graph':True,'no_original_authorship_or_parent_claim':True,
  'transaction_test':'Applied once only to isolated transaction-test.git; repeated create-only transaction refused; never run against real standalone.',
  'transaction_sha256':c.sha(transaction.encode()),'canonical_standalone_unchanged':True,'source_refs_and_dirty_custody_unchanged':True,
  'real_receiving_refs_created':0,'actual_transfer_or_activation':False,'task4_or_feature_acceptance':False}
 c.write(c.HERE/'verification.json',receipt)
 print('PASS:',48,'targets;',50,'custody commits;',total_files,'file records;',net_deletion_count,'net deletion beforeimages;',mapped_deletion_count,'mapped source deletions; zero real receiving refs',flush=True)

if __name__=='__main__':main()
