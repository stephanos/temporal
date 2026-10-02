#!/usr/bin/env python3
"""Behavioral projection of a core qualification run.

Reads every gomad3.qualification/v1 report below ARTIFACTS/qualifications/v1 and
writes one JSON object keyed by workload (target source plus argv). The
projection holds what must not move between two toolchain builds; identities
derived from the toolchain build key or the choice implementation digest are
listed separately under "identity" so a comparison can show they did move.

usage: project.py ARTIFACTS OUTPUT.json
"""
import hashlib
import json
import os
import struct
import sys

RECORD = 96


def sha(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def choice_content(path):
    with open(path, "rb") as handle:
        payload = handle.read()
    if len(payload) % RECORD:
        raise SystemExit("choice payload is not a whole number of records: " + path)
    structural = []
    sites = []
    names = {}

    def canonical(identity):
        # Identities are named by first appearance, which keeps which logical
        # goroutine or select case was chosen and drops the hash value.
        if identity == bytes(32):
            return None
        return names.setdefault(identity, len(names))

    for offset in range(0, len(payload), RECORD):
        record = payload[offset:offset + RECORD]
        ordinal, kind, flags = struct.unpack(">QBB", record[:10])
        alternatives, selected, data, site = struct.unpack(">IIIQ", record[12:32])
        observation = flags & 2 != 0
        # A decision's selected rank orders alternatives by identity, and an
        # identity hashes the text offset of the go statement or select that
        # made it. Runtime code size moves those offsets, so across toolchain
        # builds an observation's selected case index is comparable, and a
        # decision is compared by which identity it selected from which set.
        structural.append([ordinal, kind, flags, alternatives, data, selected if observation else None, canonical(record[32:64]), canonical(record[64:96])])
        sites.append(site)
    return {
        "records": len(payload) // RECORD,
        "full_sha256": sha(payload),
        "structural_sha256": sha(json.dumps(structural, separators=(",", ":")).encode()),
        "site_offsets_sha256": sha(json.dumps(sites, separators=(",", ":")).encode()),
    }


def stream(value):
    return {"full_sha256": value["full_sha256"], "total_bytes": value["total_bytes"], "truncated": value["truncated"]}


def world(value):
    return {
        "initial": value["initial"]["semantic_digest"],
        "final": value["final"]["semantic_digest"],
        "transitions": value["transitions"]["transcript_digest"],
        "transition_count": value["transitions"]["count"],
        "terminal": value.get("terminal"),
    }


def main():
    artifacts, output = sys.argv[1], sys.argv[2]
    directory = os.path.join(artifacts, "qualifications", "v1")
    result = {}
    for name in sorted(os.listdir(directory)):
        with open(os.path.join(directory, name)) as handle:
            report = json.load(handle)
        evidence = report["evidence"]
        key = evidence["target"]["source"] + " " + " ".join(evidence["target"]["argv"][1:])
        choices = evidence.get("choices") or {}
        executions = []
        for execution in report["executions"]:
            entry = {"evidence_matches_first": execution["evidence_digest"] == report["evidence_digest"]}
            trace = os.path.join(execution.get("artifact_path", ""), "choices.bin")
            if os.path.isfile(trace):
                entry["choice"] = choice_content(trace)
            replay = execution.get("replay") or {}
            entry["replay"] = {key: replay.get(key) for key in ("attempted", "match", "choice_replay_status")}
            executions.append(entry)
        if key in result:
            raise SystemExit("duplicate workload: " + key)
        result[key] = {
            "behavior": {
                "qualified": report["qualified"],
                "deterministic": report["deterministic"],
                "target_success": report["target_success"],
                "seed": report["seed"],
                "repeat": report["repeat"],
                "stdout": stream(evidence["stdout"]),
                "stderr": stream(evidence["stderr"]),
                "io_transcript": {
                    "sha256": evidence.get("io_transcript_sha256"),
                    "records": evidence.get("io_transcript_records"),
                    "complete": evidence.get("io_transcript_complete"),
                },
                "world": world(evidence["world"]),
                "outcome": evidence["outcome"],
                "virtual_time_elapsed_nanos": evidence.get("virtual_time_elapsed_nanos"),
                "semantic_coverage": evidence["semantic_coverage"]["digest"],
                "choices": {
                    name: choices.get(name)
                    for name in ("records", "decisions", "branching_records", "runnable", "select_poll", "select_result", "peak_goroutines", "terminal_state")
                },
                "choice_features": [feature for feature in choices.get("features") or [] if feature["kind"] in ("record_kind", "terminal")],
                "choice_structural": [entry.get("choice", {}).get("structural_sha256") for entry in executions],
                "executions": [{"evidence_matches_first": entry["evidence_matches_first"], "replay": entry["replay"]} for entry in executions],
            },
            "site_dependent": {
                "choice_full": [entry.get("choice", {}).get("full_sha256") for entry in executions],
                "choice_site_offsets": [entry.get("choice", {}).get("site_offsets_sha256") for entry in executions],
                "choice_trace_sha256": choices.get("sha256"),
                # These features name a text offset or a rank among identities.
                "choice_features": [feature for feature in choices.get("features") or [] if feature["kind"] not in ("record_kind", "terminal")],
            },
            "identity": {
                "toolchain_build_key": evidence["toolchain"]["build_key"],
                "choice_implementation_sha256": choices.get("implementation_sha256"),
                "choice_tape_sha256": choices.get("tape_sha256"),
                "runner_build": evidence["runner_build"],
                "target_sha256": evidence["target"]["sha256"],
                "evidence_digest": report["evidence_digest"],
            },
        }
    with open(output, "w") as handle:
        json.dump(result, handle, indent=2, sort_keys=True)
        handle.write("\n")
    print("projected %d workloads into %s" % (len(result), output))


if __name__ == "__main__":
    main()
