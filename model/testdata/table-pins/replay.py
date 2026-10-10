"""Replay the inherited interpreter dumper against immutable whole-table pins."""

import argparse
import fcntl
import gzip
import hashlib
import json
import os
import pathlib
import subprocess
import sys
import time


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(path):
    result = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1 << 20), b""):
            result.update(block)
    return result.hexdigest()


def require_universe(expected, actual, label):
    require(expected == actual,
            f"{label} subject universe: missing={sorted(expected - actual)} extra={sorted(actual - expected)}")


def ir_universe(path):
    model = json.loads(path.read_text())
    entries = [(kind, item["name"]) for kind, field in (("machine", "machines"),
                                                       ("composition", "compositions"))
               for item in model.get(field, [])]
    require(len(entries) == len(set(entries)), f"duplicate lifted subjects in {path}")
    return set(entries)


def verify_stream(path, pin):
    require(pin["status"] == "complete", f"unavailable table {pin['owner']}")
    hashed = hashlib.sha256()
    lines = 0
    first = last = None
    with gzip.open(path, "rb") as source:
        for line in source:
            if first is None:
                first = line
            last = line
            hashed.update(line)
            lines += 1
    require(first is not None and last is not None, f"empty table {path}")
    header, footer = json.loads(first), json.loads(last)
    require(header["kind"] == "header" and footer["kind"] == "footer", f"header/footer missing {path}")
    require((header["artifact"], header["subject_kind"], header["owner"]) ==
            (pin["artifact"], pin["kind"], pin["owner"]), f"table identity changed {path}")
    require(len(header["states"]) == pin["states"] and len(header["actions"]) == pin["actions"],
            f"catalog sizes changed {path}")
    for field, expected in pin.get("catalogs", {}).items():
        values = header[field] or []
        actual = dict(size=len(values), sha256=hashlib.sha256("\n".join(values).encode()).hexdigest())
        require(actual == expected, f"ordered {field} catalog differs {path}")
    if "binding_order" in pin:
        bindings = [step["action"].rsplit(".", 1)[-1] for step in header["declaration"].get("steps", [])]
        require(bindings == pin["binding_order"], f"binding order differs {path}")
    expected_footer = dict(kind="footer", status="complete", states=pin["states"],
                           actions=pin["actions"], pairs=pin["state_action_pairs"],
                           enabled=pin["enabled_pairs"], disabled=pin["disabled_pairs"],
                           unknown=pin["unknown_pairs"], results=pin["results"])
    require(footer == expected_footer, f"pair/result completeness changed {path}")
    require(lines == pin["lines"], f"ordered record count changed {path}")
    require(hashed.hexdigest() == pin["uncompressed_jsonl_sha256"],
            f"complete ordered table differs {path}")
    return dict(records=lines, sha256=hashed.hexdigest())


def verify_artifact(folder, artifact_pin, ir_path):
    expected = {(p["kind"], p["owner"]) for p in artifact_pin["subjects"]}
    require(len(expected) == len(artifact_pin["subjects"]), "duplicate pinned subjects")
    require_universe(expected, ir_universe(ir_path), ir_path.name)
    inventory = json.loads((folder / "inventory.json").read_text())
    actual = {(p["kind"], p["owner"]) for p in inventory["subjects"]}
    require(len(actual) == len(inventory["subjects"]), "duplicate interpreter subjects")
    require_universe(expected, actual, ir_path.name + " interpreter")
    expected_files = {f"{kind}-{owner}.jsonl.gz" for kind, owner in expected}
    require_universe(expected_files, {p.name for p in folder.glob("*.jsonl.gz")}, "dump files")
    require(inventory["artifact"] == ir_path.name == artifact_pin["artifact"], "artifact identity changed")
    require(inventory["scope"] == artifact_pin["scope"], "interpreter scope changed")
    for field in ("declared_machines", "declared_compositions"):
        require(inventory[field] == artifact_pin[field], f"{field} changed")
    observed = {(p["kind"], p["owner"]): p for p in inventory["subjects"]}
    summaries = []
    for pin in artifact_pin["subjects"]:
        current = observed[(pin["kind"], pin["owner"])]
        for field, value in pin.items():
            if field in ("catalogs", "binding_order"):
                continue
            require(current.get(field) == value, f"{pin['owner']}: {field} differs")
        filename = f"{pin['kind']}-{pin['owner']}.jsonl.gz"
        require(pathlib.Path(current["path"]).name == filename, "mismatched dump filename")
        summaries.append(dict(owner=pin["owner"], kind=pin["kind"],
                              **verify_stream(folder / filename, pin)))
    return summaries


def replay(arguments):
    pins_hash = digest(arguments.pins)
    pins = json.loads(arguments.pins.read_text())
    require(pins["format_version"] == 1, "unsupported pin format")
    require(arguments.dumper.is_file(), f"inconclusive: inherited dumper missing: {arguments.dumper}")
    require(digest(arguments.dumper) == pins["seam"]["binary_sha256"],
            "inconclusive: inherited dumper identity changed")
    expected_artifacts = {p["artifact"] for p in pins["artifacts"]}
    candidates = {p.name for p in arguments.ir.glob("*.json")
                  if p.name.startswith(("activity-", "nexus-")) and not p.name.endswith((".lint.json", ".waivers.json"))}
    require_universe(expected_artifacts, candidates, "activity/Nexus IR files")
    require(not arguments.out.exists(), "refusing to overwrite a replay; choose a fresh scratch directory")
    arguments.out.mkdir(parents=True)
    summaries = []
    environment = {**os.environ, "BASH_ENV": "/dev/null"}
    with open("/tmp/umpire-heavy-gates.lock", "a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        for pin in pins["artifacts"]:
            ir_path = arguments.ir / pin["artifact"]
            input_hash = digest(ir_path)
            expected = {(p["kind"], p["owner"]) for p in pin["subjects"]}
            require_universe(expected, ir_universe(ir_path), ir_path.name)
            folder = arguments.out / ir_path.stem
            command = [str(arguments.dumper.resolve()), str(ir_path.resolve()), str(folder.resolve())]
            started = time.monotonic()
            with (arguments.out / (ir_path.stem + ".log")).open("wb") as log:
                process = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT,
                                         env=environment, timeout=600)
            require(process.returncode == 0, f"inconclusive: native dump exited {process.returncode}: {command}")
            require(digest(ir_path) == input_hash, f"IR input changed during replay {ir_path}")
            subjects = verify_artifact(folder, pin, ir_path)
            summary = dict(artifact=ir_path.name, input_sha256=input_hash,
                           baseline_ir_sha256=pin["ir_sha256"], command=command,
                           seconds=time.monotonic() - started, exit_code=process.returncode, subjects=subjects)
            summaries.append(summary)
            print(json.dumps(summary), flush=True)
    require(digest(arguments.pins) == pins_hash, "pin inputs changed during replay")
    require(digest(arguments.dumper) == pins["seam"]["binary_sha256"], "dumper changed during replay")
    return dict(status="complete", pins_sha256=pins_hash,
                dumper_sha256=digest(arguments.dumper), artifacts=summaries,
                validated_records=sum(s["records"] for a in summaries for s in a["subjects"]))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ir", type=pathlib.Path, required=True)
    parser.add_argument("--dumper", type=pathlib.Path, required=True)
    parser.add_argument("--out", type=pathlib.Path, required=True)
    parser.add_argument("--pins", type=pathlib.Path, default=pathlib.Path(__file__).with_name("pins.json"))
    parser.add_argument("--receipt", type=pathlib.Path, required=True)
    arguments = parser.parse_args()
    try:
        receipt = replay(arguments)
    except (ValueError, OSError, KeyError, subprocess.TimeoutExpired, json.JSONDecodeError) as error:
        receipt = dict(status="inconclusive" if isinstance(error, (OSError, subprocess.TimeoutExpired)) or
                       str(error).startswith("inconclusive:") else "different", error=str(error))
        arguments.receipt.parent.mkdir(parents=True, exist_ok=True)
        arguments.receipt.write_text(json.dumps(receipt, indent=2) + "\n")
        print(json.dumps(receipt), file=sys.stderr)
        return 1
    arguments.receipt.parent.mkdir(parents=True, exist_ok=True)
    arguments.receipt.write_text(json.dumps(receipt, indent=2) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
