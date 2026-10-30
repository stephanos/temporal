#!/usr/bin/env python3
"""Capture/verify local R6 preparation without changing refs, source or Flow records."""
import argparse
import collections
import copy
import hashlib
import functools
import json
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent
TEMPORAL = next(path for path in HERE.parents if (path/'.flow/specs').is_dir())
REPOS = {'temporal': TEMPORAL, 'umpire': TEMPORAL.parent/'umpire'}
LEDGER = HERE.parent/'source/candidate-ledger.json'
RESEARCH = HERE/'r6-preparation-research.json'
ORIGINAL_PLAN = TEMPORAL/'.flow/artifacts/fn-257-extract-the-standalone-umpire/task6/provisional-transfer-plan.json'
ACTION = 'fn-258-complete-the-resumed-standalone-umpire.5'
PREPARED = {'umpire': '05473210b48c3db862f22051eb7154ce98258f98',
            'temporal': '3658582dac5b54c1f68c2eb54a312957c97e05b6'}
FW_OWNERS = {'framework', 'historical-framework'}
CURRENT_PREPARATION_TASKS = {
    ('temporal', 'refs/heads/work/fn2583-consumers'): 'fn-258-complete-the-resumed-standalone-umpire.3',
    ('umpire', 'refs/heads/work/fn2573-standalone'): 'fn-258-complete-the-resumed-standalone-umpire.2',
}  # Explicit current preparation units authorized by the root, not branch-number inference.


CURRENT_PREREQUISITES = [
            {'id': 'exact-current-pair-R5', 'state': 'root proof/review required; not established by this overlay'},
            {'id': 'dirty-untracked-unique-ignored-recovery', 'state': 'UNVERIFIED; required before retirement or full-preservation claim'},
            {'id': 'actual-framework-receiving-refs', 'state': 'UNCREATED/UNVERIFIED'},
            {'id': 'destination-access-and-history', 'state': 'reported SSO-blocked; no remote probe by this unit'},
            {'id': 'publication-and-fetchable-CI-pin', 'state': 'not performed; separate authorization/access required'},
            {'id': 'source-cutover-removal-authorization', 'state': 'not performed; in-tree source remains authoritative'},
            {'id': 'archive-cache-post-cutover-identity', 'state': 'separate root proof required'},
            {'id': 'carried-product-prerequisites', 'state': 'retain existing fn246/fn253/fn254/fn255/fn256/backend obligations; no closure inferred'}]


def load(path):
    return json.loads(path.read_bytes())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    assert path.parent.resolve() == HERE
    path.write_text(json.dumps(value, indent=2)+'\n')


def git(repository, *args, check=True):
    process = subprocess.run(['git', '--no-optional-locks', '--no-replace-objects', '-C', str(REPOS[repository]), *args],
                             capture_output=True)
    if check and process.returncode:
        raise RuntimeError(process.stderr.decode(errors='replace'))
    return process.stdout if check else process.returncode


class Facts:
    """Read-only Git observations; controls override observations only in memory."""
    def __init__(self):
        self.refs = {name: dict(line.split() for line in git(name, 'for-each-ref', '--format=%(refname) %(objectname)').decode().splitlines()) for name in REPOS}
        self.missing_objects = set()
        self.object_cache = {}
        self.containment = {}
        self.worktrees = {}
        for name in REPOS:
            rows = []
            for record in git(name, 'worktree', 'list', '--porcelain').decode().strip().split('\n\n'):
                fields = dict(line.split(' ', 1) for line in record.splitlines() if ' ' in line)
                rows.append(fields)
            self.worktrees[name] = rows

    def object_present(self, repo, oid):
        if not oid or (repo, oid) in self.missing_objects:
            return False
        if (repo, oid) not in self.object_cache:
            self.object_cache[repo, oid] = git(repo, 'cat-file', '-e', oid+'^{commit}', check=False) == 0
        return self.object_cache[repo, oid]

    def contains(self, repo, ref, oid):
        target = self.refs[repo].get(ref)
        key = (repo, target, oid)
        if not target or not self.object_present(repo, oid):
            return False
        if key not in self.containment:
            self.containment[key] = git(repo, 'merge-base', '--is-ancestor', oid, target, check=False) == 0
        return self.containment[key]


def bindings():
    return {name: {'path': str(path.relative_to(TEMPORAL)), 'sha256': sha(path)} for name, path in
            [('r1_ledger', LEDGER), ('owner_research', RESEARCH), ('fn257_task6_plan', ORIGINAL_PLAN)]}


@functools.lru_cache(maxsize=1)
def prior_plan_tasks():
    result = {}
    for item in load(ORIGINAL_PLAN)['entries']:
        result[item['candidate_id']] = [task['id'] for task in item.get('source_task_records', [])]
    return result


def tasks_for(entry, research):
    known = {item['id'] for item in entry['original_task_links']}
    extra = next((item for item in research['unknown_owner_research'] if item['original_ledger_index'] == entry['original_ledger_index']), None) if entry['original_ledger_index'] is not None else None
    if extra:
        known.update(item['id'] for item in extra['corroborated_task_records'])
    prior = prior_plan_tasks()
    known.update(prior.get(entry['repository']+':'+entry['ref'], prior.get(entry['ref'], [])))
    current_preparation = CURRENT_PREPARATION_TASKS.get((entry['repository'], entry['ref']))
    if current_preparation:
        known.add(current_preparation)
    preparation = entry.get('migration_preparation_tracking')
    if isinstance(preparation, dict) and preparation.get('id'):
        known.add(preparation['id'])
    return sorted(known), extra


def task_record(task_id):
    path = TEMPORAL/'.flow/tasks'/(task_id+'.json')
    if not path.is_file():
        raise ValueError('Authoritative task record absent: '+task_id)
    data = load(path)
    descriptor = path.with_suffix('.md')
    return {'id': task_id, 'repository': 'temporal', 'record': str(path.relative_to(TEMPORAL)),
            'record_sha256': sha(path), 'descriptor': str(descriptor.relative_to(TEMPORAL)),
            'descriptor_sha256': sha(descriptor), 'observed_status': data['status'],
            'declared_dependencies': data.get('depends_on', []),
            'acceptance_claim': False}


def framework_paths(entry, ledger):
    selected = {}
    for observation, changes in [('committed-delta', entry['committed_changes']), ('dirty-observation-not-backup', entry['dirty_changes'])]:
        for change in changes:
            path = change['path']
            mapping = ledger['path_mappings'].get(path)
            if mapping and mapping['ownership'] in FW_OWNERS:
                record = selected.setdefault(path, {'source_path': path, 'destination_path': mapping['path'],
                    'mapping_ownership': mapping['ownership'], 'mapping_status': mapping['mapping_status'],
                    'review_state': 'provisional-path-only; inherited assemblies and exact hunks unreviewed', 'observations': []})
                record['observations'].append({'kind': observation, 'status': change['status']})
    return [selected[path] for path in sorted(selected)]


def source_recovery(entry, facts):
    repo = entry['repository']
    oid = entry['original_head'] or entry['observed_head']
    ref = entry['ref']
    observed = entry['observed_head']
    def covers(name):
        return facts.contains(repo, name, oid) and (observed is None or facts.contains(repo, name, observed))
    candidates = [ref] if covers(ref) else []
    if not candidates:
        candidates = [name for name in sorted(facts.refs[repo]) if name.startswith('refs/recovery/') and covers(name)]
    return {'required_original_or_observed_object': oid,
            'original_ref_exists_now': ref in facts.refs[repo],
            'actual_current_ref_head': facts.refs[repo].get(ref),
            'recovery_refs': [{'repository': repo, 'ref': name, 'observed_target': facts.refs[repo][name]} for name in candidates],
            'object_verified': facts.object_present(repo, oid),
            'committed_snapshot_reachable_verified': bool(candidates) and facts.object_present(repo, oid),
            'r1_observed_head_required': observed is not None,
            'r1_observed_head_object_verified': facts.object_present(repo, observed) if observed is not None else None,
            'scope': 'Committed source objects/history only; no dirty/untracked/unique ignored backup claim.'}


def build(ledger, research, facts):
    entries = []
    for index, entry in enumerate(ledger['entries']):
        ids, owner = tasks_for(entry, research)
        scope = framework_paths(entry, ledger)
        recovery = source_recovery(entry, facts)
        freeze = bool(owner and owner['classification'] == 'unresolved-product-task-freeze')
        receiver = None
        if entry['repository'] == 'temporal' and scope:
            ref = f'refs/recovery/fn258/candidate-{index:03d}'
            receiver = {'repository': 'umpire', 'planned_ref': ref,
                'state': 'existing-unverified' if ref in facts.refs['umpire'] else 'UNCREATED/UNVERIFIED',
                'actual_ref_exists': ref in facts.refs['umpire'], 'target': None,
                'creation_performed': False, 'receiving_verified': False,
                'method': 'Review exact framework-only snapshot/hunks or independently filtered history; never import whole monorepo commits or activate main.',
                'scope_path_count': len(scope)}
        checkout_refs = [item for item in facts.worktrees[entry['repository']] if item.get('branch') == entry['ref']]
        category = ('planned-framework-receiver-with-source-retained' if receiver else
                    'existing-standalone-preparation-only' if entry['repository'] == 'umpire' else
                    'source-only-owner-freeze' if freeze else 'source-only-retention-proposed')
        entries.append({'candidate_id': f"{entry['repository']}:{entry['ref']}", 'ledger_entry_index': index,
            'original_ledger_index': entry['original_ledger_index'], 'repository': entry['repository'], 'ref': entry['ref'],
            'original_head': entry['original_head'], 'r1_observed_head': entry['observed_head'],
            'source_ownership': entry['ownership'], 'ownership_basis': 'Immutable R1 classification; provisional hunk ownership remains unresolved.',
            'disposition_category': category, 'source_recovery': recovery,
            'framework_scope': scope, 'retained_nonframework_change_count': sum(change['path'] not in {item['source_path'] for item in scope} for change in entry['committed_changes']+entry['dirty_changes']),
            'planned_receiver': receiver,
            'existing_standalone_preparation_ref': {'ref': entry['ref'], 'actual_head': facts.refs['umpire'].get(entry['ref']), 'object_verified': facts.object_present('umpire', entry['observed_head']), 'claims_original_candidate_transfer': False} if entry['repository'] == 'umpire' else None,
            'authoritative_task_links': [task_record(task_id) for task_id in ids],
            'task_link_basis': 'R1/task6 provenance plus explicit commit-body/exact-descriptor research and root-authorized current preparation units; never numeric branch attribution.',
            'owner_research_classification': owner['classification'] if owner else None,
            'unresolved_product_owner_freeze': freeze,
            'keeper': {'repository': entry['repository'], 'source_ref': entry['ref'], 'action_task': ACTION,
                       'authoritative_action_record': '.flow/tasks/'+ACTION+'.json',
                       'action': 'Retain exact original/source recovery; resolve unknown owner explicitly before accepting disposition.' if freeze else 'Review receiving/retention, exhaustive hunks and recovery evidence; keep product obligations open.'},
            'cross_repository_prerequisites': entry['cross_repository_prerequisites'],
            'historical_evidence_paths': entry['historical_evidence_paths'],
            'worktrees_observed_now': checkout_refs, 'r1_dirty_path_count': len(entry['dirty_changes']),
            'dirty_data_recovery': {'state': 'UNVERIFIED', 'backup_manifest': None,
                'unique_ignored_source_inventory': 'not performed; absence of dirty status is not proof', 'fully_preserved_candidate': False},
            'committed_source_only_freeze_preserved': freeze and recovery['committed_snapshot_reachable_verified'] and not scope,
            'disposition_review_accepted': False, 'r6_disposition_complete': False, 'transferred': False})
    prepared = []
    for repo, oid in PREPARED.items():
        prepared.append({'repository': repo, 'head': oid, 'object_verified': facts.object_present(repo, oid),
            'actual_refs_containing_head': [ref for ref in facts.refs[repo] if ref.startswith('refs/heads/') and facts.contains(repo, ref, oid)],
            'scope': 'Current local prepared source candidate only; no publication, R5 gate or original-candidate transfer claim.'})
    return {'schema_version': 1, 'status': 'local-preparation-only; R6 INCOMPLETE', 'inputs': bindings(),
        'frozen_source_commit': ledger['source_commit'], 'entries': entries, 'current_prepared_receiving_heads': prepared,
        'authoritative_tracking': task_record(ACTION),
        'current_prerequisites': copy.deepcopy(CURRENT_PREREQUISITES),
        'historical_prerequisites': ledger['historical_prerequisites'],
        'historical_prerequisite_qualification': 'Original validation-hold is historical provenance, not a current no-execution rule.',
        'r6_complete': False, 'source_authority': 'Temporal in-tree until actual authorized cutover',
        'scope': 'Read-only Git recovery observations and proposed local dispositions. No ref creation, transfer, filtering, builds, Flow mutation, publication or cutover.'}


def validate(overlay, ledger, research, facts):
    errors, drift = [], []
    def require(condition, kind, detail=None):
        if not condition:
            errors.append({'kind': kind, 'detail': detail})
    require(overlay.get('inputs') == bindings(), 'changed-input-binding')
    require(sha(LEDGER) == research['inputs']['renewed_ledger']['sha256'] and sha(ORIGINAL_PLAN) == research['inputs']['original_task6_plan']['sha256'], 'changed-research-base-input')
    expected_keys = [f"{entry['repository']}:{entry['ref']}" for entry in ledger['entries']]
    rows = overlay.get('entries', [])
    require(len(rows) == len(expected_keys) == 88 and collections.Counter(row.get('candidate_id') for row in rows) == collections.Counter(expected_keys), 'candidate-coverage')
    by_key = {row.get('candidate_id'): row for row in rows}
    for index, source in enumerate(ledger['entries']):
        key = expected_keys[index]
        row = by_key.get(key)
        if row is None:
            continue
        require(all(row.get(field) == source.get(source_field) for field, source_field in [('repository','repository'),('ref','ref'),('original_ledger_index','original_ledger_index'),('original_head','original_head'),('r1_observed_head','observed_head'),('source_ownership','ownership')]), 'unsupported-owner-or-identity', key)
        require(row.get('ledger_entry_index') == index, 'unsupported-owner-or-identity', key)
        expected_scope = framework_paths(source, ledger)
        require(row.get('framework_scope') == expected_scope, 'unsupported-path-mapping', key)
        expected_retained = sum(change['path'] not in {item['source_path'] for item in expected_scope} for change in source['committed_changes']+source['dirty_changes'])
        require(row.get('retained_nonframework_change_count') == expected_retained, 'unsupported-path-mapping', key)
        ids, owner = tasks_for(source, research)
        links = row.get('authoritative_task_links', [])
        require([link.get('id') for link in links] == ids, 'invented-or-omitted-task-link', key)
        for link in links:
            if link.get('id') not in ids:
                continue
            current = task_record(link['id'])
            require(link.get('repository') == 'temporal' and link.get('record') == current['record'] and link.get('descriptor') == current['descriptor'] and link.get('acceptance_claim') is False, 'unsupported-tracker-owner', key)
            require(link.get('declared_dependencies') == current['declared_dependencies'] and link.get('descriptor_sha256') == current['descriptor_sha256'], 'stale-or-omitted-task-prerequisite', key)
            if link != current:
                drift.append({'candidate_id':key,'task_record_drift':link['id']})
        freeze = bool(owner and owner['classification'] == 'unresolved-product-task-freeze')
        require(row.get('unresolved_product_owner_freeze') == freeze, 'unsupported-owner-freeze', key)
        require(row.get('keeper', {}).get('action_task') == ACTION and row.get('keeper', {}).get('source_ref') == source['ref'] and row.get('keeper', {}).get('repository') == source['repository'], 'unsupported-keeper', key)
        require(row.get('cross_repository_prerequisites') == source['cross_repository_prerequisites'], 'prerequisite-omitted', key)
        require(row.get('transferred') is False and row.get('r6_disposition_complete') is False and row.get('disposition_review_accepted') is False, 'false-transfer-or-completion-claim', key)
        require(row.get('dirty_data_recovery', {}).get('fully_preserved_candidate') is False and row.get('dirty_data_recovery', {}).get('state') == 'UNVERIFIED' and row.get('dirty_data_recovery', {}).get('backup_manifest') is None, 'false-dirty-recovery-claim', key)
        recovery = row.get('source_recovery', {})
        repo = source['repository']
        oid = source['original_head'] or source['observed_head']
        require(recovery.get('required_original_or_observed_object') == oid and facts.object_present(repo, oid) and (source['observed_head'] is None or facts.object_present(repo, source['observed_head'])), 'missing-original-recovery-object', key)
        refs = recovery.get('recovery_refs', [])
        require(bool(refs), 'missing-original-ref-or-recovery', key)
        for ref in refs:
            require(ref.get('repository') == repo and facts.contains(repo, ref.get('ref'), oid) and (source['observed_head'] is None or facts.contains(repo, ref.get('ref'), source['observed_head'])), 'missing-original-ref-or-recovery', key)
            actual = facts.refs[repo].get(ref.get('ref'))
            if actual != ref.get('observed_target'):
                drift.append({'candidate_id':key,'recovery_ref':ref.get('ref'),'observed':ref.get('observed_target'),'current':actual})
        require(recovery.get('object_verified') is True and recovery.get('committed_snapshot_reachable_verified') is True and recovery.get('r1_observed_head_required') == (source['observed_head'] is not None) and recovery.get('r1_observed_head_object_verified') == (True if source['observed_head'] is not None else None), 'unsupported-recovery-observation', key)
        need_receiver = repo == 'temporal' and bool(expected_scope)
        expected_category = ('planned-framework-receiver-with-source-retained' if need_receiver else 'existing-standalone-preparation-only' if repo == 'umpire' else 'source-only-owner-freeze' if freeze else 'source-only-retention-proposed')
        require(row.get('disposition_category') == expected_category, 'unsupported-disposition-category', key)
        receiver = row.get('planned_receiver')
        require(bool(receiver) == need_receiver, 'missing-or-unneeded-receiver-plan', key)
        if receiver:
            require(receiver.get('repository') == 'umpire' and receiver.get('planned_ref') == f'refs/recovery/fn258/candidate-{index:03d}' and receiver.get('receiving_verified') is False and receiver.get('creation_performed') is False and receiver.get('target') is None, 'false-receiving-ref-claim', key)
            exists = receiver.get('planned_ref') in facts.refs['umpire']
            require(receiver.get('state') == ('existing-unverified' if exists else 'UNCREATED/UNVERIFIED') and receiver.get('actual_ref_exists') == exists and receiver.get('scope_path_count') == len(expected_scope), 'false-receiving-ref-claim', key)
        if repo == 'umpire':
            existing = row.get('existing_standalone_preparation_ref') or {}
            require(existing.get('ref') == source['ref'] and existing.get('claims_original_candidate_transfer') is False and existing.get('object_verified') is True and facts.contains(repo, source['ref'], existing.get('actual_head')), 'unsupported-existing-preparation-ref', key)
        require(row.get('committed_source_only_freeze_preserved') == (freeze and bool(refs) and not expected_scope), 'false-source-only-preservation-scope', key)
    require(overlay.get('historical_prerequisites') == ledger['historical_prerequisites'], 'prerequisite-omitted')
    require(overlay.get('current_prerequisites') == CURRENT_PREREQUISITES, 'unsupported-current-proof-claim')
    require(overlay.get('r6_complete') is False and overlay.get('status') == 'local-preparation-only; R6 INCOMPLETE', 'false-overall-R6-completion')
    require(overlay.get('authoritative_tracking', {}).get('id') == ACTION and overlay.get('authoritative_tracking', {}).get('repository') == 'temporal', 'unsupported-action-tracker')
    prepared = overlay.get('current_prepared_receiving_heads', [])
    require(len(prepared) == 2 and {item.get('repository'):item.get('head') for item in prepared} == PREPARED, 'unsupported-current-prepared-head')
    for item in prepared:
        repo, oid = item['repository'], item['head']
        require(item.get('object_verified') is True and facts.object_present(repo, oid) and bool(item.get('actual_refs_containing_head')), 'missing-current-prepared-recovery')
        for ref in item.get('actual_refs_containing_head', []):
            require(facts.contains(repo, ref, oid), 'missing-current-prepared-recovery', ref)
    return {'result':'pass' if not errors else 'fail','errors':errors,'live_ref_or_task_drift':drift,
            'verified_entry_count':len(rows),'r6_complete':False,
            'qualification':'Frozen observations can drift while verified recovery remains reachable. No transfer/backup/activation/completion claim.'}


def negative_controls(overlay, ledger, research, facts):
    outputs = []
    def run(name, mutation, expected, fact_mutation=None):
        candidate = copy.deepcopy(overlay)
        mutation(candidate)
        probe = copy.copy(facts)
        probe.refs = copy.deepcopy(facts.refs)
        probe.missing_objects = set(facts.missing_objects)
        if fact_mutation:
            fact_mutation(probe)
        result = validate(candidate, ledger, research, probe)
        passed = result['result'] == 'fail' and expected in {item['kind'] for item in result['errors']}
        outputs.append({'name':name,'expected_refusal':expected,'passed':passed,'failure_kinds':sorted({item['kind'] for item in result['errors']})})
    run('missing-original-entry', lambda value:value['entries'].pop(0), 'candidate-coverage')
    run('duplicate-entry', lambda value:value['entries'].append(copy.deepcopy(value['entries'][0])), 'candidate-coverage')
    run('changed-input-hash', lambda value:value['inputs']['r1_ledger'].update(sha256='0'*64), 'changed-input-binding')
    run('unsupported-source-owner', lambda value:value['entries'][0].update(source_ownership='framework'), 'unsupported-owner-or-identity')
    run('invented-existing-task-link', lambda value:value['entries'][0]['authoritative_task_links'].append(task_record('fn-258-complete-the-resumed-standalone-umpire.2')), 'invented-or-omitted-task-link')
    run('false-original-transfer', lambda value:value['entries'][0].update(transferred=True,r6_disposition_complete=True), 'false-transfer-or-completion-claim')
    run('false-dirty-backup', lambda value:value['entries'][0]['dirty_data_recovery'].update(fully_preserved_candidate=True), 'false-dirty-recovery-claim')
    missing_ref = overlay['entries'][0]['ref']
    run('missing-original-ref-without-recovery', lambda value:None, 'missing-original-ref-or-recovery', lambda probe:probe.refs['temporal'].pop(missing_ref))
    oid = overlay['entries'][0]['original_head']
    run('missing-original-object', lambda value:None, 'missing-original-recovery-object', lambda probe:probe.missing_objects.add(('temporal',oid)))
    receiver_index = next(index for index,row in enumerate(overlay['entries']) if row['planned_receiver'])
    run('planned-ref-claimed-verified', lambda value:value['entries'][receiver_index]['planned_receiver'].update(receiving_verified=True,state='verified'), 'false-receiving-ref-claim')
    run('unsupported-path-owner', lambda value:value['entries'][receiver_index]['framework_scope'][0].update(destination_path='__invented__.rs'), 'unsupported-path-mapping')
    run('unknown-owner-freeze-cleared', lambda value:value['entries'][0].update(unresolved_product_owner_freeze=False), 'unsupported-owner-freeze')
    run('omitted-cross-repository-prerequisite', lambda value:value['entries'][0]['cross_repository_prerequisites'].pop(), 'prerequisite-omitted')
    run('false-current-pair-proof', lambda value:value['current_prerequisites'][0].update(state='verified'), 'unsupported-current-proof-claim')
    run('false-overall-R6-completion', lambda value:value.update(r6_complete=True), 'false-overall-R6-completion')
    return {'result':'pass' if all(item['passed'] for item in outputs) else 'fail','controls':outputs,
            'scope':'In-memory overlay and Git-observation mutations only; no Git writes, source mutation or transfers.'}


def markdown(overlay):
    rows = overlay['entries']
    planned = [row for row in rows if row['planned_receiver']]
    freezes = [row for row in rows if row['unresolved_product_owner_freeze']]
    lines = ['# Local R6 disposition overlay', '',
        'Status: **R6 incomplete**. All 88 recorded entries and 81 original identities are retained. This overlay creates no receiving refs and accepts no transfer or disposition.', '',
        f"There are {len(planned)} planned framework receiving refs. Their full provisional path mappings, committed/dirty provenance and source recovery objects are in disposition-overlay.json. These are path plans; selective hunk ownership and inherited assembly deduplication remain unreviewed. All receivers are UNCREATED/UNVERIFIED. Current prepared heads are standalone 05473210 and consumer 3658582d; their existing refs recover preparation only.", '',
        'Authoritative historical/current tracker records remain in Temporal, with fn-258.5 owning the migration action. The 15 direct task-trailer and two descriptor associations are carried without exclusive ownership or product-completion claims. Existing source refs recover committed history; dirty/untracked/unique ignored backups remain unverified.', '',
        '## Receiving-ref creation work', '', '| Planned neutral ref | Original/source candidate | Provisional source paths |', '| --- | --- | ---: |']
    for row in planned:
        lines.append(f"| `{row['planned_receiver']['planned_ref']}` | `{row['ref']}` | {len(row['framework_scope'])} |")
    lines += ['', 'Do not create these refs directly from monorepo commits. Review selected/excluded paths and exact hunks, preserve dirty/ignored bytes, construct and verify an isolated normalized snapshot or filtered history, then create the neutral recovery ref at its reviewed receiving commit. Keep source authored history under original refs. Snapshot recovery is not full filtered-history preservation or activation.', '',
        '## Source-only owner freezes', '', 'Each freeze names its source ref/verified committed recovery and keeper action fn-258.5. None is accepted as a transfer.']
    for row in freezes:
        lines.append(f"- `{row['ref']}`: committed snapshot reachable; no product task guessed. Whole-candidate preservation remains unverified until dirty/ignored recovery is resolved or explicitly evidenced absent.")
    lines += ['', 'These source-only experiments can receive an accepted retention disposition without implementing deferred features, after recovery scope and tracker/keeper decisions are reviewed. This unit proves committed recovery only; it does not silently complete their preservation/disposition.', '',
        'Run `python3 disposition-overlay.py --capture` in this directory to regenerate the overlay, or `python3 disposition-overlay.py` to verify it and run isolated controls. Both commands write only this preparation directory. Original R1 and fn-257 inputs remain unchanged.', '',
        'Destination SSO/access, actual publication/history inspection, fetchable CI pins, exact-pair gates, source-removal authorization and post-cutover identities remain separate root-owned requirements. In-tree source remains authoritative.']
    return '\n'.join(lines)+'\n'


def receiving_work(overlay, overlay_hash):
    planned = [row for row in overlay['entries'] if row['planned_receiver']]
    return {
            'status': 'PLANNED ONLY; no ref creation or receiving validation performed',
            'overlay_sha256': overlay_hash, 'planned_ref_count': len(planned),
            'source_path_occurrences': sum(len(row['framework_scope']) for row in planned),
            'unique_source_paths': len({item['source_path'] for row in planned for item in row['framework_scope']}),
            'work': [{'candidate_id': row['candidate_id'], 'receiver': row['planned_receiver'],
                      'source_original_head': row['original_head'], 'source_r1_observed_head': row['r1_observed_head'],
                      'source_recovery': row['source_recovery'], 'provisional_path_mappings': row['framework_scope'],
                      'required_before_create': ['review inherited assemblies and exact hunks', 'verify dirty/untracked/unique ignored recovery', 'construct isolated framework-only receiving tree/history', 'independently verify bytes/modes and source/destination accounting', 'review target commit and then create neutral recovery ref; no main activation']} for row in planned]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--capture', action='store_true', help='Regenerate only this new preparation overlay; never original R1 inputs')
    args = parser.parse_args()
    ledger, research = load(LEDGER), load(RESEARCH)
    assert len(ledger['entries']) == 88
    facts = Facts()
    path = HERE/'disposition-overlay.json'
    if args.capture:
        overlay = build(ledger, research, facts)
        save(path, overlay)
        (HERE/'disposition-overlay.md').write_text(markdown(overlay))
        save(HERE/'receiving-ref-creation-work.json', receiving_work(overlay, sha(path)))
    else:
        overlay = load(path)
    result = validate(overlay, ledger, research, facts)
    controls = negative_controls(overlay, ledger, research, facts)
    derived_work_ok = load(HERE/'receiving-ref-creation-work.json') == receiving_work(overlay, sha(path))
    derived_md_ok = (HERE/'disposition-overlay.md').read_text() == markdown(overlay)
    if not derived_work_ok or not derived_md_ok:
        result['errors'].append({'kind':'stale-or-modified-derived-evidence'})
        result['result'] = 'fail'
    result.update({'overlay_sha256':sha(path),'script_sha256':sha(Path(__file__)), 'input_bindings':bindings(),
                   'negative_controls_result':controls['result'],
                   'derived_receiving_work_verified':derived_work_ok, 'derived_markdown_verified':derived_md_ok,
                   'verified_original_candidate_count':sum(row['original_ledger_index'] is not None for row in overlay['entries']),
                   'verified_source_recovery_entries':sum(row['source_recovery']['committed_snapshot_reachable_verified'] for row in overlay['entries']),
                   'original_candidate_transfers':sum(row['transferred'] for row in overlay['entries'] if row['original_ledger_index'] is not None),
                   'r6_completed_dispositions':sum(row['r6_disposition_complete'] for row in overlay['entries']),
                   'fully_preserved_candidates':sum(row['dirty_data_recovery']['fully_preserved_candidate'] for row in overlay['entries']),
                   'planned_receiving_refs':sum(bool(row['planned_receiver'])for row in overlay['entries']),
                   'source_only_freezes':sum(row['unresolved_product_owner_freeze']for row in overlay['entries'])})
    save(HERE/'disposition-overlay-verification.json', result)
    save(HERE/'disposition-overlay-negative-controls.json', controls)
    print(json.dumps({key:value for key,value in result.items() if key not in ['input_bindings','live_ref_or_task_drift']},indent=2))
    if result['result'] != 'pass' or controls['result'] != 'pass':
        raise SystemExit(1)


if __name__ == '__main__':
    main()
