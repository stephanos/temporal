use crate::namespace::{self, Image, ImageEntry, Limits, Namespace, Open, Operation, Output};
use base64::{Engine, engine::general_purpose::STANDARD};
use serde::{Deserialize, de::MapAccess};
use serde_json::value::RawValue;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::{self, BufRead, Read, Write};

const FRAME_LIMIT: usize = 8 << 20;
const BUFFER_LIMIT: u32 = 4 << 20;
const VOLUME_MODEL: &str = "internal/model/volume";
const VOLUME_UNSUPPORTED: &str = r#"{"unsupported":true,"model":"internal/model/volume","message":"Rust volume owner awaits fn-176.2"}"#;
type Fields = BTreeMap<String, Box<RawValue>>;
type Result<T> = std::result::Result<T, String>;

const BINARY_HEADER_BYTES: u64 = 8;
const BINARY_RECORD_BYTES: u64 = 16;
const BINARY_CLOCK_RUN_BYTES: u64 = 44;

#[derive(Clone, Copy)]
struct ClockRun {
    id: u64,
    clock_id: u32,
    precision: u64,
    now: u64,
    count: u64,
}
impl ClockRun {
    fn folds(self, id: u64, clock_id: u32, precision: u64, now: u64, step: u64) -> bool {
        self.clock_id == clock_id
            && self.precision == precision
            && self.count < 1 << 28
            && self.id.checked_add(self.count) == Some(id)
            && self
                .count
                .checked_mul(step)
                .and_then(|elapsed| self.now.checked_add(elapsed))
                == Some(now)
    }
}

struct BinaryInput {
    bytes: u64,
    clock: Option<(u32, u64)>,
}

fn binary_input(
    op: &str,
    fields: &Fields,
    raw_bytes: usize,
) -> Result<(Option<Operation>, BinaryInput)> {
    let mut clock = None;
    let mut operation = None;
    let bytes = match op {
        "clock_time_get" => {
            require_field_count(fields, 2)?;
            clock = Some((field(fields, "clock_id")?, field(fields, "precision")?));
            12
        }
        "poll_oneoff" => binary_poll_bytes(fields)?,
        "sched_yield" => {
            require_fields(fields, &[])?;
            0
        }
        _ => {
            let decoded = decode_operation(op, fields)?;
            let bytes = match &decoded {
                Operation::PrestatGet(_)
                | Operation::FdstatGet(_)
                | Operation::FilestatGet(_)
                | Operation::Close(_)
                | Operation::Sync(_) => 4,
                Operation::PrestatDirName { .. } => 8,
                Operation::SetFlags { .. } => 6,
                Operation::Read { offset, .. } => 8 + u64::from(offset.is_some()) * 8,
                Operation::Write { data, offset, .. } => {
                    4 + binary_bytes(data)? + u64::from(offset.is_some()) * 8
                }
                Operation::SetSize { .. } => 12,
                Operation::Seek { .. } => 13,
                Operation::Readdir { .. } => 16,
                Operation::Open(input) => 28 + binary_bytes(&input.path)?,
                Operation::CreateDirectory { path, .. }
                | Operation::RemoveDirectory { path, .. }
                | Operation::UnlinkFile { path, .. } => 4 + binary_bytes(path)?,
                Operation::PathStat { path, .. } => 8 + binary_bytes(path)?,
                Operation::Rename { path, new_path, .. } => {
                    8 + binary_bytes(path)? + binary_bytes(new_path)?
                }
                Operation::Unsupported(_) => binary_unsupported_input(op, fields, raw_bytes)?,
            };
            operation = Some(decoded);
            bytes
        }
    };
    Ok((operation, BinaryInput { bytes, clock }))
}

fn binary_bytes(value: &[u8]) -> Result<u64> {
    if value.len() > BUFFER_LIMIT as usize {
        return Err("capacity:buffer-bytes".into());
    }
    Ok(4 + value.len() as u64)
}

fn binary_unsupported_input(op: &str, fields: &Fields, raw_bytes: usize) -> Result<u64> {
    match op {
        "path_readlink" => {
            require_field_count(fields, 3)?;
            let _: u32 = field(fields, "fd")?;
            let path = binary_bytes(&bytes_field(fields, "path_base64")?)?;
            let _: u32 = field(fields, "length")?;
            Ok(8 + path)
        }
        "path_symlink" => {
            require_field_count(fields, 3)?;
            let old_path = binary_bytes(&bytes_field(fields, "old_path_base64")?)?;
            let _: u32 = field(fields, "fd")?;
            Ok(old_path + 4 + binary_bytes(&bytes_field(fields, "path_base64")?)?)
        }
        "sock_accept" => {
            require_field_count(fields, 2)?;
            let _: u32 = field(fields, "fd")?;
            let _: u16 = field(fields, "flags")?;
            Ok(6)
        }
        "sock_shutdown" => {
            require_field_count(fields, 2)?;
            let _: u32 = field(fields, "fd")?;
            let _: u8 = field(fields, "how")?;
            Ok(5)
        }
        "args_sizes_get" | "args_get" | "environ_sizes_get" | "environ_get" => {
            require_fields(fields, &[])?;
            Ok(0)
        }
        "random_get" => {
            require_field_count(fields, 1)?;
            let _: u32 = field(fields, "length")?;
            Ok(12)
        }
        _ => {
            if op.len() > 64 << 10 {
                return Err("capacity:transcript-operation".into());
            }
            Ok(8 + op.len() as u64 + raw_bytes as u64)
        }
    }
}

fn binary_poll_bytes(fields: &Fields) -> Result<u64> {
    require_field_count(fields, 1)?;
    let subscriptions: Vec<Box<RawValue>> = field(fields, "subscriptions")?;
    if subscriptions.len() > 4096 {
        return Err("capacity:pending-events".into());
    }
    let mut bytes = 4;
    for subscription in subscriptions {
        let fields = object_fields(subscription.get())
            .map_err(|_| "invalid:operation JSON types".to_string())?;
        let kind: u8 = match fields.get("type").or_else(|| {
            fields
                .iter()
                .find(|(name, _)| name.eq_ignore_ascii_case("type"))
                .map(|(_, raw)| raw)
        }) {
            None => 0,
            Some(raw) if raw.get() == "null" => 0,
            Some(raw) => serde_json::from_str(raw.get())
                .map_err(|_| "invalid:subscription type".to_string())?,
        };
        if kind > 2 {
            return Err("invalid:subscription type".into());
        }
        require_field_count(&fields, if kind == 0 { 6 } else { 3 })?;
        let _: u64 = field(&fields, "userdata")?;
        let _: u8 = field(&fields, "type")?;
        if kind == 0 {
            let _: u32 = field(&fields, "clock_id")?;
            let _: u64 = field(&fields, "timeout")?;
            let _: u64 = field(&fields, "precision")?;
            let _: u16 = field(&fields, "flags")?;
            bytes += 31;
        } else {
            let _: u32 = field(&fields, "fd")?;
            bytes += 13;
        }
    }
    Ok(bytes)
}

fn binary_output_reservation(op: &str, fields: &Fields) -> Result<u64> {
    Ok(match op {
        "clock_time_get" => 4 + "CPU clocks outside stock WASI profile".len() as u64,
        "fd_read" | "fd_pread" | "fd_prestat_dir_name" | "fd_readdir" | "path_readlink" => {
            4 + field::<u32>(fields, "length")? as u64
        }
        "fd_write" | "fd_pwrite" | "fd_prestat_get" | "path_open" => 4,
        "fd_filestat_get" | "path_filestat_get" => 57,
        "fd_fdstat_get" => 19,
        "fd_seek" => 8,
        "poll_oneoff" => {
            let subscriptions: Vec<Box<RawValue>> = field(fields, "subscriptions")?;
            (4 + subscriptions.len() as u64 * 23)
                .max(4 + "CPU poll clock outside stock WASI profile".len() as u64)
        }
        "fd_close"
        | "fd_sync"
        | "fd_fdstat_set_flags"
        | "fd_filestat_set_size"
        | "path_create_directory"
        | "path_remove_directory"
        | "path_unlink_file"
        | "sched_yield"
        | "args_sizes_get"
        | "args_get"
        | "environ_sizes_get"
        | "environ_get"
        | "random_get" => 0,
        _ => 4 + ("operation outside stock WASI profile: ".len() + op.len()) as u64,
    })
}

fn binary_reply_bytes(op: &str, reply: &Fields) -> Result<u64> {
    if reply.contains_key("unsupported") {
        let message: String = field(reply, "message")?;
        if message.len() > 64 << 10 {
            return Err("invalid:transcript message".into());
        }
        return Ok(4 + message.len() as u64);
    }
    if field::<u16>(reply, "errno")? != 0 {
        return Ok(0);
    }
    Ok(match op {
        "fd_read" | "fd_pread" | "fd_prestat_dir_name" | "fd_readdir" | "path_readlink" => {
            let output = object_fields(reply["output"].get())?;
            binary_bytes(&bytes_field(&output, "data_base64")?)?
        }
        "poll_oneoff" => {
            let output = object_fields(reply["output"].get())?;
            4 + field::<Vec<Box<RawValue>>>(&output, "events")?.len() as u64 * 23
        }
        _ => binary_output_reservation(op, &Fields::new())?,
    })
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct SessionRequest {
    model: String,
    image: Box<RawValue>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ImageRequest {
    entries: Vec<EntryRequest>,
    writable_directories: Vec<String>,
    preopens: Vec<String>,
    working_directory: String,
    #[serde(deserialize_with = "required_nullable_base64")]
    stdin_base64: Option<String>,
    limits: LimitRequest,
    epoch_nanos: u64,
    read_step_nanos: u64,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct EntryRequest {
    path_base64: String,
    directory: bool,
    readonly: bool,
    #[serde(deserialize_with = "required_nullable_base64")]
    data_base64: Option<String>,
}

fn required_nullable_base64<'de, D: serde::Deserializer<'de>>(
    deserializer: D,
) -> std::result::Result<Option<String>, D::Error> {
    Option::deserialize(deserializer)
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct LimitRequest {
    files: u64,
    descriptors: u64,
    filesystem_bytes: u64,
    output_bytes: u64,
    calls: u64,
    transcript_bytes: u64,
    pending_events: u64,
}

struct Session {
    namespace: Namespace,
    limits: LimitRequest,
    next_call: u64,
    transcript_bytes: u64,
    clock_run: Option<ClockRun>,
    now: u64,
    epoch: u64,
    read_step: u64,
}

impl Session {
    fn new(record: &str) -> Result<Self> {
        check_json(record, 0)?;
        let request: SessionRequest =
            serde_json::from_str(record).map_err(|_| "invalid:conformance session".to_string())?;
        if request.model != "namespace" {
            return Err("invalid:conformance model".into());
        }
        let request: ImageRequest = serde_json::from_str(request.image.get())
            .map_err(|_| "invalid:namespace image".to_string())?;
        let limits = request.limits;
        if limits.files == 0
            || limits.files > 1 << 20
            || limits.descriptors < 5
            || limits.descriptors > 1 << 20
            || limits.filesystem_bytes == 0
            || limits.filesystem_bytes > 1 << 30
            || limits.output_bytes == 0
            || limits.output_bytes > 1 << 30
            || limits.calls == 0
            || limits.calls > 1 << 28
            || limits.transcript_bytes == 0
            || limits.transcript_bytes > 1 << 30
            || limits.pending_events == 0
            || limits.pending_events > 4096
        {
            return Err("invalid:limits".into());
        }
        if request.epoch_nanos > i64::MAX as u64 || request.read_step_nanos > 1_000_000_000 {
            return Err("invalid:working directory or clock policy".into());
        }
        let decode = |value: &str| {
            decode_base64(value).map_err(|_| "invalid:namespace image bytes".to_string())
        };
        let entries = request
            .entries
            .into_iter()
            .map(|entry| {
                Ok(ImageEntry {
                    path: decode(&entry.path_base64)?,
                    directory: entry.directory,
                    readonly: entry.readonly,
                    data: decode(entry.data_base64.as_deref().unwrap_or(""))?,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let image = Image {
            entries,
            writable_directories: request
                .writable_directories
                .iter()
                .map(|name| decode(name))
                .collect::<Result<_>>()?,
            preopens: request
                .preopens
                .iter()
                .map(|name| decode(name))
                .collect::<Result<_>>()?,
            working_directory: decode(&request.working_directory)?,
            stdin: decode(request.stdin_base64.as_deref().unwrap_or(""))?,
            limits: Limits {
                files: limits.files,
                descriptors: limits.descriptors,
                filesystem_bytes: limits.filesystem_bytes,
                output_bytes: limits.output_bytes,
            },
            epoch_nanos: request.epoch_nanos,
        };
        Ok(Self {
            namespace: Namespace::new(image).map_err(boundary_name)?,
            limits,
            next_call: 1,
            transcript_bytes: 0,
            clock_run: None,
            now: u64::from(request.read_step_nanos == 0),
            epoch: request.epoch_nanos,
            read_step: request.read_step_nanos,
        })
    }

    fn digest(&self) -> String {
        format!(
            r#"{{"digest":"{:x}"}}"#,
            Sha256::digest(self.namespace.canonical_state())
        )
    }

    fn handle(&mut self, record: &str) -> String {
        self.handle_record(record)
            .unwrap_or_else(|error| boundary_record(&error))
    }

    fn handle_record(&mut self, record: &str) -> Result<String> {
        let fields = object_fields(record)?;
        if fields.len() != 2 || !fields.contains_key("op") || !fields.contains_key("input") {
            return Err("invalid:conformance record".into());
        }
        let op: String = field(&fields, "op")?;
        let input = fields["input"].get();
        if op == "digest" {
            check_json(input, 0)?;
            require_fields(&object_fields(input)?, &[])?;
            return Ok(self.digest());
        }
        if self.next_call > self.limits.calls {
            return Err("capacity:callbacks".into());
        }
        let call = format!(
            r#"{{"type":"call","id":{},"op":{},"input":{}}}"#,
            self.next_call,
            go_json(&op),
            compact_json(input)
        );
        if call.len() + 1 > FRAME_LIMIT {
            return Err("capacity:callback-frame".into());
        }
        if input.len() > FRAME_LIMIT {
            return Err("capacity:input-frame".into());
        }
        check_json(input, 0)?;
        let input_fields = object_fields(input).map_err(|error| {
            if error == "invalid:operation fields" {
                "invalid:operation input".into()
            } else {
                error
            }
        })?;
        let mut length = 0;
        if let Some(raw) = input_fields.get("length") {
            if raw.get() != "null" {
                length = serde_json::from_str::<u32>(raw.get())
                    .map_err(|_| "invalid:operation length".to_string())?;
            }
            if length > BUFFER_LIMIT {
                return Err("capacity:buffer-bytes".into());
            }
        }
        let reservation = 1024
            + u64::from(length) * 4 / 3
            + 4
            + if op == "poll_oneoff" {
                input.len() as u64 * 4
            } else {
                0
            };
        if reservation + 1 > FRAME_LIMIT as u64 {
            return Err("capacity:reply-frame".into());
        }
        let header = u64::from(self.transcript_bytes == 0) * BINARY_HEADER_BYTES;
        let minimum = if op == "clock_time_get" && self.clock_run.is_some() {
            0
        } else {
            BINARY_RECORD_BYTES
        };
        if self.transcript_bytes + header + minimum > self.limits.transcript_bytes {
            return Err("capacity:transcript-bytes".into());
        }
        let (operation, binary) = binary_input(&op, &input_fields, input.len())?;
        let folded = binary.clock.is_some_and(|(clock_id, precision)| {
            self.clock_run.is_some_and(|run| {
                run.folds(
                    self.next_call,
                    clock_id,
                    precision,
                    self.now,
                    self.read_step,
                )
            })
        });
        let binary_reservation = if binary.clock.is_some_and(|(clock_id, _)| clock_id <= 1) {
            if folded {
                0
            } else {
                header + BINARY_CLOCK_RUN_BYTES
            }
        } else {
            header
                + BINARY_RECORD_BYTES
                + binary.bytes
                + binary_output_reservation(&op, &input_fields)?
        };
        if self.transcript_bytes + binary_reservation > self.limits.transcript_bytes {
            return Err("capacity:transcript-bytes".into());
        }
        let clock_now = self.now;
        if binary.clock.is_none() {
            self.clock_run = None;
        }
        let reply = if op == "clock_time_get" || op == "poll_oneoff" {
            self.clock_call(&op, &input_fields)?
        } else if op == "sched_yield" {
            require_fields(&input_fields, &[])?;
            encode_reply(0, "{}", None)
        } else {
            let reply = self
                .namespace
                .execute(operation.unwrap(), self.now)
                .map_err(boundary_name)?;
            let output = if reply.errno == 0 {
                reply
                    .output
                    .as_ref()
                    .map(encode_output)
                    .unwrap_or_else(|| "{}".into())
            } else {
                "{}".into()
            };
            encode_reply(reply.errno, &output, reply.unsupported.as_deref())
        };
        let full_reply = format!(
            r#"{{"type":"reply","id":{},{}"#,
            self.next_call,
            &reply[1..]
        );
        if full_reply.len() as u64 > reservation {
            return Err("invalid:internal transcript reservation".into());
        }
        let reply_fields = object_fields(&reply)?;
        let unsupported = reply_fields.contains_key("unsupported");
        if let Some((clock_id, precision)) = binary.clock.filter(|_| !unsupported) {
            if folded {
                self.clock_run.as_mut().unwrap().count += 1;
            } else {
                self.clock_run = Some(ClockRun {
                    id: self.next_call,
                    clock_id,
                    precision,
                    now: clock_now,
                    count: 1,
                });
                self.transcript_bytes += header + BINARY_CLOCK_RUN_BYTES;
            }
        } else {
            self.clock_run = None;
            let bytes =
                BINARY_RECORD_BYTES + binary.bytes + binary_reply_bytes(&op, &reply_fields)?;
            if bytes > FRAME_LIMIT as u64 {
                return Err("capacity:transcript-record".into());
            }
            self.transcript_bytes += header + bytes;
        }
        self.next_call += 1;
        Ok(reply)
    }

    fn clock_call(&mut self, op: &str, fields: &Fields) -> Result<String> {
        if op == "clock_time_get" {
            require_fields(fields, &["clock_id", "precision"])?;
            let clock_id: u32 = field(fields, "clock_id")?;
            let _: u64 = field(fields, "precision")?;
            if clock_id > 1 {
                return Ok(encode_reply(
                    0,
                    "{}",
                    Some("CPU clocks outside stock WASI profile"),
                ));
            }
            let bound = i64::MAX as u64 - self.epoch;
            if self.now > bound || self.read_step > bound - self.now {
                return Err("capacity:clock-nanoseconds".into());
            }
            let timestamp = self.now + if clock_id == 0 { self.epoch } else { 0 };
            self.now += self.read_step;
            return Ok(encode_reply(
                0,
                &format!(r#"{{"timestamp":{timestamp}}}"#),
                None,
            ));
        }
        require_fields(fields, &["subscriptions"])?;
        let subscriptions: Vec<Box<RawValue>> = field(fields, "subscriptions")?;
        if subscriptions.is_empty() {
            return Ok(encode_reply(namespace::INVAL, "{}", None));
        }
        if subscriptions.len() as u64 > self.limits.pending_events {
            return Err("capacity:pending-events".into());
        }
        let mut events = Vec::with_capacity(subscriptions.len());
        let mut earliest = u64::MAX;
        for raw in subscriptions {
            let sub = object_fields(raw.get()).map_err(|_| "invalid:subscription".to_string())?;
            let kind: u8 = sub
                .get("type")
                .map(|raw| serde_json::from_str(if raw.get() == "null" { "0" } else { raw.get() }))
                .transpose()
                .map_err(|_| "invalid:subscription type".to_string())?
                .ok_or_else(|| "invalid:subscription type".to_string())?;
            require_fields(
                &sub,
                if kind == 0 {
                    &[
                        "userdata",
                        "type",
                        "clock_id",
                        "timeout",
                        "precision",
                        "flags",
                    ]
                } else {
                    &["userdata", "type", "fd"]
                },
            )?;
            let userdata: u64 = field(&sub, "userdata")?;
            let mut at = self.now;
            let mut errno = 0;
            let mut nbytes = 0;
            let mut flags = 0;
            match kind {
                0 => {
                    let clock_id: u32 = field(&sub, "clock_id")?;
                    let timeout: u64 = field(&sub, "timeout")?;
                    let _: u64 = field(&sub, "precision")?;
                    let clock_flags: u16 = field(&sub, "flags")?;
                    if clock_id > 1 {
                        return Ok(encode_reply(
                            0,
                            "{}",
                            Some("CPU poll clock outside stock WASI profile"),
                        ));
                    }
                    if clock_flags & !1 != 0 {
                        return Ok(encode_reply(namespace::INVAL, "{}", None));
                    }
                    let bound = i64::MAX as u64 - self.epoch;
                    if clock_flags & 1 != 0 {
                        at = if clock_id == 0 {
                            timeout.saturating_sub(self.epoch)
                        } else {
                            timeout
                        };
                        at = at.max(self.now);
                    } else {
                        if self.now > bound || timeout > bound - self.now {
                            return Ok(encode_reply(namespace::INVAL, "{}", None));
                        }
                        at += timeout;
                    }
                    if at > bound {
                        return Ok(encode_reply(namespace::INVAL, "{}", None));
                    }
                }
                1 | 2 => {
                    let readiness = self.namespace.fd_readiness(field(&sub, "fd")?, kind == 2);
                    errno = readiness.errno;
                    nbytes = readiness.nbytes;
                    flags = readiness.flags;
                }
                _ => {
                    let _: u32 = field(&sub, "fd")?;
                    return Ok(encode_reply(namespace::INVAL, "{}", None));
                }
            }
            earliest = earliest.min(at);
            events.push((at, format!(r#"{{"userdata":{userdata},"errno":{errno},"type":{kind},"nbytes":{nbytes},"flags":{flags}}}"#)));
        }
        self.now = earliest;
        let ready = events
            .into_iter()
            .filter(|(at, _)| *at <= earliest)
            .map(|(_, event)| event)
            .collect::<Vec<_>>()
            .join(",");
        Ok(encode_reply(0, &format!(r#"{{"events":[{ready}]}}"#), None))
    }
}

fn decode_operation(op: &str, fields: &Fields) -> Result<Operation> {
    let keys: &[&str] = match op {
        "fd_prestat_get" | "fd_fdstat_get" | "fd_filestat_get" | "fd_close" | "fd_sync" => &["fd"],
        "fd_prestat_dir_name" | "fd_read" => &["fd", "length"],
        "fd_pread" => &["fd", "length", "offset"],
        "fd_fdstat_set_flags" => &["fd", "flags"],
        "fd_write" => &["fd", "data_base64"],
        "fd_pwrite" => &["fd", "data_base64", "offset"],
        "fd_filestat_set_size" => &["fd", "size"],
        "fd_seek" => &["fd", "offset", "whence"],
        "fd_readdir" => &["fd", "length", "cookie"],
        "path_open" => &[
            "fd",
            "dirflags",
            "path_base64",
            "oflags",
            "rights_base",
            "rights_inheriting",
            "fdflags",
        ],
        "path_create_directory" | "path_remove_directory" | "path_unlink_file" => {
            &["fd", "path_base64"]
        }
        "path_filestat_get" => &["fd", "flags", "path_base64"],
        "path_rename" => &["fd", "path_base64", "new_fd", "new_path_base64"],
        _ => return Ok(Operation::Unsupported(op.to_string())),
    };
    require_field_count(fields, keys.len())?;
    let fd = field(fields, "fd")?;
    Ok(match op {
        "fd_prestat_get" => Operation::PrestatGet(fd),
        "fd_fdstat_get" => Operation::FdstatGet(fd),
        "fd_filestat_get" => Operation::FilestatGet(fd),
        "fd_close" => Operation::Close(fd),
        "fd_sync" => Operation::Sync(fd),
        "fd_prestat_dir_name" => Operation::PrestatDirName {
            fd,
            length: field(fields, "length")?,
        },
        "fd_fdstat_set_flags" => Operation::SetFlags {
            fd,
            flags: field(fields, "flags")?,
        },
        "fd_read" | "fd_pread" => Operation::Read {
            fd,
            length: field(fields, "length")?,
            offset: if op == "fd_pread" {
                Some(field(fields, "offset")?)
            } else {
                None
            },
        },
        "fd_write" | "fd_pwrite" => Operation::Write {
            fd,
            data: binary_bytes_field(fields, "data_base64")?,
            offset: if op == "fd_pwrite" {
                Some(field(fields, "offset")?)
            } else {
                None
            },
        },
        "fd_filestat_set_size" => Operation::SetSize {
            fd,
            size: field(fields, "size")?,
        },
        "fd_seek" => Operation::Seek {
            fd,
            offset: field(fields, "offset")?,
            whence: field(fields, "whence")?,
        },
        "fd_readdir" => Operation::Readdir {
            fd,
            length: field(fields, "length")?,
            cookie: field(fields, "cookie")?,
        },
        "path_open" => Operation::Open(Open {
            fd,
            dirflags: field(fields, "dirflags")?,
            path: binary_bytes_field(fields, "path_base64")?,
            oflags: field(fields, "oflags")?,
            base: field(fields, "rights_base")?,
            inheriting: field(fields, "rights_inheriting")?,
            flags: field(fields, "fdflags")?,
        }),
        "path_create_directory" => Operation::CreateDirectory {
            fd,
            path: binary_bytes_field(fields, "path_base64")?,
        },
        "path_remove_directory" => Operation::RemoveDirectory {
            fd,
            path: binary_bytes_field(fields, "path_base64")?,
        },
        "path_unlink_file" => Operation::UnlinkFile {
            fd,
            path: binary_bytes_field(fields, "path_base64")?,
        },
        "path_filestat_get" => Operation::PathStat {
            fd,
            flags: field(fields, "flags")?,
            path: binary_bytes_field(fields, "path_base64")?,
        },
        "path_rename" => Operation::Rename {
            fd,
            path: binary_bytes_field(fields, "path_base64")?,
            new_fd: field(fields, "new_fd")?,
            new_path: binary_bytes_field(fields, "new_path_base64")?,
        },
        _ => unreachable!(),
    })
}

fn field<T: serde::de::DeserializeOwned>(fields: &Fields, name: &str) -> Result<T> {
    let value = fields
        .get(name)
        .filter(|raw| raw.get() != "null")
        .ok_or_else(|| format!("invalid:missing operation field {name}"))?;
    serde_json::from_str(value.get()).map_err(|_| "invalid:operation JSON types".to_string())
}

fn binary_bytes_field(fields: &Fields, name: &str) -> Result<Vec<u8>> {
    let data = bytes_field(fields, name)?;
    binary_bytes(&data)?;
    Ok(data)
}

fn bytes_field(fields: &Fields, name: &str) -> Result<Vec<u8>> {
    #[derive(Deserialize)]
    #[serde(untagged)]
    enum Bytes {
        Encoded(String),
        Array(Vec<Option<u8>>),
    }
    match field(fields, name)? {
        Bytes::Encoded(value) => {
            decode_base64(&value).map_err(|_| "invalid:operation JSON types".into())
        }
        Bytes::Array(value) => Ok(value.into_iter().map(Option::unwrap_or_default).collect()),
    }
}

fn decode_base64(value: &str) -> std::result::Result<Vec<u8>, base64::DecodeError> {
    let decoder = base64::engine::general_purpose::GeneralPurpose::new(
        &base64::alphabet::STANDARD,
        base64::engine::general_purpose::GeneralPurposeConfig::new()
            .with_decode_allow_trailing_bits(true),
    );
    decoder.decode(value.replace(['\r', '\n'], ""))
}

fn require_fields(fields: &Fields, keys: &[&str]) -> Result<()> {
    require_field_count(fields, keys.len())?;
    for key in keys {
        if fields.get(*key).is_none_or(|value| value.get() == "null") {
            return Err(format!("invalid:missing operation field {key}"));
        }
    }
    Ok(())
}

fn require_field_count(fields: &Fields, count: usize) -> Result<()> {
    if fields.len() != count {
        return Err("invalid:operation fields".into());
    }
    Ok(())
}

fn object_fields(value: &str) -> Result<Fields> {
    struct Object;
    impl<'de> serde::de::Visitor<'de> for Object {
        type Value = Fields;
        fn expecting(&self, formatter: &mut std::fmt::Formatter) -> std::fmt::Result {
            formatter.write_str("an object")
        }
        fn visit_map<M: MapAccess<'de>>(self, mut map: M) -> std::result::Result<Fields, M::Error> {
            let mut fields = Fields::new();
            while let Some((key, value)) = map.next_entry::<String, Box<RawValue>>()? {
                if fields.insert(key, value).is_some() {
                    return Err(serde::de::Error::custom("duplicate JSON field"));
                }
            }
            Ok(fields)
        }
    }
    let mut decoder = serde_json::Deserializer::from_str(value);
    let fields = serde::Deserializer::deserialize_map(&mut decoder, Object).map_err(|error| {
        if error.to_string().starts_with("duplicate JSON field") {
            "invalid:duplicate JSON field"
        } else if error.is_syntax() || error.is_eof() {
            "invalid:malformed JSON"
        } else {
            "invalid:operation fields"
        }
        .to_string()
    })?;
    decoder
        .end()
        .map_err(|_| "invalid:trailing JSON".to_string())?;
    Ok(fields)
}

fn check_json(value: &str, depth: usize) -> Result<()> {
    let mut offset = 0;
    check_json_value(value, &mut offset, depth)?;
    json_whitespace(value, &mut offset);
    if offset != value.len() {
        return Err("invalid:trailing JSON".into());
    }
    Ok(())
}

fn json_whitespace(value: &str, offset: &mut usize) {
    while matches!(
        value.as_bytes().get(*offset),
        Some(b' ' | b'\t' | b'\r' | b'\n')
    ) {
        *offset += 1;
    }
}

fn json_token<T: serde::de::DeserializeOwned>(value: &str, offset: &mut usize) -> Result<T> {
    let mut tokens = serde_json::Deserializer::from_str(&value[*offset..]).into_iter::<T>();
    let token = tokens
        .next()
        .ok_or("invalid:malformed JSON")?
        .map_err(|_| "invalid:malformed JSON".to_string())?;
    *offset += tokens.byte_offset();
    Ok(token)
}

fn check_json_value(value: &str, offset: &mut usize, depth: usize) -> Result<()> {
    if depth > 32 {
        return Err("invalid:JSON nesting".into());
    }
    json_whitespace(value, offset);
    match value.as_bytes().get(*offset) {
        Some(b'{' | b'[') => {
            let object = value.as_bytes()[*offset] == b'{';
            let close = if object { b'}' } else { b']' };
            *offset += 1;
            json_whitespace(value, offset);
            if value.as_bytes().get(*offset) == Some(&close) {
                *offset += 1;
                return Ok(());
            }
            let mut names = std::collections::BTreeSet::new();
            loop {
                if object {
                    json_whitespace(value, offset);
                    if value.as_bytes().get(*offset) != Some(&b'"') {
                        return Err("invalid:malformed JSON".into());
                    }
                    let name: String = json_token(value, offset)?;
                    if !names.insert(name) {
                        return Err("invalid:duplicate JSON field".into());
                    }
                    json_whitespace(value, offset);
                    if value.as_bytes().get(*offset) != Some(&b':') {
                        return Err("invalid:malformed JSON".into());
                    }
                    *offset += 1;
                }
                check_json_value(value, offset, depth + 1)?;
                json_whitespace(value, offset);
                match value.as_bytes().get(*offset) {
                    Some(byte) if *byte == close => {
                        *offset += 1;
                        break;
                    }
                    Some(b',') => *offset += 1,
                    _ => return Err("invalid:malformed JSON".into()),
                }
            }
        }
        Some(b'"') => {
            let _: String = json_token(value, offset)?;
        }
        _ => {
            let _: Box<RawValue> = json_token(value, offset)?;
        }
    }
    Ok(())
}

fn go_json<T: serde::Serialize>(value: &T) -> String {
    serde_json::to_string(value)
        .expect("in-memory conformance JSON")
        .replace('&', "\\u0026")
        .replace('<', "\\u003c")
        .replace('>', "\\u003e")
        .replace('\u{2028}', "\\u2028")
        .replace('\u{2029}', "\\u2029")
}

fn compact_json(value: &str) -> String {
    let mut output = String::with_capacity(value.len());
    let mut quoted = false;
    let mut escaped = false;
    for character in value.chars() {
        if quoted || !character.is_ascii_whitespace() {
            match character {
                '&' => output.push_str("\\u0026"),
                '<' => output.push_str("\\u003c"),
                '>' => output.push_str("\\u003e"),
                '\u{2028}' => output.push_str("\\u2028"),
                '\u{2029}' => output.push_str("\\u2029"),
                _ => output.push(character),
            }
        }
        if character == '"' && !escaped {
            quoted = !quoted;
        }
        escaped = quoted && character == '\\' && !escaped;
    }
    output
}

fn boundary_name(boundary: namespace::Boundary) -> String {
    format!("{}:{}", boundary.kind, boundary.dimension)
}
fn boundary_record(boundary: &str) -> String {
    format!(r#"{{"boundary":{}}}"#, go_json(&boundary))
}
fn encode_reply(errno: u16, output: &str, unsupported: Option<&str>) -> String {
    let extra = unsupported
        .map(|message| format!(r#","unsupported":true,"message":{}"#, go_json(&message)))
        .unwrap_or_default();
    format!(r#"{{"errno":{errno},"output":{output}{extra}}}"#)
}

fn encode_output(output: &Output) -> String {
    match output {
        Output::Empty => "{}".into(),
        Output::Prestat(length) => format!(r#"{{"name_length":{length}}}"#),
        Output::Fdstat(stat) => format!(
            r#"{{"filetype":{},"flags":{},"rights_base":{},"rights_inheriting":{}}}"#,
            stat.filetype, stat.flags, stat.base, stat.inheriting
        ),
        Output::Filestat(stat) => format!(
            r#"{{"dev":{},"ino":{},"filetype":{},"nlink":{},"size":{},"atim":{},"mtim":{},"ctim":{}}}"#,
            stat.dev,
            stat.ino,
            stat.filetype,
            stat.nlink,
            stat.size,
            stat.atim,
            stat.mtim,
            stat.ctim
        ),
        Output::Bytes(bytes) => format!(r#"{{"data_base64":"{}"}}"#, STANDARD.encode(bytes)),
        Output::Fd(fd) => format!(r#"{{"fd":{fd}}}"#),
        Output::Written(written) => format!(r#"{{"written":{written}}}"#),
        Output::Offset(offset) => format!(r#"{{"offset":{offset}}}"#),
    }
}

#[cfg(feature = "conformance")]
pub fn execute() -> io::Result<()> {
    run(&mut io::stdin().lock(), &mut io::stdout().lock())
}

fn read_record(reader: &mut impl BufRead) -> io::Result<Option<Result<String>>> {
    let mut record = Vec::new();
    let length = reader
        .take(FRAME_LIMIT as u64 + 1)
        .read_until(b'\n', &mut record)?;
    if length == 0 {
        return Ok(None);
    }
    if length > FRAME_LIMIT {
        return Ok(Some(Err("capacity:conformance-frame".into())));
    }
    if record.pop() != Some(b'\n') {
        return Ok(Some(Err("invalid:conformance framing".into())));
    }
    Ok(Some(
        String::from_utf8(record).map_err(|_| "invalid:malformed JSON".into()),
    ))
}

fn run(reader: &mut impl BufRead, writer: &mut impl Write) -> io::Result<()> {
    let Some(record) = read_record(reader)? else {
        return Ok(());
    };
    if let Ok(record) = &record
        && let Ok(fields) = object_fields(record)
        && field::<String>(&fields, "model").ok().as_deref() == Some(VOLUME_MODEL)
    {
        writeln!(writer, "{VOLUME_UNSUPPORTED}")?;
        return writer.flush();
    }
    let mut session = match record.and_then(|record| Session::new(&record)) {
        Ok(session) => {
            writeln!(writer, "{{}}")?;
            writer.flush()?;
            session
        }
        Err(error) => {
            writeln!(writer, "{}", boundary_record(&error))?;
            return writer.flush();
        }
    };
    while let Some(record) = read_record(reader)? {
        match record {
            Ok(record) => writeln!(writer, "{}", session.handle(&record))?,
            Err(error) => {
                writeln!(writer, "{}", boundary_record(&error))?;
                return writer.flush();
            }
        }
        writer.flush()?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn volume_selector_is_explicitly_unsupported() {
        let input = b"{\"model\":\"internal/model/volume\",\"image\":{},\"storage\":{}}\n";
        let mut output = Vec::new();
        run(&mut &input[..], &mut output).unwrap();
        assert_eq!(
            String::from_utf8(output).unwrap(),
            format!("{VOLUME_UNSUPPORTED}\n")
        );
    }

    fn request(transcript_bytes: u64, calls: u64) -> String {
        format!(
            r#"{{"model":"namespace","image":{{"entries":[{{"path_base64":"L3RtcA==","directory":true,"readonly":false,"data_base64":""}},{{"path_base64":"L3dvcmtzcGFjZQ==","directory":true,"readonly":false,"data_base64":""}}],"writable_directories":["L3dvcmtzcGFjZQ==","L3RtcA=="],"preopens":["L3RtcA==","L3dvcmtzcGFjZQ=="],"working_directory":"L3dvcmtzcGFjZQ==","stdin_base64":"","limits":{{"files":100,"descriptors":64,"filesystem_bytes":8388608,"output_bytes":1048576,"calls":{calls},"transcript_bytes":{transcript_bytes},"pending_events":4096}},"epoch_nanos":946684800000000000,"read_step_nanos":1000}}}}"#
        )
    }

    fn session(transcript_bytes: u64, calls: u64) -> Session {
        Session::new(&request(transcript_bytes, calls)).unwrap()
    }

    #[test]
    fn namespace_conformance_image_requires_nullable_byte_fields() {
        for removed in ["stdin_base64", "data_base64"] {
            let mut request: serde_json::Value =
                serde_json::from_str(&request(1 << 20, 100)).unwrap();
            let image = request["image"].as_object_mut().unwrap();
            if removed == "data_base64" {
                image["entries"][0].as_object_mut().unwrap().remove(removed);
            } else {
                image.remove(removed);
            }
            assert_eq!(
                Session::new(&go_json(&request)).err(),
                Some("invalid:namespace image".into())
            );
        }
    }

    #[test]
    fn namespace_conformance_binary_admission_validates_input_before_full_reservation() {
        let mut session = session(100, 100);
        assert_eq!(
            session.handle(r#"{"op":"fd_read","input":{"fd":0}}"#),
            r#"{"boundary":"invalid:operation fields"}"#
        );
        assert_eq!(session.transcript_bytes, 0);
    }

    #[test]
    fn namespace_conformance_binary_clock_reads_fold_without_new_reservation() {
        let mut session = session(52, 100);
        let clock = r#"{"op":"clock_time_get","input":{"clock_id":1,"precision":0}}"#;
        assert_eq!(
            session.handle(clock),
            r#"{"errno":0,"output":{"timestamp":0}}"#
        );
        assert_eq!(
            session.handle(clock),
            r#"{"errno":0,"output":{"timestamp":1000}}"#
        );
        assert_eq!(session.transcript_bytes, 52);
        assert_eq!(
            session.handle(r#"{"op":"clock_time_get","input":{"clock_id":1,"precision":1}}"#),
            r#"{"boundary":"capacity:transcript-bytes"}"#
        );
        assert_eq!(
            session.handle(r#"{"op":"clock_time_get","input":{"clock_id":1}}"#),
            r#"{"boundary":"invalid:operation fields"}"#
        );
        assert_eq!(
            session.handle(clock),
            r#"{"errno":0,"output":{"timestamp":2000}}"#
        );
        assert_eq!(session.transcript_bytes, 52);
    }

    #[test]
    fn namespace_conformance_binary_fields_validate_in_schema_order() {
        let mut session = session(1 << 20, 100);
        assert_eq!(
            session.handle(r#"{"op":"fd_read","input":{"fd":"bad","length":null}}"#),
            r#"{"boundary":"invalid:operation JSON types"}"#
        );
        assert_eq!(session.handle(r#"{"op":"path_open","input":{"fd":3,"dirflags":"bad","path_base64":null,"oflags":0,"rights_base":0,"rights_inheriting":0,"fdflags":0}}"#), r#"{"boundary":"invalid:operation JSON types"}"#);
    }

    #[test]
    fn namespace_conformance_binary_transcript_counts_actual_records() {
        let mut session = session(48, 100);
        assert_eq!(
            session.handle(r#"{"op":"fd_close","input":{"fd":999}}"#),
            r#"{"errno":8,"output":{}}"#
        );
        assert_eq!(session.transcript_bytes, 28);
        assert_eq!(
            session.handle(r#"{"op":"fd_close","input":{"fd":999}}"#),
            r#"{"errno":8,"output":{}}"#
        );
        assert_eq!(session.transcript_bytes, 48);
    }

    #[test]
    fn namespace_conformance_reserves_before_mutating() {
        let mut session = session(1, 100);
        let before = session.digest();
        let reply = session
            .handle(r#"{"op":"path_create_directory","input":{"fd":3,"path_base64":"YQ=="}}"#);
        assert_eq!(reply, r#"{"boundary":"capacity:transcript-bytes"}"#);
        assert_eq!(session.digest(), before);
    }

    #[test]
    fn namespace_conformance_callback_limit_precedes_input_validation() {
        let mut session = session(1 << 20, 1);
        assert_eq!(
            session.handle(r#"{"op":"fd_close","input":{"fd":999}}"#),
            r#"{"errno":8,"output":{}}"#
        );
        assert_eq!(
            session.handle(r#"{"op":"fd_close","input":{"fd":null}}"#),
            r#"{"boundary":"capacity:callbacks"}"#
        );
    }

    #[test]
    fn namespace_conformance_strict_input_and_length_precedence() {
        for (input, boundary) in [
            (r#"{"fd":3,"fd":4}"#, "invalid:duplicate JSON field"),
            (r#"{"\ud800":0}"#, "invalid:malformed JSON"),
            (r#"{"\udfff":0}"#, "invalid:malformed JSON"),
            (r#"{"fd":3,"extra":{"\ud800":0}}"#, "invalid:malformed JSON"),
            (
                r#"{"\ud83d\ude00":0}"#,
                "invalid:missing operation field fd",
            ),
            (r#"{"fd":null}"#, "invalid:missing operation field fd"),
            (r#"{"fd":4294967296}"#, "invalid:operation JSON types"),
            (r#"{"fd":3,"extra":0}"#, "invalid:operation fields"),
            (r#"{"fd":3,"length":4194305}"#, "capacity:buffer-bytes"),
        ] {
            let mut session = session(1 << 20, 100);
            let before = session.digest();
            let reply = session.handle(&format!(r#"{{"op":"fd_close","input":{input}}}"#));
            assert_eq!(reply, format!(r#"{{"boundary":"{boundary}"}}"#), "{input}");
            assert_eq!(session.digest(), before);
        }
    }

    #[test]
    fn namespace_conformance_json_errors_follow_wire_token_order() {
        let deep = format!("{}0{}", "[".repeat(33), "]".repeat(33));
        let malformed_deep = format!("{}|{}", "[".repeat(33), "]".repeat(33));
        for (input, boundary) in [
            (
                r#"{"fd":0,"extra":{"\ud800":0},"fd":1}"#.to_string(),
                "invalid:malformed JSON",
            ),
            (
                r#"{"z":"\ud800","a":{"x":0,"x":1}}"#.to_string(),
                "invalid:malformed JSON",
            ),
            (
                format!(r#"{{"z":{deep},"a":"\ud800"}}"#),
                "invalid:JSON nesting",
            ),
            (
                format!(r#"{{"z":{malformed_deep}}}"#),
                "invalid:JSON nesting",
            ),
            (
                r#"{"fd":0,"fd":{"\ud800":0}}"#.to_string(),
                "invalid:duplicate JSON field",
            ),
            (
                r#"{"fd":0,"\u0066d":1}"#.to_string(),
                "invalid:duplicate JSON field",
            ),
            (
                r#"{"fd":0} {"\ud800":0}"#.to_string(),
                "invalid:trailing JSON",
            ),
        ] {
            assert_eq!(check_json(&input, 0), Err(boundary.into()), "{input}");
        }
        for input in [
            "1e400",
            "-1234567890123456789012345678901234567890",
            "[{},[]]",
        ] {
            assert_eq!(check_json(input, 0), Ok(()), "{input}");
        }
    }

    #[test]
    fn namespace_conformance_poll_returns_ready_fds_before_future_clock() {
        let mut session = session(1 << 20, 100);
        assert_eq!(
            session.handle(r#"{"op":"poll_oneoff","input":{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":50,"precision":0,"flags":0},{"userdata":2,"type":1,"fd":0}]}}"#),
            r#"{"errno":0,"output":{"events":[{"userdata":2,"errno":0,"type":1,"nbytes":0,"flags":1}]}}"#
        );
        assert_eq!(
            session.handle(r#"{"op":"clock_time_get","input":{"clock_id":1,"precision":0}}"#),
            r#"{"errno":0,"output":{"timestamp":0}}"#
        );
    }

    #[test]
    fn namespace_conformance_unsupported_reply_matches_go_bytes() {
        let mut session = session(1 << 20, 100);
        assert_eq!(
            session.handle(r#"{"op":"sock_accept","input":{"fd":3,"flags":0}}"#),
            r#"{"errno":0,"output":{},"unsupported":true,"message":"operation outside stock WASI profile: sock_accept"}"#
        );
    }

    #[test]
    fn namespace_conformance_corpus() {
        #[derive(Deserialize)]
        struct Corpus {
            name: String,
            session: Box<RawValue>,
            operations: Vec<CorpusOperation>,
            replies: Vec<Box<RawValue>>,
            digest: String,
        }
        #[derive(Deserialize)]
        struct CorpusOperation {
            op: String,
            input: Box<RawValue>,
            #[serde(default)]
            repeat_data: u32,
        }
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../wasi/testdata/namespace-conformance");
        let mut paths = std::fs::read_dir(&directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|value| value == "json"))
            .collect::<Vec<_>>();
        paths.sort();
        assert!(!paths.is_empty(), "missing namespace conformance corpus");
        for path in paths {
            let corpus: Corpus = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
            let mut session = Session::new(corpus.session.get()).unwrap();
            assert_eq!(
                corpus.operations.len(),
                corpus.replies.len(),
                "{}",
                corpus.name
            );
            for (index, (operation, expected)) in corpus
                .operations
                .into_iter()
                .zip(corpus.replies)
                .enumerate()
            {
                let mut input = compact_json(operation.input.get());
                if operation.repeat_data != 0 {
                    let mut fields = object_fields(&input).unwrap();
                    fields.insert(
                        "data_base64".into(),
                        RawValue::from_string(go_json(&STANDARD.encode(vec![
                            b'x';
                            operation.repeat_data
                                as usize
                        ])))
                        .unwrap(),
                    );
                    input = go_json(&fields);
                }
                let record = format!(r#"{{"op":{},"input":{input}}}"#, go_json(&operation.op));
                assert_eq!(
                    session.handle(&record),
                    compact_json(expected.get()),
                    "{} op {index}: {}",
                    corpus.name,
                    operation.op
                );
            }
            assert_eq!(
                session.digest(),
                format!(r#"{{"digest":"{}"}}"#, corpus.digest),
                "{} final digest",
                corpus.name
            );
        }
    }

    #[test]
    fn namespace_conformance_lf_session_refuses_unknown_model() {
        let mut output = Vec::new();
        run(
            &mut std::io::Cursor::new(b"{\"model\":\"other\",\"image\":{}}\n"),
            &mut output,
        )
        .unwrap();
        assert_eq!(output, b"{\"boundary\":\"invalid:conformance model\"}\n");
    }
}
