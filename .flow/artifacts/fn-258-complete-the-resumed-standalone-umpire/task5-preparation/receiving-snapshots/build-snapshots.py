#!/usr/bin/env python3
"""Prepare complete recovery trees/commits in a new isolated bare object store only."""
import hashlib
import json
import shutil
import time
from pathlib import Path
import snapshot_common as c


def main():
 assert shutil.disk_usage(c.TEMPORAL.parent).free>30*1024**3
 assert not c.BARE.exists(), 'New isolated object store required; preserve previous attempt'
 assert not(c.HERE/'prepared-targets.json').exists()
 policy=c.read(c.HERE/'selection-policy.json');c.policy_valid(policy)
 plan=c.read(c.PLAN);ledger=c.read(c.LEDGER);custody=c.read(c.CUSTODY)
 assert len(plan['work'])==48 and custody['result']=='pass'
 source_refs={w['candidate_id'].split(':',1)[1]:w['source_r1_observed_head']or w['source_original_head']for w in plan['work']}
 source_refs.update(custody['source_refs'])
 before=c.canonical_state();assert before['head']=='05473210b48c3db862f22051eb7154ce98258f98'
 for ref,oid in source_refs.items():assert c.git(c.TEMPORAL,'rev-parse',ref).decode().strip()==oid
 original=c.import_module('source_git_objects',c.ORIGINAL_VERIFIER);source=original.Repository(c.TEMPORAL)
 c.BLOBS.mkdir(parents=True,exist_ok=True)
 blob_data={};variants=[];candidates=[];all_target_blobs=set()
 def enrich(rows):
  result=[]
  for item in rows:
   r=dict(item);oid=r['blob']
   data=blob_data.get(oid)
   if data is None:data=source.object(oid,'blob');blob_data[oid]=data
   c.safe(r['destination_path'],data)
   assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==oid
   path=c.BLOBS/oid
   if not path.exists():path.write_bytes(data)
   assert path.read_bytes()==data
   r.update(size=len(data),sha256=c.sha(data),blob_object=str(path));result.append(r)
  return sorted(result,key=lambda r:r['destination_path'])
 for number,w in enumerate(plan['work']):
  receiver=w['receiver']['planned_ref'];key=receiver.rsplit('-',1)[-1];rev=w['source_r1_observed_head']or w['source_original_head']
  entry=next(e for e in ledger['entries']if f"{e['repository']}:{e['ref']}"==w['candidate_id']);base=entry['merge_base_with_frozen_source']
  rows,excluded=c.selected(rev,policy);mapping=c.normalize(rows)
  normalized={p:{'mode':r['mode'],'blob':r['blob']}for p,r in mapping.items()}
  projector_status='exact-approved-projector-match'
  try:projected=c.original_projection(rev)
  except RuntimeError as error:
   assert rev==c.EXCEPTION_HEAD and str(error).startswith('Conflicting projection:');projected=None;projector_status='approved-four-path-consumer-ownership-correction-after-original-projector-refusal'
  if projected is not None:assert projected==normalized
  before_rows,before_excluded=c.selected(base,policy);before_map=c.normalize(before_rows)
  assert c.original_projection(base)=={p:{'mode':r['mode'],'blob':r['blob']}for p,r in before_map.items()}
  committed=enrich(rows);by_source={r['source_path']:r for r in committed}
  enriched_before=enrich(before_rows);before_source={r['source_path']:r for r in enriched_before};before_dest={r['destination_path']:r for r in enriched_before}
  after_dest={r['destination_path']:r for r in committed}
  net_deletions=[dict(before_dest[path],after_destination_absent=True)for path in sorted(before_dest.keys()-after_dest.keys())]
  mapped_deletions=[]
  for p in w['provisional_path_mappings']:
   for observation in p['observations']:
    if observation['kind']=='committed-delta':
     if observation['status']=='D':
      assert p['source_path']in before_source
      r=before_source[p['source_path']];assert r['destination_path']==p['destination_path']
      mapped_deletions.append(dict(r,source_path_after_absent=p['source_path']not in by_source,mapped_destination_after_absent=r['destination_path']not in after_dest))
     else:
      assert p['source_path']in by_source and by_source[p['source_path']]['destination_path']==p['destination_path']
  committed_name='candidate-'+key+'-committed'
  variants.append({'name':committed_name,'candidate':key,'kind':'committed-custody-snapshot','source_commit':rev,'parents':[],
   'records':committed,'buildable_or_feature_accepted':False})
  target_name=committed_name;dirty=None
  if key in {'039','075'}:
   snap=next(s for s in custody['snapshots']if s['candidate']==key);assert snap['head']==rev
   dirty,changes,dirty_excluded=c.dirty_rows(rows,snap,rev,policy,blob_data);c.normalize(dirty)
   dirty=enrich(dirty);target_name='candidate-'+key+'-dirty'
   variants.append({'name':target_name,'candidate':key,'kind':'raw-dirty-custody-variant','source_commit':rev,'parents':[committed_name],
    'records':dirty,'raw_changes':changes,'source_only_dirty_records':dirty_excluded,
    'custody_archive':custody['archives'][key],'raw_index':snap['raw_index'],'unmerged_stages':snap['unmerged_stages'],
    'qualification':'Raw working bytes, including unresolved markers, not resolution/compiled/accepted code. Parent is new committed custody snapshot only, not invented original history.',
    'buildable_or_feature_accepted':False})
   dirty_source={r['source_path']for r in dirty}
   for p in w['provisional_path_mappings']:
    if any(o['kind'].startswith('dirty')for o in p['observations']):assert p['source_path']in dirty_source
  candidates.append({'candidate_id':w['candidate_id'],'candidate':key,'source_ref':entry['ref'],'source_commit':rev,'comparison_base':base,
   'proposed_receiver_ref':receiver,'prepared_ref':'refs/prepared/fn258/candidate-'+key,'target_variant':target_name,
   'committed_variant':committed_name,'layout':'exact source-revision generic layout; no active-role rebasing',
   'original_projector':projector_status,'source_only_exception':excluded,
   'source_keeper':{'repository':'temporal','ref':entry['ref'],'head':rev,'action_task':'fn-258-complete-the-resumed-standalone-umpire.5'},
   'net_projected_deletions':net_deletions,'mapped_source_deletion_beforeimages':mapped_deletions,
   'provisional_planned_paths_accounted':len(w['provisional_path_mappings']),
   'source_authorship_history':'Retained at original source refs; these are new agent custody objects, not filtered original commits or feature transfer.'})
  print('Projected',key,len(committed),'files;',len(net_deletions),'net deletions',flush=True)
 source.close()
 for variant in variants:all_target_blobs.update(r['blob']for r in variant['records'])
 c.git(c.STORE,'init','--bare',str(c.BARE))
 assert not(c.BARE/'objects/info/alternates').exists()
 paths=''.join(str(c.BLOBS/oid)+'\n'for oid in sorted(all_target_blobs)).encode()
 copied=c.git(c.BARE,'hash-object','-w','--stdin-paths',data=paths).decode().splitlines();assert copied==sorted(all_target_blobs)
 tree_cache={}
 def tree(records):
  nested={}
  for r in records:
   cursor=nested;parts=r['destination_path'].split('/')
   for part in parts[:-1]:
    if part in cursor and not isinstance(cursor[part],dict):raise ValueError('file-directory-collision')
    cursor=cursor.setdefault(part,{})
   assert parts[-1]not in cursor;cursor[parts[-1]]=(r['mode'],r['blob'])
  def emit(node):
   parts=[]
   for name,value in sorted(node.items()):
    if isinstance(value,dict):mode,kind,oid='040000','tree',emit(value)
    else:mode,oid=value;kind='blob'
    parts.append((mode+' '+kind+' '+oid+'\t'+name).encode()+b'\0')
   raw=b''.join(parts)
   if raw not in tree_cache:tree_cache[raw]=c.git(c.BARE,'mktree','-z',data=raw).decode().strip()
   return tree_cache[raw]
  return emit(nested)
 timestamp=int(time.time());identity=policy['custody_commit_identity'];made={}
 for variant in variants:
  oid=tree(variant['records']);parents=[made[name]for name in variant['parents']]
  raw=('tree '+oid+'\n'+''.join('parent '+parent+'\n'for parent in parents)+
       f"author {identity['name']} <{identity['email']}> {timestamp} +0000\ncommitter {identity['name']} <{identity['email']}> {timestamp} +0000\n\n"+
       'fn258 '+variant['kind']+' '+variant['candidate']+'\n\nSource: '+variant['source_commit']+'\nCustody only; not original authorship/history, buildability, feature acceptance or activation.\n').encode()
  commit=c.git(c.BARE,'hash-object','-t','commit','-w','--stdin',data=raw).decode().strip();made[variant['name']]=commit
  variant.update(tree=oid,commit=commit,raw_commit_sha256=c.sha(raw),prepared_parent_oids=parents)
 for row in candidates:
  row['target_oid']=made[row['target_variant']];row['committed_oid']=made[row['committed_variant']]
  c.git(c.BARE,'update-ref',row['prepared_ref'],row['target_oid'])
 c.git(c.BARE,'fsck','--full','--strict')
 bundle=c.STORE/'prepared-snapshots.bundle';c.git(c.BARE,'bundle','create',str(bundle),*[r['prepared_ref']for r in candidates])
 recovery=c.STORE/'independent-recovery.git';c.git(c.STORE,'init','--bare',str(recovery));c.git(recovery,'bundle','unbundle',str(bundle))
 for row in candidates:c.git(recovery,'update-ref',row['prepared_ref'],row['target_oid'])
 c.git(recovery,'fsck','--full','--strict')
 after=c.canonical_state();assert before==after, 'Canonical standalone state drifted during preparation'
 for ref,oid in source_refs.items():assert c.git(c.TEMPORAL,'rev-parse',ref).decode().strip()==oid
 result={'status':'PREPARED ONLY; no real receiving ref created or imported','input_bindings':c.inputs(),'new_agent_identity':identity,'custody_commit_timestamp':timestamp,
  'isolated_bare_store':str(c.BARE),'independent_recovery':str(recovery),'prepared_bundle':{'path':str(bundle),'sha256':c.sha(bundle.read_bytes()),'size':bundle.stat().st_size},
  'candidates':candidates,'variants':variants,'source_refs_before':source_refs,'source_refs_after':source_refs,
  'canonical_standalone_before':before,'canonical_standalone_after':after,'real_receiving_refs_created':0,'actual_transfer_or_activation':False,
  'custody_qualification':'039/075 raw working snapshots preserve unmerged stages via linked verified archive; new custody edges only. Earlier source author/history retained in original refs/bundles.',
  'support_disposition':'LICENSE included. Mixed monorepo history-support remains in canonical export/source history; no tracker/System/Scala support imported.'}
 c.write(c.HERE/'prepared-targets.json',result)
 print('Prepared',len(candidates),'targets',len(variants),'custody commits; isolated recovery complete',flush=True)

if __name__=='__main__':main()
