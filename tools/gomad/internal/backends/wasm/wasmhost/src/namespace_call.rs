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
    pub fn new(operation: Operation) -> Self {
        let mut fields = Fields::default();
        let (tag, name, length, output_reservation) = match &operation {
            Operation::Write { fd, data, offset } => {
                fields.number("fd", *fd as u64, 4);
                fields.bytes("data_base64", data);
                if let Some(offset) = offset {
                    fields.number("offset", *offset, 8);
                    (9, "fd_pwrite", 0, 4)
                } else {
                    (8, "fd_write", 0, 4)
                }
            }
            Operation::Read { fd, length, offset } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("length", *length as u64, 4);
                if let Some(offset) = offset {
                    fields.number("offset", *offset, 8);
                    (11, "fd_pread", *length as u64, *length as u64 + 4)
                } else {
                    (10, "fd_read", *length as u64, *length as u64 + 4)
                }
            }
            Operation::Close(fd)
            | Operation::Sync(fd)
            | Operation::FilestatGet(fd)
            | Operation::FdstatGet(fd)
            | Operation::PrestatGet(fd) => {
                fields.number("fd", *fd as u64, 4);
                match &operation {
                    Operation::Close(_) => (12, "fd_close", 0, 0),
                    Operation::Sync(_) => (13, "fd_sync", 0, 0),
                    Operation::FilestatGet(_) => (14, "fd_filestat_get", 0, 57),
                    Operation::FdstatGet(_) => (15, "fd_fdstat_get", 0, 19),
                    _ => (16, "fd_prestat_get", 0, 4),
                }
            }
            Operation::PrestatDirName { fd, length } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("length", *length as u64, 4);
                (
                    17,
                    "fd_prestat_dir_name",
                    *length as u64,
                    *length as u64 + 4,
                )
            }
            Operation::SetFlags { fd, flags } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("flags", *flags as u64, 2);
                (
                    18,
                    "fd_fdstat_set_flags",
                    0,
                    4 + crate::namespace::FD_FLAGS_UNSUPPORTED.len() as u64,
                )
            }
            Operation::SetSize { fd, size } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("size", *size, 8);
                (19, "fd_filestat_set_size", 0, 0)
            }
            Operation::Seek { fd, offset, whence } => {
                fields.number("fd", *fd as u64, 4);
                fields.signed("offset", *offset);
                fields.number("whence", *whence as u64, 1);
                (20, "fd_seek", 0, 8)
            }
            Operation::Readdir { fd, length, cookie } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("length", *length as u64, 4);
                fields.number("cookie", *cookie, 8);
                (21, "fd_readdir", *length as u64, *length as u64 + 4)
            }
            Operation::CreateDirectory { fd, path }
            | Operation::RemoveDirectory { fd, path }
            | Operation::UnlinkFile { fd, path } => {
                fields.number("fd", *fd as u64, 4);
                fields.bytes("path_base64", path);
                match &operation {
                    Operation::CreateDirectory { .. } => (22, "path_create_directory", 0, 0),
                    Operation::RemoveDirectory { .. } => (23, "path_remove_directory", 0, 0),
                    _ => (24, "path_unlink_file", 0, 0),
                }
            }
            Operation::PathStat { fd, path, flags } => {
                fields.number("fd", *fd as u64, 4);
                fields.number("flags", *flags as u64, 4);
                fields.bytes("path_base64", path);
                (25, "path_filestat_get", 0, 57)
            }
            Operation::Open(open) => {
                fields.number("fd", open.fd as u64, 4);
                fields.number("dirflags", open.dirflags as u64, 4);
                fields.bytes("path_base64", &open.path);
                fields.number("oflags", open.oflags as u64, 2);
                fields.number("rights_base", open.base, 8);
                fields.number("rights_inheriting", open.inheriting, 8);
                fields.number("fdflags", open.flags as u64, 2);
                (
                    26,
                    "path_open",
                    0,
                    4 + crate::namespace::OPEN_FLAGS_UNSUPPORTED.len() as u64,
                )
            }
            Operation::Readlink { fd, path, length } => {
                fields.number("fd", *fd as u64, 4);
                fields.bytes("path_base64", path);
                fields.number("length", *length as u64, 4);
                (
                    27,
                    "path_readlink",
                    *length as u64,
                    (*length as u64 + 4)
                        .max(4 + crate::namespace::READLINK_UNSUPPORTED.len() as u64),
                )
            }
            Operation::Rename {
                fd,
                path,
                new_fd,
                new_path,
            } => {
                fields.number("fd", *fd as u64, 4);
                fields.bytes("path_base64", path);
                fields.number("new_fd", *new_fd as u64, 4);
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
                fields.number("fd", *fd as u64, 4);
                fields.bytes("path_base64", path);
                (
                    29,
                    "path_symlink",
                    0,
                    4 + "operation outside stock WASI profile: path_symlink".len() as u64,
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
        value.ilog10() as u64 + 1
    }
}
#[derive(Default)]
pub(crate) struct Fields {
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
    pub(crate) fn number(&mut self, name: &str, value: u64, width: usize) {
        self.field(name, digits(value));
        self.binary.extend_from_slice(&value.to_le_bytes()[..width]);
    }
    pub(crate) fn json_size(&self) -> u64 {
        self.json_bytes + 2
    }
    fn signed(&mut self, name: &str, value: i64) {
        self.field(name, digits(value.unsigned_abs()) + u64::from(value < 0));
        self.binary.extend_from_slice(&value.to_le_bytes());
    }
    fn bytes(&mut self, name: &str, value: &[u8]) {
        self.field(name, 2 + (value.len() as u64).div_ceil(3) * 4);
        self.max_buffer = self.max_buffer.max(value.len() as u64);
        self.binary
            .extend_from_slice(&(value.len() as u32).to_le_bytes());
        self.binary.extend_from_slice(value);
    }
}
pub fn admit_poll_frames(id: u64, input_bytes: u64) -> Result<(), Boundary> {
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
pub fn output_bytes(output: Option<&Output>) -> Vec<u8> {
    let mut bytes = Vec::new();
    match output {
        None | Some(Output::Empty) => (),
        Some(Output::Prestat(n) | Output::Fd(n) | Output::Written(n)) => {
            bytes.extend_from_slice(&n.to_le_bytes())
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
    fn unsupported_messages_fit_exact_shared_go_reservations() {
        for (operation, limit) in [
            (Operation::SetFlags { fd: 3, flags: 2 }, 75),
            (
                Operation::Open(crate::namespace::Open {
                    fd: 3,
                    path: b"x".to_vec(),
                    dirflags: 0,
                    oflags: 0,
                    base: 2,
                    inheriting: 0,
                    flags: 2,
                }),
                96,
            ),
            (
                Operation::Readlink {
                    fd: 3,
                    path: b"x".to_vec(),
                    length: 0,
                },
                92,
            ),
        ] {
            let call = Call::new(operation);
            assert_eq!(
                call.admit(1, 1, 0, limit - 1, false).unwrap_err(),
                Boundary {
                    kind: "capacity",
                    dimension: "transcript-bytes"
                }
            );
            assert!(call.admit(1, 1, 0, limit, false).is_ok());
            assert!(call.admit(1, 1, 0, 65536, false).is_ok());
            assert!(call.admit(1, 1, 0, 1, true).is_ok());
        }
    }
    #[test]
    fn typed_call_preserves_binary_fields_and_go_admission_order() {
        let call = Call::new(Operation::Write {
            fd: 1,
            data: b"ab".to_vec(),
            offset: Some(9),
        });
        assert_eq!(call.tag, 9);
        assert_eq!(call.name, "fd_pwrite");
        assert_eq!(
            call.input,
            [1, 0, 0, 0, 2, 0, 0, 0, b'a', b'b', 9, 0, 0, 0, 0, 0, 0, 0]
        );
        assert_eq!(
            call.admit(2, 1, 0, 0, false).unwrap_err().dimension,
            "callbacks"
        );
        assert_eq!(
            call.admit(1, 1, 0, 23, false).unwrap_err().dimension,
            "transcript-bytes"
        );
        assert!(call.admit(1, 1, 0, 46, false).is_ok());
        assert_eq!(
            call.admit(1, 1, 0, 45, false).unwrap_err().dimension,
            "transcript-bytes"
        );
        assert!(call.admit(1, 1, 0, 0, true).is_ok());
        assert_eq!(output_bytes(Some(&Output::Written(2))), [2, 0, 0, 0]);
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
    #[test]
    fn two_large_paths_refuse_callback_before_transcript_capacity() {
        let call = Call::new(Operation::Rename {
            fd: 3,
            path: vec![b'x'; 4 << 20],
            new_fd: 3,
            new_path: vec![b'y'; 4 << 20],
        });
        assert_eq!(
            call.admit(1, 1, 0, 0, false).unwrap_err().dimension,
            "callback-frame"
        );
    }
}
