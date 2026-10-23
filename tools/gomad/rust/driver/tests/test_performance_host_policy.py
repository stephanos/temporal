"""Owner policy and raw host witness mutations; no host producer is executed."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parents[1]))
import final_profiles as gate
from driver import Refusal, sha


class PerformanceHostPolicy(unittest.TestCase):
    def fixture(self, root, machine='x86_64'):
        directory = root / 'host'
        directory.mkdir()
        milestone = root / 'MILESTONES.md'
        milestone.write_text('### WASM performance measurement host (owner decision, 2026-10-17)\n\n'
            'WASM performance measurements may run on any host; no host platform qualifies or disqualifies a WASM result.\n')
        def probe(label, argv, value):
            observed = dict(schema='gomad.native-process-observation/v1', command=argv, directory=str(directory), environment={},
                pid=123, clock='CLOCK_MONOTONIC', parent_pid=122, started_monotonic_nanos=1, exit_code=0, signal=None, termination='exit', error=None, cleanup_errors=[], duration_nanos=1, reaped_monotonic_nanos=2)
            for stream in ('stdout', 'stderr'):
                path = directory / (label + '.' + stream)
                path.write_text(value + '\n' if stream == 'stdout' else '')
                observed[stream] = dict(path=str(path), bytes=path.stat().st_size, captured_bytes=path.stat().st_size,
                    sha256=sha(path), truncated=False, error=None)
                observed[stream + '_sha256'] = sha(path)
            (directory / (label + '.json')).write_text(json.dumps(observed))
        witness = dict(kernel_os='Linux', machine=machine, kernel_version='unit kernel', cpuinfo='vendor_id: unit CPU\n', translation='')
        for label, option, field in (('uname-os', '-s', 'kernel_os'), ('uname-machine', '-m', 'machine'), ('uname-kernel', '-v', 'kernel_version')):
            probe(label, ['uname', option], witness[field])
        files = {'host-witness.json': json.dumps(witness), 'host-cpuinfo.txt': witness['cpuinfo'], 'host-boot-id.txt': 'unit boot\n'}
        for name, value in files.items(): (directory / name).write_text(value)
        host = dict(schema='gomad.native-measurement-host/v1', host_os='linux', host_arch=machine, boot_id='unit boot', witness=witness,
                    inputs={str(directory / name): sha(directory / name) for name in files})
        (root / 'host.json').write_text(json.dumps(host))
        return dict(repo=str(root), files={str(milestone): sha(milestone)}), host

    def test_any_host_policy_does_not_apply_superseded_arm64_selector(self):
        for machine in ('x86_64', 'aarch64'):
            with self.subTest(machine=machine), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                config, _ = self.fixture(root, machine)
                result = gate.current_host_evidence(config, root)
                self.assertEqual(result['policy'], gate.PERFORMANCE_HOST_POLICY)
                self.assertEqual(result['original_policy']['applicable'], machine == 'aarch64')
                self.assertEqual(result['original_policy']['observed_predicate'], False if machine == 'aarch64' else None)

    def test_raw_host_changes_and_foreign_uname_command_are_refused(self):
        for mutation in ('cpu', 'command', 'parent', 'interval', 'clock', 'missing-clock', 'policy'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                config, host = self.fixture(root)
                if mutation == 'cpu':
                    path = root / 'host/host-cpuinfo.txt'
                    path.write_text('coherently rehashed foreign CPU')
                    host['inputs'][str(path)] = sha(path)
                    (root / 'host.json').write_text(json.dumps(host))
                elif mutation in ('command', 'parent', 'interval', 'clock', 'missing-clock'):
                    path = root / 'host/uname-os.json'
                    row = json.loads(path.read_text())
                    if mutation == 'command': row['command'] = ['caller-claimed-host']
                    elif mutation == 'parent': row['parent_pid'] = 999
                    elif mutation == 'clock': row['clock'] = 'CLOCK_REALTIME'
                    elif mutation == 'missing-clock': row.pop('clock')
                    else: row['started_monotonic_nanos'] = row['reaped_monotonic_nanos']
                    path.write_text(json.dumps(row))
                else:
                    path = root / 'MILESTONES.md'
                    path.write_text('caller policy')
                    config['files'][str(path)] = sha(path)
                with self.assertRaises(Refusal): gate.current_host_evidence(config, root)
