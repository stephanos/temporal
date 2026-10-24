#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle process integration belongs to successor tasks"
    )
)]
pub(crate) mod busy;
pub(crate) mod cleanup;
mod observation;
pub(crate) use observation::RawStreams;
mod domain;
pub(crate) use domain::CycleOwner;
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
    pub declared_environment: std::collections::BTreeMap<String, String>,
    pub temporary_paths: std::collections::BTreeMap<String, String>,
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
    pub supervisor_pid: u32,
    pub supervisor_parent_pid: u32,
    pub start_ns: u64,
    pub clock: &'static str,
    pub argv: Vec<String>,
    pub cwd: PathBuf,
    pub environment: std::collections::BTreeMap<String, String>,
    pub temporary_paths: std::collections::BTreeMap<String, String>,
    pub effective_environment: Option<std::collections::BTreeMap<String, String>>,
    pub execution_environment: std::collections::BTreeMap<String, String>,
    pub environment_mode: &'static str,
    pub private_context: Option<String>,
    pub started_monotonic_nanos: u64,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct Resources {
    pub schema: String,
    pub measurement: String,
    pub scope: String,
    pub exit_code: i32,
    pub user_seconds: f64,
    pub system_seconds: f64,
    pub ru_maxrss: u64,
    pub ru_maxrss_unit: String,
    pub cpu_conversion: String,
    pub rss_conversion: String,
    pub pid: u32,
    pub waited_pid: u32,
    pub wait_status: i32,
    pub reaps: u32,
    pub wait4_calls: u32,
    pub user_nanos: u64,
    pub system_nanos: u64,
    pub peak_rss_bytes: u64,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct Completed {
    pub pid: u32,
    pub parent_pid: u32,
    pub stage_pid: u32,
    pub supervisor_pid: u32,
    pub supervisor_parent_pid: u32,
    pub start_ns: u64,
    pub end_ns: u64,
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
    domain: Arc<domain::Domain>,
    registry: Arc<Registry>,
    context: Option<String>,
}
type Capture<'a> = dyn FnMut(&Registration, &Completed, &RawStreams) -> Result<Option<serde_json::Value>, String>
    + 'a;
pub(crate) struct Running {
    domain: Arc<domain::Domain>,
    registry: Arc<Registry>,
    registration: Registration,
    supervisor: i32,
    report: UnixStream,
    received: Vec<u8>,
    accepted: Vec<i32>,
    completed: Option<Completed>,
    cancelled: bool,
    terminal_error: Option<String>,
    stdout: File,
    stderr: File,
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
    pub fn create(
        owner: &CycleOwner,
        root: &Path,
        context: Option<String>,
    ) -> Result<Self, String> {
        owner.domain.lock()?.check()?;
        Ok(Self {
            domain: Arc::clone(&owner.domain),
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
        self.domain.lock()?.check()?;
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
        self.domain.lock()?.check()?;
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
        let timeout_ns = i64::try_from(command.timeout.as_nanos())
            .map_err(|_| "command timeout cannot be represented in monotonic nanoseconds")?;
        if timeout_ns == 0 {
            return Err("invalid command timeout".into());
        }
        sys::validate_deadline(timeout_ns)?;
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
        let executable_paths = if Path::new(&command.argv[0]).is_absolute() {
            vec![PathBuf::from(&command.argv[0])]
        } else if command.argv[0].contains('/') {
            vec![command.cwd.join(&command.argv[0])]
        } else {
            command
                .environment
                .values()
                .get("PATH")
                .map_or("/bin:/usr/bin", String::as_str)
                .split(':')
                .map(|part| {
                    let path = Path::new(part);
                    if path.is_absolute() {
                        path.join(&command.argv[0])
                    } else {
                        command.cwd.join(path).join(&command.argv[0])
                    }
                })
                .collect()
        };
        let executable_paths = executable_paths
            .iter()
            .map(|path| CString::new(path.as_os_str().as_bytes()).map_err(error))
            .collect::<Result<Vec<_>, _>>()?;
        let executables: Vec<_> = executable_paths.iter().map(|path| path.as_ptr()).collect();
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
        let start_ns = observation::wall_nanos()?;
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
        let admission_mask = sys::AdmissionMask::block()?;
        let prepared = sys::Prepared {
            fds: owned.each_ref().map(AsRawFd::as_raw_fd),
            argv: &argv,
            env: &env,
            executables: &executables,
            cwd: cwd.as_ptr(),
            timeout_ns,
            mask: admission_mask.previous,
        };
        let mut domain = self.domain.lock()?;
        domain.check()?;
        let mut leaf = [0u8; 32];
        let (supervisor, handshake) = crate::fork_order::spawning(|| {
            let fork_mask = match sys::AdmissionMask::during_fork() {
                Ok(mask) => mask,
                Err(error) => return (-1, Err(error)),
            };
            // SAFETY: child immediately enters the syscall-only path with every
            // string, descriptor and buffer prepared before fork. No unwinding.
            let supervisor = unsafe { sys::fork() };
            if supervisor == 0 {
                sys::supervise(&prepared);
            }
            drop(fork_mask);
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
            domain.supervisors.insert(supervisor);
            *outcome = SpawnOutcome::Ambiguous {
                supervisor: pid(supervisor)?,
            };
        }
        if let Err(failure) = handshake {
            drop(gate);
            drop(report);
            if supervisor > 0 {
                return Err(self
                    .domain
                    .abort(&mut domain, &format!("supervisor spawn refused: {failure}")));
            }
            return Err(format!("supervisor spawn refused: {failure}"));
        }
        let identity = (|| -> Result<(u32, u64, u32), String> {
            let field = |offset| -> Result<i64, String> {
                Ok(i64::from_ne_bytes(
                    leaf[offset..offset + 8].try_into().map_err(error)?,
                ))
            };
            let child = pid(i32::try_from(field(0)?).map_err(error)?)?;
            let started = u64::try_from(field(8)?).map_err(error)?;
            if field(16)? != i64::from(supervisor) || field(24)? != i64::from(std::process::id()) {
                return Err("foreign supervisor parent identity".into());
            }
            report.set_nonblocking(true).map_err(error)?;
            Ok((child, started, u32::try_from(field(24)?).map_err(error)?))
        })();
        let (child, started_monotonic_nanos, supervisor_parent_pid) = match identity {
            Ok(identity) => identity,
            Err(failure) => {
                drop(gate);
                drop(report);
                return Err(self.domain.abort(&mut domain, &failure));
            }
        };
        *outcome = SpawnOutcome::Created {
            pid: child,
            supervisor: pid(supervisor)?,
        };
        drop(domain);
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
            supervisor_pid: pid(supervisor)?,
            supervisor_parent_pid,
            start_ns,
            clock: "CLOCK_MONOTONIC",
            argv: command.argv,
            cwd: command.cwd,
            effective_environment: match &command.environment {
                Environment::Exact(values) => Some(values.clone()),
                Environment::Clean(_) => None,
            },
            execution_environment: command.environment.values().clone(),
            environment_mode: match &command.environment {
                Environment::Exact(_) => "exact",
                Environment::Clean(_) => "inherited-clean",
            },
            environment: command.declared_environment,
            temporary_paths: command.temporary_paths,
            private_context: command.private_context,
            started_monotonic_nanos,
        };
        let mut running = Running {
            domain: Arc::clone(&self.domain),
            registry: Arc::clone(&self.registry),
            registration,
            supervisor,
            report,
            received: Vec::with_capacity(size_of::<sys::Report>()),
            accepted: command.accepted,
            completed: None,
            cancelled: false,
            terminal_error: None,
            stdout,
            stderr,
        };
        let admission = self
            .registry
            .register(&running.registration)
            .and_then(|()| admit(&running.registration))
            .and_then(|()| self.registry.verify(&running.registration))
            .and_then(|()| {
                if cancelled.load(Ordering::Acquire) || sys::AdmissionMask::interrupted()? {
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
        let domain = self.domain.lock()?;
        domain.check()?;
        gate.write_all(b"1").map_err(error)?;
        drop(domain);
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
        self.poll_observed(cancelled, None)
    }
    fn poll_observed(
        &mut self,
        cancelled: &AtomicBool,
        capture: Option<&mut Capture<'_>>,
    ) -> Result<Option<Completed>, String> {
        if let Some(error) = &self.terminal_error {
            return Err(error.clone());
        }
        let result = self.poll_inner(cancelled, capture);
        if let Err(error) = &result {
            self.registry.degraded.store(true, Ordering::Release);
            self.terminal_error = Some(error.clone());
        }
        result
    }
    fn poll_inner(
        &mut self,
        cancelled: &AtomicBool,
        capture: Option<&mut Capture<'_>>,
    ) -> Result<Option<Completed>, String> {
        let ownership = Arc::clone(&self.domain);
        let mut domain = ownership.lock()?;
        if let Err(reason) = domain.check() {
            self.supervisor = 0;
            return Err(reason);
        }
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
                Ok(0) => {
                    let failure = self.domain.abort(
                        &mut domain,
                        "supervisor report incomplete; cleanup inconclusive",
                    );
                    self.supervisor = 0;
                    return Err(failure);
                }
                Ok(count) => self.received.extend_from_slice(&buffer[..count]),
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => return Ok(None),
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => {
                    let failure = self.domain.abort(&mut domain, &e.to_string());
                    self.supervisor = 0;
                    return Err(failure);
                }
            }
        }
        let report = sys::decode(&self.received);
        if let Err(failure) = sys::reap_supervisor(self.supervisor) {
            let failure = self.domain.abort(&mut domain, &failure);
            self.supervisor = 0;
            return Err(failure);
        }
        domain.supervisors.remove(&self.supervisor);
        self.supervisor = 0;
        let completed = match self.observation(report, Path::new("/proc")) {
            Ok(completed) => completed,
            Err(failure) => return Err(self.domain.abort(&mut domain, &failure)),
        };
        if !completed.cleanup_complete {
            let failure = format!(
                "{}; cleanup inconclusive: {:?}",
                completed
                    .failure
                    .as_deref()
                    .unwrap_or("command process cleanup incomplete"),
                completed.cleanup_error
            );
            return Err(self.domain.abort(&mut domain, &failure));
        }
        drop(domain);
        self.complete_observation(completed, cancelled, capture)
    }
    fn complete_observation(
        &mut self,
        mut completed: Completed,
        cancelled: &AtomicBool,
        mut capture: Option<&mut Capture<'_>>,
    ) -> Result<Option<Completed>, String> {
        if (cancelled.load(Ordering::Acquire) || self.cancelled) && completed.failure.is_none() {
            completed.failure = Some("interrupted".into());
        }
        let streams = observation::bind(
            &self.registry,
            &self.registration,
            &self.stdout,
            &self.stderr,
        )
        .inspect_err(|_| self.registry.degraded.store(true, Ordering::Release))?;
        let mut capture_error = None;
        let private = if completed.accepted()
            && let Some(capture) = capture.as_mut()
        {
            if self.registration.private_context.is_none() {
                capture_error = Some("private capture requires bound observation context".into());
                None
            } else {
                match capture(&self.registration, &completed, &streams) {
                    Ok(private) => private,
                    Err(error) => {
                        capture_error = Some(error);
                        None
                    }
                }
            }
        } else {
            None
        };
        if let Some(error) = &capture_error {
            completed.failure = Some(format!("private capture refused: {error}"));
        }
        if capture.is_some()
            && observation::bind(
                &self.registry,
                &self.registration,
                &self.stdout,
                &self.stderr,
            )
            .inspect_err(|_| self.registry.degraded.store(true, Ordering::Release))?
                != streams
        {
            self.registry.degraded.store(true, Ordering::Release);
            return Err("command streams changed during capture".into());
        }
        completed.end_ns = observation::wall_nanos()?;
        if completed.end_ns < completed.start_ns {
            return Err("command wall clock moved before its start".into());
        }
        let domain = self.domain.lock()?;
        domain.check()?;
        if let Err(publication) =
            self.registry
                .complete(&self.registration, &completed, &streams, private.as_ref())
        {
            self.registry.degraded.store(true, Ordering::Release);
            if completed.failure.is_none() {
                completed.failure = Some(format!("terminal publication failed: {publication}"));
            }
            self.completed = Some(completed);
            return Err(publication);
        }
        self.completed = Some(completed.clone());
        if let Some(error) = capture_error {
            return Err(error);
        }
        Ok(Some(completed))
    }
    fn observation(&self, report: sys::Report, proc_root: &Path) -> Result<Completed, String> {
        if report.pid != i64::from(self.registration.pid)
            || report.supervisor_pid != i64::from(self.registration.supervisor_pid)
            || report.supervisor_parent_pid != i64::from(self.registration.stage_pid)
            || report.waited_pid != report.pid
            || report.reaps != 1
            || report.leaf_wait4_calls != 1
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
            supervisor_pid: self.registration.supervisor_pid,
            supervisor_parent_pid: self.registration.supervisor_parent_pid,
            start_ns: self.registration.start_ns,
            end_ns: observation::wall_nanos()?,
            exit,
            signal: (signal != 0).then_some(signal),
            resources: Resources {
                schema: "gomad.driver-child-resources/v1".into(),
                measurement: "wait4".into(),
                scope: "whole-command-child".into(),
                exit_code: exit,
                user_seconds: observation::seconds(
                    report.usage.user.seconds,
                    report.usage.user.micros,
                )?,
                system_seconds: observation::seconds(
                    report.usage.system.seconds,
                    report.usage.system.micros,
                )?,
                ru_maxrss: u64::try_from(report.usage.counters[0]).map_err(error)?,
                ru_maxrss_unit: "KiB".into(),
                cpu_conversion: "round(seconds*1e9)".into(),
                rss_conversion: "ru_maxrss*1024".into(),
                pid: self.registration.pid,
                waited_pid: u32::try_from(report.waited_pid).map_err(error)?,
                wait_status: status,
                reaps: 1,
                wait4_calls: u32::try_from(report.leaf_wait4_calls).map_err(error)?,
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
        if completed.end_ns < completed.start_ns {
            return Err("command wall clock moved before its start".into());
        }
        Ok(completed)
    }
    pub fn wait_observed(
        mut self,
        cancelled: &AtomicBool,
        mut capture: impl FnMut(
            &Registration,
            &Completed,
            &RawStreams,
        ) -> Result<Option<serde_json::Value>, String>,
    ) -> Result<Completed, String> {
        loop {
            if let Some(completed) = self.poll_observed(cancelled, Some(&mut capture))? {
                return Ok(completed);
            }
            std::thread::park_timeout(Duration::from_millis(5));
        }
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
                        if let Ok(mut domain) = self.domain.lock() {
                            self.domain
                                .abort(&mut domain, "command drop after inconclusive observation");
                        }
                        self.supervisor = 0;
                        break;
                    }
                }
            }
        }
    }
}
#[cfg(test)]
pub(crate) mod tests;

#[cfg(test)]
mod fifo_tests;
