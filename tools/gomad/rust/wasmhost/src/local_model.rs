//! Pure clock and poll semantics of the helper-owned model. The helper model
//! and the test-only conformance driver both call these functions, so the Go
//! differential exercises the production rules.
use crate::namespace::{Boundary, Namespace};
use crate::transcript::{CPU_POLL, Event, Subscription};

pub const MAX_TIME: u64 = i64::MAX as u64;

/// The timestamp a successful clock read returns at model position `now`, or
/// `None` when advancing by `step` would leave the representable range.
pub fn clock_timestamp(now: u64, epoch: u64, step: u64, clock_id: u32) -> Option<u64> {
    if now > MAX_TIME - epoch || step > MAX_TIME - epoch - now {
        return None;
    }
    Some(now + if clock_id == 0 { epoch } else { 0 })
}

pub struct Poll {
    pub events: Vec<Event>,
    pub errno: u16,
    pub unsupported: &'static str,
    /// Model position after the poll; unchanged unless `errno` is zero.
    pub now: u64,
}

#[derive(Debug)]
pub enum PollError {
    Capacity(Boundary),
    /// A descriptor subscription without a helper-owned namespace.
    Descriptor,
}

/// Answers `poll_oneoff`: descriptor subscriptions complete at `now`; clock
/// subscriptions complete at their deadline, and the earliest deadline becomes
/// the new model position.
pub fn poll(
    now: u64,
    epoch: u64,
    pending_events: u64,
    subscriptions: &[Subscription],
    namespace: Option<&Namespace>,
) -> Result<Poll, PollError> {
    let failed = |errno, unsupported| Poll {
        events: Vec::new(),
        errno,
        unsupported,
        now,
    };
    if subscriptions.is_empty() {
        return Ok(failed(28, ""));
    }
    if subscriptions.len() as u64 > pending_events {
        return Err(PollError::Capacity(Boundary {
            kind: "capacity",
            dimension: "pending-events",
        }));
    }
    let mut deadlines = Vec::with_capacity(subscriptions.len());
    let mut events = Vec::with_capacity(subscriptions.len());
    let mut earliest = u64::MAX;
    for sub in subscriptions {
        let mut event = Event {
            userdata: sub.userdata,
            errno: 0,
            kind: sub.kind,
            nbytes: 0,
            flags: 0,
        };
        if sub.kind != 0 {
            if sub.kind > 2 {
                return Ok(failed(28, ""));
            }
            let namespace = namespace.ok_or(PollError::Descriptor)?;
            let ready = namespace.fd_readiness(sub.fd, sub.kind == 2);
            event.errno = ready.errno;
            event.nbytes = ready.nbytes;
            event.flags = ready.flags;
            earliest = earliest.min(now);
            deadlines.push(now);
            events.push(event);
            continue;
        }
        if sub.clock_id > 1 {
            return Ok(failed(0, CPU_POLL));
        }
        if sub.flags & !1 != 0 {
            return Ok(failed(28, ""));
        }
        let deadline = if sub.flags == 1 {
            (if sub.clock_id == 0 {
                sub.timeout.saturating_sub(epoch)
            } else {
                sub.timeout
            })
            .max(now)
        } else {
            if now > MAX_TIME - epoch || sub.timeout > MAX_TIME - epoch - now {
                return Ok(failed(28, ""));
            }
            now + sub.timeout
        };
        if deadline > MAX_TIME - epoch {
            return Ok(failed(28, ""));
        }
        earliest = earliest.min(deadline);
        deadlines.push(deadline);
        events.push(event);
    }
    let events = events
        .into_iter()
        .zip(deadlines)
        .filter(|(_, at)| *at <= earliest)
        .map(|(event, _)| event)
        .collect();
    Ok(Poll {
        events,
        errno: 0,
        unsupported: "",
        now: earliest,
    })
}
