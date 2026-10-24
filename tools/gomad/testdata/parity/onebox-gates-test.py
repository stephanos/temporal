#!/usr/bin/env python3
"""Bounded orchestration tests; fake processes provide no qualification evidence."""
import importlib.util
from importlib.machinery import SourceFileLoader
from pathlib import Path
import tempfile
import argparse
import base64
import json
import hashlib
import subprocess
import sys
from unittest.mock import patch
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

    def test_specs_match_independent_original_owner_for_both_rows_and_modes(self):
        path = Path(__file__).resolve().parents[2] / 'rust/workload_tools/workloads.py'
        original_spec = importlib.util.spec_from_file_location('onebox_original_workloads', path)
        original = importlib.util.module_from_spec(original_spec)
        original_spec.loader.exec_module(original)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'source.sha256'
            source.write_bytes(b'actual retained source preimage\n')
            for cell in ['smoke', 'frontend']:
                for mode in ['success', 'canary']:
                    actual = gates.qualification_spec(cell, mode, root / mode, root, source)
                    expected = original.support_spec(root, root / mode, cell, mode == 'canary', source_manifest=source)
                    self.assertEqual(actual, expected)

    def test_command_receipt_keeps_real_exit_and_logs(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            runner = gates.Commands(root, 10)
            import sys
            row = runner.run('canary', [sys.executable, '-c', 'import sys; print("witness"); sys.exit(1)'])
            self.assertEqual(row['exit'], 1)
            self.assertGreater(row['operator_process_id'], 0)
            self.assertLessEqual(row['started_nanos'], row['ended_nanos'])
            self.assertIsNone(row['environment']['GOMAD_WASM_QUALIFY_WORKLOADS'])
            self.assertEqual(Path(row['stdout']['path']).read_bytes(), b'witness\n')

    def test_measured_spec_retains_exact_raw_bytes_and_full_original_envelope(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            spec = root / 'qualification.json'
            raw = b'{ "Campaign": {"WorldTransitionLimit": 33554432}, "Manifest": {} }\n'
            spec.write_bytes(raw)
            binary = root / 'operator'
            binary.write_text('#!' + sys.executable + '\nprint("actual child output")\n')
            binary.chmod(0o700)
            row = gates.Commands(root, 10).run('qualify', [binary, 'qualify', '--spec', spec])
            self.assertEqual(row['qualification_spec'], dict(path=str(spec), sha256=gates.digest(raw),
                bytes_base64=base64.b64encode(raw).decode('ascii')))
            self.assertEqual(json.loads(base64.b64decode(row['qualification_spec']['bytes_base64']))['Campaign']['WorldTransitionLimit'], 32 << 20)
            discovery = gates.Commands(root, 10).run('discovery', [binary, 'discover'])
            self.assertNotIn('qualification_spec', discovery)

    def test_mutated_world_bound_refuses_gate_and_retains_original_spec_and_real_exit(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            spec = root / 'qualification.json'
            raw = gates.encode(dict(Campaign=dict(WorldTransitionLimit=32 << 20)))
            spec.write_bytes(raw)
            binary = root / 'operator'
            binary.write_text('#!' + sys.executable + '\n' +
                'import json,pathlib,sys\np=pathlib.Path(sys.argv[3]); value=json.loads(p.read_bytes()); '
                'value["Campaign"]["WorldTransitionLimit"]=67108864; p.write_text(json.dumps(value)); sys.exit(1)\n')
            binary.chmod(0o700)
            runner = gates.Commands(root, 10)
            with self.assertRaisesRegex(ValueError, 'spec bytes changed'):
                runner.run('mutated', [binary, 'qualify', '--spec', spec])
            row = json.loads((root / 'mutated.command.json').read_bytes())
            self.assertEqual(row['exit'], 1)
            self.assertEqual(base64.b64decode(row['qualification_spec']['bytes_base64']), raw)
            self.assertNotEqual(gates.digest(spec.read_bytes()), row['qualification_spec']['sha256'])

    def test_invalid_spec_inputs_refuse_before_any_child_process(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            specification = root / 'input.json'
            specification.write_bytes(b'{}')
            alias = root / 'alias.json'
            alias.symlink_to(specification)
            for name, command in [('relative', [sys.executable, 'qualify', '--spec', 'relative.json']),
                    ('symlink', [sys.executable, 'qualify', '--spec', str(alias)]),
                    ('directory', [sys.executable, 'qualify', '--spec', str(root)]),
                    ('duplicate', [sys.executable, 'qualify', '--spec', str(specification), '--spec', str(specification)])]:
                with self.subTest(name=name), self.assertRaises((ValueError, OSError)):
                    gates.Commands(root, 10).run(name, command)
                self.assertFalse((root / (name + '.process.json')).exists())
            with specification.open('wb') as stream:
                stream.truncate((16 << 20) + 1)
            with self.assertRaisesRegex(ValueError, '16MiB'):
                gates.Commands(root, 10).run('oversized', [sys.executable, 'qualify', '--spec', specification])
            self.assertFalse((root / 'oversized.process.json').exists())

    def test_full_guest_profile_refused_before_any_guest_or_root_read(self):
        with patch.dict('os.environ', {'GOMAD_WASM_QUALIFICATION_GUEST_PROFILE': 'full'}):
            with self.assertRaisesRegex(ValueError, 'profile to slim'):
                gates.Onebox(argparse.Namespace())

    def test_closed_output_pipes_do_not_escape_outer_timeout(self):
        with tempfile.TemporaryDirectory() as directory:
            runner = gates.Commands(Path(directory), 0.3)
            with self.assertRaisesRegex(ValueError, 'timeout'):
                runner.run('closed', [sys.executable, '-c',
                    'import os,time; os.close(1); os.close(2); time.sleep(5)'])
            receipt = json.loads((Path(directory) / 'closed.command.json').read_bytes())
            self.assertEqual(receipt['exit'], -9)

    def test_timeout_kills_nested_compiler_process_group_in_same_session(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            child = root / 'child.pid'
            runner = gates.Commands(root, 0.5)
            script = ('import subprocess,sys,time,pathlib; '
                      'p=subprocess.Popen([sys.executable,"-c","import time; time.sleep(20)"],process_group=0); '
                      f'pathlib.Path({str(child)!r}).write_text(str(p.pid)); time.sleep(20)')
            with self.assertRaisesRegex(ValueError, 'timeout'):
                runner.run('nested', [sys.executable, '-c', script])
            import os
            pid = int(child.read_text())
            try:
                os.kill(pid, 0)
            except ProcessLookupError:
                return
            # A reparented zombie is dead and awaits the container init reaper.
            self.assertEqual(Path(f'/proc/{pid}/stat').read_text().split()[2], 'Z')

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
    assert spec['Manifest']['IntegratedSourceSHA256'].startswith('sha256:')
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


class SourceInventoryTest(unittest.TestCase):
    def fixture(self, root):
        repo = root / 'repo'
        repo.mkdir()
        (repo / 'source.go').write_bytes(b'package fixture\n')
        (repo / 'odd\tname\nλ.go').write_bytes(b'package odd\n')
        (repo / 'run.sh').write_bytes(b'#!/bin/sh\nexit 0\n')
        (repo / 'run.sh').chmod(0o755)
        (repo / 'source-link').symlink_to('source.go')
        subprocess.run(['git', 'init', '-q', str(repo)], check=True)
        subprocess.run(['git', '-C', str(repo), 'add', '.'], check=True)
        subprocess.run(['git', '-C', str(repo), '-c', 'user.name=fixture',
                        '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture'], check=True)
        control = root / 'control'
        control.write_bytes(b'#!/bin/sh\nexit 0\n')
        output = root / 'run'
        output.mkdir()
        return argparse.Namespace(repo=repo, root=output, binary=control, support=control,
            compiler=control, inspection=control, smoke_config=control, frontend_config=control,
            inspection_config=control)

    def test_gitlinks_skip_only_external_content_and_keep_tracked_bytes(self):
        for checked_out, supplied in [(True, False), (False, True)]:
            with self.subTest(checked_out=checked_out, supplied=supplied), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                args = self.fixture(root)
                identity = subprocess.check_output(['git', '-C', str(args.repo), 'rev-parse', 'HEAD'], text=True).strip()
                gitlink = args.repo / 'dash\tboards'
                subprocess.run(['git', '-C', str(args.repo), 'update-index', '--add', '--cacheinfo',
                                '160000,' + identity + ',' + gitlink.name], check=True)
                if checked_out:
                    gitlink.mkdir()
                    (gitlink / 'external.go').write_bytes(b'not part of the superproject\n')
                if supplied:
                    args.source_manifest = root / 'supplied.sha256'
                    args.source_manifest.write_bytes(b'unchanged supplied inventory\n')
                driver = gates.Onebox(args)
                frozen = json.loads((args.root / 'frozen-inputs.json').read_bytes())
                actual = {name: value for name, value in frozen.items() if Path(name).is_relative_to(args.repo)}
                expected = {'source.go': b'package fixture\n', 'source-link': b'package fixture\n',
                            'odd\tname\nλ.go': b'package odd\n', 'run.sh': b'#!/bin/sh\nexit 0\n'}
                self.assertEqual(actual, {str(args.repo / name): hashlib.sha256(raw).hexdigest()
                                          for name, raw in expected.items()})
                if supplied:
                    self.assertEqual(args.source_manifest.read_bytes(), b'unchanged supplied inventory\n')
                else:
                    self.assertNotIn(str(gitlink).encode(), driver.source.read_bytes())
                    for name, raw in expected.items():
                        self.assertIn((hashlib.sha256(raw).hexdigest() + '  ' + str(args.repo / name) + '\n').encode(),
                                      driver.source.read_bytes())
                if checked_out:
                    (gitlink / 'external.go').write_bytes(b'external content changed\n')
                driver.frozen()
                (args.repo / 'source.go').write_bytes(b'package changed\n')
                with self.assertRaisesRegex(ValueError, 'frozen source/candidate changed'):
                    driver.frozen()

    def test_regular_source_type_changes_refuse_at_capture_and_recheck(self):
        for phase in ['capture', 'frozen']:
            for replacement in ['directory', 'missing', 'symlink']:
                with self.subTest(phase=phase, replacement=replacement), tempfile.TemporaryDirectory() as directory:
                    args = self.fixture(Path(directory))
                    if phase == 'frozen':
                        driver = gates.Onebox(args)
                    source = args.repo / 'run.sh'
                    source.unlink()
                    if replacement == 'directory':
                        source.mkdir()
                    elif replacement == 'symlink':
                        source.symlink_to(args.binary)
                    with self.assertRaises((OSError, ValueError)):
                        gates.Onebox(args) if phase == 'capture' else driver.frozen()

    def test_unmerged_index_refuses_before_source_capture(self):
        with tempfile.TemporaryDirectory() as directory:
            args = self.fixture(Path(directory))
            identity = subprocess.check_output(['git', '-C', str(args.repo), 'rev-parse', 'HEAD'], text=True).strip()
            subprocess.run(['git', '-C', str(args.repo), 'update-index', '--index-info'], check=True,
                           input=('160000 ' + identity + ' 1\tconflicted-gitlink\n').encode())
            (args.repo / 'conflicted-gitlink').write_bytes(b'not a resolved index entry\n')
            with self.assertRaisesRegex(ValueError, 'unmerged'):
                gates.Onebox(args)


if __name__ == '__main__':
    unittest.main()
