//! Generated protobuf checker guest and its bounded core-wasm ABI.
#![cfg(not(test))]

use prost::Message;
use std::{cell::RefCell, collections::BTreeMap, fmt::Write};
use umpire_checker::{Checker, StateId};
use umpire_dsl::Spell;

mod refusal;
use refusal::{checker_refusal, refused};

#[path = "../../tests/fixtures/sample_kettle_trait/model.rs"]
#[allow(missing_docs, unreachable_pub)]
mod kettle;

/// Generated core contract and shared well-known types.
#[allow(missing_docs, unreachable_pub, clippy::all, clippy::pedantic)]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/protocol.rs"));
}

pub use generated::{google, umpire::model::v1 as protocol};

/// One guest instance's current exploration.
#[derive(Default)]
pub struct Model {
    exploration: Option<umpire_checker::Result<kettle::KettleWorld>>,
}

impl Model {
    /// Dispatches a generated request and returns its generated response envelope.
    pub fn dispatch(&mut self, method: u32, request: &[u8]) -> protocol::Response {
        let outcome = match method {
            1 => protocol::ExploreRequest::decode(request)
                .map_err(|error| {
                    refused(protocol::RefusalCode::MalformedRequest, error.to_string())
                })
                .and_then(|request| self.explore(request.budget)),
            2 => protocol::RowsRequest::decode(request)
                .map_err(|error| {
                    refused(protocol::RefusalCode::MalformedRequest, error.to_string())
                })
                .and_then(|request| self.rows(request.state_id)),
            3 => decode_unsupported::<protocol::ResetRequest>(request, |_| true),
            4 => decode_unsupported::<protocol::JudgeRequest>(request, |request| {
                request.observation.is_some()
            }),
            5 => decode_unsupported::<protocol::AnswerRequest>(request, |request| {
                request.callback.is_some()
            }),
            _ => Err(refused(
                protocol::RefusalCode::UnknownMethod,
                format!("unknown method: {method}"),
            )),
        };
        protocol::Response { outcome: Some(outcome.unwrap_or_else(|refusal| refusal)) }
    }

    fn explore(
        &mut self,
        budget: u32,
    ) -> Result<protocol::response::Outcome, protocol::response::Outcome> {
        let result = Checker::explore(&kettle::KettleWorld, budget as usize)
            .map_err(|error| checker_refusal(&error))?;
        let report = result.verify_all().map_err(|error| checker_refusal(&error))?;
        let mut text = String::new();
        report.spell(&mut text);
        let counts = result.counts;
        let response = protocol::ExploreResponse {
            report: text,
            starts: count(counts.starts),
            reached: count(counts.reached),
            emitted: count(counts.emitted),
            evaluated: count(counts.evaluated),
            refused: count(counts.refused),
            depth: count(counts.depth),
            cut: counts.cut,
        };
        self.exploration = Some(result);
        Ok(protocol::response::Outcome::Explore(response))
    }

    fn rows(
        &self,
        state_id: u32,
    ) -> Result<protocol::response::Outcome, protocol::response::Outcome> {
        let result = self.exploration.as_ref().ok_or_else(|| {
            refused(protocol::RefusalCode::NotExplored, "Rows requires a successful Explore".into())
        })?;
        let mut rows = Vec::new();
        for edge in
            result.rows(StateId(state_id as usize)).map_err(|error| checker_refusal(&error))?
        {
            let mut row = format!("{state_id}: ");
            edge.action.spell(&mut row);
            write!(row, " choice={} outcome=", edge.choice).ok();
            ().spell(&mut row);
            row.push_str(" facts=");
            edge.emitted.1.spell(&mut row);
            row.push_str(" -> ");
            edge.emitted.2.spell(&mut row);
            row.push('\n');
            rows.push(row);
        }
        Ok(protocol::response::Outcome::Rows(protocol::RowsResponse { rows }))
    }
}

fn decode_unsupported<M: Message + Default>(
    request: &[u8],
    complete: impl FnOnce(&M) -> bool,
) -> Result<protocol::response::Outcome, protocol::response::Outcome> {
    let request = M::decode(request)
        .map_err(|error| refused(protocol::RefusalCode::MalformedRequest, error.to_string()))?;
    if !complete(&request) {
        return Err(refused(
            protocol::RefusalCode::MalformedRequest,
            "required payload is missing".into(),
        ));
    }
    Err(refused(protocol::RefusalCode::Unsupported, "stateful operation unsupported".into()))
}

fn count(value: usize) -> u32 {
    u32::try_from(value).unwrap_or_else(|_| unreachable!("finite Kettle counts fit u32"))
}

const MAX_BUFFER: usize = 1 << 20;
const MAX_LIVE: usize = 16 << 20;

#[derive(Default)]
struct Guest {
    model: Model,
    buffers: BTreeMap<u32, Vec<u8>>,
    live: usize,
}

impl Guest {
    fn allocate(&mut self, len: usize) -> u32 {
        if len == 0 || len > MAX_BUFFER || len > MAX_LIVE - self.live {
            return 0;
        }
        let mut buffer = Vec::new();
        if buffer.try_reserve_exact(len).is_err() {
            return 0;
        }
        buffer.resize(len, 0);
        let Ok(ptr) = u32::try_from(buffer.as_ptr() as usize) else { return 0 };
        if ptr == 0 {
            return 0;
        }
        self.buffers.insert(ptr, buffer);
        self.live += len;
        ptr
    }

    fn release(&mut self, ptr: u32, len: u32) {
        if (ptr, len) == (0, 0) {
            return;
        }
        assert!(
            self.buffers.get(&ptr).is_some_and(|buffer| buffer.len() == len as usize),
            "umpire_buffer_release requires an exact live allocation"
        );
        self.buffers.remove(&ptr);
        self.live -= len as usize;
    }

    fn call(&mut self, method: u32, ptr: u32, len: u32) -> u64 {
        let previous = if method == 1 { self.model.exploration.take() } else { None };
        let response = if (ptr, len) == (0, 0) {
            self.model.dispatch(method, &[])
        } else if let Some(buffer) =
            self.buffers.get(&ptr).filter(|buffer| buffer.len() == len as usize)
        {
            self.model.dispatch(method, buffer)
        } else {
            protocol::Response {
                outcome: Some(refused(
                    protocol::RefusalCode::Allocation,
                    "umpire_call requires an exact live allocation".into(),
                )),
            }
        };
        let size = response.encoded_len();
        let ptr = self.allocate(size);
        if method == 1
            && (ptr == 0
                || !matches!(response.outcome, Some(protocol::response::Outcome::Explore(_))))
        {
            self.model.exploration = previous;
        }
        if ptr == 0 {
            return 0;
        }
        let Some(buffer) = self.buffers.get_mut(&ptr) else {
            unreachable!("response allocation is live")
        };
        let mut destination = buffer.as_mut_slice();
        if response.encode(&mut destination).is_err() {
            unreachable!("response allocation fits encoded message")
        }
        (u64::from(count(size)) << 32) | u64::from(ptr)
    }
}

thread_local! {
    static GUEST: RefCell<Guest> = RefCell::new(Guest::default());
}

/// Allocates a zeroed, bounded request buffer, or returns zero on exhaustion.
#[allow(unsafe_code)]
#[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
pub extern "C" fn umpire_buffer_alloc(len: u32) -> u32 {
    GUEST.with_borrow_mut(|guest| guest.allocate(len as usize))
}

/// Releases an exact live allocation; invalid ranges trap.
#[allow(unsafe_code)]
#[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
pub extern "C" fn umpire_buffer_release(ptr: u32, len: u32) {
    GUEST.with_borrow_mut(|guest| guest.release(ptr, len));
}

/// Dispatches a request and packs a separately owned response's length and pointer.
#[allow(unsafe_code)]
#[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
pub extern "C" fn umpire_call(method: u32, ptr: u32, len: u32) -> u64 {
    GUEST.with_borrow_mut(|guest| guest.call(method, ptr, len))
}
