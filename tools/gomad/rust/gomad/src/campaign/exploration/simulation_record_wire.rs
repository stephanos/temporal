use super::simulation::{
    self, Dimension, DimensionLimits, ForcedDecision, SimulationCandidate, SimulationConfig,
};
use super::simulation_record::runtime_decision_identities;
use crate::evidence::{canonical, inspection::Simulation};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

pub const PLAN_SCHEMA: &str = "gomad.simulation-exploration-plan/v1";
pub const MAXIMUM_PLAN_BYTES: usize = 16 << 20;

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Plan {
    pub schema: String,
    pub execution_sha256: String,
    pub controller_sha256: String,
    pub base_seed: u64,
    pub overrides: Option<Vec<RootOverride>>,
    pub candidate_sha256: String,
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct RootOverride {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub site_sha256: String,
    pub alternatives: u32,
    pub alternative_set_sha256: String,
    pub selected: u32,
    pub selected_sha256: String,
    pub identity: String,
}

pub struct CandidateExecution {
    pub simulation_plan: Vec<u8>,
    pub choice_mode: String,
    pub choice_replay_plan: Option<linked::ChoicePlan>,
    pub choice_source_trace: Vec<u8>,
}

pub(super) fn domain_hash(domain: &str, bytes: &[u8]) -> String {
    let mut hash = Sha256::new();
    hash.update(domain.as_bytes());
    hash.update([0]);
    hash.update(bytes);
    format!("sha256:{:x}", hash.finalize())
}

pub(super) fn validate_digest(value: &str) -> Result<(), String> {
    if canonical::valid_digest(value.strip_prefix("sha256:").unwrap_or(value)) {
        Ok(())
    } else {
        Err("invalid SHA-256 identity".into())
    }
}

fn forced_projection(value: &RootOverride, identity: &str) -> Value {
    json!({"alternative_set_sha256":value.alternative_set_sha256,"alternatives":value.alternatives,"dimension":value.dimension,"identity":identity,"ordinal":value.ordinal,"selected":value.selected,"selected_sha256":value.selected_sha256,"site_sha256":value.site_sha256})
}

fn candidate_identity(config: &Plan) -> Result<String, String> {
    let overrides = config.overrides.as_ref().map(|values| {
        values
            .iter()
            .map(|v| forced_projection(v, &v.identity))
            .collect::<Vec<_>>()
    });
    let value = json!({"base_seed":config.base_seed,"controller_sha256":config.controller_sha256,"execution_sha256":config.execution_sha256,"overrides":overrides});
    Ok(domain_hash(
        "gomad-simulation-exploration-candidate/v1",
        &serde_json::to_vec(&value).map_err(|e| e.to_string())?,
    ))
}

pub fn plan_for_candidate(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
) -> Result<Vec<u8>, String> {
    simulation::validate_candidate(config, candidate)?;
    let mut overrides = Vec::with_capacity(candidate.overrides.len());
    for forced in &candidate.overrides {
        let mut root = RootOverride {
            dimension: forced.dimension,
            ordinal: forced.ordinal,
            site_sha256: forced.site_sha256.clone(),
            alternatives: forced.alternatives,
            alternative_set_sha256: forced.alternative_set_sha256.clone(),
            selected: forced.selected,
            selected_sha256: forced.selected_sha256.clone(),
            identity: String::new(),
        };
        root.identity = domain_hash(
            "gomad-simulation-exploration-forced-decision/v1",
            &serde_json::to_vec(&forced_projection(&root, "")).map_err(|e| e.to_string())?,
        );
        overrides.push(root);
    }
    let mut plan = Plan {
        schema: PLAN_SCHEMA.into(),
        execution_sha256: config.execution_sha256.clone(),
        controller_sha256: config.controller_sha256.clone(),
        base_seed: config.base_seed,
        overrides: if candidate.nil_overrides {
            None
        } else {
            Some(overrides)
        },
        candidate_sha256: String::new(),
    };
    plan.candidate_sha256 = candidate_identity(&plan)?;
    let bytes = serde_json::to_vec(&plan).map_err(|e| e.to_string())?;
    if bytes.is_empty() || bytes.len() > MAXIMUM_PLAN_BYTES {
        return Err("simulation exploration plan exceeds its bound".into());
    }
    Ok(bytes)
}

pub fn validate_retained_plan(bytes: &[u8]) -> Result<Plan, String> {
    if bytes.is_empty() || bytes.len() > MAXIMUM_PLAN_BYTES {
        return Err("simulation exploration plan size is invalid".into());
    }
    let plan: Plan = serde_json::from_slice(bytes).map_err(|e| e.to_string())?;
    if serde_json::to_vec(&plan).map_err(|e| e.to_string())? != bytes {
        return Err("simulation exploration plan is not canonical".into());
    }
    if plan.schema != PLAN_SCHEMA {
        return Err("simulation exploration plan schema is invalid".into());
    }
    for hash in [
        &plan.execution_sha256,
        &plan.controller_sha256,
        &plan.candidate_sha256,
    ] {
        validate_digest(hash)?;
    }
    for root in plan.overrides.iter().flatten() {
        if root.alternatives < 2 || root.selected >= root.alternatives {
            return Err("simulation exploration plan override shape is invalid".into());
        }
        for hash in [
            &root.site_sha256,
            &root.alternative_set_sha256,
            &root.selected_sha256,
            &root.identity,
        ] {
            validate_digest(hash)?;
        }
        let hash = domain_hash(
            "gomad-simulation-exploration-forced-decision/v1",
            &serde_json::to_vec(&forced_projection(root, "")).map_err(|e| e.to_string())?,
        );
        if hash != root.identity {
            return Err("simulation exploration plan override identity changed".into());
        }
    }
    if candidate_identity(&plan)? != plan.candidate_sha256 {
        return Err("simulation exploration plan candidate identity changed".into());
    }
    Ok(plan)
}

pub fn candidate_for_artifact(
    profile: &Simulation,
    bytes: &[u8],
    exact_trace: Option<&[u8]>,
    identity: Option<&linked::ChoiceIdentity>,
) -> Result<(SimulationConfig, SimulationCandidate), String> {
    let retained = validate_retained_plan(bytes)?;
    if retained.controller_sha256 != profile.controller_sha256
        || retained.execution_sha256 != profile.execution_sha256
    {
        return Err("simulation exploration artifact execution identity changed".into());
    }
    let mut limits = DimensionLimits {
        runtime: 1,
        scenario: 1,
        network: 1,
        storage: 1,
        fault: 1,
        crash: 1,
    };
    for root in retained.overrides.iter().flatten() {
        let limit = root
            .ordinal
            .checked_add(1)
            .ok_or("simulation override ordinal overflow")?;
        let current = match root.dimension {
            Dimension::Runtime => &mut limits.runtime,
            Dimension::Scenario => &mut limits.scenario,
            Dimension::Network => &mut limits.network,
            Dimension::Storage => &mut limits.storage,
            Dimension::Fault => &mut limits.fault,
            Dimension::Crash => &mut limits.crash,
        };
        *current = (*current).max(limit);
    }
    let config = SimulationConfig {
        execution_sha256: retained.execution_sha256,
        controller_sha256: retained.controller_sha256,
        base_seed: retained.base_seed,
        parallel: 1,
        max_executions: 1,
        max_forced_decisions: retained
            .overrides
            .as_ref()
            .map_or(1, |v| v.len().max(1) as u64),
        max_exploration_bytes: u64::MAX,
        max_result_bytes: u64::MAX,
        failure_budget: 1,
        dimension_limits: limits,
    };
    let nil_overrides = retained.overrides.is_none();
    let mut overrides = Vec::new();
    for root in retained.overrides.into_iter().flatten() {
        let control = if root.dimension == Dimension::Runtime {
            let trace =
                exact_trace.ok_or("runtime simulation override requires an exact choice tape")?;
            let identity =
                identity.ok_or("runtime simulation override requires an exact choice identity")?;
            linked::project_choices(trace, identity)
                .map_err(str::to_owned)?
                .prefix(&linked::PrefixControl {
                    cutoff: root
                        .ordinal
                        .checked_add(1)
                        .ok_or("runtime ordinal overflow")?,
                    r#override: Some(linked::RankOverride {
                        ordinal: root.ordinal,
                        rank: root.selected,
                        reconstruct: true,
                    }),
                })
                .map_err(str::to_owned)?
                .bytes
        } else {
            Vec::new()
        };
        let forced = ForcedDecision {
            dimension: root.dimension,
            ordinal: root.ordinal,
            site_sha256: root.site_sha256,
            alternatives: root.alternatives,
            alternative_set_sha256: root.alternative_set_sha256,
            selected: root.selected,
            selected_sha256: root.selected_sha256,
            control,
            source_trace: if root.dimension == Dimension::Runtime {
                exact_trace.unwrap_or_default().to_vec()
            } else {
                Vec::new()
            },
            identity: String::new(),
        };
        overrides.push(simulation::canonical_forced_decision(forced)?);
    }
    let candidate = simulation::canonical_candidate_collection(
        &config,
        if nil_overrides { None } else { Some(overrides) },
        String::new(),
    )?;
    if candidate.sha256 != profile.candidate_sha256 {
        return Err("reconstructed simulation candidate identity changed".into());
    }
    Ok((config, candidate))
}

pub fn execution_for_candidate(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
    identity: &linked::ChoiceIdentity,
) -> Result<CandidateExecution, String> {
    let mut execution = CandidateExecution {
        simulation_plan: plan_for_candidate(config, candidate)?,
        choice_mode: "record".into(),
        choice_replay_plan: None,
        choice_source_trace: Vec::new(),
    };
    let forced: Vec<_> = candidate
        .overrides
        .iter()
        .filter(|v| v.dimension == Dimension::Runtime)
        .collect();
    let mut plans = Vec::new();
    for value in &forced {
        let trace = &value.source_trace;
        let plan = linked::project_choices(trace, identity).map_err(str::to_owned)?;
        let original = plan
            .decisions
            .get(usize::try_from(value.ordinal).map_err(|_| "runtime ordinal overflow")?)
            .ok_or("runtime control ordinal missing")?;
        let reconstruct = original.selected() == value.selected;
        let prefix = plan
            .prefix(&linked::PrefixControl {
                cutoff: value
                    .ordinal
                    .checked_add(1)
                    .ok_or("runtime ordinal overflow")?,
                r#override: Some(linked::RankOverride {
                    ordinal: value.ordinal,
                    rank: value.selected,
                    reconstruct,
                }),
            })
            .map_err(str::to_owned)?;
        if prefix.bytes != value.control
            || !runtime_override_matches(
                value,
                prefix.decisions.last().ok_or("runtime prefix is empty")?,
            )?
        {
            return Err("runtime control does not match its forced decision".into());
        }
        plans.push(prefix);
    }
    let Some((longest, _)) = plans
        .iter()
        .enumerate()
        .max_by_key(|(_, p)| p.decisions.len())
    else {
        return Ok(execution);
    };
    let selected = plans.swap_remove(longest);
    for value in &forced {
        let decision = selected
            .decisions
            .get(usize::try_from(value.ordinal).map_err(|_| "runtime ordinal overflow")?)
            .ok_or("runtime controls do not compose into one forced prefix")?;
        if !runtime_override_matches(value, decision)? {
            return Err("runtime controls do not compose into one forced prefix".into());
        }
    }
    execution.choice_mode = "prefix".into();
    execution
        .choice_source_trace
        .clone_from(&forced[longest].source_trace);
    execution.choice_replay_plan = Some(selected);
    Ok(execution)
}

fn runtime_override_matches(
    forced: &ForcedDecision,
    decision: &linked::Decision,
) -> Result<bool, String> {
    let (site, alternatives) = runtime_decision_identities(decision)?;
    let set = domain_hash("gomad-simulation-exploration-alternative-set/v1", &serde_json::to_vec(&json!({"alternatives":alternatives,"dimension":"runtime","ordinal":decision.ordinal(),"site_sha256":site})).map_err(|e|e.to_string())?);
    Ok(forced.ordinal == decision.ordinal()
        && forced.site_sha256 == site
        && forced.alternatives as usize == alternatives.len()
        && forced.alternative_set_sha256 == set
        && forced.selected == decision.selected()
        && alternatives.get(decision.selected() as usize) == Some(&forced.selected_sha256))
}
