//! Broker-shaped cluster networking over the shared Rust network owner
//! (fn-178.12): Go `wasi/broker_network.go` (configuration, nodes, links,
//! domains, central faults, live link changes, revocation) and the broker-mode
//! half of `wasi/network_v2.go` (`runtimeNetworkStream`, `networkV2Advance`).
//! One `Shared` owner serves every instance; each instance holds an `Endpoint`
//! with its handle maps, its per-instance network World and its operations log.
//! Under the in-process broker (fn-178.7) every World-visible call attaches at
//! broker time (`Endpoint::attach_at`), an idle settles the instance's network
//! World (`Endpoint::settle`), and the broker folds `Shared::has_ready`.
use crate::network::stream_abi::{self, Call, HEADER, MAXIMUM_CHUNK_BYTES};
use crate::network::{Domain, Model as Owner, OperationResult, Request, history};
use crate::world;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::sync::{Arc, Mutex};

pub const PROFILE: &str = "gomad.simulation-network-stream/v3";
/// Go `netmodel.MaximumPendingOperations`.
const MAXIMUM_PENDING_OPERATIONS: u64 = 1 << 16;

/// Go's failure classes for a network call: `BoundaryError` invalid and
/// capacity end the instance; any other Go error is an infrastructure outcome.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Failure {
    #[cfg_attr(
        feature = "conformance",
        allow(
            dead_code,
            reason = "Broker revocation constructs boundary failures; the conformance build excludes the broker."
        )
    )]
    Boundary(&'static str, &'static str),
    Invalid(&'static str),
    Capacity(&'static str),
    /// A broker World limit exhausted, with its accounting (fn-235).
    #[cfg_attr(
        feature = "conformance",
        allow(
            dead_code,
            reason = "Only the broker World raises it; the conformance build excludes the broker."
        )
    )]
    Exhausted {
        dimension: &'static str,
        limit: u64,
        used: u64,
        delta: u64,
    },
    Infrastructure(String),
}

impl Failure {
    pub fn termination(&self) -> &'static str {
        match self {
            Self::Boundary(kind, _) => kind,
            Self::Invalid(_) => "invalid",
            Self::Capacity(_) | Self::Exhausted { .. } => "capacity",
            Self::Infrastructure(_) => "infrastructure",
        }
    }

    pub fn message(&self) -> String {
        match self {
            Self::Boundary(kind, dimension) => format!("WASI {kind}: {dimension}"),
            Self::Invalid(dimension) => format!("WASI invalid: {dimension}"),
            Self::Capacity(dimension) => format!("WASI capacity: {dimension}"),
            Self::Exhausted {
                dimension,
                limit,
                used,
                delta,
            } => format!("WASI capacity: {dimension} limit={limit} used={used} delta={delta}"),
            Self::Infrastructure(message) => message.clone(),
        }
    }
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Error classification is passed directly to Result::map_err"
)]
pub(crate) fn world_failure(error: world::Error) -> Failure {
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

fn infrastructure(error: impl Into<String>) -> Failure {
    Failure::Infrastructure(error.into())
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Limits {
    pub listeners: u64,
    pub connections: u64,
    pub bytes: u64,
    pub connection_bytes: u64,
    pub stall_nanos: u64,
    pub transitions: u64,
}

/// Go `NetworkFaultV2`: a node-local fault, which brokered nodes refuse.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct LocalFault {
    pub name: String,
    pub action: String,
    pub at_nanos: u64,
    #[serde(default)]
    pub port: u32,
    #[serde(default)]
    pub delay_nanos: u64,
}

/// Go `NetworkModelV2`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ModelV2 {
    pub profile: String,
    pub limits: Limits,
    #[serde(default)]
    pub faults: Vec<LocalFault>,
}

/// Go `BrokerLink`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Link {
    pub from: String,
    pub to: String,
    pub enabled: bool,
    #[serde(default)]
    pub delay_nanos: u64,
}

/// Go `BrokerNetworkFault`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Fault {
    pub name: String,
    pub action: String,
    pub from: String,
    pub to: String,
    pub at_nanos: u64,
    #[serde(default)]
    pub delay_nanos: u64,
    #[serde(default)]
    pub symmetric: bool,
}

/// The cluster `network` object: Go `BrokerConfig`'s network fields.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub model: ModelV2,
    #[serde(default)]
    pub addresses: BTreeMap<String, String>,
    #[serde(default)]
    pub links: Vec<Link>,
    #[serde(default)]
    pub strict_topology: bool,
    #[serde(default)]
    pub faults: Vec<Fault>,
}

/// Go `validateNetworkModelV2`.
pub fn validate_model(config: &ModelV2) -> Result<(), Failure> {
    if config.profile != PROFILE {
        return Err(Failure::Invalid("network v2 profile"));
    }
    let limits = &config.limits;
    if limits.listeners == 0
        || limits.listeners > 4096
        || limits.connections == 0
        || limits.connections > 65536
        || limits.bytes == 0
        || limits.bytes > 128 << 20
        || limits.transitions == 0
        || limits.transitions > 1 << 20
    {
        return Err(Failure::Invalid("network v2 limits"));
    }
    if limits.connection_bytes == 0
        || limits.connection_bytes > 1 << 30
        || limits.stall_nanos == 0
        || limits.stall_nanos > i64::MAX as u64 - 1
    {
        return Err(Failure::Invalid("network v2 byte or stall limits"));
    }
    if config.faults.len() as u64 > limits.transitions {
        return Err(Failure::Invalid("network v2 fault capacity"));
    }
    let mut names = BTreeSet::new();
    for (index, fault) in config.faults.iter().enumerate() {
        if fault.name.is_empty()
            || fault.name.len() > 1024
            || fault.name.contains('\0')
            || !names.insert(fault.name.as_str())
            || fault.at_nanos > i64::MAX as u64
            || fault.delay_nanos > i64::MAX as u64
            || fault.port > 65535
            || index > 0 && config.faults[index - 1].at_nanos > fault.at_nanos
        {
            return Err(Failure::Invalid("network v2 fault schedule"));
        }
        if !matches!(
            fault.action.as_str(),
            "partition" | "heal" | "reset" | "delay"
        ) {
            return Err(Failure::Invalid("network v2 fault action"));
        }
        if fault.action != "reset" && fault.port != 0 {
            return Err(Failure::Invalid(
                "network v2 topology action cannot select a port",
            ));
        }
        if fault.action != "delay" && fault.delay_nanos != 0 {
            return Err(Failure::Invalid(
                "network v2 delay applies only to delay action",
            ));
        }
    }
    Ok(())
}

/// Go `brokerIPv4`: four canonical decimal octets.
fn ipv4(address: &str) -> bool {
    let parts: Vec<&str> = address.split('.').collect();
    parts.len() == 4
        && parts.iter().all(|part| {
            !part.is_empty()
                && part.len() <= 3
                && !(part.len() > 1 && part.starts_with('0'))
                && part.bytes().all(|byte| byte.is_ascii_digit())
                && part.parse::<u16>().is_ok_and(|value| value <= 255)
        })
}

/// Go `networkV2ClockCeiling`.
pub fn clock_ceiling(epoch: u64) -> Result<u64, Failure> {
    if epoch >= i64::MAX as u64 {
        return Err(Failure::Invalid("network v2 clock epoch"));
    }
    Ok((i64::MAX - world::INITIAL_TIME)
        .min(i64::MAX - epoch.cast_signed())
        .cast_unsigned())
}

/// The shared owner and the broker's network configuration.
pub struct Shared {
    owner: TrackedOwner,
    addresses: BTreeMap<String, String>,
    clients: BTreeMap<String, String>,
    faults: Vec<Fault>,
    fault: usize,
    simulation: Option<crate::linked_simulation::Shared>,
}

/// The shared owner with a revision that every mutable access advances, so
/// equal revisions denote an identical owner `Record`.
struct TrackedOwner {
    owner: Owner,
    revision: u64,
}

impl std::ops::Deref for TrackedOwner {
    type Target = Owner;
    fn deref(&self) -> &Owner {
        &self.owner
    }
}

impl std::ops::DerefMut for TrackedOwner {
    fn deref_mut(&mut self) -> &mut Owner {
        self.revision += 1;
        &mut self.owner
    }
}

/// Process-unique endpoint identities for inspection evidence keys.
static ENDPOINT_IDENTITIES: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);

/// Equal keys denote identical [`Endpoint::evidence`] bytes: the endpoint's
/// identity and mutation revision, and the shared owner's revision.
type EvidenceKey = (u64, u64, u64);

fn joined(results: [Result<(), String>; 2]) -> Result<(), String> {
    let errors: Vec<String> = results.into_iter().filter_map(Result::err).collect();
    if errors.is_empty() {
        Ok(())
    } else {
        Err(errors.join("\n"))
    }
}

impl Shared {
    pub(crate) fn attach_simulation(&mut self, controller: crate::linked_simulation::Shared) {
        self.simulation = Some(controller);
    }

    fn simulation_choice(
        &self,
        dimension: crate::linked_simulation::Dimension,
        site: &str,
        alternatives: Vec<String>,
    ) -> Result<usize, Failure> {
        let Some(controller) = &self.simulation else {
            return Ok(0);
        };
        if alternatives.len() < 2 {
            return Ok(0);
        }
        let selected = controller
            .lock()
            .map_err(|_| infrastructure("simulation controller poisoned"))?
            .choose(dimension, site, alternatives, 0)
            .map_err(infrastructure)?;
        usize::try_from(selected).map_err(|error| infrastructure(error.to_string()))
    }

    /// Go `newBrokerNetwork` for the cluster's instances (sorted ids, all at
    /// incarnation 1), then the broker's initial advance to time 1.
    pub fn new(config: &Config, ids: &[String]) -> Result<Self, Failure> {
        Self::new_with_simulation(config, ids, None)
    }

    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub(crate) fn new_with_simulation(
        config: &Config,
        ids: &[String],
        simulation: Option<crate::linked_simulation::Shared>,
    ) -> Result<Self, Failure> {
        validate_model(&config.model)?;
        if !config.model.faults.is_empty() {
            return Err(Failure::Invalid("broker requires central fault schedule"));
        }
        let mut addresses = BTreeMap::new();
        let mut clients = BTreeMap::new();
        let mut nodes = Vec::new();
        let mut owners = BTreeMap::new();
        for (index, id) in ids.iter().enumerate() {
            let slot = index + 1;
            let address = config
                .addresses
                .get(id)
                .filter(|address| !address.is_empty())
                .cloned()
                .unwrap_or_else(|| format!("10.0.{}.{}", slot / 254, slot % 254 + 1));
            let client = format!("10.1.{}.{}", slot / 254, slot % 254 + 1);
            if !ipv4(&address) {
                return Err(Failure::Invalid("broker IPv4 address"));
            }
            nodes.push(crate::network::Node {
                node: format!("{id}.server"),
                address: address.clone(),
            });
            nodes.push(crate::network::Node {
                node: format!("{id}.client"),
                address: client.clone(),
            });
            owners.insert(format!("{id}.server"), id.clone());
            owners.insert(format!("{id}.client"), id.clone());
            addresses.insert(id.clone(), address);
            clients.insert(id.clone(), client);
        }
        let mut topology = BTreeMap::new();
        for link in &config.links {
            if link.from == link.to
                || !addresses.contains_key(&link.from)
                || !addresses.contains_key(&link.to)
            {
                return Err(Failure::Invalid("broker link endpoint"));
            }
            if topology
                .insert((link.from.clone(), link.to.clone()), link.clone())
                .is_some()
            {
                return Err(Failure::Invalid("broker duplicate link"));
            }
        }
        nodes.sort_by(|left, right| left.node.cmp(&right.node));
        let mut links = Vec::new();
        for from in &nodes {
            for to in &nodes {
                if from.node == to.node {
                    continue;
                }
                let (from_owner, to_owner) = (&owners[&from.node], &owners[&to.node]);
                let configured = topology.get(&(from_owner.clone(), to_owner.clone()));
                let (enabled, delay_nanos) = match configured {
                    _ if from_owner == to_owner => (true, 0),
                    None if !config.strict_topology => (true, 0),
                    None => continue,
                    Some(link) => (link.enabled, link.delay_nanos),
                };
                links.push(history::Link {
                    from: from.node.clone(),
                    to: to.node.clone(),
                    enabled,
                    delay_nanos,
                });
            }
        }
        let limits = &config.model.limits;
        let mut owner = Owner::new(
            1,
            crate::network::Config {
                nodes,
                links,
                limits: crate::network::Limits {
                    listeners: limits.listeners,
                    connections: limits.connections,
                    bytes: limits.bytes,
                    connection_bytes: limits.connection_bytes,
                    stall_nanos: limits.stall_nanos,
                    transitions: limits.transitions,
                },
                replay: None,
            },
        )
        .map_err(infrastructure)?;
        owner
            .limit_clock_ceiling((i64::MAX - world::INITIAL_TIME) as u64)
            .map_err(infrastructure)?;
        let mut shared = Self {
            owner: TrackedOwner { owner, revision: 0 },
            addresses,
            clients,
            faults: config.faults.clone(),
            fault: 0,
            simulation: None,
        };
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(crate::event_order::owner("controller", 0, 1));
        for id in ids {
            shared.register(id, 1).map_err(infrastructure)?;
        }
        let mut seen = BTreeSet::new();
        for (index, fault) in shared.faults.iter().enumerate() {
            if fault.name.is_empty()
                || !seen.insert(fault.name.as_str())
                || !shared.addresses.contains_key(&fault.from)
                || !shared.addresses.contains_key(&fault.to)
                || index > 0 && shared.faults[index - 1].at_nanos > fault.at_nanos
            {
                return Err(Failure::Invalid("broker fault schedule"));
            }
            if !matches!(fault.action.as_str(), "partition" | "heal" | "delay") {
                return Err(Failure::Invalid("broker fault action"));
            }
        }
        if let Some(simulation) = simulation {
            shared.attach_simulation(simulation);
        }
        shared.advance(1)?;
        Ok(shared)
    }

    #[cfg(test)]
    pub fn address(&self, id: &str) -> Option<&str> {
        self.addresses.get(id).map(String::as_str)
    }

    /// Go `brokerNetwork.domains`: the client and server domain of one
    /// incarnation.
    pub fn domains(&self, id: &str, incarnation: u64) -> (Domain, Domain) {
        (
            Domain {
                run: 1,
                node: format!("{id}.client"),
                incarnation,
                address: self.clients[id].clone(),
            },
            Domain {
                run: 1,
                node: format!("{id}.server"),
                incarnation,
                address: self.addresses[id].clone(),
            },
        )
    }

    pub fn register(&mut self, id: &str, incarnation: u64) -> Result<(), String> {
        let (client, server) = self.domains(id, incarnation);
        joined([self.owner.register(client), self.owner.register(server)])
    }

    /// Go `brokerNetwork.revoke`: both domains, with the broker's graceful flag
    /// (stop is graceful, crash is not).
    pub fn revoke(&mut self, id: &str, incarnation: u64, graceful: bool) -> Result<(), String> {
        let (client, server) = self.domains(id, incarnation);
        joined([
            self.owner.revoke(&client, graceful),
            self.owner.revoke(&server, graceful),
        ])
    }

    /// Go `brokerNetwork.deadline`: the owner's next deadline folded with the
    /// next scheduled fault (0 when there is neither).
    pub fn deadline(&self) -> u64 {
        let deadline = self.owner.next_deadline();
        match self.faults.get(self.fault) {
            Some(fault) if deadline == 0 || fault.at_nanos < deadline => fault.at_nanos,
            _ => deadline,
        }
    }

    /// Go `brokerNetwork.advance`: due faults at their own time, then `now`.
    pub fn advance(&mut self, now: u64) -> Result<(), Failure> {
        while let Some(next) = self.faults.get(self.fault)
            && next.at_nanos <= now
        {
            if self.simulation.is_some() {
                let at = next.at_nanos;
                let count = self.faults[self.fault..]
                    .iter()
                    .take_while(|fault| fault.at_nanos == at)
                    .count();
                let alternatives = self.faults[self.fault..self.fault + count]
                    .iter()
                    .map(|fault| {
                        crate::linked_simulation::domain_hash(
                            "gomad-simulation-network-fault/v1",
                            &serde_json::json!(fault),
                        )
                    })
                    .collect::<Result<Vec<_>, _>>()
                    .map_err(infrastructure)?;
                let site = crate::linked_simulation::domain_hash(
                    "gomad-simulation-network-site/v1",
                    &serde_json::json!({"fault_at_nanos":at}),
                )
                .map_err(infrastructure)?;
                let selected = self.simulation_choice(
                    crate::linked_simulation::Dimension::Fault,
                    &site,
                    alternatives,
                )?;
                let fault = self.faults.remove(self.fault + selected);
                self.faults.insert(self.fault, fault);
            }
            let fault = self.faults[self.fault].clone();
            self.owner
                .advance_before(fault.at_nanos)
                .map_err(infrastructure)?;
            self.apply_link(
                &fault.action,
                &fault.from,
                &fault.to,
                fault.delay_nanos,
                fault.symmetric,
            )?;
            self.fault += 1;
        }
        self.owner.advance_before(now).map_err(infrastructure)?;
        self.owner.advance(now).map_err(infrastructure)?;
        Ok(())
    }

    /// Go `Broker.ChangeLink`: a live link change between two instances.
    pub fn change_link(
        &mut self,
        action: &str,
        from: &str,
        to: &str,
        delay_nanos: u64,
        symmetric: bool,
    ) -> Result<(), Failure> {
        if !self.addresses.contains_key(from) || !self.addresses.contains_key(to) || from == to {
            return Err(Failure::Invalid("broker link"));
        }
        self.apply_link(action, from, to, delay_nanos, symmetric)
    }

    /// Go `brokerNetwork.changeLink`: all four client/server pairs.
    fn apply_link(
        &mut self,
        action: &str,
        from: &str,
        to: &str,
        delay_nanos: u64,
        symmetric: bool,
    ) -> Result<(), Failure> {
        if !matches!(action, "partition" | "heal" | "delay") {
            return Err(Failure::Invalid("broker link action"));
        }
        self.owner
            .change_group(
                &[format!("{from}.client"), format!("{from}.server")],
                &[format!("{to}.client"), format!("{to}.server")],
                action,
                delay_nanos,
                symmetric,
            )
            .map_err(infrastructure)
    }

    pub fn record(&self) -> Result<history::Record, String> {
        self.owner.record()
    }

    /// Go `len(n.model.Ready()) != 0`, consulted before a broker idle.
    #[cfg(not(feature = "conformance"))]
    pub fn has_ready(&mut self) -> bool {
        self.owner.ready().is_some_and(|ids| !ids.is_empty())
    }
}

/// One instance incarnation's view of the shared owner.
pub struct Endpoint {
    shared: Arc<Mutex<Shared>>,
    request_base: u64,
    client: Domain,
    server: Domain,
    epoch: u64,
    attached: bool,
    transcript_limit: u64,
    handles: BTreeMap<u64, Domain>,
    local: Option<standalone::Local>,
    requests: BTreeMap<u64, Domain>,
    world: world::Model,
    world_requests: BTreeMap<u64, u64>,
    world_ready: BTreeSet<u64>,
    log_json_bytes: u64,
    entries: Vec<u8>,
    entries_hash: Sha256,
    entry_count: u64,
    /// Process-unique identity and a revision every public `&mut self`
    /// entry point advances (private fields: no other writer exists).
    identity: u64,
    revision: u64,
    /// The inspection commitment of the last evidence built for a key.
    inspection: Option<(EvidenceKey, [u8; 32])>,
}

fn lock(shared: &Mutex<Shared>) -> std::sync::MutexGuard<'_, Shared> {
    shared
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

impl Endpoint {
    pub(crate) fn limit_transition_bytes(&mut self, limit: Option<u64>) -> Result<(), Failure> {
        self.world
            .limit_transition_bytes(limit)
            .map_err(world_failure)
    }

    /// `slot` is the broker slot (instance order, from 1); `local` is the
    /// instance's own network profile, whose limits size its network World.
    pub fn new(
        shared: Arc<Mutex<Shared>>,
        id: &str,
        incarnation: u64,
        slot: u64,
        local: &ModelV2,
        epoch: u64,
        transcript_limit: u64,
    ) -> Result<Self, Failure> {
        let (client, server) = lock(&shared).domains(id, incarnation);
        let transitions = local.limits.transitions;
        let world = world::Model::new(world::Config {
            seed: 0,
            limits: world::Limits {
                max_requests: transitions * 2,
                max_events: transitions * 2,
                max_queued_events: MAXIMUM_PENDING_OPERATIONS.min(transitions * 2),
                max_transitions: transitions * 8,
                max_payload_bytes: local.limits.bytes * 8,
                max_string_bytes: 1024,
            },
        })
        .map_err(|error| infrastructure(error.to_string()))?;
        Ok(Self {
            shared,
            request_base: slot << 48,
            client,
            server,
            epoch,
            attached: false,
            transcript_limit,
            handles: BTreeMap::new(),
            local: None,
            requests: BTreeMap::new(),
            world,
            world_requests: BTreeMap::new(),
            world_ready: BTreeSet::new(),
            log_json_bytes: 0,
            entries: Vec::new(),
            entries_hash: Sha256::new(),
            entry_count: 0,
            identity: ENDPOINT_IDENTITIES.fetch_add(1, std::sync::atomic::Ordering::Relaxed),
            revision: 0,
            inspection: None,
        })
    }

    /// Go `brokerNetwork.attach` on the first stream call: narrow the shared
    /// owner's clock ceiling to this instance's epoch headroom. The broker also
    /// advances the owner to broker time on every visible call (`attach_at`).
    fn attach(&mut self, shared: &mut Shared) -> Result<(), Failure> {
        if !self.attached {
            let ceiling = clock_ceiling(self.epoch)?;
            shared
                .owner
                .limit_clock_ceiling(ceiling)
                .map_err(infrastructure)?;
            self.attached = true;
        }
        Ok(())
    }

    /// Go `brokerNetwork.attach` on a broker-visible call: attach once, then
    /// advance the shared owner to broker time.
    pub fn attach_at(&mut self, current: u64) -> Result<(), Failure> {
        self.revision += 1;
        let shared = self.shared.clone();
        let mut shared = lock(&shared);
        self.attach(&mut shared)?;
        shared.advance(current)
    }

    /// Go `networkV2Advance` at the instance's resumed time after a broker idle.
    pub fn settle(&mut self, now: u64) -> Result<(), Failure> {
        self.revision += 1;
        let shared = self.shared.clone();
        let mut shared = lock(&shared);
        self.advance(&mut shared, now)
    }

    fn append(&mut self, json: impl AsRef<[u8]>, binary: Vec<u8>) -> Result<(), Failure> {
        let length = json.as_ref().len() as u64 + 1;
        if self.log_json_bytes.saturating_add(length) > self.transcript_limit {
            return Err(Failure::Capacity("network-v2-transcript-bytes"));
        }
        self.log_json_bytes += length;
        if let Some(local) = &mut self.local {
            local.log.push(json.as_ref().to_vec());
        }
        if self.local.is_none() {
            self.entries_hash.update(&binary);
            self.entries.extend(binary);
        }
        self.entry_count += 1;
        Ok(())
    }

    /// Go `networkV2Advance` in broker mode (no node-local faults): settle the
    /// shared owner at `now`, deliver this instance's newly ready requests
    /// through its network World, and log the quiescence.
    fn advance(&mut self, shared: &mut Shared, now: u64) -> Result<(), Failure> {
        self.advance_faults(shared, now)?;
        shared.owner.advance_before(now).map_err(infrastructure)?;
        let ids = shared
            .owner
            .advance(now)
            .map_err(infrastructure)?
            .unwrap_or_default();
        let ids = if shared.simulation.is_some() {
            self.simulation_ready_order(shared, ids, now)?
        } else {
            ids
        };
        let mut ready = false;
        for global in ids {
            if global < self.request_base || global - self.request_base >= 1 << 48 {
                continue;
            }
            let id = global - self.request_base;
            if self.world_ready.contains(&id) {
                continue;
            }
            let Some(&request) = self.world_requests.get(&id) else {
                continue;
            };
            self.world
                .ready(world::Readiness {
                    request_id: request,
                    at: world::INITIAL_TIME + now.cast_signed(),
                    kind: "network".into(),
                    payload: Vec::new(),
                    equivalence_class: String::new(),
                })
                .map_err(world_failure)?;
            self.world_ready.insert(id);
            ready = true;
            if shared.simulation.is_some() {
                self.deliver_ready(now)?;
                ready = false;
            }
        }
        if ready {
            self.deliver_ready(now)?;
        }
        Ok(())
    }

    fn deliver_ready(&mut self, now: u64) -> Result<(), Failure> {
        let quiescence = self.world.quiesce().map_err(world_failure)?;
        if quiescence.after != world::INITIAL_TIME + now.cast_signed() {
            return Err(Failure::Invalid("network readiness advanced outside idle"));
        }
        let (json, binary) = evidence::quiescence_entry(&quiescence);
        self.append(json, binary)
    }

    fn simulation_ready_order(
        &self,
        shared: &Shared,
        ids: Vec<u64>,
        now: u64,
    ) -> Result<Vec<u64>, Failure> {
        let mut pending = ids
            .into_iter()
            .filter(|global| {
                *global >= self.request_base
                    && *global - self.request_base < 1 << 48
                    && !self.world_ready.contains(&(*global - self.request_base))
                    && self
                        .world_requests
                        .contains_key(&(*global - self.request_base))
            })
            .map(|global| {
                let pending = shared.owner.pending.get(&global)
                    .ok_or_else(|| infrastructure("simulation ready request is missing"))?;
                let request = &pending.request;
                crate::linked_simulation::domain_hash(
                    "gomad-simulation-network-ready/v1",
                    &serde_json::json!({
                        "request_id":global,
                        "request":{
                            "domain":request.domain,"op":request.op,"handle":request.handle,
                            "host":request.host,"port":request.port,"bytes":request.bytes,
                            "count":request.count,"deadline":request.deadline
                        },
                        "accepted":pending.accepted,"deadline":pending.deadline,"result":pending.result
                    }),
                )
                .map(|identity| (global, identity))
                .map_err(infrastructure)
            })
            .collect::<Result<Vec<_>, _>>()
            ?;
        let site = crate::linked_simulation::domain_hash("gomad-simulation-network-site/v1",
            &serde_json::json!({"node":self.client.node,"incarnation":self.client.incarnation,"at_nanos":now}))
            .map_err(infrastructure)?;
        let mut ordered = Vec::with_capacity(pending.len());
        while !pending.is_empty() {
            let selected = shared.simulation_choice(
                crate::linked_simulation::Dimension::Network,
                &site,
                pending
                    .iter()
                    .map(|(_, identity)| identity.clone())
                    .collect(),
            )?;
            ordered.push(pending.remove(selected).0);
        }
        Ok(ordered)
    }

    /// Go `runtimeNetworkStream`: serves one `net_stream` buffer in place.
    /// `now` is the instance's modeled clock; `runtime` is whether its
    /// cooperative runtime is configured and not finished.
    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub fn serve(&mut self, data: &mut [u8], now: u64, runtime: bool) -> Result<(), Failure> {
        self.revision += 1;
        // The broker attaches before the environment sees the buffer.
        let shared = self.shared.clone();
        self.attach(&mut lock(&shared))?;
        let call: Call = stream_abi::decode(data).map_err(|invalid| Failure::Invalid(invalid.0))?;
        if call.op == 0 {
            stream_abi::query(&call, data.len()).map_err(|invalid| Failure::Invalid(invalid.0))?;
            return Ok(());
        }
        if call.op == stream_abi::READINESS_PROBE_V1 && call.id == 0 {
            if !runtime {
                return Err(Failure::Invalid("network v2 readiness probe layout"));
            }
            let ids = stream_abi::readiness_probe_v1(&call, data)
                .map_err(|invalid| Failure::Invalid(invalid.0))?;
            let mut shared = lock(&shared);
            for &id in &ids {
                let valid = self.requests.get(&id).is_some_and(|domain| {
                    shared
                        .owner
                        .pending
                        .get(&(self.request_base + id))
                        .is_some_and(|pending| pending.request.domain == *domain)
                });
                if !valid {
                    return Err(Failure::Invalid("network v2 readiness probe request"));
                }
            }
            self.advance(&mut shared, now)?;
            for (index, &id) in ids.iter().enumerate() {
                let ready = shared.owner.pending[&(self.request_base + id)]
                    .result
                    .is_some();
                stream_abi::set_u64(data, HEADER + 8 * index, if ready { id } else { 0 });
            }
            return Ok(());
        }
        if !runtime
            || call.id == 0
            || call.id >= 1 << 48
            || call.count as usize > MAXIMUM_CHUNK_BYTES
            || call.port > 65535
        {
            return Err(Failure::Invalid("network v2 operation"));
        }
        let mut shared = lock(&shared);
        self.advance(&mut shared, now)?;
        let Some(&name) = stream_abi::OPERATIONS.get(call.op as usize) else {
            return Err(Failure::Invalid("network v2 opcode"));
        };
        stream_abi::check_payload(&call, data.len())
            .map_err(|invalid| Failure::Invalid(invalid.0))?;
        let domain = match call.op {
            1 => self.server.clone(),
            2 => self.client.clone(),
            12 | 13 => self
                .requests
                .get(&call.id)
                .cloned()
                .unwrap_or_else(empty_domain),
            _ => self
                .handles
                .get(&call.handle)
                .cloned()
                .unwrap_or_else(empty_domain),
        };
        let mut host = self.server.address.clone();
        if call.op == 2 && call.address != 0 && call.address >> 24 != 127 {
            let octets = call.address.to_be_bytes();
            host = format!("{}.{}.{}.{}", octets[0], octets[1], octets[2], octets[3]);
        }
        let request = Request {
            domain: domain.clone(),
            op: name.to_owned(),
            handle: call.handle,
            host,
            port: i64::from(call.port),
            bytes: if call.op == 5 {
                data[HEADER..].to_vec()
            } else {
                Vec::new()
            },
            count: call.count,
            deadline: call.deadline,
            request_id: self.request_base + call.id,
        };
        let mut result = if call.op == 13 {
            let mut result = OperationResult {
                request_id: request.request_id,
                state: "completed".into(),
                ..OperationResult::default()
            };
            if shared.owner.cancel(&domain, request.request_id).is_err() {
                // Go's closed owner reports only the stale-handle sentinel.
                result.error = "stale file handle".into();
            } else {
                shared.owner.dispatch(Request {
                    domain: domain.clone(),
                    op: "poll".into(),
                    handle: 0,
                    host: String::new(),
                    port: 0,
                    bytes: Vec::new(),
                    count: 0,
                    deadline: 0,
                    request_id: request.request_id,
                });
            }
            if let Some(&world_request) = self.world_requests.get(&call.id) {
                self.world.cancel(world_request).map_err(world_failure)?;
            }
            result
        } else {
            shared.owner.dispatch(request.clone())
        };
        drop(shared);
        result.request_id = call.id;
        if result.bytes.len() > data.len() - HEADER {
            return Err(Failure::Invalid(
                "network v2 completed bytes exceed caller payload",
            ));
        }
        data[36..56].fill(0);
        if result.state == "pending" {
            self.requests.insert(call.id, domain);
            if !self.world_requests.contains_key(&call.id) {
                let world_request = self
                    .world
                    .register(world::Request {
                        kind: "network".into(),
                        resource: world::ResourceId {
                            adapter: "wasm".into(),
                            kind: "stream".into(),
                            key: call.id.to_string(),
                        },
                        priority: 0,
                        payload: Vec::new(),
                    })
                    .map_err(world_failure)?;
                self.world_requests.insert(call.id, world_request);
            }
            stream_abi::set_u32(data, 40, 1);
        } else {
            if result.handle != 0 {
                if let Some(local) = &mut self.local {
                    let port = if call.op == 1 || call.op == 3 {
                        result.local.port
                    } else {
                        result.remote.port
                    };
                    local.ports.insert(result.handle, port);
                }
                self.handles.insert(result.handle, domain);
            }
            self.requests.remove(&call.id);
            self.world_requests.remove(&call.id);
            self.world_ready.remove(&call.id);
            stream_abi::set_u64(data, 16, result.handle);
            stream_abi::set_u32(
                data,
                36,
                u32::try_from(result.count).expect("network count is bounded by the u32 request"),
            );
            stream_abi::set_u32(data, 44, stream_abi::errno(&result.error));
            stream_abi::set_u32(
                data,
                48,
                u32::try_from(result.local.port).expect("network ports are bounded by 65535"),
            );
            stream_abi::set_u32(
                data,
                52,
                u32::try_from(result.remote.port).expect("network ports are bounded by 65535"),
            );
            if call.op == 4 || call.op == 12 {
                data[HEADER..HEADER + result.bytes.len()].copy_from_slice(&result.bytes);
            }
        }
        let (json, binary) = evidence::operation_entry(now, &request, &result);
        let reply = self.append(json, binary);
        #[cfg(feature = "stress-jitter")]
        crate::stress_jitter::delay();
        reply
    }

    /// The instance's binary network evidence at its termination: the
    /// operations log, the shared owner's Record and its network World.
    pub fn evidence(&self) -> Result<Vec<u8>, Failure> {
        self.evidence_locked(&lock(&self.shared))
    }

    fn evidence_locked(&self, shared: &Shared) -> Result<Vec<u8>, Failure> {
        if self.local.is_some() {
            return self.standalone_evidence(shared);
        }
        let record = shared.record().map_err(infrastructure)?;
        let mut wire = Vec::new();
        world::wire_generated::write_snapshot(&self.world, &mut wire)
            .map_err(|error| infrastructure(error.to_string()))?;
        Ok(evidence::document(
            self.log_json_bytes,
            self.entry_count,
            &self.entries,
            &record,
            &wire,
        ))
    }

    fn inspection_material(&self, native: &str, entries: [u8; 32]) -> Vec<u8> {
        let mut bytes = b"gomad/network/inspection-material/v2\0".to_vec();
        for value in [PROFILE.as_bytes(), native.as_bytes()] {
            bytes.extend_from_slice(&(value.len() as u64).to_le_bytes());
            bytes.extend_from_slice(value);
        }
        for value in [
            self.log_json_bytes,
            self.entry_count,
            self.entries.len() as u64,
        ] {
            bytes.extend_from_slice(&value.to_le_bytes());
        }
        bytes.extend_from_slice(&entries);
        bytes.extend_from_slice(&self.world.inspection_commitment());
        bytes
    }

    #[cfg(test)]
    pub(crate) fn inspection_network_eager(&self) -> Result<Vec<u8>, Failure> {
        let shared = lock(&self.shared);
        let native = shared
            .owner
            .inspection_digest_eager()
            .map_err(infrastructure)?;
        Ok(self.inspection_material(&native, Sha256::digest(&self.entries).into()))
    }

    /// Commits bounded current state plus append-only transcript heads.
    /// Full canonical evidence remains available through [`Self::evidence`].
    pub fn inspection_network(
        &mut self,
        commit: impl FnOnce(&[u8]) -> [u8; 32],
    ) -> Result<([u8; 32], Option<Vec<u8>>), Failure> {
        let shared = self.shared.clone();
        let shared = lock(&shared);
        let key = (self.identity, self.revision, shared.owner.revision);
        if let Some((cached, digest)) = self.inspection
            && cached == key
        {
            return Ok((digest, None));
        }
        self.inspection = None;
        let evidence = if self.local.is_some() {
            self.evidence_locked(&shared)?
        } else {
            let native = shared.owner.inspection_digest().map_err(infrastructure)?;
            self.inspection_material(&native, self.entries_hash.clone().finalize().into())
        };
        let digest = commit(&evidence);
        self.inspection = Some((key, digest));
        Ok((digest, Some(evidence)))
    }

    /// Fails exactly when [`Self::evidence`] would, without building it: the
    /// owner `Record` is its only fallible part, since the network World's
    /// wire bounds every string by its 1 KiB limit and writes to memory.
    pub fn evidence_check(&self) -> Result<(), Failure> {
        if self.local.is_some() {
            return self.evidence_locked(&lock(&self.shared)).map(|_| ());
        }
        lock(&self.shared)
            .owner
            .inspection_check()
            .map_err(infrastructure)
    }

    #[cfg(not(feature = "conformance"))]
    pub fn transcript_limit(&self) -> u64 {
        self.transcript_limit
    }

    #[cfg(test)]
    pub fn handles(&self) -> &BTreeMap<u64, Domain> {
        &self.handles
    }
}

fn empty_domain() -> Domain {
    Domain {
        run: 0,
        node: String::new(),
        incarnation: 0,
        address: String::new(),
    }
}

pub mod evidence;
mod standalone;

#[cfg(test)]
mod tests;

#[cfg(test)]
mod corpus;
