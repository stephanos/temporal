pub(crate) mod choice;
#[cfg(all(test, feature = "helper-determinism-fixture"))]
mod counterfactual;
pub(crate) mod execution;
pub(crate) mod exploration;
mod failure_policy;
pub(crate) mod inspect;
pub(crate) mod launcher;
pub(crate) mod ledger;
pub(crate) mod origin;
pub(crate) mod preparation;
pub(crate) mod prepare;
mod retention;
pub(crate) mod scenario;
pub(crate) mod seeds;
mod shards;
pub(crate) mod simulation;
mod simulation_execute;
pub(crate) mod soft_assertions;

use crate::{
    cli::explore::ExploreOptions,
    evidence::{
        canonical,
        identity::Identity,
        record::{Outcome, Record},
        store::{Opened, Publication, Store},
    },
    replay::preflight::{self, Configuration, Inputs},
};
use ledger::{Event, Execution, Job, Journal};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    io::Write,
    path::{Path, PathBuf},
    sync::mpsc,
    thread,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

pub(crate) fn execute_shard(
    plan: &Path,
    index: u64,
    count: u64,
    artifacts: &Path,
    json: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    shards::execute_shard(plan, index, count, artifacts, json, output)
}
pub(crate) fn merge(
    plan: &Path,
    shards: &[PathBuf],
    output_path: &Path,
    partial: bool,
    json: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    self::shards::merge(plan, shards, output_path, partial, json, output)
}

fn store(root: PathBuf, maximum_bytes: u64) -> Store {
    let target_pool = root
        .parent()
        .and_then(Path::parent)
        .map(|parent| parent.join(".campaign-targets"));
    Store {
        root,
        maximum_bytes,
        target_pool,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
}

fn campaign_path(artifacts: &Path) -> Result<PathBuf, String> {
    let time = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|error| error.to_string())?
        .as_nanos();
    Ok(artifacts
        .join("campaigns")
        .join(format!("campaign-{}-{time}", std::process::id())))
}

pub(crate) fn explore(options: &ExploreOptions, output: &mut dyn Write) -> Result<i32, String> {
    let started = Instant::now();
    let prepared = prepare::prepare(options)?;
    explore_prepared(options, &prepared, started, None, output)
}

/// Observes each committed execution with its campaign root and in-process measurements.
pub(crate) type Observer<'a> = &'a mut dyn FnMut(&Path, &Execution, &execution::Metrics);

/// Runs a seed campaign over an already admitted preparation; qualification
/// reuses one frozen preparation and still executes every seed in a fresh store.
pub(crate) fn explore_prepared(
    options: &ExploreOptions,
    prepared: &prepare::Prepared,
    started: Instant,
    observer: Option<Observer<'_>>,
    output: &mut dyn Write,
) -> Result<i32, String> {
    let mut journal = seed_journal(options, prepared)?;
    let remaining = options
        .overall_timeout
        .checked_sub(started.elapsed())
        .ok_or("overall_timeout: campaign deadline exceeded during preparation")?;
    run_observed(
        &mut journal,
        prepared,
        remaining,
        options.json,
        output,
        observer,
    )
}

fn seed_journal(options: &ExploreOptions, prepared: &prepare::Prepared) -> Result<Journal, String> {
    seed_journal_policy(options, prepared, false)
}

fn seed_journal_policy(
    options: &ExploreOptions,
    prepared: &prepare::Prepared,
    launcher: bool,
) -> Result<Journal, String> {
    let selection = seeds::SeedSelection::parse(&options.seeds)?;
    if selection.count() > 1_000_000 {
        return Err("capacity: campaign execution capacity exceeds 1000000".into());
    }
    let journal_bytes = selection
        .count()
        .max(1)
        .checked_mul(1 << 20)
        .ok_or("capacity: campaign journal byte capacity overflows")?;
    let launch = origin::from_preparation(prepared)?.and_then(|origin| origin.launch);
    let path = match launch {
        Some(launch) => options.artifacts.join("campaigns").join(format!(
            "campaign-{}{}",
            launch.invocation_id,
            if launcher { "" } else { "-prepared" },
        )),
        None if launcher => return Err("launcher campaign requires launch association".into()),
        None => campaign_path(&options.artifacts)?,
    };
    let mut plan = json!({
        "schema":"gomad.seed-campaign-plan/v2","selection":selection.to_string(),"selection_count":selection.count(),"parallel":options.parallel,
        "on_failure":options.on_failure,"failure_budget":options.failure_budget,"shard":null,"journal_bytes":journal_bytes,
        "overall_millis":u64::try_from(options.overall_timeout.as_millis()).map_err(|_|"overall timeout overflows")?,
        "overall_nanos":u64::try_from(options.overall_timeout.as_nanos()).map_err(|_|"overall timeout overflows")?,
        "terminate_grace_nanos":u64::try_from(options.terminate_grace.as_nanos()).map_err(|_|"termination grace overflows")?,
        "keep_successes":options.keep_successes,"success_limit":options.success_limit,"success_bytes":options.success_bytes,
        "coverage":options.coverage,"module_sha256":canonical::digest(&prepared.module),"toolchain":prepared.toolchain,
        "configuration":prepared.configuration,"choice_identity":prepared.choice_identity,
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "scenario":prepared.scenario,"network":prepared.network,"soft_assertion_report_only":prepared.soft_assertion_report_only,
    });
    if launcher {
        plan["launcher"] = json!(true);
    }
    retain_prepared(&path, prepared, &mut plan)?;
    Journal::create(&path, &plan, journal_bytes)
}

fn prepared_publication(prepared: &prepare::Prepared) -> Result<Publication, String> {
    let configuration = &prepared.configuration;
    let inputs = Inputs {
        schema: "gomad.replay-inputs/v1".into(),
        helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        toolchain_sha256: prepared.toolchain.clone(),
        runtime_implementation_sha256: (configuration.profile == "gomad.wasi-cooperative/v1")
            .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        configuration: serde_json::from_slice(&canonical::encode(configuration)?)
            .map_err(|error| error.to_string())?,
    };
    let bytes = canonical::encode(&inputs)?;
    let identity = Identity {
        target: canonical::digest(&prepared.module),
        toolchain: prepared.toolchain.clone(),
        platform: preflight::platform(),
        implementation: preflight::implementation()?,
        configuration: canonical::digest(&canonical::encode(configuration)?),
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        profile: configuration.profile.clone(),
        seed: 0,
        arguments: configuration.arguments.clone(),
        environment: configuration.environment.clone(),
        limits: configuration.limits.clone(),
        virtual_time_nanos: 0,
        outcome: Outcome {
            class: "prepared".into(),
            termination: "prepared".into(),
            exit_code: None,
            reason: String::new(),
        },
        streams: BTreeMap::from([(
            preflight::FILE.into(),
            crate::evidence::record::Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        )]),
    };
    let payloads = BTreeMap::from([
        ("target/module.wasm".into(), prepared.module.clone()),
        (preflight::FILE.into(), bytes),
        ("record.json".into(), record.bytes()?),
        ("target/provenance.json".into(), prepared.provenance.clone()),
    ]);
    let mut publication = Publication { record, payloads };
    if let Some(origin) = origin::from_preparation(prepared)? {
        origin.bind(&mut publication)?;
    }
    Ok(publication)
}

pub(crate) fn retain_prepared(
    path: &Path,
    prepared: &prepare::Prepared,
    plan: &mut Value,
) -> Result<(), String> {
    plan["origin"] = origin::from_preparation(prepared)?
        .map_or_else(|| json!("operator"), |origin| json!(origin));
    let publication = prepared_publication(prepared)?;
    let maximum = publication
        .payloads
        .values()
        .try_fold(32_u64 << 20, |total, bytes| {
            total
                .checked_add(bytes.len() as u64)
                .ok_or("prepared payload capacity overflow")
        })?;
    let published = store(path.join("prepared"), maximum).publish(publication)?;
    prepare::retain_observations(path, prepared, &published.record_sha256)?;
    plan["prepared_artifact"] = published
        .path
        .strip_prefix(path)
        .map_err(|error| error.to_string())?
        .to_string_lossy()
        .into_owned()
        .into();
    Ok(())
}

pub(crate) fn authenticate_preparation_observations(
    root: &std::fs::File,
    plan: &Value,
) -> Result<(), String> {
    use rustix::fs::{self, AtFlags};
    match fs::statat(
        root,
        prepare::OBSERVATION_DIRECTORY,
        AtFlags::SYMLINK_NOFOLLOW,
    ) {
        Err(rustix::io::Errno::NOENT) => return Ok(()),
        Err(error) => return Err(error.to_string()),
        Ok(_) => (),
    }
    let reference = plan["prepared_artifact"]
        .as_str()
        .filter(|reference| reference.starts_with("prepared/"))
        .ok_or("preparation observations require the plan's prepared artifact")?;
    let opened = ledger::open_artifact(root, reference)?;
    let prepared = restore_from_plan(plan, &opened)?;
    prepare::open_observations_at(root, &prepared, &opened.manifest().record_sha256)?;
    Ok(())
}

fn restore(journal: &Journal) -> Result<prepare::Prepared, String> {
    let plan = journal.plan();
    if plan["schema"] != "gomad.seed-campaign-plan/v2" {
        return Err("unsupported resumable seed campaign plan".into());
    }
    let opened = journal.open_artifact(
        plan["prepared_artifact"]
            .as_str()
            .ok_or("missing campaign prepared artifact")?,
    )?;
    let mut prepared = restore_from_plan(plan, &opened)?;
    prepared.auxiliary =
        prepare::open_observations(journal.path(), &prepared, &opened.manifest().record_sha256)?;
    Ok(prepared)
}

fn restore_from_plan(plan: &Value, opened: &Opened) -> Result<prepare::Prepared, String> {
    let record = &opened.manifest().record;
    if record.outcome.class != "prepared"
        || record.outcome.termination != "prepared"
        || record.outcome.exit_code.is_some()
        || !record.outcome.reason.is_empty()
        || record.seed != 0
        || record.virtual_time_nanos != 0
    {
        return Err("campaign preparation record is not an unexecuted target".into());
    }
    if plan["host_sha256"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || plan["helper_sha256"] != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
    {
        return Err("campaign implementation identity mismatch".into());
    }
    opened.authenticate_all()?;
    let bytes = opened.payload(preflight::FILE, 16 << 20)?;
    let inputs: Inputs = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    let configuration: Configuration =
        serde_json::from_value(plan["configuration"].clone()).map_err(|error| error.to_string())?;
    if canonical::encode(&inputs.configuration)? != canonical::encode(&configuration)?
        || opened.manifest().record.identity.configuration
            != canonical::digest(&canonical::encode(&configuration)?)
        || opened.manifest().record.identity.implementation != preflight::implementation()?
        || opened.manifest().record.identity.platform != preflight::platform()
        || inputs.helper_sha256 != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || inputs.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || plan["toolchain"] != inputs.toolchain_sha256
        || plan["module_sha256"] != opened.manifest().record.identity.target
    {
        return Err("campaign immutable bounds or preparation identity mismatch".into());
    }
    let module = opened.payload("target/module.wasm", 512 << 20)?;
    let choice_identity = serde_json::from_value(plan["choice_identity"].clone())
        .map_err(|error| error.to_string())?;
    Ok(prepare::Prepared {
        auxiliary: std::collections::BTreeMap::new(),
        compile_cache_root: None,
        module,
        configuration,
        toolchain: inputs.toolchain_sha256,
        choice_identity,
        scenario: serde_json::from_value(plan["scenario"].clone())
            .map_err(|error| error.to_string())?,
        network: serde_json::from_value(plan["network"].clone())
            .map_err(|error| error.to_string())?,
        provenance: opened.payload("target/provenance.json", 16 << 20)?,
        soft_assertion_report_only: plan["soft_assertion_report_only"]
            .as_bool()
            .ok_or("campaign assertion policy missing")?,
    })
}

pub(crate) fn resume(path: &Path, json: bool, output: &mut dyn Write) -> Result<i32, String> {
    resume_with_cache(path, json, output, None)
}

pub(crate) fn resume_with_cache(
    path: &Path,
    json: bool,
    output: &mut dyn Write,
    compile_cache_root: Option<String>,
) -> Result<i32, String> {
    let mut journal = Journal::open(path, true)?;
    if journal
        .events()
        .iter()
        .any(|event| matches!(event, Event::Complete { .. }))
    {
        return Err("campaign lifecycle published is not resumable".into());
    }
    let mut prepared = restore(&journal)?;
    prepared.compile_cache_root = compile_cache_root;
    let duration = Duration::from_nanos(
        journal.plan()["overall_nanos"]
            .as_u64()
            .filter(|nanos| *nanos > 0)
            .ok_or("campaign overall bound missing")?,
    );
    run(&mut journal, &prepared, duration, json, output)
}

#[derive(Default)]
struct Frontier {
    admitted: BTreeMap<u64, Job>,
    receipts: BTreeMap<u64, Execution>,
    committed: Vec<Execution>,
    next: Option<u64>,
    stop: Option<String>,
    complete: bool,
}

struct CancelGuard<'a>(&'a [gomad_wasmhost::linked::Cancellation]);
impl Drop for CancelGuard<'_> {
    fn drop(&mut self) {
        for cancellation in self.0 {
            cancellation.cancel();
        }
    }
}

fn frontier(journal: &Journal, selection: &seeds::SeedSelection) -> Result<Frontier, String> {
    let (index, count) = shard(journal.plan())?;
    let mut result = Frontier {
        next: (index < selection.count()).then_some(index),
        ..Frontier::default()
    };
    let admit = |result: &mut Frontier, jobs: &[Job]| {
        for job in jobs {
            result.admitted.insert(job.ordinal, job.clone());
            result.next = job
                .ordinal
                .checked_add(count)
                .filter(|ordinal| *ordinal < selection.count());
        }
    };
    for event in journal.events() {
        match event {
            Event::Admit { jobs } => admit(&mut result, jobs),
            Event::Receipt { execution } => {
                result.receipts.insert(execution.ordinal, execution.clone());
            }
            Event::Outcome {
                executions,
                admitted,
                stop_reason,
            } => {
                for execution in executions {
                    result.admitted.remove(&execution.ordinal);
                    result.receipts.remove(&execution.ordinal);
                    result.committed.push(execution.clone());
                }
                admit(&mut result, admitted);
                if stop_reason.is_some() {
                    result.stop.clone_from(stop_reason);
                }
            }
            Event::Complete { stop_reason } => {
                result.complete = true;
                result.stop = Some(stop_reason.clone());
            }
        }
    }
    Ok(result)
}

fn shard(plan: &Value) -> Result<(u64, u64), String> {
    if plan["shard"].is_null() {
        return Ok((0, 1));
    }
    let index = plan["shard"]["index"]
        .as_u64()
        .ok_or("campaign shard index missing")?;
    let count = plan["shard"]["count"]
        .as_u64()
        .ok_or("campaign shard count missing")?;
    if count == 0 || index >= count {
        return Err("invalid campaign shard".into());
    }
    Ok((index, count))
}

fn admit(
    frontier: &mut Frontier,
    selection: &seeds::SeedSelection,
    count: u64,
    limit: usize,
) -> Vec<Job> {
    let mut jobs = Vec::new();
    while jobs.len() < limit {
        let Some(ordinal) = frontier.next else { break };
        let Some(seed) = selection.seed_at(ordinal) else {
            break;
        };
        let job = Job { ordinal, seed };
        frontier.admitted.insert(ordinal, job.clone());
        jobs.push(job);
        frontier.next = ordinal
            .checked_add(count)
            .filter(|ordinal| *ordinal < selection.count());
    }
    jobs
}

fn run(
    journal: &mut Journal,
    prepared: &prepare::Prepared,
    duration: Duration,
    json: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    run_observed(journal, prepared, duration, json, output, None)
}

#[expect(
    clippy::too_many_lines,
    reason = "The scoped scheduler keeps durable admission, receipt, ordered policy and cancellation in one lifetime"
)]
fn run_observed(
    journal: &mut Journal,
    prepared: &prepare::Prepared,
    duration: Duration,
    json: bool,
    output: &mut dyn Write,
    mut observer: Option<Observer<'_>>,
) -> Result<i32, String> {
    let mut metrics = BTreeMap::new();
    let plan = journal.plan().clone();
    let selection = seeds::SeedSelection::parse(
        plan["selection"]
            .as_str()
            .ok_or("missing campaign selection")?,
    )?;
    let parallel = usize::try_from(plan["parallel"].as_u64().ok_or("missing campaign slots")?)
        .map_err(|_| "campaign slots overflow")?;
    let (_, shard_count) = shard(&plan)?;
    let mut state = frontier(journal, &selection)?;
    if state.complete {
        return report(
            journal,
            &state.committed,
            state.stop.as_deref().unwrap_or("seeds_exhausted"),
            json,
            output,
        );
    }
    if let Some(reason) = state.stop.clone() {
        journal.append(Event::Complete {
            stop_reason: reason.clone(),
        })?;
        return report(journal, &state.committed, &reason, json, output);
    }
    let mut retention = retention::Retention::default();
    let mut failures = BTreeSet::new();
    let mut retained_failures = BTreeMap::new();
    for execution in &state.committed {
        retention.retained(execution)?;
        if matches!(
            execution.record.outcome.class.as_str(),
            "target_failure" | "watchdog"
        ) {
            let signature = execution.record.failure_signature()?;
            failures.insert(signature.clone());
            if let Some(artifact) = &execution.artifact {
                retained_failures.insert(signature, (artifact.clone(), execution.stored_bytes));
            }
        }
    }
    let deadline = Instant::now()
        .checked_add(duration)
        .ok_or("campaign overall timeout overflows")?;
    let mut failure = None;
    let mut failed_assertions = None;
    let mut failed_completion = None;
    let mut pending_publications = BTreeMap::new();
    let cancellations = (0..parallel)
        .map(|_| gomad_wasmhost::linked::Cancellation::default())
        .collect::<Vec<_>>();
    let cancel = || {
        for cancellation in &cancellations {
            cancellation.cancel();
        }
    };
    thread::scope(|scope| -> Result<(), String> {
        let _cancel_guard = CancelGuard(&cancellations);
        let (completion_tx, completion_rx) = mpsc::channel();
        let mut slots = Vec::new();
        for (slot, cancellation) in cancellations.iter().enumerate() {
            let (tx, rx) = mpsc::channel::<Job>();
            let completed = completion_tx.clone();
            scope.spawn(move || {
                while let Ok(job) = rx.recv() {
                    let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                        #[cfg(feature = "determinism-fixture")]
                        if std::env::var("GOMAD_FIXTURE_CAMPAIGN_PANIC_SEED")
                            .ok()
                            .as_deref()
                            == Some(job.seed.to_string().as_str())
                        {
                            panic!("campaign fixture slot panic");
                        }
                        let remaining = deadline.saturating_duration_since(Instant::now());
                        if remaining.is_zero() {
                            return Err("overall_timeout: campaign deadline exceeded".into());
                        }
                        execution::run(prepared, &job, remaining, cancellation)
                    }))
                    .unwrap_or_else(|_| {
                        Err("runner_failure: host panic in execution slot; campaign aborted".into())
                    });
                    if completed.send((slot, job, result)).is_err() {
                        break;
                    }
                }
            });
            slots.push(tx);
        }
        drop(completion_tx);
        let mut free = (0..parallel).collect::<BTreeSet<_>>();
        let mut running = BTreeSet::new();
        let available = parallel.saturating_sub(state.admitted.len());
        let initial = admit(&mut state, &selection, shard_count, available);
        if !initial.is_empty() {
            journal.append(Event::Admit { jobs: initial })?;
        }
        for job in state.admitted.values() {
            if !state.receipts.contains_key(&job.ordinal) {
                let slot = free
                    .pop_first()
                    .ok_or("campaign slot accounting mismatch")?;
                slots[slot]
                    .send(job.clone())
                    .map_err(|error| error.to_string())?;
                running.insert(job.ordinal);
            }
        }
        loop {
            while failure.is_none() {
                let Some((&ordinal, _)) = state.admitted.first_key_value() else {
                    break;
                };
                let Some(receipt) = state.receipts.get(&ordinal).cloned() else {
                    break;
                };
                let stopping = failure_policy::stop(
                    plan["on_failure"]
                        .as_str()
                        .ok_or("missing failure policy")?,
                    plan["failure_budget"]
                        .as_u64()
                        .ok_or("missing failure budget")?,
                    &receipt,
                    &failures,
                )?;
                if stopping.is_some() {
                    if stopping.as_deref() == Some("first_failure") {
                        cancel();
                    }
                    state.stop = stopping;
                    break;
                }
                let mut receipt = receipt;
                decide(
                    journal,
                    &plan,
                    &mut receipt,
                    &mut pending_publications,
                    &mut retention,
                    &mut retained_failures,
                )?;
                let admitted = admit(&mut state, &selection, shard_count, 1);
                journal.append(Event::Outcome {
                    executions: vec![receipt.clone()],
                    admitted: admitted.clone(),
                    stop_reason: None,
                })?;
                retention.retained(&receipt)?;
                if json && let Some(report) = &receipt.soft_assertions {
                    soft_assertions::progress(output, journal.path(), receipt.seed, report)?;
                }
                if matches!(
                    receipt.record.outcome.class.as_str(),
                    "target_failure" | "watchdog"
                ) {
                    failures.insert(receipt.record.failure_signature()?);
                }
                state.receipts.remove(&ordinal);
                state.admitted.remove(&ordinal);
                if let (Some(observe), Some(measured)) =
                    (observer.as_deref_mut(), metrics.remove(&ordinal))
                {
                    observe(journal.path(), &receipt, &measured);
                }
                state.committed.push(receipt);
                for job in admitted {
                    let slot = free
                        .pop_first()
                        .ok_or("campaign refill slot accounting mismatch")?;
                    slots[slot]
                        .send(job.clone())
                        .map_err(|error| error.to_string())?;
                    running.insert(job.ordinal);
                }
            }
            if failure.is_some() || state.stop.is_some() {
                if running.is_empty() {
                    break;
                }
            } else if state.admitted.is_empty() {
                break;
            }
            let (slot, job, result) = if failure.is_some() || state.stop.is_some() {
                completion_rx
                    .recv()
                    .map_err(|_| "runner_failure: execution slots disconnected")?
            } else {
                match completion_rx.recv_timeout(deadline.saturating_duration_since(Instant::now()))
                {
                    Ok(result) => result,
                    Err(mpsc::RecvTimeoutError::Timeout) => {
                        failure = Some("overall_timeout: campaign deadline exceeded".into());
                        cancel();
                        continue;
                    }
                    Err(mpsc::RecvTimeoutError::Disconnected) => {
                        return Err("runner_failure: execution slots disconnected".into());
                    }
                }
            };
            free.insert(slot);
            running.remove(&job.ordinal);
            match result {
                Ok(execution::Completion::Failed(failed_result)) => {
                    if failure.is_none() {
                        failure = Some(failed_result.message());
                        failed_completion = Some(failed_result);
                    }
                    cancel();
                }
                Ok(execution::Completion::Finalized(result)) => {
                    if failure.is_some() {
                        continue;
                    }
                    metrics.insert(job.ordinal, result.metrics);
                    let mut execution = result.execution;
                    if execution.record.outcome.class == "runner_failure"
                        && (execution.record.outcome.termination != "cancelled"
                            || failure.is_some())
                    {
                        if execution.record.outcome.termination == "capacity" {
                            failed_assertions.clone_from(&execution.soft_assertions);
                        }
                        failure.get_or_insert_with(|| backend_failure(&execution.record.outcome));
                        cancel();
                        continue;
                    }
                    if execution.record.outcome.class != "runner_failure"
                        && (execution.record.outcome.class != "success"
                            || plan["keep_successes"] != "none")
                    {
                        let published = store(
                            journal.path().join("evidence"),
                            artifact_capacity(&plan, prepared)?,
                        )
                        .publish(result.publication)?;
                        execution.artifact = Some(
                            published
                                .path
                                .strip_prefix(journal.path())
                                .map_err(|error| error.to_string())?
                                .to_string_lossy()
                                .into_owned(),
                        );
                        execution.stored_bytes = published.stored_bytes;
                    }
                    journal.append(Event::Receipt {
                        execution: execution.clone(),
                    })?;
                    state.receipts.insert(job.ordinal, execution);
                }
                Err(error) => {
                    failure.get_or_insert(error);
                    cancel();
                }
            }
        }
        if failure.is_none()
            && let Some(reason) = state.stop.clone()
        {
            let mut group = Vec::new();
            for receipt in state.receipts.values() {
                let mut receipt = receipt.clone();
                decide(
                    journal,
                    &plan,
                    &mut receipt,
                    &mut pending_publications,
                    &mut retention,
                    &mut retained_failures,
                )?;
                retention.retained(&receipt)?;
                if matches!(
                    receipt.record.outcome.class.as_str(),
                    "target_failure" | "watchdog"
                ) {
                    failures.insert(receipt.record.failure_signature()?);
                }
                group.push(receipt);
            }
            journal.append(Event::Outcome {
                executions: group.clone(),
                admitted: Vec::new(),
                stop_reason: Some(reason),
            })?;
            for receipt in &group {
                if let (Some(observe), Some(measured)) =
                    (observer.as_deref_mut(), metrics.remove(&receipt.ordinal))
                {
                    observe(journal.path(), receipt, &measured);
                }
            }
            state.committed.extend(group);
            state.receipts.clear();
            state.admitted.clear();
        }
        drop(slots);
        Ok(())
    })?;
    if let Some(observed) = failed_completion {
        return observed.report(json, output);
    }
    if let Some(error) = failure {
        if let Some(report) = failed_assertions {
            if json {
                writeln!(
                    output,
                    "{}",
                    json!({"schema":"gomad.explore-event/v3", "type":"error",
                    "classification":"capacity", "message":error,"soft_assertions":report})
                )
                .map_err(|error| error.to_string())?;
            } else {
                writeln!(output, "gomad: capacity: {error}").map_err(|error| error.to_string())?;
            }
            return Ok(3);
        }
        return Err(error);
    }
    let reason = state.stop.unwrap_or_else(|| "seeds_exhausted".into());
    journal.append(Event::Complete {
        stop_reason: reason.clone(),
    })?;
    report(journal, &state.committed, &reason, json, output)
}

fn decide(
    journal: &Journal,
    plan: &Value,
    execution: &mut Execution,
    publications: &mut BTreeMap<u64, Publication>,
    retention: &mut retention::Retention,
    failures: &mut BTreeMap<String, (String, u64)>,
) -> Result<(), String> {
    let failure = matches!(
        execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    );
    if failure {
        let signature = execution.record.failure_signature()?;
        if let Some((reference, bytes)) = failures.get(&signature) {
            execution.artifact = Some(reference.clone());
            execution.stored_bytes = *bytes;
            publications.remove(&execution.ordinal);
            return Ok(());
        }
    } else if execution.record.outcome.class != "success"
        || !retention.keep(
            plan["keep_successes"]
                .as_str()
                .ok_or("missing success retention policy")?,
            execution,
        )
    {
        publications.remove(&execution.ordinal);
        execution.artifact = None;
        execution.stored_bytes = 0;
        return Ok(());
    }
    let maximum = if failure {
        u64::MAX
    } else {
        retention.remaining(
            plan["success_limit"]
                .as_u64()
                .ok_or("missing success artifact bound")?,
            plan["success_bytes"]
                .as_u64()
                .ok_or("missing success bytes bound")?,
        )?
    };
    if let Some(reference) = &execution.artifact {
        if execution.stored_bytes > maximum {
            return Err("capacity: successful-execution retention capacity is exhausted".into());
        }
        if failure {
            failures.insert(
                execution.record.failure_signature()?,
                (reference.clone(), execution.stored_bytes),
            );
        }
        return Ok(());
    }
    let publication = publications
        .remove(&execution.ordinal)
        .ok_or("runner_failure: received execution lost its retained payloads")?;
    let published = store(journal.path().join("evidence"), u64::MAX).publish(publication)?;
    if published.stored_bytes > maximum {
        return Err("capacity: successful-execution retention capacity is exhausted".into());
    }
    let reference = published
        .path
        .strip_prefix(journal.path())
        .map_err(|error| error.to_string())?
        .to_string_lossy()
        .into_owned();
    execution.artifact = Some(reference.clone());
    execution.stored_bytes = published.stored_bytes;
    if failure {
        failures.insert(
            execution.record.failure_signature()?,
            (reference, published.stored_bytes),
        );
    }
    Ok(())
}

fn artifact_capacity(plan: &Value, prepared: &prepare::Prepared) -> Result<u64, String> {
    let per = (prepared.module.len() as u64)
        .checked_mul(2)
        .and_then(|value| value.checked_add(prepared.configuration.limits.evidence_bytes))
        .and_then(|value| {
            value.checked_add(prepared.configuration.limits.output_bytes.checked_mul(2)?)
        })
        .and_then(|value| value.checked_add(32 << 20))
        .ok_or("capacity: per-artifact capacity overflows")?;
    per.checked_mul(
        plan["selection_count"]
            .as_u64()
            .ok_or("selection count missing")?,
    )
    .and_then(|value| value.checked_add(plan["success_bytes"].as_u64()?))
    .ok_or_else(|| "capacity: aggregate artifact capacity overflows".into())
}

fn report(
    journal: &Journal,
    executions: &[Execution],
    reason: &str,
    json_output: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    let mut distinct = BTreeSet::new();
    let mut succeeded = 0_u64;
    let mut failures = 0_u64;
    let mut watchdogs = 0_u64;
    let mut cancelled = 0_u64;
    let mut successes = Vec::new();
    let mut artifacts = BTreeSet::new();
    let mut retained_bytes = 0_u64;
    let mut advertised = Vec::new();
    for execution in executions {
        match execution.record.outcome.class.as_str() {
            "success" => {
                succeeded += 1;
                if let Some(reference) = &execution.artifact {
                    successes.push(json!({"seed":execution.seed,"path":journal.path().join(reference),"record_hash":execution.record.hash()?}));
                }
            }
            "target_failure" | "watchdog" => {
                failures += 1;
                watchdogs += u64::from(execution.record.outcome.class == "watchdog");
                distinct.insert(execution.record.failure_signature()?);
            }
            _ => cancelled += 1,
        }
        if let Some(reference) = &execution.artifact
            && artifacts.insert(reference)
        {
            let opened = journal.open_artifact(reference)?;
            opened.authenticate_all()?;
            if opened.manifest().record != execution.record {
                return Err("advertised artifact differs from committed record".into());
            }
            let path = std::path::absolute(journal.path().join(reference))
                .map_err(|error| error.to_string())?;
            let text = path
                .to_str()
                .ok_or("advertised artifact path is not UTF-8")?;
            let quoted = format!("'{}'", text.replace('\'', "'\"'\"'"));
            advertised.push(json!({"path":path,"record_hash":execution.record.hash()?,
                "replay_argv":["gomad","replay",path],"replay_command":format!("gomad replay {quoted}")}));
            retained_bytes = retained_bytes
                .checked_add(execution.stored_bytes)
                .ok_or("retained bytes overflow")?;
        }
    }
    let (index, count) = shard(journal.plan())?;
    let total = journal.plan()["selection_count"]
        .as_u64()
        .ok_or("selection count missing")?;
    let selection_count = if total > index {
        1 + (total - 1 - index) / count
    } else {
        0
    };
    let mut result = json!({"type":"result","classification":if failures==0{"success"}else if watchdogs==failures{"watchdog_observation"}else if watchdogs>0{"mixed_failure"}else{"target_failure"},"campaign_path":journal.path(),"selection_count":selection_count,"attempted":executions.len(),"succeeded":succeeded,"failures":failures,"watchdogs":watchdogs,"cancelled":cancelled,"distinct_failures":distinct.len(),"stop_reason":reason,"success_artifacts":successes,"artifacts":advertised,"retained_evidence":artifacts.len(),"retained_bytes":retained_bytes});
    if journal.plan()["schema"] == "gomad.merged-seed-campaign-plan/v2" {
        result["partial"] = journal.plan()["merge_missing"]
            .as_array()
            .is_some_and(|missing| !missing.is_empty())
            .into();
        result["missing"] = journal.plan()["merge_missing"].clone();
        result["shards"] = journal.plan()["merge_sources"]
            .as_array()
            .map_or(0, Vec::len)
            .into();
        result["plan_sha256"] = journal.plan()["plan_sha256"].clone();
    }
    if json_output {
        writeln!(
            output,
            "{}",
            String::from_utf8(canonical::encode(&result)?).map_err(|error| error.to_string())?
        )
        .map_err(|error| error.to_string())?;
    } else {
        writeln!(output,"gomad explore: campaign={} selected={selection_count} attempted={} succeeded={succeeded} failures={failures} watchdogs={watchdogs} cancelled={cancelled} distinct={} stop={reason}",journal.path().display(),executions.len(),distinct.len()).map_err(|error|error.to_string())?;
    }
    Ok(i32::from(failures != 0))
}

#[cfg(test)]
mod installed_report_tests {
    use super::*;

    #[test]
    fn installed_lifecycle_report_advertises_only_authenticated_retained_artifacts() {
        let fixture = crate::installed_lifecycle_tests::retained(false);
        let plan = json!({"schema":"gomad.seed-campaign-plan/v2","selection":"7", "selection_count":1,
            "parallel":1,"on_failure":"all","failure_budget":1,"journal_bytes":1_048_576});
        let journal = Journal::create(&fixture.root.join("campaign"), &plan, 1 << 20).unwrap();
        let store = Store {
            root: journal.path().join("evidence"),
            maximum_bytes: 64 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        };
        let opened = Opened::open(&fixture.published.path).unwrap();
        let mut payloads = BTreeMap::new();
        for (name, entry) in &opened.manifest().files {
            payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
        }
        let published = store
            .publish(Publication {
                record: fixture.record.clone(),
                payloads,
            })
            .unwrap();
        let reference = published
            .path
            .strip_prefix(journal.path())
            .unwrap()
            .to_str()
            .unwrap()
            .to_owned();
        let execution = Execution {
            ordinal: 0,
            seed: 7,
            record: fixture.record.clone(),
            artifact: Some(reference),
            stored_bytes: published.stored_bytes,
            semantic_probes: vec![],
            choice_features: vec![],
            divergence: None,
            scenario_steps: None,
            soft_assertions: None,
        };
        let mut output = Vec::new();
        report(
            &journal,
            std::slice::from_ref(&execution),
            "complete",
            true,
            &mut output,
        )
        .unwrap();
        let value: Value = serde_json::from_slice(&output).unwrap();
        let artifacts = value["artifacts"]
            .as_array()
            .expect("retained replay invocation was not advertised");
        assert_eq!(artifacts.len(), 1);
        assert_eq!(artifacts[0]["path"], json!(published.path));
        assert_eq!(
            artifacts[0]["record_hash"],
            execution.record.hash().unwrap()
        );
        assert_eq!(
            artifacts[0]["replay_argv"],
            json!(["gomad", "replay", published.path])
        );
        assert_ne!(artifacts[0]["replay_command"].as_str().unwrap(), "");
        std::fs::write(
            published.path.join("output/stdout"),
            b"altered retained bytes",
        )
        .unwrap();
        assert!(report(&journal, &[execution], "complete", true, &mut Vec::new()).is_err());
    }
}

fn backend_failure(outcome: &crate::evidence::record::Outcome) -> String {
    let reason = outcome.reason.chars().take(4096).collect::<String>();
    format!(
        "runner_failure: backend {}: {}{}",
        outcome.termination,
        reason,
        if outcome.reason.chars().count() > 4096 {
            "…"
        } else {
            ""
        }
    )
}
#[cfg(test)]
mod sweep_backend_failure_tests {
    #[test]
    fn native_failure_retains_typed_termination_without_reason_inference() {
        for termination in ["unsupported", "capacity", "infrastructure"] {
            let outcome = crate::evidence::record::Outcome {
                class: "runner_failure".into(),
                termination: termination.into(),
                exit_code: None,
                reason: "opaque diagnostic".into(),
            };
            assert_eq!(
                super::backend_failure(&outcome),
                format!("runner_failure: backend {termination}: opaque diagnostic")
            );
        }
    }
}
