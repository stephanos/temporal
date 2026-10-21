use super::{
    Domain, Endpoint, Failure, LocalFault, ModelV2, Owner, PROFILE, Shared, TrackedOwner,
    clock_ceiling, evidence, history, infrastructure, lock, stream_abi, validate_model, world,
};
use std::{
    collections::BTreeMap,
    sync::{Arc, Mutex},
};

pub(super) struct Local {
    faults: Vec<LocalFault>,
    next: usize,
    pub(super) ports: BTreeMap<u64, u64>,
    pub(super) log: Vec<Vec<u8>>,
}

#[cfg(test)]
mod corpus;

impl Endpoint {
    pub(crate) fn standalone(
        config: ModelV2,
        epoch: u64,
        transcript_limit: u64,
    ) -> Result<Self, Failure> {
        validate_model(&config)?;
        let ceiling = clock_ceiling(epoch)?;
        let mut owner = Owner::new(
            1,
            crate::network::Config {
                nodes: vec![
                    crate::network::Node {
                        node: "guest.client".into(),
                        address: "127.0.0.2".into(),
                    },
                    crate::network::Node {
                        node: "guest.server".into(),
                        address: "127.0.0.1".into(),
                    },
                ],
                links: vec![
                    history::Link {
                        from: "guest.client".into(),
                        to: "guest.server".into(),
                        enabled: true,
                        delay_nanos: 0,
                    },
                    history::Link {
                        from: "guest.server".into(),
                        to: "guest.client".into(),
                        enabled: true,
                        delay_nanos: 0,
                    },
                ],
                limits: crate::network::Limits {
                    listeners: config.limits.listeners,
                    connections: config.limits.connections,
                    bytes: config.limits.bytes,
                    connection_bytes: config.limits.connection_bytes,
                    stall_nanos: config.limits.stall_nanos,
                    transitions: config.limits.transitions,
                },
                replay: None,
            },
        )
        .map_err(infrastructure)?;
        owner.limit_clock_ceiling(ceiling).map_err(infrastructure)?;
        for (node, address) in [("guest.client", "127.0.0.2"), ("guest.server", "127.0.0.1")] {
            owner
                .register(Domain {
                    run: 1,
                    node: node.into(),
                    incarnation: 1,
                    address: address.into(),
                })
                .map_err(infrastructure)?;
        }
        let shared = Arc::new(Mutex::new(Shared {
            owner: TrackedOwner { owner, revision: 0 },
            addresses: BTreeMap::from([("guest".into(), "127.0.0.1".into())]),
            clients: BTreeMap::from([("guest".into(), "127.0.0.2".into())]),
            faults: Vec::new(),
            fault: 0,
            simulation: None,
        }));
        let mut endpoint = Self::new(shared, "guest", 1, 0, &config, epoch, transcript_limit)?;
        endpoint.attached = true;
        endpoint.local = Some(Local {
            faults: config.faults,
            next: 0,
            ports: BTreeMap::new(),
            log: Vec::new(),
        });
        Ok(endpoint)
    }

    pub(super) fn advance_faults(&mut self, shared: &mut Shared, now: u64) -> Result<(), Failure> {
        loop {
            let fault = self
                .local
                .as_ref()
                .and_then(|local| local.faults.get(local.next))
                .filter(|fault| fault.at_nanos <= now)
                .cloned();
            let Some(fault) = fault else { return Ok(()) };
            shared
                .owner
                .advance_before(fault.at_nanos)
                .map_err(infrastructure)?;
            match fault.action.as_str() {
                "partition" => shared
                    .owner
                    .partition(&self.client.node, &self.server.node, true),
                "heal" => shared
                    .owner
                    .heal(&self.client.node, &self.server.node, true),
                "delay" => shared.owner.delay(
                    &self.client.node,
                    &self.server.node,
                    fault.delay_nanos,
                    true,
                ),
                "reset" => {
                    let local = self.local.as_ref().unwrap();
                    let mut result = Ok(());
                    for handle in self.handles.keys() {
                        if fault.port == 0
                            || local.ports.get(handle) == Some(&u64::from(fault.port))
                        {
                            match shared.owner.reset(*handle) {
                                Ok(()) => {}
                                Err(error) if error == "stale file handle" => {}
                                Err(error) => {
                                    result = Err(error);
                                    break;
                                }
                            }
                        }
                    }
                    result
                }
                _ => unreachable!("validated standalone fault action"),
            }
            .map_err(infrastructure)?;
            self.local.as_mut().unwrap().next += 1;
            let mut value = serde_json::json!({"name":fault.name,"action":fault.action,"at_nanos":fault.at_nanos});
            if fault.port != 0 {
                value["port"] = fault.port.into();
            }
            if fault.delay_nanos != 0 {
                value["delay_nanos"] = fault.delay_nanos.into();
            }
            let json = crate::linked_network::json_bytes(&serde_json::json!({"fault":value}))?;
            self.append(json, Vec::new())?;
        }
    }

    pub(crate) fn idle(
        &mut self,
        data: &mut [u8],
        now: &mut u64,
        advances: &mut u64,
    ) -> Result<(), Failure> {
        self.revision += 1;
        if data.len() != 16
            || u32::from_be_bytes(data[0..4].try_into().unwrap()) != 1
            || data[4..8] != [0; 4]
        {
            return Err(Failure::Invalid("runtime idle layout"));
        }
        let mut deadline = u64::from_be_bytes(data[8..16].try_into().unwrap());
        let shared = self.shared.clone();
        let mut shared = lock(&shared);
        self.advance(&mut shared, *now)?;
        let mut next = shared.owner.next_deadline();
        if let Some(fault) = self
            .local
            .as_ref()
            .and_then(|local| local.faults.get(local.next))
            && (next == 0 || fault.at_nanos < next)
        {
            next = fault.at_nanos;
        }
        if next != 0 && (deadline == 0 || next < deadline) {
            deadline = next;
        }
        if deadline > clock_ceiling(self.epoch)? {
            return Err(Failure::Invalid("network v2 idle deadline overflow"));
        }
        if deadline != 0 {
            let request_id = self
                .world
                .register(world::Request {
                    kind: "idle".into(),
                    resource: world::ResourceId {
                        adapter: "wasm".into(),
                        kind: "clock".into(),
                        key: now.to_string(),
                    },
                    priority: 0,
                    payload: Vec::new(),
                })
                .map_err(|error| infrastructure(error.to_string()))?;
            self.world
                .ready(world::Readiness {
                    request_id,
                    at: world::INITIAL_TIME + (*now).max(deadline).cast_signed(),
                    kind: "idle".into(),
                    payload: Vec::new(),
                    equivalence_class: String::new(),
                })
                .map_err(|error| infrastructure(error.to_string()))?;
        }
        let quiescence = self
            .world
            .quiesce()
            .map_err(|error| infrastructure(error.to_string()))?;
        let after = (quiescence.after - world::INITIAL_TIME).cast_unsigned();
        if after > *now {
            *advances += 1;
        }
        *now = (*now).max(after);
        self.advance(&mut shared, *now)?;
        stream_abi::set_u32(
            data,
            4,
            if deadline == 0 && quiescence.deliveries.is_empty() {
                2
            } else {
                1
            },
        );
        stream_abi::set_u64(data, 8, *now);
        let (json, binary) = evidence::quiescence_entry(&quiescence);
        self.append(json, binary)
    }

    pub(super) fn standalone_evidence(&self, shared: &Shared) -> Result<Vec<u8>, Failure> {
        let local = self.local.as_ref().unwrap();
        let record = shared.record().map_err(infrastructure)?;
        let operations = local
            .log
            .iter()
            .map(|line| {
                serde_json::from_slice::<serde_json::Value>(line)
                    .map_err(|error| infrastructure(error.to_string()))
            })
            .collect::<Result<Vec<_>, _>>()?;
        let value = serde_json::json!({"profile":PROFILE,"operations":operations,"native":record,"world":crate::linked_network::world_snapshot(&self.world)});
        let bytes = crate::linked_network::json_bytes(&value)?;
        if bytes.len() as u64 > self.transcript_limit {
            return Err(Failure::Capacity("network-v2-evidence-bytes"));
        }
        Ok(bytes)
    }
}
