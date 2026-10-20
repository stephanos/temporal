use super::{canonical, identity::Identity};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Limits {
    pub fuel: u64,
    pub memory_bytes: u64,
    pub wall_millis: u64,
    pub output_bytes: u64,
    pub evidence_bytes: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Outcome {
    pub class: String,
    pub termination: String,
    pub exit_code: Option<u32>,
    pub reason: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Stream {
    pub bytes: u64,
    pub sha256: String,
    pub records: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Record {
    pub schema: String,
    pub identity: Identity,
    pub execution_identity: String,
    pub profile: String,
    pub seed: u64,
    pub arguments: Vec<String>,
    pub environment: BTreeMap<String, String>,
    pub limits: Limits,
    pub virtual_time_nanos: u64,
    pub outcome: Outcome,
    pub streams: BTreeMap<String, Stream>,
}

impl Record {
    /// # Errors
    /// Refuses unsupported schemas, identity mismatches and invalid stream digests.
    pub fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.execution-record/v2" {
            return Err("unsupported execution record schema".into());
        }
        if self.execution_identity != self.identity.digest()? {
            return Err("execution identity mismatch".into());
        }
        for stream in self.streams.values() {
            if !canonical::valid_digest(&stream.sha256) {
                return Err("invalid stream digest".into());
            }
        }
        Ok(())
    }

    /// # Errors
    /// Refuses invalid records or a serialization failure.
    pub fn bytes(&self) -> Result<Vec<u8>, String> {
        self.validate()?;
        canonical::encode(self)
    }

    /// # Errors
    /// Refuses records that cannot be canonically encoded.
    pub fn hash(&self) -> Result<String, String> {
        Ok(canonical::digest(&self.bytes()?))
    }

    /// # Errors
    /// Refuses records that cannot be canonically encoded after excluding the seed.
    pub fn failure_signature(&self) -> Result<String, String> {
        let mut behavior = self.clone();
        behavior.seed = 0;
        behavior.hash()
    }

    /// # Errors
    /// Refuses malformed, noncanonical or invalid records.
    pub fn decode(bytes: &[u8]) -> Result<Self, String> {
        let record: Self = serde_json::from_slice(bytes).map_err(|error| error.to_string())?;
        if record.bytes()? != bytes {
            return Err("record is not canonical".into());
        }
        Ok(record)
    }
}

#[cfg(test)]
mod tests {
    // A codec round trip protects integer width and canonical key ordering, which CLI views cannot distinguish after JSON parsing.
    #[test]
    fn native_integer_round_trip() {
        let encoded =
            super::canonical::encode(&std::collections::BTreeMap::from([("value", u64::MAX)]))
                .unwrap();
        assert_eq!(encoded, br#"{"value":18446744073709551615}"#);
        let decoded: std::collections::BTreeMap<String, u64> =
            serde_json::from_slice(&encoded).unwrap();
        assert_eq!(decoded["value"], u64::MAX);
    }
}
