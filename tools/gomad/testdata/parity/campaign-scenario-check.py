#!/usr/bin/env python3
"""Exercise public typed seed scenario admission, lifecycle and retained replay."""
import argparse
import copy
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
loader = SourceFileLoader('campaign_check', str(CORPUS / 'campaign-check.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'installation', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    args.binary, args.installation, args.work = [path.resolve() for path in (args.binary, args.installation, args.work)]
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    module = CORPUS.parents[1]
    fixtures = module / 'internal/backends/wasm/command/testdata/scenario'
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    commands = []

    def run(name, arguments, status=0, installation=None):
        command = [args.binary, '--installation', installation or args.installation, *arguments]
        result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=600)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode, expected=status,
                             stdout_sha256=check.parity.digest(result.stdout), stderr_sha256=check.parity.digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(check.parity.canonical(commands) + b'\n')
        check.equal(result.returncode, status, name + ': ' + result.stdout.decode(errors='replace') + result.stderr.decode(errors='replace'))
        return result.stdout

    def explore(scenario, destination):
        return ['explore', '--json', '--scenario', scenario, '--seeds=7', '--parallel=1', '--on-failure=all',
                '--execution-timeout=30s', '--overall-timeout=10m', '--keep-successes=all', '--success-limit=4',
                '--success-bytes=256MiB', '--artifacts', destination, '--working-dir', module,
                'go-run', './internal/backends/wasm/command/testdata/scenario']

    views = []
    network_installation = copy.deepcopy(check.parity.read(args.installation))
    limits = check.parity.read(fixtures / 'network.json')['spec']['limits']
    network_installation['network'] = dict(profile='gomad.simulation-network-stream/v3', faults=[], limits={
        name: limits['network_' + name] for name in ('listeners', 'connections', 'bytes', 'connection_bytes', 'stall_nanos', 'transitions')})
    network_manifest = args.work / 'network-installation.json'
    network_manifest.write_bytes(check.parity.canonical(network_installation))
    for repetition in range(2):
        view = {}
        for name in ('durable', 'network'):
            destination = args.work / f'{name}-{repetition}'
            run(f'{name}-{repetition}', explore(fixtures / (name + '.json'), destination),
                installation=network_manifest if name == 'network' else None)
            roots = list((destination / 'campaigns').glob('campaign-*'))
            check.equal(len(roots), 1, name + '.campaign_count')
            root = roots[0]
            validated = check.authenticate_campaign(root)
            check.equal(len(validated['executions']), 1, name + '.executions')
            execution = validated['executions'][0]
            artifact = root / execution['artifact']
            retained = check.canonical_file(artifact / 'evidence/cluster-replay.json')
            handles = [(row['handle']['node'], row['handle']['incarnation']) for row in retained['retained']]
            check.equal(len(handles), 3 if name == 'durable' else 4, name + '.full_lifecycle')
            inspected = check.parity.decode(run(f'{name}-inspect-{repetition}', ['inspect', '--json', root]))
            check.equal(inspected['campaign']['executions'][0]['record'], execution['record'], name + '.inspect.record')
            run(f'{name}-replay-{repetition}', ['replay', artifact])
            view[name] = dict(plan=validated['plan'], journal=validated['journal'], records=check.execution_view(validated))
        views.append(view)
    check.equal(views[0], views[1], 'scenario.fresh_repeated_projection')

    original = check.parity.read(fixtures / 'durable.json')
    mutations = [
        ('unknown', lambda value: value.__setitem__('extra', True)),
        ('nested-unknown', lambda value: value['spec']['nodes'][0].__setitem__('extra', True)),
        ('capacity', lambda value: value['spec']['limits'].__setitem__('nodes', 1)),
        ('unknown-limit', lambda value: value['spec']['limits'].__setitem__('extra', 1)),
        ('unsorted', lambda value: value['spec']['nodes'].reverse()),
        ('unknown-boot', lambda value: value['spec']['nodes'][0].__setitem__('boot', 'missing')),
        ('mount-mismatch', lambda value: value['boots'][0]['storage']['volumes'][0].__setitem__('CapacityBytes', 1)),
        ('restart-storage', lambda value: value['boots'][1].__setitem__('storage', None)),
        ('missing-boundary', lambda value: value.__setitem__('output_crashes', [])),
        ('marker', lambda value: value['output_crashes'][0].__setitem__('marker', '!!!!')),
        ('unused-binding', lambda value: value['incarnation_boots'][0]['handle'].__setitem__('incarnation', 3)),
        ('incomplete', lambda value: value['actions'].pop()),
        ('running-restart', lambda value: value['actions'][2].__setitem__('action', 'restart')),
        ('unknown-action', lambda value: value['actions'][0].__setitem__('action', 'run')),
    ]
    for name, mutate in mutations:
        value = copy.deepcopy(original)
        mutate(value)
        source = args.work / ('invalid-' + name + '.json')
        source.write_bytes(check.parity.canonical(value))
        run('refuse-' + name, explore(source, args.work / ('invalid-' + name)), 2)
    for index, raw in enumerate((b'{} {}', b'{', b'', b'{"faults":null,"faults":null}')):
        source = args.work / f'malformed-{index}.json'
        source.write_bytes(raw)
        run(f'refuse-malformed-{index}', explore(source, args.work / f'malformed-{index}'), 2)
    golden = CORPUS / 'goldens-campaign/scenarios.json'
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    first = None
    for view in views:
        check.parity.golden(golden, dict(schema='gomad.seed-scenario-golden/v1', cases=view), regenerate)
        if first is None:
            first = golden.read_bytes()
        else:
            check.equal(golden.read_bytes(), first, 'scenario.second_regeneration_bytes')
    print('public typed seed scenarios, authenticated replay and strict refusals passed')


if __name__ == '__main__':
    main()
