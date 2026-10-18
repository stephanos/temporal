use super::super::stream_abi::{HEADER, set_u32, set_u64};
use super::*;

fn limits() -> Limits {
    Limits {
        listeners: 128,
        connections: 128,
        bytes: 8 << 20,
        connection_bytes: 4 << 20,
        stall_nanos: 10_000_000_000,
        transitions: 8192,
    }
}

fn profile() -> ModelV2 {
    ModelV2 {
        profile: PROFILE.into(),
        limits: limits(),
        faults: Vec::new(),
    }
}

fn config() -> Config {
    Config {
        model: profile(),
        addresses: BTreeMap::new(),
        links: Vec::new(),
        strict_topology: false,
        faults: Vec::new(),
    }
}

fn ids() -> Vec<String> {
    vec!["a".into(), "b".into()]
}

fn invalid(config: &Config) -> &'static str {
    match Shared::new(config, &ids()) {
        Err(Failure::Invalid(dimension)) => dimension,
        Err(other) => panic!("not invalid: {other:?}"),
        Ok(_) => panic!("accepted"),
    }
}

fn fault(name: &str, action: &str, at: u64) -> Fault {
    Fault {
        name: name.into(),
        action: action.into(),
        from: "a".into(),
        to: "b".into(),
        at_nanos: at,
        delay_nanos: 0,
        symmetric: true,
    }
}

#[test]
fn cluster_network_refuses_what_new_broker_network_refuses() {
    let mut local = config();
    local.model.faults.push(LocalFault {
        name: "local".into(),
        action: "partition".into(),
        at_nanos: 1,
        port: 0,
        delay_nanos: 0,
    });
    assert_eq!(invalid(&local), "broker requires central fault schedule");
    let mut address = config();
    address.addresses.insert("a".into(), "10.0.0.256".into());
    assert_eq!(invalid(&address), "broker IPv4 address");
    address.addresses.insert("a".into(), "10.00.0.1".into());
    assert_eq!(invalid(&address), "broker IPv4 address");
    for (from, to) in [("a", "a"), ("a", "z"), ("z", "b")] {
        let mut endpoint = config();
        endpoint.links.push(Link {
            from: from.into(),
            to: to.into(),
            enabled: true,
            delay_nanos: 0,
        });
        assert_eq!(invalid(&endpoint), "broker link endpoint", "{from}->{to}");
    }
    let mut duplicate = config();
    for _ in 0..2 {
        duplicate.links.push(Link {
            from: "a".into(),
            to: "b".into(),
            enabled: false,
            delay_nanos: 0,
        });
    }
    assert_eq!(invalid(&duplicate), "broker duplicate link");
    let mut unordered = config();
    unordered.faults = vec![fault("late", "partition", 9), fault("early", "heal", 3)];
    assert_eq!(invalid(&unordered), "broker fault schedule");
    let mut repeated = config();
    repeated.faults = vec![fault("x", "partition", 1), fault("x", "heal", 2)];
    assert_eq!(invalid(&repeated), "broker fault schedule");
    let mut unknown = config();
    unknown.faults = vec![fault("x", "reset", 1)];
    assert_eq!(invalid(&unknown), "broker fault action");
    let mut endpointless = config();
    endpointless.faults = vec![fault("x", "partition", 1)];
    endpointless.faults[0].to = "z".into();
    assert_eq!(invalid(&endpointless), "broker fault schedule");
    let mut profile = config();
    profile.model.profile = "other".into();
    assert_eq!(invalid(&profile), "network v2 profile");
    let mut zero = config();
    zero.model.limits.listeners = 0;
    assert_eq!(invalid(&zero), "network v2 limits");
    assert!(Shared::new(&config(), &ids()).is_ok());
}

#[test]
fn cluster_network_builds_go_addresses_and_strict_topology() {
    let shared = Shared::new(&config(), &ids()).unwrap();
    assert_eq!(shared.address("a"), Some("10.0.0.2"));
    assert_eq!(shared.address("b"), Some("10.0.0.3"));
    let (client, server) = shared.domains("b", 1);
    assert_eq!(
        (client.node.as_str(), client.address.as_str()),
        ("b.client", "10.1.0.3")
    );
    assert_eq!(
        (server.node.as_str(), server.address.as_str()),
        ("b.server", "10.0.0.3")
    );
    let links = shared.record().unwrap().snapshot.links.unwrap();
    assert_eq!(links.len(), 12, "every ordered pair of four nodes");
    assert!(links.iter().all(|link| link.enabled));

    let mut strict = config();
    strict.strict_topology = true;
    strict.links.push(Link {
        from: "a".into(),
        to: "b".into(),
        enabled: false,
        delay_nanos: 7,
    });
    let links = Shared::new(&strict, &ids())
        .unwrap()
        .record()
        .unwrap()
        .snapshot
        .links
        .unwrap();
    // Same-owner pairs stay enabled; a->b is configured; b->a is absent.
    assert_eq!(links.len(), 4 + 4);
    let cross: Vec<_> = links
        .iter()
        .filter(|link| link.from.starts_with('a') && link.to.starts_with('b'))
        .collect();
    assert_eq!(cross.len(), 4);
    assert!(
        cross
            .iter()
            .all(|link| !link.enabled && link.delay_nanos == 7)
    );
    assert!(
        !links
            .iter()
            .any(|link| link.from.starts_with('b') && link.to.starts_with('a'))
    );
}

#[test]
fn cluster_network_faults_and_live_links_hit_all_four_pairs() {
    let mut scheduled = config();
    scheduled.faults = vec![fault("cut", "partition", 5), fault("mend", "heal", 9)];
    let mut shared = Shared::new(&scheduled, &ids()).unwrap();
    assert_eq!(shared.deadline(), 5);
    shared.advance(6).unwrap();
    let cross = |shared: &Shared| -> Vec<bool> {
        shared
            .record()
            .unwrap()
            .snapshot
            .links
            .unwrap()
            .into_iter()
            .filter(|link| link.from[..1] != link.to[..1])
            .map(|link| link.enabled)
            .collect()
    };
    assert_eq!(
        cross(&shared),
        vec![false; 8],
        "symmetric partition, four pairs"
    );
    assert_eq!(shared.deadline(), 9);
    shared.advance(9).unwrap();
    assert_eq!(cross(&shared), vec![true; 8]);
    shared.change_link("partition", "a", "b", 0, false).unwrap();
    let directional: Vec<_> = shared
        .record()
        .unwrap()
        .snapshot
        .links
        .unwrap()
        .into_iter()
        .filter(|link| link.from[..1] != link.to[..1])
        .map(|link| (link.from.starts_with('a'), link.enabled))
        .collect();
    assert_eq!(
        directional
            .iter()
            .filter(|(a, enabled)| *a && !enabled)
            .count(),
        4
    );
    assert!(directional.iter().all(|(a, enabled)| *a || *enabled));
    assert_eq!(
        shared.change_link("reset", "a", "b", 0, true),
        Err(Failure::Invalid("broker link action"))
    );
    assert_eq!(
        shared.change_link("heal", "a", "a", 0, true),
        Err(Failure::Invalid("broker link"))
    );
}

fn buffer(op: u32, id: u64, handle: u64, port: u32, payload: &[u8]) -> Vec<u8> {
    let mut data = vec![0u8; HEADER];
    set_u32(&mut data, 0, 2);
    set_u32(&mut data, 4, op);
    set_u64(&mut data, 8, id);
    set_u64(&mut data, 16, handle);
    set_u32(&mut data, 32, port);
    set_u32(&mut data, 36, u32::try_from(payload.len()).unwrap());
    data.extend_from_slice(payload);
    data
}

fn get32(data: &[u8], at: usize) -> u32 {
    u32::from_be_bytes(data[at..at + 4].try_into().unwrap())
}

fn get64(data: &[u8], at: usize) -> u64 {
    u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
}

fn endpoint(shared: &Arc<Mutex<Shared>>, id: &str, slot: u64, transcript: u64) -> Endpoint {
    Endpoint::new(shared.clone(), id, 1, slot, &profile(), 1000, transcript).unwrap()
}

#[test]
fn cluster_network_serves_own_server_dial_and_bytes() {
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut a = endpoint(&shared, "a", 1, 1 << 20);
    let mut listen = buffer(1, 1, 0, 0, &[]);
    a.serve(&mut listen, 1, true).unwrap();
    assert_eq!(get32(&listen, 40), 0, "listen completes");
    assert_eq!(get32(&listen, 44), 0);
    let listener = get64(&listen, 16);
    let port = get32(&listen, 48);
    assert!(listener != 0 && port != 0);
    let mut dial = buffer(2, 2, 0, port, &[]);
    a.serve(&mut dial, 1, true).unwrap();
    assert_eq!((get32(&dial, 40), get32(&dial, 44)), (0, 0));
    let client = get64(&dial, 16);
    let mut accept = buffer(3, 3, listener, 0, &[]);
    a.serve(&mut accept, 1, true).unwrap();
    assert_eq!((get32(&accept, 40), get32(&accept, 44)), (0, 0));
    let server = get64(&accept, 16);
    let mut write = buffer(5, 4, client, 0, b"hello");
    a.serve(&mut write, 1, true).unwrap();
    assert_eq!((get32(&write, 36), get32(&write, 44)), (5, 0));
    let mut read = buffer(4, 5, server, 0, &[0; 5]);
    a.serve(&mut read, 1, true).unwrap();
    assert_eq!((get32(&read, 40), get32(&read, 36)), (0, 5));
    assert_eq!(&read[HEADER..], b"hello");
    // Handles are global; slot 1's request IDs are based at 1 << 48.
    assert_eq!(a.handles().len(), 3);
    let evidence = a.evidence().unwrap();
    assert!(evidence.starts_with(evidence::MAGIC));
    assert_eq!(a.entry_count, 5);
    assert!(a.log_json_bytes > 0);
}

#[test]
fn cluster_network_buffer_failures_keep_go_classification() {
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut a = endpoint(&shared, "a", 1, 1 << 20);
    let mut reserved = buffer(1, 1, 0, 0, &[]);
    reserved[41] = 1;
    assert_eq!(
        a.serve(&mut reserved, 1, true),
        Err(Failure::Invalid("network v2 stream layout"))
    );
    let mut query = buffer(0, 0, 0, 0, &[]);
    a.serve(&mut query, 1, true).unwrap();
    assert_eq!(get32(&query, 40), 0, "a brokered node has a network");
    let mut bad_query = buffer(0, 1, 0, 0, &[]);
    assert_eq!(
        a.serve(&mut bad_query, 1, true),
        Err(Failure::Invalid("network v2 query"))
    );
    for (data, active) in [
        (buffer(1, 0, 0, 0, &[]), true),
        (buffer(1, 1 << 48, 0, 0, &[]), true),
        (buffer(1, 1, 0, 0, &[]), false),
    ] {
        let mut data = data;
        assert_eq!(
            a.serve(&mut data, 1, active),
            Err(Failure::Invalid("network v2 operation"))
        );
    }
    let mut opcode = buffer(14, 1, 0, 0, &[]);
    assert_eq!(
        a.serve(&mut opcode, 1, true),
        Err(Failure::Invalid("network v2 opcode"))
    );
    let mut write = buffer(5, 1, 0, 0, b"abc");
    write.pop();
    assert_eq!(
        a.serve(&mut write, 1, true),
        Err(Failure::Invalid("network v2 write payload"))
    );
    let mut listen = buffer(1, 1, 0, 0, &[]);
    listen.push(0);
    assert_eq!(
        a.serve(&mut listen, 1, true),
        Err(Failure::Invalid("network v2 payload"))
    );
    // The operations log is bounded by the instance's transcript bytes.
    let mut tiny = endpoint(&shared, "b", 2, 16);
    let mut listen = buffer(1, 1, 0, 0, &[]);
    assert_eq!(
        tiny.serve(&mut listen, 1, true),
        Err(Failure::Capacity("network-v2-transcript-bytes"))
    );
}

#[test]
fn cluster_network_revocation_retires_both_domains() {
    for graceful in [false, true] {
        let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
        let mut b = endpoint(&shared, "b", 2, 1 << 20);
        // b holds a listener and a connected pair on its own node.
        let mut listen = buffer(1, 1, 0, 0, &[]);
        b.serve(&mut listen, 1, true).unwrap();
        let mut dial = buffer(2, 2, 0, get32(&listen, 48), &[]);
        b.serve(&mut dial, 1, true).unwrap();
        let mut guard = lock(&shared);
        // Records are canonical (lane order), so select revocations by kind.
        let revocations = |guard: &Shared| -> Vec<(String, String)> {
            guard
                .record()
                .unwrap()
                .transitions
                .unwrap_or_default()
                .into_iter()
                .filter(|transition| matches!(transition.kind.as_str(), "stop" | "crash"))
                .map(|transition| (transition.kind, transition.outcome))
                .collect()
        };
        assert_eq!(revocations(&guard).len(), 0);
        guard.revoke("b", 1, graceful).unwrap();
        let added = revocations(&guard);
        let expected = if graceful {
            ("stop", "ok")
        } else {
            ("crash", "reset")
        };
        assert_ne!(added.len(), 0);
        assert!(
            added
                .iter()
                .all(|(kind, outcome)| kind == expected.0 && outcome == expected.1),
            "{graceful}: {added:?}"
        );
        assert!(
            guard.revoke("b", 1, graceful).is_err(),
            "both domains retired"
        );
        guard.register("b", 2).unwrap();
        let live: Vec<_> = guard
            .owner
            .live
            .values()
            .map(|domain| domain.incarnation)
            .collect();
        assert_eq!(live, vec![1, 1, 2, 2]);
    }
}

#[test]
fn cluster_network_log_entries_match_go_json() {
    let request = Request {
        domain: Domain {
            run: 1,
            node: "a.client".into(),
            incarnation: 1,
            address: "10.1.0.2".into(),
        },
        op: "write".into(),
        handle: 7,
        host: "10.0.0.2".into(),
        port: 0,
        bytes: b"hi".to_vec(),
        count: 2,
        deadline: 0,
        request_id: (1 << 48) + 4,
    };
    let result = OperationResult {
        request_id: 4,
        state: "completed".into(),
        count: 2,
        error: "<&>".into(),
        ..OperationResult::default()
    };
    let (json, binary) = evidence::operation_entry(3, &request, &result);
    assert_eq!(
        String::from_utf8(json).unwrap(),
        concat!(
            r#"{"at":3,"request":{"domain":{"run":1,"node":"a.client","incarnation":1,"address":"10.1.0.2"},"#,
            r#""op":"write","handle":7,"host":"10.0.0.2","bytes":"aGk=","count":2,"request_id":281474976710660},"#,
            r#""result":{"request_id":4,"state":"completed","local":{"IP":"","Port":0},"remote":{"IP":"","Port":0},"#,
            r#""count":2,"error":"\u003c\u0026\u003e"}}"#
        )
    );
    assert_eq!(binary[0], 1);
    let quiescence = world::Quiescence {
        kind: world::QuiescenceKind::Delivered,
        before: 5,
        after: 6,
        deliveries: vec![world::Delivery {
            request_id: 1,
            event_id: 2,
            at: 6,
            kind: "network".into(),
            payload: Vec::new(),
        }],
        blocked: Vec::new(),
    };
    let (json, _) = evidence::quiescence_entry(&quiescence);
    assert_eq!(
        String::from_utf8(json).unwrap(),
        r#"{"kind":"delivered","before":"5","after":"6","deliveries":[{"requestId":"1","eventId":"2","at":"6","kind":"network"}]}"#
    );
}

/// A cross-node scenario with a pending read, readiness through the network
/// World and a completed poll. Its evidence is the checked-in fixture that the
/// Go decoder (fn-178.13) rebuilds; `GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE=1`
/// rewrites it.
fn fixture_scenario() -> Vec<(String, Vec<u8>)> {
    let mut config = config();
    config.addresses.insert("b".into(), "10.0.7.1".into());
    let shared = Arc::new(Mutex::new(Shared::new(&config, &ids()).unwrap()));
    let mut a = endpoint(&shared, "a", 1, 1 << 20);
    let mut b = endpoint(&shared, "b", 2, 1 << 20);
    let mut listen = buffer(1, 1, 0, 0, &[]);
    b.serve(&mut listen, 1, true).unwrap();
    let (listener, port) = (get64(&listen, 16), get32(&listen, 48));
    let mut dial = buffer(2, 1, 0, port, &[]);
    dial[56..60].copy_from_slice(&[10, 0, 7, 1]);
    a.serve(&mut dial, 1, true).unwrap();
    let client = get64(&dial, 16);
    let mut accept = buffer(3, 2, listener, 0, &[]);
    b.serve(&mut accept, 1, true).unwrap();
    let server = get64(&accept, 16);
    let mut read = buffer(4, 3, server, 0, &[0; 4]);
    b.serve(&mut read, 1, true).unwrap();
    assert_eq!(get32(&read, 40), 1, "the read waits for bytes");
    let mut write = buffer(5, 2, client, 0, b"ping");
    a.serve(&mut write, 1, true).unwrap();
    let mut poll = buffer(12, 3, 0, 0, &[0; 4]);
    b.serve(&mut poll, 1, true).unwrap();
    assert_eq!((get32(&poll, 40), get32(&poll, 36)), (0, 4));
    assert_eq!(&poll[HEADER..], b"ping");
    let mut cancel = buffer(13, 9, 0, 0, &[]);
    a.serve(&mut cancel, 1, true).unwrap();
    assert_eq!(
        get32(&cancel, 44),
        8,
        "cancel of an unknown request is stale"
    );
    vec![
        ("a".into(), a.evidence().unwrap()),
        ("b".into(), b.evidence().unwrap()),
    ]
}

#[test]
fn cluster_network_fixture_is_current() {
    let directory =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/cluster_network");
    let regenerate =
        std::env::var("GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE").as_deref() == Ok("1");
    for (id, evidence) in fixture_scenario() {
        let path = directory.join(format!("{id}.gmnetev1"));
        if regenerate {
            std::fs::create_dir_all(&directory).unwrap();
            std::fs::write(&path, &evidence).unwrap();
            continue;
        }
        assert_eq!(
            std::fs::read(&path).unwrap(),
            evidence,
            "stale {path:?}; rerun with GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE=1 and the Go decoder tests"
        );
    }
}

/// The cached inspection commitment always equals a fresh hash of the
/// current evidence, including after another endpoint changed the shared
/// owner (which every endpoint's evidence embeds), and is reused unchanged.
#[test]
fn inspection_network_cache_follows_own_and_foreign_mutations() {
    use sha2::{Digest, Sha256};
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut endpoints = [
        endpoint(&shared, "a", 1, 1 << 20),
        endpoint(&shared, "b", 2, 1 << 20),
    ];
    let builds = std::cell::Cell::new(0);
    let commit = |bytes: &[u8]| -> [u8; 32] {
        builds.set(builds.get() + 1);
        Sha256::digest(bytes).into()
    };
    let check = |endpoints: &mut [Endpoint; 2]| -> [[u8; 32]; 2] {
        let mut digests = [[0; 32]; 2];
        for (index, endpoint) in endpoints.iter_mut().enumerate() {
            let evidence = endpoint.evidence().unwrap();
            assert!(endpoint.evidence_check().is_ok());
            let (digest, bytes) = endpoint.inspection_network(commit).unwrap();
            assert_eq!(digest, <[u8; 32]>::from(Sha256::digest(&evidence)));
            assert!(bytes.is_none_or(|bytes| bytes == evidence));
            let before = builds.get();
            assert_eq!(endpoint.inspection_network(commit).unwrap(), (digest, None));
            assert_eq!(builds.get(), before, "unchanged evidence was rebuilt");
            digests[index] = digest;
        }
        digests
    };
    let initial = check(&mut endpoints);
    let mut listen = buffer(1, 1, 0, 0, &[]);
    endpoints[1].serve(&mut listen, 1, true).unwrap();
    let port = get32(&listen, 48);
    let after_listen = check(&mut endpoints);
    assert_ne!(
        after_listen[0], initial[0],
        "a's evidence embeds b's listener"
    );
    let mut dial = buffer(2, 2, 0, port, &[]);
    set_u32(&mut dial, 56, u32::from_be_bytes([10, 0, 0, 3]));
    endpoints[0].serve(&mut dial, 1, true).unwrap();
    let after_dial = check(&mut endpoints);
    assert_ne!(
        after_dial[1], after_listen[1],
        "b's evidence embeds a's dial"
    );
    endpoints[0].attach_at(5).unwrap();
    endpoints[1].settle(5).unwrap();
    check(&mut endpoints);
}
