#!/usr/bin/env python3
"""Apply each corpus mistake to one side, run that side's check, record where it was caught.

    model/go/corpus/run.py go|lean|scala [case-id ...]

Each edit must apply exactly once. The harness refuses to run when a target file has uncommitted
changes against HEAD, and restores every edited file byte for byte after each case. It writes
results-<side>.json beside itself.
"""
import json, os, re, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
TOOLS = os.environ.get("UMPIRE_GO_TOOLS", "/tmp/umpire-go-tools")


def run(cmd, cwd=ROOT, env=None):
    start = time.time()
    p = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True, shell=isinstance(cmd, str))
    return p.returncode, p.stdout + p.stderr, round(time.time() - start, 3)


def go_stages():
    models = "./model/go/worker/ ./model/go/nexuscaller/ ./model/go/standaloneactivity/"
    return [
        ("compile", "go build -tags test_dep ./model/go/... && go vet -tags test_dep ./model/go/..."),
        ("lint", f"{TOOLS}/exhaustive -default-signifies-exhaustive=false {models} && "
                 f"{TOOLS}/go-check-sumtype -default-signifies-exhaustive=false ./model/go/..."),
        ("test", "go test -count=1 -tags test_dep ./model/go/nexuscaller/ ./model/go/parity/"),
    ]


def lean_stages():
    env = "cd model/lean && "
    if sys.platform == "darwin":
        env += ('export SDKROOT="$(xcrun --show-sdk-path)" CC="$(xcrun --find clang)" '
                'CXX="$(xcrun --find clang++)"; PATH="$(dirname "$CC"):$PATH"; ')
    return [
        ("compile", env + "mise exec -- lake build Temporal.Feature.Nexus.Caller.Model"),
        ("pins", env + "mise exec -- lake build Temporal.Feature.Nexus.Caller.Tests"),
    ]


def scala_stages():
    here = os.path.join(ROOT, "model", "scala")
    cli = f"cd {here} && ./scala.sh"
    tools = os.environ.get("UMPIRE_SCALA_TOOLS", "/tmp/umpire-scala-tools")
    prove = (f"cd $(mktemp -d) && {tools}/stainless/stainless {here}/temporal/nexuscaller/kernel/*.scala {here}/proofs/umpire/*.scala {here}/proofs/temporal/*.scala "
             "2>&1 | sed 's/\\x1b\\[[0-9;]*m//g' | tee /dev/stderr | grep -qE 'invalid: 0 +unknown: 0'")
    return [
        ("compile", f"{cli} compile project.scala umpire temporal"),
        ("test", f"{cli} test project.scala umpire temporal"),
        ("prove", prove),
    ]


def first_error(output, side):
    output = re.sub(r"\x1b\[[0-9;]*m", "", output)
    lines = output.splitlines()
    if side == "scala":
        for i, line in enumerate(lines):
            if re.search(r"\[error\] \S+\.scala:\d+:\d+", line):
                detail = lines[i + 1].replace("[error]", "").strip() if i + 1 < len(lines) else ""
                return f"{line.replace('[error]', '').strip()} {detail}".strip()
            if "==> X" in line:
                detail = lines[i + 1].strip() if i + 1 < len(lines) else ""
                return f"{line.strip()} {detail}"[:600]
            if re.search(r"(invalid|unknown)\s+(U:|smt)", line) or "Fatal" in line:
                return line.strip()
    if side == "go":
        test = next((l.strip() for l in lines if l.strip().startswith("--- FAIL")), "")
        for i, line in enumerate(lines):
            if "Error:" in line and "Error Trace" not in line:
                detail = line.split("Error:", 1)[1].strip()
                if detail.endswith(":") and i + 1 < len(lines):
                    detail = detail + " " + lines[i + 1].strip()
                return f"{test} {detail}".strip()
        for line in lines:
            if re.search(r"\.go:\d+:\d+", line) or "--- FAIL" in line:
                return line.strip()
    else:
        for line in lines:
            if line.startswith("error:") and "build failed" not in line and "Lean exited" not in line:
                return line.strip()
    return lines[-1].strip() if lines else ""


def main():
    side = sys.argv[1]
    only = set(sys.argv[2:])
    cases = json.load(open(os.path.join(HERE, "cases.json")))
    stages = {"go": go_stages, "lean": lean_stages, "scala": scala_stages}[side]()
    results = []
    for case in cases:
        if only and case["id"] not in only:
            continue
        edits = case[side]
        files = sorted({e["file"] for e in edits})
        for f in files:
            if subprocess.run(["git", "diff", "--quiet", "HEAD", "--", f], cwd=ROOT).returncode != 0 and \
                    subprocess.run(["git", "ls-files", "--error-unmatch", f], cwd=ROOT, capture_output=True).returncode == 0:
                sys.exit(f"{f} has uncommitted changes; commit or stash them first")
        originals = {f: open(os.path.join(ROOT, f), encoding="utf-8").read() for f in files}
        try:
            texts = dict(originals)
            for e in edits:
                n = texts[e["file"]].count(e["old"])
                if n != 1:
                    raise SystemExit(f"{case['id']}: edit applies {n} times in {e['file']}")
                line = texts[e["file"]][: texts[e["file"]].index(e["old"])].count("\n") + 1
                texts[e["file"]] = texts[e["file"]].replace(e["old"], e["new"], 1)
            for f, t in texts.items():
                open(os.path.join(ROOT, f), "w", encoding="utf-8").write(t)
            caught, message, total, failing = "never", "", 0.0, []
            for name, cmd in stages:
                code, out, secs = run(cmd)
                total += secs
                if code != 0:
                    caught, message = name, first_error(out, side)
                    failing = sorted({m.group(1) for m in re.finditer(r"^--- FAIL: (\S+)", out, re.M)} |
                                     {m.group(1) for m in re.finditer(r"==> X (\S+)", re.sub(r"\x1b\[[0-9;]*m", "", out))})
                    break
            edited = os.path.basename(edits[0]["file"])
            local = edited in message and any(str(l) in message for l in range(line - 3, line + 8))
            results.append({"id": case["id"], "mistake": case["mistake"], "caught": caught,
                            "seconds": round(total, 3), "message": message[:400], "atAuthorsLine": local,
                            "editLine": line, "failingTests": failing})
            print(json.dumps(results[-1]), flush=True)
        finally:
            for f, t in originals.items():
                open(os.path.join(ROOT, f), "w", encoding="utf-8").write(t)
    json.dump(results, open(os.path.join(HERE, f"results-{side}.json"), "w"), indent=2)


if __name__ == "__main__":
    main()
