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
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

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
    'fixture': dict(directory='internal/backends/wasm/qualification/testdata/fixture', kind='go-run', source='.', tags=[]),
    'sqlite': dict(directory='../..', kind='go-run', source='./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go',
                   tags=['sqlite3_dotlk']),
    'onebox': dict(directory='../..', kind='go-test', source='./tests/gomadfunctional', tags=['test_dep']),
}


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


def spec(case, root, provisional, source_manifest=None):
    target = TARGETS[case['cell']]
    manifest = dict(Provisional=provisional, ID='rust-qualify-' + case['name'], Workload=case['cell'],
                    IntegratedSourceSHA256='sha256:' + parity.digest(source_manifest.read_bytes() if source_manifest
                                                                       else (CORPUS / 'qualify-corpus.json').read_bytes()),
                    Seed=7, Repeat=2, Unsupported=['weak-memory and data races', 'cross-host exact portability'])
    if not provisional:
        manifest.update(Cell='success' if case['qualified'] else 'canary', IntegratedSourceManifestPath=str(source_manifest))
    return dict(Manifest=manifest, ArtifactRoot=str(root), Campaign=dict(
        Target=dict(Backend='wasm', Kind=target['kind'], Source=target['source'],
                    WorkingDir=str((MODULE / target['directory']).resolve()), Args=case['arguments'], BuildTags=target['tags'],
                    PreparationRoot=str(root / 'preparation')),
        ExecutionTimeout=300_000_000_000, OverallTimeout=1_200_000_000_000, TerminateGrace=100_000_000,
        OnFailure='all', FailureBudget=1, OutputLimit=8 << 20, WorldTransitionLimit=64 << 20,
        Artifacts=str(root / 'artifacts'), RunnerBuild='qualify-parity-lane'))


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
        equal((record['identity']['target'], record['identity']['toolchain'], record['identity']['configuration']),
              (manifest['module_sha256'], manifest['toolchain_sha256'], manifest['host_config_sha256']), 'result.record_identity')
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
    budget = result['budget'] and dict(within_budget=result['budget']['within_budget'],
        violations=result['budget']['violations'], pending=result['budget']['pending'],
        checks=[{key: check_row[key] for key in ('name', 'quantity', 'limit', 'status')} for check_row in result['budget']['checks']])
    return dict(status=response_status, manifest=manifest, report=dict(
        {key: report[key] for key in ('schema', 'qualified', 'deterministic', 'target_success', 'seed', 'repeat',
                                       'record_sha256', 'first_divergence', 'failure')}, command=command, executions=executions),
        cost=cost, cost_failure=result['cost_failure'], budget=budget, budget_failure=result['budget_failure'],
        samples=[{key: sample[key] for key in ('phase', 'iteration', 'fuel_used', 'guest_peak_memory_bytes',
                                                'guest_memory_limit_bytes', 'trace_bytes', 'virtual_time_nanos')}
                 for sample in result['observations']['samples']])


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
    parser.add_argument('--cells', default='fixture,fixture-canary,sqlite')
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
        go = run('go-qualify-' + case['name'], [args.go_binary, '--installation', prepare / f'go-{cell}-installation.json',
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
                       prepare / f'rust-{cell}-installation.json', 'qualify', '--spec', rust_spec], case['status'])
            response = parity.decode(rust.stdout)
            equal(len(rust.stdout) <= 8 << 10, True, 'response.bound')
            path, result = authenticate(response, root)
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
        equal(rust_verdicts[0], go_verdict, case['name'] + '.go_rust_verdict')
        equal(go_verdict['qualified'], case['qualified'], case['name'] + '.declared_verdict')
        verdicts[case['name']] = go_verdict
        equal(views[1][case['name']], views[0][case['name']], case['name'] + '.rust_repeated_stable_view')

    # Final (non-provisional) cells enforce the operator baseline, conductor
    # source manifest and unchanged cost/budget gate on the Rust binary.
    sources = args.work / 'integrated-source.sha256'
    files = sorted(path for path in (MODULE / TARGETS['fixture']['directory']).resolve().rglob('*') if path.is_file())
    sources.write_bytes(b''.join(f'{parity.digest(path.read_bytes())}  {path}\n'.encode() for path in files))
    finals = {}
    for case in cases:
        if case['cell'] != 'fixture':
            continue
        root = args.work / f'rust-final-{case["name"]}'
        final_spec = args.work / f'rust-final-{case["name"]}.json'
        final_spec.write_bytes(parity.canonical(spec(case, root, False, sources)))
        rust = run(f'rust-final-{case["name"]}', [args.binary, '--installation', prepare / 'rust-fixture-installation.json',
                   'qualify', '--spec', final_spec], case['status'])
        path, result = authenticate(parity.decode(rust.stdout), root)
        equal(result['integrated_source_manifest'], sources.read_text(), 'final.source_manifest')
        equal((result['cost_failure'], result['budget_failure'], result['budget']['within_budget']), (None, None, True), 'final.budget')
        finals[case['name']] = stable(result, rust.returncode, {})
    view = dict(schema='gomad.rust-qualify-golden/v1', cells=views[0], final=finals, go_verdicts=verdicts, tampered=tampered)
    golden = CORPUS / 'goldens-qualify' / 'cells.json'
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    parity.golden(golden, view, regenerate)
    first = golden.read_bytes()
    parity.golden(golden, dict(view, cells=views[1]), regenerate)
    equal(golden.read_bytes(), first, 'second regeneration bytes')
    print(f'qualify: {len(cases)} cells match Go verdicts; Rust results authenticated, repeated, refused when tampered')


if __name__ == '__main__':
    main()
