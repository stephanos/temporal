"""The shared backend supervisor survives its owner long enough to reap descendants."""
import ctypes
import importlib.util
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

SUPERVISOR = Path(__file__).resolve().parents[2] / "umpire/backends/tlc/src/process.py"

spec = importlib.util.spec_from_file_location("backend_process", SUPERVISOR)
backend_process = importlib.util.module_from_spec(spec)
spec.loader.exec_module(backend_process)


class DarwinPolicy(unittest.TestCase):
    def test_only_explicit_jvm_policy_prevents_process_creation(self):
        with patch.object(backend_process.sys, "platform", "darwin"), \
             patch.object(backend_process.os, "getuid", return_value=501), \
             patch.object(backend_process.os, "geteuid", return_value=501), \
             patch.object(backend_process.resource, "setrlimit") as limits:
            self.assertIsNone(backend_process.child_setup("process"))
            backend_process.child_setup("jvm")()
            limits.assert_called_once_with(backend_process.resource.RLIMIT_NPROC, (0, 0))
        for uid, euid in [(0, 501), (501, 0), (0, 0)]:
            with self.subTest(uid=uid, euid=euid), \
                 patch.object(backend_process.sys, "platform", "darwin"), \
                 patch.object(backend_process.os, "getuid", return_value=uid), \
                 patch.object(backend_process.os, "geteuid", return_value=euid):
                with self.assertRaisesRegex(RuntimeError, "non-root"):
                    backend_process.child_setup("jvm")
        with self.assertRaisesRegex(RuntimeError, "unknown"):
            backend_process.child_setup("unknown")
        with patch.object(backend_process.sys, "platform", "linux"):
            self.assertIsNone(backend_process.child_setup("jvm"))



@unittest.skipUnless(sys.platform.startswith("linux"), "Linux subreaper proof")
class Supervisor(unittest.TestCase):
    def test_owner_group_and_direct_supervisor_termination_reap_owned_tree(self):
        libc = ctypes.CDLL(None, use_errno=True)
        prior = ctypes.c_int()
        self.assertEqual(libc.prctl(37, ctypes.byref(prior), 0, 0, 0), 0)
        self.assertEqual(libc.prctl(36, 1, 0, 0, 0), 0)
        child = """import os,subprocess,sys,time
p = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
with open(sys.argv[1], 'w') as out:
    out.write(str(os.getpid()) + ' ' + str(p.pid))
time.sleep(60)
"""
        owner = """import os,subprocess,sys,time
p = subprocess.Popen([sys.executable, sys.argv[1], "process", sys.executable, '-c', sys.argv[2], sys.argv[3]], stdin=subprocess.PIPE, process_group=0)
with open(sys.argv[4], 'w') as out:
    out.write(str(p.pid))
time.sleep(60)
"""
        try:
            for target in ["owner-group", "owner-group-kill", "supervisor"]:
                with self.subTest(target=target), tempfile.TemporaryDirectory() as tmp:
                    pids = Path(tmp) / "pids"
                    supervisor = Path(tmp) / "supervisor"
                    process = subprocess.Popen([sys.executable, "-c", owner, str(SUPERVISOR), child, str(pids), str(supervisor)], start_new_session=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                    owned = []
                    try:
                        deadline = time.monotonic() + 5
                        while (not pids.exists() or len(pids.read_text().split()) != 2) and time.monotonic() < deadline:
                            time.sleep(.005)
                        owned = [int(pid) for pid in pids.read_text().split()]
                        supervisor_pid = int(supervisor.read_text())
                        if target.startswith("owner-group"):
                            signum = signal.SIGKILL if target == "owner-group-kill" else signal.SIGTERM
                            os.killpg(process.pid, signum)
                            self.assertEqual(process.wait(timeout=5), -signum)
                        else:
                            os.kill(supervisor_pid, signal.SIGTERM)
                        deadline = time.monotonic() + 2
                        while any(Path(f"/proc/{pid}").exists() for pid in owned) and time.monotonic() < deadline:
                            time.sleep(.005)
                        self.assertFalse([pid for pid in owned if Path(f"/proc/{pid}").exists()], "owned backend PIDs survived or were not reaped before harness cleanup")
                    finally:
                        cleanup = owned + [process.pid]
                        if supervisor.exists():
                            cleanup.append(int(supervisor.read_text()))
                        for pid in cleanup:
                            try:
                                os.kill(pid, signal.SIGKILL)
                            except ProcessLookupError:
                                pass
                        process.wait(timeout=5)
                        while True:
                            try:
                                os.waitpid(-1, 0)
                            except ChildProcessError:
                                break
        finally:
            self.assertEqual(libc.prctl(36, prior.value, 0, 0, 0), 0)

    def test_owner_death_terminates_and_reaps_separate_session_descendants(self):
        child = """import os,subprocess,sys,time
p = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
with open(sys.argv[1], 'w') as out:
    out.write(str(os.getpid()) + ' ' + str(p.pid))
time.sleep(60)
"""
        owner = """import os,subprocess,sys,time
p = subprocess.Popen([sys.executable, sys.argv[1], "process", sys.executable, '-c', sys.argv[2], sys.argv[3]], stdin=subprocess.PIPE)
for _ in range(1000):
    if os.path.exists(sys.argv[3]):
        os._exit(0)
    time.sleep(.005)
raise RuntimeError('backend failed to start')
"""
        with tempfile.TemporaryDirectory() as tmp:
            pids = Path(tmp) / "pids"
            result = subprocess.run([sys.executable, "-c", owner, str(SUPERVISOR), child, str(pids)],
                                    capture_output=True, timeout=10, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, b"")
            for pid in pids.read_text().split():
                self.assertFalse(Path(f"/proc/{pid}").exists(), f"owned pid {pid} not reaped")


if __name__ == "__main__":
    unittest.main()
