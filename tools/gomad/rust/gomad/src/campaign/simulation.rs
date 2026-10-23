use super::{
    exploration::{
        plan,
        round::{RoundEvent, RoundJournal, RoundState, RoundStrategy},
        simulation::{self as controller, SimulationConfig},
        simulation_record, simulation_record_wire,
    },
    ledger::Job,
    prepare,
    retention::Retention,
};
use crate::{
    cli::exploration::ExplorationOptions,
    evidence::{canonical, inspection::Details, record::Stream},
};
use gomad_wasmhost::linked;
use serde_json::json;
use sha2::Digest as _;
use std::{
    io::Write,
    path::Path,
    time::{Duration, Instant},
};

pub(crate) fn explore(options: &ExplorationOptions, output: &mut dyn Write) -> Result<i32, String> {
    let started = Instant::now();
    let selection = super::seeds::SeedSelection::parse(&options.base.seeds)?;
    if selection.count() != 1 || !options.base.choices {
        return Err(
            "simulation exploration requires exactly one base seed and enabled choice recording"
                .into(),
        );
    }
    let prepared = prepare::prepare(&options.base)?;
    let config = SimulationConfig {
        execution_sha256: controller::domain_hash(
            "gomad-simulation-exploration-execution/v1",
            &canonical::encode(
                &json!({"module_sha256":canonical::digest(&prepared.module),"toolchain":prepared.toolchain,"configuration":prepared.configuration,"scenario":prepared.scenario,"network":prepared.network,"choice_identity":prepared.choice_identity,"soft_assertion_report_only":prepared.soft_assertion_report_only,"host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"helper_sha256":linked::IMPLEMENTATION_SHA256}),
            )?,
        ),
        controller_sha256: controller::implementation_sha256(),
        base_seed: selection.seed_at(0).ok_or("missing simulation base seed")?,
        parallel: options.base.parallel,
        max_executions: options.max_executions,
        max_forced_decisions: options.max_forced_decisions,
        max_exploration_bytes: options.max_exploration_bytes,
        max_result_bytes: options.max_exploration_result_bytes,
        failure_budget: if options.base.on_failure == "all" {
            options.max_executions
        } else {
            options.base.failure_budget
        },
        dimension_limits: options.dimension_limits.clone(),
    };
    controller::SimulationState::new(config.clone())?;
    let mut journal = plan::create(options, &prepared, RoundStrategy::Simulation(config))?;
    let remaining = options
        .base
        .overall_timeout
        .checked_sub(started.elapsed())
        .ok_or("overall_timeout: campaign deadline exceeded during preparation")?;
    run(
        &mut journal,
        &prepared,
        remaining,
        options.base.json,
        output,
    )
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
    let mut journal = RoundJournal::open(path, true)?;
    if !matches!(journal.state(), RoundState::Simulation(_)) {
        return Err("campaign is not simulation exploration".into());
    }
    let mut prepared = plan::restore(&journal)?;
    prepared.compile_cache_root = compile_cache_root;
    let timeout = plan::duration(&journal)?;
    run(&mut journal, &prepared, timeout, json, output)
}

#[expect(
    clippy::too_many_lines,
    reason = "Raw model observations and retained executions form one complete round transaction"
)]
fn run(
    journal: &mut RoundJournal,
    prepared: &prepare::Prepared,
    timeout: Duration,
    json: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    let started = Instant::now();
    let identity = prepared
        .choice_identity
        .as_ref()
        .ok_or("simulation exploration requires a cooperative choice identity")?;
    let mut retention = Retention::default();
    for event in journal.events() {
        for execution in event.executions() {
            retention.retained(execution)?;
        }
    }
    loop {
        let RoundState::Simulation(state) = journal.state().clone() else {
            return Err("campaign is not simulation exploration".into());
        };
        let Some(round) = state.next_round() else {
            break;
        };
        let remaining = timeout
            .checked_sub(started.elapsed())
            .ok_or("overall_timeout: simulation exploration deadline exceeded")?;
        let controls = round
            .candidates
            .iter()
            .map(|candidate| {
                simulation_record_wire::execution_for_candidate(&state.config, candidate, identity)
            })
            .collect::<Result<Vec<_>, _>>()?;
        let cancellations = (0..round.candidates.len())
            .map(|_| linked::Cancellation::default())
            .collect::<Vec<_>>();
        let indices = (0..controls.len() as u64).collect::<Vec<_>>();
        let observed = crate::slots::run(&indices, state.config.parallel, |index| {
            let index = usize::try_from(index).map_err(|_| "simulation slot index overflow")?;
            let control = &controls[index];
            let result = (|| {
                let ordinal = state
                    .logical_executions
                    .checked_add(index as u64)
                    .ok_or("simulation ordinal overflow")?;
                let replay = replay_request(control, identity)?;
                super::simulation_execute::run(
                    prepared,
                    &Job {
                        ordinal,
                        seed: state.config.base_seed,
                    },
                    remaining,
                    &cancellations[index],
                    replay.as_ref(),
                    replay
                        .as_ref()
                        .map(|_| control.choice_source_trace.as_slice()),
                    &control.simulation_plan,
                )
            })();
            if result.is_err()
                || result.as_ref().is_ok_and(|completed| {
                    matches!(completed, super::execution::Completion::Failed(_))
                })
            {
                for cancellation in &cancellations {
                    cancellation.cancel();
                }
            }
            Ok(result)
        })?;
        if let Some(failure) = super::execution::batch_failure(&observed) {
            return failure.report(json, output);
        }
        let observed = observed.into_iter().collect::<Result<Vec<_>, _>>()?;
        let mut executions = Vec::with_capacity(observed.len());
        let mut results = Vec::with_capacity(observed.len());
        let mut evidence = Vec::with_capacity(observed.len());
        for (index, observed) in observed.into_iter().enumerate() {
            let mut observed = match observed {
                super::execution::Completion::Finalized(observed) => observed,
                super::execution::Completion::Failed(failure) => {
                    return failure.report(json, output);
                }
            };
            if observed.execution.record.outcome.class == "runner_failure" {
                return Err(format!(
                    "runner_failure: {}",
                    observed.execution.record.outcome.reason
                ));
            }
            let trace = observed
                .publication
                .payloads
                .get("evidence/choice")
                .ok_or("simulation execution omitted choice trace")?;
            let mut runtime = simulation_record::runtime_decisions(trace, identity)?;
            for decision in &mut runtime {
                decision.source_trace.clone_from(trace);
            }
            let record = observed
                .publication
                .payloads
                .get("simulation/record.json")
                .ok_or("simulation execution omitted its combined decision record")?;
            let candidate = &round.candidates[index];
            let mut result =
                simulation_record::result_for_record(&state.config, candidate, record, &runtime)?;
            if observed.execution.record.outcome.class == "success" && result.failed {
                return Err("simulation failure completed with a successful target outcome".into());
            }
            let profile = simulation_record::project_artifact(
                &state.config,
                candidate,
                &controls[index].simulation_plan,
                record,
                &runtime,
                simulation_record::MAXIMUM_RECORD_BYTES as u64,
            )?;
            let details_file = "evidence/inspection.json";
            let mut details = observed
                .publication
                .payloads
                .get(details_file)
                .map(|bytes| serde_json::from_slice::<Details>(bytes))
                .transpose()
                .map_err(|e| e.to_string())?
                .unwrap_or(Details {
                    schema: "gomad.inspection-evidence/v1".into(),
                    outputs: None,
                    choices: None,
                    world: None,
                    simulation: None,
                    minimization: None,
                });
            details.simulation = Some(profile);
            let retained_record = record.clone();
            evidence.push(super::exploration::round::SimulationEvidence {
                record: record.clone(),
                runtime_trace: trace.clone(),
            });
            for (file, bytes) in [
                (
                    "simulation/plan.json",
                    controls[index].simulation_plan.clone(),
                ),
                ("simulation/record.json", retained_record),
                (details_file, canonical::encode(&details)?),
            ] {
                observed.publication.record.streams.insert(
                    file.into(),
                    Stream {
                        bytes: bytes.len() as u64,
                        sha256: canonical::digest(&bytes),
                        records: 1,
                    },
                );
                observed.publication.payloads.insert(file.into(), bytes);
            }
            observed.execution.record = observed.publication.record.clone();
            observed
                .publication
                .payloads
                .insert("record.json".into(), observed.publication.record.bytes()?);
            let execution = &mut observed.execution;
            if matches!(
                execution.record.outcome.class.as_str(),
                "target_failure" | "watchdog"
            ) && execution.divergence.is_none()
                && !result.diverged
            {
                result.failed = true;
                result.failure_sha256 = format!("sha256:{}", execution.record.failure_signature()?);
            }
            result.diverged |= execution.divergence.is_some();
            let plan = &journal.plan().prepared;
            let failure = execution.record.outcome.class != "success" || result.diverged;
            if failure
                || retention.keep(
                    plan["keep_successes"]
                        .as_str()
                        .ok_or("missing success retention policy")?,
                    execution,
                )
            {
                let maximum = if failure {
                    u64::MAX
                } else {
                    retention.remaining(
                        plan["success_limit"]
                            .as_u64()
                            .ok_or("missing success count bound")?,
                        plan["success_bytes"]
                            .as_u64()
                            .ok_or("missing success byte bound")?,
                    )?
                };
                let published = super::store(journal.path().join("evidence"), maximum)
                    .publish(observed.publication)?;
                execution.artifact = Some(
                    published
                        .path
                        .strip_prefix(journal.path())
                        .map_err(|e| e.to_string())?
                        .to_string_lossy()
                        .into_owned(),
                );
                execution.stored_bytes = published.stored_bytes;
                super::execution::retain_backend_metrics(
                    journal.path(),
                    execution,
                    &observed.metrics,
                )?;
            }
            retention.retained(execution)?;
            executions.push(observed.execution);
            results.push(result);
        }
        let (_, segment) = state.commit(&round, &results)?;
        journal.append(RoundEvent::Simulation {
            executions,
            segment,
            evidence,
        })?;
    }
    let summary = journal.summary()?;
    if json {
        serde_json::to_writer(&mut *output, &summary).map_err(|e| e.to_string())?;
        writeln!(output).map_err(|e| e.to_string())?;
    } else {
        writeln!(
            output,
            "{}",
            serde_json::to_string_pretty(&summary).map_err(|e| e.to_string())?
        )
        .map_err(|e| e.to_string())?;
    }
    Ok(i32::from(
        journal
            .events()
            .iter()
            .flat_map(super::exploration::round::RoundEvent::executions)
            .any(|e| e.record.outcome.class != "success"),
    ))
}

fn replay_request(
    control: &simulation_record_wire::CandidateExecution,
    identity: &linked::ChoiceIdentity,
) -> Result<Option<linked::ReplayRequest>, String> {
    let Some(tape) = &control.choice_replay_plan else {
        return Ok(None);
    };
    let source =
        linked::project_choices(&control.choice_source_trace, identity).map_err(str::to_owned)?;
    let last = tape
        .decisions
        .last()
        .ok_or("runtime forced prefix is empty")?;
    let original = source
        .decisions
        .get(usize::try_from(last.ordinal()).map_err(|_| "runtime ordinal overflow")?)
        .ok_or("runtime forced prefix ordinal is absent")?;
    Ok(Some(linked::ReplayRequest {
        schema: "gomad.replay-trace/v1".into(),
        execution_identity: identity.clone(),
        trace_sha256: canonical::digest(&control.choice_source_trace),
        tape_sha256: format!("{:x}", sha2::Sha256::digest(&tape.bytes)),
        decisions: tape.decisions.len() as u64,
        trace_bytes: control.choice_source_trace.len() as u64,
        trace_frames: (control.choice_source_trace.len() as u64).div_ceil(65536),
        prefix_control: Some(linked::PrefixControl {
            cutoff: tape.decisions.len() as u64,
            r#override: Some(linked::RankOverride {
                ordinal: last.ordinal(),
                rank: last.selected(),
                reconstruct: original.selected() == last.selected(),
            }),
        }),
    }))
}
