#!/usr/bin/env python3
"""Exercise public seed campaigns and independently authenticate retained evidence."""
import argparse
import copy
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
loader = SourceFileLoader('campaign_parity', str(CORPUS.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label}: got {actual!r}, expected {expected!r}')


def canonical_file(path):
    raw = path.read_bytes()
    value = parity.decode(raw)
    equal(parity.canonical(value), raw, str(path) + '.canonical')
    return value


def authenticate_artifact(root):
    equal(canonical_file(root / 'storage.json'), {'format': 'gomad.artifact-storage/v2'}, 'artifact.lineage')
    manifest = canonical_file(root / 'manifest.json')
    equal(manifest['schema'], 'gomad.artifact-manifest/v2', 'artifact.schema')
    record = manifest['record']
    parity.record_contract(record, str(root))
    encoded = parity.canonical(record)
    equal(parity.digest(encoded), manifest['record_sha256'], 'artifact.record_sha256')
    listed = {'storage.json', 'manifest.json', *manifest['files']}
    for child in root.rglob('*'):
        if child.is_symlink() or child.is_file() and str(child.relative_to(root)) not in listed:
            raise ValueError(f'unlisted or symlink artifact payload: {child}')
    for name, entry in manifest['files'].items():
        raw = parity.contained(root, name).read_bytes()
        equal((len(raw), parity.digest(raw)), (entry['bytes'], entry['sha256']), name + '.payload')
        equal(entry['module'], name == 'target/module.wasm', name + '.kind')
        if entry['module']:
            equal(entry['sha256'], record['identity']['target'], 'artifact.target')
    equal((root / 'record.json').read_bytes(), encoded, 'artifact.record_bytes')
    for name, stream in record['streams'].items():
        entry = manifest['files'][name]
        equal((stream['bytes'], stream['sha256']), (entry['bytes'], entry['sha256']), name + '.stream')
    return record


def seeds(selection):
    result = []
    for term in selection.split(','):
        endpoints = list(map(int, term.split('-')))
        result.extend(range(endpoints[0], endpoints[-1] + 1))
    equal(len(set(result)), len(result), 'selection.unique')
    return result


def authenticate_campaign(root, complete=True):
    storage = canonical_file(root / 'storage.json')
    equal(storage['format'], 'gomad.campaign-storage/v2', 'campaign.lineage')
    plan = canonical_file(root / 'plan.json')
    equal(storage['plan_sha256'], parity.digest((root / 'plan.json').read_bytes()), 'plan.sha256')
    if plan['schema'] not in ('gomad.seed-campaign-plan/v2', 'gomad.merged-seed-campaign-plan/v2'):
        raise ValueError('unsupported campaign plan schema')
    equal(storage['max_bytes'], plan['journal_bytes'], 'plan.journal_bound')
    selected = seeds(plan['selection'])
    equal(len(selected), plan['selection_count'], 'plan.selection_count')
    prepared = authenticate_artifact(root / plan['prepared_artifact'])
    equal(prepared['identity']['target'], plan['module_sha256'], 'plan.module')
    equal(prepared['identity']['toolchain'], plan['toolchain'], 'plan.toolchain')
    equal(prepared['identity']['configuration'], parity.digest(parity.canonical(plan['configuration'])), 'plan.configuration')
    raw = (root / 'journal.jsonl').read_bytes()
    if len(raw) > storage['max_bytes']:
        raise ValueError('journal exceeds authenticated bound')
    equal(raw.endswith(b'\n') or not raw, True, 'journal.complete_lines')
    previous = storage['plan_sha256']
    pending, receipts, executions = {}, {}, []
    shard = plan['shard'] or {'index': 0, 'count': 1}
    next_ordinal = shard['index']
    missing = set()
    for interval in plan.get('merge_missing', []):
        start, end = interval['start'], interval['end']
        if not 0 <= start <= end < len(selected) or any(index in missing for index in range(start, end + 1)):
            raise ValueError('invalid merged missing ordinal ranges')
        missing.update(range(start, end + 1))
    if plan['schema'] == 'gomad.merged-seed-campaign-plan/v2':
        parity.sha(plan['plan_sha256'], 'merge.source_plan_sha256')
    stopped = False
    completed = False
    rows = []

    def admit(jobs):
        nonlocal next_ordinal
        if jobs and stopped:
            raise ValueError('admission after failure stop')
        for job in jobs:
            while next_ordinal in missing:
                next_ordinal += shard['count']
            ordinal, seed = job['ordinal'], job['seed']
            equal(ordinal, next_ordinal, 'admission.order')
            equal(seed, selected[ordinal], 'admission.seed')
            if ordinal in pending:
                raise ValueError('duplicate admission')
            pending[ordinal] = seed
            next_ordinal += shard['count']
        if len(pending) > plan['parallel']:
            raise ValueError('admission exceeds slot bound')

    def execution_contract(execution):
        ordinal = execution['ordinal']
        equal(execution['seed'], pending[ordinal], 'execution.seed')
        record = execution['record']
        parity.record_contract(record, 'journal.execution')
        equal(record['seed'], execution['seed'], 'record.seed')
        equal(record['identity'], prepared['identity'], 'record.immutable_identity')
        equal(record['limits'], plan['configuration']['limits'], 'record.bounds')
        if execution['artifact'] is not None:
            artifact = root / execution['artifact']
            if not artifact.resolve().is_relative_to(root.resolve()):
                raise ValueError('execution artifact escapes campaign')
            retained = authenticate_artifact(artifact)
            equal(parity.digest(parity.canonical(dict(retained, seed=0))),
                  parity.digest(parity.canonical(dict(record, seed=0))), 'retained.failure_signature')
        return record

    for ordinal, line in enumerate(raw.splitlines()):
        row = parity.decode(line)
        equal(parity.canonical(row), line, 'journal.canonical')
        equal(set(row), {'ordinal', 'previous_sha256', 'sha256', 'event'}, 'journal.fields')
        equal(row['ordinal'], ordinal, 'journal.ordinal')
        equal(row['previous_sha256'], previous, 'journal.previous')
        equal(row['sha256'], parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'})), 'journal.hash')
        previous = row['sha256']
        if completed:
            raise ValueError('event after complete')
        event = row['event']
        kind = event['kind']
        if kind == 'admit':
            admit(event['jobs'])
        elif kind == 'receipt':
            execution = event['execution']
            execution_contract(execution)
            if execution['ordinal'] in receipts:
                raise ValueError('duplicate receipt')
            receipts[execution['ordinal']] = execution
        elif kind == 'outcome':
            for execution in event['executions']:
                execution_contract(execution)
                index = execution['ordinal']
                equal(index, min(pending), 'outcome.order')
                receipt = receipts.pop(index)
                equal(receipt['record'], execution['record'], 'outcome.receipt')
                del pending[index]
                executions.append(execution)
            stopped |= event['stop_reason'] is not None
            admit(event['admitted'])
        elif kind == 'complete':
            equal((pending, receipts), ({}, {}), 'completion.pending')
            completed = True
        else:
            raise ValueError(f'unknown journal event {kind}')
        rows.append(row)
    equal(completed, complete, 'campaign.complete')
    if completed and plan['schema'] == 'gomad.merged-seed-campaign-plan/v2':
        equal({item['ordinal'] for item in executions}, set(range(len(selected))) - missing, 'merge.complete_selection')
    return dict(plan=plan, executions=executions, journal=rows)


def execution_view(validated):
    return [dict(ordinal=item['ordinal'], seed=item['seed'], record=item['record'],
                 record_sha256=parity.digest(parity.canonical(item['record'])),
                 failure_signature=parity.digest(parity.canonical(dict(item['record'], seed=0))))
            for item in validated['executions']]


def rechain(root, change):
    plan = canonical_file(root / 'plan.json')
    change(plan)
    (root / 'plan.json').write_bytes(parity.canonical(plan))
    storage = canonical_file(root / 'storage.json')
    previous = parity.digest(parity.canonical(plan))
    storage['plan_sha256'] = previous
    (root / 'storage.json').write_bytes(parity.canonical(storage))
    rows = []
    for line in (root / 'journal.jsonl').read_bytes().splitlines():
        row = parity.decode(line)
        row['previous_sha256'] = previous
        row['sha256'] = parity.digest(parity.canonical({key: value for key, value in row.items() if key != 'sha256'}))
        previous = row['sha256']
        rows.append(parity.canonical(row) + b'\n')
    (root / 'journal.jsonl').write_bytes(b''.join(rows))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-compiler', 'work', 'input'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--preparation-root', type=Path)
    args = parser.parse_args()
    args.binary, args.go_compiler, args.work, args.input = [path.resolve() for path in (args.binary, args.go_compiler, args.work, args.input)]
    args.work.mkdir(parents=True, mode=0o700, exist_ok=False)
    preparation = (args.preparation_root or args.work.parent / 'campaign-preparation').resolve()
    module = CORPUS.parents[1]
    declaration = parity.read(CORPUS / 'campaign-corpus.json')
    equal(declaration['schema'], 'gomad.seed-campaign-corpus/v1', 'corpus.schema')
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    commands = []

    def run(name, command, status=0, extra_environment=None):
        result = subprocess.run(list(map(str, command)), capture_output=True, timeout=600,
                                env=dict(environment, **(extra_environment or {})))
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                             expected=status, stdout_sha256=parity.digest(result.stdout), stderr_sha256=parity.digest(result.stderr)))
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        (args.work / 'commands.json').write_bytes(parity.canonical(commands) + b'\n')
        equal(result.returncode, status, name + '.status: ' + result.stderr.decode(errors='replace'))
        return result.stdout

    fixture_root = args.work / 'identity'
    bundle = parity.decode(run('identity', [args.binary, 'fixture-replay-input', '--root', fixture_root, '--input', args.input]))
    parity.authenticate(fixture_root, bundle)
    artifacts = [fixture_root / row['artifact'] for row in bundle['records']]
    artifact = next(path for path in artifacts if parity.read(path / 'manifest.json')['record']['profile'] == 'gomad.wasi-cooperative/v1')
    inputs = canonical_file(artifact / 'evidence/replay.json')
    configuration = inputs['configuration']
    installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=parity.digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(module / 'internal/model/runtimesource'), cache_root=str(preparation),
        profile='gomad.wasi-cooperative/v1', fuel=configuration['limits']['fuel'], memory_bytes=configuration['limits']['memory_bytes'],
        helper_sha256=inputs['helper_sha256'], host_sha256=inputs['host_sha256'],
        runtime_implementation_sha256=inputs['runtime_implementation_sha256'], model=configuration['model'],
        namespace=configuration['namespace'], storage=configuration.get('storage'), network=None, build_tags=[])
    manifest = args.work / 'installation.json'
    manifest.write_bytes(parity.canonical(installation))
    binary = [args.binary, '--installation', manifest]

    def explore_command(case, destination):
        return [*binary, 'explore', '--json', '--choices', '--on-failure=all', '--parallel=1',
            '--seeds=' + declaration['seeds'], '--keep-successes=all', '--success-limit=8', '--success-bytes=256MiB',
            '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
            '--working-dir', module / case['module'],
            *['--build-tag=' + tag for tag in case['tags']], case['kind'], case['source'], '--', *case['arguments']]

    def campaign_root(destination):
        roots = sorted((destination / 'campaigns').glob('campaign-*'))
        equal(len(roots), 1, 'campaign.roots')
        return roots[0]

    def report_view(raw, root):
        reports = [parity.decode(line) for line in raw.splitlines()]
        results = [row for row in reports if row.get('type') == 'result']
        equal(len(results), 1, 'report.results')
        result = copy.deepcopy(results[0])
        equal(result['campaign_path'], str(root), 'report.campaign_path')
        result['campaign_path'] = '<campaign>'
        for retained in result.get('success_artifacts', []):
            retained['path'] = '<campaign>/' + str(Path(retained['path']).relative_to(root))
        return result

    views = []
    roots = {}
    for repetition in range(2):
        view = {}
        for case in declaration['cases']:
            name = case['name']
            destination = args.work / f'{name}-{repetition}'
            raw = run(f'{name}-{repetition}', explore_command(case, destination), case['status'])
            root = campaign_root(destination)
            validated = authenticate_campaign(root)
            equal([item['seed'] for item in validated['executions']], seeds(declaration['seeds']), name + '.seed_order')
            view[name] = dict(plan=validated['plan'], journal=validated['journal'], records=execution_view(validated), report=report_view(raw, root), status=case['status'])
            roots[name] = root
        views.append(view)
    equal(views[0], views[1], 'fresh repeated campaign projection')

    case = declaration['cases'][0]
    interrupted_destination = args.work / 'interrupted'
    run('interrupt', explore_command(case, interrupted_destination), 86,
        {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '2'})
    interrupted = campaign_root(interrupted_destination)
    prefix = authenticate_campaign(interrupted, complete=False)
    equal(len(prefix['executions']), 1, 'interrupted.complete_prefix')
    with (interrupted / 'journal.jsonl').open('ab') as stream:
        stream.write(b'{"ordinal":')
    raw = run('resume', [*binary, 'resume', '--json', interrupted], case['status'])
    resumed = authenticate_campaign(interrupted)
    equal(execution_view(resumed), views[0]['runtime']['records'], 'resume.records')
    equal(report_view(raw, interrupted), views[0]['runtime']['report'], 'resume.report')
    before = (interrupted / 'journal.jsonl').read_bytes()
    run('resume-again', [*binary, 'resume', '--json', interrupted], case['status'])
    equal((interrupted / 'journal.jsonl').read_bytes(), before, 'resume.idempotent')

    source = roots['runtime']
    shards = []
    shard_records = []
    for index in range(2):
        destination = args.work / f'shard-{index}'
        run(f'shard-{index}', [*binary, 'execute-shard', '--json', '--shard', f'{index}/2', '--artifacts', destination, source / 'plan.json'])
        root = campaign_root(destination)
        shards.append(root)
        validated = authenticate_campaign(root)
        equal(validated['plan']['shard'], {'index': index, 'count': 2}, 'shard.assignment')
        shard_records.extend(execution_view(validated))
    equal(sorted(shard_records, key=lambda item: item['ordinal']), views[0]['runtime']['records'], 'shards.unsharded_records')
    merged_root = args.work / 'merged'
    run('merge', [*binary, 'merge', '--json', '--output', merged_root, source / 'plan.json', *shards])
    merged = authenticate_campaign(merged_root)
    equal(merged['plan']['plan_sha256'], parity.digest((source / 'plan.json').read_bytes()), 'merge.original_plan')
    equal(execution_view(merged), views[0]['runtime']['records'], 'merge.unsharded_records')
    run('merge-missing', [*binary, 'merge', '--json', '--output', args.work / 'missing', source / 'plan.json', shards[0]], 2)
    partial_root = args.work / 'partial'
    run('merge-partial', [*binary, 'merge', '--json', '--partial', '--output', partial_root, source / 'plan.json', shards[0]])
    partial = authenticate_campaign(partial_root)
    equal([item['ordinal'] for item in partial['executions']], [0], 'merge.partial_ordinals')
    run('merge-duplicate', [*binary, 'merge', '--json', '--output', args.work / 'duplicate', source / 'plan.json', shards[0], shards[0]], 2)

    for name, mutate in [('bounds', lambda plan: plan['configuration']['limits'].__setitem__('fuel', plan['configuration']['limits']['fuel'] + 1)),
                         ('identity', lambda plan: plan.__setitem__('host_sha256', '0' * 64))]:
        root = args.work / ('changed-' + name)
        shutil.copytree(source, root)
        rechain(root, mutate)
        run('refuse-' + name, [*binary, 'resume', '--json', root], 2)
    corrupt = args.work / 'corrupt-journal'
    shutil.copytree(source, corrupt)
    data = (corrupt / 'journal.jsonl').read_bytes()
    first, remainder = data.split(b'\n', 1)
    row = parity.decode(first)
    row['sha256'] = '0' * 64
    (corrupt / 'journal.jsonl').write_bytes(parity.canonical(row) + b'\n' + remainder)
    run('refuse-journal', [*binary, 'resume', '--json', corrupt], 2)
    legacy = args.work / 'go-lineage'
    legacy.mkdir()
    (legacy / 'storage.json').write_bytes(parity.canonical({'format': 'gomad.campaign-storage/v1'}))
    run('refuse-go-lineage', [*binary, 'resume', '--json', legacy], 2)
    invalid = [[], ['--count=0'], ['--seeds=7', '--count=2'], ['--parallel=0'],
        ['--choice-bytes=1'], ['--coverage=semantic'], ['--guide'], ['--toolchain-root=/tmp'],
        ['--env=GOMADSEED=7'], ['--env=LD_PRELOAD=x'], ['--env=A=1', '--env=A=2'], ['--env=9BAD=x'],
        ['--strategy=simulation-exploration'], ['--max-executions=2'], ['--build-tag=test_dep', 'go-test', './io_failure', '-test.v']]
    for index, flags in enumerate(invalid):
        target = [] if not flags or 'go-test' in flags else ['go-run', './internal/backends/wasm/testdata/runtime_probes']
        run(f'grammar-{index}', [*binary, 'explore', '--json', *flags, *target], 2)
    view = dict(schema='gomad.seed-campaign-golden/v1', cases=views[0])
    golden = CORPUS / declaration['golden']
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    parity.golden(golden, view, regenerate)
    first = golden.read_bytes()
    parity.golden(golden, dict(schema=view['schema'], cases=views[1]), regenerate)
    equal(golden.read_bytes(), first, 'second regeneration bytes')
    print('campaign: public preparation, repeated authenticated records, resume, shards, merge and refusals passed')


if __name__ == '__main__':
    main()
