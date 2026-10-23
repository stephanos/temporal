//! Fresh seeded host perturbations; never a qualification or replay producer.
pub(crate) mod cluster;
mod inspection;
mod matrix;
mod process;
mod report;
#[cfg(test)]
mod tests;

use crate::{
    campaign::{execution, ledger::Job, prepare},
    cli::explore::ExploreOptions,
    evidence::{
        canonical,
        store::{Opened, Store},
    },
};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    fs,
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
};

const AXES: [&str; 4] = ["cpu", "concurrent", "layout", "handoff"];

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Workload {
    name: String,
    installation: PathBuf,
    arguments: Vec<String>,
    #[serde(default)]
    scenario: Option<PathBuf>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Spec {
    #[serde(skip)]
    requested_root: PathBuf,
    #[serde(default)]
    requested_output_root: Option<PathBuf>,
    schema: String,
    root: PathBuf,
    host_description: String,
    source_manifest: PathBuf,
    source_sha256: String,
    repetitions: usize,
    parallel: usize,
    cpu_workers: usize,
    workloads: Vec<Workload>,
    #[serde(default)]
    only: String,
    #[serde(default)]
    seed: String,
    #[serde(default)]
    axis: String,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Supervision {
    cpu_workers: usize,
    cohort_size: usize,
    repetition: usize,
    member: usize,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Request {
    requested_root: PathBuf,
    workload: String,
    seed: u64,
    axis: String,
    options: ExploreOptions,
    root: PathBuf,
    source_sha256: String,
    target_pool: PathBuf,
    supervision: Supervision,
}

fn cases() -> Vec<(&'static str, u64)> {
    (0..32)
        .map(|index| ("runtime", index * 1_000_003))
        .chain([7, 19, 42].map(|seed| ("two-cluster", seed)))
        .chain(
            ["workflow", "activity"]
                .into_iter()
                .flat_map(|name| [0, 1_000_003, 2_000_006].map(|seed| (name, seed))),
        )
        .collect()
}

fn cell_status(requested: usize, completed: usize, comparisons: &[Value]) -> &'static str {
    if comparisons
        .iter()
        .any(|report| report["status"] != "identical")
    {
        "divergence"
    } else if requested != completed || comparisons.len() != requested {
        "skipped"
    } else {
        "pass"
    }
}

fn comparable(a: &Value, b: &Value) -> Result<(), String> {
    if a["fresh"] != true || b["fresh"] != true {
        return Err("forced replay is not a stress cell".into());
    }
    if a["workload"] != b["workload"]
        || a["seed"].as_u64().is_none()
        || a["seed"] != b["seed"]
        || a["identities"]
            .as_object()
            .is_none_or(serde_json::Map::is_empty)
        || a["identities"] != b["identities"]
    {
        return Err("stress seed or identity changed".into());
    }
    Ok(())
}

fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())
}

fn directory(path: &Path) -> Result<(), String> {
    fs::DirBuilder::new()
        .mode(0o700)
        .create(path)
        .map_err(|error| error.to_string())
}

fn read<T: serde::de::DeserializeOwned>(path: &Path) -> Result<T, String> {
    crate::qualify::spec::strict(&crate::qualify::spec::read_bounded(path, 16 << 20)?)
}

fn source(spec: &Spec) -> Result<(), String> {
    if !spec.source_manifest.is_absolute() {
        return Err("stress source manifest must be absolute".into());
    }
    let raw = crate::qualify::spec::read_bounded(&spec.source_manifest, 16 << 20)?;
    if !canonical::valid_digest(&spec.source_sha256)
        || canonical::digest(&raw) != spec.source_sha256
    {
        return Err("stress integrated source manifest changed".into());
    }
    crate::qualify::result::validate_source_manifest(&raw, true)
}

fn options(workload: &Workload) -> Result<ExploreOptions, String> {
    if !workload.installation.is_absolute()
        || workload
            .scenario
            .as_ref()
            .is_some_and(|path| !path.is_absolute())
    {
        return Err("stress installation and scenario paths must be absolute".into());
    }
    let mut json = false;
    let mut options = crate::cli::explore::parse(&workload.arguments, &mut json)?;
    options.installation = Some(workload.installation.clone());
    if let Some(path) = &workload.scenario {
        options.scenario = Some(crate::qualify::spec::read_bounded(path, 256 << 10)?);
    }
    if !options.choices
        || !options.diagnostics
        || options.soft_assertion_report_only
        || options.parallel != 1
    {
        return Err(
            "stress requires choices, diagnostics, one fresh seed and unchanged assertions".into(),
        );
    }
    let expected = match workload.name.as_str() {
        "runtime" => {
            if options.target_kind != "go-run"
                || options.source != "./internal/backends/wasm/testdata/runtime_probes"
                || !options.target_args.is_empty()
                || options.environment != ["ONLY=captured", "EXTRA_ENTROPY=no_"]
                || options.scenario.is_some()
            {
                return Err("stress runtime inputs changed".into());
            }
            None
        }
        "workflow" => Some("^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"),
        "activity" => Some("^TestActivityTestSuite$"),
        "two-cluster" => Some("^TestIndependentClusterReplicationRecovery$"),
        _ => return Err("unknown stress workload".into()),
    };
    if let Some(selection) = expected
        && (options.target_kind != "go-test"
            || options.target_args
                != [
                    "-test.v".to_owned(),
                    "-test.count=1".into(),
                    if workload.name == "two-cluster" {
                        "-test.timeout=1h".into()
                    } else {
                        "-test.timeout=72h".into()
                    },
                    format!("-test.run={selection}"),
                    "-persistenceType=sql".into(),
                    "-persistenceDriver=sqlite".into(),
                ]
            || (workload.name != "two-cluster"
                && (options.source != "./tests" || options.scenario.is_some()))
            || (workload.name == "two-cluster" && options.scenario.is_none()))
    {
        return Err("stress unchanged guest selection/scenario required".into());
    }
    if options.overall_timeout != options.execution_timeout + std::time::Duration::from_secs(3600) {
        return Err(
            "stress requires the original execution timeout plus one-hour overall budget".into(),
        );
    }
    options.seeds = "0".into();
    Ok(options)
}

fn inputs(name: &str, prepared: &prepare::Prepared) -> Result<(), String> {
    if name == "two-cluster" {
        cluster::validate_scenario(
            prepared
                .scenario
                .as_deref()
                .ok_or("missing stress scenario")?,
        )?;
    }
    original_bounds(
        name,
        &prepared.configuration.profile,
        &prepared.configuration.limits,
        prepared.configuration.wall_nanos,
        prepared.configuration.namespace.as_ref(),
        prepared.configuration.model.as_ref(),
    )
}

fn original_bounds(
    name: &str,
    profile: &str,
    limits: &crate::evidence::record::Limits,
    wall_nanos: Option<u64>,
    namespace: Option<&linked::NamespaceImage>,
    model: Option<&Value>,
) -> Result<(), String> {
    let runtime = name == "runtime";
    let cluster = name == "two-cluster";
    let output = if runtime {
        1 << 20
    } else if cluster {
        8 << 20
    } else {
        256 << 20
    };
    let transcript = if runtime {
        64 << 20
    } else if cluster {
        128 << 20
    } else {
        1 << 30
    };
    let calls = if runtime {
        10_000
    } else if cluster {
        4_000_000
    } else {
        1 << 28
    };
    if profile != "gomad.wasi-cooperative/v1"
        || limits.memory_bytes != if runtime { 256 << 20 } else { 4 << 30 }
        || limits.output_bytes != output
        || namespace.is_none_or(|namespace| {
            namespace.limits.output_bytes != output
                || namespace.limits.files
                    != if runtime {
                        1000
                    } else if cluster {
                        10_000
                    } else {
                        100_000
                    }
                || namespace.limits.descriptors != if runtime { 64 } else { 4096 }
                || namespace.limits.filesystem_bytes != if runtime { 8 << 20 } else { 512 << 20 }
        })
        || limits.fuel != if runtime { 4_000_000_000 } else { 1 << 50 }
        || wall_nanos
            != Some(if runtime {
                60_000_000_000
            } else {
                900_000_000_000
            })
        || model.is_none_or(|model| {
            model["mode"] != 1
                || model.get("seed").is_some()
                || model.get("replay").is_some_and(|value| !value.is_null())
                || model["diagnostics"] != true
                || model.get("digest_only").is_some_and(|value| value != false)
                || model["limits"]["transcript_bytes"] != transcript
                || model["limits"]["calls"] != calls
                || model["limits"]["pending_events"] != 4096
                || model["limit"] != if runtime { 1 << 20 } else { 32 << 20 }
                || (!runtime && model["read_step_nanos"] != 0)
        })
    {
        return Err("stress requires original bounds and unforced cooperative inputs".into());
    }
    Ok(())
}

fn retain_tapes(
    prepared: &prepare::Prepared,
    publication: &mut crate::evidence::store::Publication,
) -> Result<(), String> {
    let identity = prepared
        .choice_identity
        .as_ref()
        .ok_or("missing stress choice identity")?;
    let tapes = publication
        .payloads
        .iter()
        .filter(|(name, _)| name.ends_with("/choice"))
        .map(|(name, raw)| {
            let tape = linked::project_choices(raw, identity)?;
            Ok::<_, String>((format!("{name}-tape"), tape.bytes))
        })
        .collect::<Result<Vec<_>, String>>()?;
    if tapes.is_empty() {
        return Err("stress execution omitted all choice streams".into());
    }
    for (name, raw) in tapes {
        publication.record.streams.insert(
            name.clone(),
            crate::evidence::record::Stream {
                bytes: raw.len() as u64,
                sha256: canonical::digest(&raw),
                records: 1,
            },
        );
        publication.payloads.insert(name, raw);
    }
    Ok(())
}

fn rerun(binary: &Path, command: &[String], root: &Path) -> Result<Vec<String>, String> {
    let mut rerun = vec![
        "env".to_owned(),
        "-C".into(),
        std::env::current_dir()
            .map_err(|error| error.to_string())?
            .to_string_lossy()
            .into_owned(),
        "-u".into(),
        "GOMAD_WASM_STRESS_JITTER".into(),
        "-u".into(),
        "GOMAD_STRESS_PADDING".into(),
        "-u".into(),
        "TMPDIR".into(),
        "GOMAD_WASM_STRESS=1".into(),
    ];
    for name in ["GOMAD_WASM_STRESS_JITTER", "GOMAD_STRESS_PADDING", "TMPDIR"] {
        if let Some(value) = std::env::var_os(name) {
            rerun.push(format!("{name}={}", value.to_string_lossy()));
        }
    }
    let request_path = command
        .windows(2)
        .find(|pair| pair[0] == "--cell")
        .ok_or("missing stress request path")?[1]
        .clone();
    rerun.extend([
        binary.to_string_lossy().into_owned(),
        "stress".into(),
        "--rerun".into(),
        request_path,
    ]);
    rerun.extend([
        "--root".into(),
        root.join("rerun").to_string_lossy().into_owned(),
    ]);
    Ok(rerun)
}

fn cell(request: &Request, binary: &Path, command: &[String]) -> Result<Value, String> {
    let _parent_lease = crate::measurement::retention::ChildLease::for_path(&request.root)?;
    let prepared = prepare::prepare(&request.options)?;
    inputs(&request.workload, &prepared)?;
    let mut result = if request.workload == "two-cluster" {
        cluster::execute(&prepared, request.seed, request.options.execution_timeout)?
    } else {
        execution::run_with_replay(
            &prepared,
            &Job {
                ordinal: 1,
                seed: request.seed,
            },
            request.options.execution_timeout,
            &linked::Cancellation::default(),
            None,
            None,
        )?
    }
    .finalized()
    .map_err(|failure| failure.message())?;
    let outcome_error = cluster::outcome(&request.workload, &prepared, &result.publication).err();
    let tape_error = retain_tapes(&prepared, &mut result.publication).err();
    let oracle_error = outcome_error.or(tape_error);
    let binding = canonical::encode(
        &json!({"schema":"gomad.stress-inputs/v1","fresh":true,"seed":request.seed,"workload":request.workload,"source_sha256":request.source_sha256}),
    )?;
    result.publication.record.streams.insert(
        "evidence/stress-inputs.json".into(),
        crate::evidence::record::Stream {
            bytes: binding.len() as u64,
            sha256: canonical::digest(&binding),
            records: 1,
        },
    );
    result
        .publication
        .payloads
        .insert("evidence/stress-inputs.json".into(), binding);
    result
        .publication
        .payloads
        .insert("record.json".into(), result.publication.record.bytes()?);
    #[cfg(feature = "helper-stress-jitter")]
    let jitter = linked::stress_jitter_statistics();
    #[cfg(not(feature = "helper-stress-jitter"))]
    let jitter = (0_u64, 0_u64);
    let perturbation_error = (request.axis == "handoff" && jitter.0 == 0)
        .then_some("handoff axis made no actual jitter delays");
    let identities = json!({"execution":result.publication.record.identity,"source":request.source_sha256,
        "helper":linked::IMPLEMENTATION_SHA256,"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "engine":linked::Conditions::compiled().configuration(),"model":canonical::digest(&canonical::encode(&prepared.configuration.model)?),
        "provenance":canonical::digest(&prepared.provenance)});
    let store = Store {
        root: request.root.join("artifacts"),
        maximum_bytes: 2 << 30,
        target_pool: Some(request.target_pool.clone()),
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    };
    let published = store.publish(result.publication)?;
    let opened = Opened::open(&published.path)?;
    opened.authenticate_all()?;
    let event_order_error = opened
        .payload("evidence/event-order", 32 << 20)
        .and_then(|raw| report::parse(&raw).map(|_| ()))
        .err();
    let rerun = rerun(binary, command, &request.requested_root)?;
    let observation = json!({"fresh":true,"workload":request.workload,"seed":request.seed,"axis":request.axis,"identities":identities,
        "oracle_error":oracle_error,"event_order_error":event_order_error,"perturbation_error":perturbation_error,"process_id":std::process::id(),"artifact":published.path,"record_sha256":published.record_sha256,"binary":binary,"command":rerun,"executed_command":command,"supervision":request.supervision,
        "host_conditions":{"cwd":std::env::current_dir().map_err(|error| error.to_string())?,"executable":std::env::current_exe().map_err(|error| error.to_string())?,
            "padding_bytes":std::env::var_os("GOMAD_STRESS_PADDING").map_or(0, |value| value.len()),"jitter":std::env::var_os("GOMAD_WASM_STRESS_JITTER").is_some(),
            "jitter_delay_calls":jitter.0,"jitter_delay_micros":jitter.1,
            "cache_path":"inapplicable: linked engine compiles in process without external helper cache"},
        "streams":opened.manifest().record.streams});
    write(
        &request.root.join("observation.json"),
        &canonical::encode(&observation)?,
    )?;
    Ok(observation)
}

#[expect(
    clippy::too_many_lines,
    reason = "Managed stress execution keeps source admission, parent retention and rerun lifecycle in one supervisor"
)]
pub(crate) fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, String> {
    if std::env::var("GOMAD_WASM_STRESS").as_deref() != Ok("1") {
        return Err("stress requires GOMAD_WASM_STRESS=1".into());
    }
    if linked::Conditions::compiled() != linked::Conditions::stress_replay() {
        return Err("stress requires the identity-bound helper-stress-jitter build for baseline and every axis".into());
    }
    let binary = std::env::current_exe().map_err(|error| error.to_string())?;
    let command: Vec<String> = std::env::args().collect();
    if arguments.len() == 4 && arguments[0] == "--rerun" && arguments[2] == "--root" {
        let mut request: Request = read(Path::new(&arguments[1]))?;
        let requested_root = Path::new(&arguments[3]);
        let inputs = std::collections::BTreeMap::from([("request".into(), arguments[1].clone())]);
        let retained = crate::measurement::retention::RetainedRun::begin(
            requested_root,
            "stress-rerun",
            &inputs,
            false,
            &|| false,
        )?;
        requested_root.clone_into(&mut request.requested_root);
        let result = process::supervise(&binary, &request, &retained.path().join("cohort"))
            .and_then(|report| {
                writeln!(output, "{report}").map_err(|error| error.to_string())?;
                Ok(i32::from(report["passed"] != true))
            });
        let finish = retained.finish(result.as_ref().is_ok_and(|status| *status == 0), &|| false);
        let status = result.as_ref().copied().unwrap_or(1);
        crate::measurement::combine(result.map(|_| ()), finish)?;
        return Ok(status);
    }
    if (arguments.len() == 2 || arguments.len() == 4 && arguments[2] == "--root")
        && arguments[0] == "--cell"
    {
        let mut request: Request = read(Path::new(&arguments[1]))?;
        if arguments.len() == 4 {
            let requested_root = PathBuf::from(&arguments[3]);
            if !requested_root.is_absolute() {
                return Err("stress rerun root must be absolute".into());
            }
            let inputs =
                std::collections::BTreeMap::from([("request".into(), arguments[1].clone())]);
            let retained = crate::measurement::retention::RetainedRun::begin(
                &requested_root,
                "stress-cell",
                &inputs,
                false,
                &|| false,
            )?;
            request.requested_root = requested_root;
            request.root = retained.path().join("cell");
            let result = directory(&request.root).and_then(|()| cell(&request, &binary, &command));
            let finish = retained.finish(result.is_ok(), &|| false);
            crate::measurement::combine(result.map(|_| ()), finish)?;
            return Ok(0);
        }
        cell(&request, &binary, &command)?;
        return Ok(0);
    }
    if arguments.len() != 2 || arguments[0] != "--spec" {
        return Err("stress requires --spec FILE".into());
    }
    let mut spec: Spec = read(Path::new(&arguments[1]))?;
    if spec.schema != "gomad.rust-stress-spec/v1"
        || !spec.root.is_absolute()
        || spec.host_description.trim().is_empty()
        || !(2..=256).contains(&spec.repetitions)
        || !(2..=256).contains(&spec.parallel)
        || !(1..=256).contains(&spec.cpu_workers)
        || spec.workloads.len() != 4
        || ["runtime", "two-cluster", "workflow", "activity"]
            .iter()
            .any(|name| {
                spec.workloads
                    .iter()
                    .filter(|item| item.name == *name)
                    .count()
                    != 1
            })
    {
        return Err("stress requires the complete workload matrix, positive bounds and explicit quiet-host evidence".into());
    }
    if (!spec.only.is_empty()
        && !["runtime", "two-cluster", "workflow", "activity"].contains(&spec.only.as_str()))
        || (!spec.seed.is_empty() && spec.seed.parse::<u64>().is_err())
        || (!spec.axis.is_empty()
            && !["reference", "cpu", "concurrent", "layout", "handoff"]
                .contains(&spec.axis.as_str()))
    {
        return Err("stress selection controls are invalid".into());
    }
    source(&spec)?;
    spec.requested_root = spec.root.clone();
    if let Some(lease) = crate::measurement::retention::ChildLease::for_path(&spec.root)? {
        let requested = spec
            .requested_output_root
            .as_ref()
            .ok_or("managed stress child requires original requested root")?;
        if !requested.is_absolute()
            || fs::canonicalize(requested).map_err(|error| error.to_string())?
                != lease.requested_root()
        {
            return Err(
                "managed stress child requested root differs from authenticated parent".into(),
            );
        }
        spec.requested_root = requested.clone();
        directory(&spec.root)?;
        write(&spec.root.join("spec.json"), &canonical::encode(&spec)?)?;
        return run_spec(&binary, &spec, &command, output);
    }
    if spec.requested_output_root.is_some() {
        return Err("requested output root requires an authenticated managed parent".into());
    }
    let mut inputs = std::collections::BTreeMap::from([(
        "source".into(),
        spec.source_manifest.to_string_lossy().into_owned(),
    )]);
    for workload in &spec.workloads {
        inputs.insert(
            format!("installation:{}", workload.name),
            workload.installation.to_string_lossy().into_owned(),
        );
        if let Some(path) = &workload.scenario {
            inputs.insert(
                format!("scenario:{}", workload.name),
                path.to_string_lossy().into_owned(),
            );
        }
    }
    let retained = crate::measurement::retention::RetainedRun::begin(
        &spec.root,
        "stress",
        &inputs,
        false,
        &|| false,
    )?;
    let spec_write =
        canonical::encode(&spec).and_then(|raw| write(&retained.path().join("spec.json"), &raw));
    retained.path().clone_into(&mut spec.root);
    let result = spec_write.and_then(|()| run_spec(&binary, &spec, &command, output));
    let finish = retained.finish(result.as_ref().is_ok_and(|status| *status == 0), &|| false);
    let status = result.as_ref().copied().unwrap_or(1);
    crate::measurement::combine(result.map(|_| ()), finish)?;
    Ok(status)
}

fn run_spec(
    binary: &Path,
    spec: &Spec,
    command: &[String],
    output: &mut dyn Write,
) -> Result<i32, String> {
    let host_before = json!({"uname":process::host("uname", &["-a"]),"load":process::host("cat", &["/proc/loadavg"]),"memory":process::host("cat", &["/proc/meminfo"])});
    write(
        &spec.root.join("host-before.json"),
        &canonical::encode(&host_before)?,
    )?;
    let prepared = matrix::prepare(spec)?;
    let matrix::Matrix {
        cells,
        references,
        rejected,
    } = matrix::run(binary, spec, &prepared)?;
    let passed = rejected.is_empty()
        && cells.iter().all(|cell| {
            cell["status"] == "pass"
                || cell["axis"] == "handoff"
                    && cell["workload"] != "two-cluster"
                    && cell["completed"] == 0
        });
    let result = json!({"schema":"gomad.rust-stress/v1","host":{"description":spec.host_description,"os":std::env::consts::OS,"arch":std::env::consts::ARCH,
        "available_parallelism":std::thread::available_parallelism().map_or(1,usize::from),"uname":process::host("uname", &["-a"]),"load_before_report":process::host("cat", &["/proc/loadavg"]),
        "quiet_admission":"operator assertion; CPU axis intentionally loads the host"},"source_sha256":spec.source_sha256,"binary_sha256":canonical::digest(&fs::read(binary).map_err(|error| error.to_string())?),
        "host_before":host_before,"command":command,"references":references,"cells":cells,"rejected":rejected,"passed":passed,"finding_owner":"fn-188-gomad-wasm-determinism-boundary-audit.6"});
    let path = spec.root.join("report.json");
    write(&path, &canonical::encode(&result)?)?;
    writeln!(output, "{}", json!({"report":path,"passed":passed}))
        .map_err(|error| error.to_string())?;
    Ok(i32::from(!passed))
}

#[cfg(test)]
mod reach_tests;

#[cfg(all(test, feature = "helper-determinism-fixture"))]
pub(crate) fn compare_counterfactual(left: &Value, right: &Value) -> Result<Value, String> {
    report::compare_counterfactual(left, right)
}

/// Admit the exact original stress recipe without preparing or running its guest.
pub(crate) fn admit_workload(value: Value) -> Result<ExploreOptions, String> {
    let workload: Workload = crate::qualify::spec::strict(&canonical::encode(&value)?)?;
    options(&workload)
}

/// Reopen both retained stress artifacts and recompute their first semantic difference.
pub(crate) fn compare_retained(left: &Value, right: &Value) -> Result<Value, String> {
    report::compare(left, right)
}

/// Authenticate original bounds on the actual retained configuration, without compiling or executing it.
pub(crate) fn admit_retained_configuration(name: &str, value: Value) -> Result<(), String> {
    if name == "two-cluster" {
        let configuration: crate::replay::cluster::Configuration =
            serde_json::from_value(value).map_err(|error| error.to_string())?;
        cluster::validate_retained(&configuration)?;
        for owner in &configuration.incarnations {
            original_bounds(
                name,
                &configuration.profile,
                &owner.limits,
                owner.wall_nanos,
                Some(&owner.namespace),
                Some(&owner.model),
            )?;
        }
        Ok(())
    } else {
        let configuration: crate::replay::preflight::Configuration =
            serde_json::from_value(value).map_err(|error| error.to_string())?;
        if name != "runtime" {
            let selection = match name {
                "workflow" => "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
                "activity" => "^TestActivityTestSuite$",
                _ => return Err("unknown retained stress workload".into()),
            };
            if configuration.arguments.get(1..).is_none_or(|arguments| {
                arguments
                    != [
                        "-test.v",
                        "-test.count=1",
                        "-test.timeout=72h",
                        &format!("-test.run={selection}"),
                        "-persistenceType=sql",
                        "-persistenceDriver=sqlite",
                    ]
            }) {
                return Err("retained stress original guest selection changed".into());
            }
        }
        original_bounds(
            name,
            &configuration.profile,
            &configuration.limits,
            configuration.wall_nanos,
            configuration.namespace.as_ref(),
            configuration.model.as_ref(),
        )
    }
}
