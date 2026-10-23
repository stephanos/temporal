//! Explicit-input linked measurement executions with optional diagnostic capture.
use crate::evidence::canonical;
use gomad_wasmhost::{linked, linked_cluster};
use serde::Deserialize;
use serde_json::{Value, json};
use std::{
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Model {
    configuration: linked::ModelConfig,
    arguments: Vec<String>,
    environment: Vec<String>,
    replay_trace: Option<Vec<u8>>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Guest {
    module: PathBuf,
    module_sha256: String,
    limits: crate::evidence::record::Limits,
    model: Option<Model>,
    namespace: Option<linked::NamespaceImage>,
    storage: Option<linked::StorageInput>,
    #[serde(default)]
    guest_profile: Option<Sampling>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Sampling {
    interval_micros: u64,
    max_samples: u64,
}
impl Guest {
    fn request(
        self,
        root: &Path,
        index: usize,
        handle: Value,
        modules: &mut Vec<Value>,
    ) -> Result<linked::Request, String> {
        let module = std::fs::read(&self.module).map_err(|error| error.to_string())?;
        let file_name = format!("module-{index}.wasm");
        write_raw(&root.join(&file_name), &module)?;
        modules.push(json!({"request_module":self.module,"file_name":file_name,
            "sha256":canonical::digest(&module),"bytes":module.len(),"handle":handle}));
        Ok(linked::Request {
            module,
            module_sha256: self.module_sha256,
            limits: linked::Limits {
                memory_bytes: self.limits.memory_bytes,
                fuel: self.limits.fuel,
                output_bytes: self.limits.output_bytes,
                evidence_bytes: self.limits.evidence_bytes,
                wall_timeout: Duration::from_millis(self.limits.wall_millis),
            },
            model: self.model.map(|model| linked::ModelInput {
                configuration: model.configuration,
                arguments: model.arguments,
                environment: model.environment,
                replay_trace: model.replay_trace,
            }),
            namespace: self.namespace,
            storage: self.storage,
        })
    }
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Incarnation {
    handle: linked_cluster::Handle,
    guest: Guest,
    network: Option<linked_cluster::NetworkModel>,
}
#[derive(Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case", deny_unknown_fields)]
enum Input {
    Standalone {
        guest: Box<Guest>,
        network: Option<linked::NetworkModel>,
        #[serde(default)]
        compile_cache: Option<PathBuf>,
    },
    Cluster {
        nodes: Vec<String>,
        broker: linked_cluster::BrokerConfig,
        network: Option<Box<linked_cluster::NetworkConfig>>,
        incarnations: Vec<Incarnation>,
        steps: Vec<linked_cluster::Step>,
        boundaries: Vec<linked_cluster::Boundary>,
        crash_groups: Vec<linked_cluster::CrashGroup>,
        wall_millis: u64,
        evidence_bytes: u64,
        #[serde(default)]
        compile_cache: Option<PathBuf>,
    },
}
fn write_raw(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())
}
fn write(path: &Path, value: &impl serde::Serialize) -> Result<(), String> {
    write_raw(
        path,
        &serde_json::to_vec(value).map_err(|error| error.to_string())?,
    )
}
fn retain(root: &Path, execution: &linked::Execution) -> Result<(), String> {
    write(&root.join("outcome.json"), &execution.outcome)?;
    let streams: std::collections::BTreeMap<_, _> = execution
        .evidence
        .iter()
        .map(|(name, bytes)| {
            (
                name,
                json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)}),
            )
        })
        .collect();
    write(&root.join("streams.json"), &streams)?;
    for (name, bytes) in &execution.evidence {
        if name.contains('/') || name.contains('\\') || name == "." || name == ".." {
            return Err("invalid evidence stream name".into());
        }
        write_raw(&root.join(name), bytes)?;
    }
    Ok(())
}
fn retain_cluster(root: &Path, execution: &linked_cluster::Execution) -> Result<(), String> {
    write(&root.join("outcome.json"), &execution.outcome)?;
    write(&root.join("executed-steps.json"), &execution.executed_steps)?;
    let streams = [
        ("world", &execution.world),
        ("network", &execution.network),
        ("inspection", &execution.inspection),
    ];
    let catalog: std::collections::BTreeMap<_, _> = streams
        .iter()
        .map(|(name, bytes)| {
            (
                name,
                json!({"bytes":bytes.len(), "sha256":canonical::digest(bytes)}),
            )
        })
        .collect();
    write(&root.join("streams.json"), &catalog)?;
    for (name, bytes) in streams {
        write_raw(&root.join(name), bytes)?;
    }
    write(
        &root.join("cutoff.json"),
        &json!({"virtual_elapsed_nanos":execution.virtual_elapsed_nanos}),
    )?;
    for (index, incarnation) in execution.incarnations.iter().enumerate() {
        let directory = root.join(format!("incarnation-{index}"));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&directory)
            .map_err(|error| error.to_string())?;
        retain(&directory, &incarnation.execution)?;
        write(&directory.join("handle.json"), &incarnation.handle)?;
    }
    Ok(())
}

/// # Errors
/// Retains actual linked execution outcomes and diagnostic profiles before validation.
pub fn run(input: &Path, root: &Path) -> Result<(), String> {
    let raw = std::fs::read(input).map_err(|error| error.to_string())?;
    let request: Input = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    execute(raw, request, root)
}

/// # Errors
/// Retains failed/interrupted observations without replacing an earlier successful result.
pub fn run_retained(
    input: &Path,
    requested_root: &Path,
    keep: bool,
) -> Result<std::path::PathBuf, String> {
    run_retained_command(input, requested_root, keep, "--retained-linked-run")
}

/// # Errors
/// Retains the exact caller grammar in rerun records.
pub fn run_retained_command(
    input: &Path,
    requested_root: &Path,
    keep: bool,
    verb: &str,
) -> Result<PathBuf, String> {
    let raw = std::fs::read(input).map_err(|error| error.to_string())?;
    let request: Input = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    let inputs = std::collections::BTreeMap::from([(
        "request".into(),
        input.to_string_lossy().into_owned(),
    )]);
    let retained = crate::measurement::retention::RetainedRun::begin(
        requested_root,
        "linked-profile",
        &inputs,
        keep,
        &|| false,
    )?;
    let output = retained.path().to_owned();
    let mut command = vec![
        std::env::current_exe()
            .map_err(|error| error.to_string())?
            .to_string_lossy()
            .into_owned(),
        verb.into(),
        input.to_string_lossy().into_owned(),
        requested_root.to_string_lossy().into_owned(),
    ];
    if keep {
        command.push("--keep".into());
    }
    let rerun = json!({"requested_root":requested_root,"input":input,
        "command":command});
    let result =
        write(&output.join("rerun.json"), &rerun).and_then(|()| execute(raw, request, &output));
    let finish = retained.finish(result.is_ok(), &|| false);
    super::combine(result, finish).map(|()| output)
}

fn execute(raw: Vec<u8>, request: Input, root: &Path) -> Result<(), String> {
    write_raw(&root.join("request.json"), &raw)?;
    let mut modules = Vec::new();
    let mut retained_incarnations = 0usize;
    let mut authorization_to_terminal_nanos = None;
    let mut terminal_to_return_nanos = None;
    #[cfg(feature = "helper-diagnostic-profile")]
    let mut native_external_owner_attempts = None;
    let (result, profile) = match request {
        Input::Standalone {
            mut guest,
            network,
            compile_cache,
        } => {
            let sampling = guest.guest_profile.take();
            if sampling.as_ref().is_some_and(|sampling| {
                !(1..=1_000_000).contains(&sampling.interval_micros)
                    || !(1..=100_000).contains(&sampling.max_samples)
            }) {
                return Err("invalid guest profile sampling bounds".into());
            }
            #[cfg(not(feature = "helper-guest-profile"))]
            if sampling.is_some() {
                return Err("guest sampling requires the actual helper-guest-profile role".into());
            }
            let request = guest.request(root, 0, Value::Null, &mut modules)?;
            write(&root.join("rootmodules.json"), &modules)?;
            let timeout = request.limits.wall_timeout;
            let (result, profile) = capture(|| {
                crate::backstop::within(timeout, || {
                    let mut prepared = linked::prepare(request)?;
                    prepared.conditions(linked::Conditions::compiled());
                    prepared.compile_cache(
                        compile_cache
                            .as_deref()
                            .map(|path| {
                                path.to_str().ok_or_else(|| {
                                    std::io::Error::other("compile cache path must be UTF-8")
                                })
                            })
                            .transpose()?,
                    )?;
                    #[cfg(feature = "helper-guest-profile")]
                    if let Some(sampling) = sampling {
                        prepared.guest_profile(linked::GuestProfileConfig {
                            path: root
                                .join("guest-profile.json")
                                .to_string_lossy()
                                .into_owned(),
                            interval_micros: sampling.interval_micros,
                            max_samples: sampling.max_samples,
                        })?;
                    }
                    if let Some(network) = network {
                        prepared.network(network)?;
                    }
                    linked::execute_prepared(prepared)
                })
            })?;
            let result = result.map_err(|error| error.to_string());
            if let Ok(execution) = &result {
                authorization_to_terminal_nanos = execution.authorization_to_terminal_nanos;
                terminal_to_return_nanos = execution
                    .timing
                    .as_ref()
                    .map(linked::ExecutionTiming::terminal_to_return_nanos);
                #[cfg(feature = "helper-diagnostic-profile")]
                {
                    native_external_owner_attempts = Some(execution.external_owner_attempts);
                }
                retain(root, execution)?;
            }
            (
                result.and_then(|execution| {
                    #[cfg(feature = "helper-guest-profile")]
                    if let Some(error) = execution.guest_profile_error {
                        return Err(format!(
                            "guest profile finalization failed after terminal retention: {error}"
                        ));
                    }
                    Ok(json!({"termination":execution.outcome.termination}))
                }),
                profile,
            )
        }
        Input::Cluster {
            nodes,
            broker,
            network,
            incarnations,
            steps,
            boundaries,
            crash_groups,
            wall_millis,
            evidence_bytes,
            compile_cache,
        } => {
            let request = linked_cluster::Request {
                nodes,
                broker,
                network: network.map(|network|*network),
                steps,
                boundaries,
                crash_groups,
                evidence_bytes,
                wall_timeout: Duration::from_millis(wall_millis),
                incarnations: incarnations
                    .into_iter().enumerate()
                    .map(|(index,incarnation)| {
                        let handle = serde_json::to_value(&incarnation.handle).map_err(|error| error.to_string())?;
                        Ok(linked_cluster::Incarnation {
                            handle: incarnation.handle,
                            request: {
                                if incarnation.guest.guest_profile.is_some() {
                                    return Err("cluster guest sampling requires a per-incarnation sampling producer".into());
                                }
                                incarnation.guest.request(root,index,handle,&mut modules)?
                            },
                            network: incarnation.network,
                        })
                    })
                    .collect::<Result<_, String>>()?,
            };
            write(&root.join("rootmodules.json"), &modules)?;
            let timeout = request.wall_timeout;
            let (result, profile) = capture(|| {
                crate::backstop::within(timeout, || {
                    let mut prepared = linked_cluster::prepare(request)?;
                    prepared.conditions(linked::Conditions::compiled());
                    prepared.compile_cache(
                        compile_cache
                            .as_deref()
                            .map(|path| {
                                path.to_str().ok_or_else(|| {
                                    std::io::Error::other("compile cache path must be UTF-8")
                                })
                            })
                            .transpose()?,
                    )?;
                    linked_cluster::execute_prepared(prepared)
                })
            })?;
            let result = result.map_err(|error| error.to_string());
            if let Ok(execution) = &result {
                retained_incarnations = execution.incarnations.len();
                retain_cluster(root, execution)?;
            }
            (
                result.and_then(|execution| {
                    if execution.outcome.termination == "watchdog_timeout" && execution.world.is_empty() {
                        return Err("cluster watchdog World prefix unavailable; interrupted prefix proof is incomplete".into());
                    }
                    Ok(json!({"termination":execution.outcome.termination}))
                }),
                profile,
            )
        }
    };
    if let Some(profile) = &profile {
        write(&root.join("profile.json"), profile)?;
    }
    let binary = std::fs::read(std::env::current_exe().map_err(|error| error.to_string())?)
        .map_err(|error| error.to_string())?;
    #[cfg(feature = "helper-diagnostic-profile")]
    let validation = profile.as_ref().map_or(Ok(()), crate::profile::validate);
    #[cfg(not(feature = "helper-diagnostic-profile"))]
    let validation: Result<(), String> = Ok(());
    #[cfg(feature = "helper-diagnostic-profile")]
    let namespace_counter=native_external_owner_attempts.map(|attempts| {
        let namespace_calls=profile.as_ref().and_then(|profile|profile["owners"]["namespace"]["count"].as_u64());
        namespace_calls.map(|calls|json!({"schema":"gomad.native-namespace-counter/v1",
            "implementation_sha256":linked::IMPLEMENTATION_SHA256,"fresh_store_id":std::process::id(),
            "counter_domain":"NoRemote::write callback-frame write attempts; not completed filesystem calls",
            "namespace_boundary":"helper-owned ABI namespace dispatch; fallback callbacks write through NoRemote",
            "namespace_calls":calls,"external_owner_calls":if attempts == 0 {std::collections::BTreeMap::new()} else {std::collections::BTreeMap::from([("linked-external-callback",attempts)])}}))
    }).flatten();
    #[cfg(not(feature = "helper-diagnostic-profile"))]
    let namespace_counter: Option<Value> = None;
    let report: Value = json!({
        "schema":"gomad.linked-profile-run/v1", "diagnostic_only":cfg!(feature="helper-diagnostic-profile"),
        "retained_incarnations":retained_incarnations,
        "authorization_to_terminal_nanos":authorization_to_terminal_nanos,
        "fresh_store_id":std::process::id(),
        "terminal_to_return_nanos":terminal_to_return_nanos,
        "timing":terminal_to_return_nanos.map(|nanos|json!({
            "schema":"gomad.native-linked-timing/v1",
            "implementation_sha256":linked::IMPLEMENTATION_SHA256,
            "fresh_store_id":std::process::id(),"terminal_to_return_nanos":nanos,
            "clock":"monotonic","scope":"validated-native-terminal-to-linked-return"})),
        "native_namespace_counter":namespace_counter,
        "request_sha256":canonical::digest(&raw), "host_sha256":canonical::digest(&binary),
        "linked_implementation_sha256":linked::IMPLEMENTATION_SHA256,
        "host_implementation_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "configuration":linked::Conditions::compiled().configuration(), "features":crate::measurement::features(),
        "invocation":std::env::args().collect::<Vec<_>>(),
        "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
        "execution":result, "attribution":if cfg!(feature="helper-diagnostic-profile") {serde_json::to_value(&validation).map_err(|error|error.to_string())?} else {Value::Null},
        "qualification":"unavailable: measurement producer; fn-185 R4 remains pending fn-232.13",
        "scope":"elapsed_wall_nanos covers linked admission/compilation/execution/teardown; input-file reads and output publication are excluded; store spans are thread-time; no elapsed Amdahl bound is inferred"
    });
    write(&root.join("report.json"), &report)?;
    validation?;
    result.map(|_| ())
}

fn capture<T>(run: impl FnOnce() -> T) -> Result<(T, Option<Value>), String> {
    #[cfg(feature = "helper-diagnostic-profile")]
    {
        crate::profile::capture(run).map(|(result, profile)| (result, Some(profile)))
    }
    #[cfg(not(feature = "helper-diagnostic-profile"))]
    {
        Ok((run(), None))
    }
}
