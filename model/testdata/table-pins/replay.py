"""Replay the inherited interpreter dumper against immutable whole-table pins."""

import argparse
import fcntl
import gzip
import hashlib
import json
import os
import pathlib
import re
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


# --- Equivalence under a rename map (fn-160.5): the same dumper over the changed IR, compared with
# the streams a baseline captured, each baseline line renamed and both without source positions.
# The rename map is the model gate's (model/check/Equivalence.scala): `old new` renames a whole
# word, `key -` removes a table (here the subject named by the key's last segment).

WORD = "A-Za-z0-9_"


class Renames:
    def __init__(self, text):
        self.renames, self.removals = {}, []
        for number, line in enumerate(text.splitlines(), 1):
            line = line.split("#", 1)[0].strip()
            if not line:
                continue
            parts = line.split()
            require(len(parts) == 2, f"rename map line {number} is `old new` or `old -`, got `{line}`")
            if parts[1] == "-":
                self.removals.append(parts[0])
            else:
                self.renames[parts[0]] = parts[1]
        self.used = set()
        alternatives = sorted(self.renames, key=len, reverse=True)
        self.pattern = re.compile(f"(?<![{WORD}])(?:" + "|".join(map(re.escape, alternatives)) +
                                  f")(?![{WORD}])") if alternatives else None

    def __call__(self, text):
        if self.pattern is None:
            return text

        def replace(match):
            self.used.add(match.group(0))
            return self.renames[match.group(0)]
        return self.pattern.sub(replace, text)

    def removed_owners(self):
        return {r.rsplit(".", 1)[-1] for r in self.removals if "/" not in r}


def without_positions(value):
    if isinstance(value, dict):
        return {k: without_positions(v) for k, v in value.items() if k not in ("position", "Position")}
    if isinstance(value, list):
        return [without_positions(v) for v in value]
    return value


def normalized_lines(path, rename):
    with gzip.open(path, "rt", encoding="utf-8") as source:
        for line in source:
            line = json.dumps(without_positions(json.loads(line)), separators=(",", ":"), sort_keys=True)
            yield rename(line)


def streams(folder):
    found = {}
    for path in sorted(folder.glob("*/*.jsonl.gz")):
        kind, owner = path.name[:-len(".jsonl.gz")].split("-", 1)
        found.setdefault((kind, owner), []).append(path)
    return found


def compare_streams(baseline, current, renames, shown=12):
    """The differences between two folders of dumps, every baseline line renamed."""
    identity = Renames("")
    was, now = streams(baseline), streams(current)
    require(was, f"{baseline} holds no dumps")
    removed = renames.removed_owners()
    differences, matched, used_removals = [], set(), set()
    for (kind, owner), paths in sorted(was.items()):
        if owner in removed:
            used_removals.add(owner)
            continue
        target = (kind, renames(owner))
        if target not in now:
            differences.append(f"{kind} {owner} (as {target[1]}) is gone")
            continue
        matched.add(target)
        current_lines = [list(normalized_lines(p, identity)) for p in now[target]]
        for variant in current_lines[1:]:
            if variant != current_lines[0]:
                differences.append(f"{kind} {target[1]} differs between the changed IR files that hold it")
        for path in paths:
            old = list(normalized_lines(path, renames))
            new = current_lines[0]
            differing = [i for i, (a, b) in enumerate(zip(old, new)) if a != b]
            for i in differing[:shown]:
                differences.append(f"{kind} {target[1]} ({path.parent.name}) line {i + 1} differs\n"
                                   f"    baseline: {old[i][:400]}\n    now:      {new[i][:400]}")
            if len(differing) > shown:
                differences.append(f"{kind} {target[1]} ({path.parent.name}): {len(differing) - shown} more lines differ")
            if len(old) != len(new):
                differences.append(f"{kind} {target[1]} ({path.parent.name}) has {len(new)} lines, the baseline {len(old)}")
    for key in sorted(set(now) - matched):
        differences.append(f"{key[0]} {key[1]} is new")
    return dict(differences=differences, used=sorted(renames.used),
                used_removals=sorted(r for r in renames.removals if r.rsplit(".", 1)[-1] in used_removals))


def dump_all(ir, dumper, out):
    """Runs the pinned dumper over every activity/Nexus IR file into a fresh folder."""
    require(dumper.is_file(), f"inconclusive: inherited dumper missing: {dumper}")
    pins = json.loads(pathlib.Path(__file__).with_name("pins.json").read_text())
    require(digest(dumper) == pins["seam"]["binary_sha256"], "inconclusive: inherited dumper identity changed")
    require(not out.exists(), "refusing to overwrite a dump; choose a fresh scratch directory")
    out.mkdir(parents=True)
    environment = {**os.environ, "BASH_ENV": "/dev/null"}
    for ir_path in sorted(ir.glob("*.json")):
        if not ir_path.name.startswith(("activity-", "nexus-")) or ir_path.name.endswith((".lint.json", ".waivers.json")):
            continue
        command = [str(dumper.resolve()), str(ir_path.resolve()), str((out / ir_path.stem).resolve())]
        with (out / (ir_path.stem + ".log")).open("wb") as log:
            process = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, env=environment, timeout=600)
        require(process.returncode == 0, f"inconclusive: native dump exited {process.returncode}: {command}")


def equivalence(arguments):
    renames = Renames(arguments.map.read_text())
    with open("/tmp/umpire-heavy-gates.lock", "a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        dump_all(arguments.ir, arguments.dumper, arguments.out)
    result = compare_streams(arguments.baseline, arguments.out, renames)
    return dict(status="equivalent" if not result["differences"] else "different", **result)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ir", type=pathlib.Path, required=True)
    parser.add_argument("--dumper", type=pathlib.Path, required=True)
    parser.add_argument("--out", type=pathlib.Path, required=True)
    parser.add_argument("--pins", type=pathlib.Path, default=pathlib.Path(__file__).with_name("pins.json"))
    parser.add_argument("--receipt", type=pathlib.Path, required=True)
    parser.add_argument("--baseline", type=pathlib.Path,
                        help="compare with this captured dump under --map instead of with the pins")
    parser.add_argument("--map", type=pathlib.Path, help="the rename map of an equivalence check")
    arguments = parser.parse_args()
    try:
        if arguments.baseline is not None:
            require(arguments.map is not None, "--baseline needs --map")
            receipt = equivalence(arguments)
            arguments.receipt.parent.mkdir(parents=True, exist_ok=True)
            arguments.receipt.write_text(json.dumps(receipt, indent=2) + "\n")
            return 0 if receipt["status"] == "equivalent" else 1
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
