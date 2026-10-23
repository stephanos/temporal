"""Receiving counterexamples; synthetic handshakes earn no compiled-Go credit."""
import copy
import importlib.util
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from driver import Refusal


class StreamingProtocol(unittest.TestCase):
    def test_actual_driver_process_has_independently_sampled_end_clocks(self):
        import os
        import tempfile
        import time
        import driver
        import interception_streaming as gate
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            config=dict(repo=str(root))
            argv=[sys.executable,'-c','pass']
            env=dict(PATH='/usr/bin:/bin')
            started=time.monotonic_ns()
            recipe=dict(pid=os.getpid(),started_ns=started,deadline_ns=started+600_000_000_000)
            driver.command(config,root,'actual-python',argv,timeout=2,exact_environment=env)
            row=gate.authenticate_command(config,root,'actual-python',argv,env,recipe)
            self.assertGreater(row['pid'],0)
            process=driver.read(root/'actual-python.process.json')
            self.assertLessEqual(process['end_ns'],row['end_ns'])
            original=copy.deepcopy(process)
            (root/'actual-python.command.json').write_text(json.dumps(dict(row,clock='CLOCK_REALTIME')))
            (root/'actual-python.process.json').write_text(json.dumps(dict(process,clock='CLOCK_REALTIME')))
            with self.assertRaises(Refusal):gate.authenticate_command(config,root,'actual-python',argv,env,recipe)
            (root/'actual-python.command.json').write_text(json.dumps(row))
            for field,value in dict(pid=row['pid']+1,stage_pid=recipe['pid']+1,exit=False,end_ns=row['end_ns']+1,
                    started_monotonic_nanos=recipe['deadline_ns']+1).items():
                (root/'actual-python.process.json').write_text(json.dumps(dict(original,**{field:value})))
                with self.subTest(field=field),self.assertRaises(Refusal):
                    gate.authenticate_command(config,root,'actual-python',argv,env,recipe)

    def test_marker_release_and_exact_stream_bindings(self):
        module=Path(__file__).parents[1]/'interception_streaming.py'
        self.assertTrue(module.is_file(),'controlled compiled-Go receiving endpoint is missing')
        spec=importlib.util.spec_from_file_location('interception_streaming',module)
        gate=importlib.util.module_from_spec(spec);spec.loader.exec_module(gate)
        stdout=gate.MARKER+b'bounded callback output\n'
        streams=dict(stdout=stdout,stderr=b'',transcript=b'wasi',choice=b'choices')
        record=dict(identity=dict(target='m'*64),outcome=dict(termination='exit',exit_code=0),streams={
            ('output/' if name in ('stdout','stderr') else 'evidence/')+name:dict(bytes=len(raw),sha256=gate.digest(raw)) for name,raw in streams.items()})
        identity=dict(host_sha256='h'*64,helper_sha256='j'*64,runtime_implementation_sha256='r'*64,
            engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',release_role=True)
        release=dict(schema='gomad.interception-streaming-release/v1',pid=123,parent_record_sha256='p'*64,
            marker_sha256=gate.digest(gate.MARKER),observed_ns=30,action_started_ns=40)
        report=dict(schema='gomad.interception-streaming-proof/v1',pid=123,parent_record_sha256='p'*64,
            module_sha256='m'*64,identity=identity,not_complete_before_release=True,
            marker_callback_ns=20,marker_observed_ns=30,release_written_ns=50,release_received_ns=60,terminal_ns=70,
            exact=True,outcome=dict(termination='exit',exit_code=0),
            streams={name:gate.digest(raw) for name,raw in streams.items()})
        gate.validate_protocol(report,release,streams,record,'p'*64,identity,123,10,80)
        for field,value in dict(pid=124,parent_record_sha256='q'*64,module_sha256='x'*64,identity={},
                not_complete_before_release=False,marker_callback_ns=35,release_written_ns=65,
                release_received_ns=75,terminal_ns=90,exact=False,outcome=dict(termination='trap',exit_code=None),streams={}).items():
            changed=dict(report,**{field:value})
            with self.subTest(report=field),self.assertRaises(Refusal):
                gate.validate_protocol(changed,release,streams,record,'p'*64,identity,123,10,80)
        for field,value in dict(pid=1,parent_record_sha256='q'*64,marker_sha256='0'*64,
                observed_ns=45,action_started_ns=60).items():
            with self.subTest(release=field),self.assertRaises(Refusal):
                gate.validate_protocol(report,dict(release,**{field:value}),streams,record,'p'*64,identity,123,10,80)
        for changed in (dict(streams,stdout=b'not the marker'),dict(streams,stderr=b'foreign')):
            with self.assertRaises(Refusal):
                gate.validate_protocol(report,release,changed,record,'p'*64,identity,123,10,80)


if __name__=='__main__':unittest.main()
