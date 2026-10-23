use crate::{
    cli::explore::ExploreOptions,
    evidence::{canonical, record::Limits},
    replay::preflight::Configuration,
};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest as _, Sha256};
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
    time::Instant,
};

pub(crate) mod driver;
pub(crate) mod inventory;
pub(crate) mod process;

const MAXIMUM: u64 = 512 << 20;

pub struct Prepared {
    pub compile_cache_root: Option<String>,
    pub module: Vec<u8>,
    pub configuration: Configuration,
    pub toolchain: String,
    pub choice_identity: Option<linked::ChoiceIdentity>,
    pub scenario: Option<Vec<u8>>,
    pub provenance: Vec<u8>,
    pub network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
    pub soft_assertion_report_only: bool,
}

pub use crate::install::manifest::Installation;

pub(crate) fn compile_cache_root(installed: &Installation) -> Result<Option<String>, String> {
    installed
        .admission
        .as_ref()
        .and_then(|admission| admission.compile_cache_root.as_ref())
        .map(|path| {
            path.to_str()
                .map(str::to_owned)
                .ok_or_else(|| failure("compile cache root is not UTF-8"))
        })
        .transpose()
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Provenance {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    runtime_workload: Option<crate::qualify::workloads::recovery::Request>,
    schema: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    origin: Option<super::origin::Intercepted>,
    inputs: Value,
    build_key: String,
    module_sha256: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    module_metadata: Option<linked::ModuleMetadata>,
    configuration: Configuration,
    choice_identity: Option<linked::ChoiceIdentity>,
    scenario: Option<Vec<u8>>,
    network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
}

/// Refuses a provenance value that is not exactly its typed, canonical form:
/// any unknown or ill-typed field at any typed level is refused.
pub(crate) fn typed_provenance(value: &Value) -> Result<(), String> {
    let typed: Provenance = serde_json::from_value(value.clone()).map_err(failure)?;
    if serde_json::to_value(&typed).map_err(failure)? != *value {
        return Err(failure("provenance is not in its typed canonical form"));
    }
    Ok(())
}

pub(crate) fn authenticate_module_metadata(
    retained: Option<&linked::ModuleMetadata>,
    inputs: &Value,
    module: &[u8],
) -> Result<(), String> {
    let bound = retained
        .map(serde_json::to_value)
        .transpose()
        .map_err(failure)?;
    if inputs.get("module_metadata") != bound.as_ref() {
        return Err(failure("prepared module metadata binding mismatch"));
    }
    if let Some(retained) = retained
        && *retained != linked::module_metadata(module).map_err(failure)?
    {
        return Err(failure("prepared module metadata mismatch"));
    }
    Ok(())
}

fn failure(error: impl std::fmt::Display) -> String {
    format!("runner_failure: {error}")
}

pub(super) fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    use std::io::Read as _;
    let mut file = fs::File::open(path).map_err(failure)?;
    if !file.metadata().map_err(failure)?.is_file() {
        return Err(failure("input is not a regular file"));
    }
    let mut bytes = Vec::new();
    file.by_ref()
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(failure)?;
    if bytes.len() as u64 > limit {
        return Err(format!("capacity: input exceeds {limit} bytes"));
    }
    Ok(bytes)
}

fn installation(options: &ExploreOptions) -> Result<Installation, String> {
    let description = crate::install::manifest::resolve(options.installation.as_deref())?;
    crate::install::manifest::admit_build(&description.manifest)?;
    Ok(description.manifest)
}

/// # Errors
/// Refuses an incompatible installation; returns its pinned runtime profile.
pub(crate) fn installation_profile(options: &ExploreOptions) -> Result<String, String> {
    installation(options).map(|installed| installed.profile)
}

/// # Errors
/// Refuses incompatible installations, unsupported targets and changed build inputs.
pub fn prepare(options: &ExploreOptions) -> Result<Prepared, String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("overall_timeout: preparation deadline exceeds clock range")?;
    let result = prepare_before_deadline(options, deadline, false);
    process::check_deadline(deadline)?;
    result
}

pub(crate) fn prepare_runtime_qualification(options: &ExploreOptions) -> Result<Prepared, String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("runtime qualification preparation deadline exceeds clock range")?;
    let result = prepare_before_deadline(options, deadline, true);
    process::check_deadline(deadline)?;
    result
}

fn prepare_before_deadline(
    options: &ExploreOptions,
    deadline: Instant,
    runtime_qualification: bool,
) -> Result<Prepared, String> {
    if options.target_kind == "exec" {
        return prepared_exec(options, &installation(options)?);
    }
    let mut recipe = driver::prepare(options, deadline)?;
    if runtime_qualification {
        recipe.configuration = crate::runtime_qualification::configuration()?;
    }
    let temporary = process::Temporary::new(&recipe.installed.cache_root)?;
    let output = temporary.0.join("module.wasm");
    let mut build = if options.target_kind == "go-test" {
        vec!["test".into(), "-c".into()]
    } else {
        vec!["build".into()]
    };
    build.extend(recipe.flags.clone());
    build.extend([
        "-trimpath".into(),
        "-buildvcs=false".into(),
        "-o".into(),
        output.to_string_lossy().into_owned(),
        options.source.clone(),
    ]);
    let mut compilation_environment = recipe.environment.clone();
    compilation_environment.insert(
        "GOCACHE".into(),
        temporary.0.join("objects").to_string_lossy().into_owned(),
    );
    process::command(
        &recipe.installed.compiler_path,
        &build,
        &recipe.directory,
        &compilation_environment,
        &temporary.0,
        4 << 20,
        deadline,
    )?;
    let module = read(&output, MAXIMUM)?;
    recipe.admit(module, options.target_args.clone(), deadline)
}

pub(crate) fn choice_implementation(key: &str) -> Result<sha2::digest::Output<Sha256>, String> {
    let mut source = Sha256::new();
    source.update(b"gomad-choice-implementation-source-v3");
    let inputs: &[&[u8]] = &[
        include_bytes!("../../../../internal/model/choice/schema/choicewire.json"),
        include_bytes!("../../../../internal/model/choice/schema/choicewire.go.tmpl"),
        include_bytes!("../../../../internal/model/choice/schema/choicewire_runtime.go.tmpl"),
        include_bytes!("../../../../internal/model/choice/schema/choicewire_test.go.tmpl"),
        include_bytes!("../../../../internal/model/runtimesource/gomad_semantics.go.txt"),
        include_bytes!("../../../../internal/model/choice/trace.go"),
        include_bytes!("../../../../internal/model/choice/tape.go"),
    ];
    for bytes in inputs {
        source.update((bytes.len() as u64).to_be_bytes());
        source.update(bytes);
    }
    let mut hash = Sha256::new();
    hash.update(b"gomad-choice-implementation-v2\0");
    hash.update(source.finalize());
    for index in (0..key.len()).step_by(2) {
        hash.update([u8::from_str_radix(&key[index..index + 2], 16).map_err(failure)?]);
    }
    Ok(hash.finalize())
}

fn configuration(
    options: &ExploreOptions,
    installed: &Installation,
) -> Result<Configuration, String> {
    let mut environment = installed
        .admission
        .as_ref()
        .map(|admission| admission.environment.clone())
        .unwrap_or_default();
    let mut supplied = std::collections::BTreeSet::new();
    for entry in &options.environment {
        let (name, value) = entry
            .split_once('=')
            .ok_or_else(|| "unsupported_target: environment requires NAME=VALUE".to_owned())?;
        if name.is_empty()
            || name.contains('\0')
            || value.contains('\0')
            || !supplied.insert(name.to_owned())
        {
            return Err("unsupported_target: invalid or duplicate environment entry".into());
        }
        environment.insert(name.into(), value.into());
    }
    let working =
        String::from_utf8(installed.namespace.working_directory.clone()).map_err(failure)?;
    environment.insert("PWD".into(), working.clone());
    environment.insert("TZ".into(), "UTC".into());
    environment.insert("GOMAD_IO_PROFILE".into(), "wasm".into());
    if options.choices {
        environment.insert(
            "GOMAD_CHOICE_PROFILE".into(),
            "gomad-choice-trace/v4".into(),
        );
    }
    if options.diagnostics {
        environment.insert(
            "GOMAD_DIAGNOSTIC_PROFILE".into(),
            "gomad-diagnostic-trace/v1".into(),
        );
    }
    let mut model = installed.model.clone();
    let object = model
        .as_object_mut()
        .ok_or_else(|| failure("model must be an object"))?;
    let choice_limit =
        super::scenario::original_choice_bytes(options.scenario.as_deref(), options.choice_bytes)?;
    object.insert("mode".into(), json!(1));
    object.insert("limit".into(), json!(choice_limit));
    object.insert("diagnostics".into(), json!(options.diagnostics));
    object.remove("seed");
    object.insert(
        "working_directory_sha256".into(),
        json!(canonical::digest(working.as_bytes())),
    );
    let mut namespace = installed.namespace.clone();
    namespace.limits.output_bytes = options.output_limit;
    let mut checked_model = model.clone();
    checked_model["seed"] = json!(0);
    let checked_model: linked::ModelConfig =
        serde_json::from_value(checked_model).map_err(failure)?;
    checked_model.validate().map_err(failure)?;
    let evidence = choice_limit
        .checked_add(
            options
                .output_limit
                .checked_mul(2)
                .ok_or("capacity: evidence bounds overflow")?,
        )
        .and_then(|v| v.checked_add(128 << 20))
        .ok_or("capacity: evidence bounds overflow")?;
    let wall_nanos = u64::try_from(options.execution_timeout.as_nanos()).map_err(failure)?;
    let wall_millis = wall_nanos
        .checked_sub(1)
        .ok_or("execution timeout must be positive")?
        / 1_000_000
        + 1;
    Ok(Configuration {
        engine: linked::Conditions::compiled().configuration(),
        profile: installed.profile.clone(),
        limits: Limits {
            fuel: installed.fuel,
            memory_bytes: installed.memory_bytes,
            wall_millis,
            output_bytes: options.output_limit,
            evidence_bytes: evidence,
        },
        arguments: std::iter::once("gomad-target".into())
            .chain(options.target_args.iter().cloned())
            .collect(),
        wall_nanos: Some(wall_nanos),
        environment,
        derive_seed_environment: true,
        runtime_sha256: None,
        model: Some(model),
        namespace: Some(namespace),
        storage: installed.storage.clone(),
        network: installed.network.clone(),
    })
}

fn prepared_exec(options: &ExploreOptions, installed: &Installation) -> Result<Prepared, String> {
    if options.provenance.is_empty() {
        return Err("unsupported_target: exec requires Rust provenance".into());
    }
    let raw = read(Path::new(&options.provenance), 16 << 20)?;
    let provenance: Provenance = serde_json::from_slice(&raw).map_err(failure)?;
    let module = read(Path::new(&options.source), MAXIMUM)?;
    authenticate_module_metadata(
        provenance.module_metadata.as_ref(),
        &provenance.inputs,
        &module,
    )?;
    if provenance.schema != "gomad.rust-wasm-provenance/v1"
        || canonical::encode(&provenance)? != raw
        || provenance.module_sha256 != canonical::digest(&module)
        || provenance.build_key != canonical::digest(&canonical::encode(&provenance.inputs)?)
        || provenance.inputs["compiler_sha256"] != installed.compiler_sha256
        || provenance.inputs["helper"] != linked::IMPLEMENTATION_SHA256
        || provenance.inputs["host"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || provenance.inputs["runtime_implementation"]
            != json!(installed.runtime_implementation_sha256)
        || provenance.configuration.profile != installed.profile
    {
        return Err(failure("prepared provenance identity mismatch"));
    }
    if provenance.inputs["schema"] != "gomad.rust-wasm-build-inputs/v1"
        || provenance.inputs["configuration"].as_object().is_none()
        || provenance.inputs["sources"]
            .as_object()
            .is_none_or(serde_json::Map::is_empty)
        || provenance.inputs["compiler"]
            .as_object()
            .is_none_or(serde_json::Map::is_empty)
        || provenance.inputs["network"] != json!(provenance.network)
        || provenance.inputs["network"] != json!(installed.network)
        || provenance.inputs["scenario"] != json!(provenance.scenario)
    {
        return Err(failure("prepared source/input inventory is incomplete"));
    }
    if let Some(identity) = &provenance.choice_identity {
        if identity.target_sha256 != <[u8; 32]>::from(Sha256::digest(&module))
            || identity.toolchain_build_key != provenance.build_key
            || identity.goos != "wasip1"
            || identity.goarch != "wasm"
            || provenance.configuration.runtime_sha256.as_deref()
                != Some(identity.runtime_implementation_sha256().as_str())
        {
            return Err(failure("prepared choice identity mismatch"));
        }
    } else if installed.profile == "gomad.wasi-cooperative/v1" {
        return Err(failure("prepared cooperative identity missing"));
    }
    let expected = configuration(options, installed)?;
    let mut actual: Value = serde_json::to_value(&provenance.configuration).map_err(failure)?;
    actual["runtime_sha256"] = Value::Null;
    if provenance.inputs["configuration"] != actual
        || actual != serde_json::to_value(expected).map_err(failure)?
        || provenance.scenario
            != options
                .scenario
                .as_deref()
                .map(super::scenario::normalize)
                .transpose()?
    {
        return Err(failure("prepared configuration identity mismatch"));
    }
    let soft_assertion_report_only = provenance.inputs["soft_assertion_report_only"]
        .as_bool()
        .ok_or_else(|| failure("prepared assertion policy missing"))?;
    Ok(Prepared {
        compile_cache_root: compile_cache_root(installed)?,
        module,
        configuration: provenance.configuration,
        toolchain: provenance.build_key,
        choice_identity: provenance.choice_identity,
        scenario: provenance.scenario,
        provenance: raw,
        network: provenance.network,
        soft_assertion_report_only,
    })
}
