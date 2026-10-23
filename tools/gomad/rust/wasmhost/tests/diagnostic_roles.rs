#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::{linked, linked::Conditions, linked_cluster as cluster};
use std::{
    sync::{
        Arc, Mutex,
        atomic::{AtomicU64, Ordering},
    },
    time::Duration,
};

#[path = "common/core_preservation_input.rs"]
mod input;

fn roles() -> [(&'static str, Conditions); 7] {
    [
        ("", Conditions::Ordinary),
        ("-diagnostic-profile", Conditions::DiagnosticProfile),
        ("-guest-profile", Conditions::GuestProfile),
        ("-diagnostic-fuel-off", Conditions::DiagnosticFuelOff),
        ("-guest-profile-fuel-off", Conditions::GuestProfileFuelOff),
        ("-stress-jitter", Conditions::StressReplay),
        ("-determinism-fixture", Conditions::DeterminismFixtureReplay),
    ]
}

fn request() -> linked::Request {
    input::request(
        r"
        (i32.store8 (i32.const 64) (i32.const 114))
        (i32.store (i32.const 0) (i32.const 64))
        (i32.store (i32.const 4) (i32.const 1))
        (if (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 16)) (then unreachable))",
    )
}

#[derive(Default)]
struct Output(Mutex<Vec<u8>>);

impl linked::OutputConsumer for Output {
    fn write(&self, _: linked::OutputStream, bytes: &[u8]) -> std::io::Result<()> {
        self.0.lock().unwrap().extend_from_slice(bytes);
        Ok(())
    }
}

#[derive(Default)]
struct StartsAndJoins {
    starts: AtomicU64,
    joins: AtomicU64,
}

impl cluster::LifecycleObserver for StartsAndJoins {
    fn before(&self, _: &cluster::LifecyclePhase) -> Result<(), String> {
        self.starts.fetch_add(1, Ordering::SeqCst);
        Ok(())
    }

    fn after(&self, _: &cluster::LifecyclePhase) -> Result<(), String> {
        self.joins.fetch_add(1, Ordering::SeqCst);
        Ok(())
    }
}

fn prepared_cluster() -> cluster::Prepared {
    let handle = cluster::Handle {
        node: "role".into(),
        incarnation: 1,
    };
    cluster::prepare(cluster::Request {
        nodes: Vec::new(),
        broker: cluster::BrokerConfig {
            seed: 7,
            pause_on_settle: false,
            transition_bytes: None,
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
        steps: [cluster::Action::Start, cluster::Action::Wait]
            .into_iter()
            .map(|action| cluster::Step {
                action,
                handle: handle.clone(),
            })
            .collect(),
        boundaries: Vec::new(),
        crash_groups: Vec::new(),
        wall_timeout: Duration::from_secs(5),
        evidence_bytes: 1 << 20,
    })
    .unwrap()
}

#[test]
fn typed_native_role_admission_requires_the_exact_compiled_identity() {
    let mut configurations = std::collections::BTreeSet::new();
    for (name, role) in roles() {
        assert_eq!(
            role.configuration(),
            format!("{}{name}", linked::CONFIGURATION)
        );
        assert!(configurations.insert(role.configuration()), "{name}");
        if role == Conditions::compiled() {
            role.validate().unwrap();
        } else {
            assert!(
                role.validate()
                    .unwrap_err()
                    .to_string()
                    .contains("identity mismatch before authorization"),
                "{name}"
            );
        }
    }
}

#[test]
fn release_binary_cannot_authorize_a_diagnostic_or_stress_identity() {
    if Conditions::compiled() == Conditions::Ordinary {
        for (_, role) in roles().into_iter().skip(1) {
            assert!(role.validate().is_err());
        }
    } else {
        assert!(Conditions::Ordinary.validate().is_err());
    }
}

#[test]
fn single_native_role_matrix_refuses_before_guest_output_and_joins_every_execution() {
    for selected in roles()
        .into_iter()
        .map(|(_, role)| Some(role))
        .chain([None])
    {
        let requested = selected.unwrap_or_default();
        let capture = Arc::new(Output::default());
        let mut prepared = linked::prepare(request()).unwrap();
        prepared.output(capture.clone());
        if let Some(role) = selected {
            prepared.conditions(role);
        }
        let result = std::thread::scope(|scope| {
            scope
                .spawn(move || linked::execute_prepared(prepared))
                .join()
                .unwrap()
        });
        if requested == Conditions::compiled() {
            let execution_result = result.unwrap();
            assert_eq!(
                execution_result.outcome.termination,
                "exit",
                "{}",
                requested.configuration()
            );
            assert_eq!(execution_result.outcome.exit_code, Some(0));
            assert_eq!(execution_result.evidence["stdout"], b"r");
            assert_eq!(&*capture.0.lock().unwrap(), b"r");
            let timing = execution_result.timing.unwrap();
            assert!(timing.returned >= timing.terminal_classified);
            assert!(timing.terminal_classified >= timing.authorization_started);
        } else {
            let error = result.err().unwrap();
            assert_eq!(
                error.to_string(),
                "WASI invalid: engine/module/import identity mismatch before authorization"
            );
            assert!(capture.0.lock().unwrap().is_empty());
        }
        assert_eq!(Arc::strong_count(&capture), 1);
    }
}

#[test]
fn cluster_native_role_matrix_refuses_before_any_store_and_joins_admitted_stores() {
    for selected in roles()
        .into_iter()
        .map(|(_, role)| Some(role))
        .chain([None])
    {
        let requested = selected.unwrap_or_default();
        let mut prepared = prepared_cluster();
        if let Some(role) = selected {
            prepared.conditions(role);
        }
        let activity = prepared.activity();
        let observer = StartsAndJoins::default();
        let result = std::thread::scope(|scope| {
            let observer = &observer;
            scope
                .spawn(move || cluster::execute_prepared_observed(prepared, Some(observer)))
                .join()
                .unwrap()
        });
        let activity = activity.snapshot();
        assert_eq!(activity.remote_callback_attempts, 0);
        if requested == Conditions::compiled() {
            let execution_result = result.unwrap();
            assert_eq!(
                execution_result.outcome.termination,
                "exit",
                "{}",
                requested.configuration()
            );
            assert_eq!(execution_result.outcome.exit_code, Some(0));
            assert_eq!(activity.launched_stores, 1);
            assert_eq!(observer.starts.load(Ordering::SeqCst), 1);
            assert_eq!(observer.joins.load(Ordering::SeqCst), 1);
            assert_eq!(execution_result.incarnations.len(), 1);
            assert_eq!(
                execution_result.incarnations[0].execution.evidence["stdout"],
                b"r"
            );
            assert!(!execution_result.incarnations[0].interrupted);
        } else {
            let error = result.err().unwrap();
            assert_eq!(
                error.to_string(),
                "WASI invalid: engine/module/import identity mismatch before authorization"
            );
            assert_eq!(activity.launched_stores, 0);
            assert_eq!(observer.starts.load(Ordering::SeqCst), 0);
            assert_eq!(observer.joins.load(Ordering::SeqCst), 0);
        }
    }
}

#[cfg(feature = "guest-profile")]
#[test]
fn private_guest_profile_configuration_refuses_original_invalid_bounds_before_execution() {
    for (path, interval_micros, max_samples) in [
        ("relative", 1, 1),
        ("/diagnostic/profile.json", 0, 1),
        ("/diagnostic/profile.json", 1, 100_001),
    ] {
        let capture = Arc::new(Output::default());
        let mut prepared = linked::prepare(request()).unwrap();
        prepared.conditions(Conditions::compiled());
        prepared.output(capture.clone());
        let error = prepared
            .guest_profile(linked::GuestProfileConfig {
                path: path.into(),
                interval_micros,
                max_samples,
            })
            .err()
            .unwrap();
        assert_eq!(error.to_string(), "invalid guest profile path or bounds");
        assert!(capture.0.lock().unwrap().is_empty());
        drop(prepared);
        assert_eq!(Arc::strong_count(&capture), 1);
    }
}

#[cfg(all(
    feature = "determinism-fixture",
    not(any(feature = "diagnostic-profile", feature = "stress-jitter"))
))]
#[test]
fn every_fixture_fault_requires_explicit_native_role_before_single_or_cluster_start() {
    for kind in [
        linked::FixtureFault::None,
        linked::FixtureFault::Draw,
        linked::FixtureFault::Clock,
        linked::FixtureFault::Handoff,
        linked::FixtureFault::OutputClock,
    ] {
        for opted_in in [false, true] {
            let capture = Arc::new(Output::default());
            let mut prepared = linked::prepare(request()).unwrap();
            prepared.output(capture.clone());
            prepared.fixture_fault(kind);
            if opted_in {
                prepared.conditions(Conditions::DeterminismFixtureReplay);
            }
            let result = std::thread::scope(|scope| {
                scope
                    .spawn(move || linked::execute_prepared(prepared))
                    .join()
                    .unwrap()
            });
            if opted_in {
                let execution_result = result.unwrap();
                assert_eq!(execution_result.outcome.termination, "exit");
                assert_eq!(execution_result.evidence["stdout"], b"r");
                assert!(execution_result.timing.is_some());
                assert_eq!(&*capture.0.lock().unwrap(), b"r");
            } else {
                assert_eq!(
                    result.err().unwrap().to_string(),
                    "fixture fault requires the compiled determinism fixture role"
                );
                assert!(capture.0.lock().unwrap().is_empty());
            }
            assert_eq!(Arc::strong_count(&capture), 1);

            let mut prepared = prepared_cluster();
            prepared.fixture_fault(kind, None).unwrap();
            if opted_in {
                prepared.conditions(Conditions::DeterminismFixtureReplay);
            }
            let activity = prepared.activity();
            let observer = StartsAndJoins::default();
            let result = std::thread::scope(|scope| {
                let observer = &observer;
                scope
                    .spawn(move || cluster::execute_prepared_observed(prepared, Some(observer)))
                    .join()
                    .unwrap()
            });
            assert_eq!(activity.snapshot().remote_callback_attempts, 0);
            assert_eq!(activity.snapshot().launched_stores, u64::from(opted_in));
            assert_eq!(observer.starts.load(Ordering::SeqCst), u64::from(opted_in));
            assert_eq!(observer.joins.load(Ordering::SeqCst), u64::from(opted_in));
            if opted_in {
                let execution_result = result.unwrap();
                assert_eq!(execution_result.outcome.termination, "exit");
                assert_eq!(execution_result.incarnations.len(), 1);
                assert_eq!(
                    execution_result.incarnations[0].execution.evidence["stdout"],
                    b"r"
                );
                assert!(!execution_result.incarnations[0].interrupted);
            } else {
                assert_eq!(
                    result.err().unwrap().to_string(),
                    "fixture fault requires the compiled determinism fixture role"
                );
            }
        }
    }
}
