"""Original mandatory workload requests for the public Rust release CLI.

This module produces source/configuration intent. It never adopts a prior
execution, preparation measurement, Result or publication receipt.
"""
import base64
import copy
import hashlib
import json
from pathlib import Path
import stat

WASM = Path("tools/gomad/internal/backends/wasm")
TAGS = ["disable_grpc_modules", "gomad_wasm_slim", "sqlite3_dotlk", "test_dep"]
REPLICATION_ARGS = ["-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"]
TEMPORAL_ENVIRONMENT = ["PWD=/workspace", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC",
                        "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db", "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3"]


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def handle(node, incarnation):
    return {"node": node, "incarnation": incarnation}


def action(kind, node, incarnation):
    return {"action": kind, "handle": handle(node, incarnation)}


def scenario_limits():
    return dict(nodes=64, directional_links=4096, volumes=64, scenario_actions=4096,
                scenario_decisions=4096, fault_actions=4096, history_operations=65536,
                observations=65536, oracle_results=4096, scenario_evidence_bytes=64 << 20,
                boot_config_bytes=1 << 20, volume_capacity_bytes=1 << 30,
                observation_bytes=64 << 20, network_listeners=128, network_connections=4096,
                network_bytes=128 << 20, network_connection_bytes=4 << 20,
                network_stall_nanos=10_000_000_000, network_transitions=1 << 20,
                volume_operations=100000, volume_transitions=1 << 20, crash_states=32,
                crash_depth=8, crash_bytes=4 << 20, crash_wall_nanos=1_000_000_000)


def base_scenario(workload):
    return dict(schema="gomad-wasm.scenario/v1", workload=workload,
                spec=dict(schema="gomad.simulation-spec/v8", backend="wasm", fidelity="simulation_model", seed=0,
                          limits=scenario_limits(), nodes=[], links=[], volumes=[]),
                boots=[], actions=[], incarnation_boots=[], output_crashes=[], output_stops=[], network_faults=[])


def sqlite_storage():
    return dict(volumes=[dict(ID="sqlite", Path="/volume", CapacityBytes=4 << 20)],
                limits=dict(PendingOperations=10000, Transitions=100000),
                faults=[dict(name="acknowledged-crash", resource="/volume/crash-trigger", operation="open", occurrence=1, action="crash-before")],
                sqlite_lock_databases=["/volume/db"])


def sqlite_scenario(canary=False):
    value = base_scenario(dict(kind="sqlite-durable-recovery", selection=None))
    value["spec"]["nodes"] = [dict(id="sqlite", boot="setup-live", address="10.0.0.1", volumes=[dict(volume="sqlite", path="/volume")])]
    value["spec"]["volumes"] = [dict(id="sqlite", capacity_bytes=4 << 20)]
    value["boots"] = [dict(id=phase, args=["unsynced" if canary and phase == "append-crash" else phase, "WAL"], storage=sqlite_storage())
                      for phase in ["setup-live", "append-crash", "recover"]]
    value["incarnation_boots"] = [dict(handle=handle("sqlite", 2), boot="append-crash"), dict(handle=handle("sqlite", 3), boot="recover")]
    value["actions"] = [action("start", "sqlite", 1), action("wait", "sqlite", 1), action("restart", "sqlite", 2),
                        action("wait" if canary else "crash", "sqlite", 2), action("restart", "sqlite", 3), action("wait", "sqlite", 3)]
    return value


def temporal_storage(volume):
    return dict(volumes=[dict(ID=volume, Path="/volume", CapacityBytes=256 << 20)],
                limits=dict(PendingOperations=100000, Transitions=10000000), sqlite_lock_databases=["/volume/temporal.db"])


def temporal_replication_scenario(canary=False):
    value = base_scenario(dict(kind="independent-temporal-replication-durable-restart", canary=canary))
    for index, role in enumerate(["active", "standby"]):
        value["spec"]["nodes"].append(dict(id=role, boot=role+"-commit", address=f"10.0.0.{index+1}", volumes=[dict(volume=role, path="/volume")]))
        value["spec"]["volumes"].append(dict(id=role, capacity_bytes=256 << 20))
        for phase in ["commit", "reopen"]:
            value["boots"].append(dict(id=role+"-"+phase, args=REPLICATION_ARGS.copy(), storage=temporal_storage(role)))
        value["incarnation_boots"].append(dict(handle=handle(role, 2), boot=role+"-reopen"))
    value["output_crashes"]=[dict(handle=handle(role,1),marker=base64.b64encode(b"GOMAD_TEMPORAL_OBSERVATION ").decode()) for role in ["active","standby"]]
    value["spec"]["links"] = [dict(**{"from": a, "to": b}, enabled=True) for a, b in [("active", "standby"), ("standby", "active")]]
    value["network_faults"] = [dict(Name="temporal-active-to-standby-"+operation, Action=operation, From="active", To="standby", AtNanos=at)
                                for operation, at in [("partition", 1), ("heal", 30_000_000_000)]]
    value["actions"] = [action(kind, role, incarnation) for incarnation in [1, 2]
                        for kind in (["start", "crash"] if incarnation == 1 else ["restart", "wait"])
                        for role in ["active", "standby"]]
    return value


def stage_temporal(repo, destination, workload):
    sources = {
        "replication": [("temporal_storage/recovery_test.go", "recovery_test.go"), ("temporal_broker/replication_test.go", "replication_test.go")],
        "recovery": [("temporal_storage/recovery_test.go", "recovery_test.go")],
        "retry": [("network_streams/"+path.name,path.name) for path in sorted((Path(repo)/WASM/"testdata/network_streams").glob("*.go"))],
    }[workload]
    destination = Path(destination)
    destination.mkdir(parents=True, mode=0o700, exist_ok=True)
    if destination.is_symlink() or set(path.name for path in destination.iterdir()) - {name for _, name in sources}:
        raise ValueError("unexpected staged Temporal input")
    manifest = {}
    for source, name in sources:
        original = Path(repo) / WASM / "testdata" / source
        if not stat.S_ISREG(original.lstat().st_mode):
            raise ValueError("original Temporal input is not regular")
        body = original.read_bytes()
        target = destination / name
        if target.exists():
            if not stat.S_ISREG(target.lstat().st_mode) or target.read_bytes() != body:
                raise ValueError("staged Temporal source differs from unchanged original")
        else:
            with target.open("xb") as stream:
                stream.write(body)
            target.chmod(0o600)
        manifest[str(original)] = hashlib.sha256(body).hexdigest()
    return manifest


def policy(workload, captured_inputs=None):
    """Complete original Go WASI policy passed to public `install --config`."""
    temporal = workload != "sqlite"
    if temporal and not captured_inputs:
        raise ValueError("actual authenticated schema/sqlite captured inputs required")
    limits = dict(OutputBytes=8 << 20, TranscriptBytes=128 << 20, Calls=4000000, Descriptors=4096,
                  Files=10000, FilesystemBytes=512 << 20, PendingEvents=4096) if temporal else dict(
                      OutputBytes=1 << 20, TranscriptBytes=32 << 20, Calls=100000, Descriptors=128,
                      Files=1024, FilesystemBytes=8 << 20, PendingEvents=4096)
    environment = ["PWD=/workspace", "TZ=UTC"]
    if temporal:
        environment = TEMPORAL_ENVIRONMENT.copy()
    result = dict(Profile="gomad.wasi-cooperative/v1", Args=None, Environment=environment, WorkingDirectory="/workspace",
                  WritableDirectories=["/workspace", "/tmp", "/volume"],
                  EntropyKey=list(hashlib.sha256(b"gomad-wasm-temporal-recovery/v1").digest()) if temporal else [0]*32,
                  Clock=dict(EpochNanos=1700000000000000000 if temporal else 946684800000000000, ReadStepNanos=0), Limits=limits)
    if temporal:
        result["CapturedInputs"] = copy.deepcopy(captured_inputs)
    if workload == "replication":
        result["network_v2"] = dict(profile="gomad.simulation-network-stream/v3", limits=dict(listeners=128, connections=4096, bytes=128 << 20,
             connection_bytes=4 << 20, stall_nanos=10_000_000_000, transitions=1 << 20))
    if workload == "recovery":
        result["storage"] = temporal_storage("temporal")
    if not temporal:
        result["storage"] = sqlite_storage()
    return result


def guest_build_tags(profile=""):
    if profile not in ("", "slim", "full"):
        raise ValueError("qualification guest profile must be full or slim")
    return [tag for tag in TAGS if profile != "full" or tag != "gomad_wasm_slim"]


def qualification_spec(repo, root, workload, canary=False, source=None, source_manifest=None, guest_profile="", *, source_working_dir=None, original_clusters=False):
    """Each caller invokes qualify in its own measured process and fresh root."""
    root = Path(root).resolve()
    temporal = workload != "sqlite"
    if original_clusters and (workload != "replication" or canary):
        raise ValueError("original clusters requires one successful replication run")
    if workload not in ("sqlite", "replication", "recovery"):
        raise ValueError("unknown mandatory workload")
    if temporal and not source:
        raise ValueError("fresh unchanged staged Temporal source required")
    if workload == "recovery" and canary:
        raise ValueError("recovery catalogue is an original success gate")
    scenario = temporal_recovery_scenario() if workload == "recovery" else temporal_replication_scenario(canary) if temporal else sqlite_scenario(canary)
    manifest = dict(Provisional=source_manifest is None, ID={"replication":"independent-temporal-replication-durable-restart","recovery":"temporal-committed-workflow-crash-reopen","sqlite":"sqlite-durable-recovery"}[workload],
                    Workload="multi-instance" if temporal else "sqlite", Cell="canary" if canary else "success", Seed=7, Repeat=1 if original_clusters else 2,
                    IntegratedSourceSHA256="sha256:"+hashlib.sha256(Path(source_manifest).read_bytes() if source_manifest else canonical(scenario)).hexdigest())
    if source_manifest:
        manifest["IntegratedSourceManifestPath"] = str(Path(source_manifest).resolve())
    target = dict(Backend="wasm", Kind="go-test" if temporal else "go-run", Source=str(source) if temporal else "./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go",
                  WorkingDir=str(Path(source_working_dir if source_working_dir is not None else repo).resolve()), BuildTags=guest_build_tags(guest_profile) if temporal else ["sqlite3_dotlk"],
                  Scenario=base64.b64encode(canonical(scenario)).decode())
    campaign = dict(Target=target, ExecutionTimeout=(900 if temporal else 60)*10**9, OverallTimeout=(3600 if temporal else 180)*10**9,
                    TerminateGrace=100_000_000, OutputLimit=8 << 20 if temporal else 1 << 20, WorldTransitionLimit=32 << 20,
                    ChoiceTraceLimit=1<<30 if workload=="recovery" else 64 << 20 if temporal else 8 << 20, SuccessBytesLimit=2 << 30 if temporal else 256 << 20,
                    Environment=TEMPORAL_ENVIRONMENT.copy() if temporal else ["PWD=/workspace","TZ=UTC"],
                    OnFailure="all", FailureBudget=1, Artifacts=str(root/"artifacts"))
    return dict(Manifest=manifest, Campaign=campaign, ArtifactRoot=str(root))

INSPECTION = {
    "shard": ("TestGomadShardDumpUsesSnapshotWithLockHeld", "temporal.shards", "temporal.shard.snapshot", "shard=7 range=19"),
    "workflow": ("TestGomadMutableStateDumpUsesSnapshotWithWorkflowLockHeld", "temporal.mutable_states", "temporal.mutable_state.snapshot", "persisted_next_event=11"),
    "onebox": ("TestReplayInspectionWorkflowFailure", "temporal.mutable_states", "temporal.*.snapshot", "snapshot="),
}


def stage_inspection(repo, destination, owner):
    original = Path(repo)/(WASM/"testdata/prototype_temporal" if owner == "onebox" else Path("service/history")/owner)
    sources = {path.name: path for path in original.iterdir()
               if path.is_file() and path.name.endswith(".go") and (owner == "onebox" or not path.name.endswith("_test.go") or path.name == "gomad_debug_test.go")}
    destination = Path(destination)
    destination.mkdir(parents=True, mode=0o700, exist_ok=True)
    if destination.is_symlink() or set(path.name for path in destination.iterdir()) - sources.keys():
        raise ValueError("unexpected staged inspection input")
    manifest = {}
    for name, original in sorted(sources.items()):
        body = original.read_bytes()
        target = destination/name
        if target.exists():
            if target.is_symlink() or not target.is_file() or target.read_bytes() != body:
                raise ValueError("staged inspection source differs from unchanged original")
        else:
            with target.open("xb") as output:
                output.write(body)
            target.chmod(0o600)
        manifest[str(original)] = hashlib.sha256(body).hexdigest()
    if not manifest:
        raise ValueError("no original inspection source selected")
    return manifest


def inspection_policy(owner, captured_inputs=None):
    if owner not in INSPECTION:
        raise ValueError("unknown inspection owner")
    result = dict(Profile="gomad.wasi-cooperative/v1", Args=None, Environment=["PWD=/workspace", "GOMAD_DEBUG_CANARY=1"],
                  WorkingDirectory="/workspace", WritableDirectories=["/workspace", "/tmp"], EntropyKey=[0]*32,
                  Clock=dict(EpochNanos=946684800000000000, ReadStepNanos=0), Limits=dict(OutputBytes=1 << 20, TranscriptBytes=8 << 20,
                       Calls=10000, Descriptors=64, Files=1000, FilesystemBytes=8 << 20, PendingEvents=4096))
    if owner == "onebox":
        if not captured_inputs:
            raise ValueError("unchanged authenticated Temporal schema inputs required")
        result["CapturedInputs"] = copy.deepcopy(captured_inputs)
        result["Environment"] += ["TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"]
        result["Limits"] = dict(OutputBytes=16 << 20, TranscriptBytes=256 << 20, Calls=1 << 28, Descriptors=4096,
                                Files=100000, FilesystemBytes=512 << 20, PendingEvents=4096)
        result["network_v2"] = policy("replication", captured_inputs)["network_v2"]
    return result


def inspection_scenario(arguments):
    result = base_scenario(dict(kind="temporal-onebox-stopped-inspection"))
    result["spec"]["nodes"] = [dict(id="history", boot="main", address="127.0.0.1")]
    result["boots"] = [dict(id="main", args=arguments)]
    result["actions"] = [action("start", "history", 1), action("wait", "history", 1)]
    return result


def temporal_recovery_scenario():
    value=base_scenario(dict(kind="temporal-committed-workflow-crash-reopen",forcing=None))
    value["spec"]["nodes"]=[dict(id="temporal",boot="commit",address="127.0.0.1",volumes=[dict(volume="temporal",path="/volume")])]
    value["spec"]["volumes"]=[dict(id="temporal",capacity_bytes=256<<20)]
    arguments=["-test.v","-test.count=1","-test.timeout=1h","-test.run=^TestCommittedWorkflowCrashRecovery$","-persistenceType=sql","-persistenceDriver=sqlite"]
    value["boots"]=[dict(id=phase,args=arguments.copy(),storage=temporal_storage("temporal")) for phase in ["commit","reopen"]]
    value["incarnation_boots"]=[dict(handle=handle("temporal",incarnation),boot="reopen") for incarnation in [2,3]]
    value["actions"]=[action("start","temporal",1),action("crash","temporal",1),action("restart","temporal",2),action("wait","temporal",2),action("restart","temporal",3),action("wait","temporal",3)]
    value["output_crashes"]=[dict(handle=handle("temporal",1),marker=base64.b64encode(b"GOMAD_TEMPORAL_OBSERVATION ").decode())]
    return value

SUPPORT_ROWS={
    "smoke":("unchanged-workflow-type-smoke","./tests","/workspace/tests","^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"),
    "frontend":("frontend-system-info","./tests/gomadfunctional","/workspace/tests/gomadfunctional","^TestFrontendSystemInfo$"),
    "retry":("temporal-named-network-reset",None,"/workspace/network-temporal","^TestTemporalWorkflowRecoversNamedNetworkReset$"),
}


def support_policy(row,captured_inputs):
    _,_,cwd,_=SUPPORT_ROWS[row]
    if not captured_inputs:
        raise ValueError("authenticated complete schema capture required")
    value=dict(Profile="gomad.wasi-cooperative/v1",Args=None,
        Environment=["PWD="+cwd,"TEMPORAL_ROOT=/workspace","TMPDIR=/tmp","TZ=UTC","TEMPORAL_TEST_LOG_LEVEL=warn"],
        WorkingDirectory=cwd,WritableDirectories=["/workspace","/tmp",cwd],CapturedInputs=copy.deepcopy(captured_inputs),
        EntropyKey=list(hashlib.sha256(b"gomad-wasm-prototype/v1").digest()),Clock=dict(EpochNanos=946684800000000000,ReadStepNanos=0),
        Limits=dict(OutputBytes=256<<20,TranscriptBytes=1<<30,Calls=1<<28,Descriptors=4096,Files=100000,FilesystemBytes=512<<20,PendingEvents=4096))
    if row=="retry":
        value["network_v2"]=policy("replication",captured_inputs)["network_v2"]
        value["network_v2"]["faults"]=[dict(name="rpc-reset",action="reset",at_nanos=40_000_000_000)]
    return value


def support_spec(repo,root,row,canary=False,source=None,source_manifest=None,guest_profile="", *, source_working_dir=None):
    identity,package,cwd,selection=SUPPORT_ROWS[row]
    root=Path(root).resolve()
    if row=="retry" and (canary or not source):
        raise ValueError("named reset requires unchanged staged original success target")
    arguments=["-test.v","-test.count=1","-test.timeout=72h","-test.run="+("[" if canary else selection),"-persistenceType=sql","-persistenceDriver=sqlite"]
    manifest=dict(Provisional=source_manifest is None,ID=identity,Workload="onebox",Cell="canary" if canary else "success",Seed=7,Repeat=2,
        IntegratedSourceSHA256="sha256:"+hashlib.sha256(Path(source_manifest).read_bytes() if source_manifest else canonical(arguments)).hexdigest())
    if source_manifest:
        manifest["IntegratedSourceManifestPath"]=str(Path(source_manifest).resolve())
    value=dict(Manifest=manifest,ArtifactRoot=str(root),Campaign=dict(Target=dict(Backend="wasm",Kind="go-test",Source=source or package,WorkingDir=str(Path(source_working_dir if source_working_dir is not None else repo).resolve()),BuildTags=(["test_dep","disable_grpc_modules","sqlite3_dotlk"] if row=="retry" else guest_build_tags(guest_profile)),Args=arguments),
        ExecutionTimeout=900*10**9,OverallTimeout=1800*10**9,TerminateGrace=10**9,OutputLimit=256<<20,WorldTransitionLimit=32<<20,
        ChoiceTraceLimit=1<<30 if row=="retry" else 64<<20,SuccessBytesLimit=2<<30,Environment=support_policy(row,{"unused":True})["Environment"],OnFailure="all",FailureBudget=1,Artifacts=str(root/"artifacts")))
    if row=="retry":
        value["RuntimeWorkload"]=dict(kind="temporal-named-network-reset")
    return value
