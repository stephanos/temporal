//! Diagnostic host profile (`--features diagnostic-profile`; never
//! qualification evidence). Besides the coarse phase, ABI-kind and nested
//! buckets of `gomad.wasm-profile/v1`, schema v2 attributes every host import
//! to its exact `module.op` (`abi_ops`) and to the model owner that served it
//! (`owners`), with exclusive wall from a span stack, a fixed log-linear
//! per-call distribution, and a reconciliation row: within `Execute`, an outer
//! timer at the import trampoline measures guest gaps directly, so
//! `execute = guest residual + outer host wall` holds exactly and the
//! attributed owner wall must close the remainder (fn-185 R1).
use serde::Serialize;
use serde_json::{Map, Value, json};
use std::{
    cell::RefCell,
    io::Write,
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, Instant},
};

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
impl Nested {
    fn owner(self) -> Owner {
        match self {
            Self::AlternativeHash | Self::TranscriptHash => Owner::Transcript,
            Self::Encoding => Owner::Encoding,
            Self::Flush => Owner::Evidence,
            Self::RemoteWrite | Self::RemoteReadWait => Owner::Remote,
        }
    }
}
/// The closed set of model owners a host import or nested span is charged to.
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Owner {
    Clock,
    Scheduler,
    SchedulerHandoff,
    Namespace,
    Volume,
    Network,
    World,
    StorageTransport,
    Cache,
    Transcript,
    Evidence,
    Encoding,
    Remote,
    Inputs,
    Inspection,
    Lifecycle,
    /// The profiler's own per-import cost: boundary entry to op-frame start
    /// plus op-frame end to boundary exit. A profiler artifact, never a lever.
    Instrumentation,
    /// A cluster instance blocked for the baton while others run: waiting
    /// time, not host work.
    HandoffWait,
}
const OWNERS: usize = 18;
const OWNER_NAMES: [&str; OWNERS] = [
    "clock",
    "scheduler",
    "scheduler_handoff",
    "namespace",
    "volume",
    "network",
    "world",
    "storage_transport",
    "cache",
    "transcript",
    "evidence",
    "encoding",
    "remote",
    "inputs",
    "inspection",
    "lifecycle",
    "instrumentation",
    "handoff_wait",
];
const OPS: usize = crate::abi::Operation::COUNT;
/// Log-linear per-call histogram: 16 exact buckets below 16 ns, then 16
/// sub-buckets per power of two up to 2^40 ns (the last bucket saturates).
/// A reported quantile is its bucket's upper bound clamped to the exact
/// maximum, so its relative error is below 1/16.
const BUCKETS: usize = 16 + (40 - 4 + 1) * 16;
const DISTRIBUTION_RELATIVE_ERROR_PPM: u64 = 62_500;
const STACK_DEPTH: usize = 16;
/// Reconciliation tolerance: max(50 µs, 0.5 % of execute wall).
const TOLERANCE_FLOOR_NANOS: u64 = 50_000;
const TOLERANCE_DIVISOR: u64 = 200;
const REPORT_LIMIT: usize = 48 * 1024;
const MAX_THREAD_ROWS: usize = 64;

fn bucket(value: u64) -> usize {
    if value < 16 {
        return usize::try_from(value).unwrap();
    }
    let exponent = 63 - value.leading_zeros() as usize;
    if exponent > 40 {
        return BUCKETS - 1;
    }
    let sub = ((value >> (exponent - 4)) & 15) as usize;
    16 + (exponent - 4) * 16 + sub
}
fn bucket_upper(index: usize) -> u64 {
    if index < 16 {
        return index as u64;
    }
    let exponent = (index - 16) / 16 + 4;
    let sub = ((index - 16) % 16) as u64;
    ((16 + sub) << (exponent - 4)) + (1u64 << (exponent - 4)) - 1
}
fn nanos(duration: Duration) -> u64 {
    u64::try_from(duration.as_nanos()).unwrap_or(u64::MAX)
}

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
/// One `abi_ops` or `owners` series. Op series time each call inclusively;
/// owner series time each charged frame (inclusive wall, exclusive of child
/// frames charged elsewhere).
#[derive(Clone, Copy)]
struct Series {
    count: u64,
    wall_nanos: u64,
    exclusive_wall_nanos: u64,
    instantiate_wall_nanos: u64,
    execute_wall_nanos: u64,
    execute_exclusive_wall_nanos: u64,
    min_nanos: u64,
    max_nanos: u64,
}
impl Default for Series {
    fn default() -> Self {
        Self {
            count: 0,
            wall_nanos: 0,
            exclusive_wall_nanos: 0,
            instantiate_wall_nanos: 0,
            execute_wall_nanos: 0,
            execute_exclusive_wall_nanos: 0,
            min_nanos: u64::MAX,
            max_nanos: 0,
        }
    }
}
impl Series {
    fn record(&mut self, region: Option<Phase>, wall: u64, exclusive: u64, sample: u64) {
        self.count = self.count.saturating_add(1);
        self.wall_nanos = self.wall_nanos.saturating_add(wall);
        self.exclusive_wall_nanos = self.exclusive_wall_nanos.saturating_add(exclusive);
        match region {
            Some(Phase::Instantiate) => {
                self.instantiate_wall_nanos = self.instantiate_wall_nanos.saturating_add(wall);
            }
            Some(Phase::Execute) => {
                self.execute_wall_nanos = self.execute_wall_nanos.saturating_add(wall);
                self.execute_exclusive_wall_nanos =
                    self.execute_exclusive_wall_nanos.saturating_add(exclusive);
            }
            _ => (),
        }
        self.min_nanos = self.min_nanos.min(sample);
        self.max_nanos = self.max_nanos.max(sample);
    }
    fn merge(&mut self, other: &Self) {
        self.count = self.count.saturating_add(other.count);
        self.wall_nanos = self.wall_nanos.saturating_add(other.wall_nanos);
        self.exclusive_wall_nanos = self
            .exclusive_wall_nanos
            .saturating_add(other.exclusive_wall_nanos);
        self.instantiate_wall_nanos = self
            .instantiate_wall_nanos
            .saturating_add(other.instantiate_wall_nanos);
        self.execute_wall_nanos = self
            .execute_wall_nanos
            .saturating_add(other.execute_wall_nanos);
        self.execute_exclusive_wall_nanos = self
            .execute_exclusive_wall_nanos
            .saturating_add(other.execute_exclusive_wall_nanos);
        self.min_nanos = self.min_nanos.min(other.min_nanos);
        self.max_nanos = self.max_nanos.max(other.max_nanos);
    }
    fn report(&self, histogram: &[u64]) -> Map<String, Value> {
        let quantile = |numerator: u64, denominator: u64| {
            let rank = (u128::from(self.count) * u128::from(numerator))
                .div_ceil(u128::from(denominator))
                .max(1);
            let mut seen = 0u128;
            for (index, count) in histogram.iter().enumerate() {
                seen += u128::from(*count);
                if seen >= rank {
                    return bucket_upper(index).clamp(self.min_nanos, self.max_nanos);
                }
            }
            self.max_nanos
        };
        let mut out = Map::new();
        for (key, value) in [
            ("count", self.count),
            ("wall_nanos", self.wall_nanos),
            ("exclusive_wall_nanos", self.exclusive_wall_nanos),
            ("instantiate_wall_nanos", self.instantiate_wall_nanos),
            ("execute_wall_nanos", self.execute_wall_nanos),
            (
                "execute_exclusive_wall_nanos",
                self.execute_exclusive_wall_nanos,
            ),
            ("min_nanos", self.min_nanos),
            ("median_nanos", quantile(1, 2)),
            ("p99_nanos", quantile(99, 100)),
            ("max_nanos", self.max_nanos),
        ] {
            out.insert(key.into(), json!(value));
        }
        out
    }
}
/// An open span on the stack: an import's op frame (`op` set) or an owner
/// frame. `child_nanos` is the inclusive wall of frames pushed directly on it.
#[derive(Clone, Copy)]
struct Frame {
    op: Option<usize>,
    owner: Option<Owner>,
    legacy: Option<Abi>,
    region: Option<Phase>,
    start: Instant,
    child_nanos: u64,
}
struct Profile {
    phases: [PhaseStats; 11],
    abi: [AbiStats; 7],
    nested: [WallStats; 6],
    abi_excluded: AbiStats,
    active_phase: Option<Phase>,
    abi_overlap_errors: u64,
    remote_send: bool,
    ops: [Series; OPS],
    served_by: [[u64; OWNERS]; OPS],
    owners: [Series; OWNERS],
    /// Op histograms first, then owner histograms, `BUCKETS` each.
    histograms: Box<[u64]>,
    stack: Vec<Frame>,
    op_frame: Option<usize>,
    unattributed_imports: u64,
    span_errors: u64,
    outer_imports: u64,
    framed_imports: u64,
    cursor: Option<Instant>,
    guest_residual_nanos: u64,
    outer_host_nanos: u64,
    /// The closed op frame of the current import: (start, end).
    boundary_op: Option<(Instant, Instant)>,
    /// Per store-thread reconciliation rows of a merged cluster profile.
    threads: Vec<((u64, u64), [u64; 6])>,
    threads_omitted: u64,
}
impl Default for Profile {
    fn default() -> Self {
        Self {
            phases: Default::default(),
            abi: Default::default(),
            nested: Default::default(),
            abi_excluded: AbiStats::default(),
            active_phase: None,
            abi_overlap_errors: 0,
            remote_send: false,
            ops: [Series::default(); OPS],
            served_by: [[0; OWNERS]; OPS],
            owners: [Series::default(); OWNERS],
            histograms: vec![0; (OPS + OWNERS) * BUCKETS].into_boxed_slice(),
            stack: Vec::with_capacity(STACK_DEPTH),
            op_frame: None,
            unattributed_imports: 0,
            span_errors: 0,
            outer_imports: 0,
            framed_imports: 0,
            cursor: None,
            guest_residual_nanos: 0,
            outer_host_nanos: 0,
            boundary_op: None,
            threads: Vec::new(),
            threads_omitted: 0,
        }
    }
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
    fn histogram(&mut self, series: usize, sample: u64) {
        let slot = &mut self.histograms[series * BUCKETS + bucket(sample)];
        *slot = slot.saturating_add(1);
    }
    /// Pushes a frame unless the innermost frame already belongs to `owner`
    /// (its time then stays there). Outside an import, owner spans during
    /// `Execute` are span errors: every host instant there belongs to an op.
    fn push(&mut self, owner: Option<Owner>, op: Option<usize>, start: Instant) -> Option<usize> {
        match self.stack.last() {
            Some(top) if op.is_none() && owner.is_some() && top.owner == owner => return None,
            None if op.is_none() && self.active_phase == Some(Phase::Execute) => {
                self.span_errors = self.span_errors.saturating_add(1);
                return None;
            }
            _ => (),
        }
        if self.stack.len() == STACK_DEPTH {
            self.span_errors = self.span_errors.saturating_add(1);
            return None;
        }
        self.stack.push(Frame {
            op,
            owner,
            legacy: None,
            region: self.active_phase,
            start,
            child_nanos: 0,
        });
        Some(self.stack.len() - 1)
    }
    /// Pops frame `index`, charges its exclusive wall to its owner and its
    /// inclusive wall to its parent. Returns the frame and its walls.
    fn pop(&mut self, index: usize, end: Instant) -> Option<(Frame, u64, u64)> {
        if self.stack.len() != index + 1 {
            self.span_errors = self.span_errors.saturating_add(1);
            if self.stack.len() <= index {
                return None;
            }
            self.stack.truncate(index + 1);
        }
        let frame = self.stack.pop()?;
        let wall = nanos(end.saturating_duration_since(frame.start));
        let exclusive = if let Some(exclusive) = wall.checked_sub(frame.child_nanos) {
            exclusive
        } else {
            self.span_errors = self.span_errors.saturating_add(1);
            0
        };
        if let Some(parent) = self.stack.last_mut() {
            parent.child_nanos = parent.child_nanos.saturating_add(wall);
        }
        if let Some(owner) = frame.owner {
            self.owners[owner as usize].record(frame.region, wall, exclusive, exclusive);
            self.histogram(OPS + owner as usize, exclusive);
        }
        Some((frame, wall, exclusive))
    }
    /// execute, attributed, guest residual, outer host, difference, tolerance.
    fn reconciliation(&self) -> [u64; 6] {
        let execute = self.phases[Phase::Execute as usize].wall_nanos;
        let attributed = self.owners.iter().fold(0u64, |sum, owner| {
            sum.saturating_add(owner.execute_exclusive_wall_nanos)
        });
        let accounted = attributed.saturating_add(self.guest_residual_nanos);
        [
            execute,
            attributed,
            self.guest_residual_nanos,
            self.outer_host_nanos,
            execute.abs_diff(accounted),
            TOLERANCE_FLOOR_NANOS.max(execute / TOLERANCE_DIVISOR),
        ]
    }
    /// Merges a store thread's profile, keeping its own reconciliation row.
    fn merge_thread(&mut self, key: (u64, u64), other: &Self) {
        if self.threads.len() < MAX_THREAD_ROWS {
            self.threads.push((key, other.reconciliation()));
        } else {
            self.threads_omitted = self.threads_omitted.saturating_add(1);
        }
        self.merge(other);
    }
    fn merge(&mut self, other: &Self) {
        for (mine, theirs) in self.phases.iter_mut().zip(&other.phases) {
            mine.count = mine.count.saturating_add(theirs.count);
            mine.wall_nanos = mine.wall_nanos.saturating_add(theirs.wall_nanos);
            mine.user_nanos = mine.user_nanos.saturating_add(theirs.user_nanos);
            mine.system_nanos = mine.system_nanos.saturating_add(theirs.system_nanos);
            mine.cpu_samples = mine.cpu_samples.saturating_add(theirs.cpu_samples);
            mine.peak_rss_bytes_high_water = mine
                .peak_rss_bytes_high_water
                .max(theirs.peak_rss_bytes_high_water);
        }
        let abi = |mine: &mut AbiStats, theirs: &AbiStats| {
            mine.count = mine.count.saturating_add(theirs.count);
            mine.wall_nanos = mine.wall_nanos.saturating_add(theirs.wall_nanos);
            mine.instantiate_wall_nanos = mine
                .instantiate_wall_nanos
                .saturating_add(theirs.instantiate_wall_nanos);
            mine.execute_wall_nanos = mine
                .execute_wall_nanos
                .saturating_add(theirs.execute_wall_nanos);
        };
        for (mine, theirs) in self.abi.iter_mut().zip(&other.abi) {
            abi(mine, theirs);
        }
        abi(&mut self.abi_excluded, &other.abi_excluded);
        for (mine, theirs) in self.nested.iter_mut().zip(&other.nested) {
            mine.count = mine.count.saturating_add(theirs.count);
            mine.wall_nanos = mine.wall_nanos.saturating_add(theirs.wall_nanos);
        }
        for (mine, theirs) in self.ops.iter_mut().zip(&other.ops) {
            mine.merge(theirs);
        }
        for (mine, theirs) in self.owners.iter_mut().zip(&other.owners) {
            mine.merge(theirs);
        }
        for (mine, theirs) in self.served_by.iter_mut().zip(&other.served_by) {
            for (mine, theirs) in mine.iter_mut().zip(theirs) {
                *mine = mine.saturating_add(*theirs);
            }
        }
        for (mine, theirs) in self.histograms.iter_mut().zip(other.histograms.iter()) {
            *mine = mine.saturating_add(*theirs);
        }
        for (mine, theirs) in [
            (&mut self.abi_overlap_errors, other.abi_overlap_errors),
            (&mut self.unattributed_imports, other.unattributed_imports),
            (&mut self.span_errors, other.span_errors),
            (&mut self.outer_imports, other.outer_imports),
            (&mut self.framed_imports, other.framed_imports),
            (&mut self.guest_residual_nanos, other.guest_residual_nanos),
            (&mut self.outer_host_nanos, other.outer_host_nanos),
        ] {
            *mine = mine.saturating_add(theirs);
        }
        // A profile still holding open frames was cut short.
        if !other.stack.is_empty() {
            self.span_errors = self.span_errors.saturating_add(other.stack.len() as u64);
        }
    }
    fn report(&self) -> Value {
        let phases: Map<_, _> = PHASE_NAMES
            .iter()
            .zip(&self.phases)
            .map(|(name, stats)| ((*name).into(), json!(stats)))
            .collect();
        let abi: Map<_, _> = ABI_NAMES
            .iter()
            .zip(&self.abi)
            .map(|(name, stats)| ((*name).into(), json!(stats)))
            .collect();
        let nested: Map<_, _> = NESTED_NAMES
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
        let mut abi_ops = Map::new();
        for (index, op) in crate::abi::Operation::ALL.iter().enumerate() {
            let series = &self.ops[index];
            if series.count == 0 {
                continue;
            }
            let mut entry = series.report(&self.histograms[index * BUCKETS..][..BUCKETS]);
            let served: Map<_, _> = OWNER_NAMES
                .iter()
                .zip(&self.served_by[index])
                .filter(|(_, count)| **count != 0)
                .map(|(name, count)| ((*name).into(), json!(count)))
                .collect();
            entry.insert("served_by".into(), Value::Object(served));
            abi_ops.insert(
                format!("{}.{}", op.module(), op.name()),
                Value::Object(entry),
            );
        }
        let mut owners = Map::new();
        for (index, name) in OWNER_NAMES.iter().enumerate() {
            let series = &self.owners[index];
            if series.count != 0 {
                let histogram = &self.histograms[(OPS + index) * BUCKETS..][..BUCKETS];
                owners.insert((*name).into(), Value::Object(series.report(histogram)));
            }
        }
        let [
            execute_wall,
            attributed,
            guest,
            outer,
            difference,
            tolerance,
        ] = self.reconciliation();
        let threads: Vec<Value> = self
            .threads
            .iter()
            .map(
                |(key, [execute, attributed, guest, outer, difference, tolerance])| {
                    json!({"slot":key.0, "incarnation":key.1,
                    "execute_wall_nanos":execute, "attributed_host_wall_nanos":attributed,
                    "guest_residual_wall_nanos":guest, "outer_host_wall_nanos":outer,
                    "difference_nanos":difference, "tolerance_nanos":tolerance})
                },
            )
            .collect();
        json!({"schema":"gomad.wasm-profile/v2", "diagnostic_only":true, "backend":"cranelift-jit",
            "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
            "phases":phases, "abi":abi, "nested":nested, "abi_overlap_errors":self.abi_overlap_errors,
            "residual":{"instantiate_wall_nanos":self.phases[Phase::Instantiate as usize].wall_nanos.saturating_sub(instantiate),
                "execute_wall_nanos":self.phases[Phase::Execute as usize].wall_nanos.saturating_sub(execute)},
            "abi_excluded":self.abi_excluded,
            "abi_ops":abi_ops, "owners":owners,
            "reconciliation":{"execute_wall_nanos":execute_wall,
                "attributed_host_wall_nanos":attributed,
                "guest_residual_wall_nanos":guest,
                "outer_host_wall_nanos":outer,
                "difference_nanos":difference,
                "tolerance_nanos":tolerance,
                "distribution_relative_error_ppm":DISTRIBUTION_RELATIVE_ERROR_PPM},
            "threads":threads, "threads_omitted":self.threads_omitted,
            "unattributed_imports":self.unattributed_imports,
            "unframed_imports":self.outer_imports.saturating_sub(self.framed_imports),
            "span_errors":self.span_errors})
    }
    fn write_report(&self, output: &mut dyn Write) -> std::io::Result<()> {
        let mut bytes = b"gomad-wasm-profile-v1 ".to_vec();
        serde_json::to_writer(&mut bytes, &self.report())?;
        bytes.push(b'\n');
        if bytes.len() >= REPORT_LIMIT {
            return Err(std::io::Error::other("diagnostic report capacity"));
        }
        output.write_all(&bytes)
    }
}
thread_local! { static PROFILE: RefCell<Option<Profile>> = const { RefCell::new(None) }; }
/// Set while the process session is open; cluster store threads profile
/// into their own thread profile and hand it to `FINISHED` on exit.
static ACTIVE: AtomicBool = AtomicBool::new(false);
static FINISHED: Mutex<Option<Profile>> = Mutex::new(None);
#[cfg(test)]
static TEST_SESSION: Mutex<()> = Mutex::new(());
fn with<T>(body: impl FnOnce(&mut Profile) -> T) -> Option<T> {
    PROFILE.with(|profile| profile.borrow_mut().as_mut().map(body))
}
/// Request-owned linked capture. Store threads receive an explicit clone of
/// its sink; the transport process globals never participate in this session.
pub struct LinkedProfile {
    sink: LinkedSink,
    start: Instant,
    finished: bool,
    _thread: std::marker::PhantomData<std::rc::Rc<()>>,
}
#[derive(Clone)]
pub(crate) struct LinkedSink(Arc<Mutex<Option<Profile>>>);
thread_local! {
    static LINKED_SINK: RefCell<Option<LinkedSink>> = const { RefCell::new(None) };
}
impl LinkedProfile {
    /// # Errors
    /// Refuses overlapping captures on the calling execution thread.
    pub fn start() -> Result<Self, String> {
        if PROFILE.with(|profile| profile.borrow().is_some()) {
            return Err("overlapping linked diagnostic session".into());
        }
        let sink = LinkedSink(Arc::new(Mutex::new(None)));
        PROFILE.with(|profile| *profile.borrow_mut() = Some(Profile::default()));
        LINKED_SINK.with(|current| *current.borrow_mut() = Some(sink.clone()));
        Ok(Self {
            sink,
            start: Instant::now(),
            finished: false,
            _thread: std::marker::PhantomData,
        })
    }
    /// # Errors
    /// Finalizes bounded diagnostic data, including interrupted execution prefixes.
    pub fn finish(mut self) -> Result<Value, String> {
        let elapsed = nanos(self.start.elapsed());
        let mut profile = PROFILE
            .with(|profile| profile.borrow_mut().take())
            .ok_or("missing linked diagnostic profile")?;
        LINKED_SINK.with(|current| *current.borrow_mut() = None);
        self.finished = true;
        let finished = self
            .sink
            .0
            .lock()
            .map_err(|_| "linked diagnostic lock poisoned")?
            .take();
        if let Some(finished) = finished {
            profile.merge(&finished);
            profile.threads = finished.threads;
            profile.threads_omitted = finished.threads_omitted;
            profile.threads.sort_by_key(|(key, _)| *key);
        }
        let mut report = profile.report();
        report["elapsed_wall_nanos"] = json!(elapsed);
        report["wall_basis"] = json!(if profile.threads.is_empty() {
            "single-store elapsed"
        } else {
            "aggregate store thread-time; overlapping spans are not elapsed shares"
        });
        report["report_rows_dropped"] = json!(0);
        let mut dropped = 0u64;
        while serde_json::to_vec(&report)
            .map_err(|error| error.to_string())?
            .len()
            >= REPORT_LIMIT
        {
            if report["threads"]
                .as_array_mut()
                .is_some_and(|rows| rows.pop().is_some())
            {
                let omitted = report["threads_omitted"].as_u64().unwrap_or(0);
                report["threads_omitted"] = json!(omitted.saturating_add(1));
            } else if let Some(name) = report["abi_ops"]
                .as_object()
                .and_then(|ops| ops.keys().next_back().cloned())
            {
                report["abi_ops"]
                    .as_object_mut()
                    .ok_or("diagnostic operation rows absent")?
                    .remove(&name);
            } else {
                return Err("diagnostic report capacity".into());
            }
            dropped = dropped.saturating_add(1);
            report["report_rows_dropped"] = json!(dropped);
        }
        Ok(report)
    }
}
impl Drop for LinkedProfile {
    fn drop(&mut self) {
        if !self.finished {
            PROFILE.with(|profile| *profile.borrow_mut() = None);
            LINKED_SINK.with(|current| *current.borrow_mut() = None);
        }
    }
}
impl LinkedSink {
    pub(crate) fn current() -> Option<Self> {
        LINKED_SINK.with(|current| current.borrow().clone())
    }
    pub(crate) fn enter(&self, slot: u64, incarnation: u64) -> Result<LinkedThread, String> {
        if PROFILE.with(|profile| profile.borrow().is_some()) {
            return Err("overlapping linked diagnostic thread".into());
        }
        PROFILE.with(|profile| *profile.borrow_mut() = Some(Profile::default()));
        Ok(LinkedThread {
            sink: self.clone(),
            key: (slot, incarnation),
            _thread: std::marker::PhantomData,
        })
    }
}
pub(crate) struct LinkedThread {
    sink: LinkedSink,
    key: (u64, u64),
    _thread: std::marker::PhantomData<std::rc::Rc<()>>,
}
impl Drop for LinkedThread {
    fn drop(&mut self) {
        if let Some(profile) = PROFILE.with(|profile| profile.borrow_mut().take()) {
            self.sink
                .0
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner)
                .get_or_insert_with(Profile::default)
                .merge_thread(self.key, &profile);
        }
    }
}
pub struct Session {
    #[cfg(test)]
    _test_session: std::sync::MutexGuard<'static, ()>,
}
impl Session {
    pub fn start() -> Self {
        // A helper process owns one session; concurrent unit tests must not
        // drain each other's completed store threads.
        #[cfg(test)]
        let test_session = TEST_SESSION
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        PROFILE.with(|profile| *profile.borrow_mut() = Some(Profile::default()));
        ACTIVE.store(true, Ordering::SeqCst);
        Self {
            #[cfg(test)]
            _test_session: test_session,
        }
    }
}
impl Drop for Session {
    fn drop(&mut self) {
        ACTIVE.store(false, Ordering::SeqCst);
        if let Some(mut profile) = PROFILE.with(|profile| profile.borrow_mut().take()) {
            // Counter merges commute; compact thread rows sort before output.
            let finished = FINISHED
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner)
                .take();
            if let Some(finished) = finished {
                profile.merge(&finished);
                profile.threads = finished.threads;
                profile.threads_omitted = finished.threads_omitted;
                profile.threads.sort_by_key(|(key, _)| *key);
            }
            // Diagnostic output cannot change the guest's result if its sink closes.
            let _ = profile.write_report(&mut std::io::stderr().lock());
        }
    }
}
/// Profiles one cluster store thread (keyed by instance slot and
/// incarnation) while the process session is open. A thread that already
/// carries the session profile keeps using it.
pub struct ThreadSession(Option<(u64, u64)>);
impl ThreadSession {
    pub fn start(slot: u64, incarnation: u64) -> Self {
        let installed = ACTIVE.load(Ordering::SeqCst)
            && PROFILE.with(|profile| {
                let mut profile = profile.borrow_mut();
                if profile.is_some() {
                    return false;
                }
                *profile = Some(Profile::default());
                true
            });
        Self(installed.then_some((slot, incarnation)))
    }
}
impl Drop for ThreadSession {
    fn drop(&mut self) {
        if let Some(key) = self.0
            && let Some(profile) = PROFILE.with(|profile| profile.borrow_mut().take())
        {
            FINISHED
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner)
                .get_or_insert_with(Profile::default)
                .merge_thread(key, &profile);
        }
    }
}
pub struct PhaseSpan {
    phase: Phase,
    timer: Option<Instant>,
    usage: Usage,
    previous: Option<Phase>,
    frame: Option<usize>,
}
impl PhaseSpan {
    #[must_use]
    pub fn new(phase: Phase) -> Self {
        let mut previous = None;
        let active = with(|profile| previous = profile.active_phase.replace(phase)).is_some();
        let usage = active
            .then(crate::metrics::diagnostic_process_usage)
            .flatten();
        let timer = active.then(Instant::now);
        let mut frame = None;
        if let Some(timer) = timer {
            with(|profile| {
                if phase == Phase::Execute {
                    profile.cursor = Some(timer);
                }
                if matches!(
                    phase,
                    Phase::CacheIdentity
                        | Phase::CacheRead
                        | Phase::EnvelopeVerify
                        | Phase::Deserialize
                        | Phase::CacheWrite
                ) {
                    frame = profile.push(Some(Owner::Cache), None, timer);
                }
            });
        }
        Self {
            phase,
            timer,
            usage,
            previous,
            frame,
        }
    }
}
impl Drop for PhaseSpan {
    fn drop(&mut self) {
        if let Some(timer) = self.timer {
            let end = Instant::now();
            let wall = nanos(end.saturating_duration_since(timer));
            let after = crate::metrics::diagnostic_process_usage();
            with(|profile| {
                if let Some(index) = self.frame {
                    profile.pop(index, end);
                }
                if self.phase == Phase::Execute {
                    if let Some(cursor) = profile.cursor.take() {
                        profile.guest_residual_nanos = profile
                            .guest_residual_nanos
                            .saturating_add(nanos(end.saturating_duration_since(cursor)));
                    }
                    if !profile.stack.is_empty() {
                        profile.span_errors = profile.span_errors.saturating_add(1);
                    }
                }
                profile.record_phase(self.phase, wall, self.usage, after);
                profile.active_phase = self.previous;
            });
        }
    }
}
/// The outer import boundary, entered by the linker trampoline before any
/// helper code: inside `Execute` it closes the guest gap since the previous
/// import and times the whole host side of this one.
pub struct ImportBoundary(Option<Instant>);
impl Default for ImportBoundary {
    fn default() -> Self {
        Self::new()
    }
}
impl ImportBoundary {
    #[must_use]
    pub fn new() -> Self {
        Self(with(|profile| {
            let now = Instant::now();
            profile.boundary_op = None;
            profile.outer_imports = profile.outer_imports.saturating_add(1);
            if profile.active_phase == Some(Phase::Execute)
                && let Some(cursor) = profile.cursor
            {
                profile.guest_residual_nanos = profile
                    .guest_residual_nanos
                    .saturating_add(nanos(now.saturating_duration_since(cursor)));
            }
            now
        }))
    }
}
impl Drop for ImportBoundary {
    fn drop(&mut self) {
        if let Some(start) = self.0 {
            with(|profile| {
                let now = Instant::now();
                let execute =
                    profile.active_phase == Some(Phase::Execute) && profile.cursor.is_some();
                // Only a framed import charges its glue; an unframed one
                // stays unattributed, so its whole wall misses reconciliation.
                if let Some((op_start, op_end)) = profile.boundary_op.take() {
                    let glue = nanos(op_start.saturating_duration_since(start))
                        .saturating_add(nanos(now.saturating_duration_since(op_end)));
                    let region = if execute {
                        Some(Phase::Execute)
                    } else {
                        profile.active_phase
                    };
                    let owner = Owner::Instrumentation as usize;
                    profile.owners[owner].record(region, glue, glue, glue);
                    profile.histogram(OPS + owner, glue);
                }
                if execute {
                    profile.outer_host_nanos = profile
                        .outer_host_nanos
                        .saturating_add(nanos(now.saturating_duration_since(start)));
                    profile.cursor = Some(now);
                }
            });
        }
    }
}
/// One host import's op frame: exact `module.op`, the owner a route tags with
/// `serve`, and the coarse v1 ABI kind.
pub struct AbiSpan(Option<usize>);
impl AbiSpan {
    #[must_use]
    pub fn new(op: crate::abi::Operation, local: bool) -> Self {
        let (module, name) = (op.module(), op.name());
        let kind = if !local && name != "proc_exit" {
            Abi::RemoteCallback
        } else {
            match (module, name) {
                (_, "clock_time_get") => Abi::Clock,
                ("gomad_wasm_v1", "decision") => Abi::Scheduler,
                (_, "poll_oneoff") => Abi::Select,
                ("gomad_wasm_v1", "observation") => Abi::Observation,
                ("gomad_wasm_v1", "idle") => Abi::Idle,
                _ => Abi::OtherLocal,
            }
        };
        Self(
            with(|profile| {
                if profile.op_frame.is_some() {
                    profile.abi_overlap_errors = profile.abi_overlap_errors.saturating_add(1);
                    return None;
                }
                let index = profile.push(None, Some(op as usize), Instant::now())?;
                profile.stack[index].legacy = Some(kind);
                profile.op_frame = Some(index);
                profile.framed_imports = profile.framed_imports.saturating_add(1);
                Some(index)
            })
            .flatten(),
        )
    }
}
impl Drop for AbiSpan {
    fn drop(&mut self) {
        if let Some(index) = self.0 {
            let end = Instant::now();
            with(|profile| {
                profile.op_frame = None;
                let Some((frame, wall, exclusive)) = profile.pop(index, end) else {
                    return;
                };
                profile.boundary_op = Some((frame.start, end));
                let op = frame.op.unwrap();
                profile.ops[op].record(frame.region, wall, exclusive, wall);
                profile.histogram(op, wall);
                match frame.owner {
                    Some(owner) => {
                        let served = &mut profile.served_by[op][owner as usize];
                        *served = served.saturating_add(1);
                    }
                    None => {
                        profile.unattributed_imports =
                            profile.unattributed_imports.saturating_add(1);
                    }
                }
                let stats = match frame.legacy {
                    Some(kind) => &mut profile.abi[kind as usize],
                    None => &mut profile.abi_excluded,
                };
                stats.count = stats.count.saturating_add(1);
                stats.wall_nanos = stats.wall_nanos.saturating_add(wall);
                if frame.region == Some(Phase::Instantiate) {
                    stats.instantiate_wall_nanos =
                        stats.instantiate_wall_nanos.saturating_add(wall);
                } else if frame.region == Some(Phase::Execute) {
                    stats.execute_wall_nanos = stats.execute_wall_nanos.saturating_add(wall);
                }
            });
        }
    }
}
fn with_op_frame(body: impl FnOnce(&mut Frame)) {
    with(|profile| {
        if let Some(index) = profile.op_frame
            && let Some(frame) = profile.stack.get_mut(index)
        {
            body(frame);
        }
    });
}
/// Tags the open import with the owner its route dispatches to.
pub fn serve(owner: Owner) {
    with_op_frame(|frame| frame.owner = Some(owner));
}
/// Keeps an import (debug exchange, sealed-inspection refusal) out of the
/// v1 `abi` buckets, which never timed it; `abi_excluded` counts it instead.
pub fn exclude_legacy() {
    with_op_frame(|frame| frame.legacy = None);
}
pub fn decision_kind(kind: u32) {
    with_op_frame(|frame| {
        if frame.legacy.is_some() {
            match kind {
                1 => frame.legacy = Some(Abi::Scheduler),
                2 => frame.legacy = Some(Abi::Select),
                _ => (),
            }
        }
    });
}
pub fn remote_callback() {
    with_op_frame(|frame| {
        if frame.legacy.is_some() {
            frame.legacy = Some(Abi::RemoteCallback);
        }
        frame.owner = Some(Owner::Remote);
    });
}
/// A model owner's span on the callee side; nested within an import it is
/// subtracted from the import's owner.
pub struct OwnerSpan(Option<usize>);
impl OwnerSpan {
    #[must_use]
    pub fn new(owner: Owner) -> Self {
        Self(with(|profile| profile.push(Some(owner), None, Instant::now())).flatten())
    }
}
impl Drop for OwnerSpan {
    fn drop(&mut self) {
        if let Some(index) = self.0 {
            let end = Instant::now();
            with(|profile| profile.pop(index, end));
        }
    }
}
pub struct WallSpan {
    kind: Nested,
    timer: Option<Instant>,
    frame: Option<usize>,
}
impl WallSpan {
    fn open(kind: Nested, enabled: impl FnOnce(&Profile) -> bool) -> Self {
        let mut frame = None;
        let timer = with(|profile| {
            if !enabled(profile) {
                return None;
            }
            let timer = Instant::now();
            frame = profile.push(Some(kind.owner()), None, timer);
            Some(timer)
        })
        .flatten();
        Self { kind, timer, frame }
    }
    #[must_use]
    pub fn new(kind: Nested) -> Self {
        Self::open(kind, |_| true)
    }
    #[must_use]
    pub fn remote_write() -> Self {
        Self::open(Nested::RemoteWrite, |profile| profile.remote_send)
    }
}
impl Drop for WallSpan {
    fn drop(&mut self) {
        if let Some(timer) = self.timer {
            let end = Instant::now();
            let wall = nanos(end.saturating_duration_since(timer));
            with(|profile| {
                if let Some(index) = self.frame {
                    profile.pop(index, end);
                }
                let stats = &mut profile.nested[self.kind as usize];
                stats.count = stats.count.saturating_add(1);
                stats.wall_nanos = stats.wall_nanos.saturating_add(wall);
            });
        }
    }
}
pub struct RemoteSend;
impl Default for RemoteSend {
    fn default() -> Self {
        Self::new()
    }
}
impl RemoteSend {
    #[must_use]
    pub fn new() -> Self {
        with(|profile| profile.remote_send = true);
        Self
    }
}
impl Drop for RemoteSend {
    fn drop(&mut self) {
        with(|profile| profile.remote_send = false);
    }
}
#[cfg(test)]
pub(crate) fn snapshot() -> Value {
    PROFILE.with(|profile| profile.borrow().as_ref().unwrap().report())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::abi::Operation;

    fn spin(nanos: u64) {
        let start = Instant::now();
        while start.elapsed() < Duration::from_nanos(nanos) {
            std::hint::spin_loop();
        }
    }
    fn number(value: &Value) -> u64 {
        value.as_u64().unwrap()
    }

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
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::ClockTimeGet, true);
                serve(Owner::Clock);
                let _hash = WallSpan::new(Nested::TranscriptHash);
                std::hint::black_box([7u8; 32]);
            }
            {
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::RuntimeDecision, true);
                serve(Owner::Scheduler);
                decision_kind(2);
                std::hint::black_box([8u8; 32]);
            }
            {
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::FdSync, true);
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
        let ops = report["abi_ops"].as_object().unwrap();
        assert_eq!(ops.len(), 3);
        assert_eq!(
            report["abi_ops"]["wasi_snapshot_preview1.fd_sync"]["served_by"],
            json!({"remote":1})
        );
        // Same-owner suppression: the remote wait stays in the remote frame.
        assert_eq!(report["owners"]["remote"]["count"], 1);
        assert_eq!(report["owners"]["transcript"]["count"], 1);
        assert_eq!(report["unattributed_imports"], 0);
        assert_eq!(report["unframed_imports"], 0);
        assert_eq!(report["span_errors"], 0);
    }

    /// Exclusive wall subtracts direct children only, owners partition the
    /// op wall exactly, and guest gaps plus outer host wall telescope to the
    /// execute wall.
    #[test]
    fn diagnostic_owner_attribution_reconciles_exactly() {
        let _session = Session::start();
        {
            let _execute = PhaseSpan::new(Phase::Execute);
            for _ in 0..3 {
                spin(20_000);
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::FdWrite, true);
                serve(Owner::Namespace);
                spin(5_000);
                {
                    let _volume = OwnerSpan::new(Owner::Volume);
                    spin(5_000);
                    let _nested = OwnerSpan::new(Owner::Volume);
                    spin(1_000);
                }
                let _flush = WallSpan::new(Nested::Flush);
                spin(2_000);
                let _encoding = WallSpan::new(Nested::Encoding);
                spin(2_000);
            }
            spin(20_000);
        }
        let report = snapshot();
        let op = &report["abi_ops"]["wasi_snapshot_preview1.fd_write"];
        assert_eq!(op["count"], 3);
        assert_eq!(op["served_by"], json!({"namespace":3}));
        let owners = report["owners"].as_object().unwrap();
        assert_eq!(
            owners.keys().cloned().collect::<Vec<_>>(),
            [
                "encoding",
                "evidence",
                "instrumentation",
                "namespace",
                "volume"
            ]
        );
        // Nested volume spans collapse into one volume frame per call.
        assert_eq!(report["owners"]["volume"]["count"], 3);
        assert_eq!(report["owners"]["instrumentation"]["count"], 3);
        let attributed: u64 = owners
            .values()
            .map(|owner| number(&owner["execute_exclusive_wall_nanos"]))
            .sum();
        let instrumentation =
            number(&report["owners"]["instrumentation"]["execute_exclusive_wall_nanos"]);
        assert_eq!(
            attributed - instrumentation,
            number(&op["execute_wall_nanos"])
        );
        let flush = &report["owners"]["evidence"];
        let encoding = &report["owners"]["encoding"];
        assert_eq!(
            number(&flush["exclusive_wall_nanos"]) + number(&encoding["wall_nanos"]),
            number(&flush["wall_nanos"])
        );
        assert!(number(&op["exclusive_wall_nanos"]) < number(&op["wall_nanos"]));
        let row = &report["reconciliation"];
        assert_eq!(number(&row["attributed_host_wall_nanos"]), attributed);
        assert_eq!(
            number(&row["execute_wall_nanos"]),
            number(&row["guest_residual_wall_nanos"]) + number(&row["outer_host_wall_nanos"])
        );
        assert!(number(&row["guest_residual_wall_nanos"]) >= 80_000);
        // Owners, instrumentation included, partition the boundary wall.
        assert_eq!(attributed, number(&row["outer_host_wall_nanos"]));
        assert_eq!(row["difference_nanos"], 0);
        assert_eq!(row["tolerance_nanos"], 50_000);
        assert_eq!(report["span_errors"], 0);
        assert_eq!(report["unattributed_imports"], 0);
        let distribution = [
            number(&op["min_nanos"]),
            number(&op["median_nanos"]),
            number(&op["p99_nanos"]),
            number(&op["max_nanos"]),
        ];
        assert!(distribution.is_sorted(), "{distribution:?}");
    }

    #[test]
    fn diagnostic_attribution_errors_are_counted_not_hidden() {
        let _session = Session::start();
        {
            let _execute = PhaseSpan::new(Phase::Execute);
            {
                // An owner span with no import open during Execute.
                let _stray = OwnerSpan::new(Owner::Clock);
            }
            {
                // An import nobody tagged.
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::FdRead, true);
            }
            {
                // A trampoline crossing with no op frame.
                let _boundary = ImportBoundary::new();
            }
            {
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::SchedYield, true);
                serve(Owner::Scheduler);
                let _overlap = AbiSpan::new(Operation::SchedYield, true);
                // Out of LIFO order: the outer frame closes first.
                let outer = OwnerSpan::new(Owner::Clock);
                let inner = OwnerSpan::new(Owner::Encoding);
                drop(outer);
                drop(inner);
            }
        }
        let report = snapshot();
        assert_eq!(report["unattributed_imports"], 1);
        assert_eq!(report["unframed_imports"], 1);
        assert_eq!(report["abi_overlap_errors"], 1);
        assert!(number(&report["span_errors"]) >= 2);
        assert_eq!(
            report["abi_ops"]["wasi_snapshot_preview1.fd_read"]["served_by"],
            json!({})
        );
    }

    /// An import whose op frame is missing stays unattributed: no
    /// instrumentation is charged for it, so its wall fails reconciliation.
    #[test]
    fn diagnostic_missing_op_frame_fails_reconciliation() {
        let _session = Session::start();
        {
            let _execute = PhaseSpan::new(Phase::Execute);
            {
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::ClockTimeGet, true);
                serve(Owner::Clock);
            }
            {
                let _boundary = ImportBoundary::new();
                spin(200_000);
            }
        }
        let report = snapshot();
        let row = &report["reconciliation"];
        assert_eq!(report["unframed_imports"], 1);
        assert_eq!(report["owners"]["instrumentation"]["count"], 1);
        assert!(number(&row["difference_nanos"]) >= 200_000);
        assert!(number(&row["difference_nanos"]) > number(&row["tolerance_nanos"]));
    }

    #[test]
    fn diagnostic_excluded_imports_stay_out_of_v1_buckets() {
        let _session = Session::start();
        {
            let _execute = PhaseSpan::new(Phase::Execute);
            let _boundary = ImportBoundary::new();
            let _abi = AbiSpan::new(Operation::DebugExchange, true);
            serve(Owner::Inspection);
            exclude_legacy();
        }
        let report = snapshot();
        let legacy: u64 = report["abi"]
            .as_object()
            .unwrap()
            .values()
            .map(|bucket| number(&bucket["count"]))
            .sum();
        assert_eq!(legacy, 0);
        assert_eq!(report["abi_excluded"]["count"], 1);
        assert_eq!(
            number(&report["abi_excluded"]["execute_wall_nanos"]),
            number(&report["abi_ops"]["gomad_debug_v1.exchange"]["execute_wall_nanos"])
        );
        assert_eq!(report["owners"]["inspection"]["count"], 1);
    }

    #[test]
    fn diagnostic_histogram_buckets_bound_their_values() {
        for value in [
            0,
            1,
            15,
            16,
            17,
            31,
            32,
            1000,
            123_456_789,
            1 << 40,
            (1 << 41) - 1,
        ] {
            let index = bucket(value);
            assert!(bucket_upper(index) >= value, "{value}");
            if index > 0 {
                assert!(bucket_upper(index - 1) < value, "{value}");
            }
            if value >= 16 {
                let error = u128::from(bucket_upper(index) - value) * 16;
                assert!(error < u128::from(value), "{value}");
            }
        }
        assert_eq!(bucket(u64::MAX), BUCKETS - 1);
        let mut series = Series::default();
        let mut histogram = vec![0; BUCKETS];
        for value in 1..=100u64 {
            series.record(
                Some(Phase::Execute),
                value * 1000,
                value * 1000,
                value * 1000,
            );
            histogram[bucket(value * 1000)] += 1;
        }
        let report = series.report(&histogram);
        assert_eq!(report["min_nanos"], 1000);
        assert_eq!(report["max_nanos"], 100_000);
        let median = number(&report["median_nanos"]);
        assert!((50_000..=53_125).contains(&median), "{median}");
        let p99 = number(&report["p99_nanos"]);
        assert!((99_000..=100_000).contains(&p99), "{p99}");
    }

    #[test]
    fn diagnostic_thread_profiles_merge_into_the_session() {
        let session = Session::start();
        let threads: Vec<_> = (0..3u64)
            .map(|slot| {
                std::thread::spawn(move || {
                    let _thread = ThreadSession::start(slot, 0x1_000f_eed5);
                    let _execute = PhaseSpan::new(Phase::Execute);
                    let _boundary = ImportBoundary::new();
                    let _abi = AbiSpan::new(Operation::RuntimeIdle, true);
                    serve(Owner::SchedulerHandoff);
                })
            })
            .collect();
        for thread in threads {
            thread.join().unwrap();
        }
        let finished = FINISHED.lock().unwrap().take().unwrap();
        assert_eq!(finished.threads.len(), 3);
        PROFILE.with(|profile| {
            let mut profile = profile.borrow_mut();
            let profile = profile.as_mut().unwrap();
            profile.merge(&finished);
            profile.threads = finished.threads;
        });
        let report = snapshot();
        assert_eq!(report["phases"]["execute"]["count"], 3);
        assert_eq!(report["owners"]["scheduler_handoff"]["count"], 3);
        let threads = report["threads"].as_array().unwrap();
        assert_eq!(threads.len(), 3);
        for thread in threads {
            assert_eq!(thread["incarnation"], 0x1_000f_eed5_u64);
            assert_eq!(thread["difference_nanos"], 0);
        }
        let row = &report["reconciliation"];
        assert_eq!(
            number(&row["execute_wall_nanos"]),
            number(&row["guest_residual_wall_nanos"]) + number(&row["outer_host_wall_nanos"])
        );
        drop(session);
    }

    #[test]
    fn diagnostic_completed_profiles_keep_one_histogram_and_bounded_rows() {
        let session = Session::start();
        for incarnation in 0..80 {
            std::thread::spawn(move || {
                let _thread = ThreadSession::start(1, incarnation);
                let _execute = PhaseSpan::new(Phase::Execute);
                let _boundary = ImportBoundary::new();
                let _abi = AbiSpan::new(Operation::RuntimeIdle, true);
                serve(Owner::SchedulerHandoff);
            })
            .join()
            .unwrap();
        }
        let finished = FINISHED.lock().unwrap();
        let profile = finished.as_ref().unwrap();
        assert_eq!(profile.threads.len(), MAX_THREAD_ROWS);
        assert_eq!(profile.threads_omitted, 16);
        assert_eq!(profile.histograms.len(), (OPS + OWNERS) * BUCKETS);
        assert_eq!(profile.ops[Operation::RuntimeIdle as usize].count, 80);
        drop(finished);
        drop(session);
    }

    // CLI runs cannot reach saturated counter widths in every bounded series.
    #[test]
    fn linked_report_capacity_retains_drop_counters() {
        let session = LinkedProfile::start().unwrap();
        with(|profile| {
            let mut series = Series::default();
            series.record(Some(Phase::Execute), u64::MAX, u64::MAX, u64::MAX);
            series.count = u64::MAX;
            profile.ops.fill(series);
            profile.owners.fill(series);
            profile.served_by.fill([u64::MAX; OWNERS]);
            profile.threads = (0..MAX_THREAD_ROWS)
                .map(|slot| ((slot as u64, 1), [u64::MAX; 6]))
                .collect();
        });
        let report = session.finish().unwrap();
        assert!(serde_json::to_vec(&report).unwrap().len() < REPORT_LIMIT);
        assert!(number(&report["report_rows_dropped"]) > 0);
        assert!(number(&report["threads_omitted"]) > 0);
        assert!(LinkedProfile::start().unwrap().finish().is_ok());
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
        assert!(output.len() < 48 * 1024);
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
        assert_eq!(decoded["schema"], "gomad.wasm-profile/v2");
    }
}

#[must_use]
pub fn operation_names() -> Vec<String> {
    crate::abi::Operation::ALL
        .iter()
        .map(|op| format!("{}.{}", op.module(), op.name()))
        .collect()
}
#[must_use]
pub fn owner_names() -> &'static [&'static str] {
    &OWNER_NAMES
}
