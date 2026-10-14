#[path = "../src/network/mod.rs"]
mod network;
use network::history;

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

const STALL_FIXTURES: &[&str] = &[
    "equal-time-heal-first-false",
    "equal-time-heal-first-true",
    "stall-epoch-invalidation",
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
            assert_eq!(value["schema"], "gomad3.network-operation-corpus/v1");
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
    assert_eq!(CONNECTION_FIXTURES.len() + 64, 90);
    assert_eq!(STALL_FIXTURES.len(), 22);
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

#[test]
fn network_registered_incarnations_match_go_snapshots() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "payload-free-history")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    for step in corpus["steps"].as_array().unwrap().iter().take(2) {
        let domain = serde_json::from_value(step["input"]["domain"].clone()).unwrap();
        model.register(domain).unwrap();
        assert_eq!(
            serde_json::to_value(model.snapshot().unwrap()).unwrap(),
            step["expected"]["snapshot"]
        );
    }
    let stale = network::Domain {
        run: 1,
        node: "a".into(),
        address: "a".into(),
        incarnation: 1,
    };
    assert_eq!(model.register(stale).unwrap_err(), "stale file handle");
}

#[test]
fn network_listen_dial_accept_preserve_global_handles_and_history() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "payload-free-history")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    for (index, step) in corpus["steps"]
        .as_array()
        .unwrap()
        .iter()
        .take(5)
        .enumerate()
    {
        if step["input"]["action"] == "register" {
            model
                .register(serde_json::from_value(step["input"]["domain"].clone()).unwrap())
                .unwrap();
        } else {
            let result = model.dispatch(request_from_json(&step["input"]["request"]));
            assert_eq!(
                result_to_json(result),
                step["expected"]["result"],
                "step {index}"
            );
        }
        assert_eq!(
            serde_json::to_value(model.snapshot().unwrap()).unwrap(),
            step["expected"]["snapshot"],
            "step {index}"
        );
    }
}

#[test]
fn network_fifo_write_and_read_match_go_recorded_bytes() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "payload-free-history")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
        if step["input"]["action"] == "register" {
            model
                .register(serde_json::from_value(step["input"]["domain"].clone()).unwrap())
                .unwrap();
        } else {
            let result = model.dispatch(request_from_json(&step["input"]["request"]));
            assert_eq!(
                result_to_json(result),
                step["expected"]["result"],
                "step {index}"
            );
        }
        assert_eq!(
            serde_json::to_value(model.snapshot().unwrap()).unwrap(),
            step["expected"]["snapshot"],
            "step {index}"
        );
    }
}

#[test]
fn network_partial_write_retains_suffix_until_poll_consumes_completion() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "partial-write-backpressure-drain")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
        match step["input"]["action"].as_str().unwrap() {
            "register" => model
                .register(serde_json::from_value(step["input"]["domain"].clone()).unwrap())
                .unwrap(),
            "ready" => assert_eq!(
                serde_json::to_value(model.ready()).unwrap(),
                step["expected"]["ready"],
                "step {index}"
            ),
            "dispatch" => {
                let result = model.dispatch(request_from_json(&step["input"]["request"]));
                assert_eq!(
                    result_to_json(result),
                    step["expected"]["result"],
                    "step {index}"
                );
            }
            action => panic!("unsupported test action {action}"),
        }
        assert_eq!(
            serde_json::to_value(model.snapshot().unwrap()).unwrap(),
            step["expected"]["snapshot"],
            "step {index}"
        );
    }
}

#[test]
fn network_held_fifo_effective_heal_resets_delay_and_redundant_heal_preserves_it() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "fifo-partition-delay-redundant-heal")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
        let input = &step["input"];
        let left = input["left"].as_str().unwrap_or_default();
        let right = input["right"].as_str().unwrap_or_default();
        let symmetric = input["symmetric"].as_bool().unwrap_or_default();
        match input["action"].as_str().unwrap() {
            "register" => model
                .register(serde_json::from_value(input["domain"].clone()).unwrap())
                .unwrap(),
            "dispatch" => assert_eq!(
                result_to_json(model.dispatch(request_from_json(&input["request"]))),
                step["expected"]["result"],
                "step {index}"
            ),
            "advance" => assert_eq!(
                serde_json::to_value(model.advance(input["nanos"].as_u64().unwrap()).unwrap())
                    .unwrap(),
                step["expected"]["ready"],
                "step {index}"
            ),
            "partition" => model.partition(left, right, symmetric).unwrap(),
            "heal" => model.heal(left, right, symmetric).unwrap(),
            "delay" => model
                .delay(left, right, input["nanos"].as_u64().unwrap(), symmetric)
                .unwrap(),
            action => panic!("unsupported test action {action}"),
        }
        assert_eq!(
            model.next_deadline(),
            step["expected"]["next_deadline"].as_u64().unwrap(),
            "step {index}"
        );
        assert_eq!(
            serde_json::to_value(model.snapshot().unwrap()).unwrap(),
            step["expected"]["snapshot"],
            "step {index}"
        );
    }
}

fn replay_corpus(name: &str, corpus: &Value, complete: bool) -> Result<(), String> {
    let config =
        serde_json::from_value(corpus["config"].clone()).map_err(|error| error.to_string())?;
    let mut model = match network::Model::new(corpus["run"].as_u64().unwrap(), config) {
        Ok(model) if corpus["creation_error"] == "" => model,
        Ok(_) => return Err(format!("{name}: accepted invalid replay creation")),
        Err(error) if corpus["creation_error"] == error => return Ok(()),
        Err(error) => return Err(format!("{name}: creation: {error}")),
    };
    let mut history_start = 0;
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
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
            "partition" => model.partition(left, right, symmetric),
            "heal" => model.heal(left, right, symmetric),
            "delay" => model.delay(
                left,
                right,
                input["nanos"].as_u64().unwrap_or_default(),
                symmetric,
            ),
            "finish" => model.finish().map(|_| ()),
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
        if action != "ready" && action != "advance" {
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
        let mut actual = serde_json::json!({
            "error":operation.err().unwrap_or_default(), "ready":ready,
            "next_deadline":model.next_deadline(),
            "state":{"now":model.now,"pending_bytes":model.pending_bytes,"history_reserved":model.history_reserved,
                "live_domains":(!live.is_empty()).then_some(live),"pending_operations":(!pending.is_empty()).then_some(pending)},
            "snapshot":model.snapshot()?,"transitions":(!added.is_empty()).then_some(added),
        });
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
    if serde_json::to_value(model.record()?).unwrap() != corpus["final"] {
        return Err(format!("{name}: final Record before Finish differs"));
    }
    if !corpus["steps"]
        .as_array()
        .unwrap()
        .iter()
        .any(|step| step["input"]["action"] == "finish")
    {
        let error = model.finish().err().unwrap_or_default();
        if error != corpus["final_error"] {
            return Err(format!("{name}: Finish: {error}"));
        }
    }
    Ok(())
}

#[test]
fn network_full_corpus_requires_all_114_scripts_every_step_and_final_record() {
    let mut failures = Vec::new();
    let mut executed = 0;
    for (name, corpus) in corpora() {
        executed += 1;
        if let Err(error) = replay_corpus(&name, &corpus, true) {
            failures.push(error);
        }
    }
    assert_eq!(executed, 114);
    assert!(
        failures.is_empty(),
        "{} of 114 network scripts fail:\n{}",
        failures.len(),
        failures.join("\n\n")
    );
}

#[test]
fn network_oversize_bytes_reject_before_stale_domain_validation() {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "payload-free-history")
        .unwrap();
    let config = serde_json::from_value(corpus["config"].clone()).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    let request = serde_json::json!({"domain":{"run":1,"node":"a","incarnation":999,"address":"a"},"op":"write","request_id":1,"handle":999,"bytes":base64::prelude::BASE64_STANDARD.encode(vec![0_u8;65537])});
    let result = model.dispatch(request_from_json(&request));
    assert_eq!(result.error, "unsupported Gomad network operation");
    assert_eq!(model.pending_bytes, 0);
}

fn request(domain: &network::Domain, op: &str, id: u64, handle: u64) -> network::Request {
    network::Request {
        domain: domain.clone(),
        op: op.into(),
        request_id: id,
        handle,
        host: String::new(),
        port: 0,
        bytes: Vec::new(),
        count: 0,
        deadline: 0,
    }
}

fn four_domains() -> (network::Model, Vec<network::Domain>) {
    four_domains_with_capacity(64, 1024)
}

fn four_domains_with_capacity(
    connection_bytes: u64,
    bytes: u64,
) -> (network::Model, Vec<network::Domain>) {
    let config: network::Config = serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"a"},{"node":"b","address":"b"},{"node":"c","address":"c"},{"node":"d","address":"d"}],
        "links":[{"from":"a","to":"b","enabled":true,"delay_nanos":0},{"from":"b","to":"a","enabled":true,"delay_nanos":0},{"from":"c","to":"d","enabled":true,"delay_nanos":0},{"from":"d","to":"c","enabled":true,"delay_nanos":0}],
        "limits":{"listeners":128,"connections":128,"bytes":bytes,"connection_bytes":connection_bytes,"stall_nanos":10,"transitions":8192}
    })).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    let domains: Vec<_> = ["a", "b", "c", "d"]
        .into_iter()
        .map(|node| network::Domain {
            run: 1,
            node: node.into(),
            address: node.into(),
            incarnation: 1,
        })
        .collect();
    for domain in &domains {
        model.register(domain.clone()).unwrap();
    }
    (model, domains)
}

#[test]
fn network_two_pairs_keep_slot_ids_and_lifecycle_independent() {
    let (mut model, domains) = four_domains();
    let (a, b, c, d) = (&domains[0], &domains[1], &domains[2], &domains[3]);
    let base = |slot: u64| slot << 48;
    let lb = model.dispatch(request(b, "listen", base(2) + 1, 0));
    let ld = model.dispatch(request(d, "listen", base(4) + 1, 0));
    assert_eq!((lb.handle, ld.handle), (1, 2));
    assert_eq!(
        model
            .dispatch(request(b, "accept", base(2) + 2, lb.handle))
            .state,
        "pending"
    );
    assert_eq!(
        model
            .dispatch(request(d, "accept", base(4) + 2, ld.handle))
            .state,
        "pending"
    );
    let mut da = request(a, "dial", base(1) + 1, 0);
    da.host = "b".into();
    da.port = 20000;
    let mut dc = request(c, "dial", base(3) + 1, 0);
    dc.host = "d".into();
    dc.port = 20000;
    let ca = model.dispatch(da);
    let cc = model.dispatch(dc);
    assert_eq!((ca.handle, cc.handle), (3, 4));
    assert_eq!(model.ready(), Some(vec![base(2) + 2, base(4) + 2]));
    let cb = model.dispatch(request(b, "poll", base(2) + 2, 0));
    let cd = model.dispatch(request(d, "poll", base(4) + 2, 0));
    assert_eq!((cb.handle, cd.handle), (5, 6));
    model.partition("a", "b", false).unwrap();
    let mut wa = request(a, "write", base(1) + 2, ca.handle);
    wa.bytes = b"held".to_vec();
    assert_eq!(model.dispatch(wa).count, 4);
    assert_eq!(model.history_reserved, 1);
    assert_eq!(model.next_deadline(), 11);
    let mut wc = request(c, "write", base(3) + 2, cc.handle);
    wc.bytes = b"live".to_vec();
    assert_eq!(model.dispatch(wc).count, 4);
    let mut rd = request(d, "read", base(4) + 3, cd.handle);
    rd.count = 8;
    assert_eq!(model.dispatch(rd).bytes, b"live");
    let mut rb = request(b, "read", base(2) + 3, cb.handle);
    rb.count = 8;
    assert_eq!(model.dispatch(rb.clone()).state, "pending");
    assert_eq!(
        model.cancel(d, base(2) + 3).unwrap_err(),
        "stale file handle"
    );
    model.cancel(b, base(2) + 3).unwrap();
    assert_eq!(
        model.dispatch(request(b, "poll", base(2) + 3, 0)).error,
        "context canceled"
    );
    model.heal("a", "b", false).unwrap();
    assert_eq!(model.history_reserved, 0);
    rb.request_id = base(2) + 4;
    assert_eq!(model.dispatch(rb).bytes, b"held");
    model.revoke(a, false).unwrap();
    let mut read_old = request(b, "read", base(2) + 5, cb.handle);
    read_old.count = 1;
    assert_eq!(
        model.dispatch(read_old).error,
        "use of closed network connection"
    );
    let mut restarted = a.clone();
    restarted.incarnation = 2;
    model.register(restarted.clone()).unwrap();
    assert_eq!(
        model
            .dispatch(request(&restarted, "close", base(1) + 3, ca.handle))
            .error,
        "stale file handle"
    );
    let mut fresh = request(&restarted, "dial", base(1) + 4, 0);
    fresh.host = "b".into();
    fresh.port = 20000;
    assert_eq!(model.dispatch(fresh).local.port, 40001);
    model.revoke(d, true).unwrap();
    let mut eof = request(c, "read", base(3) + 3, cc.handle);
    eof.count = 1;
    assert_eq!(model.dispatch(eof).error, "EOF");
    assert_eq!(model.pending_bytes, 0);
}

#[test]
fn network_listener_backlog_blocks_without_allocating_connection() {
    let (mut model, domains) = four_domains();
    let (a, b) = (&domains[0], &domains[1]);
    let listener = model.dispatch(request(b, "listen", 1, 0));
    for id in 100..164 {
        let mut dial = request(a, "dial", id, 0);
        dial.host = "b".into();
        dial.port = 20000;
        assert_eq!(model.dispatch(dial).state, "completed");
    }
    let mut blocked = request(a, "dial", 1000, 0);
    blocked.host = "b".into();
    blocked.port = 20000;
    assert_eq!(model.dispatch(blocked).state, "pending");
    assert_eq!(model.snapshot().unwrap().next_connection, 64);
    assert_eq!(model.ready(), None);
    assert_eq!(
        model
            .dispatch(request(b, "accept", 2, listener.handle))
            .handle,
        66
    );
    assert_eq!(model.ready(), Some(vec![1000]));
    assert_eq!(model.dispatch(request(a, "poll", 1000, 0)).handle, 67);
    assert_eq!(model.snapshot().unwrap().next_connection, 65);
}

fn one_pair(
    model: &mut network::Model,
    a: &network::Domain,
    b: &network::Domain,
) -> (u64, u64, u64) {
    let listener = model.dispatch(request(b, "listen", 1, 0));
    let mut dial = request(a, "dial", 2, 0);
    dial.host = b.address.clone();
    dial.port = 20000;
    let client = model.dispatch(dial);
    let server = model.dispatch(request(b, "accept", 3, listener.handle));
    assert_eq!(
        (
            listener.error.as_str(),
            client.error.as_str(),
            server.error.as_str()
        ),
        ("", "", "")
    );
    (listener.handle, client.handle, server.handle)
}

#[test]
fn network_clock_ceiling_rejects_limits_before_mutation_and_never_increases() {
    let (mut model, domains) = four_domains();
    assert_eq!(
        model.limit_clock_ceiling(0).unwrap_err(),
        "unsupported Gomad network operation"
    );
    model.limit_clock_ceiling(5).unwrap();
    model.limit_clock_ceiling(100).unwrap();
    let mut invalid = request(&domains[0], "listen", 1, 0);
    invalid.deadline = 6;
    assert_eq!(
        model.dispatch(invalid).error,
        "unsupported Gomad network operation"
    );
    assert_eq!(
        model.advance(6).unwrap_err(),
        "network clock moved backwards or overflowed"
    );
    assert_eq!(model.now, 0);
}

#[test]
fn network_clock_ceiling_cannot_exclude_queued_ready_time_or_listener_deadline() {
    let (mut model, domains) = four_domains();
    let (listener, client, server) = one_pair(&mut model, &domains[0], &domains[1]);
    model.delay("a", "b", 10, false).unwrap();
    let mut write = request(&domains[0], "write", 4, client);
    write.bytes = b"x".to_vec();
    assert_eq!(model.dispatch(write).count, 1);
    assert_eq!(
        model.limit_clock_ceiling(5).unwrap_err(),
        "unsupported Gomad network operation"
    );
    model.advance(6).unwrap();
    let mut deadline = request(&domains[1], "deadline", 5, listener);
    deadline.deadline = 20;
    assert_eq!(model.dispatch(deadline).error, "");
    assert_eq!(
        model.limit_clock_ceiling(15).unwrap_err(),
        "unsupported Gomad network operation"
    );
    model.advance(16).unwrap();
    let clear = request(&domains[1], "deadline", 6, listener);
    assert_eq!(model.dispatch(clear).error, "");
    let mut read = request(&domains[1], "read", 7, server);
    read.count = 1;
    assert_eq!(model.dispatch(read).bytes, b"x");
    model.limit_clock_ceiling(16).unwrap();
}

#[test]
fn network_advance_before_positions_equal_time_without_settling_equal_readiness() {
    let (mut model, domains) = four_domains();
    let (_listener, client, server) = one_pair(&mut model, &domains[0], &domains[1]);
    model.delay("a", "b", 10, false).unwrap();
    let mut write = request(&domains[0], "write", 4, client);
    write.bytes = b"x".to_vec();
    model.dispatch(write);
    let mut read = request(&domains[1], "read", 5, server);
    read.count = 1;
    assert_eq!(model.dispatch(read).deadline, 10);
    assert_eq!(model.advance_before(10).unwrap(), None);
    assert_eq!(model.now, 10);
    assert_eq!(
        model.dispatch(request(&domains[1], "poll", 5, 0)).state,
        "pending"
    );
    assert_eq!(model.advance(10).unwrap(), Some(vec![5]));
    assert_eq!(
        model.dispatch(request(&domains[1], "poll", 5, 0)).bytes,
        b"x"
    );
}

fn replay_fixture() -> (network::Config, history::Record) {
    let (_, corpus) = corpora()
        .into_iter()
        .find(|(name, _)| name == "payload-free-history")
        .unwrap();
    (
        serde_json::from_value(corpus["config"].clone()).unwrap(),
        serde_json::from_value(corpus["final"].clone()).unwrap(),
    )
}

fn reseal(record: &mut history::Record) {
    record.snapshot.transition_sha256 =
        history::transitions_digest(record.transitions.as_deref().unwrap_or_default()).unwrap();
    record.snapshot.identity = history::snapshot_digest(&record.snapshot).unwrap();
}

type ReplayMutation = fn(&mut history::Record);

#[test]
fn network_replay_preflight_rejects_resealed_invalid_transition_shapes() {
    let cases: &[ReplayMutation] = &[
        |record| record.transitions.as_mut().unwrap()[2].outcome = "invented".into(),
        |record| record.transitions.as_mut().unwrap()[0].outcome = "timeout".into(),
        |record| record.transitions.as_mut().unwrap()[0].kind = "reset".into(),
    ];
    for mutate in cases {
        let (mut config, mut record) = replay_fixture();
        mutate(&mut record);
        reseal(&mut record);
        config.replay = Some(record);
        let error = network::Model::new(1, config)
            .err()
            .expect("invalid transition admitted");
        assert_eq!(
            error,
            "simulation network replay transition is invalid or unordered"
        );
    }
}

#[test]
fn network_replay_preflight_rejects_resealed_invalid_snapshot_shapes() {
    let cases: &[(ReplayMutation, &str)] = &[
        (
            |record| record.snapshot.nodes.as_mut().unwrap()[0].next_client_port = 0,
            "simulation network replay node snapshot is invalid or unordered",
        ),
        (
            |record| record.snapshot.links.as_mut().unwrap().swap(0, 1),
            "simulation network replay link snapshot is invalid or unordered",
        ),
        (
            |record| record.snapshot.listeners.as_mut().unwrap()[0].endpoint.port = 0,
            "simulation network replay listener snapshot is invalid or unordered",
        ),
        (
            |record| {
                record.snapshot.connections.as_mut().unwrap()[0]
                    .client
                    .incarnation = 0
            },
            "simulation network replay connection snapshot is invalid or unordered",
        ),
        (
            |record| {
                let connection = &mut record.snapshot.connections.as_mut().unwrap()[0];
                connection.closed = true;
                connection.reset = true;
                connection.client_bytes = 1;
            },
            "simulation network replay reset connection retains bytes",
        ),
        (
            |record| record.snapshot.connections.as_mut().unwrap()[0].client_bytes = 1 << 30,
            "network resources exhausted",
        ),
    ];
    for (mutate, expected) in cases {
        let (mut config, mut record) = replay_fixture();
        mutate(&mut record);
        reseal(&mut record);
        config.replay = Some(record);
        let error = network::Model::new(1, config)
            .err()
            .expect("invalid snapshot admitted");
        assert_eq!(&error, expected);
    }
}

#[test]
fn network_group_replay_mismatch_rejects_before_links_reservations_or_history_mutate() {
    let (mut recorded, domains) = four_domains();
    one_pair(&mut recorded, &domains[0], &domains[1]);
    recorded.partition("a", "b", false).unwrap();
    let record = recorded.record().unwrap();
    let (mut replay, domains) = four_domains();
    let snapshot = replay.snapshot().unwrap();
    let config = network::Config {
        nodes: snapshot
            .nodes
            .unwrap()
            .into_iter()
            .map(|node| network::Node {
                node: node.node,
                address: node.address,
            })
            .collect(),
        links: snapshot.links.unwrap(),
        limits: network::Limits {
            listeners: 128,
            connections: 128,
            bytes: 1024,
            connection_bytes: 64,
            stall_nanos: 10,
            transitions: 8192,
        },
        replay: Some(record),
    };
    replay = network::Model::new(1, config).unwrap();
    for domain in &domains {
        replay.register(domain.clone()).unwrap();
    }
    one_pair(&mut replay, &domains[0], &domains[1]);
    let before = replay.snapshot().unwrap();
    assert_eq!(
        replay
            .change_group(&["a".into()], &["b".into()], "heal", 0, false)
            .unwrap_err(),
        "simulation network replay diverged"
    );
    assert_eq!(replay.snapshot().unwrap(), before);
    assert_eq!(replay.history_reserved, 0);
    // The rejected batch must not consume the topology replay lane.
    replay.partition("a", "b", false).unwrap();
}

#[test]
fn network_partial_write_distinguishes_caller_timeout_from_connection_deadline() {
    for caller in [false, true] {
        let (mut model, domains) = four_domains_with_capacity(2, 1024);
        let (_listener, client, _server) = one_pair(&mut model, &domains[0], &domains[1]);
        model.advance(5).unwrap();
        if !caller {
            let mut deadline = request(&domains[0], "write_deadline", 4, client);
            deadline.deadline = 5;
            model.dispatch(deadline);
        }
        let mut write = request(&domains[0], "write", 5, client);
        write.bytes = b"abcdef".to_vec();
        if caller {
            write.deadline = 5;
        }
        let result = model.dispatch(write);
        assert_eq!(result.state, "completed");
        assert_eq!(
            result.error,
            if caller {
                "context deadline exceeded"
            } else {
                "i/o timeout"
            }
        );
        assert_eq!(result.count, if caller { 0 } else { 2 });
        assert_eq!(model.pending_bytes, 2);
        assert!(model.pending.is_empty());
    }
}

#[test]
fn network_global_capacity_precedes_ready_time_overflow() {
    let (mut model, domains) = four_domains_with_capacity(2, 1);
    let (_listener, client, _server) = one_pair(&mut model, &domains[0], &domains[1]);
    model.advance(1).unwrap();
    let mut first = request(&domains[0], "write", 4, client);
    first.bytes = b"a".to_vec();
    assert_eq!(model.dispatch(first).count, 1);
    model.delay("a", "b", i64::MAX as u64, false).unwrap();
    let mut second = request(&domains[0], "write", 5, client);
    second.bytes = b"b".to_vec();
    assert_eq!(model.dispatch(second).error, "network resources exhausted");
    assert_eq!(model.pending_bytes, 1);
    assert_eq!(model.transitions.last().unwrap().outcome, "capacity");
}

#[test]
fn network_full_close_snapshot_keeps_pair_lifecycle_marker_until_reset() {
    let (mut model, domains) = four_domains();
    let (_listener, client, server) = one_pair(&mut model, &domains[0], &domains[1]);
    model.dispatch(request(&domains[0], "close", 4, client));
    model.dispatch(request(&domains[1], "close", 5, server));
    let snapshot = model.snapshot().unwrap();
    assert!(!snapshot.connections.as_ref().unwrap()[0].closed);
    assert!(!snapshot.connections.as_ref().unwrap()[0].reset);
    model.reset(client).unwrap();
    let snapshot = model.snapshot().unwrap();
    assert!(snapshot.connections.as_ref().unwrap()[0].closed);
    assert!(snapshot.connections.as_ref().unwrap()[0].reset);
}

#[test]
fn network_snapshot_uses_go_concatenated_keys_for_separator_node_names() {
    let config:network::Config=serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"a"},{"node":"a\u{0}","address":"a0"},{"node":"z","address":"z"}],
        "links":[{"from":"a\u{0}","to":"z","enabled":true,"delay_nanos":0},{"from":"a","to":"z","enabled":true,"delay_nanos":0}],
        "limits":{"listeners":128,"connections":128,"bytes":1024,"connection_bytes":64,"stall_nanos":10,"transitions":8192}
    })).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    let a = network::Domain {
        run: 1,
        node: "a".into(),
        address: "a".into(),
        incarnation: 1,
    };
    let separator = network::Domain {
        run: 1,
        node: "a\0".into(),
        address: "a0".into(),
        incarnation: 1,
    };
    model.register(a.clone()).unwrap();
    model.register(separator.clone()).unwrap();
    model.dispatch(request(&a, "listen", 1, 0));
    model.dispatch(request(&separator, "listen", 2, 0));
    let snapshot = model.snapshot().unwrap();
    assert_eq!(
        snapshot
            .links
            .unwrap()
            .iter()
            .map(|link| link.from.as_str())
            .collect::<Vec<_>>(),
        vec!["a\0", "a"]
    );
    assert_eq!(
        snapshot
            .listeners
            .unwrap()
            .iter()
            .map(|listener| listener.endpoint.node.as_str())
            .collect::<Vec<_>>(),
        vec!["a\0", "a"]
    );
}

fn config_from_model(model: &network::Model) -> network::Config {
    let snapshot = model.snapshot().unwrap();
    network::Config {
        nodes: snapshot
            .nodes
            .unwrap()
            .into_iter()
            .map(|node| network::Node {
                node: node.node,
                address: node.address,
            })
            .collect(),
        links: snapshot.links.unwrap(),
        limits: network::Limits {
            listeners: 128,
            connections: 128,
            bytes: 1024,
            connection_bytes: 64,
            stall_nanos: 10,
            transitions: 8192,
        },
        replay: None,
    }
}

#[test]
fn network_partial_group_replay_rolls_back_consumed_lane_and_reservation() {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    for (from, to) in [("a", "d"), ("c", "b")] {
        config.links.push(history::Link {
            from: from.into(),
            to: to.into(),
            enabled: true,
            delay_nanos: 0,
        });
    }
    config
        .links
        .sort_by_key(|link| format!("{}\0{}", link.from, link.to));
    let mut recorded = network::Model::new(1, config.clone()).unwrap();
    for domain in &domains {
        recorded.register(domain.clone()).unwrap();
    }
    let (_, client, _) = one_pair(&mut recorded, &domains[0], &domains[1]);
    let mut write = request(&domains[0], "write", 4, client);
    write.bytes = b"held".to_vec();
    recorded.dispatch(write.clone());
    recorded
        .change_group(
            &["a".into(), "c".into()],
            &["b".into()],
            "partition",
            0,
            false,
        )
        .unwrap();
    config.replay = Some(recorded.record().unwrap());
    let mut replay = network::Model::new(1, config).unwrap();
    for domain in &domains {
        replay.register(domain.clone()).unwrap();
    }
    one_pair(&mut replay, &domains[0], &domains[1]);
    replay.dispatch(write);
    let before = replay.snapshot().unwrap();
    assert_eq!(
        replay
            .change_group(
                &["a".into()],
                &["b".into(), "d".into()],
                "partition",
                0,
                false
            )
            .unwrap_err(),
        "simulation network replay diverged"
    );
    assert_eq!(replay.snapshot().unwrap(), before);
    assert_eq!(replay.history_reserved, 0);
    assert_eq!(replay.pending_bytes, 4);
    replay
        .change_group(
            &["a".into(), "c".into()],
            &["b".into()],
            "partition",
            0,
            false,
        )
        .unwrap();
    assert_eq!(replay.history_reserved, 1);
}

fn fill_listener_backlog(
    model: &mut network::Model,
    a: &network::Domain,
    b: &network::Domain,
) -> u64 {
    let listener = model.dispatch(request(b, "listen", 1, 0));
    for id in 100..164 {
        let mut dial = request(a, "dial", id, 0);
        dial.host = b.address.clone();
        dial.port = 20000;
        assert_eq!(model.dispatch(dial).error, "");
    }
    let server = model.dispatch(request(b, "accept", 2, listener.handle));
    let mut extra = request(a, "dial", 164, 0);
    extra.host = b.address.clone();
    extra.port = 20000;
    assert_eq!(model.dispatch(extra).error, "");
    server.handle
}

#[test]
fn network_reset_signals_network_waiters_after_replay_divergence() {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    let mut recorded = network::Model::new(1, config.clone()).unwrap();
    for domain in &domains {
        recorded.register(domain.clone()).unwrap();
    }
    let server = fill_listener_backlog(&mut recorded, &domains[0], &domains[1]);
    let mut write = request(&domains[0], "write", 200, 2);
    write.bytes = b"abc".to_vec();
    assert_eq!(recorded.dispatch(write).count, 3);
    let mut read = request(&domains[1], "read", 201, server);
    read.count = 3;
    assert_eq!(recorded.dispatch(read).bytes, b"abc");
    recorded.reset(2).unwrap();
    config.replay = Some(recorded.record().unwrap());
    let mut replay = network::Model::new(1, config).unwrap();
    for domain in &domains {
        replay.register(domain.clone()).unwrap();
    }
    fill_listener_backlog(&mut replay, &domains[0], &domains[1]);
    let mut blocked = request(&domains[0], "dial", 1000, 0);
    blocked.host = "b".into();
    blocked.port = 20000;
    assert_eq!(replay.dispatch(blocked).state, "pending");
    let mut wrong = request(&domains[0], "write", 200, 2);
    wrong.bytes = b"x".to_vec();
    assert_eq!(
        replay.dispatch(wrong).error,
        "simulation network replay diverged"
    );
    assert_eq!(replay.ready(), None);
    replay.reset(2).unwrap();
    assert_eq!(replay.ready(), Some(vec![1000]));
    assert_eq!(
        replay.dispatch(request(&domains[0], "poll", 1000, 0)).error,
        "simulation network replay diverged"
    );
}

#[test]
fn network_mixed_broker_prefix_preserves_six_domains_and_request_bases() {
    for (name, mut corpus) in corpora() {
        if !name.starts_with("broker-strict-topology-") {
            continue;
        }
        assert_eq!(corpus["steps"][22]["input"]["action"], "dispatch");
        assert_eq!(corpus["steps"][22]["input"]["request"]["op"], "dial");
        corpus["steps"].as_array_mut().unwrap().truncate(22);
        replay_corpus(&name, &corpus, false).unwrap();
    }
}

#[test]
fn network_listen_collision_precedes_history_capacity() {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    config.limits.listeners = 1;
    let mut model = network::Model::new(1, config).unwrap();
    model.register(domains[0].clone()).unwrap();
    let mut listen = request(&domains[0], "listen", 1, 0);
    listen.port = 20000;
    assert_eq!(model.dispatch(listen.clone()).error, "");
    listen.request_id = 2;
    assert_eq!(
        model.dispatch(listen.clone()).error,
        "address already in use"
    );
    assert_eq!(model.transitions.last().unwrap().outcome, "closed");
    listen.request_id = 3;
    listen.port = 20001;
    assert_eq!(model.dispatch(listen).error, "network resources exhausted");
    assert_eq!(model.transitions.last().unwrap().outcome, "capacity");
}

#[test]
fn network_cancel_parked_dial_records_deadline_outcome() {
    let (mut model, domains) = four_domains();
    fill_listener_backlog(&mut model, &domains[0], &domains[1]);
    let mut dial = request(&domains[0], "dial", 1000, 0);
    dial.host = "b".into();
    dial.port = 20000;
    assert_eq!(model.dispatch(dial).state, "pending");
    let before = model.transitions.len();
    model.cancel(&domains[0], 1000).unwrap();
    assert_eq!(model.transitions.len(), before + 1);
    let transition = model.transitions.last().unwrap();
    assert_eq!(transition.kind, "dial");
    assert_eq!(transition.outcome, "deadline");
    assert_eq!(transition.source.port, 0);
    assert_eq!(transition.destination.node, "b");
    assert_eq!(transition.destination.incarnation, 0);
    assert_eq!(
        model.dispatch(request(&domains[0], "poll", 1000, 0)).error,
        "context canceled"
    );
}

#[test]
fn network_group_ignores_duration_for_partition_and_heal() {
    let (mut model, _) = four_domains();
    model
        .change_group(&["a".into()], &["b".into()], "partition", u64::MAX, false)
        .unwrap();
    assert_eq!(model.transitions.last().unwrap().delay_nanos, 0);
    model
        .change_group(&["a".into()], &["b".into()], "heal", u64::MAX, true)
        .unwrap();
    assert_eq!(model.transitions.last().unwrap().delay_nanos, 0);
    let before = model.snapshot().unwrap();
    assert!(
        model
            .change_group(&["a".into()], &["b".into()], "delay", u64::MAX, true)
            .is_err()
    );
    assert_eq!(model.snapshot().unwrap(), before);
}

#[test]
fn network_reset_after_finish_records_crash_without_resetting_pair() {
    let (mut model, domains) = four_domains();
    let (_, client, _) = one_pair(&mut model, &domains[0], &domains[1]);
    model.finish().unwrap();
    let before = model.snapshot().unwrap();
    let count = model.transitions.len();
    assert_eq!(model.reset(client).unwrap_err(), "stale file handle");
    assert_eq!(model.snapshot().unwrap().connections, before.connections);
    assert_eq!(model.transitions.len(), count + 1);
    assert_eq!(model.transitions.last().unwrap().kind, "crash");
}

#[test]
fn network_embedded_separator_link_keys_preserve_go_alias_identity() {
    let config: network::Config = serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"1"},{"node":"a\u{0}b","address":"2"},{"node":"b\u{0}c","address":"3"},{"node":"c","address":"4"}],
        "links":[{"from":"a","to":"b\u{0}c","enabled":true,"delay_nanos":0}],
        "limits":{"listeners":1,"connections":1,"bytes":64,"connection_bytes":64,"stall_nanos":10,"transitions":64}
    })).unwrap();
    let mut model = network::Model::new(1, config).unwrap();
    model
        .change_group(&["a\u{0}b".into()], &["c".into()], "partition", 0, false)
        .unwrap();
    let links = model.snapshot().unwrap().links.unwrap();
    assert_eq!(links[0].from, "a");
    assert_eq!(links[0].to, "b\u{0}c");
    assert!(!links[0].enabled);
}

#[test]
fn network_dial_advance_before_settles_caller_deadline_before_later_stall() {
    for before in [false, true] {
        let (mut model, domains) = four_domains();
        model.partition("a", "b", false).unwrap();
        let mut dial = request(&domains[0], "dial", 1, 0);
        dial.host = "b".into();
        dial.port = 20000;
        dial.deadline = 5;
        assert_eq!(model.dispatch(dial).deadline, 5);
        if before {
            model.advance_before(100).unwrap();
        } else {
            model.advance(100).unwrap();
        }
        let result = model.dispatch(request(&domains[0], "poll", 1, 0));
        assert_eq!(
            result.error,
            if before {
                "context deadline exceeded"
            } else {
                "connection timed out"
            }
        );
        assert_eq!(
            model.transitions.last().unwrap().outcome,
            if before { "deadline" } else { "timeout" }
        );
        assert_eq!(model.history_reserved, 0);
        assert_eq!(model.next_deadline(), 0);
    }
}

#[test]
fn network_mixed_dial_and_pair_partition_admission_is_atomic_and_expiry_is_ordered() {
    for ceiling in [6, 7] {
        let (initial, domains) = four_domains();
        let mut config = config_from_model(&initial);
        config.limits.transitions = ceiling;
        config.links.push(history::Link {
            from: "c".into(),
            to: "b".into(),
            enabled: false,
            delay_nanos: 0,
        });
        config
            .links
            .sort_by_cached_key(|link| format!("{}\0{}", link.from, link.to));
        let mut model = network::Model::new(1, config).unwrap();
        for domain in &domains {
            model.register(domain.clone()).unwrap();
        }
        let (_, client, _) = one_pair(&mut model, &domains[0], &domains[1]);
        let mut dial = request(&domains[2], "dial", (3 << 48) + 1, 0);
        dial.host = "b".into();
        dial.port = 20000;
        assert_eq!(model.dispatch(dial).state, "pending");
        let mut write = request(&domains[0], "write", (1 << 48) + 4, client);
        write.bytes = b"held".to_vec();
        assert_eq!(model.dispatch(write).count, 4);
        let snapshot = model.snapshot().unwrap();
        let action = model.partition("a", "b", false);
        if ceiling == 6 {
            assert_eq!(action.unwrap_err(), "network resources exhausted");
            assert_eq!(model.snapshot().unwrap(), snapshot);
            assert_eq!(model.history_reserved, 1);
            assert_eq!(model.pending_bytes, 4);
            assert_eq!(model.next_deadline(), 11);
        } else {
            action.unwrap();
            assert_eq!(model.history_reserved, 2);
            model.advance(11).unwrap();
            assert_eq!(model.history_reserved, 0);
            assert_eq!(model.pending_bytes, 0);
            assert_eq!(model.transitions.len(), 7);
            assert_eq!(model.transitions[5].kind, "dial");
            assert_eq!(model.transitions[5].source.node, "c");
            assert_eq!(model.transitions[5].outcome, "timeout");
            assert_eq!(model.transitions[6].kind, "reset");
            assert_eq!(model.transitions[6].connection, 1);
        }
    }
}

#[test]
fn network_dial_clock_ceiling_and_heal_repartition_overflow_reject_atomically() {
    let (mut model, domains) = four_domains();
    model.limit_clock_ceiling(20).unwrap();
    model.partition("a", "b", false).unwrap();
    let mut dial = request(&domains[0], "dial", 1, 0);
    dial.host = "b".into();
    dial.port = 20000;
    assert_eq!(model.dispatch(dial).deadline, 11);
    assert_eq!(
        model.limit_clock_ceiling(10).unwrap_err(),
        "unsupported Gomad network operation"
    );
    model.advance(10).unwrap();
    model.heal("a", "b", false).unwrap();
    assert_eq!(model.history_reserved, 0);
    assert_eq!(model.next_deadline(), 0);
    let snapshot = model.snapshot().unwrap();
    assert_eq!(
        model.partition("a", "b", false).unwrap_err(),
        "unsupported Gomad network operation"
    );
    assert_eq!(model.snapshot().unwrap(), snapshot);
    assert_eq!(model.history_reserved, 0);
    assert_eq!(model.next_deadline(), 0);
}

#[test]
fn network_pending_dial_poll_keeps_wait_snapshot_until_ready_retries() {
    let (mut model, domains) = four_domains();
    model.partition("a", "b", false).unwrap();
    let mut dial = request(&domains[0], "dial", 1, 0);
    dial.host = "b".into();
    dial.port = 20000;
    assert_eq!(model.dispatch(dial).deadline, 11);
    model.advance(5).unwrap();
    model.heal("a", "b", false).unwrap();
    model.partition("a", "b", false).unwrap();
    assert_eq!(model.next_deadline(), 16);
    assert_eq!(
        model.dispatch(request(&domains[0], "poll", 1, 0)).deadline,
        11
    );
    assert_eq!(model.ready(), None);
    assert_eq!(
        model.dispatch(request(&domains[0], "poll", 1, 0)).deadline,
        16
    );
}

#[test]
fn network_cancel_backlog_dial_preserves_history_admission_failure() {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    config.limits.transitions = 67;
    let mut model = network::Model::new(1, config).unwrap();
    for domain in &domains {
        model.register(domain.clone()).unwrap();
    }
    fill_listener_backlog(&mut model, &domains[0], &domains[1]);
    let mut dial = request(&domains[0], "dial", (1 << 48) + 1000, 0);
    dial.host = "b".into();
    dial.port = 20000;
    assert_eq!(model.dispatch(dial).state, "pending");
    model.cancel(&domains[0], (1 << 48) + 1000).unwrap();
    assert_eq!(model.transitions.len(), 67);
    assert_eq!(
        model
            .dispatch(request(&domains[0], "poll", (1 << 48) + 1000, 0))
            .error,
        "network resources exhausted"
    );
}

#[test]
fn network_two_disjoint_pair_stalls_keep_global_request_ids_and_incarnations_separate() {
    let (mut model, domains) = four_domains();
    let mut handles = Vec::new();
    for (left, right, slot) in [(0, 1, 1_u64), (2, 3, 3)] {
        let base = slot << 48;
        let listener = model.dispatch(request(
            &domains[right],
            "listen",
            ((slot + 1) << 48) + 1,
            0,
        ));
        let mut dial = request(&domains[left], "dial", base + 1, 0);
        dial.host = domains[right].address.clone();
        dial.port = 20000;
        let client = model.dispatch(dial);
        let server = model.dispatch(request(
            &domains[right],
            "accept",
            ((slot + 1) << 48) + 2,
            listener.handle,
        ));
        model
            .partition(&domains[left].node, &domains[right].node, false)
            .unwrap();
        let mut write = request(&domains[left], "write", base + 2, client.handle);
        write.bytes = b"held".to_vec();
        assert_eq!(model.dispatch(write).count, 4);
        let mut read = request(
            &domains[right],
            "read",
            ((slot + 1) << 48) + 3,
            server.handle,
        );
        read.count = 4;
        assert_eq!(model.dispatch(read).state, "pending");
        handles.push(server.handle);
    }
    model.revoke(&domains[0], false).unwrap();
    assert_eq!(model.history_reserved, 1);
    let mut restarted = domains[0].clone();
    restarted.incarnation = 2;
    model.register(restarted).unwrap();
    model.advance(11).unwrap();
    assert_eq!(model.history_reserved, 0);
    assert_eq!(model.pending_bytes, 0);
    assert_eq!(
        model
            .transitions
            .iter()
            .filter(|transition| transition.kind == "reset" && transition.outcome == "timeout")
            .map(|transition| transition.connection)
            .collect::<Vec<_>>(),
        vec![2]
    );
    assert_eq!(
        model
            .dispatch(request(&domains[1], "poll", (2 << 48) + 3, 0))
            .error,
        "use of closed network connection"
    );
    assert_eq!(
        model
            .dispatch(request(&domains[3], "poll", (4 << 48) + 3, 0))
            .error,
        "connection timed out"
    );
    assert_eq!(handles.len(), 2);
}

fn completed_dial_never_rearms(capacity: bool, consume_before_partition: bool) {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    config.limits.connections = 1;
    let mut model = network::Model::new(1, config).unwrap();
    for domain in &domains {
        model.register(domain.clone()).unwrap();
    }
    if capacity {
        one_pair(&mut model, &domains[0], &domains[1]);
    }
    model.partition("a", "b", false).unwrap();
    let mut dial = request(&domains[0], "dial", 10, 0);
    dial.host = "b".into();
    dial.port = 20000;
    assert_eq!(model.dispatch(dial).state, "pending");
    model.heal("a", "b", false).unwrap();
    assert_eq!(model.ready(), Some(vec![10]));
    let expected = if capacity {
        "network resources exhausted"
    } else {
        "connection refused"
    };
    if consume_before_partition {
        assert_eq!(
            model.dispatch(request(&domains[0], "poll", 10, 0)).error,
            expected
        );
    }
    model.partition("a", "b", false).unwrap();
    assert_eq!(model.next_deadline(), 0);
    assert_eq!(model.history_reserved, 0);
    let before = model.transitions.len();
    model.advance(100).unwrap();
    assert_eq!(model.transitions.len(), before);
    if !consume_before_partition {
        assert_eq!(
            model.dispatch(request(&domains[0], "poll", 10, 0)).error,
            expected
        );
    }
}

#[test]
fn network_completed_refused_dial_never_rearms_before_poll() {
    completed_dial_never_rearms(false, false);
}
#[test]
fn network_completed_refused_dial_never_rearms_after_poll() {
    completed_dial_never_rearms(false, true);
}
#[test]
fn network_completed_capacity_dial_never_rearms_before_poll() {
    completed_dial_never_rearms(true, false);
}
#[test]
fn network_completed_capacity_dial_never_rearms_after_poll() {
    completed_dial_never_rearms(true, true);
}

fn partial_expiry_revoke_reuse(divergent: bool) {
    let (initial, domains) = four_domains();
    let mut config = config_from_model(&initial);
    config.links[0].enabled = false;
    let mut recorded = network::Model::new(1, config.clone()).unwrap();
    for domain in &domains {
        recorded.register(domain.clone()).unwrap();
    }
    let admit = |model: &mut network::Model| {
        for id in [10, 11] {
            let mut dial = request(&domains[0], "dial", id, 0);
            dial.host = "b".into();
            dial.port = 20000 + (id - 10) as i64;
            assert_eq!(model.dispatch(dial).state, "pending");
        }
    };
    admit(&mut recorded);
    assert_eq!(recorded.advance(11).unwrap(), Some(vec![10, 11]));
    let mut record = recorded.record().unwrap();
    if divergent {
        record
            .transitions
            .as_mut()
            .unwrap()
            .iter_mut()
            .find(|transition| transition.destination.port == 20001)
            .unwrap()
            .outcome = "deadline".into();
        reseal(&mut record);
    }
    config.replay = Some(record);
    let mut model = network::Model::new(1, config).unwrap();
    for domain in &domains {
        model.register(domain.clone()).unwrap();
    }
    admit(&mut model);
    if divergent {
        assert_eq!(
            model.advance(11).unwrap_err(),
            "simulation network replay diverged"
        );
        assert_eq!(model.transitions.len(), 1);
        assert_eq!(model.history_reserved, 1);
        assert!(
            model
                .pending
                .values()
                .all(|pending| pending.result.is_none())
        );
    } else {
        assert_eq!(model.advance(11).unwrap(), Some(vec![10, 11]));
        assert_eq!(model.transitions.len(), 2);
    }
    model.revoke(&domains[0], false).unwrap();
    assert_eq!(model.history_reserved, 0);
    assert!(model.pending.is_empty());
    let mut restarted = domains[0].clone();
    restarted.incarnation = 2;
    model.register(restarted.clone()).unwrap();
    let mut dial = request(&restarted, "dial", 10, 0);
    dial.host = "b".into();
    dial.port = 20000;
    let result = model.dispatch(dial);
    if divergent {
        assert_eq!(result.error, "simulation network replay diverged");
        assert_eq!(result.state, "completed");
        assert_eq!(model.history_reserved, 0);
    } else {
        assert_eq!(result.error, "");
        assert_eq!(result.state, "pending");
        assert_eq!(result.deadline, 22);
        assert_eq!(model.history_reserved, 1);
    }
}

#[test]
fn network_completed_expiry_revoke_reuse_starts_fresh_dial() {
    partial_expiry_revoke_reuse(false);
}
#[test]
fn network_partial_expiry_divergence_revoke_reuse_preserves_global_error() {
    partial_expiry_revoke_reuse(true);
}
