use super::*;
use crate::evidence::canonical;
use gomad_wasmhost::linked_cluster as cluster;
use std::os::unix::fs::PermissionsExt as _;

struct Directory(PathBuf);
impl Directory {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-inspection-cache-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        fs::create_dir(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        Self(path)
    }
    fn poison(&self) {
        let entries: Vec<_> = fs::read_dir(&self.0)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|entry| {
                entry
                    .extension()
                    .is_some_and(|suffix| suffix == "gomad-cache-v3")
            })
            .collect();
        assert_eq!(entries.len(), 1);
        let mut bytes = fs::read(&entries[0]).unwrap();
        *bytes.last_mut().unwrap() ^= 1;
        fs::write(&entries[0], bytes).unwrap();
    }
}
impl Drop for Directory {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

fn request() -> linked::Request {
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    linked::Request {
        module_sha256: canonical::digest(&module),
        module,
        limits: linked::Limits {
            memory_bytes: 65536,
            fuel: 1000,
            output_bytes: 1024,
            evidence_bytes: 16384,
            wall_timeout: Duration::from_secs(10),
        },
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(),
            stdin: Vec::new(),
            epoch_nanos: 0,
            limits: linked::NamespaceLimits {
                files: 16,
                descriptors: 16,
                filesystem_bytes: 4096,
                output_bytes: 1024,
            },
        }),
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(json!({
                "entropy_key":vec![0_u8;32], "working_directory_sha256":canonical::digest(b"/"),
                "arguments":{"bytes":0,"count":0,"chunks":0}, "environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}
            })).unwrap(),
            arguments: Vec::new(),
            environment: vec!["PWD=/".into()],
            replay_trace: None,
        }),
        storage: None,
    }
}

fn execution(root: &Path, clustered: bool) -> Result<linked::Execution, String> {
    if !clustered {
        let mut prepared = linked::prepare(request()).map_err(|error| error.to_string())?;
        prepared.conditions(linked::Conditions::compiled());
        prepared
            .compile_cache(root.to_str())
            .map_err(|error| error.to_string())?;
        return linked::execute_prepared(prepared).map_err(|error| error.to_string());
    }
    let handle = cluster::Handle {
        node: "a".into(),
        incarnation: 1,
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
            request: request(),
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
        wall_timeout: Duration::from_secs(10),
        evidence_bytes: 16384,
    })
    .map_err(|error| error.to_string())?;
    prepared.conditions(linked::Conditions::compiled());
    prepared
        .compile_cache(root.to_str())
        .map_err(|error| error.to_string())?;
    cluster::execute_prepared(prepared)
        .map_err(|error| error.to_string())?
        .incarnations
        .into_iter()
        .next()
        .map(|incarnation| incarnation.execution)
        .ok_or_else(|| "cluster produced no incarnation".into())
}

#[test]
fn standalone_and_cluster_cache_poison_propagates_status_three_through_scan_and_bisect() {
    for clustered in [false, true] {
        let root = Directory::new();
        assert_eq!(
            execution(&root.0, clustered).unwrap().outcome.termination,
            "exit"
        );
        root.poison();
        for mode in ["scan", "bisect"] {
            let search = Search {
                check: "healthy".into(),
                first: 0,
                last: 4,
                mode: mode.into(),
                attempt_budget: 10,
                wall_time: 30_000_000_000,
            };
            let error = search::run(
                &search,
                || Duration::ZERO,
                |_, _| {
                    predicate_evaluation(
                        inspection_execution(execution(&root.0, clustered))
                            .map(|_| panic!("poisoned cache reached inspection predicates")),
                        "healthy",
                    )
                },
            )
            .expect_err("cache failure became an invariant search report");
            assert_eq!(error.status, 3, "{mode}, clustered={clustered}");
            assert!(
                error.message.contains("compile cache: corrupt entry"),
                "{}",
                error.message
            );
            assert!(error.message.contains(root.0.to_str().unwrap()));
        }
    }
}

#[test]
fn exact_cache_capacity_is_host_error_and_ordinary_capacity_is_evaluation_result() {
    for (message, host) in [
        ("compile cache artifact capacity", true),
        ("initial guest tables exceed limit", false),
    ] {
        let observed = linked::Execution {
            timing: None,
            #[cfg(feature = "helper-guest-profile")]
            guest_profile_error: None,
            #[cfg(feature = "helper-diagnostic-profile")]
            external_owner_attempts: 0,
            authorization_to_terminal_nanos: None,
            outcome: linked::Outcome::new("capacity", message),
            evidence: BTreeMap::new(),
            event_order: None,
        };
        let result = inspection_execution(Ok(observed));
        assert_eq!(matches!(result, Err(EvaluationError::Host(_))), host);
    }
}
