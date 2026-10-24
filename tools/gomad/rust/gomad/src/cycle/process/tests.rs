use super::busy::BusyWorkers;
use super::*;
use std::collections::BTreeMap;
use std::sync::atomic::AtomicU64;
static SEQUENCE: AtomicU64 = AtomicU64::new(0);
struct Fixture(PathBuf);
impl Fixture {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "cycle-process-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, Ordering::Relaxed)
        ));
        Self(path)
    }
    fn runner(&self) -> Runner {
        Runner::create(&self.0, None).unwrap()
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}
fn command(script: &str) -> Command {
    Command {
        argv: vec!["/bin/sh".into(), "-c".into(), script.into()],
        cwd: "/".into(),
        environment: Environment::Exact(BTreeMap::new()),
        accepted: vec![0],
        timeout: Duration::from_secs(3),
        private_context: None,
    }
}
fn run(runner: &mut Runner, name: &str, command: Command) -> Completed {
    runner
        .start(name, command, &AtomicBool::new(false), |_| Ok(()))
        .unwrap()
        .wait(&AtomicBool::new(false))
        .unwrap()
}
#[test]
fn actual_exit_signal_and_single_child_accounting() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    for (name, script, expected) in [
        ("ok", "echo $$; echo retained >&2", 0),
        ("bad", "exit 7", 7),
        ("signal", "kill -TERM $$", -15),
    ] {
        let completed = run(&mut runner, name, command(script));
        assert_eq!(completed.exit, expected);
        assert_eq!(completed.accepted(), expected == 0);
        assert_eq!(completed.resources.pid, completed.pid);
        assert_eq!(completed.resources.waited_pid, completed.pid);
        assert_eq!(completed.resources.reaps, 1);
        assert!(completed.resources.peak_rss_bytes > 0);
        assert!(completed.started_monotonic_nanos < completed.reaped_monotonic_nanos);
        assert_ne!(completed.parent_pid, std::process::id());
        assert!(completed.cleanup_complete);
        assert!(!Path::new(&format!("/proc/{}", completed.pid)).exists());
        if name == "ok" {
            assert_eq!(
                std::fs::read_to_string(fixture.0.join("ok.stdout"))
                    .unwrap()
                    .trim(),
                completed.pid.to_string()
            );
        }
        eprintln!(
            "PROCESS_PROOF {}",
            serde_json::to_string(&completed).unwrap()
        );
    }
    let mut accepted = command("exit 7");
    accepted.accepted = vec![7];
    assert!(run(&mut runner, "accepted", accepted).accepted());
}
#[test]
fn durable_admission_refusal_and_interrupt_never_execute_leaf() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let marker = fixture.0.join("executed");
    let cmd = command(&format!("echo executed > {}", marker.display()));
    let result = runner.start("refused", cmd, &AtomicBool::new(false), |registration| {
        let record: serde_json::Value =
            serde_json::from_slice(&std::fs::read(fixture.0.join("refused.process.json")).unwrap())
                .unwrap();
        assert_eq!(record["pid"], registration.pid);
        assert_eq!(record["active"], true);
        for _ in 0..20 {
            assert!(!marker.exists());
            std::thread::park_timeout(Duration::from_millis(5));
        }
        Err("admission refused".into())
    });
    assert!(
        result
            .err()
            .unwrap()
            .to_string()
            .contains("admission refused")
    );
    assert!(!marker.exists());
    assert!(
        runner
            .start(
                "interrupt",
                command("exit 0"),
                &AtomicBool::new(true),
                |_| Ok(())
            )
            .is_err()
    );
}
#[test]
fn timeout_and_cancellation_keep_primary_failure_after_reap() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let mut cmd = command("exec sleep 30");
    cmd.timeout = Duration::from_millis(40);
    let timed = run(&mut runner, "timeout", cmd);
    assert_eq!(timed.failure.as_deref(), Some("timeout"));
    assert_eq!(timed.exit, -9);
    assert!(timed.cleanup_complete);
    let running = runner
        .start(
            "cancelled",
            command("exec sleep 30"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let cancelled = running.wait(&AtomicBool::new(true)).unwrap();
    assert_eq!(cancelled.failure.as_deref(), Some("interrupted"));
    assert_eq!(cancelled.exit, -9);
    assert!(cancelled.cleanup_complete);
}
#[test]
fn escaped_double_fork_descendants_are_killed_and_reaped() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let pidfile = fixture.0.join("descendant");
    let script = format!(
        "setsid /bin/sh -c 'echo $$ > {}; exec sleep 30' & while test ! -s {}; do :; done; exit 0",
        pidfile.display(),
        pidfile.display()
    );
    let completed = run(&mut runner, "escape", command(&script));
    let descendant = std::fs::read_to_string(pidfile).unwrap();
    assert!(!completed.accepted());
    assert!(completed.cleanup_complete);
    assert!(completed.descendants_reaped > 0);
    assert!(!Path::new(&format!("/proc/{}", descendant.trim())).exists());
    eprintln!(
        "ESCAPE_PROOF {} descendant={}",
        serde_json::to_string(&completed).unwrap(),
        descendant.trim()
    );
}
#[test]
fn registry_and_environment_refuse_stale_foreign_and_untyped_inputs() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    assert!(Runner::create(&fixture.0, None).is_err());
    let completed = run(&mut runner, "fresh", command("exit 0"));
    assert!(completed.accepted());
    assert!(
        runner
            .start("fresh", command("exit 0"), &AtomicBool::new(false), |_| Ok(
                ()
            ))
            .is_err()
    );
    let mut foreign = command("exit 0");
    foreign.private_context = Some("foreign".into());
    assert!(
        runner
            .start("foreign", foreign, &AtomicBool::new(false), |_| Ok(()))
            .err()
            .unwrap()
            .to_string()
            .contains("foreign private")
    );
    let mut invalid = command("exit 0");
    invalid.argv.clear();
    assert!(
        runner
            .start("invalid", invalid, &AtomicBool::new(false), |_| Ok(()))
            .is_err()
    );
    assert!(serde_json::from_str::<Environment>(r#"{"Exact":{"KEY":1}}"#).is_err());
    let inherited = BTreeMap::from([
        ("GIT_DIR".into(), "foreign".into()),
        ("SANDBOX_START_DIR".into(), "foreign".into()),
        ("GOROOT".into(), "foreign".into()),
        ("GOOS".into(), "foreign".into()),
        ("GOMAD_X".into(), "foreign".into()),
        ("KEEP".into(), "yes".into()),
    ]);
    let clean = Environment::clean(inherited, &BTreeMap::new(), &BTreeMap::new());
    assert_eq!(clean.values().get("KEEP").map(String::as_str), Some("yes"));
    for key in ["GIT_DIR", "SANDBOX_START_DIR", "GOROOT", "GOOS", "GOMAD_X"] {
        assert!(!clean.values().contains_key(key));
    }
    assert_eq!(clean.values().get("LC_ALL").map(String::as_str), Some("C"));
    let mut exact = command("test \"$ONLY\" = frozen && test -z \"$AMBIENT_COMPILER_HOOK\"");
    exact.environment = Environment::Exact(BTreeMap::from([("ONLY".into(), "frozen".into())]));
    assert!(run(&mut runner, "exact", exact).accepted());
}
#[test]
fn procfs_missing_malformed_and_live_members_never_mean_quiescent() {
    let fixture = Fixture::new();
    std::fs::create_dir(&fixture.0).unwrap();
    assert!(procfs::quiescent(&fixture.0, 42, 42).is_ok());
    assert!(procfs::quiescent(&fixture.0.join("missing"), 42, 42).is_err());
    std::fs::create_dir(fixture.0.join("42")).unwrap();
    let stat = fixture.0.join("42/stat");
    for (raw, ok) in [
        ("42 (worker with ) parens) Z 1 42 42", true),
        ("42 (worker) S 1 42 42", false),
        ("42 (worker) S 1 99 42", false),
        ("garbage", false),
        ("42 (worker) S 1 nope 42", false),
        ("42 (worker) S 1", false),
    ] {
        std::fs::write(&stat, raw).unwrap();
        assert_eq!(procfs::quiescent(&fixture.0, 42, 42).is_ok(), ok, "{raw}");
    }
    std::fs::remove_file(&stat).unwrap();
    std::fs::create_dir(&stat).unwrap();
    assert!(procfs::quiescent(&fixture.0, 42, 42).is_err());
}
#[test]
fn supervisor_closes_inherited_flock_before_spawn_returns() {
    use rustix::fs::{FlockOperation, flock};
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let path = fixture.0.join("lease");
    let file = File::create(&path).unwrap();
    flock(&file, FlockOperation::LockExclusive).unwrap();
    let locked = crate::fork_order::Locked::new(file);
    let running = runner
        .start(
            "long",
            command("exec sleep 30"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let leaf = running.pid();
    drop(locked);
    let reopened = File::open(path).unwrap();
    flock(&reopened, FlockOperation::NonBlockingLockExclusive).unwrap();
    assert!(Path::new(&format!("/proc/{leaf}")).exists());
    let completed = running.wait(&AtomicBool::new(true)).unwrap();
    assert!(completed.cleanup_complete);
}
#[test]
fn twelve_real_busy_workers_are_reaped_on_success_failure_and_drop() {
    for label in ["busy-boundaries", "busy-grouped"] {
        let fixture = Fixture::new();
        let mut runner = fixture.runner();
        let busy = BusyWorkers::start(&mut runner, label, &AtomicBool::new(false)).unwrap();
        let pids = busy.pids();
        assert_eq!(pids.len(), 12);
        assert_eq!(
            pids.iter().collect::<std::collections::BTreeSet<_>>().len(),
            12
        );
        for pid in &pids {
            assert!(Path::new(&format!("/proc/{pid}")).exists());
        }
        let failed = run(&mut runner, "workload", command("exit 7"));
        assert!(!failed.accepted());
        let records = busy.finish().unwrap();
        assert_eq!(records.len(), 12);
        for record in &records {
            assert_eq!(record.exit, -9);
            assert_eq!(record.resources.reaps, 1);
            assert!(record.cleanup_complete);
        }
        eprintln!("BUSY_PROOF {}", serde_json::to_string(&records).unwrap());
        for pid in pids {
            assert!(!Path::new(&format!("/proc/{pid}")).exists());
        }
    }
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let busy = BusyWorkers::start(&mut runner, "busy-boundaries", &AtomicBool::new(false)).unwrap();
    let pids = busy.pids();
    drop(busy);
    for pid in pids {
        assert!(!Path::new(&format!("/proc/{pid}")).exists());
    }
}
#[test]
fn failed_exec_is_distinct_from_an_accepted_exit_127() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let mut missing = command("exit 0");
    missing.argv = vec!["/no-such-cycle-process-executable".into()];
    missing.accepted = vec![127];
    let refused = run(&mut runner, "missing", missing);
    assert_eq!(refused.exit, 127);
    assert!(!refused.accepted());
    assert!(
        refused
            .failure
            .as_deref()
            .unwrap()
            .contains("spawn refused")
    );
    assert!(refused.cleanup_complete);
    let mut ordinary = command("exit 127");
    ordinary.accepted = vec![127];
    assert!(run(&mut runner, "ordinary", ordinary).accepted());
}
#[test]
fn registration_changes_duplicates_and_interrupt_after_binding_never_release() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let cancelled = AtomicBool::new(false);
    let root = fixture.0.clone();
    let failed = runner
        .start("mutated", command("touch forbidden"), &cancelled, |_| {
            std::fs::write(root.join("mutated.process.json"), b"{}\n").unwrap();
            Ok(())
        })
        .err()
        .unwrap();
    assert!(matches!(failed.outcome, SpawnOutcome::Created { .. }));
    assert!(failed.message.contains("registration changed"));
    let failed = runner
        .start("interrupt-bound", command("exit 0"), &cancelled, |_| {
            cancelled.store(true, Ordering::Release);
            Ok(())
        })
        .err()
        .unwrap();
    assert!(matches!(failed.outcome, SpawnOutcome::Created { .. }));
    assert!(failed.message.contains("interrupted"));
    cancelled.store(false, Ordering::Release);
    let mut running = runner
        .start(
            "duplicate",
            command("exec sleep 30"),
            &cancelled,
            |_| Ok(()),
        )
        .unwrap();
    let mut duplicate = running.registration.clone();
    duplicate.name = "duplicate-pid-fresh-name".into();
    assert!(
        runner
            .registry
            .register(&duplicate)
            .unwrap_err()
            .contains("duplicate")
    );
    running.cancel();
    assert!(running.wait(&cancelled).unwrap().cleanup_complete);
    let failed = runner
        .start(
            "invalid",
            Command {
                argv: Vec::new(),
                ..command("exit 0")
            },
            &cancelled,
            |_| Ok(()),
        )
        .err()
        .unwrap();
    assert_eq!(failed.outcome, SpawnOutcome::NoChild);
}
#[test]
fn stage_intervals_are_typed_active_and_end_after_confirmed_cleanup() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let plan = crate::cycle::plan::derive(
        &crate::cycle::model::PlanConfig {
            binary: "/bin/true".into(),
            config_path: "/config.json".into(),
            config_sha256: "a".repeat(64),
        },
        crate::cycle::model::Mode::Parallel,
    )
    .unwrap();
    let stage = plan
        .stages
        .iter()
        .find(|stage| !stage.contract.scratch.is_empty())
        .unwrap();
    assert!(runner.active_intervals().unwrap().is_empty());
    let running = runner
        .start_stage(
            stage,
            3,
            command("exec sleep 30"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let intervals = runner.active_intervals().unwrap();
    assert_eq!(intervals.len(), 1);
    assert_eq!(intervals[0].label(), stage.contract.label);
    assert_eq!(intervals[0].lane(), 3);
    assert_eq!(intervals[0].pid(), running.pid());
    assert_eq!(intervals[0].stage_pid(), std::process::id());
    assert!(intervals[0].started_monotonic_nanos() > 0);
    assert_eq!(intervals[0].scratch(), stage.contract.scratch);
    assert!(
        running
            .wait(&AtomicBool::new(true))
            .unwrap()
            .cleanup_complete
    );
    assert!(runner.active_intervals().unwrap().is_empty());
}
#[test]
fn actual_report_binding_usage_and_cleanup_mutations_are_refused() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let mut running = runner
        .start("sample", command("exit 0"), &AtomicBool::new(false), |_| {
            Ok(())
        })
        .unwrap();
    let deadline = std::time::Instant::now() + Duration::from_secs(3);
    while running.poll(&AtomicBool::new(false)).unwrap().is_none() {
        assert!(std::time::Instant::now() < deadline);
        std::thread::park_timeout(Duration::from_millis(5));
    }
    let actual = sys::decode(&running.received);
    assert!(
        running
            .observation(actual, Path::new("/proc"))
            .unwrap()
            .accepted()
    );
    for mutation in 0..8 {
        let mut changed = actual;
        match mutation {
            0 => changed.pid += 1,
            1 => changed.waited_pid += 1,
            2 => changed.reaps = 2,
            3 => changed.usage.user.seconds = -1,
            4 => changed.usage.system.micros = 1_000_000,
            5 => changed.usage.counters[0] = -1,
            6 => changed.started += 1,
            7 => changed.status = 127,
            _ => unreachable!(),
        }
        let refusal = running
            .observation(changed, Path::new("/proc"))
            .unwrap_err();
        eprintln!("REPORT_MUTATION {mutation}: {refusal}");
    }
    let mut cleanup = actual;
    cleanup.cleanup_error = 3;
    cleanup.reason = 1;
    let observation = running.observation(cleanup, Path::new("/proc")).unwrap();
    assert!(!observation.accepted());
    assert!(!observation.cleanup_complete);
    assert_eq!(observation.failure.as_deref(), Some("timeout"));
    let missing = running
        .observation(actual, &fixture.0.join("missing-procfs"))
        .unwrap();
    assert!(!missing.accepted());
    assert!(!missing.cleanup_complete);
    assert_eq!(
        running
            .poll(&AtomicBool::new(true))
            .unwrap()
            .unwrap()
            .resources
            .waited_pid,
        running.pid()
    );
}
fn fixture_command(mode: &str, path: &Path) -> Command {
    Command {
        argv: vec![
            std::env::current_exe()
                .unwrap()
                .to_string_lossy()
                .into_owned(),
            "--exact".into(),
            "cycle::process::tests::process_fixture_entry".into(),
            "--nocapture".into(),
        ],
        environment: Environment::Exact(BTreeMap::from([
            ("CYCLE_FIXTURE_MODE".into(), mode.into()),
            (
                "CYCLE_FIXTURE_PATH".into(),
                path.to_string_lossy().into_owned(),
            ),
        ])),
        ..command("exit 0")
    }
}
fn await_file(path: &Path) {
    let deadline = std::time::Instant::now() + Duration::from_secs(3);
    while !path.is_file() {
        assert!(
            std::time::Instant::now() < deadline,
            "handshake timed out: {}",
            path.display()
        );
        std::thread::park_timeout(Duration::from_millis(5));
    }
}
static FIXTURE_INTERRUPTED: AtomicBool = AtomicBool::new(false);
extern "C" fn fixture_signal(_: i32) {
    FIXTURE_INTERRUPTED.store(true, Ordering::Release);
}
#[test]
fn process_fixture_entry() {
    let Ok(mode) = std::env::var("CYCLE_FIXTURE_MODE") else {
        return;
    };
    let path = PathBuf::from(std::env::var_os("CYCLE_FIXTURE_PATH").unwrap());
    match mode.as_str() {
        "heavy" | "unrelated" => {
            let mut memory = vec![1u8; if mode == "heavy" { 64 << 20 } else { 128 << 20 }];
            let cpu = || {
                let time = rustix::time::clock_gettime(rustix::time::ClockId::ProcessCPUTime);
                i128::from(time.tv_sec) * 1_000_000_000 + i128::from(time.tv_nsec)
            };
            let until = cpu() + 100_000_000;
            while cpu() < until {
                for page in memory.chunks_mut(4096) {
                    page[0] = std::hint::black_box(page[0]).wrapping_add(1);
                }
            }
            std::hint::black_box(memory);
        }
        "light" => std::thread::park_timeout(Duration::from_millis(150)),
        "doublefork" | "middle" => {
            if mode == "middle" {
                rustix::process::setsid().unwrap();
            }
            let child = fixture_command(
                if mode == "doublefork" {
                    "middle"
                } else {
                    "escaped"
                },
                &path,
            );
            let mut host = std::process::Command::new(&child.argv[0]);
            host.args(&child.argv[1..])
                .env_clear()
                .envs(child.environment.values());
            let _child = crate::fork_order::spawn(&mut host).unwrap();
            await_file(&path);
        }
        "escaped" => {
            rustix::process::setsid().unwrap();
            std::fs::write(path, std::process::id().to_string()).unwrap();
            std::thread::park_timeout(Duration::from_secs(30));
        }
        "signal-owner" => {
            unsafe extern "C" {
                fn signal(number: i32, handler: usize) -> usize;
            }
            // SAFETY: isolated fixture process; handler only stores to a lock-free atomic.
            unsafe {
                signal(2, fixture_signal as *const () as usize);
            }
            let mut runner = Runner::create(&path.with_extension("inner"), None).unwrap();
            let running = runner
                .start(
                    "sleep",
                    command("exec sleep 30"),
                    &FIXTURE_INTERRUPTED,
                    |_| Ok(()),
                )
                .unwrap();
            std::fs::write(&path, std::process::id().to_string()).unwrap();
            let completed = running.wait(&FIXTURE_INTERRUPTED).unwrap();
            assert_eq!(completed.failure.as_deref(), Some("interrupted"));
            assert!(completed.cleanup_complete);
            std::fs::write(
                path.with_extension("result"),
                serde_json::to_vec(&completed).unwrap(),
            )
            .unwrap();
            std::process::exit(130);
        }
        _ => panic!("unknown fixture mode"),
    }
}
#[test]
fn concurrent_accounting_excludes_unrelated_child_cpu_and_rss() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let mut unrelated = std::process::Command::new(std::env::current_exe().unwrap());
    let cmd = fixture_command("unrelated", &fixture.0.join("unused"));
    unrelated
        .args(&cmd.argv[1..])
        .env_clear()
        .envs(cmd.environment.values())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null());
    let mut other = crate::fork_order::spawn(&mut unrelated).unwrap();
    let heavy = runner
        .start(
            "heavy",
            fixture_command("heavy", &fixture.0.join("unused")),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let light = runner
        .start(
            "light",
            fixture_command("light", &fixture.0.join("unused")),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let heavy = heavy.wait(&AtomicBool::new(false)).unwrap();
    let light = light.wait(&AtomicBool::new(false)).unwrap();
    let other_status = other.wait().unwrap();
    assert!(other_status.success());
    assert!(heavy.accepted());
    assert!(light.accepted());
    assert!(heavy.resources.user_nanos + heavy.resources.system_nanos >= 100_000_000);
    assert!(
        heavy.resources.user_nanos + heavy.resources.system_nanos
            > light.resources.user_nanos + light.resources.system_nanos + 50_000_000
    );
    assert!(heavy.resources.peak_rss_bytes >= 64 << 20);
    assert!(heavy.resources.peak_rss_bytes > light.resources.peak_rss_bytes + 32 * 1024 * 1024);
    eprintln!(
        "ISOLATION_PROOF heavy={} light={} unrelated_pid={}",
        serde_json::to_string(&heavy).unwrap(),
        serde_json::to_string(&light).unwrap(),
        other.id()
    );
}
#[test]
fn actual_double_fork_and_signal_interruption_leave_no_owned_children() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let escaped = fixture.0.join("escaped");
    let record = run(
        &mut runner,
        "doublefork",
        fixture_command("doublefork", &escaped),
    );
    let descendant = std::fs::read_to_string(escaped).unwrap();
    assert!(record.cleanup_complete);
    assert!(!record.accepted());
    assert!(record.descendants_reaped >= 1);
    assert!(!Path::new(&format!("/proc/{}", descendant.trim())).exists());
    let marker = fixture.0.join("signal");
    let running = runner
        .start(
            "signal-owner",
            fixture_command("signal-owner", &marker),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    await_file(&marker);
    let owner: i32 = std::fs::read_to_string(&marker).unwrap().parse().unwrap();
    assert_eq!(u32::try_from(owner).unwrap(), running.pid());
    rustix::process::kill_process(
        rustix::process::Pid::from_raw(owner).unwrap(),
        rustix::process::Signal::INT,
    )
    .unwrap();
    let owner = running.wait(&AtomicBool::new(false)).unwrap();
    assert_eq!(owner.exit, 130);
    assert!(owner.cleanup_complete);
    let child: Completed =
        serde_json::from_slice(&std::fs::read(marker.with_extension("result")).unwrap()).unwrap();
    assert!(child.cleanup_complete);
    assert_eq!(child.exit, -9);
    assert!(!Path::new(&format!("/proc/{}", child.pid)).exists());
    eprintln!(
        "SIGNAL_PROOF owner={} child={}",
        serde_json::to_string(&owner).unwrap(),
        serde_json::to_string(&child).unwrap()
    );
}
#[test]
fn interruption_after_kernel_reap_retains_exit_without_false_success() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    assert!(run(&mut runner, "control", command("exit 0")).accepted());
    let running = runner
        .start(
            "after-reap",
            command("exit 0"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    let supervisor = running.registration.parent_pid;
    let deadline = std::time::Instant::now() + Duration::from_secs(3);
    loop {
        let stat = std::fs::read_to_string(format!("/proc/{supervisor}/stat")).unwrap();
        if procfs::parse(&stat).unwrap().state == b'Z' {
            break;
        }
        assert!(std::time::Instant::now() < deadline);
        std::thread::park_timeout(Duration::from_millis(5));
    }
    let completed = running.wait(&AtomicBool::new(true)).unwrap();
    assert_eq!(completed.exit, 0);
    assert_eq!(completed.resources.reaps, 1);
    assert!(completed.cleanup_complete);
    assert!(!completed.accepted());
    assert_eq!(completed.failure.as_deref(), Some("interrupted"));
}
#[test]
fn aggregate_cleanup_failure_still_reaps_every_busy_worker() {
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let busy = BusyWorkers::start(&mut runner, "busy-grouped", &AtomicBool::new(false)).unwrap();
    let pids = busy.pids();
    std::fs::write(
        fixture.0.join("busy-grouped-busy-0.process.terminal"),
        b"collision",
    )
    .unwrap();
    assert!(busy.finish().is_err());
    for pid in pids {
        assert!(!Path::new(&format!("/proc/{pid}")).exists());
    }
}
#[test]
fn inherited_writable_fixture_descriptor_does_not_outlive_spawn() {
    use std::os::unix::fs::PermissionsExt;
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let path = fixture.0.join("executable");
    let mut writer = File::create(&path).unwrap();
    writer.write_all(b"#!/bin/sh\nexit 0\n").unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o700)).unwrap();
    let running = runner
        .start(
            "held",
            command("exec sleep 30"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    drop(writer);
    let mut executable = command("exit 0");
    executable.argv = vec![path.to_string_lossy().into_owned()];
    assert!(run(&mut runner, "execute", executable).accepted());
    assert!(
        running
            .wait(&AtomicBool::new(true))
            .unwrap()
            .cleanup_complete
    );
}
#[test]
fn cleanup_deadline_and_terminal_publication_are_fail_closed() {
    assert!(sys::cleanup_budget(100, 101).is_ok());
    assert_eq!(sys::cleanup_budget(101, 101), Err(3));
    assert_eq!(sys::cleanup_budget(-1, 101), Err(3));
    let fixture = Fixture::new();
    let mut runner = fixture.runner();
    let mut running = runner
        .start(
            "publication",
            command("exit 0"),
            &AtomicBool::new(false),
            |_| Ok(()),
        )
        .unwrap();
    std::fs::write(fixture.0.join("publication.process.terminal"), b"collision").unwrap();
    let deadline = std::time::Instant::now() + Duration::from_secs(3);
    loop {
        match running.poll(&AtomicBool::new(false)) {
            Ok(None) => {
                assert!(std::time::Instant::now() < deadline);
                std::thread::park_timeout(Duration::from_millis(5));
            }
            Err(_) => break,
            Ok(Some(_)) => panic!("publication failure was accepted"),
        }
    }
    assert!(
        !running
            .poll(&AtomicBool::new(false))
            .unwrap()
            .unwrap()
            .accepted()
    );
    assert!(runner.active_intervals().is_err());
    assert!(
        runner
            .start("later", command("exit 0"), &AtomicBool::new(false), |_| Ok(
                ()
            ))
            .is_err()
    );
}
