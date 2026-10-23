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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub world_transition_bytes: Option<u64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub wall_nanos: Option<u64>,
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

pub(crate) fn validate_profile_imports(
    profile: &str,
    metadata: &linked::ModuleMetadata,
) -> Result<(), String> {
    let cooperative = match profile {
        "gomad.wasi-cooperative/v1" => true,
        "gomad.wasi-stock-exploratory/v1" => false,
        _ => return Err("unsupported WASM import profile".into()),
    };
    for import in &metadata.imports {
        if !import.module.starts_with("gomad_") {
            continue;
        }
        if !cooperative {
            return Err("cooperative import in stock profile".into());
        }
        if !matches!(
            (import.module.as_str(), import.name.as_str()),
            (
                "gomad_wasm_v1",
                "config" | "decision" | "observation" | "finish" | "idle"
            ) | ("gomad_wasm_v2", "net_stream")
                | ("gomad_debug_v1", "exchange")
        ) || import.params != ["i32", "i32"]
            || import.results != ["i32"]
        {
            return Err("unsupported cooperative import name or signature".into());
        }
    }
    if cooperative
        && ["config", "decision", "observation", "finish", "idle"]
            .into_iter()
            .any(|name| {
                !metadata
                    .imports
                    .iter()
                    .any(|import| import.module == "gomad_wasm_v1" && import.name == name)
            })
    {
        return Err("missing cooperative v1 import".into());
    }
    Ok(())
}

pub(crate) fn wall_timeout(
    limits: &crate::evidence::record::Limits,
    nanos: Option<u64>,
) -> Result<Duration, String> {
    if let Some(nanos) = nanos {
        if nanos == 0 || (nanos - 1) / 1_000_000 + 1 != limits.wall_millis {
            return Err("WASM exact deadline and recorded limit mismatch".into());
        }
        return Ok(Duration::from_nanos(nanos));
    }
    Ok(Duration::from_millis(limits.wall_millis))
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
    Cluster(Box<super::cluster::Execution>),
}

impl Prepared {
    #[must_use]
    pub fn compile_cache_directory(&self) -> Option<&str> {
        match &self.execution {
            Execution::Standalone(prepared) => prepared.compile_cache_directory(),
            Execution::Cluster(prepared) => prepared.prepared.compile_cache_directory(),
        }
    }

    /// # Errors
    /// Refuses invalid host cache configuration before execution, without opening build inputs.
    pub fn compile_cache(&mut self, directory: Option<&str>) -> Result<(), String> {
        match &mut self.execution {
            Execution::Standalone(prepared) => prepared.compile_cache(directory),
            Execution::Cluster(prepared) => prepared.prepared.compile_cache(directory),
        }
        .map_err(|error| error.to_string())
    }
}

/// # Errors
/// Executes the admitted bytes using the matching typed helper core.
pub fn execute(execution: Execution) -> Result<linked::Execution, String> {
    match execution {
        Execution::Standalone(prepared) => crate::backstop::within(prepared.wall_timeout(), || {
            linked::execute_prepared(*prepared)
        })
        .map_err(|error| error.to_string()),
        Execution::Cluster(prepared) => super::cluster::execute_projected(*prepared),
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
            execution.prepared.conditions(conditions);
            if let Some(timeout) = timeout {
                execution
                    .prepared
                    .bound_wall_timeout(timeout)
                    .map_err(|error| error.to_string())?;
            }
        }
    }
    match prepared.execution {
        Execution::Standalone(execution) => {
            crate::backstop::within(execution.wall_timeout(), || {
                linked::execute_prepared(*execution)
            })
            .map_err(|error| error.to_string())
        }
        Execution::Cluster(execution) => {
            super::cluster::project_inspection(super::cluster::execute(*execution)?, target)
        }
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
    prepare_internal(opened, None, false)
}

pub(crate) fn prepare_runtime_qualification(opened: Opened) -> Result<Prepared, String> {
    prepare_internal(opened, None, true)
}

/// # Errors
/// Adds an authenticated inspection control to the ordinary replay preflight.
pub fn prepare_with_inspection(
    opened: Opened,
    control: gomad_wasmhost::linked_inspection::Request,
) -> Result<Prepared, String> {
    prepare_internal(opened, Some(control), false)
}

fn prepare_internal(
    opened: Opened,
    control: Option<gomad_wasmhost::linked_inspection::Request>,
    qualification_diagnostics_off: bool,
) -> Result<Prepared, String> {
    opened.authenticate_all()?;
    prepare_authenticated_internal(opened, control, qualification_diagnostics_off)
}

pub(crate) fn prepare_authenticated(opened: Opened) -> Result<Prepared, String> {
    prepare_authenticated_internal(opened, None, false)
}

#[expect(
    clippy::too_many_lines,
    reason = "Ordered replay admission keeps all identity and bound checks before guest construction"
)]
fn prepare_authenticated_internal(
    opened: Opened,
    control: Option<gomad_wasmhost::linked_inspection::Request>,
    qualification_diagnostics_off: bool,
) -> Result<Prepared, String> {
    let manifest = opened.manifest();
    let record = &manifest.record;
    if manifest.files.contains_key("record.json")
        && Record::decode(&opened.payload("record.json", 16 << 20)?)? != *record
    {
        return Err("record payload mismatch".into());
    }
    if manifest.files.contains_key(super::cluster::FILE) {
        if qualification_diagnostics_off {
            return Err("runtime qualification requires a standalone guest".into());
        }
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
        || inputs.configuration.engine != linked::Conditions::compiled().configuration()
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
    authenticate_provenance(&opened, &inputs)?;
    validate_profile_imports(
        &record.profile,
        &linked::module_metadata(&opened.payload("target/module.wasm", 512 << 20)?)
            .map_err(|error| error.to_string())?,
    )?;
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
            if inputs.configuration.derive_seed_environment
                && record.profile == "gomad.wasi-stock-exploratory/v1"
            {
                object.insert(
                    "entropy_key".into(),
                    serde_json::json!(
                        sha2::Sha256::digest(
                            format!("{}:{}", inputs.toolchain_sha256, record.seed).as_bytes()
                        )
                        .to_vec()
                    ),
                );
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
        if qualification_diagnostics_off {
            let configuration = model
                .as_mut()
                .ok_or("runtime qualification requires a model")?;
            if !available || !configuration.diagnostics {
                return Err("runtime qualification requires complete diagnostic recording".into());
            }
            configuration.diagnostics = false;
        }
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
    } else if qualification_diagnostics_off {
        return Err("runtime qualification requires exact replay".into());
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
            wall_timeout: wall_timeout(&record.limits, inputs.configuration.wall_nanos)?,
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
    let mut execution = linked::prepare(request).map_err(|error| error.to_string())?;
    execution
        .world_transition_bytes(inputs.configuration.world_transition_bytes)
        .map_err(|error| error.to_string())?;
    if let Some(network) = inputs.configuration.network {
        execution
            .network(network)
            .map_err(|error| error.to_string())?;
    }
    Ok(Prepared {
        opened,
        execution: Execution::Standalone(Box::new(execution)),
        exact,
        available,
    })
}

fn authenticate_provenance(opened: &Opened, inputs: &Inputs) -> Result<(), String> {
    if !opened
        .manifest()
        .files
        .contains_key("target/provenance.json")
    {
        return Ok(());
    }
    let raw = opened.payload("target/provenance.json", 128 << 20)?;
    let value: Value = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    crate::campaign::prepare::typed_provenance(&value)?;
    let metadata: Option<linked::ModuleMetadata> = value
        .get("module_metadata")
        .map(|metadata| serde_json::from_value(metadata.clone()))
        .transpose()
        .map_err(|error| error.to_string())?;
    crate::campaign::prepare::authenticate_module_metadata(
        metadata.as_ref(),
        &value["inputs"],
        &opened.payload("target/module.wasm", 512 << 20)?,
    )?;
    let record = &opened.manifest().record;
    let mut build_configuration =
        serde_json::to_value(&inputs.configuration).map_err(|error| error.to_string())?;
    build_configuration["runtime_sha256"] = Value::Null;
    if canonical::encode(&value)? != raw
        || value["schema"] != "gomad.rust-wasm-provenance/v1"
        || value["build_key"] != record.identity.toolchain
        || value["module_sha256"] != record.identity.target
        || value["build_key"] != canonical::digest(&canonical::encode(&value["inputs"])?)
        || value["configuration"]
            != serde_json::to_value(&inputs.configuration).map_err(|error| error.to_string())?
        || value["inputs"]["configuration"] != build_configuration
        || value["inputs"]["helper"] != inputs.helper_sha256
        || value["inputs"]["host"] != inputs.host_sha256
        || value["inputs"]["network"] != value["network"]
        || value["inputs"]["scenario"] != value["scenario"]
    {
        return Err("WASM prepared provenance identity mismatch".into());
    }
    if let Some(identity) = value["choice_identity"].as_object()
        && (identity.get("ToolchainBuildKey") != Some(&value["build_key"])
            || identity.get("TargetSHA256")
                != Some(&serde_json::json!(
                    sha2::Sha256::digest(opened.payload("target/module.wasm", u64::MAX)?).to_vec()
                )))
    {
        return Err("WASM prepared provenance choice identity mismatch".into());
    }
    Ok(())
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

#[cfg(test)]
mod module_metadata_tests {
    use super::linked;

    #[test]
    fn module_metadata_reads_actual_imports_and_initial_pages() {
        let mut bytes = b"\0asm\x01\0\0\0".to_vec();
        bytes.extend_from_slice(&[1, 8, 2, 0x60, 1, 0x7f, 0, 0x60, 0, 0]);
        let mut imports = vec![1, 22];
        imports.extend_from_slice(b"wasi_snapshot_preview1");
        imports.push(9);
        imports.extend_from_slice(b"proc_exit");
        imports.extend_from_slice(&[0, 0]);
        bytes.extend_from_slice(&[2, u8::try_from(imports.len()).unwrap()]);
        bytes.extend(imports);
        bytes.extend_from_slice(&[3, 2, 1, 1, 5, 3, 1, 0, 2]);
        bytes.extend_from_slice(&[7, 19, 2, 6]);
        bytes.extend_from_slice(b"memory");
        bytes.extend_from_slice(&[2, 0, 6]);
        bytes.extend_from_slice(b"_start");
        bytes.extend_from_slice(&[0, 1, 10, 4, 1, 2, 0, 0x0b]);
        let actual = linked::module_metadata(&bytes).unwrap();
        assert_eq!(
            actual,
            linked::ModuleMetadata {
                imports: vec![linked::ModuleImport {
                    module: "wasi_snapshot_preview1".into(),
                    name: "proc_exit".into(),
                    params: vec!["i32".into()],
                    results: Vec::new(),
                }],
                initial_memory_pages: 2,
            }
        );
        let inputs = serde_json::json!({"module_metadata":actual});
        crate::campaign::prepare::authenticate_module_metadata(Some(&actual), &inputs, &bytes)
            .unwrap();
        for imports in [true, false] {
            let mut altered = actual.clone();
            if imports {
                altered.imports.clear();
            } else {
                altered.initial_memory_pages += 1;
            }
            let inputs = serde_json::json!({"module_metadata":altered});
            assert!(
                crate::campaign::prepare::authenticate_module_metadata(
                    Some(&altered),
                    &inputs,
                    &bytes,
                )
                .is_err()
            );
        }
    }
}

#[cfg(test)]
mod profile_import_tests {
    use super::*;

    fn listed() -> linked::ModuleMetadata {
        let inventory: Value = serde_json::from_str(include_str!(
            "../../../../internal/backends/wasm/determinism/inventory.json"
        ))
        .unwrap();
        let mut imports: Vec<_> = inventory["entries"]
            .as_array()
            .unwrap()
            .iter()
            .filter_map(|entry| {
                let site = entry["site"].as_str().unwrap().strip_prefix("import:")?;
                if !site.starts_with("gomad_") {
                    return None;
                }
                let (module, name) = site.split_once("::").unwrap();
                Some(linked::ModuleImport {
                    module: module.into(),
                    name: name.into(),
                    params: vec!["i32".into(), "i32".into()],
                    results: vec!["i32".into()],
                })
            })
            .collect();
        imports.sort_by(|a, b| (&a.module, &a.name).cmp(&(&b.module, &b.name)));
        assert_ne!(imports, Vec::<gomad_wasmhost::linked::ModuleImport>::new());
        linked::ModuleMetadata {
            imports,
            initial_memory_pages: 1,
        }
    }

    #[test]
    fn cooperative_profile_preserves_the_original_complete_import_boundary() {
        const PROFILE: &str = "gomad.wasi-cooperative/v1";
        validate_profile_imports(PROFILE, &listed()).unwrap();
        for (module, name) in [
            ("gomad_wasm_v1", "clock"),
            ("gomad_wasm_v2", "datagram"),
            ("gomad_debug_v1", "peek"),
        ] {
            let mut metadata = listed();
            metadata.imports.push(linked::ModuleImport {
                module: module.into(),
                name: name.into(),
                params: vec!["i32".into(), "i32".into()],
                results: vec!["i32".into()],
            });
            assert!(
                validate_profile_imports(PROFILE, &metadata)
                    .unwrap_err()
                    .contains("import")
            );
        }
        let mut metadata = listed();
        metadata.imports.insert(
            0,
            linked::ModuleImport {
                module: "gomad_wasm_v1".into(),
                name: "decision".into(),
                params: vec!["i64".into()],
                results: vec!["i32".into()],
            },
        );
        assert!(validate_profile_imports(PROFILE, &metadata).is_err());
        let mut metadata = listed();
        metadata.imports.retain(|import| import.name != "idle");
        assert_eq!(
            validate_profile_imports(PROFILE, &metadata).unwrap_err(),
            "missing cooperative v1 import"
        );
        assert_eq!(
            validate_profile_imports("gomad.wasi-stock-exploratory/v1", &listed()).unwrap_err(),
            "cooperative import in stock profile"
        );
        let mut stock = listed();
        stock.imports.clear();
        validate_profile_imports("gomad.wasi-stock-exploratory/v1", &stock).unwrap();
    }
}
