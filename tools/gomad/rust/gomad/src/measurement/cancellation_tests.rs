use super::*;
use gomad_wasmhost::linked::Cancellation;
use std::{
    collections::BTreeMap,
    path::{Path, PathBuf},
    sync::mpsc,
    time::{Duration, Instant},
};

struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-measure-cancellation-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir(&root).unwrap();
        Self(std::fs::canonicalize(root).unwrap())
    }
    fn call(&self) -> inputs::Invocation {
        inputs::Invocation {
            name: "runtime-cost-profile".into(),
            repo: self.0.clone(),
            go_command: "go".into(),
            inputs: BTreeMap::from([(
                "GOMAD_WASM_PROFILE_ROOT".into(),
                self.0.to_string_lossy().into_owned(),
            )]),
            environ: Vec::new(),
            keep: false,
        }
    }
    fn store(&self) -> PathBuf {
        self.0.join(".gomad-measure/runtime-cost-profile")
    }
    fn latest(&self) -> PathBuf {
        let name: String =
            serde_json::from_slice(&std::fs::read(self.store().join("latest.json")).unwrap())
                .unwrap();
        self.store().join(name)
    }
    fn seed_latest(&self) -> PathBuf {
        let run = retention::RetainedRun::begin(
            &self.0,
            "runtime-cost-profile",
            &BTreeMap::new(),
            false,
            &|| false,
        )
        .unwrap();
        let path = run.path().to_owned();
        std::fs::write(path.join("result"), b"prior complete").unwrap();
        run.finish(true, &|| false).unwrap();
        path
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn collect_with_token(
    root: &Path,
    token: &Cancellation,
    stdout: &mut dyn std::io::Write,
    collect: impl FnOnce(&inputs::Invocation, &inputs::CheckoutProof, &Path) -> Result<PathBuf, String>,
) -> Result<(), String> {
    let call = inputs::Invocation {
        name: "runtime-cost-profile".into(),
        repo: root.to_owned(),
        go_command: "go".into(),
        inputs: BTreeMap::from([(
            "GOMAD_WASM_PROFILE_ROOT".into(),
            root.to_string_lossy().into_owned(),
        )]),
        environ: Vec::new(),
        keep: false,
    };
    let proof = inputs::fixture_checkout(root.to_owned());
    cancellation::within(token, || dispatch_collector(&call, &proof, stdout, collect)).unwrap()
}
fn wait_for(path: &Path, expected: &[u8]) -> bool {
    let deadline = Instant::now() + Duration::from_secs(10);
    loop {
        if std::fs::read(path).is_ok_and(|bytes| {
            expected.is_empty() || bytes.windows(expected.len()).any(|part| part == expected)
        }) {
            return true;
        }
        if Instant::now() >= deadline {
            return false;
        }
        std::thread::sleep(Duration::from_millis(1));
    }
}

#[test]
fn public_cancelled_dispatch_refuses_before_inputs_roots_or_collection() {
    let scratch = Scratch::new();
    let requested = scratch.0.join("not-created");
    let token = Cancellation::default();
    token.cancel();
    let mut call = scratch.call();
    call.name = "namespace-sample".into();
    call.inputs.clear();
    call.inputs.insert(
        "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT".into(),
        requested.to_string_lossy().into_owned(),
    );
    let (mut stdout, mut stderr) = (Vec::new(), Vec::new());
    let status = dispatch_with_cancellation(
        call,
        inputs::fixture_checkout(scratch.0.clone()),
        &mut stdout,
        &mut stderr,
        &token,
    );
    assert_eq!(status, inputs::EXIT_FAILED);
    assert!(
        String::from_utf8(stderr)
            .unwrap()
            .contains("measurement cancelled")
    );
    assert_eq!(stdout, Vec::<u8>::new());
    assert!(!requested.exists());
    let mut collected = false;
    let result = collect_with_token(&scratch.0, &token, &mut stdout, |_, _, _| {
        collected = true;
        Ok(requested.clone())
    });
    assert!(result.unwrap_err().contains("cancelled"));
    assert!(!collected);
    assert!(!scratch.store().exists());
}

#[test]
fn cancellation_after_collection_or_final_report_failure_prevents_publication() {
    struct FailedReport;
    impl std::io::Write for FailedReport {
        fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
            Err(std::io::Error::other("final report failed"))
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }
    for cancelled in [false, true] {
        let scratch = Scratch::new();
        let prior = scratch.seed_latest();
        let token = Cancellation::default();
        let mut collected_path = None;
        let mut stdout = Vec::new();
        let mut failed_report = FailedReport;
        let output: &mut dyn std::io::Write = if cancelled {
            &mut stdout
        } else {
            &mut failed_report
        };
        let result = collect_with_token(&scratch.0, &token, output, |_, _, root| {
            std::fs::write(root.join("result"), b"producer complete")
                .map_err(|error| error.to_string())?;
            collected_path = Some(root.to_owned());
            assert_eq!(scratch.latest(), prior);
            if cancelled {
                token.cancel();
            }
            Ok(root.join("result"))
        });
        let error = result.unwrap_err();
        assert!(
            error.contains(if cancelled {
                "cancelled"
            } else {
                "final report failed"
            }),
            "{error}"
        );
        let path = collected_path.unwrap();
        assert_eq!(scratch.latest(), prior);
        assert_eq!(
            std::fs::read(path.join("result")).unwrap(),
            b"producer complete"
        );
        assert!(!path.join(".complete").exists());
    }
}

#[test]
fn cancellation_while_publication_lock_is_held_returns_before_lock_release() {
    let scratch = Scratch::new();
    let prior = scratch.seed_latest();
    let token = Cancellation::default();
    let worker_token = token.clone();
    let root = scratch.0.clone();
    let (started, started_rx) = mpsc::channel();
    let (proceed, proceed_rx) = mpsc::channel();
    let (finished, finished_rx) = mpsc::channel();
    let worker = std::thread::spawn(move || {
        let result = collect_with_token(&root, &worker_token, &mut Vec::new(), |_, _, path| {
            std::fs::write(path.join("result"), b"complete").map_err(|error| error.to_string())?;
            started.send(path.to_owned()).unwrap();
            proceed_rx.recv().unwrap();
            Ok(path.join("result"))
        });
        finished.send(result).unwrap();
    });
    let path = started_rx.recv_timeout(Duration::from_secs(10)).unwrap();
    let held = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .open(scratch.store().join(".lock"))
        .unwrap();
    rustix::fs::flock(&held, rustix::fs::FlockOperation::NonBlockingLockExclusive).unwrap();
    proceed.send(()).unwrap();
    let completed = wait_for(&path.join(".complete"), b"");
    token.cancel();
    let result = finished_rx.recv_timeout(Duration::from_secs(10));
    drop(held);
    worker.join().unwrap();
    assert!(
        completed,
        "collector did not complete before the publication lock wait"
    );
    assert!(result.unwrap().unwrap_err().contains("cancelled"));
    assert_eq!(scratch.latest(), prior);
    assert!(!path.join(".complete").exists());
}

#[test]
fn subprocess_cancel_child() {
    if std::env::var("GOMAD_MEASUREMENT_CANCEL_CHILD").as_deref() != Ok("1") {
        return;
    }
    std::io::stdout()
        .write_all(b"measurement child ready\n")
        .unwrap();
    std::io::stdout().flush().unwrap();
    loop {
        std::thread::park();
    }
}

#[test]
fn cancellation_kills_and_reaps_the_actual_supervised_child_before_retaining_failure() {
    let scratch = Scratch::new();
    let prior = scratch.seed_latest();
    let token = Cancellation::default();
    let cancel_token = token.clone();
    let (ready, ready_rx) = mpsc::channel::<PathBuf>();
    let canceller = std::thread::spawn(move || {
        let stdout = ready_rx.recv_timeout(Duration::from_secs(10)).unwrap();
        let child_started = wait_for(&stdout, b"measurement child ready");
        cancel_token.cancel();
        child_started
    });
    let mut collected_path = None;
    let module = module_path!().split_once("::").unwrap().1;
    let result = collect_with_token(&scratch.0, &token, &mut Vec::new(), |_, _, root| {
        collected_path = Some(root.to_owned());
        ready.send(root.join("cancel-child.stdout")).unwrap();
        let observed = build::observe_command(
            root,
            "cancel-child",
            root,
            &BTreeMap::from([("GOMAD_MEASUREMENT_CANCEL_CHILD".into(), "1".into())]),
            &[
                std::env::current_exe()
                    .unwrap()
                    .to_string_lossy()
                    .into_owned(),
                "--exact".into(),
                format!("{module}::subprocess_cancel_child"),
                "--nocapture".into(),
            ],
            Duration::from_secs(7200),
            64 << 20,
        );
        match observed.error {
            Some(error) => Err(error),
            None => Ok(root.join("cancel-child.json")),
        }
    });
    assert!(
        canceller.join().unwrap(),
        "child readiness was never observed"
    );
    assert!(result.unwrap_err().contains("cancelled"));
    let path = collected_path.unwrap();
    let observation: build::CommandObservation =
        serde_json::from_slice(&std::fs::read(path.join("cancel-child.json")).unwrap()).unwrap();
    assert_eq!(observation.termination, "cancelled");
    assert_eq!(observation.signal, Some(9));
    assert!(observation.reaped_monotonic_nanos.is_some());
    assert!(
        observation.cleanup_errors.is_empty(),
        "{:?}",
        observation.cleanup_errors
    );
    let pid =
        rustix::process::Pid::from_raw(i32::try_from(observation.pid.unwrap()).unwrap()).unwrap();
    assert_eq!(
        rustix::process::test_kill_process_group(pid),
        Err(rustix::io::Errno::SRCH)
    );
    assert_eq!(scratch.latest(), prior);
    assert!(!path.join(".complete").exists());
}
