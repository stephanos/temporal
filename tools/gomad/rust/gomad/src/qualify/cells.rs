//! Freeze and run one qualification cell on the linked Rust host: one frozen
//! preparation, `Repeat` fresh seed campaigns (each execution in a fresh
//! Wasmtime engine and store), then a fresh replay of every retained artifact.

use super::{
    cost::{self, Observations, Sample},
    report::{self, Execution, Failure, Replay},
    result::{self, Manifest, QualificationResult, Receipt},
    spec::{self, Spec},
    usage::{self, Snapshot},
};
use crate::{
    campaign::{self, execution::Metrics, prepare},
    cli::explore::ExploreOptions,
    evidence::canonical,
    replay::{divergence, preflight},
};
use serde_json::Value;
use std::{
    path::{Path, PathBuf},
    time::Instant,
};

/// The public qualification command's outcome before process exit mapping.
pub(crate) struct Qualified {
    pub path: PathBuf,
    pub receipt: Receipt,
    pub result: QualificationResult,
    pub publication_error: Option<String>,
}

pub(crate) struct Frozen {
    pub options: ExploreOptions,
    pub prepared: prepare::Prepared,
    /// The decoded provenance the manifest identities were bound to.
    pub provenance: Value,
    pub manifest: Manifest,
    pub integrated_source_manifest: Option<String>,
    pub artifact_root: PathBuf,
    pub command: Vec<String>,
    pub preparation_wall_nanos: u64,
    /// Compiler arguments of the commands the initial preparation actually ran.
    pub initial_compiler_commands: Vec<Vec<String>>,
    pub start: Option<Snapshot>,
    pub prepared_usage: Option<Snapshot>,
}

fn nanos(value: u64, name: &str) -> Result<String, String> {
    if value == 0 {
        return Err(format!("qualification campaign {name} must be positive"));
    }
    Ok(format!("{value}ns"))
}

/// Maps the owner campaign onto the public explore grammar so qualification
/// inherits its validation, then fixes the qualification-owned selection.
fn explore_options(
    spec: &Spec,
    installation: Option<&Path>,
    seed: u64,
    cooperative: bool,
) -> Result<ExploreOptions, String> {
    let campaign = &spec.campaign;
    let target = &campaign.target;
    if target.backend != "wasm" {
        return Err("unsupported WASM source preparation request".into());
    }
    let scenario = target.scenario.as_ref().map(scenario_bytes).transpose()?;
    // A scenario owns its cluster lifecycle; explore admits it only without
    // seed-level choices or coverage, as the public grammar does.
    let cooperative = cooperative && scenario.is_none();
    if !matches!(target.kind.as_str(), "go-run" | "go-test") || target.source.is_empty() {
        return Err("unsupported WASM source preparation request".into());
    }
    if !target.preparation_root.is_empty() && !Path::new(&target.preparation_root).is_absolute() {
        return Err("qualification preparation root must be absolute".into());
    }
    let artifacts = if campaign.artifacts.is_empty() {
        spec.artifact_root.to_string_lossy().into_owned()
    } else {
        campaign.artifacts.clone()
    };
    let mut arguments = vec![
        "--json".to_owned(),
        "--seeds".into(),
        seed.to_string(),
        "--parallel".into(),
        "1".into(),
        "--execution-timeout".into(),
        nanos(campaign.execution_timeout, "execution timeout")?,
        "--overall-timeout".into(),
        nanos(campaign.overall_timeout, "overall timeout")?,
        "--on-failure".into(),
        if campaign.on_failure.is_empty() {
            "first".into()
        } else {
            campaign.on_failure.clone()
        },
        "--failure-budget".into(),
        campaign.failure_budget.max(1).to_string(),
        "--artifacts".into(),
        artifacts,
        "--keep-successes".into(),
        "all".into(),
        "--success-limit".into(),
        "1".into(),
        "--success-bytes".into(),
        campaign.success_bytes_limit.max(256 << 20).to_string(),
    ];
    if campaign.terminate_grace != 0 {
        arguments.extend([
            "--terminate-grace".into(),
            format!("{}ns", campaign.terminate_grace),
        ]);
    }
    if campaign.output_limit != 0 {
        arguments.extend(["--output-limit".into(), campaign.output_limit.to_string()]);
    }
    if campaign.world_transition_limit != 0 {
        arguments.extend([
            "--world-transition-limit".into(),
            campaign.world_transition_limit.to_string(),
        ]);
    }
    if cooperative {
        arguments.extend([
            "--choices".into(),
            "--choice-bytes".into(),
            campaign
                .choice_trace_limit
                .clamp(1 << 20, 64 << 20)
                .to_string(),
            "--coverage".into(),
            "choice".into(),
        ]);
    }
    for entry in &campaign.environment {
        arguments.extend(["--env".into(), entry.clone()]);
    }
    for tag in &target.build_tags {
        arguments.extend(["--build-tag".into(), tag.clone()]);
    }
    if !target.working_dir.is_empty() {
        arguments.extend(["--working-dir".into(), target.working_dir.clone()]);
    }
    arguments.extend([target.kind.clone(), target.source.clone()]);
    if !target.args.is_empty() {
        arguments.push("--".into());
        arguments.extend(target.args.iter().cloned());
    }
    let mut json = false;
    let mut options = crate::cli::explore::parse(&arguments, &mut json)?;
    options.installation = installation.map(Path::to_path_buf);
    options.scenario = scenario;
    original_choice_bound(spec, &mut options)?;
    Ok(options)
}

fn original_choice_bound(spec: &Spec, options: &mut ExploreOptions) -> Result<(), String> {
    if spec.manifest.id == "temporal-committed-workflow-crash-reopen" {
        if spec.campaign.choice_trace_limit != 1 << 30
            || options.scenario.as_ref().is_none_or(|raw| {
                serde_json::from_slice::<Value>(raw)
                    .ok()
                    .is_none_or(|scenario| {
                        scenario["workload"]["kind"] != "temporal-committed-workflow-crash-reopen"
                    })
            })
        {
            return Err("committed Temporal recovery requires its exact registered original 1GiB choice bound".into());
        }
        options.choice_bytes = 1 << 30;
    }
    if matches!(
        spec.runtime_workload,
        Some(super::workloads::recovery::Request::TemporalRetry)
    ) {
        if spec.manifest.id != "temporal-named-network-reset"
            || spec.campaign.choice_trace_limit != 1 << 30
        {
            return Err(
                "named Temporal reset requires its original registered 1GiB choice bound".into(),
            );
        }
        options.choice_bytes = 1 << 30;
    }
    Ok(())
}

/// The owner spec carries `Target.Scenario` as Go's base64 `[]byte` encoding of
/// the same bounded scenario JSON that `explore --scenario FILE` reads.
fn scenario_bytes(value: &Value) -> Result<Vec<u8>, String> {
    let text = value
        .as_str()
        .ok_or("qualification Target.Scenario must be base64 scenario JSON")?;
    let bytes = base64(text).ok_or("qualification Target.Scenario is not valid base64")?;
    if bytes.is_empty()
        || bytes.len() > 256 << 10
        || serde_json::from_slice::<Value>(&bytes).is_err()
    {
        return Err("scenario must contain bounded valid JSON".into());
    }
    Ok(bytes)
}

/// Standard padded base64, as Go's `encoding/json` emits for `[]byte`.
pub(crate) fn base64(text: &str) -> Option<Vec<u8>> {
    let digit = |byte: u8| match byte {
        b'A'..=b'Z' => Some(byte - b'A'),
        b'a'..=b'z' => Some(byte - b'a' + 26),
        b'0'..=b'9' => Some(byte - b'0' + 52),
        b'+' => Some(62),
        b'/' => Some(63),
        _ => None,
    };
    let bytes = text.as_bytes();
    if !bytes.len().is_multiple_of(4) {
        return None;
    }
    let mut output = Vec::with_capacity(bytes.len() / 4 * 3);
    for (index, chunk) in bytes.chunks(4).enumerate() {
        let last = index + 1 == bytes.len() / 4;
        let padding = chunk.iter().rev().take_while(|byte| **byte == b'=').count();
        if padding > 2 || padding > 0 && !last {
            return None;
        }
        let mut value = 0_u32;
        for byte in &chunk[..4 - padding] {
            value = value << 6 | u32::from(digit(*byte)?);
        }
        value <<= 6 * padding;
        let decoded = value.to_be_bytes();
        if padding > 0 && decoded[4 - padding..].iter().any(|byte| *byte != 0) {
            return None;
        }
        output.extend_from_slice(&decoded[1..4 - padding]);
    }
    Some(output)
}

fn declared(declared: &str, observed: &str, name: &str) -> Result<(), String> {
    if let Some(digest) = spec::declared_digest(declared)?
        && digest != observed
    {
        return Err(format!(
            "declared WASM qualification identity changed during freeze: {name}"
        ));
    }
    Ok(())
}

/// Freezes source, toolchain, linked implementation, model and limits from one
/// actual preparation. Declared identities must match what was observed.
#[expect(
    clippy::too_many_lines,
    reason = "One freeze binds every declared identity to its single observation"
)]
pub(crate) fn freeze(
    spec: &Spec,
    installation: Option<&Path>,
    command: Vec<String>,
    start: Option<Snapshot>,
) -> Result<Frozen, String> {
    crate::qualify::require_release_role()?;
    let declared_manifest = &spec.manifest;
    if !declared_manifest.provisional && start.is_none() {
        return Err("final qualification requires operator baseline before setup".into());
    }
    if !spec.artifact_root.is_absolute() {
        return Err("absolute WASM qualification result root required".into());
    }
    let integrated = spec::declared_digest(&declared_manifest.integrated_source_sha256)?
        .ok_or("unbound WASM qualification identity: integrated source")?;
    let integrated_source_manifest = if declared_manifest.integrated_source_manifest_path.is_empty()
    {
        if !declared_manifest.provisional {
            return Err("final WASM qualification requires source manifest".into());
        }
        None
    } else {
        let encoded = spec::read_bounded(
            Path::new(&declared_manifest.integrated_source_manifest_path),
            16 << 20,
        )?;
        if canonical::digest(&encoded) != integrated {
            return Err("integrated source manifest digest changed".into());
        }
        result::validate_source_manifest(&encoded, true)?;
        Some(String::from_utf8(encoded).map_err(|_| "integrated source manifest is not UTF-8")?)
    };
    // The installation's pinned profile selects the cooperative choice grammar.
    if let Some(request) = &spec.runtime_workload {
        let expected = match request {
            super::workloads::recovery::Request::SQLiteSelection { .. } => {
                "sqlite-durable-recovery"
            }
            super::workloads::recovery::Request::TemporalRecovery { .. } => {
                "temporal-committed-workflow-crash-reopen"
            }
            super::workloads::recovery::Request::TemporalRetry => "temporal-named-network-reset",
        };
        if declared_manifest.id != expected {
            return Err(
                "runtime workload requires its exact registered qualification identity".into(),
            );
        }
    }
    let probe = explore_options(spec, installation, declared_manifest.seed, false)?;
    let cooperative = prepare::installation_profile(&probe)? == "gomad.wasi-cooperative/v1";
    let options = explore_options(spec, installation, declared_manifest.seed, cooperative)?;
    let started = Instant::now();
    let (mut prepared, initial_compiler_commands) = prepare::prepare_observed(&options)?;
    if let Some(request) = &spec.runtime_workload {
        super::workloads::recovery::attach(&mut prepared, request)?;
    }
    let preparation_wall_nanos = u64::try_from(started.elapsed().as_nanos()).unwrap_or(u64::MAX);
    let prepared_usage = usage::snapshot();
    let configuration = &prepared.configuration;
    let provenance: Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    let inputs = &provenance["inputs"];
    let digest = |value: &Value| canonical::encode(value).map(|bytes| canonical::digest(&bytes));
    let manifest = Manifest {
        schema: result::MANIFEST_SCHEMA.into(),
        provisional: declared_manifest.provisional,
        integrated_source_manifest_path: (!declared_manifest
            .integrated_source_manifest_path
            .is_empty())
        .then(|| declared_manifest.integrated_source_manifest_path.clone()),
        id: declared_manifest.id.clone(),
        workload: declared_manifest.workload.clone(),
        profile: result::profile_name(&configuration.profile)?.into(),
        cell: (!declared_manifest.cell.is_empty()).then(|| declared_manifest.cell.clone()),
        integrated_source_sha256: integrated,
        workload_source_sha256: digest(&inputs["sources"])?,
        runtime_sha256: digest(&inputs["compiler"])?,
        compiler_sha256: inputs["compiler_sha256"]
            .as_str()
            .ok_or("prepared compiler identity is missing")?
            .into(),
        helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        model_sha256: digest(configuration.model.as_ref().unwrap_or(&Value::Null))?,
        engine_sha256: canonical::digest(configuration.engine.as_bytes()),
        host_config_sha256: canonical::digest(&canonical::encode(configuration)?),
        module_sha256: canonical::digest(&prepared.module),
        toolchain_sha256: prepared.toolchain.clone(),
        scenario_sha256: prepared.scenario.as_deref().map(canonical::digest),
        host_os: std::env::consts::OS.into(),
        host_arch: std::env::consts::ARCH.into(),
        seed: declared_manifest.seed,
        repeat: declared_manifest.repeat,
        fuel: configuration.limits.fuel,
        memory_bytes: configuration.limits.memory_bytes,
        unsupported: declared_manifest.unsupported.clone(),
    };
    // The frozen identities are exactly those `report` recomputes later.
    result::bind_provenance(&manifest, &provenance)?;
    for (name, value, observed) in [
        (
            "workload source",
            &declared_manifest.workload_source_sha256,
            &manifest.workload_source_sha256,
        ),
        (
            "runtime",
            &declared_manifest.runtime_sha256,
            &manifest.runtime_sha256,
        ),
        (
            "compiler",
            &declared_manifest.compiler_sha256,
            &manifest.compiler_sha256,
        ),
        (
            "helper",
            &declared_manifest.helper_sha256,
            &manifest.helper_sha256,
        ),
        (
            "model",
            &declared_manifest.model_sha256,
            &manifest.model_sha256,
        ),
        (
            "engine",
            &declared_manifest.engine_sha256,
            &manifest.engine_sha256,
        ),
        (
            "host config",
            &declared_manifest.host_config_sha256,
            &manifest.host_config_sha256,
        ),
    ] {
        declared(value, observed, name)?;
    }
    if !declared_manifest.profile.is_empty() && declared_manifest.profile != manifest.profile
        || !declared_manifest.host_os.is_empty() && declared_manifest.host_os != manifest.host_os
        || !declared_manifest.host_arch.is_empty()
            && declared_manifest.host_arch != manifest.host_arch
        || declared_manifest.fuel != 0 && declared_manifest.fuel != manifest.fuel
        || declared_manifest.memory_bytes != 0
            && declared_manifest.memory_bytes != manifest.memory_bytes
        || !declared_manifest.schema.is_empty()
            && !matches!(
                declared_manifest.schema.as_str(),
                result::MANIFEST_SCHEMA | "gomad-wasm.qualification-manifest/v1"
            )
    {
        return Err("declared WASM qualification profile or limits changed during freeze".into());
    }
    manifest.validate()?;
    Ok(Frozen {
        options,
        prepared,
        provenance,
        manifest,
        integrated_source_manifest,
        artifact_root: spec.artifact_root.clone(),
        command,
        preparation_wall_nanos,
        initial_compiler_commands,
        start,
        prepared_usage,
    })
}

struct Iteration {
    execution: Option<Execution>,
    samples: Vec<Sample>,
}

/// One sample per fresh store; concurrent cluster stores share the window.
fn samples(
    phase: &str,
    iteration: u64,
    metrics: &Metrics,
    virtual_time_nanos: u64,
    before: Option<usage::Usage>,
    after: Option<usage::Usage>,
) -> Vec<Sample> {
    metrics
        .stores
        .iter()
        .map(|store| Sample {
            phase: phase.into(),
            iteration,
            incarnation: store.incarnation.clone(),
            wall_nanos: metrics.wall_nanos,
            fuel_used: store.fuel_used,
            guest_peak_memory_bytes: store.guest_peak_memory_bytes,
            guest_memory_limit_bytes: store.guest_memory_limit_bytes,
            trace_bytes: store.trace_bytes,
            virtual_time_nanos,
            linked_before: before,
            linked_after: after,
            store_peak_rss_bytes: metrics.store_peak_rss_bytes,
            cache_hit: metrics.cache_hit,
        })
        .collect()
}

/// Runs one fresh single-seed campaign and returns its committed execution.
fn execute(frozen: &Frozen, iteration: u64) -> Result<Iteration, String> {
    let mut observed = None;
    let before = usage::linked();
    let mut output = Vec::new();
    let mut commit =
        |campaign: &Path, execution: &campaign::ledger::Execution, metrics: &Metrics| {
            observed = Some((campaign.to_path_buf(), execution.clone(), metrics.clone()));
        };
    let status = campaign::explore_prepared(
        &frozen.options,
        &frozen.prepared,
        Instant::now(),
        Some(&mut commit),
        &mut output,
    );
    let after = usage::linked();
    let Some((campaign_path, committed, metrics)) = observed else {
        status?;
        return Ok(Iteration {
            execution: None,
            samples: Vec::new(),
        });
    };
    let artifact = committed
        .artifact
        .as_ref()
        .map(|reference| campaign_path.join(reference).to_string_lossy().into_owned());
    let record_sha256 = committed.record.hash()?;
    let samples = samples(
        "execute",
        iteration,
        &metrics,
        committed.record.virtual_time_nanos,
        before,
        after,
    );
    let execution = Execution {
        iteration,
        campaign_path: campaign_path.to_string_lossy().into_owned(),
        artifact_path: artifact,
        record: committed.record,
        record_sha256,
        replay: None,
    };
    status?;
    Ok(Iteration {
        execution: Some(execution),
        samples,
    })
}

/// Replays one retained artifact in fresh stores through the shared opener.
/// A retained cluster replays every incarnation; each store is measured.
fn replay(
    path: &Path,
    iteration: u64,
    compile_cache: Option<&str>,
) -> Result<(Replay, Vec<Sample>), String> {
    let before = usage::linked();
    let mut prepared = preflight::open(path).map_err(|error| format!("runner_failure: {error}"))?;
    // Replay loads the module the executions compiled into the installation's
    // cache, as the public replay verb does with that installation.
    prepared
        .compile_cache(compile_cache)
        .map_err(|error| format!("runner_failure: {error}"))?;
    let record = prepared.opened.manifest().record.clone();
    if !prepared.available {
        return Ok((
            Replay {
                attempted: true,
                matched: false,
                divergence: Some("choice_profile.replay_unavailable".into()),
                choice_replay: "unavailable".into(),
            },
            Vec::new(),
        ));
    }
    let exact = prepared.exact;
    let started = Instant::now();
    let (observed, metrics, virtual_time) = match prepared.execution {
        preflight::Execution::Cluster(cluster) => {
            let projection = cluster.projection();
            usage::store_begin();
            let observed = crate::replay::cluster::execute(*cluster);
            let store_peak = usage::store_end();
            let observed = observed
                .map_err(|error| format!("runner_failure: execute replay target: {error}"))?;
            // Every retained incarnation runs under the record's shared bounds.
            let limits = observed
                .incarnations
                .iter()
                .map(|item| {
                    (
                        format!("{}/{}", item.handle.node, item.handle.incarnation),
                        (record.limits.fuel, record.limits.memory_bytes),
                    )
                })
                .collect();
            let mut metrics =
                campaign::scenario::store_metrics(&observed, &limits, started.elapsed())?;
            metrics.store_peak_rss_bytes = store_peak;
            let virtual_time = observed.virtual_elapsed_nanos;
            let projected = projection
                .project(observed)
                .map_err(|error| format!("runner_failure: project cluster replay: {error}"))?;
            (projected, metrics, virtual_time)
        }
        execution @ preflight::Execution::Standalone(_) => {
            usage::store_begin();
            let observed = preflight::execute(execution);
            let wall = started.elapsed();
            let store_peak = usage::store_end();
            let observed = observed
                .map_err(|error| format!("runner_failure: execute replay target: {error}"))?;
            let mut metrics = Metrics::observed(&observed, &record.limits, wall);
            metrics.store_peak_rss_bytes = store_peak;
            let virtual_time = observed
                .outcome
                .local_model
                .as_deref()
                .and_then(|model| model["now"].as_u64())
                .unwrap_or(0);
            (observed, metrics, virtual_time)
        }
    };
    let after = usage::linked();
    let divergence = divergence::first(&record, &observed);
    let choice_replay = if observed.outcome.divergence.is_some() {
        "diverged"
    } else if exact {
        "exact"
    } else {
        "none"
    };
    Ok((
        Replay {
            attempted: true,
            matched: divergence.is_none(),
            divergence,
            choice_replay: choice_replay.into(),
        },
        samples("replay", iteration, &metrics, virtual_time, before, after),
    ))
}

/// The largest number of distinct nodes resident in one execution window.
fn resident_stores(samples: &[Sample]) -> u64 {
    let mut windows =
        std::collections::BTreeMap::<(&str, u64), std::collections::BTreeSet<&str>>::new();
    for sample in samples {
        let node = sample.incarnation.as_deref().map_or("", |name| {
            name.split_once('/').map_or(name, |(node, _)| node)
        });
        windows
            .entry((sample.phase.as_str(), sample.iteration))
            .or_default()
            .insert(node);
    }
    windows
        .values()
        .map(|nodes| nodes.len() as u64)
        .max()
        .unwrap_or(0)
}

fn failure(error: &str, iteration: u64) -> Failure {
    let (classification, message) = report::classify_error(error);
    Failure {
        classification: classification.into(),
        message,
        iteration,
    }
}

/// Runs every repetition and replay. A failed repetition or replay stops the
/// cell and retains an explicit failure report with the completed executions.
pub(crate) fn run(frozen: &Frozen) -> Result<QualificationResult, String> {
    let manifest = &frozen.manifest;
    let mut executions = Vec::new();
    let mut samples = Vec::new();
    let mut stopped = None;
    for iteration in 1..=manifest.repeat {
        match execute(frozen, iteration) {
            Ok(Iteration {
                execution: Some(execution),
                samples: measured,
            }) => {
                let retained = execution.artifact_path.is_some();
                executions.push(execution);
                samples.extend(measured);
                if !retained {
                    stopped = Some(Failure {
                        classification: "runner_failure".into(),
                        message: "Runner omitted the retained execution artifact".into(),
                        iteration,
                    });
                    break;
                }
            }
            Ok(Iteration {
                execution: None, ..
            }) => {
                stopped = Some(Failure {
                    classification: "runner_failure".into(),
                    message: "Runner omitted bounded qualification evidence".into(),
                    iteration,
                });
                break;
            }
            Err(error) => {
                stopped = Some(failure(&error, iteration));
                break;
            }
        }
    }
    if stopped.is_none() {
        let compile_cache = match &frozen.options.installation {
            Some(path) => prepare::compile_cache_root(
                &crate::install::manifest::resolve(Some(path))?.manifest,
            )?,
            None => None,
        };
        for (index, execution) in executions.iter_mut().enumerate() {
            let path = PathBuf::from(
                execution
                    .artifact_path
                    .clone()
                    .ok_or("qualification execution lost its artifact")?,
            );
            match replay(&path, index as u64 + 1, compile_cache.as_deref()) {
                Ok((replay, measured)) => {
                    execution.replay = Some(replay);
                    samples.extend(measured);
                }
                Err(error) => {
                    execution.replay = Some(Replay {
                        attempted: true,
                        matched: false,
                        divergence: Some(error.clone()),
                        choice_replay: "unavailable".into(),
                    });
                    let mut failed = failure(&error, index as u64 + 1);
                    failed.classification = if failed.classification == "invalid_input" {
                        "runner_failure".into()
                    } else {
                        failed.classification
                    };
                    stopped = Some(failed);
                    break;
                }
            }
        }
    }
    let report = report::build(
        &frozen.command,
        manifest.seed,
        manifest.repeat,
        executions,
        stopped,
    )?;
    let observations = Observations {
        operator_process_id: u64::from(std::process::id()),
        resident_stores: resident_stores(&samples),
        start: frozen.start,
        prepared: frozen.prepared_usage,
        executed: usage::snapshot(),
        preparation_wall_nanos: frozen.preparation_wall_nanos,
        initial_preparation_compiled: cost::preparation_compiled(&frozen.initial_compiler_commands),
        initial_compiler_commands: frozen.initial_compiler_commands.clone(),
        samples,
        operator_peak_rss_bytes: usage::operator_peak(),
    };
    complete_workload(frozen, report, observations)
}

fn complete_workload(
    frozen: &Frozen,
    report: report::Report,
    observations: Observations,
) -> Result<QualificationResult, String> {
    let (cost, cost_failure, budget, budget_failure) = cost::evaluate(&observations, &report);
    let mut result = QualificationResult {
        schema: result::RESULT_SCHEMA.into(),
        manifest: frozen.manifest.clone(),
        report,
        observations,
        cost,
        cost_failure,
        budget,
        budget_failure,
        integrated_source_manifest: frozen.integrated_source_manifest.clone(),
        provenance: frozen.provenance.clone(),
        native_crash_catalogues: Vec::new(),
        native_crash_selection: None,
    };
    (
        result.native_crash_catalogues,
        result.native_crash_selection,
    ) = super::workloads::retained(&result)?;
    Ok(result)
}

/// Publishes the Result, then observes and publishes the terminal receipt.
pub(crate) fn publish(frozen: &Frozen, result: QualificationResult) -> Result<Qualified, String> {
    let path = result::publish(&frozen.artifact_root, &result)?;
    let digest = path
        .file_name()
        .and_then(|name| name.to_str())
        .and_then(|name| name.strip_prefix("result-"))
        .and_then(|name| name.strip_suffix(".json"))
        .ok_or("published Result is not content addressed")?
        .to_owned();
    let receipt = Receipt {
        schema: result::RECEIPT_SCHEMA.into(),
        result_sha256: digest.clone(),
        operator_process_id: u64::from(std::process::id()),
        linked_after: usage::linked(),
        operator_peak_rss_bytes: usage::operator_peak(),
    };
    let observation = match result::validate_receipt(&result, &receipt, &digest) {
        Ok(terminal) if terminal.status != "within" => Some(
            "terminal linked-process peak exceeds the operator-only 2GiB ceiling and cannot be attributed"
                .to_owned(),
        ),
        Ok(_) => None,
        Err(error) => Some(error),
    };
    let publication = result::publish_receipt(&path, &receipt).err();
    let publication_error = match (publication, observation) {
        (None, None) => None,
        (Some(error), None) | (None, Some(error)) => Some(error),
        (Some(first), Some(second)) => Some(format!("{first}; {second}")),
    };
    Ok(Qualified {
        path,
        receipt,
        result,
        publication_error,
    })
}

#[cfg(test)]
mod replay_reach_tests {
    /// Original `TestHostMemoryReplayErrorRetainsStagePairWithoutDispatch`: a
    /// missing retained artifact fails at open, before any store is dispatched.
    /// The Go-only stage-observer pair has no Rust counterpart.
    #[test]
    fn missing_retained_artifact_fails_at_open_without_dispatch() {
        let missing = std::env::temp_dir().join(format!(
            "gomad-missing-retained-artifact-{}/missing",
            std::process::id()
        ));
        let Err(error) = super::replay(&missing, 0, None) else {
            panic!("missing retained artifact replayed");
        };
        assert!(error.starts_with("runner_failure: "), "{error}");
        assert!(!error.contains("execute replay target"), "{error}");
    }

    /// Original `TestRetainedClusterFreezeRejectsUnobservedPreparation`: a
    /// final freeze without the operator's own observed baseline is refused
    /// before any preparation; Rust freeze always prepares itself, so a
    /// detached caller-supplied preparation cannot be passed.
    #[test]
    fn final_freeze_refuses_unobserved_operator_baseline() {
        let spec: crate::qualify::spec::Spec = serde_json::from_value(
            serde_json::json!({"Manifest":{"Provisional":false},"Campaign":{},"ArtifactRoot":"/retained"}),
        )
        .unwrap();
        let Err(error) = super::freeze(&spec, None, Vec::new(), None) else {
            panic!("unobserved preparation accepted");
        };
        assert_eq!(
            error,
            "final qualification requires operator baseline before setup"
        );
    }
}
