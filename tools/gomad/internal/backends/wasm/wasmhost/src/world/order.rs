//! Delivery order: Go `queue.go` `lessEvent` and `choice.go` `choiceRank`.
use sha2::{Digest, Sha256};
use std::cmp::Ordering;

/// One queued event's position. `Ord` is exactly Go's `lessEvent`: time,
/// priority, resource adapter/kind/key, request kind, readiness kind,
/// equivalence class, then the seeded rank (only for non-empty classes; empty
/// classes carry a zero rank, so equal keys fall through) and request ID.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct QueueKey {
    pub at: i64,
    pub priority: u16,
    pub adapter: String,
    pub resource_kind: String,
    pub key: String,
    pub request_kind: String,
    pub readiness_kind: String,
    pub equivalence_class: String,
    pub rank: [u8; 32],
    pub request_id: u64,
}

impl Ord for QueueKey {
    fn cmp(&self, other: &Self) -> Ordering {
        self.at
            .cmp(&other.at)
            .then(self.priority.cmp(&other.priority))
            .then_with(|| self.adapter.cmp(&other.adapter))
            .then_with(|| self.resource_kind.cmp(&other.resource_kind))
            .then_with(|| self.key.cmp(&other.key))
            .then_with(|| self.request_kind.cmp(&other.request_kind))
            .then_with(|| self.readiness_kind.cmp(&other.readiness_kind))
            .then_with(|| self.equivalence_class.cmp(&other.equivalence_class))
            .then(self.rank.cmp(&other.rank))
            .then(self.request_id.cmp(&other.request_id))
    }
}

impl PartialOrd for QueueKey {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

/// HMAC-SHA256 (RFC 2104) over the pinned `sha2`, for keys up to one block.
pub fn hmac_sha256(key: &[u8], message: &[&[u8]]) -> [u8; 32] {
    const BLOCK: usize = 64;
    let mut padded = [0u8; BLOCK];
    if key.len() > BLOCK {
        padded[..32].copy_from_slice(&Sha256::digest(key));
    } else {
        padded[..key.len()].copy_from_slice(key);
    }
    let mut inner = Sha256::new();
    inner.update(padded.map(|byte| byte ^ 0x36));
    for part in message {
        inner.update(part);
    }
    let mut outer = Sha256::new();
    outer.update(padded.map(|byte| byte ^ 0x5c));
    outer.update(inner.finalize());
    outer.finalize().into()
}

/// Go `choiceRank`: the seeded order of events within one equivalence class.
pub fn choice_rank(seed: u64, equivalence_class: &str, event_id: u64) -> [u8; 32] {
    let mut seed_input = b"gomad/world/seed/v1\0".to_vec();
    seed_input.extend_from_slice(&seed.to_be_bytes());
    let key = Sha256::digest(&seed_input);
    hmac_sha256(
        &key,
        &[
            b"gomad/world/equivalent-event-order/v1\0",
            &(equivalence_class.len() as u64).to_be_bytes(),
            equivalence_class.as_bytes(),
            &event_id.to_be_bytes(),
        ],
    )
}
