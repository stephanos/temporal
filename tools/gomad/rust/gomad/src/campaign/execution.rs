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

pub(super) struct Result {
    pub execution: ledger::Execution,
    pub publication: Publication,
}

pub(super) fn run(
    prepared: &Prepared,
    job: &ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
) -> std::result::Result<Result, String> {
    if prepared.scenario.is_some() {
        return super::scenario::execute(prepared, job, timeout, cancellation);
    }
    if prepared.network.is_some() {
        return Err(
            "runner_failure: standalone network execution requires its linked model owner".into(),
        );
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
            Ok(linked::ModelInput {
                configuration: serde_json::from_value(value).map_err(|error| error.to_string())?,
                arguments: configuration.arguments.clone(),
                environment,
                replay_trace: None,
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
            wall_timeout: timeout.min(Duration::from_millis(configuration.limits.wall_millis)),
        },
    };
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let mut admitted = linked::prepare(request)
        .map_err(|error| format!("runner_failure: admit campaign target: {error}"))?;
    admitted.cancellation(cancellation.clone());
    let observed = linked::execute_prepared(admitted)
        .map_err(|error| format!("runner_failure: execute campaign target: {error}"))?;
    publication(prepared, job, observed)
}

pub(super) fn publication(
    prepared: &Prepared,
    job: &ledger::Job,
    observed: linked::Execution,
) -> std::result::Result<Result, String> {
    let configuration = &prepared.configuration;
    let assertions = super::soft_assertions::assessed(prepared, &observed)?;
    let mut streams = BTreeMap::new();
    let mut payloads = BTreeMap::from([("target/module.wasm".into(), prepared.module.clone())]);
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
        && !matches!(
            observed.outcome.termination,
            "watchdog_timeout" | "cancelled"
        )
    {
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
                profile: "gomad-choice-trace/v3".into(),
                implementation_sha256: linked::IMPLEMENTATION_SHA256.into(),
                identity: identity.clone(),
                trace: Payload {
                    schema: "gomad-choice-trace/v3".into(),
                    file: "evidence/choice".into(),
                    sha256: canonical::digest(trace),
                    bytes: trace.len() as u64,
                    limit,
                },
                records: inspected.records,
                terminal_state: "complete".into(),
                tape_sha256: Some(inspected.tape_sha256),
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
            } else {
                observed.outcome.message
            },
        },
        streams,
    };
    payloads.insert("record.json".into(), record.bytes()?);
    let execution = ledger::Execution {
        ordinal: job.ordinal,
        seed: job.seed,
        record: record.clone(),
        artifact: None,
        stored_bytes: 0,
        semantic_probes: Vec::new(),
        choice_features,
        divergence: observed.outcome.divergence.map(|value| *value),
    };
    Ok(Result {
        execution,
        publication: Publication { record, payloads },
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
