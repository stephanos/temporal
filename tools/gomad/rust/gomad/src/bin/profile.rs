//! Explicit-input diagnostic producer; excluded from ordinary release builds.
use gomad::evidence::canonical;
use gomad_wasmhost::{linked, linked_cluster};
use serde::Deserialize;
use serde_json::{Value, json};
use std::{
    os::unix::fs::DirBuilderExt,
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
    limits: gomad::evidence::record::Limits,
    model: Option<Model>,
    namespace: Option<linked::NamespaceImage>,
    storage: Option<linked::StorageInput>,
}
impl Guest {
    fn request(self) -> Result<linked::Request, String> {
        let module = std::fs::read(&self.module).map_err(|error| error.to_string())?;
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
        guest: Guest,
        network: Option<linked::NetworkModel>,
    },
    Cluster {
        nodes: Vec<String>,
        broker: linked_cluster::BrokerConfig,
        network: Option<linked_cluster::NetworkConfig>,
        incarnations: Vec<Incarnation>,
        steps: Vec<linked_cluster::Step>,
        boundaries: Vec<linked_cluster::Boundary>,
        crash_groups: Vec<linked_cluster::CrashGroup>,
        wall_millis: u64,
        evidence_bytes: u64,
    },
}
fn write(path: &Path, value: &impl serde::Serialize) -> Result<(), String> {
    let bytes = serde_json::to_vec(value).map_err(|error| error.to_string())?;
    std::fs::write(path, bytes).map_err(|error| error.to_string())
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
        std::fs::write(root.join(name), bytes).map_err(|error| error.to_string())?;
    }
    Ok(())
}
fn run(input: &Path, root: &Path) -> Result<(), String> {
    let raw = std::fs::read(input).map_err(|error| error.to_string())?;
    let request: Input = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    std::fs::write(root.join("request.json"), &raw).map_err(|error| error.to_string())?;
    let (result, profile) = match request {
        Input::Standalone { guest, network } => {
            let request = guest.request()?;
            let timeout = request.limits.wall_timeout;
            let (result, profile) = gomad::profile::capture(|| {
                gomad::backstop::within(timeout, || {
                    let mut prepared = linked::prepare(request)?;
                    if let Some(network) = network {
                        prepared.network(network)?;
                    }
                    linked::execute_prepared(prepared)
                })
            })?;
            let result = result.map_err(|error| error.to_string());
            if let Ok(execution) = &result {
                retain(root, execution)?;
            }
            (
                result.map(|execution| json!({"termination":execution.outcome.termination})),
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
        } => {
            let request = linked_cluster::Request {
                nodes,
                broker,
                network,
                steps,
                boundaries,
                crash_groups,
                evidence_bytes,
                wall_timeout: Duration::from_millis(wall_millis),
                incarnations: incarnations
                    .into_iter()
                    .map(|incarnation| {
                        Ok(linked_cluster::Incarnation {
                            handle: incarnation.handle,
                            request: incarnation.guest.request()?,
                            network: incarnation.network,
                        })
                    })
                    .collect::<Result<_, String>>()?,
            };
            let timeout = request.wall_timeout;
            let (result, profile) = gomad::profile::capture(|| {
                gomad::backstop::within(timeout, || {
                    linked_cluster::prepare(request).and_then(linked_cluster::execute_prepared)
                })
            })?;
            let result = result.map_err(|error| error.to_string());
            if let Ok(execution) = &result {
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
                    std::fs::write(root.join(name), bytes).map_err(|error| error.to_string())?;
                }
                write(
                    &root.join("cutoff.json"),
                    &json!({"virtual_elapsed_nanos":execution.virtual_elapsed_nanos}),
                )?;
                for (index, incarnation) in execution.incarnations.iter().enumerate() {
                    let directory = root.join(format!("incarnation-{index}"));
                    std::fs::create_dir(&directory).map_err(|error| error.to_string())?;
                    retain(&directory, &incarnation.execution)?;
                    write(&directory.join("handle.json"), &incarnation.handle)?;
                }
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
    write(&root.join("profile.json"), &profile)?;
    let binary = std::fs::read(std::env::current_exe().map_err(|error| error.to_string())?)
        .map_err(|error| error.to_string())?;
    let validation = gomad::profile::validate(&profile);
    let report: Value = json!({
        "schema":"gomad.linked-profile-run/v1", "diagnostic_only":true,
        "request_sha256":canonical::digest(&raw), "host_sha256":canonical::digest(&binary),
        "linked_implementation_sha256":linked::IMPLEMENTATION_SHA256,
        "host_implementation_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "configuration":linked::CONFIGURATION, "features":["diagnostic-profile"],
        "invocation":std::env::args().collect::<Vec<_>>(),
        "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
        "execution":result, "attribution":validation,
        "qualification":"unavailable: diagnostic build; fn-185 R4 remains pending fn-232.13",
        "scope":"elapsed_wall_nanos covers linked admission/compilation/execution/teardown; input-file reads and output publication are excluded; store spans are thread-time; no elapsed Amdahl bound is inferred"
    });
    write(&root.join("report.json"), &report)?;
    validation?;
    result.map(|_| ())
}
fn main() {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    let result = if arguments.len() == 2 && arguments[0] == "check" {
        std::fs::read(&arguments[1])
            .map_err(|error| error.to_string())
            .and_then(|raw| gomad::profile::decode(&raw).map(|_| ()))
    } else if arguments.len() == 2 {
        run(Path::new(&arguments[0]), Path::new(&arguments[1]))
    } else {
        Err("usage: gomad-profile INPUT.json NEW-OUTPUT-DIRECTORY".into())
    };
    if let Err(error) = result {
        eprintln!("gomad-profile: {error}");
        std::process::exit(3);
    }
}
