#![cfg(all(unix, not(feature = "conformance")))]

use gomad_wasmhost::{linked, linked_cluster as cluster};
use sha2::{Digest as _, Sha256};
use std::{
    fs,
    os::unix::fs::{PermissionsExt, symlink},
    path::{Path, PathBuf},
    time::Duration,
};

struct Directory(PathBuf);
impl Directory {
    fn new() -> Self {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "gomad-linked-cache-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        Self(path)
    }
    fn entry(&self) -> PathBuf {
        let entries: Vec<_> = fs::read_dir(&self.0)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| {
                path.extension()
                    .is_some_and(|suffix| suffix == "gomad-cache-v3")
            })
            .collect();
        assert_eq!(entries.len(), 1);
        entries[0].clone()
    }
}
impl Drop for Directory {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

fn prepared() -> linked::Prepared {
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let mut prepared = linked::prepare(linked::Request {
        module_sha256: format!("{:x}", Sha256::digest(&module)),
        module,
        limits: linked::Limits {
            memory_bytes: 65536,
            fuel: 1000,
            output_bytes: 1024,
            evidence_bytes: 4096,
            wall_timeout: Duration::from_secs(5),
        },
        model: None,
        namespace: None,
        storage: None,
    })
    .unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared
}

fn execute(root: &Path) -> linked::Execution {
    let mut prepared = prepared();
    prepared
        .compile_cache(Some(root.to_str().unwrap()))
        .unwrap();
    linked::execute_prepared(prepared).unwrap()
}

#[test]
fn linked_cache_publishes_once_reuses_and_refuses_poison_or_links() {
    let root = Directory::new();
    let cold = execute(&root.0);
    assert_eq!(cold.outcome.termination, "exit");
    let cold_metrics = cold.outcome.metrics.unwrap();
    assert!(!cold_metrics.cache_hit);
    assert!(cold_metrics.compile_nanos > 0);
    let entry = root.entry();
    let metadata = fs::symlink_metadata(&entry).unwrap();
    assert!(metadata.is_file());
    assert_eq!(metadata.permissions().mode() & 0o777, 0o600);
    let original = fs::read(&entry).unwrap();
    assert_ne!(original, [0_u8; 0]);
    let warm = execute(&root.0);
    assert_eq!(warm.outcome.termination, "exit");
    assert_eq!(warm.evidence, cold.evidence);
    let warm_metrics = warm.outcome.metrics.unwrap();
    assert!(warm_metrics.cache_hit);
    assert_eq!(warm_metrics.compile_nanos, 0);
    assert!(warm_metrics.cache_load_nanos > 0);
    assert_eq!(fs::read(&entry).unwrap(), original);
    let mut poison = original.clone();
    *poison.last_mut().unwrap() ^= 1;
    fs::write(&entry, poison).unwrap();
    let refused = execute(&root.0);
    assert_eq!(refused.outcome.termination, "infrastructure");
    assert!(
        refused
            .outcome
            .message
            .contains("compile cache: corrupt entry")
    );
    assert_eq!(refused.outcome.metrics.unwrap().compile_nanos, 0);
    fs::write(&entry, &original).unwrap();
    let held = entry.with_extension("held");
    fs::rename(&entry, &held).unwrap();
    symlink(&held, &entry).unwrap();
    assert_eq!(execute(&root.0).outcome.termination, "infrastructure");
    fs::remove_file(&entry).unwrap();
    fs::hard_link(&held, &entry).unwrap();
    assert_eq!(execute(&root.0).outcome.termination, "infrastructure");
    fs::remove_file(&entry).unwrap();
    fs::rename(&held, &entry).unwrap();
    assert!(execute(&root.0).outcome.metrics.unwrap().cache_hit);
    assert_eq!(root.entry(), entry);
}

#[test]
fn linked_cache_defaults_disabled_and_rejects_relative_configuration() {
    let observed = linked::execute_prepared(prepared()).unwrap();
    let metrics = observed.outcome.metrics.unwrap();
    assert!(!metrics.cache_hit);
    assert_eq!(metrics.cache_load_nanos, 0);
    assert_eq!(metrics.cache_write_nanos, 0);
    assert!(prepared().compile_cache(Some("relative")).is_err());
}

fn cluster_prepared() -> cluster::Prepared {
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let handle = cluster::Handle {
        node: "a".into(),
        incarnation: 1,
    };
    let request = linked::Request {
        module_sha256: format!("{:x}", Sha256::digest(&module)), module,
        limits: linked::Limits { memory_bytes: 65536, fuel: 1000, output_bytes: 1024, evidence_bytes: 4096, wall_timeout: Duration::from_secs(5) },
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(), writable_directories: Vec::new(), preopens: vec![b"/".to_vec()], working_directory: b"/".to_vec(), stdin: Vec::new(), epoch_nanos: 0,
            limits: linked::NamespaceLimits { files: 16, descriptors: 16, filesystem_bytes: 4096, output_bytes: 1024 },
        }),
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(serde_json::json!({
                "entropy_key": vec![0_u8;32], "working_directory_sha256": format!("{:x}",Sha256::digest(b"/")),
                "arguments":{"bytes":0,"count":0,"chunks":0}, "environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}
            })).unwrap(),
            arguments: Vec::new(), environment: vec!["PWD=/".into()], replay_trace: None,
        }),
        storage: None,
    };
    let mut prepared = cluster::prepare(cluster::Request {
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
            request,
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
        evidence_bytes: 16384,
    })
    .unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared
}

#[test]
fn cluster_cache_uses_same_private_store_and_poison_is_host_failure() {
    let root = Directory::new();
    for _ in 0..2 {
        let mut prepared = cluster_prepared();
        prepared
            .compile_cache(Some(root.0.to_str().unwrap()))
            .unwrap();
        let result = cluster::execute_prepared(prepared).unwrap();
        assert_eq!(result.outcome.termination, "exit");
        assert_eq!(result.incarnations.len(), 1);
    }
    let entry = root.entry();
    let mut poison = fs::read(&entry).unwrap();
    *poison.last_mut().unwrap() ^= 1;
    fs::write(&entry, poison).unwrap();
    let mut prepared = cluster_prepared();
    prepared
        .compile_cache(Some(root.0.to_str().unwrap()))
        .unwrap();
    let error = cluster::execute_prepared(prepared)
        .err()
        .expect("corrupt cache ran a cluster guest");
    assert!(
        error
            .to_string()
            .contains("infrastructure: compile cache: corrupt entry")
    );
}
