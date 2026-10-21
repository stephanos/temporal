use super::choice::{ChoiceConfig, ChoiceResult, ChoiceState};
use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::Deserialize;
use serde_json::json;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    config: ChoiceConfig,
    traces: Vec<Trace>,
    failed_executions: Vec<u64>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Trace {
    selected: Vec<u32>,
    raw: Vec<u8>,
}

pub(crate) fn run(path: &std::path::Path) -> Result<(), String> {
    let bytes = std::fs::read(path).map_err(|error| error.to_string())?;
    if bytes.len() > 4 << 20 {
        return Err("choice controller fixture input exceeds bound".into());
    }
    let input: Input = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.config.max_executions > 128 || input.traces.len() > 64 {
        return Err("choice controller fixture envelope is too large".into());
    }
    for trace in &input.traces {
        let plan = linked::project_choices(&trace.raw, &input.config.execution)?;
        if plan
            .decisions
            .iter()
            .map(linked::Decision::selected)
            .collect::<Vec<_>>()
            != trace.selected
        {
            return Err("choice controller trace ranks differ from authenticated bytes".into());
        }
    }
    let mut state = ChoiceState::new(input.config)?;
    let mut states = vec![serde_json::to_value(&state).map_err(|error| error.to_string())?];
    let mut segments = Vec::new();
    let mut views = Vec::new();
    while let Some(round) = state.next_round() {
        let mut results = Vec::new();
        let mut candidates = Vec::new();
        for (index, candidate) in round.candidates.iter().enumerate() {
            let prefix = candidate
                .replay(&state.config.execution)?
                .map(|request| {
                    linked::project_choices(&candidate.source_trace, &state.config.execution)?
                        .prefix(
                            request
                                .prefix_control
                                .as_ref()
                                .ok_or("fixture prefix control missing")?,
                        )
                })
                .transpose()?;
            let trace = input
                .traces
                .iter()
                .find(|trace| {
                    prefix.as_ref().is_none_or(|prefix| {
                        prefix
                            .decisions
                            .iter()
                            .enumerate()
                            .all(|(index, decision)| {
                                trace.selected.get(index) == Some(&decision.selected())
                            })
                    })
                })
                .ok_or("choice controller fixture lacks a matching observed trace")?;
            let ordinal = state.logical_executions + index as u64;
            let failed = input.failed_executions.contains(&ordinal);
            results.push(ChoiceResult {
                candidate_sha256: candidate.sha256.clone(),
                outcome_sha256: format!("sha256:{}", canonical::digest(b"controller-outcome")),
                failed,
                failure_sha256: if failed {
                    format!(
                        "sha256:{}",
                        canonical::digest(format!("controller-failure-{ordinal}").as_bytes())
                    )
                } else {
                    String::new()
                },
                divergence: None,
                source_trace: Some(trace.raw.clone()),
            });
            candidates.push(json!({"sha256":candidate.sha256,"parent_sha256":candidate.parent_sha256,"forced_depth":candidate.forced_depth,"prefix_sha256":candidate.prefix_sha256}));
        }
        let (next, segment) = state.commit(&round, &results)?;
        if canonical::encode(&state.replay_segment(&segment)?)? != canonical::encode(&next)? {
            return Err("choice segment replay differs".into());
        }
        let mut corrupt = segment.clone();
        corrupt.after_sha256 = format!("sha256:{}", "0".repeat(64));
        if state.replay_segment(&corrupt).is_ok() {
            return Err("choice corrupt segment was accepted".into());
        }
        views.push(json!({"index":round.index,"candidates":candidates,"summary":next.summary()}));
        segments.push(serde_json::to_value(segment).map_err(|error| error.to_string())?);
        states.push(serde_json::to_value(&next).map_err(|error| error.to_string())?);
        state = next;
    }
    println!("{}", String::from_utf8(canonical::encode(&json!({"schema":"gomad.choice-controller-corpus/v1","view":views,"states":states,"segments":segments}))?).map_err(|error| error.to_string())?);
    Ok(())
}
