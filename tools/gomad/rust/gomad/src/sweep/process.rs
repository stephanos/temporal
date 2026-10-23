#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn deadline_admission_includes_preparation_and_retry() {
        let start = Instant::now();
        let context = Context::new(start + Duration::from_secs(100));
        assert!(context.fits_at(start, Duration::from_secs(100)));
        assert!(!context.fits_at(start + Duration::from_secs(1), Duration::from_secs(100)));
        assert!(!context.fits_at(start, Duration::from_secs(101)));
    }
    #[test]
    fn output_capture_has_an_explicit_bound() {
        let mut buffer = Vec::new();
        assert!(capture(&mut &b"hello"[..], &mut buffer, 4).is_err());
        assert_eq!(buffer, b"hell");
    }
    #[test]
    fn cancellation_is_distinct_from_a_guest_exit() {
        let context = Context::new(Instant::now() + Duration::from_secs(1));
        context.cancelled.store(true, Ordering::Release);
        let completed = Host.exec(
            &context,
            &Command {
                path: "/bin/sh".into(),
                args: vec!["-c".into(), "exit 0".into()],
                dir: "/".into(),
                env: BTreeMap::new(),
                timeout: Duration::from_secs(1),
            },
        );
        assert_eq!(completed.exit, -1);
        assert_eq!(completed.error.as_deref(), Some("sweep cancelled"));
    }
}

use std::{
    collections::BTreeMap,
    io::{self, Read},
    path::PathBuf,
    process::{Command as HostCommand, Stdio},
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, Instant},
};
#[derive(Clone)]
pub struct Context {
    pub deadline: Instant,
    pub cancelled: Arc<AtomicBool>,
    pub now: Arc<dyn Fn() -> Instant + Send + Sync>,
}
impl Context {
    pub fn new(deadline: Instant) -> Self {
        Self {
            deadline,
            cancelled: Arc::new(AtomicBool::new(false)),
            now: Arc::new(Instant::now),
        }
    }
    pub fn check(&self) -> Result<(), String> {
        if self.cancelled.load(Ordering::Acquire) || SIGNALLED.load(Ordering::Acquire) {
            Err("sweep cancelled".into())
        } else if (self.now)() >= self.deadline {
            Err("sweep overall budget exhausted".into())
        } else {
            Ok(())
        }
    }
    pub fn fits_at(&self, now: Instant, bound: Duration) -> bool {
        now.checked_add(bound)
            .is_some_and(|end| end <= self.deadline)
    }
    pub fn fits(&self, bound: Duration) -> bool {
        self.check().is_ok() && self.fits_at((self.now)(), bound)
    }
    pub fn interrupted(&self) -> bool {
        self.cancelled.load(Ordering::Acquire) || SIGNALLED.load(Ordering::Acquire)
    }
}
pub struct Command {
    pub path: PathBuf,
    pub args: Vec<String>,
    pub dir: PathBuf,
    pub env: BTreeMap<String, String>,
    pub timeout: Duration,
}
#[derive(Default)]
pub struct Completed {
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    pub exit: i32,
    pub error: Option<String>,
    pub elapsed: Duration,
}
pub trait Executor {
    fn exec(&mut self, context: &Context, command: &Command) -> Completed;
}
pub struct Host;
const CAPTURE: u64 = 64 << 20;
fn capture(reader: &mut impl Read, buffer: &mut Vec<u8>, limit: u64) -> Result<(), String> {
    let mut chunk = [0; 16 << 10];
    loop {
        match reader.read(&mut chunk) {
            Ok(0) => return Ok(()),
            Ok(count) => {
                let available = usize::try_from(limit)
                    .map_err(|_| "capture limit exceeds address space")?
                    .saturating_sub(buffer.len());
                buffer.extend_from_slice(&chunk[..count.min(available)]);
                if count > available {
                    return Err("subprocess output exceeded the capture bound".into());
                }
            }
            Err(error) if error.kind() == io::ErrorKind::WouldBlock => return Ok(()),
            Err(error) if error.kind() == io::ErrorKind::Interrupted => {}
            Err(error) => return Err(error.to_string()),
        }
    }
}
struct Child(std::process::Child);
impl Child {
    fn kill_group(&mut self) {
        if let Ok(raw) = i32::try_from(self.0.id())
            && let Some(pid) = rustix::process::Pid::from_raw(raw)
        {
            let _ = rustix::process::kill_process_group(pid, rustix::process::Signal::KILL);
        }
    }
}
impl Drop for Child {
    fn drop(&mut self) {
        self.kill_group();
        let _ = self.0.kill();
        let _ = self.0.wait();
    }
}
impl Executor for Host {
    fn exec(&mut self, context: &Context, command: &Command) -> Completed {
        use std::os::unix::process::CommandExt as _;
        let start = Instant::now();
        let mut output = Completed {
            exit: -1,
            ..Completed::default()
        };
        let execute = || -> Result<(Vec<u8>, Vec<u8>, i32), String> {
            context.check()?;
            if command.timeout.is_zero() {
                return Err("subprocess timeout is required".into());
            }
            let deadline = context.deadline.min(start + command.timeout);
            let child = HostCommand::new(&command.path)
                .args(&command.args)
                .current_dir(&command.dir)
                .env_clear()
                .envs(&command.env)
                .stdin(Stdio::null())
                .stdout(Stdio::piped())
                .stderr(Stdio::piped())
                .process_group(0)
                .spawn()
                .map_err(|e| e.to_string())?;
            let mut child = Child(child);
            let mut stdout = child.0.stdout.take().ok_or("subprocess stdout missing")?;
            let mut stderr = child.0.stderr.take().ok_or("subprocess stderr missing")?;
            for fd in [
                &stdout as &dyn std::os::fd::AsFd,
                &stderr as &dyn std::os::fd::AsFd,
            ] {
                rustix::fs::fcntl_setfl(fd, rustix::fs::OFlags::NONBLOCK)
                    .map_err(|e| e.to_string())?;
            }
            let mut out = Vec::new();
            let mut err = Vec::new();
            loop {
                context.check()?;
                if Instant::now() >= deadline {
                    return Err("subprocess timeout".into());
                }
                capture(&mut stdout, &mut out, CAPTURE)?;
                capture(&mut stderr, &mut err, CAPTURE)?;
                if let Some(status) = child.0.try_wait().map_err(|e| e.to_string())? {
                    child.kill_group();
                    capture(&mut stdout, &mut out, CAPTURE)?;
                    capture(&mut stderr, &mut err, CAPTURE)?;
                    return Ok((out, err, status.code().unwrap_or(-1)));
                }
                std::thread::park_timeout(
                    Duration::from_millis(10)
                        .min(deadline.saturating_duration_since(Instant::now())),
                );
            }
        };
        match execute() {
            Ok((stdout, stderr, exit)) => {
                output.stdout = stdout;
                output.stderr = stderr;
                output.exit = exit;
            }
            Err(error) => output.error = Some(error),
        }
        output.elapsed = start.elapsed();
        output
    }
}
static SIGNALLED: AtomicBool = AtomicBool::new(false);
extern "C" fn interrupted(_: i32) {
    SIGNALLED.store(true, Ordering::Release);
}
unsafe extern "C" {
    fn signal(number: i32, handler: usize) -> usize;
}
pub struct Signals {
    previous: [usize; 2],
}
impl Signals {
    pub fn install() -> Result<Self, String> {
        let mut previous = [0; 2];
        for (index, number) in [2, 15].into_iter().enumerate() {
            // SAFETY: The handler only writes a lock-free atomic and has C signal ABI.
            previous[index] = unsafe { signal(number, interrupted as *const () as usize) };
            if previous[index] == usize::MAX {
                return Err("cannot install sweep cancellation handler".into());
            }
        }
        Ok(Self { previous })
    }
}
impl Drop for Signals {
    fn drop(&mut self) {
        for (number, handler) in [2, 15].into_iter().zip(self.previous) {
            // SAFETY: Restore the exact prior C handlers after supervised children are reaped.
            unsafe { signal(number, handler) };
        }
    }
}
