//! Qualification verdicts derived only from canonical execution records and
//! their replays. Every boolean is recomputed by `validate`.

use crate::evidence::record::Record;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeSet;

pub(crate) const SCHEMA: &str = "gomad.rust-qualification/v1";
pub(crate) const MAXIMUM_REPEAT: u64 = 32;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
#[expect(
    clippy::struct_field_names,
    reason = "The retained replay keeps the existing choice-replay name"
)]
pub(crate) struct Replay {
    pub attempted: bool,
    pub matched: bool,
    pub divergence: Option<String>,
    pub choice_replay: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Execution {
    pub iteration: u64,
    pub campaign_path: String,
    pub artifact_path: Option<String>,
    pub record: Record,
    pub record_sha256: String,
    pub replay: Option<Replay>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Failure {
    pub classification: String,
    pub message: String,
    pub iteration: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Report {
    pub schema: String,
    pub qualified: bool,
    pub deterministic: bool,
    pub target_success: bool,
    pub seed: u64,
    pub repeat: u64,
    pub command: Vec<String>,
    pub record_sha256: Option<String>,
    pub first_divergence: Option<String>,
    pub executions: Vec<Execution>,
    pub failure: Option<Failure>,
}

const CLASSIFICATIONS: [&str; 8] = [
    "runner_failure",
    "unsupported_target",
    "invalid_input",
    "cancelled",
    "overall_timeout",
    "capacity",
    "semantic_coverage_failure",
    "replay_failure",
];

/// Maps a Rust host error prefix onto the existing qualification failure classes.
pub(crate) fn classify_error(error: &str) -> (&'static str, String) {
    if let Some((name, message)) = error.split_once(": ")
        && let Some(class) = CLASSIFICATIONS.iter().find(|class| **class == name)
    {
        let class = if *class == "capacity" {
            "runner_failure"
        } else {
            class
        };
        return (class, message.to_owned());
    }
    ("invalid_input", error.to_owned())
}

/// Builds a completed or failed report. `failure` marks a repetition or replay
/// that could not complete; its partial executions are still retained.
pub(crate) fn build(
    command: &[String],
    seed: u64,
    repeat: u64,
    executions: Vec<Execution>,
    failure: Option<Failure>,
) -> Result<Report, String> {
    let mut report = Report {
        schema: SCHEMA.into(),
        qualified: false,
        deterministic: false,
        target_success: false,
        seed,
        repeat,
        command: command.to_vec(),
        record_sha256: None,
        first_divergence: None,
        executions,
        failure,
    };
    let derived = derive(&report)?;
    report.deterministic = derived.deterministic;
    report.target_success = derived.target_success;
    report.qualified = derived.qualified;
    report.record_sha256 = derived.record_sha256;
    report.first_divergence = derived.first_divergence;
    validate(&report)?;
    Ok(report)
}

struct Derived {
    deterministic: bool,
    target_success: bool,
    qualified: bool,
    record_sha256: Option<String>,
    first_divergence: Option<String>,
}

fn derive(report: &Report) -> Result<Derived, String> {
    let mut first_divergence = None;
    let baseline = report.executions.first();
    let mut deterministic = baseline.is_some();
    let mut target_success = baseline.is_some();
    let mut replayed = true;
    for execution in &report.executions {
        if let Some(baseline) = baseline
            && execution.record_sha256 != baseline.record_sha256
        {
            deterministic = false;
            if first_divergence.is_none() {
                first_divergence = Some(difference(
                    &serde_json::to_value(&baseline.record).map_err(|error| error.to_string())?,
                    &serde_json::to_value(&execution.record).map_err(|error| error.to_string())?,
                    "$.record",
                ));
            }
        }
        target_success &= execution.record.outcome.class == "success";
        replayed &= execution
            .replay
            .as_ref()
            .is_some_and(|replay| replay.matched);
    }
    let failed = report.failure.is_some();
    Ok(Derived {
        deterministic: deterministic && !failed,
        target_success: target_success && !failed,
        qualified: deterministic && target_success && replayed && !failed,
        record_sha256: baseline.map(|execution| execution.record_sha256.clone()),
        first_divergence,
    })
}

/// Names the first differing path between two canonical record projections.
fn difference(left: &Value, right: &Value, path: &str) -> String {
    match (left, right) {
        (Value::Object(left), Value::Object(right)) => {
            let names = left.keys().chain(right.keys()).collect::<BTreeSet<_>>();
            for name in names {
                match (left.get(name), right.get(name)) {
                    (Some(a), Some(b)) if a == b => {}
                    (Some(a), Some(b)) => return difference(a, b, &format!("{path}.{name}")),
                    _ => return format!("{path}.{name}"),
                }
            }
            path.into()
        }
        (Value::Array(left), Value::Array(right)) => {
            for (index, (a, b)) in left.iter().zip(right).enumerate() {
                if a != b {
                    return difference(a, b, &format!("{path}[{index}]"));
                }
            }
            format!("{path}.length")
        }
        _ => path.into(),
    }
}

/// Recomputes every verdict from the retained records and replays.
pub(crate) fn validate(report: &Report) -> Result<(), String> {
    if report.schema != SCHEMA {
        return Err("unsupported qualification report schema".into());
    }
    if report.command.is_empty() || report.command[0].is_empty() {
        return Err("qualification command is required".into());
    }
    if !(2..=MAXIMUM_REPEAT).contains(&report.repeat) {
        return Err(format!(
            "qualification repeat must be between 2 and {MAXIMUM_REPEAT}"
        ));
    }
    let completed = report.executions.len() as u64;
    match &report.failure {
        Some(failure) => {
            // Only a class `classify_error` and the runner produce; `capacity`
            // is always retained as `runner_failure`.
            if !CLASSIFICATIONS.contains(&failure.classification.as_str())
                || failure.classification == "capacity"
                || failure.message.is_empty()
                || failure.iteration == 0
                || failure.iteration > report.repeat
                || completed > report.repeat
            {
                return Err("qualification failure result is inconsistent".into());
            }
        }
        None if completed != report.repeat => {
            return Err("qualification report repetition count is invalid".into());
        }
        None => {}
    }
    for (index, execution) in report.executions.iter().enumerate() {
        execution.record.validate()?;
        if execution.iteration != index as u64 + 1
            || execution.record.seed != report.seed
            || execution.campaign_path.is_empty()
            || execution.record.hash()? != execution.record_sha256
        {
            return Err(format!(
                "qualification execution {index} identity is invalid"
            ));
        }
        if let Some(replay) = &execution.replay {
            if !replay.attempted
                || execution.artifact_path.is_none()
                || replay.matched == replay.divergence.is_some()
                || !matches!(
                    replay.choice_replay.as_str(),
                    "none" | "exact" | "diverged" | "unavailable"
                )
                || replay.matched
                    && execution.record.profile == "gomad.wasi-cooperative/v1"
                    && replay.choice_replay != "exact"
            {
                return Err(format!(
                    "qualification execution {index} replay evidence is invalid"
                ));
            }
        } else if report.failure.is_none() {
            return Err(format!(
                "qualification execution {index} omitted its replay"
            ));
        }
    }
    let derived = derive(report)?;
    if report.deterministic != derived.deterministic
        || report.target_success != derived.target_success
        || report.qualified != derived.qualified
        || report.record_sha256 != derived.record_sha256
        || report.first_divergence != derived.first_divergence
    {
        return Err("qualification verdict differs from retained records and replays".into());
    }
    Ok(())
}

pub(crate) fn classification(report: &Report) -> &str {
    if let Some(failure) = &report.failure {
        return &failure.classification;
    }
    if !report.deterministic {
        return "nondeterministic";
    }
    if report.executions.iter().any(|execution| {
        execution
            .replay
            .as_ref()
            .is_some_and(|replay| !replay.matched)
    }) {
        return "replay_divergence";
    }
    if !report.target_success {
        return "target_failure";
    }
    "qualified"
}

pub(crate) fn exit_status(classification: &str) -> i32 {
    match classification {
        "qualified" => 0,
        "target_failure"
        | "nondeterministic"
        | "replay_divergence"
        | "semantic_coverage_failure" => 1,
        "unsupported_target" | "invalid_input" => 2,
        _ => 3,
    }
}
