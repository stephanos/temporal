use super::replay_plan::{ExecutionIdentity, PrefixControl, ReplayPlan};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::io::BufRead;
use wasmtime::{Error, Result};

pub const SCHEMA: &str = "gomad.replay-trace/v1";
pub const CHUNK: u64 = 65536;

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    #[serde(default)]
    pub schema: String,
    pub execution_identity: ExecutionIdentity,
    pub trace_sha256: String,
    pub tape_sha256: String,
    pub decisions: u64,
    pub trace_bytes: u64,
    pub trace_frames: u64,
    pub prefix_control: Option<PrefixControl>,
}

#[derive(Debug)]
pub struct Preflight {
    pub check: &'static str,
    message: String,
}
impl std::fmt::Display for Preflight {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            formatter,
            "replay preflight {}: {}",
            self.check, self.message
        )
    }
}
impl std::error::Error for Preflight {}

pub(super) fn failure(check: &'static str, message: impl Into<String>) -> Error {
    Error::new(Preflight {
        check,
        message: message.into(),
    })
}

impl Request {
    /// # Errors
    /// Refuses unsupported replay schemas or inconsistent bounded frame declarations.
    pub fn validate(&self, limit: u64) -> Result<()> {
        if self.schema != SCHEMA {
            return Err(failure(
                "schema",
                format!("received {:?}; expected {SCHEMA}", self.schema),
            ));
        }
        if self.trace_bytes > limit || self.trace_frames != self.trace_bytes.div_ceil(CHUNK) {
            return Err(failure(
                "framing",
                "invalid replay trace byte/frame declaration",
            ));
        }
        Ok(())
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Marker {
    #[serde(rename = "type")]
    kind: String,
    schema: String,
    trace_bytes: u64,
    trace_frames: u64,
    instance: Option<String>,
    incarnation: Option<u64>,
}

fn read_marker(
    input: &mut dyn BufRead,
    kind: &str,
    request: &Request,
    tag: Option<(&str, u64)>,
) -> Result<()> {
    let bytes = crate::protocol::read_frame(input)
        .map_err(|error| failure("framing", error.to_string()))?;
    let marker: Marker =
        serde_json::from_slice(&bytes).map_err(|error| failure("framing", error.to_string()))?;
    if marker.schema != SCHEMA {
        return Err(failure(
            "schema",
            format!("received {:?}; expected {SCHEMA}", marker.schema),
        ));
    }
    if marker.kind != kind
        || marker.trace_bytes != request.trace_bytes
        || marker.trace_frames != request.trace_frames
    {
        return Err(failure(
            "framing",
            "replay trace marker differs from declaration",
        ));
    }
    match tag {
        Some((node, incarnation))
            if marker.instance.as_deref() == Some(node)
                && marker.incarnation == Some(incarnation) =>
        {
            Ok(())
        }
        None if marker.instance.is_none() && marker.incarnation.is_none() => Ok(()),
        _ => Err(failure(
            "framing",
            "replay trace is not addressed to the waiting instance/incarnation",
        )),
    }
}

pub fn read(
    input: &mut dyn BufRead,
    request: &Request,
    mode: u8,
    limit: u64,
    tag: Option<(&str, u64)>,
) -> Result<ReplayPlan> {
    request.validate(limit)?;
    read_marker(input, "replay_trace", request, tag)?;
    let length = usize::try_from(request.trace_bytes)
        .map_err(|_| failure("framing", "replay trace exceeds address space"))?;
    let mut trace = Vec::new();
    trace
        .try_reserve_exact(length)
        .map_err(|error| failure("framing", error.to_string()))?;
    for _ in 0..request.trace_frames {
        let mut encoded = [0; 4];
        input
            .read_exact(&mut encoded)
            .map_err(|error| failure("framing", error.to_string()))?;
        let count = u64::from(u32::from_be_bytes(encoded));
        if count == 0 || count != CHUNK.min(request.trace_bytes - trace.len() as u64) {
            return Err(failure("framing", "invalid replay trace frame length"));
        }
        let count = usize::try_from(count)
            .map_err(|_| failure("framing", "replay frame exceeds address space"))?;
        let offset = trace.len();
        trace.resize(offset + count, 0);
        input
            .read_exact(&mut trace[offset..])
            .map_err(|error| failure("framing", error.to_string()))?;
    }
    read_marker(input, "replay_trace_end", request, tag)?;
    preflight(&trace, request, mode, limit)
}

pub(crate) fn preflight(
    trace: &[u8],
    request: &Request,
    mode: u8,
    limit: u64,
) -> Result<ReplayPlan> {
    request.validate(limit)?;
    if trace.len() as u64 != request.trace_bytes {
        return Err(failure(
            "framing",
            "replay trace length differs from declaration",
        ));
    }
    super::replay_plan::toolchain_key(&request.execution_identity)
        .map_err(|error| failure("execution_identity", error))?;
    if format!("{:x}", Sha256::digest(trace)) != request.trace_sha256 {
        return Err(failure("trace_sha256", "replay trace digest mismatch"));
    }
    if request
        .trace_bytes
        .checked_add(64)
        .is_none_or(|bytes| bytes > limit)
    {
        return Err(failure(
            "trace_decode",
            "replay trace exceeds its mapping limit",
        ));
    }
    let mut plan = super::replay_plan::project(trace, &request.execution_identity)
        .map_err(|error| failure("trace_decode", error))?;
    if let Some(control) = &request.prefix_control {
        if mode != 3 {
            return Err(failure(
                "tape_sha256",
                "exact replay forbids prefix controls",
            ));
        }
        plan = plan
            .prefix(control)
            .map_err(|error| failure("tape_sha256", error))?;
    }
    if crate::world::digest::hex(&plan.sha256) != request.tape_sha256 {
        return Err(failure("tape_sha256", "replay tape digest mismatch"));
    }
    if plan.decisions.len() as u64 != request.decisions {
        return Err(failure("decisions", "replay decision count mismatch"));
    }
    Ok(plan)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{Value, json};
    use std::io::Cursor;

    fn request() -> Request {
        let fixture: Value =
            serde_json::from_str(include_str!("../../tests/testdata/replay_plan/empty.json"))
                .unwrap();
        serde_json::from_value(json!({"schema":SCHEMA,"execution_identity":fixture["identity"],
            "trace_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            "tape_sha256":fixture["expected"]["sha256"],"decisions":0,"trace_bytes":0,"trace_frames":0})).unwrap()
    }

    fn marker(kind: &str, request: &Request, tag: Option<(&str, u64)>) -> Vec<u8> {
        let mut value = json!({"type":kind,"schema":request.schema,"trace_bytes":request.trace_bytes,"trace_frames":request.trace_frames});
        if let Some((node, incarnation)) = tag {
            value["instance"] = node.into();
            value["incarnation"] = incarnation.into();
        }
        let mut bytes = serde_json::to_vec(&value).unwrap();
        bytes.push(b'\n');
        bytes
    }

    fn block(request: &Request, payload: &[u8], tag: Option<(&str, u64)>) -> Vec<u8> {
        let mut bytes = marker("replay_trace", request, tag);
        for chunk in payload.chunks(usize::try_from(CHUNK).unwrap()) {
            bytes.extend(u32::try_from(chunk.len()).unwrap().to_be_bytes());
            bytes.extend(chunk);
        }
        bytes.extend(marker("replay_trace_end", request, tag));
        bytes
    }

    #[test]
    fn replay_preflight_accepts_empty_and_preserves_following_json() {
        let request = request();
        for tag in [None, Some(("node", 7))] {
            let mut bytes = block(&request, &[], tag);
            bytes.extend(b"{\"type\":\"authorize\"}\n");
            let mut input = Cursor::new(bytes);
            assert!(read(&mut input, &request, 2, 4096, tag).is_ok());
            assert_eq!(
                crate::protocol::read_frame(&mut input).unwrap(),
                b"{\"type\":\"authorize\"}\n"
            );
        }
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "One table keeps the complete ordered preflight refusal checks together."
    )]
    fn replay_preflight_refuses_malformed_blocks_and_expectations() {
        for name in [
            "schema",
            "oversized",
            "overflow",
            "frame-count",
            "missing-declaration",
            "wrong-declaration",
            "missing-end",
            "wrong-end",
            "wrong-end-count",
            "extra-frame",
            "truncated-length",
            "truncated-payload",
            "zero-frame",
            "oversized-frame",
            "wrong-tag",
            "stale-tag",
            "identity",
            "trace-digest",
            "tape-digest",
            "decisions",
            "exact-control",
        ] {
            let mut request = request();
            let mut bytes = block(&request, &[], None);
            let mut tag = None;
            let check = match name {
                "schema" => {
                    request.schema = "gomad.replay-trace/v99".into();
                    "schema"
                }
                "oversized" => {
                    request.trace_bytes = 4097;
                    request.trace_frames = 1;
                    "framing"
                }
                "overflow" => {
                    request.trace_bytes = u64::MAX;
                    request.trace_frames = u64::MAX;
                    "framing"
                }
                "frame-count" => {
                    request.trace_frames = 1;
                    "framing"
                }
                "missing-declaration" => {
                    bytes = marker("replay_trace_end", &request, None);
                    "framing"
                }
                "wrong-declaration" => {
                    bytes[10] = b'x';
                    "framing"
                }
                "missing-end" => {
                    bytes = marker("replay_trace", &request, None);
                    "framing"
                }
                "wrong-end" => {
                    bytes = marker("replay_trace", &request, None);
                    bytes.extend(marker("replay_trace", &request, None));
                    "framing"
                }
                "wrong-end-count" => {
                    bytes = marker("replay_trace", &request, None);
                    request.trace_frames = 1;
                    bytes.extend(marker("replay_trace_end", &request, None));
                    request.trace_frames = 0;
                    "framing"
                }
                "extra-frame" => {
                    bytes = marker("replay_trace", &request, None);
                    bytes.extend([0, 0, 0, 1, 255]);
                    bytes.extend(marker("replay_trace_end", &request, None));
                    "framing"
                }
                "truncated-length" | "truncated-payload" | "zero-frame" | "oversized-frame" => {
                    request.trace_bytes = 1;
                    request.trace_frames = 1;
                    bytes = marker("replay_trace", &request, None);
                    match name {
                        "truncated-length" => bytes.extend([0, 0]),
                        "truncated-payload" => bytes.extend([0, 0, 0, 1]),
                        "zero-frame" => bytes.extend([0, 0, 0, 0]),
                        _ => bytes.extend(65537u32.to_be_bytes()),
                    }
                    "framing"
                }
                "wrong-tag" | "stale-tag" => {
                    tag = Some(("node", 7));
                    bytes = block(
                        &request,
                        &[],
                        if name == "wrong-tag" {
                            Some(("other", 7))
                        } else {
                            Some(("node", 6))
                        },
                    );
                    "framing"
                }
                "identity" => {
                    request.execution_identity.target_sha256 = [0; 32];
                    "execution_identity"
                }
                "trace-digest" => {
                    request.trace_sha256 = "0".repeat(64);
                    "trace_sha256"
                }
                "tape-digest" => {
                    request.tape_sha256 = "0".repeat(64);
                    "tape_sha256"
                }
                "decisions" => {
                    request.decisions = 1;
                    "decisions"
                }
                "exact-control" => {
                    request.prefix_control = Some(PrefixControl {
                        cutoff: 0,
                        r#override: None,
                    });
                    "tape_sha256"
                }
                _ => unreachable!(),
            };
            let error = read(&mut Cursor::new(bytes), &request, 2, 4096, tag)
                .err()
                .unwrap();
            assert_eq!(
                error.downcast_ref::<Preflight>().unwrap().check,
                check,
                "{name}: {error}"
            );
            if name == "schema" {
                assert!(error.to_string().contains("gomad.replay-trace/v99"));
                assert!(error.to_string().contains(SCHEMA));
            }
        }
    }

    #[test]
    fn replay_preflight_binary_lf_and_chunk_boundaries_reach_trace_validation() {
        for length in [65535, 65536, 65537, 131_072] {
            let mut request = request();
            let payload = vec![b'\n'; length];
            request.trace_bytes = length as u64;
            request.trace_frames = request.trace_bytes.div_ceil(CHUNK);
            request.trace_sha256 = format!("{:x}", Sha256::digest(&payload));
            let mut bytes = block(&request, &payload, Some(("node", 2)));
            bytes.extend(b"{\"type\":\"next\"}\n");
            let mut input = Cursor::new(bytes);
            let error = read(&mut input, &request, 2, 1 << 20, Some(("node", 2)))
                .err()
                .unwrap();
            assert_eq!(
                error.downcast_ref::<Preflight>().unwrap().check,
                "trace_decode",
                "{length}: {error}"
            );
            assert_eq!(
                crate::protocol::read_frame(&mut input).unwrap(),
                b"{\"type\":\"next\"}\n"
            );
        }
    }
}
