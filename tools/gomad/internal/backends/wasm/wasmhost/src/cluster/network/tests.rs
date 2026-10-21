use super::*;
use crate::network::stream_abi::{HEADER, set_u32, set_u64};

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

fn endpoint(shared: &Arc<Mutex<Shared>>, id: &str, slot: u64, transcript: u64) -> Endpoint {
    Endpoint::new(shared.clone(), id, 1, slot, &profile(), 1000, transcript).unwrap()
}

/// The cached inspection commitment always equals a fresh reconstruction
/// from the complete evidence, including after another endpoint changed the shared
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
            let evidence = endpoint.inspection_network_eager().unwrap();
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

#[test]
fn simulation_network_and_fault_choices_change_real_delivery_and_topology() {
    use crate::linked_simulation::{Controller, Dimension, Override, Plan};

    let run = |overrides, alternate_ports| {
        let plan = Plan::new(
            format!("sha256:{}", "a".repeat(64)),
            format!("sha256:{}", "b".repeat(64)),
            7,
            overrides,
        )
        .unwrap();
        let controller = Arc::new(Mutex::new(Controller::new(plan, 100).unwrap()));
        let mut config = config();
        config.faults = ["partition", "heal"]
            .into_iter()
            .map(|action| Fault {
                name: action.into(),
                action: action.into(),
                from: "a".into(),
                to: "b".into(),
                at_nanos: 5,
                delay_nanos: 0,
                symmetric: true,
            })
            .collect();
        let shared = Arc::new(Mutex::new(Shared::new(&config, &ids()).unwrap()));
        lock(&shared).attach_simulation(controller.clone());
        let mut a = endpoint(&shared, "a", 1, 1 << 20);
        let mut b = endpoint(&shared, "b", 2, 1 << 20);
        let mut ports = Vec::new();
        for id in [1, 2] {
            let port = if alternate_ports {
                4500 + u32::try_from(id).unwrap()
            } else {
                0
            };
            let mut listen = buffer(1, id, 0, port, &[]);
            b.serve(&mut listen, 1, true).unwrap();
            ports.push(get32(&listen, 48));
            let handle = u64::from_be_bytes(listen[16..24].try_into().unwrap());
            let mut accept = buffer(3, 10 + id, handle, 0, &[]);
            b.serve(&mut accept, 1, true).unwrap();
            assert_eq!(get32(&accept, 40), 1);
        }
        for (index, port) in ports.into_iter().enumerate() {
            let mut dial = buffer(2, 20 + index as u64, 0, port, &[]);
            set_u32(&mut dial, 56, u32::from_be_bytes([10, 0, 0, 3]));
            a.serve(&mut dial, 1, true).unwrap();
            assert_eq!(get32(&dial, 44), 0);
        }
        b.settle(1)?;
        let delivered: Vec<_> = b
            .world
            .transitions()
            .iter()
            .filter_map(|transition| match &transition.body {
                world::TransitionBody::Quiesce { result } => {
                    Some(result.deliveries.iter().map(|delivery| delivery.request_id))
                }
                _ => None,
            })
            .flatten()
            .collect();
        lock(&shared).advance(5).unwrap();
        let links = lock(&shared).record().unwrap().snapshot.links.unwrap();
        let mut controller = controller.lock().unwrap();
        controller.finish().unwrap();
        Ok::<_, Failure>((delivered, links, controller.decisions().to_vec()))
    };
    let (base_delivery, base_links, decisions) = run(Vec::new(), false).unwrap();
    let overrides: Vec<Override> = [Dimension::Network, Dimension::Fault]
        .into_iter()
        .map(|dimension| {
            let decision = decisions
                .iter()
                .find(|decision| decision.dimension == dimension)
                .expect("live ready requests and due declared faults must produce decisions");
            Override::from_decision(decision, 1).unwrap()
        })
        .collect();
    let (forced_delivery, forced_links, _) = run(overrides.clone(), false).unwrap();
    assert_eq!(base_delivery.len(), 2);
    assert_eq!(
        forced_delivery,
        base_delivery.into_iter().rev().collect::<Vec<_>>()
    );
    assert_ne!(forced_links, base_links);
    assert!(
        matches!(run(overrides, true), Err(Failure::Infrastructure(message))
        if message == "simulation exploration replay divergence")
    );
}

#[test]
fn changed_network_inspections_never_rehash_native_history() {
    use sha2::{Digest, Sha256};
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut endpoint = endpoint(&shared, "a", 1, 1 << 20);
    crate::network::history::DIGEST_COUNTS.with(|count| count.set((0, 0)));
    for now in 1..=1000 {
        endpoint.attach_at(now).unwrap();
        endpoint.settle(now).unwrap();
        endpoint
            .inspection_network(|bytes| Sha256::digest(bytes).into())
            .unwrap();
    }
    crate::network::history::DIGEST_COUNTS.with(|count| {
        assert_eq!(
            count.get().0,
            0,
            "inspection rehashed the complete native history"
        );
    });
}
