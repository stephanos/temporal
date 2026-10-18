use serde::Serialize;
use serde_json::{Value, json};
use std::{cell::RefCell, io::Write, time::Instant};

#[derive(Clone, Copy, PartialEq)]
pub enum Phase {
    ModuleRead,
    ModuleHash,
    EngineInit,
    CacheIdentity,
    CacheRead,
    EnvelopeVerify,
    Deserialize,
    Compile,
    CacheWrite,
    Instantiate,
    Execute,
}
const PHASE_NAMES: [&str; 11] = [
    "module_read",
    "module_hash",
    "engine_init",
    "cache_identity",
    "cache_read",
    "envelope_verify",
    "deserialize",
    "compile",
    "cache_write",
    "instantiate",
    "execute",
];
#[derive(Clone, Copy)]
enum Abi {
    Clock,
    Scheduler,
    Select,
    Observation,
    Idle,
    OtherLocal,
    RemoteCallback,
}
const ABI_NAMES: [&str; 7] = [
    "clock",
    "scheduler",
    "select",
    "observation",
    "idle",
    "other_local",
    "remote_callback",
];
#[derive(Clone, Copy)]
pub enum Nested {
    AlternativeHash,
    Encoding,
    TranscriptHash,
    Flush,
    RemoteWrite,
    RemoteReadWait,
}
const NESTED_NAMES: [&str; 6] = [
    "alternative_hash",
    "encoding",
    "transcript_hash",
    "flush",
    "remote_write",
    "remote_read_wait",
];

#[derive(Default, Serialize)]
struct PhaseStats {
    count: u64,
    wall_nanos: u64,
    user_nanos: u64,
    system_nanos: u64,
    cpu_samples: u64,
    peak_rss_bytes_high_water: u64,
}
#[derive(Default, Serialize)]
struct AbiStats {
    count: u64,
    wall_nanos: u64,
    instantiate_wall_nanos: u64,
    execute_wall_nanos: u64,
}
#[derive(Default, Serialize)]
struct WallStats {
    count: u64,
    wall_nanos: u64,
}
#[derive(Default)]
struct Profile {
    phases: [PhaseStats; 11],
    abi: [AbiStats; 7],
    nested: [WallStats; 6],
    active_phase: Option<Phase>,
    active_abi: Option<Abi>,
    abi_overlap_errors: u64,
    remote_send: bool,
}
type Usage = Option<(u64, u64, u64)>;
impl Profile {
    fn record_phase(&mut self, phase: Phase, wall: u64, before: Usage, after: Usage) {
        let stats = &mut self.phases[phase as usize];
        stats.count = stats.count.saturating_add(1);
        stats.wall_nanos = stats.wall_nanos.saturating_add(wall);
        if let Some((_, _, rss)) = after {
            stats.peak_rss_bytes_high_water = stats.peak_rss_bytes_high_water.max(rss);
        }
        if let (Some((user0, sys0, _)), Some((user1, sys1, _))) = (before, after)
            && let (Some(user), Some(sys)) = (user1.checked_sub(user0), sys1.checked_sub(sys0))
        {
            stats.cpu_samples = stats.cpu_samples.saturating_add(1);
            stats.user_nanos = stats.user_nanos.saturating_add(user);
            stats.system_nanos = stats.system_nanos.saturating_add(sys);
        }
    }
    fn report(&self) -> Value {
        let phases: serde_json::Map<_, _> = PHASE_NAMES
            .iter()
            .zip(&self.phases)
            .map(|(name, stats)| ((*name).into(), json!(stats)))
            .collect();
        let abi: serde_json::Map<_, _> = ABI_NAMES
            .iter()
            .zip(&self.abi)
            .map(|(name, stats)| ((*name).into(), json!(stats)))
            .collect();
        let nested: serde_json::Map<_, _> = NESTED_NAMES
            .iter()
            .zip(&self.nested)
            .map(|(name, stats)| ((*name).into(), json!(stats)))
            .collect();
        let instantiate = self.abi.iter().fold(0u64, |sum, stats| {
            sum.saturating_add(stats.instantiate_wall_nanos)
        });
        let execute = self.abi.iter().fold(0u64, |sum, stats| {
            sum.saturating_add(stats.execute_wall_nanos)
        });
        json!({"schema":"gomad.wasm-profile/v1", "diagnostic_only":true, "backend":"cranelift-jit",
            "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
            "phases":phases, "abi":abi, "nested":nested, "abi_overlap_errors":self.abi_overlap_errors,
            "residual":{"instantiate_wall_nanos":self.phases[Phase::Instantiate as usize].wall_nanos.saturating_sub(instantiate),
                "execute_wall_nanos":self.phases[Phase::Execute as usize].wall_nanos.saturating_sub(execute)}})
    }
    fn write_report(&self, output: &mut dyn Write) -> std::io::Result<()> {
        let mut bytes = b"gomad-wasm-profile-v1 ".to_vec();
        serde_json::to_writer(&mut bytes, &self.report())?;
        bytes.push(b'\n');
        if bytes.len() >= 16 * 1024 {
            return Err(std::io::Error::other("diagnostic report capacity"));
        }
        output.write_all(&bytes)
    }
}
thread_local! { static PROFILE: RefCell<Option<Profile>> = const { RefCell::new(None) }; }
pub struct Session;
impl Session {
    pub fn start() -> Self {
        PROFILE.with(|profile| *profile.borrow_mut() = Some(Profile::default()));
        Self
    }
}
impl Drop for Session {
    fn drop(&mut self) {
        if let Some(profile) = PROFILE.with(|profile| profile.borrow_mut().take()) {
            // Diagnostic output cannot change the guest's result if its sink closes.
            let _ = profile.write_report(&mut std::io::stderr().lock());
        }
    }
}
pub struct PhaseSpan {
    phase: Phase,
    timer: Option<Instant>,
    usage: Usage,
    previous: Option<Phase>,
}
impl PhaseSpan {
    pub fn new(phase: Phase) -> Self {
        let mut previous = None;
        let active = PROFILE.with(|profile| {
            let mut profile = profile.borrow_mut();
            if let Some(profile) = profile.as_mut() {
                previous = profile.active_phase.replace(phase);
                true
            } else {
                false
            }
        });
        Self {
            phase,
            previous,
            usage: active
                .then(crate::metrics::diagnostic_process_usage)
                .flatten(),
            timer: active.then(Instant::now),
        }
    }
}
impl Drop for PhaseSpan {
    fn drop(&mut self) {
        if let Some(timer) = self.timer {
            let wall = crate::metrics::elapsed(timer);
            let after = crate::metrics::diagnostic_process_usage();
            PROFILE.with(|profile| {
                if let Some(profile) = profile.borrow_mut().as_mut() {
                    profile.record_phase(self.phase, wall, self.usage, after);
                    profile.active_phase = self.previous;
                }
            });
        }
    }
}
pub struct AbiSpan {
    timer: Option<Instant>,
    region: Option<Phase>,
}
impl AbiSpan {
    pub fn new(module: &str, op: &str, local: bool) -> Self {
        let kind = if !local && op != "proc_exit" {
            Abi::RemoteCallback
        } else {
            match (module, op) {
                (_, "clock_time_get") => Abi::Clock,
                ("gomad_wasm_v1", "decision") => Abi::Scheduler,
                (_, "poll_oneoff") => Abi::Select,
                ("gomad_wasm_v1", "observation") => Abi::Observation,
                ("gomad_wasm_v1", "idle") => Abi::Idle,
                _ => Abi::OtherLocal,
            }
        };
        let mut region = None;
        let active = PROFILE.with(|profile| {
            if let Some(profile) = profile.borrow_mut().as_mut() {
                if profile.active_abi.is_some() {
                    profile.abi_overlap_errors = profile.abi_overlap_errors.saturating_add(1);
                    return false;
                }
                region = profile.active_phase;
                profile.active_abi = Some(kind);
                true
            } else {
                false
            }
        });
        Self {
            region,
            timer: active.then(Instant::now),
        }
    }
}
impl Drop for AbiSpan {
    fn drop(&mut self) {
        if let Some(timer) = self.timer {
            let wall = crate::metrics::elapsed(timer);
            PROFILE.with(|profile| {
                if let Some(profile) = profile.borrow_mut().as_mut()
                    && let Some(kind) = profile.active_abi.take()
                {
                    let stats = &mut profile.abi[kind as usize];
                    stats.count = stats.count.saturating_add(1);
                    stats.wall_nanos = stats.wall_nanos.saturating_add(wall);
                    if self.region == Some(Phase::Instantiate) {
                        stats.instantiate_wall_nanos =
                            stats.instantiate_wall_nanos.saturating_add(wall);
                    } else if self.region == Some(Phase::Execute) {
                        stats.execute_wall_nanos = stats.execute_wall_nanos.saturating_add(wall);
                    }
                }
            });
        }
    }
}
pub fn decision_kind(kind: u32) {
    PROFILE.with(|profile| {
        if let Some(profile) = profile.borrow_mut().as_mut()
            && profile.active_abi.is_some()
        {
            match kind {
                1 => profile.active_abi = Some(Abi::Scheduler),
                2 => profile.active_abi = Some(Abi::Select),
                _ => (),
            }
        }
    });
}
pub fn remote_callback() {
    PROFILE.with(|profile| {
        if let Some(profile) = profile.borrow_mut().as_mut()
            && profile.active_abi.is_some()
        {
            profile.active_abi = Some(Abi::RemoteCallback);
        }
    });
}
pub struct WallSpan {
    kind: Nested,
    timer: Option<Instant>,
}
impl WallSpan {
    pub fn new(kind: Nested) -> Self {
        Self {
            kind,
            timer: PROFILE
                .with(|profile| profile.borrow().is_some())
                .then(Instant::now),
        }
    }
    pub fn remote_write() -> Self {
        Self {
            kind: Nested::RemoteWrite,
            timer: PROFILE
                .with(|profile| profile.borrow().as_ref().is_some_and(|p| p.remote_send))
                .then(Instant::now),
        }
    }
}
impl Drop for WallSpan {
    fn drop(&mut self) {
        if let Some(timer) = self.timer {
            let wall = crate::metrics::elapsed(timer);
            PROFILE.with(|profile| {
                if let Some(profile) = profile.borrow_mut().as_mut() {
                    let stats = &mut profile.nested[self.kind as usize];
                    stats.count = stats.count.saturating_add(1);
                    stats.wall_nanos = stats.wall_nanos.saturating_add(wall);
                }
            });
        }
    }
}
pub struct RemoteSend;
impl RemoteSend {
    pub fn new() -> Self {
        PROFILE.with(|profile| {
            if let Some(profile) = profile.borrow_mut().as_mut() {
                profile.remote_send = true;
            }
        });
        Self
    }
}
impl Drop for RemoteSend {
    fn drop(&mut self) {
        PROFILE.with(|profile| {
            if let Some(profile) = profile.borrow_mut().as_mut() {
                profile.remote_send = false;
            }
        });
    }
}
#[cfg(test)]
pub(crate) fn snapshot() -> Value {
    PROFILE.with(|profile| profile.borrow().as_ref().unwrap().report())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn diagnostic_cpu_deltas_are_phase_samples_not_process_totals() {
        let mut profile = Profile::default();
        profile.record_phase(
            Phase::CacheRead,
            37,
            Some((100, 200, 1000)),
            Some((107, 211, 3000)),
        );
        profile.record_phase(Phase::CacheRead, 5, None, Some((2000, 3000, 4000)));
        let report = profile.report();
        let phase = &report["phases"]["cache_read"];
        assert_eq!(phase["count"], 2);
        assert_eq!(phase["wall_nanos"], 42);
        assert_eq!(phase["user_nanos"], 7);
        assert_eq!(phase["system_nanos"], 11);
        assert_eq!(phase["cpu_samples"], 1);
        assert_eq!(phase["peak_rss_bytes_high_water"], 4000);
        assert_eq!(report["backend"], "cranelift-jit");
    }

    #[test]
    fn diagnostic_abi_buckets_do_not_overlap_and_residual_stays_unattributed() {
        let _session = Session::start();
        {
            let _execute = PhaseSpan::new(Phase::Execute);
            {
                let _abi = AbiSpan::new("wasi_snapshot_preview1", "clock_time_get", true);
                let _hash = WallSpan::new(Nested::TranscriptHash);
                std::hint::black_box([7u8; 32]);
            }
            {
                let _abi = AbiSpan::new("gomad_wasm_v1", "decision", true);
                decision_kind(2);
                std::hint::black_box([8u8; 32]);
            }
            {
                let _abi = AbiSpan::new("wasi_snapshot_preview1", "fd_sync", true);
                remote_callback();
                let _wait = WallSpan::new(Nested::RemoteReadWait);
                std::hint::black_box([9u8; 32]);
            }
        }
        let report = snapshot();
        assert_eq!(report["abi"]["clock"]["count"], 1);
        assert_eq!(report["abi"]["select"]["count"], 1);
        assert_eq!(report["abi"]["remote_callback"]["count"], 1);
        assert_eq!(report["abi"]["other_local"]["count"], 0);
        assert_eq!(report["abi_overlap_errors"], 0);
        assert_eq!(report["nested"]["transcript_hash"]["count"], 1);
        assert_eq!(report["nested"]["remote_read_wait"]["count"], 1);
        let measured: u64 = report["abi"]
            .as_object()
            .unwrap()
            .values()
            .map(|v| v["execute_wall_nanos"].as_u64().unwrap())
            .sum();
        assert!(measured > 0);
        assert_eq!(
            report["phases"]["execute"]["wall_nanos"].as_u64().unwrap(),
            measured + report["residual"]["execute_wall_nanos"].as_u64().unwrap()
        );
        assert_eq!(
            report["phases"]["execute"]["cpu_samples"],
            u64::from(crate::metrics::diagnostic_process_usage().is_some())
        );
    }

    #[test]
    fn diagnostic_report_is_fixed_bounded_and_counters_saturate() {
        let mut profile = Profile::default();
        for _ in 0..2 {
            profile.record_phase(
                Phase::Execute,
                u64::MAX,
                Some((0, 0, 0)),
                Some((u64::MAX, u64::MAX, u64::MAX)),
            );
        }
        let report = profile.report();
        assert_eq!(report["phases"]["execute"]["wall_nanos"], u64::MAX);
        assert_eq!(report["phases"]["execute"]["user_nanos"], u64::MAX);
        assert_eq!(report["phases"].as_object().unwrap().len(), 11);
        assert_eq!(report["abi"].as_object().unwrap().len(), 7);
        assert_eq!(report["nested"].as_object().unwrap().len(), 6);
        let mut output = Vec::new();
        profile.write_report(&mut output).unwrap();
        assert!(output.len() < 16 * 1024);
        assert!(output.starts_with(b"gomad-wasm-profile-v1 "));
        #[expect(
            clippy::naive_bytecount,
            reason = "The test counts line delimiters in one small diagnostic frame; no bytecount dependency is needed."
        )]
        {
            assert_eq!(output.iter().filter(|&&byte| byte == b'\n').count(), 1);
        }
        let decoded: serde_json::Value =
            serde_json::from_slice(&output[b"gomad-wasm-profile-v1 ".len()..]).unwrap();
        assert_eq!(decoded, report);
        assert_eq!(decoded["schema"], "gomad.wasm-profile/v1");
    }
}
