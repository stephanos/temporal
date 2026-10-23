from pathlib import Path
import sys
import unittest

HERE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(HERE))


class ConfigurationContract(unittest.TestCase):
    def test_fixture_and_original_workflow_specs_use_unchanged_bounds(self):
        import configure
        from adapters import CELLS, validate_cell_spec
        for label, wanted in CELLS.items():
            if wanted['workload'] not in ('fixture', 'onebox'):
                continue
            with self.subTest(label=label):
                spec = configure.cell_spec(label, '/evidence', '/source.sha256', 'sha256:'+'1'*64)
                validate_cell_spec(spec, wanted)
                self.assertFalse(spec['Manifest']['Provisional'])
                self.assertEqual((spec['Manifest']['Seed'], spec['Manifest']['Repeat']), (7, 2))
                self.assertEqual(spec['Campaign']['ExecutionTimeout'],
                                 900_000_000_000 if wanted['workload'] == 'onebox' else 60_000_000_000)
                if wanted['workload'] == 'onebox':
                    self.assertEqual(spec['Campaign']['Target']['Source'], './tests')
                    self.assertEqual(len(spec['Campaign']['Target']['Args']), 6)

    def test_specialized_durable_specs_cannot_be_filled_with_fixtures(self):
        import configure
        import driver
        for label in ('workload-sqlite-cooperative-success', 'workload-multi-instance-cooperative-success'):
            with self.assertRaises(driver.Unexecuted):
                configure.cell_spec(label, '/evidence', '/manifest', 'sha256:'+'1'*64)

    def test_typed_durable_requests_use_owned_scenarios_and_future_paths_only(self):
        import base64
        import hashlib
        import json
        import tempfile
        import configure
        from adapters import validate_cell_spec, CELLS
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            manifest = root / 'source.sha256'
            manifest.write_text('sealed source fixture\n')
            digest = hashlib.sha256(manifest.read_bytes()).hexdigest()
            attempt = root / 'future-attempt'
            for workload in ('sqlite', 'multi-instance'):
                for mode in ('success', 'canary'):
                    label = 'workload-' + workload + '-cooperative-' + mode
                    spec = configure.cell_spec(label, str(attempt / 'cells' / label), str(manifest), digest,
                                               bindings=dict(root=str(attempt)))
                    validate_cell_spec(spec, CELLS[label])
                    scenario = json.loads(base64.b64decode(spec['Campaign']['Target']['Scenario']))
                    self.assertEqual(scenario['workload']['kind'], 'sqlite-durable-recovery' if workload == 'sqlite'
                                     else 'independent-temporal-replication-durable-restart')
                    if workload == 'multi-instance':
                        self.assertEqual(spec['Campaign']['Target']['WorkingDir'], str(attempt / 'stages' / label / 'source-module'))
                    self.assertEqual(spec['Manifest']['IntegratedSourceSHA256'], 'sha256:' + digest)
            self.assertFalse(attempt.exists())

    def test_stress_freeze_pins_static_inputs_before_plan_and_allverb_binding(self):
        import configure
        import json
        import tempfile
        from driver import read, sha, Refusal
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            static = root / 'input'
            static.write_text('actual static bytes')
            template = root / 'template.json'
            template.write_text(json.dumps(dict(schema='gomad.all-verb-bindings/v1',
                files=dict(release=dict(path='input', sha256=sha(static))), trees={})))
            config = dict(repo='/repo', root='/attempt', source_manifest=str(static), source_sha256=sha(static),
                sent_epoch=1, deadline_s=3600, identity=dict(host='1'*64, helper='2'*64), binary_sha256='3'*64,
                stress=dict(spec=str(static), binary=str(static), build_receipt=str(static), producer_root='/attempt/stress', producer_name='stress'),
                files={str(static): sha(static), str(template): sha(template)}, public_verbs=dict(bindings=str(template)))
            configure.freeze_final_cycle(config, root / 'config.json')
            plan = read(config['stress_cycle'])
            self.assertNotIn(config['stress_cycle'], plan['files'])
            self.assertNotIn(config['public_verbs']['bindings'], plan['files'])
            binding = read(config['public_verbs']['bindings'])
            self.assertEqual(binding['files']['stress-cycle'], dict(path=config['stress_cycle'], sha256=sha(config['stress_cycle'])))
            self.assertEqual(binding['files']['release']['path'], str(static))
            self.assertEqual(config['files'][config['stress_cycle']], sha(config['stress_cycle']))
            with self.assertRaisesRegex(Refusal, 'generated for this freeze'):
                configure.freeze_final_cycle(config, root / 'again.json')

    def test_all_native_profile_roles_expand_their_actual_admission_closure(self):
        import configure
        import json
        import tempfile
        from driver import Refusal
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, artifact, copied = (root/name for name in ('source','artifact','published'))
            for path in (source,artifact,copied): path.write_text(path.name)
            receipt = root/'helper-build.json'
            receipt.write_text(json.dumps(dict(schema='gomad.native-measurement-build/v1',executable=str(artifact),inputs={str(source):'hash'})))
            admission = root/'helper-admission.json'
            admission.write_text(json.dumps(dict(schema='gomad.native-measurement-admission/v1',build_receipt=str(receipt),executable=str(copied))))
            logs = {str(root/(name+'.'+suffix)) for name in ('artifact-identity','staged-identity') for suffix in ('stdout','stderr','json')}
            for path in logs: Path(path).write_text('raw identity command')
            entry = dict(binary=str(copied),build_receipt=str(receipt),admission=str(admission))
            expected = {str(source),str(artifact),str(copied),str(receipt),str(admission)} | logs
            self.assertEqual(configure.native_role_inputs(entry), expected)
            profiles = {name:dict(entry) for name in ('measure','ordinary','diagnostic')}
            self.assertEqual(configure.native_profile_inputs(dict(final_profiles=profiles)),expected)
            without = dict(entry)
            without.pop('admission')
            with self.assertRaisesRegex(Refusal,'actual admission'):
                configure.native_role_inputs(without)
            Path(next(iter(logs))).unlink()
            with self.assertRaisesRegex(Refusal,'missing/nonregular'):
                configure.native_role_inputs(entry)

    def test_generated_manifest_uses_qualification_digest_grammar(self):
        import configure
        for digest in ('1'*64, 'sha256:'+'1'*64):
            with self.subTest(digest=digest):
                spec = configure.cell_spec('workload-fixture-cooperative-success', '/evidence', '/manifest', digest)
                self.assertEqual(spec['Manifest']['IntegratedSourceSHA256'], 'sha256:'+'1'*64)


if __name__ == '__main__':
    unittest.main()
