use super::{
    execution,
    ledger::{Event, Job},
    origin,
    prepare::Prepared,
    seeds::SeedSelection,
};
use crate::{cli::explore::ExploreOptions, evidence::canonical, replay::preflight};
use gomad_wasmhost::linked;
use serde_json::json;
use std::{
    io::{self, Write},
    path::{Path, PathBuf},
    sync::{Arc, Mutex, mpsc},
    thread,
    time::{Duration, Instant},
};

pub(crate) struct Outcome {
    pub status: i32,
    pub campaign: PathBuf,
    pub attempted: u64,
    pub unexecuted: u64,
    pub failure: Option<(String, Option<PathBuf>)>,
    execution_guard: Option<crate::backstop::Guard>,
    rows: Vec<serde_json::Value>,
}

struct Chunk {
    stream: linked::OutputStream,
    bytes: Vec<u8>,
    done: mpsc::SyncSender<io::Result<()>>,
}

struct Forwarder {
    sender: Mutex<Option<mpsc::SyncSender<Chunk>>>,
    writer: Mutex<Option<thread::JoinHandle<()>>>,
    cancellation: linked::Cancellation,
    deadline: Instant,
    failed: std::sync::atomic::AtomicBool,
}

impl Forwarder {
    fn new(cancellation: linked::Cancellation, deadline: Instant) -> io::Result<Self> {
        let (sender, receiver) = mpsc::sync_channel::<Chunk>(0);
        let writer = thread::Builder::new()
            .name("gomad-output".into())
            .spawn(move || {
                while let Ok(chunk) = receiver.recv() {
                    let result = match chunk.stream {
                        linked::OutputStream::Stdout => {
                            let mut output = io::stdout().lock();
                            output.write_all(&chunk.bytes).and_then(|()| output.flush())
                        }
                        linked::OutputStream::Stderr => {
                            let mut output = io::stderr().lock();
                            output.write_all(&chunk.bytes).and_then(|()| output.flush())
                        }
                    };
                    let failed = result.is_err();
                    if chunk.done.send(result).is_err() || failed {
                        break;
                    }
                }
            })?;
        Ok(Self {
            sender: Mutex::new(Some(sender)),
            writer: Mutex::new(Some(writer)),
            cancellation,
            deadline,
            failed: std::sync::atomic::AtomicBool::new(false),
        })
    }
    fn remaining(&self) -> io::Result<Duration> {
        if self.cancellation.cancelled() {
            return Err(io::Error::new(
                io::ErrorKind::Interrupted,
                "output forwarding cancelled",
            ));
        }
        self.deadline
            .checked_duration_since(Instant::now())
            .filter(|d| !d.is_zero())
            .ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::TimedOut,
                    "output forwarding deadline exceeded",
                )
            })
    }
}

impl linked::OutputConsumer for Forwarder {
    fn write(&self, stream: linked::OutputStream, bytes: &[u8]) -> io::Result<()> {
        let result = (|| {
            for bytes in bytes.chunks(64 << 10) {
                let (done, completed) = mpsc::sync_channel(0);
                let mut chunk = Chunk {
                    stream,
                    bytes: bytes.to_vec(),
                    done,
                };
                let sender = self
                    .sender
                    .lock()
                    .map_err(|_| io::Error::other("output handoff lock poisoned"))?;
                let sender = sender
                    .as_ref()
                    .ok_or_else(|| io::Error::other("output forwarding finished"))?;
                loop {
                    let remaining = self.remaining()?;
                    match sender.try_send(chunk) {
                        Ok(()) => break,
                        Err(mpsc::TrySendError::Full(pending)) => {
                            chunk = pending;
                            thread::park_timeout(remaining.min(Duration::from_millis(5)));
                        }
                        Err(mpsc::TrySendError::Disconnected(_)) => {
                            return Err(io::Error::new(
                                io::ErrorKind::BrokenPipe,
                                "output writer disconnected",
                            ));
                        }
                    }
                }
                loop {
                    let remaining = self.remaining()?;
                    match completed.recv_timeout(remaining.min(Duration::from_millis(10))) {
                        Ok(result) => {
                            result?;
                            break;
                        }
                        Err(mpsc::RecvTimeoutError::Timeout) => {}
                        Err(mpsc::RecvTimeoutError::Disconnected) => {
                            return Err(io::Error::new(
                                io::ErrorKind::BrokenPipe,
                                "output writer disconnected",
                            ));
                        }
                    }
                }
            }
            Ok(())
        })();
        if result.is_err() {
            self.failed
                .store(true, std::sync::atomic::Ordering::Release);
        }
        result
    }
}

impl Drop for Forwarder {
    fn drop(&mut self) {
        if let Ok(sender) = self.sender.get_mut() {
            sender.take();
        }
        if let Ok(writer) = self.writer.get_mut()
            && let Some(writer) = writer.take()
        {
            // A failed handoff can leave inherited I/O blocked. Its bounded owner
            // returns the infrastructure result; process exit ends that writer.
            if !self.failed.load(std::sync::atomic::Ordering::Acquire) || writer.is_finished() {
                let _ = writer.join();
            }
        }
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "Launcher execution preserves scoped output ownership, cancellation and final typed failure reporting in one lifetime"
)]
pub(crate) fn run(
    options: &ExploreOptions,
    prepared: &Prepared,
    deadline: Instant,
) -> Result<Outcome, String> {
    if origin::from_preparation(prepared)?.is_none() {
        return Err("launcher requires intercepted origin".into());
    }
    let mut journal = super::seed_journal_policy(options, prepared, true)?;
    let selection = SeedSelection::parse(&options.seeds)?;
    let capacity = super::artifact_capacity(journal.plan(), prepared)?;
    let mut result = Outcome {
        status: 0,
        campaign: journal.path().into(),
        attempted: 0,
        unexecuted: selection.count(),
        failure: None,
        execution_guard: None,
        rows: Vec::new(),
    };
    for ordinal in 0..selection.count() {
        let job = Job {
            ordinal,
            seed: selection
                .seed_at(ordinal)
                .ok_or("launcher selection differs")?,
        };
        journal.append(Event::Admit {
            jobs: vec![job.clone()],
        })?;
        let cancellation = linked::Cancellation::default();
        let remaining = deadline
            .checked_duration_since(Instant::now())
            .ok_or("overall_timeout: launcher deadline exceeded")?;
        let timeout = remaining.min(options.execution_timeout);
        let execution_deadline = Instant::now() + timeout;
        result.execution_guard = Some(crate::backstop::arm_until(execution_deadline));
        let output = if options.json {
            None
        } else {
            Some(Arc::new(
                Forwarder::new(cancellation.clone(), execution_deadline)
                    .map_err(|e| e.to_string())?,
            ) as Arc<dyn linked::OutputConsumer>)
        };
        let mut observed = match execution::run_with_output(
            prepared,
            &job,
            timeout,
            &cancellation,
            None,
            None,
            output,
        )? {
            execution::Completion::Finalized(observed) => observed,
            execution::Completion::Failed(failure) => {
                cancellation.cancel();
                result.status = 3;
                result.attempted += 1;
                result.unexecuted -= 1;
                result.failure = Some((failure.message(), None));
                break;
            }
        };
        let published = super::store(journal.path().join("evidence"), capacity)
            .publish(observed.publication)?;
        observed.execution.artifact = Some(
            published
                .path
                .strip_prefix(journal.path())
                .map_err(|e| e.to_string())?
                .to_string_lossy()
                .into_owned(),
        );
        observed.execution.stored_bytes = published.stored_bytes;
        let status = status(&observed.execution.record.outcome);
        let stopping = (status != 0).then(|| "first_failure".to_owned());
        journal.append(Event::Receipt {
            execution: observed.execution.clone(),
        })?;
        journal.append(Event::Outcome {
            executions: vec![observed.execution.clone()],
            admitted: Vec::new(),
            stop_reason: stopping.clone(),
        })?;
        result
            .rows
            .push(outcome_row(&observed.execution, &published)?);
        result.attempted += 1;
        result.unexecuted -= 1;
        if status != 0 {
            result.status = status;
            let replayable = preflight::open(&published.path)
                .is_ok_and(|prepared| prepared.exact && prepared.available);
            result.failure = Some((
                format!(
                    "{}: {}",
                    observed.execution.record.outcome.termination,
                    observed.execution.record.outcome.reason
                ),
                replayable.then_some(published.path),
            ));
            break;
        }
    }
    journal.append(Event::Complete {
        stop_reason: if result.status == 0 {
            "seeds_exhausted"
        } else {
            "first_failure"
        }
        .into(),
    })?;
    Ok(result)
}

pub(crate) fn outcome_row(
    execution: &super::ledger::Execution,
    published: &crate::evidence::store::Published,
) -> Result<serde_json::Value, String> {
    let record = &execution.record;
    if record.seed != execution.seed
        || canonical::digest(&record.bytes()?) != published.record_sha256
    {
        return Err("launcher outcome record differs from publication".into());
    }
    let origin = record
        .streams
        .get(origin::FILE)
        .ok_or("launcher outcome has no bound origin")?;
    Ok(json!({
        "type":"outcome", "origin":"intercepted-go", "ordinal":execution.ordinal,
        "seed":execution.seed, "status":status(&record.outcome), "class":record.outcome.class,
        "termination":record.outcome.termination, "exit_code":record.outcome.exit_code,
        "reason":record.outcome.reason, "artifact_path":published.path,
        "record_sha256":published.record_sha256, "execution_identity":record.execution_identity,
        "module_sha256":record.identity.target, "origin_sha256":origin.sha256,
        "stored_bytes":published.stored_bytes
    }))
}

pub(crate) fn status(outcome: &crate::evidence::record::Outcome) -> i32 {
    match outcome.class.as_str() {
        "success" => 0,
        "target_failure" if outcome.termination == "exit" => outcome
            .exit_code
            .and_then(|code| i32::try_from(code).ok())
            .filter(|code| (1..=255).contains(code))
            .unwrap_or(1),
        "target_failure" => 1,
        _ => crate::backstop::STATUS,
    }
}

pub(crate) fn report_process(
    outcome: &Outcome,
    json_output: bool,
    installation: &Path,
    deadline: Instant,
) -> Result<(), String> {
    let mut output = Vec::new();
    let mut diagnostics = Vec::new();
    report(
        outcome,
        json_output,
        installation,
        &mut output,
        &mut diagnostics,
    )?;
    let output = deliver(linked::OutputStream::Stdout, output, deadline);
    let diagnostics = deliver(linked::OutputStream::Stderr, diagnostics, deadline);
    output.and(diagnostics)
}

pub(crate) fn refusal_process(message: &str, deadline: Instant) -> Result<(), String> {
    deliver(
        linked::OutputStream::Stderr,
        format!(
            "gomad run-guest: {message}\ngomad run-guest: no replayable artifact was retained\n"
        )
        .into_bytes(),
        deadline,
    )
}

fn deliver(stream: linked::OutputStream, bytes: Vec<u8>, deadline: Instant) -> Result<(), String> {
    use std::os::fd::AsFd as _;
    if bytes.is_empty() {
        return Ok(());
    }
    let remaining = deadline
        .checked_add(crate::backstop::CLEANUP_GRACE)
        .and_then(|deadline| deadline.checked_duration_since(Instant::now()))
        .filter(|remaining| !remaining.is_zero())
        .ok_or("final output deadline exceeded")?
        .min(Duration::from_millis(500));
    let (completed, receive) = mpsc::sync_channel(1);
    thread::Builder::new()
        .name("gomad-final-output".into())
        .spawn(move || {
            let write = || -> Result<(), String> {
                let mut pending = bytes.as_slice();
                while !pending.is_empty() {
                    let count = match stream {
                        linked::OutputStream::Stdout => {
                            rustix::io::write(io::stdout().as_fd(), pending)
                        }
                        linked::OutputStream::Stderr => {
                            rustix::io::write(io::stderr().as_fd(), pending)
                        }
                    };
                    match count {
                        Ok(0) => return Err("final output made no progress".into()),
                        Ok(count) => pending = &pending[count..],
                        Err(rustix::io::Errno::INTR) => (),
                        Err(error) => return Err(error.to_string()),
                    }
                }
                Ok(())
            };
            let _ = completed.send(write());
        })
        .map_err(|error| error.to_string())?;
    receive
        .recv_timeout(remaining)
        .map_err(|_| "final output delivery deadline exceeded".to_owned())?
}

pub(crate) fn report(
    outcome: &Outcome,
    json_output: bool,
    installation: &Path,
    output: &mut dyn Write,
    diagnostics: &mut dyn Write,
) -> Result<(), String> {
    if json_output {
        for row in &outcome.rows {
            json_line(output, row)?;
        }
        json_line(
            output,
            &json!({"type":"result","origin":"intercepted-go","campaign_path":outcome.campaign,"status":outcome.status,"attempted":outcome.attempted,"unexecuted":outcome.unexecuted}),
        )?;
    }
    if let Some((message, replay)) = &outcome.failure {
        writeln!(diagnostics, "gomad run-guest: {message}").map_err(|e| e.to_string())?;
        if let Some(path) = replay {
            let executable = std::env::current_exe().map_err(|e| e.to_string())?;
            writeln!(
                diagnostics,
                "{} --installation {} replay {}",
                quote(executable.as_os_str()),
                quote(installation.as_os_str()),
                quote(path.as_os_str())
            )
            .map_err(|e| e.to_string())?;
        } else {
            writeln!(
                diagnostics,
                "gomad run-guest: no replayable artifact was retained"
            )
            .map_err(|e| e.to_string())?;
        }
    }
    Ok(())
}

fn json_line(output: &mut dyn Write, value: &serde_json::Value) -> Result<(), String> {
    output
        .write_all(&canonical::encode(value)?)
        .and_then(|()| output.write_all(b"\n"))
        .map_err(|error| error.to_string())
}

fn quote(value: &std::ffi::OsStr) -> String {
    use std::os::unix::ffi::OsStrExt as _;
    let bytes = value.as_bytes();
    if bytes
        .iter()
        .all(|byte| byte.is_ascii_graphic() || *byte == b' ')
    {
        return format!("'{}'", value.to_string_lossy().replace('\'', "'\\''"));
    }
    let mut result = String::from("$'");
    for byte in bytes {
        result.push('\\');
        for shift in [6, 3, 0] {
            result.push(char::from(b'0' + ((byte >> shift) & 7)));
        }
    }
    result.push('\'');
    result
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::evidence::record::Outcome;
    #[test]
    fn undrained_stderr_and_final_report_finish_within_the_bound() {
        use std::{
            io::Read as _,
            process::{Command, Stdio},
        };
        let mut child = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "campaign::launcher::tests::blocked_stderr_child",
                "--nocapture",
            ])
            .env("GOMAD_TEST_BLOCKED_STDERR", "1")
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .unwrap();
        let deadline = Instant::now() + Duration::from_secs(2);
        let status = loop {
            if let Some(status) = child.try_wait().unwrap() {
                break status;
            }
            if Instant::now() >= deadline {
                child.kill().unwrap();
                child.wait().unwrap();
                panic!("final reporting hung after the output callback deadline");
            }
            thread::sleep(Duration::from_millis(10));
        };
        assert_eq!(status.code(), Some(3));
        let mut output = Vec::new();
        child
            .stdout
            .take()
            .unwrap()
            .read_to_end(&mut output)
            .unwrap();
        assert!(String::from_utf8_lossy(&output).contains("callback deadline observed"));
        let mut stderr = Vec::new();
        child
            .stderr
            .take()
            .unwrap()
            .read_to_end(&mut stderr)
            .unwrap();
        assert!(stderr.len() >= 4096, "stderr pipe was not filled");
        assert!(stderr.iter().all(|byte| *byte == b'X'));
    }

    #[test]
    fn blocked_stderr_child() {
        use linked::OutputConsumer as _;
        if std::env::var_os("GOMAD_TEST_BLOCKED_STDERR").is_none() {
            return;
        }
        let deadline = Instant::now() + Duration::from_millis(100);
        let forwarder = Forwarder::new(linked::Cancellation::default(), deadline).unwrap();
        let error = forwarder
            .write(linked::OutputStream::Stderr, &vec![b'X'; 2 << 20])
            .unwrap_err();
        assert_eq!(error.kind(), io::ErrorKind::TimedOut);
        drop(forwarder);
        println!("callback deadline observed");
        let outcome = super::Outcome {
            status: 3,
            campaign: "unused".into(),
            attempted: 1,
            unexecuted: 0,
            failure: Some(("output forwarding deadline exceeded".into(), None)),
            execution_guard: None,
            rows: Vec::new(),
        };
        let _ = report_process(&outcome, false, Path::new("installation"), deadline);
        std::process::exit(3);
    }

    #[test]
    fn guest_failures_and_infrastructure_preserve_separate_statuses() {
        let outcome = |class: &str, termination: &str, exit_code| Outcome {
            class: class.into(),
            termination: termination.into(),
            exit_code,
            reason: String::new(),
        };
        assert_eq!(status(&outcome("success", "exit", Some(0))), 0);
        assert_eq!(status(&outcome("target_failure", "exit", Some(7))), 7);
        assert_eq!(status(&outcome("target_failure", "trap", None)), 1);
        for termination in ["capacity", "fuel", "infrastructure", "watchdog_timeout"] {
            assert_eq!(status(&outcome("runner_failure", termination, None)), 3);
        }
    }
    #[test]
    fn exact_replay_quoting_preserves_spaces_quotes_and_unix_bytes() {
        use std::os::unix::ffi::OsStrExt as _;
        assert_eq!(quote(std::ffi::OsStr::new("a b'c")), "'a b'\\''c'");
        assert_eq!(
            quote(std::ffi::OsStr::from_bytes(b"a\n\xff")),
            "$'\\141\\012\\377'"
        );
    }
}
