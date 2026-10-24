#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle process integration belongs to successor tasks"
    )
)]
pub(crate) mod busy;
pub(crate) mod cleanup;
mod environment;
mod procfs;
mod registry;
mod sys;
pub(crate) use environment::Environment;
use registry::Registry;
use serde::{Deserialize, Serialize};
use std::{
    ffi::CString,
    fs::File,
    io::{Read, Write},
    os::{
        fd::{AsRawFd, OwnedFd},
        unix::{ffi::OsStrExt, net::UnixStream},
    },
    path::{Path, PathBuf},
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    time::Duration,
};

#[derive(Clone, Debug)]
pub(crate) struct Command {
    pub argv: Vec<String>,
    pub cwd: PathBuf,
    pub environment: Environment,
    pub accepted: Vec<i32>,
    pub timeout: Duration,
    pub private_context: Option<String>,
}
#[derive(Clone, Debug, Serialize)]
pub(crate) struct Registration {
    pub name: String,
    pub interval: Option<OwnedInterval>,
    pub pid: u32,
    pub parent_pid: u32,
    pub stage_pid: u32,
    pub active: bool,
    pub argv: Vec<String>,
    pub cwd: PathBuf,
    pub environment: Environment,
    pub started_monotonic_nanos: u64,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct Resources {
    pub pid: u32,
    pub waited_pid: u32,
    pub wait_status: i32,
    pub reaps: u32,
    pub user_nanos: u64,
    pub system_nanos: u64,
    pub peak_rss_bytes: u64,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct Completed {
    pub pid: u32,
    pub parent_pid: u32,
    pub stage_pid: u32,
    pub exit: i32,
    pub signal: Option<i32>,
    pub resources: Resources,
    pub started_monotonic_nanos: u64,
    pub reaped_monotonic_nanos: u64,
    pub cleaned_monotonic_nanos: u64,
    pub cleanup_complete: bool,
    pub cleanup_error: Option<String>,
    pub descendants_reaped: u64,
    pub failure: Option<String>,
    pub accepted_exits: Vec<i32>,
}
impl Completed {
    pub fn accepted(&self) -> bool {
        self.failure.is_none() && self.cleanup_complete && self.accepted_exits.contains(&self.exit)
    }
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub(crate) enum SpawnOutcome {
    NoChild,
    Created { pid: u32, supervisor: u32 },
    Ambiguous { supervisor: u32 },
}
#[derive(Debug)]
pub(crate) struct StartFailure {
    pub outcome: SpawnOutcome,
    pub message: String,
}
impl std::fmt::Display for StartFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        self.message.fmt(f)
    }
}
impl std::error::Error for StartFailure {}
#[derive(Clone, Debug, Serialize)]
pub(crate) struct OwnedInterval {
    label: String,
    lane: usize,
    scratch: Vec<String>,
    pid: u32,
    stage_pid: u32,
    started_monotonic_nanos: u64,
}
impl OwnedInterval {
    pub fn pid(&self) -> u32 {
        self.pid
    }
    pub fn stage_pid(&self) -> u32 {
        self.stage_pid
    }
    pub fn started_monotonic_nanos(&self) -> u64 {
        self.started_monotonic_nanos
    }
    pub fn label(&self) -> &str {
        &self.label
    }
    pub fn lane(&self) -> usize {
        self.lane
    }
    pub fn scratch(&self) -> &[String] {
        &self.scratch
    }
}
pub(crate) struct Runner {
    registry: Arc<Registry>,
    context: Option<String>,
}
pub(crate) struct Running {
    registry: Arc<Registry>,
    registration: Registration,
    supervisor: i32,
    report: UnixStream,
    received: Vec<u8>,
    accepted: Vec<i32>,
    completed: Option<Completed>,
    cancelled: bool,
}
fn error(error: impl std::fmt::Display) -> String {
    error.to_string()
}
fn pid(value: i32) -> Result<u32, String> {
    u32::try_from(value)
        .ok()
        .filter(|pid| *pid > 0)
        .ok_or_else(|| "invalid process PID".into())
}
fn nanos(seconds: i64, micros: i64) -> Result<u64, String> {
    let seconds = u64::try_from(seconds).map_err(|_| "invalid child CPU seconds")?;
    let micros = u64::try_from(micros).map_err(|_| "invalid child CPU microseconds")?;
    if micros >= 1_000_000 {
        return Err("invalid child CPU microseconds".into());
    }
    seconds
        .checked_mul(1_000_000_000)
        .and_then(|n| n.checked_add(micros * 1000))
        .ok_or_else(|| "invalid child CPU usage overflow".into())
}
fn duplicate(file: &impl std::os::fd::AsFd) -> Result<OwnedFd, String> {
    rustix::io::fcntl_dupfd_cloexec(file, 10).map_err(error)
}
impl Runner {
    pub fn create(root: &Path, context: Option<String>) -> Result<Self, String> {
        Ok(Self {
            registry: Arc::new(Registry::create(root)?),
            context,
        })
    }
    pub fn start(
        &mut self,
        name: &str,
        command: Command,
        cancelled: &AtomicBool,
        admit: impl FnOnce(&Registration) -> Result<(), String>,
    ) -> Result<Running, StartFailure> {
        let mut outcome = SpawnOutcome::NoChild;
        self.start_inner(name, command, cancelled, admit, &mut outcome, None)
            .map_err(|message| StartFailure { outcome, message })
    }
    pub fn active_intervals(&self) -> Result<Vec<OwnedInterval>, String> {
        self.registry.check()?;
        if self.registry.degraded.load(Ordering::Acquire) {
            return Err("process registry terminal observation inconclusive".into());
        }
        Ok(self
            .registry
            .intervals
            .lock()
            .map_err(|_| "interval registry poisoned")?
            .values()
            .cloned()
            .collect())
    }
    pub fn start_stage(
        &mut self,
        stage: &crate::cycle::model::Stage,
        lane: usize,
        command: Command,
        cancelled: &AtomicBool,
        admit: impl FnOnce(&Registration) -> Result<(), String>,
    ) -> Result<Running, StartFailure> {
        let mut outcome = SpawnOutcome::NoChild;
        let interval = OwnedInterval {
            label: stage.contract.label.clone(),
            lane,
            scratch: stage.contract.scratch.clone(),
            pid: 0,
            stage_pid: std::process::id(),
            started_monotonic_nanos: 0,
        };
        self.start_inner(
            &stage.contract.label,
            command,
            cancelled,
            admit,
            &mut outcome,
            Some(interval),
        )
        .map_err(|message| StartFailure { outcome, message })
    }
    #[expect(
        clippy::too_many_lines,
        reason = "One gate transaction owns all rollback and child custody"
    )]
    fn start_inner(
        &mut self,
        name: &str,
        command: Command,
        cancelled: &AtomicBool,
        admit: impl FnOnce(&Registration) -> Result<(), String>,
        outcome: &mut SpawnOutcome,
        interval: Option<OwnedInterval>,
    ) -> Result<Running, String> {
        if self.registry.degraded.load(Ordering::Acquire) {
            return Err("process registry terminal observation inconclusive".into());
        }
        if !cfg!(all(
            target_os = "linux",
            target_pointer_width = "64",
            target_env = "gnu"
        )) {
            return Err("Linux 64-bit GNU per-child wait4 process custody unavailable".into());
        }
        if cancelled.load(Ordering::Acquire) {
            return Err("interrupted".into());
        }
        if name.is_empty()
            || !name
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b"-_".contains(&b))
        {
            return Err("invalid command registration name".into());
        }
        if command.argv.is_empty()
            || command.argv.iter().any(|a| a.contains('\0'))
            || command.argv[0].is_empty()
        {
            return Err("invalid argv".into());
        }
        if command.timeout.is_zero() || command.timeout > Duration::from_secs(3600) {
            return Err("invalid command timeout".into());
        }
        if command.private_context.is_some() && command.private_context != self.context {
            return Err("foreign private observation context".into());
        }
        command.environment.validate()?;
        if !command.cwd.is_absolute() {
            return Err("command cwd must be absolute".into());
        }
        if self.registry.exists(name)? {
            return Err("command process registration already exists".into());
        }
        let executable = if Path::new(&command.argv[0]).is_absolute() {
            PathBuf::from(&command.argv[0])
        } else if command.argv[0].contains('/') {
            command.cwd.join(&command.argv[0])
        } else {
            command
                .environment
                .values()
                .get("PATH")
                .ok_or("relative command requires explicit PATH")?
                .split(':')
                .filter(|part| Path::new(part).is_absolute())
                .map(|part| Path::new(part).join(&command.argv[0]))
                .find(|path| path.is_file())
                .ok_or("command executable not found")?
        };
        let executable = CString::new(executable.as_os_str().as_bytes()).map_err(error)?;
        let cwd = CString::new(command.cwd.as_os_str().as_bytes()).map_err(error)?;
        let arguments = command
            .argv
            .iter()
            .map(|a| CString::new(a.as_str()).map_err(error))
            .collect::<Result<Vec<_>, _>>()?;
        let environment = command
            .environment
            .values()
            .iter()
            .map(|(key, value)| CString::new(format!("{key}={value}")).map_err(error))
            .collect::<Result<Vec<_>, _>>()?;
        let argv: Vec<_> = arguments
            .iter()
            .map(|a| a.as_ptr())
            .chain(std::iter::once(std::ptr::null()))
            .collect();
        let env: Vec<_> = environment
            .iter()
            .map(|a| a.as_ptr())
            .chain(std::iter::once(std::ptr::null()))
            .collect();
        let stdout = self.registry.create_file(&format!("{name}.stdout"))?;
        let stderr = self.registry.create_file(&format!("{name}.stderr"))?;
        let stdin = File::open("/dev/null").map_err(error)?;
        let (mut gate, read_gate) = UnixStream::pair().map_err(error)?;
        let (mut report, write_report) = UnixStream::pair().map_err(error)?;
        report
            .set_read_timeout(Some(Duration::from_secs(5)))
            .map_err(error)?;
        let (read_exec, write_exec) = UnixStream::pair().map_err(error)?;
        let owned = [
            duplicate(&read_gate)?,
            duplicate(&write_report)?,
            duplicate(&stdout)?,
            duplicate(&stderr)?,
            duplicate(&stdin)?,
            duplicate(&read_exec)?,
            duplicate(&write_exec)?,
        ];
        let prepared = sys::Prepared {
            fds: owned.each_ref().map(AsRawFd::as_raw_fd),
            argv: &argv,
            env: &env,
            executable: executable.as_ptr(),
            cwd: cwd.as_ptr(),
            timeout_ns: i64::try_from(command.timeout.as_nanos()).map_err(error)?,
        };
        let mut leaf = [0u8; 16];
        let (supervisor, handshake) = crate::fork_order::spawning(|| {
            // SAFETY: child immediately enters the syscall-only path with every
            // string, descriptor and buffer prepared before fork. No unwinding.
            let supervisor = unsafe { sys::fork() };
            if supervisor == 0 {
                sys::supervise(&prepared);
            }
            // Close parent copies before waiting for EOF/error from supervisor.
            drop(owned);
            drop(write_report);
            drop(read_gate);
            drop(read_exec);
            drop(write_exec);
            let handshake = if supervisor > 0 {
                report.read_exact(&mut leaf).map_err(error)
            } else {
                Err(std::io::Error::last_os_error().to_string())
            };
            (supervisor, handshake)
        });
        if supervisor > 0 {
            *outcome = SpawnOutcome::Ambiguous {
                supervisor: pid(supervisor)?,
            };
        }
        if let Err(failure) = handshake {
            drop(gate);
            drop(report);
            if supervisor > 0 {
                sys::reap_supervisor(supervisor)?;
            }
            return Err(format!("supervisor spawn refused: {failure}"));
        }
        let child = pid(
            i32::try_from(i64::from_ne_bytes(leaf[..8].try_into().map_err(error)?))
                .map_err(error)?,
        )?;
        let started_monotonic_nanos =
            u64::try_from(i64::from_ne_bytes(leaf[8..].try_into().map_err(error)?))
                .map_err(error)?;
        *outcome = SpawnOutcome::Created {
            pid: child,
            supervisor: pid(supervisor)?,
        };
        let registration = Registration {
            name: name.into(),
            interval: interval.map(|mut interval| {
                interval.pid = child;
                interval.started_monotonic_nanos = started_monotonic_nanos;
                interval
            }),
            pid: child,
            parent_pid: pid(supervisor)?,
            stage_pid: std::process::id(),
            active: true,
            argv: command.argv,
            cwd: command.cwd,
            environment: command.environment,
            started_monotonic_nanos,
        };
        report.set_nonblocking(true).map_err(error)?;
        let mut running = Running {
            registry: Arc::clone(&self.registry),
            registration,
            supervisor,
            report,
            received: Vec::with_capacity(size_of::<sys::Report>()),
            accepted: command.accepted,
            completed: None,
            cancelled: false,
        };
        let admission = self
            .registry
            .register(&running.registration)
            .and_then(|()| admit(&running.registration))
            .and_then(|()| self.registry.verify(&running.registration))
            .and_then(|()| {
                if cancelled.load(Ordering::Acquire) {
                    Err("interrupted".into())
                } else {
                    Ok(())
                }
            });
        if let Err(failure) = admission {
            drop(gate);
            running.cancel();
            return match running.wait(&AtomicBool::new(false)) {
                Ok(completed) if completed.cleanup_complete => Err(failure),
                Ok(completed) => Err(format!(
                    "{failure}; cleanup inconclusive: {:?}",
                    completed.cleanup_error
                )),
                Err(cleanup) => Err(format!("{failure}; cleanup inconclusive: {cleanup}")),
            };
        }
        gate.write_all(b"1").map_err(error)?;
        drop(gate);
        Ok(running)
    }
}
impl Running {
    pub fn pid(&self) -> u32 {
        self.registration.pid
    }
    pub fn cancel(&mut self) {
        if !self.cancelled {
            let _ = self.report.write_all(b"x");
            self.cancelled = true;
        }
    }
    pub fn poll(&mut self, cancelled: &AtomicBool) -> Result<Option<Completed>, String> {
        if let Some(completed) = &self.completed {
            return Ok(Some(completed.clone()));
        }
        if cancelled.load(Ordering::Acquire) {
            self.cancel();
        }
        let mut buffer = [0u8; 256];
        while self.received.len() < size_of::<sys::Report>() {
            let length = (size_of::<sys::Report>() - self.received.len()).min(buffer.len());
            match self.report.read(&mut buffer[..length]) {
                Ok(0) => return Err("supervisor report incomplete; cleanup inconclusive".into()),
                Ok(count) => self.received.extend_from_slice(&buffer[..count]),
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => return Ok(None),
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => return Err(e.to_string()),
            }
        }
        let report = sys::decode(&self.received);
        sys::reap_supervisor(self.supervisor)?;
        self.supervisor = 0;
        let mut completed = self.observation(report, Path::new("/proc"))?;
        if (cancelled.load(Ordering::Acquire) || self.cancelled) && completed.failure.is_none() {
            completed.failure = Some("interrupted".into());
        }
        if let Err(publication) = self.registry.complete(&self.registration, &completed) {
            self.registry.degraded.store(true, Ordering::Release);
            if completed.failure.is_none() {
                completed.failure = Some(format!("terminal publication failed: {publication}"));
            }
            self.completed = Some(completed);
            return Err(publication);
        }
        self.completed = Some(completed.clone());
        Ok(Some(completed))
    }
    fn observation(&self, report: sys::Report, proc_root: &Path) -> Result<Completed, String> {
        if report.pid != i64::from(self.registration.pid)
            || report.waited_pid != report.pid
            || report.reaps != 1
            || u64::try_from(report.started).ok() != Some(self.registration.started_monotonic_nanos)
            || report.started < 0
            || report.reaped <= report.started
            || report.cleaned < report.reaped
        {
            return Err("resource measurement lost its exact child".into());
        }
        let status = i32::try_from(report.status).map_err(error)?;
        let signal = status & 0x7f;
        if signal > 64 || !(0..=65535).contains(&status) {
            return Err("invalid child terminal wait status".into());
        }
        let exit = if signal == 0 {
            (status >> 8) & 0xff
        } else {
            -signal
        };
        let failure = match report.reason {
            0 if report.exec_error != 0 => {
                Some(format!("spawn refused: errno {}", report.exec_error))
            }
            0 if report.descendants > 0 => {
                Some("producer exited with remaining descendants".into())
            }
            0 => None,
            1 => Some("timeout".into()),
            2 => Some("interrupted".into()),
            _ => Some("process observation inconclusive".into()),
        };
        let cleanup_error = if report.cleanup_error == 0 {
            procfs::quiescent(
                proc_root,
                self.registration.pid,
                self.registration.parent_pid,
            )
            .err()
        } else {
            Some(format!(
                "command process cleanup incomplete: supervisor code {}",
                report.cleanup_error
            ))
        };
        let completed = Completed {
            pid: self.registration.pid,
            parent_pid: self.registration.parent_pid,
            stage_pid: self.registration.stage_pid,
            exit,
            signal: (signal != 0).then_some(signal),
            resources: Resources {
                pid: self.registration.pid,
                waited_pid: u32::try_from(report.waited_pid).map_err(error)?,
                wait_status: status,
                reaps: 1,
                user_nanos: nanos(report.usage.user.seconds, report.usage.user.micros)?,
                system_nanos: nanos(report.usage.system.seconds, report.usage.system.micros)?,
                peak_rss_bytes: u64::try_from(report.usage.counters[0])
                    .map_err(error)?
                    .checked_mul(1024)
                    .ok_or("child RSS overflow")?,
            },
            started_monotonic_nanos: u64::try_from(report.started).map_err(error)?,
            reaped_monotonic_nanos: u64::try_from(report.reaped).map_err(error)?,
            cleaned_monotonic_nanos: u64::try_from(report.cleaned).map_err(error)?,
            cleanup_complete: cleanup_error.is_none(),
            cleanup_error,
            descendants_reaped: u64::try_from(report.descendants).map_err(error)?,
            failure,
            accepted_exits: self.accepted.clone(),
        };
        Ok(completed)
    }
    pub fn wait(mut self, cancelled: &AtomicBool) -> Result<Completed, String> {
        loop {
            if let Some(completed) = self.poll(cancelled)? {
                return Ok(completed);
            }
            std::thread::park_timeout(Duration::from_millis(5));
        }
    }
}
impl Drop for Running {
    fn drop(&mut self) {
        if self.supervisor > 0 {
            self.cancel();
            loop {
                match self.poll(&AtomicBool::new(false)) {
                    Ok(Some(_)) => break,
                    Ok(None) => std::thread::park_timeout(Duration::from_millis(5)),
                    Err(_) => {
                        let _ = sys::reap_supervisor(self.supervisor);
                        self.supervisor = 0;
                        break;
                    }
                }
            }
        }
    }
}
#[cfg(test)]
mod tests;
