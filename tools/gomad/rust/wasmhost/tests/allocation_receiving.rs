include!("../src/lib.rs");
#[path = "../../allocation_probe.rs"]
mod allocation_measure;

#[test]
#[ignore = "explicit allocation receiving operator harness"]
fn allocation_receiving() {
    allocation_measure::self_check();
    allocation_metadata_probe(|read| {
        let (sizes, facts) = allocation_measure::capture(|| {
            let mut sizes = [0_u64; 100];
            for size in &mut sizes {
                let (_, actual) = std::hint::black_box(read)();
                *size = actual;
            }
            sizes
        });
        assert!(sizes.iter().all(|size| *size == 1 << 19));
        allocation_measure::row(
            "volume-metadata",
            serde_json::json!({"file_bytes":1<<19,"reads":100}),
            facts,
            serde_json::json!({"sizes":sizes.as_slice()}),
        );
        assert!(facts.allocated_bytes <= 1 << 20);
    });
    namespace_allocation();
    world_allocation(false, 20_000);
    world_allocation(true, 20_000);
    world_allocation(true, 200_000);
}

fn namespace_allocation() {
    // The native namespace owner receives one captured image, replacing mount RPC frames.
    let mut wire = b"GMWN\x01\0\0\0".to_vec();
    for value in [1_u64, 5, 2, 1, 0, 0, 0, 0, 0] {
        wire.extend_from_slice(&value.to_le_bytes());
    }
    wire.extend_from_slice(&1_u64.to_le_bytes());
    wire.push(b'/');
    wire.extend_from_slice(&3_u64.to_le_bytes());
    assert!(namespace::decode_image(&mut wire.as_slice(), 1, 1, 0).is_err());
    let (errors, facts) = allocation_measure::capture(|| {
        let mut errors = 0;
        for _ in 0..100 {
            errors += usize::from(namespace::decode_image(&mut wire.as_slice(), 1, 1, 0).is_err());
        }
        errors
    });
    assert_eq!(errors, 100);
    allocation_measure::row(
        "namespace-file-length",
        serde_json::json!({"declared_bytes":3,"limit_bytes":2,"runs":100}),
        facts,
        serde_json::json!({"rejections":errors}),
    );
    assert!(facts.allocation_calls <= 4 * 100);
}

fn world_allocation(compressed: bool, requests: u64) {
    let case = if compressed {
        "compressed-world"
    } else {
        "binary-world"
    };
    let inputs = serde_json::json!({"requests":requests,"seed":7,"payload_bytes_per_request":32});
    let mut cumulative = 0;
    let (model, facts) = allocation_measure::capture(|| populate_world(requests));
    stage(case, "model", &inputs, facts, &mut cumulative, compressed);
    let cardinality = model.cardinality();
    let state_digest = model.state_digest();
    let config = model.config().clone();
    let mut model = Some(model);
    let (expanded, facts) = allocation_measure::capture(|| {
        let mut wire = Vec::new();
        world::wire_generated::write_snapshot(model.as_ref().unwrap(), &mut wire).unwrap();
        if compressed {
            drop(model.take());
        }
        wire
    });
    stage(
        case,
        if compressed {
            "terminal_transfer"
        } else {
            "encode_restore_canonical"
        },
        &inputs,
        facts,
        &mut cumulative,
        compressed,
    );
    let (restored, envelope) = restore_world(
        compressed,
        &expanded,
        model.as_ref(),
        &config,
        &inputs,
        &mut cumulative,
        case,
    );
    assert_eq!(restored, expanded);
    assert_eq!(cardinality.requests, requests);
    assert_eq!(cardinality.events, requests);
    assert_eq!(cardinality.transitions, requests * 3);
    let expanded_file = retain_bytes(case, requests, "expanded", &expanded);
    let restored_file = retain_bytes(case, requests, "restored", &restored);
    let envelope_file = envelope
        .as_ref()
        .map(|bytes| retain_bytes(case, requests, "compressed", bytes));
    let envelope_sha256 = envelope
        .as_ref()
        .map(|bytes| format!("{:x}", sha2::Sha256::digest(bytes)));
    let ((), facts) = allocation_measure::capture(|| ());
    allocation_measure::row(
        &format!("{case}/round-trip"),
        inputs,
        facts,
        serde_json::json!({
            "expanded_bytes":expanded.len(),"expanded_file":expanded_file,"restored_file":restored_file,
            "envelope_file":envelope_file,"envelope_sha256":envelope_sha256,
            "expanded_sha256":format!("{:x}",sha2::Sha256::digest(&expanded)),
            "restored_sha256":format!("{:x}",sha2::Sha256::digest(&restored)),
            "requests":cardinality.requests,"events":cardinality.events,"transitions":cardinality.transitions,
            "transcript_digest":world::digest::hex(&cardinality.transcript_digest),"state_digest":world::digest::hex(&state_digest)
        }),
    );
}

fn populate_world(requests: u64) -> world::Model {
    use world::{Config, Limits, Model, Readiness, Request, ResourceId};
    let mut model = Model::new(Config {
        seed: 7,
        limits: Limits {
            max_requests: requests + 1,
            max_events: requests + 1,
            max_queued_events: 1,
            max_transitions: (requests * 3 + 1).max(100_000),
            max_payload_bytes: 32 << 20,
            max_string_bytes: 1024,
        },
    })
    .unwrap();
    for ordinal in 0..requests {
        let id = model
            .register(Request {
                kind: "runtime_net_stream".into(),
                resource: ResourceId {
                    adapter: "wasm-broker".into(),
                    kind: "incarnation".into(),
                    key: "active/1".into(),
                },
                priority: 0,
                payload: vec![0; 32],
            })
            .unwrap();
        model
            .ready(Readiness {
                request_id: id,
                at: world::INITIAL_TIME + i64::try_from(ordinal + 1).unwrap(),
                kind: "runtime_net_stream".into(),
                payload: Vec::new(),
                equivalence_class: String::new(),
            })
            .unwrap();
        model.quiesce().unwrap();
    }
    model
}

fn restore_world(
    compressed: bool,
    expanded: &[u8],
    model: Option<&world::Model>,
    config: &world::Config,
    inputs: &serde_json::Value,
    cumulative: &mut u64,
    case: &str,
) -> (Vec<u8>, Option<Vec<u8>>) {
    let mut envelope = None;
    let restored = if compressed {
        let (encoded, facts) =
            allocation_measure::capture(|| world::retained::compress(expanded).unwrap());
        stage(case, "compressed_encode", inputs, facts, cumulative, true);
        let (restored, facts) = allocation_measure::capture(|| {
            let restored = world::retained::expand(&encoded).unwrap();
            world::retained::validate(&restored, config).unwrap();
            restored
        });
        stage(
            case,
            "compressed_streaming_semantic_decode",
            inputs,
            facts,
            cumulative,
            true,
        );
        envelope = Some(encoded);
        restored
    } else {
        let ((), facts) =
            allocation_measure::capture(|| world::retained::validate(expanded, config).unwrap());
        stage(
            case,
            "decode_restore_canonical",
            inputs,
            facts,
            cumulative,
            false,
        );
        let (again, facts) = allocation_measure::capture(|| {
            let mut wire = Vec::new();
            world::wire_generated::write_snapshot(model.unwrap(), &mut wire).unwrap();
            wire
        });
        stage(
            case,
            "reencode_restore_canonical",
            inputs,
            facts,
            cumulative,
            false,
        );
        again
    };
    (restored, envelope)
}

fn retain_bytes(case: &str, requests: u64, suffix: &str, bytes: &[u8]) -> String {
    use std::io::Write as _;
    use std::os::unix::fs::OpenOptionsExt as _;
    let path = format!("{case}-{requests}.{suffix}.wire");
    let mut output = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&path)
        .unwrap();
    output.write_all(bytes).unwrap();
    output.sync_all().unwrap();
    path
}

fn stage(
    case: &str,
    stage: &str,
    inputs: &serde_json::Value,
    facts: allocation_measure::Facts,
    cumulative: &mut u64,
    bounded: bool,
) {
    *cumulative += facts.allocated_bytes;
    let status =
        std::fs::read_to_string("/proc/self/status").expect("operator probe requires Linux /proc");
    let bytes = |name: &str| -> u64 {
        status
            .lines()
            .find(|line| line.starts_with(name))
            .unwrap()
            .split_whitespace()
            .nth(1)
            .unwrap()
            .parse::<u64>()
            .unwrap()
            * 1024
    };
    let peak = bytes("VmHWM:");
    allocation_measure::row(
        &format!("{case}/{stage}"),
        inputs.clone(),
        facts,
        serde_json::json!({
            "cumulative_allocated_bytes":cumulative,"process_rss_bytes":bytes("VmRSS:"),"process_peak_rss_bytes":peak
        }),
    );
    if bounded {
        assert!(
            peak <= 2 << 30,
            "original compressed World 2GiB investigation budget"
        );
    }
}

use sha2::Digest as _;
