use super::proposals::State;
use crate::campaign::exploration::simulation::{SimulationCandidate, SimulationConfig};
use serde::Deserialize;
use serde_json::json;
use std::{collections::BTreeSet, path::Path};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    config: SimulationConfig,
    original: SimulationCandidate,
    attempt_budget: u64,
    accepted_indices: BTreeSet<u64>,
}

pub(crate) fn run(path: &Path) -> Result<(), String> {
    let bytes = std::fs::read(path).map_err(|error| error.to_string())?;
    if bytes.len() > 1 << 20 {
        return Err("minimizer fixture input exceeds bound".into());
    }
    let input: Input = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.attempt_budget > 128 || input.original.overrides.len() > 32 {
        return Err("minimizer fixture envelope is too large".into());
    }
    let mut state = State::simulation(input.config, input.original, input.attempt_budget)?;
    let mut proposals = Vec::new();
    while let Some(attempt) = state.next()? {
        let accepted = input.accepted_indices.contains(&attempt.index);
        state = state.commit(&attempt, accepted)?;
        proposals.push(attempt);
    }
    println!(
        "{}",
        json!({
            "config":state.config,"original":state.original,"proposals":proposals,
            "accepted":state.accepted,"attempts":state.attempts,"stop_reason":state.stop_reason,
        })
    );
    Ok(())
}
