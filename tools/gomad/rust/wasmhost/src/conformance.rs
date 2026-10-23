use crate::namespace::{self, Image, ImageEntry, Limits, Namespace, Open, Operation, Output};
use crate::transcript::{self, Subscription};
use crate::{local_model, namespace::call as namespace_call, volume};
use base64::{Engine, engine::general_purpose::STANDARD};
use serde::{Deserialize, de::MapAccess};
use serde_json::value::RawValue;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::{self, BufRead, Read, Write};

const FRAME_LIMIT: usize = 8 << 20;
const BUFFER_LIMIT: u32 = 4 << 20;
const VOLUME_MODEL: &str = "internal/model/volume";
type Fields = BTreeMap<String, Box<RawValue>>;
type Result<T> = std::result::Result<T, String>;

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
                Operation::Readlink { .. }
                | Operation::Symlink { .. }
                | Operation::Unsupported(_) => binary_unsupported_input(op, fields, raw_bytes)?,
                #[cfg(not(feature = "conformance"))]
                Operation::SockAccept { .. } | Operation::SockShutdown { .. } => {
                    binary_unsupported_input(op, fields, raw_bytes)?
                }
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
            4 + u64::from(field::<u32>(fields, "length")?)
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
    #[serde(default)]
    binary_image: Option<BinaryImageRequest>,
    #[serde(default)]
    storage: Option<Box<RawValue>>,
}

/// Go's production namespace image (`GMWN` bytes) for the same namespace as
/// `image`. The driver decodes it through the helper's image decoder, requires
/// the same canonical state, and then runs the session on the decoded namespace.
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct BinaryImageRequest {
    data_base64: String,
    files: u64,
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
    volume: Option<VolumeSession>,
    limits: LimitRequest,
    next_call: u64,
    transcript_bytes: u64,
    clock: transcript::Encoder,
    now: u64,
    epoch: u64,
    read_step: u64,
}

impl Session {
    #[expect(
        clippy::too_many_lines,
        reason = "The strict corpus record constructor validates related model configuration fields together."
    )]
    fn new(record: &str) -> Result<Self> {
        check_json(record, 0)?;
        let request: SessionRequest =
            serde_json::from_str(record).map_err(|_| "invalid:conformance session".to_string())?;
        let volume_model = request.model == VOLUME_MODEL;
        if request.model != "namespace" && !volume_model
            || request.storage.is_some() != volume_model
            || volume_model && request.binary_image.is_some()
        {
            return Err("invalid:conformance model".into());
        }
        let storage = request.storage;
        let request_binary = request.binary_image;
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
        let mut volume = None;
        let mut namespace =
            Namespace::new(image.clone()).map_err(|boundary| boundary_name(&boundary))?;
        if let Some(storage) = storage {
            let model: Option<volume::StorageModel> = serde_json::from_str(storage.get())
                .map_err(|_| "invalid:storage model".to_string())?;
            let model = model.ok_or_else(|| "invalid:conformance model".to_string())?;
            let session = VolumeSession::new(image, model, limits.transcript_bytes);
            namespace = session.namespace(None)?;
            volume = Some(session);
        }
        if let Some(binary) = request_binary {
            let bytes = decode(&binary.data_base64)?;
            let decoded = crate::namespace::decode_image(
                &mut &bytes[..],
                binary.files,
                limits.output_bytes,
                request.epoch_nanos,
            )
            .map_err(|boundary| boundary_name(&boundary))?;
            let production =
                Namespace::new(decoded).map_err(|boundary| boundary_name(&boundary))?;
            if production.canonical_state() != namespace.canonical_state() {
                return Err("invalid:binary namespace image mismatch".into());
            }
            namespace = production;
        }
        Ok(Self {
            namespace,
            volume,
            limits,
            next_call: 1,
            transcript_bytes: 0,
            clock: transcript::Encoder::default(),
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
        if self.volume.is_some() {
            return self.handle_volume(record);
        }
        self.handle_record(record).unwrap_or_else(reply_record)
    }

    fn handle_record(&mut self, record: &str) -> Result<String> {
        let fields = object_fields(record)?;
        if fields.len() != 2 || !fields.contains_key("op") || !fields.contains_key("input") {
            return Err("invalid:conformance record".into());
        }
        let op: String = field(&fields, "op")?;
        self.call(&op, fields["input"].get())
    }

    #[expect(
        clippy::too_many_lines,
        reason = "The reference corpus dispatch keeps operation-specific wire outputs together."
    )]
    fn call(&mut self, op: &str, input: &str) -> Result<String> {
        let op = op.to_string();
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
        let header = u64::from(self.transcript_bytes == 0) * transcript::HEADER.len() as u64;
        let minimum = if op == "clock_time_get" && self.clock.is_open() {
            0
        } else {
            transcript::RECORD_HEADER
        };
        if self.transcript_bytes + header + minimum > self.limits.transcript_bytes {
            return Err("capacity:transcript-bytes".into());
        }
        let (operation, binary) = binary_input(&op, &input_fields, input.len())?;
        let folded = binary.clock.is_some_and(|(clock_id, precision)| {
            self.clock.folds(
                self.next_call,
                clock_id,
                precision,
                self.now,
                self.read_step,
            )
        });
        let subscriptions = if op == "poll_oneoff" {
            Some(poll_subscriptions(&input_fields)?)
        } else {
            None
        };
        // Typed filesystem operations admit and record through the helper's
        // namespace-owner code; only Go's unknown operations stay driver-local.
        let (call, unsupported_operation) = match operation {
            Some(Operation::Unsupported(name)) => (None, Some(Operation::Unsupported(name))),
            Some(operation) => (Some(namespace_call::Call::new(operation)), None),
            None => (None, None),
        };
        if let Some(call) = &call {
            call.admit(
                self.next_call,
                self.limits.calls,
                self.transcript_bytes,
                self.limits.transcript_bytes,
                false,
            )
            .map_err(|boundary| boundary_name(&boundary))?;
        } else {
            let reservation = if binary.clock.is_some_and(|(clock_id, _)| clock_id <= 1) {
                if folded {
                    0
                } else {
                    header + transcript::CLOCK_RUN_BYTES
                }
            } else if let Some(subscriptions) = &subscriptions {
                namespace_call::admit_poll_frames(self.next_call, subscriptions)
                    .map_err(|boundary| boundary_name(&boundary))?;
                header + transcript::poll_reservation(subscriptions)
            } else {
                header
                    + transcript::RECORD_HEADER
                    + binary.bytes
                    + binary_output_reservation(&op, &input_fields)?
            };
            if self.transcript_bytes + reservation > self.limits.transcript_bytes {
                return Err("capacity:transcript-bytes".into());
            }
        }
        let clock_now = self.now;
        if binary.clock.is_none() {
            self.clock.close();
        }
        let mut record_bytes = None;
        let reply = if op == "clock_time_get" {
            self.clock_call(&input_fields)?
        } else if let Some(subscriptions) = &subscriptions {
            let (reply, bytes) = self.poll_call(subscriptions)?;
            record_bytes = Some(bytes);
            reply
        } else if op == "sched_yield" {
            require_fields(&input_fields, &[])?;
            record_bytes = Some(transcript::yield_record(self.next_call).len() as u64);
            encode_reply(0, "{}", None)
        } else if let Some(call) = call {
            let (tag, input) = (call.tag, call.input);
            let reply = self
                .namespace
                .execute(call.operation, self.now)
                .map_err(stop_name)?;
            let output = namespace_call::output_bytes(reply.output.as_ref());
            record_bytes = Some(
                transcript::record(
                    tag,
                    self.next_call,
                    input,
                    &output,
                    reply.errno,
                    reply.unsupported.as_deref().unwrap_or(""),
                )
                .len() as u64,
            );
            namespace_reply(&reply)
        } else {
            let reply = self
                .namespace
                .execute(unsupported_operation.unwrap(), self.now)
                .map_err(stop_name)?;
            namespace_reply(&reply)
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
            if !folded {
                self.clock.close();
                self.transcript_bytes += header + transcript::CLOCK_RUN_BYTES;
            }
            self.clock
                .read(self.next_call, clock_id, precision, clock_now);
        } else {
            self.clock.close();
            let bytes = match record_bytes {
                Some(bytes) => bytes,
                None => {
                    transcript::RECORD_HEADER
                        + binary.bytes
                        + binary_reply_bytes(&op, &reply_fields)?
                }
            };
            if bytes > FRAME_LIMIT as u64 {
                return Err("capacity:transcript-record".into());
            }
            self.transcript_bytes += header + bytes;
        }
        self.next_call += 1;
        Ok(reply)
    }

    fn clock_call(&mut self, fields: &Fields) -> Result<String> {
        require_fields(fields, &["clock_id", "precision"])?;
        let clock_id: u32 = field(fields, "clock_id")?;
        let _: u64 = field(fields, "precision")?;
        if clock_id > 1 {
            return Ok(encode_reply(0, "{}", Some(transcript::CPU_CLOCK)));
        }
        let timestamp =
            local_model::clock_timestamp(self.now, self.epoch, self.read_step, clock_id)
                .ok_or_else(|| "capacity:clock-nanoseconds".to_string())?;
        self.now += self.read_step;
        Ok(encode_reply(
            0,
            &format!(r#"{{"timestamp":{timestamp}}}"#),
            None,
        ))
    }

    /// Answers a poll through the helper model's poll semantics and returns the
    /// reply with the exact binary record length.
    fn poll_call(&mut self, subscriptions: &[Subscription]) -> Result<(String, u64)> {
        let poll = local_model::poll(
            self.now,
            self.epoch,
            self.limits.pending_events,
            subscriptions,
            Some(&self.namespace),
        )
        .map_err(|error| match error {
            local_model::PollError::Capacity(boundary) => boundary_name(&boundary),
            local_model::PollError::Descriptor => "invalid:local descriptor poll".to_string(),
        })?;
        self.now = poll.now;
        let bytes = transcript::poll_record(
            self.next_call,
            subscriptions,
            &poll.events,
            poll.errno,
            poll.unsupported,
        )
        .len() as u64;
        let reply = if !poll.unsupported.is_empty() {
            encode_reply(0, "{}", Some(poll.unsupported))
        } else if poll.errno != 0 {
            encode_reply(poll.errno, "{}", None)
        } else {
            let ready = poll
                .events
                .iter()
                .map(|event| {
                    format!(
                        r#"{{"userdata":{},"errno":{},"type":{},"nbytes":{},"flags":{}}}"#,
                        event.userdata, event.errno, event.kind, event.nbytes, event.flags
                    )
                })
                .collect::<Vec<_>>()
                .join(",");
            encode_reply(0, &format!(r#"{{"events":[{ready}]}}"#), None)
        };
        Ok((reply, bytes))
    }
}

/// Parses already field-validated poll subscriptions in Go's exact field set.
fn poll_subscriptions(fields: &Fields) -> Result<Vec<Subscription>> {
    require_fields(fields, &["subscriptions"])?;
    let subscriptions: Vec<Box<RawValue>> = field(fields, "subscriptions")?;
    subscriptions
        .iter()
        .map(|raw| {
            let sub = object_fields(raw.get()).map_err(|_| "invalid:subscription".to_string())?;
            let kind: u8 = sub
                .get("type")
                .map(|raw| serde_json::from_str(if raw.get() == "null" { "0" } else { raw.get() }))
                .transpose()
                .map_err(|_| "invalid:subscription type".to_string())?
                .ok_or_else(|| "invalid:subscription type".to_string())?;
            let mut subscription = Subscription {
                userdata: 0,
                kind,
                clock_id: 0,
                timeout: 0,
                precision: 0,
                flags: 0,
                fd: 0,
            };
            if kind == 0 {
                require_fields(
                    &sub,
                    &[
                        "userdata",
                        "type",
                        "clock_id",
                        "timeout",
                        "precision",
                        "flags",
                    ],
                )?;
                subscription.userdata = field(&sub, "userdata")?;
                subscription.clock_id = field(&sub, "clock_id")?;
                subscription.timeout = field(&sub, "timeout")?;
                subscription.precision = field(&sub, "precision")?;
                subscription.flags = field(&sub, "flags")?;
            } else {
                require_fields(&sub, &["userdata", "type", "fd"])?;
                subscription.userdata = field(&sub, "userdata")?;
                subscription.fd = field(&sub, "fd")?;
            }
            Ok(subscription)
        })
        .collect()
}

fn namespace_reply(reply: &namespace::Reply) -> String {
    let output = if reply.errno == 0 {
        reply
            .output
            .as_ref()
            .map_or_else(|| "{}".into(), encode_output)
    } else {
        "{}".into()
    };
    encode_reply(reply.errno, &output, reply.unsupported.as_deref())
}

#[expect(
    clippy::too_many_lines,
    reason = "Each namespace operation decodes its complete strict field set in one match."
)]
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
        "path_readlink" => &["fd", "path_base64", "length"],
        "path_symlink" => {
            require_field_count(fields, 3)?;
            return Ok(Operation::Symlink {
                old_path: binary_bytes_field(fields, "old_path_base64")?,
                fd: field(fields, "fd")?,
                path: binary_bytes_field(fields, "path_base64")?,
            });
        }
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
        "path_readlink" => Operation::Readlink {
            fd,
            path: binary_bytes_field(fields, "path_base64")?,
            length: field(fields, "length")?,
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

fn boundary_name(boundary: &namespace::Boundary) -> String {
    format!("{}:{}", boundary.kind, boundary.dimension)
}
/// Boundaries keep the `kind:dimension` error form; storage crashes and Go
/// errors outside the taxonomy are complete reply records (they start with
/// `{`), as the Go volume harness reports them.
fn stop_name(stop: namespace::Stop) -> String {
    match stop {
        namespace::Stop::Boundary(boundary) => boundary_name(&boundary),
        namespace::Stop::StorageCrash(fault) => {
            format!(r#"{{"storage_crash":{}}}"#, go_json(&fault))
        }
        namespace::Stop::Fatal(message) => error_record(&message),
    }
}
fn error_record(message: &str) -> String {
    format!(r#"{{"error":{}}}"#, go_json(&message))
}
fn reply_record(error: String) -> String {
    if error.starts_with('{') {
        error
    } else {
        boundary_record(&error)
    }
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
    let mut session = match record.and_then(|record| Session::new(&record)) {
        Ok(session) => {
            writeln!(writer, "{{}}")?;
            writer.flush()?;
            session
        }
        Err(error) => {
            writeln!(writer, "{}", reply_record(error))?;
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
    fn token_validator_accepts_non_object_json() {
        for input in [
            "1e400",
            "-1234567890123456789012345678901234567890",
            "[{},[]]",
        ] {
            assert_eq!(check_json(input, 0), Ok(()), "{input}");
        }
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The corpus validator preserves its complete ordered literal and observation checks."
    )]
    fn namespace_conformance_corpus() {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Corpus {
            name: String,
            #[serde(rename = "origin")]
            _origin: String,
            session: Box<RawValue>,
            operations: Vec<CorpusOperation>,
            replies: Vec<Box<RawValue>>,
            digest: String,
            digests: Option<Vec<String>>,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct CorpusOperation {
            op: String,
            input: Box<RawValue>,
            #[serde(default)]
            repeat_data: u32,
        }
        const LEGACY_NAMES: [&str; 18] = [
            "capacity-buffer-bytes",
            "capacity-callbacks",
            "capacity-descriptors",
            "capacity-files",
            "capacity-filesystem-bytes",
            "capacity-offered-output-bytes",
            "capacity-pending-events",
            "capacity-transcript-bytes",
            "capacity-write-buffer",
            "detached-directory",
            "directories-and-rename",
            "errno-and-containment",
            "invalid-fields-before-transcript",
            "metadata-and-data",
            "output-and-mixed-poll",
            "readonly",
            "recorded-stock-guest",
            "strict-and-unsupported",
        ];
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/namespace-conformance");
        let mut paths = std::fs::read_dir(&directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|value| value == "json"))
            .collect::<Vec<_>>();
        paths.sort();
        assert!(!paths.is_empty(), "missing namespace conformance corpus");
        for name in LEGACY_NAMES {
            assert!(
                directory
                    .join(format!("{name}-binary-image-and-step-digests.json"))
                    .is_file(),
                "missing enhanced companion for {name}"
            );
        }
        for path in paths {
            let raw = std::fs::read(&path).unwrap();
            let corpus: Corpus = serde_json::from_slice(&raw).unwrap();
            assert_eq!(
                path.file_stem().unwrap().to_str(),
                Some(corpus.name.as_str())
            );
            let fields = object_fields(std::str::from_utf8(&raw).unwrap()).unwrap();
            let session_fields = object_fields(corpus.session.get()).unwrap();
            let legacy = LEGACY_NAMES.contains(&corpus.name.as_str());
            if legacy {
                assert!(
                    !fields.contains_key("digests"),
                    "{} legacy schema",
                    corpus.name
                );
                assert!(
                    !session_fields.contains_key("binary_image"),
                    "{} legacy session",
                    corpus.name
                );
            } else {
                assert!(
                    session_fields
                        .get("binary_image")
                        .is_some_and(|image| image.get() != "null"),
                    "{} missing enhanced binary image",
                    corpus.name
                );
                assert_eq!(
                    corpus
                        .digests
                        .as_ref()
                        .expect("enhanced corpus digests required")
                        .len(),
                    corpus.operations.len(),
                    "{}",
                    corpus.name
                );
            }
            if let Some(original) = corpus.name.strip_suffix("-binary-image-and-step-digests") {
                assert!(LEGACY_NAMES.contains(&original));
                let original_bytes =
                    std::fs::read(directory.join(format!("{original}.json"))).unwrap();
                let mut original_fields =
                    object_fields(std::str::from_utf8(&original_bytes).unwrap()).unwrap();
                let mut enhanced_fields =
                    object_fields(std::str::from_utf8(&raw).unwrap()).unwrap();
                assert_eq!(original_fields["name"].get(), go_json(&original));
                enhanced_fields.insert(
                    "name".into(),
                    RawValue::from_string(go_json(&original)).unwrap(),
                );
                enhanced_fields.remove("digests").unwrap();
                let mut enhanced_session =
                    object_fields(enhanced_fields.remove("session").unwrap().get()).unwrap();
                enhanced_session.remove("binary_image").unwrap();
                let original_session =
                    object_fields(original_fields.remove("session").unwrap().get()).unwrap();
                let raw_values = |fields: Fields| -> BTreeMap<String, String> {
                    fields
                        .into_iter()
                        .map(|(key, value)| (key, value.get().to_owned()))
                        .collect()
                };
                assert_eq!(
                    raw_values(enhanced_fields),
                    raw_values(original_fields),
                    "{} companion changed original projection",
                    corpus.name
                );
                assert_eq!(
                    raw_values(enhanced_session),
                    raw_values(original_session),
                    "{} companion changed original session",
                    corpus.name
                );
            }
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
                if let Some(digests) = &corpus.digests {
                    assert_eq!(
                        session.handle(r#"{"op":"digest","input":{}}"#),
                        format!(r#"{{"digest":"{}"}}"#, digests[index]),
                        "{} op {index} state",
                        corpus.name
                    );
                }
            }
            assert_eq!(
                session.handle(r#"{"op":"digest","input":{}}"#),
                format!(r#"{{"digest":"{}"}}"#, corpus.digest),
                "{} final digest",
                corpus.name
            );
        }
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The corpus validator preserves its complete ordered literal and observation checks."
    )]
    fn namespace_image_conformance_corpus() {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Corpus {
            name: String,
            data_base64: String,
            files: u64,
            output_bytes: u64,
            epoch_nanos: u64,
            #[serde(default)]
            image_sha256: Option<String>,
            #[serde(default)]
            namespace_sha256: Option<String>,
            #[serde(default)]
            boundary: Option<String>,
            #[serde(default)]
            repeat: Option<Repeat>,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Repeat {
            offset: usize,
            count: usize,
            byte: u8,
        }
        for malformed in [
            r#"{"name":"unknown","data_base64":"","files":1,"output_bytes":3,"epoch_nanos":4,"boundary":"invalid:namespace image","extra":0}"#,
            r#"{"name":"wrong-bytes","data_base64":[],"files":1,"output_bytes":3,"epoch_nanos":4,"boundary":"invalid:namespace image"}"#,
            r#"{"name":"unknown-repeat","data_base64":"","files":1,"output_bytes":3,"epoch_nanos":4,"boundary":"invalid:namespace image","repeat":{"offset":0,"count":1,"byte":0,"extra":0}}"#,
        ] {
            assert!(
                serde_json::from_str::<Corpus>(malformed).is_err(),
                "accepted malformed image fixture: {malformed}"
            );
        }
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/namespace-image-conformance");
        let mut paths = std::fs::read_dir(directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|value| value == "json"))
            .collect::<Vec<_>>();
        paths.sort();
        assert!(!paths.is_empty(), "missing namespace image corpus");
        for path in paths {
            let corpus: Corpus = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            assert_ne!(
                corpus.boundary.is_some(),
                corpus.image_sha256.is_some(),
                "{}",
                corpus.name
            );
            let mut bytes = STANDARD.decode(&corpus.data_base64).unwrap();
            if let Some(repeat) = corpus.repeat {
                assert!(repeat.offset <= bytes.len() && repeat.count <= (4 << 20) + 1);
                bytes.splice(
                    repeat.offset..repeat.offset,
                    std::iter::repeat_n(repeat.byte, repeat.count),
                );
            }
            let decoded = crate::namespace::decode_image(
                &mut &bytes[..],
                corpus.files,
                corpus.output_bytes,
                corpus.epoch_nanos,
            );
            let image = match decoded {
                Err(error) => {
                    assert_eq!(
                        Some(boundary_name(&error)),
                        corpus.boundary,
                        "{}",
                        corpus.name
                    );
                    continue;
                }
                Ok(image) => {
                    assert!(
                        corpus.boundary.is_none(),
                        "{} accepted invalid image",
                        corpus.name
                    );
                    image
                }
            };
            let paths = |values: &[Vec<u8>]| {
                values
                    .iter()
                    .map(|value| STANDARD.encode(value))
                    .collect::<Vec<_>>()
            };
            let projection = serde_json::json!({
                "entries": image.entries.iter().map(|entry| serde_json::json!({
                    "path_base64": STANDARD.encode(&entry.path),
                    "directory": entry.directory,
                    "readonly": entry.readonly,
                    "data_base64": STANDARD.encode(&entry.data),
                })).collect::<Vec<_>>(),
                "writable_directories": paths(&image.writable_directories),
                "preopens": paths(&image.preopens),
                "working_directory": STANDARD.encode(&image.working_directory),
                "stdin_base64": STANDARD.encode(&image.stdin),
                "epoch_nanos": image.epoch_nanos,
                "limits": {
                    "files": image.limits.files,
                    "descriptors": image.limits.descriptors,
                    "filesystem_bytes": image.limits.filesystem_bytes,
                    "output_bytes": image.limits.output_bytes,
                },
            });
            assert_eq!(
                Some(format!("{:x}", Sha256::digest(go_json(&projection)))),
                corpus.image_sha256,
                "{} decoded image",
                corpus.name
            );
            let namespace = Namespace::new(image)
                .unwrap_or_else(|error| panic!("{} namespace: {error}", corpus.name));
            if let Some(expected) = corpus.namespace_sha256 {
                assert_eq!(
                    format!("{:x}", Sha256::digest(namespace.canonical_state())),
                    expected,
                    "{} canonical namespace",
                    corpus.name
                );
            }
        }
    }

    #[test]
    fn namespace_wire_conformance_corpus() {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Corpus {
            name: String,
            input: String,
            output: String,
        }
        assert!(
            serde_json::from_str::<Corpus>(
                r#"{"name":"unknown","input":"","output":"","extra":0}"#
            )
            .is_err()
        );
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/namespace-wire-conformance");
        let mut paths = std::fs::read_dir(directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|value| value == "json"))
            .collect::<Vec<_>>();
        paths.sort();
        assert!(!paths.is_empty(), "missing namespace wire corpus");
        for path in paths {
            let corpus: Corpus = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            let mut output = Vec::new();
            run(
                &mut std::io::Cursor::new(corpus.input.as_bytes()),
                &mut output,
            )
            .unwrap();
            assert_eq!(
                String::from_utf8(output).unwrap(),
                corpus.output,
                "{}",
                corpus.name
            );
        }
    }

    #[test]
    fn volume_conformance_corpus() {
        #[derive(Deserialize)]
        struct Corpus {
            name: String,
            seed: u64,
            session: Box<RawValue>,
            initial_reply_base64: String,
            operations: Option<Vec<Box<RawValue>>>,
            observations: Option<Vec<serde_json::Value>>,
        }
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/volume-conformance");
        let mut paths = std::fs::read_dir(&directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|value| value == "json"))
            .collect::<Vec<_>>();
        paths.sort();
        assert!(paths.len() >= 44, "missing volume conformance corpus");
        for path in paths {
            let corpus: Corpus = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
            let initial =
                String::from_utf8(STANDARD.decode(&corpus.initial_reply_base64).unwrap()).unwrap();
            let mut session = match Session::new(&compact_json(corpus.session.get())) {
                Ok(session) => {
                    assert_eq!(initial, "{}", "{}", corpus.name);
                    session
                }
                Err(error) => {
                    assert_eq!(reply_record(error), initial, "{}", corpus.name);
                    assert!(corpus.observations.is_none(), "{}", corpus.name);
                    continue;
                }
            };
            let observations = corpus.observations.unwrap_or_default();
            let operations = corpus.operations.unwrap_or_default();
            assert_eq!(operations.len(), observations.len(), "{}", corpus.name);
            for (step, (operation, expected)) in operations.iter().zip(observations).enumerate() {
                let got: serde_json::Value =
                    serde_json::from_str(&session.handle(&compact_json(operation.get()))).unwrap();
                assert_eq!(
                    got, expected,
                    "{} seed={} first divergence at step={step}",
                    corpus.name, corpus.seed
                );
            }
        }
    }
}

/// The `internal/model/volume` session state beyond the namespace: the
/// configuration for fresh/recovered environments and the named controls.
struct VolumeSession {
    image: Image,
    writable: Vec<Vec<u8>>,
    model: volume::StorageModel,
    transcript_limit: u64,
    images: BTreeMap<String, volume::Image>,
    pages: BTreeMap<String, volume::CrashEnumeration>,
    selections: BTreeMap<String, Selections>,
    image_digest: String,
    image_data: Option<Vec<u8>>,
}

type Selections = Option<BTreeMap<String, Option<Vec<u64>>>>;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct EnumerateInput {
    #[serde(default)]
    volume: String,
    #[serde(default)]
    limits: Option<volume::CrashEnumerationLimits>,
    #[serde(default)]
    cursor: String,
    #[serde(default)]
    cursor_input: Option<volume::CrashExploration>,
    #[serde(default)]
    save: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct SelectInput {
    #[serde(default)]
    seed: u64,
    #[serde(default)]
    save: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct CrashInput {
    #[serde(default)]
    selections: Selections,
    #[serde(default)]
    selection: String,
    #[serde(default)]
    page: String,
    #[serde(default)]
    state: i64,
    #[serde(default)]
    image: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RecoverInput {
    #[serde(default)]
    image: String,
    #[serde(default)]
    image_base64: Option<String>,
    #[serde(default)]
    image_digest: String,
}

impl VolumeSession {
    fn new(image: Image, model: volume::StorageModel, transcript_limit: u64) -> Self {
        Self {
            writable: image.writable_directories.clone(),
            image,
            model,
            transcript_limit,
            images: BTreeMap::new(),
            pages: BTreeMap::new(),
            selections: BTreeMap::new(),
            image_digest: String::new(),
            image_data: None,
        }
    }

    fn image_limit(&self) -> u64 {
        (1 << 20) + self.image.limits.filesystem_bytes
    }

    /// Go `NewEnvironment` (fresh) or `NewRecoveredEnvironment` (recovery
    /// image): a namespace with the validated storage model attached.
    fn namespace(&self, recovery: Option<&volume::Image>) -> Result<Namespace> {
        let mut namespace =
            Namespace::new(self.image.clone()).map_err(|boundary| boundary_name(&boundary))?;
        self.model
            .validate(&self.writable, self.image.limits.filesystem_bytes)
            .map_err(|dimension| format!("invalid:{dimension}"))?;
        let storage = match recovery {
            None => volume::Storage::new(self.model.clone(), self.transcript_limit),
            Some(image) => volume::Storage::recover(
                self.model.clone(),
                self.transcript_limit,
                &image.bytes,
                &image.digest,
                self.image_limit(),
            ),
        }
        .map_err(storage_stop_name)?;
        namespace.attach_storage(storage).map_err(stop_name)?;
        Ok(namespace)
    }
}

fn storage_stop_name(stop: volume::StorageStop) -> String {
    match stop {
        volume::StorageStop::Boundary(kind, dimension) => format!("{kind}:{dimension}"),
        volume::StorageStop::Crash(fault) => {
            format!(r#"{{"storage_crash":{}}}"#, go_json(&fault))
        }
        volume::StorageStop::Fatal(message) => error_record(&message),
    }
}

fn strict<T: serde::de::DeserializeOwned>(input: &str) -> Result<T> {
    serde_json::from_str(input).map_err(|_| "invalid:conformance control input".to_string())
}

impl Session {
    fn handle_volume(&mut self, record: &str) -> String {
        match self.volume_step(record) {
            Ok(observation) => observation,
            Err(error) => reply_record(error),
        }
    }

    /// One Go `volumeConformanceObservation` for an `{op,input,now}` record.
    #[expect(
        clippy::too_many_lines,
        reason = "The corpus volume scenario keeps each operation and its serialized outcome together."
    )]
    fn volume_step(&mut self, record: &str) -> Result<String> {
        let fields = object_fields(record)?;
        if fields.len() != 3
            || !fields.contains_key("op")
            || !fields.contains_key("input")
            || !fields.contains_key("now")
        {
            return Err("invalid:conformance record".into());
        }
        let op: String = field(&fields, "op")?;
        let now: u64 = field(&fields, "now")?;
        let input = fields["input"].get();
        check_json(input, 0)?;
        self.now = now;
        let mut before = self.storage_log().len();
        let mut enumeration = None;
        let mut selections: Option<Selections> = None;
        let reply = match op.as_str() {
            "enumerate" => {
                let input: EnumerateInput = strict(input)?;
                let state = self.volume.as_ref().unwrap();
                let cursor = if input.cursor.is_empty() {
                    input.cursor_input
                } else {
                    let page = state
                        .pages
                        .get(&input.cursor)
                        .and_then(|page| page.choice_exploration.clone())
                        .ok_or_else(|| "invalid:missing conformance cursor".to_string())?;
                    Some(page)
                };
                let limits = input.limits.unwrap_or(volume::CrashEnumerationLimits {
                    states: 0,
                    operations: 0,
                    depth: 0,
                    bytes: 0,
                    work: 0,
                });
                let result = self.namespace.storage().unwrap().enumerate(
                    &input.volume,
                    limits,
                    cursor.as_ref(),
                );
                match result {
                    Ok(page) => {
                        enumeration = Some(page.clone());
                        self.volume.as_mut().unwrap().pages.insert(input.save, page);
                        "{}".to_string()
                    }
                    Err(error) => {
                        enumeration = Some(volume::CrashEnumeration {
                            states: None,
                            choice_exploration: None,
                            complete: false,
                            capacity: String::new(),
                        });
                        error_record(&volume::storage_error_text(&error))
                    }
                }
            }
            "select" => {
                let input: SelectInput = strict(input)?;
                let selected: Selections = Some(
                    self.namespace
                        .storage()
                        .unwrap()
                        .fs
                        .select_crash_operations(input.seed)
                        .into_iter()
                        .map(|(volume, operations)| (volume, Some(operations)))
                        .collect(),
                );
                selections = Some(selected.clone());
                self.volume
                    .as_mut()
                    .unwrap()
                    .selections
                    .insert(input.save, selected);
                "{}".to_string()
            }
            "crash" => {
                let input: CrashInput = strict(input)?;
                let state = self.volume.as_ref().unwrap();
                let mut selected = input.selections;
                if !input.selection.is_empty() {
                    selected = state
                        .selections
                        .get(&input.selection)
                        .cloned()
                        .ok_or_else(|| "invalid:missing conformance selection".to_string())?;
                }
                if !input.page.is_empty() {
                    let page = state.pages.get(&input.page);
                    let states = page.and_then(|page| page.states.as_ref());
                    let crash = states
                        .and_then(|states| {
                            usize::try_from(input.state)
                                .ok()
                                .and_then(|index| states.get(index))
                        })
                        .ok_or_else(|| "invalid:conformance page state".to_string())?;
                    selected = Some(BTreeMap::from([(
                        crash.volume.clone(),
                        Some(crash.selected_operations.clone()),
                    )]));
                }
                selections = Some(selected.clone());
                let plain: BTreeMap<String, Vec<u64>> = selected
                    .unwrap_or_default()
                    .into_iter()
                    .map(|(volume, operations)| (volume, operations.unwrap_or_default()))
                    .collect();
                let maximum = state.image_limit();
                match self.namespace.crash_storage(&plain, maximum) {
                    Ok(image) => {
                        let state = self.volume.as_mut().unwrap();
                        state.image_digest.clone_from(&image.digest);
                        state.image_data = Some(image.bytes.clone());
                        state.images.insert(input.image, image);
                        "{}".to_string()
                    }
                    Err(volume::Error::Capacity {
                        resource: "image-bytes",
                        ..
                    }) => boundary_record("capacity:storage-image-bytes"),
                    Err(error) => error_record(&volume::storage_error_text(&error)),
                }
            }
            "recover" => {
                let input: RecoverInput = strict(input)?;
                let state = self.volume.as_ref().unwrap();
                let supplied;
                let image = if let Some(data) = input.image_base64 {
                    supplied = volume::Image {
                        bytes: decode_base64(&data)
                            .map_err(|_| "invalid:conformance image bytes".to_string())?,
                        digest: input.image_digest,
                    };
                    &supplied
                } else {
                    state
                        .images
                        .get(&input.image)
                        .ok_or_else(|| "invalid:missing conformance image".to_string())?
                };
                match state.namespace(Some(image)) {
                    Ok(namespace) => {
                        let (digest, data) = (image.digest.clone(), image.bytes.clone());
                        self.namespace = namespace;
                        self.next_call = 1;
                        self.transcript_bytes = 0;
                        self.clock = transcript::Encoder::default();
                        self.now = now;
                        before = 0;
                        let state = self.volume.as_mut().unwrap();
                        state.image_digest = digest;
                        state.image_data = Some(data);
                        "{}".to_string()
                    }
                    Err(error) => reply_record(error),
                }
            }
            _ => self.call(&op, input).unwrap_or_else(reply_record),
        };
        let storage = self.namespace.storage().unwrap();
        let events = (!storage.log.is_empty()).then(|| STANDARD.encode(&storage.log[before..]));
        let operations: Vec<_> = storage
            .operations()
            .into_iter()
            .map(|(volume, operations)| serde_json::json!({"volume": volume, "operations": operations}))
            .collect();
        let state = self.volume.as_ref().unwrap();
        let observation = serde_json::json!({
            "reply_base64": STANDARD.encode(reply.as_bytes()),
            "events_base64": events,
            "snapshots": storage.snapshots(),
            "operations": operations,
            "enumeration": enumeration,
            "selections": selections.flatten(),
            "image_digest": state.image_digest,
            "image_base64": state.image_data.as_ref().map(|data| STANDARD.encode(data)),
            "image_identity": volume::IMAGE_VERSION,
            "namespace_digest": format!("{:x}", Sha256::digest(self.namespace.canonical_state())),
        });
        Ok(observation.to_string())
    }

    fn storage_log(&self) -> &[u8] {
        self.namespace
            .storage()
            .map_or(&[][..], |storage| storage.log.as_slice())
    }
}
