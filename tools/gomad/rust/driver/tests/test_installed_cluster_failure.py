"""Counterexamples for the raw reader; these supply no installed CLI execution credit."""
from pathlib import Path
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import installed_cluster_failure as gate
from driver import Refusal


class OriginalOperatorFailure(unittest.TestCase):
    def test_exact_original_explore_arguments_include_preparation_in_wall(self):
        args = gate.explore_arguments(Path('/private'), 'capacity')
        self.assertEqual(args, ['explore', '--json', '--scenario', '/private/capacity.json',
            '--seeds=7', '--parallel=1', '--on-failure=all', '--keep-successes=all',
            '--success-limit=32', '--success-bytes=536870912', '--execution-timeout=3m',
            '--overall-timeout=4m', '--terminate-grace=100ms', '--artifacts',
            '/private/capacity-artifacts', '--working-dir', '/private/source', 'go-run', '.'])

    def test_late_or_shortened_outer_interval_cannot_pass_child_timing(self):
        command = dict(start_ns=100, end_ns=200, started_monotonic_nanos=50,
                       reaped_monotonic_nanos=150)
        wall = dict(start_ns=99, end_ns=201, started_monotonic_nanos=49,
                    reaped_monotonic_nanos=151)
        gate.prompt_interval(command, wall)
        for field, value in [('start_ns', 101), ('end_ns', 199),
                             ('started_monotonic_nanos', 51), ('reaped_monotonic_nanos', 149),
                             ('reaped_monotonic_nanos', 10_000_000_050)]:
            with self.subTest(field=field, value=value), self.assertRaises(Refusal):
                gate.prompt_interval(command, dict(wall, **{field: value}))

    def test_native_capture_source_manifest_is_bound_to_main_build_digest(self):
        raw = b'abc  /repo/source.rs\n'
        build = dict(inputs={'/repo/source.rs': 'abc', '/outside/generated': 'def'},
                     source_sha256=gate.receiving.digest(raw))
        self.assertEqual(gate.source_manifest(build, Path('/repo')), raw)
        for name, value in [('/repo/source.rs', 'changed'), ('/repo/extra.rs', 'abc')]:
            changed = dict(build, inputs=dict(build['inputs'], **{name: value}))
            with self.subTest(name=name), self.assertRaises(Refusal):
                gate.source_manifest(changed, Path('/repo'))

    def test_top_level_inspection_totals_cannot_be_replaced_by_cluster_totals(self):
        streams = {('a', 1, 'stdout'): b'client', ('b', 1, 'stdout'): b'server',
                   ('a', 1, 'stderr'): b'partial'}
        report = dict(artifact=dict(stdout=dict(total_bytes=12), stderr=dict(total_bytes=7)))
        gate.inspection_totals(report, streams)
        for stream in ('stdout', 'stderr'):
            changed = dict(artifact=dict(report['artifact'], **{stream: dict(total_bytes=0)}))
            with self.subTest(stream=stream), self.assertRaises(Refusal):
                gate.inspection_totals(changed, streams)

    def test_native_capture_reaches_campaign_refusal_after_real_source_and_binary_pins(self):
        import native_observations
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in ('compiler', 'source', 'runtime', 'module-cache'):
                (root / name).mkdir()
                (root / name / 'input').write_bytes(b'input')
            binary = root / 'gomad'
            binary.write_bytes(b'not executable; parser counterexample only')
            raw = b'actual pinned source manifest'
            (root / 'source-inputs.tsv').write_bytes(raw)
            gate.write(root / 'producer.json', dict(pid=1, start_ns=1, started_monotonic_nanos=1))
            build = dict(inputs={str(binary): gate.sha(binary)}, binary=str(binary),
                         binary_sha256=gate.sha(binary), source_sha256=gate.receiving.digest(raw),
                         host_sha256='a' * 64, helper_sha256='b' * 64)
            run = gate.Run(root, dict(repo=str(root)), build, True)
            with self.assertRaisesRegex(ValueError, 'empty/duplicate private campaign'):
                native_observations.capture(run.config, 2, [])

    def test_dependency_cache_is_explicit_and_guest_object_cache_is_not_prewarmed(self):
        environment = gate.environment(Path('/private'))
        self.assertEqual(environment['GOMODCACHE'], '/private/module-cache')
        self.assertEqual(environment['HOME'], '/private/home')
        self.assertNotIn('GOCACHE', environment)

    def test_actual_command_requires_completed_owned_distinct_pid(self):
        owner = dict(pid=70, start_ns=99, started_monotonic_nanos=40)
        row = dict(pid=71, parent_pid=70, argv=['gomad', 'discover'], cwd='/repo',
                   start_ns=100, end_ns=200, started_monotonic_nanos=50,
                   reaped_monotonic_nanos=150, exit=0, observation='completed',
                   clock='CLOCK_MONOTONIC', environment_mode='exact', effective_environment={'GOENV': 'off'})
        process = dict(row, stage_pid=70, active=False)
        gate.command_identity(row, process, owner, row['argv'], '/repo', 0, {'GOENV': 'off'}, set())
        for mutation in ('pid', 'parent', 'active', 'exit', 'observation', 'reused', 'argv', 'cleanup'):
            changed = dict(process)
            seen = set()
            argv = row['argv']
            if mutation == 'pid': changed['pid'] = 72
            elif mutation == 'parent': changed['stage_pid'] = 72
            elif mutation == 'active': changed['active'] = True
            elif mutation == 'exit': changed['exit'] = 1
            elif mutation == 'observation': changed['observation'] = 'inconclusive'
            elif mutation == 'reused': seen.add(71)
            elif mutation == 'cleanup': changed['cleanup_signal'] = 9
            else: argv = ['gomad', 'explore']
            with self.subTest(mutation=mutation), self.assertRaises(Refusal):
                gate.command_identity(row, changed, owner, argv, '/repo', 0, {'GOENV': 'off'}, seen)


if __name__ == '__main__':
    unittest.main()
