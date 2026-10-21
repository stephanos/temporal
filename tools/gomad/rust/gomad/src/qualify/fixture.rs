//! Fixture-only accounting corpus: controlled raw observations drive the real
//! cost, budget and receipt evaluators so threshold and missing-measurement
//! regressions are deterministic. Absent from release builds.

use super::{
    cost::{self, Observations},
    report::{self, Execution, Replay},
    usage::Usage,
};
use crate::evidence::{
    canonical,
    identity::Identity,
    record::{Limits, Outcome, Record},
};
use serde::Deserialize;
use serde_json::json;
use std::{collections::BTreeMap, path::Path};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Run {
    class: String,
    exit_code: Option<u32>,
    replayed: bool,
    virtual_time_nanos: u64,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    runs: Vec<Run>,
    observations: Observations,
    terminal: Option<Usage>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    schema: String,
    cases: Vec<Case>,
}

fn record(run: &Run, seed: u64, memory_bytes: u64) -> Result<Record, String> {
    let identity = Identity {
        target: canonical::digest(b"qualification-fixture-module"),
        toolchain: canonical::digest(b"qualification-fixture-toolchain"),
        platform: canonical::digest(b"qualification-fixture-platform"),
        implementation: canonical::digest(b"qualification-fixture-implementation"),
        configuration: canonical::digest(b"qualification-fixture-configuration"),
    };
    Ok(Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        seed,
        arguments: Vec::new(),
        environment: BTreeMap::new(),
        limits: Limits {
            fuel: 1 << 32,
            memory_bytes,
            wall_millis: 60_000,
            output_bytes: 1 << 20,
            evidence_bytes: 1 << 20,
        },
        virtual_time_nanos: run.virtual_time_nanos,
        outcome: Outcome {
            class: run.class.clone(),
            termination: "exit".into(),
            exit_code: run.exit_code,
            reason: String::new(),
        },
        streams: BTreeMap::new(),
    })
}

pub(crate) fn run(path: &Path) -> Result<(), String> {
    let input: Input = super::spec::strict(&super::spec::read_bounded(path, 1 << 20)?)?;
    if input.schema != "gomad.qualify-accounting-corpus/v1" || input.cases.len() > 64 {
        return Err("unsupported qualification accounting corpus".into());
    }
    let mut rows = Vec::new();
    for case in input.cases {
        let memory = case
            .observations
            .samples
            .first()
            .map_or(1 << 20, |sample| sample.guest_memory_limit_bytes);
        let mut executions = Vec::new();
        for (index, run) in case.runs.iter().enumerate() {
            let record = record(run, 7, memory)?;
            executions.push(Execution {
                iteration: index as u64 + 1,
                campaign_path: format!("campaign-{}", index + 1),
                artifact_path: Some(format!("artifact-{}", index + 1)),
                record_sha256: record.hash()?,
                record,
                replay: Some(Replay {
                    attempted: true,
                    matched: run.replayed,
                    divergence: (!run.replayed).then(|| "$.record.outcome".into()),
                    choice_replay: "none".into(),
                }),
            });
        }
        let report = report::build(
            &["gomad".into(), "qualify".into()],
            7,
            executions.len() as u64,
            executions,
            None,
        )?;
        let (cost, cost_failure, budget, budget_failure) =
            cost::evaluate(&case.observations, &report);
        let terminal = case
            .terminal
            .map(|after| cost::terminal(&case.observations, after));
        rows.push(json!({
            "name": case.name,
            "classification": report::classification(&report),
            "cost": cost,
            "cost_failure": cost_failure,
            "budget": budget,
            "budget_failure": budget_failure,
            "terminal": terminal.map(|result| match result {
                Ok(cpu) => json!({"cpu_nanos": cpu}),
                Err(error) => json!({"failure": error}),
            }),
        }));
    }
    println!(
        "{}",
        String::from_utf8(canonical::encode(&json!({"cases": rows}))?)
            .map_err(|error| error.to_string())?
    );
    Ok(())
}
