use super::{Config, Limits, Model, Readiness, Request, ResourceId, retained, wire_generated};
use base64::{Engine, engine::general_purpose::STANDARD};
use sha2::{Digest, Sha256};

fn config(requests: u64) -> Config {
    Config {
        seed: 7,
        limits: Limits {
            max_requests: requests + 1,
            max_events: requests + 1,
            max_queued_events: 1,
            max_transitions: (requests * 3 + 1).max(100_000),
            max_payload_bytes: 32 << 20,
            max_string_bytes: 1024,
        },
    }
}

fn populate(requests: u64) -> Model {
    let mut model = Model::new(config(requests)).unwrap();
    for ordinal in 0..requests {
        let request_id = model
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
                request_id,
                at: super::INITIAL_TIME + i64::try_from(ordinal + 1).unwrap(),
                kind: "runtime_net_stream".into(),
                payload: Vec::new(),
                equivalence_class: String::new(),
            })
            .unwrap();
        model.quiesce().unwrap();
    }
    model
}

fn wire(model: &Model) -> Vec<u8> {
    let mut bytes = Vec::new();
    wire_generated::write_snapshot(model, &mut bytes).unwrap();
    assert_eq!(bytes.len() as u64, model.snapshot_bytes());
    bytes
}

fn measure(stage: &str, requests: u64, maximum_peak: Option<u64>) {
    let status = std::fs::read_to_string("/proc/self/status").unwrap();
    let value = |key: &str| {
        let line = status.lines().find(|line| line.starts_with(key)).unwrap();
        line.split_whitespace()
            .nth(1)
            .unwrap()
            .parse::<u64>()
            .unwrap()
            * 1024
    };
    let rss = value("VmRSS:");
    let peak = value("VmHWM:");
    eprintln!("stage={stage} requests={requests} process_rss={rss} process_peak_rss={peak}");
    if let Some(maximum) = maximum_peak {
        assert!(
            peak <= maximum,
            "operator whole-process peak exceeds the investigation budget"
        );
    }
}

#[test]
fn binary_world_owner_rejects_foreign_seed_capacities_and_trailing_bytes() {
    let model = populate(3);
    let bytes = wire(&model);
    retained::validate(&bytes, model.config()).unwrap();
    let mut changed = model.config().clone();
    changed.seed += 1;
    let error = retained::validate(&bytes, &changed).unwrap_err();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"binary_world_foreign_configuration","inputs":{"wire_sha256":format!("{:x}",Sha256::digest(&bytes)),"seed":changed.seed,"max_requests":changed.limits.max_requests},"observed":{"error":error}})
    );
    changed = model.config().clone();
    changed.limits.max_requests -= 1;
    let error = retained::validate(&bytes, &changed).unwrap_err();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"binary_world_foreign_configuration","inputs":{"wire_sha256":format!("{:x}",Sha256::digest(&bytes)),"seed":changed.seed,"max_requests":changed.limits.max_requests},"observed":{"error":error}})
    );
    let mut trailing = bytes.clone();
    trailing.push(0);
    assert!(retained::validate(&trailing, model.config()).is_err());
    let envelope = retained::compress(&bytes).unwrap();
    assert!(retained::validate(&envelope, model.config()).is_err());
    assert!(retained::expand(&bytes).is_err());
    let restored = retained::expand(&envelope).unwrap();
    assert_eq!(restored, bytes);
    retained::validate(&restored, model.config()).unwrap();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"binary_world_owner_profile_round_trip","inputs":{"seed":model.config().seed,"requests":3,"wire_sha256":format!("{:x}",Sha256::digest(&bytes))},"observed":{"expanded_base64":STANDARD.encode(&bytes),"envelope_base64":STANDARD.encode(&envelope),"restored_base64":STANDARD.encode(&restored),"envelope_sha256":format!("{:x}",Sha256::digest(&envelope)),"restored_sha256":format!("{:x}",Sha256::digest(&restored)),"trailing_error":retained::validate(&trailing,model.config()).unwrap_err(),"wrong_raw_profile_error":retained::validate(&envelope,model.config()).unwrap_err(),"wrong_compressed_profile_error":retained::expand(&bytes).unwrap_err()}})
    );
}

#[test]
#[ignore = "explicit operator probe; allocation_receiving runs the fixed measured workloads"]
fn world_owner_codec_measured_cardinality() {
    assert_eq!(
        std::env::var("GOMAD_WORLD_CODEC_PROBE").as_deref(),
        Ok("1"),
        "operator probe must be explicitly activated"
    );
    let requests = 20_000;
    let model = populate(requests);
    measure("model", requests, None);
    let encoded = wire(&model);
    measure("encode_restore_canonical", requests, None);
    retained::validate(&encoded, model.config()).unwrap();
    measure("decode_restore_canonical", requests, None);
    assert_eq!(wire(&model), encoded);
    measure("reencode_restore_canonical", requests, None);
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"world_owner_codec_measured_cardinality","inputs":{"requests":requests,"seed":7,"payload_bytes_per_request":32},"observed":{"expanded_sha256":format!("{:x}",Sha256::digest(&encoded)),"transcript_digest":super::digest::hex(&model.cardinality().transcript_digest),"expanded_bytes":encoded.len()}})
    );
    eprintln!(
        "encoded_bytes={} bytes_per_request={} transitions={}",
        encoded.len(),
        encoded.len() as u64 / requests,
        model.cardinality().transitions
    );
}

#[test]
#[ignore = "explicit operator probe; allocation_receiving runs both fixed cardinalities"]
fn compressed_world_owner_measured_cardinality() {
    assert_eq!(
        std::env::var("GOMAD_COMPRESSED_WORLD_PROBE").as_deref(),
        Ok("1"),
        "operator probe must be explicitly activated"
    );
    let requests = match std::env::var("GOMAD_COMPRESSED_WORLD_PROBE_REQUESTS").as_deref() {
        Err(_) | Ok("" | "20000") => 20_000,
        Ok("200000") => 200_000,
        _ => panic!("probe cardinality must be explicitly 20000 or 200000"),
    };
    let model = populate(requests);
    measure("model", requests, Some(2 << 30));
    let cardinality = model.cardinality();
    let config = model.config().clone();
    let expanded = wire(&model);
    drop(model);
    measure("terminal_transfer", requests, Some(2 << 30));
    let envelope = retained::compress(&expanded).unwrap();
    measure("compressed_encode", requests, Some(2 << 30));
    let restored = retained::expand(&envelope).unwrap();
    assert_eq!(restored, expanded);
    retained::validate(&restored, &config).unwrap();
    assert_eq!(cardinality.requests, requests);
    assert_eq!(cardinality.events, requests);
    assert_eq!(cardinality.transitions, requests * 3);
    measure("compressed_semantic_decode", requests, Some(2 << 30));
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"compressed_world_owner_measured_cardinality","inputs":{"requests":requests,"seed":7,"payload_bytes_per_request":32},"observed":{"expanded_sha256":format!("{:x}",Sha256::digest(&expanded)),"envelope_sha256":format!("{:x}",Sha256::digest(&envelope)),"restored_sha256":format!("{:x}",Sha256::digest(&restored)),"transcript_digest":super::digest::hex(&cardinality.transcript_digest),"expanded_bytes":expanded.len(),"envelope_bytes":envelope.len()}})
    );
    eprintln!(
        "envelope_bytes={} expanded_bytes={} transcript_digest={}",
        envelope.len(),
        expanded.len(),
        super::digest::hex(&cardinality.transcript_digest)
    );
}
