mod connection;
pub mod history;
mod stall;
pub mod stream_abi;
#[cfg(test)]
mod tests;

use history::Endpoint;
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet, VecDeque};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct Domain {
    pub run: u64,
    pub node: String,
    pub incarnation: u64,
    pub address: String,
}

#[derive(Clone, Debug, Deserialize)]
pub struct Limits {
    pub listeners: u64,
    pub connections: u64,
    pub bytes: u64,
    pub connection_bytes: u64,
    pub stall_nanos: u64,
    pub transitions: u64,
}

#[derive(Clone, Debug, Deserialize)]
pub struct Node {
    pub node: String,
    pub address: String,
}

#[derive(Clone, Debug, Deserialize)]
pub struct Config {
    pub nodes: Vec<Node>,
    pub links: Vec<history::Link>,
    pub limits: Limits,
    #[serde(default)]
    pub replay: Option<history::Record>,
}

use history::zero;

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Address {
    #[serde(rename = "IP")]
    pub ip: String,
    #[serde(rename = "Port")]
    pub port: u64,
}

#[derive(Clone, Debug, Deserialize)]
#[expect(
    clippy::struct_field_names,
    reason = "Field names mirror the Go network request contract"
)]
pub struct Request {
    pub domain: Domain,
    pub op: String,
    #[serde(default)]
    pub handle: u64,
    #[serde(default)]
    pub host: String,
    #[serde(default)]
    pub port: i64,
    #[serde(default)]
    pub bytes: Vec<u8>,
    #[serde(default)]
    pub count: u32,
    #[serde(default)]
    pub deadline: u64,
    pub request_id: u64,
}

#[derive(Clone, Debug, Default, Eq, PartialEq, Serialize)]
pub struct OperationResult {
    pub request_id: u64,
    pub state: String,
    #[serde(skip_serializing_if = "zero")]
    pub handle: u64,
    pub local: Address,
    pub remote: Address,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub bytes: Vec<u8>,
    #[serde(skip_serializing_if = "zero")]
    pub count: u64,
    #[serde(skip_serializing_if = "zero")]
    pub deadline: u64,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub error: String,
}

struct Listener {
    endpoint: history::Endpoint,
    closed: bool,
    pending: std::collections::VecDeque<u64>,
    deadline: u64,
    generation: u64,
}

#[derive(Clone, Copy)]
enum WaitSource {
    Connection(u64),
    Listener(usize),
    Network,
}

#[derive(Clone)]
enum Handle {
    Listener(Domain, usize),
    Connection(Domain, u64, bool),
}

impl Handle {
    fn domain(&self) -> &Domain {
        match self {
            Self::Listener(domain, _) | Self::Connection(domain, _, _) => domain,
        }
    }
}

pub(crate) struct Pending {
    pub(crate) request: Request,
    pub(crate) accepted: u64,
    pub(crate) deadline: u64,
    pub(crate) result: Option<OperationResult>,
    source: WaitSource,
    generation: u64,
}

struct Dial {
    identity: u64,
    source: history::Endpoint,
    destination: history::Endpoint,
    active: bool,
    stalled: bool,
    reserved: bool,
    due: u64,
    epoch: u64,
    terminal: Option<String>,
}

pub struct Model {
    run: u64,
    limits: Limits,
    nodes: BTreeMap<String, history::NodeSnapshot>,
    links: BTreeMap<String, history::Link>,
    pub(crate) live: BTreeMap<String, Domain>,
    #[cfg(not(test))]
    transitions: Vec<history::Transition>,
    #[cfg(test)]
    pub(crate) transitions: Vec<history::Transition>,
    inspection_history: history::InspectionHistory,
    listeners: Vec<Listener>,
    connections: BTreeMap<u64, history::ConnectionSnapshot>,
    states: BTreeMap<(u64, bool), connection::State>,
    reservations: std::collections::BTreeSet<u64>,
    pub(crate) history_reserved: u64,
    pub(crate) pending_bytes: u64,
    pub(crate) now: u64,
    handles: BTreeMap<u64, Handle>,
    pub(crate) pending: BTreeMap<u64, Pending>,
    next_handle: u64,
    next_connection: u64,
    finished: bool,
    replay: Option<history::Record>,
    replay_next: BTreeMap<String, usize>,
    replay_lanes: BTreeMap<String, Vec<history::Transition>>,
    divergence: bool,
    pair_generation: BTreeMap<u64, u64>,
    network_generation: u64,
    clock_ceiling: u64,
    cancelled: bool,
    dials: BTreeMap<u64, Dial>,
    next_dial: u64,
}

impl Model {
    pub fn limit_clock_ceiling(&mut self, ceiling: u64) -> Result<(), String> {
        if self.finished {
            return Err("stale file handle".into());
        }
        if ceiling == 0 || ceiling > i64::MAX as u64 || self.now > ceiling {
            return Err("unsupported Gomad network operation".into());
        }
        if ceiling >= self.clock_ceiling {
            return Ok(());
        }
        for pending in self
            .pending
            .values()
            .filter(|pending| pending.result.is_none())
        {
            if pending.request.deadline > ceiling || pending.deadline > ceiling {
                return Err("unsupported Gomad network operation".into());
            }
        }
        if self
            .dials
            .values()
            .any(|dial| dial.active && dial.stalled && dial.due > ceiling)
        {
            return Err("unsupported Gomad network operation".into());
        }
        for (identity, pair) in &self.connections {
            if pair.closed || pair.reset {
                continue;
            }
            for client in [true, false] {
                let state = &self.states[&(*identity, client)];
                if state.read_deadline > ceiling
                    || state.write_deadline > ceiling
                    || state.stalled && state.stall_due > ceiling
                    || state.incoming.iter().any(|chunk| chunk.ready > ceiling)
                {
                    return Err("unsupported Gomad network operation".into());
                }
            }
        }
        if self
            .listeners
            .iter()
            .any(|listener| !listener.closed && listener.deadline > ceiling)
        {
            return Err("unsupported Gomad network operation".into());
        }
        self.clock_ceiling = ceiling;
        Ok(())
    }
    pub fn advance_before(&mut self, now: u64) -> Result<Option<Vec<u64>>, String> {
        self.validate_position(now)?;
        let mut ids = None;
        loop {
            let next = self.next_deadline();
            if next == 0 || next >= now {
                break;
            }
            ids = self.advance(next.max(self.now))?;
        }
        if now > self.now {
            ids = self.advance(now - 1)?;
        }
        self.now = now;
        Ok(ids)
    }
    fn validate_position(&self, now: u64) -> Result<(), String> {
        if self.finished {
            return Err("stale file handle".into());
        }
        if now < self.now || now > self.clock_ceiling {
            return Err("network clock moved backwards or overflowed".into());
        }
        Ok(())
    }
    pub fn cancel(&mut self, domain: &Domain, id: u64) -> Result<(), String> {
        if self.finished {
            return Err("stale file handle".into());
        }
        let Some(pending) = self.pending.get(&id) else {
            return Err("stale file handle".into());
        };
        if &pending.request.domain != domain {
            return Err("stale file handle".into());
        }
        let mut pending = self.pending.remove(&id).unwrap();
        let mut result = OperationResult {
            request_id: id,
            state: "completed".into(),
            error: "context canceled".into(),
            ..Default::default()
        };
        if pending.request.op == "dial" && pending.result.is_none() {
            self.cancelled = true;
            result.error.clear();
            if let Err(error) = self.attempt(&pending.request, &mut result) {
                result.error = error;
            }
            self.cancelled = false;
            if self.dials.get(&id).is_some_and(|dial| !dial.active) {
                self.dials.remove(&id);
            }
        }
        pending.result = Some(result);
        self.pending.insert(id, pending);
        Ok(())
    }
    fn release_pair(&mut self, identity: u64) {
        if self.reservations.remove(&identity) {
            self.history_reserved -= 1;
        }
        for client in [true, false] {
            let state = self.states.get_mut(&(identity, client)).unwrap();
            state.stalled = false;
            state.stall_due = 0;
            state.stall_start = 0;
            state.stall_epoch += 1;
        }
    }
    fn drop_bytes(&mut self, identity: u64, client: bool, incoming_only: bool) {
        let state = self.states.get_mut(&(identity, client)).unwrap();
        let discarded = if incoming_only {
            state
                .incoming
                .iter()
                .map(|chunk| chunk.bytes.len() as u64)
                .sum()
        } else {
            state.retained_bytes
        };
        state.incoming.clear();
        if !incoming_only {
            state.received.clear();
            state.stalled = false;
            state.stall_due = 0;
            state.stall_start = 0;
            state.stall_epoch += 1;
        }
        state.retained_bytes -= discarded;
        self.pending_bytes -= discarded;
        let pair = self.connections.get_mut(&identity).unwrap();
        if client {
            pair.client_bytes -= discarded;
        } else {
            pair.server_bytes -= discarded;
        }
        if self.reservations.contains(&identity)
            && !self.states[&(identity, true)].stalled
            && !self.states[&(identity, false)].stalled
        {
            self.reservations.remove(&identity);
            self.history_reserved -= 1;
        }
    }
    #[allow(
        dead_code,
        reason = "The network corpus exercises direct handle reset; cluster lifecycle uses domain revocation."
    )]
    pub fn reset(&mut self, handle: u64) -> Result<(), String> {
        let Some(Handle::Connection(_, identity, client)) = self.handles.get(&handle).cloned()
        else {
            return Err("stale file handle".into());
        };
        let pair = self.connections[&identity].clone();
        if pair.reset {
            return Ok(());
        }
        let released = self.reservations.contains(&identity);
        if released {
            self.history_reserved -= 1;
        }
        let committed = self.commit(history::Transition {
            kind: "crash".into(),
            source: if client {
                history::Endpoint {
                    port: 0,
                    ..pair.client
                }
            } else {
                history::Endpoint {
                    port: 0,
                    ..pair.server
                }
            },
            count: 1,
            bytes: pair.client_bytes + pair.server_bytes,
            outcome: "reset".into(),
            ..Default::default()
        });
        if released {
            self.history_reserved += 1;
        }
        committed?;
        if self.finished {
            return Err("stale file handle".into());
        }
        self.release_pair(identity);
        for client in [true, false] {
            self.drop_bytes(identity, client, false);
            let state = self.states.get_mut(&(identity, client)).unwrap();
            state.read_closed = true;
            state.write_closed = true;
        }
        let pair = self.connections.get_mut(&identity).unwrap();
        pair.closed = true;
        pair.reset = true;
        *self.pair_generation.get_mut(&identity).unwrap() += 1;
        self.network_generation += 1;
        Ok(())
    }
    pub fn revoke(&mut self, domain: &Domain, graceful: bool) -> Result<(), String> {
        if self.finished || self.live.get(&domain.node) != Some(domain) {
            return Err("stale file handle".into());
        }
        let matches = |endpoint: &history::Endpoint| {
            endpoint.node == domain.node && endpoint.incarnation == domain.incarnation
        };
        let pairs: Vec<_> = self
            .connections
            .iter()
            .filter(|(_, pair)| matches(&pair.client) || matches(&pair.server))
            .map(|(id, _)| *id)
            .collect();
        let listeners: Vec<_> = self
            .listeners
            .iter()
            .enumerate()
            .filter(|(_, listener)| !listener.closed && matches(&listener.endpoint))
            .map(|(index, _)| index)
            .collect();
        let dials: Vec<_> = self
            .dials
            .iter()
            .filter(|(_, dial)| matches(&dial.source))
            .map(|(&id, _)| id)
            .collect();
        let released = pairs
            .iter()
            .filter(|id| self.reservations.contains(id))
            .count() as u64
            + dials.iter().filter(|id| self.dials[id].reserved).count() as u64;
        let bytes = pairs
            .iter()
            .map(|id| self.connections[id].client_bytes + self.connections[id].server_bytes)
            .sum();
        self.history_reserved -= released;
        let committed = if !pairs.is_empty() || !listeners.is_empty() {
            self.commit(history::Transition {
                kind: if graceful {
                    "stop".into()
                } else {
                    "crash".into()
                },
                source: Self::endpoint(domain, 0),
                count: (pairs.len() + listeners.len()) as u64,
                bytes,
                outcome: if graceful {
                    "ok".into()
                } else {
                    "reset".into()
                },
                ..Default::default()
            })
        } else {
            Ok(())
        };
        self.history_reserved += released;
        committed?;
        for id in dials {
            if self.dials[&id].reserved {
                self.history_reserved -= 1;
            }
            self.finish_dial(id);
            self.dials.remove(&id);
        }
        for index in listeners {
            self.listeners[index].closed = true;
            self.listeners[index].generation += 1;
        }
        for identity in pairs {
            let local_client = matches(&self.connections[&identity].client);
            self.release_pair(identity);
            for client in [true, false] {
                self.drop_bytes(identity, client, false);
                if !graceful || client == local_client {
                    let state = self.states.get_mut(&(identity, client)).unwrap();
                    state.read_closed = true;
                    state.write_closed = true;
                }
            }
            let closed = [true, false].into_iter().all(|client| {
                self.states[&(identity, client)].read_closed
                    && self.states[&(identity, client)].write_closed
            });
            let pair = self.connections.get_mut(&identity).unwrap();
            pair.closed = closed;
            pair.reset |= !graceful;
            *self.pair_generation.get_mut(&identity).unwrap() += 1;
        }
        self.live.remove(&domain.node);
        self.pending
            .retain(|_, pending| &pending.request.domain != domain);
        self.network_generation += 1;
        Ok(())
    }
    pub fn record(&self) -> Result<history::Record, String> {
        history::validate_transitions(&self.transitions)?;
        let mut transitions = self.transitions.clone();
        history::canonicalize(&mut transitions);
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = history::transitions_digest(&transitions)?;
        snapshot.identity = history::snapshot_digest(&snapshot)?;
        Ok(history::Record {
            transitions: (!transitions.is_empty()).then_some(transitions),
            snapshot,
        })
    }
    /// Fails exactly when [`Self::record`] would, without cloning, sorting
    /// or hashing: transition validity is order-independent (per-string
    /// bounds and a monotone total length), and the snapshot digest's only
    /// transition-dependent input is the fixed-length transition digest.
    #[cfg(test)]
    pub fn record_check(&self) -> Result<(), String> {
        history::validate_transitions(&self.transitions)?;
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = history::TRANSITION_DIGEST_PLACEHOLDER.into();
        history::validate_snapshot(&snapshot)
    }

    pub(crate) fn inspection_check(&self) -> Result<(), String> {
        self.inspection_history.check()?;
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = history::TRANSITION_DIGEST_PLACEHOLDER.into();
        history::validate_snapshot(&snapshot)
    }

    pub(crate) fn inspection_digest(&self) -> Result<String, String> {
        self.inspection_history.check()?;
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = self.inspection_history.digest()?;
        history::snapshot_digest(&snapshot)
    }

    #[cfg(test)]
    pub(crate) fn inspection_digest_eager(&self) -> Result<String, String> {
        let mut history = history::InspectionHistory::default();
        for transition in &self.record()?.transitions.unwrap_or_default() {
            history.append(transition);
        }
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = history.digest()?;
        history::snapshot_digest(&snapshot)
    }
    #[allow(
        dead_code,
        reason = "The network corpus retains standalone terminal evidence; clusters record shared live state."
    )]
    pub fn finish(&mut self) -> Result<history::Record, String> {
        if self.finished {
            return Err("simulation network is already finished".into());
        }
        self.finished = true;
        self.reservations.clear();
        self.dials.clear();
        self.history_reserved = 0;
        for state in self.states.values_mut() {
            state.stalled = false;
            state.stall_epoch += 1;
        }
        history::canonicalize(&mut self.transitions);
        let record = self.record()?;
        if self.divergence
            || self.replay.as_ref().is_some_and(|replay| {
                replay.transitions.as_deref().unwrap_or_default().len() != self.transitions.len()
                    || replay.snapshot.identity != record.snapshot.identity
            })
        {
            return Err("simulation network replay diverged".into());
        }
        Ok(record)
    }
    #[allow(
        dead_code,
        reason = "The network corpus retains standalone link controls; clusters use directional change_group."
    )]
    pub fn partition(&mut self, left: &str, right: &str, symmetric: bool) -> Result<(), String> {
        self.change_group(
            &[left.to_owned()],
            &[right.to_owned()],
            "partition",
            0,
            symmetric,
        )
    }
    #[allow(
        dead_code,
        reason = "The network corpus retains standalone link controls; clusters use directional change_group."
    )]
    pub fn heal(&mut self, left: &str, right: &str, symmetric: bool) -> Result<(), String> {
        self.change_group(
            &[left.to_owned()],
            &[right.to_owned()],
            "heal",
            0,
            symmetric,
        )
    }
    #[allow(
        dead_code,
        reason = "The network corpus retains standalone link controls; clusters use directional change_group."
    )]
    pub fn delay(
        &mut self,
        left: &str,
        right: &str,
        nanos: u64,
        symmetric: bool,
    ) -> Result<(), String> {
        self.change_group(
            &[left.to_owned()],
            &[right.to_owned()],
            "delay",
            nanos,
            symmetric,
        )
    }
    pub fn advance(&mut self, now: u64) -> Result<Option<Vec<u64>>, String> {
        self.validate_position(now)?;
        self.now = now;
        self.expire_dials()?;
        self.expire_data_stalls()?;
        Ok(self.ready())
    }
    pub fn next_deadline(&self) -> u64 {
        if self.finished {
            return 0;
        }
        self.states
            .values()
            .filter(|state| state.stalled)
            .map(|state| state.stall_due)
            .chain(
                self.dials
                    .values()
                    .filter(|dial| dial.active && dial.stalled)
                    .map(|dial| dial.due),
            )
            .chain(
                self.pending
                    .values()
                    .filter(|pending| pending.result.is_none())
                    .map(|pending| self.pending_deadline(pending)),
            )
            .filter(|deadline| *deadline != 0)
            .min()
            .unwrap_or_default()
    }
    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub fn change_group(
        &mut self,
        left: &[String],
        right: &[String],
        action: &str,
        nanos: u64,
        symmetric: bool,
    ) -> Result<(), String> {
        let valid_group = |nodes: &[String]| {
            !nodes.is_empty()
                && nodes.iter().all(|node| !node.is_empty())
                && nodes.windows(2).all(|nodes| nodes[0] < nodes[1])
        };
        if self.finished
            || !valid_group(left)
            || !valid_group(right)
            || action == "delay" && nanos > i64::MAX as u64
        {
            return Err("unsupported Gomad network operation".into());
        }
        let kind = match (action, symmetric) {
            ("partition", true) => "partition",
            ("partition", false) => "disconnect",
            ("heal", true) => "heal",
            ("heal", false) => "reconnect",
            ("delay", true) => "delay",
            ("delay", false) => "directional_delay",
            _ => return Err("unsupported Gomad network operation".into()),
        };
        let nanos = if action == "delay" { nanos } else { 0 };
        let mut changes = BTreeMap::new();
        let mut transitions = Vec::new();
        for from in left {
            for to in right {
                if from == to || !self.nodes.contains_key(from) || !self.nodes.contains_key(to) {
                    return Err("unsupported Gomad network operation".into());
                }
                let mut keys = vec![Self::link_key(from, to)];
                if symmetric {
                    keys.push(Self::link_key(to, from));
                }
                for key in keys {
                    let Some(mut link) = self.links.get(&key).cloned() else {
                        return Err("unsupported Gomad network operation".into());
                    };
                    if action == "delay" {
                        link.delay_nanos = nanos;
                    } else {
                        link.enabled = action == "heal";
                    }
                    changes.insert(key, link);
                }
                transitions.push(history::Transition {
                    kind: kind.into(),
                    source: history::Endpoint {
                        node: from.clone(),
                        address: self.nodes[from].address.clone(),
                        ..Default::default()
                    },
                    destination: history::Endpoint {
                        node: to.clone(),
                        address: self.nodes[to].address.clone(),
                        ..Default::default()
                    },
                    count: if symmetric { 2 } else { 1 },
                    delay_nanos: if action == "delay" { nanos } else { 0 },
                    outcome: "ok".into(),
                    ..Default::default()
                });
            }
        }
        let mut plans = Vec::new();
        let mut reservations = self.reservations.clone();
        for (&identity, pair) in &self.connections {
            if pair.closed || pair.reset {
                continue;
            }
            let mut needs_reservation = false;
            for client in [true, false] {
                let (source, target) = if client {
                    (&pair.server, &pair.client)
                } else {
                    (&pair.client, &pair.server)
                };
                let key = Self::link_key(&source.node, &target.node);
                let prior = self.links.get(&key);
                let next = changes.get(&key).or(prior);
                let enabled = next.is_some_and(|link| link.enabled);
                let prior_enabled = prior.is_some_and(|link| link.enabled);
                let state = &self.states[&(identity, client)];
                let stalled = !enabled
                    && !state.incoming.is_empty()
                    && !self.states[&(identity, true)].close_called
                    && !self.states[&(identity, false)].close_called;
                let due = if stalled && !state.stalled {
                    self.now
                        .checked_add(self.limits.stall_nanos)
                        .and_then(|due| due.checked_add(1))
                        .filter(|due| *due <= self.clock_ceiling)
                        .ok_or("unsupported Gomad network operation")?
                } else {
                    state.stall_due
                };
                let ready = if !prior_enabled && enabled && !state.incoming.is_empty() {
                    Some(
                        self.now
                            .checked_add(next.unwrap().delay_nanos)
                            .filter(|ready| *ready <= self.clock_ceiling)
                            .ok_or("unsupported Gomad network operation")?,
                    )
                } else {
                    None
                };
                let discard =
                    prior_enabled && !enabled && self.states[&(identity, !client)].close_called;
                plans.push((identity, client, stalled, due, ready, discard));
                needs_reservation |= stalled;
            }
            if needs_reservation {
                reservations.insert(identity);
            } else {
                reservations.remove(&identity);
            }
        }
        let mut dial_plans = Vec::new();
        let mut next_reserved = reservations.len() as u64;
        for (&id, dial) in &self.dials {
            if !dial.active {
                continue;
            }
            let key = Self::link_key(&dial.source.node, &dial.destination.node);
            let link = changes.get(&key).or_else(|| self.links.get(&key));
            let stalled = !link.is_some_and(|link| link.enabled);
            let due = if stalled && !dial.stalled {
                self.stall_deadline()?
            } else {
                dial.due
            };
            next_reserved += u64::from(stalled);
            dial_plans.push((id, stalled, due));
        }
        self.commit_batch(
            &transitions,
            next_reserved.saturating_sub(self.history_reserved),
            self.history_reserved.saturating_sub(next_reserved),
        )?;
        self.reservations = reservations;
        for (id, stalled, due) in dial_plans {
            let dial = self.dials.get_mut(&id).unwrap();
            if dial.stalled != stalled {
                dial.epoch += 1;
                dial.stalled = stalled;
                dial.reserved = stalled;
                dial.due = due;
            }
        }
        self.links.extend(changes);
        for (identity, client, stalled, due, ready, discard) in plans {
            let state = self.states.get_mut(&(identity, client)).unwrap();
            if discard {
                let discarded: u64 = state
                    .incoming
                    .iter()
                    .map(|chunk| chunk.bytes.len() as u64)
                    .sum();
                state.incoming.clear();
                state.retained_bytes -= discarded;
                self.pending_bytes -= discarded;
                let pair = self.connections.get_mut(&identity).unwrap();
                if client {
                    pair.client_bytes -= discarded;
                } else {
                    pair.server_bytes -= discarded;
                }
            }
            if state.stalled != stalled {
                state.stall_epoch += 1;
                state.stalled = stalled;
                state.stall_start = if stalled { self.now } else { 0 };
                state.stall_due = if stalled { due } else { 0 };
            }
            if let Some(ready) = ready {
                for chunk in &mut state.incoming {
                    chunk.ready = ready;
                }
            }
        }
        for generation in self.pair_generation.values_mut() {
            *generation += 1;
        }
        self.network_generation += 1;
        Ok(())
    }
    pub fn ready(&mut self) -> Option<Vec<u64>> {
        if self.finished {
            return None;
        }
        let ids: Vec<_> = self.pending.keys().copied().collect();
        let mut completed = Vec::new();
        for id in ids {
            let mut pending = self.pending.remove(&id).unwrap();
            let signaled = self.generation(pending.source) != pending.generation
                || pending.deadline != 0 && pending.deadline <= self.now;
            if pending.result.is_none() && signaled {
                let mut result = OperationResult {
                    request_id: id,
                    state: "completed".into(),
                    ..Default::default()
                };
                if let Err(error) = self.attempt(&pending.request, &mut result) {
                    result.error = error;
                    result.state = "completed".into();
                    result.deadline = 0;
                }
                if pending.request.op == "write" {
                    let accepted = result.count;
                    pending.accepted += accepted;
                    pending.request.bytes.drain(
                        ..usize::try_from(accepted)
                            .expect("accepted bytes are bounded by the request buffer"),
                    );
                    result.count = pending.accepted;
                }
                Self::wait_deadline(&pending.request, self.now, &mut result);
                pending.deadline = result.deadline;
                pending.generation = self.generation(pending.source);
                if result.state == "completed" {
                    if self.dials.get(&id).is_some_and(|dial| !dial.active) {
                        self.dials.remove(&id);
                    }
                    pending.result = Some(result);
                }
            }
            if pending.result.is_some() {
                completed.push(id);
            }
            self.pending.insert(id, pending);
        }
        (!completed.is_empty()).then_some(completed)
    }

    fn wait_deadline(request: &Request, now: u64, result: &mut OperationResult) {
        if result.state != "pending" {
            return;
        }
        if request.deadline != 0 && (result.deadline == 0 || request.deadline < result.deadline) {
            result.deadline = request.deadline;
        }
        if result.deadline != 0 && result.deadline <= now {
            *result = OperationResult {
                request_id: request.request_id,
                state: "completed".into(),
                error: "context deadline exceeded".into(),
                ..Default::default()
            };
        }
    }
    pub fn dispatch(&mut self, request: Request) -> OperationResult {
        let mut result = OperationResult {
            request_id: request.request_id,
            state: "completed".into(),
            ..Default::default()
        };
        if self.finished {
            result.error = "stale file handle".into();
            return result;
        }
        if request.op == "poll" {
            let Some(pending) = self.pending.get(&request.request_id) else {
                result.error = "stale file handle".into();
                return result;
            };
            if pending.request.domain != request.domain {
                result.error = "stale file handle".into();
                return result;
            }
            if pending.result.is_some() {
                return self
                    .pending
                    .remove(&request.request_id)
                    .unwrap()
                    .result
                    .unwrap();
            }
            result.state = "pending".into();
            result.deadline = pending.deadline;
            return result;
        }
        if request.request_id == 0
            || self.pending.contains_key(&request.request_id)
            || request.deadline > self.clock_ceiling
            || request.count > 65536
            || request.bytes.len() > 65536
        {
            result.error = "unsupported Gomad network operation".into();
            return result;
        }
        if self.live.get(&request.domain.node) != Some(&request.domain) {
            result.error = "stale file handle".into();
            return result;
        }
        if self.pending.len() >= 65536 {
            result.error = "network resources exhausted".into();
            return result;
        }
        if request.op != "listen"
            && request.op != "dial"
            && self
                .handles
                .get(&request.handle)
                .is_none_or(|handle| handle.domain() != &request.domain)
        {
            result.error = "stale file handle".into();
            return result;
        }
        if let Err(error) = self.attempt(&request, &mut result) {
            result.error = error;
            result.state = "completed".into();
            result.deadline = 0;
        }
        Self::wait_deadline(&request, self.now, &mut result);
        if result.state == "pending" {
            let mut retained = request;
            if retained.op == "write" {
                retained.bytes.drain(
                    ..usize::try_from(result.count)
                        .expect("accepted bytes are bounded by the request buffer"),
                );
            }
            let source = self.wait_source(&retained);
            let generation = self.generation(source);
            self.pending.insert(
                retained.request_id,
                Pending {
                    request: retained,
                    accepted: result.count,
                    deadline: result.deadline,
                    result: None,
                    source,
                    generation,
                },
            );
        } else if request.op == "dial"
            && self
                .dials
                .get(&request.request_id)
                .is_some_and(|dial| !dial.active)
        {
            self.dials.remove(&request.request_id);
        }
        result
    }

    fn endpoint(domain: &Domain, port: u64) -> history::Endpoint {
        history::Endpoint {
            node: domain.node.clone(),
            incarnation: domain.incarnation,
            address: domain.address.clone(),
            port,
        }
    }

    fn commit(&mut self, transition: history::Transition) -> Result<(), String> {
        self.commit_silent(transition)?;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        self.emit_transition(self.transitions.len() - 1);
        Ok(())
    }

    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn emit_transition(&self, index: usize) {
        crate::event_order::emit_current(
            "model-transition",
            index as u64,
            self.now.cast_signed(),
            serde_json::json!({"model": "network", "transition": &self.transitions[index]}),
        );
    }

    fn commit_silent(&mut self, transition: history::Transition) -> Result<(), String> {
        if !history::valid_transition(&transition) {
            return Err("simulation network transition is invalid".into());
        }
        if self.transitions.len() as u64 + self.history_reserved >= self.limits.transitions {
            return Err("network resources exhausted".into());
        }
        if self.replay.is_some() {
            let lane = history::lane(&transition);
            let next = self.replay_next.get(&lane).copied().unwrap_or_default();
            let expected = self
                .replay_lanes
                .get(&lane)
                .and_then(|transitions| transitions.get(next));
            let equal = expected.is_some_and(|expected| {
                let mut expected = expected.clone();
                expected.ordinal = 0;
                expected == transition
            });
            if !equal {
                self.divergence = true;
                return Err("simulation network replay diverged".into());
            }
            self.replay_next.insert(lane, next + 1);
        }
        self.inspection_history.append(&transition);
        self.transitions.push(transition);
        Ok(())
    }

    fn commit_batch(
        &mut self,
        transitions: &[history::Transition],
        reserve: u64,
        release: u64,
    ) -> Result<(), String> {
        let next = self
            .history_reserved
            .checked_sub(release)
            .and_then(|reserved| reserved.checked_add(reserve))
            .ok_or("network resources exhausted")?;
        if self.transitions.len() as u64 + transitions.len() as u64 + next > self.limits.transitions
        {
            return Err("network resources exhausted".into());
        }
        let start = self.transitions.len();
        let previous_reserved = self.history_reserved;
        let previous_replay = self.replay_next.clone();
        let previous_inspection = self.inspection_history.clone();
        self.history_reserved = next;
        for transition in transitions {
            if let Err(error) = self.commit_silent(transition.clone()) {
                self.transitions.truncate(start);
                self.history_reserved = previous_reserved;
                self.replay_next = previous_replay;
                self.inspection_history = previous_inspection;
                return Err(error);
            }
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        for index in start..self.transitions.len() {
            self.emit_transition(index);
        }
        Ok(())
    }

    fn add_handle(&mut self, handle: Handle) -> u64 {
        self.next_handle += 1;
        self.handles.insert(self.next_handle, handle);
        self.next_handle
    }

    fn failure(
        &mut self,
        kind: &str,
        source: history::Endpoint,
        destination: history::Endpoint,
        outcome: &str,
        error: &str,
    ) -> Result<(), String> {
        self.commit(history::Transition {
            kind: kind.into(),
            source,
            destination,
            outcome: outcome.into(),
            ..Default::default()
        })?;
        Err(error.into())
    }

    fn dial_failure(
        &mut self,
        request_id: u64,
        source: history::Endpoint,
        destination: history::Endpoint,
        outcome: &str,
        error: &str,
    ) -> Result<(), String> {
        self.commit(history::Transition {
            kind: "dial".into(),
            source,
            destination,
            outcome: outcome.into(),
            ..Default::default()
        })?;
        self.finish_dial(request_id);
        Err(error.into())
    }

    fn wait_source(&self, request: &Request) -> WaitSource {
        match (request.op.as_str(), self.handles.get(&request.handle)) {
            ("accept", Some(Handle::Listener(_, index))) => WaitSource::Listener(*index),
            ("read" | "write", Some(Handle::Connection(_, identity, _))) => {
                WaitSource::Connection(*identity)
            }
            _ => WaitSource::Network,
        }
    }

    fn generation(&self, source: WaitSource) -> u64 {
        match source {
            WaitSource::Connection(identity) => self.pair_generation[&identity],
            WaitSource::Listener(index) => self.listeners[index].generation,
            WaitSource::Network => self.network_generation,
        }
    }

    fn attempt(&mut self, request: &Request, result: &mut OperationResult) -> Result<(), String> {
        let outcome = self.attempt_inner(request, result);
        if outcome.is_ok() || result.count != 0 {
            match (request.op.as_str(), self.handles.get(&request.handle)) {
                ("read" | "write", Some(Handle::Connection(_, identity, _)))
                    if result.count != 0 =>
                {
                    *self.pair_generation.get_mut(identity).unwrap() += 1;
                }
                (
                    "close" | "close_read" | "close_write" | "deadline" | "read_deadline"
                    | "write_deadline",
                    Some(Handle::Connection(_, identity, _)),
                ) => {
                    *self.pair_generation.get_mut(identity).unwrap() += 1;
                }
                ("close" | "deadline", Some(Handle::Listener(_, index))) => {
                    self.listeners[*index].generation += 1;
                    if request.op == "close" {
                        self.network_generation += 1;
                    }
                }
                _ => (),
            }
        }
        if outcome.is_ok() {
            Self::wait_deadline(request, self.now, result);
        }
        outcome
    }

    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    fn attempt_inner(
        &mut self,
        request: &Request,
        result: &mut OperationResult,
    ) -> Result<(), String> {
        match request.op.as_str() {
            "listen" => {
                if !(0..=65535).contains(&request.port)
                    || !request.host.is_empty()
                        && request.host != "0.0.0.0"
                        && request.host != request.domain.address
                {
                    return Err("unsupported Gomad network operation".into());
                }
                let mut port = request.port.cast_unsigned();
                if port == 0 {
                    port = self.nodes[&request.domain.node].next_listener_port;
                    while port <= 65535
                        && self.listeners.iter().any(|listener| {
                            !listener.closed
                                && listener.endpoint.address == request.domain.address
                                && listener.endpoint.port == port
                        })
                    {
                        port += 1;
                    }
                }
                let endpoint = Self::endpoint(&request.domain, port);
                if port > 65535 {
                    return self.failure(
                        "listen",
                        history::Endpoint {
                            port: 0,
                            ..endpoint
                        },
                        Endpoint::default(),
                        "capacity",
                        "network resources exhausted",
                    );
                }
                if self.listeners.iter().any(|listener| {
                    !listener.closed
                        && listener.endpoint.address == endpoint.address
                        && listener.endpoint.port == port
                }) {
                    return self.failure(
                        "listen",
                        endpoint,
                        Endpoint::default(),
                        "closed",
                        "address already in use",
                    );
                }
                if self.listeners.len() as u64 >= self.limits.listeners {
                    return self.failure(
                        "listen",
                        endpoint,
                        Endpoint::default(),
                        "capacity",
                        "network resources exhausted",
                    );
                }
                self.commit(history::Transition {
                    kind: "listen".into(),
                    source: endpoint.clone(),
                    outcome: "ok".into(),
                    ..Default::default()
                })?;
                if request.port == 0 {
                    self.nodes
                        .get_mut(&request.domain.node)
                        .unwrap()
                        .next_listener_port = port + 1;
                }
                let index = self.listeners.len();
                self.listeners.push(Listener {
                    endpoint,
                    closed: false,
                    pending: VecDeque::default(),
                    deadline: 0,
                    generation: 0,
                });
                self.network_generation += 1;
                result.handle = self.add_handle(Handle::Listener(request.domain.clone(), index));
                result.local = Address {
                    ip: request.domain.address.clone(),
                    port,
                };
            }
            "dial" => {
                if request.host.is_empty() || !(1..=65535).contains(&request.port) {
                    return Err("unsupported Gomad network operation".into());
                }
                if let Some(error) = self
                    .dials
                    .get(&request.request_id)
                    .and_then(|dial| dial.terminal.clone())
                {
                    return Err(error);
                }
                if self.divergence {
                    return Err("simulation network replay diverged".into());
                }
                let source = Self::endpoint(&request.domain, 0);
                let mut destination = history::Endpoint {
                    address: request.host.clone(),
                    port: request.port.cast_unsigned(),
                    ..Default::default()
                };
                let target = self
                    .nodes
                    .values()
                    .find(|node| node.address == request.host)
                    .cloned();
                if self.cancelled || request.deadline != 0 && request.deadline <= self.now {
                    destination.node = target
                        .as_ref()
                        .map(|node| node.node.clone())
                        .unwrap_or_default();
                    let release = self
                        .dials
                        .get(&request.request_id)
                        .is_some_and(|dial| dial.reserved);
                    self.commit_batch(
                        &[history::Transition {
                            kind: "dial".into(),
                            source,
                            destination,
                            outcome: "deadline".into(),
                            ..Default::default()
                        }],
                        0,
                        u64::from(release),
                    )?;
                    self.finish_dial(request.request_id);
                    return Err(if self.cancelled {
                        "context canceled"
                    } else {
                        "context deadline exceeded"
                    }
                    .into());
                }
                let Some(target) = target else {
                    return self.failure(
                        "dial",
                        source,
                        destination,
                        "refused",
                        "connection refused",
                    );
                };
                destination.node.clone_from(&target.node);
                if request.domain.node == target.node {
                    return Err("unsupported Gomad network operation".into());
                }
                let link = self
                    .links
                    .get(&Self::link_key(&request.domain.node, &target.node))
                    .cloned();
                if !link.as_ref().is_some_and(|link| link.enabled) {
                    if !self
                        .dials
                        .get(&request.request_id)
                        .is_some_and(|dial| dial.stalled)
                    {
                        let due = self.stall_deadline()?;
                        if self.transitions.len() as u64 + self.history_reserved
                            >= self.limits.transitions
                        {
                            return Err("network resources exhausted".into());
                        }
                        if !self.dials.contains_key(&request.request_id) {
                            self.next_dial += 1;
                            self.dials.insert(
                                request.request_id,
                                Dial {
                                    identity: self.next_dial,
                                    source,
                                    destination,
                                    active: true,
                                    stalled: false,
                                    reserved: false,
                                    due: 0,
                                    epoch: 0,
                                    terminal: None,
                                },
                            );
                        }
                        let dial = self.dials.get_mut(&request.request_id).unwrap();
                        dial.active = true;
                        dial.stalled = true;
                        dial.reserved = true;
                        dial.due = due;
                        dial.epoch += 1;
                        self.history_reserved += 1;
                    }
                    result.state = "pending".into();
                    result.deadline = self.dials[&request.request_id].due;
                    return Ok(());
                }
                let link = link.unwrap();
                let listener_index = self.listeners.iter().position(|listener| {
                    !listener.closed
                        && listener.endpoint.address == request.host
                        && listener.endpoint.port == request.port.cast_unsigned()
                });
                let Some(listener_index) = listener_index else {
                    return self.dial_failure(
                        request.request_id,
                        source,
                        destination,
                        "refused",
                        "connection refused",
                    );
                };
                if self.listeners[listener_index].pending.len() >= 64 {
                    result.state = "pending".into();
                    return Ok(());
                }
                let port = self.nodes[&request.domain.node].next_client_port;
                if self.connections.len() as u64 >= self.limits.connections || port > 65535 {
                    return self.dial_failure(
                        request.request_id,
                        source,
                        destination,
                        "capacity",
                        "network resources exhausted",
                    );
                }
                let client = Self::endpoint(&request.domain, port);
                let server = self.listeners[listener_index].endpoint.clone();
                let identity = self.next_connection + 1;
                self.commit(history::Transition {
                    kind: "dial".into(),
                    source: client.clone(),
                    destination: server.clone(),
                    connection: identity,
                    delay_nanos: link.delay_nanos,
                    outcome: "ok".into(),
                    ..Default::default()
                })?;
                self.finish_dial(request.request_id);
                self.nodes
                    .get_mut(&request.domain.node)
                    .unwrap()
                    .next_client_port += 1;
                self.next_connection = identity;
                self.connections.insert(
                    identity,
                    history::ConnectionSnapshot {
                        identity,
                        client,
                        server,
                        client_bytes: 0,
                        server_bytes: 0,
                        closed: false,
                        reset: false,
                    },
                );
                self.states
                    .insert((identity, true), connection::State::default());
                self.states
                    .insert((identity, false), connection::State::default());
                self.pair_generation.insert(identity, 0);
                self.listeners[listener_index].pending.push_back(identity);
                self.listeners[listener_index].generation += 1;
                self.network_generation += 1;
                result.handle =
                    self.add_handle(Handle::Connection(request.domain.clone(), identity, true));
                result.local = Address {
                    ip: request.domain.address.clone(),
                    port,
                };
                result.remote = Address {
                    ip: request.host.clone(),
                    port: request.port.cast_unsigned(),
                };
            }
            "accept" => {
                let Some(Handle::Listener(_, index)) = self.handles.get(&request.handle).cloned()
                else {
                    return Err("unsupported Gomad network operation".into());
                };
                let listener = &self.listeners[index];
                let Some(identity) = listener.pending.front().copied() else {
                    if listener.closed {
                        return Err("use of closed network connection".into());
                    }
                    if listener.deadline != 0 && listener.deadline <= self.now {
                        self.commit(history::Transition {
                            kind: "accept".into(),
                            source: listener.endpoint.clone(),
                            outcome: "deadline".into(),
                            ..Default::default()
                        })?;
                        return Err("i/o timeout".into());
                    }
                    result.state = "pending".into();
                    result.deadline = listener.deadline;
                    return Ok(());
                };
                let connection = self.connections[&identity].clone();
                self.commit(history::Transition {
                    kind: "accept".into(),
                    source: connection.server.clone(),
                    destination: connection.client.clone(),
                    connection: identity,
                    outcome: "ok".into(),
                    ..Default::default()
                })?;
                self.listeners[index].pending.pop_front();
                self.listeners[index].generation += 1;
                self.network_generation += 1;
                result.local = Address {
                    ip: connection.server.address,
                    port: connection.server.port,
                };
                result.remote = Address {
                    ip: connection.client.address,
                    port: connection.client.port,
                };
                result.handle =
                    self.add_handle(Handle::Connection(request.domain.clone(), identity, false));
            }
            "write" => {
                let Some(Handle::Connection(_, identity, client)) =
                    self.handles.get(&request.handle).cloned()
                else {
                    return Err("unsupported Gomad network operation".into());
                };
                let bytes = &request.bytes;
                if bytes.len() > 65536 {
                    return Err("unsupported Gomad network operation".into());
                }
                if bytes.is_empty() {
                    return Ok(());
                }
                if self.divergence {
                    return Err("simulation network replay diverged".into());
                }
                if let Some(error) = self.states[&(identity, client)]
                    .terminal
                    .clone()
                    .or_else(|| self.states[&(identity, !client)].terminal.clone())
                {
                    return Err(error);
                }
                let pair = self.connections[&identity].clone();
                let (source, destination) = if client {
                    (pair.client, pair.server)
                } else {
                    (pair.server, pair.client)
                };
                if pair.reset
                    || self.states[&(identity, client)].write_closed
                    || self.states[&(identity, !client)].read_closed
                {
                    return Err("use of closed network connection".into());
                }
                let link = self
                    .links
                    .get(&Self::link_key(&source.node, &destination.node))
                    .cloned();
                let enabled = link.as_ref().is_some_and(|link| link.enabled);
                let delay = link.as_ref().map_or(0, |link| link.delay_nanos);
                let retained = self.states[&(identity, !client)].retained_bytes;
                let length = bytes.len().min(
                    usize::try_from(self.limits.connection_bytes - retained).unwrap_or(usize::MAX),
                );
                if length == 0 {
                    let deadline = self.states[&(identity, client)].write_deadline;
                    if deadline != 0 && deadline <= self.now {
                        return Err("i/o timeout".into());
                    }
                    result.state = "pending".into();
                    result.deadline = deadline;
                    return Ok(());
                }
                let mut transition = history::Transition {
                    kind: "write".into(),
                    source,
                    destination,
                    connection: identity,
                    bytes: length as u64,
                    delay_nanos: delay,
                    outcome: if enabled { "ok".into() } else { "held".into() },
                    ..Default::default()
                };
                if length as u64 > self.limits.bytes - self.pending_bytes {
                    transition.outcome = "capacity".into();
                    self.commit(transition)?;
                    return Err("network resources exhausted".into());
                }
                let ready = self
                    .now
                    .checked_add(delay)
                    .filter(|ready| *ready <= self.clock_ceiling)
                    .ok_or("unsupported Gomad network operation")?;
                let stalled = !enabled && !self.states[&(identity, !client)].stalled;
                let reservation = stalled && !self.reservations.contains(&identity);
                let due = if stalled {
                    self.now
                        .checked_add(self.limits.stall_nanos)
                        .and_then(|due| due.checked_add(1))
                        .filter(|due| *due <= self.clock_ceiling)
                        .ok_or("unsupported Gomad network operation")?
                } else {
                    0
                };
                if reservation
                    && self.transitions.len() as u64 + self.history_reserved + 2
                        > self.limits.transitions
                {
                    return Err("network resources exhausted".into());
                }
                self.commit(transition)?;
                if reservation {
                    self.reservations.insert(identity);
                    self.history_reserved += 1;
                }
                let state = self.states.get_mut(&(identity, !client)).unwrap();
                if stalled {
                    state.stalled = true;
                    state.stall_start = self.now;
                    state.stall_due = due;
                    state.stall_epoch += 1;
                }
                state.incoming.push_back(connection::Chunk {
                    bytes: bytes[..length].iter().copied().collect(),
                    ready,
                });
                state.retained_bytes += length as u64;
                self.pending_bytes += length as u64;
                result.count = length as u64;
                if length < bytes.len() {
                    result.state = "pending".into();
                    result.deadline = self.states[&(identity, client)].write_deadline;
                }
                let pair = self.connections.get_mut(&identity).unwrap();
                if client {
                    pair.server_bytes += length as u64;
                } else {
                    pair.client_bytes += length as u64;
                }
                if result.state == "pending" && result.deadline != 0 && result.deadline <= self.now
                {
                    return Err("i/o timeout".into());
                }
            }
            "read" => {
                let Some(Handle::Connection(_, identity, client)) =
                    self.handles.get(&request.handle).cloned()
                else {
                    return Err("unsupported Gomad network operation".into());
                };
                if request.count == 0 {
                    return Ok(());
                }
                if self.divergence {
                    return Err("simulation network replay diverged".into());
                }
                if let Some(error) = self.states[&(identity, client)]
                    .terminal
                    .clone()
                    .or_else(|| self.states[&(identity, !client)].terminal.clone())
                {
                    return Err(error);
                }
                let pair = &self.connections[&identity];
                if pair.reset {
                    return Err("use of closed network connection".into());
                }
                let (owner, target) = if client {
                    (&pair.client, &pair.server)
                } else {
                    (&pair.server, &pair.client)
                };
                let enabled = self
                    .links
                    .get(&Self::link_key(&target.node, &owner.node))
                    .is_some_and(|link| link.enabled);
                let peer_closed = self.states[&(identity, !client)].write_closed;
                let state = self.states.get_mut(&(identity, client)).unwrap();
                if state.read_closed {
                    return Err("use of closed network connection".into());
                }
                if state.received.is_empty()
                    && let Some(chunk) = state.incoming.front()
                {
                    if !enabled || chunk.ready > self.now {
                        if state.read_deadline != 0 && state.read_deadline <= self.now {
                            return Err("i/o timeout".into());
                        }
                        result.state = "pending".into();
                        if enabled {
                            result.deadline = chunk.ready;
                        }
                        if state.read_deadline != 0
                            && (result.deadline == 0 || state.read_deadline < result.deadline)
                        {
                            result.deadline = state.read_deadline;
                        }
                        return Ok(());
                    }
                    state.received = state.incoming.pop_front().unwrap().bytes;
                }
                if state.received.is_empty() {
                    if peer_closed {
                        return Err("EOF".into());
                    }
                    result.state = "pending".into();
                    if state.read_deadline != 0 && state.read_deadline <= self.now {
                        return Err("i/o timeout".into());
                    }
                    result.deadline = state.read_deadline;
                    return Ok(());
                }
                let length = (request.count as usize).min(state.received.len());
                let bytes: Vec<_> = state.received.drain(..length).collect();
                state.retained_bytes -= length as u64;
                self.pending_bytes -= length as u64;
                result.count = length as u64;
                result.bytes = bytes;
                let pair = self.connections.get_mut(&identity).unwrap();
                if client {
                    pair.client_bytes -= length as u64;
                } else {
                    pair.server_bytes -= length as u64;
                }
            }
            "deadline" | "read_deadline" | "write_deadline" => {
                match self.handles.get(&request.handle).cloned().unwrap() {
                    Handle::Listener(_, index) => {
                        if request.op != "deadline" {
                            return Err("unsupported Gomad network operation".into());
                        }
                        self.listeners[index].deadline = request.deadline;
                    }
                    Handle::Connection(_, identity, client) => {
                        let state = self.states.get_mut(&(identity, client)).unwrap();
                        if request.op != "write_deadline" {
                            state.read_deadline = request.deadline;
                        }
                        if request.op != "read_deadline" {
                            state.write_deadline = request.deadline;
                        }
                    }
                }
            }
            "close" | "close_read" | "close_write" => {
                match self.handles.get(&request.handle).cloned().unwrap() {
                    Handle::Listener(_, index) => {
                        if request.op != "close" {
                            return Err("unsupported Gomad network operation".into());
                        }
                        if self.listeners[index].closed {
                            return Err("use of closed network connection".into());
                        }
                        self.commit(history::Transition {
                            kind: "listener_close".into(),
                            source: self.listeners[index].endpoint.clone(),
                            outcome: "ok".into(),
                            ..Default::default()
                        })?;
                        self.listeners[index].closed = true;
                    }
                    Handle::Connection(_, identity, client) => {
                        let state = &self.states[&(identity, client)];
                        if request.op == "close" && state.close_called
                            || request.op == "close_read" && state.read_closed
                            || request.op == "close_write" && state.write_closed
                        {
                            return Err("use of closed network connection".into());
                        }
                        let pair = self.connections[&identity].clone();
                        let (source, destination) = if client {
                            (pair.client, pair.server)
                        } else {
                            (pair.server, pair.client)
                        };
                        if request.op == "close" {
                            let released = self.reservations.contains(&identity);
                            if released {
                                self.history_reserved -= 1;
                            }
                            let committed = self.commit(history::Transition {
                                kind: "close".into(),
                                source: source.clone(),
                                destination: destination.clone(),
                                connection: identity,
                                outcome: "ok".into(),
                                ..Default::default()
                            });
                            if released {
                                self.history_reserved += 1;
                            }
                            committed?;
                            self.release_pair(identity);
                            if !self
                                .links
                                .get(&Self::link_key(&source.node, &destination.node))
                                .is_some_and(|link| link.enabled)
                            {
                                self.drop_bytes(identity, !client, true);
                            }
                            self.states
                                .get_mut(&(identity, client))
                                .unwrap()
                                .close_called = true;
                        }
                        if request.op != "close_write" {
                            self.drop_bytes(identity, client, false);
                            self.states
                                .get_mut(&(identity, client))
                                .unwrap()
                                .read_closed = true;
                        }
                        if request.op != "close_read" {
                            self.states
                                .get_mut(&(identity, client))
                                .unwrap()
                                .write_closed = true;
                        }
                    }
                }
            }
            _ => return Err("unsupported Gomad network operation".into()),
        }
        Ok(())
    }
    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub fn new(run: u64, config: Config) -> Result<Self, String> {
        let limits = &config.limits;
        if run == 0
            || limits.listeners == 0
            || limits.connections == 0
            || limits.bytes == 0
            || limits.transitions == 0
        {
            return Err("simulation network limits are invalid".into());
        }
        if limits.connection_bytes == 0 || limits.connection_bytes > 1 << 30 {
            return Err("simulation network connection byte limit is invalid".into());
        }
        if limits.stall_nanos == 0 || limits.stall_nanos > i64::MAX as u64 - 1 {
            return Err("simulation network stall duration limit is invalid".into());
        }
        if let Some(replay) = &config.replay {
            let transitions = replay.transitions.as_deref().unwrap_or_default();
            if transitions.len() as u64 > limits.transitions {
                return Err("network resources exhausted".into());
            }
            for (index, transition) in transitions.iter().enumerate() {
                if transition.ordinal != index as u64
                    || !history::valid_transition(transition)
                    || index != 0
                        && history::lane(&transitions[index - 1]) > history::lane(transition)
                {
                    return Err(
                        "simulation network replay transition is invalid or unordered".into(),
                    );
                }
            }
            if history::transitions_digest(transitions)? != replay.snapshot.transition_sha256 {
                return Err("simulation network replay transition digest does not match".into());
            }
            if history::snapshot_digest(&replay.snapshot)? != replay.snapshot.identity {
                return Err("simulation network replay snapshot identity does not match".into());
            }
        }
        if let Some(replay) = &config.replay {
            Self::validate_replay_snapshot(&replay.snapshot, limits)?;
        }
        let mut nodes = BTreeMap::new();
        let mut addresses = std::collections::BTreeSet::new();
        let mut previous = String::new();
        for node in config.nodes {
            if node.node <= previous
                || node.address.is_empty()
                || !addresses.insert(node.address.clone())
            {
                return Err("simulation network nodes are invalid or unordered".into());
            }
            previous.clone_from(&node.node);
            nodes.insert(
                node.node.clone(),
                history::NodeSnapshot {
                    node: node.node,
                    address: node.address,
                    last_incarnation: 0,
                    next_listener_port: 20000,
                    next_client_port: 40000,
                },
            );
        }
        let mut links = BTreeMap::new();
        let mut previous = String::new();
        for link in config.links {
            let key = Self::link_key(&link.from, &link.to);
            if key <= previous
                || !nodes.contains_key(&link.from)
                || !nodes.contains_key(&link.to)
                || link.from == link.to
            {
                return Err("simulation network links are invalid or unordered".into());
            }
            previous.clone_from(&key);
            links.insert(key, link);
        }
        let mut replay_lanes: BTreeMap<String, Vec<history::Transition>> = BTreeMap::new();
        if let Some(replay) = &config.replay {
            for transition in replay.transitions.as_deref().unwrap_or_default() {
                replay_lanes
                    .entry(history::lane(transition))
                    .or_default()
                    .push(transition.clone());
            }
        }
        Ok(Self {
            run,
            limits: config.limits,
            nodes,
            links,
            live: BTreeMap::new(),
            transitions: Vec::new(),
            inspection_history: history::InspectionHistory::default(),
            listeners: Vec::new(),
            connections: BTreeMap::new(),
            states: BTreeMap::new(),
            reservations: BTreeSet::default(),
            history_reserved: 0,
            pending_bytes: 0,
            now: 0,
            handles: BTreeMap::new(),
            pending: BTreeMap::new(),
            next_handle: 0,
            next_connection: 0,
            finished: false,
            replay_lanes,
            replay: config.replay,
            replay_next: BTreeMap::new(),
            divergence: false,
            pair_generation: BTreeMap::new(),
            network_generation: 0,
            clock_ceiling: i64::MAX as u64,
            cancelled: false,
            dials: BTreeMap::new(),
            next_dial: 0,
        })
    }
    fn link_key(from: &str, to: &str) -> String {
        format!("{from}\0{to}")
    }
    fn stall_deadline(&self) -> Result<u64, String> {
        self.now
            .checked_add(self.limits.stall_nanos)
            .and_then(|due| due.checked_add(1))
            .filter(|due| *due <= self.clock_ceiling)
            .ok_or_else(|| "unsupported Gomad network operation".into())
    }
    fn finish_dial(&mut self, id: u64) {
        if let Some(dial) = self.dials.get_mut(&id) {
            dial.active = false;
            dial.stalled = false;
            dial.reserved = false;
            dial.epoch += 1;
        }
    }
    fn pending_deadline(&self, pending: &Pending) -> u64 {
        if pending.request.op != "dial" {
            return pending.deadline;
        }
        let due = self
            .dials
            .get(&pending.request.request_id)
            .filter(|dial| dial.active && dial.stalled)
            .map_or(0, |dial| dial.due);
        let caller = pending.request.deadline;
        match (caller, due) {
            (0, _) => due,
            (_, 0) => caller,
            _ => caller.min(due),
        }
    }
    fn expire_dials(&mut self) -> Result<(), String> {
        let mut ids: Vec<_> = self
            .dials
            .iter()
            .filter(|(_, dial)| {
                dial.active && dial.stalled && dial.reserved && dial.due <= self.now
            })
            .map(|(&id, dial)| (dial.identity, id))
            .collect();
        ids.sort_unstable();
        for (_, id) in ids {
            let dial = &self.dials[&id];
            self.commit_batch(
                &[history::Transition {
                    kind: "dial".into(),
                    source: dial.source.clone(),
                    destination: dial.destination.clone(),
                    outcome: "timeout".into(),
                    ..Default::default()
                }],
                0,
                1,
            )?;
            self.dials.get_mut(&id).unwrap().terminal = Some("connection timed out".into());
            self.finish_dial(id);
            self.network_generation += 1;
        }
        Ok(())
    }
    fn validate_replay_snapshot(
        snapshot: &history::Snapshot,
        limits: &Limits,
    ) -> Result<(), String> {
        let listeners = snapshot.listeners.as_deref().unwrap_or_default();
        let connections = snapshot.connections.as_deref().unwrap_or_default();
        if listeners.len() as u64 > limits.listeners
            || connections.len() as u64 > limits.connections
        {
            return Err("network resources exhausted".into());
        }
        let nodes = snapshot.nodes.as_deref().unwrap_or_default();
        for (index, node) in nodes.iter().enumerate() {
            if node.node.is_empty()
                || node.address.is_empty()
                || node.next_listener_port == 0
                || node.next_client_port == 0
                || index != 0 && nodes[index - 1].node >= node.node
            {
                return Err(
                    "simulation network replay node snapshot is invalid or unordered".into(),
                );
            }
        }
        let links = snapshot.links.as_deref().unwrap_or_default();
        let link_key = |link: &history::Link| Self::link_key(&link.from, &link.to);
        for (index, link) in links.iter().enumerate() {
            if link.from.is_empty()
                || link.to.is_empty()
                || link.from == link.to
                || index != 0 && link_key(&links[index - 1]) >= link_key(link)
            {
                return Err(
                    "simulation network replay link snapshot is invalid or unordered".into(),
                );
            }
        }
        for (index, listener) in listeners.iter().enumerate() {
            if !history::valid_endpoint(&listener.endpoint, true)
                || !(1..=65535).contains(&listener.endpoint.port)
                || index != 0
                    && (history::endpoint_key(&listeners[index - 1].endpoint)
                        > history::endpoint_key(&listener.endpoint)
                        || history::endpoint_key(&listeners[index - 1].endpoint)
                            == history::endpoint_key(&listener.endpoint)
                            && !listeners[index - 1].closed)
            {
                return Err(
                    "simulation network replay listener snapshot is invalid or unordered".into(),
                );
            }
        }
        let mut retained = 0;
        for (index, connection) in connections.iter().enumerate() {
            if connection.identity == 0
                || !history::valid_endpoint(&connection.client, true)
                || !history::valid_endpoint(&connection.server, true)
                || !(1..=65535).contains(&connection.client.port)
                || !(1..=65535).contains(&connection.server.port)
                || connection.reset && !connection.closed
                || index != 0 && connections[index - 1].identity >= connection.identity
            {
                return Err(
                    "simulation network replay connection snapshot is invalid or unordered".into(),
                );
            }
            if connection.reset && (connection.client_bytes != 0 || connection.server_bytes != 0) {
                return Err("simulation network replay reset connection retains bytes".into());
            }
            for count in [connection.client_bytes, connection.server_bytes] {
                if count > limits.connection_bytes || count > limits.bytes - retained {
                    return Err("network resources exhausted".into());
                }
                retained += count;
            }
        }
        Ok(())
    }
    pub fn register(&mut self, domain: Domain) -> Result<(), String> {
        let Some(node) = self.nodes.get_mut(&domain.node) else {
            return Err("stale file handle".into());
        };
        if self.finished
            || domain.run != self.run
            || domain.incarnation == 0
            || domain.address != node.address
            || self
                .live
                .get(&domain.node)
                .is_some_and(|previous| previous != &domain)
            || domain.incarnation <= node.last_incarnation
        {
            return Err("stale file handle".into());
        }
        node.last_incarnation = domain.incarnation;
        self.live.insert(domain.node.clone(), domain);
        Ok(())
    }
    #[allow(
        dead_code,
        reason = "The network corpus inspects detached snapshots; cluster evidence uses record."
    )]
    pub fn snapshot(&self) -> Result<history::Snapshot, String> {
        let transition_sha256 = history::transitions_digest(&self.transitions)?;
        let mut snapshot = self.snapshot_fields();
        snapshot.transition_sha256 = transition_sha256;
        snapshot.identity = history::snapshot_digest(&snapshot)?;
        Ok(snapshot)
    }

    fn snapshot_fields(&self) -> history::Snapshot {
        let mut snapshot = history::Snapshot {
            nodes: (!self.nodes.is_empty()).then(|| self.nodes.values().cloned().collect()),
            links: (!self.links.is_empty()).then(|| self.links.values().cloned().collect()),
            listeners: (!self.listeners.is_empty()).then(|| {
                self.listeners
                    .iter()
                    .map(|listener| history::ListenerSnapshot {
                        endpoint: listener.endpoint.clone(),
                        closed: listener.closed,
                    })
                    .collect()
            }),
            connections: (!self.connections.is_empty())
                .then(|| self.connections.values().cloned().collect()),
            next_connection: self.next_connection,
            ..Default::default()
        };
        if let Some(listeners) = &mut snapshot.listeners {
            listeners.sort_by_cached_key(|listener| {
                (history::endpoint_key(&listener.endpoint), !listener.closed)
            });
        }
        if let Some(links) = &mut snapshot.links {
            links.sort_by_cached_key(|link| Self::link_key(&link.from, &link.to));
        }
        snapshot
    }
}
