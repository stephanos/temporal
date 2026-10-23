#!/usr/bin/env python3
"""Run every native committed-workflow crash state through the public Rust lane."""
import argparse
import hashlib
import json
from pathlib import Path
import sys
sys.dont_write_bytecode=True
import producer
import workloads


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary",required=True,type=Path)
    parser.add_argument("--primary",required=True,type=Path,help="fresh recovery producer root")
    parser.add_argument("--root",required=True,type=Path)
    args=parser.parse_args()
    root=args.root.resolve()
    root.mkdir(mode=0o700,parents=True,exist_ok=False)
    producer.owned_root(root)
    primary=args.primary.resolve()
    producer_receipt=json.loads((primary/"producer.json").read_bytes())
    if producer_receipt["workload"]!="recovery" or producer_receipt["mode"]!="success":
        raise ValueError("requires the actual current recovery primary producer")
    result=Path(producer_receipt["result_path"])
    if digest(result.read_bytes())!=producer_receipt["result_sha256"]:
        raise ValueError("primary result changed")
    producer.run(root,"primary-report",[str(args.binary.resolve()),"report","--result",str(result)],0)
    retained=json.loads(result.read_bytes())
    artifact=Path(retained["report"]["executions"][0]["artifact_path"])
    manifest=json.loads((artifact/"manifest.json").read_bytes())
    state=(artifact/"evidence/cluster/temporal/1/storage_state").read_bytes()
    catalogues=retained["native_crash_catalogues"]
    if len(catalogues)!=1 or catalogues[0]["volume"]!="temporal" or catalogues[0]["storage_state_sha256"]!=digest(state):
        raise ValueError("actual authenticated primary Temporal catalogue unavailable")
    enumeration,context=catalogues[0]["enumeration"],catalogues[0]["context"]
    prefix=(artifact/"evidence/cluster/temporal/1/choice").read_bytes()
    base=json.loads((primary/"qualification-spec.json").read_bytes())
    results=[]
    for index,selected in enumerate(enumeration["States"]):
        cell=root/str(index)
        cell.mkdir(mode=0o700)
        spec=json.loads(json.dumps(base))
        spec["ArtifactRoot"]=str(cell/"cell")
        spec["Campaign"]["Artifacts"]=str(cell/"cell/artifacts")
        spec["RuntimeWorkload"]=dict(kind="temporal-committed-workflow-crash-reopen",primary=dict(artifact_path=str(artifact),
            record_sha256=digest(workloads.canonical(manifest["record"])),prefix_sha256=digest(prefix),storage_state_sha256=digest(state),state=selected,context=context))
        producer.write(cell/"spec.json",spec)
        response=producer.run(cell,"qualify",[str(args.binary.resolve()),"--installation",str(primary/"installation.json"),"qualify","--spec",str(cell/"spec.json")],0)
        path=Path(response["result_path"])
        producer.run(cell,"report",[str(args.binary.resolve()),"report","--result",str(path)],0)
        actual=json.loads(path.read_bytes())
        if not actual["report"]["qualified"] or len(actual["report"]["executions"])!=2 or any(not row["replay"]["matched"] for row in actual["report"]["executions"]):
            raise ValueError("selected original durable state did not recover and replay exactly")
        results.append(dict(index=index,state=selected["Identity"],result_path=str(path),result_sha256=digest(path.read_bytes())))
    if digest(result.read_bytes())!=producer_receipt["result_sha256"] or json.loads((artifact/"manifest.json").read_bytes())["record"]!=manifest["record"]:
        raise ValueError("original primary Result or committed record changed during selected-state producers")
    producer.run(root,"primary-final-report",[str(args.binary.resolve()),"report","--result",str(result)],0)
    producer.write(root/"recovery-receipt.json",dict(schema="gomad.rust-temporal-crash-recovery/v1",primary_result=str(result),primary_result_sha256=digest(result.read_bytes()),
        catalogue=enumeration,context=context,results=results,subgates=[dict(id="TestTemporalCommittedWorkflowFreshGuestCrashReopen",status="PASS",executed=True,exit=0)]))
    print(json.dumps(dict(receipt=str(root/"recovery-receipt.json"))))


if __name__=="__main__":
    main()
