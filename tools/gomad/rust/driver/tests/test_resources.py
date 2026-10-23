"""Real bounded child-process accounting; no compiler, guest, or qualification."""
from concurrent.futures import ThreadPoolExecutor
import os
from pathlib import Path
import resource
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class ChildResourceContract(unittest.TestCase):
    def check(self, root, name, exit_code):
        receipt = driver.read(root / (name + '.command.json'))
        resources = receipt['resources']
        self.assertEqual(resources['schema'], 'gomad.driver-child-resources/v1')
        self.assertEqual(resources['measurement'], 'wait4')
        self.assertEqual(resources['scope'], 'whole-command-child')
        self.assertEqual(resources['pid'], receipt['pid'])
        self.assertEqual(resources['waited_pid'], receipt['pid'])
        self.assertEqual(resources['exit_code'], exit_code)
        self.assertEqual(receipt['exit'], exit_code)
        self.assertEqual(os.waitstatus_to_exitcode(resources['wait_status']), exit_code)
        self.assertEqual(resources['ru_maxrss_unit'], 'KiB')
        self.assertEqual(resources['peak_rss_bytes'], resources['ru_maxrss'] * 1024)
        self.assertGreater(resources['peak_rss_bytes'], 0)
        for field, seconds in (('user_nanos', 'user_seconds'), ('system_nanos', 'system_seconds')):
            self.assertEqual(resources[field], round(resources[seconds] * 1_000_000_000))
            self.assertGreaterEqual(resources[field], 0)
        process = driver.read(root / (name + '.process.json'))
        self.assertEqual(process['resources'], resources)
        self.assertEqual(receipt['clock'], 'CLOCK_MONOTONIC')
        self.assertEqual(receipt['parent_pid'], os.getpid())
        self.assertLess(receipt['started_monotonic_nanos'], receipt['reaped_monotonic_nanos'])
        for field in ('clock', 'parent_pid', 'started_monotonic_nanos', 'reaped_monotonic_nanos'):
            self.assertEqual(receipt[field], process[field])
        self.assertFalse(process['active'])
        with self.assertRaises(ProcessLookupError):
            os.kill(receipt['pid'], 0)
        return receipt

    def test_optional_exact_environment_does_not_inherit_ambient_controls(self):
        import json
        exact={'PATH':'/usr/bin:/bin','LC_ALL':'C','ONLY':'frozen'}
        with tempfile.TemporaryDirectory() as directory,patch.dict(os.environ,{'AMBIENT_COMPILER_HOOK':'foreign'}):
            root=Path(directory)
            raw=driver.command({'repo':directory},root,'exact',[sys.executable,'-c','import json,os; print(json.dumps(dict(os.environ)))'],timeout=5,exact_environment=exact)
            self.assertEqual(json.loads(raw),exact)
            receipt=self.check(root,'exact',0)
            self.assertEqual(receipt['effective_environment'],exact)
            self.assertEqual(receipt['environment_mode'],'exact')
            driver.command({'repo':directory},root,'default',[sys.executable,'-c','pass'],timeout=5)
            self.assertIsNone(driver.read(root/'default.command.json')['effective_environment'])

    def test_nested_actual_child_ancestry_and_shared_clock_enclosure(self):
        import copy
        import json
        import primary_stages
        program = """import json,os,subprocess,sys,time
start=time.clock_gettime_ns(time.CLOCK_MONOTONIC)
child=subprocess.Popen([sys.executable,'-c','pass'])
child.wait()
print(json.dumps(dict(pid=child.pid,parent_pid=os.getpid(),clock='CLOCK_MONOTONIC',started_monotonic_nanos=start,reaped_monotonic_nanos=time.clock_gettime_ns(time.CLOCK_MONOTONIC))))
"""
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            driver.command({'repo':directory},root,'outer',[sys.executable,'-c',program],timeout=5)
            receipt=self.check(root,'outer',0)
            process=driver.read(root/'outer.process.json')
            path=root/'probe.json';path.write_bytes((root/'outer.stdout').read_bytes())
            observed=driver.read(path)
            entry=dict(observed,path=str(path),sha256=driver.sha(path))
            primary_stages.enclose_host_commands(receipt,process,{'actual':entry})
            for field,value in (('parent_pid',receipt['pid']+1),('started_monotonic_nanos',receipt['started_monotonic_nanos']-1),
                                ('reaped_monotonic_nanos',receipt['reaped_monotonic_nanos']+1),('clock','CLOCK_REALTIME')):
                altered=dict(observed,**{field:value});path.write_text(json.dumps(altered))
                changed=dict(altered,path=str(path),sha256=driver.sha(path))
                with self.subTest(field=field),self.assertRaises(driver.Refusal):
                    primary_stages.enclose_host_commands(receipt,process,{'copied':changed})
            stale=copy.deepcopy(process);stale['reaped_monotonic_nanos']+=1
            with self.assertRaises(driver.Refusal):primary_stages.enclose_host_commands(receipt,stale,{'actual':entry})

    def test_success_stdout_stderr_actual_pid_and_exact_single_wait4(self):
        program = "import os,sys,time; data=bytearray(32<<20); print(os.getpid(),flush=True); print('retained stderr',file=sys.stderr); end=time.process_time()+0.03\nwhile time.process_time()<end: pass\n"
        with tempfile.TemporaryDirectory() as directory, patch.object(driver.os, 'wait4', wraps=os.wait4) as wait:
            root = Path(directory)
            output = driver.command({'repo': directory}, root, 'success', [sys.executable, '-c', program], timeout=5)
            receipt = self.check(root, 'success', 0)
            self.assertEqual(int(output), receipt['pid'])
            self.assertEqual((root / 'success.stderr').read_bytes(), b'retained stderr\n')
            self.assertGreaterEqual(receipt['resources']['user_nanos'] + receipt['resources']['system_nanos'], 30_000_000)
            wait.assert_called_once_with(receipt['pid'], os.WNOHANG)

    def test_nonzero_accepted_and_refused_retain_original_status(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(driver.os, 'wait4', wraps=os.wait4) as wait:
            root = Path(directory)
            argv = [sys.executable, '-c', "import sys; print('output'); print('error',file=sys.stderr); sys.exit(7)"]
            self.assertEqual(driver.command({'repo': directory}, root, 'accepted', argv, accepted=(7,), timeout=5), b'output\n')
            with self.assertRaisesRegex(driver.Refusal, 'exit 7'):
                driver.command({'repo': directory}, root, 'refused', argv, timeout=5)
            for name in ('accepted', 'refused'):
                receipt = self.check(root, name, 7)
                self.assertEqual(receipt['observation'], 'completed')
                self.assertEqual((root / (name + '.stderr')).read_bytes(), b'error\n')
            self.assertEqual(wait.call_count, 2)

    def test_signal_exit_is_negative_and_measured_once(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(driver.os, 'wait4', wraps=os.wait4) as wait:
            root = Path(directory)
            driver.command({'repo': directory}, root, 'signal', [sys.executable, '-c',
                'import os,signal; os.kill(os.getpid(),signal.SIGTERM)'], accepted=(-signal.SIGTERM,), timeout=5)
            receipt = self.check(root, 'signal', -signal.SIGTERM)
            self.assertTrue(os.WIFSIGNALED(receipt['resources']['wait_status']))
            self.assertEqual(wait.call_count, 1)

    def test_timeout_kills_reaps_and_keeps_timeout_precedence(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(driver.os, 'wait4', wraps=os.wait4) as wait:
            root = Path(directory)
            with self.assertRaises(subprocess.TimeoutExpired):
                driver.command({'repo': directory}, root, 'timeout',
                               [sys.executable, '-c', 'import time; time.sleep(30)'], timeout=0.03)
            receipt = self.check(root, 'timeout', -signal.SIGKILL)
            self.assertEqual(receipt['observation'], 'inconclusive')
            self.assertEqual(wait.call_count, 1)

    def test_interrupt_after_reap_uses_cached_status_without_double_wait(self):
        actual_wait4 = os.wait4
        def interrupt_after_reap(pid, options):
            result = actual_wait4(pid, options)
            os.kill(os.getpid(), signal.SIGINT)  # blocked until sample/status retained
            return result
        original = signal.signal(signal.SIGINT, signal.default_int_handler)
        self.addCleanup(signal.signal, signal.SIGINT, original)
        with tempfile.TemporaryDirectory() as directory, patch.object(driver.os, 'wait4', side_effect=interrupt_after_reap) as wait:
            root = Path(directory)
            with self.assertRaises(KeyboardInterrupt):
                driver.command({'repo': directory}, root, 'interrupted', [sys.executable, '-c', 'pass'], timeout=5)
            receipt = self.check(root, 'interrupted', 0)
            self.assertEqual(receipt['observation'], 'inconclusive')
            self.assertEqual(wait.call_count, 1)

    def test_concurrent_children_and_unrelated_child_have_separate_usage(self):
        # Linux child peaks include the forked parent's resident memory before exec.
        allocation = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024 + (48 << 20)
        heavy = f"import os,time; data=bytearray({allocation}); print(os.getpid(),flush=True); end=time.process_time()+0.08\nwhile time.process_time()<end: pass\n"
        light = 'import os,time; print(os.getpid(),flush=True); time.sleep(0.08)'
        unrelated = f'import time; data=bytearray({allocation + (48 << 20)}); time.sleep(0.12)'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            other = subprocess.Popen([sys.executable, '-c', unrelated])
            try:
                with patch.object(driver.os, 'wait4', wraps=os.wait4) as wait, ThreadPoolExecutor(max_workers=2) as pool:
                    results = [pool.submit(driver.command, {'repo': directory}, root, name,
                                           [sys.executable, '-c', source], timeout=5)
                               for name, source in (('heavy', heavy), ('light', light))]
                    pids = [int(result.result()) for result in results]
                    measurements = [self.check(root, name, 0)['resources'] for name in ('heavy', 'light')]
                    self.assertEqual({call.args[0] for call in wait.call_args_list}, set(pids))
                    self.assertEqual(wait.call_count, 2)
                    self.assertNotIn(other.pid, pids)
                    self.assertGreater(measurements[0]['user_nanos']+measurements[0]['system_nanos'],
                                       measurements[1]['user_nanos']+measurements[1]['system_nanos'])
                    self.assertGreater(measurements[0]['peak_rss_bytes'], measurements[1]['peak_rss_bytes'])
            finally:
                other.wait(timeout=5)
