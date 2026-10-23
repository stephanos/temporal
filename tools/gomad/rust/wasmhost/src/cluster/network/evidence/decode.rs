use crate::network::{Address, Domain, Limits, OperationResult, Request, history};
use crate::world;

pub(crate) fn owner_limits(limits: &super::super::Limits) -> Limits {
    Limits {
        listeners: limits.listeners,
        connections: limits.connections,
        bytes: limits.bytes,
        connection_bytes: limits.connection_bytes,
        stall_nanos: limits.stall_nanos,
        transitions: limits.transitions,
    }
}

pub(crate) struct Decoded {
    pub log: Vec<u8>,
    pub record: history::Record,
    pub world: Vec<u8>,
}

struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
}

impl<'a> Reader<'a> {
    fn raw(&mut self, count: usize) -> Result<&'a [u8], String> {
        let end = self.at.checked_add(count).ok_or("network wire overflow")?;
        let bytes = self
            .bytes
            .get(self.at..end)
            .ok_or("network wire truncated")?;
        self.at = end;
        Ok(bytes)
    }

    fn number(&mut self) -> Result<u64, String> {
        let mut value = 0;
        for shift in (0..70).step_by(7) {
            let byte = self.raw(1)?[0];
            if shift == 63 && byte > 1 {
                return Err("network wire varint overflow".into());
            }
            value |= u64::from(byte & 127) << shift;
            if byte & 128 == 0 {
                if shift != 0 && byte == 0 {
                    return Err("network wire non-minimal varint".into());
                }
                return Ok(value);
            }
        }
        Err("network wire varint overflow".into())
    }

    fn count(&mut self) -> Result<usize, String> {
        let count = usize::try_from(self.number()?).map_err(|error| error.to_string())?;
        if count > self.bytes.len() - self.at {
            return Err("network wire count exceeds remaining bytes".into());
        }
        Ok(count)
    }

    fn blob(&mut self) -> Result<Vec<u8>, String> {
        let count = self.count()?;
        Ok(self.raw(count)?.to_vec())
    }

    fn text(&mut self) -> Result<String, String> {
        String::from_utf8(self.blob()?).map_err(|error| error.to_string())
    }

    fn flag(&mut self) -> Result<bool, String> {
        match self.raw(1)?[0] {
            0 => Ok(false),
            1 => Ok(true),
            _ => Err("network wire flag".into()),
        }
    }

    fn optional<T>(
        &mut self,
        mut read: impl FnMut(&mut Self) -> Result<T, String>,
    ) -> Result<Option<Vec<T>>, String> {
        if !self.flag()? {
            return Ok(None);
        }
        let count = self.count()?;
        let mut values = Vec::new();
        for _ in 0..count {
            values.push(read(self)?);
        }
        Ok(Some(values))
    }

    fn endpoint(&mut self) -> Result<history::Endpoint, String> {
        Ok(history::Endpoint {
            node: self.text()?,
            incarnation: self.number()?,
            address: self.text()?,
            port: self.number()?,
        })
    }

    fn address(&mut self) -> Result<Address, String> {
        let ip = self.text()?;
        let port = self.number()?;
        i64::try_from(port).map_err(|error| error.to_string())?;
        Ok(Address { ip, port })
    }

    fn operation(&mut self) -> Result<Vec<u8>, String> {
        let at = self.number()?;
        let request = Request {
            domain: Domain {
                run: self.number()?,
                node: self.text()?,
                incarnation: self.number()?,
                address: self.text()?,
            },
            op: self.text()?,
            handle: self.number()?,
            host: self.text()?,
            port: i64::try_from(self.number()?).map_err(|error| error.to_string())?,
            bytes: self.blob()?,
            count: u32::try_from(self.number()?).map_err(|error| error.to_string())?,
            deadline: self.number()?,
            request_id: self.number()?,
        };
        let result = OperationResult {
            request_id: self.number()?,
            state: self.text()?,
            handle: self.number()?,
            local: self.address()?,
            remote: self.address()?,
            bytes: self.blob()?,
            count: self.number()?,
            deadline: self.number()?,
            error: self.text()?,
        };
        i64::try_from(result.count).map_err(|error| error.to_string())?;
        Ok(super::operation_entry(at, &request, &result).0)
    }

    fn quiescence(&mut self) -> Result<Vec<u8>, String> {
        let kind = match self.text()?.as_str() {
            "delivered" => world::QuiescenceKind::Delivered,
            "deadlock" => world::QuiescenceKind::Deadlock,
            "idle" => world::QuiescenceKind::Idle,
            _ => return Err("network wire quiescence kind".into()),
        };
        let before = self.number()?.cast_signed();
        let after = self.number()?.cast_signed();
        let mut deliveries = Vec::new();
        for _ in 0..self.count()? {
            deliveries.push(world::Delivery {
                request_id: self.number()?,
                event_id: self.number()?,
                at: self.number()?.cast_signed(),
                kind: self.text()?,
                payload: self.blob()?,
            });
        }
        let mut blocked = Vec::new();
        for _ in 0..self.count()? {
            blocked.push(self.number()?);
        }
        Ok(super::quiescence_entry(&world::Quiescence {
            kind,
            before,
            after,
            deliveries,
            blocked,
        })
        .0)
    }

    fn record(&mut self) -> Result<history::Record, String> {
        let transitions = self.optional(|reader| {
            Ok(history::Transition {
                ordinal: reader.number()?,
                kind: reader.text()?,
                source: reader.endpoint()?,
                destination: reader.endpoint()?,
                connection: reader.number()?,
                bytes: reader.number()?,
                count: reader.number()?,
                delay_nanos: reader.number()?,
                outcome: reader.text()?,
            })
        })?;
        let snapshot = history::Snapshot {
            nodes: self.optional(|reader| {
                Ok(history::NodeSnapshot {
                    node: reader.text()?,
                    address: reader.text()?,
                    last_incarnation: reader.number()?,
                    next_listener_port: reader.number()?,
                    next_client_port: reader.number()?,
                })
            })?,
            links: self.optional(|reader| {
                Ok(history::Link {
                    from: reader.text()?,
                    to: reader.text()?,
                    enabled: reader.flag()?,
                    delay_nanos: reader.number()?,
                })
            })?,
            listeners: self.optional(|reader| {
                Ok(history::ListenerSnapshot {
                    endpoint: reader.endpoint()?,
                    closed: reader.flag()?,
                })
            })?,
            connections: self.optional(|reader| {
                Ok(history::ConnectionSnapshot {
                    client_bytes: reader.number()?,
                    server_bytes: reader.number()?,
                    identity: reader.number()?,
                    client: reader.endpoint()?,
                    server: reader.endpoint()?,
                    closed: reader.flag()?,
                    reset: reader.flag()?,
                })
            })?,
            next_connection: self.number()?,
            transition_sha256: self.text()?,
            identity: self.text()?,
        };
        Ok(history::Record {
            transitions,
            snapshot,
        })
    }
}

pub(crate) fn decode(
    bytes: &[u8],
    transcript_limit: u64,
    limits: &Limits,
    world_config: &world::Config,
) -> Result<Decoded, String> {
    if bytes.len() as u64 > transcript_limit {
        return Err("network evidence byte bound".into());
    }
    let mut reader = Reader { bytes, at: 0 };
    if reader.raw(8)? != super::MAGIC
        || reader.number()? != super::VERSION
        || reader.text()? != super::super::PROFILE
    {
        return Err("network evidence header".into());
    }
    let declared = reader.number()?;
    let mut log = Vec::new();
    for _ in 0..reader.count()? {
        let entry = match reader.raw(1)?[0] {
            1 => reader.operation()?,
            3 => reader.quiescence()?,
            _ => return Err("network evidence operation tag".into()),
        };
        log.extend_from_slice(&entry);
        log.push(b'\n');
        if log.len() as u64 > transcript_limit {
            return Err("network evidence log bound".into());
        }
    }
    if log.len() as u64 != declared {
        return Err("network evidence log accounting".into());
    }
    let record = reader.record()?;
    if !reader.flag()? {
        return Err("network evidence missing World".into());
    }
    let world = reader.blob()?;
    if reader.at != bytes.len() {
        return Err("network evidence trailing bytes".into());
    }
    if record.transitions.as_deref().unwrap_or_default().len() as u64 > limits.transitions
        || record
            .snapshot
            .listeners
            .as_deref()
            .unwrap_or_default()
            .len() as u64
            > limits.listeners
        || record
            .snapshot
            .connections
            .as_deref()
            .unwrap_or_default()
            .len() as u64
            > limits.connections
        || history::transitions_digest(record.transitions.as_deref().unwrap_or_default())?
            != record.snapshot.transition_sha256
        || history::snapshot_digest(&record.snapshot)? != record.snapshot.identity
    {
        return Err("network evidence record identity or bound".into());
    }
    crate::network::Model::new(
        1,
        crate::network::Config {
            nodes: record
                .snapshot
                .nodes
                .as_deref()
                .unwrap_or_default()
                .iter()
                .map(|node| crate::network::Node {
                    node: node.node.clone(),
                    address: node.address.clone(),
                })
                .collect(),
            links: record.snapshot.links.clone().unwrap_or_default(),
            limits: limits.clone(),
            replay: Some(record.clone()),
        },
    )?;
    world::retained::validate_with_transition_limit(&world, world_config, None)?;
    Ok(Decoded { log, record, world })
}

#[cfg(test)]
mod tests {
    use super::*;
    use sha2::{Digest, Sha256};

    fn world_config() -> world::Config {
        world::Config {
            seed: 0,
            limits: world::Limits {
                max_requests: 16384,
                max_events: 16384,
                max_queued_events: 16384,
                max_transitions: 65536,
                max_payload_bytes: 64 << 20,
                max_string_bytes: 1024,
            },
        }
    }

    fn limits() -> super::super::super::Limits {
        super::super::super::Limits {
            listeners: 128,
            connections: 128,
            bytes: 8 << 20,
            connection_bytes: 4 << 20,
            stall_nanos: 10_000_000_000,
            transitions: 8192,
        }
    }

    #[test]
    fn cluster_network_admission_separates_shared_record_and_local_world_limits() {
        let bytes = include_bytes!("../../../../tests/testdata/cluster_network/b.gmnetev1");
        let local = crate::linked_cluster::NetworkModel {
            profile: super::super::super::PROFILE.into(),
            limits: limits(),
            faults: Vec::new(),
        };
        let mut shared = local.clone();
        shared.limits.transitions *= 2;
        shared.limits.bytes *= 2;
        shared.limits.listeners *= 2;
        shared.limits.connections *= 2;
        crate::linked_cluster::validate_network_evidence(bytes, 1 << 20, &shared, &local).unwrap();
        let mut too_small_shared = shared.clone();
        too_small_shared.limits.listeners = 0;
        let shared_error = crate::linked_cluster::validate_network_evidence(
            bytes,
            1 << 20,
            &too_small_shared,
            &local,
        )
        .unwrap_err();
        assert!(
            shared_error.contains("record identity or bound"),
            "{shared_error}"
        );
        let mut foreign_local = local.clone();
        foreign_local.limits.transitions += 1;
        let local_error = crate::linked_cluster::validate_network_evidence(
            bytes,
            1 << 20,
            &shared,
            &foreign_local,
        )
        .unwrap_err();
        assert!(
            local_error.contains("configuration mismatch"),
            "{local_error}"
        );
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({
                "schema":"gomad.core-predicate-observation/v1",
                "case":"network_shared_record_local_world_limits",
                "inputs":{"evidence_sha256":format!("{:x}",Sha256::digest(bytes)),"shared":shared,"local":local,"small_shared":too_small_shared,"foreign_local":foreign_local},
                "observed":{"shared_error":shared_error,"local_error":local_error,"admitted_world_sha256":format!("{:x}",Sha256::digest(decode(bytes,1 << 20,&owner_limits(&shared.limits),&world_config()).unwrap().world))}
            })
        );
    }

    #[test]
    fn cluster_network_admission_rejects_resealed_semantically_invalid_records() {
        let bytes = include_bytes!("../../../../tests/testdata/cluster_network/b.gmnetev1");
        let local = crate::linked_cluster::NetworkModel {
            profile: super::super::super::PROFILE.into(),
            limits: limits(),
            faults: Vec::new(),
        };
        let valid = decode(
            bytes,
            1 << 20,
            &owner_limits(&local.limits),
            &world_config(),
        )
        .unwrap();
        let mut reader = Reader { bytes, at: 8 };
        reader.number().unwrap();
        reader.text().unwrap();
        let declared_log_bytes = reader.number().unwrap();
        let count = reader.count().unwrap();
        let start = reader.at;
        for _ in 0..count {
            match reader.raw(1).unwrap()[0] {
                1 => {
                    reader.operation().unwrap();
                }
                3 => {
                    reader.quiescence().unwrap();
                }
                _ => unreachable!(),
            }
        }
        let entries = &bytes[start..reader.at];
        for mutation in [
            "unknown_transition_kind",
            "ordinal_gap",
            "listener_port_zero",
            "reset_open_connection",
            "connection_byte_bound",
            "total_byte_bound",
        ] {
            let mut shared = local.clone();
            let mut record = valid.record.clone();
            match mutation {
                "unknown_transition_kind" => {
                    record.transitions.as_mut().unwrap()[0].kind = "unknown".into()
                }
                "ordinal_gap" => record.transitions.as_mut().unwrap()[0].ordinal += 1,
                "listener_port_zero" => {
                    record.snapshot.listeners.as_mut().unwrap()[0].endpoint.port = 0
                }
                "reset_open_connection" => {
                    let connection = &mut record.snapshot.connections.as_mut().unwrap()[0];
                    connection.reset = true;
                    connection.closed = false;
                }
                "connection_byte_bound" => {
                    record.snapshot.connections.as_mut().unwrap()[0].client_bytes =
                        shared.limits.connection_bytes + 1
                }
                "total_byte_bound" => {
                    shared.limits.bytes = 1;
                    record.snapshot.connections.as_mut().unwrap()[0].client_bytes = 2;
                }
                _ => unreachable!(),
            }
            record.snapshot.transition_sha256 =
                history::transitions_digest(record.transitions.as_deref().unwrap()).unwrap();
            record.snapshot.identity = history::snapshot_digest(&record.snapshot).unwrap();
            let changed = super::super::document(
                declared_log_bytes,
                count as u64,
                entries,
                &record,
                &valid.world,
            );
            let error = crate::linked_cluster::validate_network_evidence(
                &changed,
                1 << 20,
                &shared,
                &local,
            )
            .unwrap_err();
            assert!(
                !error.contains("identity or bound"),
                "reseal was not authenticated: {mutation}: {error}"
            );
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                serde_json::json!({
                    "schema":"gomad.core-predicate-observation/v1","case":"network_resealed_semantic_refusal",
                    "inputs":{"mutation":mutation,"evidence_sha256":format!("{:x}",Sha256::digest(&changed)),"record":record,"shared":shared,"local":local},
                    "observed":{"error":error}
                })
            );
        }
    }

    #[test]
    fn cluster_network_decoder_refuses_malformed_evidence() {
        let bytes = include_bytes!("../../../../tests/testdata/cluster_network/b.gmnetev1");
        let config = world_config();
        let limits = limits();
        let model = crate::linked_cluster::NetworkModel {
            profile: super::super::super::PROFILE.into(),
            limits: limits.clone(),
            faults: Vec::new(),
        };
        crate::linked_cluster::validate_network_evidence(bytes, 1 << 20, &model, &model).unwrap();
        let valid = decode(bytes, 1 << 20, &owner_limits(&limits), &config).unwrap();
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"cluster_network_decoder_valid_fixture","inputs":{"path":"tests/testdata/cluster_network/b.gmnetev1","sha256":format!("{:x}",Sha256::digest(bytes))},"observed":{"record":valid.record,"log_sha256":format!("{:x}",Sha256::digest(&valid.log)),"world_sha256":format!("{:x}",Sha256::digest(&valid.world))}})
        );
        let refused = |name: &str, changed: &[u8], bound| {
            let error =
                crate::linked_cluster::validate_network_evidence(changed, bound, &model, &model)
                    .err()
                    .expect(name);
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"cluster_network_decoder_refusal","inputs":{"mutation":name,"bytes":changed.len(),"sha256":format!("{:x}",Sha256::digest(changed)),"transcript_limit":bound},"observed":{"error":error}})
            );
        };
        let mut prefix_errors = sha2::Sha256::new();
        for length in 0..bytes.len() {
            let error = crate::linked_cluster::validate_network_evidence(
                &bytes[..length],
                1 << 20,
                &model,
                &model,
            )
            .err()
            .unwrap_or_else(|| panic!("prefix {length} admitted"));
            prefix_errors.update((length as u64).to_be_bytes());
            prefix_errors.update(error.as_bytes());
            prefix_errors.update([0]);
        }
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"cluster_network_decoder_all_prefixes","inputs":{"path":"tests/testdata/cluster_network/b.gmnetev1","sha256":format!("{:x}",Sha256::digest(bytes)),"prefix_end_exclusive":bytes.len()},"observed":{"ordered_error_sha256":format!("{:x}",prefix_errors.finalize())}})
        );
        let mut trailing = bytes.to_vec();
        trailing.push(0);
        refused("trailing", &trailing, 1 << 20);
        refused("oversize", bytes, bytes.len() as u64 - 1);
        let mut mutations = vec![
            ("magic", 0, b'X'),
            ("version", 8, 2),
            ("invalid_utf8", 10, 255),
        ];
        let profile = 10 + super::super::super::PROFILE.len();
        let tag = profile + 3;
        assert_eq!(bytes[tag], 1);
        for value in [0, 2, 4, 255] {
            mutations.push(("unknown_tag", tag, value));
        }
        mutations.push(("log_accounting", profile, bytes[profile] + 1));
        let world = bytes
            .windows(8)
            .position(|part| part == b"GMWSNAP1")
            .unwrap();
        let mut flag = world - 1;
        while bytes[flag] != 1 {
            flag -= 1;
        }
        mutations.push(("flag", flag, 2));
        mutations.push(("world_wire", world + 20, bytes[world + 20] ^ 1));
        let digest = bytes
            .windows(7)
            .rposition(|part| part == b"sha256:")
            .unwrap();
        mutations.push(("record_identity", digest + 7, bytes[digest + 7] ^ 1));
        for (name, at, value) in mutations {
            let mut changed = bytes.to_vec();
            changed[at] = value;
            refused(name, &changed, 1 << 20);
        }
        let mut changed = bytes[..profile].to_vec();
        changed.extend_from_slice(&[bytes[profile] | 128, bytes[profile + 1] | 128, 0]);
        changed.extend_from_slice(&bytes[profile + 2..]);
        refused("non_minimal_varint", &changed, 1 << 20);
    }
}
