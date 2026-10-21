"""Independent raw-wire authentication for actual public exploration campaigns."""
import copy
import json
import re
from pathlib import Path
import runpy
import shutil

seed = runpy.run_path(str(Path(__file__).with_name('campaign-check.py')))
parity, equal = seed['parity'], seed['equal']
canonical_file, authenticate_artifact = seed['canonical_file'], seed['authenticate_artifact']


def domain(name, value):
    return 'sha256:' + parity.digest(name.encode() + b'\0' + parity.canonical(value))


def initial_state(kind, config):
    """Hash the admitted root only; intermediate states are commitments, not exposed snapshots."""
    common = dict(config=config, outcomes=[], failure_signatures=[], logical_executions=0,
                  committed_rounds=0, omitted_by_execution_bound=0, omitted_by_depth=0,
                  omitted_by_capacity=0)
    if kind == 'choice':
        identity = config['execution']
        projection = dict(target_sha256='sha256:' + bytes(identity['TargetSHA256']).hex(),
            toolchain_build_key=identity['ToolchainBuildKey'], goos=identity['GOOS'], goarch=identity['GOARCH'],
            implementation_sha256='sha256:' + bytes(identity['ImplementationSHA256']).hex(),
            controller_sha256=config['controller_sha256'], base_seed=str(config['base_seed']), decisions=[])
        encoded = json.dumps(projection, separators=(',', ':'), ensure_ascii=False).encode()
        sha = 'sha256:' + parity.digest(b'gomad-choice-exploration-candidate/v1\0' + encoded)
        root = dict(sha256=sha, parent_sha256='', source_trace_sha256='', forced_depth=0,
                    prefix_sha256='', prefix_bytes=[], source_trace=[], rank=None)
        pending = len(json.dumps(dict(sha256=sha, forced_depth=0), separators=(',', ':')).encode()) + 1
        state = dict(common, queue=[root], seen=[sha], pending_bytes=pending, deepest_prefix=0,
                     omitted_by_select_readiness=0, stop_reason='')
        if 'site_scope' in config:
            state['site_scope_resolution'] = dict(packages={}, fingerprints=[])
        return domain('gomad-choice-exploration-state/v2', state)
    projection = {key: config[key] for key in ('base_seed', 'controller_sha256', 'execution_sha256')}
    sha = domain('gomad-simulation-exploration-candidate/v1', dict(projection, overrides=None))
    root = dict(sha256=sha, overrides=None)
    state = dict(common, queue=[root], seen=[sha], deepest_override=0, omitted_by_dimension=0,
                 pending_bytes=len(parity.canonical(dict(sha256=sha, overrides=None))))
    return domain('gomad-simulation-exploration-state/v1', state)


def snapshot(root):
    return {str(path.relative_to(root)): parity.digest(path.read_bytes())
            for path in root.rglob('*') if path.is_file()}


def artifact_view(root):
    record = authenticate_artifact(root)
    manifest = canonical_file(root / 'manifest.json')
    return dict(record_sha256=manifest['record_sha256'],
                failure_signature=parity.digest(parity.canonical(dict(record, seed=0))),
                files=manifest['files'])


def authenticate(root):
    root = Path(root)
    storage, plan = canonical_file(root / 'storage.json'), canonical_file(root / 'plan.json')
    equal(set(storage), {'format', 'max_bytes', 'plan_sha256'}, 'round.storage.fields')
    equal(storage['format'], 'gomad.exploration-round-storage/v1', 'round.storage.format')
    equal(storage['plan_sha256'], parity.digest((root / 'plan.json').read_bytes()), 'round.plan.sha256')
    equal(set(plan), {'prepared', 'strategy'}, 'round.plan.fields')
    prepared, strategy = plan['prepared'], plan['strategy']
    kind, config = strategy['kind'], strategy['config']
    equal(kind in ('choice', 'simulation'), True, 'round.strategy')
    equal(prepared['schema'], 'gomad.exploration-preparation/v1', 'round.prepared.schema')
    equal(storage['max_bytes'], prepared['journal_bytes'], 'round.journal.bound')
    equal(prepared['overall_millis'], (prepared['overall_nanos'] + 999999) // 1000000, 'round.timeout')
    for name in ('parallel', 'max_executions', 'max_exploration_bytes'):
        equal(type(config[name]) is int and config[name] > 0, True, 'round.bound.' + name)
    prepared_root = parity.contained(root, prepared['prepared_artifact'] + '/storage.json').parent
    prepared_record = authenticate_artifact(prepared_root)
    equal(prepared_record['identity']['target'], prepared['module_sha256'], 'round.prepared.target')
    equal(prepared_record['identity']['toolchain'], prepared['toolchain'], 'round.prepared.toolchain')
    equal(prepared_record['identity']['configuration'], parity.digest(parity.canonical(prepared['configuration'])),
          'round.prepared.configuration')
    if kind == 'choice':
        equal(config['execution'], prepared['choice_identity'], 'round.choice.identity')
        if 'site_scope' in config:
            scope_contract(config['site_scope'])
    else:
        keys = ('module_sha256', 'toolchain', 'configuration', 'scenario', 'network', 'choice_identity',
                'soft_assertion_report_only', 'host_sha256', 'helper_sha256')
        equal(config['execution_sha256'], domain('gomad-simulation-exploration-execution/v1',
              {name: prepared[name] for name in keys}), 'round.simulation.identity')
        for dimension, bound in config['dimension_limits'].items():
            equal(bound > 0, True, 'round.dimension.' + dimension)
    raw = (root / 'journal.jsonl').read_bytes()
    equal(not raw or raw.endswith(b'\n'), True, 'round.complete_lines')
    equal(len(raw) <= storage['max_bytes'], True, 'round.journal.capacity')
    previous, state, ordinal = storage['plan_sha256'], initial_state(kind, config), 0
    events, artifacts, references = [], [], set()
    for index, line in enumerate(raw.splitlines()):
        row = parity.decode(line)
        equal(parity.canonical(row), line, 'round.row.canonical')
        equal(set(row), {'ordinal', 'previous_sha256', 'sha256', 'event'}, 'round.row.fields')
        equal(row['ordinal'], index, 'round.row.ordinal')
        equal(row['previous_sha256'], previous, 'round.row.previous')
        equal(row['sha256'], parity.digest(parity.canonical({name: row[name]
              for name in ('ordinal', 'previous_sha256', 'event')})), 'round.row.sha256')
        previous = row['sha256']
        event = row['event']
        equal(event['kind'], kind, 'round.event.kind')
        equal(set(event), {'kind', 'executions', 'segment'} | ({'evidence'} if kind == 'simulation' else set()),
              'round.event.fields')
        segment, executions = event['segment'], event['executions']
        equal(segment['index'], index, 'round.segment.index')
        equal(segment['schema'], 'gomad.' + ('choice-exploration-round/v3' if kind == 'choice'
              else 'simulation-exploration-round/v2'), 'round.segment.schema')
        unsigned = dict(segment, sha256='')
        if kind == 'simulation':
            unsigned.pop('sha256')
        equal(segment['sha256'], domain('gomad-choice-exploration-round-segment/v3' if kind == 'choice'
              else 'gomad-simulation-exploration-round-segment/v1', unsigned), 'round.segment.sha256')
        for name in ('before_sha256', 'after_sha256'):
            equal(segment[name].startswith('sha256:'), True, 'round.state.domain')
            parity.sha(segment[name][7:], 'round.state.sha256')
        if state is not None:
            equal(segment['before_sha256'], state, 'round.state.link')
        state = segment['after_sha256']
        if 'site_scope' in config:
            scope_contract(segment['site_scope'])
            equal(segment['site_scope']['patterns'], config['site_scope']['patterns'], 'round.scope.patterns')
            counts = segment['scope_counts']
            equal(set(counts), {'forced_in_scope', 'skipped_out_of_scope'}, 'round.scope.counts.fields')
            equal(all(type(value) is int and value >= 0 for value in counts.values()), True, 'round.scope.counts')
        equal(len(executions), len(segment['results']), 'round.result.count')
        equal(0 < len(executions) <= config['parallel'], True, 'round.slots')
        if kind == 'simulation':
            equal(len(event['evidence']), len(executions), 'round.evidence.count')
        for offset, (execution, result) in enumerate(zip(executions, segment['results'])):
            equal(execution['ordinal'], ordinal, 'round.execution.order')
            equal(execution['seed'], config['base_seed'], 'round.execution.seed')
            ordinal += 1
            record = execution['record']
            parity.record_contract(record, 'round.execution.record')
            equal(record['seed'], execution['seed'], 'round.record.seed')
            equal(record['outcome']['class'] != 'runner_failure', True, 'round.runner_failure')
            for name in ('target', 'toolchain', 'platform', 'implementation'):
                equal(record['identity'][name], prepared_record['identity'][name], 'round.record.' + name)
            if prepared['scenario'] is None:
                equal(record['limits'], prepared['configuration']['limits'], 'round.record.bounds')
            failed = record['outcome']['class'] in ('target_failure', 'watchdog')
            if result['failed']:
                equal(failed, True, 'round.result.failed')
                equal(result['failure_sha256'], 'sha256:' + parity.digest(parity.canonical(dict(record, seed=0))),
                      'round.result.failure_signature')
            reference = execution['artifact']
            artifact = None
            if reference is not None:
                artifact = parity.contained(root, reference + '/storage.json').parent
                retained = authenticate_artifact(artifact)
                equal(retained, record, 'round.artifact.exact_record')
                if prepared['scenario'] is not None:
                    inputs = canonical_file(artifact / 'evidence/cluster-replay.json')
                    equal(record['limits'], inputs['configuration']['limits'], 'round.cluster.bounds')
                    equal(record['identity']['configuration'], parity.digest(parity.canonical(inputs['configuration'])),
                          'round.cluster.configuration')
                    equal(inputs['configuration']['broker']['seed'], execution['seed'], 'round.cluster.seed')
                equal(sum(path.stat().st_size for path in artifact.rglob('*') if path.is_file()),
                      execution['stored_bytes'], 'round.artifact.stored_bytes')
                references.add(reference)
                artifacts.append(artifact_view(artifact))
            if kind == 'choice':
                equal(result['divergence'], execution['divergence'], 'round.divergence.raw')
                equal(result['failed'], failed and execution['divergence'] is None, 'round.choice.failed')
                if execution['divergence'] is None:
                    equal(result['outcome_sha256'], domain('gomad-choice-exploration-outcome/v2', record),
                          'round.choice.outcome')
                trace = result['source_trace']
                if trace is not None:
                    raw_trace = bytes(trace)
                    stream = record['streams']['evidence/choice']
                    equal((len(raw_trace), parity.digest(raw_trace)), (stream['bytes'], stream['sha256']),
                          'round.choice.source_trace')
                    if artifact:
                        equal((artifact / 'evidence/choice').read_bytes(), raw_trace, 'round.choice.payload')
            else:
                evidence = event['evidence'][offset]
                for field, name in (('record', 'simulation/record.json'), ('runtime_trace', 'evidence/choice')):
                    payload = bytes(evidence[field])
                    stream = record['streams'].get(name)
                    if stream is None:
                        equal(payload, b'', 'round.absent_runtime_trace')
                    else:
                        equal((len(payload), parity.digest(payload)), (stream['bytes'], stream['sha256']),
                              'round.simulation.evidence.' + name)
                        if artifact:
                            equal((artifact / name).read_bytes(), payload, 'round.simulation.payload.' + name)
                observed = parity.decode(bytes(evidence['record']))
                actual_model = []
                for decision in result['decisions']:
                    if decision['dimension'] == 'runtime':
                        equal(bytes(decision['source_trace']), bytes(evidence['runtime_trace']),
                              'round.simulation.runtime_source')
                    else:
                        equal(decision.get('alternative_controls', []), [], 'round.model.controls')
                        equal(decision.get('source_trace', []), [], 'round.model.source')
                        actual_model.append({name: value for name, value in decision.items()
                                             if name not in ('alternative_controls', 'source_trace')})
                equal(sorted(actual_model, key=parity.canonical),
                      sorted(observed['exploration_decisions'], key=parity.canonical), 'round.model.decisions')
                control = observed['exploration_plan']
                equal(control['base_seed'], execution['seed'], 'round.simulation.seed')
                equal(control['execution_sha256'], config['execution_sha256'], 'round.simulation.execution')
                for forced in control['overrides'] or []:
                    equal(forced['identity'], domain('gomad-simulation-exploration-forced-decision/v1',
                          dict(forced, identity='')), 'round.simulation.forced_identity')
                equal(control['candidate_sha256'], domain('gomad-simulation-exploration-candidate/v1',
                      {name: control[name] for name in ('base_seed', 'controller_sha256', 'execution_sha256', 'overrides')}),
                      'round.simulation.plan_candidate')
                if artifact:
                    equal(parity.decode((artifact / 'simulation/plan.json').read_bytes()), control, 'round.simulation.plan')
                    profile = canonical_file(artifact / 'evidence/inspection.json')['simulation']
                    equal(profile['candidate_sha256'], result['candidate_sha256'], 'round.simulation.result_candidate')
                    equal(profile['outcome_sha256'], result['outcome_sha256'], 'round.simulation.result_outcome')
        events.append(event)
    equal(ordinal <= config['max_executions'], True, 'round.execution.bound')
    if (root / 'evidence').exists():
        equal({str(path.relative_to(root)) for path in (root / 'evidence').iterdir()
               if path.name not in ('.targets', '.store.json', '.lock')}, references,
              'round.unreferenced_artifacts')
    return dict(plan=plan, events=events, prepared_artifact=artifact_view(prepared_root),
                artifacts=sorted(artifacts, key=parity.canonical))


def scope_contract(scope):
    equal(set(scope), {'patterns', 'resolved_sha256', 'resolved_count'}, 'round.scope.fields')
    equal(type(scope['patterns']) is list and bool(scope['patterns']), True, 'round.scope.patterns')
    for pattern in scope['patterns']:
        path = pattern[:-4] if pattern.endswith('/...') else pattern
        equal(all(re.fullmatch(r'[A-Za-z0-9_+~-](?:[A-Za-z0-9_.+~-]*[A-Za-z0-9_+~-])?', part)
                  for part in path.split('/')), True, 'round.scope.package')
    equal(scope['resolved_sha256'].startswith('sha256:'), True, 'round.scope.digest.domain')
    parity.sha(scope['resolved_sha256'][7:], 'round.scope.digest')
    equal(type(scope['resolved_count']) is int and scope['resolved_count'] >= 0, True, 'round.scope.count')


def scope_semantics(root, report, envelope):
    """Cross-check public symbols against authenticated raw trace positions and scope commitments."""
    checked = authenticate(root)
    config = checked['plan']['strategy']['config']
    patterns = config['site_scope']['patterns']

    def matches(symbol):
        package = symbol.get('package')
        return package is not None and any(package == pattern or pattern.endswith('/...')
            and (package == pattern[:-4] or package.startswith(pattern[:-3])) for pattern in patterns)

    projected = report['campaign']['executions']
    executions = [execution for event in checked['events'] for execution in event['executions']]
    equal(len(projected), len(executions), 'round.scope.inspect.executions')
    def candidate(decisions):
        identity = config['execution']
        projection = dict(target_sha256='sha256:' + bytes(identity['TargetSHA256']).hex(),
            toolchain_build_key=identity['ToolchainBuildKey'], goos=identity['GOOS'], goarch=identity['GOARCH'],
            implementation_sha256='sha256:' + bytes(identity['ImplementationSHA256']).hex(),
            controller_sha256=config['controller_sha256'], base_seed=str(config['base_seed']), decisions=decisions)
        return 'sha256:' + parity.digest(b'gomad-choice-exploration-candidate/v1\0' +
            json.dumps(projection, separators=(',', ':'), ensure_ascii=False).encode())

    fingerprints, forced_ordinals = set(), []
    proposals = {candidate([]): (0, True)}
    runnable_in_scope = runnable_out_of_scope = mixed_runnable = 0
    position = 0
    for index, event in enumerate(checked['events']):
        skipped = 0
        for execution, result in zip(event['executions'], event['segment']['results']):
            depth, allowed = proposals[result['candidate_sha256']]
            equal(allowed, True, 'round.scope.forced_package')
            if depth:
                forced_ordinals.append(depth - 1)
            view = projected[position]
            position += 1
            equal(view['ordinal'], execution['ordinal'], 'round.scope.inspect.ordinal')
            symbols = {site['fingerprint']: site['symbol'] for site in view['choices']['sites']}
            trace = bytes(result['source_trace'])
            target = bytes.fromhex(execution['record']['identity']['target'])
            decisions, pending, readiness, raw_positions = [], [], {}, []
            equal(len(trace) % 96, 0, 'round.scope.trace.framing')
            for offset in range(0, len(trace), 96):
                raw = trace[offset:offset + 96]
                kind, flags = raw[8], raw[9]
                if flags & 2:
                    origin = int.from_bytes(raw[32:40], 'big')
                    for poll in range(max(0, int.from_bytes(raw[20:24], 'big') - 1)):
                        readiness[origin + poll] = int.from_bytes(raw[10:12], 'big')
                missing = bool(flags & 4)
                fingerprint = 'missing' if missing else parity.digest(target + raw[8:9] + raw[24:32])
                equal(fingerprint in symbols, True, 'round.scope.raw.symbol')
                admitted = not missing and matches(symbols[fingerprint])
                if admitted:
                    fingerprints.add(fingerprint)
                if kind == 4:
                    pending.append(admitted)
                    continue
                allowed = any(pending) if kind == 1 else admitted
                mixed = any(pending) and not all(pending)
                pending = []
                alternatives = int.from_bytes(raw[12:16], 'big')
                if not flags & 2 and alternatives > 1:
                    decision = dict(ordinal=str(len(decisions)), kind=kind,
                        site_offset=str(int.from_bytes(raw[24:32], 'big')), site_missing=missing,
                        rank_override=False, alternatives=alternatives,
                        selected=int.from_bytes(raw[16:20], 'big'), data=int.from_bytes(raw[20:24], 'big'))
                    if any(raw[32:64]):
                        decision['selected_identity'] = 'sha256:' + raw[32:64].hex()
                    decision['alternative_set_digest'] = 'sha256:' + raw[64:96].hex()
                    decisions.append((decision, allowed))
                    raw_positions.append(offset // 96)
                    if kind == 1:
                        runnable_in_scope += int(allowed)
                        runnable_out_of_scope += int(not allowed)
                        mixed_runnable += int(mixed)
            for ordinal in range(depth, len(decisions)):
                decision, allowed = decisions[ordinal]
                word = readiness.get(raw_positions[ordinal], 0)
                next_poll = ordinal + 1 < len(decisions) and (
                    decisions[ordinal + 1][0]['kind'], decisions[ordinal + 1][0]['site_offset'],
                    decisions[ordinal + 1][0]['alternatives']) == (2, decision['site_offset'], 3)
                no_op = decision['kind'] == 2 and decision['alternatives'] == 2 and not next_poll and word & 1 and (
                    (word >> 6 == 0 and word & 62 in (0, 2, 8)) or
                    (word >> 6 == 1 and word & 62 in (0, 4, 8, 16)))
                if ordinal >= config['start_ordinal'] and not allowed and not no_op:
                    skipped += decision['alternatives'] - 1
                for rank in range(decision['alternatives']):
                    if rank == decision['selected']:
                        continue
                    prefix = [dict(value) for value, _ in decisions[:ordinal + 1]]
                    prefix[-1].update(rank_override=True, selected=rank)
                    prefix[-1].pop('selected_identity', None)
                    proposals[candidate(prefix)] = ordinal + 1, allowed
        equal(event['segment']['scope_counts']['skipped_out_of_scope'], skipped, 'round.scope.skipped_alternatives')
        resolved = event['segment']['site_scope']
        expected = 'sha256:' + parity.digest(b'gomad-explore-site-scope/v1\0' + ''.join(sorted(fingerprints)).encode())
        equal((resolved['resolved_sha256'], resolved['resolved_count']),
              (expected, len(fingerprints)), 'round.scope.resolution')
        if index == 0:
            equal(resolved, config['site_scope'], 'round.scope.base_resolution')
    admitted = sum(event['segment']['scope_counts']['forced_in_scope'] for event in checked['events'])
    equal(admitted, len(executions) - 1 + envelope['pending'], 'round.scope.admitted_frontier')
    equal(envelope['site_scope'], checked['events'][-1]['segment']['site_scope'], 'round.scope.envelope')
    return dict(forced_ordinals=forced_ordinals, admitted=admitted,
        runnable_in_scope=runnable_in_scope, runnable_out_of_scope=runnable_out_of_scope,
        mixed_runnable=mixed_runnable,
        skipped=sum(event['segment']['scope_counts']['skipped_out_of_scope'] for event in checked['events']))


def scope_digest_refusal(run, prefix, root, destination):
    """Reseal outer commitments to reach the recorded-base scope re-resolution check."""
    authenticate(root)
    shutil.copytree(root, destination)
    plan = canonical_file(destination / 'plan.json')
    plan['strategy']['config']['site_scope']['resolved_sha256'] = 'sha256:' + '0' * 64
    (destination / 'plan.json').write_bytes(parity.canonical(plan))
    storage = canonical_file(destination / 'storage.json')
    storage['plan_sha256'] = parity.digest((destination / 'plan.json').read_bytes())
    (destination / 'storage.json').write_bytes(parity.canonical(storage))
    previous, rows = storage['plan_sha256'], []
    for index, line in enumerate((destination / 'journal.jsonl').read_bytes().splitlines()):
        row = parity.decode(line)
        if index == 0:
            segment = row['event']['segment']
            segment['before_sha256'] = initial_state('choice', plan['strategy']['config'])
            segment['sha256'] = domain('gomad-choice-exploration-round-segment/v3', dict(segment, sha256=''))
        row['previous_sha256'] = previous
        row['sha256'] = parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'}))
        previous = row['sha256']
        rows.append(parity.canonical(row))
    (destination / 'journal.jsonl').write_bytes(b'\n'.join(rows) + b'\n')
    before = snapshot(destination)
    for verb in ('inspect', 'resume'):
        refusal = run('scope-digest-' + verb, [*prefix, verb, '--json', destination], (2,))
        if verb == 'resume':
            equal('exploration plan identity or bounds changed' in json.loads(refusal)['message'],
                  True, 'round.scope.digest.refusal')
        equal(snapshot(destination), before, 'round.scope.digest.immutable')


def scope_block_refusals(run, prefix, root, work):
    for name, changed in [('patterns', {'patterns': []}), ('digest', {'resolved_sha256': 'invalid'}),
                          ('count', {'resolved_count': -1}), ('unknown', {'extra': True})]:
        destination = work / ('scope-malformed-' + name)
        shutil.copytree(root, destination)
        plan = canonical_file(destination / 'plan.json')
        plan['strategy']['config']['site_scope'].update(changed)
        (destination / 'plan.json').write_bytes(parity.canonical(plan))
        storage = canonical_file(destination / 'storage.json')
        storage['plan_sha256'] = parity.digest((destination / 'plan.json').read_bytes())
        (destination / 'storage.json').write_bytes(parity.canonical(storage))
        before = snapshot(destination)
        for verb in ('inspect', 'resume'):
            run('scope-malformed-' + name + '-' + verb, [*prefix, verb, '--json', destination], (2,))
            equal(snapshot(destination), before, 'round.scope.malformed.immutable')


def negative_checks(root, work):
    """Reject corrupt real copies, independently of the product's own open path."""
    work.mkdir()
    for name in ('plan', 'envelope', 'segment', 'payload', 'manifest', 'record'):
        copied = work / name
        shutil.copytree(root, copied)
        rows = (copied / 'journal.jsonl').read_bytes().splitlines()
        row = parity.decode(rows[0])
        artifact = copied / row['event']['executions'][0]['artifact']
        if name == 'plan':
            (copied / 'plan.json').write_bytes((copied / 'plan.json').read_bytes() + b' ')
        elif name in ('envelope', 'segment'):
            if name == 'envelope':
                row['previous_sha256'] = '0' * 64
            else:
                row['event']['segment']['after_sha256'] = 'sha256:' + '0' * 64
                row['sha256'] = parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'}))
            (copied / 'journal.jsonl').write_bytes(b'\n'.join([parity.canonical(row), *rows[1:]]) + b'\n')
        elif name == 'payload':
            (artifact / 'evidence/choice').write_bytes((artifact / 'evidence/choice').read_bytes() + b'x')
        elif name == 'manifest':
            manifest = canonical_file(artifact / 'manifest.json')
            manifest['record_sha256'] = '0' * 64
            (artifact / 'manifest.json').write_bytes(parity.canonical(manifest))
        else:
            (artifact / 'record.json').write_bytes((artifact / 'record.json').read_bytes() + b' ')
        try:
            authenticate(copied)
        except (ValueError, KeyError, TypeError):
            continue
        raise ValueError('independent round verifier accepted corrupt ' + name)


def plan_refusals(run, prefix, root, work):
    for name in ('plan-digest', 'plan-resealed'):
        copied = work / name
        shutil.copytree(root, copied)
        plan = canonical_file(copied / 'plan.json')
        plan['strategy']['config']['parallel'] = 0
        (copied / 'plan.json').write_bytes(parity.canonical(plan))
        if name == 'plan-resealed':
            storage = canonical_file(copied / 'storage.json')
            storage['plan_sha256'] = parity.digest((copied / 'plan.json').read_bytes())
            (copied / 'storage.json').write_bytes(parity.canonical(storage))
            previous = storage['plan_sha256']
            rows = []
            for line in (copied / 'journal.jsonl').read_bytes().splitlines():
                row = parity.decode(line)
                row['previous_sha256'] = previous
                row['sha256'] = parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'}))
                previous = row['sha256']
                rows.append(parity.canonical(row))
            (copied / 'journal.jsonl').write_bytes(b'\n'.join(rows) + b'\n')
        before = snapshot(copied)
        for verb in ('inspect', 'resume'):
            run(name + '-' + verb, [*prefix, verb, '--json', copied], (2,))
            equal(snapshot(copied), before, 'round.plan_refusal.immutable')


def runtime_tamper(root, destination):
    """Reseal an actual artifact whose claimed runtime forcing never occurred."""
    authenticate_artifact(root)
    shutil.copytree(root, destination)
    plan = parity.decode((destination / 'simulation/plan.json').read_bytes())
    forced = next(value for value in plan['overrides'] or [] if value['dimension'] == 'runtime')
    forced['ordinal'] = len((destination / 'evidence/choice').read_bytes()) + 1
    forced['identity'] = domain('gomad-simulation-exploration-forced-decision/v1', dict(forced, identity=''))
    plan['candidate_sha256'] = domain('gomad-simulation-exploration-candidate/v1',
        {name: plan[name] for name in ('base_seed', 'controller_sha256', 'execution_sha256', 'overrides')})
    # Plan uses declaration-order JSON, unlike the canonical outer artifact.
    raw_plan = json.dumps(plan, separators=(',', ':'), ensure_ascii=False).encode()
    (destination / 'simulation/plan.json').write_bytes(raw_plan)
    observation = parity.decode((destination / 'simulation/record.json').read_bytes())
    observation['exploration_plan'] = plan
    observation['identity'] = ''
    unsigned = json.dumps(observation, separators=(',', ':'), ensure_ascii=False).encode()
    observation['identity'] = 'sha256:' + parity.digest(b'gomad-rust-simulation-observation/v1\0' + unsigned)
    raw_record = json.dumps(observation, separators=(',', ':'), ensure_ascii=False).encode()
    (destination / 'simulation/record.json').write_bytes(raw_record)
    inspection = canonical_file(destination / 'evidence/inspection.json')
    for name, raw in (('plan', raw_plan), ('record', raw_record)):
        inspection['simulation'][name]['bytes'] = len(raw)
        inspection['simulation'][name]['sha256'] = parity.digest(raw)
    (destination / 'evidence/inspection.json').write_bytes(parity.canonical(inspection))
    manifest = canonical_file(destination / 'manifest.json')
    record = copy.deepcopy(manifest['record'])
    for name in ('simulation/plan.json', 'simulation/record.json', 'evidence/inspection.json'):
        raw = (destination / name).read_bytes()
        for entry in (manifest['files'][name], record['streams'][name]):
            entry['bytes'], entry['sha256'] = len(raw), parity.digest(raw)
    encoded = parity.canonical(record)
    (destination / 'record.json').write_bytes(encoded)
    manifest['files']['record.json'].update(bytes=len(encoded), sha256=parity.digest(encoded))
    manifest.update(record=record, record_sha256=parity.digest(encoded))
    (destination / 'manifest.json').write_bytes(parity.canonical(manifest))
    authenticate_artifact(destination)
    return destination
