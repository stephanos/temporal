use serde::Serialize;
use std::time::Instant;

#[derive(Default, Serialize)]
pub struct Metrics {
    pub prepare_nanos: u64,
    pub compile_nanos: u64,
    pub cache_load_nanos: u64,
    pub cache_write_nanos: u64,
    pub cache_hit: bool,
    pub instantiate_nanos: u64,
    pub execute_nanos: u64,
    pub process_user_nanos: u64,
    pub process_system_nanos: u64,
    pub peak_rss_bytes: u64,
    pub resource_usage_available: bool,
}
impl Metrics {
    pub fn sample_process(&mut self) {
        if let Some((user, system, rss)) = process_usage() {
            self.process_user_nanos = user;
            self.process_system_nanos = system;
            self.peak_rss_bytes = rss;
            self.resource_usage_available = true;
        }
    }
}
pub fn elapsed(start: Instant) -> u64 {
    u64::try_from(start.elapsed().as_nanos()).unwrap_or(u64::MAX)
}

#[cfg(feature = "diagnostic-profile")]
pub(crate) fn diagnostic_process_usage() -> Option<(u64, u64, u64)> {
    process_usage()
}

#[cfg(all(
    target_pointer_width = "64",
    any(target_arch = "aarch64", target_arch = "x86_64"),
    any(target_os = "linux", target_os = "macos")
))]
fn process_usage() -> Option<(u64, u64, u64)> {
    usage::sample()
}
#[cfg(not(all(
    target_pointer_width = "64",
    any(target_arch = "aarch64", target_arch = "x86_64"),
    any(target_os = "linux", target_os = "macos")
)))]
fn process_usage() -> Option<(u64, u64, u64)> {
    None
}

#[cfg(all(
    target_pointer_width = "64",
    any(target_arch = "aarch64", target_arch = "x86_64"),
    any(target_os = "linux", target_os = "macos")
))]
mod usage {
    use std::ffi::{c_int, c_long};
    #[cfg(target_os = "linux")]
    type Microseconds = c_long;
    #[cfg(target_os = "macos")]
    type Microseconds = i32;

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
    fn rss_bytes(rss: c_long) -> Option<u64> {
        let rss = u64::try_from(rss).ok()?;
        #[cfg(target_os = "linux")]
        {
            rss.checked_mul(1024)
        }
        #[cfg(target_os = "macos")]
        {
            Some(rss)
        }
    }
    pub fn sample() -> Option<(u64, u64, u64)> {
        let mut usage = std::mem::MaybeUninit::<Rusage>::zeroed();
        // RUSAGE_SELF writes the platform's complete 64-bit rusage layout.
        if unsafe { getrusage(0, usage.as_mut_ptr()) } != 0 {
            return None;
        }
        let usage = unsafe { usage.assume_init() };
        Some((
            nanos(&usage.user)?,
            nanos(&usage.system)?,
            rss_bytes(usage.counters[0])?,
        ))
    }
    #[cfg(test)]
    mod tests {
        use super::*;
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
                    seconds: -1,
                    micros: 0
                }),
                None
            );
            assert_eq!(
                nanos(&Timeval {
                    seconds: 0,
                    micros: 1_000_000
                }),
                None
            );
            assert_eq!(
                nanos(&Timeval {
                    seconds: c_long::MAX,
                    micros: 0
                }),
                None
            );
            #[cfg(target_os = "linux")]
            assert_eq!(rss_bytes(2), Some(2048));
            #[cfg(target_os = "macos")]
            assert_eq!(rss_bytes(2), Some(2));
            let (_, _, rss) = sample().unwrap();
            assert!(rss > 0);
        }
    }
}
