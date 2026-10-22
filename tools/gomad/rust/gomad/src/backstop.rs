//! Process-wide fail-stop deadline backstop (fn-232 R1).
//!
//! Wasmtime epoch interruption stops guest instructions, not a blocked native
//! compiler or store teardown. Every linked execution therefore arms this
//! backstop before compilation and keeps it armed through teardown. Cooperative
//! cancellation and the epoch watchdog act first; if an armed execution is
//! still inside its window `CLEANUP_GRACE` after its wall deadline, the whole
//! CLI exits with the existing infrastructure status without joining any
//! (possibly stalled) slot thread. Published evidence stays authenticated and
//! interrupted campaign tails recover through ordinary `resume`.

use std::collections::BTreeMap;
use std::os::fd::AsFd;
use std::sync::{Condvar, Mutex, OnceLock, PoisonError};
use std::time::{Duration, Instant};

/// Bounded cooperative cleanup after an execution's wall deadline; the same
/// bound Go applies when stopping a live cluster helper (`clusterStopGrace`).
pub const CLEANUP_GRACE: Duration = Duration::from_secs(5);
/// The existing CLI status for execution infrastructure failures.
pub const STATUS: i32 = 3;
/// Upper bound on waiting for the best-effort backstop diagnostic.
const DIAGNOSTIC_GRACE: Duration = Duration::from_millis(500);

#[derive(Default)]
struct Armed {
    next: u64,
    deadlines: BTreeMap<u64, Instant>,
}

struct Backstop {
    armed: Mutex<Armed>,
    changed: Condvar,
}

fn backstop() -> &'static Backstop {
    static BACKSTOP: OnceLock<&'static Backstop> = OnceLock::new();
    BACKSTOP.get_or_init(|| {
        let backstop: &'static Backstop = Box::leak(Box::new(Backstop {
            armed: Mutex::new(Armed::default()),
            changed: Condvar::new(),
        }));
        std::thread::Builder::new()
            .name("gomad-deadline-backstop".into())
            .spawn(move || monitor(backstop))
            .unwrap_or_else(|error| fail(&format!("start deadline backstop: {error}")));
        backstop
    })
}

fn monitor(backstop: &'static Backstop) -> ! {
    let mut armed = backstop
        .armed
        .lock()
        .unwrap_or_else(PoisonError::into_inner);
    loop {
        let earliest = armed.deadlines.values().min().copied();
        armed = match earliest {
            None => backstop
                .changed
                .wait(armed)
                .unwrap_or_else(PoisonError::into_inner),
            Some(deadline) => {
                let now = Instant::now();
                if now >= deadline {
                    fail(
                        "execution exceeded its wall deadline and bounded cleanup; deadline backstop ended the process",
                    );
                }
                backstop
                    .changed
                    .wait_timeout(armed, deadline - now)
                    .unwrap_or_else(PoisonError::into_inner)
                    .0
            }
        };
    }
}

/// Exits with the infrastructure status without joining any thread. The
/// diagnostic is best effort: a detached writer bypasses the standard error
/// lock, which a stalled thread may hold, and termination waits on it for at
/// most `DIAGNOSTIC_GRACE`, so a full, undrained stderr pipe (or any other
/// blocked write) cannot keep the process alive. The shared open file
/// description is never switched to non-blocking mode.
fn fail(message: &str) -> ! {
    let line = format!("gomad: runner_failure: {message}\n");
    let (written, delivered) = std::sync::mpsc::channel();
    let writer = std::thread::Builder::new()
        .name("gomad-deadline-diagnostic".into())
        .spawn(move || {
            let _ = rustix::io::write(std::io::stderr().as_fd(), line.as_bytes());
            let _ = written.send(());
        });
    if writer.is_ok() {
        let _ = delivered.recv_timeout(DIAGNOSTIC_GRACE);
    }
    std::process::exit(STATUS)
}

/// Keeps one execution's window armed until dropped.
pub struct Guard(u64);

impl Drop for Guard {
    fn drop(&mut self) {
        let backstop = backstop();
        backstop
            .armed
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .deadlines
            .remove(&self.0);
        backstop.changed.notify_all();
    }
}

/// Arms the backstop for an execution whose wall bound is `wall`.
#[must_use]
pub fn arm(wall: Duration) -> Guard {
    let backstop = backstop();
    let deadline = Instant::now()
        .checked_add(wall.saturating_add(CLEANUP_GRACE))
        .unwrap_or_else(|| Instant::now() + Duration::from_secs(u64::from(u32::MAX)));
    let mut armed = backstop
        .armed
        .lock()
        .unwrap_or_else(PoisonError::into_inner);
    let id = armed.next;
    armed.next += 1;
    armed.deadlines.insert(id, deadline);
    drop(armed);
    backstop.changed.notify_all();
    Guard(id)
}

/// Runs one linked execution, from compilation through teardown, inside an
/// armed backstop window.
pub fn within<T>(wall: Duration, execution: impl FnOnce() -> T) -> T {
    let _guard = arm(wall);
    #[cfg(feature = "determinism-fixture")]
    let ordinal = crate::fixture::backstop_window();
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::backstop_stall("compile", ordinal);
    let result = execution();
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::backstop_stall("teardown", ordinal);
    result
}
