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
        // SAFETY: the supported platform ABI writes the complete Rusage layout
        // to this aligned, writable allocation; who=0 selects RUSAGE_SELF.
        if unsafe { getrusage(0, usage.as_mut_ptr()) } != 0 {
            return None;
        }
        // SAFETY: successful getrusage initialized the complete Rusage value.
        let usage = unsafe { usage.assume_init() };
        let rusage_peak = rss_bytes(usage.counters[0])?;
        #[cfg(target_os = "linux")]
        let peak = std::fs::read_to_string("/proc/self/status")
            .ok()
            .and_then(|status| own_peak_rss(&status))
            .unwrap_or(rusage_peak);
        #[cfg(target_os = "macos")]
        let peak = rusage_peak;
        Some((nanos(&usage.user)?, nanos(&usage.system)?, peak))
    }
    /// Linux `ru_maxrss` (`RUSAGE_SELF` and wait4) also folds in the high-water RSS
    /// of the address space replaced by execve. Go spawns helpers with
    /// `CLONE_VM|CLONE_VFORK`, so that is the operator's lifetime peak, not the
    /// helper's. `VmHWM` is the high-water mark of this process image only.
    #[cfg(any(target_os = "linux", test))]
    fn own_peak_rss(status: &str) -> Option<u64> {
        let line = status.lines().find(|line| line.starts_with("VmHWM:"))?;
        let mut fields = line["VmHWM:".len()..].split_whitespace();
        let kib = fields.next()?.parse::<u64>().ok()?;
        if fields.next() != Some("kB") || fields.next().is_some() || kib == 0 {
            return None;
        }
        kib.checked_mul(1024)
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
        #[test]
        fn own_peak_rss_reads_only_wellformed_vmhwm() {
            let status = "Name:\tx\nVmPeak:\t  9 kB\nVmHWM:\t    1792 kB\nVmRSS:\t 1 kB\n";
            assert_eq!(own_peak_rss(status), Some(1792 * 1024));
            for bad in [
                "VmRSS:\t1 kB\n",
                "VmHWM:\t1 MB\n",
                "VmHWM:\t-1 kB\n",
                "VmHWM:\t0 kB\n",
                "VmHWM:\t1 kB extra\n",
                "VmHWM:\t18446744073709551615 kB\n",
            ] {
                assert_eq!(own_peak_rss(bad), None, "{bad}");
            }
        }
        /// A child image must not report the parent's peak. Spawn this test
        /// binary after a large touched allocation and compare the child's
        /// own sample with the inherited rusage figure.
        #[cfg(target_os = "linux")]
        #[test]
        fn exec_does_not_inherit_parent_peak() {
            if std::env::var_os("GOMAD_WASMHOST_PEAK_CHILD").is_some() {
                let (_, _, rss) = sample().unwrap();
                let mut usage = std::mem::MaybeUninit::<Rusage>::zeroed();
                // SAFETY: usage is writable and aligned for the supported ABI.
                assert_eq!(unsafe { getrusage(0, usage.as_mut_ptr()) }, 0);
                // SAFETY: getrusage succeeded and initialized the complete value.
                let inherited = rss_bytes(unsafe { usage.assume_init() }.counters[0]).unwrap();
                println!("child-peak={rss}\nchild-rusage={inherited}");
                return;
            }
            let ballast = vec![1u8; 128 << 20];
            std::hint::black_box(&ballast);
            let output = std::process::Command::new(std::env::current_exe().unwrap())
                .args([
                    "--exact",
                    "metrics::usage::tests::exec_does_not_inherit_parent_peak",
                    "--nocapture",
                    "--test-threads=1",
                ])
                .env("GOMAD_WASMHOST_PEAK_CHILD", "1")
                .output()
                .unwrap();
            drop(ballast);
            let stdout = String::from_utf8(output.stdout).unwrap();
            let field = |name: &str| -> u64 {
                stdout
                    .lines()
                    .find_map(|line| line.split_once(name).map(|(_, value)| value))
                    .unwrap()
                    .parse()
                    .unwrap()
            };
            let (child, inherited) = (field("child-peak="), field("child-rusage="));
            // The kernel behaviour this guards against must be observable here.
            assert!(inherited >= 128 << 20, "rusage {inherited} did not inherit");
            assert!(
                child > 0 && child < 64 << 20,
                "child peak {child} inherited parent peak"
            );
        }
    }
}
