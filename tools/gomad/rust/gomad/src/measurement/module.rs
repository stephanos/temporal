use super::inputs::{CheckoutProof, Invocation};
use crate::{campaign::prepare as preparation, cli::explore::ExploreOptions, evidence::canonical};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs,
    io::{Read, Write},
    path::{Path, PathBuf},
    time::Duration,
};
const COOPERATIVE: &str = "gomad.wasi-cooperative/v1";
#[derive(Clone, Debug)]
pub enum Selection {
    Runtime,
    Activity,
    Workflow,
    Selected(String),
    Cluster,
    Target {
        source: String,
        cwd: String,
        run: String,
    },
}
#[derive(Debug, PartialEq, Eq)]
pub struct Bounds {
    pub fuel: u64,
    pub memory_bytes: u64,
    pub timeout: Duration,
    pub output_bytes: u64,
}
pub fn resource_bounds(selection: &Selection) -> Bounds {
    if matches!(selection, Selection::Runtime) {
        Bounds {
            fuel: 1 << 40,
            memory_bytes: 256 << 20,
            timeout: Duration::from_secs(180),
            output_bytes: 1 << 20,
        }
    } else {
        Bounds {
            fuel: 1 << 50,
            memory_bytes: 4 << 30,
            timeout: Duration::from_secs(1800),
            output_bytes: 256 << 20,
        }
    }
}

pub struct Prepared {
    pub guest: preparation::Prepared,
    pub module_path: PathBuf,
    pub provenance_path: PathBuf,
    pub installation_path: PathBuf,
    pub module_metadata: Value,
    pub preparation_arguments: Vec<String>,
    pub scenario_path: Option<PathBuf>,
}
impl Prepared {
    pub fn native_input(&self, seed: u64) -> Result<Value, String> {
        if self.guest.scenario.is_some() {
            return Err("cluster preparation requires actual broker scenario assembly".into());
        }
        if read(&self.module_path, 512 << 20)? != self.guest.module
            || read(&self.provenance_path, 16 << 20)? != self.guest.provenance
        {
            return Err("retained native module or provenance changed".into());
        }
        let config = &self.guest.configuration;
        let model=config.model.as_ref().map(|value|{
            let mut value=value.clone();let object=value.as_object_mut().ok_or("invalid prepared native model")?;
            if object.insert("seed".into(),seed.into()).is_some(){return Err("prepared native model must derive seed".to_string());}
            let checked:gomad_wasmhost::linked::ModelConfig=serde_json::from_value(value.clone()).map_err(|error|error.to_string())?;checked.validate().map_err(|error|error.to_string())?;
            let mut environment=config.environment.iter().map(|(key,value)|format!("{key}={value}")).collect::<Vec<_>>();
            if config.derive_seed_environment {if config.environment.contains_key("GOMADSEED"){return Err("prepared native seed environment must derive from selection".into());}environment.push(format!("GOMADSEED={seed}"));}
            Ok(json!({"configuration":value,"arguments":config.arguments,"environment":environment,"replay_trace":null}))
        }).transpose()?;
        Ok(
            json!({"kind":"standalone","guest":{"module":self.module_path,"module_sha256":canonical::digest(&self.guest.module),"limits":config.limits,"model":model,"namespace":config.namespace,"storage":config.storage},"network":config.network}),
        )
    }
}
fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let mut bytes = Vec::new();
    fs::File::open(path)
        .map_err(|error| error.to_string())?
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > limit {
        return Err("native module input exceeds bound".into());
    }
    Ok(bytes)
}
fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())
}

pub fn temporal_configuration(
    repo: &Path,
    cwd: &str,
    arguments: &[String],
) -> Result<Value, String> {
    let package = cwd
        .strip_prefix("/workspace/")
        .or_else(|| (cwd == "/workspace").then_some("."))
        .ok_or("Temporal measurement working directory must be beneath /workspace")?;
    let mut config =
        crate::install::config::build(COOPERATIVE, None, Some((repo, package)))?.original;
    let args = config["Args"]
        .as_array_mut()
        .ok_or("Temporal installation configuration has no arguments")?;
    args.extend(arguments.iter().map(|argument| json!(argument)));
    Ok(config)
}
pub fn runtime_configuration() -> Result<Value, String> {
    let mut config = crate::install::config::build(COOPERATIVE, None, None)?.original;
    config["Args"] = json!(["guest", "literal $arg"]);
    config["Environment"] = json!(["PWD=/workspace", "TZ=UTC", "ONLY=captured"]);
    let mut entropy = vec![0_u8; 32];
    entropy[..3].copy_from_slice(&[1, 2, 3]);
    config["EntropyKey"] = json!(entropy);
    config["Limits"]["OutputBytes"] = json!(1_u64 << 20);
    config["Limits"]["FilesystemBytes"] = json!(8_u64 << 20);
    config["Clock"]["ReadStepNanos"] = json!(0);
    config["Limits"]["Calls"] = json!(1_u64 << 20);
    config["Limits"]["TranscriptBytes"] = json!(64_u64 << 20);
    Ok(config)
}

pub fn diagnostic_module_commands(
    compiler: &str,
    overlay: &str,
    output: &str,
    target: &str,
    kind: &str,
    tags: &[String],
) -> Result<(Vec<String>, Vec<String>), String> {
    if !matches!(kind, "test" | "build") {
        return Err(format!("diagnostic module kind {kind:?}"));
    }
    let mut list = vec![
        compiler.into(),
        "list".into(),
        "-p=1".into(),
        "-deps".into(),
    ];
    let mut build = vec![compiler.into(), kind.into()];
    if kind == "test" {
        list.push("-test".into());
        build.push("-c".into());
    }
    list.extend(["-json".into(), "-overlay".into(), overlay.into()]);
    build.extend(["-p=1".into(), "-trimpath".into()]);
    if kind == "build" {
        build.push("-buildvcs=false".into());
    }
    build.extend([
        "-overlay".into(),
        overlay.into(),
        "-o".into(),
        output.into(),
    ]);
    if !tags.is_empty() {
        list.extend(["-tags".into(), tags.join(",")]);
        build.extend(["-tags".into(), tags.join(",")]);
    }
    list.push(target.into());
    build.push(target.into());
    Ok((list, build))
}
pub(crate) fn compiler_controls(
    call: &Invocation,
) -> Result<preparation::CompilerControls, String> {
    let mut environment = BTreeMap::new();
    environment.insert("GOMAXPROCS".into(), "1".into());
    for entry in &call.environ {
        if let Some((key, value)) = entry.split_once('=')
            && key == "GOTMPDIR"
        {
            if !Path::new(value).is_absolute() {
                return Err("absolute compiler GOTMPDIR required".into());
            }
            environment.insert(key.into(), value.into());
            break;
        }
    }
    Ok(preparation::CompilerControls {
        environment,
        parallelism: Some(1),
    })
}

pub fn prepare(
    proof: &CheckoutProof,
    call: &Invocation,
    root: &Path,
    selection: Selection,
) -> Result<Prepared, String> {
    if matches!(selection, Selection::Cluster) {
        let scenario = read(&root.join("scenario.json"), 256 << 10)?;
        return prepare_scenario(proof, call, root, "./tests", scenario);
    }
    prepare_selection(proof, call, root, selection, None)
}
pub fn prepare_scenario(
    proof: &CheckoutProof,
    call: &Invocation,
    root: &Path,
    source: &str,
    scenario: Vec<u8>,
) -> Result<Prepared, String> {
    prepare_selection(
        proof,
        call,
        root,
        Selection::Cluster,
        Some((source.into(), scenario)),
    )
}
fn options(
    root: &Path,
    directory: &Path,
    installation: &Path,
    kind: &str,
    source: &str,
    args: &[String],
    tags: &[String],
    runtime: bool,
    timeout: Duration,
    scenario: Option<Vec<u8>>,
) -> Result<(ExploreOptions, Vec<String>, Option<PathBuf>), String> {
    let timeout = format!("{}ns", timeout.as_nanos());
    let mut arguments = vec![
        "--working-dir".into(),
        directory.to_string_lossy().into_owned(),
        "--artifacts".into(),
        root.join("artifacts").to_string_lossy().into_owned(),
        "--execution-timeout".into(),
        timeout,
        "--overall-timeout".into(),
        "3600s".into(),
        "--parallel".into(),
        "1".into(),
        "--output-limit".into(),
        if runtime {
            (1_u64 << 20).to_string()
        } else {
            (256_u64 << 20).to_string()
        },
    ];
    if scenario.is_none() {
        arguments.extend([
            "--choices".into(),
            "--choice-bytes".into(),
            (64_u64 << 20).to_string(),
        ]);
    }
    for tag in tags {
        arguments.extend(["--build-tag".into(), tag.clone()]);
    }
    arguments.extend([kind.into(), source.into(), "--".into()]);
    arguments.extend_from_slice(args);
    let scenario_path = scenario
        .as_ref()
        .map(|_| root.join("preparation-scenario.json"));
    if let (Some(path), Some(bytes)) = (&scenario_path, &scenario) {
        write(path, bytes)?;
        arguments.splice(
            0..0,
            ["--scenario".into(), path.to_string_lossy().into_owned()],
        );
    }
    let mut json = false;
    let mut options = crate::cli::explore::parse(&arguments, &mut json)?;
    options.installation = Some(installation.into());
    options.scenario = scenario;
    Ok((options, arguments, scenario_path))
}
fn prepare_selection(
    proof: &CheckoutProof,
    call: &Invocation,
    root: &Path,
    selection: Selection,
    scenario: Option<(String, Vec<u8>)>,
) -> Result<Prepared, String> {
    prepare_selection_controlled(proof, call, root, selection, scenario, "")
}
pub fn prepare_variant(
    proof: &CheckoutProof,
    call: &Invocation,
    root: &Path,
    selection: Selection,
    gowasm: &str,
) -> Result<Prepared, String> {
    if !matches!(gowasm, "" | "satconv,signext") {
        return Err("unexpected diagnostic GOWASM variant".into());
    }
    prepare_selection_controlled(proof, call, root, selection, None, gowasm)
}
fn prepare_selection_controlled(
    proof: &CheckoutProof,
    call: &Invocation,
    root: &Path,
    selection: Selection,
    scenario: Option<(String, Vec<u8>)>,
    gowasm: &str,
) -> Result<Prepared, String> {
    if proof.repo() != call.repo || !root.is_absolute() {
        return Err(
            "native preparation requires selected checkout proof and absolute evidence root".into(),
        );
    }
    let compiler = call
        .inputs
        .get("GOMAD_STOCK_GO")
        .filter(|value| Path::new(value).is_absolute())
        .ok_or("native measurement requires explicit absolute --stock-go")?;
    fs::create_dir_all(root).map_err(|error| error.to_string())?;
    let runtime = matches!(selection, Selection::Runtime);
    let mut bounds = resource_bounds(&selection);
    if call.name == "legacy-workflow-profile" {
        bounds.timeout = Duration::from_secs(180);
    } else if call.name == "failing-suite-profile" {
        bounds.timeout = Duration::from_secs(300);
    }
    let selectors = match &selection {
        Selection::Runtime => vec![],
        Selection::Activity => vec!["-test.run=^TestActivityTestSuite$".into()],
        Selection::Cluster => vec![],
        Selection::Workflow => {
            vec!["-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$".into()]
        }
        Selection::Selected(run) | Selection::Target { run, .. } => {
            vec![format!("-test.run={run}")]
        }
    };
    let config = if runtime {
        runtime_configuration()?
    } else {
        let mut args = selectors;
        args.extend([
            "-persistenceType=sql".into(),
            "-persistenceDriver=sqlite".into(),
        ]);
        temporal_configuration(
            proof.repo(),
            if let Selection::Target { cwd, .. } = &selection {
                cwd
            } else {
                "/workspace/tests"
            },
            &args,
        )?
    };
    let config_path = root.join("guest-config.json");
    write(&config_path, &canonical::encode(&config)?)?;
    let installation_path = root.join("installation.json");
    let cache = call
        .inputs
        .get("GOMAD_WASM_TOOLCHAIN_CACHE")
        .map(PathBuf::from)
        .unwrap_or_else(|| proof.repo().join("tools/gomad/.toolchain"));
    let argv = vec![
        "--compiler".into(),
        compiler.clone(),
        "--cache".into(),
        cache.to_string_lossy().into_owned(),
        "--output".into(),
        installation_path.to_string_lossy().into_owned(),
        "--profile".into(),
        COOPERATIVE.into(),
        "--runtime-root".into(),
        proof
            .repo()
            .join("tools/gomad/internal/model/runtimesource")
            .to_string_lossy()
            .into_owned(),
        "--config".into(),
        config_path.to_string_lossy().into_owned(),
        "--fuel".into(),
        bounds.fuel.to_string(),
        "--memory-bytes".into(),
        bounds.memory_bytes.to_string(),
    ];
    let mut installation_output = Vec::new();
    let status =
        crate::cli::install::run(&argv, &mut installation_output).map_err(|error| error.message)?;
    write(&root.join("installation-output.txt"), &installation_output)?;
    if status != 0 {
        return Err("native guest installation failed".into());
    }
    let directory = if runtime {
        proof.repo().join("tools/gomad")
    } else {
        proof.repo().to_owned()
    };
    let (source, scenario) = scenario.map_or_else(
        || {
            (
                if runtime {
                    "./internal/model/wasi/testdata/runtime_cost".into()
                } else if let Selection::Target { source, .. } = &selection {
                    source.clone()
                } else {
                    "./tests".into()
                },
                None,
            )
        },
        |(source, scenario)| (source, Some(scenario)),
    );
    let args = config["Args"]
        .as_array()
        .map(|args| {
            args.iter()
                .skip(1)
                .map(|arg| {
                    arg.as_str()
                        .map(str::to_owned)
                        .ok_or("native Temporal argument must be a string")
                })
                .collect::<Result<Vec<_>, _>>()
        })
        .transpose()?
        .unwrap_or_default();
    let tags = if runtime {
        vec![]
    } else {
        [
            "disable_grpc_modules",
            "gomad_wasm_slim",
            "sqlite3_dotlk",
            "test_dep",
        ]
        .into_iter()
        .map(str::to_owned)
        .collect()
    };
    let (options, preparation_arguments, scenario_path) = options(
        root,
        &directory,
        &installation_path,
        if runtime { "go-run" } else { "go-test" },
        &source,
        &args,
        &tags,
        runtime,
        bounds.timeout,
        scenario,
    )?;
    let mut controls = compiler_controls(call)?;
    if !gowasm.is_empty() {
        controls.environment.insert("GOWASM".into(), gowasm.into());
    }
    let guest = preparation::prepare_with_compiler_controls(&options, &controls)?;
    let module_path = root.join("module.wasm");
    let provenance_path = root.join("module-provenance.json");
    write(&module_path, &guest.module)?;
    write(&provenance_path, &guest.provenance)?;
    let module_metadata = serde_json::to_value(
        gomad_wasmhost::linked::module_metadata(&guest.module)
            .map_err(|error| error.to_string())?,
    )
    .map_err(|error| error.to_string())?;
    Ok(Prepared {
        guest,
        module_path,
        provenance_path,
        installation_path,
        module_metadata,
        preparation_arguments,
        scenario_path,
    })
}

pub fn source_bindings(
    listed: &[u8],
    compiler: &Path,
    overlay: &Path,
    repo: &Path,
) -> Result<BTreeMap<PathBuf, String>, String> {
    let root = compiler
        .parent()
        .and_then(Path::parent)
        .ok_or("compiler distribution root required")?;
    let replacement: Value =
        serde_json::from_slice(&read(overlay, 16 << 20)?).map_err(|error| error.to_string())?;
    let replacements: BTreeMap<String, String> =
        serde_json::from_value(replacement["Replace"].clone())
            .map_err(|error| error.to_string())?;
    let mut result =
        preparation::inventory::sources(listed, &root.join("pkg/include"), &replacements)?;
    let mut paths = vec![
        compiler.to_owned(),
        root.join("VERSION"),
        overlay.to_owned(),
        repo.join("go.mod"),
        repo.join("go.sum"),
    ];
    for (from, to) in replacements {
        if !to.is_empty() {
            paths.push(to.into());
        }
        let from = PathBuf::from(from);
        if from.try_exists().map_err(|error| error.to_string())? {
            paths.push(from);
        }
    }
    let platform = format!(
        "{}_{}",
        if std::env::consts::OS == "macos" {
            "darwin"
        } else {
            std::env::consts::OS
        },
        match std::env::consts::ARCH {
            "x86_64" => "amd64",
            "aarch64" => "arm64",
            other => other,
        }
    );
    for tool in ["compile", "link", "asm"] {
        paths.push(root.join("pkg/tool").join(&platform).join(tool));
    }
    for path in paths {
        result.insert(path.clone(), canonical::digest(&read(&path, 512 << 20)?));
    }
    Ok(result)
}
pub fn verify_inputs(inputs: &BTreeMap<PathBuf, String>) -> Result<(), String> {
    preparation::inventory::verify(inputs)
}
#[derive(Debug)]
pub struct VariantProof {
    before_module: PathBuf,
    after_module: PathBuf,
    before_sha256: String,
    after_sha256: String,
}
impl VariantProof {
    pub fn module_pair(&self) -> (&str, &str) {
        (&self.before_sha256, &self.after_sha256)
    }
    pub fn matches_requests(&self, before: &Value, after: &Value) -> bool {
        before["kind"] == "standalone"
            && after["kind"] == "standalone"
            && before["guest"]["module"] == json!(self.before_module)
            && after["guest"]["module"] == json!(self.after_module)
            && before["guest"]["module_sha256"] == self.before_sha256
            && after["guest"]["module_sha256"] == self.after_sha256
    }
}
pub fn authenticate_prepared(prepared: &Prepared) -> Result<Value, String> {
    if read(&prepared.module_path, 512 << 20)? != prepared.guest.module
        || read(&prepared.provenance_path, 16 << 20)? != prepared.guest.provenance
    {
        return Err("module variant retained module/provenance changed".into());
    }
    let provenance = crate::qualify::spec::strict(&prepared.guest.provenance)?;
    preparation::typed_provenance(&provenance)?;
    let inputs = &provenance["inputs"];
    if provenance["module_sha256"] != canonical::digest(&prepared.guest.module)
        || provenance["build_key"] != prepared.guest.toolchain
        || canonical::digest(&canonical::encode(inputs)?) != prepared.guest.toolchain
    {
        return Err("module variant build key does not bind exact native inputs".into());
    }
    let metadata = serde_json::to_value(linked_metadata(&prepared.guest.module)?)
        .map_err(|error| error.to_string())?;
    if inputs["module_metadata"] != metadata || provenance["module_metadata"] != metadata {
        return Err("module variant actual metadata mismatch".into());
    }
    crate::install::manifest::executable(
        Path::new(
            inputs["compiler_path"]
                .as_str()
                .ok_or("native module compiler path absent")?,
        ),
        inputs["compiler_sha256"]
            .as_str()
            .ok_or("native module compiler digest absent")?,
    )?;
    for name in ["compiler", "sources"] {
        let files: BTreeMap<PathBuf, String> =
            serde_json::from_value(inputs[name].clone()).map_err(|error| error.to_string())?;
        if files.is_empty() {
            return Err("module variant requires source/compiler inventory".into());
        }
        verify_inputs(&files)?;
    }
    if inputs["host"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || inputs["helper"] != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || inputs["runtime_implementation"] != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
    {
        return Err("module variant implementation/source lineage mismatch".into());
    }
    let identity = prepared
        .guest
        .choice_identity
        .as_ref()
        .ok_or("module variant cooperative identity absent")?;
    if identity.toolchain_build_key != prepared.guest.toolchain
        || provenance["choice_identity"]
            != serde_json::to_value(identity).map_err(|error| error.to_string())?
        || prepared.guest.configuration.runtime_sha256.as_deref()
            != Some(identity.runtime_implementation_sha256().as_str())
    {
        return Err("module variant derived runtime identity mismatch".into());
    }
    Ok(provenance)
}
fn linked_metadata(module: &[u8]) -> Result<gomad_wasmhost::linked::ModuleMetadata, String> {
    gomad_wasmhost::linked::module_metadata(module).map_err(|error| error.to_string())
}
pub fn prove_gowasm_variant(before: &Prepared, after: &Prepared) -> Result<VariantProof, String> {
    let before_provenance = authenticate_prepared(before)?;
    let after_provenance = authenticate_prepared(after)?;
    let mut before_inputs = before_provenance["inputs"].clone();
    let mut after_inputs = after_provenance["inputs"].clone();
    if before_inputs["environment"]["GOWASM"] != ""
        || after_inputs["environment"]["GOWASM"] != "satconv,signext"
    {
        return Err("module variant requires exact empty -> satconv,signext build control".into());
    }
    for inputs in [&mut before_inputs, &mut after_inputs] {
        inputs["environment"]
            .as_object_mut()
            .ok_or("module variant compiler environment absent")?
            .remove("GOWASM");
        inputs
            .as_object_mut()
            .ok_or("module variant inputs absent")?
            .remove("module_metadata");
    }
    if before_inputs != after_inputs {
        return Err("GOWASM module variant changed undeclared native build input".into());
    }
    Ok(VariantProof {
        before_module: before.module_path.clone(),
        after_module: after.module_path.clone(),
        before_sha256: canonical::digest(&before.guest.module),
        after_sha256: canonical::digest(&after.guest.module),
    })
}

#[cfg(test)]
#[path = "module_tests.rs"]
mod tests;
