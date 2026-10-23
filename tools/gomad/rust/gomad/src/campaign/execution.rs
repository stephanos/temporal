use super::{ledger, prepare::Prepared};
use crate::{
    evidence::{
        canonical,
        identity::Identity,
        inspection::{ChoiceEvidence, Details, Payload},
        record::{Outcome, Record, Stream},
        store::Publication,
    },
    replay::preflight::{self, Inputs},
};
use gomad_wasmhost::linked;
use serde_json::json;
use std::{collections::BTreeMap, time::Duration};

pub(crate) struct Result {
    pub execution: ledger::Execution,
    pub publication: Publication,
    pub metrics: Metrics,
}

/// In-process observations of one execution. They never enter records.
#[derive(Clone, Debug, Default)]
pub(crate) struct Metrics {
    pub wall_nanos: u64,
    /// The linked image high water of this store window, when qualification
    /// attributes process memory by phase.
    pub store_peak_rss_bytes: Option<u64>,
    /// One entry per fresh guest store; a cluster has one per incarnation.
    pub stores: Vec<StoreMetrics>,
}

#[derive(Clone, Debug)]
pub(crate) struct StoreMetrics {
    pub incarnation: Option<String>,
    pub fuel_used: u64,
    pub trace_bytes: u64,
    pub guest_peak_memory_bytes: u64,
    pub guest_memory_limit_bytes: u64,
}

pub(crate) const TRACE_STREAMS: [&str; 5] =
    ["transcript", "choice", "diagnostic", "storage", "network"];

impl StoreMetrics {
    pub(crate) fn observed(
        incarnation: Option<String>,
        execution: &linked::Execution,
        fuel: u64,
        memory_bytes: u64,
    ) -> Self {
        let outcome = &execution.outcome;
        let cluster = incarnation.is_some();
        Self {
            incarnation,
            fuel_used: fuel.saturating_sub(outcome.fuel_remaining),
            // The Go qualification owner's trace bytes: WASI transcript,
            // choice, diagnostic, storage and network traces. Output streams,
            // inspection data and storage snapshots are not traces.
            // A cluster's network trace is one shared stream, counted once by
            // the cluster (`scenario::store_metrics`), never per store.
            trace_bytes: execution
                .evidence
                .iter()
                .filter(|(name, _)| {
                    TRACE_STREAMS.contains(&name.as_str())
                        && !(cluster && name.as_str() == "network")
                })
                .map(|(_, bytes)| bytes.len() as u64)
                .sum(),
            guest_peak_memory_bytes: outcome.peak_memory_bytes as u64,
            guest_memory_limit_bytes: memory_bytes,
        }
    }
}

impl Metrics {
    pub(crate) fn observed(
        execution: &linked::Execution,
        limits: &crate::evidence::record::Limits,
        wall: Duration,
    ) -> Self {
        Self {
            wall_nanos: nanos(wall),
            store_peak_rss_bytes: None,
            stores: vec![StoreMetrics::observed(
                None,
                execution,
                limits.fuel,
                limits.memory_bytes,
            )],
        }
    }
}

pub(crate) fn nanos(wall: Duration) -> u64 {
    u64::try_from(wall.as_nanos()).unwrap_or(u64::MAX)
}

pub(super) fn run(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
) -> std::result::Result<Result, String> {
    run_with_replay(prepared, job, timeout, cancellation, None, None)
}

pub(crate) fn run_with_replay(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    replay: Option<&linked::ReplayRequest>,
    raw_trace: Option<&[u8]>,
) -> std::result::Result<Result, String> {
    run_with_output(
        prepared,
        job,
        timeout,
        cancellation,
        replay,
        raw_trace,
        None,
    )
}

pub(crate) fn run_with_output(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    replay: Option<&linked::ReplayRequest>,
    raw_trace: Option<&[u8]>,
    output: Option<std::sync::Arc<dyn linked::OutputConsumer>>,
) -> std::result::Result<Result, String> {
    run_with_controls(
        prepared,
        job,
        timeout,
        cancellation,
        Controls {
            replay,
            raw_trace,
            output,
            #[cfg(all(test, feature = "helper-determinism-fixture"))]
            fault: None,
        },
    )
}

struct Controls<'a> {
    replay: Option<&'a linked::ReplayRequest>,
    raw_trace: Option<&'a [u8]>,
    output: Option<std::sync::Arc<dyn linked::OutputConsumer>>,
    #[cfg(all(test, feature = "helper-determinism-fixture"))]
    fault: Option<super::counterfactual::Fault>,
}

#[cfg(all(test, feature = "helper-determinism-fixture"))]
pub(super) fn run_counterfactual(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    fault: super::counterfactual::Fault,
) -> std::result::Result<Result, String> {
    run_with_controls(
        prepared,
        job,
        timeout,
        cancellation,
        Controls {
            replay: None,
            raw_trace: None,
            output: None,
            fault: Some(fault),
        },
    )
}

fn run_with_controls(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    controls: Controls<'_>,
) -> std::result::Result<Result, String> {
    let Controls {
        replay,
        raw_trace,
        output,
        #[cfg(all(test, feature = "helper-determinism-fixture"))]
        fault,
    } = controls;
    #[cfg(feature = "determinism-fixture")]
    crate::live_backend::before_execution(job.seed, cancellation)?;
    if prepared.scenario.is_some() {
        if replay.is_some() || raw_trace.is_some() {
            return Err(
                "unsupported_target: scenario runtime prefix forcing is unsupported".into(),
            );
        }
        #[cfg(all(test, feature = "helper-determinism-fixture"))]
        if let Some(fault) = fault {
            return super::scenario::execute_counterfactual(
                prepared,
                job,
                timeout,
                cancellation,
                fault,
            );
        }
        return super::scenario::execute(prepared, job, timeout, cancellation);
    }
    let configuration = &prepared.configuration;
    let model = configuration
        .model
        .as_ref()
        .map(|value| {
            let mut value = value.clone();
            let object = value
                .as_object_mut()
                .ok_or("invalid campaign model configuration")?;
            if object.insert("seed".into(), job.seed.into()).is_some() {
                return Err("campaign model seed must derive from selection".to_owned());
            }
            if configuration.profile == "gomad.wasi-stock-exploratory/v1" {
                use sha2::Digest as _;
                object.insert(
                    "entropy_key".into(),
                    json!(
                        sha2::Sha256::digest(
                            format!("{}:{}", prepared.toolchain, job.seed).as_bytes()
                        )
                        .to_vec()
                    ),
                );
            }
            let mut environment = configuration
                .environment
                .iter()
                .map(|(name, value)| format!("{name}={value}"))
                .collect::<Vec<_>>();
            if configuration.derive_seed_environment {
                environment.push(format!("GOMADSEED={}", job.seed));
            }
            let mut model_configuration: linked::ModelConfig =
                serde_json::from_value(value).map_err(|error| error.to_string())?;
            if replay.is_some() {
                model_configuration.mode =
                    if replay.is_some_and(|request| request.prefix_control.is_some()) {
                        3
                    } else {
                        2
                    };
                model_configuration.replay = replay.cloned();
            }
            Ok(linked::ModelInput {
                configuration: model_configuration,
                arguments: configuration.arguments.clone(),
                environment,
                replay_trace: raw_trace.map(<[u8]>::to_vec),
            })
        })
        .transpose()?;
    let request = linked::Request {
        module: prepared.module.clone(),
        module_sha256: canonical::digest(&prepared.module),
        model,
        namespace: configuration.namespace.clone(),
        storage: configuration.storage.clone(),
        limits: linked::Limits {
            fuel: configuration.limits.fuel,
            memory_bytes: configuration.limits.memory_bytes,
            output_bytes: configuration.limits.output_bytes,
            evidence_bytes: configuration.limits.evidence_bytes,
            wall_timeout: timeout.min(preflight::wall_timeout(
                &configuration.limits,
                configuration.wall_nanos,
            )?),
        },
    };
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let mut admitted = linked::prepare(request)
        .map_err(|error| format!("runner_failure: admit campaign target: {error}"))?;
    admitted.conditions(linked::Conditions::compiled());
    #[cfg(all(test, feature = "helper-determinism-fixture"))]
    if let Some(fault) = fault {
        if fault.owner.is_some() {
            return Err("standalone fixture fault cannot select a cluster owner".into());
        }
        admitted.fixture_fault(fault.kind);
    }
    if let Some(network) = &configuration.network {
        admitted
            .network(network.clone())
            .map_err(|error| format!("runner_failure: admit standalone network: {error}"))?;
    }
    admitted.cancellation(cancellation.clone());
    if let Some(output) = output {
        admitted.output(output);
    }
    let (observed, metrics) = measured(admitted, &configuration.limits)?;
    #[cfg(feature = "determinism-fixture")]
    crate::live_backend::after_execution(job.ordinal, cancellation)?;
    let mut result = publication(prepared, job, observed)?;
    result.metrics = metrics;
    Ok(result)
}

/// Runs one admitted store inside the whole-process backstop and its
/// qualification phase window (`qualify::usage`).
fn measured(
    admitted: linked::Prepared,
    limits: &crate::evidence::record::Limits,
) -> std::result::Result<(linked::Execution, Metrics), String> {
    crate::qualify::usage::store_begin();
    let started = std::time::Instant::now();
    let observed = crate::backstop::within(admitted.wall_timeout(), || {
        linked::execute_prepared(admitted)
    });
    let wall = started.elapsed();
    let store_peak = crate::qualify::usage::store_end();
    let observed =
        observed.map_err(|error| format!("runner_failure: execute campaign target: {error}"))?;
    let mut metrics = Metrics::observed(&observed, limits, wall);
    metrics.store_peak_rss_bytes = store_peak;
    Ok((observed, metrics))
}

#[expect(
    clippy::too_many_lines,
    reason = "Publication binds raw helper streams, typed outcome and immutable replay inputs in one record"
)]
pub(crate) fn publication(
    prepared: &Prepared,
    job: &ledger::Job,
    observed: linked::Execution,
) -> std::result::Result<Result, String> {
    let mut observed = observed;
    if let Some(raw) = observed.event_order.take() {
        observed.evidence.insert("event-order".into(), raw);
    }
    let configuration = &prepared.configuration;
    let assertions = super::soft_assertions::assessed(prepared, &observed)?;
    let mut streams = BTreeMap::new();
    let mut payloads = BTreeMap::from([("target/module.wasm".into(), prepared.module.clone())]);
    insert(
        &mut payloads,
        &mut streams,
        "target/provenance.json",
        prepared.provenance.clone(),
        1,
    );
    if let Some(bytes) = runtime_terminal(observed.outcome.local_model.as_deref())? {
        payloads.insert("evidence/runtime-terminal".into(), bytes);
    }
    for (name, bytes) in &observed.evidence {
        let path = format!(
            "{}/{name}",
            if matches!(name.as_str(), "stdout" | "stderr") {
                "output"
            } else {
                "evidence"
            }
        );
        let records = observed
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| {
                model[if name == "transcript" {
                    "calls"
                } else {
                    "records"
                }]
                .as_u64()
            })
            .unwrap_or(0);
        insert(&mut payloads, &mut streams, &path, bytes.clone(), records);
    }
    let mut choice_features = Vec::new();
    if let Some(identity) = &prepared.choice_identity
        && matches!(
            observed.outcome.termination,
            "exit" | "trap" | "divergence" | "diverged"
        )
    {
        if !payloads.contains_key("evidence/choice")
            && observed
                .outcome
                .local_model
                .as_deref()
                .is_some_and(|model| {
                    model["configured"].as_bool() == Some(true)
                        && (model["finished"].as_bool() == Some(true)
                            || observed.outcome.divergence.is_some())
                        && model["records"].as_u64() == Some(0)
                })
        {
            insert(
                &mut payloads,
                &mut streams,
                "evidence/choice",
                Vec::new(),
                0,
            );
        }
        let trace = payloads
            .get("evidence/choice")
            .ok_or("runner_failure: missing campaign choice evidence")?;
        let inspected = linked::inspect_choices(trace, identity)?;
        choice_features = linked::choice_features(
            trace,
            identity,
            if observed.outcome.divergence.is_some() {
                "diverged"
            } else {
                "complete"
            },
        )
        .map_err(str::to_owned)?;
        let limit = configuration
            .model
            .as_ref()
            .and_then(|model| model["limit"].as_u64())
            .ok_or("campaign choice limit missing")?;
        let details = Details {
            schema: "gomad.inspection-evidence/v1".into(),
            outputs: None,
            choices: Some(ChoiceEvidence {
                profile: "gomad-choice-trace/v4".into(),
                implementation_sha256: linked::IMPLEMENTATION_SHA256.into(),
                identity: identity.clone(),
                trace: Payload {
                    schema: "gomad-choice-trace/v4".into(),
                    file: "evidence/choice".into(),
                    sha256: canonical::digest(trace),
                    bytes: trace.len() as u64,
                    limit,
                },
                records: inspected.records,
                terminal_state: if observed.outcome.divergence.is_some() {
                    "diverged"
                } else {
                    "complete"
                }
                .into(),
                tape_sha256: observed
                    .outcome
                    .divergence
                    .is_none()
                    .then_some(inspected.tape_sha256),
            }),
            world: None,
            simulation: None,
            minimization: None,
        };
        insert(
            &mut payloads,
            &mut streams,
            "evidence/inspection.json",
            canonical::encode(&details)?,
            1,
        );
    }
    let inputs = Inputs {
        schema: "gomad.replay-inputs/v1".into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        toolchain_sha256: prepared.toolchain.clone(),
        runtime_implementation_sha256: (configuration.profile == "gomad.wasi-cooperative/v1")
            .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        configuration: serde_json::from_slice(&canonical::encode(configuration)?)
            .map_err(|error| error.to_string())?,
    };
    insert(
        &mut payloads,
        &mut streams,
        preflight::FILE,
        canonical::encode(&inputs)?,
        1,
    );
    let termination = observed.outcome.termination;
    let class = match termination {
        "divergence" | "diverged" if observed.outcome.divergence.is_some() => "target_failure",
        "exit"
            if assertions.as_ref().is_some_and(|report| {
                report.inconclusive || !report.report_only && !report.messages.is_empty()
            }) =>
        {
            "target_failure"
        }
        "exit" if observed.outcome.exit_code == Some(0) => "success",
        "exit" | "trap" => "target_failure",
        "watchdog_timeout" => "watchdog",
        _ => "runner_failure",
    };
    if let Some(report) = &assertions {
        insert(
            &mut payloads,
            &mut streams,
            "evidence/soft-assertions.json",
            canonical::encode(report)?,
            1,
        );
    }
    let identity = Identity {
        target: canonical::digest(&prepared.module),
        toolchain: prepared.toolchain.clone(),
        platform: preflight::platform(),
        implementation: preflight::implementation()?,
        configuration: canonical::digest(&canonical::encode(configuration)?),
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        profile: configuration.profile.clone(),
        seed: job.seed,
        arguments: configuration.arguments.clone(),
        environment: configuration.environment.clone(),
        limits: configuration.limits.clone(),
        virtual_time_nanos: observed
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| model["now"].as_u64())
            .unwrap_or(0),
        outcome: Outcome {
            class: class.into(),
            termination: termination.into(),
            exit_code: observed.outcome.exit_code,
            reason: if termination == "cancelled" {
                "runner_cancelled".into()
            } else if matches!(termination, "exit" | "trap") {
                super::soft_assertions::reason(assertions.as_ref(), observed.outcome.message)
            } else {
                observed.outcome.message
            },
        },
        streams,
    };
    payloads.insert("record.json".into(), record.bytes()?);
    let mut publication = Publication { record, payloads };
    if let Some(origin) = super::origin::from_preparation(prepared)? {
        origin.bind(&mut publication)?;
    }
    let record = &publication.record;
    let execution = ledger::Execution {
        ordinal: job.ordinal,
        seed: job.seed,
        record: record.clone(),
        artifact: None,
        stored_bytes: 0,
        semantic_probes: Vec::new(),
        choice_features,
        divergence: observed.outcome.divergence.map(|value| *value),
        scenario_steps: None,
        soft_assertions: assertions
            .as_ref()
            .map(serde_json::to_value)
            .transpose()
            .map_err(|error| error.to_string())?,
    };
    Ok(Result {
        execution,
        publication,
        metrics: Metrics::default(),
    })
}

fn insert(
    payloads: &mut BTreeMap<String, Vec<u8>>,
    streams: &mut BTreeMap<String, Stream>,
    name: &str,
    bytes: Vec<u8>,
    records: u64,
) {
    streams.insert(
        name.into(),
        Stream {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            records,
        },
    );
    payloads.insert(name.into(), bytes);
}

fn runtime_terminal(
    model: Option<&serde_json::Value>,
) -> std::result::Result<Option<Vec<u8>>, String> {
    let Some(model) = model else {
        return Ok(None);
    };
    let finished = model["finished"] == true;
    let Some(value) = model.get("terminal_base64") else {
        return if finished {
            Err("completed runtime terminal is missing".into())
        } else {
            Ok(None)
        };
    };
    let text = value
        .as_str()
        .ok_or("runtime terminal is not base64 text")?;
    if text.is_empty() && !finished {
        return Ok(None);
    }
    let bytes = super::scenario::canonical_base64(text)?;
    if !finished || bytes.len() != 16 || bytes[..4] != 1_u32.to_be_bytes() {
        return Err("invalid runtime finish terminal".into());
    }
    Ok(Some(bytes))
}

#[cfg(test)]
mod runtime_terminal_tests {
    #[test]
    fn runtime_terminal_exact_finish_and_malformed_refusals() {
        let bytes = [0, 0, 0, 1, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 9];
        let text = super::super::exploration::base64(&bytes);
        let model = serde_json::json!({"finished":true,"terminal_base64":text});
        assert_eq!(
            super::runtime_terminal(Some(&model)).unwrap(),
            Some(bytes.to_vec())
        );
        assert_eq!(super::runtime_terminal(None).unwrap(), None);
        assert_eq!(
            super::runtime_terminal(Some(
                &serde_json::json!({"finished":false,"terminal_base64":""})
            ))
            .unwrap(),
            None
        );
        for model in [
            serde_json::json!({"finished":true}),
            serde_json::json!({"finished":true,"terminal_base64":""}),
            serde_json::json!({"finished":true,"terminal_base64":"!!!!"}),
            serde_json::json!({"finished":true,"terminal_base64":4}),
            serde_json::json!({"finished":false,"terminal_base64":text}),
            serde_json::json!({"finished":true,"terminal_base64":format!("{text}\n")}),
            serde_json::json!({"finished":true,"terminal_base64":super::super::exploration::base64(&[0;16])}),
            serde_json::json!({"finished":true,"terminal_base64":super::super::exploration::base64(&bytes[..15])}),
        ] {
            assert!(
                super::runtime_terminal(Some(&model)).is_err(),
                "malformed runtime finish accepted: {model}"
            );
        }
    }
}
