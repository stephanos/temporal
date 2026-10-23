#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::{linked, linked_cluster as cluster};
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::time::Duration;

// Original ActivityTestSuite preparation: 1 GiB choices, 256 MiB output,
// plus the current campaign producer's checked aggregate headroom.
const OUTPUT: u64 = 256 << 20;
const EVIDENCE: u64 = (1 << 30) + 2 * OUTPUT + (128 << 20);

fn request(evidence: u64) -> linked::Request {
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    linked::Request {
        module_sha256: format!("{:x}", Sha256::digest(&module)),
        module,
        limits: linked::Limits {
            memory_bytes: 65536, fuel: 10000, output_bytes: OUTPUT,
            evidence_bytes: evidence, wall_timeout: Duration::from_secs(5),
        },
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(), writable_directories: Vec::new(), preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(), stdin: Vec::new(), epoch_nanos: 0,
            limits: linked::NamespaceLimits { files: 16, descriptors: 16, filesystem_bytes: 4096, output_bytes: OUTPUT },
        }),
        storage: None,
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(json!({
                "entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
                "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":1_u64<<30,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":1_u64<<30,"pending_events":16}
            })).unwrap(),
            arguments: Vec::new(), environment: vec!["PWD=/".into()], replay_trace: None,
        }),
    }
}

#[test]
fn standalone_accepts_original_activity_aggregate_without_reserving_it() {
    let mut prepared = linked::prepare(request(EVIDENCE)).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    let execution = linked::execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "exit");
    assert!(execution.evidence.values().map(Vec::len).sum::<usize>() < 65536);
}

#[test]
fn cluster_accepts_original_activity_aggregate_without_reserving_it() {
    let handle = cluster::Handle {
        node: "a".into(),
        incarnation: 1,
    };
    let request = cluster::Request {
        nodes: Vec::new(),
        broker: cluster::BrokerConfig {
            seed: 0,
            pause_on_settle: false,
            limits: cluster::WorldLimits {
                max_requests: 100,
                max_events: 100,
                max_queued_events: 100,
                max_transitions: 1000,
                max_payload_bytes: 4096,
                max_string_bytes: 1024,
            },
        },
        network: None,
        incarnations: vec![cluster::Incarnation {
            handle: handle.clone(),
            request: request(EVIDENCE),
            network: None,
        }],
        steps: vec![
            cluster::Step {
                action: cluster::Action::Start,
                handle: handle.clone(),
            },
            cluster::Step {
                action: cluster::Action::Wait,
                handle,
            },
        ],
        boundaries: Vec::new(),
        crash_groups: Vec::new(),
        wall_timeout: Duration::from_secs(5),
        evidence_bytes: EVIDENCE,
    };
    let mut prepared = cluster::prepare(request).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    let execution = cluster::execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "exit");
    assert_eq!(execution.incarnations.len(), 1);
}

#[test]
fn aggregate_zero_is_still_refused() {
    assert!(linked::prepare(request(0)).is_err());
}
