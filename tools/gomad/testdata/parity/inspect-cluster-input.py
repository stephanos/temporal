#!/usr/bin/env python3
"""Produce a fresh helper transport cluster input consumer fixture for inspect.

This exercises a retained helper executable, not the linked Rust host cluster
producer, and does not qualify either implementation.

The consumer payload encodings are canonical UTF-8 JSON with sorted keys,
compact separators and no trailing newline. The request fixture schema
gomad.inspect-helper-cluster-request-fixture/v1 omits only $.module_path;
the result fixture schema gomad.inspect-helper-cluster-result-fixture/v1
omits only $.metrics. All permitted metric names are checked explicitly.
Every other request/result field is retained, including module_sha256 and
every broker/instance/model setting. Raw request/result transport lines,
including their LF, are retained separately and hashed in provenance.json.
These producer encodings happen before payload hashes are computed.
"""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import threading


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"),
                      ensure_ascii=False).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def keys(value, expected, path):
    if not isinstance(value, dict) or set(value) != set(expected.split()):
        raise ValueError(f"unexpected fields at {path}: {value!r}")


def check_request(request):
    keys(request, "broker instances module_path module_sha256 schema type", "request")
    keys(request["broker"], "limits seed", "request.broker")
    keys(request["broker"]["limits"],
         "max_events max_payload_bytes max_queued_events max_requests max_string_bytes max_transitions",
         "request.broker.limits")
    for instance in request["instances"]:
        keys(instance, "fuel id local_model memory_bytes output_bytes", "request.instances[]")
        model = instance["local_model"]
        keys(model, "arguments diagnostics entropy_key environment epoch_nanos limit limits mode now "
             "read_step_nanos seed working_directory_sha256", "request.instances[].local_model")
        for name in ("arguments", "environment"):
            keys(model[name], "bytes chunks count", "request.instances[].local_model." + name)
        keys(model["limits"], "calls pending_events transcript_bytes", "request.instances[].local_model.limits")


def check_result(result):
    keys(result, "instances message metrics termination type virtual_elapsed_nanos", "result")
    for instance in result["instances"]:
        keys(instance, "id termination", "result.instances[]")
    keys(result["metrics"], "prepare_nanos compile_nanos cache_load_nanos cache_write_nanos cache_hit "
         "instantiate_nanos execute_nanos process_user_nanos process_system_nanos peak_rss_bytes "
         "resource_usage_available", "result.metrics")


def snapshot_summary(raw):
    if raw[:12] != b"GMWSNAP1\x00\x00\x00\x01":
        raise ValueError("unsupported expanded World snapshot")
    seed = struct.unpack_from(">Q", raw, 12)[0]
    now = struct.unpack_from(">q", raw, 64)[0]
    next_transition = struct.unpack_from(">Q", raw, 88)[0]
    # The generated snapshot wire stores the next transition ordinal, starting at 1.
    return seed, now, next_transition - 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--helper", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--corpus", type=Path, required=True)
    args = parser.parse_args()
    helper = args.helper.resolve(strict=True)
    corpus = args.corpus.resolve(strict=True)
    output = args.output.resolve()
    output.mkdir(mode=0o700)
    case_file = corpus / "cluster_broker_idles_on_shared_clock.json"
    case = json.loads(case_file.read_bytes())
    guest = output / "guest.wat"
    guest.write_bytes(case["guest"]["wat"].encode())
    request = case["request"]
    request["module_path"] = str(guest)
    request["module_sha256"] = sha(guest.read_bytes())
    check_request(request)
    plan = canonical({key: value for key, value in request.items() if key != "module_path"})
    (output / "simulation-plan.json").write_bytes(plan)
    frames = []
    timed_out = threading.Event()
    failure = None
    status = None
    with (output / "helper.stdout").open("wb") as stdout, \
            (output / "helper.stderr").open("wb") as stderr, \
            (output / "helper.stdin").open("wb") as stdin:
        child = subprocess.Popen([str(helper)], stdin=subprocess.PIPE,
                                 stdout=subprocess.PIPE, stderr=stderr)

        def timeout():
            timed_out.set()
            child.kill()

        timer = threading.Timer(30, timeout)
        timer.start()

        def send(frame):
            # Cluster routing requires instance to be the first field.
            if "instance" in frame:
                frame = {"instance": frame["instance"],
                         **{key: value for key, value in frame.items() if key != "instance"}}
            raw = json.dumps(frame, separators=(",", ":")).encode() + b"\n"
            stdin.write(raw)
            stdin.flush()
            child.stdin.write(raw)
            child.stdin.flush()
            return raw

        try:
            (output / "raw-request.json").write_bytes(send(request))
            for frame in case["inputs"]:
                send(frame)
            authorized = False
            for raw in child.stdout:
                stdout.write(raw)
                stdout.flush()
                frame = json.loads(raw)
                frames.append(frame)
                if frame["type"] == "cluster_result":
                    (output / "raw-result.json").write_bytes(raw)
                if frame["type"] == "started":
                    if authorized:
                        raise ValueError("duplicate helper start")
                    send({"type": "authorize", "module_sha256": frame["module_sha256"],
                          "configuration": frame["engine"]["configuration"]})
                    authorized = True
                elif frame["type"] in ("call", "sync"):
                    raise ValueError("unexpected callback in automatic cluster fixture")
            status = child.wait(timeout=1)
            if timed_out.is_set():
                raise ValueError("helper exceeded 30 second timeout")
            if status != 0:
                raise ValueError(f"helper exited with status {status}")
        except Exception as error:
            failure = str(error)
            child.kill()
            status = child.wait()
            stdout.write(child.stdout.read())
        finally:
            timer.cancel()
            timer.join()
            child.stdin.close()
            child.stdout.close()
            (output / "commands.json").write_bytes(canonical({
                "label": "helper transport cluster input consumer fixture",
                "qualification": False, "linked_host_cluster_producer": False,
                "command": [str(helper)], "helper_sha256": sha(helper.read_bytes()),
                "corpus": str(case_file), "corpus_sha256": sha(case_file.read_bytes()),
                "timeout_seconds": 30, "timed_out": timed_out.is_set(),
                "status": status, "failure": failure,
                "stdin": "helper.stdin", "stdout": "helper.stdout", "stderr": "helper.stderr",
            }))
    if failure:
        raise ValueError(failure)
    terminals = [frame for frame in frames if frame["type"] == "cluster_result"]
    evidence = [frame for frame in frames if frame["type"] == "evidence"
                and frame.get("stream") == "world"]
    results = [frame for frame in frames if frame["type"] == "result"]
    if (len(terminals) != 1 or terminals[0]["termination"] != "exit"
            or len(evidence) != 1 or len(results) != len(request["instances"])
            or any(frame["termination"] != "exit" or frame["exit_code"] != 0
                   for frame in results)):
        raise ValueError("helper refused or returned incomplete cluster evidence")
    world = base64.b64decode(evidence[0]["data_base64"], validate=True)
    seed, now, transitions = snapshot_summary(world)
    if seed != request["broker"]["seed"]:
        raise ValueError("World seed differs from request")
    check_result(terminals[0])
    record = canonical({key: value for key, value in terminals[0].items() if key != "metrics"})
    (output / "world").write_bytes(world)
    (output / "simulation-record.json").write_bytes(record)
    (output / "provenance.json").write_bytes(canonical({
        "schema": "gomad.inspect-helper-cluster-provenance/v1",
        "raw_request": {"file": "raw-request.json", "sha256": sha((output / "raw-request.json").read_bytes())},
        "raw_result": {"file": "raw-result.json", "sha256": sha((output / "raw-result.json").read_bytes())},
        "omitted_request_paths": ["$.module_path"], "omitted_result_paths": ["$.metrics"],
        "omitted_result_metric_names": sorted(terminals[0]["metrics"]),
    }))
    payloads = {"evidence/world": world, "evidence/simulation-plan.json": plan,
                "evidence/simulation-record.json": record}

    def payload(name, schema):
        raw = payloads[name]
        return {"schema": schema, "file": name, "sha256": sha(raw),
                "bytes": len(raw), "limit": len(raw)}

    details = {
        "schema": "gomad.inspection-evidence/v1", "choices": None,
        "world": {"profile": "gomad.world-expanded-snapshot/v1",
                  "implementation_sha256": sha(helper.read_bytes()), "seed": seed,
                  "now_nanos": now, "transitions": transitions,
                  "terminal_kind": terminals[0]["termination"],
                  "terminal_detail": terminals[0]["message"],
                  "history": payload("evidence/world", "gomad.world-expanded-snapshot/v1")},
        "simulation": {"profile": request["schema"], "controller_sha256": sha(plan),
                       "execution_sha256": sha(plan), "candidate_sha256": sha(plan),
                       "outcome_sha256": sha(record), "failure_sha256": None,
                       "plan": payload("evidence/simulation-plan.json", "gomad.inspect-helper-cluster-request-fixture/v1"),
                       "record": payload("evidence/simulation-record.json", "gomad.inspect-helper-cluster-result-fixture/v1")},
        "minimization": None,
    }
    (output / "input.json").write_bytes(canonical({
        "seed": seed, "details": details,
        "payloads": {name: list(raw) for name, raw in payloads.items()},
    }))
    print(json.dumps({"input": str(output / "input.json"), "world_bytes": len(world),
                      "now_nanos": now, "transitions": transitions}))


if __name__ == "__main__":
    main()
