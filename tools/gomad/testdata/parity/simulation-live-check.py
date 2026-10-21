#!/usr/bin/env python3
"""Exercise public Rust simulation exploration over real guests and retained model evidence."""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess

CORPUS = Path(__file__).resolve().parent
MODULE = CORPUS.parents[1]


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-compiler', 'input', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    args.binary, args.go_compiler, args.input, args.work = (
        path.resolve() for path in (args.binary, args.go_compiler, args.input, args.work))
    args.work.mkdir(parents=True, exist_ok=False)
    environment = {name: value for name, value in os.environ.items() if not name.startswith('GOMAD_')}
    receipts = []

    def run(name, command, statuses=(0,)):
        result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=900)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(encode(receipts))
        assert result.returncode in statuses, (name, result.returncode, result.stderr.decode(errors='replace'))
        return result.stdout

    identity = args.work / 'identity'
    bundle = json.loads(run('identity', [args.binary, 'fixture-replay-input', '--root', identity, '--input', args.input]))
    cooperative = next(identity / row['artifact'] for row in bundle['records']
                       if json.loads((identity / row['artifact'] / 'record.json').read_bytes())['profile'] == 'gomad.wasi-cooperative/v1')
    replay = json.loads((cooperative / 'evidence/replay.json').read_bytes())
    configuration = replay['configuration']
    model = configuration['model']
    model['mode'], model['read_step_nanos'] = 1, 0
    model.pop('seed', None)
    namespace = configuration['namespace']
    namespace['writable_directories'] = [list(b'/workspace')]
    namespace['preopens'] = [list(b'/workspace')]
    namespace['working_directory'] = list(b'/workspace')
    model['working_directory_sha256'] = digest(b'/workspace')
    namespace['limits']['filesystem_bytes'] = 8 << 20
    network = dict(profile='gomad.simulation-network-stream/v3', limits=dict(listeners=8, connections=8, bytes=1 << 20,
        connection_bytes=1 << 20, stall_nanos=10_000_000_000, transitions=10000), faults=[])
    installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(MODULE / 'internal/model/runtimesource'), cache_root=str(args.work / 'preparation'),
        profile='gomad.wasi-cooperative/v1', fuel=1 << 60, memory_bytes=256 << 20,
        helper_sha256=replay['helper_sha256'], host_sha256=replay['host_sha256'],
        runtime_implementation_sha256=replay['runtime_implementation_sha256'], model=model,
        namespace=namespace, storage=None, network=network, build_tags=[])
    manifest = args.work / 'installation.json'
    manifest.write_bytes(encode(installation))

    scenario = json.loads((MODULE / 'internal/backends/wasm/command/testdata/scenario/durable.json').read_bytes())
    scenario['boots'][0]['args'] = ['simulation-live']
    scenario['boots'][1]['args'] = ['simulation-recover']
    scenario['output_crashes'][0]['marker'] = base64.b64encode(b'pending simulation write').decode()
    scenario['spec']['limits']['network_connection_bytes'] = 1 << 20
    scenario['network_faults'] = [
        dict(Name='partition', Action='partition', From='a', To='b', AtNanos=1),
        dict(Name='heal', Action='heal', From='a', To='b', AtNanos=1),
    ]
    scenario_file = args.work / 'scenario.json'
    scenario_file.write_bytes(encode(scenario))
    prefix = [args.binary, '--installation', manifest]
    destination = args.work / 'artifacts'
    run('explore', [*prefix, 'explore', '--json', '--strategy=simulation-exploration', '--choices',
        '--on-failure=all', '--parallel=1', '--seeds=89', '--max-executions=16', '--max-forced-decisions=1',
        '--max-exploration-bytes=128MiB', '--max-exploration-result-bytes=16MiB',
        *['--max-' + dimension + '-decisions=2' for dimension in ('runtime', 'scenario', 'network', 'storage', 'fault', 'crash')],
        '--keep-successes=all', '--success-limit=32', '--success-bytes=512MiB',
        '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
        '--working-dir', MODULE, '--scenario', scenario_file, 'go-run',
        './internal/backends/wasm/command/testdata/scenario'], (0, 1))
    roots = list((destination / 'campaigns').glob('campaign-*'))
    assert len(roots) == 1, roots
    campaign = roots[0]
    run('inspect', [*prefix, 'inspect', '--json', campaign])
    run('resume', [*prefix, 'resume', '--json', campaign], (0, 1))

    observed, forced, observations = set(), set(), []
    for path in campaign.rglob('simulation/record.json'):
        record = json.loads(path.read_bytes())
        assert record['schema'] == 'gomad.cluster-record/v8', path
        assert record['producer'] == 'gomad.rust-live-models/v1', path
        artifact = path.parent.parent
        execution = json.loads((artifact / 'record.json').read_bytes())
        assert execution['outcome']['class'] != 'runner_failure', execution
        for decision in record['exploration_decisions']:
            observed.add(decision['dimension'])
            assert len(decision['alternatives']) > 1
            assert decision['selected'] < len(decision['alternatives'])
        for override in record['exploration_plan']['overrides'] or []:
            forced.add(override['dimension'])
        observations.append(dict(identity=record['identity'], outcome=record['outcome'],
            decisions=record['exploration_decisions'], observations=record['observations']))
    assert observations, 'no retained actual simulation observations'
    required = {'scenario', 'network', 'storage', 'fault', 'crash'}
    assert observed >= required, ('missing actual model decisions', sorted(required - observed))
    assert forced >= required | {'runtime'}, ('missing real forced executions', sorted((required | {'runtime'}) - forced))
    (args.work / 'observations.json').write_bytes(encode(dict(schema='gomad.rust-simulation-live-check/v1',
        binary_sha256=digest(args.binary.read_bytes()), dimensions=sorted(observed), forced_dimensions=sorted(forced),
        observations=observations)))


if __name__ == '__main__':
    main()
