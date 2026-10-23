#!/usr/bin/env python3
"""Original shard, workflow and onebox retained stopped-inspection producers."""
import argparse
import base64
import gzip
import hashlib
import json
import os
from pathlib import Path
import sys

sys.dont_write_bytecode = True
import producer
import workloads


def file_hashes(root):
    return {str(path.relative_to(root)): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in root.rglob("*") if path.is_file()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ["binary", "repo", "root", "compiler", "runtime-root", "cache"]:
        parser.add_argument("--"+name, type=Path, required=True)
    parser.add_argument("--owner", choices=[*workloads.INSPECTION, "all"], required=True)
    parser.add_argument("--captured-config", type=Path)
    parser.add_argument("--private-observation-config", type=Path)
    args = parser.parse_args()
    binary, repo, root = args.binary.resolve(), args.repo.resolve(), args.root.resolve()
    private = producer.private_configuration(args.private_observation_config, binary, repo, root)
    root.mkdir(parents=True, mode=0o700, exist_ok=False)
    producer.owned_root(root)
    producer.private_admission(private, root)
    if args.owner == "all":
        receipts = {}
        for owner in workloads.INSPECTION:
            command = [sys.executable, str(Path(__file__).resolve()), "--owner", owner]
            for name in ["binary", "repo", "compiler", "runtime-root", "cache"]:
                command += ["--"+name, str(getattr(args, name.replace("-", "_")).resolve())]
            command += ["--root", str(root/owner)]
            if args.captured_config:
                command += ["--captured-config", str(args.captured_config.resolve())]
            if args.private_observation_config:
                command += ["--private-observation-config", str(args.private_observation_config.resolve())]
            receipts[owner] = producer.run(root, owner, command, 0)
        print(json.dumps(dict(receipts=receipts)))
        return
    selection, dump, probe, snapshot = workloads.INSPECTION[args.owner]
    captured = producer.captured(repo, args.captured_config) if args.owner == "onebox" and args.captured_config else None
    policy = root/"policy.json"
    producer.write(policy, workloads.inspection_policy(args.owner, captured))
    stage = repo/"service/history"/(".gomad-inspection-"+args.owner+"-"+hashlib.sha256(str(root).encode()).hexdigest()[:16])
    producer.write(root/"unchanged-source-inputs.json", workloads.stage_inspection(repo, stage, args.owner))
    installation = root/"installation.json"
    producer.run(root,"install",[str(binary),"install","--compiler",str(args.compiler.resolve()),"--runtime-root",str(args.runtime_root.resolve()),
        "--cache",str(args.cache.resolve()),"--output",str(installation),"--profile","gomad.wasi-cooperative/v1","--config",str(policy),
        "--fuel",str(1<<50 if args.owner=="onebox" else 50_000_000_000),"--memory-bytes",str(4<<30 if args.owner=="onebox" else 1<<30)],0,json_output=False)
    arguments=["-test.run=^"+selection+"$","-test.v"]
    command=[str(binary),"--installation",str(installation),"explore","--json","--seeds","7","--parallel","1",
        "--execution-timeout","30m" if args.owner=="onebox" else "5m","--overall-timeout","45m" if args.owner=="onebox" else "15m",
        "--terminate-grace","1s","--on-failure","first","--failure-budget","1","--output-limit",str(1<<20),"--world-transition-limit",str(1<<20),
        "--artifacts",str(root/"artifacts"),"--env","GOMAD_DEBUG_CANARY=1"]
    if args.owner=="onebox":
        scenario=root/"scenario.json"
        producer.write(scenario,workloads.inspection_scenario(arguments))
        command += ["--scenario",str(scenario)]
        for entry in ["TEMPORAL_ROOT=/workspace","TMPDIR=/tmp","TEMPORAL_TEST_LOG_LEVEL=warn"]:
            command += ["--env",entry]
    else:
        command += ["--choices","--choice-bytes",str(8<<20)]
    for tag in workloads.TAGS+["gomad"]:
        command += ["--build-tag",tag]
    command += ["--working-dir",str(repo),"go-test","./"+str(stage.relative_to(repo)),"--",*arguments]
    response=producer.run(root,"explore",command,1,**({'private_observations': private} if private is not None else {}))
    if response["attempted"]!=1 or response["failures"]!=1 or response["retained_evidence"]!=1:
        raise ValueError("original inspection canary did not retain exactly one actual failure")
    artifacts=[]
    for manifest in (root/"artifacts").rglob("manifest.json"):
        value=json.loads(manifest.read_bytes())
        if value.get("record",{}).get("outcome",{}).get("class")=="target_failure":
            artifacts.append(manifest.parent)
    if len(artifacts)!=1:
        raise ValueError("original inspection failure artifact unavailable or ambiguous")
    artifact=artifacts[0]
    before=file_hashes(artifact)
    if args.owner=="onebox":
        output=(artifact/"evidence/cluster/history/1/stdout").read_bytes()
        for marker in [b"workflow completed [step 0 attempt 2 step 1 attempt 2]",b"Temporal inspection canary after workflow timers and activity retries completed"]:
            if marker not in output:
                raise ValueError("onebox canary preceded completed timers and retries")
    detached=stage.with_name(stage.name+"-unavailable")
    stage.rename(detached)
    environment=dict(os.environ,PATH="",GOMAD_WASM_INSTALLATION=str(root/"installation-unavailable.json"),GOMAD_STOCK_GO=str(root/"compiler-unavailable"),
                     GOMAD_WASM_RUNTIME_ROOT=str(root/"runtime-unavailable"))
    producer.run(root,"detached-replay",[str(binary),"replay",str(artifact)],1,json_output=False,environment=environment)
    replay_output=(root/"detached-replay.stdout").read_text()
    if "reproduced=true" not in replay_output or "choice-replay=exact" not in replay_output:
        raise ValueError("detached replay did not reproduce the exact retained failure")
    command=[str(binary),"replay","--stop=failure","--dump=stacks","--dump=heap","--dump="+dump,"--probe="+probe,
             "--debug-bytes",str(8<<20),"--dump-bytes",str(4<<20),"--debug-output",str(root/"inspection")]
    if args.owner=="onebox":
        command += ["--instance=history","--incarnation=1","--dump=temporal.shards"]
    report=producer.run(root,"stopped-inspection",[*command,str(artifact)],0,environment=environment)
    if report["status"]!="inspected_prefix" or not report["prefix_verified"] or not report["witness_match"]:
        raise ValueError("original inspection failed its authenticated prefix/witness")
    debug=json.loads(Path(report["output_path"]).read_bytes())
    if not debug["stacks"] or not gzip.decompress((root/"inspection/heap.pprof").read_bytes()) or not debug["probes"]:
        raise ValueError("original inspection omitted useful stacks/heap/probes")
    wanted={dump:snapshot}
    if args.owner=="onebox":
        wanted["temporal.shards"]="snapshot="
        if not {"temporal.shard.snapshot","temporal.mutable_state.snapshot"} <= {item["name"] for item in debug["probes"]}:
            raise ValueError("onebox inspection omitted original probe owners")
    for name,marker in wanted.items():
        rows=[item for item in debug["dumps"] if item["name"]==name]
        if len(rows)!=1 or rows[0]["status"]!="ok":
            raise ValueError("original registered snapshot dump unavailable")
        body=base64.b64decode(rows[0]["data_base64"],validate=True)
        if marker.encode() not in body or (name==dump and b"values may precede stop" not in body):
            raise ValueError("original lock-held/completed snapshot assertion failed")
    if before!=file_hashes(artifact):
        raise ValueError("stopped inspection modified original retained artifact")
    producer.write(root/"inspection-receipt.json",dict(schema="gomad.rust-temporal-inspection/v1",owner=args.owner,artifact=str(artifact),source_detached=str(detached),
        report=report,artifact_sha256=before,inspection_sha256=file_hashes(root/"inspection"),subgates=[dict(id="TestTemporalInspectionRetainedFailure/"+args.owner,status="PASS",executed=True,exit=0)]))
    print(json.dumps(dict(receipt=str(root/"inspection-receipt.json"))))


if __name__=="__main__":
    main()
