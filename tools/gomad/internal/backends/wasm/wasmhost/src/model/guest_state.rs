use super::{CHUNK, Config, Decision, MAX_TIME, Model, StringDeclaration, inspection};
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
            || self.mode == 1 && (self.tape.is_some() || self.tape_chunks != 0)
            || self.inspection.is_some() && self.mode != 2
            || self.tape_chunks > (self.limit - 64) / 96
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
        Ok(())
    }
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
    pub fn read_tape(&mut self, input: &mut dyn BufRead) -> Result<()> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Tape {
            #[serde(rename = "type")]
            kind: String,
            decisions: Vec<Decision>,
        }
        if self.mode != 1 && self.tape.is_none() {
            self.tape = Some(Vec::new());
        }
        for _ in 0..self.tape_chunks {
            let chunk: Tape =
                serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
            if chunk.kind != "tape" || chunk.decisions.is_empty() {
                return Err(Error::msg("invalid tape chunk"));
            }
            let decisions = self.tape.get_or_insert_with(Vec::new);
            if (decisions.len() + chunk.decisions.len()) as u64 > (self.limit - 64) / 96 {
                return Err(Error::msg("tape decision capacity"));
            }
            decisions.extend(chunk.decisions);
        }
        Ok(())
    }
}
impl Model {
    pub fn new(config: Config) -> Self {
        // Split ownership confirms the broker-adopted clock before its first
        // local call, as the Go-owned path raises `now` at its first call.
        let sync_pending = config.split;
        let mut inspection = inspection::Transport::default();
        inspection.request.clone_from(&config.inspection);
        Self {
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_decisions: 0,
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_owner: None,
            #[cfg(feature = "determinism-fixture")]
            fixture_fault: crate::determinism_fixture::Fault::from_environment(),
            now: config.now,
            config,
            advances: 0,
            transcript_bytes: 0,
            transcript_hash: Sha256::new(),
            transcript_encoder: crate::transcript::Encoder::default(),
            records: 0,
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
