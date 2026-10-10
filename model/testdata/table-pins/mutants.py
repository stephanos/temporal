"""Seed named mutations into actual independently captured interpreter streams."""

import argparse
import copy
import gzip
import json
import pathlib

import replay


def mutate(records, name):
    if name == "binding-order":
        header = records[0]
        header["declaration"]["steps"][0:2] = reversed(header["declaration"]["steps"][0:2])
        bindings = [item for item in records if item["kind"] == "binding"]
        bindings[0]["declaration"], bindings[1]["declaration"] = bindings[1]["declaration"], bindings[0]["declaration"]
        return
    if name == "missing-pair":
        index = next(i for i, item in enumerate(records) if item["kind"] == "disabled")
        del records[index]
        return
    if name in ("result-order", "fact-order"):
        row = next(item for item in records if item["kind"] == "row" and
                   (len(item["results"]) > 1 if name == "result-order" else
                    any(len(set(result["facts"])) > 1 for result in item["results"])))
    else:
        row = next(item for item in records if item["kind"] == "row")
    if name == "row":
        row["source"] = next(state for state in records[0]["states"] if state != row["source"])
        row["key"] = row["source"] + "-" + row["action"]
    elif name == "result":
        result = row["results"][0]
        result["state"] = next(state for state in records[0]["states"] if state != result["state"])
    elif name == "result-order":
        row["results"].reverse()
        row["native_steps"].reverse()
    elif name == "fact-order":
        index = next(i for i, result in enumerate(row["results"]) if len(set(result["facts"])) > 1)
        row["results"][index]["facts"].reverse()
        row["native_steps"][index]["Fields"][2]["Items"].reverse()
    elif name == "explanation":
        row["results"][0]["because"] += " seeded-difference"
        row["native_steps"][0]["Fields"][3]["Text"] += " seeded-difference"
    else:
        raise ValueError("unknown mutant " + name)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--native", type=pathlib.Path, required=True)
    parser.add_argument("--out", type=pathlib.Path, required=True)
    parser.add_argument("--pins", type=pathlib.Path, default=pathlib.Path(__file__).with_name("pins.json"))
    args = parser.parse_args()
    pins = json.loads(args.pins.read_text())
    artifact = next(a for a in pins["artifacts"] if a["artifact"] == "activity-standalone-race.json")
    subject = next(s for s in artifact["subjects"] if s["owner"] == "activityProduct")
    path = args.native / "activity-standalone-race/machine-activityProduct.jsonl.gz"
    replay.verify_stream(path, subject)
    with gzip.open(path, "rb") as stream:
        original_lines = list(stream)
    records = [json.loads(line) for line in original_lines]
    replay.require(not args.out.exists(), "refusing to overwrite mutant observations")
    args.out.mkdir(parents=True)
    observations = []
    for name in ("row", "result", "result-order", "fact-order", "binding-order", "explanation", "missing-pair"):
        changed = copy.deepcopy(records)
        mutate(changed, name)
        replay.require(changed != records, "seed did not alter behavior: " + name)
        mutant = args.out / (name + ".jsonl.gz")
        with gzip.open(mutant, "wb") as stream:
            for index, item in enumerate(changed):
                line = original_lines[index] if index < len(records) and item == records[index] else (
                    json.dumps(item, separators=(",", ":"), sort_keys=True) + "\n").encode()
                stream.write(line)
        try:
            replay.verify_stream(mutant, subject)
        except ValueError as error:
            observations.append(dict(mutant=name, status="rejected", error=str(error)))
        else:
            raise ValueError("mutant escaped: " + name)
    expected = {(s["kind"], s["owner"]) for s in artifact["subjects"]}
    for name, actual in (("missing-machine", expected - {("machine", "heldDispatch")}),
                         ("extra-machine", expected | {("machine", "seededExtraMachine")})):
        try:
            replay.require_universe(expected, actual, artifact["artifact"])
        except ValueError as error:
            observations.append(dict(mutant=name, status="rejected", error=str(error)))
        else:
            raise ValueError("mutant escaped: " + name)
    receipt = dict(status="complete", baseline_stream=str(path), pins_sha256=replay.digest(args.pins),
                   observations=observations)
    (args.out / "receipt.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print(json.dumps(receipt))


if __name__ == "__main__":
    main()
