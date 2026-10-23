"""Original two-cluster baseline and fresh per-incarnation prefix/full replay."""
import argparse
import os
import stat
from pathlib import Path
import sys

from driver import command, configured, guard, read, require, sha, write
import live
import receiving

HANDLES = {(node, number) for node in ('active', 'standby') for number in (1, 2)}


def baseline(config, root):
    stage = Path(config['root']) / 'stages/clusters'
    require(root == stage / 'primary/replication', 'original clusters has a foreign owning stage')
    owner = read(root / 'producer-root.json')
    outer = read(stage / 'primary-workload.command.json')
    require(owner['pid'] == outer['pid'] and owner['argv'] == outer['argv'][1:]
            and outer['start_ns'] <= owner['start_ns'] <= outer['end_ns'],
            'original cluster baseline borrowed another producer')
    reader = receiving.Retained(config)
    result = reader.producer(root, 'replication', original_clusters=True)
    reader.verify_pending_preparations()
    require(result['manifest']['repeat'] == 1 and len(result['report']['executions']) == 1,
            'original clusters baseline must execute exactly once')
    execution = result['report']['executions'][0]
    require(execution['replay']['attempted'] is True and execution['replay']['matched'] is True
            and execution['replay']['choice_replay'] == 'exact', 'original cluster baseline lost its exact retained replay')
    artifact = Path(execution['artifact_path'])
    require(artifact.is_relative_to(root), 'original clusters borrowed another baseline')
    manifest = reader.artifact(artifact)
    inputs = read(artifact / 'evidence/cluster-replay.json')
    require(inputs['helper_sha256'] == config['identity']['helper'], 'original clusters has another network owner')
    summary = read(artifact / 'evidence/cluster_summary')
    owners = {(item['handle']['node'], item['handle']['incarnation']): item for item in summary['incarnations']}
    require(set(owners) == HANDLES and len(summary['incarnations']) == 4, 'original clusters omitted an incarnation')
    identities, histories = [], []
    for node in ('active', 'standby'):
        first, last = owners[node, 1], owners[node, 2]
        require(first['termination'] == 'crash' and first['interrupted'] is True
                and last['termination'] == 'exit' and last['exit_code'] == 0 and last['interrupted'] is False,
                'original cluster crash/reopen terminals changed')
        before = receiving.history((artifact / f'evidence/cluster/{node}/1/stdout').read_bytes())
        after = receiving.history((artifact / f'evidence/cluster/{node}/2/stdout').read_bytes())
        require(before['cluster_id'] == after['cluster_id'] and before['history'] == after['history']
                and sorted(after['event_ids']) == [1, 2, 3, 4, 5], 'original committed cluster history changed on reopen')
        identities.append(after['cluster_id']); histories.append(after['history'])
    require(identities[0] != identities[1] and histories[0] == histories[1],
            'original independent clusters did not converge after heal')
    return artifact, manifest, inputs, dict(summary, original_outcome=manifest['record']['outcome'])


def payload(artifact, descriptor, manifest):
    name = descriptor['file']
    require(isinstance(name, str) and name and not Path(name).is_absolute() and '..' not in Path(name).parts,
            'original forced replay has an unsafe payload path')
    bound = manifest['files'].get(name)
    require(bound is not None and all(bound[key] == descriptor[key] for key in ('bytes', 'sha256'))
            and descriptor['bytes'] <= descriptor['limit'], 'original forced replay payload is not retained')
    path = receiving.regular(artifact / name)
    require(path.stat().st_size == descriptor['bytes'] and sha(path) == descriptor['sha256'],
            'original forced replay payload changed')


def replay_input(artifact, inputs, manifest):
    configuration = inputs['configuration']
    incarnations = {(row['handle']['node'], row['handle']['incarnation']): row for row in configuration['incarnations']}
    retained = {(row['handle']['node'], row['handle']['incarnation']): row for row in inputs['retained']}
    require(set(incarnations) == set(retained) == HANDLES
            and len(configuration['incarnations']) == len(inputs['retained']) == 4,
            'original forced replay omitted or duplicated an owner')
    modules, forcing = {}, {}
    for (node, number), row in incarnations.items():
        descriptor = row['module']
        payload(artifact, descriptor, manifest)
        modules[descriptor['file']] = str(artifact / descriptor['file'])
        retained_row = retained[node, number]
        require(retained_row['interrupted'] is (number == 1), 'original prefix/full phase assignment changed')
        choice = retained_row['evidence']['choice']
        payload(artifact, choice, manifest)
        require(choice['bytes'] > 0, 'original cluster replay has no actual forced tape')
        forcing[f'{node}/{number}'] = dict(trace_path=str(artifact / choice['file']), mode=3 if number == 1 else 2,
                                          identity=row['choice_identity'])
    return dict(schema='gomad.live-cluster-observation/v1', configuration=configuration, modules=modules, forcing=forcing)


def replay_file(path):
    path = receiving.regular(path)
    value = path.stat()
    require(value.st_nlink == 1 and value.st_uid == os.geteuid() and stat.S_IMODE(value.st_mode) == 0o600,
            'forced replay output is not a fresh private regular file')
    return path


def compare(artifact, summary, output):
    observed = read(replay_file(output / 'result.json'))
    require(read(receiving.regular(output.parent / 'observe.stdout')) == observed,
            'actual forced replay command output differs from its private result')
    require(observed['schema'] == 'gomad.live-cluster-observation/v1' and observed['execution_returned'] is True
            and observed['wall_elapsed_nanos'] > 0, 'original forced cluster replay did not execute')
    expected_outcome = summary['original_outcome']
    require(observed['outcome']['termination'] == expected_outcome['termination']
            and observed['outcome'].get('exit_code') == expected_outcome.get('exit_code')
            and observed['outcome']['message'] == expected_outcome['reason'],
            'original forced replay aggregate terminal differs')
    rows = {(row['handle']['node'], row['handle']['incarnation']): row for row in observed['incarnations']}
    require(set(rows) == HANDLES and len(observed['incarnations']) == 4, 'forced replay lost an original phase')
    for expected in summary['incarnations']:
        node, number = expected['handle']['node'], expected['handle']['incarnation']
        actual = rows[node, number]
        require(actual['interrupted'] == expected['interrupted']
                and actual['outcome']['termination'] == expected['termination']
                and actual['outcome'].get('exit_code') == expected['exit_code']
                and actual['outcome']['message'] == expected['reason']
                and actual['outcome'].get('divergence') is None
                and actual['authorization_to_terminal_nanos'] > 0,
                'original prefix/full replay terminal, choice or completion differs')
        require(set(actual['streams']) == set(expected['streams']), 'forced replay omitted an original evidence stream')
        for stream, descriptor in expected['streams'].items():
            original = receiving.regular(artifact / f'evidence/cluster/{node}/{number}/{stream}')
            replayed = replay_file(output / f'incarnations/{node}/{number}/{stream}')
            require(len(original.read_bytes()) == descriptor['bytes'] and sha(original) == descriptor['sha256']
                    and original.read_bytes() == replayed.read_bytes(),
                    'original forced cluster replay stream differs: ' + f'{node}/{number}/{stream}')
    for stream in ('world', 'network'):
        original = receiving.regular(artifact / ('evidence/' + stream))
        replayed = replay_file(output / ('streams/' + stream))
        require(sha(original) == summary[stream + '_sha256'] and original.read_bytes() == replayed.read_bytes(),
                'original forced cluster replay changed shared ' + stream)
    return dict(schema='gomad.original-clusters-receiving/v1', baseline=str(artifact), baseline_record_sha256=sha(artifact / 'record.json'),
        phases=[dict(node=node, incarnation=number, forcing='interrupted-prefix' if number == 1 else 'full',
                     streams=sorted(rows[node, number]['streams'])) for node, number in sorted(HANDLES)],
        compared=['all incarnation evidence including storage', 'shared World bytes', 'shared network Record bytes'])


def observation_proof(config, artifact, summary, root):
    proof = compare(artifact, summary, root / 'observed')
    row = read(root / 'observe.command.json')
    proof.update(network_owner='rust-cluster', network_owner_implementation_sha256=config['identity']['helper'],
        forcing_input_sha256=sha(root / 'input.json'), observation_process={key: row[key]
        for key in ('pid', 'parent_pid', 'clock', 'started_monotonic_nanos', 'reaped_monotonic_nanos')})
    return proof


def collect(config, stage):
    artifact, manifest, inputs, summary = baseline(config, stage / 'primary/replication')
    root = stage / 'primary/cluster-replay'
    payload = replay_input(artifact, inputs, manifest)
    require(read(root / 'input.json') == payload, 'actual forced prefix/full input changed')
    binary = live.role(config, 'host-fixture')['binary']
    argv = [binary, 'fixture-live-cluster', '--root', str(root / 'observed'), '--input', str(root / 'input.json')]
    reader = receiving.Retained(config)
    reader.command(config, root, 'observe', argv, timeout=5400)
    from primary_stages import enclose_child
    outer = read(stage / 'primary-final.command.json')
    child, process = read(root / 'observe.command.json'), read(root / 'observe.process.json')
    for field in ('parent_pid', 'clock', 'started_monotonic_nanos', 'reaped_monotonic_nanos'):
        require(child[field] == process[field], 'forced replay process/command lifecycle differs')
    require(process['stage_pid'] == child['parent_pid'] == outer['pid'], 'forced replay borrowed another parent')
    enclose_child(outer, read(stage / 'primary-final.process.json'), child)
    proof = observation_proof(config, artifact, summary, root)
    require(read(root / 'proof.json') == proof, 'original clusters proof differs from actual raw comparison')
    return proof


def run(config, stage):
    guard(config)
    artifact, manifest, inputs, summary = baseline(config, stage / 'primary/replication')
    root = stage / 'primary/cluster-replay'
    root.mkdir(mode=0o700)
    payload = replay_input(artifact, inputs, manifest)
    write(root / 'input.json', payload)
    binary = live.role(config, 'host-fixture')['binary']
    command(config, root, 'observe', [binary, 'fixture-live-cluster', '--root', str(root / 'observed'),
                                    '--input', str(root / 'input.json')], timeout=5400)
    proof = observation_proof(config, artifact, summary, root)
    write(root / 'proof.json', proof)
    guard(config)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    args = parser.parse_args()
    config = configured(args.config)
    config.update(_config_path=str(args.config), _config_sha256=sha(args.config))
    run(config, Path(config['root']) / 'stages/clusters')


if __name__ == '__main__':main()
