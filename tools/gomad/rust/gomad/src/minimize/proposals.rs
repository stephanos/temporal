use crate::campaign::exploration::simulation::{
    self, Dimension, ForcedDecision, SimulationCandidate, SimulationConfig,
};
use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Decision {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub identity: String,
}

impl From<&ForcedDecision> for Decision {
    fn from(value: &ForcedDecision) -> Self {
        Self {
            dimension: value.dimension,
            ordinal: value.ordinal,
            identity: value.identity.clone(),
        }
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Reduction {
    pub kind: String,
    pub before_sha256: String,
    pub after_sha256: String,
    pub removed: Vec<Decision>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Attempt {
    pub index: u64,
    pub reduction: Reduction,
    pub candidate: SimulationCandidate,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct ChoiceSource {
    raw: Vec<u8>,
    identity: linked::ChoiceIdentity,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct State {
    pub schema: String,
    pub config: Option<SimulationConfig>,
    pub original: SimulationCandidate,
    pub current: SimulationCandidate,
    pub attempt_budget: u64,
    pub attempts: u64,
    pub evaluated: Vec<String>,
    pub accepted: Vec<Reduction>,
    pub stop_reason: String,
    pub sha256: String,
    choice_source: Option<ChoiceSource>,
}

impl State {
    pub fn choice(
        raw: Vec<u8>,
        identity: linked::ChoiceIdentity,
        budget: u64,
    ) -> Result<Self, String> {
        let source = ChoiceSource { raw, identity };
        let plan = linked::project_choices(&source.raw, &source.identity)?;
        let original = choice_candidate(&source, plan.decisions.len() as u64, String::new())?;
        Self::initial(None, original, budget, Some(source))
    }

    pub fn simulation(
        config: SimulationConfig,
        original: SimulationCandidate,
        budget: u64,
    ) -> Result<Self, String> {
        Self::initial(Some(config), original, budget, None)
    }

    fn initial(
        config: Option<SimulationConfig>,
        original: SimulationCandidate,
        budget: u64,
        choice_source: Option<ChoiceSource>,
    ) -> Result<Self, String> {
        if budget == 0 {
            return Err("minimizer attempt budget must be positive".into());
        }
        let canonical = match (&config, &choice_source) {
            (Some(config), None) => simulation::canonical_candidate(
                config,
                original.overrides.clone(),
                original.parent_sha256.clone(),
            )?,
            (None, Some(source)) => {
                let plan = linked::project_choices(&source.raw, &source.identity)?;
                choice_candidate(source, plan.decisions.len() as u64, String::new())?
            }
            _ => return Err("minimizer state mode is invalid".into()),
        };
        if canonical != original {
            return Err("minimizer candidate identity does not match its contents".into());
        }
        let mut state = Self {
            schema: if choice_source.is_some() {
                "gomad.rust-minimizer-state/choice-prefix/v1"
            } else {
                "gomad.rust-minimizer-state/v1"
            }
            .into(),
            config,
            current: original.clone(),
            original,
            attempt_budget: budget,
            attempts: 0,
            evaluated: Vec::new(),
            accepted: Vec::new(),
            stop_reason: String::new(),
            sha256: String::new(),
            choice_source,
        };
        state.seal()?;
        Ok(state)
    }

    pub fn implementation(&self) -> String {
        if self.choice_source.is_some() {
            simulation::domain_hash(
                "gomad-minimizer-controller/choice-prefix/v1",
                b"effective-prefix-suffixes;deterministic;bounded/v1",
            )
        } else {
            simulation::domain_hash(
                "gomad-minimizer-controller/v1",
                b"suffix/ranges/faults;deterministic;bounded/v1",
            )
        }
    }

    pub fn next(&self) -> Result<Option<Attempt>, String> {
        self.validate()?;
        if !self.stop_reason.is_empty() {
            return Ok(None);
        }
        self.proposal()?
            .map(Some)
            .ok_or_else(|| "active minimizer state has no proposal".into())
    }

    pub fn commit(&self, attempt: &Attempt, accepted: bool) -> Result<Self, String> {
        if self.next()?.as_ref() != Some(attempt) {
            return Err("minimizer attempt does not match the current state".into());
        }
        let mut next = self.clone();
        next.apply(attempt, accepted)?;
        Ok(next)
    }

    fn apply(&mut self, attempt: &Attempt, accepted: bool) -> Result<(), String> {
        self.attempts = self
            .attempts
            .checked_add(1)
            .ok_or("minimizer attempt count overflow")?;
        self.evaluated.push(attempt.candidate.sha256.clone());
        if accepted {
            self.current = attempt.candidate.clone();
            self.accepted.push(attempt.reduction.clone());
        }
        self.seal()
    }

    fn seal(&mut self) -> Result<(), String> {
        self.stop_reason = if self.attempts == self.attempt_budget {
            "attempt_budget"
        } else if self.proposal()?.is_none() {
            "minimal"
        } else {
            ""
        }
        .into();
        self.sha256 = self.hash()?;
        Ok(())
    }

    fn hash(&self) -> Result<String, String> {
        let mut projection = self.clone();
        projection.sha256.clear();
        simulation::identity("gomad-rust-minimizer-state/v1", &projection)
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.attempts > self.attempt_budget
            || self.evaluated.len() as u64 != self.attempts
            || self.accepted.len() as u64 > self.attempts
            || self.sha256 != self.hash()?
        {
            return Err("minimizer state shape or identity is invalid".into());
        }
        let mut replay = Self::initial(
            self.config.clone(),
            self.original.clone(),
            self.attempt_budget,
            self.choice_source.clone(),
        )?;
        let mut accepted_index = 0;
        for evaluated in &self.evaluated {
            if !replay.stop_reason.is_empty() {
                return Err("minimizer history continued after its stop".into());
            }
            let attempt = replay
                .proposal()?
                .ok_or("minimizer history exceeds its proposal search")?;
            if attempt.candidate.sha256 != *evaluated {
                return Err("minimizer evaluated candidates are not in controller order".into());
            }
            let accepted = self.accepted.get(accepted_index) == Some(&attempt.reduction);
            if accepted {
                accepted_index += 1;
            }
            replay.apply(&attempt, accepted)?;
        }
        if canonical::encode(&replay)? != canonical::encode(self)? {
            return Err("minimizer state does not follow its recorded reductions".into());
        }
        Ok(())
    }

    fn proposal(&self) -> Result<Option<Attempt>, String> {
        let seen = self
            .evaluated
            .iter()
            .chain(std::iter::once(&self.current.sha256))
            .collect::<BTreeSet<_>>();
        if let Some(source) = &self.choice_source {
            if !self.accepted.is_empty() {
                return Ok(None);
            }
            for length in 0..self.current.overrides.len() {
                let candidate =
                    choice_candidate(source, length as u64, self.current.sha256.clone())?;
                if !seen.contains(&candidate.sha256) {
                    let removed = self.current.overrides[length..]
                        .iter()
                        .map(Decision::from)
                        .collect();
                    return Ok(Some(self.attempt("schedule_suffix", candidate, removed)));
                }
            }
            return Ok(None);
        }
        let config = self
            .config
            .as_ref()
            .ok_or("simulation minimizer configuration is missing")?;
        for (kind, indexes, suffix_only) in [
            (
                "schedule_suffix",
                self.indexes(|dimension| dimension == Dimension::Runtime),
                true,
            ),
            (
                "schedule_range",
                self.indexes(|dimension| dimension != Dimension::Fault),
                false,
            ),
            (
                "fault_entries",
                self.indexes(|dimension| dimension == Dimension::Fault),
                false,
            ),
        ] {
            for length in (1..=indexes.len()).rev() {
                let last_start = indexes.len() - length;
                let first_start = if suffix_only { last_start } else { 0 };
                for start in first_start..=last_start {
                    let removed_indexes = &indexes[start..start + length];
                    let overrides = self
                        .current
                        .overrides
                        .iter()
                        .enumerate()
                        .filter(|(index, _)| removed_indexes.binary_search(index).is_err())
                        .map(|(_, forced)| forced.clone())
                        .collect();
                    let candidate = simulation::canonical_candidate(
                        config,
                        overrides,
                        self.current.sha256.clone(),
                    )?;
                    if !seen.contains(&candidate.sha256) {
                        let removed = removed_indexes
                            .iter()
                            .map(|index| Decision::from(&self.current.overrides[*index]))
                            .collect();
                        return Ok(Some(self.attempt(kind, candidate, removed)));
                    }
                }
            }
        }
        Ok(None)
    }

    fn indexes(&self, include: impl Fn(Dimension) -> bool) -> Vec<usize> {
        self.current
            .overrides
            .iter()
            .enumerate()
            .filter_map(|(index, forced)| include(forced.dimension).then_some(index))
            .collect()
    }

    fn attempt(
        &self,
        kind: &str,
        candidate: SimulationCandidate,
        removed: Vec<Decision>,
    ) -> Attempt {
        Attempt {
            index: self.attempts,
            reduction: Reduction {
                kind: kind.into(),
                before_sha256: self.current.sha256.clone(),
                after_sha256: candidate.sha256.clone(),
                removed,
            },
            candidate,
        }
    }
}

fn choice_candidate(
    source: &ChoiceSource,
    length: u64,
    parent_sha256: String,
) -> Result<SimulationCandidate, String> {
    let prefix =
        linked::project_choices(&source.raw, &source.identity)?.prefix(&linked::PrefixControl {
            cutoff: length,
            r#override: None,
        })?;
    let mut control = Vec::with_capacity(40);
    control.extend_from_slice(&length.to_be_bytes());
    control.extend_from_slice(&prefix.sha256);
    let sha256 = simulation::domain_hash("gomad-minimization-choice-prefix/v1", &control);
    let overrides = prefix
        .decisions
        .iter()
        .map(|decision| ForcedDecision {
            dimension: Dimension::Runtime,
            ordinal: decision.ordinal(),
            site_sha256: String::new(),
            alternatives: 0,
            alternative_set_sha256: String::new(),
            selected: 0,
            selected_sha256: String::new(),
            control: Vec::new(),
            source_trace: Vec::new(),
            identity: format!(
                "sha256:{}",
                crate::campaign::exploration::choice::hex(&decision.selected_identity())
            ),
        })
        .collect();
    Ok(SimulationCandidate {
        sha256,
        parent_sha256,
        overrides,
        nil_overrides: false,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::campaign::exploration::simulation::{DimensionLimits, canonical_forced_decision};

    fn simulation() -> State {
        let config = SimulationConfig {
            execution_sha256: simulation::domain_hash("test", b"execution"),
            controller_sha256: simulation::implementation_sha256(),
            base_seed: 7,
            parallel: 1,
            max_executions: 100,
            max_forced_decisions: 10,
            max_exploration_bytes: 100_000,
            max_result_bytes: 100_000,
            failure_budget: 10,
            dimension_limits: DimensionLimits {
                runtime: 10,
                scenario: 10,
                network: 10,
                storage: 10,
                fault: 10,
                crash: 10,
            },
        };
        let overrides = [
            Dimension::Runtime,
            Dimension::Runtime,
            Dimension::Scenario,
            Dimension::Network,
            Dimension::Storage,
            Dimension::Fault,
            Dimension::Crash,
        ]
        .into_iter()
        .enumerate()
        .map(|(index, dimension)| {
            canonical_forced_decision(ForcedDecision {
                dimension,
                ordinal: index as u64,
                site_sha256: simulation::domain_hash("test", b"site"),
                alternatives: 2,
                alternative_set_sha256: simulation::domain_hash("test", b"set"),
                selected: 1,
                selected_sha256: simulation::domain_hash("test", b"selected"),
                control: if dimension == Dimension::Runtime {
                    vec![1]
                } else {
                    vec![]
                },
                source_trace: vec![],
                identity: String::new(),
            })
            .unwrap()
        })
        .collect();
        let original = simulation::canonical_candidate(&config, overrides, String::new()).unwrap();
        State::simulation(config, original, 100).unwrap()
    }

    #[test]
    fn simulation_proposal_order_and_rejected_candidate_deduplication() {
        let mut state = simulation();
        let mut proposals = Vec::new();
        while let Some(attempt) = state.next().unwrap() {
            proposals.push((
                attempt.reduction.kind.clone(),
                attempt
                    .reduction
                    .removed
                    .iter()
                    .map(|decision| decision.ordinal)
                    .collect::<Vec<_>>(),
            ));
            state = state.commit(&attempt, false).unwrap();
        }
        assert_eq!(proposals[0], ("schedule_suffix".into(), vec![0, 1]));
        assert_eq!(proposals[1], ("schedule_suffix".into(), vec![1]));
        assert_eq!(
            proposals[2],
            ("schedule_range".into(), vec![0, 1, 2, 3, 4, 6])
        );
        assert_eq!(
            proposals.last().unwrap(),
            &("fault_entries".into(), vec![5])
        );
        assert_eq!(proposals.len(), 22);
        assert_eq!(
            proposals[3..7]
                .iter()
                .map(|(_, removed)| removed.clone())
                .collect::<Vec<_>>(),
            vec![
                vec![0, 1, 2, 3, 4],
                vec![1, 2, 3, 4, 6],
                vec![0, 1, 2, 3],
                vec![1, 2, 3, 4]
            ],
        );
        assert_eq!(state.stop_reason, "minimal");
        assert_eq!(
            state.evaluated.iter().collect::<BTreeSet<_>>().len(),
            state.attempts as usize
        );
    }

    #[test]
    fn state_replays_reductions_and_rejects_forged_attempts_and_history() {
        let state = simulation();
        let attempt = state.next().unwrap().unwrap();
        let accepted = state.commit(&attempt, true).unwrap();
        assert_eq!(accepted.current, attempt.candidate);
        assert!(accepted.commit(&attempt, false).is_err());
        let wire = serde_json::to_vec(&accepted).unwrap();
        let restored: State = serde_json::from_slice(&wire).unwrap();
        restored.validate().unwrap();
        let mut forged = restored;
        forged.evaluated[0] = simulation::domain_hash("forged", b"candidate");
        forged.sha256 = forged.hash().unwrap();
        assert!(forged.validate().is_err());
    }

    #[test]
    fn attempt_budget_wins_over_minimal() {
        let mut state = simulation();
        state = State::simulation(state.config.take().unwrap(), state.original, 1).unwrap();
        let attempt = state.next().unwrap().unwrap();
        state = state.commit(&attempt, true).unwrap();
        assert_eq!(state.stop_reason, "attempt_budget");
        assert!(state.next().unwrap().is_none());
    }

    fn choice(budget: u64) -> State {
        let mut raw = vec![0; 3 * 96];
        for (ordinal, record) in raw.chunks_exact_mut(96).enumerate() {
            record[..8].copy_from_slice(&(ordinal as u64).to_be_bytes());
            record[8] = 1;
            record[9] = 1;
            record[12..16].copy_from_slice(&(if ordinal == 1 { 1_u32 } else { 2 }).to_be_bytes());
            record[24..32].copy_from_slice(&8_u64.to_be_bytes());
            record[32..64].fill(1);
            record[64..96].fill(2);
        }
        State::choice(
            raw,
            linked::ChoiceIdentity {
                target_sha256: [1; 32],
                toolchain_build_key: "02".repeat(32),
                goos: "wasip1".into(),
                goarch: "wasm".into(),
                implementation_sha256: [3; 32],
            },
            budget,
        )
        .unwrap()
    }

    #[test]
    fn choice_search_uses_effective_prefixes_and_stops_at_first_accept() {
        let mut state = choice(10);
        assert_eq!(state.original.overrides.len(), 2);
        let zero = state.next().unwrap().unwrap();
        assert!(zero.candidate.overrides.is_empty());
        assert_eq!(
            zero.reduction
                .removed
                .iter()
                .map(|decision| decision.ordinal)
                .collect::<Vec<_>>(),
            vec![0, 1]
        );
        state = state.commit(&zero, false).unwrap();
        let one = state.next().unwrap().unwrap();
        assert_eq!(one.candidate.overrides.len(), 1);
        state = state.commit(&one, true).unwrap();
        assert_eq!(state.stop_reason, "minimal");
        assert!(state.next().unwrap().is_none());
        state.validate().unwrap();
    }

    #[test]
    fn choice_reject_all_and_budget_precedence() {
        for accepted in [false, true] {
            let state = choice(1);
            let state = state
                .commit(&state.next().unwrap().unwrap(), accepted)
                .unwrap();
            assert_eq!(state.stop_reason, "attempt_budget");
        }
        let mut state = choice(10);
        while let Some(attempt) = state.next().unwrap() {
            state = state.commit(&attempt, false).unwrap();
        }
        assert_eq!(state.attempts, 2);
        assert_eq!(state.current, state.original);
        assert_eq!(state.stop_reason, "minimal");
    }
}
