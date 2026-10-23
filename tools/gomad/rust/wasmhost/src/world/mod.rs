//! Rust duplicate of the Go World model (`internal/model/world`) for the
//! in-process cluster scheduler (fn-178). Go stays authoritative for the native
//! backend and the runner; the checked-in conformance corpus
//! (`tests/testdata/world_conformance`, pinned to `world.ImplementationSHA256`)
//! keeps the two equal. Single-threaded: the serialized scheduler is the only
//! caller. Replay plans and recordings are not ported; the broker uses neither.
mod deflate;
pub mod digest;
pub(crate) mod gzip;
pub mod order;
#[cfg(not(feature = "conformance"))]
pub(crate) mod retained;
#[rustfmt::skip]
pub mod wire_generated;

use order::{QueueKey, choice_rank};
use std::collections::{BTreeMap, BTreeSet};
use std::fmt;

pub const SCHEMA_VERSION: u32 = 1;
pub const INITIAL_TIME: i64 = 946_684_800_000_000_000;

#[derive(Clone, Debug, PartialEq, Eq)]
#[expect(
    clippy::struct_field_names,
    reason = "Field names mirror the Go World limits contract"
)]
pub struct Limits {
    pub max_requests: u64,
    pub max_events: u64,
    pub max_queued_events: u64,
    pub max_transitions: u64,
    pub max_payload_bytes: u64,
    pub max_string_bytes: u32,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Config {
    pub seed: u64,
    pub limits: Limits,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ResourceId {
    pub adapter: String,
    pub kind: String,
    pub key: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Request {
    pub kind: String,
    pub resource: ResourceId,
    pub priority: u16,
    pub payload: Vec<u8>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Readiness {
    pub request_id: u64,
    pub at: i64,
    pub kind: String,
    pub payload: Vec<u8>,
    pub equivalence_class: String,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CancelStatus {
    Won,
    AlreadyCanceled,
    AlreadyDelivered,
}

impl CancelStatus {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Won => "won",
            Self::AlreadyCanceled => "already-canceled",
            Self::AlreadyDelivered => "already-delivered",
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Cancellation {
    pub request_id: u64,
    pub event_id: u64,
    pub status: CancelStatus,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Delivery {
    pub request_id: u64,
    pub event_id: u64,
    pub at: i64,
    pub kind: String,
    pub payload: Vec<u8>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum QuiescenceKind {
    Delivered,
    Deadlock,
    Idle,
}

impl QuiescenceKind {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Delivered => "delivered",
            Self::Deadlock => "deadlock",
            Self::Idle => "idle",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Quiescence {
    pub kind: QuiescenceKind,
    pub before: i64,
    pub after: i64,
    pub deliveries: Vec<Delivery>,
    pub blocked: Vec<u64>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RequestState {
    Pending,
    Queued,
    Canceled,
    Delivered,
}

impl RequestState {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Pending => "pending",
            Self::Queued => "queued",
            Self::Canceled => "canceled",
            Self::Delivered => "delivered",
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EventState {
    Queued,
    Canceled,
    Delivered,
}

impl EventState {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Queued => "queued",
            Self::Canceled => "canceled",
            Self::Delivered => "delivered",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum TransitionBody {
    Register {
        request: Request,
        request_id: u64,
    },
    Ready {
        readiness: Readiness,
        event_id: u64,
    },
    Cancel {
        request_id: u64,
        cancellation: Cancellation,
    },
    Quiesce {
        result: Quiescence,
    },
}

impl TransitionBody {
    pub fn kind(&self) -> &'static str {
        match self {
            Self::Register { .. } => "register",
            Self::Ready { .. } => "ready",
            Self::Cancel { .. } => "cancel",
            Self::Quiesce { .. } => "quiesce",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Transition {
    pub sequence: u64,
    pub previous_digest: [u8; 32],
    pub digest: [u8; 32],
    pub body: TransitionBody,
}

/// Go's error classes (`errors.go`), with Go's exact message text.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ErrorClass {
    InvalidConfig,
    InvalidRequest,
    UnknownRequest,
    RequestState,
    TimeRegression,
}

impl ErrorClass {
    fn sentinel(self) -> &'static str {
        match self {
            Self::InvalidConfig => "invalid World config",
            Self::InvalidRequest => "invalid World request",
            Self::UnknownRequest => "unknown World request",
            Self::RequestState => "invalid World request state",
            Self::TimeRegression => "World time regression",
        }
    }

    #[cfg(test)]
    pub fn name(self) -> &'static str {
        match self {
            Self::InvalidConfig => "invalid-config",
            Self::InvalidRequest => "invalid-request",
            Self::UnknownRequest => "unknown-request",
            Self::RequestState => "request-state",
            Self::TimeRegression => "time-regression",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Error {
    Capacity {
        dimension: &'static str,
        limit: u64,
        used: u64,
        delta: u64,
    },
    Classified {
        class: ErrorClass,
        detail: String,
    },
}

impl Error {
    #[cfg(test)]
    pub fn class(&self) -> &'static str {
        match self {
            Self::Capacity { .. } => "capacity",
            Self::Classified { class, .. } => class.name(),
        }
    }

    fn classified(class: ErrorClass, detail: impl Into<String>) -> Self {
        Self::Classified {
            class,
            detail: detail.into(),
        }
    }
}

impl fmt::Display for Error {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Capacity {
                dimension,
                limit,
                used,
                delta,
            } => write!(
                formatter,
                "World capacity exhausted: {dimension} limit={limit} used={used} delta={delta}"
            ),
            Self::Classified { class, detail } => {
                write!(formatter, "{}: {detail}", class.sentinel())
            }
        }
    }
}

impl std::error::Error for Error {}

pub type Result<T> = std::result::Result<T, Error>;

#[derive(Clone, Debug)]
pub struct RequestEntry {
    pub request: Request,
    pub state: RequestState,
    pub event_id: u64,
}

#[derive(Clone, Debug)]
pub struct Event {
    pub readiness: Readiness,
    pub state: EventState,
    /// Present only while queued; delivered and canceled events drop it.
    key: Option<QueueKey>,
}

/// Cheap per-step summary, Go `DiagnosticCardinality`.
#[derive(Clone, Debug, PartialEq, Eq)]
#[allow(
    dead_code,
    reason = "Detached cardinality is retained for the differential World corpus."
)]
pub struct Cardinality {
    pub now: i64,
    pub requests: u64,
    pub events: u64,
    pub queued_events: u64,
    pub transitions: u64,
    pub payload_bytes: u64,
    pub transcript_digest: [u8; 32],
}

pub struct Model {
    config: Config,
    now: i64,
    next_request_id: u64,
    next_event_id: u64,
    next_transition: u64,
    payload_bytes: u64,
    requests: BTreeMap<u64, RequestEntry>,
    events: BTreeMap<u64, Event>,
    queue: BTreeSet<QueueKey>,
    history: Vec<Transition>,
    transcript: [u8; 32],
    snapshot_bytes: u64,
    transition_limit: Option<u64>,
    transition_bytes: u64,
    /// [`digest::state_digest`]'s framed transition encodings for a prefix
    /// of the append-only history, so a digest re-encodes only transitions
    /// past that prefix.
    transition_frames: std::cell::RefCell<TransitionFrames>,
}

#[derive(Default)]
struct TransitionFrames {
    count: usize,
    bytes: Vec<u8>,
}

/// Bound on the retained framed transitions; transitions past it are not
/// retained and are framed while streaming into the hasher on every digest.
const MAXIMUM_TRANSITION_FRAME_BYTES: usize = 256 << 20;

fn capacity(dimension: &'static str, limit: u64, used: u64, delta: u64) -> Error {
    Error::Capacity {
        dimension,
        limit,
        used,
        delta,
    }
}

fn check_capacity(dimension: &'static str, limit: u64, used: u64, delta: u64) -> Result<()> {
    if used > limit || delta > limit - used {
        return Err(capacity(dimension, limit, used, delta));
    }
    Ok(())
}

/// Go `^[a-z][a-z0-9._-]{0,63}$`.
fn resource_component(value: &str) -> bool {
    let bytes = value.as_bytes();
    !bytes.is_empty()
        && bytes.len() <= 64
        && bytes[0].is_ascii_lowercase()
        && bytes[1..].iter().all(|byte| {
            byte.is_ascii_lowercase() || byte.is_ascii_digit() || matches!(byte, b'.' | b'_' | b'-')
        })
}

fn valid_string(limits: &Limits, value: &str, empty_allowed: bool) -> bool {
    (empty_allowed || !value.is_empty()) && value.len() as u64 <= u64::from(limits.max_string_bytes)
}

fn validate_config(config: &Config) -> Result<()> {
    let limits = &config.limits;
    if limits.max_requests == 0
        || limits.max_events == 0
        || limits.max_queued_events == 0
        || limits.max_transitions == 0
        || limits.max_payload_bytes == 0
        || limits.max_string_bytes == 0
    {
        return Err(Error::classified(
            ErrorClass::InvalidConfig,
            "every limit must be nonzero",
        ));
    }
    if limits.max_queued_events > limits.max_events {
        return Err(Error::classified(
            ErrorClass::InvalidConfig,
            "maxQueuedEvents exceeds maxEvents",
        ));
    }
    Ok(())
}

fn validate_request(limits: &Limits, request: &Request) -> Result<()> {
    if !valid_string(limits, &request.kind, false) {
        return Err(Error::classified(
            ErrorClass::InvalidRequest,
            "invalid request.kind",
        ));
    }
    let resource = &request.resource;
    let max = u64::from(limits.max_string_bytes);
    if !resource_component(&resource.adapter)
        || !resource_component(&resource.kind)
        || resource.adapter.len() as u64 > max
        || resource.kind.len() as u64 > max
    {
        return Err(Error::classified(
            ErrorClass::InvalidRequest,
            "request.resource",
        ));
    }
    if !valid_string(limits, &resource.key, true) {
        return Err(Error::classified(
            ErrorClass::InvalidRequest,
            "invalid request.resource.key",
        ));
    }
    if request.payload.len() as u64 > limits.max_payload_bytes {
        return Err(capacity(
            "payload-bytes",
            limits.max_payload_bytes,
            0,
            request.payload.len() as u64,
        ));
    }
    Ok(())
}

fn validate_readiness(limits: &Limits, readiness: &Readiness) -> Result<()> {
    if readiness.request_id == 0 {
        return Err(Error::classified(
            ErrorClass::UnknownRequest,
            "readiness.requestId",
        ));
    }
    if !valid_string(limits, &readiness.kind, false) {
        return Err(Error::classified(
            ErrorClass::InvalidRequest,
            "invalid readiness.kind",
        ));
    }
    if !valid_string(limits, &readiness.equivalence_class, true) {
        return Err(Error::classified(
            ErrorClass::InvalidRequest,
            "invalid readiness.equivalenceClass",
        ));
    }
    Ok(())
}

impl Model {
    pub fn new(config: Config) -> Result<Self> {
        validate_config(&config)?;
        let mut model = Self {
            config,
            now: INITIAL_TIME,
            next_request_id: 1,
            next_event_id: 1,
            next_transition: 1,
            payload_bytes: 0,
            requests: BTreeMap::new(),
            events: BTreeMap::new(),
            queue: BTreeSet::new(),
            history: Vec::new(),
            transcript: digest::empty_digest(),
            snapshot_bytes: 0,
            transition_limit: None,
            transition_bytes: 0,
            transition_frames: std::cell::RefCell::default(),
        };
        let mut size = WireSize::default();
        wire_generated::write_snapshot(&model, &mut size).expect("World wire size");
        model.snapshot_bytes = size.0;
        Ok(model)
    }

    pub fn config(&self) -> &Config {
        &self.config
    }

    pub fn now(&self) -> i64 {
        self.now
    }

    #[allow(
        dead_code,
        reason = "Detached cardinality is retained for the differential World corpus."
    )]
    pub fn cardinality(&self) -> Cardinality {
        Cardinality {
            now: self.now,
            requests: self.requests.len() as u64,
            events: self.events.len() as u64,
            queued_events: self.queue.len() as u64,
            transitions: self.history.len() as u64,
            payload_bytes: self.payload_bytes,
            transcript_digest: self.transcript,
        }
    }

    pub fn next_ids(&self) -> (u64, u64, u64) {
        (
            self.next_request_id,
            self.next_event_id,
            self.next_transition,
        )
    }

    pub fn payload_bytes(&self) -> u64 {
        self.payload_bytes
    }

    pub fn transcript_digest(&self) -> &[u8; 32] {
        &self.transcript
    }

    pub fn requests(&self) -> impl ExactSizeIterator<Item = (u64, &RequestEntry)> {
        self.requests.iter().map(|(id, entry)| (*id, entry))
    }

    pub fn events(&self) -> impl ExactSizeIterator<Item = (u64, &Event)> {
        self.events.iter().map(|(id, event)| (*id, event))
    }

    pub fn transitions(&self) -> &[Transition] {
        &self.history
    }

    /// Go `stateDigest` of the current snapshot (replay progress is not bound).
    pub fn state_digest(&self) -> [u8; 32] {
        self.state_digest_within(MAXIMUM_TRANSITION_FRAME_BYTES)
    }

    /// A fixed-size commitment to the initial configuration and ordered
    /// transition chain. Every successful mutation extends that chain.
    pub fn inspection_commitment(&self) -> [u8; 32] {
        digest::inspection_commitment(self)
    }

    pub fn snapshot_bytes(&self) -> u64 {
        self.snapshot_bytes
    }

    /// [`Self::state_digest`] reusing at most `maximum` retained bytes of
    /// framed transitions; history is append-only, so a retained prefix
    /// stays exact. Transitions past the bound are framed while hashing and
    /// not retained, so the retained prefix keeps serving later digests.
    fn state_digest_within(&self, maximum: usize) -> [u8; 32] {
        let mut frames = self.transition_frames.borrow_mut();
        if frames.count > self.history.len() {
            // Unreachable for the append-only history; re-encode it all.
            debug_assert!(false, "retained transition frames exceed history");
            *frames = TransitionFrames::default();
        }
        let mut encoded = Vec::new();
        while frames.count < self.history.len() {
            encoded.clear();
            digest::append_transition_frame(&mut encoded, &self.history[frames.count]);
            if frames.bytes.len() + encoded.len() > maximum {
                break;
            }
            frames.bytes.extend_from_slice(&encoded);
            frames.count += 1;
        }
        digest::state_digest_framed(
            digest::StateInput {
                config: &self.config,
                now: self.now,
                next_request_id: self.next_request_id,
                next_event_id: self.next_event_id,
                next_transition: self.next_transition,
                payload_bytes: self.payload_bytes,
                requests: self.requests(),
                events: self.events(),
                transitions: &self.history,
                transcript: &self.transcript,
            },
            &frames.bytes,
            frames.count,
        )
    }

    #[cfg(test)]
    fn uncached_state_digest(&self) -> [u8; 32] {
        digest::state_digest(digest::StateInput {
            config: &self.config,
            now: self.now,
            next_request_id: self.next_request_id,
            next_event_id: self.next_event_id,
            next_transition: self.next_transition,
            payload_bytes: self.payload_bytes,
            requests: self.requests(),
            events: self.events(),
            transitions: &self.history,
            transcript: &self.transcript,
        })
    }

    fn check_payload(&self, delta: u64) -> Result<()> {
        check_capacity(
            "payload-bytes",
            self.config.limits.max_payload_bytes,
            self.payload_bytes,
            delta,
        )
    }

    pub(crate) fn limit_transition_bytes(&mut self, limit: Option<u64>) -> Result<()> {
        if limit == Some(0) {
            return Err(Error::classified(
                ErrorClass::InvalidConfig,
                "transition byte limit must be positive",
            ));
        }
        if let Some(limit) = limit {
            check_capacity("transition-bytes", limit, self.transition_bytes, 0)?;
        }
        self.transition_limit = limit;
        Ok(())
    }

    fn check_transition(&self) -> Result<()> {
        if self.next_transition == 0 || self.next_transition == u64::MAX {
            return Err(capacity("transition-sequences", u64::MAX, u64::MAX, 1));
        }
        check_capacity(
            "transitions",
            self.config.limits.max_transitions,
            self.history.len() as u64,
            1,
        )
    }

    fn next_transition(&self, body: TransitionBody) -> Result<Transition> {
        let mut transition = Transition {
            sequence: self.next_transition,
            previous_digest: self.transcript,
            digest: [0; 32],
            body,
        };
        transition.digest = digest::transition_digest(&transition);
        if let Some(limit) = self.transition_limit {
            let mut size = WireSize::default();
            wire_generated::write_transitions(std::slice::from_ref(&transition), &mut size)
                .expect("World transition wire size");
            check_capacity("transition-bytes", limit, self.transition_bytes, size.0)?;
        }
        Ok(transition)
    }

    fn commit(&mut self, transition: Transition) {
        let mut size = WireSize::default();
        wire_generated::write_transitions(std::slice::from_ref(&transition), &mut size)
            .expect("World transition wire size");
        self.transition_bytes += size.0;
        self.snapshot_bytes += size.0;
        match &transition.body {
            TransitionBody::Register { request, .. } => {
                self.snapshot_bytes += 8
                    + 4 * 4
                    + 2
                    + 8
                    + 4
                    + 7
                    + 8
                    + (request.kind.len()
                        + request.resource.adapter.len()
                        + request.resource.kind.len()
                        + request.resource.key.len()
                        + request.payload.len()) as u64;
            }
            TransitionBody::Ready { readiness, .. } => {
                // Pending -> queued shortens the request state by one byte.
                self.snapshot_bytes += 8 + 8 + 8 + 4 + 8 + 4 + 4 + 6 - 1
                    + (readiness.kind.len()
                        + readiness.payload.len()
                        + readiness.equivalence_class.len()) as u64;
            }
            TransitionBody::Cancel { cancellation, .. }
                if cancellation.status == CancelStatus::Won =>
            {
                self.snapshot_bytes += if cancellation.event_id == 0 { 1 } else { 4 };
            }
            TransitionBody::Quiesce { result } => {
                self.snapshot_bytes += result.deliveries.len() as u64 * 6;
            }
            TransitionBody::Cancel { .. } => {}
        }
        self.transcript = transition.digest;
        self.history.push(transition);
        self.next_transition = self.next_transition.saturating_add(1);
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        {
            let transition = self.history.last().unwrap();
            // The broker's shared clock timestamps the commit. World.now is
            // only its delivery cursor and can lag until quiesce commits.
            crate::event_order::emit_current_context(
                "world-order",
                transition.sequence,
                serde_json::json!({
                    "transition": transition.body.kind(),
                    "digest": transition.digest,
                    "world_time_nanos": self.now - INITIAL_TIME,
                }),
            );
        }
    }

    pub fn register(&mut self, request: Request) -> Result<u64> {
        validate_request(&self.config.limits, &request)?;
        if self.next_request_id == 0 || self.next_request_id == u64::MAX {
            return Err(capacity("request-ids", u64::MAX, u64::MAX, 1));
        }
        check_capacity(
            "requests",
            self.config.limits.max_requests,
            self.requests.len() as u64,
            1,
        )?;
        let payload_delta = request.payload.len() as u64 * 2;
        self.check_payload(payload_delta)?;
        self.check_transition()?;
        let id = self.next_request_id;
        let transition = self.next_transition(TransitionBody::Register {
            request: request.clone(),
            request_id: id,
        })?;
        self.requests.insert(
            id,
            RequestEntry {
                request,
                state: RequestState::Pending,
                event_id: 0,
            },
        );
        self.next_request_id += 1;
        self.payload_bytes += payload_delta;
        self.commit(transition);
        Ok(id)
    }

    fn lookup_pending(&self, id: u64) -> Result<&RequestEntry> {
        let request = self
            .requests
            .get(&id)
            .filter(|_| id != 0)
            .ok_or_else(|| Error::classified(ErrorClass::UnknownRequest, "requestId"))?;
        if request.state != RequestState::Pending {
            return Err(Error::classified(
                ErrorClass::RequestState,
                format!("request {id} is {}", request.state.as_str()),
            ));
        }
        Ok(request)
    }

    pub fn ready(&mut self, readiness: Readiness) -> Result<u64> {
        let request = &self.lookup_pending(readiness.request_id)?.request;
        let (priority, adapter, resource_kind, key, request_kind) = (
            request.priority,
            request.resource.adapter.clone(),
            request.resource.kind.clone(),
            request.resource.key.clone(),
            request.kind.clone(),
        );
        let limits = &self.config.limits;
        validate_readiness(limits, &readiness)?;
        if readiness.at < self.now {
            return Err(Error::classified(
                ErrorClass::TimeRegression,
                "readiness.at",
            ));
        }
        if self.next_event_id == 0 || self.next_event_id == u64::MAX {
            return Err(capacity("event-ids", u64::MAX, u64::MAX, 1));
        }
        check_capacity("events", limits.max_events, self.events.len() as u64, 1)?;
        check_capacity(
            "queued-events",
            limits.max_queued_events,
            self.queue.len() as u64,
            1,
        )?;
        let payload_delta = readiness.payload.len() as u64 * 2;
        self.check_payload(payload_delta)?;
        self.check_transition()?;
        let id = self.next_event_id;
        let rank = if readiness.equivalence_class.is_empty() {
            [0; 32]
        } else {
            choice_rank(self.config.seed, &readiness.equivalence_class, id)
        };
        let key = QueueKey {
            at: readiness.at,
            priority,
            adapter,
            resource_kind,
            key,
            request_kind,
            readiness_kind: readiness.kind.clone(),
            equivalence_class: readiness.equivalence_class.clone(),
            rank,
            request_id: readiness.request_id,
        };
        let transition = self.next_transition(TransitionBody::Ready {
            readiness: readiness.clone(),
            event_id: id,
        })?;
        let entry = self.requests.get_mut(&readiness.request_id).unwrap();
        entry.state = RequestState::Queued;
        entry.event_id = id;
        self.queue.insert(key.clone());
        self.events.insert(
            id,
            Event {
                readiness,
                state: EventState::Queued,
                key: Some(key),
            },
        );
        self.next_event_id += 1;
        self.payload_bytes += payload_delta;
        self.commit(transition);
        Ok(id)
    }

    pub fn cancel(&mut self, request_id: u64) -> Result<Cancellation> {
        let request = self
            .requests
            .get(&request_id)
            .filter(|_| request_id != 0)
            .ok_or_else(|| Error::classified(ErrorClass::UnknownRequest, "requestId"))?;
        let (state, event_id) = (request.state, request.event_id);
        self.check_transition()?;
        let status = match state {
            RequestState::Pending | RequestState::Queued => CancelStatus::Won,
            RequestState::Canceled => CancelStatus::AlreadyCanceled,
            RequestState::Delivered => CancelStatus::AlreadyDelivered,
        };
        let result = Cancellation {
            request_id,
            event_id,
            status,
        };
        let transition = self.next_transition(TransitionBody::Cancel {
            request_id,
            cancellation: result,
        })?;
        if status == CancelStatus::Won {
            self.requests.get_mut(&request_id).unwrap().state = RequestState::Canceled;
            if event_id != 0 {
                let event = self.events.get_mut(&event_id).unwrap();
                let removed = event.key.take().is_some_and(|key| self.queue.remove(&key));
                assert!(
                    event.state == EventState::Queued && removed,
                    "queued request has no queued event"
                );
                event.state = EventState::Canceled;
            }
        }
        self.commit(transition);
        Ok(result)
    }

    pub fn quiesce(&mut self) -> Result<Quiescence> {
        self.check_transition()?;
        let mut result = Quiescence {
            kind: QuiescenceKind::Idle,
            before: self.now,
            after: self.now,
            deliveries: Vec::new(),
            blocked: Vec::new(),
        };
        let mut batch = Vec::new();
        if let Some(first) = self.queue.first() {
            let at = first.at;
            let mut payload_delta = 0u64;
            for key in self.queue.iter().take_while(|key| key.at == at) {
                let event_id = self.requests[&key.request_id].event_id;
                let event = &self.events[&event_id];
                payload_delta += event.readiness.payload.len() as u64;
                result.deliveries.push(Delivery {
                    request_id: key.request_id,
                    event_id,
                    at,
                    kind: event.readiness.kind.clone(),
                    payload: event.readiness.payload.clone(),
                });
                batch.push(key.clone());
            }
            result.kind = QuiescenceKind::Delivered;
            result.after = at;
            self.check_payload(payload_delta)?;
        } else {
            result.blocked = self
                .requests
                .iter()
                .filter(|(_, request)| request.state == RequestState::Pending)
                .map(|(id, _)| *id)
                .collect();
            if !result.blocked.is_empty() {
                result.kind = QuiescenceKind::Deadlock;
            }
        }
        let transition = self.next_transition(TransitionBody::Quiesce {
            result: result.clone(),
        })?;
        if result.kind == QuiescenceKind::Delivered {
            for (key, delivery) in batch.iter().zip(&result.deliveries) {
                self.queue.remove(key);
                let event = self.events.get_mut(&delivery.event_id).unwrap();
                event.key = None;
                event.state = EventState::Delivered;
                self.requests.get_mut(&delivery.request_id).unwrap().state =
                    RequestState::Delivered;
                self.payload_bytes += delivery.payload.len() as u64;
            }
            self.now = result.after;
        }
        self.commit(transition);
        Ok(result)
    }
}

#[derive(Default)]
struct WireSize(u64);

impl std::io::Write for WireSize {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        self.0 += bytes.len() as u64;
        Ok(bytes.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests;

#[cfg(all(test, not(feature = "conformance")))]
mod owner_preservation_tests;
