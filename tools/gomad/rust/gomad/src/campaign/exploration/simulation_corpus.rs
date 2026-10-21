use super::simulation::{
    self, Decision, Dimension, SimulationConfig, SimulationResult, SimulationState,
};
use crate::evidence::canonical;
use serde::Deserialize;
use serde_json::{Value, json};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    config: SimulationConfig,
    decisions: Vec<Template>,
    #[serde(default)]
    diverged_execution: Option<u64>,
    #[serde(default)]
    failed_execution: Option<u64>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Template {
    dimension: Dimension,
    ordinal: u64,
    site_sha256: String,
    alternatives: Vec<String>,
    selected: u32,
}

fn decision(template: &Template, selected: u32) -> Result<Decision, String> {
    let controls = if template.dimension == Dimension::Runtime {
        (0..template.alternatives.len())
            .map(|rank| {
                if rank == selected as usize {
                    Vec::new()
                } else {
                    vec![u8::try_from(rank + 1).unwrap_or(255)]
                }
            })
            .collect()
    } else {
        Vec::new()
    };
    simulation::canonical_decision(
        template.dimension,
        template.ordinal,
        template.site_sha256.clone(),
        template.alternatives.clone(),
        controls,
        selected,
    )
}

pub(crate) fn run(path: &std::path::Path) -> Result<(), String> {
    let bytes = std::fs::read(path).map_err(|error| error.to_string())?;
    if bytes.len() > 1 << 20 {
        return Err("simulation controller fixture input exceeds bound".into());
    }
    let input: Input = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.config.max_executions > 128 || input.decisions.len() > 32 {
        return Err("simulation controller fixture envelope is too large".into());
    }
    let mut state = SimulationState::new(input.config)?;
    let mut states = vec![serde_json::to_value(&state).map_err(|error| error.to_string())?];
    let mut segments = Vec::new();
    let mut views = Vec::new();
    while let Some(round) = state.next_round() {
        let mut results = Vec::new();
        for (index, candidate) in round.candidates.iter().enumerate() {
            let ordinal = state.logical_executions + index as u64;
            let mut decisions = Vec::new();
            for template in &input.decisions {
                let selected = candidate
                    .overrides
                    .iter()
                    .find(|forced| {
                        (forced.dimension, forced.ordinal) == (template.dimension, template.ordinal)
                    })
                    .map_or(template.selected, |forced| forced.selected);
                decisions.push(decision(template, selected)?);
            }
            let failed = input.failed_execution == Some(ordinal);
            results.push(SimulationResult {
                candidate_sha256: candidate.sha256.clone(),
                outcome_sha256: format!("sha256:{}", canonical::digest(b"controller-outcome")),
                failed,
                failure_sha256: if failed {
                    format!("sha256:{}", canonical::digest(b"controller-failure"))
                } else {
                    String::new()
                },
                diverged: input.diverged_execution == Some(ordinal),
                decisions,
            });
        }
        let (next, segment) = state.commit(&round, &results)?;
        if state.replay_segment(&segment)? != next {
            return Err("simulation segment replay differs".into());
        }
        let mut corrupt = segment.clone();
        corrupt.after_sha256 =
            "sha256:0000000000000000000000000000000000000000000000000000000000000000".into();
        if state.replay_segment(&corrupt).is_ok() {
            return Err("simulation corrupt segment was accepted".into());
        }
        views.push(json!({"index":round.index,"candidates":round.candidates.iter().map(|candidate| json!({"sha256":candidate.sha256,"parent_sha256":candidate.parent_sha256,"overrides":candidate.overrides.iter().map(|forced| json!({"dimension":forced.dimension,"ordinal":forced.ordinal,"selected":forced.selected,"identity":forced.identity})).collect::<Vec<_>>()})).collect::<Vec<_>>(),"summary":next.summary()}));
        segments.push(serde_json::to_value(segment).map_err(|error| error.to_string())?);
        states.push(serde_json::to_value(&next).map_err(|error| error.to_string())?);
        state = next;
    }
    let output: Value = json!({"schema":"gomad.simulation-controller-corpus/v1","view":views,"states":states,"segments":segments});
    println!(
        "{}",
        String::from_utf8(canonical::encode(&output)?).map_err(|error| error.to_string())?
    );
    Ok(())
}
