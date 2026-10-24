"""The shared backend supervisor survives its owner long enough to reap descendants."""
import ctypes
import importlib.util
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

SUPERVISOR = Path(__file__).resolve().parents[2] / "umpire/backends/tla/src/process.py"

spec = importlib.util.spec_from_file_location("backend_process", SUPERVISOR)
backend_process = importlib.util.module_from_spec(spec)
spec.loader.exec_module(backend_process)


class DarwinPolicy(unittest.TestCase):
    def test_all_darwin_helpers_prevent_process_creation(self):
        with patch.object(backend_process.sys, "platform", "darwin"), \
             patch.object(backend_process.os, "getuid", return_value=501), \
             patch.object(backend_process.os, "geteuid", return_value=501), \
             patch.object(backend_process.resource, "setrlimit") as limits:
            backend_process.child_setup()()
            limits.assert_called_once_with(backend_process.resource.RLIMIT_NPROC, (0, 0))
        for uid, euid in [(0, 501), (501, 0), (0, 0)]:
            with self.subTest(uid=uid, euid=euid), \
                 patch.object(backend_process.sys, "platform", "darwin"), \
                 patch.object(backend_process.os, "getuid", return_value=uid), \
                 patch.object(backend_process.os, "geteuid", return_value=euid):
                with self.assertRaisesRegex(RuntimeError, "non-root"):
                    backend_process.child_setup()
        with patch.object(backend_process.sys, "platform", "linux"):
            self.assertIsNone(backend_process.child_setup())



@unittest.skipUnless(sys.platform.startswith("linux"), "Linux subreaper proof")
class Supervisor(unittest.TestCase):
    @unittest.skipIf(os.getuid() == 0 or os.geteuid() == 0, "NPROC requires a non-root user")
    def test_simulated_darwin_helper_cannot_start_an_escaped_descendant(self):
        libc = ctypes.CDLL(None, use_errno=True)
        prior = ctypes.c_int()
        self.assertEqual(libc.prctl(37, ctypes.byref(prior), 0, 0, 0), 0)
        self.assertEqual(libc.prctl(36, 1, 0, 0, 0), 0)
        helper = """import subprocess,sys
try:
    p = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
except BlockingIOError:
    print('descendant creation refused')
    sys.exit(75)
print('escaped ' + str(p.pid))
"""
        launch = "import sys; sys.platform='darwin'; exec(compile(sys.argv.pop(1), '<supervisor>', 'exec'))"
        owned = []
        try:
            with subprocess.Popen([sys.executable, "-c", launch, SUPERVISOR.read_text(), sys.executable, "-c", helper], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE) as process:
                process.wait(timeout=5)
                output, errors = process.communicate()
            if output.startswith(b"escaped "):
                owned.append(int(output.split()[1]))
            self.assertEqual(process.returncode, 75, output + errors)
            self.assertEqual(output, b"descendant creation refused\n")
        finally:
            for pid in owned:
                try:
                    os.kill(pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            while True:
                try:
                    os.waitpid(-1, 0)
                except ChildProcessError:
                    break
            self.assertEqual(libc.prctl(36, prior.value, 0, 0, 0), 0)

    @unittest.skipIf(os.getuid() == 0 or os.geteuid() == 0, "NPROC requires a non-root user")
    def test_installer_containment_preflight_requires_no_descendants(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "workspace"
            scripts = root / "model/scripts"
            scripts.mkdir(parents=True)
            script = scripts / "fetch-tlc.sh"
            shutil.copy2(SUPERVISOR.parents[4] / "scripts/fetch-tlc.sh", script)
            outside = Path(tmp) / "outside"
            outside.mkdir()
            (root / "model/build").symlink_to(outside, target_is_directory=True)
            self._assert_preflight(script)
            self.assertFalse((outside / "tlc").exists())

    def _assert_preflight(self, script, timeout=2):
        result = subprocess.run(["bash", str(script)], preexec_fn=backend_process.restrict_descendants, capture_output=True, timeout=timeout, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"TLC cache escapes repository", result.stdout + result.stderr)

    @unittest.skipIf(os.getuid() == 0 or os.geteuid() == 0, "NPROC requires a non-root user")
    def test_diagnostic_text_cannot_make_a_hung_preflight_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp) / "hang.sh"
            script.write_text("printf 'TLC cache escapes repository\\n' >&2; while :; do :; done\n")
            with self.assertRaises(subprocess.TimeoutExpired):
                self._assert_preflight(script, timeout=.1)

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
p = subprocess.Popen([sys.executable, sys.argv[1], sys.executable, '-c', sys.argv[2], sys.argv[3]], stdin=subprocess.PIPE, process_group=0)
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
p = subprocess.Popen([sys.executable, sys.argv[1], sys.executable, '-c', sys.argv[2], sys.argv[3]], stdin=subprocess.PIPE)
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
