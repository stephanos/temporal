#!/usr/bin/env python3
"""Stage-set and command equivalence between the parallel driver and the serial reference.

static: before any stage runs (source-only, never invokes a gate)
  * required labels = serial reference labels minus the retired native-* stages;
  * DAG labels == required labels, each exactly once;
  * every required label resolves to one stage call in lib.sh whose command
    text is identical to the reference call (stage / named_test / rust_test),
    and the shared helper functions are byte-identical to the reference;
  * the workload entry list matches the reference;
  * cwd and every GOMAD_* export at each stage call match the serial order (envtrace.py).
verify: after the run
  * stages.log holds exactly one START and one `EXIT <label> 0` per required
    label, no other label and no STEP-FAIL line; every stage has its command
    record and a receipt with its label, exit 0 and the run's source manifest.
"""
import argparse
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import envtrace  # noqa: E402

DROPPED_PREFIX = "native-"  # native backend retired 2026-10-19 (specs closed won't-do)
CALL = re.compile(r'^\s*(stage|named_test|named_native_test|rust_test) +("?)([^\s"]+)\2(?: +(.*))?$', re.M)
# Shared bodies that must stay byte-identical to the reference driver.
IDENTICAL_FUNCTIONS = ["clean_measure", "busy12", "digest_file", "named_test", "one_path", "check_pair",
                       "check_primary", "capture_host_cpu", "rust_test", "prune_bulk"]
# Differs only by writing canonical-helper.sha256 atomically (concurrent readers).
NEAR_IDENTICAL = {"publish_ordinary_helper": ('sha256sum -- "$destination" > "$QUAL_ROOT/canonical-helper.sha256"',
                                              'atomic_output "$QUAL_ROOT/canonical-helper.sha256" sha256sum -- "$destination"')}
WORKLOAD_ENTRY = re.compile(r"'([\w-]+) (\w+) (Test\w+)'")


def joined(text):
    return text.replace("\\\n", " ")


def normalize(command):
    return " ".join((command or "").split())


def function_body(text, name):
    match = re.search(r"^" + re.escape(name) + r"\(\) [({]\n(.*?)^[)}]\n", text, re.M | re.S)
    match = match or re.search(r"^" + re.escape(name) + r"\(\) (\{ .* \})$", text, re.M)
    if not match:
        raise SystemExit(f"function {name} not found")
    return match.group(1)


def reference_calls(text, templates=None):
    """label -> (kind, command) with the reference's loops expanded from their literal bounds."""
    text = joined(text)
    calls = {}
    templates = {} if templates is None else templates
    for kind, _, label, command in CALL.findall(text):
        if label == "$label":
            continue
        if label == "runtime-$index":
            assert "for index in $(seq 0 31); do" in text
            expanded = [f"runtime-{i}" for i in range(32)]
        elif label == "sqlite-selection-$index":
            assert "for index in 0 1 2 3; do" in text
            expanded = [f"sqlite-selection-{i}" for i in range(4)]
        elif label == "workload-$workload-$profile-$cell":
            entries = WORKLOAD_ENTRY.findall(text)
            assert len(entries) == 5 and "for cell in success canary; do" in text
            expanded = [f"workload-{w}-{p}-{c}" for w, p, _ in entries for c in ("success", "canary")]
        else:
            assert "$" not in label, label
            expanded = [label]
        if "$" in label:
            templates[label] = expanded
        for name in expanded:
            assert name not in calls, f"duplicate reference label {name}"
            calls[name] = (kind, normalize(command))
    return calls


def lib_calls(text, templates):
    """label -> list of (kind, command, template); loop templates expand exactly as in the reference."""
    text = joined(text)
    calls = {}
    for kind, _, label, command in CALL.findall(text):
        if label == "$label":
            continue
        matches = templates.get(label, [label])
        for name in matches:
            calls.setdefault(name, []).append((kind, normalize(command), label))
    return calls


def static(args):
    reference_text = Path(args.reference).read_text()
    lib_text = Path(args.lib).read_text()
    templates = {}
    reference = reference_calls(reference_text, templates)
    dropped = sorted(label for label in reference if label.startswith(DROPPED_PREFIX))
    required = sorted(set(reference) - set(dropped))
    dag = [line.split("\t")[0] for line in Path(args.dag).read_text().splitlines()
           if line and not line.startswith("#")][1:]
    problems = []
    if len(dag) != len(set(dag)):
        problems.append("duplicate DAG labels")
    if sorted(dag) != required:
        problems.append({"dag_missing": sorted(set(required) - set(dag)), "dag_extra": sorted(set(dag) - set(required))})
    lib = lib_calls(lib_text, templates)
    for label in required:
        found = lib.get(label, [])
        if len(found) != 1:
            problems.append(f"{label}: {len(found)} stage calls in lib.sh")
            continue
        kind, command, _ = found[0]
        if (kind, command) != reference[label]:
            problems.append({"label": label, "reference": reference[label], "lib": [kind, command]})
    extra = sorted(set(lib) - set(required))
    if extra:
        problems.append({"lib_calls_outside_required_set": extra})
    if WORKLOAD_ENTRY.findall(joined(lib_text)) != WORKLOAD_ENTRY.findall(joined(reference_text)):
        problems.append("workload entry list differs from the reference")
    for name in IDENTICAL_FUNCTIONS:
        if function_body(lib_text, name) != function_body(reference_text, name):
            problems.append(f"function {name} differs from the reference")
    # `stage "$label"` escapes the per-label comparison; allow it only inside the
    # reference-identical helper bodies (named_test, rust_test).
    dynamic = len(re.findall(r'^\s*stage "\$label"', lib_text, re.M))
    allowed = sum(len(re.findall(r'^\s*stage "\$label"', function_body(lib_text, name), re.M)) for name in IDENTICAL_FUNCTIONS)
    if dynamic != allowed:
        problems.append(f"{dynamic - allowed} stage \"$label\" call(s) outside the reference helper bodies")
    _, _, env_problems = envtrace.compare(args.reference, args.lib)
    problems.extend(f"environment: {problem}" for problem in env_problems)
    for name, (old, new) in NEAR_IDENTICAL.items():
        if function_body(lib_text, name) != function_body(reference_text, name).replace(old, new):
            problems.append(f"function {name} differs from the reference beyond the atomic write")
    report = {"reference_labels": len(reference), "dropped": dropped, "required": len(required),
              "dag": len(dag), "commands_identical": not problems, "problems": problems}
    print(json.dumps(report, indent=1))
    return 0 if not problems else 4


def verify(args):
    dag = [line.split("\t")[0] for line in Path(args.dag).read_text().splitlines()
           if line and not line.startswith("#")][1:]
    starts, exits, problems_seen = {}, {}, []
    for line in Path(args.stages_log).read_text().splitlines():
        fields = line.split()
        if len(fields) >= 3 and fields[1] == "START":
            starts[fields[2]] = starts.get(fields[2], 0) + 1
        elif len(fields) >= 4 and fields[1] == "EXIT":
            exits.setdefault(fields[2], []).append(fields[3])
        elif len(fields) >= 2 and fields[1] == "STEP-FAIL":
            problems_seen.append(line)
    problems = [f"step failure recorded: {line}" for line in problems_seen]
    for label in dag:
        if starts.get(label) != 1 or exits.get(label) != ["0"]:
            problems.append(f"{label}: START x{starts.get(label, 0)}, EXIT {exits.get(label, [])}")
        receipt = Path(args.root) / f"{label}.receipt.json"
        record = json.loads(receipt.read_text()) if receipt.is_file() else {}
        if record.get("exit") != 0 or record.get("label") != label:
            problems.append(f"{label}: receipt missing, mislabeled or non-zero")
        if args.source and record.get("source_manifest_sha256") != args.source:
            problems.append(f"{label}: receipt source manifest differs")
        if not (Path(args.root) / f"{label}.command").is_file():
            problems.append(f"{label}: command record missing")
    unexpected = sorted((set(starts) | set(exits)) - set(dag))
    if unexpected:
        problems.append({"unexpected_labels": unexpected})
    print(json.dumps({"required": len(dag), "executed_exactly_once": not problems, "problems": problems}, indent=1))
    return 0 if not problems else 4


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="mode", required=True)
    s = sub.add_parser("static")
    s.add_argument("--reference", required=True)
    s.add_argument("--lib", required=True)
    s.add_argument("--dag", required=True)
    v = sub.add_parser("verify")
    v.add_argument("--dag", required=True)
    v.add_argument("--stages-log", required=True)
    v.add_argument("--root", required=True)
    v.add_argument("--source", help="expected source_manifest_sha256 in every receipt")
    args = parser.parse_args()
    return static(args) if args.mode == "static" else verify(args)


if __name__ == "__main__":
    sys.exit(main())
