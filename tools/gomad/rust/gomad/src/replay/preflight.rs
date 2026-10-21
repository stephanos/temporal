use crate::evidence::{canonical, inspection::Details, record::Record, store::Opened};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::Digest as _;
use std::{collections::BTreeMap, path::Path, time::Duration};

pub const FILE: &str = "evidence/replay.json";

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Configuration {
    pub engine: String,
    pub profile: String,
    pub limits: crate::evidence::record::Limits,
    pub arguments: Vec<String>,
    pub environment: BTreeMap<String, String>,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub derive_seed_environment: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
    pub runtime_sha256: Option<String>,
    pub model: Option<Value>,
    pub namespace: Option<linked::NamespaceImage>,
    #[serde(default)]
    pub storage: Option<linked::StorageInput>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Inputs {
    pub schema: String,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub toolchain_sha256: String,
    pub runtime_implementation_sha256: Option<String>,
    pub configuration: Configuration,
}

pub struct Prepared {
    pub opened: Opened,
    pub execution: Execution,
    pub exact: bool,
    pub available: bool,
}

pub enum Execution {
    Standalone(Box<linked::Prepared>),
    Cluster(Box<gomad_wasmhost::linked_cluster::Prepared>),
}

/// # Errors
/// Executes the admitted bytes using the matching typed helper core.
pub fn execute(execution: Execution) -> Result<linked::Execution, String> {
    match execution {
        Execution::Standalone(prepared) => {
            linked::execute_prepared(*prepared).map_err(|error| error.to_string())
        }
        Execution::Cluster(prepared) => super::cluster::project(
            gomad_wasmhost::linked_cluster::execute_prepared(*prepared)
                .map_err(|error| error.to_string())?,
        ),
    }
}

/// # Errors
/// Executes a fresh inspection attempt with a shortened remaining budget.
pub fn execute_inspection(
    mut prepared: Prepared,
    target: &gomad_wasmhost::linked_inspection::Handle,
    conditions: linked::Conditions,
    timeout: Option<Duration>,
) -> Result<linked::Execution, String> {
    match &mut prepared.execution {
        Execution::Standalone(execution) => {
            execution.conditions(conditions);
            if let Some(timeout) = timeout {
                execution
                    .bound_wall_timeout(timeout)
                    .map_err(|error| error.to_string())?;
            }
        }
        Execution::Cluster(execution) => {
            execution.conditions(conditions);
            if let Some(timeout) = timeout {
                execution
                    .bound_wall_timeout(timeout)
                    .map_err(|error| error.to_string())?;
            }
        }
    }
    match prepared.execution {
        Execution::Standalone(execution) => {
            linked::execute_prepared(*execution).map_err(|error| error.to_string())
        }
        Execution::Cluster(execution) => super::cluster::project_inspection(
            gomad_wasmhost::linked_cluster::execute_prepared(*execution)
                .map_err(|error| error.to_string())?,
            target,
        ),
    }
}

/// # Errors
/// Refuses unsupported, unbound or incompatible retained inputs before guest creation.
pub fn open(path: &Path) -> Result<Prepared, String> {
    prepare(Opened::open(path)?)
}

/// # Errors
/// Uses only authenticated pinned-root payloads and validates all streams before execution.
pub fn prepare(opened: Opened) -> Result<Prepared, String> {
    prepare_internal(opened, None)
}

/// # Errors
/// Adds an authenticated inspection control to the ordinary replay preflight.
pub fn prepare_with_inspection(
    opened: Opened,
    control: gomad_wasmhost::linked_inspection::Request,
) -> Result<Prepared, String> {
    prepare_internal(opened, Some(control))
}

#[expect(
    clippy::too_many_lines,
    reason = "Ordered replay admission keeps all identity and bound checks before guest construction"
)]
fn prepare_internal(
    opened: Opened,
    control: Option<gomad_wasmhost::linked_inspection::Request>,
) -> Result<Prepared, String> {
    opened.authenticate_all()?;
    let manifest = opened.manifest();
    let record = &manifest.record;
    if manifest.files.contains_key("record.json")
        && Record::decode(&opened.payload("record.json", 16 << 20)?)? != *record
    {
        return Err("record payload mismatch".into());
    }
    if manifest.files.contains_key(super::cluster::FILE) {
        let prepared = super::cluster::prepare(opened, control.as_ref())?;
        return Ok(Prepared {
            opened: prepared.opened,
            execution: Execution::Cluster(Box::new(prepared.execution)),
            exact: prepared.exact,
            available: prepared.available,
        });
    }
    if !record.streams.contains_key(FILE) {
        return Err("missing retained replay inputs".into());
    }
    let bytes = opened.payload(FILE, 16 << 20)?;
    let inputs: Inputs = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if canonical::encode(&inputs)? != bytes || inputs.schema != "gomad.replay-inputs/v1" {
        return Err("unsupported or noncanonical retained replay inputs".into());
    }
    if inputs.helper_sha256 != linked::IMPLEMENTATION_SHA256
        || inputs.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || record.identity.implementation != implementation()?
        || inputs.configuration.engine != linked::CONFIGURATION
    {
        return Err("WASM compiler/helper/engine/model/limits identity mismatch".into());
    }
    if inputs.toolchain_sha256 != record.identity.toolchain
        || inputs.configuration.profile != record.profile
        || inputs.configuration.limits != record.limits
        || inputs.configuration.arguments != record.arguments
        || inputs.configuration.environment != record.environment
        || record.identity.configuration
            != canonical::digest(&canonical::encode(&inputs.configuration)?)
        || record.identity.platform != platform()
    {
        return Err("WASM immutable configuration identity mismatch".into());
    }
    let exact = match record.profile.as_str() {
        "gomad.wasi-cooperative/v1" => {
            if inputs.runtime_implementation_sha256.as_deref()
                != Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"))
                || !inputs
                    .configuration
                    .runtime_sha256
                    .as_deref()
                    .is_some_and(canonical::valid_digest)
            {
                return Err("WASM runtime implementation identity mismatch".into());
            }
            true
        }
        "gomad.wasi-stock-exploratory/v1"
            if inputs.configuration.runtime_sha256.is_none()
                && inputs.runtime_implementation_sha256.is_none() =>
        {
            false
        }
        _ => return Err("WASM artifact replay profile mismatch".into()),
    };
    let details = Details::read(&opened)?;
    let mut model = inputs
        .configuration
        .model
        .map(|mut value| {
            let object = value
                .as_object_mut()
                .ok_or("invalid retained model configuration")?;
            if object.insert("seed".into(), record.seed.into()).is_some() {
                return Err("retained configuration must derive seed from record".to_owned());
            }
            serde_json::from_value::<linked::ModelConfig>(value).map_err(|error| error.to_string())
        })
        .transpose()?;
    let mut trace = None;
    let mut available = true;
    if let Some(configuration) = &mut model
        && (configuration.seed != record.seed
            || configuration.replay.is_some()
            || configuration.mode != 1
            || configuration.split
            || configuration.inspection.is_some())
    {
        return Err("WASM retained model configuration mismatch".into());
    }
    if exact {
        let choices = details
            .as_ref()
            .and_then(|details| details.choices.as_ref())
            .ok_or("WASM artifact exact choice identity mismatch")?;
        if choices.implementation_sha256 != inputs.helper_sha256 {
            return Err("WASM artifact exact choice implementation identity mismatch".into());
        }
        available = choices.terminal_state == "complete";
        if available {
            let configuration = model
                .as_mut()
                .ok_or("exact replay requires helper-owned model")?;
            let raw = opened.payload(&choices.trace.file, choices.trace.limit)?;
            let projected = linked::inspect_choices(&raw, &choices.identity)?;
            if choices.identity.toolchain_build_key != record.identity.toolchain
                || choices.identity.goos != "wasip1"
                || choices.identity.goarch != "wasm"
                || inputs.configuration.runtime_sha256.as_deref()
                    != Some(choices.identity.runtime_implementation_sha256().as_str())
                || choices.identity.target_sha256.as_slice()
                    != sha2::Sha256::digest(&opened.payload("target/module.wasm", 512 << 20)?)
                        .as_slice()
            {
                return Err("WASM artifact exact choice identity mismatch".into());
            }
            configuration.mode = 2;
            configuration.replay = Some(
                serde_json::from_value(serde_json::json!({
                    "schema":"gomad.replay-trace/v1", "execution_identity":choices.identity,
                    "trace_sha256":choices.trace.sha256,"tape_sha256":projected.tape_sha256,
                    "decisions":projected.decisions,"trace_bytes":raw.len(),
                    "trace_frames":(raw.len() as u64).div_ceil(65536),"prefix_control":null
                }))
                .map_err(|error| error.to_string())?,
            );
            trace = Some(raw);
        }
    } else if details
        .as_ref()
        .is_some_and(|details| details.choices.is_some())
    {
        return Err("WASM artifact replay profile mismatch".into());
    }
    if let Some(control) = control {
        if !exact || !available {
            return Err("inspection requires a complete cooperative replay tape".into());
        }
        model
            .as_mut()
            .ok_or("inspection requires helper-owned model")?
            .inspection = Some(control);
    }
    let module = opened.payload("target/module.wasm", 512 << 20)?;
    let mut environment = record
        .environment
        .iter()
        .map(|(key, value)| format!("{key}={value}"))
        .collect::<Vec<_>>();
    if inputs.configuration.derive_seed_environment {
        if record.environment.contains_key("GOMADSEED") {
            return Err("retained seed environment must derive from record".into());
        }
        environment.push(format!("GOMADSEED={}", record.seed));
    }
    let request = linked::Request {
        module,
        module_sha256: record.identity.target.clone(),
        limits: linked::Limits {
            memory_bytes: record.limits.memory_bytes,
            fuel: record.limits.fuel,
            output_bytes: record.limits.output_bytes,
            evidence_bytes: record.limits.evidence_bytes,
            wall_timeout: Duration::from_millis(record.limits.wall_millis),
        },
        model: model.map(|configuration| linked::ModelInput {
            configuration,
            arguments: record.arguments.clone(),
            environment,
            replay_trace: trace,
        }),
        namespace: inputs.configuration.namespace,
        storage: inputs.configuration.storage,
    };
    let execution = linked::prepare(request).map_err(|error| error.to_string())?;
    Ok(Prepared {
        opened,
        execution: Execution::Standalone(Box::new(execution)),
        exact,
        available,
    })
}

/// # Errors
/// Returns a canonical identity for the actual linked host and helper components.
pub fn implementation() -> Result<String, String> {
    Ok(canonical::digest(&canonical::encode(&serde_json::json!({
        "host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"), "helper":linked::IMPLEMENTATION_SHA256
    }))?))
}

#[must_use]
pub fn platform() -> String {
    canonical::digest(format!("{}/{}", std::env::consts::OS, std::env::consts::ARCH).as_bytes())
}
