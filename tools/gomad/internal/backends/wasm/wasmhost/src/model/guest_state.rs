use super::{CHUNK, Config, MAX_TIME, Model, StringDeclaration, inspection};
use crate::protocol;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::Deserialize;
#[cfg(test)]
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::io::BufRead;
use wasmtime::{Error, Result};

impl StringDeclaration {
    pub(crate) fn from_values(values: &[String]) -> Result<Self> {
        let mut packed = Vec::new();
        let mut offsets = Vec::new();
        for value in values {
            if value.contains('\0')
                || packed
                    .len()
                    .checked_add(value.len())
                    .and_then(|bytes| bytes.checked_add(1))
                    .is_none_or(|bytes| bytes > 4 * 1024 * 1024)
            {
                return Err(Error::msg("invalid local model string inputs"));
            }
            offsets.push(u32::try_from(packed.len()).map_err(Error::new)?);
            packed.extend_from_slice(value.as_bytes());
            packed.push(0);
        }
        Ok(Self {
            bytes: packed.len(),
            count: values.len(),
            chunks: packed.len().div_ceil(CHUNK),
            packed,
            offsets,
        })
    }
    fn validate(&self) -> bool {
        self.bytes <= 4 * 1024 * 1024
            && self.count <= self.bytes
            && self.chunks == self.bytes.div_ceil(CHUNK)
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Validated string declarations cap packed bytes and offsets at four MiB."
    )]
    fn read(&mut self, input: &mut dyn BufRead, stream: &str) -> Result<()> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Chunk {
            #[serde(rename = "type")]
            kind: String,
            stream: String,
            sequence: usize,
            data_base64: String,
        }
        self.packed.reserve(self.bytes);
        for sequence in 0..self.chunks {
            let chunk: Chunk =
                serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
            if chunk.kind != "model_input"
                || chunk.stream != stream
                || chunk.sequence != sequence
                || chunk.data_base64.len() > CHUNK.div_ceil(3) * 4
            {
                return Err(Error::msg("invalid local model input chunk"));
            }
            let bytes = STANDARD.decode(chunk.data_base64).map_err(Error::new)?;
            if bytes.len() != CHUNK.min(self.bytes - self.packed.len()) {
                return Err(Error::msg("invalid local model input chunk length"));
            }
            self.packed.extend(bytes);
        }
        if self.packed.len() != self.bytes || self.bytes != 0 && self.packed.last() != Some(&0) {
            return Err(Error::msg("invalid local model string framing"));
        }
        let mut offset = 0;
        for value in self.packed.split_inclusive(|byte| *byte == 0) {
            std::str::from_utf8(&value[..value.len() - 1]).map_err(Error::new)?;
            self.offsets.push(offset as u32);
            offset += value.len();
        }
        if self.offsets.len() != self.count {
            return Err(Error::msg("invalid local model string count"));
        }
        Ok(())
    }
}
impl Config {
    /// # Errors
    /// Refuses inconsistent model declarations, unsupported modes and invalid bounds.
    pub fn validate(&self) -> Result<()> {
        if !(1..=3).contains(&self.mode)
            || !self.arguments.validate()
            || !self.environment.validate()
            || self.working_directory_sha256.len() != 64
            || !self
                .working_directory_sha256
                .bytes()
                .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
            || !(160..=1 << 30).contains(&self.limit)
            || self.inspection.is_some() && self.mode != 2
            || self.epoch_nanos > MAX_TIME
            || self.now > MAX_TIME - self.epoch_nanos
            || self.read_step_nanos > 1_000_000_000
            || self.limits.calls == 0
            || self.limits.calls > 1 << 28
            || self.limits.transcript_bytes == 0
            || self.limits.transcript_bytes > 1 << 30
            || self.limits.pending_events == 0
            || self.limits.pending_events > 4096
            || self.diagnostics && 64 + (self.limit - 64) / 96 * 96 > 64 * 1024 * 1024
        {
            return Err(Error::msg("invalid local model configuration"));
        }
        match (&self.replay, self.mode) {
            (Some(_), 1) => {
                return Err(super::replay::failure(
                    "framing",
                    "record mode forbids a replay trace",
                ));
            }
            (None, 2 | 3) => {
                return Err(super::replay::failure(
                    "schema",
                    format!("missing replay schema; expected {}", super::replay::SCHEMA),
                ));
            }
            (Some(request), _) => request.validate(self.limit)?,
            _ => {}
        }
        Ok(())
    }
    /// # Errors
    /// Refuses malformed bounded argument/environment frames or their working-directory identity.
    pub fn read_inputs(&mut self, input: &mut dyn BufRead) -> Result<()> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct End {
            #[serde(rename = "type")]
            kind: String,
        }
        self.arguments.read(input, "arguments")?;
        self.environment.read(input, "environment")?;
        let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
        if end.kind != "model_inputs_end" {
            return Err(Error::msg("invalid local model input end"));
        }
        self.validate_environment()
    }

    pub(crate) fn set_inputs(
        &mut self,
        arguments: &[String],
        environment: &[String],
    ) -> Result<()> {
        self.arguments = StringDeclaration::from_values(arguments)?;
        self.environment = StringDeclaration::from_values(environment)?;
        self.validate()?;
        self.validate_environment()
    }

    fn validate_environment(&self) -> Result<()> {
        let mut names = std::collections::HashSet::new();
        let mut pwd = None;
        for entry in self.environment.packed.split_inclusive(|byte| *byte == 0) {
            let value = std::str::from_utf8(&entry[..entry.len() - 1]).map_err(Error::new)?;
            let (name, content) = value
                .split_once('=')
                .ok_or_else(|| Error::msg("invalid local model environment"))?;
            if name.is_empty() || !names.insert(name) {
                return Err(Error::msg("invalid local model environment names"));
            }
            if name == "PWD" {
                pwd = Some(content);
            }
        }
        let pwd = pwd.ok_or_else(|| Error::msg("missing local model PWD"))?;
        let clean = pwd == "/"
            || pwd.starts_with('/')
                && !pwd.ends_with('/')
                && pwd[1..]
                    .split('/')
                    .all(|part| !part.is_empty() && part != "." && part != "..");
        if !clean
            || format!("{:x}", Sha256::digest(pwd.as_bytes())) != self.working_directory_sha256
        {
            return Err(Error::msg("invalid local model working directory identity"));
        }
        Ok(())
    }
    /// # Errors
    /// Refuses replay framing or the helper-owned trace preflight.
    pub fn read_tape(&mut self, input: &mut dyn BufRead) -> Result<()> {
        if let Some(request) = &self.replay {
            let plan = super::replay::read(input, request, self.mode, self.limit, None)?;
            self.tape = Some(plan.decisions);
        }
        Ok(())
    }
}
impl Model {
    pub fn new(config: Config) -> Self {
        Self::construct(config, false)
    }

    pub fn new_linked(config: Config) -> Self {
        Self::construct(config, true)
    }

    fn construct(config: Config, linked: bool) -> Self {
        #[cfg(not(feature = "determinism-fixture"))]
        let _ = linked;
        // Split ownership confirms the broker-adopted clock before its first
        // local call, as the Go-owned path raises `now` at its first call.
        let sync_pending = config.split;
        let mut inspection = inspection::Transport::default();
        inspection.request.clone_from(&config.inspection);
        Self {
            linked_evidence: None,
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_decisions: 0,
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_owner: None,
            #[cfg(feature = "determinism-fixture")]
            fixture_fault: if linked {
                crate::determinism_fixture::Fault::none()
            } else {
                crate::determinism_fixture::Fault::from_environment()
            },
            now: config.now,
            config,
            advances: 0,
            transcript_bytes: 0,
            transcript_hash: Sha256::new(),
            transcript_encoder: crate::transcript::Encoder::default(),
            records: 0,
            branching_records: 0,
            choice_hash: Sha256::new(),
            cursor: 0,
            configured: false,
            finished: false,
            peak: 0,
            elapsed: 0,
            terminal: Vec::new(),
            pending: HashMap::new(),
            entropy_offset: 0,
            divergence: None,
            sync_pending,
            inspection,
        }
    }
    /// Whether the cooperative runtime is configured and not yet finished.
    pub fn runtime_active(&self) -> bool {
        self.configured && !self.finished
    }
    pub(crate) fn epoch_nanos(&self) -> u64 {
        self.config.epoch_nanos
    }
    pub(crate) fn transcript_limit(&self) -> u64 {
        self.config.limits.transcript_bytes
    }
    pub fn split(&self) -> bool {
        self.config.split
    }
    pub fn call_limit(&self) -> u64 {
        self.config.limits.calls
    }
    #[cfg(test)]
    pub fn state(&self) -> Value {
        json!({"now":self.now,"advances":self.advances,"transcript_bytes":self.transcript_bytes})
    }
}
