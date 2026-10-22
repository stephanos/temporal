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

pub(crate) fn run(options: &ExploreOptions, prepared: &Prepared) -> Result<Outcome, String> {
    if origin::from_preparation(prepared)?.is_none() {
        return Err("launcher requires intercepted origin".into());
    }
    let mut journal = super::seed_journal_policy(options, prepared, true)?;
    let selection = SeedSelection::parse(&options.seeds)?;
    let deadline = Instant::now()
        .checked_add(options.overall_timeout)
        .ok_or("launcher deadline exceeds clock range")?;
    let capacity = super::artifact_capacity(journal.plan(), prepared)?;
    let mut result = Outcome {
        status: 0,
        campaign: journal.path().into(),
        attempted: 0,
        unexecuted: selection.count(),
        failure: None,
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
        let output = if options.json {
            None
        } else {
            Some(Arc::new(
                Forwarder::new(cancellation.clone(), Instant::now() + timeout)
                    .map_err(|e| e.to_string())?,
            ) as Arc<dyn linked::OutputConsumer>)
        };
        let mut observed =
            execution::run_with_output(prepared, &job, timeout, &cancellation, None, None, output)?;
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

pub(crate) fn report(
    outcome: &Outcome,
    json_output: bool,
    installation: &Path,
    output: &mut dyn Write,
    diagnostics: &mut dyn Write,
) -> Result<(), String> {
    if json_output {
        writeln!(output, "{}", String::from_utf8(canonical::encode(&json!({"type":"result","origin":"intercepted-go","campaign_path":outcome.campaign,"status":outcome.status,"attempted":outcome.attempted,"unexecuted":outcome.unexecuted}))?).map_err(|e| e.to_string())?).map_err(|e| e.to_string())?;
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
