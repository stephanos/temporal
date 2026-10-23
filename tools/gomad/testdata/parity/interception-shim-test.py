#!/usr/bin/env python3
"""Offline checker/byte-capture regressions, with no real Go or Gomad execution."""
import copy
import base64
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
loader = SourceFileLoader('shim_acceptance', str(HERE / 'interception-shim-check.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)


class ByteCapture(unittest.TestCase):
    def test_non_utf8_empty_and_shell_metacharacters_round_trip_without_shell(self):
        with tempfile.TemporaryDirectory(prefix='interception-shim-offline-') as directory:
            root = Path(directory)
            check.program(root / 'go', 'compiler')
            check.program(root / 'child', 'host-child')
            env = dict(os.environb, PATH=os.fsencode(root) + b':' + os.environb[b'PATH'],
                       INTERCEPTION_CAPTURE_ROOT=os.fsencode(root), LC_ALL=b'C.UTF-8',
                       INTERCEPTION_INVALID_ENV=b'\xff\x80 env value')
            # Dict keyword arguments above are text; convert the entire map to bytes.
            env = {os.fsencode(key): os.fsencode(value) for key, value in env.items()}
            result = subprocess.run([os.fsencode(root / 'child'), b'list', *check.RAW_ARGS], env=env,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stderr)
            received = [json.loads(path.read_bytes()) for path in root.glob('compiler-*.json')]
            self.assertEqual(len(received), 1)
            self.assertEqual(received[0]['argv'], [value.hex() for value in [b'list', *check.RAW_ARGS]])
            self.assertEqual(received[0]['environment'], json.loads((root / 'inherited.json').read_bytes())['environment'])
            self.assertEqual(received[0]['executable'], str(root / 'go'))

    def test_fixture_goos_uses_host_not_guest(self):
        with tempfile.TemporaryDirectory(prefix='interception-shim-offline-') as directory:
            root = Path(directory)
            check.program(root / 'go', 'compiler')
            result = subprocess.run([root / 'go', 'env', 'GOOS'], env=dict(os.environ, INTERCEPTION_CAPTURE_ROOT=str(root)),
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, (os.uname().sysname.lower() + '\n').encode())

    def test_source_case_set_covers_host_refusals_boundaries_and_cache(self):
        self.assertEqual(len(check.CASES), len(check.CORPUS['cases']))
        for command in ('env','list','mod','generate','vet','tool','version','install','clean','run','build'):
            self.assertIn(('unmatched-' if command in ('run','build') else 'host-') + command, check.CASES)
        for flag in ('race','fuzz','cover','coverprofile','coverpkg','msan','asan'):
            for channel in ('argv','goflags'):
                self.assertIn('refuse-' + flag + '-' + channel, check.CASES)
        for name in ('boundary-args','boundary-double-args','boundary-terminator','guest-lookalikes',
                     'goflags-quoted','repeated-effective-flags','empty-effective-exec','external-launcher',
                     'exec-true-count0','multi-package-json','auto-schema-mutation','auto-schema-missing',
                     'auto-changed-executable','concurrent-identities','explicit-excluded-tag'):
            self.assertTrue(check.CASES[name]['requires_real_inputs'])


class PublishedPlans(unittest.TestCase):
    def fixture(self, root):
        fixtures = check.load('shim_plan_fixtures', HERE / 'interception-check-test.py')
        artifact, receipt = fixtures.sealed_preparation(root)
        return artifact, receipt

    def test_public_shim_accepts_published_admission_and_incomplete_zero_guest_campaign(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            expected = dict(argv=['test','./supported'], campaigns=0, outcome_classes=[], limits={})
            self.assertEqual(check.authenticate_campaigns(root, expected, '1' * 64), [])

    def test_public_make_reaches_expected_count_after_authenticating_preparation(self):
        make = check.load('shim_make_plan_test', HERE / 'interception-make-check.py')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            with self.assertRaisesRegex(ValueError, 'make.three_sequential_recipes'):
                make.authenticate_positive(root, Path('/actual/gomad'), '1' * 64, '2' * 64)

    def test_complete_launcher_and_preparation_are_both_authenticated(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            fixtures = check.load('shim_complete_fixtures', HERE / 'interception-check-test.py')
            executed = root / 'campaigns/campaign-executed'
            fixtures.sealed_campaign(executed)
            fixtures.check.campaign.rechain(executed, lambda plan: plan.update(launcher=True))
            rows = check.authenticated_plans(root)
            self.assertEqual(sorted(row['complete'] for row in rows), [False, True])
            self.assertEqual([len(row['validated']['executions']) for row in rows if row['complete']], [1])

    def build_fixture(self, root, wrong=None):
        fixtures = check.load('shim_build_plan_fixtures', HERE / 'interception-check-test.py')
        artifact, old_receipt = fixtures.sealed_preparation(root)
        canonical = fixtures.check.parity.canonical
        record = json.loads((artifact / 'record.json').read_bytes())
        provenance = json.loads((artifact / 'target/provenance.json').read_bytes())
        origin = provenance.pop('origin')
        invocation = origin['invocation']
        invocation['original_argv'] = [value.encode().hex() for value in ['go','build','./main']]
        invocation['driver_argv'] = [value.encode().hex() for value in ['/stock/go','build','./main']]
        bindings = [['example/main','2'*64]]
        invocation['installation_sha256'] = check.sha(canonical(bindings))
        marker = dict(schema='gomad.build-launcher/v1', module_sha256=origin['module_sha256'],
                      installation_sha256='2'*64, invocation_sha256=check.sha(canonical(invocation)),
                      catalog_sha256=invocation['installation_sha256'], recipe_sha256=invocation['recipe_sha256'])
        if wrong:
            marker[wrong] = 'f'*64
        provenance['inputs'] = dict(shared_driver_installations=bindings, go_build_info=dict(path='example/main'),
             build_launcher=marker, build_driver_registration=dict(schema='gomad.driver-registration/v1', process_id=7,
             invocation_sha256=check.sha(canonical(invocation)), compiler_sha256=invocation['compiler_sha256'],
             driver_argv_sha256=check.sha(canonical(invocation['driver_argv'])), module_root='/private/outputs'))
        origin['preparation_sha256'] = check.sha(canonical(provenance))
        provenance['origin'] = origin
        raw_origin = canonical(origin)
        record['streams']['target/intercepted-origin.json'] = dict(bytes=len(raw_origin),sha256=check.sha(raw_origin),records=1)
        payloads = {'target/module.wasm':(artifact / 'target/module.wasm').read_bytes(),
                    'target/provenance.json':canonical(provenance), 'target/intercepted-origin.json':raw_origin}
        fixtures.seal_artifact(artifact, record, payloads)
        campaign = artifact.parent
        plan = json.loads((campaign / 'plan.json').read_bytes())
        plan['origin'] = origin
        (campaign / 'plan.json').write_bytes(canonical(plan))
        storage = json.loads((campaign / 'storage.json').read_bytes())
        storage['plan_sha256'] = check.sha(canonical(plan))
        (campaign / 'storage.json').write_bytes(canonical(storage))
        receipt = json.loads(old_receipt.read_bytes())
        old_receipt.unlink()
        old_receipt.with_name('plan.json').unlink()
        published = root / 'build-launchers' / invocation['invocation_id'] / ('d'*64) / 'plan.json'
        published.parent.mkdir(parents=True)
        published.write_bytes(canonical(plan))
        receipt.update(plan=str(published), plan_sha256=check.sha(canonical(plan)),
                       record_sha256=check.sha(canonical(record)), command=['os-hex:'+value for value in invocation['original_argv']])
        receipt.update(provenance_bytes=len(payloads['target/provenance.json']),provenance_sha256=check.sha(payloads['target/provenance.json']))
        published.with_name('plan.json.preparation.json').write_bytes(canonical(receipt))
        return published

    def test_public_shim_accepts_authenticated_build_launcher_publication(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.build_fixture(root)
            self.assertEqual(check.authenticate_campaigns(root, dict(argv=[], campaigns=0,outcome_classes=[]), '1'*64), [])

    def test_self_consistent_wrong_build_module_installation_or_invocation_marker_is_refused(self):
        for key in ('module_sha256','installation_sha256','invocation_sha256'):
            with self.subTest(key=key), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.build_fixture(root,key)
                with self.assertRaisesRegex(ValueError, 'build_launcher'):
                    check.authenticated_plans(root)

    def test_build_publication_invocation_and_original_command_cannot_be_rebound(self):
        for mode in ('invocation-directory','original-command'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                published = self.build_fixture(root)
                receipt = json.loads(published.with_name('plan.json.preparation.json').read_bytes())
                if mode == 'invocation-directory':
                    original = published.parent.parent
                    changed = original.with_name('e'*64)
                    original.rename(changed)
                    published = changed / published.parent.name / 'plan.json'
                    receipt['plan'] = str(published)
                else:
                    receipt['command'] = ['os-hex:wrong-original-command']
                published.with_name('plan.json.preparation.json').write_bytes(check.encode(receipt))
                with self.assertRaisesRegex(ValueError, 'build_launcher'):
                    check.authenticated_plans(root)

    def test_unknown_publication_namespace_is_not_skipped(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.build_fixture(root)
            (root / 'build-launchers').rename(root / 'unknown-publications')
            with self.assertRaisesRegex(ValueError, 'unknown noncampaign'):
                check.authenticated_plans(root)

    def test_published_receipt_digest_tamper_is_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            _, receipt = self.fixture(root)
            value = json.loads(receipt.read_bytes())
            value['plan_sha256'] = '0' * 64
            receipt.write_bytes(check.encode(value))
            with self.assertRaisesRegex(ValueError, 'receipt.plan_digest'):
                check.authenticated_plans(root)


class ExternalCapture(unittest.TestCase):
    def candidate(self, root):
        compiler, source, installation = [root / name for name in ('compiler','source.go','install.json')]
        compiler.write_bytes(b'offline compiler fixture, never Go')
        source.write_bytes(b'offline source inventory fixture')
        installed = dict(compiler_path=str(compiler), compiler_sha256=check.file_sha(compiler))
        installation.write_bytes(check.encode(installed))
        original = ['test','./supported','-run=^TestA$','-v','-count=1','-exec=/actual/external']
        driver = [str(compiler), *original]
        bindings = [dict(import_path='example/supported', path=str(installation), sha256=check.file_sha(installation))]
        package = dict(schema='gomad.driver-recipe/v1', import_path='example/supported', installed=installed,
                       directory=str(root), configuration={},
                       options=dict(installation=str(installation), target_kind='go-test', source='example/supported'),
                       compiler={str(compiler):check.file_sha(compiler)}, sources={str(source):check.file_sha(source)},
                       environment=dict(GOOS='wasip1', GOARCH='wasm', CGO_ENABLED='0', GOTOOLCHAIN='local'))
        package['inputs'] = dict(compiler=package['compiler'], sources=package['sources'],
                                 source='example/supported', kind='go-test', directory=str(root), configuration={})
        shared = dict(schema='gomad.shim-driver/v1', packages=[package],
                      installations=[bindings[0]['sha256']], driver_argv=[value.encode().hex() for value in driver])
        invocation = dict(schema='gomad.intercepted-invocation/v1', wrapper_run_id='a'*64, invocation_id='b'*64,
                          compiler_sha256=check.file_sha(compiler), recipe_file='driver-recipe.json',
                          original_argv=[value.encode().hex() for value in ['go',*original]], driver_argv=shared['driver_argv'],
                          installation_sha256=check.sha(check.encode([[bindings[0]['import_path'],bindings[0]['sha256']]])))
        captured = dict(invocation=invocation, argv=[value.encode().hex() for value in
                        ['/actual/temporary.wasm','-test.run=^TestA$','-test.v=true']],
                        module_bytes=8, module_sha256='c'*64,
                        inventories=[dict(compiler=package['compiler'],sources=package['sources'])])
        expected = dict(argv=original, compiler_sha256=check.file_sha(compiler), external_launcher='/actual/external',
                        launcher_arguments=['-test.run=^TestA$','-test.v=true'], installation_bindings=bindings)
        self.seal(captured, shared)
        return compiler, captured, shared, expected

    def seal(self, captured, shared):
        recipe_raw = check.encode(shared)
        captured['recipe_base64'] = base64.b64encode(recipe_raw).decode()
        captured['invocation']['recipe_sha256'] = check.sha(recipe_raw)
        raw = check.encode(captured['invocation'])
        captured['invocation_base64'] = base64.b64encode(raw).decode()
        captured['invocation_sha256'] = check.sha(raw)
        captured['environment'] = {key.encode().hex():value.encode().hex() for key,value in dict(
            GOMAD_SHIM_INVOCATION_SHA256=check.sha(raw), GOMAD_SHIM_RUN='a'*64).items()}

    def test_exact_external_capture_is_accepted_without_a_guest_claim(self):
        with tempfile.TemporaryDirectory() as directory:
            compiler, captured, _, expected = self.candidate(Path(directory))
            self.assertEqual(check.authenticate_external([captured], expected, compiler), captured)

    def test_external_fixture_captures_inventory_before_private_inputs_disappear(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            compiler, captured, shared, expected = self.candidate(root)
            invocation = root / 'invocation.json'
            invocation.write_bytes(base64.b64decode(captured['invocation_base64']))
            recipe = root / 'driver-recipe.json'
            recipe.write_bytes(base64.b64decode(captured['recipe_base64']))
            module = root / 'actual.wasm'
            module.write_bytes(b'\0asm\x01\0\0\0')
            launcher = root / 'fixture-launcher'
            check.program(launcher, 'external-launcher')
            env = dict(os.environ, INTERCEPTION_CAPTURE_ROOT=str(root), GOMAD_SHIM_INVOCATION=str(invocation),
                       GOMAD_SHIM_INVOCATION_SHA256=captured['invocation_sha256'], GOMAD_SHIM_RUN='a'*64)
            result = subprocess.run([launcher, module, *expected['launcher_arguments']], env=env,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stderr)
            observed = [json.loads(path.read_bytes()) for path in root.glob('external-*.json')]
            recipe.unlink()
            invocation.unlink()
            (root / 'source.go').unlink()
            check.authenticate_external(observed, expected, compiler)

    def test_self_consistent_wrong_original_order_compiler_installation_or_received_arguments_are_refused(self):
        for mode in ('original-order','compiler','installation','received-order','invocation','inventory'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                compiler, captured, shared, expected = self.candidate(Path(directory))
                if mode == 'original-order':
                    values = captured['invocation']['original_argv']
                    values[3],values[4] = values[4],values[3]
                elif mode == 'compiler':
                    captured['invocation']['compiler_sha256'] = 'f'*64
                elif mode == 'installation':
                    shared['installations'] = ['f'*64]
                    captured['invocation']['installation_sha256'] = check.sha(check.encode([['example/supported','f'*64]]))
                elif mode == 'received-order':
                    captured['argv'][1],captured['argv'][2] = captured['argv'][2],captured['argv'][1]
                elif mode == 'inventory':
                    captured['inventories'][0] = dict(compiler={str(compiler):'f'*64}, sources=shared['packages'][0]['sources'])
                self.seal(captured, shared)
                if mode == 'invocation':
                    captured['invocation_sha256'] = 'f'*64
                with self.assertRaises(ValueError):
                    check.authenticate_external([captured], expected, compiler)

    def test_missing_exact_capture_cannot_pass_as_external_execution(self):
        with self.assertRaisesRegex(ValueError, 'external.exactly_one_capture'):
            check.authenticate_external([], dict(argv=['test','./supported']), Path('/actual/compiler'))


class DriverFlags(unittest.TestCase):
    def test_effective_empty_exec_and_count_preserve_last_value_before_guest_boundary(self):
        flags = check.driver_flags(['-exec=false','-exec=gomad run-guest','-count=2',
                                    '-args','-exec=guest-only','-count=guest-only'])
        self.assertEqual(flags, dict(exec='gomad run-guest', count='2'))
        self.assertEqual(check.driver_flags(['-exec=false','-exec=']), dict(exec=''))
        for token in ('-args','--args','--'):
            self.assertEqual(check.driver_flags(['-exec','gomad run-guest',token,'-exec=wrong']),
                             dict(exec='gomad run-guest'))


class InstallationMapping(unittest.TestCase):
    def fixture(self):
        bindings = [['go.temporal.io/server/tests/gomadfunctional', 'a' * 64]]
        origin = dict(invocation=dict(installation_sha256=check.sha(check.encode(bindings))))
        provenance = dict(inputs=dict(shared_driver_installations=bindings))
        return provenance, origin, copy.deepcopy(bindings)

    def test_correct_retained_mapping_binds_aggregate_and_actual_manifest(self):
        provenance, origin, expected = self.fixture()
        check.authenticate_installations(provenance, origin, expected)

    def test_self_consistent_false_mapping_cannot_borrow_supplied_manifest(self):
        provenance, origin, expected = self.fixture()
        provenance['inputs']['shared_driver_installations'][0][1] = 'b' * 64
        origin['invocation']['installation_sha256'] = check.sha(check.encode(provenance['inputs']['shared_driver_installations']))
        with self.assertRaisesRegex(ValueError, 'exact_installation_mapping'):
            check.authenticate_installations(provenance, origin, expected)

    def test_duplicate_unknown_and_commitment_changes_are_refused(self):
        for mode in ('duplicate','unknown','commitment'):
            provenance, origin, expected = self.fixture()
            if mode == 'duplicate':
                provenance['inputs']['shared_driver_installations'] *= 2
            elif mode == 'unknown':
                provenance['inputs']['shared_driver_installations'].append(['other/package','b' * 64])
            else:
                origin['invocation']['installation_sha256'] = 'f' * 64
            with self.subTest(mode=mode), self.assertRaises(ValueError):
                check.authenticate_installations(provenance, origin, expected)


if __name__ == '__main__':
    unittest.main()
