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
        usage.peak_rss_bytes = peak;
    }
    Some(usage)
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
