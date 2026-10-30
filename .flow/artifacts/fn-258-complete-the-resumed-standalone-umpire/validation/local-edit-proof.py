"""Demonstrate actual consumers use uncommitted sibling sources, then restore bytes."""
import argparse, hashlib, json, os, subprocess, time
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--pair", required=True, type=Path)
parser.add_argument("--output", required=True, type=Path)
parser.add_argument("--target", required=True, type=Path)
args = parser.parse_args()
pair = args.pair.resolve()
out = args.output.resolve()
out.mkdir(parents=True, exist_ok=True)
env = os.environ.copy()
env.pop("BASH_ENV", None)
env.update(CARGO_BUILD_JOBS="4", CARGO_INCREMENTAL="0", MISE_TRUSTED_CONFIG_PATHS=str(pair))
env["CARGO_TARGET_DIR"] = str(args.target.resolve())
def git(repo, *argv):
    return subprocess.check_output(["git", "-C", str(repo), *argv], env=env).decode().strip()
def digest(data):
    return hashlib.sha256(data).hexdigest()
heads = {name: git(pair/name, "rev-parse", "HEAD") for name in ("umpire", "temporal")}
assert not git(pair/"umpire", "status", "--porcelain"), "framework copy must start clean"
assert not git(pair/"temporal", "status", "--porcelain"), "consumer copy must start clean"
trials = [
    ("rust", "author/dsl/src/lib.rs", b"\nconst _: () = fn258_uncommitted_rust_probe;\n", "fn258_uncommitted_rust_probe", pair/"temporal/model", ["mise", "exec", "--", "cargo", "check", "--locked", "-p", "umpire-temporal"]),
    ("go", "hosts/go/host/host.go", b"\nvar fn258UncommittedGoProbe = fn258_uncommitted_go_probe\n", "fn258_uncommitted_go_probe", pair/"temporal", ["mise", "exec", "--", "go", "test", "-tags", "test_dep", "-run", "^$", "./common/testing/testpilot/temporal/act"]),
]
results = []
for language, relative, mutation, marker, cwd, command in trials:
    source = pair/"umpire"/relative
    original = source.read_bytes()
    runs = []
    def run(phase):
        started = time.monotonic()
        log = out/f"{language}-{phase}.log"
        with log.open("wb") as stream:
            result = subprocess.run(command, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT)
        runs.append(dict(phase=phase, command=command, cwd=str(cwd), exit_code=result.returncode, seconds=time.monotonic()-started, log=log.name, log_sha256=digest(log.read_bytes())))
        return result.returncode, log.read_text(errors="replace")
    try:
        code, _ = run("baseline")
        assert code == 0, f"{language} baseline failed"
        source.write_bytes(original+mutation)
        code, output = run("uncommitted")
        assert code != 0 and marker in output, f"{language} consumer did not report the expected local-source marker"
    finally:
        source.write_bytes(original)
        assert source.read_bytes() == original
    code, _ = run("restored")
    assert code == 0, f"{language} restoration failed"
    assert {name: git(pair/name, "rev-parse", "HEAD") for name in heads} == heads
    assert not git(pair/"umpire", "status", "--porcelain")
    results.append(dict(language=language, source=relative, original_sha256=digest(original), restored_sha256=digest(source.read_bytes()), mutation_marker=marker, runs=runs))
report = dict(version=1, pair=heads, go_workspace=str(pair/"go.work"), result="both actual consumers fail on recognizable uncommitted sibling edits and pass after exact restoration", no_commits_between_runs=True, trials=results)
(out/"local-edit-proof.json").write_text(json.dumps(report, indent=2)+"\n")
print(json.dumps({"pair":heads,"result":report["result"]}))
