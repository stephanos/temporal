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

mod inventory;
mod process;

const MAXIMUM: u64 = 512 << 20;

pub struct Prepared {
    pub module: Vec<u8>,
    pub configuration: Configuration,
    pub toolchain: String,
    pub choice_identity: Option<linked::ChoiceIdentity>,
    pub scenario: Option<Vec<u8>>,
    pub provenance: Vec<u8>,
    pub network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
    pub soft_assertion_report_only: bool,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Installation {
    pub schema: String,
    pub compiler_path: PathBuf,
    pub compiler_sha256: String,
    pub go_version: String,
    pub runtime_root: PathBuf,
    #[serde(default)]
    pub overlay_module_root: Option<PathBuf>,
    pub cache_root: PathBuf,
    pub profile: String,
    pub fuel: u64,
    pub memory_bytes: u64,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub runtime_implementation_sha256: Option<String>,
    pub model: Value,
    pub namespace: linked::NamespaceImage,
    pub storage: Option<linked::StorageInput>,
    #[serde(default)]
    pub network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
    #[serde(default)]
    pub build_tags: Vec<String>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Provenance {
    schema: String,
    inputs: Value,
    build_key: String,
    module_sha256: String,
    configuration: Configuration,
    choice_identity: Option<linked::ChoiceIdentity>,
    scenario: Option<Vec<u8>>,
    network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Overlay {
    path: PathBuf,
    identity: String,
    implementation_sha256: String,
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

fn executable(path: &Path, expected: &str) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt as _;
    let meta = fs::symlink_metadata(path).map_err(failure)?;
    if !path.is_absolute()
        || !meta.is_file()
        || meta.permissions().mode() & 0o111 == 0
        || !canonical::valid_digest(expected)
        || canonical::digest(&read(path, MAXIMUM)?) != expected
    {
        return Err(failure("pinned compiler identity mismatch"));
    }
    Ok(())
}

fn installation(options: &ExploreOptions) -> Result<Installation, String> {
    let path = if let Some(path) = &options.installation {
        path.clone()
    } else if let Some(path) = std::env::var_os("GOMAD_WASM_INSTALLATION").filter(|s| !s.is_empty())
    {
        PathBuf::from(path)
    } else {
        std::env::current_exe()
            .map_err(failure)?
            .with_file_name("gomad-wasm.json")
    };
    let installed: Installation =
        serde_json::from_slice(&read(&path, 16 << 20)?).map_err(failure)?;
    if installed.schema != "gomad.rust-wasm-installation/v1"
        || installed.helper_sha256 != linked::IMPLEMENTATION_SHA256
        || installed.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || !installed.cache_root.is_absolute()
        || installed.fuel == 0
        || installed.memory_bytes == 0
        || !matches!(installed.go_version.as_str(), "go1.27.0" | "go1.27.1")
    {
        return Err(failure(
            "unsupported Rust installation or pinned implementation mismatch",
        ));
    }
    match installed.profile.as_str() {
        "gomad.wasi-cooperative/v1"
            if installed.go_version == "go1.27.1"
                && installed.runtime_root.is_absolute()
                && installed.runtime_implementation_sha256.as_deref()
                    == Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")) => {}
        "gomad.wasi-stock-exploratory/v1"
            if installed.runtime_implementation_sha256.is_none()
                && installed.runtime_root.as_os_str().is_empty() => {}
        _ => return Err(failure("unsupported runtime profile or implementation")),
    }
    executable(&installed.compiler_path, &installed.compiler_sha256)?;
    Ok(installed)
}

/// # Errors
/// Refuses incompatible installations, unsupported targets and changed build inputs.
pub fn prepare(options: &ExploreOptions) -> Result<Prepared, String> {
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("overall_timeout: preparation deadline exceeds clock range")?;
    let result = prepare_before_deadline(options, deadline);
    process::check_deadline(deadline)?;
    result
}

#[expect(
    clippy::too_many_lines,
    reason = "Source admission keeps inventory and post-build verification in one ordered transaction"
)]
fn prepare_before_deadline(
    options: &ExploreOptions,
    deadline: Instant,
) -> Result<Prepared, String> {
    process::check_deadline(deadline)?;
    let installed = installation(options)?;
    if options.target_kind == "exec" {
        return prepared_exec(options, &installed);
    }
    if options.soft_assertion_report_only && options.target_kind != "go-test" {
        return Err("unsupported_target: soft assertion report-only requires go-test".into());
    }
    if !matches!(options.target_kind.as_str(), "go-run" | "go-test")
        || !options.provenance.is_empty()
    {
        return Err("unsupported_target: unsupported source preparation request".into());
    }
    let scenario = options
        .scenario
        .as_deref()
        .map(super::scenario::normalize)
        .transpose()?;
    let cooperative = installed.profile == "gomad.wasi-cooperative/v1";
    if cooperative && !options.choices && scenario.is_none()
        || !cooperative && (options.choices || options.diagnostics || scenario.is_some())
    {
        return Err("unsupported_target: choices/scenario do not match installed profile".into());
    }
    let directory = fs::canonicalize(&options.working_dir).map_err(failure)?;
    let temporary = process::Temporary::new(&installed.cache_root)?;
    let mut environment = process::environment()?;
    let version = process::command(
        &installed.compiler_path,
        &["version".into()],
        &directory,
        &environment,
        &temporary.0,
        64 << 10,
        deadline,
    )?;
    if !String::from_utf8_lossy(&version).contains(&format!("{} ", installed.go_version)) {
        return Err(failure("compiler version does not match installation"));
    }
    let (mut compiler, root) = inventory::compiler(
        &installed.compiler_path,
        &directory,
        &environment,
        &temporary.0,
        deadline,
    )?;
    let mut replacements = BTreeMap::new();
    let mut runtime = None;
    let mut overlay = None;
    if cooperative {
        let cache = process::command(
            &installed.compiler_path,
            &["env".into(), "GOMODCACHE".into()],
            &directory,
            &environment,
            &temporary.0,
            64 << 10,
            deadline,
        )?;
        let cache = PathBuf::from(String::from_utf8_lossy(&cache).trim());
        if root.starts_with(fs::canonicalize(&cache).unwrap_or(cache)) {
            return Err(failure("cooperative compiler must be outside module cache"));
        }
        environment.insert("GOROOT".into(), root.to_string_lossy().into_owned());
        let module_root = installed
            .overlay_module_root
            .as_deref()
            .or_else(|| {
                installed
                    .runtime_root
                    .parent()
                    .and_then(Path::parent)
                    .and_then(Path::parent)
            })
            .ok_or_else(|| failure("overlay module owner is required"))?;
        if !module_root.is_absolute() {
            return Err(failure("absolute overlay module owner is required"));
        }
        let mut host_environment = environment.clone();
        for name in ["GOOS", "GOARCH", "GOEXPERIMENT"] {
            host_environment.remove(name);
        }
        let result = process::command(
            &installed.compiler_path,
            &[
                "run".into(),
                "-mod=readonly".into(),
                "./internal/backends/wasm/toolchain/cmd/overlay".into(),
                root.to_string_lossy().into_owned(),
                installed.runtime_root.to_string_lossy().into_owned(),
                installed
                    .cache_root
                    .join("runtime-overlay")
                    .to_string_lossy()
                    .into_owned(),
            ],
            module_root,
            &host_environment,
            &temporary.0,
            4 << 20,
            deadline,
        )?;
        let generated: Overlay = serde_json::from_slice(&result).map_err(failure)?;
        if !canonical::valid_digest(&generated.identity)
            || generated.implementation_sha256 != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
        {
            return Err(failure(
                "invalid overlay identity or runtime implementation changed",
            ));
        }
        let value: Value =
            serde_json::from_slice(&read(&generated.path, 16 << 20)?).map_err(failure)?;
        replacements = serde_json::from_value(
            value
                .get("Replace")
                .cloned()
                .ok_or_else(|| failure("overlay replacements missing"))?,
        )
        .map_err(failure)?;
        for (original, replacement) in &replacements {
            for path in [Path::new(original), Path::new(replacement)] {
                if path.is_file() {
                    compiler.insert(path.into(), canonical::digest(&read(path, MAXIMUM)?));
                }
            }
        }
        for name in [
            "internal/model/choice/schema/choicewire.json",
            "internal/model/choice/schema/choicewire.go.tmpl",
            "internal/model/choice/schema/choicewire_runtime.go.tmpl",
            "internal/model/choice/schema/choicewire_test.go.tmpl",
            "internal/model/choice/trace.go",
            "internal/model/choice/tape.go",
            "internal/backends/wasm/toolchain/overlay.go",
            "internal/backends/wasm/toolchain/runtime_wasm.go.txt",
            "internal/backends/wasm/toolchain/runtime_inspect_wasm.go.txt",
            "internal/backends/wasm/toolchain/net_stream_wasm.go.txt",
            "internal/backends/wasm/toolchain/net_stream_status_generated.go.txt",
            "internal/backends/wasm/toolchain/cmd/overlay/main.go",
        ] {
            let path = module_root.join(name);
            compiler.insert(path.clone(), canonical::digest(&read(&path, MAXIMUM)?));
        }
        for path in [
            &generated.path,
            &installed.runtime_root.join("gomad_semantics.go.txt"),
            &installed
                .runtime_root
                .join("gomad_choicewire_generated.go.txt"),
        ] {
            compiler.insert(path.clone(), canonical::digest(&read(path, MAXIMUM)?));
        }
        runtime = Some(generated.identity);
        overlay = Some(generated.path);
    }
    let mut tags = options.build_tags.clone();
    tags.extend(installed.build_tags.iter().cloned());
    tags.sort();
    tags.dedup();
    let mut flags = vec!["-mod=readonly".into(), "-pgo=off".into()];
    if let Some(path) = &overlay {
        flags.extend(["-overlay".into(), path.to_string_lossy().into_owned()]);
    }
    if !tags.is_empty() {
        flags.extend(["-tags".into(), tags.join(",")]);
    }
    let mut list = vec!["list".into()];
    list.extend(flags.clone());
    list.extend(["-deps".into(), "-json".into()]);
    if options.target_kind == "go-test" {
        list.push("-test".into());
    }
    list.push(options.source.clone());
    let sources = inventory::sources(
        &process::command(
            &installed.compiler_path,
            &list,
            &directory,
            &environment,
            &temporary.0,
            128 << 20,
            deadline,
        )?,
        &root.join("pkg/include"),
        &replacements,
    )?;
    let mut configuration = configuration(options, &installed)?;
    let build_environment = environment
        .iter()
        .filter(|(name, _)| {
            matches!(
                name.as_str(),
                "GOENV"
                    | "GOFLAGS"
                    | "GOTOOLCHAIN"
                    | "GOWORK"
                    | "GOOS"
                    | "GOARCH"
                    | "CGO_ENABLED"
                    | "GOEXPERIMENT"
                    | "GOCACHEPROG"
                    | "GOWASM"
                    | "GODEBUG"
            )
        })
        .collect::<BTreeMap<_, _>>();
    let inputs = json!({"schema":"gomad.rust-wasm-build-inputs/v1", "compiler":compiler,"compiler_sha256":installed.compiler_sha256,
        "go_version":installed.go_version,"sources":sources,"runtime":runtime,"runtime_implementation":installed.runtime_implementation_sha256,
        "helper":linked::IMPLEMENTATION_SHA256,"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"kind":options.target_kind,"source":options.source,
        "directory":directory,"arguments":options.target_args,"tags":tags,"configuration":configuration,"scenario":scenario,"network":installed.network,
        "build_flags":["-mod=readonly","-trimpath","-buildvcs=false","-pgo=off"],"environment":build_environment,"soft_assertion_report_only":options.soft_assertion_report_only});
    let toolchain = canonical::digest(&canonical::encode(&inputs)?);
    let output = temporary.0.join("module.wasm");
    let mut build = if options.target_kind == "go-test" {
        vec!["test".into(), "-c".into()]
    } else {
        vec!["build".into()]
    };
    build.extend(flags);
    build.extend([
        "-trimpath".into(),
        "-buildvcs=false".into(),
        "-o".into(),
        output.to_string_lossy().into_owned(),
        options.source.clone(),
    ]);
    environment.insert(
        "GOCACHE".into(),
        temporary.0.join("objects").to_string_lossy().into_owned(),
    );
    process::command(
        &installed.compiler_path,
        &build,
        &directory,
        &environment,
        &temporary.0,
        4 << 20,
        deadline,
    )?;
    let module = read(&output, MAXIMUM)?;
    let after = inventory::sources(
        &process::command(
            &installed.compiler_path,
            &list,
            &directory,
            &environment,
            &temporary.0,
            128 << 20,
            deadline,
        )?,
        &root.join("pkg/include"),
        &replacements,
    )?;
    if sources != after {
        return Err(failure(
            "selected source inventory changed during compilation",
        ));
    }
    inventory::verify(&compiler)?;
    inventory::verify(&sources)?;
    executable(&installed.compiler_path, &installed.compiler_sha256)?;
    let (after_compiler, after_root) = inventory::compiler(
        &installed.compiler_path,
        &directory,
        &environment,
        &temporary.0,
        deadline,
    )?;
    if after_root != root
        || after_compiler
            .iter()
            .any(|(path, digest)| compiler.get(path) != Some(digest))
    {
        return Err(failure("compiler inventory changed during compilation"));
    }
    let choice_identity = if cooperative {
        let implementation = choice_implementation(&toolchain)?;
        inventory::verify(&compiler)?;
        configuration.runtime_sha256 = Some(format!("{implementation:x}"));
        Some(linked::ChoiceIdentity {
            target_sha256: Sha256::digest(&module).into(),
            toolchain_build_key: toolchain.clone(),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: implementation.into(),
        })
    } else {
        None
    };
    let provenance = canonical::encode(&Provenance {
        schema: "gomad.rust-wasm-provenance/v1".into(),
        inputs,
        build_key: toolchain.clone(),
        module_sha256: canonical::digest(&module),
        configuration,
        choice_identity,
        scenario,
        network: installed.network,
    })?;
    let decoded: Provenance = serde_json::from_slice(&provenance).map_err(failure)?;
    Ok(Prepared {
        module,
        configuration: decoded.configuration,
        toolchain,
        choice_identity: decoded.choice_identity,
        scenario: decoded.scenario,
        provenance,
        network: decoded.network,
        soft_assertion_report_only: options.soft_assertion_report_only,
    })
}

fn choice_implementation(key: &str) -> Result<sha2::digest::Output<Sha256>, String> {
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
    let mut environment = BTreeMap::new();
    for entry in &options.environment {
        let (name, value) = entry
            .split_once('=')
            .ok_or_else(|| "unsupported_target: environment requires NAME=VALUE".to_owned())?;
        if name.is_empty()
            || name.contains('\0')
            || value.contains('\0')
            || environment.insert(name.into(), value.into()).is_some()
        {
            return Err("unsupported_target: invalid or duplicate environment entry".into());
        }
    }
    let working =
        String::from_utf8(installed.namespace.working_directory.clone()).map_err(failure)?;
    environment.insert("PWD".into(), working.clone());
    environment.insert("TZ".into(), "UTC".into());
    environment.insert("GOMAD_IO_PROFILE".into(), "wasm".into());
    if options.choices {
        environment.insert(
            "GOMAD_CHOICE_PROFILE".into(),
            "gomad-choice-trace/v3".into(),
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
    let choice_limit = if options.scenario.is_some() && options.choice_bytes == 0 {
        8 << 20
    } else {
        options.choice_bytes.max(160)
    };
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
    let evidence = options
        .choice_bytes
        .checked_add(
            options
                .output_limit
                .checked_mul(2)
                .ok_or("capacity: evidence bounds overflow")?,
        )
        .and_then(|v| v.checked_add(128 << 20))
        .ok_or("capacity: evidence bounds overflow")?;
    Ok(Configuration {
        engine: linked::CONFIGURATION.into(),
        profile: installed.profile.clone(),
        limits: Limits {
            fuel: installed.fuel,
            memory_bytes: installed.memory_bytes,
            wall_millis: u64::try_from(options.execution_timeout.as_millis()).map_err(failure)?,
            output_bytes: options.output_limit,
            evidence_bytes: evidence,
        },
        arguments: std::iter::once("gomad-target".into())
            .chain(options.target_args.iter().cloned())
            .collect(),
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
