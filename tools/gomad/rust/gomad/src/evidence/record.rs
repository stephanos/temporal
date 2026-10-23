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

impl Outcome {
    fn validate(&self) -> Result<(), String> {
        if !matches!(
            self.class.as_str(),
            "success" | "target_failure" | "runner_failure" | "watchdog" | "prepared"
        ) || !matches!(
            self.termination.as_str(),
            "exit"
                | "trap"
                | "capacity"
                | "invalid"
                | "unsupported"
                | "infrastructure"
                | "watchdog_timeout"
                | "cancelled"
                | "storage_crash"
                | "scenario_failure"
                | "divergence"
                | "diverged"
                | "debug_error"
                | "instrumentation_divergence"
                | "prepared"
        ) {
            return Err("invalid record outcome class or termination".into());
        }
        if (self.termination == "exit") != self.exit_code.is_some() {
            return Err("invalid record outcome exit status".into());
        }
        let consistent = match self.class.as_str() {
            "success" => self.termination == "exit" && self.exit_code == Some(0),
            "prepared" => self.termination == "prepared",
            "watchdog" => self.termination == "watchdog_timeout",
            _ => self.termination != "prepared",
        };
        if !consistent {
            return Err("invalid record outcome class and termination combination".into());
        }
        Ok(())
    }
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
    /// Refuses unsupported schemas, identity mismatches, inconsistent outcomes and invalid stream digests.
    pub fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.execution-record/v2" {
            return Err("unsupported execution record schema".into());
        }
        if self.execution_identity != self.identity.digest()? {
            return Err("execution identity mismatch".into());
        }
        self.outcome.validate()?;
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
    use super::Outcome;

    fn outcome(class: &str, termination: &str, exit_code: Option<u32>) -> Outcome {
        Outcome {
            class: class.into(),
            termination: termination.into(),
            exit_code,
            reason: String::new(),
        }
    }

    #[test]
    fn outcomes_preserve_actual_publication_class_projections_and_preparation() {
        for (class, termination, exit_code) in [
            ("success", "exit", Some(0)),
            ("target_failure", "exit", Some(0)),
            ("target_failure", "exit", Some(7)),
            ("runner_failure", "exit", Some(7)),
            ("target_failure", "exit", Some(u32::MAX)),
            ("prepared", "prepared", None),
            ("watchdog", "watchdog_timeout", None),
        ] {
            outcome(class, termination, exit_code).validate().unwrap();
        }
        for termination in [
            "trap",
            "capacity",
            "invalid",
            "unsupported",
            "infrastructure",
            "watchdog_timeout",
            "cancelled",
            "storage_crash",
            "scenario_failure",
            "divergence",
            "diverged",
            "debug_error",
            "instrumentation_divergence",
        ] {
            for class in ["target_failure", "runner_failure"] {
                outcome(class, termination, None).validate().unwrap();
            }
        }
    }

    #[test]
    fn outcomes_refuse_unknown_root_kinds_and_contradictory_exit_or_class_claims() {
        for termination in [
            "future_reply",
            "exited",
            "crash",
            "stop",
            "inspected_prefix",
            "inspection_aborted",
            "probe_fuel_limit",
            "oracle_failed",
        ] {
            assert!(
                outcome("runner_failure", termination, None)
                    .validate()
                    .is_err(),
                "{termination}"
            );
        }
        for (class, termination, exit_code) in [
            ("future_class", "exit", Some(0)),
            ("success", "exit", Some(7)),
            ("success", "trap", None),
            ("success", "prepared", None),
            ("success", "exit", None),
            ("target_failure", "exit", None),
            ("runner_failure", "exit", None),
            ("target_failure", "trap", Some(0)),
            ("runner_failure", "capacity", Some(7)),
            ("prepared", "prepared", Some(0)),
            ("prepared", "exit", Some(0)),
            ("prepared", "trap", None),
            ("target_failure", "prepared", None),
            ("runner_failure", "prepared", None),
            ("watchdog", "exit", Some(0)),
            ("watchdog", "trap", None),
        ] {
            assert!(
                outcome(class, termination, exit_code).validate().is_err(),
                "{class}/{termination}/{exit_code:?}"
            );
        }
    }

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
