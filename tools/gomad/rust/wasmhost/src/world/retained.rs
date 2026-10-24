use super::{Config, Model, Readiness, Request, ResourceId, wire_generated};

const MAXIMUM_BYTES: usize = 384 << 20;

struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
    config: &'a Config,
}

impl<'a> Reader<'a> {
    fn take(&mut self, size: usize) -> Result<&'a [u8], String> {
        let end = self.at.checked_add(size).ok_or("World wire overflow")?;
        let value = self.bytes.get(self.at..end).ok_or("World wire truncated")?;
        self.at = end;
        Ok(value)
    }

    fn number(&mut self, size: usize) -> Result<u64, String> {
        let mut value = [0; 8];
        value[8 - size..].copy_from_slice(self.take(size)?);
        Ok(u64::from_be_bytes(value))
    }

    fn count(&mut self, maximum: u64) -> Result<usize, String> {
        let count = self.number(8)?;
        if count > maximum || count > (self.bytes.len() - self.at) as u64 {
            return Err("World wire count exceeds bound".into());
        }
        usize::try_from(count).map_err(|error| error.to_string())
    }

    fn text(&mut self) -> Result<String, String> {
        self.text_with_limit(u64::from(self.config.limits.max_string_bytes))
    }

    fn text_with_limit(&mut self, maximum: u64) -> Result<String, String> {
        let count = self.number(4)?;
        if count > maximum {
            return Err("World wire string exceeds bound".into());
        }
        let count = usize::try_from(count).map_err(|error| error.to_string())?;
        std::str::from_utf8(self.take(count)?)
            .map(str::to_owned)
            .map_err(|error| error.to_string())
    }

    fn owner_tag(&mut self, choices: &[&str]) -> Result<(), String> {
        let maximum = choices
            .iter()
            .map(|choice| choice.len() as u64)
            .max()
            .unwrap_or(0);
        let tag = self.text_with_limit(maximum)?;
        if !choices.contains(&tag.as_str()) {
            return Err("World owner tag".into());
        }
        Ok(())
    }

    fn payload(&mut self) -> Result<Vec<u8>, String> {
        let count = self.count(self.config.limits.max_payload_bytes)?;
        Ok(self.take(count)?.to_vec())
    }

    fn request(&mut self) -> Result<Request, String> {
        Ok(Request {
            kind: self.text()?,
            resource: ResourceId {
                adapter: self.text()?,
                kind: self.text()?,
                key: self.text()?,
            },
            priority: u16::try_from(self.number(2)?).map_err(|error| error.to_string())?,
            payload: self.payload()?,
        })
    }

    fn readiness(&mut self) -> Result<Readiness, String> {
        Ok(Readiness {
            request_id: self.number(8)?,
            at: i64::try_from(self.number(8)?).map_err(|error| error.to_string())?,
            kind: self.text()?,
            payload: self.payload()?,
            equivalence_class: self.text()?,
        })
    }

    fn quiescence(&mut self) -> Result<(), String> {
        self.owner_tag(&["delivered", "deadlock", "idle"])?;
        self.take(16)?;
        for _ in 0..self.count(self.config.limits.max_events)? {
            self.take(24)?;
            self.text()?;
            self.payload()?;
        }
        let blocked = self.count(self.config.limits.max_requests)?;
        self.take(
            blocked
                .checked_mul(8)
                .ok_or("World blocked count overflow")?,
        )?;
        Ok(())
    }
}

/// Validates caller-owned expanded World evidence by replaying only its model transitions.
/// No guest code, engine, filesystem or host callback participates in admission.
#[cfg(test)]
pub(crate) fn validate(bytes: &[u8], config: &Config) -> Result<(), String> {
    validate_with_transition_limit(bytes, config, None)
}

pub(crate) fn validate_with_transition_limit(
    bytes: &[u8],
    config: &Config,
    limit: Option<u64>,
) -> Result<(), String> {
    admit(bytes, config, limit).map(|_| ())
}

pub(crate) fn admit(bytes: &[u8], config: &Config, limit: Option<u64>) -> Result<Model, String> {
    if bytes.is_empty() || bytes.len() > MAXIMUM_BYTES {
        return Err("World expanded byte bound".into());
    }
    let mut input = Reader {
        bytes,
        at: 0,
        config,
    };
    if input.take(8)? != wire_generated::MAGIC
        || input.number(4)? != u64::from(super::SCHEMA_VERSION)
    {
        return Err("World wire header".into());
    }
    for expected in [
        config.seed,
        config.limits.max_requests,
        config.limits.max_events,
        config.limits.max_queued_events,
        config.limits.max_transitions,
        config.limits.max_payload_bytes,
    ] {
        if input.number(8)? != expected {
            return Err("World wire configuration mismatch".into());
        }
    }
    if input.number(4)? != u64::from(config.limits.max_string_bytes) {
        return Err("World wire string configuration mismatch".into());
    }
    input.take(40)?;
    if input.number(8)? != 0 || input.number(8)? != 0 {
        return Err("World snapshot carries unexpected replay progress".into());
    }
    input.take(64)?;
    for _ in 0..input.count(config.limits.max_requests)? {
        input.take(8)?;
        input.request()?;
        input.owner_tag(&["pending", "queued", "canceled", "delivered"])?;
        input.take(8)?;
    }
    for _ in 0..input.count(config.limits.max_events)? {
        input.take(8)?;
        input.readiness()?;
        input.owner_tag(&["queued", "canceled", "delivered"])?;
    }
    let mut model = Model::new(config.clone()).map_err(|error| error.to_string())?;
    model
        .limit_transition_bytes(limit)
        .map_err(|error| error.to_string())?;
    for _ in 0..input.count(config.limits.max_transitions)? {
        input.take(72)?;
        match input.number(1)? {
            1 => {
                let request = input.request()?;
                input.take(8)?;
                model.register(request).map_err(|error| error.to_string())?;
            }
            2 => {
                let readiness = input.readiness()?;
                input.take(8)?;
                model.ready(readiness).map_err(|error| error.to_string())?;
            }
            3 => {
                let request = input.number(8)?;
                input.take(16)?;
                input.owner_tag(&["won", "already-canceled", "already-delivered"])?;
                model.cancel(request).map_err(|error| error.to_string())?;
            }
            4 => {
                input.quiescence()?;
                model.quiesce().map_err(|error| error.to_string())?;
            }
            _ => return Err("World transition kind".into()),
        }
    }
    if input.at != bytes.len() {
        return Err("World wire trailing bytes".into());
    }
    let mut actual = Vec::new();
    wire_generated::write_snapshot(&model, &mut actual).map_err(|error| error.to_string())?;
    if actual != bytes {
        return Err("World snapshot differs from replayed canonical state".into());
    }
    Ok(model)
}

pub(crate) const COMPRESSED: &str = "gomad/world/compressed-snapshot/v2";

pub(crate) fn compress(raw: &[u8]) -> Result<Vec<u8>, String> {
    use sha2::{Digest, Sha256};
    if raw.is_empty() || raw.len() > MAXIMUM_BYTES {
        return Err("World expanded byte bound".into());
    }
    let gzip = super::gzip::encode(raw, 64 << 20)?;
    let mut envelope = b"GMWORLD2".to_vec();
    envelope.extend_from_slice(&64u32.to_be_bytes());
    envelope.extend_from_slice(&0u32.to_be_bytes());
    envelope.extend_from_slice(&(raw.len() as u64).to_be_bytes());
    envelope.extend_from_slice(&(gzip.len() as u64).to_be_bytes());
    envelope.extend_from_slice(&Sha256::digest(raw));
    envelope.extend_from_slice(&gzip);
    Ok(envelope)
}

pub(crate) fn expand(bytes: &[u8]) -> Result<Vec<u8>, String> {
    use sha2::{Digest, Sha256};
    let config = Config {
        seed: 0,
        limits: super::Limits {
            max_requests: 0,
            max_events: 0,
            max_queued_events: 0,
            max_transitions: 0,
            max_payload_bytes: 0,
            max_string_bytes: 0,
        },
    };
    let mut reader = Reader {
        bytes,
        at: 0,
        config: &config,
    };
    if reader.take(8)? != b"GMWORLD2" || reader.number(4)? != 64 || reader.number(4)? != 0 {
        return Err("World compressed envelope header".into());
    }
    let expanded = usize::try_from(reader.number(8)?).map_err(|error| error.to_string())?;
    let compressed = usize::try_from(reader.number(8)?).map_err(|error| error.to_string())?;
    if expanded == 0
        || expanded > MAXIMUM_BYTES
        || compressed > 64 << 20
        || compressed != bytes.len().saturating_sub(64)
    {
        return Err("World compressed envelope bounds".into());
    }
    let digest = reader.take(32)?;
    let (raw, consumed) = super::gzip::decode(&bytes[64..], expanded)?;
    if consumed != compressed || raw.len() != expanded || Sha256::digest(&raw).as_slice() != digest
    {
        return Err("World compressed envelope digest or expanded identity".into());
    }
    Ok(raw)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::world::Limits;
    use sha2::Digest as _;

    #[test]
    fn fixed_owner_tags_remain_valid_below_application_string_capacity() {
        let config = Config {
            seed: 7,
            limits: Limits {
                max_requests: 4,
                max_events: 4,
                max_queued_events: 4,
                max_transitions: 20,
                max_payload_bytes: 8,
                max_string_bytes: 1,
            },
        };
        let mut model = Model::new(config.clone()).unwrap();
        let request = Request {
            kind: "k".into(),
            resource: ResourceId {
                adapter: "a".into(),
                kind: "r".into(),
                key: "k".into(),
            },
            priority: 0,
            payload: Vec::new(),
        };
        let canceled = model.register(request.clone()).unwrap();
        model.cancel(canceled).unwrap();
        model.cancel(canceled).unwrap();
        let delivered = model.register(request.clone()).unwrap();
        model
            .ready(Readiness {
                request_id: delivered,
                at: super::super::INITIAL_TIME + 1,
                kind: "k".into(),
                payload: Vec::new(),
                equivalence_class: String::new(),
            })
            .unwrap();
        model.quiesce().unwrap();
        model.cancel(delivered).unwrap();
        let mut raw = Vec::new();
        wire_generated::write_snapshot(&model, &mut raw).unwrap();
        validate(&raw, &config).unwrap();
        let mut invalid_request = request;
        invalid_request.kind = "kk".into();
        let application_error = model.register(invalid_request).unwrap_err().to_string();
        let at = raw
            .windows(b"already-canceled".len())
            .position(|part| part == b"already-canceled")
            .unwrap();
        let mut forged = raw.clone();
        forged[at] = b'x';
        assert_eq!(validate(&forged, &config).unwrap_err(), "World owner tag");
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"world.fixed-owner-tags","inputs":{"configuration":{"seed":7,"max_string_bytes":1},"wire_sha256":super::super::digest::hex(&sha2::Sha256::digest(&raw).into())},"observed":{"forged_tag_error":validate(&forged,&config).unwrap_err(),"application_kind":"kk","application_error":application_error}})
        );
    }

    #[test]
    fn detached_world_replays_before_admitting_header_digest_or_trailing_bytes() {
        let config = Config {
            seed: 7,
            limits: Limits {
                max_requests: 100,
                max_events: 100,
                max_queued_events: 100,
                max_transitions: 1000,
                max_payload_bytes: 4096,
                max_string_bytes: 1024,
            },
        };
        let mut model = Model::new(config.clone()).unwrap();
        let request = model
            .register(Request {
                kind: "sleep".into(),
                resource: ResourceId {
                    adapter: "clock".into(),
                    kind: "timer".into(),
                    key: "a".into(),
                },
                priority: 0,
                payload: vec![1],
            })
            .unwrap();
        model
            .ready(Readiness {
                request_id: request,
                at: super::super::INITIAL_TIME + 1,
                kind: "timer".into(),
                payload: vec![2],
                equivalence_class: String::new(),
            })
            .unwrap();
        model.quiesce().unwrap();
        let mut raw = Vec::new();
        wire_generated::write_snapshot(&model, &mut raw).unwrap();
        validate(&raw, &config).unwrap();
        let envelope = compress(&raw).unwrap();
        assert_eq!(expand(&envelope).unwrap(), raw);
        for offset in [0, 8, 12, 16, 24, 32, envelope.len() - 1] {
            let mut changed = envelope.clone();
            changed[offset] ^= 1;
            assert!(expand(&changed).is_err(), "compressed offset {offset}");
        }
        let mut trailing_envelope = envelope;
        trailing_envelope.push(0);
        assert!(expand(&trailing_envelope).is_err());
        for offset in [0, 32, 120, raw.len() - 1] {
            let mut changed = raw.clone();
            changed[offset] ^= 1;
            assert!(validate(&changed, &config).is_err(), "offset {offset}");
        }
        let mut trailing = raw.clone();
        trailing.push(0);
        assert!(validate(&trailing, &config).is_err());
        for end in 0..raw.len() {
            assert!(validate(&raw[..end], &config).is_err(), "prefix {end}");
        }
    }
}
