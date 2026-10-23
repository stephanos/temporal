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
                self.assertTrue(executor.cleanup_commands({123}))
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
                 mock.patch.object(driver.time, 'monotonic', side_effect=[0, 6]), \
                 mock.patch.object(driver, 'wait_process_session', return_value=False):
                with self.assertRaisesRegex(driver.Refusal, 'process group.*quiesce'):
                    executor.cleanup_commands({123})
            self.assertTrue(driver.read(path)['active'])

    def test_command_waits_for_group_before_inactive_publication(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            seen = []
            def wait(pid):
                self.assertTrue(driver.read(root/'held.process.json')['active'])
                seen.append(pid)
            with mock.patch.object(driver, 'wait_process_group', side_effect=wait):
                driver.command({'repo': str(root)}, root, 'held', [sys.executable, '-c', 'pass'])
            self.assertEqual(seen, [driver.read(root/'held.process.json')['pid']])
            self.assertFalse(driver.read(root/'held.process.json')['active'])

    def test_command_quiescence_refusal_keeps_active_and_inconclusive_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with mock.patch.object(driver, 'wait_process_group', side_effect=driver.Refusal('group remains active')):
                with self.assertRaisesRegex(driver.Refusal, 'group remains active'):
                    driver.command({'repo': str(root)}, root, 'held', [sys.executable, '-c', 'pass'])
            self.assertTrue(driver.read(root/'held.process.json')['active'])
            self.assertEqual(driver.read(root/'held.command.json')['observation'], 'inconclusive')

    def test_cleanup_timeout_still_kills_and_observes_other_groups(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, pid in [('a', 456), ('b', 789)]:
                driver.write(root/(name+'.process.json'), dict(stage_pid=123, pid=pid, active=True))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            killed = []
            observed = []
            def wait(pid):
                self.assertCountEqual(killed, [456, 789])
                observed.append(pid)
                if pid == 456:
                    raise driver.Refusal('process group did not quiesce after SIGKILL')
            with mock.patch.object(driver.os, 'killpg', side_effect=lambda pid, sig: killed.append(pid)), \
                 mock.patch.object(driver, 'wait_process_group', side_effect=wait):
                with self.assertRaises(driver.Refusal):
                    executor.cleanup_commands({123})
            self.assertCountEqual(observed, [456, 789])
            self.assertTrue(driver.read(root/'a.process.json')['active'])
            self.assertFalse(driver.read(root/'b.process.json')['active'])

    def test_terminate_cleanup_failure_still_attempts_every_stage(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            executor.stage_pids = dict(a=123, b=456)
            for name, stage, pid in [('a', 123, 789), ('b', 456, 987)]:
                driver.write(root/(name+'.process.json'), dict(stage_pid=stage, pid=pid, active=True))
            killed, observed = [], []
            def wait(pid):
                self.assertCountEqual(killed, [789, 987])
                observed.append(pid)
                if pid == 789:
                    raise driver.Refusal('group remains active')
            with mock.patch.object(driver.os, 'killpg', side_effect=lambda pid, sig: killed.append(pid)), \
                 mock.patch.object(driver, 'wait_process_group', side_effect=wait):
                with self.assertRaisesRegex(driver.Refusal, 'group remains active'):
                    executor.terminate()
            self.assertCountEqual(observed, [789, 987])
            self.assertTrue(driver.read(root/'a.process.json')['active'])
            self.assertFalse(driver.read(root/'b.process.json')['active'])

    def test_run_cleanup_failure_keeps_outcome_and_closes_logs(self):
        import time
        for live_stage, close_failure in ((False, False), (True, False), (True, True)):
            with self.subTest(live_stage=live_stage, close_failure=close_failure), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)/'run'
                config = dict(root=str(root), sent_epoch=time.time(), deadline_s=3600,
                              cores=1, mem_gib=1, identity={}, _config_sha256='a'*64)
                executor = mock.Mock()
                proc, log = mock.Mock(), mock.Mock()
                if close_failure:
                    log.close.side_effect = OSError('log close failed')
                executor.procs = {'stage': (proc, log, None)} if live_stage else {}
                executor.terminate.side_effect = driver.Refusal('group remains active')
                with mock.patch('host_admission.admit'), mock.patch('adapters.admission'), \
                     mock.patch.object(driver, 'guard'), \
                     mock.patch.object(driver, 'StageExecutor', return_value=executor), \
                     mock.patch.object(driver.scheduler, 'run', side_effect=driver.Refusal('stage cleanup failed')):
                    self.assertEqual(driver.run(config, root/'config.json', 'parallel'), 2)
                executor.terminate.assert_called_once()
                outcome = driver.read(root/'outcome.json')
                self.assertEqual(outcome['status'], 'INCOMPLETE')
                self.assertIn('group remains active', outcome['reason'])
                if close_failure:
                    self.assertIn('log close failed', outcome['reason'])
                if live_stage:
                    log.close.assert_called_once()

    def test_stage_group_refusal_still_attempts_owned_session_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root/'held.process.json'
            driver.write(path, dict(stage_pid=123, pid=456, active=True))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            with mock.patch.object(driver.os, 'killpg'), \
                 mock.patch.object(driver, 'process_session_groups', return_value={}), \
                 mock.patch.object(driver, 'wait_process_group', side_effect=driver.Refusal('primary group stuck')), \
                 mock.patch.object(driver, 'wait_process_session', side_effect=driver.Refusal('late compiler stuck')) as session:
                with self.assertRaisesRegex(driver.Refusal, 'primary group stuck.*late compiler stuck'):
                    executor.cleanup_commands({123})
            session.assert_called_once_with(456)
            self.assertTrue(driver.read(path)['active'])

    def test_command_group_refusal_still_attempts_owned_session_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with mock.patch.object(driver, 'wait_process_group', side_effect=driver.Refusal('primary group stuck')), \
                 mock.patch.object(driver, 'wait_process_session', return_value=False) as session:
                with self.assertRaisesRegex(driver.Refusal, 'primary group stuck'):
                    driver.command({'repo': str(root)}, root, 'held', [sys.executable, '-c', 'pass'])
            record = driver.read(root/'held.process.json')
            session.assert_called_once_with(record['pid'])
            self.assertTrue(record['active'])
            self.assertEqual(driver.read(root/'held.command.json')['observation'], 'inconclusive')

    def test_cleanup_follows_registered_tree_and_owned_sessions_only(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            rows = [('wrapper', 123, 456, False), ('native', 456, 789, True), ('foreign', 999, 987, True)]
            for name, stage, pid, active in rows:
                driver.write(root/(name+'.process.json'), dict(stage_pid=stage, pid=pid, active=active))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            killed = []
            def wait(pid):
                self.assertCountEqual(killed, [789, 790])
            with mock.patch.object(driver.os, 'killpg', side_effect=lambda pid, sig: killed.append(pid)), \
                 mock.patch.object(driver, 'process_session_groups', return_value={789: {789, 790}}, create=True), \
                 mock.patch.object(driver, 'wait_process_group', side_effect=wait), \
                 mock.patch.object(driver, 'wait_process_session', create=True):
                self.assertTrue(executor.cleanup_commands({123}))
            self.assertFalse(driver.read(root/'native.process.json')['active'])
            self.assertTrue(driver.read(root/'foreign.process.json')['active'])

    def test_cleanup_rescans_registration_published_before_wrapper_quiesces(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            driver.write(root/'wrapper.process.json', dict(stage_pid=123, pid=456, active=True))
            executor = driver.StageExecutor([], str(root/'scheduler'), {})
            killed = []
            def wait(pid):
                if pid == 456:
                    driver.write(root/'native.process.json', dict(stage_pid=456, pid=789, active=True))
            with mock.patch.object(driver.os, 'killpg', side_effect=lambda pid, sig: killed.append(pid)), \
                 mock.patch.object(driver, 'process_session_groups', return_value={}), \
                 mock.patch.object(driver, 'wait_process_group', side_effect=wait), \
                 mock.patch.object(driver, 'wait_process_session', return_value=False):
                self.assertTrue(executor.cleanup_commands({123}))
            self.assertCountEqual(killed, [456, 789])
            self.assertFalse(driver.read(root/'native.process.json')['active'])

    def test_global_teardown_reaps_registered_nested_native_and_compiler_groups(self):
        import time
        for teardown in ('terminate', 'killed-stage'):
            with self.subTest(teardown=teardown), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                compiler = "import time; time.sleep(30)"
                native = root/'native.py'
                native.write_text("import subprocess,sys,time; from pathlib import Path; child=subprocess.Popen([sys.executable,'-c',"+repr(compiler)+"],process_group=0); Path("+repr(str(root/'compiler.pid'))+").write_text(str(child.pid)); time.sleep(30)")
                wrapper = root/'wrapper.py'
                wrapper.write_text("import sys; from pathlib import Path; sys.path.insert(0,"+repr(str(driver.HERE))+"); import driver; driver.command({'repo':"+repr(str(root))+"},Path("+repr(str(root))+"),'native',[sys.executable,"+repr(str(native))+"])")
                stage_file = root/'stage.py'
                stage_file.write_text("import sys,signal; signal.pthread_sigmask(signal.SIG_UNBLOCK,{signal.SIGTERM,signal.SIGHUP,signal.SIGINT}); from pathlib import Path; sys.path.insert(0,"+repr(str(driver.HERE))+"); import driver; driver.command({'repo':"+repr(str(root))+"},Path("+repr(str(root))+"),'wrapper',[sys.executable,"+repr(str(wrapper))+"])")
                executor = driver.StageExecutor([sys.executable, str(stage_file)], str(root/'scheduler'), dict(os.environ))
                stage = driver.scheduler.load_dag(driver.DAG)['busy-boundaries']
                foreign = subprocess.Popen([sys.executable, '-c', compiler], start_new_session=True)
                proc = None
                try:
                    executor.start(stage, 0)
                    proc = executor.procs[stage.label][0]
                    until = time.monotonic()+3
                    while not (root/'compiler.pid').exists() and proc.poll() is None and time.monotonic()<until:
                        time.sleep(0.01)
                    self.assertTrue((root/'compiler.pid').exists())
                    compiler_pid = int((root/'compiler.pid').read_text())
                    native_pid = driver.read(root/'native.process.json')['pid']
                    self.assertEqual(os.getsid(compiler_pid), native_pid)
                    self.assertEqual(os.getpgid(compiler_pid), compiler_pid)
                    if teardown == 'terminate':
                        executor.terminate()
                    else:
                        os.killpg(proc.pid, signal.SIGKILL)
                        proc.wait(timeout=3)
                        self.assertNotEqual(executor.wait_any()[1], 0)
                    self.assertFalse(driver.process_group_live(compiler_pid))
                    for name in ('wrapper', 'native'):
                        record = driver.read(root/(name+'.process.json'))
                        self.assertFalse(record['active'])
                        self.assertFalse(driver.process_group_live(record['pid']))
                    self.assertIsNone(foreign.poll())
                finally:
                    for name in ('compiler.pid',):
                        if (root/name).exists():
                            try: os.killpg(int((root/name).read_text()), signal.SIGKILL)
                            except ProcessLookupError: pass
                    for path in root.glob('*.process.json'):
                        try: os.killpg(driver.read(path)['pid'], signal.SIGKILL)
                        except ProcessLookupError: pass
                    foreign.kill()
                    foreign.wait()
                    if proc is not None:
                        self.stop(executor, proc)

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
