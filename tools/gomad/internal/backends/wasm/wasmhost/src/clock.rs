//! Rust duplicate of the shared simulation clock (`internal/model/clock`:
//! `simulation_time.go`, `simulation_progress.go`, `simulation_broker.go`) for
//! the in-process cluster scheduler (fn-178). Go stays authoritative for the
//! native backend and the runner; the checked-in conformance corpus
//! (`tests/testdata/clock_conformance`, pinned to
//! `clock.BrokerImplementationSHA256`) keeps the two equal.
//!
//! Go parks a quiescing participant on a channel and answers it from whichever
//! goroutine settles the arbiter. Here the scheduler is the only caller
//! (fn-178.1: serialized scheduling), so `quiesce` returns at once with either
//! an answer or `Parked`, and answers produced for parked participants by later
//! calls are queued until `take_answers`.
use std::collections::BTreeMap;
use std::fmt::Write as _;

pub const INITIAL_TIME: i64 = 946_684_800_000_000_000;

pub const ADVANCE: u8 = 1;
pub const RETRY: u8 = 2;
pub const DEADLOCK: u8 = 3;
pub const EXTERNAL: u8 = 4;

/// A participant handle, the counterpart of Go's `*BrokerParticipant` pointer
/// identity: a removed participant's handle stays stale even if its name is
/// registered again.
pub type Handle = u64;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct TimeRequest {
    pub generation: u64,
    pub current: i64,
    pub deadline: i64,
    pub arrivals: u32,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct TimeResponse {
    pub generation: u64,
    pub kind: u8,
    pub time: i64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Quiesced {
    Answered(TimeResponse),
    Parked,
}

/// An answer to a parked participant, Go's value sent on its waiter channel.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Answer {
    pub handle: Handle,
    pub result: Result<TimeResponse, String>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Phase {
    ResponseReserved,
    WaitReserved,
    WaitSuspended,
}

#[derive(Debug)]
struct Participant {
    name: String,
    active: bool,
    external: u64,
    delivered: u64,
    handling: u64,
    generation: u64,
    deadline: i64,
    waiting: bool,
}

/// The progress events of `simulation_progress.go`, by their Go facade names.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[allow(
    dead_code,
    reason = "Progress and clock entry points retained for the differential clock corpus."
)]
pub enum Event {
    Accepted {
        participant: Option<Handle>,
        request: u64,
        arrivals: u32,
    },
    ControlForwarded {
        participant: Option<Handle>,
        node: Option<Handle>,
        request: u64,
        arrivals: u32,
    },
    WaitAccepted {
        participant: Option<Handle>,
        request: u64,
        arrivals: u32,
    },
    WaitSuspended {
        participant: Option<Handle>,
        request: u64,
    },
    WaitResumed {
        participant: Option<Handle>,
        request: u64,
    },
    CompletionReserved {
        participant: Option<Handle>,
        request: u64,
    },
    Delivered {
        participant: Option<Handle>,
        request: u64,
    },
    RequestAccepted {
        participant: Option<Handle>,
        arrivals: u32,
    },
    ResponseDelivered {
        participant: Option<Handle>,
    },
    Forwarded {
        source: Option<Handle>,
        destination: Option<Handle>,
        arrivals: u32,
        current: i64,
    },
    Dispatched {
        coordinator: Option<Handle>,
    },
    DispatchUnavailable {
        coordinator: Option<Handle>,
    },
    ResponseArrived {
        coordinator: Option<Handle>,
        node: Option<Handle>,
        arrivals: u32,
    },
    AbandonedResponseDiscarded {
        coordinator: Option<Handle>,
        arrivals: u32,
    },
    Consume {
        participant: Option<Handle>,
        arrivals: u32,
    },
    Runnable {
        participant: Option<Handle>,
    },
    Remove {
        participant: Option<Handle>,
    },
}

/// Go `%q` for the participant names the cluster uses.
fn quote(name: &str) -> String {
    let mut quoted = String::with_capacity(name.len() + 2);
    quoted.push('"');
    for character in name.chars() {
        match character {
            '"' => quoted.push_str("\\\""),
            '\\' => quoted.push_str("\\\\"),
            '\n' => quoted.push_str("\\n"),
            '\t' => quoted.push_str("\\t"),
            '\r' => quoted.push_str("\\r"),
            '\u{7}' => quoted.push_str("\\a"),
            '\u{8}' => quoted.push_str("\\b"),
            '\u{c}' => quoted.push_str("\\f"),
            '\u{b}' => quoted.push_str("\\v"),
            character if u32::from(character) < 0x80 && character.is_control() => {
                write!(quoted, "\\x{:02x}", u32::from(character)).unwrap();
            }
            character if character.is_control() => {
                write!(quoted, "\\u{:04x}", u32::from(character)).unwrap();
            }
            character => quoted.push(character),
        }
    }
    quoted.push('"');
    quoted
}

const INACTIVE: &str = "simulation time participant is inactive";

pub struct Clock {
    current: i64,
    forward: bool,
    /// Live participants by handle; a removed handle is absent, so it stays
    /// inactive even when its name is registered again.
    participants: BTreeMap<Handle, Participant>,
    index: BTreeMap<String, Handle>,
    next_handle: Handle,
    responses: BTreeMap<(Handle, u64), Phase>,
    answers: Vec<Answer>,
}

impl Clock {
    /// Go `NewBrokerClock`: strict epochs, and time one past the initial
    /// instant because cooperative Go reserves zero.
    pub fn broker() -> Self {
        let mut clock = Self::process(false);
        clock.current += 1;
        clock
    }

    /// Go `NewProcessClock(forward)`.
    pub fn process(forward: bool) -> Self {
        Self {
            current: INITIAL_TIME,
            forward,
            participants: BTreeMap::new(),
            index: BTreeMap::new(),
            next_handle: 1,
            responses: BTreeMap::new(),
            answers: Vec::new(),
        }
    }

    pub fn current(&self) -> i64 {
        self.current
    }

    #[allow(
        dead_code,
        reason = "Progress and clock entry points retained for the differential clock corpus."
    )]
    pub fn is_quiescent(&self, handle: Option<Handle>) -> bool {
        self.live(handle)
            .is_some_and(|participant| participant.waiting)
    }

    /// Answers produced for parked participants since the last call.
    pub fn take_answers(&mut self) -> Vec<Answer> {
        std::mem::take(&mut self.answers)
    }

    /// A live (current) participant; Go's `participants[p.name] == p`.
    fn live(&self, handle: Option<Handle>) -> Option<&Participant> {
        self.participants.get(&handle?)
    }

    fn participant(&mut self, handle: Option<Handle>) -> Result<&mut Participant, String> {
        handle
            .and_then(|handle| self.participants.get_mut(&handle))
            .ok_or_else(|| INACTIVE.into())
    }

    /// Go `BrokerClock.RegisterInactive`.
    pub fn register_inactive(&mut self, name: &str) -> Result<Handle, String> {
        if name.is_empty() || name.len() > 512 {
            return Err("simulation time participant identity is invalid".into());
        }
        if self.index.contains_key(name) {
            return Err(format!(
                "simulation time participant {} is duplicated",
                quote(name)
            ));
        }
        let handle = self.next_handle;
        self.next_handle += 1;
        self.index.insert(name.to_owned(), handle);
        self.participants.insert(
            handle,
            Participant {
                name: name.to_owned(),
                active: false,
                external: 0,
                delivered: 0,
                handling: 0,
                generation: 0,
                deadline: 0,
                waiting: false,
            },
        );
        Ok(handle)
    }

    /// Go `BrokerClock.Register`: register and activate.
    pub fn register(&mut self, name: &str) -> Result<Handle, String> {
        let handle = self.register_inactive(name)?;
        self.activate(Some(handle));
        Ok(handle)
    }

    pub fn activate(&mut self, handle: Option<Handle>) -> i64 {
        if let Ok(participant) = self.participant(handle) {
            participant.active = true;
        }
        self.current
    }

    #[allow(
        dead_code,
        reason = "Progress and clock entry points retained for the differential clock corpus."
    )]
    pub fn activate_at(&mut self, handle: Option<Handle>, current: i64) -> Result<i64, String> {
        let participant = self.live(handle).ok_or(INACTIVE)?;
        if current < INITIAL_TIME || !self.forward && current > self.current {
            return Err(format!(
                "simulation activation time does not match the current epoch: participant={} current={current} cluster={}",
                quote(&participant.name),
                self.current
            ));
        }
        self.current = self.current.max(current);
        self.participant(handle)?.active = true;
        Ok(self.current)
    }

    /// Go `simulationTimeArbiter.quiesce` without the blocking wait.
    pub fn quiesce(
        &mut self,
        handle: Option<Handle>,
        request: TimeRequest,
    ) -> Result<Quiesced, String> {
        let (current, forward) = (self.current, self.forward);
        let participant = self.live(handle).ok_or(INACTIVE)?;
        debug_assert!(
            self.answers
                .iter()
                .all(|answer| Some(answer.handle) != handle),
            "take_answers must drain a participant's answer before it quiesces again"
        );
        if participant.waiting {
            return Err("simulation time participant is already quiescent".into());
        }
        if request.generation != participant.generation.wrapping_add(1)
            || request.deadline < request.current
            || !forward && request.current > current
        {
            return Err(format!(
                "simulation time request does not match the current epoch: participant={} active={} generation={} want={} current={} cluster={current} deadline={}",
                quote(&participant.name),
                participant.active,
                request.generation,
                participant.generation.wrapping_add(1),
                request.current,
                request.deadline
            ));
        }
        self.validate_credits(handle, request.arrivals)?;
        self.current = self.current.max(request.current);
        let current = self.current;
        let participant = self.participant(handle)?;
        participant.generation = request.generation;
        consume(participant, request.arrivals);
        let answer = |kind| {
            Ok(Quiesced::Answered(TimeResponse {
                generation: request.generation,
                kind,
                time: current,
            }))
        };
        if request.current < current && !forward || !participant.active {
            return answer(ADVANCE);
        }
        if participant.delivered != 0 {
            return answer(RETRY);
        }
        if participant.external != 0 {
            return answer(EXTERNAL);
        }
        participant.deadline = request.deadline;
        participant.waiting = true;
        self.settle();
        // Settling may answer this very call; Go hands it back on the waiter.
        if let Some(index) = self
            .answers
            .iter()
            .position(|answer| Some(answer.handle) == handle)
        {
            let answer = self.answers.remove(index);
            return answer.result.map(Quiesced::Answered);
        }
        Ok(Quiesced::Parked)
    }

    /// Go's context cancellation of a parked quiesce: the waiter is dropped
    /// without settling, and the caller sees `context canceled`.
    #[allow(
        dead_code,
        reason = "Progress and clock entry points retained for the differential clock corpus."
    )]
    pub fn abandon(&mut self, handle: Option<Handle>) -> Option<Answer> {
        let participant = self.participant(handle).ok()?;
        if !participant.waiting {
            return None;
        }
        participant.waiting = false;
        Some(Answer {
            handle: handle?,
            result: Err("context canceled".into()),
        })
    }

    fn answer(&mut self, handle: Option<Handle>, kind: u8) {
        let current = self.current;
        if let Ok(participant) = self.participant(handle)
            && participant.waiting
        {
            participant.waiting = false;
            let response = TimeResponse {
                generation: participant.generation,
                kind,
                time: current,
            };
            self.answers.push(Answer {
                handle: handle.unwrap(),
                result: Ok(response),
            });
        }
    }

    fn runnable(&mut self, handle: Option<Handle>) {
        self.answer(handle, RETRY);
    }

    fn external(&mut self, handle: Option<Handle>) {
        self.answer(handle, EXTERNAL);
    }

    fn validate_participant(&self, handle: Option<Handle>) -> Result<(), String> {
        self.live(handle).map(|_| ()).ok_or_else(|| INACTIVE.into())
    }

    fn validate_credits(&self, handle: Option<Handle>, arrivals: u32) -> Result<(), String> {
        let participant = self.live(handle).ok_or(INACTIVE)?;
        if u64::from(arrivals) > participant.delivered {
            return Err(format!(
                "simulation time request acknowledged unknown external work: participant={} arrivals={arrivals} external={} delivered={}",
                quote(&participant.name),
                participant.external,
                participant.delivered
            ));
        }
        Ok(())
    }

    fn validate_admission(
        &self,
        handle: Option<Handle>,
        request: u64,
        arrivals: u32,
    ) -> Result<(), String> {
        self.validate_credits(handle, arrivals)?;
        if self.responses.contains_key(&(handle.unwrap(), request)) {
            return Err("simulation response barrier is duplicated".into());
        }
        Ok(())
    }

    fn validate_forward(
        &self,
        source: Option<Handle>,
        arrivals: u32,
        current: i64,
        destination: Option<Handle>,
    ) -> Result<(), String> {
        if source == destination {
            return Err(INACTIVE.into());
        }
        self.validate_participant(source)?;
        self.validate_participant(destination)?;
        if current != 0 && current < INITIAL_TIME || !self.forward && current > self.current {
            return Err(format!(
                "simulation forwarded time does not match the current epoch: participant={} current={current} cluster={}",
                quote(&self.live(source).unwrap().name),
                self.current
            ));
        }
        self.validate_credits(source, arrivals)
    }

    fn validate_delivery(&self, handle: Option<Handle>) -> Result<(), String> {
        let participant = self.live(handle).ok_or(INACTIVE)?;
        if participant.delivered >= participant.external {
            return Err("simulation time external arrival is unexpected".into());
        }
        Ok(())
    }

    fn with(&mut self, handle: Option<Handle>) -> &mut Participant {
        self.participant(handle).unwrap()
    }

    fn reserve(&mut self, handle: Option<Handle>) {
        let participant = self.with(handle);
        participant.external = participant.external.wrapping_add(1);
        participant.handling = participant.handling.wrapping_add(1);
        self.external(handle);
    }

    fn deliver(&mut self, handle: Option<Handle>) {
        let participant = self.with(handle);
        participant.delivered = participant.delivered.wrapping_add(1);
        if participant.handling != 0 {
            participant.handling -= 1;
        }
        self.runnable(handle);
    }

    fn forward_request(
        &mut self,
        source: Option<Handle>,
        arrivals: u32,
        current: i64,
        destination: Option<Handle>,
    ) {
        self.current = self.current.max(current);
        let participant = self.with(source);
        consume(participant, arrivals);
        participant.external = participant.external.wrapping_add(1);
        self.external(source);
        self.reserve(destination);
    }

    /// Go `simulationProgress.apply`: validate, mutate, then settle.
    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub fn apply(&mut self, event: Event) -> Result<(), String> {
        match event {
            Event::Accepted {
                participant,
                request,
                arrivals,
            } => {
                self.validate_admission(participant, request, arrivals)?;
                consume(self.with(participant), arrivals);
                self.reserve(participant);
                self.responses
                    .insert((participant.unwrap(), request), Phase::ResponseReserved);
            }
            Event::ControlForwarded {
                participant,
                node,
                request,
                arrivals,
            } => {
                self.validate_forward(participant, arrivals, 0, node)?;
                let identity = (participant.unwrap(), request);
                if self.responses.contains_key(&identity) {
                    return Err("simulation response barrier is duplicated".into());
                }
                self.forward_request(participant, arrivals, 0, node);
                self.responses.insert(identity, Phase::ResponseReserved);
            }
            Event::WaitAccepted {
                participant,
                request,
                arrivals,
            } => {
                self.validate_admission(participant, request, arrivals)?;
                consume(self.with(participant), arrivals);
                self.runnable(participant);
                self.reserve(participant);
                self.responses
                    .insert((participant.unwrap(), request), Phase::WaitReserved);
            }
            Event::WaitSuspended {
                participant,
                request,
            } => {
                self.validate_participant(participant)?;
                let identity = (participant.unwrap(), request);
                if self.responses.get(&identity) != Some(&Phase::WaitReserved) {
                    return Err("simulation wait response barrier is unavailable".into());
                }
                let state = self.with(participant);
                if state.external <= state.delivered || state.handling == 0 {
                    return Err("simulation wait response reservation is unavailable".into());
                }
                state.external -= 1;
                state.handling -= 1;
                self.responses.insert(identity, Phase::WaitSuspended);
            }
            Event::WaitResumed {
                participant,
                request,
            } => {
                self.validate_participant(participant)?;
                let identity = (participant.unwrap(), request);
                match self.responses.get(&identity) {
                    Some(Phase::WaitReserved) => return Ok(()),
                    Some(Phase::WaitSuspended) => {}
                    _ => return Err("simulation wait response barrier is unavailable".into()),
                }
                self.responses.insert(identity, Phase::WaitReserved);
                self.reserve(participant);
            }
            Event::CompletionReserved {
                participant,
                request,
            } => {
                self.validate_participant(participant)?;
                let identity = (participant.unwrap(), request);
                if self.responses.contains_key(&identity) {
                    return Ok(());
                }
                self.responses.insert(identity, Phase::ResponseReserved);
                self.reserve(participant);
            }
            Event::Delivered {
                participant,
                request,
            } => {
                self.validate_participant(participant)?;
                let identity = (participant.unwrap(), request);
                let Some(phase) = self.responses.get(&identity).copied() else {
                    return Err("simulation response identity is unknown".into());
                };
                let state = self.with(participant);
                if phase != Phase::WaitSuspended && state.delivered >= state.external {
                    return Err("simulation time external arrival is unexpected".into());
                }
                if phase == Phase::WaitSuspended {
                    self.reserve(participant);
                }
                self.responses.remove(&identity);
                self.deliver(participant);
            }
            Event::RequestAccepted {
                participant,
                arrivals,
            } => {
                self.validate_credits(participant, arrivals)?;
                consume(self.with(participant), arrivals);
                self.reserve(participant);
            }
            Event::Forwarded {
                source,
                destination,
                arrivals,
                current,
            } => {
                self.validate_forward(source, arrivals, current, destination)?;
                self.forward_request(source, arrivals, current, destination);
            }
            Event::ResponseDelivered { participant } => {
                self.validate_delivery(participant)?;
                self.deliver(participant);
            }
            Event::Dispatched { coordinator } => {
                self.validate_delivery(coordinator)?;
                self.deliver(coordinator);
            }
            Event::DispatchUnavailable { coordinator } => {
                self.validate_participant(coordinator)?;
                let state = self.with(coordinator);
                if state.external <= state.delivered || state.handling == 0 {
                    return Err("simulation model dispatch reservation is unavailable".into());
                }
                state.external -= 1;
                state.handling -= 1;
            }
            Event::ResponseArrived {
                coordinator,
                node,
                arrivals,
            } => {
                if self.live(coordinator).is_none() {
                    return Err("simulation time external arrival source is inactive".into());
                }
                if node == coordinator || self.live(node).is_none() {
                    return Err("simulation time external arrival destination is inactive".into());
                }
                self.validate_credits(coordinator, arrivals)?;
                let state = self.with(node);
                if state.delivered >= state.external {
                    return Err("simulation time external arrival is unexpected".into());
                }
                consume(self.with(coordinator), arrivals);
                let state = self.with(node);
                state.delivered = state.delivered.wrapping_add(1);
                self.runnable(node);
            }
            Event::AbandonedResponseDiscarded {
                coordinator,
                arrivals,
            }
            | Event::Consume {
                participant: coordinator,
                arrivals,
            } => {
                self.validate_credits(coordinator, arrivals)?;
                consume(self.with(coordinator), arrivals);
            }
            Event::Runnable { participant } => {
                self.validate_participant(participant)?;
                self.runnable(participant);
            }
            Event::Remove { participant } => {
                let waiting = self.live(participant).ok_or(INACTIVE)?.waiting;
                let handle = participant.unwrap();
                if waiting {
                    self.answers.push(Answer {
                        handle,
                        result: Err("simulation time participant was removed".into()),
                    });
                }
                let removed = self.participants.remove(&handle).unwrap();
                self.index.remove(&removed.name);
                self.responses.retain(|identity, _| identity.0 != handle);
            }
        }
        self.settle();
        Ok(())
    }

    /// Go `settleLocked`: when every participant is active, none is handling
    /// or holds delivered work, and every non-external one is parked, time
    /// advances to the earliest deadline (deadlock if none) and they all wake.
    fn settle(&mut self) {
        let mut active = 0;
        for participant in self.participants.values() {
            if !participant.active || participant.handling != 0 || participant.delivered != 0 {
                return;
            }
            if participant.external != 0 {
                continue;
            }
            active += 1;
            if !participant.waiting {
                return;
            }
        }
        if active == 0 {
            return;
        }
        let deadline = self
            .participants
            .values()
            .filter(|participant| participant.active && participant.external == 0)
            .map(|participant| participant.deadline)
            .min()
            .unwrap_or(i64::MAX);
        if deadline != i64::MAX {
            self.current = self.current.max(deadline);
        }
        let kind = if deadline == i64::MAX {
            DEADLOCK
        } else {
            ADVANCE
        };
        let current = self.current;
        for (handle, participant) in &mut self.participants {
            if !participant.active || participant.external != 0 {
                continue;
            }
            participant.waiting = false;
            self.answers.push(Answer {
                handle: *handle,
                result: Ok(TimeResponse {
                    generation: participant.generation,
                    kind,
                    time: current,
                }),
            });
        }
    }
}

fn consume(participant: &mut Participant, arrivals: u32) {
    participant.external = participant.external.wrapping_sub(u64::from(arrivals));
    participant.delivered = participant.delivered.wrapping_sub(u64::from(arrivals));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clock_advances_every_participant_to_the_earliest_deadline() {
        let mut clock = Clock::process(false);
        let coordinator = clock.register("coordinator").unwrap();
        let node = clock.register("node/1").unwrap();
        let request = |deadline| TimeRequest {
            generation: 1,
            current: INITIAL_TIME,
            deadline: INITIAL_TIME + deadline,
            arrivals: 0,
        };
        clock.quiesce(Some(coordinator), request(20)).unwrap();
        let direct = clock.quiesce(Some(node), request(10)).unwrap();
        assert!(matches!(direct, Quiesced::Answered(_)));
        assert_eq!(
            clock
                .take_answers()
                .iter()
                .map(|answer| answer.handle)
                .collect::<Vec<_>>(),
            vec![coordinator]
        );
    }

    #[test]
    fn clock_restart_hold_blocks_advance_until_removed() {
        let mut clock = Clock::broker();
        let a = clock.register("a/1").unwrap();
        let b = clock.register("b/1").unwrap();
        let hold = clock.register("restart/b/1").unwrap();
        clock
            .apply(Event::Remove {
                participant: Some(b),
            })
            .unwrap();
        let request = TimeRequest {
            generation: 1,
            current: clock.current(),
            deadline: clock.current() + 5,
            arrivals: 0,
        };
        clock.quiesce(Some(a), request).unwrap();
        let restarted = clock.register("b/2").unwrap();
        clock
            .apply(Event::Remove {
                participant: Some(hold),
            })
            .unwrap();
        let direct = clock.quiesce(Some(restarted), request).unwrap();
        assert!(matches!(direct, Quiesced::Answered(_)));
        assert_eq!(
            clock
                .take_answers()
                .iter()
                .map(|answer| answer.handle)
                .collect::<Vec<_>>(),
            vec![a]
        );
    }
}
