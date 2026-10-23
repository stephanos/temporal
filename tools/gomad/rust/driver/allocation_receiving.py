#!/usr/bin/env python3
"""Derive the fixed allocation harness predicates from raw numeric observations."""

import argparse
import hashlib
import json
import os
import stat
from pathlib import Path

MARKER = "GOMAD_ALLOCATION_OBSERVATION "
METRICS = (
    "allocated_bytes", "allocation_calls", "live_requested_bytes_before",
    "live_requested_bytes_after", "peak_live_requested_bytes",
)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def derive(rows, target, artifact_root=None):
    expected = {
        ("allocator-control", None),
        *((name, None) for name in (
            ("bounded-file-read", "opened-payload", "retained-decode")
            if target == "gomad" else ("volume-metadata", "namespace-file-length")
        )),
    }
    stages = {
        "binary-world": ["model", "encode_restore_canonical", "decode_restore_canonical", "reencode_restore_canonical"],
        "compressed-world": ["model", "terminal_transfer", "compressed_encode", "compressed_streaming_semantic_decode"],
    }
    if target == "wasmhost":
        expected.update((f"{kind}/{stage}", count)
                        for kind, counts in (("binary-world", (20000,)), ("compressed-world", (20000, 200000)))
                        for count in counts for stage in [*stages[kind], "round-trip"])
    by_key = {}
    for row in rows:
        require(row.get("schema") == "gomad.native-allocation-observation/v1", "unexpected allocation schema")
        require(row.get("allocation_scope") == "selected-test-thread" and row.get("live_scope") == "whole-process", "foreign allocator measurement scope")
        key = (row["case"], row["inputs"].get("requests"))
        require(key in expected and key not in by_key, f"unexpected/duplicate allocation row: {key}")
        facts = row["allocator"]
        require(set(facts) == set(METRICS), f"allocator metric shape: {key}")
        require(all(type(facts[name]) is int and 0 <= facts[name] < 2**64 for name in METRICS), "invalid allocator integer")
        require(facts["peak_live_requested_bytes"] >= max(facts["live_requested_bytes_before"], facts["live_requested_bytes_after"]), "invalid peak live metric")
        by_key[key] = row
    require(set(by_key) == expected, f"missing allocation rows: {sorted(expected - set(by_key), key=str)}")
    control = by_key[("allocator-control", None)]
    facts = control["allocator"]
    require(control["inputs"] == {"bytes": 8192} and control["observed"] == {"length": 8192, "last": 165}, "allocator control changed")
    require(facts["allocated_bytes"] == 8192 and facts["allocation_calls"] == 1, "allocator not measuring actual calls/bytes")
    require(facts["live_requested_bytes_after"] - facts["live_requested_bytes_before"] == 8192, "allocator live control changed")
    derived = []
    if target == "gomad":
        cases = [
            ("bounded-file-read", {"file_bytes": 1 << 20, "limit_bytes": 512 << 20}, 2*(1 << 20)+(64 << 10), 1 << 20, b"x"),
            ("opened-payload", {"payload_bytes": 8 << 20, "limit_bytes": 8 << 20}, 9 << 20, 8 << 20, b"\xa5"),
            ("retained-decode", {"payload_bytes": 8 << 20}, 56 << 20, 8 << 20, b"\xa5"),
        ]
        for case, inputs, ceiling, length, byte in cases:
            row = by_key[(case, None)]
            observed = row["observed"]
            require(row["inputs"] == inputs and observed["length"] == length, f"fixed workload changed: {case}")
            require(observed["sha256"] == "sha256:"+hashlib.sha256(byte*length).hexdigest(), f"read changed retained bytes: {case}")
            require(row["allocator"]["allocated_bytes"] <= ceiling, f"original byte ceiling exceeded: {case}")
            if case == "retained-decode":
                require(observed["schema"] == "gomad.cluster-replay-inputs/v1" and observed["descriptor_sha256"] == observed["sha256"], "retained descriptor binding changed")
            derived.append({"case": case, "unit": "allocated_bytes", "actual": row["allocator"]["allocated_bytes"], "original_ceiling": ceiling})
    else:
        require(artifact_root is not None, "World receiving requires retained raw wire artifacts")
        from core_world import decode_measured_world, expand
        row = by_key[("volume-metadata", None)]
        require(row["inputs"] == {"file_bytes": 1 << 19, "reads": 100}, "metadata workload changed")
        require(row["observed"]["sizes"] == [1 << 19]*100, "metadata no longer returns file sizes")
        require(row["allocator"]["allocated_bytes"] <= 1 << 20, "metadata copied file payload")
        row = by_key[("namespace-file-length", None)]
        require(row["inputs"] == {"declared_bytes": 3, "limit_bytes": 2, "runs": 100}, "namespace workload changed")
        require(row["observed"]["rejections"] == 100 and row["allocator"]["allocation_calls"] <= 400, "native namespace oversized blob refusal count exceeded")
        derived.extend([
            {"case": "volume-metadata", "unit": "allocated_bytes", "actual": by_key[("volume-metadata", None)]["allocator"]["allocated_bytes"], "original_ceiling": 1 << 20},
            {"case": "namespace-file-length", "unit": "allocation_calls_per_run", "actual_numerator": row["allocator"]["allocation_calls"], "actual_denominator": 100, "original_ceiling": 4},
        ])
        for kind, counts in (("binary-world", (20000,)), ("compressed-world", (20000, 200000))):
            for count in counts:
                cumulative = 0
                for stage in stages[kind]:
                    row = by_key[(f"{kind}/{stage}", count)]
                    require(row["inputs"] == {"requests": count, "seed": 7, "payload_bytes_per_request": 32}, "World workload changed")
                    cumulative += row["allocator"]["allocated_bytes"]
                    observed = row["observed"]
                    require(observed["cumulative_allocated_bytes"] == cumulative, "World cumulative bytes inconsistent")
                    rss, peak = observed["process_rss_bytes"], observed["process_peak_rss_bytes"]
                    require(type(rss) is int and type(peak) is int and 0 < rss <= peak, "invalid actual process RSS")
                    if kind == "compressed-world":
                        require(peak <= 2 << 30, "original compressed World 2GiB investigation budget exceeded")
                terminal = by_key[(f"{kind}/round-trip", count)]["observed"]
                require(terminal["requests"] == count and terminal["events"] == count and terminal["transitions"] == 3*count, "World terminal cardinality changed")
                require(terminal["expanded_bytes"] > 0 and terminal["expanded_sha256"] == terminal["restored_sha256"], "World roundtrip digest changed")
                expected_digest = terminal["expanded_sha256"]
                raw = frozen_wire(artifact_root, terminal["expanded_file"], f"{kind}-{count}.expanded.wire")
                require(hashlib.sha256(raw).hexdigest() == expected_digest and len(raw) == terminal["expanded_bytes"], "World expanded artifact identity differs")
                decoded = decode_measured_world(raw, count)
                require(decoded["request_count"] == count and decoded["event_count"] == count and decoded["transition_count"] == 3*count, "World wire cardinality differs")
                require(decoded["transcript_digest"] == terminal["transcript_digest"] and decoded["state_digest"] == terminal["state_digest"], "World wire terminal digests differ")
                del raw
                restored = frozen_wire(artifact_root, terminal["restored_file"], f"{kind}-{count}.restored.wire")
                require(hashlib.sha256(restored).hexdigest() == expected_digest and len(restored) == terminal["expanded_bytes"], "World actual restored artifact differs")
                del restored
                if kind == "compressed-world":
                    envelope = frozen_wire(artifact_root, terminal["envelope_file"], f"{kind}-{count}.compressed.wire", maximum=(64 << 20)+64)
                    require(hashlib.sha256(envelope).hexdigest() == terminal["envelope_sha256"], "World actual envelope artifact differs")
                    independently_expanded = expand(envelope)
                    require(hashlib.sha256(independently_expanded).hexdigest() == expected_digest, "World independent envelope expansion differs")
                    del envelope, independently_expanded
                else:
                    require(terminal["envelope_file"] is None and terminal["envelope_sha256"] is None, "binary World has a foreign envelope")
                derived.append({"case": kind, "requests": count, "unit": "allocated_bytes", "cumulative": cumulative, "allocation_ceiling": None, "process_peak_ceiling": (2 << 30) if kind == "compressed-world" else None})
    return {"schema": "gomad.native-allocation-derived/v1", "target": target, "observation_rows": len(rows), "predicates": derived,
            "claim": "numeric receiving predicates only; original root equivalence and qualification remain separate"}


def frozen_wire(root, declared, expected, maximum=384 << 20):
    require(declared == expected, "foreign World artifact path")
    path = Path(root) / expected
    with os.fdopen(os.open(path, os.O_RDONLY | os.O_NOFOLLOW), "rb") as source:
        before = os.fstat(source.fileno())
        require(stat.S_ISREG(before.st_mode) and 0 < before.st_size <= maximum, "World artifact is not a bounded regular file")
        raw = source.read(maximum + 1)
        after = os.fstat(source.fileno())
    identity = lambda item: (item.st_dev, item.st_ino, item.st_size, item.st_mtime_ns, item.st_ctime_ns)
    require(identity(before) == identity(after) and len(raw) == before.st_size, "World artifact changed while reading")
    return raw


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=("gomad", "wasmhost"), required=True)
    parser.add_argument("log", type=Path)
    parser.add_argument("--artifact-root", type=Path, help="fixed wire outputs (defaults to log directory)")
    arguments = parser.parse_args()
    rows = [json.loads(line.split(MARKER, 1)[1]) for line in arguments.log.read_text().splitlines() if MARKER in line]
    print(json.dumps(derive(rows, arguments.target, arguments.artifact_root or arguments.log.parent), indent=2))


if __name__ == "__main__":
    main()
