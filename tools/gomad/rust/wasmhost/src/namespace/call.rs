use crate::namespace::{Boundary, Operation, Output};

pub struct Call {
    pub operation: Operation,
    pub tag: u8,
    pub name: &'static str,
    pub input: Vec<u8>,
    json_bytes: u64,
    length: u64,
    max_buffer: u64,
    output_reservation: u64,
}
impl Call {
    #[expect(
        clippy::too_many_lines,
        reason = "Each namespace operation keeps its complete canonical field order in one dispatch."
    )]
    pub fn new(operation: Operation) -> Self {
        let mut fields = Fields::default();
        let (tag, name, length, output_reservation) = match &operation {
            Operation::Write { fd, data, offset } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.bytes("data_base64", data);
                if let Some(offset) = offset {
                    fields.number("offset", *offset, 8);
                    (9, "fd_pwrite", 0, 4)
                } else {
                    (8, "fd_write", 0, 4)
                }
            }
            Operation::Read { fd, length, offset } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("length", u64::from(*length), 4);
                if let Some(offset) = offset {
                    fields.number("offset", *offset, 8);
                    (11, "fd_pread", u64::from(*length), u64::from(*length) + 4)
                } else {
                    (10, "fd_read", u64::from(*length), u64::from(*length) + 4)
                }
            }
            Operation::Close(fd)
            | Operation::Sync(fd)
            | Operation::FilestatGet(fd)
            | Operation::FdstatGet(fd)
            | Operation::PrestatGet(fd) => {
                fields.number("fd", u64::from(*fd), 4);
                match &operation {
                    Operation::Close(_) => (12, "fd_close", 0, 0),
                    Operation::Sync(_) => (13, "fd_sync", 0, 0),
                    Operation::FilestatGet(_) => (14, "fd_filestat_get", 0, 57),
                    Operation::FdstatGet(_) => (15, "fd_fdstat_get", 0, 19),
                    _ => (16, "fd_prestat_get", 0, 4),
                }
            }
            Operation::PrestatDirName { fd, length } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("length", u64::from(*length), 4);
                (
                    17,
                    "fd_prestat_dir_name",
                    u64::from(*length),
                    u64::from(*length) + 4,
                )
            }
            Operation::SetFlags { fd, flags } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("flags", u64::from(*flags), 2);
                (
                    18,
                    "fd_fdstat_set_flags",
                    0,
                    4 + crate::namespace::FD_FLAGS_UNSUPPORTED.len() as u64,
                )
            }
            Operation::SetSize { fd, size } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("size", *size, 8);
                (19, "fd_filestat_set_size", 0, 0)
            }
            Operation::Seek { fd, offset, whence } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.signed("offset", *offset);
                fields.number("whence", u64::from(*whence), 1);
                (20, "fd_seek", 0, 8)
            }
            Operation::Readdir { fd, length, cookie } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("length", u64::from(*length), 4);
                fields.number("cookie", *cookie, 8);
                (21, "fd_readdir", u64::from(*length), u64::from(*length) + 4)
            }
            Operation::CreateDirectory { fd, path }
            | Operation::RemoveDirectory { fd, path }
            | Operation::UnlinkFile { fd, path } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.bytes("path_base64", path);
                match &operation {
                    Operation::CreateDirectory { .. } => (22, "path_create_directory", 0, 0),
                    Operation::RemoveDirectory { .. } => (23, "path_remove_directory", 0, 0),
                    _ => (24, "path_unlink_file", 0, 0),
                }
            }
            Operation::PathStat { fd, path, flags } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("flags", u64::from(*flags), 4);
                fields.bytes("path_base64", path);
                (25, "path_filestat_get", 0, 57)
            }
            Operation::Open(open) => {
                fields.number("fd", u64::from(open.fd), 4);
                fields.number("dirflags", u64::from(open.dirflags), 4);
                fields.bytes("path_base64", &open.path);
                fields.number("oflags", u64::from(open.oflags), 2);
                fields.number("rights_base", open.base, 8);
                fields.number("rights_inheriting", open.inheriting, 8);
                fields.number("fdflags", u64::from(open.flags), 2);
                (
                    26,
                    "path_open",
                    0,
                    4 + crate::namespace::OPEN_FLAGS_UNSUPPORTED.len() as u64,
                )
            }
            Operation::Readlink { fd, path, length } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.bytes("path_base64", path);
                fields.number("length", u64::from(*length), 4);
                (
                    27,
                    "path_readlink",
                    u64::from(*length),
                    (u64::from(*length) + 4)
                        .max(4 + crate::namespace::READLINK_UNSUPPORTED.len() as u64),
                )
            }
            Operation::Rename {
                fd,
                path,
                new_fd,
                new_path,
            } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.bytes("path_base64", path);
                fields.number("new_fd", u64::from(*new_fd), 4);
                fields.bytes("new_path_base64", new_path);
                (
                    28,
                    "path_rename",
                    0,
                    4 + "operation outside stock WASI profile: path_rename".len() as u64,
                )
            }
            Operation::Symlink { old_path, fd, path } => {
                fields.bytes("old_path_base64", old_path);
                fields.number("fd", u64::from(*fd), 4);
                fields.bytes("path_base64", path);
                (
                    29,
                    "path_symlink",
                    0,
                    4 + "operation outside stock WASI profile: path_symlink".len() as u64,
                )
            }
            Operation::SockAccept { fd, flags } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("flags", u64::from(*flags), 2);
                (
                    30,
                    "sock_accept",
                    0,
                    4 + "operation outside stock WASI profile: sock_accept".len() as u64,
                )
            }
            Operation::SockShutdown { fd, how } => {
                fields.number("fd", u64::from(*fd), 4);
                fields.number("how", u64::from(*how), 1);
                (
                    31,
                    "sock_shutdown",
                    0,
                    4 + "operation outside stock WASI profile: sock_shutdown".len() as u64,
                )
            }
            #[cfg(any(test, feature = "conformance"))]
            Operation::Unsupported(_) => unreachable!("typed namespace operation required"),
        };
        Self {
            operation,
            tag,
            name,
            input: fields.binary,
            json_bytes: fields.json_bytes + 2,
            length,
            max_buffer: fields.max_buffer,
            output_reservation,
        }
    }
    pub fn admit(
        &self,
        id: u64,
        calls: u64,
        transcript: u64,
        limit: u64,
        digest_only: bool,
    ) -> Result<(), Boundary> {
        let fail = |dimension| Boundary {
            kind: "capacity",
            dimension,
        };
        if id > calls {
            return Err(fail("callbacks"));
        }
        let callback = b"{\"type\":\"call\",\"id\":".len() as u64
            + digits(id)
            + b",\"op\":\"\",\"input\":}".len() as u64
            + self.name.len() as u64
            + self.json_bytes;
        if callback + 1 > 8 << 20 {
            return Err(fail("callback-frame"));
        }
        if self.length > 4 << 20 {
            return Err(fail("buffer-bytes"));
        }
        if 1028 + self.length * 4 / 3 + 1 > 8 << 20 {
            return Err(fail("reply-frame"));
        }
        let header = if transcript == 0 { 8 } else { 0 };
        if !digest_only && transcript + header + 16 > limit {
            return Err(fail("transcript-bytes"));
        }
        if self.max_buffer > 4 << 20 {
            return Err(fail("buffer-bytes"));
        }
        if !digest_only
            && transcript + header + 16 + self.input.len() as u64 + self.output_reservation > limit
        {
            return Err(fail("transcript-bytes"));
        }
        Ok(())
    }
}
fn digits(value: u64) -> u64 {
    if value == 0 {
        1
    } else {
        u64::from(value.ilog10()) + 1
    }
}
#[derive(Default)]
struct Fields {
    binary: Vec<u8>,
    json_bytes: u64,
    count: usize,
    max_buffer: u64,
}
impl Fields {
    fn field(&mut self, name: &str, bytes: u64) {
        self.json_bytes += u64::from(self.count != 0) + name.len() as u64 + 3 + bytes;
        self.count += 1;
    }
    fn number(&mut self, name: &str, value: u64, width: usize) {
        self.field(name, digits(value));
        self.binary.extend_from_slice(&value.to_le_bytes()[..width]);
    }
    fn json_size(&self) -> u64 {
        self.json_bytes + 2
    }
    fn signed(&mut self, name: &str, value: i64) {
        self.field(name, digits(value.unsigned_abs()) + u64::from(value < 0));
        self.binary.extend_from_slice(&value.to_le_bytes());
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Namespace call byte fields come from validated WASI paths or BUFFER_LIMIT-bounded writes."
    )]
    fn bytes(&mut self, name: &str, value: &[u8]) {
        self.field(name, 2 + (value.len() as u64).div_ceil(3) * 4);
        self.max_buffer = self.max_buffer.max(value.len() as u64);
        self.binary
            .extend_from_slice(&(value.len() as u32).to_le_bytes());
        self.binary.extend_from_slice(value);
    }
}
/// Go's callback and reply frame bounds for a `poll_oneoff` with these
/// subscriptions, using the exact JSON size Go's callback input would have.
pub fn admit_poll_frames(
    id: u64,
    subscriptions: &[crate::transcript::Subscription],
) -> Result<(), Boundary> {
    let mut input_bytes = b"{\"subscriptions\":[]}".len() as u64;
    for (index, sub) in subscriptions.iter().enumerate() {
        let mut fields = Fields::default();
        fields.number("userdata", sub.userdata, 8);
        fields.number("type", u64::from(sub.kind), 1);
        if sub.kind == 0 {
            fields.number("clock_id", u64::from(sub.clock_id), 4);
            fields.number("timeout", sub.timeout, 8);
            fields.number("precision", sub.precision, 8);
            fields.number("flags", u64::from(sub.flags), 2);
        } else {
            fields.number("fd", u64::from(sub.fd), 4);
        }
        input_bytes += fields.json_size() + u64::from(index != 0);
    }
    let fail = |dimension| Boundary {
        kind: "capacity",
        dimension,
    };
    let callback = b"{\"type\":\"call\",\"id\":".len() as u64
        + digits(id)
        + b",\"op\":\"poll_oneoff\",\"input\":}".len() as u64
        + input_bytes;
    if callback + 1 > 8 << 20 {
        return Err(fail("callback-frame"));
    }
    if 1028 + input_bytes * 4 + 1 > 8 << 20 {
        return Err(fail("reply-frame"));
    }
    Ok(())
}
#[expect(
    clippy::cast_possible_truncation,
    reason = "Namespace output bytes are bounded by admitted WASI buffers."
)]
pub fn output_bytes(output: Option<&Output>) -> Vec<u8> {
    let mut bytes = Vec::new();
    match output {
        None | Some(Output::Empty) => (),
        Some(Output::Prestat(n) | Output::Fd(n) | Output::Written(n)) => {
            bytes.extend_from_slice(&n.to_le_bytes());
        }
        Some(Output::Offset(n)) => bytes.extend_from_slice(&n.to_le_bytes()),
        Some(Output::Bytes(data)) => {
            bytes.extend_from_slice(&(data.len() as u32).to_le_bytes());
            bytes.extend_from_slice(data);
        }
        Some(Output::Fdstat(stat)) => {
            bytes.push(stat.filetype);
            bytes.extend_from_slice(&stat.flags.to_le_bytes());
            bytes.extend_from_slice(&stat.base.to_le_bytes());
            bytes.extend_from_slice(&stat.inheriting.to_le_bytes());
        }
        Some(Output::Filestat(stat)) => {
            bytes.extend_from_slice(&stat.dev.to_le_bytes());
            bytes.extend_from_slice(&stat.ino.to_le_bytes());
            bytes.push(stat.filetype);
            for n in [stat.nlink, stat.size, stat.atim, stat.mtim, stat.ctim] {
                bytes.extend_from_slice(&n.to_le_bytes());
            }
        }
    }
    bytes
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The corpus validator preserves its complete ordered literal and observation checks."
    )]
    fn namespace_binary_transcript_codec_corpus() {
        use base64::{Engine as _, engine::general_purpose::STANDARD};
        use serde::Deserialize;
        use sha2::{Digest, Sha256};

        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Input {
            fd: u32,
            data_base64: String,
            #[serde(default)]
            offset: Option<u64>,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct ReplyOutput {
            #[serde(default)]
            written: Option<u32>,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Reply {
            #[serde(rename = "type")]
            kind: String,
            id: u64,
            errno: u16,
            output: ReplyOutput,
            #[serde(default)]
            unsupported: bool,
            #[serde(default)]
            message: String,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Expected {
            reply: Reply,
            transcript_base64: String,
            sha256: String,
            tag: u8,
            binary_input_base64: String,
            binary_output_base64: Option<String>,
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Corpus {
            schema: String,
            name: String,
            kind: String,
            op: String,
            input: Input,
            limit: u64,
            reference_limit: u64,
            digest_transcript: bool,
            reference: Option<serde_json::Value>,
            expected: Expected,
        }

        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/namespace-transcript-conformance");
        for name in [
            "fd_pwrite-stdout-ab-offset9-record",
            "fd_write-stdout-ab-record",
        ] {
            let corpus: Corpus = serde_json::from_slice(
                &std::fs::read(directory.join(format!("{name}.json"))).unwrap(),
            )
            .unwrap();
            assert_eq!(corpus.schema, "gomad.namespace-public-corpus/v1");
            assert_eq!(corpus.name, name);
            assert_eq!(corpus.kind, "codec");
            assert!(corpus.limit > 0 && corpus.reference_limit == 0);
            assert!(!corpus.digest_transcript && corpus.reference.is_none());
            assert_eq!(corpus.input.offset.is_some(), corpus.op == "fd_pwrite");
            let call = Call::new(Operation::Write {
                fd: corpus.input.fd,
                data: STANDARD.decode(&corpus.input.data_base64).unwrap(),
                offset: corpus.input.offset,
            });
            assert_eq!(call.name, corpus.op, "{name}");
            assert_eq!(call.tag, corpus.expected.tag, "{name}");
            assert_eq!(
                call.input,
                STANDARD
                    .decode(&corpus.expected.binary_input_base64)
                    .unwrap(),
                "{name} input"
            );
            let reply = &corpus.expected.reply;
            assert_eq!(reply.kind, "reply");
            assert_eq!(reply.id, 1);
            assert!(!reply.unsupported && reply.message.is_empty());
            let output = reply.output.written.map(Output::Written);
            let encoded_output = output_bytes(output.as_ref());
            assert_eq!(
                encoded_output,
                corpus
                    .expected
                    .binary_output_base64
                    .as_ref()
                    .map(|bytes| STANDARD.decode(bytes).unwrap())
                    .unwrap_or_default(),
                "{name} output"
            );
            let mut transcript = crate::transcript::HEADER.to_vec();
            transcript.extend_from_slice(&crate::transcript::record(
                call.tag,
                reply.id,
                call.input,
                &encoded_output,
                reply.errno,
                &reply.message,
            ));
            assert_eq!(
                transcript,
                STANDARD.decode(&corpus.expected.transcript_base64).unwrap(),
                "{name} complete record"
            );
            assert_eq!(
                format!("{:x}", Sha256::digest(&transcript)),
                corpus.expected.sha256,
                "{name} digest"
            );
        }
    }

    #[test]
    fn typed_admission_retains_lower_layer_callback_and_zero_limit_guards() {
        let call = Call::new(Operation::Write {
            fd: 1,
            data: b"ab".to_vec(),
            offset: Some(9),
        });
        assert_eq!(
            call.admit(2, 1, 0, 0, false).unwrap_err().dimension,
            "callbacks"
        );
        assert!(call.admit(1, 1, 0, 0, true).is_ok());
    }

    #[test]
    fn read_buffer_refusal_precedes_transcript_refusal() {
        let call = Call::new(Operation::Read {
            fd: 99,
            length: (4 << 20) + 1,
            offset: None,
        });
        assert_eq!(
            call.admit(1, 1, 0, 0, false).unwrap_err().dimension,
            "buffer-bytes"
        );
    }
}
