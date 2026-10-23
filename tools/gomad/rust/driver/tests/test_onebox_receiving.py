from types import SimpleNamespace
import copy
import base64
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import receiving
import onebox_receiving as onebox


class OneboxReceiving(unittest.TestCase):
    def test_support_inputs_and_frozen_reader_require_actual_typed_example(self):
        import main_cli_build
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);binary=root/'fn222-support';binary.write_bytes(b'synthetic')
            path=root/'main-build.json';path.write_text('{}')
            raw=root/'cargo.stdout';raw.write_text('raw')
            plan=root/'plan.json';plan.write_text(json.dumps(dict(repo=str(root))))
            receipt=dict(binary=str(binary),source_sha256='s',head='h',symlinks={},tools_plan=str(plan),inputs={str(raw):driver.sha(raw)})
            config=dict(repo=str(root),source_sha256='s',head='h',symlinks={},fn222=dict(
                argv=['python','producer','--support',str(binary)],support_build_receipt=str(path)))
            with patch.object(main_cli_build,'read_build',return_value=receipt) as reader:
                self.assertEqual(onebox.support_inputs(config),{str(path),str(raw)})
                reader.assert_called_with(str(path),None,expected_role='fn222-support')
                self.assertEqual(onebox.support_binding(config),receipt)
                reader.assert_called_with(str(path),config,expected_role='fn222-support')
                changed=copy.deepcopy(config);changed['fn222']['argv'][-1]='/foreign/support'
                with self.assertRaises(driver.Refusal):onebox.support_inputs(changed)
                changed=copy.deepcopy(config);changed['head']='foreign'
                with self.assertRaises(driver.Refusal):onebox.support_inputs(changed)
                for arguments in [config['fn222']['argv']+['--support=/foreign/support'],
                                  config['fn222']['argv']+['--supp','/foreign/support'],
                                  config['fn222']['argv']+['--support','/foreign/support'],
                                  config['fn222']['argv']+['--cache'],
                                  config['fn222']['argv']+['--cache','--support'], None]:
                    changed=copy.deepcopy(config);changed['fn222']['argv']=arguments
                    with self.subTest(arguments=arguments),self.assertRaises(driver.Refusal):onebox.support_inputs(changed)
                changed=copy.deepcopy(config);del changed['fn222']['support_build_receipt']
                with self.assertRaises(driver.Refusal):onebox.support_inputs(changed)

    def test_gate_matrix_refuses_missing_duplicate_foreign_and_failed(self):
        rows = [dict(cell=cell, name=gate, exit=0) for _, cell in onebox.CELLS for gate in onebox.GATES]
        onebox.gates(dict(gates=rows))
        for changed in [rows[:-1], rows + rows[:1], [dict(row, cell='foreign') for row in rows],
                        [dict(row, exit=1) for row in rows], list(reversed(rows))]:
            with self.subTest(changed=changed), self.assertRaises(driver.Refusal):
                onebox.gates(dict(gates=changed))

    def test_actual_command_schema_pid_binary_output_and_private_seal(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            executable = str(Path(sys.executable).resolve())
            config = dict(root=str(root), repo=str(root), binary=executable, sent_epoch=1,
                          files={executable: driver.sha(executable)})
            reader = receiving.Retained(config)
            stdout, stderr = root / 'cli.stdout', root / 'cli.stderr'
            stdout.write_bytes(b'raw evidence\n')
            stderr.write_bytes(b'')
            row = dict(schema='gomad.rust-onebox-command/v1', command=[executable, 'qualify', '--spec', '/spec'],
                executable=dict(path=executable, sha256=driver.sha(executable)), cwd=str(root),
                operator_process_id=123, exit=1, started_nanos=2_000_000_000, ended_nanos=3_000_000_000,
                elapsed_nanos=1_000_000_000, mandatory_workloads=True,
                environment={'GOMAD_WASM_QUALIFY_WORKLOADS': '1'}, environment_sha256='f' * 64,
                stdout=dict(path=str(stdout), sha256=driver.sha(stdout), bytes=stdout.stat().st_size),
                stderr=dict(path=str(stderr), sha256=driver.sha(stderr), bytes=0))
            path = root / 'cli.command.json'
            def put(value):
                registration = dict(active=False, observation='completed', pid=value['operator_process_id'],
                    stage_pid=124, argv=value['command'], exit=value['exit'], start_ns=value['started_nanos'],
                    end_ns=value['ended_nanos'])
                (root / 'cli.process.json').write_bytes(receiving.canonical(registration) + b'\n')
                path.write_bytes(receiving.canonical(value) + b'\n')
            put(row)
            self.assertEqual(onebox.command(reader, path, accepted=(1,))[1], stdout.read_bytes())
            for changes in [dict(schema='foreign'), dict(operator_process_id=True), dict(exit=0),
                            dict(cwd='/foreign'), dict(started_nanos=0),
                            dict(executable=dict(path=executable, sha256='0' * 64)),
                            dict(environment={'GOMAD_WASM_QUALIFY_WORKLOADS': None})]:
                put(dict(row, **changes))
                with self.subTest(changes=changes), self.assertRaises(driver.Refusal):
                    onebox.command(reader, path, accepted=(1,))
            put(row)
            with self.assertRaisesRegex(driver.Refusal, 'private observation seal'):
                onebox.command(reader, path, accepted=(1,), native=True)
            row['private_observations'] = {'retained': 'original seal'}
            raw_spec = b'{"Campaign":{"WorldTransitionLimit":33554432}}'
            row['qualification_spec'] = dict(path='/spec', sha256=receiving.digest(raw_spec),
                bytes_base64=base64.b64encode(raw_spec).decode('ascii'))
            put(row)
            class Observations:
                def verify_snapshot(self, actual, pid, seal):
                    self.assertions = (actual, pid, seal)
            observations = Observations()
            with patch.object(onebox, 'load_module', return_value=observations):
                onebox.command(reader, path, accepted=(1,), native=True)
            self.assertEqual(observations.assertions, (config, 123, row['private_observations']))
            for witness in [None, dict(row['qualification_spec'], path='/foreign'),
                            dict(row['qualification_spec'], sha256='0' * 64)]:
                put(dict(row, qualification_spec=witness))
                with self.assertRaises(driver.Refusal):
                    onebox.command(reader, path, accepted=(1,), native=True)
            put(row)
            stdout.write_bytes(b'forged evidence\n')
            with self.assertRaises(driver.Refusal):
                onebox.command(reader, path, accepted=(1,))

    def test_missing_support_binding_remains_unexecuted(self):
        with self.assertRaises(driver.Unexecuted):
            onebox.binding({})

    def test_primary_inspection_uses_driver_candidate_before_mandatory_result(self):
        class NativeReached(Exception):
            pass
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            receipt = root / 'inspection-receipt.json'
            receipt.write_text('{}')
            reader = SimpleNamespace(owner=lambda _: None, root=root,
                config=dict(_config_path='/sealed/config.json', _config_sha256='a' * 64,
                            binary='/sealed/gomad', binary_sha256='b' * 64))
            with patch.object(onebox, 'validate', side_effect=NativeReached) as validate:
                with self.assertRaises(NativeReached):
                    onebox.inspection(reader, root, standalone=True)
                self.assertEqual(validate.call_args.args[1], ['validate-inspection-driver', receipt,
                    onebox.sha(receipt), '/sealed/config.json', 'a' * 64, '/sealed/gomad', 'b' * 64])
            with self.assertRaises(FileNotFoundError):
                onebox.inspection(reader, root)

    def test_onebox_inspection_uses_new_reader_and_other_owners_remain_strict(self):
        reader = receiving.Retained(dict(root='/attempt'))
        with patch.object(onebox, 'inspection', return_value='authenticated') as inspect:
            self.assertEqual(reader.inspection(Path('/attempt/onebox'), 'onebox'), 'authenticated')
            inspect.assert_called_once_with(reader, Path('/attempt/onebox'))
        with self.assertRaises((driver.Refusal, FileNotFoundError)):
            reader.inspection(Path('/attempt/shard'), 'shard')


if __name__ == '__main__':
    unittest.main()
