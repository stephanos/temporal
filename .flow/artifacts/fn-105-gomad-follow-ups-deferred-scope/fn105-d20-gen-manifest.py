#!/usr/bin/env python3
"""Writes a scratch qualification-set manifest for TestWorkflowTaskTestSuite with the
TestWorkflowTaskHeartbeatingWithEmptyResult skip lifted. The suite entry is copied from the
tracked generated manifest; only the skip, the tick policy, and the tracing fields change.
usage: fn105-d20-gen-manifest.py TESTS_JSON OUT <strict|forward> <traced|untraced>
"""
import json, sys
src, out, tick, trace = sys.argv[1:5]
m = json.load(open(src))
suite = next(s for s in m["suites"] if s["name"] == "TestWorkflowTaskTestSuite")
assert suite.pop("skip") == ["TestWorkflowTaskHeartbeatingWithEmptyResult"]
assert "clock_tick" not in suite
if tick == "forward":
    suite["clock_tick"] = "forward"
if trace == "traced":
    suite.update(choice_bytes=67108864, replay_successes=True,
                 success_artifact_limit=1, success_bytes_limit=1073741824)
m["description"] = "fn-105 D20 scratch manifest: TestWorkflowTaskTestSuite, heartbeat skip lifted, %s tick, %s" % (tick, trace)
m["suites"] = [suite]
json.dump(m, open(out, "w"), indent=1)
