"""Verify selected recovery producer capture with synthetic local files only."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

TOOLS = Path(__file__).resolve().parents[2] / 'workload_tools'
sys.path.insert(0, str(TOOLS))
spec = importlib.util.spec_from_file_location('recovery_capture', TOOLS / 'temporal-recovery.py')
recovery = importlib.util.module_from_spec(spec)
spec.loader.exec_module(recovery)


class RecoveryCapture(unittest.TestCase):
    def test_each_selected_qualification_captures_private_preparation(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            primary, artifact, selected = base / 'primary', base / 'artifact', base / 'selected-result.json'
            primary.mkdir()
            artifact.mkdir()
            streams = artifact / 'evidence/cluster/temporal/1'
            streams.mkdir(parents=True)
            (streams / 'storage_state').write_bytes(b'state')
            (streams / 'choice').write_bytes(b'choice')
            (artifact / 'manifest.json').write_text('{"record":{}}')
            result = base / 'primary-result.json'
            result.write_text(json.dumps(dict(report=dict(executions=[dict(artifact_path=str(artifact))]),
                native_crash_catalogues=[dict(volume='temporal', storage_state_sha256=hashlib.sha256(b'state').hexdigest(),
                    enumeration=dict(States=[dict(Identity='one'), dict(Identity='two')]), context={})])))
            (primary / 'producer.json').write_text(json.dumps(dict(workload='recovery', mode='success',
                result_path=str(result), result_sha256=hashlib.sha256(result.read_bytes()).hexdigest())))
            (primary / 'qualification-spec.json').write_text('{"Campaign":{}}')
            selected.write_text(json.dumps(dict(report=dict(qualified=True,
                executions=[dict(replay=dict(matched=True)), dict(replay=dict(matched=True))]))))
            private, captures = object(), []
            def run(root, name, argv, expected, **kwargs):
                if name == 'qualify':
                    captures.append(kwargs.get('private_observations'))
                    return dict(result_path=str(selected))
                return {}
            argv = ['temporal-recovery.py', '--binary', '/sealed/gomad', '--repo', '/sealed/repo',
                    '--private-observation-config', '/sealed/config', '--primary', str(primary), '--root', str(base / 'fresh')]
            with patch.object(sys, 'argv', argv), patch.object(recovery.producer, 'run', side_effect=run), \
                 patch.object(recovery.producer, 'private_configuration', return_value=private), \
                 patch.object(recovery.producer, 'private_admission'), patch.object(recovery.producer, 'owned_root'):
                recovery.main()
            self.assertEqual(captures, [private, private])


if __name__ == '__main__':
    unittest.main()
