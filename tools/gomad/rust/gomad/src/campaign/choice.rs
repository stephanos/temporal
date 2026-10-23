use super::{
    execution,
    exploration::{
        self,
        choice::{ChoiceConfig, ChoiceResult},
        plan,
        round::{RoundEvent, RoundJournal, RoundState, RoundStrategy},
    },
    ledger::{Execution, Job},
    prepare::Prepared,
    retention::Retention,
};
use crate::{
    cli::exploration::ExplorationOptions,
    evidence::{canonical, store::Publication},
};
use gomad_wasmhost::linked;
use serde_json::json;
use std::{
    collections::BTreeSet,
    io::Write,
    path::Path,
    time::{Duration, Instant},
};

pub(crate) fn explore(options: &ExplorationOptions, output: &mut dyn Write) -> Result<i32, String> {
    let started = Instant::now();
    let prepared = super::prepare::prepare(&options.base)?;
    let (mut journal, initial) = prepare_journal(options, &prepared, started)?;
    let remaining = options
        .base
        .overall_timeout
        .checked_sub(started.elapsed())
        .ok_or("overall_timeout: choice exploration deadline exceeded during preparation")?;
    run(
        &mut journal,
        &prepared,
        remaining,
        options.base.json,
        output,
        initial,
        false,
    )
}

pub(super) fn prepare_journal(
    options: &ExplorationOptions,
    prepared: &Prepared,
    started: Instant,
) -> Result<(RoundJournal, Option<execution::Result>), String> {
    if prepared.scenario.is_some() {
        return Err(
            "unsupported_target: cluster-global forced choice exploration is unavailable".into(),
        );
    }
    let execution = prepared
        .choice_identity
        .clone()
        .ok_or("choice exploration requires an enabled cooperative choice trace")?;
    let selection = super::seeds::SeedSelection::parse(&options.base.seeds)?;
    let mut config = ChoiceConfig {
        execution,
        controller_sha256: exploration::choice::implementation(),
        base_seed: selection
            .seed_at(0)
            .ok_or("choice exploration base seed unavailable")?,
        parallel: options.base.parallel,
        max_executions: options.max_executions,
        max_choice_depth: options.max_choice_depth,
        start_ordinal: options.choice_start_ordinal,
        max_exploration_bytes: options.max_exploration_bytes,
        failure_policy: options.base.on_failure.clone(),
        failure_budget: options.base.failure_budget,
        site_scope: None,
    };
    exploration::choice::ChoiceState::new(config.clone())?;
    let mut initial = None;
    if let Some(patterns) = &options.site_scope {
        exploration::site_scope::require_symbols(&prepared.module)?;
        let remaining = options
            .base
            .overall_timeout
            .checked_sub(started.elapsed())
            .ok_or("overall_timeout: choice exploration deadline exceeded during preparation")?;
        let completion = execution::run(
            prepared,
            &Job {
                ordinal: 0,
                seed: config.base_seed,
            },
            remaining,
            &linked::Cancellation::default(),
        )?
        .finalized()
        .map_err(|failure| failure.message())?;
        let mut resolution = exploration::site_scope::Resolution::default();
        if let Some(raw) = completion.publication.payloads.get("evidence/choice") {
            resolution.observe(&prepared.module, raw, &config.execution, patterns)?;
        }
        config.site_scope = Some(resolution.scope(patterns));
        initial = Some(completion);
    }
    let journal = plan::create(options, prepared, RoundStrategy::Choice(config))?;
    Ok((journal, initial))
}

pub(super) fn commit_planned_base(
    journal: &mut RoundJournal,
    prepared: &Prepared,
    remaining: Duration,
    initial: execution::Result,
) -> Result<(), String> {
    run(
        journal,
        prepared,
        remaining,
        false,
        &mut std::io::sink(),
        Some(initial),
        true,
    )?;
    Ok(())
}

pub(crate) fn resume(path: &Path, json: bool, output: &mut dyn Write) -> Result<i32, String> {
    let mut journal = RoundJournal::open(path, true)?;
    let summary = journal.summary()?;
    resume_admission(
        summary["stop_reason"]
            .as_str()
            .ok_or("campaign completion state missing")?,
    )?;
    let prepared = plan::restore(&journal)?;
    let remaining = plan::duration(&journal)?;
    run(
        &mut journal,
        &prepared,
        remaining,
        json,
        output,
        None,
        false,
    )
}

#[expect(
    clippy::too_many_lines,
    reason = "The complete round is retained and committed only after every execution finishes"
)]
fn run(
    journal: &mut RoundJournal,
    prepared: &Prepared,
    duration: Duration,
    json_output: bool,
    output: &mut dyn Write,
    mut initial: Option<execution::Result>,
    base_only: bool,
) -> Result<i32, String> {
    let deadline = Instant::now()
        .checked_add(duration)
        .ok_or("choice exploration timeout overflow")?;
    let mut retention = Retention::default();
    for execution in journal.events().iter().flat_map(RoundEvent::executions) {
        retention.retained(execution)?;
    }
    loop {
        let RoundState::Choice(state) = journal.state() else {
            return Err("not a choice exploration campaign".into());
        };
        let Some(round) = state.next_round() else {
            break;
        };
        let state = state.clone();
        let indices = (0..round.candidates.len() as u64).collect::<Vec<_>>();
        let cancellation = linked::Cancellation::default();
        let completions = if let Some(completion) = initial.take() {
            if state.logical_executions != 0 || round.candidates.len() != 1 {
                return Err("scoped base execution differs from its root round".into());
            }
            vec![execution::Completion::Finalized(completion)]
        } else {
            crate::slots::run(&indices, state.config.parallel, |index| {
                let result = (|| {
                    let candidate = &round.candidates
                        [usize::try_from(index).map_err(|_| "choice candidate index overflow")?];
                    let replay = candidate.replay(&state.config.execution)?;
                    let remaining = deadline
                        .checked_duration_since(Instant::now())
                        .ok_or("overall_timeout: choice exploration deadline exceeded")?;
                    let job = Job {
                        ordinal: state.logical_executions + index,
                        seed: state.config.base_seed,
                    };
                    execution::run_with_replay(
                        prepared,
                        &job,
                        remaining,
                        &cancellation,
                        replay.as_ref(),
                        replay.as_ref().map(|_| candidate.source_trace.as_slice()),
                    )
                })();
                if result.is_err()
                    || result.as_ref().is_ok_and(|completed| {
                        matches!(completed, execution::Completion::Failed(_))
                    })
                {
                    cancellation.cancel();
                }
                result
            })?
        };
        let mut finalized = Vec::with_capacity(completions.len());
        for completion in completions {
            match completion {
                execution::Completion::Finalized(result) => finalized.push(result),
                execution::Completion::Failed(failure) => return failure.report(json, output),
            }
        }
        let completions = finalized;
        let context = journal
            .search_context(completions.first().and_then(|completion| {
                completion
                    .publication
                    .payloads
                    .get("evidence/choice")
                    .map(Vec::as_slice)
            }))?
            .map(|context| context.bytes())
            .transpose()?;
        let mut results = Vec::new();
        let mut executions = Vec::new();
        for (candidate, mut completion) in round.candidates.iter().zip(completions) {
            let execution = &completion.execution;
            if execution.record.outcome.class == "runner_failure" {
                return Err(format!(
                    "runner_failure: {}",
                    execution.record.outcome.reason
                ));
            }
            let divergence = execution.divergence.clone();
            let result = if divergence.is_some() {
                ChoiceResult {
                    candidate_sha256: candidate.sha256.clone(),
                    outcome_sha256: String::new(),
                    failed: false,
                    failure_sha256: String::new(),
                    divergence,
                    source_trace: None,
                }
            } else {
                let failed = matches!(
                    execution.record.outcome.class.as_str(),
                    "target_failure" | "watchdog"
                );
                ChoiceResult {
                    candidate_sha256: candidate.sha256.clone(),
                    outcome_sha256: exploration::hash(
                        "gomad-choice-exploration-outcome/v2",
                        &execution.record,
                    )?,
                    failed,
                    failure_sha256: if failed {
                        format!("sha256:{}", execution.record.failure_signature()?)
                    } else {
                        String::new()
                    },
                    divergence: None,
                    source_trace: completion
                        .publication
                        .payloads
                        .get("evidence/choice")
                        .cloned(),
                }
            };
            if let Some(context) = &context {
                completion.publication.payloads.insert(
                    crate::evidence::search_context::FILE.into(),
                    context.clone(),
                );
            }
            retain(
                journal,
                &mut completion.execution,
                completion.publication,
                &mut retention,
                base_only,
            )?;
            executions.push(completion.execution);
            results.push(result);
        }
        let (_, segment) = state.commit(&round, &results)?;
        let reports = executions
            .iter()
            .filter_map(|execution| {
                execution
                    .soft_assertions
                    .as_ref()
                    .map(|report| (execution.seed, report.clone()))
            })
            .collect::<Vec<_>>();
        journal.append(RoundEvent::Choice {
            executions,
            segment,
        })?;
        if base_only {
            break;
        }
        if json_output {
            for (seed, report) in reports {
                super::soft_assertions::progress(output, journal.path(), seed, &report)?;
            }
            writeln!(output, "{}", json!({"schema":"gomad.explore-event/v3","type":"round",
                "campaign":journal.path(),"strategy":"choice-exploration","envelope":journal.summary()?}))
                .map_err(|error| error.to_string())?;
        }
    }
    report(journal, json_output, output)
}

fn retain(
    journal: &RoundJournal,
    execution: &mut Execution,
    publication: Publication,
    retention: &mut Retention,
    base_only: bool,
) -> Result<(), String> {
    let metadata = &journal.plan().prepared;
    let failed = matches!(
        execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    ) || execution.divergence.is_some();
    let scoped_base = execution.ordinal == 0
        && matches!(&journal.plan().strategy, RoundStrategy::Choice(config) if config.site_scope.as_ref().is_some_and(|scope| base_only || scope.resolved_count == 0));
    let keep = scoped_base
        || failed
        || retention.keep(
            metadata["keep_successes"]
                .as_str()
                .ok_or("exploration retention policy missing")?,
            execution,
        );
    if !keep {
        retention.retained(execution)?;
        return Ok(());
    }
    let remaining = if failed || scoped_base {
        u64::MAX
    } else {
        retention.remaining(
            metadata["success_limit"]
                .as_u64()
                .ok_or("missing success count bound")?,
            metadata["success_bytes"]
                .as_u64()
                .ok_or("missing success bytes bound")?,
        )?
    };
    let published =
        super::store(journal.path().join("evidence"), remaining).publish(publication)?;
    execution.artifact = Some(
        published
            .path
            .strip_prefix(journal.path())
            .map_err(|error| error.to_string())?
            .to_string_lossy()
            .into_owned(),
    );
    execution.stored_bytes = published.stored_bytes;
    retention.retained(execution)
}

pub(crate) fn report(
    journal: &RoundJournal,
    json_output: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    let executions = journal
        .events()
        .iter()
        .flat_map(RoundEvent::executions)
        .collect::<Vec<_>>();
    let failures = executions
        .iter()
        .filter(|execution| execution.record.outcome.class != "success")
        .count();
    let signatures = executions
        .iter()
        .filter(|execution| {
            matches!(
                execution.record.outcome.class.as_str(),
                "target_failure" | "watchdog"
            )
        })
        .map(|execution| execution.record.failure_signature())
        .collect::<Result<BTreeSet<_>, _>>()?;
    let envelope = journal.summary()?;
    if json_output {
        writeln!(
            output,
            "{}",
            json!({"schema":"gomad.explore-event/v3","type":"complete","campaign":journal.path(),
            "attempted":executions.len(),"failures":failures,"distinct_failures":signatures.len(),
            "stop_reason":envelope["stop_reason"],"envelope":envelope})
        )
        .map_err(|error| error.to_string())?;
    } else {
        writeln!(
            output,
            "gomad exploration: campaign={} attempted={} failures={} stop={} envelope={}",
            journal.path().display(),
            executions.len(),
            failures,
            envelope["stop_reason"],
            String::from_utf8(canonical::encode(&envelope)?).map_err(|error| error.to_string())?
        )
        .map_err(|error| error.to_string())?;
    }
    Ok(i32::from(failures != 0))
}

fn resume_admission(stop_reason: &str) -> Result<(), String> {
    if stop_reason.is_empty() {
        Ok(())
    } else {
        Err("choice exploration campaign is complete".into())
    }
}

#[cfg(test)]
mod live_backend_tests {
    #[test]
    fn completed_choice_resume_refuses_before_restoration() {
        assert!(super::resume_admission("").is_ok());
        for reason in [
            "max_executions",
            "choice_depth_complete",
            "exploration_capacity",
            "exploration_exhausted",
        ] {
            assert!(
                super::resume_admission(reason).is_err(),
                "completed campaign admitted: {reason}"
            );
        }
    }
}
