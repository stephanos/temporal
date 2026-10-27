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
    Ok(Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?)
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
struct ResumeProcess(String);
impl Drop for ResumeProcess {
    fn drop(&mut self) {
        let _ = Command::new("kill")
            .args(["-CONT", &self.0])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();
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
        run_process(&mut command, scratch.path(), "native", Limits::default(), None)?;
    assert!(status.success());
    assert_eq!(output, "native\n");
    scratch.finish(Ok(()))
}

#[test]
fn direct_helper_timeout_retains_output_and_cleans_scratch() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let failure = run_process_cancelled(
        &mut fixture("started"),
        scratch.path(),
        "direct",
        Limits { timeout: Duration::from_secs(1), ..Limits::default() },
        None,
        &AtomicBool::new(false),
    )
    .err()
    .ok_or_else(|| Refusal("direct helper exceeded timeout".into()))?;
    assert!(failure.refusal.0.contains("limit"));
    assert_eq!(failure.output, "started\n");
    let path = scratch.path().to_path_buf();
    scratch.finish::<_, Refusal>(Ok(()))?;
    assert!(!path.exists());
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn timeout_cleans_descendants_before_return() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let pids = scratch.path().join("pids");
    let _cleanup = TreeCleanup(pids.clone());
    let mut command = fixture("tree");
    command.arg(&pids).arg(scratch.path().join("supervisor"));
    let limits = Limits { timeout: Duration::from_secs(1), ..Limits::default() };
    assert!(run_process(&mut command, scratch.path(), "tree", limits, None).is_err());
    assert_reaped(&pids)?;
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn cancellation_reaps_descendants_in_separate_sessions_and_closes_stdin() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let pids = scratch.path().join("pids");
    let _cleanup = TreeCleanup(pids.clone());
    let mut command = fixture("tree");
    command.arg(&pids).arg(scratch.path().join("supervisor"));
    let cancelled = Arc::new(AtomicBool::new(false));
    let flag = cancelled.clone();
    let ready = pids.clone();
    let cancel = thread::spawn(move || {
        let _ = wait_ready(&ready);
        flag.store(true, Ordering::Relaxed);
    });
    let result = run_process_cancelled(
        &mut command,
        scratch.path(),
        "cancel",
        Limits::default(),
        None,
        &cancelled,
    );
    join(cancel)?;
    assert!(result.is_err_and(|e| e.refusal.0.contains("cancelled")));
    assert_reaped(&pids)?;
    let path = scratch.path().to_path_buf();
    scratch.finish::<_, Refusal>(Ok(()))?;
    assert!(!path.exists());
    Ok(())
}

#[test]
fn java_environment_is_clean_and_supervisor_group_is_private() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let mut command = java_command(Path::new(FIXTURE));
    command.arg("policy");
    let (status, output) =
        run_process(&mut command, scratch.path(), "policy", Limits::default(), None)?;
    assert!(status.success());
    assert_eq!(output, "None\nTrue\n");
    scratch.finish(Ok(()))
}

#[test]
fn substituted_log_symlink_cannot_modify_an_external_file() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let work = Scratch::new(scratch.path())?;
    let protected = scratch.path().join("protected");
    fs::write(&protected, "unchanged")?;
    let mut command = fixture("symlink");
    command.arg(&protected).current_dir(work.path());
    let result = run_process(&mut command, work.path(), "capture", Limits::default(), None);
    assert_eq!(fs::read_to_string(&protected)?, "unchanged");
    assert!(result.is_err());
    work.finish::<_, Refusal>(Ok(()))?;
    scratch.finish(Ok(()))
}

#[test]
fn substituted_log_fifo_returns_without_waiting_for_a_reader() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    assert!(
        Command::new("mkfifo").arg(scratch.path().join("replacement.fifo")).status()?.success()
    );
    let work = scratch.path().to_path_buf();
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
        Some(
            fs::OpenOptions::new()
                .read(true)
                .write(true)
                .open(scratch.path().join("capture.stdout"))?,
        )
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
            scratch.path(),
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
            assert_eq!(fs::read(scratch.path().join("utf8.stdout"))?, [255]);
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
    let spawning = {
        let stop = stop.clone();
        thread::spawn(move || -> Result<(), Refusal> {
            while !stop.load(Ordering::Relaxed) {
                if !Command::new("true").status()?.success() {
                    return Err(Refusal("concurrent spawn failed".into()));
                }
            }
            Ok(())
        })
    };
    let result = (|| {
        let runs: Vec<_> = (0..4)
            .map(|_| {
                thread::spawn(|| -> Result<(), Refusal> {
                    let scratch = Scratch::new(&root()?)?;
                    for index in 0..12 {
                        let (status, output) = run_process(
                            &mut fixture("echo"),
                            scratch.path(),
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
    spawning.join().map_err(|_| Refusal("spawn stress panicked".into()))??;
    result
}

#[test]
fn bounded_reads_dump_and_scratch_fail_closed() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let text = scratch.path().join("text");
    fs::write(&text, "abcd")?;
    assert_eq!(bounded_text(&text, 4)?, "abcd");
    assert!(bounded_text(&text, 3).is_err());
    let link = scratch.path().join("link");
    std::os::unix::fs::symlink(&text, &link)?;
    assert!(bounded_text(&link, 4).is_err());
    assert!(bounded_text(scratch.path(), 4).is_err());
    let invalid = scratch.path().join("invalid");
    fs::write(&invalid, [255])?;
    assert!(bounded_text(&invalid, 4).is_err());
    let fifo = scratch.path().join("dump-fifo");
    assert!(Command::new("mkfifo").arg(&fifo).status()?.success());
    assert!(bounded_text(&fifo, 4).is_err());
    let deep = scratch.path().join("deep");
    let mut directory = deep.clone();
    for _ in 0..66 {
        fs::create_dir(&directory)?;
        directory.push("d");
    }
    let wide = scratch.path().join("wide");
    fs::create_dir(&wide)?;
    for index in 0..4096 {
        fs::write(wide.join(index.to_string()), [])?;
    }
    for (label, dump) in [
        ("dump-budget", &text),
        ("dump-link", &link),
        ("dump-fifo", &fifo),
        ("dump-depth", &deep),
        ("dump-entries", &wide),
    ] {
        assert!(
            run_process(
                &mut fixture("echo"),
                scratch.path(),
                label,
                Limits { dump_bytes: 3, ..Limits::default() },
                Some(dump)
            )
            .is_err()
        );
    }
    for relative in ["../escape", "/absolute", "link/child", "text/child"] {
        assert!(Scratch::new_in(scratch.path(), relative).is_err());
    }
    let removed = Scratch::new(scratch.path())?;
    fs::remove_dir(removed.path())?;
    let failure = removed
        .finish::<(), Refusal>(Err(Refusal("prior failure".into())))
        .err()
        .ok_or_else(|| Refusal("cleanup failure was hidden".into()))?;
    assert!(failure.0.contains("scratch cleanup failed"));
    assert!(failure.0.contains("prior failure"));
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
    let mut command = fixture("restricted-exec");
    command.arg(script).env("BASH_ENV", "/dev/null");
    command
}

#[test]
#[cfg(target_os = "linux")]
fn simulated_darwin_helper_cannot_start_an_escaped_descendant() -> Result<(), Refusal> {
    if !non_root()? {
        return Ok(());
    }
    let scratch = Scratch::new(&root()?)?;
    let mut command = fixture("descendant-attempt");
    let (status, output) =
        run_process(&mut command, scratch.path(), "nproc", Limits::default(), None)?;
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
    let workspace = scratch.path().join("workspace");
    let scripts = workspace.join("scripts");
    fs::create_dir_all(&scripts)?;
    let script = scripts.join("fetch-tlc.sh");
    fs::copy(root()?.join("scripts/fetch-tlc.sh"), &script)?;
    let outside = scratch.path().join("outside");
    fs::create_dir(&outside)?;
    std::os::unix::fs::symlink(&outside, workspace.join("build"))?;
    let (status, output) = run_process(
        &mut restricted(&script),
        scratch.path(),
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
    let script = scratch.path().join("hang.sh");
    fs::write(&script, "printf 'TLC cache escapes repository\\n' >&2; while :; do :; done\n")?;
    let failure = run_process_cancelled(
        &mut restricted(&script),
        scratch.path(),
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

#[test]
#[cfg(target_os = "linux")]
fn owner_group_and_direct_supervisor_termination_reap_owned_tree() -> Result<(), Refusal> {
    for mode in ["owner-group", "owner-group-kill", "supervisor"] {
        let scratch = Scratch::new(&root()?)?;
        let output = fixture("lifecycle-harness").arg(mode).arg(scratch.path()).output()?;
        assert!(output.status.success(), "{mode}: {}", String::from_utf8_lossy(&output.stderr));
        assert_reaped(&scratch.path().join("pids"))?;
        scratch.finish::<_, Refusal>(Ok(()))?;
    }
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn owner_death_terminates_and_reaps_separate_session_descendants() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let output = fixture("lifecycle-harness").arg("owner-eof").arg(scratch.path()).output()?;
    assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
    assert_eq!(output.stderr, b"");
    assert_reaped(&scratch.path().join("pids"))?;
    scratch.finish(Ok(()))
}

#[test]
fn launch_status_and_prepared_command_contracts_are_preserved() -> Result<(), Refusal> {
    use std::os::unix::fs::PermissionsExt;
    let scratch = Scratch::new(&root()?)?;
    let nonexec = scratch.path().join("not-executable");
    fs::write(&nonexec, "not executable")?;
    fs::set_permissions(&nonexec, fs::Permissions::from_mode(0o600))?;
    let missing = scratch.path().join("missing-executable");
    for (label, executable) in [("missing", &missing), ("nonexec", &nonexec)] {
        let (status, _) = run_process(
            &mut Command::new(executable),
            scratch.path(),
            label,
            Limits::default(),
            None,
        )?;
        assert_eq!(status.code(), Some(1), "{label}");
    }
    let mut missing_cwd = fixture("echo");
    missing_cwd.current_dir(scratch.path().join("missing-directory"));
    let failure = run_process(&mut missing_cwd, scratch.path(), "cwd", Limits::default(), None)
        .err()
        .ok_or_else(|| Refusal("missing working directory was admitted".into()))?;
    assert!(failure.0.contains("cannot spawn cwd"), "{failure}");
    std::os::unix::fs::symlink(FIXTURE, scratch.path().join("tool"))?;
    for (index, program) in ["tool", "./tool"].into_iter().enumerate() {
        let mut command = Command::new(program);
        command
            .args(["argv-env", "space ' \" unicode λ"])
            .current_dir(scratch.path())
            .env("PATH", ".")
            .env("UMPIRE_PROCESS_VALUE", "prepared")
            .env("UMPIRE_PROCESS_REMOVED", "present")
            .env_remove("UMPIRE_PROCESS_REMOVED");
        let (status, output) = run_process(
            &mut command,
            scratch.path(),
            &format!("prepare-{index}"),
            Limits::default(),
            None,
        )?;
        assert!(status.success());
        assert_eq!(output, "arg:space ' \" unicode λ\nvalue:prepared\nremoved:true\n");
    }
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn pthread_atfork_handlers_do_not_run_in_supervisor_or_target() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let output = fixture("poison-atfork").arg(scratch.path()).output()?;
    assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
    scratch.finish(Ok(()))
}

#[test]
fn path_search_tries_candidates_after_exec_errors() -> Result<(), Refusal> {
    use std::os::unix::fs::PermissionsExt;
    let scratch = Scratch::new(&root()?)?;
    let first = scratch.path().join("first");
    let later = scratch.path().join("later");
    fs::create_dir(&first)?;
    fs::create_dir(&later)?;
    std::os::unix::fs::symlink(FIXTURE, later.join("fallback-tool"))?;
    let path = std::env::join_paths([&first, &later]).map_err(|e| Refusal(e.to_string()))?;
    let mut results = Vec::new();
    for (label, unusable) in [
        ("interpreter", "#!/missing/backend-interpreter\n"),
        ("format", "not an executable format\n"),
    ] {
        let candidate = first.join("fallback-tool");
        fs::write(&candidate, unusable)?;
        fs::set_permissions(&candidate, fs::Permissions::from_mode(0o755))?;
        let mut command = Command::new("fallback-tool");
        command.arg("echo").env("PATH", &path);
        let (status, output) =
            run_process(&mut command, scratch.path(), label, Limits::default(), None)?;
        results.push((status.code(), output));
    }
    assert_eq!(results, [(Some(0), "native\n".into()), (Some(0), "native\n".into())]);
    scratch.finish(Ok(()))
}

#[test]
fn scratch_owns_private_unique_directories_and_preserves_finish_results() -> Result<(), Refusal> {
    use std::os::unix::fs::PermissionsExt;
    let first = Scratch::new(&root()?)?;
    let second = Scratch::new(&root()?)?;
    let first_path = first.path().to_path_buf();
    let second_path = second.path().to_path_buf();
    assert_ne!(first_path, second_path);
    assert!(
        first_path.file_name().is_some_and(|name| name.as_encoded_bytes().starts_with(b"run-"))
    );
    assert_eq!(fs::metadata(&first_path)?.permissions().mode() & 0o777, 0o700);
    fs::write(first_path.join("owned"), "content")?;
    assert_eq!(first.finish::<_, Refusal>(Ok(42))?, 42);
    assert!(!first_path.exists());
    drop(second);
    assert!(!second_path.exists());
    let failed = Scratch::new(&root()?)?;
    let failed_path = failed.path().to_path_buf();
    let error = failed
        .finish::<(), Refusal>(Err(Refusal("original result".into())))
        .err()
        .ok_or_else(|| Refusal("original failure was hidden".into()))?;
    assert_eq!(error.0, "original result");
    assert!(!failed_path.exists());
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn owned_child_reaping_includes_other_process_groups() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    let (status, output) = run_process(
        &mut fixture("reap-all-groups"),
        scratch.path(),
        "reap-all-groups",
        Limits { timeout: Duration::from_secs(5), ..Limits::default() },
        None,
    )?;
    assert!(status.success(), "any-child wait failed: {status}: {output}");
    scratch.finish(Ok(()))
}

#[test]
fn orchestration_preserves_streams_status_environment_and_cwd() -> Result<(), Refusal> {
    use umpire_backend_process::{DescendantPolicy, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let stdout = fs::File::create(scratch.path().join("out"))?;
    let stderr = fs::File::create(scratch.path().join("err"))?;
    let mut command = Command::new("/bin/sh");
    command.args(["-c", "printf '%s' \"$UMPIRE_OWNED_VALUE\"; printf error >&2; test \"$PWD\" = \"$UMPIRE_OWNED_CWD\" || exit 99; exit 37"])
        .env("UMPIRE_OWNED_VALUE", "actual-output").env("UMPIRE_OWNED_CWD", scratch.path()).current_dir(scratch.path());
    let mut child = OwnedProcess::spawn(
        &command,
        scratch.path(),
        &stdout,
        &stderr,
        DescendantPolicy::Orchestration,
    )?;
    let start = Instant::now();
    loop {
        if let Some(status) = child.try_wait()? {
            assert_eq!(status.code(), Some(37));
            break;
        }
        assert!(start.elapsed() < Duration::from_secs(5));
        thread::sleep(Duration::from_millis(5));
    }
    assert_eq!(fs::read_to_string(scratch.path().join("out"))?, "actual-output");
    assert_eq!(fs::read_to_string(scratch.path().join("err"))?, "error");
    scratch.finish(Ok(()))
}

#[test]
fn leased_child_adopts_real_descriptor_and_rejects_forged_locator() -> Result<(), Refusal> {
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let lease = JvmLease::inherited_or_acquire()?;
    let stdout = fs::File::create(scratch.path().join("out"))?;
    let stderr = fs::File::create(scratch.path().join("err"))?;
    let mut command = fixture("lease-child");
    command.arg(scratch.path().join("adopted"));
    let mut child = OwnedProcess::spawn_leased(&command, scratch.path(), &stdout, &stderr, &lease)?;
    let start = Instant::now();
    loop {
        if let Some(status) = child.try_wait()? {
            assert!(status.success(), "{}", fs::read_to_string(scratch.path().join("err"))?);
            break;
        }
        assert!(start.elapsed() < Duration::from_secs(5), "inherited lock deadlocked");
        thread::sleep(Duration::from_millis(5));
    }
    assert_eq!(fs::read_to_string(scratch.path().join("adopted"))?, "verified");
    let output = command.env("UMPIRE_JVM_LEASE_FD", "1").output()?;
    assert!(!output.status.success(), "forged descriptor locator admitted");
    let foreign = scratch.path().join("foreign");
    fs::write(&foreign, "unrelated")?;
    let output = Command::new("/bin/sh")
        .args(["-c", "exec 9<\"$1\"; exec \"$2\" lease-child \"$3\"", "forged"])
        .arg(&foreign)
        .arg(FIXTURE)
        .arg(scratch.path().join("forged"))
        .env("UMPIRE_JVM_LEASE_FD", "9")
        .output()?;
    assert!(!output.status.success(), "foreign regular descriptor admitted");
    assert!(!scratch.path().join("forged").exists());
    assert_eq!(fs::read_to_string(foreign)?, "unrelated");
    scratch.finish(Ok(()))
}

#[test]
fn leased_clean_command_preserves_only_explicit_environment() -> Result<(), Refusal> {
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let lease = JvmLease::inherited_or_acquire()?;
    let stdout = fs::File::create(scratch.path().join("out"))?;
    let stderr = fs::File::create(scratch.path().join("err"))?;
    let mut command = Command::new("/bin/sh");
    command.env_clear().env("ONLY_THIS", "kept").args(["-c", "test \"${HOME-unset}\" = unset && test \"$ONLY_THIS\" = kept && test \"$UMPIRE_JVM_LEASE_FD\" = 9"]);
    let mut child =
        OwnedProcess::spawn_leased_clean(&command, scratch.path(), &stdout, &stderr, &lease)?;
    let start = Instant::now();
    loop {
        if let Some(status) = child.try_wait()? {
            assert!(status.success());
            break;
        }
        assert!(start.elapsed() < Duration::from_secs(5));
        thread::sleep(Duration::from_millis(5));
    }
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn default_checker_guardian_retains_inherited_custody_without_delegating_to_target()
-> Result<(), Refusal> {
    use std::os::unix::fs::MetadataExt;
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let allocation = JvmLease::inherited_or_acquire()?;
    let ready = scratch.path().join("ready");
    let release = scratch.path().join("release");
    let stdout = fs::File::create(scratch.path().join("owner.out"))?;
    let stderr = fs::File::create(scratch.path().join("owner.err"))?;
    let mut command = fixture("checker-owner");
    command.arg(scratch.path()).arg(&ready).arg(&release);
    let mut owner =
        OwnedProcess::spawn_leased(&command, scratch.path(), &stdout, &stderr, &allocation)?;
    assert!(wait_ready(&ready), "checker target did not become ready");
    let ids = fs::read_to_string(&ready)?;
    let ids: Vec<_> = ids.split_whitespace().collect();
    assert!(!Path::new("/proc").join(ids[0]).join("fd/9").exists(), "Checker target received FD9");
    let guardian = fs::metadata(format!("/proc/{}/fd/9", ids[1])).map_err(|error| {
        Refusal(format!("Checker guardian lost inherited lease custody: {error}"))
    })?;
    let lock = fs::metadata(
        fs::canonicalize("/tmp")?
            .join(format!("umpire-validation-{}/jvm.lock", rustix::process::getuid().as_raw())),
    )?;
    assert_eq!((guardian.dev(), guardian.ino()), (lock.dev(), lock.ino()));
    fs::write(&release, "release")?;
    let started = Instant::now();
    loop {
        if let Some(status) = owner.try_wait()? {
            assert!(status.success(), "{}", fs::read_to_string(scratch.path().join("owner.err"))?);
            break;
        }
        assert!(started.elapsed() < Duration::from_secs(5));
        thread::sleep(Duration::from_millis(5));
    }
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn checker_inheritance_is_optional_and_invalid_present_custody_refuses_before_launch()
-> Result<(), Refusal> {
    use umpire_backend_process::JvmLease;
    let scratch = Scratch::new(&root()?)?;
    let _allocation = JvmLease::inherited_or_acquire()?;
    for (label, locator, foreign) in [
        ("absent", None, false),
        ("malformed", Some("1"), false),
        ("closed", Some("9"), false),
        ("foreign", Some("9"), true),
    ] {
        let directory = scratch.path().join(label);
        fs::create_dir(&directory)?;
        let ready = directory.join("ready");
        let release = directory.join("release");
        fs::write(&release, "release")?;
        let victim = directory.join("foreign-file");
        fs::write(&victim, "unchanged")?;
        let mut command = Command::new("/bin/sh");
        let script = if foreign {
            "exec 9<\"$1\"; exec \"$2\" checker-owner \"$3\" \"$4\" \"$5\""
        } else {
            "exec 9<&-; exec \"$2\" checker-owner \"$3\" \"$4\" \"$5\""
        };
        command
            .args(["-c", script, "checker-inheritance"])
            .arg(&victim)
            .arg(FIXTURE)
            .arg(&directory)
            .arg(&ready)
            .arg(&release)
            .env_remove("UMPIRE_JVM_LEASE_FD")
            .stdout(fs::File::create(directory.join("out"))?)
            .stderr(fs::File::create(directory.join("err"))?);
        if let Some(locator) = locator {
            command.env("UMPIRE_JVM_LEASE_FD", locator);
        }
        let mut child = command.spawn()?;
        let started = Instant::now();
        let status = loop {
            if let Some(status) = child.try_wait()? {
                break status;
            }
            if started.elapsed() > Duration::from_secs(5) {
                child.kill()?;
                child.wait()?;
                return Err(Refusal(format!(
                    "{label}: optional Checker custody acquired or blocked on host allocation"
                )));
            }
            thread::sleep(Duration::from_millis(5));
        };
        assert_eq!(
            status.success(),
            locator.is_none(),
            "{label}: {}",
            fs::read_to_string(directory.join("err"))?
        );
        assert_eq!(
            ready.exists(),
            locator.is_none(),
            "{label}: invalid inheritance reached target"
        );
        assert_eq!(fs::read_to_string(victim)?, "unchanged");
    }
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn linux_root_reaps_paused_nested_guardian_before_releasing_lease() -> Result<(), Refusal> {
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let allocation = JvmLease::inherited_or_acquire()?;
    let ready = scratch.path().join("ready");
    let _cleanup = TreeCleanup(ready.clone());
    let out = fs::File::create(scratch.path().join("root.out"))?;
    let err = fs::File::create(scratch.path().join("root.err"))?;
    let mut command = fixture("lifecycle-carrier");
    command.arg(scratch.path());
    let mut owner = OwnedProcess::spawn_leased(&command, scratch.path(), &out, &err, &allocation)?;
    drop(allocation);
    assert!(wait_ready(&ready));
    let ids = fs::read_to_string(&ready)?;
    let guardian =
        ids.split_whitespace().nth(1).ok_or_else(|| Refusal("missing guardian".into()))?;
    assert!(Command::new("kill").args(["-STOP", guardian]).status()?.success());
    let _resume = ResumeProcess(guardian.to_owned());
    fs::write(scratch.path().join("carrier-release"), "release")?;
    let started = Instant::now();
    loop {
        if let Some(status) = owner.try_wait()? {
            assert_eq!(status.code(), Some(37));
            break;
        }
        let lock = fs::canonicalize("/tmp")?
            .join(format!("umpire-validation-{}/jvm.lock", rustix::process::getuid().as_raw()));
        assert!(
            matches!(fs::File::open(lock)?.try_lock(), Err(fs::TryLockError::WouldBlock)),
            "cleanup released JVM lease"
        );
        assert!(started.elapsed() < Duration::from_secs(5), "Linux adopted guardian did not drain");
        thread::sleep(Duration::from_millis(5));
    }
    assert_reaped(&ready)?;
    scratch.finish(Ok(()))
}

#[test]
#[cfg(target_os = "linux")]
fn root_broadcast_cancels_checker_while_rust_owner_is_stopped() -> Result<(), Refusal> {
    use umpire_backend_process::{DescendantPolicy, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let ready = scratch.path().join("ready");
    let _cleanup = TreeCleanup(ready.clone());
    let out = fs::File::create(scratch.path().join("root.out"))?;
    let err = fs::File::create(scratch.path().join("root.err"))?;
    let mut command = fixture("lifecycle-carrier");
    command.arg(scratch.path());
    let mut owner =
        OwnedProcess::spawn(&command, scratch.path(), &out, &err, DescendantPolicy::Orchestration)?;
    assert!(wait_ready(&ready));
    let rust_owner = fs::read_to_string(scratch.path().join("checker-owner-pid"))?;
    assert!(Command::new("kill").args(["-STOP", rust_owner.trim()]).status()?.success());
    let _resume = ResumeProcess(rust_owner.trim().to_owned());
    fs::write(scratch.path().join("carrier-release"), "release")?;
    let ids = fs::read_to_string(&ready)?;
    let java =
        ids.split_whitespace().next().ok_or_else(|| Refusal("missing checker PID".into()))?;
    let started = Instant::now();
    while Path::new("/proc").join(java).exists() {
        assert!(
            started.elapsed() < Duration::from_secs(5),
            "checker guardian depended on stopped Rust owner"
        );
        thread::sleep(Duration::from_millis(5));
    }
    assert!(Command::new("kill").args(["-CONT", rust_owner.trim()]).status()?.success());
    let started = Instant::now();
    loop {
        if let Some(status) = owner.try_wait()? {
            assert_eq!(status.code(), Some(37));
            break;
        }
        assert!(started.elapsed() < Duration::from_secs(5));
        thread::sleep(Duration::from_millis(5));
    }
    scratch.finish(Ok(()))
}

#[test]
fn nested_normal_completion_does_not_cancel_following_stage() -> Result<(), Refusal> {
    use umpire_backend_process::{DescendantPolicy, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let out = fs::File::create(scratch.path().join("out"))?;
    let err = fs::File::create(scratch.path().join("err"))?;
    let mut command = fixture("lifecycle-stages");
    command.arg(scratch.path());
    let mut child =
        OwnedProcess::spawn(&command, scratch.path(), &out, &err, DescendantPolicy::Orchestration)?;
    let started = Instant::now();
    loop {
        if let Some(status) = child.try_wait()? {
            assert!(status.success(), "{}", fs::read_to_string(scratch.path().join("err"))?);
            break;
        }
        assert!(started.elapsed() < Duration::from_secs(5));
        thread::sleep(Duration::from_millis(5));
    }
    assert_eq!(fs::read_to_string(scratch.path().join("stages"))?, "two stages finished");
    scratch.finish(Ok(()))
}

#[test]
fn lifecycle_locators_without_both_pipe_capabilities_refuse_before_launch() -> Result<(), Refusal> {
    let scratch = Scratch::new(&root()?)?;
    for (completion, cancellation) in [("10", "11"), ("1", "11"), ("10", "1")] {
        let ready = scratch.path().join("ready");
        let output = fixture("checker-owner")
            .arg(scratch.path())
            .arg(&ready)
            .arg(scratch.path().join("release"))
            .env("UMPIRE_COMPLETION_FD", completion)
            .env("UMPIRE_CANCELLATION_FD", cancellation)
            .output()?;
        assert!(!output.status.success(), "locator-only authority admitted");
        assert!(!ready.exists(), "invalid lifecycle reached checker target");
    }
    scratch.finish(Ok(()))
}

#[test]
fn cancellation_keeps_controller_alive_to_wait_its_separate_child_group() -> Result<(), Refusal> {
    use umpire_backend_process::{DescendantPolicy, OwnedProcess};
    let scratch = Scratch::new(&root()?)?;
    let ready = scratch.path().join("ready");
    let ack = scratch.path().join("waited");
    let out = fs::File::create(scratch.path().join("out"))?;
    let err = fs::File::create(scratch.path().join("err"))?;
    let mut command = Command::new("/bin/bash");
    command
        .args([
            "-c",
            r#"
        trap 'kill -TERM "$child"; wait "$child"; printf waited > "$3"; exit 143' INT
        "$1" group-sleep "$4" &
        child=$!
        while ! test -f "$4"; do sleep 0.01; done
        printf '%s %s %s' "$$" "$PPID" "$child" > "$2"
        wait "$child"
    "#,
            "controller",
        ])
        .arg(FIXTURE)
        .arg(&ready)
        .arg(&ack)
        .arg(scratch.path().join("group-ready"));
    let mut owner =
        OwnedProcess::spawn(&command, scratch.path(), &out, &err, DescendantPolicy::Orchestration)?;
    assert!(wait_ready(&ready));
    let pids = fs::read_to_string(&ready)?;
    let guardian = pids
        .split_whitespace()
        .nth(1)
        .ok_or_else(|| Refusal("missing controller guardian".into()))?;
    assert!(Command::new("kill").args(["-TERM", guardian]).status()?.success());
    let started = Instant::now();
    loop {
        if let Some(status) = owner.try_wait()? {
            assert_eq!(status.code(), Some(143));
            break;
        }
        assert!(
            started.elapsed() < Duration::from_secs(5),
            "controller did not acknowledge child wait"
        );
        thread::sleep(Duration::from_millis(5));
    }
    assert_eq!(fs::read_to_string(ack)?, "waited");
    #[cfg(target_os = "linux")]
    assert_reaped(&ready)?;
    scratch.finish(Ok(()))
}
