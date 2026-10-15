# Summarize fn-182 R4 runs: <EV>/<side>-<workload>-rep<N>/workloads/...
import json, os, sys, glob, statistics, collections
ev = sys.argv[1]
rows = collections.defaultdict(lambda: collections.defaultdict(list))
ids = collections.defaultdict(set)
for root in sorted(glob.glob(os.path.join(ev, "*-rep*"))):
    side, rest = os.path.basename(root).split("-", 1); workload = rest.rsplit("-rep", 1)[0]
    results = [p for p in glob.glob(os.path.join(root, "workloads/wasm-qualifications/v1/*.json")) if not p.endswith(".publication.json")]
    if len(results) != 1: rows[(workload, side)]["missing"].append(root); continue
    d = json.load(open(results[0])); m = rows[(workload, side)]
    t = d["Qualification"]["evidence"]["target"]
    ids[(workload, side)].add((t["sha256"], d["Manifest"]["HelperSHA256"], d["Manifest"]["EngineSHA256"]))
    m["qualified"].append(int(d["Qualification"]["qualified"]))
    m["module_bytes"].append(int(t["size"]))
    caches = [f for f in glob.glob(os.path.join(root, "workloads/*engine-cache/*.gomad-cache-v3"))]
    m["compiled_cache_bytes"].append(sum(os.path.getsize(f) for f in caches))
    m["go_preparation_ns"].append(int(d["InitialPreparationWallNanos"]))
    s = d["Samples"]
    cold = [x for x in s if x["Phase"] == "execute" and not x["CompileCacheHit"]]
    warm = [x for x in s if x["CompileCacheHit"]]
    for x in cold[:1]:
        m["cold_compile_ns"].append(x["EngineCompileNanos"]); m["cold_helper_peak_rss"].append(x["HostPeakRSSBytes"])
    for x in warm:
        m["warm_load_ns"].append(x["CacheLoadNanos"]); m["warm_engine_prepare_ns"].append(x["EnginePrepareNanos"])
        m["warm_helper_peak_rss"].append(x["HostPeakRSSBytes"])
    for x in s:
        kind = "warm" if x["CompileCacheHit"] else "cold"
        m[f"instantiation_ns_{kind}"].append(x["InstantiationNanos"]); m[f"execution_wall_ns_{kind}"].append(x["ExecutionWallNanos"])
    m["budget_within"].append(int(d["Budget"]["WithinBudget"]) if "WithinBudget" in d["Budget"] else int(not d["Budget"]["Violations"]))
print("| workload | metric | baseline median [min,max] (n) | candidate median [min,max] (n) | delta median | beyond spread |\n|---|---|---|---|---|---|")
for workload in sorted({w for w, _ in rows}):
    b, c = rows[(workload, "base")], rows[(workload, "cand")]
    for metric in sorted(set(b) | set(c)):
        if metric == "missing": continue
        def fmt(v): return f"{statistics.median(v):,.0f} [{min(v):,},{max(v):,}] ({len(v)})" if v else "unavailable"
        bv, cv = b.get(metric, []), c.get(metric, [])
        if bv and cv:
            dm = statistics.median(cv) - statistics.median(bv)
            spread = max(max(bv) - min(bv), max(cv) - min(cv))
            beyond = "yes" if abs(dm) > spread else "no (within spread)"
            print(f"| {workload} | {metric} | {fmt(bv)} | {fmt(cv)} | {dm:+,.0f} | {beyond} |")
        else:
            print(f"| {workload} | {metric} | {fmt(bv)} | {fmt(cv)} | unavailable | |")
for k, v in sorted(rows.items()):
    if v.get("missing"): print(f"\nMISSING results {k}: {v['missing']}")
for k, v in sorted(ids.items()): print(f"\nidentities {k}: {sorted(v)}")
