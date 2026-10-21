#!/usr/bin/env python3
"""Fresh Go/Rust qualify --spec verdict parity and the Rust qualification golden lane.

Each declared cell runs once on the frozen Go reference and twice on the Rust
binary, each run in a fresh operator process and artifact root. Rust Results
and receipts are authenticated on raw bytes (content address, canonical
encoding, execution-record hashes, retained artifacts, receipt binding)
before the stable qualification view is compared across the two Rust runs and
with the reviewed golden. Measured PID/CPU/RSS/wall values are validated by
the binary's own recomputation and here for presence and binding, never used
as byte-equality targets. Tampered Results and receipts must be refused by
the installation-free report verb.
"""
import argparse
import base64
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
MODULE = CORPUS.parents[1]


def load(name, path):
    loader = SourceFileLoader(name, str(path))
    module = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(module)
    return module


check = load('qualify_campaign_check', CORPUS / 'campaign-check.py')
parity = check.parity
equal = check.equal

TARGETS = {
    'fixture': dict(directory='internal/backends/wasm/qualification/testdata/fixture', kind='go-run', source='.', tags=[],
                    installation='fixture'),
    'sqlite': dict(directory='../..', kind='go-run', source='./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go',
                   tags=['sqlite3_dotlk'], installation='sqlite'),
    # The one-box family cell of qualify-corpus.json; its installation pins the
    # Temporal guest configuration and build tags.
    'onebox': dict(directory='../..', kind='go-test', source='./tests/gomadfunctional', tags=[], installation='onebox'),
    # The multi-instance cell: two resident nodes, a crash and a durable restart.
    'cluster': dict(directory='.', kind='go-run', source='./internal/backends/wasm/command/testdata/scenario', tags=[],
                    installation='fixture', scenario='internal/backends/wasm/command/testdata/scenario/durable.json'),
}

# Canonical cell budgets are the qualification owner's declared values in
# internal/backends/wasm/qualification/{integration,sqlite,onebox}_test.go,
# used unchanged. The installation owns fuel and memory, so canonical runs use
# an installation copy pinned to the owner's values. The cluster row reuses the
# owner fixture/cluster bounds (cluster_test.go): the Go cluster cell is a test
# API cell, not a public `qualify --spec`.
CANONICAL = {
    'fixture': dict(campaign=dict(ExecutionTimeout=60_000_000_000, OverallTimeout=180_000_000_000, TerminateGrace=100_000_000,
                                  OutputLimit=1 << 20, WorldTransitionLimit=1 << 20), fuel=4_000_000_000, memory=256 << 20),
    'sqlite': dict(campaign=dict(ExecutionTimeout=60_000_000_000, OverallTimeout=180_000_000_000, TerminateGrace=100_000_000,
                                 OutputLimit=1 << 20, WorldTransitionLimit=1 << 20), fuel=4_000_000_000, memory=256 << 20),
    'cluster': dict(campaign=dict(ExecutionTimeout=60_000_000_000, OverallTimeout=180_000_000_000, TerminateGrace=100_000_000,
                                  OutputLimit=1 << 20, WorldTransitionLimit=1 << 20), fuel=4_000_000_000, memory=256 << 20),
    'onebox': dict(campaign=dict(ExecutionTimeout=900_000_000_000, OverallTimeout=1_800_000_000_000, TerminateGrace=1_000_000_000,
                                 OutputLimit=256 << 20, WorldTransitionLimit=32 << 20, ChoiceTraceLimit=64 << 20,
                                 SuccessBytesLimit=2 << 30,
                                 Environment=['TEMPORAL_ROOT=/workspace', 'TMPDIR=/tmp', 'TEMPORAL_TEST_LOG_LEVEL=warn']),
                   fuel=1 << 50, memory=4 << 30),
}

# Lane-local parity budgets. They are not qualification budgets: parity runs
# compare Go/Rust verdicts under shared host load, so the execution and overall
# deadlines are widened for the lane only. Canonical runs below use CANONICAL.
LANE_EXECUTION_TIMEOUT = 300_000_000_000
LANE_OVERALL_TIMEOUT = 1_200_000_000_000
LANE_CAMPAIGN = dict(ExecutionTimeout=LANE_EXECUTION_TIMEOUT, OverallTimeout=LANE_OVERALL_TIMEOUT, TerminateGrace=100_000_000,
                     OutputLimit=8 << 20, WorldTransitionLimit=64 << 20)


def go_classification(qualification):
    if qualification.get('Failure') or qualification.get('failure'):
        return (qualification.get('failure') or qualification['Failure'])['classification']
    if not qualification['deterministic']:
        return 'nondeterministic'
    if any(run.get('replay') and not run['replay']['match'] for run in qualification['executions']):
        return 'replay_divergence'
    if not qualification['target_success']:
        return 'target_failure'
    return 'qualified'


def spec(case, root, provisional, source_manifest=None, canonical=False):
    target = TARGETS[case['cell']]
    manifest = dict(Provisional=provisional, ID='rust-qualify-' + case['name'], Workload=case['workload'],
                    IntegratedSourceSHA256='sha256:' + parity.digest(source_manifest.read_bytes() if source_manifest
                                                                       else (CORPUS / 'qualify-corpus.json').read_bytes()),
                    Seed=7, Repeat=2, Unsupported=['weak-memory and data races', 'cross-host exact portability'])
    if not provisional:
        manifest.update(Cell='success' if case['qualified'] else 'canary', IntegratedSourceManifestPath=str(source_manifest))
    campaign = dict(CANONICAL[case['cell']]['campaign'] if canonical or case['cell'] == 'onebox' else LANE_CAMPAIGN)
    target_spec = dict(Backend='wasm', Kind=target['kind'], Source=target['source'],
                       WorkingDir=str((MODULE / target['directory']).resolve()), Args=case['arguments'], BuildTags=target['tags'],
                       PreparationRoot=str(root / 'preparation'))
    if 'scenario' in target:
        target_spec['Scenario'] = base64.b64encode((MODULE / target['scenario']).read_bytes()).decode()
    campaign.update(Target=target_spec, OnFailure='all', FailureBudget=1, Artifacts=str(root / 'artifacts'),
                    RunnerBuild='qualify-parity-lane')
    return dict(Manifest=manifest, ArtifactRoot=str(root), Campaign=campaign)


def canonical_installation(prepare, cell, destination):
    """A copy of the lane's Rust installation pinned to the owner's fuel and memory."""
    installation = parity.decode((prepare / f'rust-{TARGETS[cell]["installation"]}-installation.json').read_bytes())
    installation.update(fuel=CANONICAL[cell]['fuel'], memory_bytes=CANONICAL[cell]['memory'])
    destination.write_bytes(parity.canonical(installation))
    return destination


def wait_for_quiet_host(limit=2.5):
    """Canonical timing runs need the host to themselves."""
    while float(Path('/proc/loadavg').read_text().split()[0]) >= limit:
        time.sleep(15)


def authenticate(response, expected_root):
    """Authenticates a Rust Result and receipt on raw bytes; returns the Result."""
    path = Path(response['result_path'])
    equal(path.parent, expected_root / 'rust-qualifications' / 'v1', 'result.directory')
    raw = path.read_bytes()
    equal(path.name, 'result-' + parity.digest(raw) + '.json', 'result.content_address')
    result = parity.decode(raw)
    equal(parity.canonical(result) + b'\n', raw, 'result.canonical')
    equal(result['schema'], 'gomad.rust-qualification-result/v1', 'result.schema')
    equal(oct(path.stat().st_mode & 0o777), oct(0o600), 'result.mode')
    receipt_raw = Path(str(path) + '.publication.json').read_bytes()
    receipt = parity.decode(receipt_raw)
    equal(parity.canonical(receipt) + b'\n', receipt_raw, 'receipt.canonical')
    equal(receipt, response['operator_receipt'], 'receipt.response')
    equal((receipt['schema'], receipt['result_sha256']), ('gomad.rust-qualification-receipt/v1', parity.digest(raw)), 'receipt.binding')
    observations = result['observations']
    equal(receipt['operator_process_id'], observations['operator_process_id'], 'receipt.process')
    for key in ('start', 'prepared', 'executed'):
        equal(observations[key] is not None and observations[key]['linked']['peak_rss_bytes'] > 0, True, 'observations.' + key)
    report = result['report']
    manifest = result['manifest']
    for execution in report['executions']:
        record = execution['record']
        parity.record_contract(record, 'result.execution')
        equal(parity.digest(parity.canonical(record)), execution['record_sha256'], 'result.record_sha256')
        bound = (record['identity']['target'], record['identity']['toolchain'])
        if manifest['scenario_sha256'] is None:
            bound += (record['identity']['configuration'],)
        equal(bound, (manifest['module_sha256'], manifest['toolchain_sha256'], manifest['host_config_sha256'])[:len(bound)],
              'result.record_identity')
        artifact = check.authenticate_artifact(Path(execution['artifact_path']))
        equal(artifact, record, 'result.artifact_record')
        equal(Path(execution['artifact_path']).is_relative_to(Path(execution['campaign_path'])), True, 'result.artifact_containment')
        equal(Path(execution['campaign_path']).is_relative_to(expected_root / 'artifacts' / 'campaigns'), True, 'result.campaign_root')
    return path, result


def stable(result, response_status, tokens):
    """The declared qualification golden projection; variable measurements are excluded."""
    manifest = dict(result['manifest'])
    manifest['integrated_source_manifest_path'] = manifest['integrated_source_manifest_path'] and '$SOURCE_MANIFEST'
    report = result['report']
    executions = []
    for execution in report['executions']:
        campaign = Path(execution['campaign_path'])
        token = tokens.setdefault(campaign.name, f'campaign-{len(tokens) + 1}')
        executions.append(dict(iteration=execution['iteration'], campaign=token,
            artifact=str(Path(execution['artifact_path']).relative_to(campaign)), record=execution['record'],
            record_sha256=execution['record_sha256'], replay=execution['replay']))
    command = list(report['command'])
    equal((command[1], command[3], command[4], len(command)), ('--installation', 'qualify', '--spec', 6), 'report.command')
    command[2], command[5] = '$INSTALLATION', '$SPEC'
    cost = result['cost'] and {key: result['cost'][key] for key in (
        'executions', 'replays', 'distinct_outcomes', 'replayable_bugs', 'fuel_used', 'trace_bytes',
        'virtual_time_nanos', 'guest_peak_memory_bytes')}
    # Linked-process upper-bound statuses (and so `pending`/`within_budget`)
    # follow measured RSS; the golden keeps their names and limits, and every
    # exact check's status. The binary's own recomputation and expected_status
    # authenticate the measured statuses on each run.
    budget = result['budget'] and dict(violations=result['budget']['violations'],
        checks=[{key: check_row[key] for key in {'name', 'quantity', 'limit'}
                 | ({'status'} if check_row['quantity'] != 'linked process upper bound' else set())}
                for check_row in result['budget']['checks']])
    return dict(status=response_status, manifest=manifest, report=dict(
        {key: report[key] for key in ('schema', 'qualified', 'deterministic', 'target_success', 'seed', 'repeat',
                                       'record_sha256', 'first_divergence', 'failure')}, command=command, executions=executions),
        cost=cost, cost_failure=result['cost_failure'], budget=budget, budget_failure=result['budget_failure'],
        samples=[{key: sample[key] for key in ('phase', 'iteration', 'fuel_used', 'guest_peak_memory_bytes',
                                                'guest_memory_limit_bytes', 'trace_bytes', 'virtual_time_nanos')}
                 for sample in result['observations']['samples']])


def go_scenario_reference(run, args, case, installation, go_spec):
    """The Go host's public `qualify --spec` refuses typed scenarios (its owner
    forces cooperative choice coverage); Go qualifies clusters only through its
    in-package test API. Record that refusal, then derive the reference verdict
    from the same qualification steps on Go's public verbs: Repeat fresh
    `explore --scenario` campaigns and a fresh `replay` of every retained artifact."""
    refused = run('go-qualify-' + case['name'] + '-refused', [args.go_binary, '--installation', installation,
                  'qualify', '--spec', go_spec], 2)
    failure = json.loads(Path(json.loads(refused.stdout)['result_path']).read_bytes())['Qualification']['failure']
    equal((failure['classification'], 'typed scenarios require' in failure['message']), ('invalid_input', True),
          case['name'] + '.go_public_refusal')
    target = TARGETS[case['cell']]
    identities, replays, outcomes = [], [], []
    for attempt in range(2):
        root = args.work / f'go-{case["name"]}-explore-{attempt}'
        events = run(f'go-explore-{case["name"]}-{attempt}', [args.go_binary, '--installation', installation, 'explore', '--json',
                     '--scenario', MODULE / target['scenario'], '--seeds=7', '--parallel=1', '--on-failure=all',
                     f'--execution-timeout={LANE_EXECUTION_TIMEOUT}ns', f'--overall-timeout={LANE_OVERALL_TIMEOUT}ns',
                     '--keep-successes=all', '--success-limit=1', '--success-bytes=256MiB', '--artifacts', root,
                     '--working-dir', (MODULE / target['directory']).resolve(), target['kind'], target['source']], None)
        rows = [json.loads(line) for line in events.stdout.splitlines() if line.strip()]
        artifacts = [row for row in rows if row.get('type') == 'artifact']
        equal(len(artifacts), 1, case['name'] + '.go_retained_artifact')
        outcomes.append(artifacts[0]['classification'])
        identities.append(Path(artifacts[0]['path']).name)
        replay = run(f'go-replay-{case["name"]}-{attempt}', [args.go_binary, '--installation', installation, 'replay',
                     artifacts[0]['path']], None)
        replays.append(replay.returncode == 0 and b'reproduced=true' in replay.stdout + replay.stderr
                       or replay.returncode == 1 and outcomes[-1] != 'success' and b'reproduced=true' in replay.stdout + replay.stderr)
    deterministic = identities[0] == identities[1] and outcomes[0] == outcomes[1]
    target_success = all(outcome == 'success' for outcome in outcomes)
    qualified = deterministic and target_success and all(replays)
    classification = ('nondeterministic' if not deterministic else 'replay_divergence' if not all(replays)
                      else 'target_failure' if not target_success else 'qualified')
    status = {'qualified': 0, 'nondeterministic': 1, 'replay_divergence': 1, 'target_failure': 1}[classification]
    return dict(status=status, qualified=qualified, deterministic=deterministic, target_success=target_success,
                classification=classification)


def unattributed(result, response):
    """Linked-process peaks over a former operator-only or helper-only ceiling
    stay pending (never passed and never charged to one owner)."""
    terminal = response['operator_receipt']['linked_after']['peak_rss_bytes'] > 2 << 30
    return bool(result['budget'] and result['budget']['pending']) or terminal


def expected_status(case, result, response, rust):
    """The declared verdict status, or 3 when only the unattributable linked
    peak gate is open. That gate must name its reason."""
    if not unattributed(result, response):
        return case['status']
    equal(b'cannot be attributed' in rust.stderr or b'unattributed' in rust.stderr, True, case['name'] + '.attribution_reason')
    equal((result['cost_failure'], result['budget']['violations']), (None, []), case['name'] + '.only_attribution_pending')
    return 3


def tamper(run, binary, path, work):
    """Every mutation of retained raw bytes or bindings must be refused (status 3)."""
    directory = path.parent

    def copy(name):
        destination = work / name / 'rust-qualifications' / 'v1'
        destination.mkdir(parents=True, mode=0o700)
        for parent in (work / name, work / name / 'rust-qualifications'):
            parent.chmod(0o700)
        for child in directory.iterdir():
            shutil.copy2(child, destination / child.name)
            (destination / child.name).chmod(0o600)
        return destination / path.name

    def rewrite(name, mutate, receipt_mutate=None):
        copied = copy(name)
        result = parity.decode(copied.read_bytes())
        mutate(result)
        encoded = parity.canonical(result) + b'\n'
        target = copied.parent / ('result-' + parity.digest(encoded) + '.json')
        receipt_path = Path(str(copied) + '.publication.json')
        receipt = parity.decode(receipt_path.read_bytes())
        receipt['result_sha256'] = parity.digest(encoded)
        if receipt_mutate:
            receipt_mutate(receipt)
        copied.unlink()
        receipt_path.unlink()
        target.write_bytes(encoded)
        target.chmod(0o600)
        Path(str(target) + '.publication.json').write_bytes(parity.canonical(receipt) + b'\n')
        Path(str(target) + '.publication.json').chmod(0o600)
        return target

    cases = []
    flipped = copy('flipped-byte')
    raw = bytearray(flipped.read_bytes())
    raw[raw.index(b'"qualified":') + len(b'"qualified":')] ^= 0x01
    flipped.write_bytes(bytes(raw))
    cases.append(('flipped-byte', flipped))

    def observation(result):
        result['observations']['executed']['linked']['user_nanos'] += 1
    cases.append(('observation-recomputed', rewrite('observation', observation)))

    def verdict(result):
        result['report']['qualified'] = not result['report']['qualified']
    cases.append(('verdict-claim', rewrite('verdict', verdict)))

    def record(result):
        result['report']['executions'][1]['record']['virtual_time_nanos'] += 1
        execution = result['report']['executions'][1]
        execution['record_sha256'] = parity.digest(parity.canonical(execution['record']))
    cases.append(('record-rehashed', rewrite('record', record)))

    def missing(result):
        result['observations']['samples'][0]['guest_peak_memory_bytes'] = 0
    cases.append(('measurement-erased', rewrite('measurement', missing)))

    def cost_claim(result):
        if result['cost']:
            result['cost']['cpu_nanos'] += 1
    cases.append(('cost-claim', rewrite('cost', cost_claim)))
    cases.append(('receipt-process', rewrite('receipt-process', lambda result: None,
                                             lambda receipt: receipt.__setitem__('operator_process_id', receipt['operator_process_id'] + 1))))
    unbound = copy('receipt-unbound')
    receipt_path = Path(str(unbound) + '.publication.json')
    receipt = parity.decode(receipt_path.read_bytes())
    receipt['result_sha256'] = '0' * 64
    receipt_path.write_bytes(parity.canonical(receipt) + b'\n')
    cases.append(('receipt-unbound', unbound))
    absent = copy('receipt-absent')
    Path(str(absent) + '.publication.json').unlink()
    cases.append(('receipt-absent', absent))
    noncanonical = copy('noncanonical')
    pretty = json.dumps(parity.decode(noncanonical.read_bytes()), indent=1, sort_keys=True, ensure_ascii=False).encode() + b'\n'
    renamed = noncanonical.parent / ('result-' + parity.digest(pretty) + '.json')
    os.rename(Path(str(noncanonical) + '.publication.json'), Path(str(renamed) + '.publication.json'))
    noncanonical.unlink()
    renamed.write_bytes(pretty)
    renamed.chmod(0o600)
    cases.append(('noncanonical', renamed))
    for name, target in cases:
        run('rust-report-refuse-' + name, [binary, 'report', '--result', target], 3)
    return [name for name, _ in cases]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-binary', 'go-helper', 'go-source', 'go-compiler', 'input', 'work', 'preparation-root'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--cells', default='fixture,fixture-canary,sqlite,cluster')
    args = parser.parse_args()
    for key, value in vars(args).items():
        if isinstance(value, Path):
            setattr(args, key, value.resolve())
    args.work.mkdir(parents=True, exist_ok=False)
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_') and key != 'GOROOT'}
    environment.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2')
    receipts = []

    def run(name, argv, status, timeout=1800):
        result = subprocess.run(list(map(str, argv)), env=environment, capture_output=True, timeout=timeout)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, argv)), status=result.returncode, expected=status,
                             stdout_sha256=parity.digest(result.stdout), stderr_sha256=parity.digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(parity.canonical(receipts) + b'\n')
        if status is not None:
            equal(result.returncode, status, f'{name}.status: {result.stderr.decode(errors="replace")[-2000:]}')
        return result

    prepare = args.work / 'prepare'
    run('prepare', [sys.executable, CORPUS / 'qualify-check.py', '--prepare-only', '--binary', args.binary,
        '--go-binary', args.go_binary, '--go-source', args.go_source, '--go-helper', args.go_helper,
        '--go-compiler', args.go_compiler, '--input', args.input, '--work', prepare,
        '--preparation-root', args.preparation_root], 0)
    declaration = parity.decode((CORPUS / 'qualify-corpus.json').read_bytes())
    selected = args.cells.split(',')
    cases = [case for case in declaration['cases'] if case['name'] in selected]
    equal([case['name'] for case in cases], selected, 'corpus.cells')

    # Live grammar and owner-spec refusals; none compiles or executes a workload.
    for case in declaration['grammar']:
        argv = [str(args.work / 'missing-spec.json') if item == '$MISSING_SPEC' else item for item in case['argv']]
        result = run('rust-grammar-' + case['name'], [args.binary, *argv], case['status'])
        equal(case['stderr_contains'].encode() in result.stderr, True, 'grammar.' + case['name'] + '.stderr')
    for case in declaration['refusals']:
        result = run('rust-refusal-' + case['name'], [args.binary, *case['argv']], case['status'])
        equal(b'unsupported command' in result.stderr, True, 'refusal.' + case['name'])
    run('rust-report-grammar', [args.binary, 'report'], 2)
    run('rust-report-unknown-flag', [args.binary, 'report', '--json'], 2)
    base = spec(cases[0], args.work / 'refused', True)
    invalid = {
        'options': (dict(base, Options={}), 2),
        'unknown-field': (dict(base, Campaign=dict(base['Campaign'], Guide=True)), 2),
        'backend': (dict(base, Campaign=dict(base['Campaign'], Target=dict(base['Campaign']['Target'], Backend='native'))), 3),
        'scenario': (dict(base, Campaign=dict(base['Campaign'], Target=dict(base['Campaign']['Target'], Scenario={}))), 3),
        'repeat': (dict(base, Manifest=dict(base['Manifest'], Repeat=1)), 3),
        'final-without-source-manifest': (dict(base, Manifest=dict(base['Manifest'], Provisional=False, Cell='success')), 3),
    }
    for name, (value, status) in invalid.items():
        path = args.work / ('refused-' + name + '.json')
        path.write_bytes(parity.canonical(value))
        run('rust-spec-refuse-' + name, [args.binary, '--installation', prepare / 'rust-fixture-installation.json',
            'qualify', '--spec', path], status)
    duplicate = args.work / 'refused-duplicate.json'
    duplicate.write_bytes(b'{"Manifest":{},"Manifest":{},"Campaign":{}}')
    run('rust-spec-refuse-duplicate', [args.binary, 'qualify', '--spec', duplicate], 2)
    trailing = args.work / 'refused-trailing.json'
    trailing.write_bytes(parity.canonical(base) + b' {}')
    run('rust-spec-refuse-trailing', [args.binary, 'qualify', '--spec', trailing], 2)

    views, verdicts, tampered = [{}, {}], {}, []
    for case in cases:
        cell = case['cell']
        go_root = args.work / ('go-' + case['name'])
        go_spec = args.work / ('go-' + case['name'] + '.json')
        go_spec.write_bytes(parity.canonical(spec(case, go_root, True)))
        go_installation = prepare / f'go-{TARGETS[cell]["installation"]}-installation.json'
        if 'scenario' in TARGETS[cell]:
            go_verdict = go_scenario_reference(run, args, case, go_installation, go_spec)
        else:
            go = run('go-qualify-' + case['name'], [args.go_binary, '--installation', go_installation,
                     'qualify', '--spec', go_spec], case['status'])
            go_response = json.loads(go.stdout)
            go_result = json.loads(Path(go_response['result_path']).read_bytes())
            go_report = run('go-report-' + case['name'], [args.go_binary, '--installation', '/missing/report-installation.json',
                            'report', '--result', go_response['result_path']], 0)
            equal(json.loads(go_report.stdout)['Qualification'], go_result['Qualification'], case['name'] + '.go_report')
            go_verdict = dict(status=go.returncode, qualified=go_result['Qualification']['qualified'],
                              deterministic=go_result['Qualification']['deterministic'],
                              target_success=go_result['Qualification']['target_success'],
                              classification=go_classification(go_result['Qualification']))
        rust_verdicts = []
        for attempt in range(2):
            root = args.work / f'rust-{case["name"]}-{attempt}'
            rust_spec = args.work / f'rust-{case["name"]}-{attempt}.json'
            rust_spec.write_bytes(parity.canonical(spec(case, root, True)))
            rust = run(f'rust-qualify-{case["name"]}-{attempt}', [args.binary, '--installation',
                       prepare / f'rust-{TARGETS[cell]["installation"]}-installation.json', 'qualify', '--spec', rust_spec], None)
            response = parity.decode(rust.stdout)
            equal(len(rust.stdout) <= 8 << 10, True, 'response.bound')
            path, result = authenticate(response, root)
            equal(rust.returncode, expected_status(case, result, response, rust), f'rust-qualify-{case["name"]}-{attempt}.status')
            report = run(f'rust-report-{case["name"]}-{attempt}', [args.binary, '--installation', '/missing/report-installation.json',
                         'report', '--result', path], 0)
            reported = parity.decode(report.stdout)
            equal(reported['result'], result, 'report.result')
            classification = reported['classification']
            rust_verdicts.append(dict(status=rust.returncode, qualified=result['report']['qualified'],
                                      deterministic=result['report']['deterministic'],
                                      target_success=result['report']['target_success'], classification=classification))
            views[attempt][case['name']] = stable(result, rust.returncode, {})
            if attempt == 0 and case['name'] == cases[0]['name']:
                tampered = tamper(run, args.binary, path, args.work / 'tampered')
        equal(rust_verdicts[1], rust_verdicts[0], case['name'] + '.rust_repeated_verdict')
        semantic = lambda verdict: {key: value for key, value in verdict.items() if key != 'status'}
        equal(semantic(rust_verdicts[0]), semantic(go_verdict), case['name'] + '.go_rust_verdict')
        equal(go_verdict['status'], case['status'], case['name'] + '.go_status')
        equal(go_verdict['qualified'], case['qualified'], case['name'] + '.declared_verdict')
        verdicts[case['name']] = go_verdict
        equal(views[1][case['name']], views[0][case['name']], case['name'] + '.rust_repeated_stable_view')

    # Canonical cells: final (non-provisional) Rust runs with the owner's
    # declared budgets unchanged, on a quiet host. They enforce the operator
    # baseline, conductor source manifest and unchanged cost/budget gate.
    sources = args.work / 'integrated-source.sha256'
    files = sorted({path for case in cases for path in (MODULE / TARGETS[case['cell']]['directory']).resolve().glob('*')
                    if path.is_file()})
    sources.write_bytes(b''.join(f'{parity.digest(path.read_bytes())}  {path}\n'.encode() for path in files))
    finals = {}
    for case in cases:
        root = args.work / f'rust-canonical-{case["name"]}'
        final_spec = args.work / f'rust-canonical-{case["name"]}.json'
        final_spec.write_bytes(parity.canonical(spec(case, root, False, sources, canonical=True)))
        installation = canonical_installation(prepare, case['cell'], args.work / f'rust-canonical-{case["name"]}-installation.json')
        wait_for_quiet_host()
        rust = run(f'rust-canonical-{case["name"]}', [args.binary, '--installation', installation,
                   'qualify', '--spec', final_spec], None)
        response = parity.decode(rust.stdout)
        path, result = authenticate(response, root)
        equal(rust.returncode, expected_status(case, result, response, rust), f'rust-canonical-{case["name"]}.status')
        equal(result['integrated_source_manifest'], sources.read_text(), 'canonical.source_manifest')
        equal((result['manifest']['fuel'], result['manifest']['memory_bytes']),
              (CANONICAL[case['cell']]['fuel'], CANONICAL[case['cell']]['memory']), 'canonical.installation_bounds')
        for execution in result['report']['executions']:
            equal(execution['record']['limits']['wall_millis'],
                  CANONICAL[case['cell']]['campaign']['ExecutionTimeout'] // 1_000_000, 'canonical.execution_wall')
        equal((result['cost_failure'], result['budget_failure'], result['budget']['violations']), (None, None, []),
              'canonical.budget')
        equal(result['budget']['within_budget'], not result['budget']['pending'], 'canonical.budget_within')
        finals[case['name']] = stable(result, rust.returncode, {})
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    goldens = CORPUS / 'goldens-qualify'
    targets = [(goldens / 'tamper.json', [dict(schema='gomad.rust-qualify-tamper-golden/v1', refused=tampered)] * 2)]
    for case in cases:
        targets.append((goldens / 'cells' / (case['name'] + '.json'),
                        [dict(schema='gomad.rust-qualify-golden/v1', cell=views[attempt][case['name']],
                              canonical=finals[case['name']], go_verdict=verdicts[case['name']]) for attempt in range(2)]))
    for golden, (first_view, second_view) in targets:
        parity.golden(golden, first_view, regenerate)
        first = golden.read_bytes()
        parity.golden(golden, second_view, regenerate)
        equal(golden.read_bytes(), first, str(golden) + ' second regeneration bytes')
    print(f'qualify: {len(cases)} cells match Go verdicts; Rust results authenticated, repeated, refused when tampered')


if __name__ == '__main__':
    main()
