//! The `gomad_wasm_v2::net_stream` buffer codec, ported from Go
//! `wasi/network_v2.go` `runtimeNetworkStream` (layout and validation) and
//! `networkV2Errno` (fn-178.12). The header is 64 bytes, big-endian:
//! `[0:4]` version 2, `[4:8]` op, `[8:16]` request ID, `[16:24]` handle,
//! `[24:32]` deadline, `[32:36]` port, `[36:40]` count, `[40:44]` status,
//! `[44:48]` errno, `[48:52]` local port, `[52:56]` remote port, `[56:60]`
//! dial address; `[40:48]`, `[52:56]` and `[60:64]` are zero on input.

pub const HEADER: usize = 64;
pub const MAXIMUM_CHUNK_BYTES: usize = 64 << 10;
pub const READINESS_PROBE_V1: u32 = 14;
pub const MAXIMUM_READINESS_REQUESTS: usize = 128;
/// Go `netmodel.StreamStallTimeout`: above the WASI errno range.
pub const STREAM_STALL_TIMEOUT: u32 = 1 << 16 | 0x16;

pub const OPERATIONS: [&str; 14] = [
    "query",
    "listen",
    "dial",
    "accept",
    "read",
    "write",
    "close",
    "close_read",
    "close_write",
    "deadline",
    "read_deadline",
    "write_deadline",
    "poll",
    "cancel",
];

/// Errors the Go environment reports as `WASI invalid: <dimension>`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Invalid(pub &'static str);

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Call {
    pub op: u32,
    pub id: u64,
    pub handle: u64,
    pub deadline: u64,
    pub port: u32,
    pub count: u32,
    pub address: u32,
}

fn u32_at(data: &[u8], at: usize) -> u32 {
    u32::from_be_bytes(data[at..at + 4].try_into().unwrap())
}

fn u64_at(data: &[u8], at: usize) -> u64 {
    u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
}

/// Go's layout check, applied before any network state is consulted.
pub fn decode(data: &[u8]) -> Result<Call, Invalid> {
    if data.len() < HEADER
        || data.len() > HEADER + MAXIMUM_CHUNK_BYTES
        || u32_at(data, 0) != 2
        || data[40..48].iter().any(|byte| *byte != 0)
        || data[52..56].iter().any(|byte| *byte != 0)
        || data[60..64].iter().any(|byte| *byte != 0)
    {
        return Err(Invalid("network v2 stream layout"));
    }
    Ok(Call {
        op: u32_at(data, 4),
        id: u64_at(data, 8),
        handle: u64_at(data, 16),
        deadline: u64_at(data, 24),
        port: u32_at(data, 32),
        count: u32_at(data, 36),
        address: u32_at(data, 56),
    })
}

pub fn query(call: &Call, length: usize) -> Result<(), Invalid> {
    if length != HEADER
        || call.id != 0
        || call.handle != 0
        || call.deadline != 0
        || call.port != 0
        || call.count != 0
    {
        return Err(Invalid("network v2 query"));
    }
    Ok(())
}

/// The separately versioned, non-consuming `readiness_probe_v1` operation.
pub fn readiness_probe_v1(call: &Call, data: &[u8]) -> Result<Vec<u64>, Invalid> {
    if call.id != 0
        || call.handle != 0
        || call.deadline != 0
        || call.port != 0
        || call.address != 0
        || call.count as usize > MAXIMUM_READINESS_REQUESTS
        || data.len() != HEADER + 8 * call.count as usize
        || data[48..52].iter().any(|byte| *byte != 0)
    {
        return Err(Invalid("network v2 readiness probe layout"));
    }
    let mut ids = Vec::with_capacity(call.count as usize);
    for payload in data[HEADER..].chunks_exact(8) {
        let id = u64_at(payload, 0);
        if id == 0 || id >= 1 << 48 || ids.contains(&id) {
            return Err(Invalid("network v2 readiness probe request"));
        }
        ids.push(id);
    }
    Ok(ids)
}

/// Go's per-operation payload rule: writes and reads carry exactly `count`
/// payload bytes, polls any buffer, every other operation only the header.
pub fn check_payload(call: &Call, length: usize) -> Result<(), Invalid> {
    if call.op == 5 {
        if length != HEADER + call.count as usize {
            return Err(Invalid("network v2 write payload"));
        }
    } else if call.op == 4 && length != HEADER + call.count as usize
        || call.op != 4 && call.op != 5 && call.op != 12 && length != HEADER
    {
        return Err(Invalid("network v2 payload"));
    }
    Ok(())
}

/// Go `networkV2Errno`.
pub fn errno(message: &str) -> u32 {
    match message {
        "" | "EOF" => 0,
        "context canceled" => 11,
        "connection timed out" => STREAM_STALL_TIMEOUT,
        "context deadline exceeded" | "i/o timeout" | "file closing" | "deadline exceeded" => 73,
        "connection refused" => 14,
        "network resources exhausted" => 42,
        "use of closed network connection" => 15,
        message if message.contains("deadline") => 73,
        _ => 8,
    }
}

pub fn set_u32(data: &mut [u8], at: usize, value: u32) {
    data[at..at + 4].copy_from_slice(&value.to_be_bytes());
}

pub fn set_u64(data: &mut [u8], at: usize, value: u64) {
    data[at..at + 8].copy_from_slice(&value.to_be_bytes());
}

#[cfg(test)]
mod corpus;
