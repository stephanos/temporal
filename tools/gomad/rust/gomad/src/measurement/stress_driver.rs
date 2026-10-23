use super::{
    build, cluster_driver, host,
    inputs::{CheckoutProof, Invocation},
    retention::ChildLease,
    workloads,
};
use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::Digest as _;
use std::{
    collections::BTreeMap,
    fmt::Write as _,
    fs,
    io::{Read, Write},
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
    time::Duration,
};

mod reader;
pub use reader::read;

const PROFILE: &str = "gomad.wasi-cooperative/v1";
const NAMES: [&str; 4] = ["runtime", "two-cluster", "workflow", "activity"];
const RESULT_SCHEMA: &str = "gomad.native-measurement-stress/v1";

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Controls {
    host_description: String,
    repetitions: usize,
    parallel: usize,
    cpu_workers: usize,
    only: String,
    seed: String,
    axis: String,
}
impl Controls {
    fn parse(call: &Invocation, cpus: usize) -> Result<Self, String> {
        let positive = |name: &str, fallback| {
            let value = input(call, name);
            if value.is_empty() {
                return Ok(fallback);
            }
            value
                .parse::<usize>()
                .ok()
                .filter(|value| (1..=256).contains(value))
                .ok_or_else(|| format!("refused: --{} must be 1..256", super::inputs::flag(name)))
        };
        let result = Self {
            host_description: input(call, "GOMAD_WASM_STRESS_HOST_DESCRIPTION").into(),
            repetitions: positive("GOMAD_WASM_STRESS_REPETITIONS", 3)?,
            parallel: positive("GOMAD_WASM_STRESS_PARALLEL", 2)?,
            cpu_workers: positive(
                "GOMAD_WASM_STRESS_CPU_WORKERS",
                cpus.saturating_sub(1).max(1),
            )?,
            only: input(call, "GOMAD_WASM_STRESS_ONLY").into(),
            seed: input(call, "GOMAD_WASM_STRESS_SEED").into(),
            axis: input(call, "GOMAD_WASM_STRESS_AXIS").into(),
        };
        result.validate()?;
        Ok(result)
    }
    fn validate(&self) -> Result<(), String> {
        let result = self;
        if result.host_description.trim().is_empty() {
            return Err("refused: provide the host identity and quiet-host admission".into());
        }
        if !(2..=256).contains(&result.repetitions)
            || !(2..=256).contains(&result.parallel)
            || !(1..=256).contains(&result.cpu_workers)
        {
            return Err("refused: repetitions and concurrent fan-out must be at least two; CPU workers at most 256".into());
        }
        if (!result.only.is_empty() && !NAMES.contains(&result.only.as_str()))
            || (!result.seed.is_empty()
                && (!result.seed.bytes().all(|byte| byte.is_ascii_digit())
                    || result.seed.parse::<u64>().is_err()))
            || (!result.axis.is_empty()
                && !["reference", "cpu", "concurrent", "layout", "handoff"]
                    .contains(&result.axis.as_str()))
        {
            return Err("refused: invalid stress workload, seed or perturbation axis".into());
        }
        Ok(())
    }
    fn selected(&self, name: &str, seed: u64) -> bool {
        (self.only.is_empty() || self.only == name)
            && (self.seed.is_empty() || self.seed == seed.to_string())
    }
    fn timeout(&self) -> Duration {
        let per_case = 1 + self.repetitions * (2 + self.parallel + 1);
        Duration::from_secs(
            u64::try_from(per_case)
                .unwrap_or(u64::MAX)
                .saturating_mul(32 * 3660 + 9 * 4500),
        )
    }
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Receipt {
    schema: String,
    repo: PathBuf,
    root: PathBuf,
    requested_root: PathBuf,
    executable: PathBuf,
    executable_sha256: String,
    identity: build::NativeIdentity,
    controls: Controls,
    source_manifest: PathBuf,
    source_sha256: String,
    spec: PathBuf,
    spec_sha256: String,
    report: PathBuf,
    report_sha256: String,
    producer: PathBuf,
    complete: bool,
}
fn input<'a>(call: &'a Invocation, name: &str) -> &'a str {
    call.inputs.get(name).map_or("", String::as_str)
}
fn directory(path: &Path) -> Result<(), String> {
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(path)
        .map_err(|error| error.to_string())?;
    crate::evidence::store::open::pin(path).map(|_| ())
}
fn bytes(path: &Path, maximum: u64) -> Result<Vec<u8>, String> {
    if !path.is_absolute() {
        return Err("absolute regular stress input required".into());
    }
    if !fs::symlink_metadata(path)
        .map_err(|error| error.to_string())?
        .is_file()
    {
        return Err("absolute regular stress input required".into());
    }
    let mut raw = Vec::new();
    fs::File::open(path)
        .map_err(|error| error.to_string())?
        .take(maximum + 1)
        .read_to_end(&mut raw)
        .map_err(|error| error.to_string())?;
    if raw.len() as u64 > maximum {
        return Err("stress input capacity exceeded".into());
    }
    Ok(raw)
}
fn digest(path: &Path) -> Result<String, String> {
    Ok(canonical::digest(&bytes(path, 512 << 20)?))
}
fn write(path: &Path, raw: &[u8]) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(raw)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())
}
fn value(path: &Path) -> Result<Value, String> {
    crate::qualify::spec::strict(&bytes(path, 64 << 20)?)
}

fn configuration(repo: &Path, name: &str) -> Result<Value, String> {
    if name == "runtime" {
        let mut config = crate::install::config::build(PROFILE, None, None)?.original;
        config["Args"] = json!(["guest", "literal $arg"]);
        config["Environment"] = json!([
            "PWD=/workspace",
            "TZ=UTC",
            "ONLY=captured",
            "EXTRA_ENTROPY=no_"
        ]);
        let mut entropy = vec![0_u8; 32];
        entropy[..3].copy_from_slice(&[1, 2, 3]);
        config["EntropyKey"] = json!(entropy);
        config["Clock"] = json!({"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":1000});
        config["Limits"] = json!({"OutputBytes":1<<20,"TranscriptBytes":64<<20,"Calls":10000,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096});
        return Ok(config);
    }
    let mut config = crate::install::config::build(PROFILE, None, Some((repo, "tests")))?.original;
    if name == "two-cluster" {
        config["Args"] = json!([
            "replication.test",
            "-test.v",
            "-test.count=1",
            "-test.timeout=1h",
            "-test.run=^TestIndependentClusterReplicationRecovery$",
            "-persistenceType=sql",
            "-persistenceDriver=sqlite"
        ]);
        config["WorkingDirectory"] = json!("/workspace");
        config["Environment"] = json!([
            "PWD=/workspace",
            "TEMPORAL_ROOT=/workspace",
            "TMPDIR=/tmp",
            "TZ=UTC",
            "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db",
            "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3"
        ]);
        config["WritableDirectories"] = json!(["/workspace", "/tmp", "/volume"]);
        config["EntropyKey"] =
            json!(sha2::Sha256::digest(b"gomad-wasm-temporal-recovery/v1").to_vec());
        config["Clock"]["EpochNanos"] = json!(1_700_000_000_000_000_000_u64);
        config["Limits"] = json!({"OutputBytes":8<<20,"TranscriptBytes":128<<20,"Calls":4_000_000,"Descriptors":4096,"Files":10000,"FilesystemBytes":512<<20,"PendingEvents":4096});
        config["storage"] = json!({"volumes":[{"ID":"temporal","Path":"/volume","CapacityBytes":256<<20}],"limits":{"PendingOperations":100_000,"Transitions":10_000_000},"sqlite_lock_databases":["/volume/temporal.db"]});
    }
    config["Clock"]["ReadStepNanos"] = json!(0);
    Ok(config)
}
fn arguments(repo: &Path, root: &Path, name: &str, source: &str) -> Vec<String> {
    let runtime = name == "runtime";
    let timeout = if runtime { 60 } else { 900 };
    let mut result = vec![
        "--working-dir".into(),
        if runtime {
            repo.join("tools/gomad")
        } else {
            repo.into()
        }
        .to_string_lossy()
        .into_owned(),
        "--artifacts".into(),
        root.join("unused-artifacts").to_string_lossy().into_owned(),
        "--choices".into(),
        "--diagnostics".into(),
        "--parallel".into(),
        "1".into(),
        "--execution-timeout".into(),
        format!("{timeout}s"),
        "--overall-timeout".into(),
        format!("{}s", timeout + 3600),
        "--choice-bytes".into(),
        if runtime { 1_u64 << 20 } else { 32_u64 << 20 }.to_string(),
        "--output-limit".into(),
        if runtime {
            1_u64 << 20
        } else if name == "two-cluster" {
            8_u64 << 20
        } else {
            256_u64 << 20
        }
        .to_string(),
    ];
    if runtime {
        for entry in ["ONLY=captured", "EXTRA_ENTROPY=no_"] {
            result.extend(["--env".into(), entry.into()]);
        }
    } else {
        for tag in [
            "disable_grpc_modules",
            "gomad_wasm_slim",
            "sqlite3_dotlk",
            "test_dep",
        ] {
            result.extend(["--build-tag".into(), tag.into()]);
        }
    }
    result.extend([
        if runtime { "go-run" } else { "go-test" }.into(),
        source.into(),
    ]);
    if !runtime {
        let selection = match name {
            "two-cluster" => "^TestIndependentClusterReplicationRecovery$",
            "workflow" => "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
            _ => "^TestActivityTestSuite$",
        };
        result.extend([
            "--".into(),
            "-test.v".into(),
            "-test.count=1".into(),
            if name == "two-cluster" {
                "-test.timeout=1h"
            } else {
                "-test.timeout=72h"
            }
            .into(),
            format!("-test.run={selection}"),
            "-persistenceType=sql".into(),
            "-persistenceDriver=sqlite".into(),
        ]);
    }
    result
}
fn installation(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    name: &str,
) -> Result<PathBuf, String> {
    directory(root)?;
    let config = root.join("config.json");
    write(
        &config,
        &canonical::encode(&configuration(proof.repo(), name)?)?,
    )?;
    let compiler = Path::new(input(call, "GOMAD_STOCK_GO"));
    if !compiler.is_absolute() {
        return Err("refused: stress requires explicit absolute --stock-go".into());
    }
    let output = root.join("installation.json");
    let cache = call.inputs.get("GOMAD_WASM_TOOLCHAIN_CACHE").map_or_else(
        || proof.repo().join("tools/gomad/.toolchain"),
        PathBuf::from,
    );
    let runtime = name == "runtime";
    let argv = vec![
        "--compiler".into(),
        compiler.to_string_lossy().into_owned(),
        "--cache".into(),
        cache.to_string_lossy().into_owned(),
        "--output".into(),
        output.to_string_lossy().into_owned(),
        "--profile".into(),
        PROFILE.into(),
        "--runtime-root".into(),
        proof
            .repo()
            .join("tools/gomad/internal/model/runtimesource")
            .to_string_lossy()
            .into_owned(),
        "--config".into(),
        config.to_string_lossy().into_owned(),
        "--fuel".into(),
        if runtime {
            4_000_000_000_u64
        } else {
            1_u64 << 50
        }
        .to_string(),
        "--memory-bytes".into(),
        if runtime { 256_u64 << 20 } else { 4_u64 << 30 }.to_string(),
    ];
    write(
        &root.join("installation-command.json"),
        &canonical::encode(&argv)?,
    )?;
    let mut stdout = Vec::new();
    let code = crate::cli::install::run(&argv, &mut stdout).map_err(|error| error.message)?;
    write(&root.join("installation.stdout"), &stdout)?;
    if code != 0 {
        return Err(format!("stress installation {name} failed: {code}"));
    }
    Ok(output)
}
fn sources(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
) -> Result<(PathBuf, String), String> {
    let env = call
        .environ
        .iter()
        .filter_map(|item| item.split_once('='))
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect();
    let listed = build::command_bounded(
        root,
        "source-list",
        proof.repo(),
        &env,
        &[
            "git".into(),
            "-C".into(),
            proof.repo().to_string_lossy().into_owned(),
            "ls-files".into(),
            "-z".into(),
            "--".into(),
            "*.go".into(),
            "*.txt".into(),
            "*.rs".into(),
            "*.toml".into(),
            "*.lock".into(),
            "*go.mod".into(),
            "*go.sum".into(),
            "*json".into(),
        ],
        Duration::from_secs(30),
        16 << 20,
    )?;
    let mut paths = BTreeMap::new();
    for item in listed
        .split(|byte| *byte == 0)
        .filter(|item| !item.is_empty())
    {
        let relative = Path::new(std::str::from_utf8(item).map_err(|error| error.to_string())?);
        if relative.starts_with(".flow") {
            continue;
        }
        if relative.is_absolute()
            || relative
                .components()
                .any(|part| !matches!(part, std::path::Component::Normal(_)))
        {
            return Err("unsafe stress source inventory entry".into());
        }
        let path = proof.repo().join(relative);
        paths.insert(path.clone(), digest(&path)?);
    }
    let mut encoded = String::new();
    for (path, hash) in paths {
        writeln!(encoded, "{hash}  {}", path.display()).map_err(|error| error.to_string())?;
    }
    crate::qualify::result::validate_source_manifest(encoded.as_bytes(), true)?;
    let path = root.join("source.sha256");
    write(&path, encoded.as_bytes())?;
    Ok((path, canonical::digest(encoded.as_bytes())))
}

fn excluded_selection(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    controls: &Controls,
) -> Result<PathBuf, String> {
    let lease =
        ChildLease::for_path(root)?.ok_or("catalogue stress requires its parent owned stage")?;
    let requested = Path::new(input(call, "GOMAD_WASM_STRESS_ROOT"));
    if fs::canonicalize(requested).map_err(|error| error.to_string())? != lease.requested_root() {
        return Err("stress selection requested root differs from parent stage".into());
    }
    let host_root = root.join("host");
    directory(&host_root)?;
    write(
        &root.join("host.json"),
        &canonical::encode(&host::current(&host_root, &call.environ)?)?,
    )?;
    let (manifest, source) = sources(call, proof, root)?;
    let cells = reader::cases().into_iter().flat_map(|(name, seed)| ["cpu", "concurrent", "layout", "handoff"].map(move |axis|
        json!({"workload":name,"seed":seed,"axis":axis,"status":"skipped","reason":"excluded by --stress-only","requested":controls.repetitions*if axis=="concurrent"{controls.parallel}else{1},"completed":0,"runs":[],"comparisons":[]}))).collect::<Vec<_>>();
    let path = root.join("selection.json");
    write(
        &path,
        &canonical::encode(
            &json!({"schema":"gomad.native-stress-selection/v1","repo":proof.repo(),"root":root,"requested_root":requested,
        "complete":false,"executed":false,"controls":controls,"source_manifest":manifest,"source_sha256":source,"references":[],"cells":cells}),
        )?,
    )?;
    read(&path, proof)?;
    Ok(path)
}

fn prepare_workloads(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    controls: &Controls,
) -> Result<Vec<Value>, String> {
    let mut workloads = Vec::new();
    for name in NAMES {
        let preparation = root.join("inputs").join(name);
        let selected = controls.only.is_empty() || controls.only == name;
        let install = if selected {
            installation(call, proof, &preparation, name)?
        } else {
            preparation.join("installation.json")
        };
        let source = if name == "runtime" {
            "./internal/backends/wasm/testdata/runtime_probes".into()
        } else if name == "two-cluster" && selected {
            cluster_driver::stage_broker(proof.repo())?
        } else if name == "two-cluster" {
            "./.tmp/wasm-backend-temporal/unselected".into()
        } else {
            "./tests".into()
        };
        let scenario = if name == "two-cluster" && selected {
            let path = preparation.join("scenario.json");
            write(&path, &cluster_driver::scenario_bytes()?)?;
            Some(path)
        } else {
            None
        };
        let workload = json!({"name":name,"installation":install,"arguments":arguments(proof.repo(),&preparation,name,&source),"scenario":scenario});
        if selected {
            crate::stress::admit_workload(&workload)?;
        }
        workloads.push(workload);
    }
    Ok(workloads)
}

fn producer_environment(call: &Invocation) -> BTreeMap<String, String> {
    let mut environment: BTreeMap<String, String> = call
        .environ
        .iter()
        .filter_map(|item| item.split_once('='))
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect();
    environment.insert("GOMAD_WASM_STRESS".into(), "1".into());
    environment
}

/// # Errors
/// Refuses changed original stress controls, mismatched linked roles or a nonpassing selected cell.
pub fn run(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let controls = Controls::parse(
        call,
        std::thread::available_parallelism().map_or(1, usize::from),
    )?;
    if !reader::cases()
        .iter()
        .any(|(name, seed)| controls.selected(name, *seed))
    {
        return excluded_selection(call, proof, root, &controls);
    }
    if linked::Conditions::compiled() != linked::Conditions::stress_replay() {
        return Err("refused: catalogue stress requires the identity-bound helper-stress-jitter gomad-measure".into());
    }
    let executable = workloads::admit_executable(
        call,
        proof,
        root,
        "stress",
        PathBuf::from(input(call, "GOMAD_WASM_TEST_HELPER")),
        build::Role::Stress,
        None,
    )?;
    let current = std::env::current_exe().map_err(|error| error.to_string())?;
    if digest(&current)? != executable.sha256 {
        return Err(
            "refused: --test-helper must match the current linked stress executable".into(),
        );
    }
    let lease =
        ChildLease::for_path(root)?.ok_or("catalogue stress requires its parent owned stage")?;
    let requested_root = PathBuf::from(input(call, "GOMAD_WASM_STRESS_ROOT"));
    if fs::canonicalize(&requested_root).map_err(|error| error.to_string())?
        != lease.requested_root()
    {
        return Err("catalogue stress requested root differs from parent lease".into());
    }
    let host_root = root.join("host");
    directory(&host_root)?;
    let host = host::current(&host_root, &call.environ)?;
    write(&root.join("host.json"), &canonical::encode(&host)?)?;
    let (source_manifest, source_sha256) = sources(call, proof, root)?;
    let workloads = prepare_workloads(call, proof, root, &controls)?;
    let spec = root.join("stress-spec.json");
    write(
        &spec,
        &canonical::encode(
            &json!({"schema":"gomad.rust-stress-spec/v1","root":root.join("matrix"),"requested_output_root":requested_root,
        "host_description":controls.host_description,"source_manifest":source_manifest,"source_sha256":source_sha256,"repetitions":controls.repetitions,"parallel":controls.parallel,
        "cpu_workers":controls.cpu_workers,"only":controls.only,"seed":controls.seed,"axis":controls.axis,"workloads":workloads}),
        )?,
    )?;
    let argv = vec![
        executable.path.to_string_lossy().into_owned(),
        "stress".into(),
        "--spec".into(),
        spec.to_string_lossy().into_owned(),
    ];
    let observed = build::observe_command(
        root,
        "stress-producer",
        proof.repo(),
        &producer_environment(call),
        &argv,
        controls.timeout(),
        64 << 20,
    );
    executable.verify(proof)?;
    let report = root.join("matrix/report.json");
    let receipt = Receipt {
        schema: RESULT_SCHEMA.into(),
        repo: proof.repo().into(),
        root: root.into(),
        requested_root,
        executable: executable.path.clone(),
        executable_sha256: executable.sha256,
        identity: serde_json::from_value(super::identity()?).map_err(|error| error.to_string())?,
        controls,
        source_manifest,
        source_sha256,
        spec: spec.clone(),
        spec_sha256: digest(&spec)?,
        report: report.clone(),
        report_sha256: digest(&report)?,
        producer: root.join("stress-producer.json"),
        complete: false,
    };
    let checked = reader::check(&receipt, &observed)?;
    let path = root.join("result.json");
    write(
        &path,
        &canonical::encode(&Receipt {
            complete: checked,
            ..receipt
        })?,
    )?;
    read(&path, proof)?;
    Ok(path)
}

#[cfg(test)]
mod tests;
