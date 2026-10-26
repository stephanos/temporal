"""Exercise real process failure and timeout propagation without Cargo or Java."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess

here = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("spike", here / "runner.py")
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)
evidence = here / "evidence/runner-controls"
evidence.mkdir(parents=True, exist_ok=True)
environment = dict(os.environ)
environment.pop("BASH_ENV", None)
for name, command, limit, failure in [
    ("nonzero", ["/bin/sh", "-c", "exit 7"], 2, subprocess.CalledProcessError),
    ("timeout", ["/bin/sh", "-c", "sleep 2"], 0.1, subprocess.TimeoutExpired),
]:
    try:
        runner.bounded(command, limit, environment, evidence / name)
    except failure:
        pass
    else:
        raise AssertionError(f"{name} incorrectly passed")
assert json.loads((evidence / "nonzero.json").read_text())["exit"] == 7
assert json.loads((evidence / "timeout.json").read_text())["exit"] == -9
print("process nonzero and timeout controls passed")
