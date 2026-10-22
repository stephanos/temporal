#!/usr/bin/env python3
"""Execute roots 13-15: preserve real guest failures before markers or retention."""
import base64
import json
from pathlib import Path
import runpy

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal = common['Lane'], common['equal']
MODULE, REPOSITORY = common['MODULE'], common['REPOSITORY']


def scenario(action, marker=None):
    value = json.loads((MODULE / 'internal/backends/wasm/command/testdata/scenario/durable.json').read_bytes())
    value['spec']['limits'] = dict(nodes=64, directional_links=4096, volumes=64,
        scenario_actions=4096, scenario_decisions=4096, fault_actions=4096,
        history_operations=1 << 16, observations=1 << 16, oracle_results=4096,
        scenario_evidence_bytes=64 << 20, boot_config_bytes=1 << 20, volume_capacity_bytes=1 << 30,
        observation_bytes=64 << 20, network_listeners=4096, network_connections=65536,
        network_bytes=1 << 30, network_connection_bytes=4 << 20, network_stall_nanos=10000000000,
        network_transitions=1 << 20, volume_operations=1 << 20, volume_transitions=1 << 20,
        crash_states=1 << 16, crash_depth=256, crash_bytes=128 << 20, crash_wall_nanos=60000000000)
    value['spec']['nodes'] = [dict(id='a', boot='main', address='10.0.0.1')]
    value['spec']['volumes'] = []
    value['boots'] = [dict(id='main')]
    value.pop('incarnation_boots')
    value.pop('output_crashes')
    handle = dict(node='a', incarnation=1)
    value['actions'] = [dict(action='start', handle=handle), dict(action=action, handle=handle)]
    if marker is not None:
        value['output_crashes' if action == 'crash' else 'output_stops'] = [
            dict(handle=handle, marker=base64.b64encode(marker.encode()).decode())]
    return value


def main():
    lane = Lane(__doc__)
    installation = lane.installation('markers', fuel=1 << 60)
    cases = [
        (13, action, './common/gomaddebug/testdata/fixture', REPOSITORY,
         'after initial runtime choices', 'runtime-choice-bytes', True)
        for action in ('crash', 'stop')]
    cases += [
        (14, action, '.', MODULE / 'internal/backends/wasm/testdata/unsupported_clock',
         'after CPU clock call', 'CPU clocks outside stock WASI profile', False)
        for action in ('crash', 'stop')]
    cases += [(15, 'wait', './common/gomaddebug/testdata/fixture', REPOSITORY,
               None, 'runtime-choice-bytes', True)]
    for root_id, action, source, working, marker, reason, capacity in cases:
        name = str(root_id) + '-' + action
        path = lane.args.work / (name + '-scenario.json')
        path.write_bytes(common['encode'](scenario(action, marker)))
        flags = ['--scenario', path, '--build-tag=gomad', '--build-tag=test_dep']
        if capacity:
            flags += ['--choices', '--choice-bytes=160']
        root, output = lane.explore(name, ['go-run', source], working, flags,
                                    status=(3,), installation=installation, overall='180s', seeds='0')
        error = json.loads(output.stdout.splitlines()[-1])
        equal(reason in error['message'], True, name + '.original_reason')
        equal(('capacity' if capacity else 'unsupported') in error['message'].lower(),
              True, name + '.classification')
        equal('marker was not' in error['message'].lower(), False, name + '.precedence')
        checked = lane.authenticate(root, complete=False)
        equal(checked['executions'], [], name + '.no_fabricated_outcome')
        lane.roots.append(dict(root=root_id, original={
            13: 'TestScenarioPreservesChoiceCapacityBeforeDeclaredMarker',
            14: 'TestScenarioPreservesUnsupportedBeforeDeclaredMarker',
            15: 'TestScenarioPreservesChoiceCapacityBeforeRetention'}[root_id], variant=action, reason=reason))
    lane.finish()


if __name__ == '__main__':
    main()
