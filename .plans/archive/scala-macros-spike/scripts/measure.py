#!/usr/bin/env python3
"""Wall-time measurements for the macro spike: today's package+classpath+lift vs the prototype's
compile-with-macros+export, cold/warm/one-line-edit, with and without a (private) Bloop server.
Each scenario runs 3 times; results go to out/timings.json and a log. Only /tmp is touched."""
import json, os, shutil, statistics, subprocess, sys, time

SPIKE = "/tmp/umpire-macro-spike"
SC = f"{SPIKE}/scripts/sc"
OUT = f"{SPIKE}/out/measure"
BLOOP = ["--bloop-daemon-dir", f"{SPIKE}/bloop-daemon"]
REPO_KETTLE = f"{SPIKE}/repo/model/framework/samples/Kettle.scala"
PROTO_KETTLE = f"{SPIKE}/proto/model/framework/samples/Kettle.scala"
LIFTER = f"{SPIKE}/proto/reify/Lifter.scala"
REPO_BUILDS = [f"{SPIKE}/repo/model/.scala-build", f"{SPIKE}/repo/model/framework/export/.scala-build"]
PROTO_BUILDS = [f"{SPIKE}/proto/model/.scala-build"]

os.makedirs(OUT, exist_ok=True)
log = open(f"{OUT}/measure.log", "a")


def say(*a):
    print(*a, flush=True)
    print(*a, file=log, flush=True)


def clean(paths):
    for p in paths:
        shutil.rmtree(p, ignore_errors=True)


def bloop_exit():
    subprocess.run([SC, "--power", "bloop", "exit", *BLOOP], capture_output=True)


toggle_state = {}


def edit(path):
    """A one-line edit to the `heat` step function: `+ 1` <-> `+ 2`."""
    s = open(path).read()
    if "state.reading + 1)" in s:
        s = s.replace("state.reading + 1)", "state.reading + 2)")
    else:
        s = s.replace("state.reading + 2)", "state.reading + 1)")
    open(path, "w").write(s)


def restore(path):
    s = open(path).read().replace("state.reading + 2)", "state.reading + 1)")
    open(path, "w").write(s)


macro_n = [0]


def macro_edit():
    """An edit to the macro implementation (a comment), then repackaging the reifier jar."""
    macro_n[0] += 1
    with open(LIFTER, "a") as f:
        f.write(f"// macro edit {macro_n[0]}\n")
    subprocess.run(
        [SC, "--power", "package", "--server=false", "--library", ".", "-f", "-o", f"{SPIKE}/proto/reify.jar"],
        cwd=f"{SPIKE}/proto/reify", capture_output=True, check=True)


def run(script, outdir, server):
    args = [f"{SPIKE}/scripts/{script}", outdir] + (BLOOP if server else ["--server=false"])
    t0 = time.time()
    p = subprocess.run(args, capture_output=True, text=True)
    wall = time.time() - t0
    if p.returncode != 0:
        say("FAILED", script, p.stdout[-2000:], p.stderr[-2000:])
        sys.exit(1)
    parts = json.loads(p.stdout.strip().splitlines()[-1])
    parts["wall"] = wall
    return parts


def check(script, outdir):
    if script == "today.sh":
        same = subprocess.run(["cmp", "-s", f"{outdir}/Kettle.json", f"{SPIKE}/out/today0/Kettle.json"]).returncode == 0
        return {"ir_equals_baseline": same}
    text = open(f"{outdir}/run.log").read()
    same = subprocess.run(["cmp", "-s", f"{outdir}/reified.json", f"{outdir}/lifted-subset.json"]).returncode == 0
    times = []
    if os.path.exists(f"{outdir}/macro-times.log"):
        times = [int(l.split()[-1]) for l in open(f"{outdir}/macro-times.log") if l.strip()]
    return {"messages_equal": "equal as messages: true" in text, "bytes_equal": same,
            "macro_expansions": len(times), "macro_micros": sum(times)}


SCENARIOS = {
    "today_cold_noserver": ("today.sh", False, lambda: clean(REPO_BUILDS)),
    "today_warm_noserver": ("today.sh", False, lambda: None),
    "today_edit_noserver": ("today.sh", False, lambda: edit(REPO_KETTLE)),
    "today_cold_server": ("today.sh", True, lambda: (bloop_exit(), clean(REPO_BUILDS))),
    "today_warm_server": ("today.sh", True, lambda: None),
    "today_edit_server": ("today.sh", True, lambda: edit(REPO_KETTLE)),
    "proto_cold_noserver": ("proto.sh", False, lambda: clean(PROTO_BUILDS)),
    "proto_warm_noserver": ("proto.sh", False, lambda: None),
    "proto_edit_noserver": ("proto.sh", False, lambda: edit(PROTO_KETTLE)),
    "proto_cold_server": ("proto.sh", True, lambda: (bloop_exit(), clean(PROTO_BUILDS))),
    "proto_warm_server": ("proto.sh", True, lambda: None),
    "proto_edit_server": ("proto.sh", True, lambda: edit(PROTO_KETTLE)),
    "proto_macroedit_server": ("proto.sh", True, macro_edit),
    # True daemon-cold: the private Bloop daemon is stopped and the build caches removed each run.
    "today_daemoncold_server": ("today.sh", True, lambda: (restore(REPO_KETTLE), bloop_exit(), clean(REPO_BUILDS))),
    "proto_daemoncold_server": ("proto.sh", True, lambda: (restore(PROTO_KETTLE), bloop_exit(), clean(PROTO_BUILDS))),
}

results = json.load(open(f"{OUT}/timings.json")) if os.path.exists(f"{OUT}/timings.json") else {}
only = sys.argv[1:] or list(SCENARIOS)
for name in only:
    script, server, prep = SCENARIOS[name]
    runs = []
    for i in range(3):
        prep()
        outdir = f"{OUT}/{name}-{i}"
        shutil.rmtree(outdir, ignore_errors=True)
        r = run(script, outdir, server)
        r.update(check(script, outdir))
        say(name, i, json.dumps(r))
        runs.append(r)
    keys = [k for k in runs[0] if isinstance(runs[0][k], float)]
    results[name] = {"runs": runs, "median": {k: statistics.median(r[k] for r in runs) for k in keys}}
    say(name, "median", json.dumps(results[name]["median"]))
    json.dump(results, open(f"{OUT}/timings.json", "w"), indent=1)
restore(REPO_KETTLE)
restore(PROTO_KETTLE)
