"""Original private qualification controls; bounded fixtures, no tool dispatch."""
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import producer
import workloads


class QualificationControls(unittest.TestCase):
    def test_original_guest_default_slim_full_unknown_and_fresh_return(self):
        slim = ['disable_grpc_modules', 'gomad_wasm_slim', 'sqlite3_dotlk', 'test_dep']
        for profile in ['', 'slim']:
            self.assertEqual(workloads.guest_build_tags(profile), slim)
        self.assertEqual(workloads.guest_build_tags('full'), [slim[0], *slim[2:]])
        for profile in ['unknown', 'SLIM', ' full', 'full ', '\x00']:
            with self.subTest(profile=profile), self.assertRaises(ValueError):
                workloads.guest_build_tags(profile)
        workloads.guest_build_tags('').clear()
        self.assertEqual(workloads.guest_build_tags(''), slim)

    def test_specs_bind_full_variant_without_changing_other_inputs(self):
        for workload in ['replication', 'recovery', 'smoke', 'frontend']:
            function = workloads.support_spec if workload in workloads.SUPPORT_ROWS else workloads.qualification_spec
            slim = function('/candidate', '/evidence', workload, source='./unchanged')
            full = function('/candidate', '/evidence', workload, source='./unchanged', guest_profile='full')
            full['Campaign']['Target']['BuildTags'].insert(1, 'gomad_wasm_slim')
            self.assertEqual(full, slim)

    def test_original_private_default_and_explicit_shared_cache(self):
        self.assertEqual(producer.qualification_build_cache('/attempt'), Path('/attempt/temporal-cluster-build-cache'))
        self.assertEqual(producer.qualification_build_cache('/attempt', '/verified/shared'), Path('/verified/shared'))
        for value in ['relative/cache', '/', '/verified/../redirected', '/verified//cache', '/verified/cache/', '//verified/cache']:
            with self.subTest(value=value), self.assertRaises(ValueError):
                producer.qualification_build_cache('/attempt', value)

    def observation(self, raw, minimum, platform='linux'):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary)/'meminfo'
            path.write_bytes(raw)
            return producer.memory_admission(platform, path, minimum)

    def test_original_linux_available_boundary(self):
        raw = b'MemTotal: 100 kB\nMemAvailable: 8 kB\n'
        with self.assertRaises(ValueError):
            self.observation(raw, '8193')
        for minimum in ['8192', '8191']:
            self.assertEqual(self.observation(raw, minimum), dict(minimum_bytes=int(minimum), available_bytes=8192))

    def test_original_malformed_and_oversized_linux_observations(self):
        rows = ['', 'MemTotal: 100 kB\n', 'MemAvailable: 8 kB\nMemAvailable: 8 kB\n',
                'memavailable: 8 kB\n', 'MemAvailable: -1 kB\n', 'MemAvailable: +1 kB\n',
                'MemAvailable: 1 MB\n', 'MemAvailable:\x1c8 kB\n', 'MemAvailable: 1 kB extra\n', 'MemAvailable: 1\n',
                'MemAvailable: 8 kB\nMemAvailable : 8 kB\n', 'MemAvailable: 8 kB\nMemAvailable:8 kB\n',
                'MemAvailable: 18446744073709551616 kB\n', 'MemAvailable: 18014398509481984 kB\n',
                'MemAvailable: 8 kB\n'+' '*(8 << 10)]
        for raw in rows:
            with self.subTest(raw=raw[:80]), self.assertRaises(ValueError):
                self.observation(raw.encode(), '1')

    def test_original_explicit_threshold_platform_and_no_optin_no_read(self):
        for minimum in ['0', '-1', '+1', ' 1', '18446744073709551616', '１']:
            with self.subTest(minimum=minimum), self.assertRaises(ValueError):
                producer.memory_admission('linux', Path('/unreadable'), minimum)
        with self.assertRaises(ValueError):
            producer.memory_admission('darwin', Path('/unreadable'), '1')
        with patch.object(Path, 'open', side_effect=AssertionError('unexpected read')):
            self.assertEqual(producer.memory_admission('darwin', Path('/unreadable'), ''), dict(minimum_bytes=0, available_bytes=0))

    def test_explicit_bad_admission_stops_real_producer_before_any_dispatch(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)/'fresh'
            arguments = ['producer.py', '--binary', '/abs/binary', '--repo', '/candidate', '--root', str(root),
                         '--compiler', '/abs/compiler', '--runtime-root', '/abs/runtime', '--cache', '/abs/cache',
                         '--workload', 'replication', '--mode', 'success']
            with patch.object(sys, 'argv', arguments), patch.dict(os.environ, {'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES': '0'}), patch.object(producer, 'run', side_effect=AssertionError('must not dispatch')):
                with self.assertRaisesRegex(ValueError, 'threshold'):
                    producer.main()

    def test_admission_retains_exact_raw_bytes_before_success_or_refusal(self):
        for minimum in ['8192', '8193']:
            with tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                original = root/'input'
                raw = b'MemAvailable: 8 kB\n'
                original.write_bytes(raw)
                retained = root/'host-memory.raw'
                if minimum == '8193':
                    with self.assertRaises(ValueError):
                        producer.memory_admission('linux', original, minimum, retained)
                else:
                    producer.memory_admission('linux', original, minimum, retained)
                self.assertEqual(retained.read_bytes(), raw)
                self.assertEqual(retained.stat().st_mode & 0o777, 0o600)
                with self.assertRaises(FileExistsError):
                    producer.memory_admission('linux', original, '8192', retained)
                self.assertEqual(retained.read_bytes(), raw)

    def test_go_unicode_fields_accepts_nbsp_but_not_extra_python_separators(self):
        self.assertEqual(self.observation('MemAvailable:\u00a08\u00a0kB\n'.encode(), '1')['available_bytes'], 8192)
        with self.assertRaises(ValueError):
            self.observation(b'MemAvailable:\x1c8 kB\n', '1')

    def test_competing_private_cache_refuses_before_dispatch(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)/'fresh'
            arguments = ['producer.py', '--binary', '/abs/binary', '--repo', '/candidate', '--root', str(root),
                         '--compiler', '/abs/compiler', '--runtime-root', '/abs/runtime', '--cache', '/abs/cache',
                         '--workload', 'replication', '--mode', 'success']
            with patch.object(sys, 'argv', arguments), patch.dict(os.environ, {'GOMAD_WASM_TEMPORAL_BUILD_CACHE': '/other/cache'}), patch.object(producer, 'run', side_effect=AssertionError('must not dispatch')):
                with self.assertRaisesRegex(ValueError, 'differs from frozen'):
                    producer.main()

    def test_accepted_cache_alias_records_the_same_canonical_install_path(self):
        class DispatchStopped(Exception):
            pass
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            real = base/'real-cache'
            real.mkdir()
            alias = base/'accepted-cache-alias'
            alias.symlink_to(real, target_is_directory=True)
            root = base/'fresh'
            arguments = ['producer.py', '--binary', '/abs/binary', '--repo', '/candidate', '--root', str(root),
                         '--compiler', '/abs/compiler', '--runtime-root', '/abs/runtime', '--cache', str(alias),
                         '--captured-config', str(base/'fixture-config'), '--workload', 'smoke', '--mode', 'success']
            commands = []
            def stop(path, name, argv, *args, **kwargs):
                commands.append(argv)
                raise DispatchStopped()
            env = {name: '' for name in ['GOMAD_WASM_TEMPORAL_BUILD_CACHE', 'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES', 'GOMAD_WASM_QUALIFICATION_GUEST_PROFILE']}
            env['GOMAD_WASM_TEMPORAL_BUILD_CACHE'] = str(alias)
            with patch.object(sys, 'argv', arguments), patch.dict(os.environ, env), patch.object(producer, 'captured', return_value={'schema': 'bounded-fixture'}), patch.object(producer, 'run', side_effect=stop):
                with self.assertRaises(DispatchStopped):
                    producer.main()
            controls = producer.json.loads((root/'qualification-controls.json').read_bytes())
            installed = commands[0][commands[0].index('--cache')+1]
            self.assertEqual(controls['build_cache'], installed)
            self.assertEqual(installed, str(real))
            self.assertEqual(controls['requested_build_cache'], str(alias))
