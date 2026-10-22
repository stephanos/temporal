//! Canonical framing and digests: Go `snapshot.go` `appendX`, `stateDigest`
//! and `replay.go` `transitionBytes`/`transitionDigest`.
use super::{
    Config, Event, Quiescence, Readiness, Request, RequestEntry, SCHEMA_VERSION, Transition,
    TransitionBody,
};
use sha2::{Digest, Sha256};

pub fn hex(digest: &[u8; 32]) -> String {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut text = String::with_capacity(64);
    for byte in digest {
        text.push(DIGITS[usize::from(byte >> 4)] as char);
        text.push(DIGITS[usize::from(byte & 15)] as char);
    }
    text
}

pub fn empty_digest() -> [u8; 32] {
    Sha256::digest([]).into()
}

pub fn append_u64(data: &mut Vec<u8>, value: u64) {
    data.extend_from_slice(&value.to_be_bytes());
}

pub fn append_u32(data: &mut Vec<u8>, value: u32) {
    data.extend_from_slice(&value.to_be_bytes());
}

pub fn append_bytes(data: &mut Vec<u8>, value: &[u8]) {
    append_u64(data, value.len() as u64);
    data.extend_from_slice(value);
}

/// Digests are framed as their 64-character lowercase hex text, as in Go.
pub fn append_digest(data: &mut Vec<u8>, value: &[u8; 32]) {
    append_bytes(data, hex(value).as_bytes());
}

fn append_config(data: &mut Vec<u8>, config: &Config) {
    append_u64(data, config.seed);
    let limits = &config.limits;
    append_u64(data, limits.max_requests);
    append_u64(data, limits.max_events);
    append_u64(data, limits.max_queued_events);
    append_u64(data, limits.max_transitions);
    append_u64(data, limits.max_payload_bytes);
    append_u32(data, limits.max_string_bytes);
}

pub(super) fn inspection_commitment(model: &super::Model) -> [u8; 32] {
    let mut data = b"gomad/world/inspection-chain/v2\0".to_vec();
    append_u32(&mut data, SCHEMA_VERSION);
    append_config(&mut data, model.config());
    append_u64(&mut data, model.now().cast_unsigned());
    let (request, event, transition) = model.next_ids();
    for value in [
        request,
        event,
        transition,
        model.payload_bytes(),
        model.snapshot_bytes(),
    ] {
        append_u64(&mut data, value);
    }
    append_digest(&mut data, model.transcript_digest());
    Sha256::digest(data).into()
}

pub fn append_request(data: &mut Vec<u8>, request: &Request) {
    append_bytes(data, request.kind.as_bytes());
    append_bytes(data, request.resource.adapter.as_bytes());
    append_bytes(data, request.resource.kind.as_bytes());
    append_bytes(data, request.resource.key.as_bytes());
    append_u32(data, u32::from(request.priority));
    append_bytes(data, &request.payload);
}

pub fn append_readiness(data: &mut Vec<u8>, readiness: &Readiness) {
    append_u64(data, readiness.request_id);
    append_u64(data, readiness.at.cast_unsigned());
    append_bytes(data, readiness.kind.as_bytes());
    append_bytes(data, &readiness.payload);
    append_bytes(data, readiness.equivalence_class.as_bytes());
}

pub fn append_quiescence(data: &mut Vec<u8>, quiescence: &Quiescence) {
    append_bytes(data, quiescence.kind.as_str().as_bytes());
    append_u64(data, quiescence.before.cast_unsigned());
    append_u64(data, quiescence.after.cast_unsigned());
    append_u64(data, quiescence.deliveries.len() as u64);
    for delivery in &quiescence.deliveries {
        append_u64(data, delivery.request_id);
        append_u64(data, delivery.event_id);
        append_u64(data, delivery.at.cast_unsigned());
        append_bytes(data, delivery.kind.as_bytes());
        append_bytes(data, &delivery.payload);
    }
    append_u64(data, quiescence.blocked.len() as u64);
    for blocked in &quiescence.blocked {
        append_u64(data, *blocked);
    }
}

pub fn transition_bytes(transition: &Transition, include_digest: bool) -> Vec<u8> {
    let mut data = Vec::with_capacity(256);
    append_u64(&mut data, transition.sequence);
    append_bytes(&mut data, transition.body.kind().as_bytes());
    append_digest(&mut data, &transition.previous_digest);
    if include_digest {
        append_digest(&mut data, &transition.digest);
    }
    match &transition.body {
        TransitionBody::Register {
            request,
            request_id,
        } => {
            append_request(&mut data, request);
            append_u64(&mut data, *request_id);
        }
        TransitionBody::Ready {
            readiness,
            event_id,
        } => {
            append_readiness(&mut data, readiness);
            append_u64(&mut data, *event_id);
        }
        TransitionBody::Cancel {
            request_id,
            cancellation,
        } => {
            append_u64(&mut data, *request_id);
            append_u64(&mut data, cancellation.request_id);
            append_u64(&mut data, cancellation.event_id);
            append_bytes(&mut data, cancellation.status.as_str().as_bytes());
        }
        TransitionBody::Quiesce { result } => append_quiescence(&mut data, result),
    }
    data
}

pub fn transition_digest(transition: &Transition) -> [u8; 32] {
    let mut data = b"gomad/world/transcript/v1\0".to_vec();
    append_digest(&mut data, &transition.previous_digest);
    data.extend_from_slice(&transition_bytes(transition, false));
    Sha256::digest(&data).into()
}

pub struct StateInput<'a, R, E> {
    pub config: &'a Config,
    pub now: i64,
    pub next_request_id: u64,
    pub next_event_id: u64,
    pub next_transition: u64,
    pub payload_bytes: u64,
    pub requests: R,
    pub events: E,
    pub transitions: &'a [Transition],
    pub transcript: &'a [u8; 32],
}

/// Go `stateDigest`, streamed in the same byte order.
#[cfg(test)]
pub fn state_digest<'a, R, E>(input: StateInput<'a, R, E>) -> [u8; 32]
where
    R: ExactSizeIterator<Item = (u64, &'a RequestEntry)>,
    E: ExactSizeIterator<Item = (u64, &'a Event)>,
{
    state_digest_framed(input, &[], 0)
}

fn hash_transition_frame(hasher: &mut Sha256, data: &mut Vec<u8>, transition: &Transition) {
    let encoded = transition_bytes(transition, true);
    data.clear();
    append_u64(data, encoded.len() as u64);
    hasher.update(&*data);
    hasher.update(&encoded);
}

/// One transition's [`state_digest`] framing: its length and encoding.
pub fn append_transition_frame(data: &mut Vec<u8>, transition: &Transition) {
    let encoded = transition_bytes(transition, true);
    append_u64(data, encoded.len() as u64);
    data.extend_from_slice(&encoded);
}

/// [`state_digest`] with the frames of the first `framed` transitions
/// already concatenated in `frames`; the remaining transitions are framed
/// while streaming into the hasher, without being retained.
pub fn state_digest_framed<'a, R, E>(
    input: StateInput<'a, R, E>,
    frames: &[u8],
    framed_count: usize,
) -> [u8; 32]
where
    R: ExactSizeIterator<Item = (u64, &'a RequestEntry)>,
    E: ExactSizeIterator<Item = (u64, &'a Event)>,
{
    let rest = &input.transitions[framed_count..];
    state_digest_with(input, |hasher, data| {
        hasher.update(frames);
        for transition in rest {
            hash_transition_frame(hasher, data, transition);
        }
    })
}

fn state_digest_with<'a, R, E>(
    input: StateInput<'a, R, E>,
    transitions: impl FnOnce(&mut Sha256, &mut Vec<u8>),
) -> [u8; 32]
where
    R: ExactSizeIterator<Item = (u64, &'a RequestEntry)>,
    E: ExactSizeIterator<Item = (u64, &'a Event)>,
{
    let mut hasher = Sha256::new();
    let mut data = b"gomad/world/state/v1\0".to_vec();
    append_u32(&mut data, SCHEMA_VERSION);
    append_config(&mut data, input.config);
    append_u64(&mut data, input.now.cast_unsigned());
    append_u64(&mut data, input.next_request_id);
    append_u64(&mut data, input.next_event_id);
    append_u64(&mut data, input.next_transition);
    append_u64(&mut data, input.payload_bytes);
    append_u64(&mut data, input.requests.len() as u64);
    hasher.update(&data);
    for (id, request) in input.requests {
        data.clear();
        append_u64(&mut data, id);
        append_request(&mut data, &request.request);
        append_bytes(&mut data, request.state.as_str().as_bytes());
        append_u64(&mut data, request.event_id);
        hasher.update(&data);
    }
    data.clear();
    append_u64(&mut data, input.events.len() as u64);
    hasher.update(&data);
    for (id, event) in input.events {
        data.clear();
        append_u64(&mut data, id);
        append_readiness(&mut data, &event.readiness);
        append_bytes(&mut data, event.state.as_str().as_bytes());
        hasher.update(&data);
    }
    data.clear();
    append_u64(&mut data, input.transitions.len() as u64);
    hasher.update(&data);
    transitions(&mut hasher, &mut data);
    data.clear();
    append_digest(&mut data, input.transcript);
    hasher.update(&data);
    hasher.finalize().into()
}
