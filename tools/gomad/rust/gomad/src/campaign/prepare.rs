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
    os::unix::fs::PermissionsExt as _,
    path::{Path, PathBuf},
    time::Instant,
};

#[cfg(test)]
mod actual_tests;
pub(crate) mod compilation_cache;
pub(crate) mod driver;
pub(crate) mod inventory;
mod observation_sidecar;
pub(crate) mod observations;
pub(crate) mod process;
#[cfg(test)]
mod stock_reach_tests;

const MAXIMUM: u64 = 512 << 20;
pub(crate) const OBSERVATION_DIRECTORY: &str = "preparation-observations";

#[derive(Default)]
pub(crate) struct CompilerControls {
    pub environment: BTreeMap<String, String>,
    pub parallelism: Option<u32>,
}

impl CompilerControls {
    fn apply(&self, environment: &mut BTreeMap<String, String>) -> Result<(), String> {
        if self.parallelism.is_some_and(|value| value != 1) {
            return Err(
                "unsupported_target: original measured compiler parallelism must be one".into(),
            );
        }
        for (name, value) in &self.environment {
            if !(name == "GOMAXPROCS" && value == "1"
                || name == "GOTMPDIR" && Path::new(value).is_absolute() && !value.contains('\0')
                || name == "GOWASM" && matches!(value.as_str(), "" | "satconv,signext"))
            {
                return Err("unsupported_target: unsupported scoped compiler control".into());
            }
            environment.insert(name.clone(), value.clone());
        }
        Ok(())
    }
}

pub struct Prepared {
    pub auxiliary: BTreeMap<String, Vec<u8>>,
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

pub(crate) fn retain_observations(
    campaign: &Path,
    prepared: &Prepared,
    record: &str,
) -> Result<(), String> {
    observation_sidecar::publish(campaign, prepared, record)
}

pub(crate) fn open_observations_from_receipt(
    campaign: &Path,
    prepared: &Prepared,
    record: &str,
    receipt_sha256: &str,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    observation_sidecar::open_from_receipt(campaign, prepared, record, receipt_sha256)
}

pub(crate) fn open_observations(
    campaign: &Path,
    prepared: &Prepared,
    record: &str,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    observation_sidecar::open(campaign, prepared, record)
}

pub(crate) fn open_observations_at(
    root: &fs::File,
    prepared: &Prepared,
    record: &str,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    observation_sidecar::open_at(root, prepared, record)
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
    prepare_with_compiler_controls(options, &CompilerControls::default())
}

pub(crate) fn prepare_with_compiler_controls(
    options: &ExploreOptions,
    controls: &CompilerControls,
) -> Result<Prepared, String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("overall_timeout: preparation deadline exceeds clock range")?;
    let result = prepare_before_deadline(options, controls, deadline, false);
    process::check_deadline(deadline)?;
    result
}

pub(crate) fn prepare_runtime_qualification(options: &ExploreOptions) -> Result<Prepared, String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("runtime qualification preparation deadline exceeds clock range")?;
    let result = prepare_before_deadline(options, &CompilerControls::default(), deadline, true);
    process::check_deadline(deadline)?;
    result
}

/// Prepares like [`prepare`] and also returns the compiler arguments of every
/// command this preparation actually ran; a compile-cache hit runs no build.
pub(crate) fn prepare_observed(
    options: &ExploreOptions,
) -> Result<(Prepared, Vec<Vec<String>>), String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("overall_timeout: preparation deadline exceeds clock range")?;
    let result =
        prepare_observed_before_deadline(options, &CompilerControls::default(), deadline, false);
    process::check_deadline(deadline)?;
    result
}

fn prepare_before_deadline(
    options: &ExploreOptions,
    controls: &CompilerControls,
    deadline: Instant,
    runtime_qualification: bool,
) -> Result<Prepared, String> {
    prepare_observed_before_deadline(options, controls, deadline, runtime_qualification)
        .map(|(prepared, _)| prepared)
}

fn prepare_observed_before_deadline(
    options: &ExploreOptions,
    controls: &CompilerControls,
    deadline: Instant,
    runtime_qualification: bool,
) -> Result<(Prepared, Vec<Vec<String>>), String> {
    if options.target_kind == "exec" {
        return Ok((prepared_exec(options, &installation(options)?)?, Vec::new()));
    }
    let mut recipe = if controls.environment.is_empty() && controls.parallelism.is_none() {
        driver::prepare(options, deadline)?
    } else {
        driver::prepare_with_compiler_controls(options, controls, deadline)?
    };
    if runtime_qualification {
        recipe.configuration = crate::runtime_qualification::configuration()?;
    }
    let compile_inputs = recipe.compilation_inputs();
    let compile_key = compilation_cache::key(&compile_inputs)?;
    if let Some(cached) =
        compilation_cache::load(&recipe.installed.cache_root, &compile_key, &compile_inputs)?
    {
        let commands = compiler_arguments(&recipe);
        let mut prepared = recipe.admit(cached.module, options.target_args.clone(), deadline)?;
        // Cached compilation observations stay original; current admission stays fresh.
        prepared
            .auxiliary
            .retain(|path, _| !path.starts_with("target/preparation-inputs/compile/"));
        prepared.auxiliary.extend(cached.auxiliary);
        return Ok((prepared, commands));
    }
    let cache_directory =
        compilation_cache::build_directory(&recipe.installed.cache_root, &compile_key)?;
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
        cache_directory
            .join("objects")
            .to_string_lossy()
            .into_owned(),
    );
    let command = process::observed_command(
        &recipe.installed.compiler_path,
        &build,
        &recipe.directory,
        &compilation_environment,
        &temporary.0,
        4 << 20,
        deadline,
    )?;
    recipe
        .observations
        .as_mut()
        .ok_or("native compilation observations missing")?
        .record("compile-module", command, &recipe.installed.cache_root)?;
    let module = read(&output, MAXIMUM)?;
    let module_mode = fs::metadata(&output).map_err(failure)?.permissions().mode() & 0o777;
    let cache_root = recipe.installed.cache_root.clone();
    let commands = compiler_arguments(&recipe);
    let mut prepared = recipe.admit(module, options.target_args.clone(), deadline)?;
    let auxiliary = prepared
        .auxiliary
        .iter()
        .filter(|(path, _)| path.starts_with("target/preparation-inputs/compile/"))
        .map(|(path, raw)| (path.clone(), raw.clone()))
        .collect();
    let cached = compilation_cache::publish_with_mode(
        &cache_root,
        &compile_key,
        &compile_inputs,
        &prepared.module,
        module_mode,
        auxiliary,
        || process::check_deadline(deadline),
    )?;
    prepared
        .auxiliary
        .retain(|path, _| !path.starts_with("target/preparation-inputs/compile/"));
    prepared.auxiliary.extend(cached.auxiliary);
    Ok((prepared, commands))
}

fn compiler_arguments(recipe: &driver::Recipe) -> Vec<Vec<String>> {
    recipe
        .observations
        .as_ref()
        .map(observations::Observations::compiler_arguments)
        .unwrap_or_default()
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

pub(crate) fn configuration(
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
        world_transition_bytes: Some(options.world_transition_limit),
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
        auxiliary: BTreeMap::new(),
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

#[cfg(test)]
mod receiving_compilation_contract {
    use super::*;
    fn recipe() -> driver::Recipe {
        let mut recipe = driver::collection::tests::recipe("probe");
        recipe.inputs = json!({"compiler":recipe.compiler,"compiler_path":recipe.installed.compiler_path,
            "compiler_sha256":recipe.installed.compiler_sha256,"go_version":recipe.installed.go_version,
            "sources":recipe.sources,"runtime":"a".repeat(64),"runtime_implementation":"b".repeat(64),
            "kind":"go-test","source":"./probe","directory":recipe.directory,"tags":[],
            "build_flags":["-mod=readonly","-trimpath","-buildvcs=false","-pgo=off"],"environment":{"GOOS":"wasip1","GOARCH":"wasm","GOWASM":""},
            "configuration":recipe.configuration,"arguments":["-test.run=^Original$"],"helper":"c".repeat(64),"host":"d".repeat(64),"soft_assertion_report_only":false});
        recipe
    }
    #[test]
    fn receiving_compilation_key_excludes_exact_five_execution_only_changes_but_full_provenance_changes()
     {
        for changed in [
            "selector",
            "wasi-config",
            "helper",
            "fuel",
            "soft-assertion",
        ] {
            let mut recipe = recipe();
            let original = recipe.compilation_inputs();
            let full = canonical::digest(&canonical::encode(&recipe.inputs).unwrap());
            match changed {
                "selector" => recipe.inputs["arguments"] = json!(["-test.run=^Other$"]),
                "wasi-config" => {
                    recipe.inputs["configuration"]["environment"]["ONLY"] = json!("new");
                }
                "helper" => recipe.inputs["helper"] = json!("f".repeat(64)),
                "fuel" => recipe.inputs["configuration"]["limits"]["fuel"] = json!(999),
                "soft-assertion" => recipe.inputs["soft_assertion_report_only"] = json!(true),
                _ => unreachable!(),
            }
            assert_eq!(recipe.compilation_inputs(), original, "{changed}");
            assert_ne!(
                canonical::digest(&canonical::encode(&recipe.inputs).unwrap()),
                full,
                "{changed}"
            );
        }
    }
    #[test]
    fn receiving_compilation_key_binds_source_tags_compiler_tools_headers_runtime_overlay_environment_kind_and_source()
     {
        for changed in [
            "source-bytes",
            "tags",
            "compiler",
            "tool",
            "header",
            "runtime",
            "runtime-implementation",
            "overlay",
            "environment",
            "kind",
            "source",
        ] {
            let mut recipe = recipe();
            let original = compilation_cache::key(&recipe.compilation_inputs()).unwrap();
            match changed {
                "source-bytes" => {
                    recipe.inputs["sources"]["/source/probe/test.go"] = json!("f".repeat(64));
                }
                "tags" => recipe.inputs["tags"] = json!(["changed"]),
                "compiler" => recipe.inputs["compiler_sha256"] = json!("f".repeat(64)),
                "tool" => {
                    recipe.inputs["compiler"]["/pinned/pkg/tool/compile"] = json!("f".repeat(64));
                }
                "header" => {
                    recipe.inputs["compiler"]["/pinned/pkg/include/textflag.h"] =
                        json!("f".repeat(64));
                }
                "runtime" => recipe.inputs["runtime"] = json!("f".repeat(64)),
                "runtime-implementation" => {
                    recipe.inputs["runtime_implementation"] = json!("f".repeat(64));
                }
                "overlay" => {
                    recipe.inputs["compiler"]["/cache/runtime-overlay/overlay.json"] =
                        json!("f".repeat(64));
                }
                "environment" => recipe.inputs["environment"]["GOWASM"] = json!("satconv,signext"),
                "kind" => recipe.inputs["kind"] = json!("go-run"),
                "source" => recipe.inputs["source"] = json!("./different"),
                _ => unreachable!(),
            }
            assert_ne!(
                compilation_cache::key(&recipe.compilation_inputs()).unwrap(),
                original,
                "{changed}"
            );
        }
    }
    #[test]
    fn receiving_measurement_compiler_controls_are_scoped_and_default_environment_is_unchanged() {
        let mut baseline = BTreeMap::from([
            ("GOFLAGS".into(), String::new()),
            ("GOOS".into(), "wasip1".into()),
        ]);
        let mut environment = baseline.clone();
        CompilerControls::default().apply(&mut environment).unwrap();
        assert_eq!(environment, baseline);
        let controls = CompilerControls {
            environment: BTreeMap::from([
                ("GOMAXPROCS".into(), "1".into()),
                ("GOTMPDIR".into(), "/private/compiler-temp".into()),
                ("GOWASM".into(), "satconv,signext".into()),
            ]),
            parallelism: Some(1),
        };
        controls.apply(&mut environment).unwrap();
        assert_eq!(environment["GOMAXPROCS"], "1");
        assert_eq!(environment["GOTMPDIR"], "/private/compiler-temp");
        assert_eq!(environment["GOWASM"], "satconv,signext");
        assert_eq!(baseline.len(), 2);
        for (name, value) in [
            ("GOFLAGS", "-race"),
            ("GOMAXPROCS", "2"),
            ("GOTMPDIR", "relative"),
            ("GOWASM", "other"),
            ("SECRET_TOKEN", "secret"),
        ] {
            let controls = CompilerControls {
                environment: BTreeMap::from([(name.into(), value.into())]),
                parallelism: Some(1),
            };
            assert!(controls.apply(&mut baseline.clone()).is_err(), "{name}");
        }
        assert!(
            CompilerControls {
                environment: BTreeMap::new(),
                parallelism: Some(2)
            }
            .apply(&mut baseline)
            .is_err()
        );
    }
}
