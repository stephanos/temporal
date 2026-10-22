//! Per-instance binary network evidence (`GMNETEV1`, fn-178.12) and the exact
//! Go JSON of each operations-log entry.
//!
//! Go's broker-mode document is `{"profile","operations","native","world"}`
//! (`wasi/network_v2.go` `NetworkV2Evidence`). The helper sends its structure
//! instead; Go rebuilds the document with its own codecs (fn-178.13). The
//! per-append log bound (`network-v2-transcript-bytes`) is enforced during the
//! call, so the helper computes each entry's Go JSON exactly.
//!
//! Layout (integers are unsigned LEB128 varints, minimal; `str` and `bytes`
//! are a varint length then the bytes; `flag` is one byte 0 or 1):
//!
//! ```text
//! "GMNETEV1" version=1 str(profile) log_json_bytes entry_count
//! entry*: u8 kind
//!   1 operation:  at request result
//!   3 quiescence: str(kind) before after
//!                 n {request_id event_id at str(kind) bytes(payload)}*
//!                 n {request_id}*
//! request: run str(node) incarnation str(address) str(op) handle str(host)
//!          port bytes(bytes) count deadline request_id
//! result:  request_id str(state) handle str(local.ip) local.port
//!          str(remote.ip) remote.port bytes(bytes) count deadline str(error)
//! native:  flag(transitions) [n transition*]
//!          flag(nodes) [n {str node, str address, last_incarnation,
//!                          next_listener_port, next_client_port}*]
//!          flag(links) [n {str from, str to, flag enabled, delay_nanos}*]
//!          flag(listeners) [n {endpoint, flag closed}*]
//!          flag(connections) [n {client_bytes, server_bytes, identity,
//!                                endpoint client, endpoint server,
//!                                flag closed, flag reset}*]
//!          next_connection str(transition_sha256) str(identity)
//! transition: ordinal str(kind) endpoint endpoint connection bytes count
//!             delay_nanos str(outcome)
//! endpoint: str(node) incarnation str(address) port
//! world:   flag [bytes(expanded GMWSNAP1 wire)]
//! ```
//!
//! Kind 2 (a node-local fault) is reserved: brokered nodes refuse local fault
//! schedules. Varints, raw bytes and the absence of field names keep the stream
//! no longer than the JSON document it encodes, so bounding it by the
//! instance's transcript bytes never refuses evidence Go would accept; Go
//! re-applies the exact `network-v2-evidence-bytes` bound to the rebuilt JSON.
use crate::network::{OperationResult, Request, history};
use crate::world;
use base64::{Engine as _, engine::general_purpose::STANDARD};

pub const MAGIC: &[u8; 8] = b"GMNETEV1";
pub const VERSION: u64 = 1;

pub fn varint(out: &mut Vec<u8>, mut value: u64) {
    loop {
        let byte = (value & 0x7f) as u8;
        value >>= 7;
        if value == 0 {
            out.push(byte);
            return;
        }
        out.push(byte | 0x80);
    }
}

fn text(out: &mut Vec<u8>, value: &str) {
    bytes(out, value.as_bytes());
}

fn bytes(out: &mut Vec<u8>, value: &[u8]) {
    varint(out, value.len() as u64);
    out.extend_from_slice(value);
}

fn flag(out: &mut Vec<u8>, value: bool) {
    out.push(u8::from(value));
}

/// Go `encoding/json` string encoding (HTML-safe, short escapes for `\n`,
/// `\r`, `\t`, `\b`, `\f` (Go 1.22+), `\u00XX` for other controls, ` `/` `).
fn json_string(out: &mut Vec<u8>, value: &str) {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    out.push(b'"');
    for character in value.chars() {
        match character {
            '"' => out.extend_from_slice(b"\\\""),
            '\\' => out.extend_from_slice(b"\\\\"),
            '\n' => out.extend_from_slice(b"\\n"),
            '\r' => out.extend_from_slice(b"\\r"),
            '\t' => out.extend_from_slice(b"\\t"),
            '\u{8}' => out.extend_from_slice(b"\\b"),
            '\u{c}' => out.extend_from_slice(b"\\f"),
            '<' | '>' | '&' | '\u{2028}' | '\u{2029}' => {
                out.extend_from_slice(format!("\\u{:04x}", u32::from(character)).as_bytes());
            }
            character if (character as u32) < 0x20 => {
                let value = character as u32;
                out.extend_from_slice(b"\\u00");
                out.push(HEX[(value >> 4) as usize]);
                out.push(HEX[(value & 15) as usize]);
            }
            character => {
                let mut buffer = [0; 4];
                out.extend_from_slice(character.encode_utf8(&mut buffer).as_bytes());
            }
        }
    }
    out.push(b'"');
}

fn json_field(out: &mut Vec<u8>, first: &mut bool, key: &str) {
    if !*first {
        out.push(b',');
    }
    *first = false;
    json_string(out, key);
    out.push(b':');
}

fn json_number(out: &mut Vec<u8>, first: &mut bool, key: &str, value: impl ToString + Copy) {
    json_field(out, first, key);
    out.extend_from_slice(value.to_string().as_bytes());
}

fn json_text(out: &mut Vec<u8>, first: &mut bool, key: &str, value: &str) {
    json_field(out, first, key);
    json_string(out, value);
}

/// Go `json.Marshal(netmodel.Request)`.
fn request_json(out: &mut Vec<u8>, request: &Request) {
    let first = &mut true;
    out.push(b'{');
    json_field(out, first, "domain");
    out.push(b'{');
    let inner = &mut true;
    json_number(out, inner, "run", request.domain.run);
    json_text(out, inner, "node", &request.domain.node);
    json_number(out, inner, "incarnation", request.domain.incarnation);
    json_text(out, inner, "address", &request.domain.address);
    out.push(b'}');
    json_text(out, first, "op", &request.op);
    if request.handle != 0 {
        json_number(out, first, "handle", request.handle);
    }
    if !request.host.is_empty() {
        json_text(out, first, "host", &request.host);
    }
    if request.port != 0 {
        json_number(out, first, "port", request.port);
    }
    if !request.bytes.is_empty() {
        json_text(out, first, "bytes", &STANDARD.encode(&request.bytes));
    }
    if request.count != 0 {
        json_number(out, first, "count", request.count);
    }
    if request.deadline != 0 {
        json_number(out, first, "deadline", request.deadline);
    }
    json_number(out, first, "request_id", request.request_id);
    out.push(b'}');
}

/// Go `json.Marshal(netmodel.Result)`; `Address` has no JSON tags and a struct
/// is never empty, so `local` and `remote` are always present.
fn result_json(out: &mut Vec<u8>, result: &OperationResult) {
    let first = &mut true;
    out.push(b'{');
    json_number(out, first, "request_id", result.request_id);
    json_text(out, first, "state", &result.state);
    if result.handle != 0 {
        json_number(out, first, "handle", result.handle);
    }
    for (key, address) in [("local", &result.local), ("remote", &result.remote)] {
        json_field(out, first, key);
        out.push(b'{');
        let inner = &mut true;
        json_text(out, inner, "IP", &address.ip);
        json_number(out, inner, "Port", address.port);
        out.push(b'}');
    }
    if !result.bytes.is_empty() {
        json_text(out, first, "bytes", &STANDARD.encode(&result.bytes));
    }
    if result.count != 0 {
        json_number(out, first, "count", result.count);
    }
    if result.deadline != 0 {
        json_number(out, first, "deadline", result.deadline);
    }
    if !result.error.is_empty() {
        json_text(out, first, "error", &result.error);
    }
    out.push(b'}');
}

fn request_binary(out: &mut Vec<u8>, request: &Request) {
    varint(out, request.domain.run);
    text(out, &request.domain.node);
    varint(out, request.domain.incarnation);
    text(out, &request.domain.address);
    text(out, &request.op);
    varint(out, request.handle);
    text(out, &request.host);
    varint(out, request.port.cast_unsigned());
    bytes(out, &request.bytes);
    varint(out, u64::from(request.count));
    varint(out, request.deadline);
    varint(out, request.request_id);
}

fn result_binary(out: &mut Vec<u8>, result: &OperationResult) {
    varint(out, result.request_id);
    text(out, &result.state);
    varint(out, result.handle);
    text(out, &result.local.ip);
    varint(out, result.local.port);
    text(out, &result.remote.ip);
    varint(out, result.remote.port);
    bytes(out, &result.bytes);
    varint(out, result.count);
    varint(out, result.deadline);
    text(out, &result.error);
}

/// One `{at, request, result}` log entry: Go's JSON and the binary record.
pub fn operation_entry(at: u64, request: &Request, result: &OperationResult) -> (Vec<u8>, Vec<u8>) {
    let mut json = Vec::with_capacity(256);
    let first = &mut true;
    json.push(b'{');
    json_number(&mut json, first, "at", at);
    json_field(&mut json, first, "request");
    request_json(&mut json, request);
    json_field(&mut json, first, "result");
    result_json(&mut json, result);
    json.push(b'}');
    let mut binary = vec![1];
    varint(&mut binary, at);
    request_binary(&mut binary, request);
    result_binary(&mut binary, result);
    (json, binary)
}

/// One World quiescence log entry: Go's `json.Marshal(world.Quiescence)`
/// (identities and times as decimal strings, empty lists omitted) and the
/// binary record.
pub fn quiescence_entry(quiescence: &world::Quiescence) -> (Vec<u8>, Vec<u8>) {
    let mut json = Vec::with_capacity(128);
    let first = &mut true;
    let quoted = |value: String| format!("\"{value}\"");
    json.push(b'{');
    json_text(&mut json, first, "kind", quiescence.kind.as_str());
    json_field(&mut json, first, "before");
    json.extend_from_slice(quoted(quiescence.before.to_string()).as_bytes());
    json_field(&mut json, first, "after");
    json.extend_from_slice(quoted(quiescence.after.to_string()).as_bytes());
    if !quiescence.deliveries.is_empty() {
        json_field(&mut json, first, "deliveries");
        json.push(b'[');
        for (index, delivery) in quiescence.deliveries.iter().enumerate() {
            if index != 0 {
                json.push(b',');
            }
            let inner = &mut true;
            json.push(b'{');
            for (key, value) in [
                ("requestId", delivery.request_id.to_string()),
                ("eventId", delivery.event_id.to_string()),
                ("at", delivery.at.to_string()),
            ] {
                json_field(&mut json, inner, key);
                json.extend_from_slice(quoted(value).as_bytes());
            }
            json_text(&mut json, inner, "kind", &delivery.kind);
            if !delivery.payload.is_empty() {
                json_text(
                    &mut json,
                    inner,
                    "payload",
                    &STANDARD.encode(&delivery.payload),
                );
            }
            json.push(b'}');
        }
        json.push(b']');
    }
    if !quiescence.blocked.is_empty() {
        json_field(&mut json, first, "blocked");
        json.push(b'[');
        for (index, blocked) in quiescence.blocked.iter().enumerate() {
            if index != 0 {
                json.push(b',');
            }
            json.extend_from_slice(quoted(blocked.to_string()).as_bytes());
        }
        json.push(b']');
    }
    json.push(b'}');
    let mut binary = vec![3];
    text(&mut binary, quiescence.kind.as_str());
    varint(&mut binary, quiescence.before.cast_unsigned());
    varint(&mut binary, quiescence.after.cast_unsigned());
    varint(&mut binary, quiescence.deliveries.len() as u64);
    for delivery in &quiescence.deliveries {
        varint(&mut binary, delivery.request_id);
        varint(&mut binary, delivery.event_id);
        varint(&mut binary, delivery.at.cast_unsigned());
        text(&mut binary, &delivery.kind);
        bytes(&mut binary, &delivery.payload);
    }
    varint(&mut binary, quiescence.blocked.len() as u64);
    for blocked in &quiescence.blocked {
        varint(&mut binary, *blocked);
    }
    (json, binary)
}

fn endpoint(out: &mut Vec<u8>, endpoint: &history::Endpoint) {
    text(out, &endpoint.node);
    varint(out, endpoint.incarnation);
    text(out, &endpoint.address);
    varint(out, endpoint.port);
}

fn optional<T>(out: &mut Vec<u8>, values: Option<&[T]>, mut each: impl FnMut(&mut Vec<u8>, &T)) {
    flag(out, values.is_some());
    if let Some(values) = values {
        varint(out, values.len() as u64);
        for value in values {
            each(out, value);
        }
    }
}

fn record(out: &mut Vec<u8>, record: &history::Record) {
    optional(out, record.transitions.as_deref(), |out, transition| {
        varint(out, transition.ordinal);
        text(out, &transition.kind);
        endpoint(out, &transition.source);
        endpoint(out, &transition.destination);
        varint(out, transition.connection);
        varint(out, transition.bytes);
        varint(out, transition.count);
        varint(out, transition.delay_nanos);
        text(out, &transition.outcome);
    });
    let snapshot = &record.snapshot;
    optional(out, snapshot.nodes.as_deref(), |out, node| {
        text(out, &node.node);
        text(out, &node.address);
        varint(out, node.last_incarnation);
        varint(out, node.next_listener_port);
        varint(out, node.next_client_port);
    });
    optional(out, snapshot.links.as_deref(), |out, link| {
        text(out, &link.from);
        text(out, &link.to);
        flag(out, link.enabled);
        varint(out, link.delay_nanos);
    });
    optional(out, snapshot.listeners.as_deref(), |out, listener| {
        endpoint(out, &listener.endpoint);
        flag(out, listener.closed);
    });
    optional(out, snapshot.connections.as_deref(), |out, connection| {
        varint(out, connection.client_bytes);
        varint(out, connection.server_bytes);
        varint(out, connection.identity);
        endpoint(out, &connection.client);
        endpoint(out, &connection.server);
        flag(out, connection.closed);
        flag(out, connection.reset);
    });
    varint(out, snapshot.next_connection);
    text(out, &snapshot.transition_sha256);
    text(out, &snapshot.identity);
}

pub fn document(
    log_json_bytes: u64,
    entry_count: u64,
    entries: &[u8],
    native: &history::Record,
    world_wire: &[u8],
) -> Vec<u8> {
    let mut out = Vec::with_capacity(64 + entries.len() + world_wire.len());
    out.extend_from_slice(MAGIC);
    varint(&mut out, VERSION);
    text(&mut out, super::PROFILE);
    varint(&mut out, log_json_bytes);
    varint(&mut out, entry_count);
    out.extend_from_slice(entries);
    record(&mut out, native);
    flag(&mut out, true);
    bytes(&mut out, world_wire);
    out
}
