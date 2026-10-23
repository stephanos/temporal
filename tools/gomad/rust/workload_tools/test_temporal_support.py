"""Bounded runner-control fixtures; these supply no qualification evidence."""
import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))
import producer

spec = importlib.util.spec_from_file_location('temporal_support', Path(__file__).with_name('temporal-support.py'))
support = importlib.util.module_from_spec(spec)
spec.loader.exec_module(support)


class OneboxRunner(unittest.TestCase):
    def exercise(self, prepare_only=False, fail_prepare=False, private=False):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        parent = Path(temporary.name)
        root = parent / 'attempt'
        (parent/'source-manifest').write_text('bounded source-manifest fixture, not a qualification input')
        argv = ['temporal-support.py']
        for name in ('binary','repo','compiler','runtime-root','cache','captured-config','source-manifest'):
            argv += ['--'+name, str(parent/name)]
        argv += ['--root', str(root)]
        if prepare_only:
            argv.append('--prepare-only')
        if private:
            argv += ['--private-observation-config', str(parent/'frozen-driver-config')]
        calls = []
        self.captures = []
        context = {'explicit': 'bounded mocked context, never candidate evidence'} if private else None
        def run(cell, name, command, expected, json_output=True, environment=None, **kwargs):
            self.captures.append((name, kwargs))
            row = cell.name
            calls.append((row,name,(environment or os.environ).get('GOMAD_WASM_QUALIFY_WORKLOADS'),list(command)))
            if name == 'install':
                return None
            if name == 'prepare':
                if fail_prepare:
                    raise ValueError('actual fixture preparation refusal')
                receipt = cell/'plan.json.preparation.json'
                receipt.write_bytes(b'bounded unit fixture, not native evidence')
                return dict(guest_executions=0, module_sha256='1'*64,
                            preparation_receipt=str(receipt), preparation_receipt_sha256=hashlib.sha256(receipt.read_bytes()).hexdigest())
            if name in ('success','canary'):
                directory = cell/name
                directory.mkdir()
                executions = []
                for index in range(2):
                    artifact = directory/str(index)
                    (artifact/'output').mkdir(parents=True)
                    selected = 'TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds' if row == 'smoke' else 'TestFrontendSystemInfo'
                    raw = ('    --- PASS: '+selected+' (0.01s)\n').encode()
                    (artifact/'output/stdout').write_bytes(raw)
                    executions.append(dict(artifact_path=str(artifact),record=dict(streams={'output/stdout':dict(bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest())})))
                result = directory/'result.json'
                result.write_text(json.dumps(dict(manifest=dict(module_sha256='1'*64), report=dict(executions=executions))))
                return dict(result_path=str(result))
            if name == 'inspection':
                return dict(receipt='explicit unit fixture, not actual inspection evidence')
            raise AssertionError(name)
        output = io.StringIO()
        with patch.object(sys,'argv',argv), patch.object(support.producer,'captured',return_value={'fixture':'not qualification input'}), \
             patch.object(support.producer,'run',side_effect=run), \
             patch.object(support.producer,'private_configuration',return_value=context), \
             patch.object(support.producer,'private_admission'), patch.dict(os.environ,{'GOMAD_WASM_QUALIFY_WORKLOADS':'inherited'}), \
             contextlib.redirect_stdout(output):
            if fail_prepare:
                with self.assertRaisesRegex(ValueError,'preparation refusal'):
                    support.main()
            else:
                support.main()
        return root,calls,output.getvalue()

    def test_prepare_only_stops_before_all_measured_and_other_row_work(self):
        root,calls,output = self.exercise(prepare_only=True)
        self.assertEqual([(row,name) for row,name,_,_ in calls],[('smoke','install'),('smoke','prepare')])
        self.assertTrue(all(intent is None for _,_,intent,_ in calls))
        self.assertFalse((root/'frontend').exists())
        self.assertFalse((root/'inspection').exists())
        self.assertFalse((root/'support-receipt.json').exists())
        response = json.loads(output)
        self.assertEqual(response['row'],'smoke')
        self.assertNotIn('subgates',response)
        self.assertEqual(response['preparation']['guest_executions'],0)

    def test_preparation_failure_stops_before_measured_work(self):
        _,calls,output = self.exercise(fail_prepare=True)
        self.assertEqual([name for _,name,_,_ in calls],['install','prepare'])
        self.assertEqual(output,'')

    def test_intent_is_enabled_only_for_smoke_measured_modes(self):
        _,calls,_ = self.exercise()
        for row,name,intent,_ in calls:
            with self.subTest(row=row,name=name):
                self.assertEqual(intent,'1' if row=='smoke' and name in ('success','canary') else None)
        self.assertEqual(sum(name=='prepare' for _,name,_,_ in calls),2)
        self.assertEqual(sum(name=='inspection' for _,name,_,_ in calls),1)

    def test_private_context_reaches_only_native_preparations_and_nested_producers(self):
        _, calls, _ = self.exercise(private=True)
        for name, kwargs in self.captures:
            self.assertEqual(bool(kwargs), name == 'prepare')
            if kwargs:
                self.assertEqual(set(kwargs), {'private_observations'})
        for _, name, _, command in calls:
            self.assertEqual('--private-observation-config' in command, name in ('success', 'canary', 'inspection'))

    def test_named_pass_requires_one_exact_verbose_leaf(self):
        raw=b'    --- PASS: TestFrontendSystemInfo (0.01s)\n'
        support.selected_pass(raw,'frontend')
        for invalid in (raw+raw, raw.replace(b' (0.01s)',b''), raw.replace(b'Info',b'InfoOther'),
                        raw+b'--- SKIP: another\n',raw+b'testing: warning: no tests to run\n'):
            with self.subTest(raw=invalid),self.assertRaises(ValueError):
                support.selected_pass(invalid,'frontend')

    def test_actual_child_intent_observation_matches_gate_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            for row,measured in (('smoke',False),('smoke',True),('frontend',True)):
                name=row+str(measured)
                response=producer.run(root,name,[sys.executable,'-c',
                    'import json,os; print(json.dumps(dict(intent=os.environ.get("GOMAD_WASM_QUALIFY_WORKLOADS"))))'],0,
                    environment=support.gate_environment(row,measured))
                expected='1' if row=='smoke' and measured else None
                self.assertEqual(response['intent'],expected)
                self.assertEqual(json.loads((root/(name+'.command.json')).read_bytes())['environment']['GOMAD_WASM_QUALIFY_WORKLOADS'],expected)


if __name__=='__main__':
    unittest.main()
