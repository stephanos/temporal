#!/usr/bin/env python3
"""Bounded orchestration tests; fake processes provide no qualification evidence."""
import importlib.util
from importlib.machinery import SourceFileLoader
from pathlib import Path
import tempfile
import argparse
import json
import subprocess
import sys
import unittest

loader = SourceFileLoader('onebox_gates', str(Path(__file__).with_name('onebox-gates.py')))
spec = importlib.util.spec_from_loader(loader.name, loader)
gates = importlib.util.module_from_spec(spec)
loader.exec_module(gates)


class OrderedGatesTest(unittest.TestCase):
    def test_preparation_failure_stops_before_measured_work(self):
        attempted = []
        def perform(name):
            attempted.append(name)
            if name == 'smoke-preparation':
                raise ValueError('compiler failed')
        with self.assertRaisesRegex(ValueError, 'compiler failed'):
            gates.ordered(perform)
        self.assertEqual(attempted, ['smoke-preparation'])

    def test_full_order_and_inspection_failure_stops_frontend(self):
        attempted = []
        gates.ordered(attempted.append)
        self.assertEqual(attempted, ['smoke-preparation', 'smoke-success', 'smoke-canary',
                                    'inspection', 'frontend-preparation', 'frontend-success',
                                    'frontend-canary', 'publication'])
        attempted = []
        def perform(name):
            attempted.append(name)
            if name == 'inspection':
                raise ValueError('missing snapshot')
        with self.assertRaises(ValueError):
            gates.ordered(perform)
        self.assertEqual(attempted[-1], 'inspection')

    def test_canary_preserves_cli_exit_and_requires_independent_admission(self):
        gates.measured_status('canary', 1, 0, 0)
        gates.measured_status('success', 0, 0, 0)
        for values in [('canary', 0, 0, 0), ('canary', 3, 0, 0),
                       ('canary', 1, 3, 0), ('canary', 1, 0, 3), ('success', 1, 0, 0)]:
            with self.subTest(values=values), self.assertRaises(ValueError):
                gates.measured_status(*values)

    def test_missing_gate_refuses_publication(self):
        rows = [dict(cell=gates.CELLS[cell][0], name=name, exit=0) for cell in gates.CELLS for name in gates.GATE_NAMES]
        gates.complete(rows)
        for changed in [rows[:-1], rows + rows[:1], [dict(row, exit=1) if index == 2 else row
                                                   for index, row in enumerate(rows)]]:
            with self.assertRaises(ValueError):
                gates.complete(changed)

    def test_exact_argv_bounds_and_fresh_roots(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            smoke = gates.qualification_spec('smoke', 'success', root / 'success', root, root / 'source')
            canary = gates.qualification_spec('smoke', 'canary', root / 'canary', root, root / 'source')
            expected = ['-test.v', '-test.count=1', '-test.timeout=72h',
                        '-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$',
                        '-persistenceType=sql', '-persistenceDriver=sqlite']
            self.assertEqual(smoke['Campaign']['Target']['Args'], expected)
            expected[3] = '-test.run=['
            self.assertEqual(canary['Campaign']['Target']['Args'], expected)
            self.assertEqual(smoke['Manifest']['Repeat'], 2)
            self.assertEqual(smoke['Campaign']['ExecutionTimeout'], 900_000_000_000)
            self.assertEqual(smoke['Campaign']['SuccessBytesLimit'], 2 << 30)
            self.assertNotEqual(smoke['ArtifactRoot'], canary['ArtifactRoot'])

    def test_command_receipt_keeps_real_exit_and_logs(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            runner = gates.Commands(root, 10)
            import sys
            row = runner.run('canary', [sys.executable, '-c', 'import sys; print("witness"); sys.exit(1)'])
            self.assertEqual(row['exit'], 1)
            self.assertGreater(row['operator_process_id'], 0)
            self.assertEqual(Path(row['stdout']['path']).read_bytes(), b'witness\n')

    def test_closed_output_pipes_do_not_escape_outer_timeout(self):
        with tempfile.TemporaryDirectory() as directory:
            runner = gates.Commands(Path(directory), 0.3)
            with self.assertRaisesRegex(ValueError, 'timeout'):
                runner.run('closed', [sys.executable, '-c',
                    'import os,time; os.close(1); os.close(2); time.sleep(5)'])
            receipt = json.loads((Path(directory) / 'closed.command.json').read_bytes())
            self.assertEqual(receipt['exit'], -9)

    def test_fresh_process_driver_retains_raw_canary_and_stops_on_bad_report(self):
        with tempfile.TemporaryDirectory() as temporary:
            temporary = Path(temporary)
            repo = temporary / 'repo'
            repo.mkdir()
            (repo / 'source.go').write_text('package fixture\n')
            subprocess.run(['git', 'init', '-q', str(repo)], check=True)
            subprocess.run(['git', '-C', str(repo), 'add', 'source.go'], check=True)
            binary = temporary / 'gomad'
            binary.write_text('#!' + sys.executable + "\n" + r'''
import json, os, pathlib, sys
args = sys.argv[1:]
def value(flag): return args[args.index(flag) + 1]
if 'install' in args:
    pathlib.Path(value('--output')).write_text('{}')
elif 'plan' in args:
    path = pathlib.Path(value('--output'))
    path.write_text('{}')
    print(json.dumps(dict(schema='gomad.plan-result/v1',guest_executions=0,module_sha256='a'*64,
        preparation_receipt=str(path),preparation_receipt_sha256='b'*64)))
elif 'qualify' in args:
    spec = json.loads(pathlib.Path(value('--spec')).read_bytes())
    root = pathlib.Path(spec['ArtifactRoot'])
    root.mkdir(parents=True)
    path = root / 'result.json'
    path.write_text(json.dumps(spec))
    print(json.dumps(dict(result_path=str(path))))
    sys.exit(1 if spec['Manifest']['Cell'] == 'canary' else 0)
elif 'report' in args:
    if pathlib.Path(__file__).with_name('fail-report').exists(): sys.exit(3)
    print('{}')
''')
            binary.chmod(0o700)
            support = temporary / 'fn222-support'
            support.write_text('#!' + sys.executable + "\n" + r'''
import json, pathlib, sys
if sys.argv[1] == 'check-result': print(json.dumps(dict(module_sha256='a'*64)))
elif sys.argv[1] == 'publish':
    request = json.loads(pathlib.Path(sys.argv[2]).read_bytes())
    assert len(request['results']) == 4
    assert len(request['gates']) == 8
    pathlib.Path(sys.argv[3] + '.json').write_text('{}')
''')
            support.chmod(0o700)
            inspection = temporary / 'inspection.py'
            inspection.write_text("import json; print(json.dumps(dict(receipt=__file__)))\n")
            config = temporary / 'config.json'
            config.write_text('{}')
            args = argparse.Namespace(repo=repo, binary=binary, support=support, inspection=inspection,
                smoke_config=config, frontend_config=config, inspection_config=config,
                compiler=binary, runtime_root=repo, root=temporary / 'run')
            args.root.mkdir()
            driver = gates.Onebox(args)
            gates.ordered(driver.perform)
            canary = json.loads((args.root / 'smoke-canary.command.json').read_bytes())
            stage = json.loads((args.root / 'smoke-measured-pass-replay-canary.gate.json').read_bytes())
            self.assertEqual((canary['exit'], stage['exit']), (1, 0))
            self.assertTrue(canary['mandatory_workloads'])
            frontend = json.loads((args.root / 'frontend-success.command.json').read_bytes())
            self.assertFalse(frontend['mandatory_workloads'])
            self.assertFalse(json.loads((args.root / 'smoke-plan.command.json').read_bytes())['mandatory_workloads'])
            (temporary / 'fail-report').touch()
            args.root = temporary / 'bad-report'
            args.root.mkdir()
            driver = gates.Onebox(args)
            with self.assertRaisesRegex(ValueError, 'report/admission exits'):
                gates.ordered(driver.perform)
            self.assertFalse((args.root / 'smoke-canary.command.json').exists())
            self.assertFalse((args.root / 'support-request.json').exists())
            (repo / 'source.go').write_text('package changed\n')
            with self.assertRaisesRegex(ValueError, 'frozen source/candidate changed'):
                driver.frozen()


if __name__ == '__main__':
    unittest.main()
