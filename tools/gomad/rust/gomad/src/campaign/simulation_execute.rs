use super::{execution, ledger, prepare::Prepared, scenario};
use crate::evidence::{canonical, record::Stream};
use gomad_wasmhost::{linked, linked_cluster, linked_simulation};
use serde::Serialize;
use serde_json::{Value, json, value::RawValue};
use std::time::Duration;

#[derive(Serialize)]
struct Observation {
    schema: &'static str,
    producer: &'static str,
    seed: u64,
    identity: String,
    outcome: &'static str,
    #[serde(skip_serializing_if = "String::is_empty")]
    failure_identity: String,
    exploration_plan: Box<RawValue>,
    exploration_decisions: Vec<linked_simulation::Decision>,
    #[serde(skip_serializing_if = "Option::is_none")]
    divergence: Option<linked_simulation::Divergence>,
    observations: Value,
}

#[expect(
    clippy::too_many_lines,
    reason = "The producer binds observed model decisions and their raw execution evidence before publication"
)]
pub(super) fn run(
    prepared: &Prepared,
    job: &ledger::Job,
    remaining: Duration,
    cancellation: &linked::Cancellation,
    replay: Option<&linked::ReplayRequest>,
    raw_trace: Option<&[u8]>,
    simulation_plan: &[u8],
) -> Result<execution::Result, String> {
    let plan = linked_simulation::Plan::decode(simulation_plan)?;
    if plan.base_seed != job.seed {
        return Err("simulation plan seed differs from execution".into());
    }
    let (mut result, decisions, divergence, observations) = if let Some(raw) = &prepared.scenario {
        let mut bound = scenario::bind(
            raw,
            &prepared.configuration,
            &prepared.module,
            prepared
                .choice_identity
                .as_ref()
                .ok_or("missing simulation choice identity")?,
            prepared.network.as_ref(),
            job.seed,
        )?;
        scenario::simulation_crash_bounds(raw, &mut bound)?;
        bound.request.wall_timeout = bound.request.wall_timeout.min(remaining);
        if let Some(replay) = replay {
            let primary = bound
                .request
                .incarnations
                .first_mut()
                .ok_or("simulation runtime replay has no incarnation")?;
            let model = primary
                .request
                .model
                .as_mut()
                .ok_or("simulation runtime replay has no cooperative model")?;
            model.configuration.mode = if replay.prefix_control.is_some() {
                3
            } else {
                2
            };
            model.configuration.replay = Some(replay.clone());
            model.replay_trace = raw_trace.map(<[u8]>::to_vec);
        }
        let mut admitted = linked_cluster::prepare(bound.request)
            .map_err(|error| format!("runner_failure: {error}"))?;
        admitted
            .compile_cache(prepared.compile_cache_root.as_deref())
            .map_err(|error| format!("runner_failure: admit compile cache: {error}"))?;
        admitted.conditions(linked::Conditions::compiled());
        admitted.cancellation(cancellation.clone());
        #[cfg(feature = "determinism-fixture")]
        crate::fixture::execution_started().map_err(|error| error.message)?;
        let observed = crate::backstop::within(admitted.wall_timeout(), || {
            linked_simulation::execute(admitted, plan, 65536)
        })
        .map_err(|error| format!("runner_failure: {error}"))?;
        let primary = observed.cluster.incarnations.first();
        let trace = primary
            .and_then(|item| item.execution.evidence.get("choice"))
            .cloned()
            .unwrap_or_default();
        let runtime_divergence = primary
            .and_then(|item| item.execution.outcome.divergence.as_deref())
            .cloned();
        let observations = json!({
            "world_sha256":canonical::digest(&observed.cluster.world),
            "network_sha256":canonical::digest(&observed.cluster.network),
            "inspection_sha256":canonical::digest(&observed.cluster.inspection),
            "incarnations":observed.cluster.incarnations.iter().map(|item| json!({
                "handle":item.handle,"interrupted":item.interrupted,
                "termination":item.execution.outcome.termination,"exit_code":item.execution.outcome.exit_code
            })).collect::<Vec<_>>()
        });
        let mut result =
            scenario::publication(prepared, job, bound.configuration, observed.cluster, true)?;
        result.execution.divergence = runtime_divergence;
        payload(&mut result, "evidence/choice", trace);
        (
            result,
            observed.decisions,
            observed.divergence,
            observations,
        )
    } else {
        if plan
            .overrides
            .iter()
            .flatten()
            .any(|value| value.dimension != linked_simulation::Dimension::Runtime)
        {
            return Err("non-runtime simulation override requires an executed scenario".into());
        }
        let result =
            execution::run_with_replay(prepared, job, remaining, cancellation, replay, raw_trace)?;
        let observations = json!({"execution_record_sha256":result.execution.record.hash()?});
        (result, Vec::new(), None, observations)
    };
    let diverged = divergence.is_some() || result.execution.divergence.is_some();
    if let Some(divergence) = &result.execution.divergence {
        result.execution.record.outcome.class = "target_failure".into();
        result.execution.record.outcome.termination = "divergence".into();
        result.execution.record.outcome.reason = divergence.reason_name().into();
        result.publication.record = result.execution.record.clone();
    } else if diverged && result.execution.record.outcome.class == "runner_failure" {
        result.execution.record.outcome.class = "target_failure".into();
        result.execution.record.outcome.reason = "simulation_replay_divergence".into();
        result.publication.record = result.execution.record.clone();
    }
    if result.execution.record.outcome.class == "runner_failure" {
        return Err(format!(
            "runner_failure: {}",
            result.execution.record.outcome.reason
        ));
    }
    let failed = matches!(
        result.execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    );
    let behavioral_record = result.execution.record.bytes()?;
    let mut observation = Observation {
        schema: "gomad.cluster-record/v8",
        producer: "gomad.rust-live-models/v1",
        seed: job.seed,
        identity: String::new(),
        outcome: if diverged {
            "replay_diverged"
        } else if failed {
            "scenario_failed"
        } else {
            "completed"
        },
        failure_identity: if failed && !diverged {
            format!("sha256:{}", result.execution.record.failure_signature()?)
        } else {
            String::new()
        },
        exploration_plan: RawValue::from_string(
            String::from_utf8(simulation_plan.to_vec()).map_err(|error| error.to_string())?,
        )
        .map_err(|error| error.to_string())?,
        exploration_decisions: decisions,
        divergence,
        observations,
    };
    observation.identity = super::exploration::domain_hash(
        "gomad-rust-simulation-observation/v1",
        &serde_json::to_vec(&observation).map_err(|error| error.to_string())?,
    );
    payload(
        &mut result,
        "simulation/record.json",
        serde_json::to_vec(&observation).map_err(|error| error.to_string())?,
    );
    payload(
        &mut result,
        "simulation/behavioral-record.json",
        behavioral_record,
    );
    let record = result.publication.record.bytes()?;
    payload(&mut result, "record.json", record);
    Ok(result)
}

fn payload(result: &mut execution::Result, name: &str, bytes: Vec<u8>) {
    if name != "record.json" {
        let stream = Stream {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            records: 1,
        };
        result
            .publication
            .record
            .streams
            .insert(name.into(), stream.clone());
        result.execution.record.streams.insert(name.into(), stream);
    }
    result.publication.payloads.insert(name.into(), bytes);
}
