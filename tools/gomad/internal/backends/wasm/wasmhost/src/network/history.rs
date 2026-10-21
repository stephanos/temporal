use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

#[expect(
    clippy::trivially_copy_pass_by_ref,
    reason = "Serde skip_serializing_if passes fields by reference"
)]
pub(crate) fn zero(value: &u64) -> bool {
    *value == 0
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Endpoint {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub node: String,
    #[serde(default, skip_serializing_if = "zero")]
    pub incarnation: u64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub address: String,
    #[serde(default, skip_serializing_if = "zero")]
    pub port: u64,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Transition {
    pub ordinal: u64,
    pub kind: String,
    pub source: Endpoint,
    pub destination: Endpoint,
    #[serde(default, skip_serializing_if = "zero")]
    pub connection: u64,
    #[serde(default, skip_serializing_if = "zero")]
    pub bytes: u64,
    #[serde(default, skip_serializing_if = "zero")]
    pub count: u64,
    #[serde(default, skip_serializing_if = "zero")]
    pub delay_nanos: u64,
    pub outcome: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct NodeSnapshot {
    pub node: String,
    pub address: String,
    pub last_incarnation: u64,
    pub next_listener_port: u64,
    pub next_client_port: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct Link {
    pub from: String,
    pub to: String,
    pub enabled: bool,
    pub delay_nanos: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct ListenerSnapshot {
    pub endpoint: Endpoint,
    pub closed: bool,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct ConnectionSnapshot {
    pub client_bytes: u64,
    pub server_bytes: u64,
    pub identity: u64,
    pub client: Endpoint,
    pub server: Endpoint,
    pub closed: bool,
    pub reset: bool,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Snapshot {
    pub nodes: Option<Vec<NodeSnapshot>>,
    pub links: Option<Vec<Link>>,
    pub listeners: Option<Vec<ListenerSnapshot>>,
    pub connections: Option<Vec<ConnectionSnapshot>>,
    pub next_connection: u64,
    pub transition_sha256: String,
    pub identity: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct Record {
    pub transitions: Option<Vec<Transition>>,
    pub snapshot: Snapshot,
}

/// Inspection-only lane chains preserve canonical history's inter-lane
/// equivalence without re-encoding the accumulated transition transcript.
#[derive(Clone)]
pub(crate) struct InspectionHistory {
    lanes: std::collections::BTreeMap<String, (u64, [u8; 32])>,
    wire_length: usize,
    wire_error: Option<String>,
}

impl Default for InspectionHistory {
    fn default() -> Self {
        Self {
            lanes: std::collections::BTreeMap::default(),
            wire_length: 8,
            wire_error: None,
        }
    }
}

impl InspectionHistory {
    pub(crate) fn append(&mut self, transition: &Transition) {
        if self.wire_error.is_none() {
            let mut encoder = Encoder {
                length: self.wire_length,
                validate_only: true,
                ..Default::default()
            };
            self.wire_error = encoder.transition(transition).err();
            self.wire_length = encoder.length;
        }
        if self.wire_error.is_some() {
            return;
        }
        let mut transition = transition.clone();
        transition.ordinal = 0;
        let mut encoder = Encoder::default();
        encoder
            .transition(&transition)
            .expect("validated transition encoding");
        let key = lane(&transition);
        let (count, head) = self.lanes.entry(key).or_default();
        let mut hash = Sha256::new();
        hash.update(b"gomad/network/inspection-lane/v2\0");
        hash.update(*head);
        hash.update(count.to_le_bytes());
        hash.update((encoder.bytes.len() as u64).to_le_bytes());
        hash.update(&encoder.bytes);
        *head = hash.finalize().into();
        *count += 1;
    }

    pub(crate) fn check(&self) -> Result<(), String> {
        self.wire_error.clone().map_or(Ok(()), Err)
    }

    pub(crate) fn digest(&self) -> Result<String, String> {
        self.check()?;
        let mut hash = Sha256::new();
        hash.update(b"gomad/network/inspection-history/v2\0");
        hash.update((self.lanes.len() as u64).to_le_bytes());
        for (lane, (count, head)) in &self.lanes {
            hash.update((lane.len() as u64).to_le_bytes());
            hash.update(lane.as_bytes());
            hash.update(count.to_le_bytes());
            hash.update(head);
        }
        Ok(format!("sha256:{:x}", hash.finalize()))
    }
}

#[derive(Default)]
struct Encoder {
    bytes: Vec<u8>,
    length: usize,
    validate_only: bool,
}

impl Encoder {
    fn raw(&mut self, value: &[u8]) {
        self.length += value.len();
        if !self.validate_only {
            self.bytes.extend_from_slice(value);
        }
    }
    fn uint64(&mut self, value: u64) {
        self.raw(&value.to_le_bytes());
    }
    fn boolean(&mut self, value: bool) {
        self.uint64(u64::from(value));
    }
    fn string(&mut self, value: &str) -> Result<(), String> {
        if value.len() > 1 << 20 {
            return Err("simulation network wire string exceeds its byte limit".into());
        }
        self.uint64(value.len() as u64);
        self.raw(value.as_bytes());
        self.check_size()
    }
    fn check_size(&self) -> Result<(), String> {
        if self.length > 128 << 20 {
            Err("simulation network wire value exceeds its byte limit".into())
        } else {
            Ok(())
        }
    }
    fn endpoint(&mut self, endpoint: &Endpoint) -> Result<(), String> {
        self.string(&endpoint.node)?;
        self.uint64(endpoint.incarnation);
        self.string(&endpoint.address)?;
        self.uint64(endpoint.port);
        self.check_size()
    }
    fn transition(&mut self, transition: &Transition) -> Result<(), String> {
        self.uint64(transition.ordinal);
        self.string(&transition.kind)?;
        self.endpoint(&transition.source)?;
        self.endpoint(&transition.destination)?;
        self.uint64(transition.connection);
        self.uint64(transition.bytes);
        self.uint64(transition.count);
        self.uint64(transition.delay_nanos);
        self.string(&transition.outcome)
    }
}

fn digest(prefix: &[u8], bytes: &[u8]) -> String {
    let mut hash = Sha256::new();
    hash.update(prefix);
    hash.update(bytes);
    format!("sha256:{:x}", hash.finalize())
}

pub fn transitions_digest(transitions: &[Transition]) -> Result<String, String> {
    let mut encoder = Encoder::default();
    encode_transitions(&mut encoder, transitions)?;
    #[cfg(test)]
    DIGEST_COUNTS.with(|count| {
        let (transitions, snapshots) = count.get();
        count.set((transitions + 1, snapshots));
    });
    Ok(digest(
        b"gomad-simulation-network-transitions/v3\0",
        &encoder.bytes,
    ))
}

#[allow(
    dead_code,
    reason = "Detached history validation remains covered by the network conformance corpus."
)]
pub(crate) fn validate_transitions(transitions: &[Transition]) -> Result<(), String> {
    encode_transitions(
        &mut Encoder {
            validate_only: true,
            ..Default::default()
        },
        transitions,
    )
}

fn encode_transitions(encoder: &mut Encoder, transitions: &[Transition]) -> Result<(), String> {
    encoder.uint64(transitions.len() as u64);
    for transition in transitions {
        encoder.transition(transition)?;
    }
    Ok(())
}

/// A value of [`transitions_digest`]'s fixed length.
#[allow(
    dead_code,
    reason = "Detached history validation remains covered by the network conformance corpus."
)]
pub(crate) const TRANSITION_DIGEST_PLACEHOLDER: &str =
    "sha256:0000000000000000000000000000000000000000000000000000000000000000";

/// Fails exactly when [`snapshot_digest`] would.
#[allow(
    dead_code,
    reason = "Detached history validation remains covered by the network conformance corpus."
)]
pub(crate) fn validate_snapshot(snapshot: &Snapshot) -> Result<(), String> {
    encode_snapshot(
        &mut Encoder {
            validate_only: true,
            ..Default::default()
        },
        snapshot,
    )
}

pub fn snapshot_digest(snapshot: &Snapshot) -> Result<String, String> {
    let mut encoder = Encoder::default();
    encode_snapshot(&mut encoder, snapshot)?;
    #[cfg(test)]
    DIGEST_COUNTS.with(|count| {
        let (transitions, snapshots) = count.get();
        count.set((transitions, snapshots + 1));
    });
    Ok(digest(
        b"gomad-simulation-network-snapshot/v3\0",
        &encoder.bytes,
    ))
}

fn encode_snapshot(encoder: &mut Encoder, snapshot: &Snapshot) -> Result<(), String> {
    let nodes = snapshot.nodes.as_deref().unwrap_or_default();
    encoder.uint64(nodes.len() as u64);
    for node in nodes {
        encoder.string(&node.node)?;
        encoder.string(&node.address)?;
        encoder.uint64(node.last_incarnation);
        encoder.uint64(node.next_listener_port);
        encoder.uint64(node.next_client_port);
    }
    let links = snapshot.links.as_deref().unwrap_or_default();
    encoder.uint64(links.len() as u64);
    for link in links {
        encoder.string(&link.from)?;
        encoder.string(&link.to)?;
        encoder.boolean(link.enabled);
        encoder.uint64(link.delay_nanos);
    }
    let listeners = snapshot.listeners.as_deref().unwrap_or_default();
    encoder.uint64(listeners.len() as u64);
    for listener in listeners {
        encoder.endpoint(&listener.endpoint)?;
        encoder.boolean(listener.closed);
    }
    let connections = snapshot.connections.as_deref().unwrap_or_default();
    encoder.uint64(connections.len() as u64);
    for connection in connections {
        encoder.uint64(connection.identity);
        encoder.endpoint(&connection.client)?;
        encoder.endpoint(&connection.server)?;
        encoder.boolean(connection.closed);
        encoder.boolean(connection.reset);
        encoder.uint64(connection.client_bytes);
        encoder.uint64(connection.server_bytes);
    }
    encoder.uint64(snapshot.next_connection);
    encoder.string(&snapshot.transition_sha256)?;
    encoder.string("")
}

#[cfg(test)]
thread_local! {
    pub(crate) static DIGEST_COUNTS: std::cell::Cell<(usize, usize)> = const { std::cell::Cell::new((0, 0)) };
}

pub(crate) fn endpoint_key(endpoint: &Endpoint) -> String {
    format!(
        "{}\0{:020}\0{}\0{:05}",
        endpoint.node, endpoint.incarnation, endpoint.address, endpoint.port
    )
}

pub(crate) fn valid_endpoint(endpoint: &Endpoint, incarnation: bool) -> bool {
    !endpoint.node.is_empty()
        && !endpoint.address.is_empty()
        && (!incarnation || endpoint.incarnation != 0)
}

#[allow(
    dead_code,
    reason = "Detached history validation remains covered by the network conformance corpus."
)]
pub(crate) fn valid_transition(transition: &Transition) -> bool {
    match transition.kind.as_str() {
        "listen" | "dial" | "accept" | "write" | "close" | "listener_close" | "stop" | "crash"
        | "reset" => {
            if !valid_endpoint(&transition.source, true) {
                return false;
            }
        }
        "partition" | "heal" | "delay" | "disconnect" | "reconnect" | "directional_delay" => {
            if !valid_endpoint(&transition.source, false)
                || !valid_endpoint(&transition.destination, false)
                || transition.source.node == transition.destination.node
            {
                return false;
            }
        }
        _ => return false,
    }
    if !matches!(
        transition.outcome.as_str(),
        "ok" | "closed"
            | "refused"
            | "deadline"
            | "reset"
            | "capacity"
            | "unsupported"
            | "held"
            | "timeout"
    ) {
        return false;
    }
    if transition.kind == "reset"
        || transition.outcome == "held"
        || transition.outcome == "timeout" && transition.kind == "write"
    {
        if !matches!(transition.kind.as_str(), "reset" | "write")
            || transition.connection == 0
            || !valid_endpoint(&transition.destination, true)
            || transition.source.node == transition.destination.node
            || !(1..=65535).contains(&transition.source.port)
            || !(1..=65535).contains(&transition.destination.port)
        {
            return false;
        }
        if transition.kind == "reset" && !matches!(transition.outcome.as_str(), "reset" | "timeout")
        {
            return false;
        }
        if transition.outcome == "held" && transition.bytes == 0 {
            return false;
        }
    }
    if transition.outcome == "timeout"
        && !matches!(transition.kind.as_str(), "write" | "reset")
        && (transition.kind != "dial"
            || transition.connection != 0
            || transition.source.port > 65535
            || !valid_endpoint(&transition.destination, false)
            || transition.source.node == transition.destination.node
            || !(1..=65535).contains(&transition.destination.port))
    {
        return false;
    }
    true
}

pub(crate) fn lane(transition: &Transition) -> String {
    match transition.kind.as_str() {
        "partition" | "heal" | "delay" | "disconnect" | "reconnect" | "directional_delay" => {
            "topology".into()
        }
        "listen" | "listener_close" => format!("listener\0{}", endpoint_key(&transition.source)),
        "dial" | "accept" | "write" | "close" | "reset" if transition.connection != 0 => {
            format!("connection\0{:020}", transition.connection)
        }
        "dial" | "accept" | "write" | "close" | "reset" => format!(
            "operation\0{}\0{}\0{}",
            transition.kind,
            endpoint_key(&transition.source),
            endpoint_key(&transition.destination)
        ),
        "stop" | "crash" => format!("lifecycle\0{}", endpoint_key(&transition.source)),
        _ => format!("unknown\0{}", transition.kind),
    }
}

#[allow(
    dead_code,
    reason = "Detached history validation remains covered by the network conformance corpus."
)]
pub fn canonicalize(transitions: &mut [Transition]) {
    transitions.sort_by_cached_key(lane);
    for (ordinal, transition) in transitions.iter_mut().enumerate() {
        transition.ordinal = ordinal as u64;
    }
}
