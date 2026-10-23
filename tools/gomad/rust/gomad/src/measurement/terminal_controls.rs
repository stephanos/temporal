use super::*;
use crate::measurement::build;
use gomad_wasmhost::linked;
use std::{fs, os::unix::fs::DirBuilderExt as _, path::PathBuf, time::Duration};

const CONTROL_DELAY: Duration = Duration::from_millis(1100);

fn native_observation(preparation: Duration, cleanup: Duration, trap: bool) -> Observation {
    let mut module = b"\0asm\x01\0\0\0\x01\x04\x01\x60\0\0\x03\x02\x01\0\x05\x03\x01\0\x01\x07\x13\x02\x06memory\x02\0\x06_start\0\0".to_vec();
    module.extend_from_slice(if trap {
        &[10, 5, 1, 3, 0, 0, 11]
    } else {
        &[10, 4, 1, 2, 0, 11]
    });
    let started = Instant::now();
    let mut prepared = linked::prepare(linked::Request {
        module_sha256: canonical::digest(&module),
        module,
        limits: linked::Limits {
            memory_bytes: 128 << 20,
            fuel: 100_000_000,
            output_bytes: 1 << 20,
            evidence_bytes: 16 << 20,
            wall_timeout: Duration::from_secs(5),
        },
        model: None,
        namespace: None,
        storage: None,
    })
    .unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared
        .fixture_timing_delays(preparation, cleanup)
        .unwrap();
    let execution = linked::execute_prepared(prepared).unwrap();
    let timing = execution
        .timing
        .as_ref()
        .expect("actual linked terminal missing");
    let teardown = TeardownInterval {
        terminal: timing.terminal_classified,
        returned: timing.returned,
        recorded_nanos: timing.terminal_to_return_nanos(),
    };
    let wall_nanos = u64::try_from(started.elapsed().as_nanos()).unwrap();
    Observation {
        execution,
        phases: BTreeMap::new(),
        wall_nanos,
        teardown,
        fresh_store_id: u64::from(std::process::id()),
    }
}

#[test]
fn terminal_controls_actual_native_prompt_slow_preparation_and_delayed_return() {
    for (name, preparation, cleanup, accepted) in [
        ("prompt-exit", Duration::ZERO, Duration::ZERO, true),
        ("slow-preparation", CONTROL_DELAY, Duration::ZERO, true),
        ("delayed-exit", Duration::ZERO, CONTROL_DELAY, false),
    ] {
        let observation = native_observation(preparation, cleanup, false);
        assert_eq!(observation.execution.outcome.termination, "exit", "{name}");
        if !preparation.is_zero() {
            assert!(observation.wall_nanos >= 1_100_000_000, "{name}");
            assert!(
                observation
                    .execution
                    .outcome
                    .metrics
                    .as_ref()
                    .unwrap()
                    .prepare_nanos
                    >= 1_100_000_000,
                "{name}"
            );
        }
        if !cleanup.is_zero() {
            assert!(
                observation.teardown.recorded_nanos >= 1_100_000_000,
                "{name}"
            );
        }
        let run = Run::observed(observation).unwrap();
        let admitted = run.validate(true, false);
        eprintln!(
            "terminal-control={name} admitted={} teardown_nanos={} prepare_nanos={}",
            admitted.is_ok(),
            run.teardown_nanos,
            run.prepare_nanos
        );
        assert_eq!(admitted.is_ok(), accepted, "{name}: {admitted:?}");
    }
}

#[test]
fn terminal_controls_typed_missing_and_invalid_native_timing_are_refused() {
    for invalid in [false, true] {
        let mut observation = native_observation(Duration::ZERO, Duration::ZERO, false);
        if invalid {
            let timing = observation.execution.timing.as_mut().unwrap();
            timing.returned = timing.terminal_classified - Duration::from_nanos(1);
        } else {
            observation.execution.timing = None;
        }
        let error = match Run::observed(observation) {
            Err(error) => error,
            Ok(_) => panic!("invalid native terminal timing admitted"),
        };
        assert!(error.contains("actual native terminal timing"), "{error}");
    }
    let observation = native_observation(Duration::ZERO, Duration::ZERO, true);
    assert_eq!(observation.execution.outcome.termination, "trap");
    assert!(observation.execution.timing.is_some());
    assert!(Run::observed(observation).is_err());
}

struct Directory(PathBuf);
impl Directory {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-terminal-controls-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        fs::DirBuilder::new().mode(0o700).create(&path).unwrap();
        Self(path)
    }
}
impl Drop for Directory {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn terminal_controls_actual_supervisor_rejects_death_and_postreceipt_watchdog_and_reaps() {
    for (name, after, timeout, accepted) in [
        ("prompt-exit", "", Duration::from_secs(5), true),
        (
            "helper-death",
            "kill -KILL $$\n",
            Duration::from_secs(5),
            false,
        ),
        (
            "post-terminal-watchdog",
            "/bin/sleep 30\n",
            Duration::from_millis(200),
            false,
        ),
    ] {
        let root = Directory::new();
        let command = vec![
            "/bin/sh".into(),
            "-c".into(),
            format!("printf '%s\\n' 'native-lifecycle-fixture-terminal'\n{after}"),
        ];
        let result = build::command_bounded(
            &root.0,
            name,
            &root.0,
            &BTreeMap::new(),
            &command,
            timeout,
            64 << 10,
        );
        assert_eq!(result.is_ok(), accepted, "{name}: {result:?}");
        let observation: build::CommandObservation =
            serde_json::from_slice(&fs::read(root.0.join(format!("{name}.json"))).unwrap())
                .unwrap();
        assert!(
            fs::read(&observation.stdout.path)
                .unwrap()
                .starts_with(b"native-lifecycle-fixture-terminal\n"),
            "{name}"
        );
        let pid = rustix::process::Pid::from_raw(
            i32::try_from(observation.pid.expect("actual child PID absent")).unwrap(),
        )
        .unwrap();
        assert!(
            observation.cleanup_errors.is_empty(),
            "{name}: {:?}",
            observation.cleanup_errors
        );
        assert_eq!(
            rustix::process::test_kill_process_group(pid),
            Err(rustix::io::Errno::SRCH),
            "{name}"
        );
        if name == "helper-death" {
            assert_eq!(observation.signal, Some(9));
        }
        if name == "post-terminal-watchdog" {
            assert_eq!(observation.termination, "timeout");
            assert!(observation.duration_nanos >= 200_000_000);
        }
        eprintln!(
            "terminal-control={name} admitted={} termination={} group_gone=true",
            result.is_ok(),
            observation.termination
        );
    }
}
