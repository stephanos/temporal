use super::digest::hex;
use super::order::{choice_rank, hmac_sha256};
use super::*;

fn limits() -> Limits {
    Limits {
        max_requests: 100,
        max_events: 100,
        max_queued_events: 100,
        max_transitions: 500,
        max_payload_bytes: 1 << 20,
        max_string_bytes: 1024,
    }
}

fn request(key: &str) -> Request {
    Request {
        kind: "wait".into(),
        resource: ResourceId {
            adapter: "memory".into(),
            kind: "cell".into(),
            key: key.into(),
        },
        priority: 0,
        payload: Vec::new(),
    }
}

#[test]
fn exhausted_transition_revision_refuses_every_mutation_before_state_changes() {
    let mut model = Model::new(Config {
        seed: 7,
        limits: limits(),
    })
    .unwrap();
    let id = model.register(request("pending")).unwrap();
    model.next_transition = u64::MAX;
    let before = model.state_digest();
    assert!(model.register(request("new")).is_err());
    assert!(
        model
            .ready(Readiness {
                request_id: id,
                at: INITIAL_TIME + 1,
                kind: "timer".into(),
                payload: vec![],
                equivalence_class: String::new()
            })
            .is_err()
    );
    assert!(model.cancel(id).is_err());
    assert!(model.quiesce().is_err());
    assert_eq!(model.state_digest(), before);
    assert_eq!(model.next_transition, u64::MAX);
}

fn decode_hex(text: &str) -> Vec<u8> {
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

#[test]
fn world_hmac_matches_rfc4231() {
    // RFC 4231 test cases 1 and 2.
    assert_eq!(
        hex(&hmac_sha256(&[0x0b; 20], &[b"Hi There"])),
        "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"
    );
    assert_eq!(
        hex(&hmac_sha256(
            b"Jefe",
            &[b"what do ya want ", b"for nothing?"]
        )),
        "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"
    );
    // RFC 4231 test case 6: a key longer than one block is hashed first.
    assert_eq!(
        hex(&hmac_sha256(
            &[0xaa; 131],
            &[b"Test Using Larger Than Block-Size Key - Hash Key First"]
        )),
        "60e431591ee0b67f0d8a26aacbf5b77f8e0bc6213728c5140546040f0ee37f54"
    );
}

#[test]
fn world_choice_rank_matches_go_vectors() {
    // Produced by Go `choiceRank` (internal/model/world/choice.go).
    for (seed, class, event, expected) in [
        (
            0,
            "exchangeable",
            1,
            "25613d06e5e10c10308c173981de90cb59d72d532965bb43ec61244e527a5f01",
        ),
        (
            7,
            "x",
            42,
            "f42bf35fc2608db6f226758dd47dee92aa145eb0353c8570979262ff35503116",
        ),
        (
            u64::MAX,
            "network",
            u64::MAX - 1,
            "8d89fed0092f6f11c100399441c049cd138a6246e4a87c609af57635006ac905",
        ),
        (
            41,
            "üñî",
            3,
            "a1c8f07d1efaa89603fef2ea7fbfebb27648399459d9c41b90681bfab185795b",
        ),
        (
            1,
            "",
            1,
            "205a5ac465d13452c932de3e6c7f07b2fd5ed8db365d99113d45d82e432d9fc4",
        ),
    ] {
        assert_eq!(
            choice_rank(seed, class, event).to_vec(),
            decode_hex(expected),
            "seed {seed} class {class:?} event {event}"
        );
    }
}

#[test]
fn world_identity_and_transition_exhaustion_do_not_wrap_or_mutate() {
    // Go capacity_test.go TestWorldIdentityAndTransitionExhaustionDoNotWrap,
    // which the public-API corpus cannot reach.
    let config = Config {
        seed: 1,
        limits: limits(),
    };
    let mut model = Model::new(config.clone()).unwrap();
    model.next_request_id = u64::MAX;
    let before = model.state_digest();
    let error = model.register(request("a")).unwrap_err();
    assert_eq!(
        error.to_string(),
        format!(
            "World capacity exhausted: request-ids limit={0} used={0} delta=1",
            u64::MAX
        )
    );
    assert_eq!(model.state_digest(), before);

    let mut model = Model::new(config.clone()).unwrap();
    let id = model.register(request("a")).unwrap();
    model.next_event_id = u64::MAX;
    let before = model.state_digest();
    let ready = Readiness {
        request_id: id,
        at: INITIAL_TIME,
        kind: "done".into(),
        payload: Vec::new(),
        equivalence_class: String::new(),
    };
    assert_eq!(model.ready(ready).unwrap_err().class(), "capacity");
    assert_eq!(model.state_digest(), before);

    let mut model = Model::new(config).unwrap();
    model.next_transition = u64::MAX;
    let before = model.state_digest();
    assert_eq!(
        model.register(request("a")).unwrap_err().to_string(),
        format!(
            "World capacity exhausted: transition-sequences limit={0} used={0} delta=1",
            u64::MAX
        )
    );
    assert_eq!(model.state_digest(), before);
}

#[test]
fn world_quiesce_delivers_one_instant_in_order_and_cancel_removes_queued() {
    let mut model = Model::new(Config {
        seed: 7,
        limits: limits(),
    })
    .unwrap();
    let a = model.register(request("b")).unwrap();
    let b = model.register(request("a")).unwrap();
    let c = model.register(request("c")).unwrap();
    for (id, at) in [(a, 10), (b, 10), (c, 11)] {
        model
            .ready(Readiness {
                request_id: id,
                at: INITIAL_TIME + at,
                kind: "done".into(),
                payload: vec![1],
                equivalence_class: String::new(),
            })
            .unwrap();
    }
    assert_eq!(
        model.cancel(c).unwrap(),
        Cancellation {
            request_id: c,
            event_id: 3,
            status: CancelStatus::Won
        }
    );
    let delivered = model.quiesce().unwrap();
    assert_eq!(delivered.kind, QuiescenceKind::Delivered);
    assert_eq!(
        delivered
            .deliveries
            .iter()
            .map(|delivery| delivery.request_id)
            .collect::<Vec<_>>(),
        vec![b, a]
    );
    assert_eq!(model.now(), INITIAL_TIME + 10);
    assert_eq!(model.quiesce().unwrap().kind, QuiescenceKind::Idle);
    assert_eq!(model.cardinality().payload_bytes, 3 * 2 + 2);
}

/// The retained transition frames reproduce the uncached Go `stateDigest`
/// after every mutation, refusal and cap fallback.
#[test]
fn cached_state_digest_matches_uncached_after_every_mutation() {
    for maximum in [usize::MAX, 0, 600, 4096] {
        let mut model = Model::new(Config {
            seed: 7,
            limits: limits(),
        })
        .unwrap();
        let mut ids = Vec::new();
        for step in 0..120u64 {
            let _ = match step % 5 {
                0 | 3 => model
                    .register(request(&format!("key-{}", step % 7)))
                    .map(|id| {
                        ids.push(id);
                    }),
                1 => ids
                    .get((step as usize / 5) % ids.len().max(1))
                    .copied()
                    .map_or(Ok(()), |id| {
                        model
                            .ready(Readiness {
                                request_id: id,
                                at: INITIAL_TIME + step as i64,
                                kind: "timer".into(),
                                payload: vec![step as u8; (step % 4) as usize],
                                equivalence_class: String::new(),
                            })
                            .map(|_| ())
                    }),
                2 => model.quiesce().map(|_| ()),
                _ => ids.pop().map_or(Ok(()), |id| model.cancel(id).map(|_| ())),
            };
            let expected = model.uncached_state_digest();
            assert_eq!(
                model.state_digest_within(maximum),
                expected,
                "{maximum} {step}"
            );
            assert_eq!(
                model.state_digest_within(maximum),
                expected,
                "{maximum} {step}"
            );
            assert!(model.transition_frames.borrow().bytes.len() <= maximum);
        }
    }
}
