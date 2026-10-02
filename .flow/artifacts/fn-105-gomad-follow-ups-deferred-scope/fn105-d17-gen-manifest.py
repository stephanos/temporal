#!/usr/bin/env python3
"""Writes a scratch qualification-set manifest for TestNexusOTELSuite.
usage: gen.py OUT [traced|untraced] [two|one] [forward|strict] [skip-csv|-] [repeat] [seeds-csv]
"""
import json, sys
out, trace, clusters, tick, skip, repeat, seeds = (sys.argv[1:] + [None]*7)[:7]
suite = {
 "id": "tests-nexus-otel-suite", "name": "TestNexusOTELSuite", "tier": 3,
 "invariant": "TestNexusOTELSuite passes through the one-box cluster under virtual time",
 "package": "./tests", "test": "TestNexusOTELSuite",
 "build_tags": ["disable_grpc_modules", "gomad", "test_dep"],
 "capability_mode": "closure",
 "read_only_mounts": [{"source": "./schema", "target": "/go.temporal.io/server/schema"}],
 "choice_bytes": 67108864 if trace == "traced" else 0,
 "replay_successes": trace == "traced",
 "success_artifact_limit": 1 if trace == "traced" else 0,
 "success_bytes_limit": 1073741824 if trace == "traced" else 0,
 "overall_timeout": "20m", "test_parallel": 8,
 "expectation": {"classification": "qualified"},
}
if clusters == "two":
    suite["environment"] = ["TEMPORAL_TEST_DEDICATED_CLUSTERS=2"]
if tick == "forward":
    suite["clock_tick"] = "forward"
if skip and skip != "-":
    suite["skip"] = sorted(skip.split(","))
m = {"schema": "gomad3.qualification-set/v3", "name": "temporal-tests",
 "description": "fn-105 D17 scratch manifest: TestNexusOTELSuite",
 "module": "go.temporal.io/server",
 "seeds": [int(s) for s in (seeds or "11,17").split(",")],
 "repeat": int(repeat or 2), "run_timeout": "2m", "overall_timeout": "5m",
 "terminate_grace": "2s", "output_bytes": 8388608, "world_transition_bytes": 67108864,
 "suites": [suite]}
json.dump(m, open(out, "w"), indent=1)
