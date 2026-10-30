#!/usr/bin/env python3
"""Bind current readiness only; historical overlays, source and lifecycle are never rewritten."""
import datetime
import hashlib
import json
from pathlib import Path
import subprocess

HERE=Path(__file__).resolve().parent
TEMPORAL=next(p for p in HERE.parents if (p/'.flow/specs').is_dir())
ROOT=TEMPORAL.parent
PAIR={'temporal':'cde1ee67efd266a058193694c2ddb287598b9ea5','umpire':'05473210b48c3db862f22051eb7154ce98258f98'}
HISTORICAL_FOCUSED_PAIR={'temporal':'3f5f613578f2857b738a7792fad08f12b3ea0a41','umpire':'05473210b48c3db862f22051eb7154ce98258f98'}
def read(p):return json.loads(p.read_bytes())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def git(repo,*args):return subprocess.check_output(['git','--no-optional-locks','--no-replace-objects','-C',str(repo),*args])
archive=HERE/'historical-preparation-3658582'
for record in read(archive/'preservation-manifest.json')['files']:
 assert sha(archive/record['file'])==record['sha256']
assert git(TEMPORAL,'rev-parse','refs/heads/work/fn2583-consumers').decode().strip()==PAIR['temporal']
assert git(ROOT/'umpire','rev-parse','HEAD').decode().strip()==PAIR['umpire']
worktree=TEMPORAL/'.worktrees/fn2583-consumers'
assert git(worktree,'rev-parse','HEAD').decode().strip()==PAIR['temporal']
tracker_observations=read(HERE/'current-tracker-observations.json')['observations']
requested_tasks=[row['requested_task']for row in tracker_observations]
assert len(requested_tasks)==len(set(requested_tasks))
tracker_count=len(set(requested_tasks))
status=git(worktree,'status','--porcelain=v1','--untracked-files=all').decode()
assert not status
closures=read(HERE/'source-readiness-closures.json')
observations={item['id']:item for item in closures['observations']}
assert all(observations['fn-258-complete-the-resumed-standalone-umpire.'+str(n)]['status']=='done'for n in [1,2,3])
assert observations['fn-258-complete-the-resumed-standalone-umpire.4']['status']=='in_progress'
focused_path=HERE.parent/'consumer/current-pair-3f5-054/receipt.json'
focused=read(focused_path)
assert focused['status']=='PASS'and focused['pair']==HISTORICAL_FOCUSED_PAIR and focused['failures']==0
assert focused['clean_before_checkout']and not any(focused['clean_after_checks'].values())
checks=[]
for record in focused['checks']:
 path=focused_path.parent/record['log'];assert record['exit']==0 and sha(path)==record['sha256']
 checks.append({'name':record['name'],'exit':0,'log':str(path.relative_to(TEMPORAL)),'sha256':sha(path)})
workspace_path=focused_path.parent/focused['workspace_observation']
workspace=read(workspace_path)
assert workspace['observed_pair']==HISTORICAL_FOCUSED_PAIR and workspace['all_dirs_verified'] is True and workspace['no_workspace_overwrite'] is True
assert workspace['workspace_sha256_before']==workspace['workspace_sha256_after'] and len(workspace['modules'])==3
persistent=[]
for number in [1,2,3]:
 path='.flow/tasks/fn-258-complete-the-resumed-standalone-umpire.'+str(number)+'.md'
 data=git(TEMPORAL,'show','HEAD:'+path);assert b'## Done summary'in data and b'## Evidence'in data
 persistent.append({'task':observations['fn-258-complete-the-resumed-standalone-umpire.'+str(number)]['id'],
  'path':path,'committed_blob':git(TEMPORAL,'rev-parse','HEAD:'+path).decode().strip(),
  'sha256':hashlib.sha256(data).hexdigest(),'latest_path_commit':git(TEMPORAL,'log','-1','--format=%H','--',path).decode().strip()})
overlay=read(HERE/'disposition-overlay.json');verify=read(HERE/'disposition-overlay-verification.json')
assert verify['result']=='pass'and verify['planned_receiving_refs']==48 and not overlay['r6_complete']
assert all(row['planned_receiver']['state']=='UNCREATED/UNVERIFIED'for row in overlay['entries']if row['planned_receiver'])
snapshot=read(HERE/'receiving-snapshots/verification.json')
assert snapshot['result']=='pass'and snapshot['verified_targets']==48 and snapshot['real_receiving_refs_created']==0
old=read(archive/'receiving-readiness.json')
result={'status':'CURRENT source cde/054 observation; historical3f5 focused readiness; task4 active; R6 incomplete','observed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),
 'current_pair':PAIR,'consumer_worktree_clean_now':True,'consumer_worktree':str(worktree),
 'source_readiness_closures':{'path':'source-readiness-closures.json','sha256':sha(HERE/'source-readiness-closures.json'),'source':'flowctl show --json; status_source flow-state',
  'task1':'done','task2':'done source-readiness','task3':'done source-readiness at historical3f5; currentcde harness correction owned by fn258.4','task4':'in_progress broad pair/current-input freshness proof','persistent_committed_summaries':persistent},
 'tracker_statuses':{'path':'current-tracker-observations.json','sha256':sha(HERE/'current-tracker-observations.json'),'unique_task_count':tracker_count,
  'definition_vs_current_differences':[{'id':link['id'],'definition_only':link['definition_status'],'current_observed':link['observed_status'],'status_source':link['status_source']} for link in {link['id']:link for row in overlay['entries'] for link in row['authoritative_task_links']}.values() if link['definition_status']!=link['observed_status']],
  'definition_statuses':'Definition-only, not authoritative lifecycle. Current overlay uses captured flowctl statuses; e.g fn254.17 is done.'},
 'historical_focused_pair':{'receipt':str(focused_path.relative_to(TEMPORAL)),'sha256':sha(focused_path),'status':'PASS','unit_tests':focused['unit_tests'],'integration_tests':focused['integration_tests'],
  'pair':HISTORICAL_FOCUSED_PAIR,'historical_only':True,'checks':checks,'guest_regenerated':focused['guest_regenerated'],'guest_freshness_qualification':focused['guest_freshness'],
  'workspace_observation':{'path':str(workspace_path.relative_to(TEMPORAL)),'sha256':sha(workspace_path),'all3_module_dirs_verified':True,'parent_workspace_not_overwritten':True},
  'scope':'Focused all-target/all-feature compilation,16unit+7integration,Makeguard and parent-Go-workspace observations only. No broad task4/guest-freshness/activation claim.'},
 'current_source_qualification':{'owner':'fn-258-complete-the-resumed-standalone-umpire.4','scope':'cde integration-harness correction only; task3 remains historically closed at3f5','new_focused_Go_workspace_or_wholegate_claim':False,'new_fresh_pair_emptytarget_proof':'PENDING','prior_full_paired_default':'3f5/054 FAILED exit2; preserved under validation/final-pair-3f5-054/failed-default-3f5'},
 'receiving_state':{'actual_real_refs':'48 UNCREATED/UNVERIFIED','isolated_snapshot_targets':48,'new_custody_commits':50,
  'proof':'receiving-snapshots/verification.json','proof_sha256':sha(HERE/'receiving-snapshots/verification.json'),'source_only_ownership_correction':'Exact four files in049; source retained, original maps unchanged',
  'real_transfers':0,'R6_complete':False},
 'custody':{'manifest':'custody/custody-manifest.json','verification':'custody/custody-verification.json','manifest_sha256':sha(HERE/'custody/custody-manifest.json'),'verification_sha256':sha(HERE/'custody/custody-verification.json'),
  'scope':'039/075 raw/index custody and six source-only committed freezes verified; historical ignored uncertainty retained, no blanket full preservation/accepted disposition.'},
 'dated_supersession':{'previous_observed_utc':old['observed_utc'],'historical_readiness':'historical-preparation-3658582/receiving-readiness.json',
  'historical_sha256':sha(archive/'receiving-readiness.json'),'previous_statement':'consumer365 worktree had7 pending tracked missing-sibling/docs/wiring edits',
  'current_observation':'superseded first by committed3f5f613, now cde1ee67 with ownHEAD/ref/clean checked; historical bytes retained; historical bytes retained exactly, not relabeled'},
 'historical_snapshot_inputs':'receiving-snapshots/*.input.json remain exact old overlay/plan/custody inputs; prepared50 commit/tree OIDs unchanged by current-only refresh',
 'open_boundaries':['task4 broad proof/current-input guestfreshness active','root review and explicit actual import/ref admission','destination SSO/access/history inspection/publication/fetchable CI activation','source-cutover/removal authorization and post-cutover proof']}
(HERE/'receiving-readiness.json').write_text(json.dumps(result,indent=2)+'\n')
(HERE/'receiving-readiness.md').write_text(f'''# Current local preparation readiness

Current exact source pair: consumer `cde1ee67efd266a058193694c2ddb287598b9ea5`, framework `05473210b48c3db862f22051eb7154ce98258f98`. The consumer worktree is clean. This supersedes the earlier dated observation of seven pending tracked edits at365; exact old overlay/readiness bytes remain under `historical-preparation-3658582/`. Historical reports were not retroactively rewritten.

Tasks1/2/3 are DONE in captured authoritative `flowctl show` state. Task2/3 closures are source-readiness only. Their committed markdown Done summaries/evidence are bound in the JSON. Immutable definition JSON todo fields are definition-only; the current overlay uses {tracker_count} unique actual CLI observations, including fn254.17 DONE, without inventing task completion from candidate snapshots.

The historical clean3f5/054 focused receipt passed all-target/all-feature compilation,16unit and7integration controls, Make sibling guards and automatic parent Go workspace checks. Its logs/hashes are verified as historical evidence; no currentcde Go-workspace or whole-gate claim follows. The earlier full paired default at3f5/054 FAILED with exit2 and remains preserved. Currentcde is the task4-owned integration-harness correction; fresh empty-target whole-pair proof remains pending. This is not task4 broad proof or current-input guest freshness: task4 remains in_progress, and its outstanding freshness scope is explicit.

All48 real receiving refs remain UNCREATED/UNVERIFIED. The48 proposed custody targets/50 commits have independent isolated snapshot proof; none was imported or activated in the real standalone.039/075 dirty/index custody and six source-only committed freezes are separately verified, with historical ignored-state uncertainty retained. There are zero transfers or completed R6 dispositions.

Source-only experimental retention needs no deferred feature implementation. Actual repository/ref/tracker/prerequisite acceptance remains a later root decision. Root review/task4 acceptance precede real import or create-only ref transaction. SSO/access, publication/CI activation, source-removal authorization and post-cutover proof remain open.

`receiving-snapshots/*.input.json` preserve the historical overlay/plan/custody bytes used for snapshot preparation. Their locators explicitly name frozen files and retain original paths separately. The current-only365→3f5 refresh does not alter any of the50 prepared tree/commit OIDs. R1, fn257 plans, original source, refs and lifecycle state remain unchanged by this unit.
''')
print('Current readiness refreshed: cde/054 ownHEAD/ref/clean observed; dated3f5focusedPASS; source closures DONE,unique CLI statuses; task4active/R6incomplete; historical7pending observation explicitly superseded.')
