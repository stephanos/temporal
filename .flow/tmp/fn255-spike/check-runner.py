"""Exercise real process failure and timeout propagation without Cargo or Java."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from contextlib import redirect_stderr, redirect_stdout
from io import StringIO
from unittest.mock import patch

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

with tempfile.TemporaryDirectory(prefix="preflight-", dir=evidence) as temporary:
    root = Path(temporary)
    missing = root / "missing-tlc.jar"
    destination = root / "execute"
    stderr = StringIO()
    argv = [str(here / "runner.py"), str(destination),
            "--execute", "--tlc-jar", str(missing)]
    with patch.object(sys, "argv", argv), patch.object(
        runner, "execute", side_effect=AssertionError("execution reached before missing-jar refusal")
    ) as execute, redirect_stderr(stderr), redirect_stdout(StringIO()):
        try:
            runner.main()
        except SystemExit as error:
            assert error.code == 2
        else:
            raise AssertionError("missing jar incorrectly passed")
        execute.assert_not_called()
    assert "--tlc-jar must name" in stderr.getvalue(), stderr.getvalue()
    assert not destination.exists(), "missing jar left a prepared destination"
    source = root / "source-only"
    argv = [str(here / "runner.py"), str(source), "--tlc-jar", str(missing)]
    with patch.object(sys, "argv", argv), patch.object(
        runner, "execute", side_effect=AssertionError("source-only preparation executed")
    ) as execute, redirect_stdout(StringIO()):
        runner.main()
        execute.assert_not_called()
    assert (source / "src/case_0.rs").is_file()
    (evidence / "preflight.json").write_text(json.dumps({
        "execute_missing_jar_exit": 2,
        "execute_destination_absent": True,
        "execute_not_called": True,
        "execute_stderr": stderr.getvalue(),
        "source_only_model_created": True,
        "source_only_execute_not_called": True,
    }, indent=2))
print("missing-jar preflight and source-only preparation controls passed")
