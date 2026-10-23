use serde::{Deserialize, Serialize, ser::SerializeStruct};
use serde_json::Value;
use std::io::{BufRead, Write};
use wasmtime::{Error, Result};

pub const SCHEMA: &str = "gomad.wasm-host/v1";
pub const FRAME_LIMIT: usize = 8 * 1024 * 1024;
pub const BUFFER_LIMIT: usize = 4 * 1024 * 1024;

#[derive(Default)]
pub struct WriteBuffers {
    pub bytes: Vec<u8>,
    pub encoded: String,
}
impl WriteBuffers {
    pub fn begin(&mut self, length: usize) -> Result<()> {
        self.clear();
        if length > BUFFER_LIMIT {
            return Err(Error::msg("iovec buffer capacity"));
        }
        self.bytes
            .try_reserve_exact(length)
            .map_err(|_| Error::msg("iovec buffer capacity"))?;
        if self.bytes.capacity() > BUFFER_LIMIT {
            return Err(Error::msg("iovec buffer capacity"));
        }
        Ok(())
    }
    pub fn encode(&mut self) -> Result<()> {
        use base64::Engine as _;
        self.encoded.clear();
        if self.bytes.len() > BUFFER_LIMIT || self.bytes.capacity() > BUFFER_LIMIT {
            return Err(Error::msg("iovec buffer capacity"));
        }
        let length = self.bytes.len().div_ceil(3) * 4;
        self.encoded
            .try_reserve_exact(length)
            .map_err(|_| Error::msg("iovec base64 capacity"))?;
        if self.encoded.capacity() > BUFFER_LIMIT.div_ceil(3) * 4 {
            return Err(Error::msg("iovec base64 capacity"));
        }
        base64::engine::general_purpose::STANDARD.encode_string(&self.bytes, &mut self.encoded);
        if self.encoded.capacity() > BUFFER_LIMIT.div_ceil(3) * 4 {
            return Err(Error::msg("iovec base64 capacity"));
        }
        Ok(())
    }
    pub fn clear(&mut self) {
        self.bytes.clear();
        self.encoded.clear();
        if self.bytes.capacity() > BUFFER_LIMIT {
            self.bytes = Vec::new();
        }
        if self.encoded.capacity() > BUFFER_LIMIT.div_ceil(3) * 4 {
            self.encoded = String::new();
        }
    }
}

pub enum CallbackInput<'a> {
    Typed {
        operation: crate::abi::Operation,
        input: &'a Value,
    },
    #[cfg(test)]
    Json {
        op: &'a str,
        input: &'a Value,
    },
    Write {
        fd: u32,
        data_base64: &'a str,
    },
    PWrite {
        fd: u32,
        data_base64: &'a str,
        offset: u64,
    },
}
impl CallbackInput<'_> {
    #[cfg(test)]
    pub fn operation(&self) -> Option<crate::abi::Operation> {
        use crate::abi::Operation;
        match self {
            Self::Typed { operation, .. } => Some(*operation),
            Self::Write { .. } => Some(Operation::FdWrite),
            Self::PWrite { .. } => Some(Operation::FdPwrite),
            #[cfg(test)]
            Self::Json { op, .. } => {
                if let Some(name) = op.strip_prefix("runtime_") {
                    Operation::resolve(
                        if name == "net_stream" {
                            "gomad_wasm_v2"
                        } else {
                            "gomad_wasm_v1"
                        },
                        name,
                    )
                } else {
                    Operation::resolve("wasi_snapshot_preview1", op)
                }
            }
        }
    }
    pub fn op(&self) -> &str {
        match self {
            Self::Typed { operation, .. } => operation.callback(),
            #[cfg(test)]
            Self::Json { op, .. } => op,
            Self::Write { .. } => "fd_write",
            Self::PWrite { .. } => "fd_pwrite",
        }
    }
    #[cfg(test)]
    pub fn json(&self) -> Option<&Value> {
        match self {
            Self::Typed { input, .. } => Some(input),
            #[cfg(test)]
            Self::Json { input, .. } => Some(input),
            Self::Write { .. } | Self::PWrite { .. } => None,
        }
    }
}
impl Serialize for CallbackInput<'_> {
    fn serialize<S: serde::Serializer>(
        &self,
        serializer: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        match self {
            Self::Typed { input, .. } => input.serialize(serializer),
            #[cfg(test)]
            Self::Json { input, .. } => input.serialize(serializer),
            Self::Write { fd, data_base64 } => {
                let mut fields = serializer.serialize_struct("Write", 2)?;
                fields.serialize_field("data_base64", data_base64)?;
                fields.serialize_field("fd", fd)?;
                fields.end()
            }
            Self::PWrite {
                fd,
                data_base64,
                offset,
            } => {
                let mut fields = serializer.serialize_struct("PWrite", 3)?;
                fields.serialize_field("data_base64", data_base64)?;
                fields.serialize_field("fd", fd)?;
                fields.serialize_field("offset", offset)?;
                fields.end()
            }
        }
    }
}
#[derive(Serialize)]
pub struct RequestState {
    pub advances: u64,
    pub now: u64,
    pub transcript_bytes: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub offered_output_bytes: Option<u64>,
}
#[derive(Serialize)]
struct CallbackFrame<'a, 'b> {
    id: u64,
    input: &'a CallbackInput<'b>,
    #[serde(skip_serializing_if = "Option::is_none")]
    local_state: Option<RequestState>,
    op: &'a str,
    #[serde(rename = "type")]
    kind: &'static str,
}
struct BoundedFrame<'a> {
    bytes: &'a mut Vec<u8>,
    capacity_failed: bool,
}
impl Write for BoundedFrame<'_> {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        let length = self.bytes.len().checked_add(bytes.len());
        if length.is_none_or(|length| length > FRAME_LIMIT) {
            self.capacity_failed = true;
            return Err(std::io::Error::other(FrameCapacity));
        }
        let length = length.unwrap();
        if length > self.bytes.capacity() {
            // Reserve a finite chunk without Vec's geometric growth; actual capacity is checked too.
            let desired = length
                .max(self.bytes.capacity().saturating_add(4096))
                .min(FRAME_LIMIT);
            if self
                .bytes
                .try_reserve_exact(desired - self.bytes.len())
                .is_err()
            {
                self.capacity_failed = true;
                return Err(std::io::Error::other(FrameCapacity));
            }
        }
        if self.bytes.capacity() > FRAME_LIMIT {
            self.capacity_failed = true;
            return Err(std::io::Error::other(FrameCapacity));
        }
        self.bytes.extend_from_slice(bytes);
        Ok(bytes.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}
pub fn send_callback(
    output: &mut dyn Write,
    scratch: &mut Vec<u8>,
    id: u64,
    input: &CallbackInput<'_>,
    local_state: Option<RequestState>,
) -> Result<()> {
    #[cfg(feature = "diagnostic-profile")]
    let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
    scratch.clear();
    if scratch.capacity() > FRAME_LIMIT {
        *scratch = Vec::new();
    }
    let frame = CallbackFrame {
        id,
        input,
        local_state,
        op: input.op(),
        kind: "call",
    };
    let mut writer = BoundedFrame {
        bytes: scratch,
        capacity_failed: false,
    };
    let encoded = serde_json::to_writer(&mut writer, &frame);
    let result = match encoded {
        Ok(()) => writer.write_all(b"\n").map_err(Error::new),
        Err(error) => Err(Error::new(error)),
    };
    let capacity_failed = writer.capacity_failed;
    let result = if capacity_failed {
        Err(Error::new(FrameCapacity))
    } else {
        result
    };
    #[cfg(feature = "diagnostic-profile")]
    drop(encoding);
    let result = result.and_then(|()| {
        #[cfg(feature = "diagnostic-profile")]
        let _write = crate::diagnostic::WallSpan::remote_write();
        output
            .write_all(scratch)
            .and_then(|()| output.flush())
            .map_err(Error::new)
    });
    scratch.clear();
    if scratch.capacity() > FRAME_LIMIT {
        *scratch = Vec::new();
    }
    result
}

#[derive(Debug)]
pub struct FrameCapacity;
impl std::fmt::Display for FrameCapacity {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str("protocol frame exceeds 8 MiB")
    }
}
impl std::error::Error for FrameCapacity {}

#[derive(Debug)]
pub struct Unsupported(pub String);
impl std::fmt::Display for Unsupported {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(&self.0)
    }
}
impl std::error::Error for Unsupported {}

pub fn decode_boundary<T: serde::de::DeserializeOwned>(bytes: &[u8]) -> Result<T> {
    serde_json::from_slice(bytes).map_err(|error| {
        if error.to_string().starts_with("unknown field ") {
            Error::new(Unsupported(error.to_string()))
        } else {
            Error::new(error)
        }
    })
}

pub fn classification(error: &Error) -> &'static str {
    if error.downcast_ref::<Unsupported>().is_some() {
        "unsupported"
    } else if let Some(boundary) = error.downcast_ref::<crate::namespace::Boundary>() {
        boundary.kind
    } else if let Some(stop) = error.downcast_ref::<crate::namespace::Stop>() {
        match stop {
            crate::namespace::Stop::Boundary(boundary) => boundary.kind,
            crate::namespace::Stop::StorageCrash(_) => "storage_crash",
            crate::namespace::Stop::Fatal(_) => "infrastructure",
        }
    } else if let Some(interrupt) = error.downcast_ref::<Interrupt>() {
        interrupt.termination.as_str()
    } else if error.downcast_ref::<FrameCapacity>().is_some()
        || error
            .downcast_ref::<crate::model::RemoteCapacity>()
            .is_some()
    {
        "capacity"
    } else if error.downcast_ref::<std::io::Error>().is_some() {
        "infrastructure"
    } else {
        "invalid"
    }
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum InterruptTermination {
    StorageCrash,
    Crash,
    Stop,
}
impl InterruptTermination {
    fn as_str(&self) -> &'static str {
        match self {
            Self::StorageCrash => "storage_crash",
            Self::Crash => "crash",
            Self::Stop => "stop",
        }
    }
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Interrupt {
    #[serde(rename = "type")]
    pub kind: String,
    pub termination: InterruptTermination,
    pub message: String,
    /// Split ownership: the canonical record of a remote call the broker
    /// completed before revoking the incarnation (an output boundary).
    #[serde(default)]
    pub transcript_base64: Option<String>,
    /// Split ownership: Go completed the interrupted remote call before the
    /// broker revoked the incarnation; its resulting clock state.
    #[serde(default)]
    pub completed: Option<CompletedCall>,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CompletedCall {
    pub now: u64,
    pub advances: u64,
}
impl Interrupt {
    pub fn validate(&self) -> Result<()> {
        if self.kind != "interrupt" || self.message.len() > 65536 {
            return Err(Error::msg("invalid modeled interrupt"));
        }
        Ok(())
    }
}
impl std::fmt::Display for Interrupt {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(&self.message)
    }
}
impl std::error::Error for Interrupt {}

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
    /// Host-local directory of compiled modules; never replay evidence.
    #[serde(default)]
    pub compile_cache: Option<String>,
    #[serde(default)]
    pub local_model: Option<crate::model::Config>,
    #[serde(default)]
    pub namespace: Option<crate::namespace::transport::NamespaceDescriptor>,
    #[serde(default)]
    pub storage: Option<crate::storage_transport::StorageDescriptor>,
    #[cfg(feature = "guest-profile")]
    #[serde(default)]
    pub guest_profile: Option<crate::guest_profile::Config>,
}
impl Execute {
    pub fn validate(&self) -> Result<()> {
        #[cfg(feature = "guest-profile")]
        if let Some(profile) = &self.guest_profile {
            profile.validate()?;
        }
        if let Some(model) = &self.local_model {
            model.validate()?;
        }
        if let Some(namespace) = &self.namespace {
            if self.local_model.is_none() {
                return Err(Error::msg("namespace requires local model ownership"));
            }
            namespace.validate()?;
        }
        if let Some(storage) = &self.storage {
            if self.namespace.is_none() {
                return Err(Error::msg("storage requires helper-owned namespace"));
            }
            storage.validate()?;
        }
        if self.schema != SCHEMA {
            return Err(Error::new(Unsupported(format!(
                "unknown execute schema {}",
                self.schema
            ))));
        }
        if self.kind != "execute" {
            return Err(Error::new(Unsupported(format!(
                "unknown execute kind {}",
                self.kind
            ))));
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
        if self
            .compile_cache
            .as_ref()
            .is_some_and(|dir| !std::path::Path::new(dir).is_absolute())
        {
            return Err(Error::msg("invalid compile cache directory"));
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
    #[serde(default)]
    pub local_state: Option<LocalState>,
    #[serde(default)]
    pub transcript_base64: Option<String>,
    /// Split ownership: the broker evaluates delivery after this reply, so the
    /// helper confirms before its next local effect (`sync`/`continue`).
    #[serde(default)]
    pub revocable: bool,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct LocalState {
    pub now: u64,
    pub advances: u64,
}
#[derive(Serialize)]
struct ReplayRefusal {
    error: &'static str,
    check: &'static str,
}

#[derive(Serialize)]
pub struct Outcome {
    #[serde(rename = "type")]
    kind: &'static str,
    pub termination: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exit_code: Option<u32>,
    pub message: String,
    #[serde(flatten, skip_serializing_if = "Option::is_none")]
    replay_refusal: Option<Box<ReplayRefusal>>,
    pub fuel_remaining: u64,
    pub peak_memory_bytes: usize,
    pub output_bytes: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub local_model: Option<Box<Value>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub divergence: Option<Box<crate::model::Divergence>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metrics: Option<Box<crate::metrics::Metrics>>,
}
/// Bounds a terminal message to 64 KiB with an explicit truncation marker.
pub fn bounded_message(message: impl Into<String>) -> String {
    let mut message = message.into();
    if message.len() > 65536 {
        let mut limit = 65536 - " (message truncated)".len();
        while !message.is_char_boundary(limit) {
            limit -= 1;
        }
        message.truncate(limit);
        message.push_str(" (message truncated)");
    }
    message
}
impl Outcome {
    pub fn new(termination: &'static str, message: impl Into<String>) -> Self {
        let message = bounded_message(message);
        Self {
            kind: "result",
            termination,
            exit_code: None,
            message,
            replay_refusal: None,
            fuel_remaining: 0,
            peak_memory_bytes: 0,
            output_bytes: 0,
            local_model: None,
            divergence: None,
            metrics: None,
        }
    }
    /// Stable execution evidence retains every outcome field except host resource observations.
    /// The transport outcome and its raw metrics remain available independently.
    #[must_use]
    pub fn backend_evidence(&self) -> impl Serialize + '_ {
        #[derive(Serialize)]
        struct Evidence<'a> {
            schema: &'static str,
            #[serde(rename = "type")]
            kind: &'static str,
            termination: &'static str,
            #[serde(skip_serializing_if = "Option::is_none")]
            exit_code: Option<u32>,
            message: &'a str,
            #[serde(flatten, skip_serializing_if = "Option::is_none")]
            replay_refusal: Option<&'a ReplayRefusal>,
            fuel_remaining: u64,
            peak_memory_bytes: usize,
            output_bytes: u64,
            #[serde(skip_serializing_if = "Option::is_none")]
            local_model: Option<&'a Value>,
            #[serde(skip_serializing_if = "Option::is_none")]
            divergence: Option<&'a crate::model::Divergence>,
        }
        Evidence {
            schema: "gomad.backend-outcome/v1",
            kind: self.kind,
            termination: self.termination,
            exit_code: self.exit_code,
            message: &self.message,
            replay_refusal: self.replay_refusal.as_deref(),
            fuel_remaining: self.fuel_remaining,
            peak_memory_bytes: self.peak_memory_bytes,
            output_bytes: self.output_bytes,
            local_model: self.local_model.as_deref(),
            divergence: self.divergence.as_deref(),
        }
    }

    #[must_use]
    pub fn refusal(error: &Error) -> Self {
        let mut outcome = Self::new(classification(error), error.to_string());
        if let Some(preflight) = error.downcast_ref::<crate::model::replay::Preflight>() {
            outcome.replay_refusal = Some(Box::new(ReplayRefusal {
                error: "replay_preflight",
                check: preflight.check,
            }));
        }
        outcome
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
    #[cfg(feature = "diagnostic-profile")]
    let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
    let mut bytes = serde_json::to_vec(frame).map_err(|e| Error::msg(e.to_string()))?;
    bytes.push(b'\n');
    if bytes.len() > FRAME_LIMIT {
        return Err(Error::new(FrameCapacity));
    }
    #[cfg(feature = "diagnostic-profile")]
    drop(encoding);
    #[cfg(feature = "diagnostic-profile")]
    let _write = crate::diagnostic::WallSpan::remote_write();
    output
        .write_all(&bytes)
        .and_then(|()| output.flush())
        .map_err(Error::new)
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Continue {
    #[serde(rename = "type")]
    pub(crate) kind: String,
    pub(crate) now: u64,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Chunk {
    #[serde(rename = "type")]
    pub(crate) _kind: String,
    pub(crate) data_base64: String,
}

#[cfg(test)]
mod backend_evidence_tests {
    use super::*;

    #[test]
    fn stable_backend_view_retains_all_protocol_fields_except_volatile_metrics() {
        let mut outcome = Outcome::new("exit", "original deterministic failure");
        outcome.exit_code = Some(1);
        outcome.fuel_remaining = 91;
        outcome.peak_memory_bytes = 65536;
        outcome.output_bytes = 19;
        outcome.local_model = Some(Box::new(serde_json::json!({"now":17,"calls":3})));
        outcome.divergence = Some(Box::new(crate::model::Divergence {
            ordinal: 7,
            reason: 2,
            expected: None,
            observed: None,
            tape_records: 8,
        }));
        outcome.replay_refusal = Some(Box::new(ReplayRefusal {
            error: "replay_preflight",
            check: "identity",
        }));
        outcome.metrics = Some(Box::new(crate::metrics::Metrics {
            execute_nanos: 123,
            ..Default::default()
        }));
        let native = serde_json::to_value(&outcome).unwrap();
        let mut expected = native.clone();
        expected.as_object_mut().unwrap().remove("metrics");
        expected["schema"] = serde_json::json!("gomad.backend-outcome/v1");
        assert_eq!(
            serde_json::to_value(outcome.backend_evidence()).unwrap(),
            expected
        );
        outcome.metrics.as_mut().unwrap().execute_nanos = 456;
        assert_ne!(serde_json::to_value(&outcome).unwrap(), native);
        assert_eq!(
            serde_json::to_value(outcome.backend_evidence()).unwrap(),
            expected
        );
        outcome.fuel_remaining -= 1;
        assert_ne!(
            serde_json::to_value(outcome.backend_evidence()).unwrap(),
            expected
        );
    }
}
