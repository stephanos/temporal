//! Repository gate seam linked to the ordinary release library. The public
//! operator replay grammar and diagnostic comparison policy remain unchanged.

use crate::{
    evidence::{canonical, store::Opened},
    replay::{divergence, preflight},
};
use std::path::Path;

/// Linked component identities emitted by the harness before admitting evidence.
#[must_use]
pub fn identity() -> serde_json::Value {
    serde_json::json!({
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "runtime_implementation_sha256":env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"),
        "engine":gomad_wasmhost::linked::Conditions::compiled().configuration(),
        "release_role":!cfg!(debug_assertions) && !cfg!(feature="determinism-fixture")
            && gomad_wasmhost::linked::Conditions::compiled()==gomad_wasmhost::linked::Conditions::Ordinary
    })
}

pub(crate) fn configuration() -> Result<preflight::Configuration, String> {
    serde_json::from_value(serde_json::json!({
        "engine":gomad_wasmhost::linked::Conditions::compiled().configuration(),
        "profile":"gomad.wasi-cooperative/v1", "runtime_sha256":null,
        "arguments":["guest","literal $arg"],
        "environment":{"PWD":"/workspace","TZ":"UTC","ONLY":"captured","EXTRA_ENTROPY":"no_"},
        "limits":{"fuel":4_000_000_000_u64,"memory_bytes":256<<20,"wall_millis":60000,"output_bytes":1<<20,"evidence_bytes":128<<20},
        "model":{"entropy_key":[1,2,3,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
            "working_directory_sha256":canonical::digest(b"/workspace"),
            "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
            "mode":1,"limit":1<<20,"diagnostics":true,"epoch_nanos":946_684_800_000_000_000_i64,
            "read_step_nanos":1000,"now":0,"limits":{"calls":10000,"transcript_bytes":8<<20,"pending_events":4096}},
        "namespace":{"entries":[],"writable_directories":[b"/workspace".as_slice(),b"/tmp".as_slice()],
            "preopens":[b"/tmp".as_slice(),b"/workspace".as_slice()],"working_directory":b"/workspace".as_slice(),"stdin":[],
            "epoch_nanos":946_684_800_000_000_000_i64,"limits":{"files":1000,"descriptors":64,"filesystem_bytes":8<<20,"output_bytes":1<<20}}
    })).map_err(|error| error.to_string())
}

/// Records exactly 25 fresh stores for one original runtime seed and forces
/// one fresh full-tape replay. Source preparation and artifact publication use
/// the production owners and the original guest request configuration.
///
/// # Errors
/// Refuses indices outside 0..32, nonrelease roles, divergent recordings,
/// execution failures, and existing output roots.
pub fn produce(
    installation: &Path,
    source: &Path,
    index: u64,
    output: &Path,
) -> Result<(), String> {
    release_role()?;
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(600);
    if index >= 32 {
        return Err("runtime qualification index must be 0..31".into());
    }
    std::fs::create_dir(output).map_err(|error| error.to_string())?;
    let options = options(installation, source)?;
    let prepared = crate::campaign::prepare::prepare_runtime_qualification(&options)?;
    let mut artifacts = Vec::new();
    let mut baseline = None;
    for instance in 0..25 {
        let remaining = deadline
            .checked_duration_since(std::time::Instant::now())
            .filter(|duration| !duration.is_zero())
            .ok_or("runtime qualification shard deadline exceeded")?;
        let result = crate::campaign::execution::run(
            &prepared,
            &crate::campaign::ledger::Job {
                ordinal: instance,
                seed: index * 1_000_003,
            },
            std::time::Duration::from_secs(60).min(remaining),
            &gomad_wasmhost::linked::Cancellation::default(),
        )?;
        let publication = result.publication;
        if publication.record.outcome.termination != "exit"
            || !matches!(publication.record.outcome.exit_code, Some(0 | 17))
        {
            return Err("runtime qualification guest did not complete".into());
        }
        if let Some((first_record, first_payloads)) = &baseline {
            if first_record != &publication.record || first_payloads != &publication.payloads {
                return Err(format!(
                    "runtime qualification fresh instance {instance} diverged"
                ));
            }
        } else {
            baseline = Some((publication.record.clone(), publication.payloads.clone()));
        }
        let store = crate::evidence::store::Store {
            root: output.join(format!("instance-{instance:02}")),
            maximum_bytes: 512 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        };
        artifacts.push(store.publish(publication)?.path);
    }
    replay_before_deadline(&artifacts[0], &output.join("replay"), Some(deadline))?;
    std::fs::write(
        output.join("artifacts.json"),
        canonical::encode(&artifacts)?,
    )
    .map_err(|error| error.to_string())
}

fn options(
    installation: &Path,
    source: &Path,
) -> Result<crate::cli::explore::ExploreOptions, String> {
    let arguments: Vec<String> = [
        "--choices",
        "--diagnostics",
        "--parallel=1",
        "--execution-timeout=60s",
        "--overall-timeout=600s",
        "--output-limit=1048576",
        "--choice-bytes=1048576",
        "go-run",
        ".",
        "--",
        "literal $arg",
    ]
    .map(str::to_owned)
    .into();
    let mut options = crate::cli::explore::parse(&arguments, &mut false)?;
    options.installation = Some(installation.into());
    options.working_dir = source.into();
    Ok(options)
}

/// Runs the runtime gate's fresh full-tape replay with diagnostics disabled,
/// retaining raw observed streams and a binding to the authenticated parent.
///
/// # Errors
/// Refuses nonrelease roles, incomplete recordings, divergent replay, and an
/// existing output directory.
pub fn replay(artifact: &Path, output: &Path) -> Result<(), String> {
    replay_before_deadline(artifact, output, None)
}

fn replay_before_deadline(
    artifact: &Path,
    output: &Path,
    deadline: Option<std::time::Instant>,
) -> Result<(), String> {
    release_role()?;
    let opened = Opened::open(artifact)?;
    let parent = opened.manifest().record_sha256.clone();
    let mut prepared = preflight::prepare_runtime_qualification(opened)?;
    if let Some(deadline) = deadline {
        let remaining = deadline
            .checked_duration_since(std::time::Instant::now())
            .filter(|duration| !duration.is_zero())
            .ok_or("runtime qualification shard deadline exceeded")?;
        if let preflight::Execution::Standalone(execution) = &mut prepared.execution {
            execution
                .bound_wall_timeout(remaining)
                .map_err(|error| error.to_string())?;
        }
    }
    let mut record = prepared.opened.manifest().record.clone();
    record.streams.remove("evidence/diagnostic");
    let observed = preflight::execute(prepared.execution)?;
    if !observed
        .evidence
        .get("diagnostic")
        .is_none_or(Vec::is_empty)
    {
        return Err("runtime qualification replay unexpectedly collected diagnostics".into());
    }
    if let Some(reason) = divergence::first(&record, &observed) {
        return Err(format!("runtime qualification replay diverged: {reason}"));
    }
    if observed.outcome.termination != "exit" || !matches!(observed.outcome.exit_code, Some(0 | 17))
    {
        return Err("runtime qualification replay did not complete".into());
    }
    std::fs::create_dir(output).map_err(|error| error.to_string())?;
    let mut streams = std::collections::BTreeMap::new();
    for name in ["stdout", "stderr", "transcript", "choice"] {
        let bytes = observed.evidence.get(name).map_or(&[][..], Vec::as_slice);
        std::fs::write(output.join(name), bytes).map_err(|error| error.to_string())?;
        streams.insert(name, canonical::digest(bytes));
    }
    let report = serde_json::json!({
        "schema":"gomad.runtime-qualification-replay/v1", "parent_record_sha256":parent,
        "diagnostics":false, "exact":true, "matched":true,
        "outcome":observed.outcome, "streams":streams,
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "runtime_implementation_sha256":env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
    });
    std::fs::write(output.join("report.json"), canonical::encode(&report)?)
        .map_err(|error| error.to_string())
}

fn release_role() -> Result<(), String> {
    crate::qualify::require_release_role()?;
    if cfg!(debug_assertions) || cfg!(feature = "determinism-fixture") {
        return Err("runtime qualification requires the ordinary release library".into());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    #[test]
    fn original_preparation_options_preserve_target_arguments_and_budgets() {
        let options = super::options(
            std::path::Path::new("/installation"),
            std::path::Path::new("/source"),
        )
        .unwrap();
        assert_eq!(options.target_args, ["literal $arg"]);
        assert_eq!(options.source, ".");
        assert_eq!(options.target_kind, "go-run");
        assert_eq!(
            options.execution_timeout,
            std::time::Duration::from_secs(60)
        );
        assert_eq!(options.overall_timeout, std::time::Duration::from_secs(600));
        assert!(options.choices && options.diagnostics);
    }

    #[test]
    fn original_request_configuration_is_seed_independent_and_exact() {
        let value = serde_json::to_value(super::configuration().unwrap()).unwrap();
        assert_eq!(
            value["arguments"],
            serde_json::json!(["guest", "literal $arg"])
        );
        assert_eq!(
            value["environment"],
            serde_json::json!({"PWD":"/workspace","TZ":"UTC","ONLY":"captured","EXTRA_ENTROPY":"no_"})
        );
        assert_eq!(value["model"]["read_step_nanos"], 1000);
        assert_eq!(value["model"]["diagnostics"], true);
        assert!(value["model"].get("seed").is_none());
        assert_eq!(value["limits"]["fuel"], 4_000_000_000_u64);
        assert_eq!(value["limits"]["wall_millis"], 60000);
        assert_eq!(value["limits"]["memory_bytes"], 256 << 20);
        assert!(value.get("derive_seed_environment").is_none());
        assert!(value["runtime_sha256"].is_null());
    }
}
