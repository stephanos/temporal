"""Own a backend session until its descendants have exited and been reaped."""
import ctypes
import os
import resource
import select
import signal
import subprocess
import sys
import time


def restrict_jvm_descendants():
    resource.setrlimit(resource.RLIMIT_NPROC, (0, 0))


def child_setup(policy):
    if policy not in ("jvm", "process"):
        raise RuntimeError("unknown backend process ownership policy")
    if sys.platform == "darwin" and policy == "jvm":
        if os.getuid() == 0 or os.geteuid() == 0:
            raise RuntimeError("Darwin JVM descendant prevention requires a non-root user")
        return restrict_jvm_descendants
    return None


def main():
    setup = child_setup(sys.argv[1])
    if sys.platform.startswith("linux"):
        libc = ctypes.CDLL(None, use_errno=True)
        if libc.prctl(36, 1, 0, 0, 0) != 0:  # PR_SET_CHILD_SUBREAPER
            raise OSError(ctypes.get_errno(), "cannot establish backend subreaper")
    elif sys.platform != "darwin":
        raise RuntimeError("backend process ownership supports Linux and macOS only")
    child = subprocess.Popen(sys.argv[2:], stdin=subprocess.DEVNULL, start_new_session=True, preexec_fn=setup)
    try:
        while child.poll() is None:
            readable, _, _ = select.select([sys.stdin.buffer], [], [], .005)
            if readable and not os.read(sys.stdin.fileno(), 1):
                break
        status = child.poll()
    finally:
        # The child session is disjoint from the supervisor and its Rust owner.
        try:
            os.killpg(child.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        child.wait()
        if sys.platform.startswith("linux"):
            while True:
                try:
                    pid, _ = os.waitpid(-1, os.WNOHANG)
                except ChildProcessError:
                    break
                if pid == 0:
                    # Reap even descendants that started another process group.
                    with open(f"/proc/self/task/{os.getpid()}/children", encoding="ascii") as source:
                        children = source.read()
                    for descendant in children.split():
                        try:
                            os.kill(int(descendant), signal.SIGKILL)
                        except ProcessLookupError:
                            pass
                    time.sleep(.005)
    return 125 if status is None else status if status >= 0 else 128 - status


if __name__ == "__main__":
    sys.exit(main())
