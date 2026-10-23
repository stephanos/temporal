//! Rust-lineage port of the Go per-test support contract. Raw Result hashes
//! remain unchanged; the support policy removes only argv and the choice
//! runtime identity that authenticated provenance derives from those argv.

mod inspection;
mod inspection_driver;
mod inventory;
mod source;

use super::{
    result::{self, Manifest, QualificationResult},
    spec,
};
use crate::{
    evidence::{
        canonical,
        store::{Opened, open},
    },
    replay::preflight::Configuration,
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::File,
    io::{Read as _, Write as _},
    path::{Component, Path, PathBuf},
};

const SCHEMA: &str = "gomad.rust-wasm-support-list/v1";
const STATEMENT: &str = "Per-test support on exactly this candidate. Only the listed test selections passed every WASM qualification cell gate with the candidate identities recorded here. Every test that is not listed is unsupported on this candidate. This is not a functional-suite, package-level or cross-host claim, and a new candidate starts from an empty list.";
const GATE_ORDER: [&str; 4] = [
    "unchanged-source-preparation",
    "measured-pass-replay-success",
    "measured-pass-replay-canary",
    "storage-transport-inspection",
];
const TAGS: [&str; 4] = [
    "disable_grpc_modules",
    "gomad_wasm_slim",
    "sqlite3_dotlk",
    "test_dep",
];
const MAXIMUM_LIST: u64 = 32 << 20;

#[derive(Clone, Copy)]
struct Selection {
    cell: &'static str,
    package: &'static str,
    function: &'static str,
    run: &'static str,
    directory: &'static str,
}
const SELECTIONS: [Selection; 2] = [
    Selection {
        cell: "onebox/unchanged-workflow-type-smoke",
        package: "./tests",
        function: "TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds",
        run: "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
        directory: "/workspace/tests",
    },
    Selection {
        cell: "onebox/frontend-system-info",
        package: "./tests/gomadfunctional",
        function: "TestFrontendSystemInfo",
        run: "^TestFrontendSystemInfo$",
        directory: "/workspace/tests/gomadfunctional",
    },
];
fn selection(key: &str) -> Result<Selection, String> {
    SELECTIONS
        .into_iter()
        .find(|entry| entry.cell == key)
        .ok_or_else(|| format!("unknown support cell {key}"))
}
fn argv(selection: Selection, canary: bool) -> Vec<String> {
    [
        "gomad-target",
        "-test.v",
        "-test.count=1",
        "-test.timeout=72h",
        &format!("-test.run={}", if canary { "[" } else { selection.run }),
        "-persistenceType=sql",
        "-persistenceDriver=sqlite",
    ]
    .into_iter()
    .map(str::to_owned)
    .collect()
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct FileBinding {
    path: PathBuf,
    sha256: String,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct ProcessBinding {
    result: PathBuf,
    path: PathBuf,
    sha256: String,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Gate {
    cell: String,
    name: String,
    command: Vec<String>,
    exit: i32,
    log: PathBuf,
    log_sha256: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    preparation_module_sha256: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    preparation_receipt: Option<PathBuf>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    preparation_receipt_sha256: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    inspection_receipt: Option<PathBuf>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    inspection_receipt_sha256: Option<String>,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Request {
    executable: FileBinding,
    results: Vec<PathBuf>,
    process_receipts: Vec<ProcessBinding>,
    gates: Vec<Gate>,
    #[serde(default)]
    inventory: Option<inventory::Inventory>,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Candidate {
    integrated_source_sha256: String,
    integrated_source_manifest: String,
    source_root: PathBuf,
    executable: FileBinding,
    host_sha256: String,
    helper_sha256: String,
    model_implementation_owner_sha256: String,
    compiler_sha256: String,
    runtime_inventory_sha256: String,
    inventory: inventory::Inventory,
    runtime_implementation_sha256: String,
    engine_sha256: String,
    host_os: String,
    host_arch: String,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Reference {
    path: PathBuf,
    result_sha256: String,
    receipt_sha256: String,
    process_receipt: ProcessBinding,
    raw_manifest: Manifest,
    executions: u64,
    matched_replays: u64,
    replayable_bugs: u64,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Row {
    package: String,
    function: String,
    run: String,
    cell: String,
    policy: Value,
    policy_sha256: String,
    build_inputs: Value,
    module_sha256: String,
    module_bytes: u64,
    build_tags: Vec<String>,
    workload_source_sha256: String,
    seed: u64,
    repeat: u64,
    success: Reference,
    canary: Reference,
    gates: Vec<Gate>,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct SupportList {
    schema: String,
    statement: String,
    candidate: Candidate,
    rows: Vec<Row>,
}

fn absolute(path: &Path) -> Result<(), String> {
    if !path.is_absolute()
        || path.to_str().is_none_or(|text| {
            text.split('/')
                .skip(1)
                .any(|part| matches!(part, "" | "." | ".."))
        })
        || path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
    {
        return Err("support evidence requires clean absolute paths".into());
    }
    Ok(())
}
fn read(path: &Path, bound: u64) -> Result<Vec<u8>, String> {
    absolute(path)?;
    let parent = open::pin(path.parent().ok_or("support evidence parent missing")?)?;
    open::read(
        &parent,
        path.file_name()
            .and_then(|name| name.to_str())
            .ok_or("support evidence filename missing")?,
        bound,
        false,
    )
}
fn executable_digest(path: &Path) -> Result<String, String> {
    use rustix::fs::{self, FileType, Mode, OFlags};
    absolute(path)?;
    let mut file = File::from(
        fs::open(
            path,
            OFlags::RDONLY | OFlags::NOFOLLOW | OFlags::CLOEXEC | OFlags::NONBLOCK,
            Mode::empty(),
        )
        .map_err(|error| error.to_string())?,
    );
    let before = fs::fstat(&file).map_err(|error| error.to_string())?;
    if FileType::from_raw_mode(before.st_mode) != FileType::RegularFile
        || before.st_size < 0
        || before.st_size > 1 << 30
    {
        return Err("support executable must be a bounded regular file".into());
    }
    let mut bytes = Vec::new();
    (&mut file)
        .take((1 << 30) + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() > 1 << 30
        || !open::same(
            &before,
            &fs::fstat(&file).map_err(|error| error.to_string())?,
        )
    {
        return Err("support executable changed while reading".into());
    }
    Ok(canonical::digest(&bytes))
}

fn bound_file(binding: &FileBinding, bound: u64) -> Result<Vec<u8>, String> {
    let bytes = read(&binding.path, bound)?;
    if !canonical::valid_digest(&binding.sha256) || canonical::digest(&bytes) != binding.sha256 {
        return Err("support evidence file digest changed".into());
    }
    Ok(bytes)
}
fn encoded<T: Serialize>(value: &T) -> Result<Vec<u8>, String> {
    let mut bytes = canonical::encode(value)?;
    bytes.push(b'\n');
    Ok(bytes)
}
fn digest<T: Serialize>(value: &T) -> Result<String, String> {
    Ok(canonical::digest(&canonical::encode(value)?))
}

fn named_pass(output: &[u8], function: &str) -> Result<(), String> {
    let text = std::str::from_utf8(output).map_err(|_| "support stdout is not UTF-8")?;
    let pass = format!("--- PASS: {function} (");
    let skip = format!("--- SKIP: {function} (");
    let mut count = 0;
    for line in text.lines().map(str::trim) {
        if line.starts_with(&skip) {
            return Err("selected support test was skipped".into());
        }
        if let Some(duration) = line
            .strip_prefix(&pass)
            .and_then(|rest| rest.strip_suffix("s)"))
        {
            if duration.is_empty()
                || !duration
                    .bytes()
                    .all(|byte| byte.is_ascii_digit() || byte == b'.')
                || duration.parse::<f64>().is_err()
            {
                return Err("invalid named PASS duration".into());
            }
            count += 1;
        }
    }
    if count != 1 {
        return Err("support requires exactly one selected named PASS per execution".into());
    }
    Ok(())
}

fn row_inputs(result: &QualificationResult) -> Result<Value, String> {
    let mut inputs = result.provenance["inputs"].clone();
    let fields = inputs
        .as_object_mut()
        .ok_or("support build inputs missing")?;
    fields.remove("arguments");
    fields.remove("configuration");
    Ok(inputs)
}

/// A support row's success and canary cells differ only in their
/// arguments and configuration: policy, build inputs, module and workload
/// source must match exactly.
fn matched_pair(
    success: (&QualificationResult, &Value, u64),
    canary: (&QualificationResult, &Value, u64),
) -> Result<(), String> {
    let ((success, success_policy, success_bytes), (canary, canary_policy, canary_bytes)) =
        (success, canary);
    if success_policy != canary_policy
        || row_inputs(success)? != row_inputs(canary)?
        || success.manifest.module_sha256 != canary.manifest.module_sha256
        || success_bytes != canary_bytes
        || success.manifest.workload_source_sha256 != canary.manifest.workload_source_sha256
    {
        return Err("support success/canary policy, source or module differ".into());
    }
    Ok(())
}

fn policy(
    result: &QualificationResult,
    selected: Selection,
    canary: bool,
) -> Result<Value, String> {
    // Result admission has independently derived both removed fields from the
    // complete provenance, including the choice implementation build key.
    let mut configuration: Configuration =
        spec::strict(&canonical::encode(&result.provenance["configuration"])?)?;
    let inputs = &result.provenance["inputs"];
    if configuration.arguments != argv(selected, canary)
        || inputs["kind"] != "go-test"
        || inputs["source"] != selected.package
        || inputs["tags"] != json!(TAGS)
        || inputs["arguments"] != json!(&configuration.arguments[1..])
        || inputs["network"] != Value::Null
        || inputs["soft_assertion_report_only"] != false
        || configuration.network.is_some()
        || configuration.storage.is_some()
        || result.manifest.profile != "cooperative"
        || result.manifest.seed != 7
        || result.manifest.repeat != 2
        || configuration.limits.fuel != 1 << 50
        || configuration.limits.memory_bytes != 4 << 30
        || configuration.limits.output_bytes != 256 << 20
        || configuration.limits.evidence_bytes != 704 << 20
        || configuration.limits.wall_millis != 900_000
        || configuration.wall_nanos != Some(900_000_000_000)
        || !configuration.derive_seed_environment
    {
        return Err(
            "support target argv, package, tags, policy or execution limits changed".into(),
        );
    }
    let model = configuration
        .model
        .as_ref()
        .ok_or("support requires configured model")?;
    let entropy = hex_bytes(&canonical::digest(b"gomad-wasm-prototype/v1"))?;
    let expected_model = json!({"entropy_key":entropy,"working_directory_sha256":canonical::digest(selected.directory.as_bytes()),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"mode":1,"limit":64<<20,"diagnostics":false,"epoch_nanos":946_684_800_000_000_000_u64,"read_step_nanos":0,"now":1,"limits":{"calls":1<<28,"transcript_bytes":1<<30,"pending_events":4096}});
    let expected_environment = BTreeMap::from([
        ("PWD".to_owned(), selected.directory.to_owned()),
        ("TEMPORAL_ROOT".into(), "/workspace".into()),
        ("TMPDIR".into(), "/tmp".into()),
        ("TZ".into(), "UTC".into()),
        ("TEMPORAL_TEST_LOG_LEVEL".into(), "warn".into()),
        ("GOMAD_IO_PROFILE".into(), "wasm".into()),
        (
            "GOMAD_CHOICE_PROFILE".into(),
            "gomad-choice-trace/v4".into(),
        ),
    ]);
    let namespace = configuration
        .namespace
        .as_ref()
        .ok_or("support requires captured schema namespace")?;
    let writable: Vec<Vec<u8>> = ["/workspace", "/tmp", selected.directory]
        .into_iter()
        .map(|path| path.as_bytes().to_vec())
        .collect();
    let mut preopens = writable.clone();
    preopens.push(b"/workspace/schema/sqlite".to_vec());
    preopens.sort();
    preopens.dedup();
    if *model != expected_model
        || configuration.environment != expected_environment
        || namespace.working_directory != selected.directory.as_bytes()
        || namespace.writable_directories != writable
        || namespace.preopens != preopens
        || !namespace.stdin.is_empty()
        || namespace.epoch_nanos != 946_684_800_000_000_000
        || namespace.limits.files != 100_000
        || namespace.limits.descriptors != 4096
        || namespace.limits.filesystem_bytes != 512 << 20
        || namespace.limits.output_bytes != 256 << 20
        || !namespace.entries.iter().any(|entry| {
            entry.path == b"/workspace/schema/sqlite" && entry.directory && entry.readonly
        })
        || namespace.entries.iter().any(|entry| {
            (!entry.directory || !entry.data.is_empty())
                && (!entry.readonly || !entry.path.starts_with(b"/workspace/schema/sqlite/"))
        })
    {
        return Err(
            "support complete qualification model, namespace or environment changed".into(),
        );
    }
    configuration.arguments.clear();
    configuration.runtime_sha256 = None;
    serde_json::to_value(configuration).map_err(|error| error.to_string())
}
fn hex_bytes(text: &str) -> Result<Vec<u8>, String> {
    if !canonical::valid_digest(text) {
        return Err("invalid digest".into());
    }
    text.as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            std::str::from_utf8(pair)
                .map_err(|error| error.to_string())
                .and_then(|pair| u8::from_str_radix(pair, 16).map_err(|error| error.to_string()))
        })
        .collect()
}

fn campaign_plan(
    result: &QualificationResult,
    execution: &super::report::Execution,
) -> Result<(), String> {
    let bytes = read(
        &Path::new(&execution.campaign_path).join("plan.json"),
        MAXIMUM_LIST,
    )?;
    let plan: Value = spec::strict(&bytes)?;
    if canonical::encode(&plan)? != bytes
        || plan["schema"] != "gomad.seed-campaign-plan/v2"
        || plan["selection"] != "7"
        || plan["selection_count"] != 1
        || plan["parallel"] != 1
        || plan["on_failure"] != "all"
        || plan["failure_budget"] != 1
        || plan["overall_nanos"] != 1_800_000_000_000_u64
        || plan["terminate_grace_nanos"] != 1_000_000_000
        || plan["keep_successes"] != "all"
        || plan["success_limit"] != 1
        || plan["success_bytes"] != 2_u64 << 30
        || plan["coverage"] != "choice"
        || plan["configuration"] != result.provenance["configuration"]
        || plan["choice_identity"] != result.provenance["choice_identity"]
        || plan["module_sha256"] != result.manifest.module_sha256
        || plan["toolchain"] != result.manifest.toolchain_sha256
        || plan["host_sha256"] != result.manifest.host_sha256
        || plan["helper_sha256"] != result.manifest.helper_sha256
        || plan["scenario"] != Value::Null
        || plan["network"] != Value::Null
        || plan["soft_assertion_report_only"] != false
    {
        return Err("support retained campaign selection, preparation or limits changed".into());
    }
    Ok(())
}

fn measured(
    result: &QualificationResult,
    selected: Selection,
    canary: bool,
) -> Result<u64, String> {
    let report = &result.report;
    let observations = &result.observations;
    let cost = result
        .cost
        .as_ref()
        .ok_or("support requires measured costs")?;
    let budget = result
        .budget
        .as_ref()
        .ok_or("support requires measured budget")?;
    if result.manifest.provisional
        || !result.manifest.unsupported.is_empty()
        || result.cost_failure.is_some()
        || result.budget_failure.is_some()
        || report.qualified == canary
        || !report.deterministic
        || report.failure.is_some()
        || report.target_success == canary
        || !budget.within_budget
        || !budget.pending.is_empty()
        || !budget.violations.is_empty()
        || observations.operator_process_id == 0
        || observations.start.is_none()
        || observations.prepared.is_none()
        || observations.executed.is_none()
        || observations.preparation_wall_nanos == 0
        || observations.resident_stores != 1
        || report.executions.len() != 2
        || cost.executions != 2
        || cost.replays != 2
        || cost.replayable_bugs != u64::from(canary)
    {
        return Err("support requires final fresh measured deterministic success/canary and complete budget".into());
    }
    let mut module_bytes = None;
    for execution in &report.executions {
        campaign_plan(result, execution)?;
        if !execution.replay.as_ref().is_some_and(|replay| {
            replay.attempted
                && replay.matched
                && replay.divergence.is_none()
                && replay.choice_replay == "exact"
        }) {
            return Err("support requires every own fresh matched replay".into());
        }
        let artifact = Opened::open(Path::new(
            execution
                .artifact_path
                .as_deref()
                .ok_or("support execution artifact missing")?,
        ))?;
        artifact.authenticate_all()?;
        if artifact.manifest().record != execution.record
            || artifact.manifest().record_sha256 != execution.record_sha256
        {
            return Err("support execution artifact record changed".into());
        }
        let module = artifact.payload("target/module.wasm", 1 << 30)?;
        if canonical::digest(&module) != result.manifest.module_sha256
            || module_bytes.is_some_and(|size| size != module.len() as u64)
        {
            return Err("support execution module changed".into());
        }
        module_bytes = Some(module.len() as u64);
        let stdout = artifact.payload("stdout", 256 << 20)?;
        let stream = execution
            .record
            .streams
            .get("stdout")
            .ok_or("support requires retained complete stdout")?;
        if stream.bytes != stdout.len() as u64
            || canonical::digest(&stdout) != stream.sha256
            || stream.bytes >= execution.record.limits.output_bytes
        {
            return Err("support stdout is missing, truncated or capacity-limited".into());
        }
        if canary {
            let stderr = artifact.payload("stderr", 256 << 20)?;
            let stderr_stream = execution
                .record
                .streams
                .get("stderr")
                .ok_or("support canary stderr record binding missing")?;
            if stderr_stream.bytes != stderr.len() as u64
                || stderr_stream.sha256 != canonical::digest(&stderr)
                || stderr_stream.bytes >= execution.record.limits.output_bytes
            {
                return Err("support canary stderr is incomplete".into());
            }
            let witness = String::from_utf8_lossy(&stderr);
            if execution.record.outcome.class != "target_failure"
                || !witness.contains("invalid regexp") && !witness.contains("error parsing regexp")
            {
                return Err("support canary lacks invalid-regexp target-failure witness".into());
            }
        } else {
            named_pass(&stdout, selected.function)?;
        }
    }
    module_bytes.ok_or_else(|| "support module missing".into())
}

struct Admitted {
    result: QualificationResult,
    reference: Reference,
    policy: Value,
    module_bytes: u64,
}
fn admit(
    path: &Path,
    process: ProcessBinding,
    executable: &FileBinding,
) -> Result<Admitted, String> {
    absolute(path)?;
    if process.result != path {
        return Err("support process Result reference changed".into());
    }
    let result = result::open(path)?;
    let (receipt, terminal) = result::open_receipt(path, &result)?;
    if terminal.status != "within" {
        return Err("support terminal receipt budget incomplete".into());
    }
    let key = format!("{}/{}", result.manifest.workload, result.manifest.id);
    let selected = selection(&key)?;
    let canary = result.manifest.cell.as_deref() == Some("canary");
    let module_bytes = measured(&result, selected, canary)?;
    validate_process(&process, &result, executable)?;
    let derived_policy = policy(&result, selected, canary)?;
    let cost = result
        .cost
        .as_ref()
        .ok_or("support measured costs missing")?;
    let reference = Reference {
        path: path.to_path_buf(),
        result_sha256: receipt.result_sha256.clone(),
        receipt_sha256: canonical::digest(&encoded(&receipt)?),
        process_receipt: process,
        raw_manifest: result.manifest.clone(),
        executions: 2,
        matched_replays: 2,
        replayable_bugs: cost.replayable_bugs,
    };
    Ok(Admitted {
        result,
        reference,
        policy: derived_policy,
        module_bytes,
    })
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct StreamBinding {
    path: PathBuf,
    sha256: String,
    bytes: u64,
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct SpecWitness {
    path: PathBuf,
    sha256: String,
    bytes_base64: String,
}

fn specification(
    witness: &SpecWitness,
    result: &QualificationResult,
) -> Result<spec::Spec, String> {
    absolute(&witness.path)?;
    if witness.bytes_base64.len() > (16 << 20) * 4 / 3 + 4 {
        return Err("support qualification spec exceeds capture bound".into());
    }
    let bytes = super::cells::base64(&witness.bytes_base64)
        .filter(|bytes| bytes.len() <= 16 << 20)
        .ok_or("support qualification spec capture is invalid")?;
    if canonical::digest(&bytes) != witness.sha256 {
        return Err("support qualification spec raw hash changed".into());
    }
    let captured: spec::Spec = spec::strict(&bytes)?;
    let declared = &captured.manifest;
    let manifest = &result.manifest;
    let selected = selection(&format!("onebox/{}", manifest.id))?;
    let campaign = &captured.campaign;
    let target = &campaign.target;
    let expected_environment = [
        format!("PWD={}", selected.directory),
        "TEMPORAL_ROOT=/workspace".into(),
        "TMPDIR=/tmp".into(),
        "TZ=UTC".into(),
        "TEMPORAL_TEST_LOG_LEVEL=warn".into(),
    ];
    absolute(&captured.artifact_root)?;
    absolute(Path::new(&campaign.artifacts))?;
    if captured.runtime_workload.is_some()
        || declared.provisional
        || !declared.unsupported.is_empty()
        || declared.id != manifest.id
        || declared.workload != "onebox"
        || declared.cell != manifest.cell.as_deref().unwrap_or_default()
        || (!declared.profile.is_empty() && declared.profile != manifest.profile)
        || declared.seed != manifest.seed
        || declared.repeat != manifest.repeat
        || spec::declared_digest(&declared.integrated_source_sha256)?.as_deref()
            != Some(manifest.integrated_source_sha256.as_str())
        || target.backend != "wasm"
        || target.kind != "go-test"
        || target.source != selected.package
        || target.args != argv(selected, declared.cell == "canary")[1..]
        || target.build_tags != TAGS
        || result.provenance["inputs"]["directory"] != target.working_dir
        || !target.preparation_root.is_empty()
        || target.scenario.is_some()
        || campaign.environment != expected_environment
        || campaign.execution_timeout != 900_000_000_000
        || campaign.overall_timeout != 1_800_000_000_000
        || campaign.terminate_grace != 1_000_000_000
        || campaign.on_failure != "all"
        || campaign.failure_budget != 1
        || campaign.output_limit != 256 << 20
        || campaign.world_transition_limit != 32 << 20
        || campaign.choice_trace_limit != 64 << 20
        || campaign.success_bytes_limit != 2_u64 << 30
        || !Path::new(&campaign.artifacts).starts_with(&captured.artifact_root)
    {
        return Err(
            "support qualification spec selection, identity or campaign envelope changed".into(),
        );
    }
    for (declared, actual) in [
        (
            &declared.workload_source_sha256,
            &manifest.workload_source_sha256,
        ),
        (&declared.compiler_sha256, &manifest.compiler_sha256),
        (&declared.runtime_sha256, &manifest.runtime_sha256),
        (&declared.helper_sha256, &manifest.helper_sha256),
        (&declared.engine_sha256, &manifest.engine_sha256),
        (&declared.model_sha256, &manifest.model_sha256),
        (&declared.host_config_sha256, &manifest.host_config_sha256),
    ] {
        if spec::declared_digest(declared)?.is_some_and(|value| value != *actual) {
            return Err("support declared qualification identity changed".into());
        }
    }
    if (!declared.host_os.is_empty() && declared.host_os != manifest.host_os)
        || (!declared.host_arch.is_empty() && declared.host_arch != manifest.host_arch)
        || (declared.fuel != 0 && declared.fuel != manifest.fuel)
        || (declared.memory_bytes != 0 && declared.memory_bytes != manifest.memory_bytes)
        || Some(&declared.integrated_source_manifest_path)
            != manifest.integrated_source_manifest_path.as_ref()
    {
        return Err("support declared qualification platform or manifest reference changed".into());
    }
    Ok(captured)
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct ProcessReceipt {
    schema: String,
    command: Vec<String>,
    executable: FileBinding,
    operator_process_id: u64,
    exit: i32,
    elapsed_nanos: u64,
    started_nanos: u64,
    ended_nanos: u64,
    cwd: PathBuf,
    environment: BTreeMap<String, Option<String>>,
    environment_sha256: String,
    stdout: StreamBinding,
    stderr: StreamBinding,
    mandatory_workloads: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    private_observations: Option<Value>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    qualification_spec: Option<SpecWitness>,
}
fn validate_process(
    binding: &ProcessBinding,
    result: &QualificationResult,
    executable: &FileBinding,
) -> Result<(), String> {
    let bytes = bound_file(
        &FileBinding {
            path: binding.path.clone(),
            sha256: binding.sha256.clone(),
        },
        32 << 20,
    )?;
    let process: ProcessReceipt = spec::strict(&bytes)?;
    if encoded(&process)? != bytes
        || process.schema != "gomad.rust-onebox-command/v1"
        || process.executable != *executable
        || process.operator_process_id != result.observations.operator_process_id
        || result.report.command.first().map(String::as_str) != Some("gomad")
        || process.command.get(1..) != result.report.command.get(1..)
        || process.command.first().map(String::as_str) != executable.path.to_str()
        || process.exit != i32::from(result.manifest.cell.as_deref() == Some("canary"))
        || process.elapsed_nanos == 0
        || process.started_nanos == 0
        || process.ended_nanos < process.started_nanos
        || !canonical::valid_digest(&process.environment_sha256)
        || process.mandatory_workloads != (result.manifest.id == "unchanged-workflow-type-smoke")
    {
        return Err(
            "support process receipt command, candidate, PID, intent or numeric exit changed"
                .into(),
        );
    }
    let witness = process
        .qualification_spec
        .as_ref()
        .ok_or("support measured qualification requires captured raw specification")?;
    let captured = specification(witness, result)?;
    if process.command.len() != 6
        || process.command[1] != "--installation"
        || absolute(Path::new(&process.command[2])).is_err()
        || process
            .command
            .get(3..)
            .map(|args| args.iter().map(String::as_str).collect::<Vec<_>>())
            != Some(vec![
                "qualify",
                "--spec",
                witness
                    .path
                    .to_str()
                    .ok_or("support spec path is not UTF-8")?,
            ])
        || !binding.result.starts_with(&captured.artifact_root)
        || result.report.executions.iter().any(|execution| {
            !Path::new(&execution.campaign_path).starts_with(&captured.campaign.artifacts)
        })
    {
        return Err("support measured qualification command or output roots changed".into());
    }
    absolute(&process.cwd)?;
    if process
        .environment
        .get("GOMAD_WASM_QUALIFY_WORKLOADS")
        .and_then(Option::as_deref)
        != if process.mandatory_workloads {
            Some("1")
        } else {
            None
        }
    {
        return Err("support mandatory-workload environment intent changed".into());
    }
    for stream in [&process.stdout, &process.stderr] {
        let bytes = bound_file(
            &FileBinding {
                path: stream.path.clone(),
                sha256: stream.sha256.clone(),
            },
            64 << 20,
        )?;
        if bytes.len() as u64 != stream.bytes {
            return Err("support process log byte count changed".into());
        }
    }
    let response: Value = spec::strict(&read(&process.stdout.path, 64 << 20)?)?;
    if response["result_path"] != binding.result.to_string_lossy().as_ref() {
        return Err("support process stdout references another Result".into());
    }
    let publication: result::Receipt = serde_json::from_value(response["operator_receipt"].clone())
        .map_err(|error| error.to_string())?;
    result::validate_receipt(
        result,
        &publication,
        &canonical::digest(&result::encode(result)?),
    )?;
    Ok(())
}

fn linked_candidate(result: &QualificationResult) -> Result<(), String> {
    let m = &result.manifest;
    if m.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || m.helper_sha256 != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || m.engine_sha256
            != canonical::digest(
                gomad_wasmhost::linked::Conditions::compiled()
                    .configuration()
                    .as_bytes(),
            )
        || m.host_os != std::env::consts::OS
        || m.host_arch != std::env::consts::ARCH
    {
        return Err("support receiver does not link the exact candidate implementation".into());
    }
    Ok(())
}

fn candidate(
    result: &QualificationResult,
    executable: &FileBinding,
    retained: Option<&inventory::Inventory>,
) -> Result<Candidate, String> {
    let m = &result.manifest;
    linked_candidate(result)?;
    let inventory = inventory::from_provenance(
        &result.provenance,
        retained.map(|inventory| inventory.overlay.clone()),
    )?;
    if digest(&inventory.raw)? != m.runtime_sha256
        || retained.is_some_and(|retained| *retained != inventory)
    {
        return Err("support raw compiler inventory differs from candidate Result".into());
    }
    let runtime = result.provenance["inputs"]["runtime_implementation"]
        .as_str()
        .filter(|digest| canonical::valid_digest(digest))
        .ok_or("support runtime implementation binding missing")?;
    Ok(Candidate {
        integrated_source_sha256: m.integrated_source_sha256.clone(),
        integrated_source_manifest: result
            .integrated_source_manifest
            .clone()
            .ok_or("support integrated source preimage missing")?,
        source_root: PathBuf::from(
            result.provenance["inputs"]["directory"]
                .as_str()
                .ok_or("support source root missing")?,
        ),
        executable: executable.clone(),
        host_sha256: m.host_sha256.clone(),
        helper_sha256: m.helper_sha256.clone(),
        model_implementation_owner_sha256: m.helper_sha256.clone(),
        compiler_sha256: m.compiler_sha256.clone(),
        runtime_inventory_sha256: m.runtime_sha256.clone(),
        inventory,
        runtime_implementation_sha256: runtime.into(),
        engine_sha256: m.engine_sha256.clone(),
        host_os: m.host_os.clone(),
        host_arch: m.host_arch.clone(),
    })
}

fn validate_stage(gate: &Gate) -> Result<(), String> {
    let bytes = bound_file(
        &FileBinding {
            path: gate.log.clone(),
            sha256: gate.log_sha256.clone(),
        },
        64 << 20,
    )?;
    let stage: Value = spec::strict(&bytes)?;
    let nested = stage["commands"]
        .as_array()
        .ok_or("support stage nested commands missing")?;
    let expected_count = if gate.name == GATE_ORDER[3] { 1 } else { 3 };
    if stage["schema"] != "gomad.rust-onebox-stage/v1"
        || stage["exit"] != 0
        || stage["command"] != json!(gate.command)
        || nested.len() != expected_count
    {
        return Err("support stage command or exit changed".into());
    }
    for (index, value) in nested.iter().enumerate() {
        let process: ProcessReceipt =
            serde_json::from_value(value.clone()).map_err(|error| error.to_string())?;
        let expected_exit = i32::from(gate.name == GATE_ORDER[2] && index == 0);
        let mandatory = gate.cell == SELECTIONS[0].cell
            && [GATE_ORDER[1], GATE_ORDER[2]].contains(&gate.name.as_str())
            && index == 0;
        if process.schema != "gomad.rust-onebox-command/v1"
            || process.exit != expected_exit
            || process.operator_process_id == 0
            || process.started_nanos == 0
            || process.ended_nanos < process.started_nanos
            || process.elapsed_nanos == 0
            || process.command.is_empty()
            || process.mandatory_workloads != mandatory
            || !canonical::valid_digest(&process.environment_sha256)
        {
            return Err(
                "support stage did not preserve actual nested command exits or environment intent"
                    .into(),
            );
        }
        absolute(&process.cwd)?;
        absolute(Path::new(&process.command[0]))?;
        if process
            .environment
            .get("GOMAD_WASM_QUALIFY_WORKLOADS")
            .and_then(Option::as_deref)
            != if process.mandatory_workloads {
                Some("1")
            } else {
                None
            }
        {
            return Err("support stage workload intent differs from its actual environment".into());
        }
        for stream in [&process.stdout, &process.stderr] {
            let bytes = bound_file(
                &FileBinding {
                    path: stream.path.clone(),
                    sha256: stream.sha256.clone(),
                },
                64 << 20,
            )?;
            if bytes.len() as u64 != stream.bytes {
                return Err("support stage log size changed".into());
            }
        }
        if index + 1 == nested.len() && process.command != gate.command {
            return Err("support validated-stage wrapper command differs".into());
        }
    }
    Ok(())
}

fn stage_commands(gate: &Gate) -> Result<Vec<ProcessReceipt>, String> {
    let stage: Value = spec::strict(&bound_file(
        &FileBinding {
            path: gate.log.clone(),
            sha256: gate.log_sha256.clone(),
        },
        64 << 20,
    )?)?;
    serde_json::from_value(stage["commands"].clone()).map_err(|error| error.to_string())
}

fn measured_stage(
    gate: &Gate,
    admitted: &Admitted,
    executable: &FileBinding,
) -> Result<(), String> {
    let commands = stage_commands(gate)?;
    let [measured, report, checked] = commands.as_slice() else {
        return Err("support measured stage requires three commands".into());
    };
    let reference = &admitted.reference;
    let expected_report = vec![
        executable.path.to_string_lossy().into_owned(),
        "report".into(),
        "--result".into(),
        reference.path.to_string_lossy().into_owned(),
    ];
    if canonical::digest(&encoded(measured)?) != reference.process_receipt.sha256
        || report.command != expected_report
        || report.executable != *executable
        || checked
            .command
            .get(1..)
            .map(|args| args.iter().map(String::as_str).collect::<Vec<_>>())
            != Some(vec![
                "check-result",
                reference
                    .path
                    .to_str()
                    .ok_or("support Result path is not UTF-8")?,
            ])
    {
        return Err("support measured stage references another process or Result".into());
    }
    let reported: Value = spec::strict(&read(&report.stdout.path, 64 << 20)?)?;
    let checked: Value = spec::strict(&read(&checked.stdout.path, 64 << 20)?)?;
    if reported["result"]
        != serde_json::to_value(&admitted.result).map_err(|error| error.to_string())?
        || checked["manifest"] != json!(admitted.result.manifest)
        || checked["module_sha256"] != admitted.result.manifest.module_sha256
        || checked["operator_process_id"] != admitted.result.observations.operator_process_id
    {
        return Err("support report/admission stdout does not bind this Result".into());
    }
    Ok(())
}

fn validate_gates(gates: &[Gate], key: &str, module: &str) -> Result<(), String> {
    if gates.len() != 4 {
        return Err("support requires all four ordered gates".into());
    }
    for (gate, name) in gates.iter().zip(GATE_ORDER) {
        if gate.cell != key
            || gate.name != name
            || gate.exit != 0
            || gate.command.is_empty()
            || gate.command.iter().any(String::is_empty)
        {
            return Err("support gate is missing, reordered, duplicate or nonzero".into());
        }
        validate_stage(gate)?;
        if name == GATE_ORDER[0] {
            if gate.preparation_module_sha256.as_deref() != Some(module) {
                return Err("support preparation and measured modules differ".into());
            }
            let path = gate
                .preparation_receipt
                .as_ref()
                .ok_or("support preparation receipt missing")?;
            let sha256 = gate
                .preparation_receipt_sha256
                .as_ref()
                .ok_or("support preparation receipt hash missing")?;
            let (receipt, _) = crate::campaign::preparation::open_receipt(path, sha256)?;
            let commands = stage_commands(gate)?;
            let planned: Value = spec::strict(&read(&commands[1].stdout.path, 64 << 20)?)?;
            if commands[1].command.get(1..) != receipt.command.get(1..)
                || planned["preparation_receipt"] != path.to_string_lossy().as_ref()
                || planned["preparation_receipt_sha256"] != *sha256
                || planned["module_sha256"] != module
                || commands[2]
                    .command
                    .get(1..)
                    .map(|args| args.iter().map(String::as_str).collect::<Vec<_>>())
                    != Some(vec![
                        "prepare-check",
                        path.to_str()
                            .ok_or("support preparation path is not UTF-8")?,
                        sha256,
                    ])
            {
                return Err("support preparation stage references another receipt".into());
            }
            if receipt.guest_executions != 0
                || receipt.exit_code != 0
                || receipt.module_sha256 != module
            {
                return Err(
                    "support preparation must authenticate a zero-execution matching module".into(),
                );
            }
        } else if gate.preparation_module_sha256.is_some()
            || gate.preparation_receipt.is_some()
            || gate.preparation_receipt_sha256.is_some()
        {
            return Err("preparation references belong only to preparation gate".into());
        }
        if name != GATE_ORDER[3]
            && (gate.inspection_receipt.is_some() || gate.inspection_receipt_sha256.is_some())
        {
            return Err("inspection references belong only to inspection gate".into());
        }
    }
    Ok(())
}
#[expect(
    clippy::too_many_lines,
    reason = "The closed two-row transaction authenticates all references before returning a publishable document"
)]
fn build(request: &Request) -> Result<SupportList, String> {
    if request.results.len() != 4 || request.process_receipts.len() != 4 || request.gates.len() != 8
    {
        return Err("support publication requires both complete registered rows".into());
    }
    // The retained operator executable is independently authenticated; process
    // receipts bind every measured Result to this exact executable and PID.
    if executable_digest(&request.executable.path)? != request.executable.sha256 {
        return Err("support operator executable changed".into());
    }
    let mut processes = BTreeSet::new();
    let mut row_roots = BTreeMap::new();
    let mut cells = BTreeMap::new();
    let mut common = None;
    for path in &request.results {
        let bindings: Vec<_> = request
            .process_receipts
            .iter()
            .filter(|binding| &binding.result == path)
            .collect();
        if bindings.len() != 1 {
            return Err("support Result requires one process receipt".into());
        }
        let admitted = admit(path, bindings[0].clone(), &request.executable)?;
        let identity = candidate(
            &admitted.result,
            &request.executable,
            request.inventory.as_ref(),
        )?;
        let configuration: Configuration =
            serde_json::from_value(admitted.result.provenance["configuration"].clone())
                .map_err(|error| error.to_string())?;
        source::schema(
            configuration
                .namespace
                .as_ref()
                .ok_or("support schema namespace missing")?,
            &identity,
        )?;
        if common.as_ref().is_some_and(|common| *common != identity) {
            return Err("support rows belong to different compiled candidates".into());
        }
        common = Some(identity);
        if !processes.insert(admitted.result.observations.operator_process_id) {
            return Err("support cells require distinct fresh operator processes".into());
        }
        let m = &admitted.result.manifest;
        let parent = path.parent().ok_or("support Result parent missing")?;
        let row_key = format!("{}/{}", m.workload, m.id);
        if row_roots
            .insert(parent.to_path_buf(), row_key.clone())
            .is_some_and(|old| old != row_key)
        {
            return Err("support roots may contain only their own registered row".into());
        }
        let pinned = open::pin(parent)?;
        for entry in rustix::fs::Dir::read_from(&pinned).map_err(|error| error.to_string())? {
            let entry = entry.map_err(|error| error.to_string())?;
            let name = entry
                .file_name()
                .to_str()
                .map_err(|_| "support Result root filename is not UTF-8")?;
            if name.starts_with("result-")
                && Path::new(name)
                    .extension()
                    .is_some_and(|extension| extension.eq_ignore_ascii_case("json"))
                && !name.contains(".publication-")
                && !request.results.contains(&parent.join(name))
            {
                return Err("support root contains an unrequested Result".into());
            }
        }
        let key = (
            format!("{}/{}", m.workload, m.id),
            m.cell.clone().ok_or("support mode missing")?,
        );
        if cells.insert(key, admitted).is_some() {
            return Err("duplicate support cell mode".into());
        }
    }
    let mut rows = Vec::new();
    for selected in SELECTIONS {
        let success = cells
            .remove(&(selected.cell.into(), "success".into()))
            .ok_or("support success missing")?;
        let canary = cells
            .remove(&(selected.cell.into(), "canary".into()))
            .ok_or("support canary missing")?;
        let m = &success.result.manifest;
        matched_pair(
            (&success.result, &success.policy, success.module_bytes),
            (&canary.result, &canary.policy, canary.module_bytes),
        )?;
        let gates: Vec<_> = request
            .gates
            .iter()
            .filter(|gate| gate.cell == selected.cell)
            .cloned()
            .collect();
        validate_gates(&gates, selected.cell, &m.module_sha256)?;
        measured_stage(&gates[1], &success, &request.executable)?;
        measured_stage(&gates[2], &canary, &request.executable)?;
        let preparation = &gates[0];
        let preparation_commands = stage_commands(preparation)?;
        if preparation_commands[..2].iter().any(|command| {
            command.executable != request.executable
                || command.command.first().map(String::as_str) != request.executable.path.to_str()
        }) {
            return Err("support preparation commands used another executable".into());
        }
        let (_, prepared) = crate::campaign::preparation::open_receipt(
            preparation
                .preparation_receipt
                .as_ref()
                .ok_or("support preparation missing")?,
            preparation
                .preparation_receipt_sha256
                .as_deref()
                .ok_or("support preparation hash missing")?,
        )?;
        let mut prepared_view = success.result.clone();
        prepared_view.provenance = spec::strict(&prepared.provenance)?;
        if policy(&prepared_view, selected, false)? != success.policy
            || row_inputs(&prepared_view)? != row_inputs(&success.result)?
            || prepared.module.len() as u64 != success.module_bytes
            || candidate(
                &prepared_view,
                &request.executable,
                request.inventory.as_ref(),
            )? != candidate(
                &success.result,
                &request.executable,
                request.inventory.as_ref(),
            )?
        {
            return Err(
                "support preparation policy or implementation differs from measured cell".into(),
            );
        }
        let inspection_gate = &gates[3];
        let inspection_commands = stage_commands(inspection_gate)?;
        let inspection_output: Value =
            spec::strict(&read(&inspection_commands[0].stdout.path, 64 << 20)?)?;
        if inspection_output["receipt"] != json!(inspection_gate.inspection_receipt) {
            return Err("support inspection stage references another receipt".into());
        }
        inspection::validate(
            inspection_gate
                .inspection_receipt
                .as_deref()
                .ok_or("support inspection receipt missing")?,
            inspection_gate
                .inspection_receipt_sha256
                .as_deref()
                .ok_or("support inspection receipt hash missing")?,
            common.as_ref().ok_or("support candidate missing")?,
        )?;
        rows.push(Row {
            package: selected.package.into(),
            function: selected.function.into(),
            run: selected.run.into(),
            cell: selected.cell.into(),
            policy_sha256: digest(&success.policy)?,
            build_inputs: row_inputs(&success.result)?,
            policy: success.policy,
            module_sha256: m.module_sha256.clone(),
            module_bytes: success.module_bytes,
            build_tags: TAGS.into_iter().map(str::to_owned).collect(),
            workload_source_sha256: m.workload_source_sha256.clone(),
            seed: m.seed,
            repeat: m.repeat,
            success: success.reference,
            canary: canary.reference,
            gates,
        });
    }
    rows.sort_by(|left, right| {
        (&left.package, &left.function).cmp(&(&right.package, &right.function))
    });
    Ok(SupportList {
        schema: SCHEMA.into(),
        statement: STATEMENT.into(),
        candidate: common.ok_or("support candidate missing")?,
        rows,
    })
}
fn validate(list: &SupportList) -> Result<(), String> {
    let request = Request {
        executable: list.candidate.executable.clone(),
        inventory: Some(list.candidate.inventory.clone()),
        results: list
            .rows
            .iter()
            .flat_map(|row| [row.success.path.clone(), row.canary.path.clone()])
            .collect(),
        process_receipts: list
            .rows
            .iter()
            .flat_map(|row| {
                [
                    row.success.process_receipt.clone(),
                    row.canary.process_receipt.clone(),
                ]
            })
            .collect(),
        gates: list.rows.iter().flat_map(|row| row.gates.clone()).collect(),
    };
    if build(&request)? != *list {
        return Err(
            "retained support list differs from independently authenticated evidence".into(),
        );
    }
    Ok(())
}
fn markdown(list: &SupportList) -> String {
    use std::fmt::Write as _;
    let mut out = format!(
        "# WASM per-test support list ({})\n\n{}\n\nCandidate source `{}`, executable `{}`, model implementation owner `{}`.\n\n| Package | Test | Selection | Module | Success Result | Canary Result |\n| --- | --- | --- | --- | --- | --- |\n",
        list.schema,
        list.statement,
        list.candidate.integrated_source_sha256,
        list.candidate.executable.sha256,
        list.candidate.model_implementation_owner_sha256
    );
    for row in &list.rows {
        let _ = writeln!(
            out,
            "| `{}` | `{}` | `{}` | `{}` | `{}` | `{}` |",
            row.package,
            row.function,
            row.run,
            row.module_sha256,
            row.success.result_sha256,
            row.canary.result_sha256
        );
    }
    for row in &list.rows {
        let _ = writeln!(
            out,
            "\n### `{}`\n\nSeed {}, repeats {}, module {} B, tags `{}`; source `{}`, derived policy `{}`.\n\n| Mode | Retained Result | Raw configured model | Raw configuration | Raw toolchain |\n| --- | --- | --- | --- | --- |",
            row.function,
            row.seed,
            row.repeat,
            row.module_bytes,
            row.build_tags.join(","),
            row.workload_source_sha256,
            row.policy_sha256
        );
        for (mode, reference) in [("success", &row.success), ("canary", &row.canary)] {
            let _ = writeln!(
                out,
                "| {} | `{}` | `{}` | `{}` | `{}` |",
                mode,
                markdown_cell(&reference.path.to_string_lossy()),
                reference.raw_manifest.model_sha256,
                reference.raw_manifest.host_config_sha256,
                reference.raw_manifest.toolchain_sha256
            );
        }
        out.push_str("\n| Gate | Exit | Command | Log |\n| --- | --- | --- | --- |\n");
        for gate in &row.gates {
            let _ = writeln!(
                out,
                "| {} | {} | `{}` | `{}` |",
                gate.name,
                gate.exit,
                markdown_cell(&format!("{:?}", gate.command)),
                markdown_cell(&gate.log.to_string_lossy())
            );
        }
    }
    out
}
fn markdown_cell(text: &str) -> String {
    text.replace('|', "\\|")
        .replace('`', "'")
        .replace('\n', " ")
}
fn write_new(path: &Path, bytes: &[u8]) -> Result<(), String> {
    absolute(path)?;
    let parent = open::pin(path.parent().ok_or("support output parent missing")?)?;
    let name = path.file_name().ok_or("support output filename missing")?;
    let mut file = File::from(
        rustix::fs::openat(
            &parent,
            name,
            rustix::fs::OFlags::WRONLY
                | rustix::fs::OFlags::CREATE
                | rustix::fs::OFlags::EXCL
                | rustix::fs::OFlags::NOFOLLOW
                | rustix::fs::OFlags::CLOEXEC,
            rustix::fs::Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .and_then(|()| parent.sync_all())
        .map_err(|error| error.to_string())
}

/// Bounded repository qualification harness, linked to the ordinary release
/// library. This is not a public Gomad command.
///
/// # Errors
/// Refuses incomplete gates, changed evidence and unsupported selections.
#[expect(
    clippy::too_many_lines,
    reason = "The bounded repository harness keeps its closed command grammar together"
)]
pub fn run(arguments: &[String], output: &mut dyn std::io::Write) -> Result<(), String> {
    super::require_release_role()?;
    let response = match arguments {
        [
            command,
            receipt,
            receipt_sha,
            result_path,
            executable,
            executable_sha,
        ] if command == "validate-inspection" => {
            let result = result::open(Path::new(result_path))?;
            let (_, terminal) = result::open_receipt(Path::new(result_path), &result)?;
            if result.manifest.provisional
                || !result.report.deterministic
                || result.report.failure.is_some()
                || result.cost.is_none()
                || result.cost_failure.is_some()
                || result.budget_failure.is_some()
                || !result.budget.as_ref().is_some_and(|budget| {
                    budget.within_budget
                        && budget.pending.is_empty()
                        && budget.violations.is_empty()
                })
                || terminal.status != "within"
            {
                return Err("inspection candidate requires a final measured Result".into());
            }
            let executable = FileBinding {
                path: PathBuf::from(executable),
                sha256: executable_sha.clone(),
            };
            if executable_digest(&executable.path)? != executable.sha256 {
                return Err("inspection candidate executable changed".into());
            }
            let receipt_value: Value = spec::strict(&bound_file(
                &FileBinding {
                    path: PathBuf::from(receipt),
                    sha256: receipt_sha.clone(),
                },
                MAXIMUM_LIST,
            )?)?;
            let alias: inventory::Overlay =
                serde_json::from_value(receipt_value["compiler_overlay"].clone())
                    .map_err(|error| error.to_string())?;
            let private = receipt_value["private_inputs_detached"]
                .as_str()
                .and_then(|path| path.strip_suffix("-unavailable"))
                .ok_or("inspection private alias reference missing")?;
            let original_inventory = inventory::from_alias(
                &result.provenance,
                &alias,
                &Path::new(private).join("compiler"),
            )?;
            let candidate = candidate(&result, &executable, Some(&original_inventory))?;
            inspection::validate(Path::new(receipt), receipt_sha, &candidate)?;
            json!({"validated":true,"receipt_sha256":receipt_sha,"candidate_source_sha256":candidate.integrated_source_sha256})
        }
        [
            command,
            receipt,
            receipt_sha,
            driver,
            driver_sha,
            executable,
            executable_sha,
        ] if command == "validate-inspection-driver" => inspection_driver::validate(
            &FileBinding {
                path: PathBuf::from(receipt),
                sha256: receipt_sha.clone(),
            },
            &FileBinding {
                path: PathBuf::from(driver),
                sha256: driver_sha.clone(),
            },
            &FileBinding {
                path: PathBuf::from(executable),
                sha256: executable_sha.clone(),
            },
        )?,
        [command, root, package, policy, path] if command == "temporal-policy" => {
            let policy = match policy.as_str() {
                "qualification" => crate::install::config::TemporalPolicy::Qualification,
                "inspection" => crate::install::config::TemporalPolicy::Inspection,
                _ => return Err("unsupported support policy".into()),
            };
            let config = crate::install::config::temporal_policy(Path::new(root), package, policy)?;
            write_new(Path::new(path), &encoded(&config)?)?;
            json!({"path":path,"sha256":canonical::digest(&encoded(&config)?)})
        }
        [command, path, sha256] if command == "prepare-check" => {
            let (receipt, prepared) =
                crate::campaign::preparation::open_receipt(Path::new(path), sha256)?;
            if receipt.guest_executions != 0 || receipt.exit_code != 0 {
                return Err("support prepare-check requires zero guest executions".into());
            }
            json!({"module_sha256":receipt.module_sha256,"module_bytes":prepared.module.len(),"guest_executions":receipt.guest_executions})
        }
        [command, path] if command == "check-result" => {
            let result = result::open(Path::new(path))?;
            let (_, terminal) = result::open_receipt(Path::new(path), &result)?;
            let selected = selection(&format!(
                "{}/{}",
                result.manifest.workload, result.manifest.id
            ))?;
            linked_candidate(&result)?;
            let canary = result.manifest.cell.as_deref() == Some("canary");
            policy(&result, selected, canary)?;
            let module_bytes = measured(&result, selected, canary)?;
            if terminal.status != "within" {
                return Err("support terminal receipt budget incomplete".into());
            }
            json!({"cell":selected.cell,"mode":result.manifest.cell,"module_sha256":result.manifest.module_sha256,"operator_process_id":result.observations.operator_process_id,"manifest":result.manifest,"module_bytes":module_bytes})
        }
        [command, path, base] if command == "publish" => {
            let request: Request = spec::strict(&read(Path::new(path), MAXIMUM_LIST)?)?;
            let list = build(&request)?;
            validate(&list)?;
            let base = Path::new(base);
            absolute(base)?;
            if base.extension().is_some() {
                return Err("support output requires extensionless base".into());
            }
            let json_path = base.with_extension("json");
            let md_path = base.with_extension("md");
            write_new(&json_path, &encoded(&list)?)?;
            write_new(&md_path, markdown(&list).as_bytes())?;
            json!({"json":json_path,"markdown":md_path})
        }
        [command, path] if command == "validate" => {
            let bytes = read(Path::new(path), MAXIMUM_LIST)?;
            let list: SupportList = spec::strict(&bytes)?;
            if encoded(&list)? != bytes {
                return Err("support list is not canonical".into());
            }
            validate(&list)?;
            json!({"schema":list.schema,"rows":list.rows.len(),"validated":true})
        }
        _ => {
            return Err(
                "usage: fn222-support check-result RESULT | publish REQUEST BASE | validate LIST"
                    .into(),
            );
        }
    };
    output
        .write_all(&encoded(&response)?)
        .map_err(|error| error.to_string())
}

#[cfg(test)]
mod tests;
