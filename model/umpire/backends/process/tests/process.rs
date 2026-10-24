//! Compatibility controls for native backend process ownership.
#![forbid(unsafe_code)]
use std::{
    fs,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    thread,
    time::{Duration, Instant},
};
use umpire_backend_process::{
    Limits, Refusal, Scratch, bounded_text, java_command, run_process, run_process_cancelled,
};

const FIXTURE: &str = env!("CARGO_BIN_EXE_umpire-process-fixture");

fn root() -> Result<PathBuf, Refusal> {
    Ok(Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?)
}
fn fixture(mode: &str) -> Command {
    let mut command = Command::new(FIXTURE);
    command.arg(mode);
    command
}
fn join(worker: thread::JoinHandle<()>) -> Result<(), Refusal> {
    worker.join().map_err(|_| Refusal("control thread panicked".into()))
}
fn wait_ready(path: &Path) -> bool {
    let start = Instant::now();
    while start.elapsed() < Duration::from_secs(5) {
        if fs::read_to_string(path).is_ok_and(|text| text.split_whitespace().count() >= 2) {
            return true;
        }
        thread::sleep(Duration::from_millis(5));
    }
    false
}
struct TreeCleanup(PathBuf);
impl Drop for TreeCleanup {
    fn drop(&mut self) {
        if let Ok(pids) = fs::read_to_string(&self.0) {
            for pid in pids.split_whitespace() {
                let _ = Command::new("kill")
                    .args(["-KILL", pid])
                    .stdout(Stdio::null())
                    .stderr(Stdio::null())
                    .status();
            }
        }
    }
}
fn assert_reaped(pids: &Path) -> Result<(), Refusal> {
    for pid in fs::read_to_string(pids)?.split_whitespace() {
        assert!(
            !Path::new("/proc").join(pid).exists(),
            "owned pid {pid} not reaped before fallback cleanup"
        );
    }
    Ok(())
}

#[test]
fn cold_native_execution_does_not_need_python_or_a_sibling_supervisor() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let mut command = fixture("echo");
    command.env("PATH", "");
    let (status, output) =
        run_process(&mut command, &scratch.0, "native", Limits::default(), None)?;
    assert!(status.success());
    assert_eq!(output, "native\n");
    scratch.finish(Ok(()))
}

#[test]
fn direct_helper_timeout_retains_output_and_cleans_scratch() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let failure = run_process_cancelled(
        &mut fixture("started"),
        &scratch.0,
        "direct",
        Limits { timeout: Duration::from_secs(1), ..Limits::default() },
        None,
        &AtomicBool::new(false),
    )
    .err()
    .ok_or_else(|| Refusal("direct helper exceeded timeout".into()))?;
    assert!(failure.refusal.0.contains("limit"));
    assert_eq!(failure.output, "started\n");
    let path = scratch.0.clone();
    scratch.finish::<_, Refusal>(Ok(()))?;
    assert!(!path.exists());
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn timeout_cleans_descendants_before_return() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let pids = scratch.0.join("pids");
    let _cleanup = TreeCleanup(pids.clone());
    let mut command = fixture("tree");
    command.arg(&pids).arg(scratch.0.join("supervisor"));
    let limits = Limits { timeout: Duration::from_secs(1), ..Limits::default() };
    assert!(run_process(&mut command, &scratch.0, "tree", limits, None).is_err());
    assert_reaped(&pids)?;
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn cancellation_reaps_descendants_in_separate_sessions_and_closes_stdin() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let pids = scratch.0.join("pids");
    let _cleanup = TreeCleanup(pids.clone());
    let mut command = fixture("tree");
    command.arg(&pids).arg(scratch.0.join("supervisor"));
    let cancelled = Arc::new(AtomicBool::new(false));
    let flag = cancelled.clone();
    let ready = pids.clone();
    let cancel = thread::spawn(move || {
        let _ = wait_ready(&ready);
        flag.store(true, Ordering::Relaxed);
    });
    let result = run_process_cancelled(
        &mut command,
        &scratch.0,
        "cancel",
        Limits::default(),
        None,
        &cancelled,
    );
    join(cancel)?;
    assert!(result.is_err_and(|e| e.refusal.0.contains("cancelled")));
    assert_reaped(&pids)?;
    let path = scratch.0.clone();
    scratch.finish::<_, Refusal>(Ok(()))?;
    assert!(!path.exists());
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn java_environment_is_clean_and_supervisor_group_is_private() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let mut command = java_command(Path::new(FIXTURE));
    command.arg("policy");
    let (status, output) =
        run_process(&mut command, &scratch.0, "policy", Limits::default(), None)?;
    assert!(status.success());
    assert_eq!(output, "None\nTrue\n");
    scratch.finish(Ok(()))
}

#[test]
fn substituted_log_symlink_cannot_modify_an_external_file() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let work = Scratch::new(&scratch.0)?;
    let protected = scratch.0.join("protected");
    fs::write(&protected, "unchanged")?;
    let mut command = fixture("symlink");
    command.arg(&protected).current_dir(&work.0);
    let result = run_process(&mut command, &work.0, "capture", Limits::default(), None);
    assert_eq!(fs::read_to_string(&protected)?, "unchanged");
    assert!(result.is_err());
    work.finish::<_, Refusal>(Ok(()))?;
    scratch.finish(Ok(()))
}

#[test]
fn substituted_log_fifo_returns_without_waiting_for_a_reader() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let work = scratch.0.clone();
    let (send, receive) = std::sync::mpsc::channel();
    let worker = thread::spawn(move || {
        let mut command = fixture("fifo");
        command.current_dir(&work);
        let result = run_process(&mut command, &work, "capture", Limits::default(), None);
        let _ = send.send(result);
    });
    let result = receive.recv_timeout(Duration::from_secs(2));
    let blocked = result.is_err();
    let unblock = if blocked {
        Some(fs::OpenOptions::new().read(true).write(true).open(scratch.0.join("capture.stdout"))?)
    } else {
        None
    };
    let result = result
        .or_else(|_| receive.recv_timeout(Duration::from_secs(2)))
        .map_err(|e| Refusal(e.to_string()))?;
    join(worker)?;
    drop(unblock);
    assert!(!blocked, "log publication waited for a FIFO reader");
    assert!(result.is_err());
    scratch.finish(Ok(()))
}

#[test]
fn failed_captures_keep_one_combined_budget_and_refuse_invalid_utf8() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    for (label, limit, expected) in [("combined", 6, "abcdef"), ("utf8", 1, "")] {
        let limits = Limits { output_bytes: limit, ..Limits::default() };
        let failure = run_process_cancelled(
            &mut fixture(label),
            &scratch.0,
            label,
            limits,
            None,
            &AtomicBool::new(false),
        )
        .err()
        .ok_or_else(|| Refusal("malformed or oversized output admitted".into()))?;
        assert!(failure.output.len() as u64 <= limit);
        assert_eq!(failure.output, expected);
        if label == "utf8" {
            assert!(failure.refusal.0.contains("non-UTF8"));
            assert_eq!(fs::read(scratch.0.join("utf8.stdout"))?, [255]);
        }
    }
    scratch.finish(Ok(()))
}

#[test]
fn parent_allocator_threads_do_not_deadlock_forked_supervisors() -> Result<(), Refusal> {
    let stop = Arc::new(AtomicBool::new(false));
    let workers: Vec<_> = (0..4)
        .map(|_| {
            let stop = stop.clone();
            thread::spawn(move || {
                while !stop.load(Ordering::Relaxed) {
                    let values: Vec<_> =
                        (0..128).map(|i| format!("allocation-{i}-{}", "x".repeat(100))).collect();
                    std::hint::black_box(values);
                }
            })
        })
        .collect();
    let result = (|| {
        let runs: Vec<_> = (0..4)
            .map(|_| {
                thread::spawn(|| -> Result<(), Refusal> {
                    let scratch = Scratch::new(&root()?)?;
                    for index in 0..12 {
                        let (status, output) = run_process(
                            &mut fixture("echo"),
                            &scratch.0,
                            &format!("stress-{index}"),
                            Limits { timeout: Duration::from_secs(5), ..Limits::default() },
                            None,
                        )?;
                        assert!(status.success());
                        assert_eq!(output, "native\n");
                    }
                    scratch.finish(Ok(()))
                })
            })
            .collect();
        for run in runs {
            run.join().map_err(|_| Refusal("fork stress panicked".into()))??;
        }
        Ok(())
    })();
    stop.store(true, Ordering::Relaxed);
    for worker in workers {
        join(worker)?;
    }
    result
}

#[test]
fn bounded_reads_dump_and_scratch_fail_closed() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let text = scratch.0.join("text");
    fs::write(&text, "abcd")?;
    assert_eq!(bounded_text(&text, 4)?, "abcd");
    assert!(bounded_text(&text, 3).is_err());
    let link = scratch.0.join("link");
    std::os::unix::fs::symlink(&text, &link)?;
    assert!(bounded_text(&link, 4).is_err());
    assert!(bounded_text(&scratch.0, 4).is_err());
    for (label, dump) in [("dump-budget", &text), ("dump-link", &link)] {
        assert!(
            run_process(
                &mut fixture("echo"),
                &scratch.0,
                label,
                Limits { dump_bytes: 3, ..Limits::default() },
                Some(dump)
            )
            .is_err()
        );
    }
    assert!(Scratch::new_in(&scratch.0, "../escape").is_err());
    assert!(Scratch::new_in(&scratch.0, "link/child").is_err());
    scratch.finish(Ok(()))
}

fn non_root() -> Result<bool, Refusal> {
    for args in [["-u"], ["-ru"]] {
        let output = Command::new("id").args(args).output()?;
        if !output.status.success() {
            return Err(Refusal("cannot inspect control UID".into()));
        }
        if output.stdout == b"0\n" {
            return Ok(false);
        }
    }
    Ok(true)
}
fn restricted(script: &Path) -> Command {
    let mut command = Command::new("bash");
    command
        .args(["-c", "ulimit -u 0; exec bash \"$1\"", "control"])
        .arg(script)
        .env("BASH_ENV", "/dev/null");
    command
}

#[test]
#[cfg(target_os = "linux")]
fn simulated_darwin_helper_cannot_start_an_escaped_descendant() -> Result<(), Refusal> {
    if !non_root()? {
        return Ok(());
    }
    let scratch = Scratch::new(&root()?)?;
    let mut command = Command::new("bash");
    command
        .args(["-c", "ulimit -u 0; exec \"$1\" descendant-attempt", "control", FIXTURE])
        .env("BASH_ENV", "/dev/null");
    let (status, output) = run_process(&mut command, &scratch.0, "nproc", Limits::default(), None)?;
    assert_eq!(status.code(), Some(75));
    assert_eq!(output, "descendant creation refused\n");
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn installer_containment_preflight_requires_no_descendants() -> Result<(), Refusal> {
    if !non_root()? {
        return Ok(());
    }
    let scratch = Scratch::new(&root()?)?;
    let workspace = scratch.0.join("workspace");
    let scripts = workspace.join("model/scripts");
    fs::create_dir_all(&scripts)?;
    let script = scripts.join("fetch-tlc.sh");
    fs::copy(root()?.join("model/scripts/fetch-tlc.sh"), &script)?;
    let outside = scratch.0.join("outside");
    fs::create_dir(&outside)?;
    std::os::unix::fs::symlink(&outside, workspace.join("model/build"))?;
    let (status, output) = run_process(
        &mut restricted(&script),
        &scratch.0,
        "preflight",
        Limits { timeout: Duration::from_secs(2), ..Limits::default() },
        None,
    )?;
    assert!(!status.success());
    assert!(output.contains("TLC cache escapes repository"), "{output}");
    assert!(!outside.join("tlc").exists());
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn diagnostic_text_cannot_make_a_hung_preflight_pass() -> Result<(), Refusal> {
    if !non_root()? {
        return Ok(());
    }
    let scratch = Scratch::new(&root()?)?;
    let script = scratch.0.join("hang.sh");
    fs::write(&script, "printf 'TLC cache escapes repository\\n' >&2; while :; do :; done\n")?;
    let failure = run_process_cancelled(
        &mut restricted(&script),
        &scratch.0,
        "hung",
        Limits { timeout: Duration::from_millis(100), ..Limits::default() },
        None,
        &AtomicBool::new(false),
    )
    .err()
    .ok_or_else(|| Refusal("hung preflight was admitted".into()))?;
    assert!(failure.refusal.0.contains("limit"));
    assert!(failure.output.contains("TLC cache escapes repository"));
    scratch.finish(Ok(()))
}
