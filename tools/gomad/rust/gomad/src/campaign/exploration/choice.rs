use super::{hash, valid_hash};
use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::collections::{BTreeMap, BTreeSet};

#[cfg(test)]
#[path = "choice_tests.rs"]
mod tests;

pub(crate) const SCHEMA: &str = "gomad.choice-exploration-round/v3";

pub(crate) fn implementation() -> String {
    super::domain_hash(
        "gomad-choice-exploration-controller/v1",
        b"deterministic-rounds/breadth-first-rank-prefix/v3",
    )
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceConfig {
    pub execution: linked::ChoiceIdentity,
    pub controller_sha256: String,
    pub base_seed: u64,
    pub parallel: usize,
    pub max_executions: u64,
    pub max_choice_depth: u64,
    pub start_ordinal: u64,
    pub max_exploration_bytes: u64,
    pub failure_policy: String,
    pub failure_budget: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceCandidate {
    pub sha256: String,
    pub parent_sha256: String,
    pub source_trace_sha256: String,
    pub forced_depth: u64,
    pub prefix_sha256: String,
    pub prefix_bytes: Vec<u8>,
    pub source_trace: Vec<u8>,
    pub rank: Option<u32>,
}

pub(crate) use linked::Decision;

pub(crate) fn decisions(plan: &linked::ChoicePlan) -> Result<Vec<Decision>, String> {
    Ok(plan.decisions.clone())
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceResult {
    pub candidate_sha256: String,
    pub outcome_sha256: String,
    pub failed: bool,
    pub failure_sha256: String,
    pub divergence: Option<linked::Divergence>,
    pub source_trace: Option<Vec<u8>>,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceState {
    pub config: ChoiceConfig,
    pub queue: Vec<ChoiceCandidate>,
    pub seen: BTreeSet<String>,
    pub outcomes: BTreeSet<String>,
    pub failure_signatures: BTreeSet<String>,
    pub logical_executions: u64,
    pub committed_rounds: u64,
    pub pending_bytes: u64,
    pub deepest_prefix: u64,
    pub omitted_by_execution_bound: u64,
    pub omitted_by_depth: u64,
    pub omitted_by_capacity: u64,
    pub omitted_by_select_readiness: u64,
    pub stop_reason: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceRound {
    pub index: u64,
    pub candidates: Vec<ChoiceCandidate>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChoiceRoundSegment {
    pub schema: String,
    pub index: u64,
    pub before_sha256: String,
    pub after_sha256: String,
    pub results: Vec<ChoiceResult>,
    pub sha256: String,
}

impl ChoiceCandidate {
    pub(crate) fn replay(
        &self,
        identity: &linked::ChoiceIdentity,
    ) -> Result<Option<linked::ReplayRequest>, String> {
        if self.forced_depth == 0 {
            if self.rank.is_some()
                || !self.prefix_bytes.is_empty()
                || !self.prefix_sha256.is_empty()
                || !self.source_trace.is_empty()
                || !self.source_trace_sha256.is_empty()
            {
                return Err("root choice candidate contains a prefix".into());
            }
            return Ok(None);
        }
        let plan = linked::project_choices(&self.source_trace, identity)?;
        let control = linked::PrefixControl {
            cutoff: self.forced_depth,
            r#override: Some(linked::RankOverride {
                ordinal: self.forced_depth - 1,
                rank: self.rank.ok_or("choice candidate rank is missing")?,
                reconstruct: false,
            }),
        };
        let full_digest = hex(&plan.sha256);
        let full_decisions = plan.decisions.len();
        let prefix = plan.prefix(&control)?;
        if prefix.bytes != self.prefix_bytes
            || format!("sha256:{}", hex(&prefix.sha256)) != self.prefix_sha256
            || format!("sha256:{}", canonical::digest(&self.source_trace))
                != self.source_trace_sha256
        {
            return Err("choice candidate prefix identity mismatch".into());
        }
        Ok(Some(linked::ReplayRequest {
            schema: "gomad.replay-trace/v1".into(),
            execution_identity: identity.clone(),
            trace_sha256: canonical::digest(&self.source_trace),
            tape_sha256: full_digest,
            decisions: full_decisions as u64,
            trace_bytes: self.source_trace.len() as u64,
            trace_frames: (self.source_trace.len() as u64).div_ceil(65536),
            prefix_control: Some(control),
        }))
    }
}

pub(crate) fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}

impl ChoiceState {
    pub(crate) fn new(config: ChoiceConfig) -> Result<Self, String> {
        if config.parallel == 0
            || config.max_executions == 0
            || config.max_choice_depth == 0
            || config.max_exploration_bytes == 0
        {
            return Err(
                "choice exploration requires positive parallel, execution, depth, and byte bounds"
                    .into(),
            );
        }
        if config.controller_sha256 != implementation() {
            return Err("choice exploration controller identity mismatch".into());
        }
        match config.failure_policy.as_str() {
            "first" | "all" if config.failure_budget == 1 => {}
            "budget" if config.failure_budget > 0 => {}
            _ => return Err("invalid choice exploration failure policy or budget".into()),
        }
        // The helper validates the identity even for a trace with no branching decisions.
        linked::project_choices(&[], &config.execution)?;
        let root = candidate(&config, None, "", &[])?;
        let pending_bytes = queue_bytes(std::slice::from_ref(&root))?;
        if pending_bytes > config.max_exploration_bytes {
            return Err("choice exploration root exceeds the exploration byte bound".into());
        }
        Ok(Self {
            config,
            seen: BTreeSet::from([root.sha256.clone()]),
            queue: vec![root],
            outcomes: BTreeSet::new(),
            failure_signatures: BTreeSet::new(),
            logical_executions: 0,
            committed_rounds: 0,
            pending_bytes,
            deepest_prefix: 0,
            omitted_by_execution_bound: 0,
            omitted_by_depth: 0,
            omitted_by_capacity: 0,
            omitted_by_select_readiness: 0,
            stop_reason: String::new(),
        })
    }

    pub(crate) fn next_round(&self) -> Option<ChoiceRound> {
        if !self.stop_reason.is_empty() || self.queue.is_empty() {
            return None;
        }
        Some(ChoiceRound {
            index: self.committed_rounds,
            candidates: self.queue[..self.config.parallel.min(self.queue.len())].to_vec(),
        })
    }

    pub(crate) fn hash(&self) -> Result<String, String> {
        hash("gomad-choice-exploration-state/v2", self)
    }

    pub(crate) fn summary(&self) -> serde_json::Value {
        json!({
            "parallel":self.config.parallel,"max_executions":self.config.max_executions,
            "max_choice_depth":self.config.max_choice_depth,"start_ordinal":self.config.start_ordinal,
            "max_exploration_bytes":self.config.max_exploration_bytes,
            "logical_executions":self.logical_executions,"committed_rounds":self.committed_rounds,
            "pending":self.queue.len(),"pending_bytes":self.pending_bytes,"seen_prefixes":self.seen.len(),
            "deduplicated_outcomes":self.outcomes.len(),"deepest_prefix":self.deepest_prefix,
            "omitted_by_execution_bound":self.omitted_by_execution_bound,
            "omitted_by_depth":self.omitted_by_depth,"omitted_by_capacity":self.omitted_by_capacity,
            "omitted_by_select_readiness":self.omitted_by_select_readiness,"stop_reason":self.stop_reason,
            "bounded_complete":matches!(self.stop_reason.as_str(),"exploration_exhausted"|"choice_depth_complete")
        })
    }

    pub(crate) fn commit(
        &self,
        round: &ChoiceRound,
        results: &[ChoiceResult],
    ) -> Result<(Self, ChoiceRoundSegment), String> {
        if self.next_round().as_ref() != Some(round) || results.len() != round.candidates.len() {
            return Err("choice exploration round or result count mismatch".into());
        }
        let mut next = self.clone();
        next.queue.drain(..round.candidates.len());
        next.logical_executions = next
            .logical_executions
            .checked_add(results.len() as u64)
            .ok_or("choice execution accounting overflow")?;
        next.committed_rounds = next
            .committed_rounds
            .checked_add(1)
            .ok_or("choice round accounting overflow")?;
        let mut children = BTreeMap::new();
        let mut start_unreached = false;
        for (parent, result) in round.candidates.iter().zip(results) {
            if result.candidate_sha256 != parent.sha256 {
                return Err("choice exploration result candidate mismatch".into());
            }
            if let Some(divergence) = &result.divergence {
                if !result.outcome_sha256.is_empty()
                    || result.failed
                    || !result.failure_sha256.is_empty()
                    || result.source_trace.is_some()
                {
                    return Err("choice divergence contains outcome evidence".into());
                }
                validate_divergence(parent, &self.config.execution, divergence)?;
                if next.config.failure_policy == "first" {
                    next.stop_reason = "first_failure".into();
                }
                continue;
            }
            if !valid_hash(&result.outcome_sha256)
                || result.failed && !valid_hash(&result.failure_sha256)
                || !result.failed && !result.failure_sha256.is_empty()
            {
                return Err("choice result outcome or failure identity is invalid".into());
            }
            next.outcomes.insert(result.outcome_sha256.clone());
            if result.failed {
                next.failure_signatures
                    .insert(result.failure_sha256.clone());
                if next.config.failure_policy == "first" {
                    next.stop_reason = "first_failure".into();
                } else if next.config.failure_policy == "budget"
                    && next.failure_signatures.len() as u64 >= next.config.failure_budget
                {
                    next.stop_reason = "failure_budget".into();
                }
            }
            if let Some(raw) = &result.source_trace {
                let plan = linked::project_choices(raw, &next.config.execution)?;
                let observed = decisions(&plan)?;
                validate_trace(parent, &next.config.execution, &observed)?;
                if parent.forced_depth == 0
                    && next.config.start_ordinal != 0
                    && observed.len() as u64 <= next.config.start_ordinal
                {
                    start_unreached = true;
                }
                if next.stop_reason.is_empty() {
                    let no_op = plan.no_op_select_polls();
                    for (ordinal, decision) in observed.iter().enumerate() {
                        let ordinal = ordinal as u64;
                        if ordinal < parent.forced_depth.max(next.config.start_ordinal)
                            || decision.alternatives() <= 1
                        {
                            continue;
                        }
                        if no_op[ordinal as usize] {
                            next.omitted_by_select_readiness +=
                                u64::from(decision.alternatives() - 1);
                            continue;
                        }
                        if ordinal - next.config.start_ordinal + 1 > next.config.max_choice_depth {
                            next.omitted_by_depth += u64::from(decision.alternatives() - 1);
                            continue;
                        }
                        for rank in 0..decision.alternatives() {
                            if rank == decision.selected() {
                                continue;
                            }
                            let child = candidate(
                                &next.config,
                                Some((ordinal + 1, rank)),
                                &parent.sha256,
                                raw,
                            )?;
                            if next.seen.contains(&child.sha256) {
                                continue;
                            }
                            if children
                                .get(&child.sha256)
                                .is_some_and(|previous| previous != &child)
                            {
                                return Err("choice candidate identity collision".into());
                            }
                            children.insert(child.sha256.clone(), child);
                        }
                    }
                }
            } else if parent.forced_depth == 0 && next.config.start_ordinal != 0 {
                start_unreached = true;
            }
        }
        let mut children = children.into_values().collect::<Vec<_>>();
        sort(&mut children);
        let mut pending = queue_bytes(&next.queue)?;
        for (index, child) in children.iter().enumerate() {
            let omitted = (children.len() - index) as u64;
            if next.seen.len() as u64 >= next.config.max_executions {
                next.omitted_by_execution_bound += omitted;
                break;
            }
            let bytes = candidate_bytes(child)?;
            let total = pending
                .checked_add(bytes)
                .ok_or("choice exploration byte accounting overflow")?;
            if total > next.config.max_exploration_bytes {
                next.omitted_by_capacity += omitted;
                if next.stop_reason.is_empty() {
                    next.stop_reason = "exploration_capacity".into();
                }
                break;
            }
            pending = total;
            next.seen.insert(child.sha256.clone());
            next.deepest_prefix = next.deepest_prefix.max(child.forced_depth);
            next.queue.push(child.clone());
        }
        sort(&mut next.queue);
        if next.stop_reason.is_empty() && next.queue.is_empty() {
            next.stop_reason = if start_unreached {
                "choice_start_unreached"
            } else if next.omitted_by_execution_bound != 0
                || next.logical_executions >= next.config.max_executions
            {
                "max_executions"
            } else if next.omitted_by_depth != 0 {
                "choice_depth_complete"
            } else {
                "exploration_exhausted"
            }
            .into();
        }
        next.pending_bytes = queue_bytes(&next.queue)?;
        let mut segment = ChoiceRoundSegment {
            schema: SCHEMA.into(),
            index: round.index,
            before_sha256: self.hash()?,
            after_sha256: next.hash()?,
            results: results.to_vec(),
            sha256: String::new(),
        };
        segment.sha256 = segment.hash()?;
        Ok((next, segment))
    }

    pub(crate) fn replay_segment(&self, segment: &ChoiceRoundSegment) -> Result<Self, String> {
        if segment.schema != SCHEMA
            || segment.sha256 != segment.hash()?
            || segment.before_sha256 != self.hash()?
        {
            return Err("choice round segment identity or state link mismatch".into());
        }
        let round = self
            .next_round()
            .ok_or("choice segment round unavailable")?;
        let (next, regenerated) = self.commit(&round, &segment.results)?;
        if &regenerated != segment {
            return Err("choice segment replay changed its recorded result".into());
        }
        Ok(next)
    }
}

impl ChoiceRoundSegment {
    fn hash(&self) -> Result<String, String> {
        let mut copy = self.clone();
        copy.sha256.clear();
        hash("gomad-choice-exploration-round-segment/v3", &copy)
    }
}

fn candidate(
    config: &ChoiceConfig,
    forcing: Option<(u64, u32)>,
    parent: &str,
    raw: &[u8],
) -> Result<ChoiceCandidate, String> {
    let mut value = ChoiceCandidate {
        sha256: String::new(),
        parent_sha256: parent.into(),
        source_trace_sha256: String::new(),
        forced_depth: 0,
        prefix_sha256: String::new(),
        prefix_bytes: Vec::new(),
        source_trace: Vec::new(),
        rank: None,
    };
    let mut projected = Vec::new();
    if let Some((depth, rank)) = forcing {
        if depth <= config.start_ordinal {
            return Err("choice prefix precedes its start ordinal".into());
        }
        let plan =
            linked::project_choices(raw, &config.execution)?.prefix(&linked::PrefixControl {
                cutoff: depth,
                r#override: Some(linked::RankOverride {
                    ordinal: depth - 1,
                    rank,
                    reconstruct: false,
                }),
            })?;
        projected = decisions(&plan)?;
        value.forced_depth = depth;
        value.prefix_sha256 = format!("sha256:{}", hex(&plan.sha256));
        value.prefix_bytes = plan.bytes;
        value.source_trace = raw.to_vec();
        value.source_trace_sha256 = format!("sha256:{}", canonical::digest(raw));
        value.rank = Some(rank);
    }
    #[derive(Serialize)]
    struct DecisionProjection {
        ordinal: String,
        kind: u8,
        site_offset: String,
        site_missing: bool,
        rank_override: bool,
        alternatives: u32,
        selected: u32,
        data: u32,
        #[serde(skip_serializing_if = "String::is_empty")]
        selected_identity: String,
        alternative_set_digest: String,
    }
    #[derive(Serialize)]
    struct Projection<'a> {
        target_sha256: String,
        toolchain_build_key: &'a str,
        goos: &'a str,
        goarch: &'a str,
        implementation_sha256: String,
        controller_sha256: &'a str,
        base_seed: String,
        decisions: Vec<DecisionProjection>,
    }
    let projection = Projection {
        target_sha256: format!("sha256:{}", hex(&config.execution.target_sha256)),
        toolchain_build_key: &config.execution.toolchain_build_key,
        goos: &config.execution.goos,
        goarch: &config.execution.goarch,
        implementation_sha256: format!("sha256:{}", hex(&config.execution.implementation_sha256)),
        controller_sha256: &config.controller_sha256,
        base_seed: config.base_seed.to_string(),
        decisions: projected
            .iter()
            .map(|decision| DecisionProjection {
                ordinal: decision.ordinal().to_string(),
                kind: decision.kind(),
                site_offset: decision.site().0.to_string(),
                site_missing: decision.site().1,
                rank_override: decision.rank_override(),
                alternatives: decision.alternatives(),
                selected: decision.selected(),
                data: decision.data(),
                selected_identity: if decision.selected_identity() == [0; 32] {
                    String::new()
                } else {
                    format!("sha256:{}", hex(&decision.selected_identity()))
                },
                alternative_set_digest: format!(
                    "sha256:{}",
                    hex(&decision.alternative_set_digest())
                ),
            })
            .collect(),
    };
    value.sha256 = super::domain_hash(
        "gomad-choice-exploration-candidate/v1",
        &serde_json::to_vec(&projection).map_err(|error| error.to_string())?,
    );
    Ok(value)
}

pub(crate) fn validate_divergence(
    candidate: &ChoiceCandidate,
    identity: &linked::ChoiceIdentity,
    divergence: &linked::Divergence,
) -> Result<(), String> {
    candidate
        .replay(identity)?
        .ok_or("choice divergence requires a forced prefix")?;
    if divergence.ordinal >= candidate.forced_depth
        || divergence.tape_records != candidate.forced_depth
        || !matches!(divergence.reason, 1..=5 | 7 | 11)
    {
        return Err("choice divergence does not reference its forced prefix".into());
    }
    let plan = linked::project_choices(&candidate.source_trace, identity)?.prefix(
        &linked::PrefixControl {
            cutoff: candidate.forced_depth,
            r#override: Some(linked::RankOverride {
                ordinal: candidate.forced_depth - 1,
                rank: candidate.rank.ok_or("missing candidate rank")?,
                reconstruct: false,
            }),
        },
    )?;
    let expected = plan
        .decisions
        .get(divergence.ordinal as usize)
        .ok_or("missing divergence decision")?;
    if divergence.expected.as_ref() != Some(expected) {
        return Err("choice divergence expected decision differs from candidate prefix".into());
    }
    if let Some(observed) = &divergence.observed {
        if observed.ordinal() != divergence.ordinal || observed.rank_override() {
            return Err("choice divergence observed decision is invalid".into());
        }
    }
    Ok(())
}

fn validate_trace(
    candidate: &ChoiceCandidate,
    identity: &linked::ChoiceIdentity,
    observed: &[Decision],
) -> Result<(), String> {
    if candidate.forced_depth == 0 {
        return Ok(());
    }
    candidate.replay(identity)?;
    let prefix = linked::project_choices(&candidate.source_trace, identity)?.prefix(
        &linked::PrefixControl {
            cutoff: candidate.forced_depth,
            r#override: Some(linked::RankOverride {
                ordinal: candidate.forced_depth - 1,
                rank: candidate.rank.ok_or("missing candidate rank")?,
                reconstruct: false,
            }),
        },
    )?;
    let expected = decisions(&prefix)?;
    if observed.len() < expected.len() {
        return Err("choice trace ended before its forced prefix".into());
    }
    for (expected, observed) in expected.iter().zip(observed) {
        let matches = if expected.rank_override() {
            expected.ordinal() == observed.ordinal()
                && expected.kind() == observed.kind()
                && expected.site() == observed.site()
                && expected.alternatives() == observed.alternatives()
                && expected.selected() == observed.selected()
                && expected.data() == observed.data()
                && expected.alternative_set_digest() == observed.alternative_set_digest()
                && !observed.rank_override()
                && observed.selected_identity() != [0; 32]
        } else {
            expected == observed
        };
        if !matches {
            return Err(format!(
                "choice prefix diverged at decision {}",
                expected.ordinal()
            ));
        }
    }
    Ok(())
}

fn sort(candidates: &mut [ChoiceCandidate]) {
    candidates.sort_by(|left, right| {
        (left.forced_depth, &left.sha256).cmp(&(right.forced_depth, &right.sha256))
    });
}

fn candidate_bytes(candidate: &ChoiceCandidate) -> Result<u64, String> {
    #[derive(Serialize)]
    struct Projection<'a> {
        sha256: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        parent_sha256: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        source_trace_sha256: &'a str,
        forced_depth: u64,
        #[serde(skip_serializing_if = "str::is_empty")]
        prefix_sha256: &'a str,
        #[serde(skip_serializing_if = "String::is_empty")]
        prefix_bytes: String,
        #[serde(skip_serializing_if = "String::is_empty")]
        source_trace: String,
    }
    let bytes = serde_json::to_vec(&Projection {
        sha256: &candidate.sha256,
        parent_sha256: &candidate.parent_sha256,
        source_trace_sha256: &candidate.source_trace_sha256,
        forced_depth: candidate.forced_depth,
        prefix_sha256: &candidate.prefix_sha256,
        prefix_bytes: super::base64(&candidate.prefix_bytes),
        source_trace: super::base64(&candidate.source_trace),
    })
    .map_err(|error| error.to_string())?;
    (bytes.len() as u64)
        .checked_add(1)
        .ok_or("choice candidate byte overflow".into())
}

fn queue_bytes(candidates: &[ChoiceCandidate]) -> Result<u64, String> {
    candidates.iter().try_fold(0_u64, |total, candidate| {
        total
            .checked_add(candidate_bytes(candidate)?)
            .ok_or("choice exploration byte accounting overflow".into())
    })
}
