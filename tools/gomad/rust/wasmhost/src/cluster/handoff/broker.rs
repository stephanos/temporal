//! In-process broker (fn-178.7): the Go broker's serialized scheduler
//! (`wasi/broker.go`, `broker_process.go`) over the Rust simulation clock
//! (fn-178.5), the Rust World (fn-178.3) and the shared Rust network owner
//! (fn-178.12). One instance holds the continuation at a time; the next is the
//! first runnable instance in sorted id order; when every live instance is idle
//! and the clock has answered them all, the network advances to the current
//! time and they all become runnable again.
//!
//! World-visible operations are `runtime_idle` and `runtime_net_stream`. Each
//! registers a World request for its incarnation, becomes ready at the clock's
//! current time and must quiesce to exactly that one delivery.
//!
//! The Go broker also accounts every helper callback with the clock's progress
//! credits (`Accepted`, `Delivered`, arrivals). Here instances run serialized in
//! one process and handle every non-visible call locally, so a running instance
//! is simply not idle; settling needs every live participant idle, as the
//! credits enforce in Go, and idles carry no arrivals.
use crate::clock::{self, Clock, Quiesced, TimeRequest, TimeResponse};
use crate::cluster::network::{Endpoint, Failure, Shared};
use crate::cluster::transport::{SharedInput, SharedOutput, lock};
use crate::world;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, VecDeque};
use std::sync::mpsc;
use std::sync::{Arc, Condvar, Mutex};
use std::time::Instant;

#[derive(Clone, Default, Serialize)]
pub struct Metrics {
    pub handoffs: u64,
    pub handoff_nanos: u64,
    pub handoff_samples: u64,
    pub active_nanos: u64,
    pub overlap_nanos: u64,
    pub operation_samples: u64,
    pub runnable_samples: u64,
    pub guest_nanos: u64,
    pub host_nanos: u64,
}

pub enum Command {
    Restart {
        index: usize,
        incarnation: u64,
    },
    Revoke {
        index: usize,
        incarnation: u64,
        kind: &'static str,
    },
    Link {
        action: String,
        from: String,
        to: String,
        delay: u64,
        symmetric: bool,
    },
}

const INITIAL_TIME: i64 = world::INITIAL_TIME;
/// Go `math.MaxInt64 - world.InitialTime`: the largest relative deadline.
const MAXIMUM_RELATIVE: u64 = (i64::MAX - INITIAL_TIME) as u64;

/// The cluster `broker` object: Go `BrokerConfig`'s World fields.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub seed: u64,
    pub limits: WorldLimits,
    /// Stop at every clock advance for live control frames (link changes).
    #[serde(default)]
    pub pause_on_settle: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct WorldLimits {
    pub max_requests: u64,
    pub max_events: u64,
    pub max_queued_events: u64,
    pub max_transitions: u64,
    pub max_payload_bytes: u64,
    pub max_string_bytes: u32,
}

impl Config {
    fn world(&self) -> world::Config {
        let limits = &self.limits;
        world::Config {
            seed: self.seed,
            limits: world::Limits {
                max_requests: limits.max_requests,
                max_events: limits.max_events,
                max_queued_events: limits.max_queued_events,
                max_transitions: limits.max_transitions,
                max_payload_bytes: limits.max_payload_bytes,
                max_string_bytes: limits.max_string_bytes,
            },
        }
    }

    /// Go `world.ValidateCompressedSnapshotConfiguration` with the compressed
    /// profile, as `NewBroker` checks before anything starts.
    ///
    /// # Errors
    /// Refuses bounds outside the existing compressed World configuration.
    pub fn validate(&self) -> Result<(), String> {
        let limits = &self.limits;
        for (dimension, actual, maximum) in [
            ("requests", limits.max_requests, 2 << 20),
            ("events", limits.max_events, 2 << 20),
            ("queued-events", limits.max_queued_events, 250_000),
            ("transitions", limits.max_transitions, 8 << 20),
            ("payload-bytes", limits.max_payload_bytes, 64 << 20),
        ] {
            if actual > maximum {
                return Err(format!(
                    "World capacity exhausted: compressed-snapshot-config-{dimension} limit={maximum} used=0 delta={actual}"
                ));
            }
        }
        world::Model::new(self.world())
            .map(|_| ())
            .map_err(|error| error.to_string())
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum State {
    Dormant,
    Runnable,
    Active,
    Idle,
    Finished,
}

struct Session {
    id: String,
    incarnation: u64,
    slot: u64,
    revoked: Option<&'static str>,
    output_revoked: bool,
    hold: Option<clock::Handle>,
    marker: Vec<u8>,
    boundary: &'static str,
    observed: Option<mpsc::SyncSender<()>>,
    output: Vec<u8>,
    key: String,
    handle: clock::Handle,
    state: State,
    resume: Option<Result<TimeResponse, String>>,
    generation: u64,
    pending: u64,
}

struct Inner {
    #[cfg(feature = "determinism-fixture")]
    fixture_fault: crate::determinism_fixture::Fault,
    commands: VecDeque<(Command, mpsc::SyncSender<Result<(), Failure>>)>,
    metrics: Metrics,
    observed_at: Instant,
    granted_at: Option<Instant>,
    barrier: BTreeMap<String, bool>,
    barrier_marker: Vec<u8>,
    sessions: Vec<Session>,
    active: Option<usize>,
    manual: bool,
    driver: Option<(usize, u64)>,
    aborted: Option<(&'static str, String)>,
    terminal_turn: Option<usize>,
    /// The abort when the broker raised it itself (stall, pause, World or
    /// clock refusal) rather than an instance reporting its own failure.
    raised: Option<(&'static str, String)>,
    clock: Clock,
    world: world::Model,
    /// The generated encoding of a prefix of the append-only history, the
    /// wire's final section, reused by every later revision.
    #[cfg(test)]
    inspection_transitions: InspectionTransitions,
    inspection_world: Option<(u64, Option<[u8; 32]>, [u8; 32])>,
    network: Option<Arc<Mutex<Shared>>>,
    pause: Option<(SharedInput, SharedOutput)>,
    paused_at: u64,
}

pub struct Broker {
    inner: Mutex<Inner>,
    wake: Vec<Condvar>,
    idle: Condvar,
    pub inspection: Arc<Mutex<crate::model::inspection::Recorder>>,
    #[cfg(test)]
    pub world_materializations: std::sync::atomic::AtomicU64,
}

/// Go `invalid`: an instance-level `WASI invalid: <dimension>` failure.
fn invalid(dimension: &'static str) -> Failure {
    Failure::Invalid(dimension)
}

/// The control frames a paused cluster accepts (fn-178.1 draft).
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Control {
    #[serde(rename = "type")]
    kind: String,
    #[serde(default)]
    command: String,
    #[serde(default)]
    action: String,
    #[serde(default)]
    from: String,
    #[serde(default)]
    to: String,
    #[serde(default)]
    delay_nanos: u64,
    #[serde(default)]
    symmetric: bool,
}

impl Broker {
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn event_context(inner: &Inner, me: usize) -> crate::event_order::Context {
        let session = &inner.sessions[me];
        crate::event_order::owner(
            &session.id,
            session.incarnation,
            inner.clock.current() - INITIAL_TIME,
        )
    }

    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn event(inner: &Inner, me: usize, action: &str) {
        crate::event_order::emit(
            &Self::event_context(inner, me),
            if matches!(action, "grant" | "acquire" | "resume") {
                "world-order"
            } else {
                "model-transition"
            },
            inner.sessions[me].generation,
            serde_json::json!({"action": action}),
        );
    }

    /// `ids` are the instances in sorted order, all at incarnation 1.
    pub fn new(
        config: &Config,
        ids: &[String],
        network: Option<Arc<Mutex<Shared>>>,
        transport: Option<(SharedInput, SharedOutput)>,
    ) -> Result<Self, String> {
        let world = world::Model::new(config.world()).map_err(|error| error.to_string())?;
        let mut clock = Clock::broker();
        let mut sessions = Vec::new();
        for id in ids {
            let key = format!("{id}/1");
            let handle = clock.register(&key)?;
            sessions.push(Session {
                id: id.clone(),
                incarnation: 1,
                slot: sessions.len() as u64 + 1,
                revoked: None,
                output_revoked: false,
                hold: None,
                marker: Vec::new(),
                boundary: "crash",
                observed: None,
                output: Vec::new(),
                key,
                handle,
                state: State::Runnable,
                resume: None,
                generation: 0,
                pending: 0,
            });
        }
        Ok(Self {
            inner: Mutex::new(Inner {
                #[cfg(feature = "determinism-fixture")]
                fixture_fault: crate::determinism_fixture::Fault::for_broker(),
                commands: VecDeque::new(),
                metrics: Metrics::default(),
                observed_at: Instant::now(),
                granted_at: None,
                barrier: BTreeMap::new(),
                barrier_marker: Vec::new(),
                sessions,
                active: None,
                manual: false,
                driver: None,
                aborted: None,
                terminal_turn: None,
                raised: None,
                clock,
                world,
                #[cfg(test)]
                inspection_transitions: InspectionTransitions::default(),
                inspection_world: None,
                network,
                pause: if config.pause_on_settle {
                    transport
                } else {
                    None
                },
                paused_at: 1,
            }),
            wake: ids.iter().map(|_| Condvar::new()).collect(),
            idle: Condvar::new(),
            inspection: Arc::new(Mutex::new(crate::model::inspection::Recorder::default())),
            #[cfg(test)]
            world_materializations: std::sync::atomic::AtomicU64::new(0),
        })
    }

    pub(crate) fn incarnation_slot(&self, me: usize) -> (u64, u64) {
        let inner = lock(&self.inner);
        (inner.sessions[me].incarnation, inner.sessions[me].slot)
    }

    /// Go `Broker.Current`: relative virtual time.
    pub fn current(&self) -> u64 {
        (lock(&self.inner).clock.current() - INITIAL_TIME).cast_unsigned()
    }

    pub fn check(&self, me: usize, incarnation: u64) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        Self::observe(&mut inner);
        self.commands(&mut inner);
        inner.metrics.operation_samples += 1;
        if inner
            .sessions
            .iter()
            .filter(|session| matches!(session.state, State::Runnable | State::Active))
            .count()
            > 1
        {
            inner.metrics.runnable_samples += 1;
        }
        Self::check_locked(&inner, me, incarnation)
    }

    fn observe(inner: &mut Inner) {
        let elapsed = crate::metrics::elapsed(inner.observed_at);
        inner.observed_at = Instant::now();
        if inner.active.is_some() {
            inner.metrics.active_nanos += elapsed;
            if inner
                .sessions
                .iter()
                .filter(|session| matches!(session.state, State::Runnable | State::Active))
                .count()
                > 1
            {
                inner.metrics.overlap_nanos += elapsed;
            }
        }
    }

    pub fn metrics(&self) -> Metrics {
        let mut inner = lock(&self.inner);
        Self::observe(&mut inner);
        inner.metrics.clone()
    }

    pub fn command(&self, command: Command) -> Result<(), Failure> {
        let (send, receive) = mpsc::sync_channel(1);
        {
            let mut inner = lock(&self.inner);
            if let Some((_, message)) = &inner.aborted {
                return Err(Failure::Infrastructure(message.clone()));
            }
            inner.commands.push_back((command, send));
            for wake in &self.wake {
                wake.notify_all();
            }
            if inner.active.is_none() {
                self.commands(&mut inner);
                if inner.manual {
                    self.schedule(&mut inner);
                }
            }
        }
        receive
            .recv()
            .map_err(|_| Failure::Infrastructure("cluster command abandoned".into()))?
    }

    fn commands(&self, inner: &mut Inner) {
        while let Some((command, reply)) = inner.commands.pop_front() {
            if matches!(command, Command::Restart { .. }) && inner.active.is_some() {
                inner.commands.push_front((command, reply));
                break;
            }
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            let _event_scope = crate::event_order::enter(crate::event_order::owner(
                "controller",
                0,
                inner.clock.current() - INITIAL_TIME,
            ));
            let result = match command {
                Command::Restart { index, incarnation } => {
                    self.restart_locked(inner, index, incarnation).map(|_| ())
                }
                Command::Revoke {
                    index,
                    incarnation,
                    kind,
                } => Self::check_locked(inner, index, incarnation)
                    .and_then(|()| self.revoke_locked(inner, index, kind)),
                Command::Link {
                    action,
                    from,
                    to,
                    delay,
                    symmetric,
                } => match &inner.network {
                    Some(network) => {
                        lock(network).change_link(&action, &from, &to, delay, symmetric)
                    }
                    None => Err(invalid("cluster link without network")),
                },
            };
            if result.is_ok() {
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                crate::event_order::emit(
                    &crate::event_order::owner(
                        "controller",
                        0,
                        inner.clock.current() - INITIAL_TIME,
                    ),
                    "model-transition",
                    0,
                    serde_json::json!({"action": "controller_effect"}),
                );
                lock(&self.inspection).event(
                    "controller_effect",
                    &crate::model::inspection::Handle {
                        node: "controller".into(),
                        incarnation: 0,
                    },
                );
            }
            let _ = reply.send(result);
        }
    }

    fn check_locked(inner: &Inner, me: usize, incarnation: u64) -> Result<(), Failure> {
        let session = &inner.sessions[me];
        if session.incarnation != incarnation || session.state == State::Finished {
            return Err(invalid("stale broker incarnation"));
        }
        if let Some(kind) = session.revoked {
            return Err(Failure::Boundary(kind, "broker incarnation revoked"));
        }
        if matches!(&inner.aborted, Some(("trap" | "capacity", _))) {
            return Err(Failure::Boundary("stop", "cluster failed"));
        }
        Ok(())
    }

    pub fn boundary(
        &self,
        me: usize,
        incarnation: u64,
        kind: &'static str,
        marker: Vec<u8>,
    ) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        Self::check_locked(&inner, me, incarnation)?;
        if marker.is_empty()
            || marker.len() > 4096
            || !matches!(kind, "crash" | "stop")
            || !inner.sessions[me].marker.is_empty()
        {
            return Err(invalid("broker crash marker"));
        }
        inner.sessions[me].marker = marker;
        inner.sessions[me].boundary = kind;
        Ok(())
    }

    pub fn observe_output(
        &self,
        me: usize,
        incarnation: u64,
        marker: Vec<u8>,
        observed: mpsc::SyncSender<()>,
    ) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        Self::check_locked(&inner, me, incarnation)?;
        if marker.is_empty() || marker.len() > 4096 || !inner.sessions[me].marker.is_empty() {
            return Err(invalid("broker observation marker"));
        }
        let session = &mut inner.sessions[me];
        session.marker = marker;
        session.boundary = "observe";
        session.observed = Some(observed);
        Ok(())
    }

    pub fn barrier(&self, ids: &[String], marker: Vec<u8>) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        if ids.len() < 2
            || marker.is_empty()
            || marker.len() > 4096
            || !inner.barrier_marker.is_empty()
        {
            return Err(invalid("broker output crash barrier"));
        }
        let mut barrier = BTreeMap::new();
        for id in ids {
            if !inner
                .sessions
                .iter()
                .any(|session| &session.id == id && session.incarnation == 1)
                || barrier.insert(id.clone(), false).is_some()
            {
                return Err(invalid("broker crash barrier nodes"));
            }
        }
        inner.barrier = barrier;
        inner.barrier_marker = marker;
        Ok(())
    }

    pub fn output(&self, me: usize, bytes: &[u8]) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        let incarnation = inner.sessions[me].incarnation;
        Self::check_locked(&inner, me, incarnation)?;
        let barrier_marker = inner.barrier_marker.clone();
        let session = &mut inner.sessions[me];
        let maximum = session.marker.len().max(barrier_marker.len());
        if maximum == 0 {
            return Ok(());
        }
        let contains = |data: &[u8], marker: &[u8]| {
            !marker.is_empty() && data.windows(marker.len()).any(|part| part == marker)
        };
        session.output.extend_from_slice(bytes);
        let own = contains(&session.output, &session.marker).then_some(session.boundary);
        let seen = contains(&session.output, &barrier_marker);
        let id = session.id.clone();
        if session.output.len() >= maximum {
            session.output.drain(..session.output.len() - (maximum - 1));
        }
        if seen && let Some(value) = inner.barrier.get_mut(&id) {
            *value = true;
        }
        let all = !inner.barrier.is_empty() && inner.barrier.values().all(|seen| *seen);
        if own == Some("observe") {
            inner.driver = None;
            inner.sessions[me].marker.clear();
            let observed = inner.sessions[me]
                .observed
                .take()
                .ok_or_else(|| invalid("broker observation receiver"))?;
            observed
                .send(())
                .map_err(|_| invalid("broker observation abandoned"))?;
            loop {
                self.commands(&mut inner);
                Self::check_locked(&inner, me, incarnation)?;
                if let Some((_, message)) = &inner.aborted {
                    return Err(Failure::Infrastructure(message.clone()));
                }
                inner = self.wake[me]
                    .wait(inner)
                    .unwrap_or_else(std::sync::PoisonError::into_inner);
            }
        }
        if own == Some("crash") {
            self.revoke_locked(&mut inner, me, "crash")?;
            inner.sessions[me].output_revoked = true;
            return Err(Failure::Boundary("crash", "broker output trigger"));
        } else if all {
            let nodes: Vec<_> = inner
                .sessions
                .iter()
                .enumerate()
                .filter(|(_, session)| inner.barrier.contains_key(&session.id))
                .map(|(index, _)| index)
                .collect();
            inner.barrier.clear();
            for node in nodes {
                self.revoke_locked(&mut inner, node, "crash")?;
                inner.sessions[node].output_revoked = true;
            }
            return Err(Failure::Boundary("crash", "broker output barrier"));
        } else if own == Some("stop") {
            self.revoke_locked(&mut inner, me, "stop")?;
            inner.sessions[me].output_revoked = true;
            return Err(Failure::Boundary("stop", "broker output trigger"));
        }
        Self::check_locked(&inner, me, incarnation)
    }

    pub fn finished(&self, me: usize) -> bool {
        lock(&self.inner).sessions[me].state == State::Finished
    }

    pub(crate) fn output_terminal(&self, me: usize, incarnation: u64) -> Option<&'static str> {
        let inner = lock(&self.inner);
        inner
            .sessions
            .get(me)
            .filter(|session| session.incarnation == incarnation && session.output_revoked)
            .and_then(|session| session.revoked)
    }

    fn revoke_locked(
        &self,
        inner: &mut Inner,
        me: usize,
        kind: &'static str,
    ) -> Result<(), Failure> {
        let session = &inner.sessions[me];
        Self::check_locked(inner, me, session.incarnation)?;
        let hold = inner
            .clock
            .register(&format!("restart/{}/{}", session.id, session.incarnation))
            .map_err(Failure::Infrastructure)?;
        let session = &mut inner.sessions[me];
        session.hold = Some(hold);
        inner
            .clock
            .apply(clock::Event::Remove {
                participant: Some(session.handle),
            })
            .map_err(Failure::Infrastructure)?;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(crate::event_order::owner(
            &session.id,
            session.incarnation,
            inner.clock.current() - INITIAL_TIME,
        ));
        if let Some(network) = &inner.network {
            lock(network)
                .revoke(&session.id, session.incarnation, kind == "stop")
                .map_err(Failure::Infrastructure)?;
        }
        let pending = std::mem::take(&mut session.pending);
        if pending != 0 {
            inner.world.cancel(pending).map_err(world_failure)?;
        }
        session.revoked = Some(kind);
        lock(&self.inspection).event(
            kind,
            &crate::model::inspection::Handle {
                node: session.id.clone(),
                incarnation: session.incarnation,
            },
        );
        if inner.manual && session.state == State::Idle {
            session.state = State::Runnable;
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        Self::event(inner, me, kind);
        Self::deliver_answers(inner);
        self.wake[me].notify_all();
        Ok(())
    }

    pub fn restart(&self, me: usize, incarnation: u64) -> Result<(u64, u64), Failure> {
        self.command(Command::Restart {
            index: me,
            incarnation,
        })?;
        let mut inner = lock(&self.inner);
        let slot = inner.sessions[me].slot;
        self.schedule(&mut inner);
        Ok((incarnation, slot))
    }

    fn restart_locked(
        &self,
        inner: &mut Inner,
        me: usize,
        incarnation: u64,
    ) -> Result<u64, Failure> {
        let old = &inner.sessions[me];
        if old.state != State::Finished || old.incarnation.checked_add(1) != Some(incarnation) {
            return Err(invalid("broker restart lifecycle"));
        }
        let slot = old.slot + inner.sessions.len() as u64;
        if slot >= 1 << 16 {
            return Err(Failure::Capacity("broker incarnation slots"));
        }
        let id = old.id.clone();
        let key = format!("{id}/{incarnation}");
        let handle = inner
            .clock
            .register(&key)
            .map_err(Failure::Infrastructure)?;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(crate::event_order::owner(
            &id,
            incarnation,
            inner.clock.current() - INITIAL_TIME,
        ));
        if let Some(network) = &inner.network {
            lock(network)
                .register(&id, incarnation)
                .map_err(Failure::Infrastructure)?;
        }
        inner.sessions[me] = Session {
            id,
            incarnation,
            slot,
            key,
            handle,
            state: State::Runnable,
            revoked: None,
            output_revoked: false,
            hold: None,
            marker: Vec::new(),
            boundary: "crash",
            observed: None,
            output: Vec::new(),
            resume: None,
            generation: 0,
            pending: 0,
        };
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        Self::event(inner, me, "restart");
        lock(&self.inspection).event(
            "restart",
            &crate::model::inspection::Handle {
                node: inner.sessions[me].id.clone(),
                incarnation,
            },
        );
        Ok(slot)
    }

    pub fn start(&self) {
        let mut inner = lock(&self.inner);
        self.schedule(&mut inner);
    }

    pub fn defer_starts(&self) {
        let mut inner = lock(&self.inner);
        inner.manual = true;
        for session in &mut inner.sessions {
            session.state = State::Dormant;
        }
    }

    pub fn run(&self, me: usize, incarnation: u64) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        if inner.manual
            && inner.sessions[me].incarnation == incarnation
            && inner.sessions[me].state == State::Finished
        {
            return Ok(());
        }
        if !inner.manual
            || inner.driver.is_some()
            || inner.sessions[me].incarnation != incarnation
            || matches!(inner.sessions[me].state, State::Dormant | State::Finished)
        {
            return Err(invalid("broker driver lifecycle"));
        }
        inner.driver = Some((me, incarnation));
        self.schedule(&mut inner);
        Ok(())
    }

    /// Whether Go's controls drive the session.
    pub fn manual(&self) -> bool {
        lock(&self.inner).manual
    }

    pub fn activate(&self, me: usize) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        Self::observe(&mut inner);
        if inner.sessions[me].state != State::Dormant {
            return Err(invalid("broker start lifecycle"));
        }
        inner.sessions[me].state = State::Runnable;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        Self::event(&inner, me, "activate");
        self.schedule(&mut inner);
        Ok(())
    }

    pub fn acquire(&self, me: usize) -> bool {
        let mut inner = lock(&self.inner);
        while inner.active != Some(me)
            && inner.aborted.is_none()
            && (inner.manual || inner.sessions[me].revoked.is_none())
        {
            inner = self.wake[me]
                .wait(inner)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
        if inner.active == Some(me)
            && inner.sessions[me].revoked.is_none()
            && let Some(granted) = inner.granted_at.take()
        {
            inner.metrics.handoff_nanos += crate::metrics::elapsed(granted);
            inner.metrics.handoff_samples += 1;
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if inner.aborted.is_none()
            && inner.active == Some(me)
            && inner.sessions[me].revoked.is_none()
        {
            Self::event(&inner, me, "acquire");
        }
        inner.aborted.is_none()
    }

    pub fn abort(&self, termination: &'static str, message: String) {
        let mut inner = lock(&self.inner);
        if matches!(
            termination,
            "inspected_prefix" | "instrumentation_divergence" | "debug_error" | "probe_fuel_limit"
        ) && inner.aborted.is_none()
        {
            inner.raised = Some((termination, message.clone()));
        }
        self.abort_locked(&mut inner, termination, message);
    }

    pub fn abort_terminal(&self, me: usize, termination: &'static str, message: String) {
        let mut inner = lock(&self.inner);
        if inner.terminal_turn.is_none()
            && (matches!(&inner.aborted, Some(("trap" | "capacity", _)))
                || inner.aborted.is_none() && matches!(termination, "trap" | "capacity"))
        {
            inner.terminal_turn = Some(me);
        }
        self.abort_locked(&mut inner, termination, message);
    }

    pub fn wait_terminal(&self, me: usize) {
        let mut inner = lock(&self.inner);
        while matches!(&inner.aborted, Some(("trap" | "capacity", _)))
            && inner.terminal_turn != Some(me)
            && !matches!(inner.sessions[me].state, State::Dormant | State::Finished)
        {
            inner = self.wake[me]
                .wait(inner)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
    }

    /// An abort the broker raises itself: its classification is the run's.
    fn raise(&self, inner: &mut Inner, termination: &'static str, message: String) {
        if inner.aborted.is_none() {
            inner.raised = Some((termination, message.clone()));
        }
        self.abort_locked(inner, termination, message);
    }

    fn abort_locked(&self, inner: &mut Inner, termination: &'static str, message: String) {
        Self::observe(inner);
        if inner.aborted.is_none() {
            if inner.terminal_turn.is_none() && matches!(termination, "trap" | "capacity") {
                inner.terminal_turn = inner.active;
            }
            inner.aborted = Some((termination, message));
        }
        inner.active = None;
        for wake in &self.wake {
            wake.notify_all();
        }
        self.idle.notify_all();
        for (_, reply) in inner.commands.drain(..) {
            let _ = reply.send(Err(Failure::Infrastructure("cluster aborted".into())));
        }
    }

    /// The whole-run failure the broker itself raised, when it raised the
    /// first abort; it takes precedence over what the aborted instances report.
    pub fn failure(&self) -> Option<(&'static str, String)> {
        lock(&self.inner).raised.clone()
    }

    pub fn terminal_failed(&self) -> bool {
        matches!(&lock(&self.inner).aborted, Some(("trap" | "capacity", _)))
    }

    pub fn inspection_stopped(&self) -> bool {
        self.failure().is_some_and(|(kind, _)| {
            matches!(
                kind,
                "inspected_prefix"
                    | "instrumentation_divergence"
                    | "debug_error"
                    | "probe_fuel_limit"
            )
        })
    }

    /// Blocks the coordinator until every instance finished (true) or the run
    /// was aborted (false).
    pub fn wait(&self) -> bool {
        let mut inner = lock(&self.inner);
        loop {
            if inner.aborted.is_some() {
                return false;
            }
            if inner
                .sessions
                .iter()
                .all(|session| session.state == State::Finished)
            {
                return true;
            }
            inner = self
                .idle
                .wait(inner)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
    }

    /// Go `Broker.schedule`.
    fn schedule(&self, inner: &mut Inner) {
        Self::observe(inner);
        self.commands(inner);
        if inner.active.is_some()
            || inner.aborted.is_some()
            || (inner.manual && inner.driver.is_none())
        {
            return;
        }
        let next = inner
            .sessions
            .iter()
            .position(|session| session.state == State::Runnable);
        #[cfg(feature = "determinism-fixture")]
        let next = inner
            .fixture_fault
            .handoff(
                inner.metrics.handoffs,
                inner
                    .sessions
                    .iter()
                    .map(|session| session.state == State::Runnable),
            )
            .or(next);
        if let Some(next) = next {
            inner.sessions[next].state = State::Active;
            inner.active = Some(next);
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            Self::event(inner, next, "grant");
            if inner.metrics.handoffs != 0 {
                inner.granted_at = Some(Instant::now());
            }
            #[cfg(feature = "stress-jitter")]
            crate::stress_jitter::delay();
            self.wake[next].notify_one();
            return;
        }
        let mut ready = false;
        for session in &inner.sessions {
            if session.state == State::Idle {
                if session.resume.is_none() {
                    return;
                }
                ready = true;
            }
        }
        if !ready {
            self.idle.notify_all();
            return;
        }
        let current = (inner.clock.current() - INITIAL_TIME).cast_unsigned();
        if let Some(network) = &inner.network {
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            let _event_scope = crate::event_order::enter(crate::event_order::owner(
                "controller",
                0,
                inner.clock.current() - INITIAL_TIME,
            ));
            let advanced = lock(network).advance(current);
            if let Err(failure) = advanced {
                for session in &mut inner.sessions {
                    if session.state == State::Idle {
                        session.resume = Some(Err(failure.message()));
                    }
                }
            }
        }
        // Opt-in live control when time advanced: every instance is idle and
        // due faults are applied, so link changes take effect at this time.
        if inner.pause.is_some() && current > inner.paused_at {
            inner.paused_at = current;
            if let Err((termination, message)) = Self::pause(inner, current) {
                self.raise(inner, termination, message);
                return;
            }
        }
        for session in &mut inner.sessions {
            if session.state == State::Idle {
                session.state = State::Runnable;
            }
        }
        self.schedule(inner);
    }

    /// A stalled cluster: no instance can run and the clock answers nobody.
    /// In Go the helpers would wait for the process watchdog.
    fn check_stall(&self, inner: &mut Inner) {
        if inner.active.is_none()
            && inner.aborted.is_none()
            && (!inner.manual || inner.driver.is_some())
            && inner
                .sessions
                .iter()
                .any(|session| session.state == State::Idle && session.resume.is_none())
            && !inner
                .sessions
                .iter()
                .any(|session| matches!(session.state, State::Runnable | State::Dormant))
        {
            self.raise(
                inner,
                "infrastructure",
                "in-process broker stalled: no instance can run and the clock answers none".into(),
            );
        }
    }

    fn deliver_answers(inner: &mut Inner) {
        for answer in inner.clock.take_answers() {
            if let Some(session) = inner
                .sessions
                .iter_mut()
                .find(|session| session.handle == answer.handle)
            {
                session.resume = Some(answer.result);
            }
        }
    }

    /// Opt-in live control at a clock advance (fn-178.1: commands are read
    /// only at deterministic pause points). Every other instance is idle.
    fn pause(inner: &mut Inner, at: u64) -> Result<(), (&'static str, String)> {
        let Some((input, output)) = inner.pause.clone() else {
            return Ok(());
        };
        // A settle pause is a Go round trip, charged like any remote wait.
        #[cfg(feature = "diagnostic-profile")]
        let _remote = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Remote);
        let frame = serde_json::json!({"type":"pause","reason":"settle","at":at});
        crate::protocol::send_frame(&mut **lock(&output), &frame)
            .map_err(|error| ("infrastructure", format!("pause transport: {error:#}")))?;
        loop {
            let bytes = crate::protocol::read_frame(&mut **lock(&input))
                .map_err(|error| ("invalid", format!("cluster control: {error:#}")))?;
            let control: Control = serde_json::from_slice(&bytes)
                .map_err(|error| ("invalid", format!("cluster control: {error}")))?;
            match (control.kind.as_str(), control.command.as_str()) {
                ("resume", "") => return Ok(()),
                ("control", "link") => {
                    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                    let _event_scope = crate::event_order::enter(crate::event_order::owner(
                        "controller",
                        0,
                        inner.clock.current() - INITIAL_TIME,
                    ));
                    let network = inner.network.clone().ok_or((
                        "invalid",
                        "cluster control: link without a network".to_owned(),
                    ))?;
                    lock(&network)
                        .change_link(
                            &control.action,
                            &control.from,
                            &control.to,
                            control.delay_nanos,
                            control.symmetric,
                        )
                        .map_err(|failure| {
                            ("invalid", format!("cluster control: {}", failure.message()))
                        })?;
                    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                    crate::event_order::emit_current_context(
                        "model-transition",
                        0,
                        serde_json::json!({"action": "controller_effect"}),
                    );
                }
                _ => return Err(("invalid", "cluster control: unknown frame".into())),
            }
        }
    }

    /// Go `BrokerSession.register` for a visible operation; `input` is the
    /// callback's JSON input bytes, whose SHA-256 is the request payload.
    fn register(&self, me: usize, op: &str, data: &[u8]) -> Result<(), Failure> {
        use base64::Engine as _;
        let input = format!(
            "{{\"data_base64\":\"{}\"}}",
            base64::engine::general_purpose::STANDARD.encode(data)
        );
        let mut inner = lock(&self.inner);
        Self::check_locked(&inner, me, inner.sessions[me].incarnation)?;
        let key = inner.sessions[me].key.clone();
        #[cfg(feature = "diagnostic-profile")]
        let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(Self::event_context(&inner, me));
        let id = inner
            .world
            .register(world::Request {
                kind: op.to_owned(),
                resource: world::ResourceId {
                    adapter: "wasm-broker".into(),
                    kind: "incarnation".into(),
                    key,
                },
                priority: 0,
                payload: Sha256::digest(input.as_bytes()).to_vec(),
            })
            .map_err(world_failure)?;
        inner.sessions[me].pending = id;
        Ok(())
    }

    /// Go `BrokerSession.ready`: ready at the clock's time, exactly one delivery.
    fn ready(&self, me: usize, op: &str) -> Result<(), Failure> {
        #[cfg(feature = "diagnostic-profile")]
        let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
        let mut inner = lock(&self.inner);
        Self::check_locked(&inner, me, inner.sessions[me].incarnation)?;
        let pending = inner.sessions[me].pending;
        let at = inner.clock.current();
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(Self::event_context(&inner, me));
        inner
            .world
            .ready(world::Readiness {
                request_id: pending,
                at,
                kind: op.to_owned(),
                payload: Vec::new(),
                equivalence_class: String::new(),
            })
            .map_err(world_failure)?;
        let quiescence = inner.world.quiesce().map_err(world_failure)?;
        if quiescence.deliveries.len() != 1 || quiescence.deliveries[0].request_id != pending {
            return Err(invalid("broker delivery order"));
        }
        inner.sessions[me].pending = 0;
        Ok(())
    }

    /// Go `prepareIdle`, the clock quiesce and `resumeIdle`, then the wait for
    /// the continuation; returns the relative time to resume at.
    fn idle(&self, me: usize, deadline: u64) -> Result<u64, Failure> {
        let mut deadline = if deadline == 0 {
            MAXIMUM_RELATIVE
        } else {
            deadline
        };
        if deadline > MAXIMUM_RELATIVE {
            return Err(invalid("broker deadline"));
        }
        let mut inner = lock(&self.inner);
        Self::observe(&mut inner);
        inner.metrics.handoffs += 1;
        if inner.active != Some(me) {
            return Err(invalid("broker continuation"));
        }
        let current = (inner.clock.current() - INITIAL_TIME).cast_unsigned();
        if let Some(network) = inner.network.clone() {
            let mut network = lock(&network);
            let next = network.deadline();
            if next != 0 {
                deadline = deadline.min(next);
            }
            if network.has_ready() {
                deadline = deadline.min(current.max(1));
            }
        }
        let session = &mut inner.sessions[me];
        session.generation += 1;
        let request = TimeRequest {
            generation: session.generation,
            current: INITIAL_TIME + current.cast_signed(),
            deadline: INITIAL_TIME + deadline.cast_signed(),
            arrivals: 0,
        };
        let handle = session.handle;
        session.state = State::Idle;
        session.resume = None;
        inner.active = None;
        self.schedule(&mut inner);
        #[cfg(feature = "diagnostic-profile")]
        let clock_span = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Clock);
        let quiesced = inner.clock.quiesce(Some(handle), request);
        match quiesced {
            Ok(Quiesced::Answered(response)) => inner.sessions[me].resume = Some(Ok(response)),
            Ok(Quiesced::Parked) => {}
            Err(error) => inner.sessions[me].resume = Some(Err(error)),
        }
        Self::deliver_answers(&mut inner);
        #[cfg(feature = "diagnostic-profile")]
        drop(clock_span);
        self.schedule(&mut inner);
        self.check_stall(&mut inner);
        // Blocked until this instance holds the baton again: waiting time
        // while other instances run, never this instance's host work.
        #[cfg(feature = "diagnostic-profile")]
        let waiting = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::HandoffWait);
        while inner.active != Some(me)
            && inner.aborted.is_none()
            && (inner.manual || inner.sessions[me].revoked.is_none())
        {
            inner = self.wake[me]
                .wait(inner)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
        #[cfg(feature = "diagnostic-profile")]
        drop(waiting);
        Self::check_locked(&inner, me, inner.sessions[me].incarnation)?;
        if let Some((_, message)) = inner.aborted.clone() {
            // The coordinator reports the run's own classification.
            return Err(Failure::Infrastructure(message));
        }
        if let Some(granted) = inner.granted_at.take() {
            inner.metrics.handoff_nanos += crate::metrics::elapsed(granted);
            inner.metrics.handoff_samples += 1;
        }
        let response = inner.sessions[me]
            .resume
            .take()
            .unwrap_or_else(|| Err("broker resumed without a clock answer".into()))
            .map_err(Failure::Infrastructure)?;
        if response.kind == clock::DEADLOCK {
            return Err(Failure::Infrastructure(
                "WASM broker global deadlock".into(),
            ));
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        crate::event_order::emit(
            &crate::event_order::owner(
                &inner.sessions[me].id,
                inner.sessions[me].incarnation,
                response.time - INITIAL_TIME,
            ),
            "model-transition",
            inner.sessions[me].generation,
            serde_json::json!({"action": "resume"}),
        );
        Ok((response.time - INITIAL_TIME).cast_unsigned())
    }

    /// Go `BrokerSession.Finished` (after the instance's evidence): cancel a
    /// pending visible request, leave the clock and pass the continuation.
    pub fn finish(&self, me: usize) {
        self.wait_terminal(me);
        let mut inner = lock(&self.inner);
        Self::observe(&mut inner);
        if inner.sessions[me].state == State::Finished {
            return;
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(Self::event_context(&inner, me));
        let pending = std::mem::take(&mut inner.sessions[me].pending);
        let finished = (|| {
            if pending != 0 {
                #[cfg(feature = "diagnostic-profile")]
                let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
                inner.world.cancel(pending).map_err(world_failure)?;
            }
            let session = &inner.sessions[me];
            let handle = session.hold.unwrap_or(session.handle);
            if session.revoked.is_none()
                && let Some(network) = inner.network.clone()
            {
                lock(&network)
                    .revoke(&session.id, session.incarnation, true)
                    .map_err(Failure::Infrastructure)?;
            }
            inner
                .clock
                .apply(clock::Event::Remove {
                    participant: Some(handle),
                })
                .map_err(Failure::Infrastructure)
        })();
        if let Err(failure) = finished {
            self.raise(&mut inner, failure.termination(), failure.message());
            if !matches!(&inner.aborted, Some(("trap" | "capacity", _))) {
                return;
            }
        }
        Self::deliver_answers(&mut inner);
        inner.sessions[me].hold = None;
        inner.sessions[me].state = State::Finished;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        Self::event(&inner, me, "finish");
        if inner.driver == Some((me, inner.sessions[me].incarnation)) {
            inner.driver = None;
        }
        if inner.active == Some(me) {
            inner.active = None;
        }
        if matches!(&inner.aborted, Some(("trap" | "capacity", _)))
            && (inner.terminal_turn == Some(me) || inner.terminal_turn.is_none())
        {
            inner.terminal_turn = inner
                .sessions
                .iter()
                .position(|session| !matches!(session.state, State::Dormant | State::Finished));
            for wake in &self.wake {
                wake.notify_all();
            }
        }
        self.schedule(&mut inner);
        self.check_stall(&mut inner);
        self.idle.notify_all();
    }

    /// The broker World's expanded `GMWSNAP1` wire, bounded while it is
    /// written by the compressed profile's expanded limit, as Go's
    /// `encodeCompressedSnapshot` is.
    pub fn world_wire(&self) -> Result<Vec<u8>, Failure> {
        self.world_wire_locked(&lock(&self.inner).world)
    }

    /// The v2 commitment binds the World transition chain and the calling
    /// instance's network commitment without materializing either history.
    pub fn inspection_world_commitment(
        &self,
        network: Option<[u8; 32]>,
    ) -> Result<[u8; 32], Failure> {
        #[cfg(feature = "diagnostic-profile")]
        let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
        let mut inner = lock(&self.inner);
        // Every successful World mutation commits a transition; refusals precede state writes.
        let revision = inner.world.next_ids().2;
        if let Some((prior, identity, digest)) = inner.inspection_world
            && prior == revision
            && identity == network
        {
            return Ok(digest);
        }
        if inner.world.snapshot_bytes() > MAXIMUM_EXPANDED_WORLD_BYTES as u64 {
            inner.inspection_world = None;
            return Err(Failure::Capacity("compressed-snapshot-expanded-bytes"));
        }
        let world = inner.world.inspection_commitment();
        let digest = crate::model::inspection::world_chain_commitment(&world, network);
        let digest: [u8; 32] = digest.try_into().unwrap();
        inner.inspection_world = Some((revision, network, digest));
        Ok(digest)
    }

    /// The current World's expanded wire as its generated head followed by
    /// the retained transition encodings, extended by the new transitions:
    /// the same bytes [`Self::world_wire`] writes, refused at the same bound.
    #[cfg(test)]
    fn inspection_wire_locked(
        &self,
        inner: &mut Inner,
        maximum: usize,
    ) -> Result<InspectionWire, Failure> {
        #[cfg(feature = "diagnostic-profile")]
        let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
        #[cfg(test)]
        self.world_materializations
            .fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let mut head = Bounded {
            bytes: Vec::new(),
            limit: maximum,
        };
        world::wire_generated::write_snapshot_head(&inner.world, &mut head)
            .map_err(world_wire_failure)?;
        let retained = &mut inner.inspection_transitions;
        let history = inner.world.transitions();
        let limit = maximum - head.bytes.len();
        if retained.count > history.len() {
            // Unreachable for the append-only history; re-encode it all.
            *retained = InspectionTransitions::default();
        }
        if retained.bytes.len() > limit {
            // Over the bound: release the retained encoding, as a refused
            // write releases its partial wire.
            *retained = InspectionTransitions::default();
            return Err(Failure::Capacity("compressed-snapshot-expanded-bytes"));
        }
        let mut transitions = Bounded {
            bytes: std::mem::take(&mut retained.bytes),
            limit,
        };
        if let Err(error) =
            world::wire_generated::write_transitions(&history[retained.count..], &mut transitions)
        {
            *retained = InspectionTransitions::default();
            return Err(world_wire_failure(error));
        }
        retained.bytes = transitions.bytes;
        retained.count = history.len();
        Ok(InspectionWire {
            head: head.bytes.into_boxed_slice(),
            transitions: retained.bytes.len(),
        })
    }

    #[cfg_attr(
        not(test),
        expect(
            clippy::unused_self,
            reason = "Tests count World materializations per broker instance"
        )
    )]
    fn world_wire_locked(&self, world: &world::Model) -> Result<Vec<u8>, Failure> {
        #[cfg(feature = "diagnostic-profile")]
        let _world = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::World);
        #[cfg(test)]
        self.world_materializations
            .fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let mut wire = Bounded {
            bytes: Vec::new(),
            limit: MAXIMUM_EXPANDED_WORLD_BYTES,
        };
        world::wire_generated::write_snapshot(world, &mut wire).map_err(world_wire_failure)?;
        Ok(wire.bytes)
    }
}

/// A broker World refusal: exhausting a World limit is the run's capacity
/// outcome naming that dimension and its accounting, as instance-level World
/// limits are (fn-235).
#[expect(
    clippy::needless_pass_by_value,
    reason = "Error classification is passed directly to Result::map_err"
)]
fn world_failure(error: world::Error) -> Failure {
    match error {
        world::Error::Capacity {
            dimension,
            limit,
            used,
            delta,
        } => Failure::Exhausted {
            dimension,
            limit,
            used,
            delta,
        },
        world::Error::Classified { .. } => Failure::Infrastructure(error.to_string()),
    }
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Error classification is passed directly to Result::map_err"
)]
fn world_wire_failure(error: std::io::Error) -> Failure {
    if error.kind() == std::io::ErrorKind::FileTooLarge {
        Failure::Capacity("compressed-snapshot-expanded-bytes")
    } else {
        Failure::Infrastructure(format!("World evidence: {error}"))
    }
}

#[cfg(test)]
struct InspectionWire {
    head: Box<[u8]>,
    transitions: usize,
}

/// See `Inner::inspection_transitions`.
#[cfg(test)]
#[derive(Default)]
struct InspectionTransitions {
    count: usize,
    bytes: Vec<u8>,
}

/// Go `world.CompressedSnapshotProfile().MaxExpandedBytes`.
const MAXIMUM_EXPANDED_WORLD_BYTES: usize = 384 << 20;

/// A `Vec` writer that refuses to grow past `limit`.
struct Bounded {
    bytes: Vec<u8>,
    limit: usize,
}

impl std::io::Write for Bounded {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        if data.len() > self.limit - self.bytes.len() {
            return Err(std::io::ErrorKind::FileTooLarge.into());
        }
        self.bytes.extend_from_slice(data);
        Ok(data.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

/// One instance's view of the broker: its slot and its network endpoint.
pub struct Member {
    pub broker: Arc<Broker>,
    pub index: usize,
    pub incarnation: u64,
    pub network: Option<Endpoint>,
}

/// How a broker call writes the instance's network commitment.
#[derive(Clone, Copy)]
enum NetworkWrite {
    Eager,
    Deferred,
}

/// The modeled clock state of the calling instance's local model.
pub struct Clockwork<'a> {
    pub now: &'a mut u64,
    pub advances: &'a mut u64,
    pub runtime_active: bool,
}

impl Member {
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn event_scope(&self, now: u64) -> crate::event_order::Scope {
        let node = lock(&self.broker.inner).sessions[self.index].id.clone();
        crate::event_order::enter(crate::event_order::owner(
            &node,
            self.incarnation,
            now.cast_signed(),
        ))
    }
    /// Records the instance's current network commitment in its cut.
    /// `Deferred` only checks the evidence would encode and leaves the cut
    /// pending: valid where the instance keeps the continuation until its
    /// next eager network write (`Eager` before a handoff, or the next
    /// boundary or terminal snapshot) supersedes it.
    fn inspection_network_snapshot(&mut self, write: NetworkWrite) {
        #[cfg(test)]
        if crate::model::inspection::eager_oracle() {
            return self.inspection_network_snapshot_eager();
        }
        let Some(endpoint) = &mut self.network else {
            return;
        };
        let node = lock(&self.broker.inner).sessions[self.index].id.clone();
        let mut recorder = lock(&self.broker.inspection);
        if !recorder.observed || recorder.exhausted {
            return;
        }
        let handle = crate::model::inspection::Handle {
            node,
            incarnation: self.incarnation,
        };
        match write {
            NetworkWrite::Eager => {
                match endpoint.inspection_network(crate::model::inspection::network_commitment) {
                    Ok((digest, _)) => recorder.network_digest(&handle, digest),
                    Err(_) => recorder.exhausted = true,
                }
            }
            NetworkWrite::Deferred => match endpoint.evidence_check() {
                Ok(()) => recorder.network_pending(&handle),
                Err(_) => recorder.exhausted = true,
            },
        }
    }
    /// Refuses a handoff that would leave this instance's deferred network
    /// commitment observable by another instance: the deferral relies on an
    /// eager write superseding it first.
    pub fn check_network_commitment(&self, at: &str) -> Result<(), String> {
        if self.network.is_none() {
            return Ok(());
        }
        let handle = crate::model::inspection::Handle {
            node: lock(&self.broker.inner).sessions[self.index].id.clone(),
            incarnation: self.incarnation,
        };
        if lock(&self.broker.inspection).deferred_network_pending(&handle) {
            return Err(format!(
                "deferred network inspection commitment pending at {at}"
            ));
        }
        Ok(())
    }
    /// Test oracle: the original hook, which rebuilds and hashes the
    /// evidence at every call.
    #[cfg(test)]
    fn inspection_network_snapshot_eager(&self) {
        let Some(endpoint) = &self.network else {
            return;
        };
        let node = lock(&self.broker.inner).sessions[self.index].id.clone();
        let mut recorder = lock(&self.broker.inspection);
        if !recorder.observed || recorder.exhausted {
            return;
        }
        match endpoint.inspection_network_eager() {
            Ok(bytes) => recorder.network_snapshot(
                &crate::model::inspection::Handle {
                    node,
                    incarnation: self.incarnation,
                },
                &bytes,
            ),
            Err(_) => recorder.exhausted = true,
        }
    }
    pub fn begin_output(&mut self, now: &mut u64) -> Result<(), Failure> {
        self.broker.check(self.index, self.incarnation)?;
        let current = self.broker.current();
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = self.event_scope(current);
        if let Some(endpoint) = &mut self.network {
            endpoint.attach_at(current)?;
        }
        *now = (*now).max(current);
        Ok(())
    }

    /// Go `beginCall` for a visible call: the broker's clock bounds the
    /// instance clock from below, and the network attaches and advances to
    /// broker time.
    fn begin(&mut self, op: &str, data: &[u8], clock: &mut Clockwork<'_>) -> Result<(), Failure> {
        self.broker.check(self.index, self.incarnation)?;
        self.broker.register(self.index, op, data)?;
        let current = self.broker.current();
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = self.event_scope(current);
        if let Some(endpoint) = &mut self.network {
            endpoint.attach_at(current)?;
        }
        *clock.now = (*clock.now).max(current);
        Ok(())
    }

    /// `runtime_idle` under the broker: Go `BrokerSession.Handle` for an idle
    /// call and the broker-time branch of `Environment.runtimeIdle`.
    pub fn idle(&mut self, data: &mut [u8], clock: &mut Clockwork<'_>) -> Result<(), Failure> {
        self.begin("runtime_idle", data, clock)?;
        if !clock.runtime_active {
            return Err(invalid("runtime idle outside active lifetime"));
        }
        let word = |at: usize| u32::from_be_bytes(data[at..at + 4].try_into().unwrap());
        if data.len() != 16 || word(0) != 1 || word(4) != 0 {
            return Err(invalid("runtime idle layout"));
        }
        let deadline = u64::from_be_bytes(data[8..16].try_into().unwrap());
        // Other instances may record boundaries while this one is parked.
        self.inspection_network_snapshot(NetworkWrite::Eager);
        self.check_network_commitment("idle handoff")
            .map_err(Failure::Infrastructure)?;
        let now = self.broker.idle(self.index, deadline)?;
        if now < *clock.now {
            return Err(invalid("broker clock moved backwards"));
        }
        if now > *clock.now {
            *clock.advances += 1;
        }
        *clock.now = now;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = self.event_scope(now);
        if let Some(endpoint) = &mut self.network {
            #[cfg(feature = "diagnostic-profile")]
            let _network = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Network);
            endpoint.settle(now)?;
            data[4..8].copy_from_slice(&1u32.to_be_bytes());
        }
        data[8..16].copy_from_slice(&now.to_be_bytes());
        self.inspection_network_snapshot(NetworkWrite::Deferred);
        self.broker.ready(self.index, "runtime_idle")
    }

    /// `runtime_net_stream` under the broker: the World brackets fn-178.12's
    /// in-helper stream serving.
    pub fn net_stream(
        &mut self,
        data: &mut [u8],
        clock: &mut Clockwork<'_>,
    ) -> Result<(), Failure> {
        self.begin("runtime_net_stream", data, clock)?;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = self.event_scope(*clock.now);
        let endpoint = self
            .network
            .as_mut()
            .ok_or(invalid("broker node requires stream network profile"))?;
        endpoint.serve(data, *clock.now, clock.runtime_active)?;
        self.inspection_network_snapshot(NetworkWrite::Deferred);
        self.broker.ready(self.index, "runtime_net_stream")
    }
}

#[cfg(test)]
mod tests;
