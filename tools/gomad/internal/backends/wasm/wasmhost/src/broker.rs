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
use crate::cluster::transport::{SharedInput, SharedOutput, lock};
use crate::network::cluster::{Endpoint, Failure, Shared};
use crate::world;
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::sync::{Arc, Condvar, Mutex};

const INITIAL_TIME: i64 = world::INITIAL_TIME;
/// Go `math.MaxInt64 - world.InitialTime`: the largest relative deadline.
const MAXIMUM_RELATIVE: u64 = (i64::MAX - INITIAL_TIME) as u64;

/// The cluster `broker` object: Go `BrokerConfig`'s World fields.
#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub seed: u64,
    pub limits: WorldLimits,
    /// Stop at every clock advance for live control frames (link changes).
    #[serde(default)]
    pub pause_on_settle: bool,
}

#[derive(Clone, Debug, Deserialize)]
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
    Runnable,
    Active,
    Idle,
    Finished,
}

struct Session {
    key: String,
    handle: clock::Handle,
    state: State,
    resume: Option<Result<TimeResponse, String>>,
    generation: u64,
    pending: u64,
}

struct Inner {
    sessions: Vec<Session>,
    active: Option<usize>,
    aborted: Option<(&'static str, String)>,
    /// The abort when the broker raised it itself (stall, pause, World or
    /// clock refusal) rather than an instance reporting its own failure.
    raised: Option<(&'static str, String)>,
    clock: Clock,
    world: world::Model,
    network: Option<Arc<Mutex<Shared>>>,
    pause: Option<(SharedInput, SharedOutput)>,
    paused_at: u64,
}

pub struct Broker {
    inner: Mutex<Inner>,
    wake: Vec<Condvar>,
    idle: Condvar,
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
                sessions,
                active: None,
                aborted: None,
                raised: None,
                clock,
                world,
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
        })
    }

    /// Go `Broker.Current`: relative virtual time.
    pub fn current(&self) -> u64 {
        (lock(&self.inner).clock.current() - INITIAL_TIME) as u64
    }

    pub fn start(&self) {
        let mut inner = lock(&self.inner);
        self.schedule(&mut inner);
    }

    pub fn acquire(&self, me: usize) -> bool {
        let mut inner = lock(&self.inner);
        while inner.active != Some(me) && inner.aborted.is_none() {
            inner = self.wake[me]
                .wait(inner)
                .unwrap_or_else(|poisoned| poisoned.into_inner());
        }
        inner.aborted.is_none()
    }

    pub fn abort(&self, termination: &'static str, message: String) {
        let mut inner = lock(&self.inner);
        self.abort_locked(&mut inner, termination, message);
    }

    /// An abort the broker raises itself: its classification is the run's.
    fn raise(&self, inner: &mut Inner, termination: &'static str, message: String) {
        if inner.aborted.is_none() {
            inner.raised = Some((termination, message.clone()));
        }
        self.abort_locked(inner, termination, message);
    }

    fn abort_locked(&self, inner: &mut Inner, termination: &'static str, message: String) {
        if inner.aborted.is_none() {
            inner.aborted = Some((termination, message));
        }
        inner.active = None;
        for wake in &self.wake {
            wake.notify_all();
        }
        self.idle.notify_all();
    }

    /// The whole-run failure the broker itself raised, when it raised the
    /// first abort; it takes precedence over what the aborted instances report.
    pub fn failure(&self) -> Option<(&'static str, String)> {
        lock(&self.inner).raised.clone()
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
                .unwrap_or_else(|poisoned| poisoned.into_inner());
        }
    }

    /// Go `Broker.schedule`.
    fn schedule(&self, inner: &mut Inner) {
        if inner.active.is_some() || inner.aborted.is_some() {
            return;
        }
        if let Some(next) = inner
            .sessions
            .iter()
            .position(|session| session.state == State::Runnable)
        {
            inner.sessions[next].state = State::Active;
            inner.active = Some(next);
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
        let current = (inner.clock.current() - INITIAL_TIME) as u64;
        if let Some(network) = &inner.network {
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
            if let Err((termination, message)) = self.pause(inner, current) {
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
            && inner
                .sessions
                .iter()
                .any(|session| session.state == State::Idle && session.resume.is_none())
            && !inner
                .sessions
                .iter()
                .any(|session| session.state == State::Runnable)
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
    fn pause(&self, inner: &mut Inner, at: u64) -> Result<(), (&'static str, String)> {
        let Some((input, output)) = inner.pause.clone() else {
            return Ok(());
        };
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
        let key = inner.sessions[me].key.clone();
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
            .map_err(|error| Failure::Infrastructure(error.to_string()))?;
        inner.sessions[me].pending = id;
        Ok(())
    }

    /// Go `BrokerSession.ready`: ready at the clock's time, exactly one delivery.
    fn ready(&self, me: usize, op: &str) -> Result<(), Failure> {
        let mut inner = lock(&self.inner);
        let pending = inner.sessions[me].pending;
        let at = inner.clock.current();
        inner
            .world
            .ready(world::Readiness {
                request_id: pending,
                at,
                kind: op.to_owned(),
                payload: Vec::new(),
                equivalence_class: String::new(),
            })
            .map_err(|error| Failure::Infrastructure(error.to_string()))?;
        let quiescence = inner
            .world
            .quiesce()
            .map_err(|error| Failure::Infrastructure(error.to_string()))?;
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
        if inner.active != Some(me) {
            return Err(invalid("broker continuation"));
        }
        let current = (inner.clock.current() - INITIAL_TIME) as u64;
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
            current: INITIAL_TIME + current as i64,
            deadline: INITIAL_TIME + deadline as i64,
            arrivals: 0,
        };
        let handle = session.handle;
        session.state = State::Idle;
        session.resume = None;
        inner.active = None;
        self.schedule(&mut inner);
        let quiesced = inner.clock.quiesce(Some(handle), request);
        match quiesced {
            Ok(Quiesced::Answered(response)) => inner.sessions[me].resume = Some(Ok(response)),
            Ok(Quiesced::Parked) => {}
            Err(error) => inner.sessions[me].resume = Some(Err(error)),
        }
        Self::deliver_answers(&mut inner);
        self.schedule(&mut inner);
        self.check_stall(&mut inner);
        while inner.active != Some(me) && inner.aborted.is_none() {
            inner = self.wake[me]
                .wait(inner)
                .unwrap_or_else(|poisoned| poisoned.into_inner());
        }
        if let Some((_, message)) = inner.aborted.clone() {
            // The coordinator reports the run's own classification.
            return Err(Failure::Infrastructure(message));
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
        Ok((response.time - INITIAL_TIME) as u64)
    }

    /// Go `BrokerSession.Finished` (after the instance's evidence): cancel a
    /// pending visible request, leave the clock and pass the continuation.
    pub fn finish(&self, me: usize) {
        let mut inner = lock(&self.inner);
        if inner.sessions[me].state == State::Finished {
            return;
        }
        let pending = std::mem::take(&mut inner.sessions[me].pending);
        if pending != 0
            && let Err(error) = inner.world.cancel(pending)
        {
            self.raise(
                &mut inner,
                "infrastructure",
                format!("broker World: {error}"),
            );
            return;
        }
        let handle = inner.sessions[me].handle;
        if let Err(error) = inner.clock.apply(clock::Event::Remove {
            participant: Some(handle),
        }) {
            self.raise(&mut inner, "infrastructure", error);
            return;
        }
        Self::deliver_answers(&mut inner);
        inner.sessions[me].state = State::Finished;
        if inner.active == Some(me) {
            inner.active = None;
        }
        self.schedule(&mut inner);
        self.check_stall(&mut inner);
        self.idle.notify_all();
    }

    /// The expanded `GMWSNAP1` wire of the broker World, for evidence at finish.
    /// The broker World's expanded `GMWSNAP1` wire, bounded while it is
    /// written by the compressed profile's expanded limit, as Go's
    /// `encodeCompressedSnapshot` is.
    pub fn world_wire(&self) -> Result<Vec<u8>, Failure> {
        let inner = lock(&self.inner);
        let mut wire = Bounded {
            bytes: Vec::new(),
            limit: MAXIMUM_EXPANDED_WORLD_BYTES,
        };
        match world::wire_generated::write_snapshot(&inner.world, &mut wire) {
            Ok(()) => Ok(wire.bytes),
            Err(error) if error.kind() == std::io::ErrorKind::FileTooLarge => {
                Err(Failure::Capacity("compressed-snapshot-expanded-bytes"))
            }
            Err(error) => Err(Failure::Infrastructure(format!("World evidence: {error}"))),
        }
    }
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
    pub network: Option<Endpoint>,
}

/// The modeled clock state of the calling instance's local model.
pub struct Clockwork<'a> {
    pub now: &'a mut u64,
    pub advances: &'a mut u64,
    pub runtime_active: bool,
}

impl Member {
    /// Go `beginCall` for a visible call: the broker's clock bounds the
    /// instance clock from below, and the network attaches and advances to
    /// broker time.
    fn begin(&mut self, op: &str, data: &[u8], clock: &mut Clockwork<'_>) -> Result<(), Failure> {
        self.broker.register(self.index, op, data)?;
        let current = self.broker.current();
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
        let now = self.broker.idle(self.index, deadline)?;
        if now < *clock.now {
            return Err(invalid("broker clock moved backwards"));
        }
        if now > *clock.now {
            *clock.advances += 1;
        }
        *clock.now = now;
        if let Some(endpoint) = &mut self.network {
            endpoint.settle(now)?;
            data[4..8].copy_from_slice(&1u32.to_be_bytes());
        }
        data[8..16].copy_from_slice(&now.to_be_bytes());
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
        let endpoint = self
            .network
            .as_mut()
            .ok_or(invalid("broker node requires stream network profile"))?;
        endpoint.serve(data, *clock.now, clock.runtime_active)?;
        self.broker.ready(self.index, "runtime_net_stream")
    }
}

#[cfg(test)]
mod tests;
