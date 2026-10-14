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
            "8377a1353561b9b1c37e1ced20e72f367d217dae84beaaf475563f8f133685b7",
        ),
        (
            7,
            "x",
            42,
            "1a1bcb67df62813067ed476d177911ee857e062162867fa14bd4b33df0cde2dc",
        ),
        (
            u64::MAX,
            "network",
            u64::MAX - 1,
            "ce051e61cf9eceb978d38cb2b4655066566025a6246fafb2dd4b77711f79080f",
        ),
        (
            41,
            "üñî",
            3,
            "f97a4392f8e702293266bc0f144ca07b97b530e8a7f099c4de941e930b7a0456",
        ),
        (
            1,
            "",
            1,
            "1f7e2d32333e08aa190a1c3cd921539316a5c3291bc345bbcd51220836895404",
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
