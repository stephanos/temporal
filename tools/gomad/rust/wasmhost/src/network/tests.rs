use super::{Config, Domain, Model, Request, history};

fn independent_listeners() -> Model {
    let config: Config = serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"a"},{"node":"z","address":"z"}],
        "links":[],
        "limits":{"listeners":4,"connections":4,"bytes":64,"connection_bytes":64,"stall_nanos":10,"transitions":8}
    })).unwrap();
    let mut model = Model::new(1, config).unwrap();
    for node in ["z", "a"] {
        let domain = Domain {
            run: 1,
            node: node.into(),
            incarnation: 1,
            address: node.into(),
        };
        model.register(domain.clone()).unwrap();
        let result = model.dispatch(Request {
            domain,
            op: "listen".into(),
            handle: 0,
            host: String::new(),
            port: 7,
            bytes: vec![],
            count: 0,
            deadline: 0,
            request_id: 1,
        });
        assert_eq!(result.error, "");
    }
    model
}

#[test]
fn record_hashes_each_digest_once() {
    let model = independent_listeners();
    let mut transitions = model.transitions.clone();
    history::canonicalize(&mut transitions);
    assert_ne!(model.transitions, transitions);
    history::DIGEST_COUNTS.with(|count| count.set((0, 0)));
    model.record().unwrap();
    assert_eq!(history::DIGEST_COUNTS.with(std::cell::Cell::get), (1, 1));
}

#[test]
fn record_preserves_history_and_snapshot_wire_validation() {
    for field in ["kind", "outcome", "source", "destination"] {
        let mut model = independent_listeners();
        let oversized = "x".repeat((1 << 20) + 1);
        match field {
            "kind" => model.transitions[0].kind = oversized,
            "outcome" => model.transitions[0].outcome = oversized,
            "source" => model.transitions[0].source.node = oversized,
            "destination" => model.transitions[0].destination.address = oversized,
            _ => unreachable!(),
        }
        assert_eq!(
            model.snapshot().unwrap_err(),
            "simulation network wire string exceeds its byte limit",
            "{field}"
        );
        assert_eq!(
            model.record().unwrap_err(),
            "simulation network wire string exceeds its byte limit",
            "{field}"
        );
        assert!(model.record_check().is_err(), "{field}");
    }
    let mut model = independent_listeners();
    model.nodes.get_mut("a").unwrap().address = "x".repeat((1 << 20) + 1);
    assert!(model.record_check().is_err());
}

/// `record_check` fails exactly when `record` does, at both wire limits
/// and on either side of the total-length bound.
#[test]
fn record_check_matches_record_failure_at_every_wire_limit() {
    let model = independent_listeners();
    assert!(model.record().is_ok() && model.record_check().is_ok());
    let base = model.transitions[0].clone();
    for (length, count, valid) in [
        (1 << 20, 127, true),
        (1 << 20, 128, false),
        ((1 << 20) + 1, 1, false),
        (100, 1, true),
    ] {
        let mut model = independent_listeners();
        let mut transition = base.clone();
        transition.outcome = "x".repeat(length);
        model.transitions = vec![transition; count];
        assert_eq!(model.record().is_ok(), valid, "{length} x {count}");
        assert_eq!(model.record_check().is_ok(), valid, "{length} x {count}");
        let mut inspection = history::InspectionHistory::default();
        for transition in &model.transitions {
            inspection.append(transition);
        }
        assert_eq!(
            inspection.check(),
            model.record_check(),
            "inspection {length} x {count}"
        );
    }
    let mut model = independent_listeners();
    model.nodes.get_mut("a").unwrap().node = "x".repeat((1 << 20) + 1);
    assert!(model.record().is_err());
    assert!(model.record_check().is_err());
}

#[test]
fn inspection_lane_chains_preserve_canonical_order_and_bind_every_transition() {
    let mut model = independent_listeners();
    let a = model.transitions[0].clone();
    let b = model.transitions[1].clone();
    let mut changed = a.clone();
    changed.outcome = "closed".into();
    let digest = |transitions: &[history::Transition]| {
        let mut chain = history::InspectionHistory::default();
        for transition in transitions {
            chain.append(transition);
        }
        chain.digest().unwrap()
    };
    assert_eq!(
        digest(&[a.clone(), b.clone(), changed.clone()]),
        digest(&[b, a.clone(), changed.clone()])
    );
    assert_ne!(
        digest(&[a.clone(), changed.clone()]),
        digest(&[changed, a.clone()])
    );
    assert_ne!(
        digest(std::slice::from_ref(&a)),
        digest(&[a.clone(), a.clone()])
    );
    let mut split = a.clone();
    split.source.node = "ab".into();
    split.source.address = "c".into();
    let mut other = split.clone();
    other.source.node = "a".into();
    other.source.address = "bc".into();
    assert_ne!(digest(&[split]), digest(&[other]));
    let before = model.inspection_digest().unwrap();
    assert_eq!(before, model.inspection_digest_eager().unwrap());
    let mut invalid = a.clone();
    invalid.outcome = "invalid".into();
    assert!(model.commit_batch(&[a, invalid], 0, 0).is_err());
    assert_eq!(
        model.inspection_digest().unwrap(),
        before,
        "rollback leaked a lane head"
    );
    model.nodes.get_mut("a").unwrap().next_client_port += 1;
    assert_ne!(
        model.inspection_digest().unwrap(),
        before,
        "snapshot state was not bound"
    );
}

#[test]
fn record_validates_original_history_before_canonical_reordering_changes_error_precedence() {
    let mut model = independent_listeners();
    let mut transition = model.transitions[0].clone();
    transition.outcome = "x".repeat(1 << 20);
    model.transitions = vec![transition; 128];
    let mut first_lane = model.transitions[0].clone();
    first_lane.source.node = "a".into();
    first_lane.outcome.push('x');
    model.transitions.push(first_lane);
    let expected = "simulation network wire value exceeds its byte limit";
    assert_eq!(model.snapshot().unwrap_err(), expected);
    assert_eq!(model.record().unwrap_err(), expected);
    assert!(model.record_check().is_err());
    let mut inspection = history::InspectionHistory::default();
    for transition in &model.transitions {
        inspection.append(transition);
    }
    assert_eq!(inspection.check().unwrap_err(), expected);
}

#[test]
fn implicit_listen_exhaustion_records_zero_port_binary_identity() {
    let config: Config = serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"a"}],
        "links":[],
        "limits":{"listeners":4,"connections":4,"bytes":64,"connection_bytes":64,"stall_nanos":10,"transitions":8}
    })).unwrap();
    let mut model = Model::new(1, config).unwrap();
    let domain = Domain {
        run: 1,
        node: "a".into(),
        incarnation: 1,
        address: "a".into(),
    };
    model.register(domain.clone()).unwrap();
    model.nodes.get_mut("a").unwrap().next_listener_port = 65535;
    let listen = |id| Request {
        domain: domain.clone(),
        op: "listen".into(),
        handle: 0,
        host: String::new(),
        port: 0,
        bytes: Vec::new(),
        count: 0,
        deadline: 0,
        request_id: id,
    };
    let last = model.dispatch(listen(1));
    assert_eq!(last.error, "");
    assert_eq!(last.local.port, 65535);
    let exhausted = model.dispatch(listen(2));
    assert_eq!(exhausted.error, "network resources exhausted");
    let failure = model.transitions.last().unwrap();
    assert_eq!(
        failure,
        &history::Transition {
            kind: "listen".into(),
            source: history::Endpoint {
                node: "a".into(),
                incarnation: 1,
                address: "a".into(),
                port: 0
            },
            outcome: "capacity".into(),
            ..Default::default()
        }
    );
    assert_eq!(
        history::lane(failure),
        concat!("listener\0a\0", "00000000000000000001", "\0a\0", "00000")
    );
    assert_eq!(
        history::transitions_digest(std::slice::from_ref(failure)).unwrap(),
        "sha256:4bd4c13af7629a8cf6f8b35992963f1d5b44c43be9d206040c3ecc5e26b8116f"
    );
    let record = model.record().unwrap();
    assert_eq!(
        record.snapshot.transition_sha256,
        "sha256:55b912a7e4d21194a609401aeaf271f491b2637ac1ecb09e05d4eba519ec9fcc"
    );
    assert_eq!(
        record.snapshot.identity,
        "sha256:10cff68de9f5ea8ab8b6527ba17003828fcff6305a3f4690993f30997d71110a"
    );
    assert_eq!(record.snapshot.nodes.unwrap()[0].next_listener_port, 65536);
}
