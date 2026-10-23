#!/usr/bin/env python3
"""Execute the original fn222 smoke, detached inspection and frontend receiving gate."""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import sys
sys.dont_write_bytecode=True
import producer
import workloads


def selected_pass(raw, row):
    selected = "TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds" if row == "smoke" else "TestFrontendSystemInfo"
    marker = re.compile(rb'^[ \t]*--- PASS: ' + re.escape(selected.encode()) + rb' \([0-9]+(?:\.[0-9]+)?s\)[ \t]*$', re.M)
    if len(marker.findall(raw)) != 1 or b"--- SKIP:" in raw or b"no tests to run" in raw:
        raise ValueError("original exact selected workflow/frontend test did not pass exactly once")


def gate_environment(row, measured=False):
    environment = dict(os.environ)
    environment.pop('GOMAD_WASM_QUALIFY_WORKLOADS', None)
    if row == 'smoke' and measured:
        environment['GOMAD_WASM_QUALIFY_WORKLOADS'] = '1'
    return environment


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ["binary","repo","root","compiler","runtime-root","cache","captured-config"]:
        parser.add_argument("--"+name,required=True,type=Path)
    parser.add_argument("--source-manifest",required=True,type=Path)
    parser.add_argument("--prepare-only",action="store_true",help="retain only the first smoke preparation; run no measured cell or inspection")
    args=parser.parse_args()
    root=args.root.resolve()
    root.mkdir(mode=0o700,parents=True,exist_ok=False)
    producer.owned_root(root)
    repo=args.repo.resolve()
    binary=args.binary.resolve()
    capture=producer.captured(repo,args.captured_config)
    common=[]
    for name in ["binary","repo","compiler","runtime-root","cache","captured-config"]:
        common += ["--"+name,str(getattr(args,name.replace("-","_")).resolve())]
    rows=[]
    for row in ["smoke","frontend"]:
        cell=root/row
        cell.mkdir(mode=0o700)
        policy=cell/"policy.json"
        installation=cell/"installation.json"
        producer.write(policy,workloads.support_policy(row,capture))
        producer.run(cell,"install",[str(binary),"install","--compiler",str(args.compiler.resolve()),"--runtime-root",str(args.runtime_root.resolve()),"--cache",str(args.cache.resolve()),
            "--output",str(installation),"--profile","gomad.wasi-cooperative/v1","--config",str(policy),"--fuel",str(1<<50),"--memory-bytes",str(4<<30)],0,json_output=False,environment=gate_environment(row))
        spec=workloads.support_spec(repo,cell/"unused",row,False,source_manifest=args.source_manifest)
        target=spec["Campaign"]["Target"]
        command=[str(binary),"--installation",str(installation),"plan","--json","--output",str(cell/"plan.json"),"--artifacts",str(cell/"plan-artifacts"),"--choices","--seeds=7","--working-dir",str(repo),
            "--execution-timeout=15m","--overall-timeout=30m","--terminate-grace=1s","--output-limit",str(256<<20),"--choice-bytes",str(64<<20)]
        for environment in spec["Campaign"]["Environment"]:
            command += ["--env",environment]
        for tag in target["BuildTags"]:
            command += ["--build-tag",tag]
        command += ["go-test",target["Source"],"--",*target["Args"]]
        prepared=producer.run(cell,"prepare",command,0,environment=gate_environment(row))
        receipt=Path(prepared["preparation_receipt"])
        if prepared["guest_executions"]!=0 or hashlib.sha256(receipt.read_bytes()).hexdigest()!=prepared["preparation_receipt_sha256"]:
            raise ValueError("original unchanged-source preparation receipt failed authentication")
        if args.prepare_only:
            print(json.dumps(dict(row=row,preparation=prepared)))
            return
        results={}
        for mode in ["success","canary"]:
            response=producer.run(cell,mode,[sys.executable,str(Path(__file__).with_name("producer.py")),*common,"--root",str(cell/mode),"--workload",row,"--mode",mode,"--source-manifest",str(args.source_manifest.resolve())],0,environment=gate_environment(row,measured=True))
            result_path=Path(response["result_path"])
            actual=json.loads(result_path.read_bytes())
            if actual["manifest"]["module_sha256"]!=prepared["module_sha256"]:
                raise ValueError("row preparation and both measured cells must use the same unchanged module")
            for execution in actual["report"]["executions"]:
                if mode=="success":
                    stdout=Path(execution["artifact_path"])/"output/stdout"
                    raw=stdout.read_bytes()
                    stream=execution["record"]["streams"]["output/stdout"]
                    if len(raw)!=stream["bytes"] or hashlib.sha256(raw).hexdigest()!=stream["sha256"]:
                        raise ValueError("support named-PASS stdout changed")
                    selected_pass(raw,row)
            results[mode]=str(result_path)
        if row=="smoke":
            inspection=producer.run(cell,"inspection",[sys.executable,str(Path(__file__).with_name("temporal-inspection.py")),*common,"--root",str(root/"inspection"),"--owner","onebox"],0,environment=gate_environment(row))
        rows.append(dict(row=row,preparation=prepared,results=results))
    producer.write(root/"support-receipt.json",dict(schema="gomad.rust-temporal-support-receiving/v1",rows=rows,inspection=inspection,
        subgates=[dict(id=name,status="PASS",executed=True,exit=0) for name in ["unchanged-source-preparation","measured-pass-replay-success","measured-pass-replay-canary","storage-transport-inspection"]]))
    print(json.dumps(dict(receipt=str(root/"support-receipt.json"))))


if __name__=="__main__":
    main()
