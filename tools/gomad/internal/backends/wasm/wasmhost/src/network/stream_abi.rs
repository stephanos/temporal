//! The `gomad_wasm_v2::net_stream` buffer codec, ported from Go
//! `wasi/network_v2.go` `runtimeNetworkStream` (layout and validation) and
//! `networkV2Errno` (fn-178.12). The header is 64 bytes, big-endian:
//! `[0:4]` version 2, `[4:8]` op, `[8:16]` request ID, `[16:24]` handle,
//! `[24:32]` deadline, `[32:36]` port, `[36:40]` count, `[40:44]` status,
//! `[44:48]` errno, `[48:52]` local port, `[52:56]` remote port, `[56:60]`
//! dial address; `[40:48]`, `[52:56]` and `[60:64]` are zero on input.

pub const HEADER: usize = 64;
pub const MAXIMUM_CHUNK_BYTES: usize = 64 << 10;
/// Go `netmodel.StreamStallTimeout`: above the WASI errno range.
pub const STREAM_STALL_TIMEOUT: u32 = 1 << 16 | 22;

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
mod tests {
    use super::*;

    fn header(op: u32) -> Vec<u8> {
        let mut data = vec![0u8; HEADER];
        set_u32(&mut data, 0, 2);
        set_u32(&mut data, 4, op);
        data
    }

    #[test]
    fn stream_abi_refuses_nonzero_reserved_fields_and_bad_versions() {
        assert!(decode(&header(1)).is_ok());
        for at in [40, 44, 47, 52, 55, 60, 63] {
            let mut data = header(1);
            data[at] = 1;
            assert_eq!(
                decode(&data),
                Err(Invalid("network v2 stream layout")),
                "{at}"
            );
        }
        let mut data = header(1);
        set_u32(&mut data, 0, 3);
        assert!(decode(&data).is_err());
        assert!(decode(&data[..63]).is_err());
        assert!(decode(&vec![0; HEADER + MAXIMUM_CHUNK_BYTES + 1]).is_err());
        // The dial address at [56:60] and the in-range fields are free.
        let mut data = header(2);
        data[56..60].copy_from_slice(&[10, 0, 0, 2]);
        assert_eq!(decode(&data).unwrap().address, 0x0a00_0002);
    }

    #[test]
    fn stream_abi_payload_rules_and_errno_table_match_go() {
        let call = |op, count| Call {
            op,
            id: 1,
            handle: 0,
            deadline: 0,
            port: 0,
            count,
            address: 0,
        };
        assert!(check_payload(&call(5, 3), HEADER + 3).is_ok());
        assert_eq!(
            check_payload(&call(5, 3), HEADER + 2),
            Err(Invalid("network v2 write payload"))
        );
        assert!(check_payload(&call(4, 8), HEADER + 8).is_ok());
        assert!(check_payload(&call(4, 8), HEADER).is_err());
        assert!(check_payload(&call(12, 0), HEADER + 100).is_ok());
        assert!(check_payload(&call(1, 0), HEADER + 1).is_err());
        for (message, value) in [
            ("", 0),
            ("EOF", 0),
            ("context canceled", 11),
            ("connection timed out", STREAM_STALL_TIMEOUT),
            ("i/o timeout", 73),
            ("read deadline passed", 73),
            ("connection refused", 14),
            ("network resources exhausted", 42),
            ("use of closed network connection", 15),
            ("stale file handle", 8),
        ] {
            assert_eq!(errno(message), value, "{message}");
        }
    }
}
