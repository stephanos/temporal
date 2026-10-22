//! Whole-process resource observations. The Rust host links the helper, so one
//! process self figure covers operator and guest stores; only the Go compiler
//! runs as a child and is observed through the waited-children counters.

use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Usage {
    pub user_nanos: u64,
    pub system_nanos: u64,
    pub peak_rss_bytes: u64,
}

impl Usage {
    /// Counters of one process or its waited children never decrease.
    pub(crate) fn not_before(self, earlier: Self) -> bool {
        self.user_nanos >= earlier.user_nanos
            && self.system_nanos >= earlier.system_nanos
            && self.peak_rss_bytes >= earlier.peak_rss_bytes
    }
}

/// `linked` is this process (operator, Wasmtime engines and guest stores);
/// `children` accumulates every waited compiler process and its descendants.
#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Snapshot {
    pub linked: Usage,
    pub children: Usage,
}

pub(crate) fn snapshot() -> Option<Snapshot> {
    Some(Snapshot {
        linked: linked()?,
        children: platform::sample(platform::CHILDREN)?,
    })
}

pub(crate) fn linked() -> Option<Usage> {
    let mut usage = platform::sample(platform::SELF)?;
    if let Some(peak) = own_peak_rss() {
        // After a phase reset the image high water restarts; the lifetime peak
        // is the largest high water observed across every phase.
        usage.peak_rss_bytes = peak.max(phases::LIFETIME.load(Ordering::Acquire));
    }
    Some(usage)
}

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};

/// Phase attribution of the linked process's image high water. Qualification
/// enables it once per process; every other command leaves it off and never
/// resets the kernel counter. On Linux, writing `5` to `/proc/self/clear_refs`
/// restarts `VmHWM`, so the high water read at the next boundary belongs to
/// the phase in between: operator work outside guest stores, or one fresh
/// store window (Wasmtime compilation, instantiation and guest execution).
mod phases {
    use super::{AtomicBool, AtomicU64};
    pub(super) static ENABLED: AtomicBool = AtomicBool::new(false);
    pub(super) static ATTRIBUTED: AtomicBool = AtomicBool::new(false);
    pub(super) static OPERATOR: AtomicU64 = AtomicU64::new(0);
    pub(super) static LIFETIME: AtomicU64 = AtomicU64::new(0);
}

#[cfg(target_os = "linux")]
fn reset_peak() -> bool {
    std::fs::write("/proc/self/clear_refs", b"5").is_ok()
}

#[cfg(not(target_os = "linux"))]
fn reset_peak() -> bool {
    false
}

/// Returns heap the closed phase freed to the kernel. `clear_refs` restarts
/// `VmHWM` at the current RSS, and glibc keeps freed arena pages resident, so
/// without this the next phase would start from the previous phase's freed
/// memory (a store window's compiler and guest heap charged to the operator).
/// Live memory stays resident and stays charged to the phase that holds it.
#[cfg(all(target_os = "linux", target_env = "gnu"))]
fn release_freed_heap() {
    unsafe extern "C" {
        fn malloc_trim(pad: usize) -> std::ffi::c_int;
    }
    // SAFETY: glibc `malloc_trim` only returns free heap pages; it has no
    // preconditions and its result (whether memory was released) is advisory.
    unsafe {
        malloc_trim(0);
    }
}

#[cfg(not(all(target_os = "linux", target_env = "gnu")))]
fn release_freed_heap() {}

/// Reads the current phase's high water, folds it into the lifetime peak,
/// releases freed heap and restarts it. Returns `None` once attribution has
/// failed.
fn close_phase() -> Option<u64> {
    let peak = own_peak_rss();
    if let Some(peak) = peak {
        phases::LIFETIME.fetch_max(peak, Ordering::AcqRel);
    }
    release_freed_heap();
    if peak.is_none() || !reset_peak() {
        phases::ATTRIBUTED.store(false, Ordering::Release);
        return None;
    }
    peak.filter(|_| phases::ATTRIBUTED.load(Ordering::Acquire))
}

/// Enables phase attribution for this process; the first operator phase
/// starts now. Unavailable counters leave attribution off (pending).
pub(crate) fn enable_phases() {
    phases::ATTRIBUTED.store(true, Ordering::Release);
    phases::ENABLED.store(true, Ordering::Release);
    if let Some(peak) = close_phase() {
        phases::OPERATOR.fetch_max(peak, Ordering::AcqRel);
    }
}

/// Ends the preceding operator phase before a fresh guest store starts.
pub(crate) fn store_begin() {
    if phases::ENABLED.load(Ordering::Acquire)
        && let Some(peak) = close_phase()
    {
        phases::OPERATOR.fetch_max(peak, Ordering::AcqRel);
    }
}

/// Ends a store window and returns its own image high water when attributed.
pub(crate) fn store_end() -> Option<u64> {
    if !phases::ENABLED.load(Ordering::Acquire) {
        return None;
    }
    close_phase()
}

/// The largest operator-phase high water so far, including the open phase.
pub(crate) fn operator_peak() -> Option<u64> {
    if !phases::ENABLED.load(Ordering::Acquire) {
        return None;
    }
    let open = own_peak_rss()?;
    phases::LIFETIME.fetch_max(open, Ordering::AcqRel);
    phases::ATTRIBUTED
        .load(Ordering::Acquire)
        .then(|| phases::OPERATOR.load(Ordering::Acquire).max(open))
}

/// Linux `ru_maxrss` for this process also folds in the address space replaced
/// by execve; `VmHWM` is the high-water mark of this image only.
#[cfg(target_os = "linux")]
fn own_peak_rss() -> Option<u64> {
    let status = std::fs::read_to_string("/proc/self/status").ok()?;
    let line = status.lines().find(|line| line.starts_with("VmHWM:"))?;
    let mut fields = line["VmHWM:".len()..].split_whitespace();
    let kib = fields.next()?.parse::<u64>().ok()?;
    if fields.next() != Some("kB") || fields.next().is_some() || kib == 0 {
        return None;
    }
    kib.checked_mul(1024)
}

#[cfg(not(target_os = "linux"))]
fn own_peak_rss() -> Option<u64> {
    None
}

#[cfg(all(
    target_pointer_width = "64",
    any(target_arch = "aarch64", target_arch = "x86_64"),
    any(target_os = "linux", target_os = "macos")
))]
mod platform {
    use super::Usage;
    use std::ffi::{c_int, c_long};
    #[cfg(target_os = "linux")]
    type Microseconds = c_long;
    #[cfg(target_os = "macos")]
    type Microseconds = i32;

    pub(super) const SELF: c_int = 0;
    pub(super) const CHILDREN: c_int = -1;

    #[repr(C)]
    struct Timeval {
        seconds: c_long,
        micros: Microseconds,
    }
    #[repr(C)]
    struct Rusage {
        user: Timeval,
        system: Timeval,
        counters: [c_long; 14],
        #[cfg(any(target_env = "musl", target_env = "ohos"))]
        reserved: [c_long; 16],
    }
    unsafe extern "C" {
        fn getrusage(who: c_int, usage: *mut Rusage) -> c_int;
    }

    fn nanos(time: &Timeval) -> Option<u64> {
        let seconds = u64::try_from(time.seconds).ok()?;
        let micros = u64::try_from(time.micros).ok()?;
        if micros >= 1_000_000 {
            return None;
        }
        seconds
            .checked_mul(1_000_000_000)?
            .checked_add(micros * 1000)
    }

    pub(super) fn sample(who: c_int) -> Option<Usage> {
        let mut usage = std::mem::MaybeUninit::<Rusage>::zeroed();
        // SAFETY: the supported platform ABI writes the complete Rusage layout
        // to this aligned, writable allocation for RUSAGE_SELF/RUSAGE_CHILDREN.
        if unsafe { getrusage(who, usage.as_mut_ptr()) } != 0 {
            return None;
        }
        // SAFETY: successful getrusage initialized the complete Rusage value.
        let usage = unsafe { usage.assume_init() };
        let rss = u64::try_from(usage.counters[0]).ok()?;
        #[cfg(target_os = "linux")]
        let rss = rss.checked_mul(1024)?;
        Some(Usage {
            user_nanos: nanos(&usage.user)?,
            system_nanos: nanos(&usage.system)?,
            peak_rss_bytes: rss,
        })
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        // Invariant: the FFI layout matches the platform ABI. A CLI run cannot
        // distinguish a misread field from a genuine measurement.
        #[test]
        fn platform_layout_and_units_are_checked() {
            assert_eq!(std::mem::size_of::<Timeval>(), 16);
            assert_eq!(
                std::mem::size_of::<Rusage>(),
                if cfg!(any(target_env = "musl", target_env = "ohos")) {
                    272
                } else {
                    144
                }
            );
            assert_eq!(
                nanos(&Timeval {
                    seconds: 2,
                    micros: 3
                }),
                Some(2_000_003_000)
            );
            assert_eq!(
                nanos(&Timeval {
                    seconds: 0,
                    micros: 1_000_000
                }),
                None
            );
        }
    }
}

#[cfg(not(all(
    target_pointer_width = "64",
    any(target_arch = "aarch64", target_arch = "x86_64"),
    any(target_os = "linux", target_os = "macos")
)))]
mod platform {
    pub(super) const SELF: i32 = 0;
    pub(super) const CHILDREN: i32 = -1;
    pub(super) fn sample(_: i32) -> Option<super::Usage> {
        None
    }
}
