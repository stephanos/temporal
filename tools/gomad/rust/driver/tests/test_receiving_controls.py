import copy
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import receiving

FLAGS = ['GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', 'GOMAD_WASM_TEMPORAL_BUILD_CACHE', 'GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES']


class ReceivingControls(unittest.TestCase):
    def setUp(self):
        platform = patch.object(receiving.sys, 'platform', 'linux')
        platform.start()
        self.addCleanup(platform.stop)
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.config = dict(root=str(self.root), environment={FLAGS[2]: '8192'})
        self.owner = dict(argv=['producer.py', '--cache', '/private/cache'])
        self.controls = dict(guest_profile='', requested_build_cache='/private/cache', build_cache='/private/cache', memory_minimum='8192', memory_platform=sys.platform,
                             memory_path='/proc/meminfo', memory=dict(minimum_bytes=8192, available_bytes=8192))
        (self.root/'host-memory.raw').write_bytes(b'MemAvailable: 8 kB\n')
        self.save()

    def save(self):
        (self.root/'qualification-controls.json').write_bytes(receiving.canonical(self.controls))
        for stage in ['install', 'qualify', 'report']:
            value = dict(environment={flag: self.config['environment'].get(flag) for flag in FLAGS},
                         argv=['/sealed/release', 'install', '--cache', '/private/cache'] if stage == 'install' else ['/sealed/release', stage])
            (self.root/(stage+'.command.json')).write_bytes(receiving.canonical(value))

    def verify(self):
        return receiving.Retained(self.config).private_controls(self.root, self.owner)

    def test_bound_raw_admission_and_frozen_private_controls(self):
        self.assertEqual(self.verify(), self.controls)

    def test_missing_tampered_or_nonexact_memory_observation_refuses(self):
        for raw in [b'MemAvailable: 7 kB\n', b'MemAvailable: 8 MB\n', b'MemAvailable: 8 kB\nMemAvailable: 8 kB\n', b' '*(8193), b'MemAvailable:\x1c8 kB\n']:
            with self.subTest(raw=raw[:40]):
                (self.root/'host-memory.raw').write_bytes(raw)
                with self.assertRaises(driver.Refusal):
                    self.verify()
        (self.root/'host-memory.raw').unlink()
        with self.assertRaises((driver.Refusal, FileNotFoundError)):
            self.verify()

    def test_self_reported_available_bytes_cannot_replace_raw_measurement(self):
        self.controls['memory']['available_bytes'] = 9000
        self.save()
        with self.assertRaises(driver.Refusal):
            self.verify()

    def test_metadata_cannot_replace_frozen_env_or_cache_pin(self):
        original = copy.deepcopy(self.controls)
        for field, value in [('guest_profile', 'full'), ('requested_build_cache', '/forged/requested'), ('build_cache', '/shared/other'), ('memory_minimum', '1'), ('memory_path', '/foreign'), ('memory_platform', 'foreign')]:
            with self.subTest(field=field):
                self.controls = dict(original, **{field: value})
                self.save()
                with self.assertRaises(driver.Refusal):
                    self.verify()

    def test_nested_command_env_cannot_disagree_with_frozen_controls(self):
        path = self.root/'qualify.command.json'
        path.write_bytes(receiving.canonical(dict(environment={flag: 'forged' for flag in FLAGS})))
        with self.assertRaises(driver.Refusal):
            self.verify()

    def test_absent_optin_requires_absent_raw_file_and_zero_observation(self):
        self.config['environment'] = {}
        self.controls['memory_minimum'] = ''
        self.controls['memory'] = dict(minimum_bytes=0, available_bytes=0)
        self.save()
        with self.assertRaises(driver.Refusal):
            self.verify()
        (self.root/'host-memory.raw').unlink()
        self.assertEqual(self.verify(), self.controls)

    def test_threshold_overflow_and_reading_overflow_refuse(self):
        for threshold, raw in [('0', b'MemAvailable: 8 kB\n'), ('8193', b'MemAvailable: 8 kB\n'), ('18446744073709551616', b'MemAvailable: 8 kB\n'), ('1', b'MemAvailable: 18014398509481984 kB\n')]:
            with self.subTest(threshold=threshold):
                self.config['environment'][FLAGS[2]] = threshold
                self.controls['memory_minimum'] = threshold
                self.save()
                (self.root/'host-memory.raw').write_bytes(raw)
                with self.assertRaises(driver.Refusal):
                    self.verify()

    def test_omitted_command_controls_and_boolean_measurements_refuse(self):
        (self.root/'report.command.json').write_bytes(receiving.canonical(dict(environment={})))
        with self.assertRaises(driver.Refusal):
            self.verify()
        self.config['environment'][FLAGS[2]] = '1'
        self.controls['memory_minimum'] = '1'
        self.controls['memory']['minimum_bytes'] = True
        self.save()
        with self.assertRaises(driver.Refusal):
            self.verify()

    def test_actual_command_observation_matches_closed_detached_environment(self):
        directory = Path(receiving.__file__).resolve().parents[1]/'workload_tools'
        sys.path.insert(0, str(directory))
        self.addCleanup(lambda: sys.path.remove(str(directory)))
        producer = driver.load_module('receiving_controls_fixture_producer', directory/'producer.py')
        environment = dict(PATH='', GOMAD_WASM_INSTALLATION=str(self.root/'installation-unavailable.json'),
                           GOMAD_STOCK_GO=str(self.root/'compiler-unavailable'), GOMAD_WASM_RUNTIME_ROOT=str(self.root/'runtime-unavailable'),
                           GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES='8192')
        producer.run(self.root, 'inspection-fixture', [sys.executable, '-c', 'print("{}")'], 0, environment=environment)
        actual = driver.read(self.root/'inspection-fixture.command.json')['environment']
        self.assertEqual(len(actual),8)
        self.assertIsNone(actual['GOMAD_WASM_QUALIFY_WORKLOADS'])
        receiving.inspection_environment(self.config, self.root, actual)
        for changed in [dict(actual,GOMAD_WASM_QUALIFY_WORKLOADS='1'),{key:value for key,value in actual.items() if key!='GOMAD_WASM_QUALIFY_WORKLOADS'},dict(actual, PATH='/borrowed'), dict(actual, unknown='forged'), {key:value for key,value in actual.items() if key not in FLAGS}]:
            with self.assertRaises(driver.Refusal):
                receiving.inspection_environment(self.config, self.root, changed)

    def test_canonical_cache_alias_binding_survives_alias_and_cache_detachment(self):
        real = self.root/'actual-cache'
        real.mkdir()
        alias = self.root/'requested-cache'
        alias.symlink_to(real, target_is_directory=True)
        self.owner['argv'][-1] = str(alias)
        self.config['environment'][FLAGS[1]] = str(alias)
        self.controls['requested_build_cache'] = str(alias)
        self.controls['build_cache'] = str(real)
        self.save()
        path = self.root/'install.command.json'
        command = driver.read(path)
        command['argv'] = ['/sealed/release', 'install', '--cache', str(real)]
        path.write_bytes(receiving.canonical(command))
        alias.unlink()
        real.rmdir()
        self.assertEqual(self.verify(), self.controls)
