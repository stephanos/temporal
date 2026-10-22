use crate::network::{self, history};

use base64::Engine;
use serde_json::Value;
use std::collections::BTreeSet;
use std::path::PathBuf;

fn request_from_json(value: &Value) -> network::Request {
    let mut value = value.clone();
    let bytes = match value.get("bytes").and_then(Value::as_str) {
        Some(bytes) => base64::prelude::BASE64_STANDARD.decode(bytes).unwrap(),
        None => Vec::new(),
    };
    value["bytes"] = serde_json::to_value(bytes).unwrap();
    serde_json::from_value(value).unwrap()
}

fn result_to_json(result: network::OperationResult) -> Value {
    let mut value = serde_json::to_value(&result).unwrap();
    if !result.bytes.is_empty() {
        value["bytes"] = base64::prelude::BASE64_STANDARD.encode(result.bytes).into();
    }
    value
}

const ADMISSION_FIXTURES: &[&str] = &[
    "admission-two-pair-slot-handles-and-lifecycle",
    "admission-backlog-blocks-before-connection-allocation",
    "admission-global-capacity-precedes-ready-overflow",
    "admission-full-close-snapshot-marker-until-reset",
    "admission-stale-domain-registration-and-revocation",
    "admission-oversize-chunks-precede-stale-domain",
    "admission-clock-ceiling-monotone-and-caller-admission",
    "admission-clock-ceiling-cannot-exclude-queue-or-listener-deadline",
    "admission-advance-before-preserves-equal-ready-time",
    "admission-listen-collision-precedes-history-capacity",
    "admission-cancel-backlog-dial-history-67",
    "admission-cancel-backlog-dial-history-8192",
    "admission-group-duration-ignored-unless-delay",
    "admission-reset-after-finish-keeps-pair-and-records-crash",
    "admission-dial-caller-deadline-advance-before-false",
    "admission-dial-caller-deadline-advance-before-true",
    "admission-dial-clock-ceiling-heal-repartition-rejection",
    "admission-pending-dial-poll-keeps-old-wait-until-ready",
    "admission-partial-write-caller-versus-connection-deadline-false",
    "admission-partial-write-caller-versus-connection-deadline-true",
    "admission-completed-dial-no-rearm-capacity-false-consumed-false",
    "admission-completed-dial-no-rearm-capacity-false-consumed-true",
    "admission-completed-dial-no-rearm-capacity-true-consumed-false",
    "admission-completed-dial-no-rearm-capacity-true-consumed-true",
    "admission-mixed-dial-pair-atomic-partition-6",
    "admission-mixed-dial-pair-atomic-partition-7",
    "admission-two-pair-stalls-revocation-and-global-request-ids",
    "admission-replay-source",
    "admission-resealed-transition-outcome",
    "admission-resealed-listener-timeout",
    "admission-resealed-listener-reset",
    "admission-resealed-node-port",
    "admission-resealed-link-order",
    "admission-resealed-listener-port-zero",
    "admission-resealed-connection-incarnation",
    "admission-resealed-reset-retains-bytes",
    "admission-resealed-connection-bytes-capacity",
    "admission-replay-listener-history-[true true]",
    "admission-replay-listener-history-[true false]",
    "admission-replay-listener-history-[true true true false]",
    "admission-replay-listener-history-[false true]",
    "admission-replay-listener-history-[false false]",
    "admission-replay-listener-port-0",
    "admission-replay-listener-port-65536",
    "admission-replay-listener-port-18446744073709551615",
    "admission-replay-listener-history-unordered",
    "admission-expiry-revoke-id-reuse-divergent-false",
    "admission-expiry-revoke-id-reuse-divergent-true",
    "admission-group-replay-rollback-partial-false",
    "admission-group-replay-rollback-partial-true",
    "admission-reset-wakes-backlog-after-replay-divergence",
    "admission-separator-node-snapshot-order",
    "admission-separator-link-alias-identity",
];

const STALL_FIXTURES: &[&str] = &[
    "replay-mismatch-preserves-timeout-reservation-and-bytes",
    "equal-time-held-delivery-heal-first-false",
    "equal-time-held-delivery-heal-first-true",
    "expiry-wakes-partial-writer-and-blocked-reader",
    "reserved-expiry-drops-seven-received-and-held-bytes",
    "simultaneous-expiry-second-connection-stalled-first",
    "equal-time-heal-first-false",
    "equal-time-heal-first-true",
    "stall-epoch-invalidation",
    "healed-stall-epoch-exact-100-101-201",
    "autonomous-expiry-reserved-terminal-slot",
    "simultaneous-pair-expiry",
    "held-reservation-ceiling-5",
    "held-reservation-ceiling-6",
    "partition-reservation-atomic-rejection",
    "stall-overflow-queued-false",
    "stall-overflow-queued-true",
    "held-timeout-replay-admission-valid",
    "partial-write-stall-timeout",
    "pending-dial-caller-0",
    "pending-dial-caller-5",
    "pending-dial-caller-100",
    "pending-dial-heal-cancel",
    "pending-dial-heal-revoke",
    "pending-dial-heal-refused",
    "pending-dial-heal-capacity",
    "pending-dial-heal-repartition-epoch",
    "pending-dial-reservation-overflow-false",
    "pending-dial-reservation-overflow-true",
];
const CONNECTION_FIXTURES: &[&str] = &[
    "oversized-node-address-wire-refusal",
    "listener-history-keys-1-closed-false",
    "listener-history-keys-1-closed-true",
    "listener-history-keys-17-closed-false",
    "listener-history-keys-17-closed-true",
    "listener-history-closed-pending-accept",
    "listener-history-restart-graceful-false",
    "listener-history-restart-graceful-true",
    "listener-history-capacity-keeps-closed",
    "close-before-partition-close",
    "close-before-partition-close_write",
    "closed-partition-atomic-rejection-symmetric-false",
    "closed-partition-atomic-rejection-symmetric-true",
    "delayed-drain-close",
    "delayed-drain-close_read",
    "delayed-drain-close_write",
    "fifo-partition-delay-redundant-heal",
    "global-capacity-and-read-deadline",
    "held-suffix-close",
    "held-suffix-close_write",
    "held-suffix-reset",
    "held-timeout-replay-admission-connection",
    "held-timeout-replay-admission-incarnation",
    "held-timeout-replay-admission-kind",
    "held-timeout-replay-admission-ordinal",
    "held-timeout-replay-admission-snapshot",
    "listener-connection-and-request-limits",
    "partial-write-backpressure-drain",
    "payload-free-history",
    "payload-free-replay-xy",
    "payload-free-replay-xyz",
    "received-tail-repartition-delayed-suffix",
    "revoke-restart-graceful-false",
    "revoke-restart-graceful-true",
    "self-dial-and-refused-dial",
];

fn corpora() -> Vec<(String, Value)> {
    let directory = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/network_v2");
    let mut files: Vec<_> = std::fs::read_dir(directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .collect();
    files.sort();
    files
        .into_iter()
        .map(|path| {
            assert_eq!(path.extension().unwrap(), "json");
            let name = path.file_stem().unwrap().to_str().unwrap().to_owned();
            let value: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            assert_eq!(value["schema"], "gomad.network-operation-corpus/v1");
            assert_eq!(value["name"], name);
            assert_eq!(
                value["implementation_sha256"],
                include_str!("../src/network/PINNED_GO_NETWORK_SHA256").trim()
            );
            (name, value)
        })
        .collect()
}

#[test]
fn network_corpus_classification_rejects_unclassified_fixtures() {
    let mut expected: BTreeSet<String> = CONNECTION_FIXTURES
        .iter()
        .chain(STALL_FIXTURES)
        .chain(ADMISSION_FIXTURES)
        .map(|name| (*name).to_owned())
        .collect();
    for seed in 1..=32 {
        for negative in [false, true] {
            assert!(expected.insert(format!("random-{seed:02}-negative-{negative}")));
        }
    }
    for graceful in [false, true] {
        assert!(expected.insert(format!("broker-strict-topology-graceful-{graceful}")));
    }
    assert_eq!(CONNECTION_FIXTURES.len() + 64, 99);
    assert_eq!(STALL_FIXTURES.len(), 29);
    assert_eq!(
        corpora()
            .into_iter()
            .map(|(name, _)| name)
            .collect::<BTreeSet<_>>(),
        expected
    );
}

#[test]
fn network_history_binary_identity_matches_every_authentic_step_and_record() {
    for (name, corpus) in corpora() {
        let mut transitions: Vec<history::Transition> = Vec::new();
        for (index, step) in corpus["steps"]
            .as_array()
            .map(Vec::as_slice)
            .unwrap_or_default()
            .iter()
            .enumerate()
        {
            if let Some(added) = step["expected"]["transitions"].as_array() {
                for transition in added {
                    transitions.push(serde_json::from_value(transition.clone()).unwrap());
                }
            }
            if step["input"]["action"] == "finish" {
                history::canonicalize(&mut transitions);
            }
            if step["expected"]["snapshot"].is_null() {
                assert_eq!(
                    step["expected"]["snapshot_error"],
                    "encode simulation network snapshot: simulation network wire string exceeds its byte limit"
                );
                continue;
            }
            let snapshot: history::Snapshot =
                serde_json::from_value(step["expected"]["snapshot"].clone()).unwrap();
            assert_eq!(
                history::transitions_digest(&transitions).unwrap(),
                snapshot.transition_sha256,
                "{name} step {index}"
            );
            assert_eq!(
                history::snapshot_digest(&snapshot).unwrap(),
                snapshot.identity,
                "{name} step {index}"
            );
            assert_eq!(
                serde_json::to_value(&snapshot).unwrap(),
                step["expected"]["snapshot"],
                "{name} step {index}: snapshot shape"
            );
        }
        if let Some(record) = corpus.get("final") {
            let final_transitions: Option<Vec<history::Transition>> =
                serde_json::from_value(record["transitions"].clone()).unwrap();
            let snapshot: history::Snapshot =
                serde_json::from_value(record["snapshot"].clone()).unwrap();
            history::canonicalize(&mut transitions);
            assert_eq!(
                final_transitions.as_deref().unwrap_or_default(),
                transitions,
                "{name}: canonical lanes"
            );
            assert_eq!(
                history::transitions_digest(&transitions).unwrap(),
                snapshot.transition_sha256,
                "{name}: final history"
            );
            assert_eq!(
                history::snapshot_digest(&snapshot).unwrap(),
                snapshot.identity,
                "{name}: final snapshot"
            );
        }
    }
}

fn go_wire_error(model: &network::Model, error: &str) -> String {
    if matches!(
        error,
        "simulation network wire string exceeds its byte limit"
            | "simulation network wire value exceeds its byte limit"
    ) {
        let stage = if history::transitions_digest(&model.transitions).is_err() {
            "transitions"
        } else {
            "snapshot"
        };
        format!("encode simulation network {stage}: {error}")
    } else {
        error.to_owned()
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "Replay checks preserve the complete ordered corpus observation sequence."
)]
fn replay_corpus(name: &str, corpus: &Value, complete: bool) -> Result<(), String> {
    let mut config_json = corpus["config"].clone();
    if let Some(links) = config_json.get_mut("links")
        && links.is_null()
    {
        *links = Value::Array(Vec::new());
    }
    let config =
        serde_json::from_value(config_json).map_err(|error| format!("{name}: config: {error}"))?;
    let mut model = match network::Model::new(corpus["run"].as_u64().unwrap(), config) {
        Ok(model) if corpus["creation_error"] == "" => model,
        Ok(_) => return Err(format!("{name}: accepted invalid replay creation")),
        Err(error) if corpus["creation_error"] == error => return Ok(()),
        Err(error) => return Err(format!("{name}: creation: {error}")),
    };
    let steps = match corpus.get("steps") {
        Some(Value::Null) => &[][..],
        Some(Value::Array(steps)) => steps.as_slice(),
        _ => {
            return Err(format!(
                "{name}: steps must be an array or Go's null empty slice"
            ));
        }
    };
    let mut history_start = 0;
    for (index, step) in steps.iter().enumerate() {
        let input = &step["input"];
        let mut result = None;
        let mut ready = None;
        let left = input["left"].as_str().unwrap_or_default();
        let right = input["right"].as_str().unwrap_or_default();
        let symmetric = input["symmetric"].as_bool().unwrap_or_default();
        let action = input["action"].as_str().unwrap();
        let operation = match action {
            "register" => model.register(serde_json::from_value(input["domain"].clone()).unwrap()),
            "dispatch" => {
                result = Some(model.dispatch(request_from_json(&input["request"])));
                Ok(())
            }
            "ready" => {
                ready = model.ready();
                Ok(())
            }
            "advance" => model.advance(input["nanos"].as_u64().unwrap()).map(|ids| {
                ready = ids;
            }),
            "advance_before" => model
                .advance_before(input["nanos"].as_u64().unwrap())
                .map(|ids| {
                    ready = ids;
                }),
            "partition" => model.partition(left, right, symmetric),
            "clock_ceiling" => {
                model.limit_clock_ceiling(input["nanos"].as_u64().unwrap_or_default())
            }
            "group" => model.change_group(
                &serde_json::from_value::<Vec<String>>(input["left_nodes"].clone()).unwrap(),
                &serde_json::from_value::<Vec<String>>(input["right_nodes"].clone()).unwrap(),
                input["link_action"].as_str().unwrap(),
                input["nanos"].as_u64().unwrap_or_default(),
                symmetric,
            ),
            "heal" => model.heal(left, right, symmetric),
            "delay" => model.delay(
                left,
                right,
                input["nanos"].as_u64().unwrap_or_default(),
                symmetric,
            ),
            "finish" => {
                let finished = model.finish().map(|_| ());
                finished.map_err(|error| go_wire_error(&model, &error))
            }
            "cancel" => model.cancel(
                &serde_json::from_value(input["domain"].clone()).unwrap(),
                input["request_id"].as_u64().unwrap(),
            ),
            "reset" => model.reset(input["handle"].as_u64().unwrap()),
            "revoke" => model.revoke(
                &serde_json::from_value(input["domain"].clone()).unwrap(),
                input["graceful"].as_bool().unwrap_or_default(),
            ),
            _ => {
                return Err(format!(
                    "{name} step {index}: owner API action {action} is not implemented"
                ));
            }
        };
        if action != "ready" && action != "advance" && action != "advance_before" {
            let completed: Vec<_> = model
                .pending
                .iter()
                .filter(|(_, pending)| pending.result.is_some())
                .map(|(id, _)| *id)
                .collect();
            ready = (!completed.is_empty()).then_some(completed);
        }
        let live: Vec<_> = model.live.values().cloned().collect();
        let pending: Vec<_> = model.pending.iter().map(|(id, pending)| serde_json::json!({"request_id":id,"state":if pending.result.is_some() { "completed" } else { "pending" },"accepted":pending.accepted})).collect();
        let added = &model.transitions[history_start..];
        let snapshot = model.snapshot();
        let mut actual = serde_json::json!({
            "error":operation.err().unwrap_or_default(), "ready":ready,
            "next_deadline":model.next_deadline(),
            "state":{"now":model.now,"pending_bytes":model.pending_bytes,"history_reserved":model.history_reserved,
                "live_domains":(!live.is_empty()).then_some(live),"pending_operations":(!pending.is_empty()).then_some(pending)},
            "snapshot":snapshot.as_ref().ok(),"transitions":(!added.is_empty()).then_some(added),
        });
        if let Err(error) = snapshot {
            actual["snapshot_error"] = go_wire_error(&model, &error).into();
        }
        if let Some(result) = result {
            actual["result"] = result_to_json(result);
        }
        if actual != step["expected"] {
            return Err(format!(
                "{name} step {index}: expected {}\nactual {actual}",
                step["expected"]
            ));
        }
        history_start = model.transitions.len();
    }
    if !complete {
        return Ok(());
    }
    match model.record() {
        Ok(record)
            if corpus.get("record_error").is_none()
                && serde_json::to_value(&record).unwrap() == corpus["final"] => {}
        Err(error) if corpus["record_error"] == go_wire_error(&model, &error) => {}
        _ => return Err(format!("{name}: final Record before Finish differs")),
    }
    if !steps.iter().any(|step| step["input"]["action"] == "finish") {
        let error = model.finish().err().unwrap_or_default();
        let error = go_wire_error(&model, &error);
        if error != corpus["final_error"] {
            return Err(format!("{name}: Finish: {error}"));
        }
    }
    Ok(())
}

#[test]
fn network_full_corpus_requires_all_scripts_every_step_and_final_record() {
    let mut failures = Vec::new();
    let mut executed = 0;
    for (name, corpus) in corpora() {
        executed += 1;
        if let Err(error) = replay_corpus(&name, &corpus, true) {
            failures.push(error);
        }
    }
    assert_eq!(executed, 130 + ADMISSION_FIXTURES.len());
    assert!(
        failures.is_empty(),
        "{} network scripts fail:\n{}",
        failures.len(),
        failures.join("\n\n")
    );
}
