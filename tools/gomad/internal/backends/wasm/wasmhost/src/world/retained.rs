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
        let count = self.number(4)?;
        if count > u64::from(self.config.limits.max_string_bytes) {
            return Err("World wire string exceeds bound".into());
        }
        let count = usize::try_from(count).map_err(|error| error.to_string())?;
        std::str::from_utf8(self.take(count)?)
            .map(str::to_owned)
            .map_err(|error| error.to_string())
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
        self.text()?;
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
pub(crate) fn validate(bytes: &[u8], config: &Config) -> Result<(), String> {
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
        input.text()?;
        input.take(8)?;
    }
    for _ in 0..input.count(config.limits.max_events)? {
        input.take(8)?;
        input.readiness()?;
        input.text()?;
    }
    let mut model = Model::new(config.clone()).map_err(|error| error.to_string())?;
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
                input.text()?;
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
    Ok(())
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
