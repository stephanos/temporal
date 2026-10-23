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
fn inspection_chain_tracks_every_mutation_and_exact_expanded_wire_size() {
    let mut model = Model::new(Config {
        seed: 7,
        limits: limits(),
    })
    .unwrap();
    let check = |model: &Model| {
        let mut wire = Vec::new();
        wire_generated::write_snapshot(model, &mut wire).unwrap();
        assert_eq!(model.snapshot_bytes(), wire.len() as u64);
        let mut head = digest::empty_digest();
        for transition in model.transitions() {
            assert_eq!(transition.previous_digest, head);
            head = digest::transition_digest(transition);
            assert_eq!(transition.digest, head);
        }
        assert_eq!(&head, model.transcript_digest());
    };
    check(&model);
    for queued in [false, true] {
        let id = model
            .register(Request {
                payload: vec![1, 2, 3],
                ..request("λ")
            })
            .unwrap();
        check(&model);
        if queued {
            model
                .ready(Readiness {
                    request_id: id,
                    at: model.now() + 3,
                    kind: "timer".into(),
                    payload: vec![4, 5],
                    equivalence_class: "group".into(),
                })
                .unwrap();
            check(&model);
        } else {
            model.quiesce().unwrap();
            check(&model);
        }
        model.cancel(id).unwrap();
        check(&model);
        model.cancel(id).unwrap();
        check(&model);
    }
    for _ in 0..2 {
        let id = model.register(request("batch")).unwrap();
        check(&model);
        model
            .ready(Readiness {
                request_id: id,
                at: model.now() + 9,
                kind: "ready".into(),
                payload: vec![7],
                equivalence_class: String::new(),
            })
            .unwrap();
        check(&model);
    }
    model.quiesce().unwrap();
    check(&model);
    model.cancel(3).unwrap();
    check(&model);
    model.quiesce().unwrap();
    check(&model);
    let before = model.inspection_commitment();
    assert!(model.cancel(0).is_err());
    assert_eq!(model.inspection_commitment(), before);
    check(&model);
}

#[test]
fn inspection_chain_binds_configuration_order_and_length_framed_payloads() {
    let make = |seed, keys: &[&str], payload: &[u8]| {
        let mut model = Model::new(Config {
            seed,
            limits: limits(),
        })
        .unwrap();
        for key in keys {
            model
                .register(Request {
                    payload: payload.to_vec(),
                    ..request(key)
                })
                .unwrap();
        }
        model.inspection_commitment()
    };
    let cases = [
        make(7, &["ab", "c"], &[1, 2]),
        make(7, &["a", "bc"], &[1, 2]),
        make(7, &["c", "ab"], &[1, 2]),
        make(7, &["ab", "c"], &[1, 2, 0]),
        make(8, &["ab", "c"], &[1, 2]),
        make(7, &["ab"], &[1, 2]),
    ];
    for (index, digest) in cases.iter().enumerate() {
        for other in &cases[index + 1..] {
            assert_ne!(digest, other);
        }
    }
    assert_eq!(cases[0], make(7, &["ab", "c"], &[1, 2]));
    let mut model = Model::new(Config {
        seed: 7,
        limits: limits(),
    })
    .unwrap();
    let before = model.inspection_commitment();
    model.config.limits.max_payload_bytes += 1;
    assert_ne!(before, model.inspection_commitment());
    let before = model.inspection_commitment();
    model.now += 1;
    assert_ne!(before, model.inspection_commitment());
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
        let mut retained = 0;
        for step in 0..120u64 {
            let _ = match step % 5 {
                0 | 3 => model
                    .register(request(&format!("key-{}", step % 7)))
                    .map(|id| {
                        ids.push(id);
                    }),
                1 => ids
                    .get((usize::try_from(step).unwrap() / 5) % ids.len().max(1))
                    .copied()
                    .map_or(Ok(()), |id| {
                        model
                            .ready(Readiness {
                                request_id: id,
                                at: INITIAL_TIME + step.cast_signed(),
                                kind: "timer".into(),
                                payload: vec![step.to_le_bytes()[0]; (step % 4) as usize],
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
            let frames = model.transition_frames.borrow();
            assert!(frames.bytes.len() <= maximum);
            // Crossing the bound keeps the retained prefix.
            assert!(frames.count >= retained, "{maximum} {step}");
            retained = frames.count;
        }
    }
}

#[test]
fn state_digest_past_the_frame_bound_keeps_the_retained_prefix() {
    let mut model = Model::new(Config {
        seed: 7,
        limits: limits(),
    })
    .unwrap();
    let mut frame = Vec::new();
    let mut maximum = 0;
    while model.transitions().len() < 3 {
        model
            .register(request(&format!("key-{}", model.transitions().len())))
            .unwrap();
    }
    for transition in &model.transitions()[..2] {
        frame.clear();
        digest::append_transition_frame(&mut frame, transition);
        maximum += frame.len();
    }
    assert_eq!(
        model.state_digest_within(maximum),
        model.uncached_state_digest()
    );
    let retained = model.transition_frames.borrow().bytes.clone();
    assert_eq!(model.transition_frames.borrow().count, 2);
    for step in 0..20 {
        model.register(request(&format!("more-{step}"))).unwrap();
        assert_eq!(
            model.state_digest_within(maximum),
            model.uncached_state_digest(),
            "{step}"
        );
        let frames = model.transition_frames.borrow();
        assert_eq!(frames.count, 2, "{step}");
        assert_eq!(frames.bytes, retained, "{step}");
    }
}

#[test]
fn transition_byte_limit_refuses_before_mutation_at_the_exact_native_record_boundary() {
    let config = Config {
        seed: 4,
        limits: limits(),
    };
    let mut baseline = Model::new(config.clone()).unwrap();
    baseline.register(request("a")).unwrap();
    let mut size = WireSize::default();
    wire_generated::write_transitions(baseline.transitions(), &mut size).unwrap();
    for maximum in [1, size.0 - 1, size.0] {
        let mut model = Model::new(config.clone()).unwrap();
        model.limit_transition_bytes(Some(maximum)).unwrap();
        let before = model.state_digest();
        let result = model.register(request("a"));
        if maximum == size.0 {
            assert_eq!(result.unwrap(), 1);
            assert_eq!(model.state_digest(), baseline.state_digest());
        } else {
            assert!(matches!(
                result,
                Err(Error::Capacity {
                    dimension: "transition-bytes",
                    ..
                })
            ));
            assert_eq!(model.state_digest(), before);
            assert!(model.transitions().is_empty());
        }
    }
}

#[test]
fn every_world_transition_byte_refusal_preserves_the_accepted_prefix() {
    for stage in [
        "register",
        "ready",
        "cancel",
        "delivered",
        "deadlock",
        "idle",
    ] {
        let config = Config {
            seed: 7,
            limits: limits(),
        };
        let prefix = || {
            let mut model = Model::new(config.clone()).unwrap();
            if !matches!(stage, "register" | "idle") {
                model.register(request("a")).unwrap();
            }
            if matches!(stage, "cancel" | "delivered") {
                model
                    .ready(Readiness {
                        request_id: 1,
                        at: INITIAL_TIME + 10,
                        kind: "ready".into(),
                        payload: vec![1, 2],
                        equivalence_class: "same".into(),
                    })
                    .unwrap();
            }
            model
        };
        let execute = |model: &mut Model| -> Result<()> {
            match stage {
                "register" => {
                    model.register(request("a"))?;
                }
                "ready" => {
                    model.ready(Readiness {
                        request_id: 1,
                        at: INITIAL_TIME + 10,
                        kind: "ready".into(),
                        payload: vec![1, 2],
                        equivalence_class: "same".into(),
                    })?;
                }
                "cancel" => {
                    model.cancel(1)?;
                }
                _ => {
                    model.quiesce()?;
                }
            }
            Ok(())
        };
        let mut expected = prefix();
        execute(&mut expected).unwrap();
        let exact = expected.transition_bytes;
        for bound in [exact - 1, exact] {
            let mut model = prefix();
            model.limit_transition_bytes(Some(bound)).unwrap();
            let before = model.state_digest();
            let prefix_bytes = model.transition_bytes;
            let history = model.transitions().to_vec();
            let snapshot_bytes = model.snapshot_bytes();
            let observed = execute(&mut model);
            if bound == exact {
                observed.unwrap();
                assert_eq!(model.state_digest(), expected.state_digest(), "{stage}");
                assert_eq!(model.transition_bytes, exact);
                let mut retained = Vec::new();
                wire_generated::write_snapshot(&model, &mut retained).unwrap();
                retained::validate_with_transition_limit(&retained, &config, Some(exact)).unwrap();
                assert!(
                    retained::validate_with_transition_limit(&retained, &config, Some(exact - 1))
                        .is_err()
                );
            } else {
                assert!(
                    matches!(observed, Err(Error::Capacity { dimension:"transition-bytes",used,limit,.. }) if used==prefix_bytes && limit==bound),
                    "{stage}"
                );
                assert_eq!(model.state_digest(), before, "{stage}");
                assert_eq!(model.transitions(), history);
                assert_eq!(model.transition_bytes, prefix_bytes);
                assert_eq!(model.snapshot_bytes(), snapshot_bytes);
            }
        }
    }
}
