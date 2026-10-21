use crate::evidence::canonical;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Copy, Debug, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Dimension {
    Runtime,
    Scenario,
    Network,
    Storage,
    Fault,
    Crash,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct DimensionLimits {
    pub runtime: u64,
    pub scenario: u64,
    pub network: u64,
    pub storage: u64,
    pub fault: u64,
    pub crash: u64,
}

impl DimensionLimits {
    pub fn limit(&self, dimension: Dimension) -> u64 {
        match dimension {
            Dimension::Runtime => self.runtime,
            Dimension::Scenario => self.scenario,
            Dimension::Network => self.network,
            Dimension::Storage => self.storage,
            Dimension::Fault => self.fault,
            Dimension::Crash => self.crash,
        }
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationConfig {
    pub execution_sha256: String,
    pub controller_sha256: String,
    pub base_seed: u64,
    pub parallel: usize,
    pub max_executions: u64,
    pub max_forced_decisions: u64,
    pub max_exploration_bytes: u64,
    pub max_result_bytes: u64,
    pub failure_budget: u64,
    pub dimension_limits: DimensionLimits,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Decision {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub site_sha256: String,
    pub alternatives: Vec<String>,
    #[serde(
        default,
        skip_serializing_if = "Vec::is_empty",
        deserialize_with = "decode_controls"
    )]
    pub alternative_controls: Vec<Vec<u8>>,
    #[serde(
        default,
        skip_serializing_if = "Vec::is_empty",
        deserialize_with = "decode_bytes"
    )]
    pub source_trace: Vec<u8>,
    pub alternative_set_sha256: String,
    pub selected: u32,
    pub identity: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ForcedDecision {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub site_sha256: String,
    pub alternatives: u32,
    pub alternative_set_sha256: String,
    pub selected: u32,
    pub selected_sha256: String,
    #[serde(
        default,
        skip_serializing_if = "Vec::is_empty",
        deserialize_with = "decode_bytes"
    )]
    pub control: Vec<u8>,
    #[serde(
        default,
        skip_serializing_if = "Vec::is_empty",
        deserialize_with = "decode_bytes"
    )]
    pub source_trace: Vec<u8>,
    pub identity: String,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct SimulationCandidate {
    pub sha256: String,
    pub parent_sha256: String,
    pub overrides: Vec<ForcedDecision>,
    pub nil_overrides: bool,
}

impl Serialize for SimulationCandidate {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        #[derive(Serialize)]
        struct Wire<'a> {
            sha256: &'a str,
            #[serde(skip_serializing_if = "str::is_empty")]
            parent_sha256: &'a str,
            overrides: Option<&'a [ForcedDecision]>,
        }
        Wire {
            sha256: &self.sha256,
            parent_sha256: &self.parent_sha256,
            overrides: if self.nil_overrides {
                None
            } else {
                Some(&self.overrides)
            },
        }
        .serialize(serializer)
    }
}

impl<'de> Deserialize<'de> for SimulationCandidate {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Wire {
            sha256: String,
            #[serde(default)]
            parent_sha256: String,
            overrides: Option<Vec<ForcedDecision>>,
        }
        let wire = Wire::deserialize(deserializer)?;
        Ok(Self {
            sha256: wire.sha256,
            parent_sha256: wire.parent_sha256,
            nil_overrides: wire.overrides.is_none(),
            overrides: wire.overrides.unwrap_or_default(),
        })
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationResult {
    pub candidate_sha256: String,
    pub outcome_sha256: String,
    pub failed: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub failure_sha256: String,
    pub diverged: bool,
    pub decisions: Vec<Decision>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub(crate) enum SimulationStopReason {
    ExplorationExhausted,
    SimulationDepthComplete,
    DimensionDepthComplete,
    MaxExecutions,
    ExplorationCapacity,
    FailureBudget,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationState {
    pub config: SimulationConfig,
    pub queue: Vec<SimulationCandidate>,
    pub seen: BTreeSet<String>,
    pub outcomes: BTreeSet<String>,
    pub failure_signatures: BTreeSet<String>,
    pub logical_executions: u64,
    pub committed_rounds: u64,
    pub pending_bytes: u64,
    pub deepest_override: u64,
    pub omitted_by_execution_bound: u64,
    pub omitted_by_depth: u64,
    pub omitted_by_dimension: u64,
    pub omitted_by_capacity: u64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub stop_reason: Option<SimulationStopReason>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationRound {
    pub index: u64,
    pub candidates: Vec<SimulationCandidate>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationRoundSegment {
    pub schema: String,
    pub index: u64,
    pub before_sha256: String,
    pub after_sha256: String,
    pub results: Vec<SimulationResult>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub sha256: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub(crate) struct SimulationSummary {
    pub parallel: usize,
    pub max_executions: u64,
    pub max_forced_decisions: u64,
    pub max_exploration_bytes: u64,
    pub max_result_bytes: u64,
    pub failure_budget: u64,
    pub dimension_limits: DimensionLimits,
    pub logical_executions: u64,
    pub committed_rounds: u64,
    pub pending: u64,
    pub pending_bytes: u64,
    pub seen_candidates: u64,
    pub deduplicated_outcomes: u64,
    pub distinct_failures: u64,
    pub deepest_override: u64,
    pub omitted_by_execution_bound: u64,
    pub omitted_by_depth: u64,
    pub omitted_by_dimension: u64,
    pub omitted_by_capacity: u64,
    pub stop_reason: Option<SimulationStopReason>,
    pub bounded_complete: bool,
}

pub(crate) fn domain_hash(domain: &str, value: &[u8]) -> String {
    let mut bytes = Vec::with_capacity(domain.len() + 1 + value.len());
    bytes.extend_from_slice(domain.as_bytes());
    bytes.push(0);
    bytes.extend_from_slice(value);
    format!("sha256:{}", canonical::digest(&bytes))
}

pub(crate) fn identity(domain: &str, value: &impl Serialize) -> Result<String, String> {
    Ok(domain_hash(domain, &canonical::encode(value)?))
}

pub(crate) fn implementation_sha256() -> String {
    domain_hash(
        "gomad-simulation-exploration-controller/v1",
        b"deterministic-rounds/breadth-first-combined-rank-overrides/v2",
    )
}

pub(crate) fn valid_sha256(value: &str) -> bool {
    canonical::valid_digest(value.strip_prefix("sha256:").unwrap_or(value))
}

pub(crate) fn validate_config(config: &SimulationConfig) -> Result<(), String> {
    if config.parallel == 0
        || config.max_executions == 0
        || config.max_exploration_bytes == 0
        || config.max_result_bytes == 0
        || config.failure_budget == 0
    {
        return Err("simulation exploration requires positive parallel, execution, exploration, result, and failure bounds".into());
    }
    for dimension in [
        Dimension::Runtime,
        Dimension::Scenario,
        Dimension::Network,
        Dimension::Storage,
        Dimension::Fault,
        Dimension::Crash,
    ] {
        if config.dimension_limits.limit(dimension) == 0 {
            return Err(format!(
                "simulation exploration {dimension:?} dimension bound is zero"
            ));
        }
    }
    if !valid_sha256(&config.execution_sha256) {
        return Err("simulation exploration execution identity is invalid".into());
    }
    if config.controller_sha256 != implementation_sha256() {
        return Err(
            "simulation exploration controller identity does not match this implementation".into(),
        );
    }
    Ok(())
}

fn set_identity(decision: &Decision) -> Result<String, String> {
    identity(
        "gomad-simulation-exploration-alternative-set/v1",
        &json!({
            "alternatives": decision.alternatives, "dimension": decision.dimension,
            "ordinal": decision.ordinal, "site_sha256": decision.site_sha256,
        }),
    )
}

fn decision_identity(decision: &Decision) -> Result<String, String> {
    let mut projection = json!({
        "alternative_set_sha256": decision.alternative_set_sha256,
        "alternatives": decision.alternatives, "dimension": decision.dimension,
        "identity": "", "ordinal": decision.ordinal, "selected": decision.selected,
        "site_sha256": decision.site_sha256,
    });
    if !decision.alternative_controls.is_empty() {
        projection["alternative_controls"] = Value::Array(
            decision
                .alternative_controls
                .iter()
                .map(|control| {
                    if control.is_empty() {
                        Value::Null
                    } else {
                        Value::String(base64(control))
                    }
                })
                .collect(),
        );
    }
    identity("gomad-simulation-exploration-decision/v1", &projection)
}

fn forced_projection(forced: &ForcedDecision) -> Value {
    let mut value = json!({
        "alternative_set_sha256": forced.alternative_set_sha256,
        "alternatives": forced.alternatives, "dimension": forced.dimension,
        "identity": forced.identity, "ordinal": forced.ordinal,
        "selected": forced.selected, "selected_sha256": forced.selected_sha256,
        "site_sha256": forced.site_sha256,
    });
    if !forced.control.is_empty() {
        value["control"] = base64(&forced.control).into();
    }
    value
}

fn forced_wire(forced: &ForcedDecision) -> Value {
    let mut value = forced_projection(forced);
    if !forced.source_trace.is_empty() {
        value["source_trace"] = base64(&forced.source_trace).into();
    }
    value
}

fn same_forced(left: &ForcedDecision, right: &ForcedDecision) -> bool {
    forced_projection(left) == forced_projection(right)
}

fn candidate_bytes(candidate: &SimulationCandidate) -> Result<u64, String> {
    let mut value = json!({"sha256":candidate.sha256,"overrides":if candidate.nil_overrides { Value::Null } else { Value::Array(candidate.overrides.iter().map(forced_wire).collect()) }});
    if !candidate.parent_sha256.is_empty() {
        value["parent_sha256"] = candidate.parent_sha256.clone().into();
    }
    Ok(canonical::encode(&value)?.len() as u64)
}

fn result_bytes(result: &SimulationResult) -> Result<u64, String> {
    let mut value = serde_json::to_value(result).map_err(|error| error.to_string())?;
    for (value, decision) in value["decisions"]
        .as_array_mut()
        .ok_or("invalid decision projection")?
        .iter_mut()
        .zip(&result.decisions)
    {
        if !decision.alternative_controls.is_empty() {
            value["alternative_controls"] = Value::Array(
                decision
                    .alternative_controls
                    .iter()
                    .map(|bytes| {
                        if bytes.is_empty() {
                            Value::Null
                        } else {
                            base64(bytes).into()
                        }
                    })
                    .collect(),
            );
        }
        if !decision.source_trace.is_empty() {
            value["source_trace"] = base64(&decision.source_trace).into();
        }
    }
    Ok(canonical::encode(&value)?.len() as u64)
}

fn forced_identity(forced: &ForcedDecision) -> Result<String, String> {
    let mut value = forced_projection(forced);
    value["identity"] = "".into();
    identity("gomad-simulation-exploration-forced-decision/v1", &value)
}

pub(crate) fn base64(bytes: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut encoded = String::new();
    for chunk in bytes.chunks(3) {
        let value = (u32::from(chunk[0]) << 16)
            | (u32::from(*chunk.get(1).unwrap_or(&0)) << 8)
            | u32::from(*chunk.get(2).unwrap_or(&0));
        encoded.push(char::from(ALPHABET[((value >> 18) & 63) as usize]));
        encoded.push(char::from(ALPHABET[((value >> 12) & 63) as usize]));
        encoded.push(if chunk.len() > 1 {
            char::from(ALPHABET[((value >> 6) & 63) as usize])
        } else {
            '='
        });
        encoded.push(if chunk.len() > 2 {
            char::from(ALPHABET[(value & 63) as usize])
        } else {
            '='
        });
    }
    encoded
}

fn bytes_value<E: serde::de::Error>(value: Value) -> Result<Vec<u8>, E> {
    match value {
        Value::Null => Ok(Vec::new()),
        Value::Array(bytes) => bytes
            .into_iter()
            .map(|byte| {
                byte.as_u64()
                    .and_then(|byte| u8::try_from(byte).ok())
                    .ok_or_else(|| E::custom("invalid simulation control byte"))
            })
            .collect(),
        Value::String(value) => {
            if value.len() % 4 != 0 {
                return Err(E::custom("invalid simulation control base64 length"));
            }
            let mut decoded = Vec::new();
            for (index, chunk) in value.as_bytes().chunks_exact(4).enumerate() {
                let mut digits = [0u8; 4];
                let mut padding = 0;
                for (offset, byte) in chunk.iter().enumerate() {
                    digits[offset] = match byte {
                        b'A'..=b'Z' => byte - b'A',
                        b'a'..=b'z' => byte - b'a' + 26,
                        b'0'..=b'9' => byte - b'0' + 52,
                        b'+' => 62,
                        b'/' => 63,
                        b'=' if offset >= 2 && index + 1 == value.len() / 4 => {
                            padding += 1;
                            0
                        }
                        _ => return Err(E::custom("invalid simulation control base64")),
                    };
                    if padding != 0 && *byte != b'=' {
                        return Err(E::custom("invalid simulation control base64 padding"));
                    }
                }
                decoded.push(digits[0] << 2 | digits[1] >> 4);
                if padding < 2 {
                    decoded.push(digits[1] << 4 | digits[2] >> 2);
                }
                if padding == 0 {
                    decoded.push(digits[2] << 6 | digits[3]);
                }
            }
            if base64(&decoded) != value {
                return Err(E::custom("noncanonical simulation control base64"));
            }
            Ok(decoded)
        }
        _ => Err(E::custom("invalid simulation control bytes")),
    }
}

fn decode_bytes<'de, D: serde::Deserializer<'de>>(deserializer: D) -> Result<Vec<u8>, D::Error> {
    bytes_value(Value::deserialize(deserializer)?)
}

fn decode_controls<'de, D: serde::Deserializer<'de>>(
    deserializer: D,
) -> Result<Vec<Vec<u8>>, D::Error> {
    let values: Option<Vec<Value>> = Option::deserialize(deserializer)?;
    values
        .unwrap_or_default()
        .into_iter()
        .map(bytes_value)
        .collect()
}

pub(crate) fn canonical_decision(
    dimension: Dimension,
    ordinal: u64,
    site_sha256: String,
    alternatives: Vec<String>,
    alternative_controls: Vec<Vec<u8>>,
    selected: u32,
) -> Result<Decision, String> {
    let mut decision = Decision {
        dimension,
        ordinal,
        site_sha256,
        alternatives,
        alternative_controls,
        source_trace: Vec::new(),
        alternative_set_sha256: String::new(),
        selected,
        identity: String::new(),
    };
    decision.alternative_set_sha256 = set_identity(&decision)?;
    decision.identity = decision_identity(&decision)?;
    validate_decision(&decision)?;
    Ok(decision)
}

pub(crate) fn validate_decision(decision: &Decision) -> Result<(), String> {
    if !valid_sha256(&decision.site_sha256)
        || decision.alternatives.len() < 2
        || decision.alternatives.len() > u32::MAX as usize
        || decision.selected as usize >= decision.alternatives.len()
    {
        return Err("simulation exploration decision shape is invalid".into());
    }
    let mut alternatives = BTreeSet::new();
    for alternative in &decision.alternatives {
        if !valid_sha256(alternative) || !alternatives.insert(alternative) {
            return Err(
                "simulation exploration decision alternatives are invalid or duplicated".into(),
            );
        }
    }
    if decision.dimension == Dimension::Runtime {
        if decision.alternative_controls.len() != decision.alternatives.len()
            || decision
                .alternative_controls
                .iter()
                .enumerate()
                .any(|(rank, control)| control.is_empty() != (rank == decision.selected as usize))
        {
            return Err("runtime decision controls do not match its alternatives".into());
        }
    } else if !decision.alternative_controls.is_empty() {
        return Err("non-runtime decision contains runtime controls".into());
    }
    if set_identity(decision)? != decision.alternative_set_sha256
        || decision_identity(decision)? != decision.identity
    {
        return Err("simulation exploration decision identity does not match".into());
    }
    Ok(())
}

pub(crate) fn canonical_forced_decision(
    mut forced: ForcedDecision,
) -> Result<ForcedDecision, String> {
    forced.identity = forced_identity(&forced)?;
    validate_forced_decision(&forced)?;
    Ok(forced)
}

pub(crate) fn validate_forced_decision(forced: &ForcedDecision) -> Result<(), String> {
    if forced.alternatives < 2
        || forced.selected >= forced.alternatives
        || [
            &forced.site_sha256,
            &forced.alternative_set_sha256,
            &forced.selected_sha256,
            &forced.identity,
        ]
        .iter()
        .any(|value| !valid_sha256(value))
    {
        return Err("forced decision shape is invalid".into());
    }
    if (forced.dimension == Dimension::Runtime) == forced.control.is_empty() {
        return Err("forced decision runtime control is missing or unexpected".into());
    }
    if forced_identity(forced)? != forced.identity {
        return Err("forced decision identity does not match".into());
    }
    Ok(())
}

pub(crate) fn force_decision(decision: &Decision, selected: u32) -> Result<ForcedDecision, String> {
    validate_decision(decision)?;
    let selected_sha256 = decision
        .alternatives
        .get(selected as usize)
        .ok_or("forced decision selected rank is invalid")?
        .clone();
    canonical_forced_decision(ForcedDecision {
        dimension: decision.dimension,
        ordinal: decision.ordinal,
        site_sha256: decision.site_sha256.clone(),
        alternatives: decision.alternatives.len() as u32,
        alternative_set_sha256: decision.alternative_set_sha256.clone(),
        selected,
        selected_sha256,
        control: if decision.dimension == Dimension::Runtime {
            decision.alternative_controls[selected as usize].clone()
        } else {
            Vec::new()
        },
        source_trace: if decision.dimension == Dimension::Runtime {
            decision.source_trace.clone()
        } else {
            Vec::new()
        },
        identity: String::new(),
    })
}

pub(crate) fn canonical_candidate(
    config: &SimulationConfig,
    overrides: Vec<ForcedDecision>,
    parent_sha256: String,
) -> Result<SimulationCandidate, String> {
    new_candidate(config, overrides, parent_sha256, false)
}

pub(crate) fn canonical_candidate_collection(
    config: &SimulationConfig,
    overrides: Option<Vec<ForcedDecision>>,
    parent_sha256: String,
) -> Result<SimulationCandidate, String> {
    let nil = overrides.is_none();
    new_candidate(config, overrides.unwrap_or_default(), parent_sha256, nil)
}

fn new_candidate(
    config: &SimulationConfig,
    mut overrides: Vec<ForcedDecision>,
    parent_sha256: String,
    nil_overrides: bool,
) -> Result<SimulationCandidate, String> {
    validate_config(config)?;
    if nil_overrides && !overrides.is_empty() {
        return Err("nil simulation override collection contains decisions".into());
    }
    overrides.sort_by_key(|forced| (forced.dimension, forced.ordinal));
    for (index, forced) in overrides.iter().enumerate() {
        validate_forced_decision(forced)?;
        if index > 0
            && (overrides[index - 1].dimension, overrides[index - 1].ordinal)
                == (forced.dimension, forced.ordinal)
        {
            return Err("candidate contains duplicate forced decisions".into());
        }
    }
    if !parent_sha256.is_empty() && !valid_sha256(&parent_sha256) {
        return Err("candidate parent identity is invalid".into());
    }
    let projection = if nil_overrides {
        Value::Null
    } else {
        Value::Array(overrides.iter().map(forced_projection).collect())
    };
    let sha256 = identity(
        "gomad-simulation-exploration-candidate/v1",
        &json!({
            "base_seed": config.base_seed, "controller_sha256": config.controller_sha256,
            "execution_sha256": config.execution_sha256, "overrides": projection,
        }),
    )?;
    Ok(SimulationCandidate {
        sha256,
        parent_sha256,
        overrides,
        nil_overrides,
    })
}

pub(crate) fn validate_candidate(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
) -> Result<(), String> {
    if new_candidate(
        config,
        candidate.overrides.clone(),
        candidate.parent_sha256.clone(),
        candidate.nil_overrides,
    )? != *candidate
    {
        return Err("simulation exploration candidate identity does not match its contents".into());
    }
    Ok(())
}

fn forced_matches(forced: &ForcedDecision, decision: &Decision) -> bool {
    forced.dimension == decision.dimension
        && forced.ordinal == decision.ordinal
        && forced.site_sha256 == decision.site_sha256
        && forced.alternatives as usize == decision.alternatives.len()
        && forced.alternative_set_sha256 == decision.alternative_set_sha256
        && forced.selected == decision.selected
        && decision.alternatives.get(decision.selected as usize) == Some(&forced.selected_sha256)
}

fn canonical_result(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
    result: &SimulationResult,
) -> Result<SimulationResult, String> {
    if result.candidate_sha256 != candidate.sha256
        || !valid_sha256(&result.outcome_sha256)
        || (result.failed && !valid_sha256(&result.failure_sha256))
        || (!result.failed && !result.failure_sha256.is_empty())
    {
        return Err("simulation exploration result identity does not match".into());
    }
    let mut result = result.clone();
    result
        .decisions
        .sort_by_key(|decision| (decision.dimension, decision.ordinal));
    for (index, decision) in result.decisions.iter().enumerate() {
        validate_decision(decision)?;
        if index > 0
            && (
                result.decisions[index - 1].dimension,
                result.decisions[index - 1].ordinal,
            ) == (decision.dimension, decision.ordinal)
        {
            return Err("decisions contain a duplicate dimension ordinal".into());
        }
    }
    if !result.diverged
        && candidate.overrides.iter().any(|forced| {
            !result
                .decisions
                .iter()
                .any(|decision| forced_matches(forced, decision))
        })
    {
        return Err("result does not prove a forced decision".into());
    }
    if result_bytes(&result)? > config.max_result_bytes {
        return Err("simulation exploration result exceeds its byte bound".into());
    }
    Ok(result)
}

pub(crate) fn validate_result(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
    result: &SimulationResult,
) -> Result<(), String> {
    canonical_result(config, candidate, result).map(|_| ())
}

fn sort_candidates(candidates: &mut [SimulationCandidate]) {
    candidates.sort_by(|left, right| {
        left.overrides
            .len()
            .cmp(&right.overrides.len())
            .then_with(|| left.sha256.cmp(&right.sha256))
    });
}

fn queue_bytes(queue: &[SimulationCandidate]) -> Result<u64, String> {
    queue.iter().try_fold(0u64, |total, candidate| {
        total
            .checked_add(candidate_bytes(candidate)?)
            .ok_or_else(|| "simulation exploration byte accounting overflow".into())
    })
}

impl SimulationState {
    pub fn new(config: SimulationConfig) -> Result<Self, String> {
        validate_config(&config)?;
        let root = new_candidate(&config, Vec::new(), String::new(), true)?;
        let pending_bytes = queue_bytes(std::slice::from_ref(&root))?;
        if pending_bytes > config.max_exploration_bytes {
            return Err("simulation exploration root exceeds exploration byte bound".into());
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
            deepest_override: 0,
            omitted_by_execution_bound: 0,
            omitted_by_depth: 0,
            omitted_by_dimension: 0,
            omitted_by_capacity: 0,
            stop_reason: None,
        })
    }

    pub fn hash(&self) -> Result<String, String> {
        identity("gomad-simulation-exploration-state/v1", self)
    }

    pub fn next_round(&self) -> Option<SimulationRound> {
        if self.stop_reason.is_some() || self.queue.is_empty() {
            return None;
        }
        Some(SimulationRound {
            index: self.committed_rounds,
            candidates: self.queue[..self.queue.len().min(self.config.parallel)].to_vec(),
        })
    }

    pub fn summary(&self) -> SimulationSummary {
        SimulationSummary {
            parallel: self.config.parallel,
            max_executions: self.config.max_executions,
            max_forced_decisions: self.config.max_forced_decisions,
            max_exploration_bytes: self.config.max_exploration_bytes,
            max_result_bytes: self.config.max_result_bytes,
            failure_budget: self.config.failure_budget,
            dimension_limits: self.config.dimension_limits.clone(),
            logical_executions: self.logical_executions,
            committed_rounds: self.committed_rounds,
            pending: self.queue.len() as u64,
            pending_bytes: self.pending_bytes,
            seen_candidates: self.seen.len() as u64,
            deduplicated_outcomes: self.outcomes.len() as u64,
            distinct_failures: self.failure_signatures.len() as u64,
            deepest_override: self.deepest_override,
            omitted_by_execution_bound: self.omitted_by_execution_bound,
            omitted_by_depth: self.omitted_by_depth,
            omitted_by_dimension: self.omitted_by_dimension,
            omitted_by_capacity: self.omitted_by_capacity,
            stop_reason: self.stop_reason.clone(),
            bounded_complete: matches!(
                self.stop_reason,
                Some(
                    SimulationStopReason::ExplorationExhausted
                        | SimulationStopReason::SimulationDepthComplete
                        | SimulationStopReason::DimensionDepthComplete
                )
            ),
        }
    }

    pub fn commit(
        &self,
        round: &SimulationRound,
        results: &[SimulationResult],
    ) -> Result<(Self, SimulationRoundSegment), String> {
        if self.next_round().as_ref() != Some(round) {
            return Err("simulation exploration round does not match the current state".into());
        }
        if results.len() != round.candidates.len() {
            return Err("simulation exploration result count does not match its round".into());
        }
        let mut next = self.clone();
        next.queue.drain(..round.candidates.len());
        next.logical_executions = next
            .logical_executions
            .checked_add(results.len() as u64)
            .ok_or("simulation execution count overflow")?;
        next.committed_rounds = next
            .committed_rounds
            .checked_add(1)
            .ok_or("simulation round count overflow")?;
        let mut children = BTreeMap::<String, SimulationCandidate>::new();
        let mut canonical_results = Vec::with_capacity(results.len());
        for (candidate, result) in round.candidates.iter().zip(results) {
            let result = canonical_result(&self.config, candidate, result)?;
            next.outcomes.insert(result.outcome_sha256.clone());
            if result.failed {
                next.failure_signatures
                    .insert(result.failure_sha256.clone());
                if next.failure_signatures.len() as u64 >= next.config.failure_budget {
                    next.stop_reason = Some(SimulationStopReason::FailureBudget);
                }
            }
            if next.stop_reason.is_none() && !result.diverged {
                next.expand(candidate, &result.decisions, &mut children)?;
            }
            canonical_results.push(result);
        }
        let mut children = children.into_values().collect::<Vec<_>>();
        sort_candidates(&mut children);
        next.admit(&children)?;
        if next.stop_reason.is_none() && next.queue.is_empty() {
            next.stop_reason = Some(
                if next.omitted_by_execution_bound != 0
                    || next.logical_executions >= next.config.max_executions
                {
                    SimulationStopReason::MaxExecutions
                } else if next.omitted_by_depth != 0 {
                    SimulationStopReason::SimulationDepthComplete
                } else if next.omitted_by_dimension != 0 {
                    SimulationStopReason::DimensionDepthComplete
                } else {
                    SimulationStopReason::ExplorationExhausted
                },
            );
        }
        next.pending_bytes = queue_bytes(&next.queue)?;
        let mut segment = SimulationRoundSegment {
            schema: "gomad.simulation-exploration-round/v2".into(),
            index: round.index,
            before_sha256: self.hash()?,
            after_sha256: next.hash()?,
            results: canonical_results,
            sha256: String::new(),
        };
        segment.sha256 = segment.hash()?;
        Ok((next, segment))
    }

    fn expand(
        &mut self,
        parent: &SimulationCandidate,
        decisions: &[Decision],
        children: &mut BTreeMap<String, SimulationCandidate>,
    ) -> Result<(), String> {
        for decision in decisions {
            if parent.overrides.iter().any(|forced| {
                (forced.dimension, forced.ordinal) == (decision.dimension, decision.ordinal)
            }) {
                continue;
            }
            let alternatives = (decision.alternatives.len() - 1) as u64;
            if decision.ordinal >= self.config.dimension_limits.limit(decision.dimension) {
                self.omitted_by_dimension = self
                    .omitted_by_dimension
                    .checked_add(alternatives)
                    .ok_or("simulation dimension omission overflow")?;
                continue;
            }
            if parent.overrides.len() as u64 >= self.config.max_forced_decisions {
                self.omitted_by_depth = self
                    .omitted_by_depth
                    .checked_add(alternatives)
                    .ok_or("simulation depth omission overflow")?;
                continue;
            }
            for rank in 0..decision.alternatives.len() {
                if rank == decision.selected as usize {
                    continue;
                }
                let mut overrides = parent.overrides.clone();
                overrides.push(force_decision(decision, rank as u32)?);
                let child = canonical_candidate(&self.config, overrides, parent.sha256.clone())?;
                if self.seen.contains(&child.sha256) {
                    continue;
                }
                if let Some(existing) = children.get(&child.sha256) {
                    if existing.overrides.len() != child.overrides.len()
                        || !existing
                            .overrides
                            .iter()
                            .zip(&child.overrides)
                            .all(|(left, right)| same_forced(left, right))
                    {
                        return Err("simulation exploration candidate identity collision".into());
                    }
                    if existing.parent_sha256 <= child.parent_sha256 {
                        continue;
                    }
                }
                children.insert(child.sha256.clone(), child);
            }
        }
        Ok(())
    }

    fn admit(&mut self, children: &[SimulationCandidate]) -> Result<(), String> {
        let mut pending_bytes = queue_bytes(&self.queue)?;
        for (index, child) in children.iter().enumerate() {
            let remaining = (children.len() - index) as u64;
            if self.seen.len() as u64 >= self.config.max_executions {
                self.omitted_by_execution_bound = self
                    .omitted_by_execution_bound
                    .checked_add(remaining)
                    .ok_or("simulation execution omission overflow")?;
                break;
            }
            let bytes = candidate_bytes(child)?;
            let total = pending_bytes
                .checked_add(bytes)
                .ok_or("simulation exploration byte accounting overflow")?;
            if total > self.config.max_exploration_bytes {
                self.omitted_by_capacity = self
                    .omitted_by_capacity
                    .checked_add(remaining)
                    .ok_or("simulation capacity omission overflow")?;
                if self.stop_reason.is_none() {
                    self.stop_reason = Some(SimulationStopReason::ExplorationCapacity);
                }
                break;
            }
            pending_bytes = total;
            self.deepest_override = self.deepest_override.max(child.overrides.len() as u64);
            self.seen.insert(child.sha256.clone());
            self.queue.push(child.clone());
        }
        sort_candidates(&mut self.queue);
        self.pending_bytes = pending_bytes;
        Ok(())
    }

    pub fn replay_segment(&self, segment: &SimulationRoundSegment) -> Result<Self, String> {
        if segment.hash()? != segment.sha256
            || self.hash()? != segment.before_sha256
            || segment.schema != "gomad.simulation-exploration-round/v2"
        {
            return Err(
                "simulation exploration segment does not authenticate or link to its state".into(),
            );
        }
        let round = self
            .next_round()
            .ok_or("simulation exploration segment round is unavailable")?;
        if round.index != segment.index {
            return Err("simulation exploration segment round is unavailable".into());
        }
        let (next, regenerated) = self.commit(&round, &segment.results)?;
        if regenerated != *segment {
            return Err(
                "simulation exploration segment replay changed its canonical result".into(),
            );
        }
        Ok(next)
    }
}

impl SimulationRoundSegment {
    pub fn hash(&self) -> Result<String, String> {
        let mut segment = self.clone();
        segment.sha256.clear();
        identity("gomad-simulation-exploration-round-segment/v1", &segment)
    }
}
