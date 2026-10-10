use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{BufRead, Write};
use wasmtime::{Error, Result};

pub const SCHEMA: &str = "gomad3.wasm-host/v1";
pub const FRAME_LIMIT: usize = 8 * 1024 * 1024;
pub const BUFFER_LIMIT: usize = 4 * 1024 * 1024;

#[derive(Debug)]
pub struct FrameCapacity;
impl std::fmt::Display for FrameCapacity {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str("protocol frame exceeds 8 MiB")
    }
}
impl std::error::Error for FrameCapacity {}
pub fn classification(error: &Error) -> &'static str {
    if error.downcast_ref::<FrameCapacity>().is_some() {
        "capacity"
    } else if error.downcast_ref::<std::io::Error>().is_some() {
        "infrastructure"
    } else {
        "invalid"
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Execute {
    pub schema: String,
    #[serde(rename = "type")]
    pub kind: String,
    pub module_path: String,
    pub module_sha256: String,
    pub memory_bytes: u64,
    pub fuel: u64,
    pub output_bytes: u64,
}
impl Execute {
    pub fn validate(&self) -> Result<()> {
        if self.schema != SCHEMA || self.kind != "execute" {
            return Err(Error::msg("unknown execute schema or type"));
        }
        if !std::path::Path::new(&self.module_path).is_absolute()
            || self.module_sha256.len() != 64
            || !self
                .module_sha256
                .bytes()
                .all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c))
        {
            return Err(Error::msg("invalid module path or SHA-256"));
        }
        if self.memory_bytes == 0
            || self.memory_bytes > 1u64 << 32
            || !self.memory_bytes.is_multiple_of(65536)
            || self.fuel == 0
            || self.output_bytes == 0
            || self.output_bytes > 1 << 30
        {
            return Err(Error::msg("invalid execution limits"));
        }
        Ok(())
    }
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Authorization {
    #[serde(rename = "type")]
    pub kind: String,
    pub module_sha256: String,
    pub configuration: String,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Reply {
    #[serde(rename = "type")]
    pub kind: String,
    pub id: u64,
    pub errno: u16,
    pub output: Value,
    #[serde(default)]
    pub unsupported: bool,
    #[serde(default)]
    pub message: String,
}
#[derive(Serialize)]
pub struct Outcome {
    #[serde(rename = "type")]
    kind: &'static str,
    pub termination: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exit_code: Option<u32>,
    pub message: String,
    pub fuel_remaining: u64,
    pub peak_memory_bytes: usize,
    pub output_bytes: u64,
}
impl Outcome {
    pub fn new(termination: &'static str, message: impl Into<String>) -> Self {
        let mut message = message.into();
        if message.len() > 65536 {
            let mut limit = 65536 - " (message truncated)".len();
            while !message.is_char_boundary(limit) {
                limit -= 1;
            }
            message.truncate(limit);
            message.push_str(" (message truncated)");
        }
        Self {
            kind: "result",
            termination,
            exit_code: None,
            message,
            fuel_remaining: 0,
            peak_memory_bytes: 0,
            output_bytes: 0,
        }
    }
}
pub fn read_frame(input: &mut dyn BufRead) -> Result<Vec<u8>> {
    let mut bytes = Vec::new();
    loop {
        let available = input.fill_buf().map_err(Error::new)?;
        if available.is_empty() {
            return Err(Error::msg("unexpected protocol EOF"));
        }
        let count = available
            .iter()
            .position(|&c| c == b'\n')
            .map_or(available.len(), |n| n + 1);
        if bytes.len() + count > FRAME_LIMIT {
            return Err(Error::new(FrameCapacity));
        }
        let done = available[count - 1] == b'\n';
        bytes.extend_from_slice(&available[..count]);
        input.consume(count);
        if done {
            return Ok(bytes);
        }
    }
}
pub fn send_frame(output: &mut dyn Write, frame: &impl Serialize) -> Result<()> {
    let mut bytes = serde_json::to_vec(frame).map_err(|e| Error::msg(e.to_string()))?;
    bytes.push(b'\n');
    if bytes.len() > FRAME_LIMIT {
        return Err(Error::new(FrameCapacity));
    }
    output
        .write_all(&bytes)
        .and_then(|_| output.flush())
        .map_err(Error::new)
}
