use super::simulation::{
    self, Decision, Dimension, SimulationCandidate, SimulationConfig, SimulationResult,
};
use super::simulation_record_wire::{self as wire, MAXIMUM_PLAN_BYTES, PLAN_SCHEMA};
use crate::evidence::{
    canonical,
    inspection::{Payload, Simulation},
};
use gomad_wasmhost::linked;
use serde::Serialize;
use serde_json::value::RawValue;
use std::collections::{BTreeMap, BTreeSet};

pub const RECORD_SCHEMA: &str = "gomad.cluster-record/v8";
pub const MAXIMUM_RECORD_BYTES: usize = 128 << 20;

pub fn runtime_decisions(
    trace: &[u8],
    identity: &linked::ChoiceIdentity,
) -> Result<Vec<Decision>, String> {
    let tape = linked::project_choices(trace, identity).map_err(str::to_owned)?;
    let mut decisions = Vec::with_capacity(tape.decisions.len());
    for runtime in &tape.decisions {
        if runtime.alternatives() < 2 {
            continue;
        }
        let (site, alternatives) = runtime_decision_identities(runtime)?;
        let mut controls = Vec::with_capacity(runtime.alternatives() as usize);
        for rank in 0..runtime.alternatives() {
            if rank == runtime.selected() {
                controls.push(Vec::new());
                continue;
            }
            let prefix = linked::project_choices(trace, identity)
                .map_err(str::to_owned)?
                .prefix(&linked::PrefixControl {
                    cutoff: runtime
                        .ordinal()
                        .checked_add(1)
                        .ok_or("runtime choice ordinal overflow")?,
                    r#override: Some(linked::RankOverride {
                        ordinal: runtime.ordinal(),
                        rank,
                        reconstruct: false,
                    }),
                })
                .map_err(str::to_owned)?;
            controls.push(prefix.bytes);
        }
        decisions.push(simulation::canonical_decision(
            Dimension::Runtime,
            runtime.ordinal(),
            site,
            alternatives,
            controls,
            runtime.selected(),
        )?);
    }
    Ok(decisions)
}

pub(super) fn runtime_decision_identities(
    runtime: &linked::Decision,
) -> Result<(String, Vec<String>), String> {
    #[derive(Serialize)]
    struct Site<'a> {
        kind: u8,
        site_offset: String,
        site_missing: bool,
        data: u32,
        alternatives: u32,
        alternative_set_sha256: &'a str,
    }
    let mut actual_digest = String::from("sha256:");
    use std::fmt::Write as _;
    for byte in runtime.alternative_set_digest() {
        write!(actual_digest, "{byte:02x}").map_err(|e| e.to_string())?;
    }
    let bytes = serde_json::to_vec(&Site {
        kind: runtime.kind(),
        site_offset: runtime.site().0.to_string(),
        site_missing: runtime.site().1,
        data: runtime.data(),
        alternatives: runtime.alternatives(),
        alternative_set_sha256: &actual_digest,
    })
    .map_err(|e| e.to_string())?;
    let site = wire::domain_hash("gomad-runtime-choice-site/v1", &bytes);
    #[derive(Serialize)]
    struct Alternative<'a> {
        site_sha256: &'a str,
        rank: u32,
    }
    let alternatives = (0..runtime.alternatives())
        .map(|rank| {
            let bytes = serde_json::to_vec(&Alternative {
                site_sha256: &site,
                rank,
            })
            .map_err(|e| e.to_string())?;
            Ok(wire::domain_hash(
                "gomad-runtime-choice-alternative/v1",
                &bytes,
            ))
        })
        .collect::<Result<_, String>>()?;
    Ok((site, alternatives))
}

pub fn result_for_record(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
    record: &[u8],
    runtime: &[Decision],
) -> Result<SimulationResult, String> {
    simulation::validate_candidate(config, candidate)?;
    let plan = wire::plan_for_candidate(config, candidate)?;
    let projected = project_retained_record(record, &plan)?;
    if projected.seed != config.base_seed {
        return Err("simulation exploration record seed does not match its candidate".into());
    }
    let mut decisions = runtime.to_vec();
    decisions.extend(projected.decisions);
    validate_decisions(&decisions)?;
    let result = SimulationResult {
        candidate_sha256: candidate.sha256.clone(),
        outcome_sha256: projected.outcome_sha256,
        failed: projected.failed,
        failure_sha256: projected.failure_sha256,
        diverged: projected.diverged,
        decisions,
    };
    simulation::validate_result(config, candidate, &result)?;
    Ok(result)
}

pub fn project_artifact(
    config: &SimulationConfig,
    candidate: &SimulationCandidate,
    plan: &[u8],
    record: &[u8],
    runtime: &[Decision],
    record_limit: u64,
) -> Result<Simulation, String> {
    if wire::plan_for_candidate(config, candidate)? != plan {
        return Err("simulation exploration artifact plan does not match its candidate".into());
    }
    let result = result_for_record(config, candidate, record, runtime)?;
    let profile = Simulation {
        profile: "gomad-simulation-exploration/v1".into(),
        controller_sha256: config.controller_sha256.clone(),
        execution_sha256: config.execution_sha256.clone(),
        candidate_sha256: candidate.sha256.clone(),
        outcome_sha256: result.outcome_sha256,
        failure_sha256: (!result.failure_sha256.is_empty()).then_some(result.failure_sha256),
        plan: Payload {
            schema: PLAN_SCHEMA.into(),
            file: "simulation/plan.json".into(),
            sha256: canonical::digest(plan),
            bytes: plan.len() as u64,
            limit: MAXIMUM_PLAN_BYTES as u64,
        },
        record: Payload {
            schema: RECORD_SCHEMA.into(),
            file: "simulation/record.json".into(),
            sha256: canonical::digest(record),
            bytes: record.len() as u64,
            limit: record_limit,
        },
    };
    validate_artifact(&profile, plan, record)?;
    Ok(profile)
}

pub fn validate_artifact(profile: &Simulation, plan: &[u8], record: &[u8]) -> Result<(), String> {
    if profile.profile != "gomad-simulation-exploration/v1"
        || profile.plan.schema != PLAN_SCHEMA
        || profile.plan.file != "simulation/plan.json"
        || profile.record.schema != RECORD_SCHEMA
        || profile.record.file != "simulation/record.json"
    {
        return Err("simulation exploration artifact identity is invalid".into());
    }
    for digest in [
        &profile.controller_sha256,
        &profile.execution_sha256,
        &profile.candidate_sha256,
        &profile.outcome_sha256,
        &profile.plan.sha256,
        &profile.record.sha256,
    ] {
        wire::validate_digest(digest)?;
    }
    if let Some(digest) = &profile.failure_sha256 {
        wire::validate_digest(digest)?;
    }
    if plan.is_empty()
        || plan.len() > MAXIMUM_PLAN_BYTES
        || canonical::digest(plan) != profile.plan.sha256
        || plan.len() as u64 != profile.plan.bytes
    {
        return Err("simulation exploration artifact plan identity changed".into());
    }
    if record.is_empty()
        || record.len() > MAXIMUM_RECORD_BYTES
        || profile.record.limit == 0
        || profile.record.limit > MAXIMUM_RECORD_BYTES as u64
        || record.len() as u64 > profile.record.limit
        || canonical::digest(record) != profile.record.sha256
        || record.len() as u64 != profile.record.bytes
    {
        return Err("simulation exploration artifact record identity or bound changed".into());
    }
    let retained = wire::validate_retained_plan(plan)?;
    if retained.controller_sha256 != profile.controller_sha256
        || retained.execution_sha256 != profile.execution_sha256
    {
        return Err("simulation exploration artifact plan execution identity changed".into());
    }
    let projected = project_retained_record(record, plan)?;
    if projected.seed != retained.base_seed
        || projected.outcome_sha256 != profile.outcome_sha256
        || projected.failure_sha256 != profile.failure_sha256.as_deref().unwrap_or("")
    {
        return Err("simulation exploration artifact result identity changed".into());
    }
    Ok(())
}

struct RetainedRecord {
    seed: u64,
    failed: bool,
    diverged: bool,
    outcome_sha256: String,
    failure_sha256: String,
    decisions: Vec<Decision>,
}

fn project_retained_record(record: &[u8], expected_plan: &[u8]) -> Result<RetainedRecord, String> {
    if record.is_empty() || record.len() > MAXIMUM_RECORD_BYTES {
        return Err("simulation exploration record size is invalid".into());
    }
    serde_json::from_slice::<super::super::scenario::UniqueNames>(record)
        .map_err(|e| e.to_string())?;
    let mut fields: BTreeMap<String, Box<RawValue>> =
        serde_json::from_slice(record).map_err(|e| e.to_string())?;
    let string = |name: &str| -> Result<String, String> {
        serde_json::from_str(
            fields
                .get(name)
                .ok_or_else(|| format!("simulation exploration record field {name:?} is missing"))?
                .get(),
        )
        .map_err(|e| e.to_string())
    };
    if string("schema")? != RECORD_SCHEMA {
        return Err("simulation exploration record schema is invalid".into());
    }
    let seed: u64 = serde_json::from_str(
        fields
            .get("seed")
            .ok_or("simulation exploration record seed missing")?
            .get(),
    )
    .map_err(|e| e.to_string())?;
    wire::validate_digest(&string("identity")?)?;
    if fields.get("exploration_plan").map(|v| v.get().as_bytes()) != Some(expected_plan) {
        return Err("simulation exploration record plan does not match its candidate".into());
    }
    let decisions: Vec<Decision> = fields
        .get("exploration_decisions")
        .map(|v| serde_json::from_str(v.get()))
        .transpose()
        .map_err(|e| e.to_string())?
        .unwrap_or_default();
    validate_decisions(&decisions)?;
    let outcome = string("outcome")?;
    let (failed, diverged) = match outcome.as_str() {
        "completed" => (false, false),
        "scenario_failed" | "oracle_failed" => (true, false),
        "replay_diverged" => (false, true),
        _ => {
            return Err(format!(
                "simulation exploration outcome {outcome:?} is invalid"
            ));
        }
    };
    let failure_sha256 = if failed {
        let hash = string("failure_identity")?;
        wire::validate_digest(&hash)?;
        hash
    } else {
        String::new()
    };
    for name in [
        "identity",
        "spec_sha256",
        "exploration_plan",
        "exploration_decisions",
    ] {
        fields.remove(name);
    }
    let bytes = compact_semantic_json(&serde_json::to_vec(&fields).map_err(|e| e.to_string())?);
    let outcome_sha256 = wire::domain_hash("gomad-simulation-exploration-outcome/v1", &bytes);
    Ok(RetainedRecord {
        seed,
        failed,
        diverged,
        outcome_sha256,
        failure_sha256,
        decisions,
    })
}

fn compact_semantic_json(raw: &[u8]) -> Vec<u8> {
    let mut bytes = Vec::with_capacity(raw.len());
    let mut quoted = false;
    let mut escaped = false;
    let mut index = 0;
    while index < raw.len() {
        let byte = raw[index];
        if quoted {
            if !escaped {
                match byte {
                    b'"' => quoted = false,
                    b'\\' => escaped = true,
                    b'<' | b'>' | b'&' => {
                        bytes.extend_from_slice(match byte {
                            b'<' => b"\\u003c",
                            b'>' => b"\\u003e",
                            _ => b"\\u0026",
                        });
                        index += 1;
                        continue;
                    }
                    0xe2 if raw.get(index + 1) == Some(&0x80)
                        && matches!(raw.get(index + 2), Some(0xa8 | 0xa9)) =>
                    {
                        bytes.extend_from_slice(if raw[index + 2] == 0xa8 {
                            b"\\u2028"
                        } else {
                            b"\\u2029"
                        });
                        index += 3;
                        continue;
                    }
                    _ => {}
                }
            } else {
                escaped = false;
            }
        } else if matches!(byte, b' ' | b'\t' | b'\n' | b'\r') {
            index += 1;
            continue;
        } else if byte == b'"' {
            quoted = true;
        }
        bytes.push(byte);
        index += 1;
    }
    bytes
}

fn validate_decisions(decisions: &[Decision]) -> Result<(), String> {
    let mut seen = BTreeSet::new();
    for decision in decisions {
        let controls = if decision.dimension == Dimension::Runtime {
            decision.alternative_controls.clone()
        } else {
            Vec::new()
        };
        let canonical = simulation::canonical_decision(
            decision.dimension,
            decision.ordinal,
            decision.site_sha256.clone(),
            decision.alternatives.clone(),
            controls,
            decision.selected,
        )?;
        if canonical.identity != decision.identity
            || canonical.alternative_set_sha256 != decision.alternative_set_sha256
        {
            return Err(
                "simulation exploration decision identity does not match its contents".into(),
            );
        }
        if !seen.insert((format!("{:?}", decision.dimension), decision.ordinal)) {
            return Err(
                "simulation exploration decisions contain a duplicate dimension ordinal".into(),
            );
        }
    }
    Ok(())
}
