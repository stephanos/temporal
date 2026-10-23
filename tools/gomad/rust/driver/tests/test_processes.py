import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


FIXTURE = '''import json,os,signal,sys
from pathlib import Path
sys.path.insert(0, {driver_path!r})
import driver
root=Path(sys.argv[1]); mode=sys.argv[2]
config=root/'config.json'; config.write_text('{{}}')
def exercise(config, label):
    blocked=sorted(int(s) for s in signal.pthread_sigmask(signal.SIG_BLOCK, set()))
    (root/'mask.json').write_text(json.dumps(blocked))
    assert not ({{1,2,15}} & set(blocked)), blocked
    if mode=='busy':
        with driver.busy_host(label):
            pass
        (root/'busy-returned').write_text('done')
    else:
        driver.command({{'repo':str(root)}}, root, 'held',
                       [sys.executable,'-c','import time; time.sleep(30)'])
    return 0
driver.configured=lambda path: {{}}
driver.stage=exercise
sys.argv=['driver.py','stage','--config',str(config),'busy-boundaries']
raise SystemExit(driver.main())
'''


class ProcessContract(unittest.TestCase):
    def launch(self, root, mode, receiver=False):
        fixture = root / 'stage.py'
        fixture.write_text(FIXTURE.format(driver_path=str(driver.HERE)))
        state = root / 'scheduler'
        state.mkdir()
        executor_class = driver.StageExecutor if receiver else driver.scheduler.RealExecutor
        executor = executor_class([sys.executable, str(fixture), str(root), mode], str(state), dict(os.environ))
        stage = driver.scheduler.load_dag(driver.DAG)['busy-boundaries']
        executor.start(stage, 0)
        return executor, executor.procs[stage.label][0]

    def stop(self, executor, proc):
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        proc.wait()
        for _, log, _ in executor.procs.values():
            log.close()

    def test_exited_producer_cannot_leave_live_descendants(self):
        import time
        for status in (0, 7):
            with self.subTest(status=status), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                program = "import subprocess,sys; from pathlib import Path; child=subprocess.Popen([sys.executable,'-c','import time; time.sleep(30)']); Path('child.pid').write_text(str(child.pid)); sys.exit("+str(status)+")"
                try:
                    with self.assertRaises(driver.Refusal):
                        driver.command({'repo': str(root)}, root, 'producer', [sys.executable, '-c', program])
                    child = int((root/'child.pid').read_text())
                    until = time.monotonic()+2
                    while time.monotonic()<until:
                        try:
                            state = Path(f'/proc/{child}/stat').read_text().split(') ',1)[1].split()[0]
                        except FileNotFoundError:
                            state = 'gone'
                        if state in ('gone','Z','X'): break
                        time.sleep(0.01)
                    self.assertIn(state, ('gone','Z','X'))
                    record = driver.read(root/'producer.process.json')
                    self.assertEqual(record['exit'], status)
                    self.assertFalse(record['active'])
                    self.assertEqual(record['cleanup_signal'], int(signal.SIGKILL))
                    self.assertEqual(record['observation'], 'inconclusive')
                finally:
                    if (root/'producer.process.json').exists():
                        try: os.killpg(driver.read(root/'producer.process.json')['pid'], signal.SIGKILL)
                        except ProcessLookupError: pass

    def test_actual_frozen_launcher_allows_busy_pool_to_finish(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executor, proc = self.launch(root, 'busy')
            try:
                self.assertEqual(proc.wait(timeout=3), 0, (root/'scheduler/busy-boundaries.runner.log').read_text())
                self.assertEqual(json.loads((root/'mask.json').read_text()), [])
                self.assertTrue((root/'busy-returned').exists())
            finally:
                self.stop(executor, proc)

    def test_actual_frozen_launcher_delivers_interrupt_and_reaps_command(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executor, proc = self.launch(root, 'command')
            try:
                import time
                until = time.monotonic()+3
                while not (root/'held.process.json').exists() and proc.poll() is None and time.monotonic()<until:
                    time.sleep(0.01)
                self.assertTrue((root/'held.process.json').exists())
                os.killpg(proc.pid, signal.SIGTERM)
                self.assertNotEqual(proc.wait(timeout=3), 0)
                record = driver.read(root/'held.command.json')
                self.assertEqual(record['observation'], 'inconclusive')
                self.assertIsNotNone(record['exit'])
                self.assertFalse(driver.read(root/'held.process.json')['active'])
            finally:
                self.stop(executor, proc)

    def test_cleanup_waits_for_group_quiescence_before_marking_inactive(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root/'held.process.json'
            driver.write(path, dict(stage_pid=123, pid=456, active=True))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            def live(pid):
                self.assertEqual(pid, 456)
                self.assertTrue(driver.read(path)['active'])
                return live_calls.pop(0)
            live_calls = [True, False]
            with mock.patch.object(driver.os, 'killpg') as kill, \
                 mock.patch.object(driver, 'process_group_live', live, create=True), \
                 mock.patch.object(driver.time, 'sleep'):
                self.assertTrue(executor.cleanup_commands(123))
            self.assertEqual(live_calls, [])
            kill.assert_called_once_with(456, signal.SIGKILL)
            self.assertFalse(driver.read(path)['active'])

    def test_cleanup_timeout_preserves_active_registration(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root/'held.process.json'
            driver.write(path, dict(stage_pid=123, pid=456, active=True))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            with mock.patch.object(driver.os, 'killpg'), \
                 mock.patch.object(driver, 'process_group_live', return_value=True, create=True), \
                 mock.patch.object(driver.time, 'monotonic', side_effect=[0, 6]):
                with self.assertRaisesRegex(driver.Refusal, 'process group.*quiesce'):
                    executor.cleanup_commands(123)
            self.assertTrue(driver.read(path)['active'])

    def test_killed_stage_cannot_leave_detached_command_running(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executor, proc = self.launch(root, 'command', receiver=True)
            try:
                import time
                until = time.monotonic()+3
                while not (root/'held.process.json').exists() and proc.poll() is None and time.monotonic()<until:
                    time.sleep(0.01)
                self.assertTrue((root/'held.process.json').exists(), (root/'scheduler/busy-boundaries.runner.log').read_text())
                child = driver.read(root/'held.process.json')['pid']
                os.killpg(proc.pid, signal.SIGKILL)
                proc.wait(timeout=3)
                _, code, _ = executor.wait_any()
                self.assertNotEqual(code, 0)
                self.assertFalse(driver.read(root/'held.process.json')['active'])
                try:
                    state = Path(f'/proc/{child}/stat').read_text().split(') ',1)[1].split()[0]
                except FileNotFoundError:
                    state = 'gone'
                self.assertIn(state, ('gone','Z','X'))
            finally:
                self.stop(executor, proc)


if __name__ == '__main__':
    unittest.main()
