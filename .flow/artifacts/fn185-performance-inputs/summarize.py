"""Summarize retained fn-182 cells, separating cluster process and store costs."""
import collections
import glob
import json
import os
import statistics
import sys

root = sys.argv[1]
rows = collections.defaultdict(lambda: collections.defaultdict(list))
identities = collections.defaultdict(list)
failures = []
for cell in sorted(glob.glob(os.path.join(root, "*-rep*"))):
    if not os.path.isdir(cell):
        continue
    side, rest = os.path.basename(cell).split("-", 1)
    workload = rest.rsplit("-rep", 1)[0]
    results = [path for path in glob.glob(os.path.join(cell, "workloads/wasm-qualifications/v1/*.json"))
               if not path.endswith(".publication.json")]
    if len(results) != 1:
        failures.append(f"{cell}: expected one sealed Result, found {len(results)}")
        continue
    with open(results[0]) as stream:
        result = json.load(stream)
    values = rows[workload, side]
    target = result["Qualification"]["evidence"]["target"]
    identities[workload, side].append({
        "result": results[0], "module": target["sha256"], "manifest": result["Manifest"],
        "preparation_provenance": result["PreparationProvenanceSHA256"],
    })
    if (not result["Qualification"]["qualified"] or not result["Budget"]["WithinBudget"]
            or result["Manifest"]["Provisional"]):
        failures.append(f"{cell}: qualification, final-source binding or budget failed")
    values["module_bytes"].append(int(target["size"]))
    caches = glob.glob(os.path.join(cell, "workloads/*engine-cache/*.gomad-cache-v3"))
    if len(caches) != 1:
        failures.append(f"{cell}: expected one compiled module, found {len(caches)}")
    values["compiled_cache_bytes"].append(sum(os.path.getsize(path) for path in caches))
    values["initial_go_preparation_ns"].append(result["InitialPreparationWallNanos"])
    samples = result["Samples"]
    clustered = any(sample.get("ClusterHelper") for sample in samples)
    processes = [sample for sample in samples if sample.get("ClusterHelper")] if clustered else samples
    process_index = {(sample["Phase"], sample["Iteration"]): sample for sample in processes}
    for sample in processes:
        state = "warm" if sample["CompileCacheHit"] else "cold"
        if sample["HostPeakRSSBytes"] == 0 or sample["HostPeakRSSUpperBound"]:
            failures.append(f"{cell}: unavailable helper image RSS")
        values[f"{state}_helper_peak_rss_bytes"].append(sample["HostPeakRSSBytes"])
        if state == "cold":
            if not sample["EngineCompileNanos"]:
                failures.append(f"{cell}: unavailable cold compilation")
            values["cold_compile_ns"].append(sample["EngineCompileNanos"])
        else:
            if sample["EngineCompileNanos"] != 0:
                failures.append(f"{cell}: warm helper compiled")
            if not sample["CacheLoadNanos"] or not sample["EnginePrepareNanos"]:
                failures.append(f"{cell}: unavailable warm load/preparation")
            values["warm_load_ns"].append(sample["CacheLoadNanos"])
            values["warm_engine_prepare_ns"].append(sample["EnginePrepareNanos"])
    for sample in samples:
        if sample.get("ClusterHelper"):
            continue
        process = process_index[sample["Phase"], sample["Iteration"]]
        state = "warm" if process["CompileCacheHit"] else "cold"
        values[f"{state}_fresh_instance_startup_ns"].append(sample["InstantiationNanos"])
        values[f"{state}_guest_execution_ns"].append(sample["ExecutionWallNanos"])
        if not sample["InstantiationNanos"] or not sample["ExecutionWallNanos"]:
            failures.append(f"{cell}: unavailable per-instance startup/execution")

if not rows:
    failures.append("no retained cells")
summary = {}
print("| workload | metric | baseline median [min,max] (n) | candidate median [min,max] (n) | delta | beyond spread |")
print("|---|---|---|---|---|---|")
for workload in sorted({workload for workload, _ in rows}):
    baseline, candidate = rows[workload, "base"], rows[workload, "cand"]
    summary[workload] = {}
    for metric in sorted(set(baseline) | set(candidate)):
        bv, cv = baseline.get(metric, []), candidate.get(metric, [])
        if not bv or not cv:
            failures.append(f"{workload}/{metric}: missing matched baseline or candidate")
        def describe(values):
            return {"median": statistics.median(values), "min": min(values), "max": max(values), "n": len(values), "raw": values} if values else None
        b, c = describe(bv), describe(cv)
        delta = c["median"] - b["median"] if b and c else None
        beyond = abs(delta) > max(b["max"] - b["min"], c["max"] - c["min"]) if b and c else None
        summary[workload][metric] = {"base": b, "cand": c, "delta": delta, "beyond_spread": beyond}
        def display(value):
            return f'{value["median"]:,.0f} [{value["min"]:,},{value["max"]:,}] ({value["n"]})' if value else "unavailable"
        difference = f"{delta:+,.0f}" if delta is not None else "unavailable"
        label = "yes" if beyond else "within spread" if beyond is not None else "unavailable"
        print(f"| {workload} | {metric} | {display(b)} | {display(c)} | {difference} | {label} |")
with open(os.path.join(root, "summary.json"), "w") as stream:
    json.dump({"measurements": summary, "identities": {f"{w}/{s}": ids for (w, s), ids in identities.items()}, "failures": failures}, stream, indent=2)
for failure in failures:
    print(f"\nUNAVAILABLE: {failure}")
if failures:
    sys.exit(1)
