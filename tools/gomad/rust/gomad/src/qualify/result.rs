//! Retained qualification `Result` and `PublicationReceipt` in the Rust lineage.
//! Both are canonical JSON, content addressed and published without replace;
//! the receipt is `result-SHA256.json.publication-SHA256.json`.
//!
//! Receipts and Results are content addressed, not signed. `report` refuses any
//! edit that is not consistently re-hashed and recomputes every verdict, cost
//! and budget input from the retained records, configuration and raw
//! observations. A writer who rewrites records, Result and receipt together
//! can still produce a self-consistent forgery of identity fields that do not
//! feed the verdict; that is outside this format's threat model.

use super::{
    cost::{self, Budget, Cost, Observations},
    report::{self, Report},
    usage::Usage,
};
use crate::evidence::{
    canonical,
    store::{durable_directory, open},
};
use rustix::fs::{self as fsx, Mode, OFlags};
use serde::{Deserialize, Serialize};
use std::{
    fs::File,
    io::Write as _,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

pub(crate) const MANIFEST_SCHEMA: &str = "gomad.rust-qualification-manifest/v1";
pub(crate) const RESULT_SCHEMA: &str = "gomad.rust-qualification-result/v1";
pub(crate) const RECEIPT_SCHEMA: &str = "gomad.rust-qualification-receipt/v1";
const DIRECTORY: [&str; 2] = ["rust-qualifications", "v1"];
const MAXIMUM_RESULT_BYTES: u64 = 64 << 20;
const MAXIMUM_RECEIPT_BYTES: u64 = 8 << 10;
const COOPERATIVE_PROFILE: &str = "gomad.wasi-cooperative/v1";
/// The build-input fields, in canonical (sorted) order.
const INPUT_FIELDS: [&str; 22] = [
    "arguments",
    "build_flags",
    "compiler",
    "compiler_path",
    "compiler_sha256",
    "configuration",
    "directory",
    "environment",
    "go_version",
    "helper",
    "host",
    "kind",
    "module_metadata",
    "network",
    "runtime",
    "runtime_implementation",
    "scenario",
    "schema",
    "soft_assertion_report_only",
    "source",
    "sources",
    "tags",
];
/// The provenance fields, in canonical (sorted) order.
const PROVENANCE_FIELDS: [&str; 9] = [
    "build_key",
    "choice_identity",
    "configuration",
    "inputs",
    "module_metadata",
    "module_sha256",
    "network",
    "scenario",
    "schema",
];
const CONTROLLED_PROFILE: &str = "gomad.wasi-stock-exploratory/v1";

/// The manifest profile name of an installed runtime profile; any other
/// runtime profile has no admitted qualification owner.
pub(crate) fn profile_name(runtime: &str) -> Result<&'static str, String> {
    match runtime {
        COOPERATIVE_PROFILE => Ok("cooperative"),
        CONTROLLED_PROFILE => Ok("controlled"),
        _ => Err(format!(
            "unsupported WASM profile {runtime:?}: no admitted backend owner"
        )),
    }
}

/// The manifest identities derived from the retained provenance; freeze and
/// `report` both bind the manifest to them.
struct Derived {
    pub workload_source_sha256: String,
    pub runtime_sha256: String,
    pub compiler_sha256: String,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub model_sha256: String,
    pub engine_sha256: String,
    pub host_config_sha256: String,
    pub module_sha256: String,
    pub toolchain_sha256: String,
    pub scenario_sha256: Option<String>,
    pub profile: &'static str,
    pub fuel: u64,
    pub memory_bytes: u64,
}

fn derive(provenance: &serde_json::Value) -> Result<Derived, String> {
    use serde_json::Value;
    let refuse = || "retained provenance does not bind the qualification identities".to_owned();
    let digest = |value: &Value| canonical::encode(value).map(|bytes| canonical::digest(&bytes));
    let text = |value: &Value| value.as_str().map(str::to_owned).ok_or_else(refuse);
    if provenance["schema"] != "gomad.rust-wasm-provenance/v1"
        || !provenance.as_object().is_some_and(|object| {
            object
                .keys()
                .map(String::as_str)
                .filter(|field| *field != "runtime_workload")
                .eq(PROVENANCE_FIELDS)
        })
    {
        return Err(refuse());
    }
    // Every typed level is exact, and the build inputs carry exactly their
    // declared fields.
    crate::campaign::prepare::typed_provenance(provenance).map_err(|_| refuse())?;
    let inputs = &provenance["inputs"];
    if !inputs
        .as_object()
        .is_some_and(|object| object.keys().map(String::as_str).eq(INPUT_FIELDS))
    {
        return Err(refuse());
    }
    if !Path::new(&text(&inputs["compiler_path"])?).is_absolute()
        || inputs["module_metadata"] != provenance["module_metadata"]
    {
        return Err(refuse());
    }
    let configuration = &provenance["configuration"];
    let toolchain = digest(inputs)?;
    if text(&provenance["build_key"])? != toolchain {
        return Err(refuse());
    }
    // The duplicated fields agree with the hashed build inputs, as the
    // `exec` opener requires: scenario, network and the configuration whose
    // only post-build field is the cooperative runtime implementation.
    let mut built = configuration.clone();
    if let Some(object) = built.as_object_mut() {
        object.insert("runtime_sha256".into(), Value::Null);
    }
    let cooperative = configuration["profile"] == COOPERATIVE_PROFILE;
    let choice = &provenance["choice_identity"];
    // The cooperative runtime implementation is the choice identity's; a
    // controlled configuration carries none.
    // The choice implementation is a pure function of the build key and the
    // choice sources compiled into this lineage, so it is recomputed.
    let runtime = if cooperative {
        let expected = hex(&crate::campaign::prepare::choice_implementation(
            &toolchain,
        )?);
        serde_json::from_value::<[u8; 32]>(choice["ImplementationSHA256"].clone())
            .ok()
            .map(|implementation| hex(&implementation))
            .filter(|implementation| *implementation == expected)
            .map(Value::String)
    } else {
        Some(Value::Null)
    };
    if inputs["schema"] != "gomad.rust-wasm-build-inputs/v1"
        || runtime.as_ref() != Some(&configuration["runtime_sha256"])
        || !choice.is_null() && (choice["GOOS"] != "wasip1" || choice["GOARCH"] != "wasm")
        || inputs["scenario"] != provenance["scenario"]
        || inputs["network"] != provenance["network"]
        || inputs["configuration"] != built
        || cooperative == choice.is_null()
        || !choice.is_null()
            && (choice["ToolchainBuildKey"] != provenance["build_key"]
                || serde_json::from_value::<[u8; 32]>(choice["TargetSHA256"].clone())
                    .ok()
                    .map(|target| hex(&target))
                    .as_deref()
                    != provenance["module_sha256"].as_str())
    {
        return Err(refuse());
    }
    let scenario = match &provenance["scenario"] {
        Value::Null => None,
        bytes => Some(canonical::digest(
            &serde_json::from_value::<Vec<u8>>(bytes.clone()).map_err(|_| refuse())?,
        )),
    };
    Ok(Derived {
        workload_source_sha256: digest(&inputs["sources"])?,
        runtime_sha256: digest(&inputs["compiler"])?,
        compiler_sha256: text(&inputs["compiler_sha256"])?,
        helper_sha256: text(&inputs["helper"])?,
        host_sha256: text(&inputs["host"])?,
        model_sha256: digest(configuration.get("model").unwrap_or(&Value::Null))?,
        engine_sha256: canonical::digest(text(&configuration["engine"])?.as_bytes()),
        host_config_sha256: digest(configuration)?,
        module_sha256: text(&provenance["module_sha256"])?,
        toolchain_sha256: toolchain,
        scenario_sha256: scenario,
        profile: profile_name(configuration["profile"].as_str().ok_or_else(refuse)?)?,
        fuel: configuration["limits"]["fuel"]
            .as_u64()
            .ok_or_else(refuse)?,
        memory_bytes: configuration["limits"]["memory_bytes"]
            .as_u64()
            .ok_or_else(refuse)?,
    })
}

fn hex(bytes: &[u8]) -> String {
    use std::fmt::Write as _;
    bytes.iter().fold(String::new(), |mut out, byte| {
        let _ = write!(out, "{byte:02x}");
        out
    })
}

/// A standalone record ran exactly the retained configuration's limits,
/// arguments and environment, as replay preflight requires.
fn standalone_inputs(
    record: &crate::evidence::record::Record,
    configuration: &serde_json::Value,
) -> bool {
    let observed = serde_json::json!({
        "limits": record.limits,
        "arguments": record.arguments,
        "environment": record.environment,
    });
    ["limits", "arguments", "environment"]
        .iter()
        .all(|field| observed[field] == configuration[field])
}

/// A cluster execution ran the retained configuration's limits, except that
/// its evidence bound is exactly the smaller of the configuration's and the
/// retained scenario's `scenario_evidence_bytes`.
fn cluster_limits(
    limits: &crate::evidence::record::Limits,
    provenance: &serde_json::Value,
) -> bool {
    let retained = &provenance["configuration"]["limits"];
    let scenario = serde_json::from_value::<Vec<u8>>(provenance["scenario"].clone())
        .ok()
        .and_then(|bytes| crate::campaign::scenario::evidence_limit(&bytes).ok());
    let mut expected = retained.clone();
    match (retained["evidence_bytes"].as_u64(), scenario) {
        (Some(bound), Some(scenario)) => {
            expected["evidence_bytes"] = bound.min(scenario).into();
        }
        _ => return false,
    }
    serde_json::json!(limits) == expected
}

/// The Go owner's trace bytes an execution record retains: each store's
/// transcript, choice, diagnostic and storage streams plus the network trace,
/// which a cluster shares once across its incarnations.
fn recorded_trace_bytes(record: &crate::evidence::record::Record) -> Result<u64, String> {
    record
        .streams
        .iter()
        .filter(|(name, _)| {
            let Some(rest) = name.strip_prefix("evidence/") else {
                return false;
            };
            let stream = match rest.strip_prefix("cluster/") {
                Some(store) => match store.splitn(3, '/').nth(2) {
                    Some(stream) if stream != "network" => stream,
                    _ => return false,
                },
                None => rest,
            };
            crate::campaign::execution::TRACE_STREAMS.contains(&stream)
        })
        .map(|(_, stream)| stream.bytes)
        .try_fold(0_u64, u64::checked_add)
        .ok_or_else(|| "qualification trace bytes overflow".into())
}

/// Refuses a manifest whose identities differ from its retained provenance.
pub(crate) fn bind_provenance(
    manifest: &Manifest,
    provenance: &serde_json::Value,
) -> Result<(), String> {
    let derived = derive(provenance)?;
    let digests = [
        (
            &derived.workload_source_sha256,
            &manifest.workload_source_sha256,
        ),
        (&derived.runtime_sha256, &manifest.runtime_sha256),
        (&derived.compiler_sha256, &manifest.compiler_sha256),
        (&derived.helper_sha256, &manifest.helper_sha256),
        (&derived.host_sha256, &manifest.host_sha256),
        (&derived.model_sha256, &manifest.model_sha256),
        (&derived.engine_sha256, &manifest.engine_sha256),
        (&derived.host_config_sha256, &manifest.host_config_sha256),
        (&derived.module_sha256, &manifest.module_sha256),
        (&derived.toolchain_sha256, &manifest.toolchain_sha256),
    ];
    if digests.iter().any(|(derived, frozen)| derived != frozen)
        || derived.scenario_sha256 != manifest.scenario_sha256
        || derived.profile != manifest.profile
        || (derived.fuel, derived.memory_bytes) != (manifest.fuel, manifest.memory_bytes)
    {
        return Err("qualification manifest identity differs from its retained provenance".into());
    }
    Ok(())
}

/// A frozen comparison cell, not an assertion that it passed. Every identity
/// is a Rust-lineage digest observed during freeze.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Manifest {
    pub schema: String,
    pub provisional: bool,
    pub integrated_source_manifest_path: Option<String>,
    pub id: String,
    pub workload: String,
    pub profile: String,
    pub cell: Option<String>,
    pub integrated_source_sha256: String,
    pub workload_source_sha256: String,
    pub runtime_sha256: String,
    pub compiler_sha256: String,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub model_sha256: String,
    pub engine_sha256: String,
    pub host_config_sha256: String,
    pub module_sha256: String,
    pub toolchain_sha256: String,
    /// The normalized multi-instance scenario of a cluster cell. Its records
    /// bind a per-execution cluster configuration instead of `host_config_sha256`.
    pub scenario_sha256: Option<String>,
    pub host_os: String,
    pub host_arch: String,
    pub seed: u64,
    pub repeat: u64,
    pub fuel: u64,
    pub memory_bytes: u64,
    pub unsupported: Vec<String>,
}

impl Manifest {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if self
            .cell
            .as_deref()
            .is_some_and(|cell| !matches!(cell, "success" | "canary"))
            || !self.provisional && self.cell.is_none()
        {
            return Err("final qualification requires explicit success/canary cell".into());
        }
        if !self.provisional
            && !self
                .integrated_source_manifest_path
                .as_deref()
                .is_some_and(|path| Path::new(path).is_absolute())
        {
            return Err(
                "final WASM qualification requires conductor integrated-source manifest reference"
                    .into(),
            );
        }
        if self.schema != MANIFEST_SCHEMA
            || self.id.is_empty()
            || self.host_os.is_empty()
            || self.host_arch.is_empty()
            || !(2..=report::MAXIMUM_REPEAT).contains(&self.repeat)
            || self.fuel == 0
            || self.memory_bytes == 0
        {
            return Err("frozen WASM identity, limits and 2..32 repeats are required".into());
        }
        for digest in [
            &self.integrated_source_sha256,
            &self.workload_source_sha256,
            &self.runtime_sha256,
            &self.compiler_sha256,
            &self.helper_sha256,
            &self.host_sha256,
            &self.model_sha256,
            &self.engine_sha256,
            &self.host_config_sha256,
            &self.module_sha256,
            &self.toolchain_sha256,
        ] {
            if !canonical::valid_digest(digest) {
                return Err("unbound WASM qualification identity".into());
            }
        }
        if self
            .scenario_sha256
            .as_deref()
            .is_some_and(|digest| !canonical::valid_digest(digest))
            || self.scenario_sha256.is_some()
                != (self.workload == "multi-instance"
                    || (self.workload == "sqlite" && self.id == "sqlite-durable-recovery"))
        {
            return Err("cluster workload cells require their frozen scenario identity".into());
        }
        if !matches!(
            self.workload.as_str(),
            "fixture" | "sqlite" | "onebox" | "multi-instance"
        ) {
            return Err("unsupported WASM qualification workload".into());
        }
        if !matches!(self.profile.as_str(), "controlled" | "cooperative") {
            return Err(format!(
                "unsupported WASM profile {:?}: no admitted backend owner",
                self.profile
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct QualificationResult {
    pub schema: String,
    pub manifest: Manifest,
    pub report: Report,
    pub observations: Observations,
    pub cost: Option<Cost>,
    pub cost_failure: Option<String>,
    pub budget: Option<Budget>,
    pub budget_failure: Option<String>,
    pub integrated_source_manifest: Option<String>,
    /// The prepared target's `gomad.rust-wasm-provenance/v1` preimage: build
    /// inputs, build key, module digest, final configuration and scenario.
    /// Every manifest identity is recomputed from it.
    pub provenance: serde_json::Value,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub native_crash_catalogues: Vec<super::workloads::native_crash::Catalogue>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub native_crash_selection: Option<super::workloads::native_crash::Selection>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Receipt {
    pub schema: String,
    pub result_sha256: String,
    pub operator_process_id: u64,
    pub linked_after: Option<Usage>,
    /// The operator-phase high water through publication, when attributed.
    #[serde(default)]
    pub operator_peak_rss_bytes: Option<u64>,
}

/// The conductor's sha256sum manifest: lowercase hex, two spaces, absolute
/// path. With `check_files` every listed file is re-read and authenticated.
pub(crate) fn validate_source_manifest(encoded: &[u8], check_files: bool) -> Result<(), String> {
    if encoded.is_empty() || encoded.len() > 16 << 20 {
        return Err("bounded nonempty integrated source manifest required".into());
    }
    let text = std::str::from_utf8(encoded)
        .map_err(|_| "malformed integrated source manifest line".to_owned())?;
    let lines = text
        .strip_suffix('\n')
        .unwrap_or(text)
        .split('\n')
        .collect::<Vec<_>>();
    if lines.len() > 20_000 {
        return Err("integrated source manifest entry limit".into());
    }
    let mut seen = std::collections::BTreeSet::new();
    let mut total = 0_u64;
    for line in lines {
        // Bytes 64 and 65 are ASCII spaces, so both are character boundaries.
        if line.len() < 67 || line.as_bytes()[64..66] != *b"  " {
            return Err("malformed integrated source manifest line".into());
        }
        let (digest, path) = (&line[..64], &line[66..]);
        if !canonical::valid_digest(digest) {
            return Err("malformed integrated source manifest digest".into());
        }
        let parsed = Path::new(path);
        if !parsed.is_absolute()
            || path.contains(['\0', '\r'])
            || parsed.components().any(|part| {
                matches!(
                    part,
                    std::path::Component::CurDir | std::path::Component::ParentDir
                )
            })
            || path.ends_with('/')
            || path.contains("//")
            || !seen.insert(path)
        {
            return Err("duplicate or malformed integrated source manifest path".into());
        }
        if check_files {
            let data = super::spec::read_bounded(parsed, 128 << 20)?;
            total += data.len() as u64;
            if total > 512 << 20 {
                return Err("integrated source inventory byte limit".into());
            }
            if canonical::digest(&data) != digest {
                return Err("integrated source manifest file identity changed".into());
            }
        }
    }
    Ok(())
}

fn authenticate_provenance_module(
    provenance: &serde_json::Value,
    module: &[u8],
) -> Result<(), String> {
    let metadata: gomad_wasmhost::linked::ModuleMetadata =
        serde_json::from_value(provenance["module_metadata"].clone())
            .map_err(|error| format!("qualification module metadata: {error}"))?;
    crate::campaign::prepare::authenticate_module_metadata(
        Some(&metadata),
        &provenance["inputs"],
        module,
    )
}

fn authenticate_artifacts(result: &QualificationResult) -> Result<(), String> {
    for execution in &result.report.executions {
        if let Some(path) = &execution.artifact_path {
            let opened = crate::evidence::store::Opened::open(Path::new(path))?;
            opened.authenticate_all()?;
            if opened.manifest().record != execution.record {
                return Err("qualification raw artifact record differs from Result".into());
            }
            authenticate_provenance_module(
                &result.provenance,
                &opened.payload("target/module.wasm", 512 << 20)?,
            )?;
        } else if result.report.failure.is_none() {
            return Err("qualification module authentication requires retained artifact".into());
        }
    }
    Ok(())
}

/// Installation-free validation: recomputes report verdicts, costs and budget
/// from retained raw observations and refuses any claim that differs.
pub(crate) fn validate(result: &QualificationResult) -> Result<(), String> {
    if result.schema != RESULT_SCHEMA {
        return Err("unsupported qualification result schema".into());
    }
    let manifest = &result.manifest;
    manifest.validate()?;
    if !manifest.provisional && result.observations.operator_process_id == 0 {
        return Err("final result requires actual operator process identity".into());
    }
    match &result.integrated_source_manifest {
        Some(encoded) => {
            if canonical::digest(encoded.as_bytes()) != manifest.integrated_source_sha256 {
                return Err("retained integrated source manifest identity changed".into());
            }
            validate_source_manifest(encoded.as_bytes(), false)?;
        }
        None if !manifest.provisional => {
            return Err("final result omitted integrated source manifest".into());
        }
        None => {}
    }
    report::validate(&result.report)?;
    if result.report.seed != manifest.seed || result.report.repeat != manifest.repeat {
        return Err("retained qualification report differs from frozen sample selection".into());
    }
    // The record's implementation and platform digests are recomputed from
    // the frozen helper/host identities and host platform, exactly as the
    // runner derives them (`replay::preflight::{implementation, platform}`).
    let implementation = canonical::digest(&canonical::encode(&serde_json::json!({
        "host": manifest.host_sha256, "helper": manifest.helper_sha256
    }))?);
    let platform =
        canonical::digest(format!("{}/{}", manifest.host_os, manifest.host_arch).as_bytes());
    bind_provenance(manifest, &result.provenance)?;
    authenticate_artifacts(result)?;
    for execution in &result.report.executions {
        let identity = &execution.record.identity;
        if identity.target != manifest.module_sha256
            || identity.toolchain != manifest.toolchain_sha256
            || identity.implementation != implementation
            || identity.platform != platform
            || profile_name(&execution.record.profile).ok() != Some(manifest.profile.as_str())
            || manifest.scenario_sha256.is_none()
                && (identity.configuration != manifest.host_config_sha256
                    || !standalone_inputs(&execution.record, &result.provenance["configuration"]))
            || manifest.scenario_sha256.is_some()
                && !cluster_limits(&execution.record.limits, &result.provenance)
            || execution.record.limits.fuel != manifest.fuel
            || execution.record.limits.memory_bytes != manifest.memory_bytes
        {
            return Err("qualification execution identity differs from the frozen manifest".into());
        }
    }
    bind_cluster_handles(result)?;
    let (catalogues, selection) = super::workloads::retained(result)?;
    if catalogues != result.native_crash_catalogues || selection != result.native_crash_selection {
        return Err(
            "retained native crash projection differs from authenticated workload records".into(),
        );
    }
    // Each execution's samples, and each matched replay's, measured exactly
    // the trace bytes its record retains.
    for execution in &result.report.executions {
        let recorded = recorded_trace_bytes(&execution.record)?;
        let matched = execution
            .replay
            .as_ref()
            .is_some_and(|replay| replay.matched);
        for phase in ["execute", "replay"] {
            let measured = result
                .observations
                .samples
                .iter()
                .filter(|sample| sample.phase == phase && sample.iteration == execution.iteration)
                .map(|sample| sample.trace_bytes)
                .try_fold(0_u64, u64::checked_add)
                .ok_or("qualification trace bytes overflow")?;
            if (phase == "execute" || matched) && measured != recorded {
                return Err("qualification trace bytes differ from the execution record".into());
            }
        }
    }
    // Every store, executed or replayed, ran under the frozen guest limit;
    // only multi-instance stores carry node/incarnation handles.
    let cluster = manifest.scenario_sha256.is_some();
    if result.observations.samples.iter().any(|sample| {
        sample.guest_memory_limit_bytes != manifest.memory_bytes
            || sample.incarnation.is_some() != cluster
    }) {
        return Err("qualification store sample differs from the frozen limits or workload".into());
    }
    let (cost, cost_failure, budget, budget_failure) =
        cost::evaluate(&result.observations, &result.report);
    if cost != result.cost || cost_failure != result.cost_failure {
        return Err(if result.cost.is_some() && cost.is_none() {
            "unmeasured WASM costs were claimed complete".into()
        } else {
            "retained WASM cost differs from raw measured samples".into()
        });
    }
    if budget != result.budget || budget_failure != result.budget_failure {
        return Err(if result.budget.is_some() && budget.is_none() {
            "unmeasured budget was claimed complete".into()
        } else {
            "retained budget differs from raw measurements".into()
        });
    }
    Ok(())
}

/// A cluster execution's incarnations are named by its record's evidence
/// streams (`evidence/cluster/NODE/INCARNATION/...`). Every measured cluster
/// window, executed or replayed, carries exactly that handle set, and every
/// execution was measured, so no store's measurement can be dropped.
fn bind_cluster_handles(result: &QualificationResult) -> Result<(), String> {
    use std::collections::{BTreeMap, BTreeSet};
    if result.manifest.scenario_sha256.is_none() {
        return Ok(());
    }
    let mut measured: BTreeMap<(&str, u64), BTreeSet<&str>> = BTreeMap::new();
    for sample in &result.observations.samples {
        measured
            .entry((sample.phase.as_str(), sample.iteration))
            .or_default()
            .insert(sample.incarnation.as_deref().unwrap_or_default());
    }
    for execution in &result.report.executions {
        let expected = execution
            .record
            .streams
            .keys()
            .filter_map(|name| {
                let mut parts = name.strip_prefix("evidence/cluster/")?.splitn(3, '/');
                let (node, incarnation) = (parts.next()?, parts.next()?);
                parts.next()?;
                Some(format!("{node}/{incarnation}"))
            })
            .collect::<BTreeSet<_>>();
        let expected = expected.iter().map(String::as_str).collect::<BTreeSet<_>>();
        if expected.is_empty()
            || measured.get(&("execute", execution.iteration)) != Some(&expected)
            || measured
                .get(&("replay", execution.iteration))
                .is_some_and(|handles| *handles != expected)
        {
            return Err("cluster store samples differ from the record's incarnations".into());
        }
    }
    Ok(())
}

pub(crate) fn encode(result: &QualificationResult) -> Result<Vec<u8>, String> {
    let mut encoded = canonical::encode(result)?;
    encoded.push(b'\n');
    Ok(encoded)
}

fn directory(root: &Path) -> Result<(PathBuf, File), String> {
    if !root.is_absolute() {
        return Err("absolute WASM qualification result root required".into());
    }
    let path = DIRECTORY
        .iter()
        .fold(root.to_path_buf(), |path, part| path.join(part));
    let pinned = durable_directory(&path, |_| Ok(()))?;
    Ok((path, pinned))
}

static STAGING: AtomicU64 = AtomicU64::new(0);

/// Stages, syncs and exclusively renames immutable bytes; a collision is
/// reused only when the existing file authenticates to the same bytes.
fn publish_bytes(root: &File, name: &str, bytes: &[u8], limit: u64) -> Result<(), String> {
    if bytes.is_empty() || bytes.len() as u64 > limit {
        return Err("WASM qualification result byte limit".into());
    }
    let stage = format!(
        ".result-{}-{}.partial",
        std::process::id(),
        STAGING.fetch_add(1, Ordering::Relaxed)
    );
    let mut file = File::from(
        fsx::openat(
            root,
            stage.as_str(),
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    let staged = (|| {
        file.write_all(bytes).map_err(|error| error.to_string())?;
        file.sync_all().map_err(|error| error.to_string())?;
        rename_exclusive(root, &stage, name)
    })();
    let outcome = match staged {
        Ok(()) => Ok(()),
        Err(error) if error == rustix::io::Errno::EXIST.to_string() => {
            open::read(root, name, limit, false).and_then(|existing| {
                if existing == bytes {
                    Ok(())
                } else {
                    Err("immutable WASM qualification result collision".into())
                }
            })
        }
        Err(error) => Err(error),
    };
    match fsx::unlinkat(root, stage.as_str(), fsx::AtFlags::empty()) {
        Ok(()) | Err(rustix::io::Errno::NOENT) => {}
        Err(error) => return Err(error.to_string()),
    }
    outcome?;
    root.sync_all().map_err(|error| error.to_string())
}

fn rename_exclusive(root: &File, source: &str, target: &str) -> Result<(), String> {
    #[cfg(any(target_os = "linux", target_os = "macos"))]
    {
        fsx::renameat_with(root, source, root, target, fsx::RenameFlags::NOREPLACE)
            .map_err(|error| error.to_string())
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        let _ = (root, source, target);
        Err("exclusive qualification publication is unsupported on this platform".into())
    }
}

fn name(bytes: &[u8]) -> String {
    format!("result-{}.json", canonical::digest(bytes))
}

/// Validates, encodes and publishes the Result; returns its absolute path.
pub(crate) fn publish(root: &Path, result: &QualificationResult) -> Result<PathBuf, String> {
    validate(result)?;
    let encoded = encode(result)?;
    let (path, pinned) = directory(root)?;
    let name = name(&encoded);
    publish_bytes(&pinned, &name, &encoded, MAXIMUM_RESULT_BYTES)?;
    Ok(path.join(name))
}

fn result_identity(path: &Path) -> Result<String, String> {
    path.file_name()
        .and_then(|name| name.to_str())
        .and_then(|name| name.strip_prefix("result-"))
        .and_then(|name| name.strip_suffix(".json"))
        .filter(|digest| canonical::valid_digest(digest))
        .map(str::to_owned)
        .ok_or_else(|| "publication requires content-addressed Result path".into())
}

fn pinned_parent(path: &Path) -> Result<(File, String), String> {
    let parent = path
        .parent()
        .ok_or("qualification result path has no directory")?;
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("qualification result path is not UTF-8")?;
    Ok((open::pin(parent)?, name.to_owned()))
}

fn receipt_prefix(name: &str) -> String {
    format!("{name}.publication-")
}

/// The single retained receipt of a Result: `NAME.publication-SHA256.json`,
/// content addressed like the Result itself. Several receipts are ambiguous.
fn receipt_names(pinned: &File, name: &str) -> Result<Vec<String>, String> {
    let prefix = receipt_prefix(name);
    let mut names = Vec::new();
    for entry in fsx::Dir::read_from(pinned).map_err(|error| error.to_string())? {
        let entry = entry.map_err(|error| error.to_string())?;
        if let Ok(entry) = entry.file_name().to_str()
            && let Some(digest) = entry
                .strip_prefix(prefix.as_str())
                .and_then(|rest| rest.strip_suffix(".json"))
        {
            if !canonical::valid_digest(digest) {
                return Err("malformed qualification publication receipt name".into());
            }
            names.push(entry.to_owned());
        }
    }
    Ok(names)
}

/// Publishes the receipt measured after the sealed Result publication under
/// its own content address; a different receipt for the same Result collides.
pub(crate) fn publish_receipt(path: &Path, receipt: &Receipt) -> Result<(), String> {
    let (pinned, name) = pinned_parent(path)?;
    let mut encoded = canonical::encode(receipt)?;
    encoded.push(b'\n');
    let receipt_name = format!(
        "{}{}.json",
        receipt_prefix(&name),
        canonical::digest(&encoded)
    );
    if receipt_names(&pinned, &name)?
        .iter()
        .any(|existing| *existing != receipt_name)
    {
        return Err("immutable qualification publication receipt collision".into());
    }
    publish_bytes(&pinned, &receipt_name, &encoded, MAXIMUM_RECEIPT_BYTES)
}

/// Opens a retained Result without installation, source or compiler: bounded
/// private regular file, content address, strict decode and recomputation.
pub(crate) fn open(path: &Path) -> Result<QualificationResult, String> {
    let digest = result_identity(path)?;
    let (pinned, name) = pinned_parent(path)?;
    let encoded = open::read(&pinned, &name, MAXIMUM_RESULT_BYTES, false)?;
    if canonical::digest(&encoded) != digest {
        return Err("WASM qualification result content identity changed".into());
    }
    let result: QualificationResult = super::spec::strict(&encoded)?;
    if encode(&result)? != encoded {
        return Err("WASM qualification result is not canonical".into());
    }
    validate(&result)?;
    Ok(result)
}

/// Opens and binds the receipt to its already opened Result; returns the
/// linked self CPU measured after the Result boundary.
pub(crate) fn open_receipt(
    path: &Path,
    result: &QualificationResult,
) -> Result<(Receipt, cost::Terminal), String> {
    let digest = result_identity(path)?;
    let (pinned, name) = pinned_parent(path)?;
    let [receipt_name] = <[String; 1]>::try_from(receipt_names(&pinned, &name)?)
        .map_err(|_| "qualification Result requires exactly one publication receipt")?;
    let encoded = open::read(&pinned, &receipt_name, MAXIMUM_RECEIPT_BYTES, false)?;
    if format!(
        "{}{}.json",
        receipt_prefix(&name),
        canonical::digest(&encoded)
    ) != receipt_name
    {
        return Err("qualification publication receipt content identity changed".into());
    }
    let receipt: Receipt = super::spec::strict(&encoded)?;
    let mut canonical_bytes = canonical::encode(&receipt)?;
    canonical_bytes.push(b'\n');
    if canonical_bytes != encoded {
        return Err("qualification publication receipt is not canonical".into());
    }
    if canonical::digest(&encode(result)?) != digest {
        return Err("publication receipt Result content binding changed".into());
    }
    let tail = validate_receipt(result, &receipt, &digest)?;
    Ok((receipt, tail))
}

pub(crate) fn validate_receipt(
    result: &QualificationResult,
    receipt: &Receipt,
    digest: &str,
) -> Result<cost::Terminal, String> {
    if receipt.schema != RECEIPT_SCHEMA
        || receipt.result_sha256 != digest
        || receipt.operator_process_id == 0
        || receipt.operator_process_id != result.observations.operator_process_id
    {
        return Err("missing or mismatched publication self observation".into());
    }
    let after = receipt
        .linked_after
        .ok_or("missing or mismatched publication self observation")?;
    cost::terminal(&result.observations, after, receipt.operator_peak_rss_bytes)
}

#[cfg(test)]
#[path = "receiving_publication_tests.rs"]
mod receiving_publication_tests;

#[cfg(test)]
#[path = "receiving_provenance_tests.rs"]
mod receiving_provenance_tests;
