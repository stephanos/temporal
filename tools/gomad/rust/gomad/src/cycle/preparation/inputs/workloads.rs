pub(crate) use super::canonical;
use super::{digest, read, regular, require};
use crate::campaign::exploration::base64;
use serde_json::{Value, json};
use sha2::{Digest as _, Sha256};
use std::{
    collections::BTreeSet,
    fs,
    io::Write,
    path::{Component, Path, PathBuf},
};
pub(crate) const WASM: &str = "tools/gomad/internal/backends/wasm";
pub(crate) const TAGS: &[&str] = &[
    "disable_grpc_modules",
    "gomad_wasm_slim",
    "sqlite3_dotlk",
    "test_dep",
];
pub(crate) const REPLICATION_ARGS: &[&str] = &[
    "-test.v",
    "-test.count=1",
    "-test.timeout=1h",
    "-test.run=^TestIndependentClusterReplicationRecovery$",
    "-persistenceType=sql",
    "-persistenceDriver=sqlite",
];
pub(crate) const TEMPORAL_ENVIRONMENT: &[&str] = &[
    "PWD=/workspace",
    "TEMPORAL_ROOT=/workspace",
    "TMPDIR=/tmp",
    "TZ=UTC",
    "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db",
    "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3",
];
pub(crate) const INSPECTION: &[(&str, [&str; 4])] = &[
    (
        "shard",
        [
            "TestGomadShardDumpUsesSnapshotWithLockHeld",
            "temporal.shards",
            "temporal.shard.snapshot",
            "shard=7 range=19",
        ],
    ),
    (
        "workflow",
        [
            "TestGomadMutableStateDumpUsesSnapshotWithWorkflowLockHeld",
            "temporal.mutable_states",
            "temporal.mutable_state.snapshot",
            "persisted_next_event=11",
        ],
    ),
    (
        "onebox",
        [
            "TestReplayInspectionWorkflowFailure",
            "temporal.mutable_states",
            "temporal.*.snapshot",
            "snapshot=",
        ],
    ),
];
pub(crate) const SUPPORT_ROWS: &[(&str, &str, Option<&str>, &str, &str)] = &[
    (
        "smoke",
        "unchanged-workflow-type-smoke",
        Some("./tests"),
        "/workspace/tests",
        "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
    ),
    (
        "frontend",
        "frontend-system-info",
        Some("./tests/gomadfunctional"),
        "/workspace/tests/gomadfunctional",
        "^TestFrontendSystemInfo$",
    ),
    (
        "retry",
        "temporal-named-network-reset",
        None,
        "/workspace/network-temporal",
        "^TestTemporalWorkflowRecoversNamedNetworkReset$",
    ),
];
pub(crate) fn handle(node: &str, incarnation: u64) -> Value {
    json!({"node":node,"incarnation":incarnation})
}
pub(crate) fn action(kind: &str, node: &str, incarnation: u64) -> Value {
    json!({"action":kind,"handle":handle(node,incarnation)})
}
pub(crate) fn scenario_limits() -> Value {
    json!({"nodes":64,"directional_links":4096,"volumes":64,"scenario_actions":4096,"scenario_decisions":4096,"fault_actions":4096,"history_operations":65536,"observations":65536,"oracle_results":4096,"scenario_evidence_bytes":64_u64<<20,"boot_config_bytes":1_u64<<20,"volume_capacity_bytes":1_u64<<30,"observation_bytes":64_u64<<20,"network_listeners":128,"network_connections":4096,"network_bytes":128_u64<<20,"network_connection_bytes":4_u64<<20,"network_stall_nanos":10_000_000_000_u64,"network_transitions":1_u64<<20,"volume_operations":100_000,"volume_transitions":1_u64<<20,"crash_states":32,"crash_depth":8,"crash_bytes":4_u64<<20,"crash_wall_nanos":1_000_000_000})
}
pub(crate) fn base_scenario(workload: Value) -> Value {
    let mut value = json!({"schema":"gomad-wasm.scenario/v1","workload":null,"spec":{"schema":"gomad.simulation-spec/v8","backend":"wasm","fidelity":"simulation_model","seed":0,"limits":scenario_limits(),"nodes":[],"links":[],"volumes":[]},"boots":[],"actions":[],"incarnation_boots":[],"output_crashes":[],"output_stops":[],"network_faults":[]});
    value["workload"] = workload;
    value
}
pub(crate) fn sqlite_storage() -> Value {
    json!({"volumes":[{"ID":"sqlite","Path":"/volume","CapacityBytes":4_u64<<20}],"limits":{"PendingOperations":10000,"Transitions":100_000},"faults":[{"name":"acknowledged-crash","resource":"/volume/crash-trigger","operation":"open","occurrence":1,"action":"crash-before"}],"sqlite_lock_databases":["/volume/db"]})
}
pub(crate) fn sqlite_scenario(canary: bool) -> Value {
    let mut v = base_scenario(json!({"kind":"sqlite-durable-recovery","selection":null}));
    v["spec"]["nodes"] = json!([{"id":"sqlite","boot":"setup-live","address":"10.0.0.1","volumes":[{"volume":"sqlite","path":"/volume"}]}]);
    v["spec"]["volumes"] = json!([{"id":"sqlite","capacity_bytes":4_u64<<20}]);
    v["boots"]=["setup-live","append-crash","recover"].iter().map(|phase|json!({"id":phase,"args":[if canary&&*phase=="append-crash"{"unsynced"}else{phase},"WAL"],"storage":sqlite_storage()})).collect();
    v["incarnation_boots"] = json!([{"handle":handle("sqlite",2),"boot":"append-crash"},{"handle":handle("sqlite",3),"boot":"recover"}]);
    v["actions"] = json!([
        action("start", "sqlite", 1),
        action("wait", "sqlite", 1),
        action("restart", "sqlite", 2),
        action(if canary { "wait" } else { "crash" }, "sqlite", 2),
        action("restart", "sqlite", 3),
        action("wait", "sqlite", 3)
    ]);
    v
}
pub(crate) fn temporal_storage(volume: &str) -> Value {
    json!({"volumes":[{"ID":volume,"Path":"/volume","CapacityBytes":256_u64<<20}],"limits":{"PendingOperations":100_000,"Transitions":10_000_000},"sqlite_lock_databases":["/volume/temporal.db"]})
}
fn observation_marker() -> String {
    base64(b"GOMAD_TEMPORAL_OBSERVATION ")
}
pub(crate) fn temporal_replication_scenario(canary: bool) -> Value {
    let mut v = base_scenario(
        json!({"kind":"independent-temporal-replication-durable-restart","canary":canary}),
    );
    for (index, role) in ["active", "standby"].iter().enumerate() {
        v["spec"]["nodes"].as_array_mut().unwrap().push(json!({"id":role,"boot":format!("{role}-commit"),"address":format!("10.0.0.{}",index+1),"volumes":[{"volume":role,"path":"/volume"}]}));
        v["spec"]["volumes"]
            .as_array_mut()
            .unwrap()
            .push(json!({"id":role,"capacity_bytes":256_u64<<20}));
        for phase in ["commit", "reopen"] {
            v["boots"].as_array_mut().unwrap().push(json!({"id":format!("{role}-{phase}"),"args":REPLICATION_ARGS,"storage":temporal_storage(role)}));
        }
        v["incarnation_boots"]
            .as_array_mut()
            .unwrap()
            .push(json!({"handle":handle(role,2),"boot":format!("{role}-reopen")}));
    }
    v["output_crashes"] = ["active", "standby"]
        .iter()
        .map(|role| json!({"handle":handle(role,1),"marker":observation_marker()}))
        .collect();
    v["spec"]["links"] = json!([{"from":"active","to":"standby","enabled":true},{"from":"standby","to":"active","enabled":true}]);
    v["network_faults"]=[("partition",1_u64),("heal",30_000_000_000_u64)].iter().map(|(operation,at)|json!({"Name":format!("temporal-active-to-standby-{operation}"),"Action":operation,"From":"active","To":"standby","AtNanos":at})).collect();
    let mut actions = Vec::new();
    for incarnation in [1, 2] {
        for kind in if incarnation == 1 {
            ["start", "crash"]
        } else {
            ["restart", "wait"]
        } {
            for role in ["active", "standby"] {
                actions.push(action(kind, role, incarnation));
            }
        }
    }
    v["actions"] = actions.into();
    v
}
pub(crate) fn temporal_recovery_scenario() -> Value {
    let mut v =
        base_scenario(json!({"kind":"temporal-committed-workflow-crash-reopen","forcing":null}));
    v["spec"]["nodes"] = json!([{"id":"temporal","boot":"commit","address":"127.0.0.1","volumes":[{"volume":"temporal","path":"/volume"}]}]);
    v["spec"]["volumes"] = json!([{"id":"temporal","capacity_bytes":256_u64<<20}]);
    let args = [
        "-test.v",
        "-test.count=1",
        "-test.timeout=1h",
        "-test.run=^TestCommittedWorkflowCrashRecovery$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite",
    ];
    v["boots"] = ["commit", "reopen"]
        .iter()
        .map(|phase| json!({"id":phase,"args":args,"storage":temporal_storage("temporal")}))
        .collect();
    v["incarnation_boots"] = [2, 3]
        .iter()
        .map(|i| json!({"handle":handle("temporal",*i),"boot":"reopen"}))
        .collect();
    v["actions"] = json!([
        action("start", "temporal", 1),
        action("crash", "temporal", 1),
        action("restart", "temporal", 2),
        action("wait", "temporal", 2),
        action("restart", "temporal", 3),
        action("wait", "temporal", 3)
    ]);
    v["output_crashes"] = json!([{"handle":handle("temporal",1),"marker":observation_marker()}]);
    v
}
pub(crate) fn inspection_scenario(arguments: Value) -> Value {
    let mut v = base_scenario(json!({"kind":"temporal-onebox-stopped-inspection"}));
    v["spec"]["nodes"] = json!([{"id":"history","boot":"main","address":"127.0.0.1"}]);
    v["boots"] = json!([{"id":"main","args":null}]);
    v["boots"][0]["args"] = arguments;
    v["actions"] = json!([action("start", "history", 1), action("wait", "history", 1)]);
    v
}
fn truthy(v: &Value) -> bool {
    match v {
        Value::Null => false,
        Value::Bool(v) => *v,
        Value::Number(v) => v.as_f64() != Some(0.0),
        Value::String(v) => !v.is_empty(),
        Value::Array(v) => !v.is_empty(),
        Value::Object(v) => !v.is_empty(),
    }
}
fn captured(v: Option<&Value>, message: &str) -> Result<Value, String> {
    require(v.is_some_and(truthy), message)?;
    Ok(v.unwrap().clone())
}
fn entropy(domain: &[u8]) -> Value {
    json!(Sha256::digest(domain).to_vec())
}
pub(crate) fn policy(workload: &str, captured_inputs: Option<&Value>) -> Result<Value, String> {
    let temporal = workload != "sqlite";
    let capture = if temporal {
        Some(captured(
            captured_inputs,
            "actual authenticated schema/sqlite captured inputs required",
        )?)
    } else {
        None
    };
    let limits = if temporal {
        json!({"OutputBytes":8_u64<<20,"TranscriptBytes":128_u64<<20,"Calls":4_000_000,"Descriptors":4096,"Files":10000,"FilesystemBytes":512_u64<<20,"PendingEvents":4096})
    } else {
        json!({"OutputBytes":1_u64<<20,"TranscriptBytes":32_u64<<20,"Calls":100_000,"Descriptors":128,"Files":1024,"FilesystemBytes":8_u64<<20,"PendingEvents":4096})
    };
    let mut v = json!({"Profile":"gomad.wasi-cooperative/v1","Args":null,"Environment":if temporal{json!(TEMPORAL_ENVIRONMENT)}else{json!(["PWD=/workspace","TZ=UTC"] )},"WorkingDirectory":"/workspace","WritableDirectories":["/workspace","/tmp","/volume"],"EntropyKey":if temporal{entropy(b"gomad-wasm-temporal-recovery/v1")}else{json!(vec![0_u8;32])},"Clock":{"EpochNanos":if temporal{1_700_000_000_000_000_000_u64}else{946_684_800_000_000_000_u64},"ReadStepNanos":0},"Limits":limits});
    if let Some(capture) = capture {
        v["CapturedInputs"] = capture;
    }
    if workload == "replication" {
        v["network_v2"] = json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":4096,"bytes":128_u64<<20,"connection_bytes":4_u64<<20,"stall_nanos":10_000_000_000_u64,"transitions":1_u64<<20}});
    }
    if workload == "recovery" {
        v["storage"] = temporal_storage("temporal");
    }
    if !temporal {
        v["storage"] = sqlite_storage();
    }
    Ok(v)
}
pub(crate) fn guest_build_tags(profile: &str) -> Result<Vec<&'static str>, String> {
    require(
        matches!(profile, "" | "slim" | "full"),
        "qualification guest profile must be full or slim",
    )?;
    Ok(TAGS
        .iter()
        .copied()
        .filter(|tag| profile != "full" || *tag != "gomad_wasm_slim")
        .collect())
}
pub(crate) fn inspection_policy(
    owner: &str,
    captured_inputs: Option<&Value>,
) -> Result<Value, String> {
    require(
        INSPECTION.iter().any(|(name, _)| *name == owner),
        "unknown inspection owner",
    )?;
    let mut v = json!({"Profile":"gomad.wasi-cooperative/v1","Args":null,"Environment":["PWD=/workspace","GOMAD_DEBUG_CANARY=1"],"WorkingDirectory":"/workspace","WritableDirectories":["/workspace","/tmp"],"EntropyKey":vec![0_u8;32],"Clock":{"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":0},"Limits":{"OutputBytes":1_u64<<20,"TranscriptBytes":8_u64<<20,"Calls":10000,"Descriptors":64,"Files":1000,"FilesystemBytes":8_u64<<20,"PendingEvents":4096}});
    if owner == "onebox" {
        v["CapturedInputs"] = captured(
            captured_inputs,
            "unchanged authenticated Temporal schema inputs required",
        )?;
        v["Environment"].as_array_mut().unwrap().extend(
            [
                "TEMPORAL_ROOT=/workspace",
                "TMPDIR=/tmp",
                "TZ=UTC",
                "TEMPORAL_TEST_LOG_LEVEL=warn",
            ]
            .map(Value::from),
        );
        v["Limits"] = json!({"OutputBytes":16_u64<<20,"TranscriptBytes":256_u64<<20,"Calls":1_u64<<28,"Descriptors":4096,"Files":100_000,"FilesystemBytes":512_u64<<20,"PendingEvents":4096});
        v["network_v2"] = policy("replication", captured_inputs)?["network_v2"].clone();
    }
    Ok(v)
}
fn support_row(
    row: &str,
) -> Result<
    (
        &'static str,
        Option<&'static str>,
        &'static str,
        &'static str,
    ),
    String,
> {
    SUPPORT_ROWS
        .iter()
        .find(|(name, _, _, _, _)| *name == row)
        .map(|(_, id, package, cwd, selection)| (*id, *package, *cwd, *selection))
        .ok_or_else(|| format!("unknown support row: {row}"))
}
pub(crate) fn support_policy(row: &str, captured_inputs: &Value) -> Result<Value, String> {
    let (_, _, cwd, _) = support_row(row)?;
    let capture = captured(
        Some(captured_inputs),
        "authenticated complete schema capture required",
    )?;
    let mut v = json!({"Profile":"gomad.wasi-cooperative/v1","Args":null,"Environment":[format!("PWD={cwd}"),"TEMPORAL_ROOT=/workspace","TMPDIR=/tmp","TZ=UTC","TEMPORAL_TEST_LOG_LEVEL=warn"],"WorkingDirectory":cwd,"WritableDirectories":["/workspace","/tmp",cwd],"CapturedInputs":capture,"EntropyKey":entropy(b"gomad-wasm-prototype/v1"),"Clock":{"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":0},"Limits":{"OutputBytes":256_u64<<20,"TranscriptBytes":1_u64<<30,"Calls":1_u64<<28,"Descriptors":4096,"Files":100_000,"FilesystemBytes":512_u64<<20,"PendingEvents":4096}});
    if row == "retry" {
        v["network_v2"] = policy("replication", Some(captured_inputs))?["network_v2"].clone();
        v["network_v2"]["faults"] =
            json!([{"name":"rpc-reset","action":"reset","at_nanos":40_000_000_000_u64}]);
    }
    Ok(v)
}
#[derive(Default)]
pub(crate) struct QualificationOptions<'a> {
    pub(crate) canary: bool,
    pub(crate) source: Option<&'a str>,
    pub(crate) source_manifest: Option<&'a Path>,
    pub(crate) guest_profile: &'a str,
    pub(crate) source_working_dir: Option<&'a Path>,
    pub(crate) original_clusters: bool,
}
#[derive(Default)]
pub(crate) struct SupportOptions<'a> {
    pub(crate) canary: bool,
    pub(crate) source: Option<&'a str>,
    pub(crate) source_manifest: Option<&'a Path>,
    pub(crate) guest_profile: &'a str,
    pub(crate) source_working_dir: Option<&'a Path>,
}
fn absolute(path: &Path) -> Result<PathBuf, String> {
    let path = if path.is_absolute() {
        path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(|e| e.to_string())?
            .join(path)
    };
    let mut out = PathBuf::new();
    for part in path.components() {
        match part {
            Component::CurDir => (),
            Component::ParentDir => {
                out.pop();
            }
            part => out.push(part.as_os_str()),
        }
    }
    Ok(out)
}
fn resolve(path: &Path) -> Result<PathBuf, String> {
    let absolute = if path.is_absolute() {
        path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(|e| e.to_string())?
            .join(path)
    };
    let mut result = PathBuf::new();
    for component in absolute.components() {
        match component {
            Component::CurDir => {}
            Component::ParentDir => {
                result.pop();
            }
            other => {
                result.push(other);
                match fs::canonicalize(&result) {
                    Ok(resolved) => result = resolved,
                    Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
                    Err(e) => return Err(e.to_string()),
                }
            }
        }
    }
    Ok(result)
}
fn manifest_hash(manifest: Option<&Path>, fallback: &Value) -> Result<String, String> {
    let bytes = if let Some(path) = manifest.filter(|p| !p.as_os_str().is_empty()) {
        read(&absolute(path)?, 512 << 20)?
    } else {
        canonical(fallback)?
    };
    Ok(format!("sha256:{}", digest(&bytes)))
}
pub(crate) fn qualification_spec(
    repo: &Path,
    root: &Path,
    workload: &str,
    o: &QualificationOptions<'_>,
) -> Result<Value, String> {
    let root = resolve(root)?;
    let temporal = workload != "sqlite";
    require(
        !o.original_clusters || (workload == "replication" && !o.canary),
        "original clusters requires one successful replication run",
    )?;
    require(
        matches!(workload, "sqlite" | "replication" | "recovery"),
        "unknown mandatory workload",
    )?;
    require(
        !temporal || o.source.is_some_and(|s| !s.is_empty()),
        "fresh unchanged staged Temporal source required",
    )?;
    require(
        workload != "recovery" || !o.canary,
        "recovery catalogue is an original success gate",
    )?;
    let scenario = match workload {
        "recovery" => temporal_recovery_scenario(),
        "replication" => temporal_replication_scenario(o.canary),
        _ => sqlite_scenario(o.canary),
    };
    let identity = match workload {
        "recovery" => "temporal-committed-workflow-crash-reopen",
        "replication" => "independent-temporal-replication-durable-restart",
        _ => "sqlite-durable-recovery",
    };
    let mut manifest = json!({"Provisional":o.source_manifest.is_none(),"ID":identity,"Workload":if temporal{"multi-instance"}else{"sqlite"},"Cell":if o.canary{"canary"}else{"success"},"Seed":7,"Repeat":if o.original_clusters{1}else{2},"IntegratedSourceSHA256":manifest_hash(o.source_manifest,&scenario)?});
    if let Some(path) = o.source_manifest.filter(|p| !p.as_os_str().is_empty()) {
        manifest["IntegratedSourceManifestPath"] = json!(resolve(path)?);
    }
    let target = json!({"Backend":"wasm","Kind":if temporal{"go-test"}else{"go-run"},"Source":if temporal{o.source.unwrap()}else{"./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go"},"WorkingDir":resolve(o.source_working_dir.unwrap_or(repo))?,"BuildTags":if temporal{guest_build_tags(o.guest_profile)?}else{vec!["sqlite3_dotlk"]},"Scenario":base64(&canonical(&scenario)?)});
    let environment: Vec<_> = if temporal {
        TEMPORAL_ENVIRONMENT
    } else {
        &["PWD=/workspace"]
    }
    .iter()
    .copied()
    .filter(|v| !v.starts_with("TZ="))
    .collect();
    Ok(
        json!({"Manifest":manifest,"Campaign":{"Target":target,"ExecutionTimeout":if temporal{900_u64*1_000_000_000}else{60_u64*1_000_000_000},"OverallTimeout":if temporal{3600_u64*1_000_000_000}else{180_u64*1_000_000_000},"TerminateGrace":100_000_000,"OutputLimit":if temporal{8_u64<<20}else{1_u64<<20},"WorldTransitionLimit":32_u64<<20,"ChoiceTraceLimit":if workload=="recovery"{1_u64<<30}else if temporal{64_u64<<20}else{8_u64<<20},"SuccessBytesLimit":if temporal{2_u64<<30}else{256_u64<<20},"Environment":environment,"OnFailure":"all","FailureBudget":1,"Artifacts":root.join("artifacts")},"ArtifactRoot":root}),
    )
}
pub(crate) fn support_spec(
    repo: &Path,
    root: &Path,
    row: &str,
    o: &SupportOptions<'_>,
) -> Result<Value, String> {
    let (identity, package, _, selection) = support_row(row)?;
    let root = resolve(root)?;
    require(
        row != "retry" || (!o.canary && o.source.is_some_and(|s| !s.is_empty())),
        "named reset requires unchanged staged original success target",
    )?;
    let arguments = json!([
        "-test.v",
        "-test.count=1",
        "-test.timeout=72h",
        format!("-test.run={}", if o.canary { "[" } else { selection }),
        "-persistenceType=sql",
        "-persistenceDriver=sqlite"
    ]);
    let mut manifest = json!({"Provisional":o.source_manifest.is_none(),"ID":identity,"Workload":"onebox","Cell":if o.canary{"canary"}else{"success"},"Seed":7,"Repeat":2,"IntegratedSourceSHA256":manifest_hash(o.source_manifest,&arguments)?});
    if let Some(path) = o.source_manifest.filter(|p| !p.as_os_str().is_empty()) {
        manifest["IntegratedSourceManifestPath"] = json!(resolve(path)?);
    }
    let mut v = json!({"Manifest":manifest,"ArtifactRoot":root,"Campaign":{"Target":{"Backend":"wasm","Kind":"go-test","Source":o.source.filter(|s|!s.is_empty()).or(package),"WorkingDir":resolve(o.source_working_dir.unwrap_or(repo))?,"BuildTags":if row=="retry"{vec!["test_dep","disable_grpc_modules","sqlite3_dotlk"]}else{guest_build_tags(o.guest_profile)?},"Args":arguments},"ExecutionTimeout":900_u64*1_000_000_000,"OverallTimeout":1800_u64*1_000_000_000,"TerminateGrace":1_000_000_000,"OutputLimit":256_u64<<20,"WorldTransitionLimit":32_u64<<20,"ChoiceTraceLimit":if row=="retry"{1_u64<<30}else{64_u64<<20},"SuccessBytesLimit":2_u64<<30,"Environment":support_policy(row,&json!({"unused":true}))?["Environment"],"OnFailure":"all","FailureBudget":1,"Artifacts":root.join("artifacts")}});
    if row == "retry" {
        v["RuntimeWorkload"] = json!({"kind":"temporal-named-network-reset"});
    }
    Ok(v)
}
fn entries(path: &Path) -> Result<Vec<PathBuf>, String> {
    let mut paths = fs::read_dir(path)
        .map_err(|e| format!("{}: {e}", path.display()))?
        .map(|entry| entry.map(|e| e.path()))
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| e.to_string())?;
    paths.sort();
    Ok(paths)
}
#[expect(
    clippy::too_many_lines,
    reason = "Descriptor relative exclusive staging and byte reopening form one custody transaction"
)]
fn stage_sources(
    destination: &Path,
    sources: Vec<(PathBuf, String)>,
    label: &str,
) -> Result<Value, String> {
    use crate::evidence::store::open;
    use rustix::fs::{self as descriptor, FlockOperation, Mode, OFlags};
    let destination = absolute(destination)?;
    let mut directory = std::fs::File::from(
        descriptor::open(
            "/",
            OFlags::RDONLY | OFlags::DIRECTORY | OFlags::CLOEXEC,
            Mode::empty(),
        )
        .map_err(|e| e.to_string())?,
    );
    for component in destination.components().skip(1) {
        let name = component.as_os_str();
        match descriptor::mkdirat(&directory, name, Mode::from_raw_mode(0o700)) {
            Ok(()) | Err(rustix::io::Errno::EXIST) => (),
            Err(e) => return Err(e.to_string()),
        }
        directory = std::fs::File::from(
            descriptor::openat(
                &directory,
                name,
                OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                Mode::empty(),
            )
            .map_err(|e| format!("unexpected staged {label} input: {e}"))?,
        );
    }
    super::canonical_path(&destination)?;
    let pinned = open::pin(&destination)?;
    require(
        descriptor::fstat(&directory)
            .map_err(|e| e.to_string())?
            .st_ino
            == descriptor::fstat(&pinned)
                .map_err(|e| e.to_string())?
                .st_ino,
        "staged directory changed",
    )?;
    descriptor::flock(&pinned, FlockOperation::LockExclusive).map_err(|e| e.to_string())?;
    let names: BTreeSet<_> = sources.iter().map(|(_, name)| name.clone()).collect();
    require(
        entries(&destination)?.iter().all(|p| {
            p.file_name()
                .and_then(|s| s.to_str())
                .is_some_and(|name| names.contains(name))
        }),
        &format!("unexpected staged {label} input"),
    )?;
    let mut manifest = serde_json::Map::new();
    for (original, name) in sources {
        let original: PathBuf = original
            .components()
            .filter(|part| !matches!(part, Component::CurDir))
            .collect();
        require(
            !name.is_empty() && !name.contains('/') && !matches!(name.as_str(), "." | ".."),
            "unsafe staged filename",
        )?;
        let bytes = regular(&absolute(&original)?, 512 << 20, None).map_err(|error| {
            if label == "Temporal" {
                format!("original Temporal input is not regular: {error}")
            } else {
                error
            }
        })?;
        let target = destination.join(&name);
        match fs::symlink_metadata(&target) {
            Ok(_) => {
                let staged = regular(&target, 512 << 20, None).map_err(|e| {
                    format!("staged {label} source differs from unchanged original: {e}")
                })?;
                require(
                    staged == bytes,
                    &format!("staged {label} source differs from unchanged original"),
                )?;
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                let mut output = std::fs::File::from(
                    descriptor::openat(
                        &pinned,
                        name.as_str(),
                        OFlags::WRONLY
                            | OFlags::CREATE
                            | OFlags::EXCL
                            | OFlags::NOFOLLOW
                            | OFlags::CLOEXEC,
                        Mode::from_raw_mode(0o600),
                    )
                    .map_err(|e| e.to_string())?,
                );
                output
                    .write_all(&bytes)
                    .and_then(|()| output.sync_all())
                    .map_err(|e| e.to_string())?;
                require(
                    regular(&target, 512 << 20, Some(0o600))? == bytes,
                    &format!("staged {label} source differs from unchanged original"),
                )?;
            }
            Err(e) => return Err(e.to_string()),
        }
        manifest.insert(
            original
                .to_str()
                .ok_or("non-UTF8 original source path")?
                .to_owned(),
            json!(digest(&bytes)),
        );
    }
    pinned.sync_all().map_err(|e| e.to_string())?;
    super::canonical_path(&destination)?;
    Ok(Value::Object(manifest))
}
pub(crate) fn stage_temporal(
    repo: &Path,
    destination: &Path,
    workload: &str,
) -> Result<Value, String> {
    let testdata = repo.join(WASM).join("testdata");
    let sources = match workload {
        "replication" => vec![
            (
                testdata.join("temporal_storage/recovery_test.go"),
                "recovery_test.go".to_owned(),
            ),
            (
                testdata.join("temporal_broker/replication_test.go"),
                "replication_test.go".to_owned(),
            ),
        ],
        "recovery" => vec![(
            testdata.join("temporal_storage/recovery_test.go"),
            "recovery_test.go".to_owned(),
        )],
        "retry" => entries(&testdata.join("network_streams"))?
            .into_iter()
            .filter(|p| p.extension().is_some_and(|s| s == "go"))
            .map(|p| {
                let name = p
                    .file_name()
                    .and_then(|s| s.to_str())
                    .ok_or("non-UTF8 source name")?
                    .to_owned();
                Ok((p, name))
            })
            .collect::<Result<Vec<_>, String>>()?,
        _ => return Err(format!("unknown Temporal workload: {workload}")),
    };
    stage_sources(destination, sources, "Temporal")
}
pub(crate) fn stage_inspection(
    repo: &Path,
    destination: &Path,
    owner: &str,
) -> Result<Value, String> {
    require(
        INSPECTION.iter().any(|(name, _)| *name == owner),
        "unknown inspection owner",
    )?;
    let original = if owner == "onebox" {
        repo.join(WASM).join("testdata/prototype_temporal")
    } else {
        repo.join("service/history").join(owner)
    };
    let mut sources = Vec::new();
    for path in entries(&original)? {
        let Some(name) = path.file_name().and_then(|s| s.to_str()) else {
            return Err("non-UTF8 inspection source name".into());
        };
        if path.is_file()
            && path.extension().is_some_and(|extension| extension == "go")
            && (owner == "onebox" || !name.ends_with("_test.go") || name == "gomad_debug_test.go")
        {
            sources.push((path.clone(), name.to_owned()));
        }
    }
    let manifest = stage_sources(destination, sources, "inspection")?;
    require(
        manifest.as_object().is_some_and(|v| !v.is_empty()),
        "no original inspection source selected",
    )?;
    Ok(manifest)
}
#[cfg(test)]
#[path = "workloads_tests.rs"]
mod tests;
