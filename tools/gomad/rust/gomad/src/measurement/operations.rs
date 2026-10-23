use std::collections::BTreeMap;

const OPERATIONS: &[(&str, &str, &str)] = &[
    ("", "", ""),
    (
        "clock_time_get",
        "clock_id:u32 precision:u64",
        "timestamp:u64",
    ),
    ("sched_yield", "", ""),
    ("args_sizes_get", "", ""),
    ("args_get", "", ""),
    ("environ_sizes_get", "", ""),
    ("environ_get", "", ""),
    ("random_get", "length:u32 offset:u64", ""),
    ("fd_write", "fd:u32 data_base64:bytes", "written:u32"),
    (
        "fd_pwrite",
        "fd:u32 data_base64:bytes offset:u64",
        "written:u32",
    ),
    ("fd_read", "fd:u32 length:u32", "data_base64:bytes"),
    (
        "fd_pread",
        "fd:u32 length:u32 offset:u64",
        "data_base64:bytes",
    ),
    ("fd_close", "fd:u32", ""),
    ("fd_sync", "fd:u32", ""),
    (
        "fd_filestat_get",
        "fd:u32",
        "dev:u64 ino:u64 filetype:u8 nlink:u64 size:u64 atim:u64 mtim:u64 ctim:u64",
    ),
    (
        "fd_fdstat_get",
        "fd:u32",
        "filetype:u8 flags:u16 rights_base:u64 rights_inheriting:u64",
    ),
    ("fd_prestat_get", "fd:u32", "name_length:u32"),
    (
        "fd_prestat_dir_name",
        "fd:u32 length:u32",
        "data_base64:bytes",
    ),
    ("fd_fdstat_set_flags", "fd:u32 flags:u16", ""),
    ("fd_filestat_set_size", "fd:u32 size:u64", ""),
    ("fd_seek", "fd:u32 offset:i64 whence:u8", "offset:u64"),
    (
        "fd_readdir",
        "fd:u32 length:u32 cookie:u64",
        "data_base64:bytes",
    ),
    ("path_create_directory", "fd:u32 path_base64:bytes", ""),
    ("path_remove_directory", "fd:u32 path_base64:bytes", ""),
    ("path_unlink_file", "fd:u32 path_base64:bytes", ""),
    (
        "path_filestat_get",
        "fd:u32 flags:u32 path_base64:bytes",
        "dev:u64 ino:u64 filetype:u8 nlink:u64 size:u64 atim:u64 mtim:u64 ctim:u64",
    ),
    (
        "path_open",
        "fd:u32 dirflags:u32 path_base64:bytes oflags:u16 rights_base:u64 rights_inheriting:u64 fdflags:u16",
        "fd:u32",
    ),
    (
        "path_readlink",
        "fd:u32 path_base64:bytes length:u32",
        "data_base64:bytes",
    ),
    (
        "path_rename",
        "fd:u32 path_base64:bytes new_fd:u32 new_path_base64:bytes",
        "",
    ),
    (
        "path_symlink",
        "old_path_base64:bytes fd:u32 path_base64:bytes",
        "",
    ),
    ("sock_accept", "fd:u32 flags:u16", "fd:u32"),
    ("sock_shutdown", "fd:u32 how:u8", ""),
    (
        "poll_oneoff",
        "subscriptions:subscriptions",
        "events:events",
    ),
    ("", "", ""),
    ("unsupported_operation", "", ""),
];

#[derive(Debug, Default, PartialEq, Eq)]
pub struct Counts {
    pub calls: u64,
    pub local: u64,
    pub remote: u64,
    pub operations: BTreeMap<String, u64>,
}
struct Reader<'a>(&'a [u8]);
impl<'a> Reader<'a> {
    fn take(&mut self, length: usize) -> Result<&'a [u8], String> {
        if length > self.0.len() {
            return Err("truncated canonical transcript".into());
        }
        let (bytes, rest) = self.0.split_at(length);
        self.0 = rest;
        Ok(bytes)
    }
    fn number(&mut self, width: usize) -> Result<u64, String> {
        let mut word = [0u8; 8];
        word[..width].copy_from_slice(self.take(width)?);
        Ok(u64::from_le_bytes(word))
    }
    fn sized(&mut self, limit: usize) -> Result<&'a [u8], String> {
        let length = usize::try_from(self.number(4)?).map_err(|error| error.to_string())?;
        if length > limit {
            return Err("canonical transcript field capacity".into());
        }
        self.take(length)
    }
    fn object(
        &mut self,
        schema: &str,
        fd: &mut Option<u32>,
        local_poll: &mut bool,
    ) -> Result<(), String> {
        for field in schema.split_whitespace() {
            let (name, kind) = field.split_once(':').ok_or("invalid canonical schema")?;
            match kind {
                "bytes" => {
                    self.sized(4 << 20)?;
                }
                "subscriptions" | "events" => {
                    let count = self.number(4)?;
                    if count > 4096 {
                        return Err("canonical transcript array capacity".into());
                    }
                    for _ in 0..count {
                        if kind == "subscriptions" {
                            let tag = *self.0.get(8).ok_or("truncated canonical subscription")?;
                            match tag {
                                0 => {
                                    self.take(31)?;
                                }
                                1 | 2 => {
                                    *local_poll = false;
                                    self.take(13)?;
                                }
                                _ => return Err("invalid canonical subscription".into()),
                            }
                        } else {
                            let event = self.take(21)?;
                            if event[10] > 2 {
                                return Err("invalid canonical event".into());
                            }
                        }
                    }
                }
                _ => {
                    let width = match kind {
                        "u8" => 1,
                        "u16" => 2,
                        "u32" => 4,
                        "u64" | "i64" => 8,
                        _ => return Err("invalid canonical field".into()),
                    };
                    let value = self.number(width)?;
                    if name == "fd" {
                        *fd = Some(u32::try_from(value).map_err(|error| error.to_string())?);
                    }
                }
            }
        }
        Ok(())
    }
}
fn add(total: &mut u64, count: u64) -> Result<(), String> {
    *total = total
        .checked_add(count)
        .ok_or("canonical operation count overflow")?;
    Ok(())
}

/// # Errors
/// Validates the entire retained binary trace before counting operation ownership.
pub fn count_with_unknown(
    data: &[u8],
    maximum: u64,
    unknown_input: impl Fn(&[u8]) -> Result<Option<u32>, String>,
) -> Result<Counts, String> {
    if u64::try_from(data.len()).map_err(|error| error.to_string())? > maximum
        || data.len() > 1 << 30
    {
        return Err("retained canonical byte bound".into());
    }
    let mut result = Counts::default();
    if data.is_empty() {
        return Ok(result);
    }
    let mut reader = Reader(data);
    if reader.take(8)? != b"GMWT\x01\0\0\0" {
        return Err("invalid canonical transcript header".into());
    }
    let mut previous = 0;
    while !reader.0.is_empty() {
        let header = reader.take(16)?;
        let tag = usize::from(header[0]);
        let flags = header[1];
        if tag == 0 || tag >= OPERATIONS.len() || flags > 1 {
            return Err("invalid canonical transcript record".into());
        }
        let errno = u16::from_le_bytes(header[2..4].try_into().map_err(|_| "invalid errno")?);
        let id = u64::from_le_bytes(header[4..12].try_into().map_err(|_| "invalid id")?);
        if id == 0 || id <= previous {
            return Err("nonincreasing canonical transcript ordinal".into());
        }
        let length = u32::from_le_bytes(header[12..16].try_into().map_err(|_| "invalid length")?);
        if length > (8 << 20) - 16 {
            return Err("canonical transcript record capacity".into());
        }
        let body = reader.take(usize::try_from(length).map_err(|error| error.to_string())?)?;
        let mut payload = Reader(body);
        let (mut name, input, output) = OPERATIONS[tag];
        let mut fd = None;
        let mut local_poll = true;
        let mut count = 1;
        previous = id;
        let unknown;
        if tag == 33 {
            if flags != 0 || errno != 0 || body.len() != 28 {
                return Err("invalid canonical clock run".into());
            }
            let clock = payload.number(4)?;
            payload.number(8)?;
            let now = payload.number(8)?;
            count = payload.number(8)?;
            let last = id
                .checked_add(count)
                .and_then(|value| value.checked_sub(1))
                .ok_or("clock ordinal overflow")?;
            if clock > 1 || count == 0 || count > 1 << 28 || now > i64::MAX as u64 || last > 1 << 28
            {
                return Err("invalid canonical clock run".into());
            }
            previous = last;
            name = "clock_time_get";
        } else {
            if tag == 34 {
                if flags != 1 {
                    return Err("unmarked canonical unsupported operation".into());
                }
                unknown = std::str::from_utf8(payload.sized(64 << 10)?)
                    .map_err(|error| error.to_string())?;
                if OPERATIONS.iter().any(|(name, _, _)| {
                    !name.is_empty() && *name == unknown && *name != "unsupported_operation"
                }) {
                    return Err("known operation in unsupported canonical record".into());
                }
                let raw = payload.sized(8 << 20)?;
                fd = unknown_input(raw)?;
                name = unknown;
            }
            let mut input_fd = fd;
            payload.object(input, &mut input_fd, &mut local_poll)?;
            fd = input_fd;
            if tag == 1
                && (flags != 1
                    || body
                        .get(..4)
                        .is_none_or(|bytes| u32::from_le_bytes(bytes.try_into().unwrap()) <= 1))
            {
                return Err("invalid canonical unsupported clock".into());
            }
            if flags == 1 {
                std::str::from_utf8(payload.sized(64 << 10)?).map_err(|error| error.to_string())?;
            } else if errno == 0 && !matches!(tag, 3..=7) {
                let mut ignored_fd = None;
                payload.object(output, &mut ignored_fd, &mut local_poll)?;
            }
        }
        if !payload.0.is_empty() {
            return Err("trailing canonical transcript data".into());
        }
        add(&mut result.calls, count)?;
        let local = matches!(
            name,
            "clock_time_get"
                | "sched_yield"
                | "random_get"
                | "args_sizes_get"
                | "args_get"
                | "environ_sizes_get"
                | "environ_get"
        ) || name == "poll_oneoff" && local_poll;
        if local {
            add(&mut result.local, count)?;
        } else {
            add(&mut result.remote, count)?;
            let fd = fd.map_or_else(|| "no-root-fd".into(), |fd| format!("fd={fd}"));
            add(
                result
                    .operations
                    .entry(format!("wasi_snapshot_preview1/{name}/{fd}"))
                    .or_default(),
                count,
            )?;
        }
    }
    Ok(result)
}

#[cfg(test)]
mod tests {
    use super::*;
    fn count(data: &[u8], maximum: u64) -> Result<Counts, String> {
        count_with_unknown(data, maximum, |_| {
            Err("unknown JSON validated separately by the native collector".into())
        })
    }
    fn record(id: u64, tag: u8, body: &[u8]) -> Vec<u8> {
        let mut value = vec![tag, 0, 0, 0];
        value.extend_from_slice(&id.to_le_bytes());
        value.extend_from_slice(&u32::try_from(body.len()).unwrap().to_le_bytes());
        value.extend_from_slice(body);
        value
    }
    fn trace(records: impl IntoIterator<Item = Vec<u8>>) -> Vec<u8> {
        let mut data = b"GMWT\x01\0\0\0".to_vec();
        for record in records {
            data.extend(record);
        }
        data
    }
    #[test]
    fn original_operation_count_golden_keeps_folded_ordinals_and_input_immutable() {
        let mut clock = vec![0u8; 28];
        clock[12] = 1;
        clock[20] = 2;
        let mut clock_poll = vec![0u8; 39];
        clock_poll[0] = 1;
        let mut mixed_poll = vec![0u8; 52];
        mixed_poll[0] = 2;
        mixed_poll[43] = 1;
        mixed_poll[44] = 4;
        let mut stdout = vec![0u8; 12];
        stdout[0] = 1;
        let mut stderr = stdout.clone();
        stderr[0] = 2;
        let mut pwrite = vec![0u8; 20];
        pwrite[0] = 4;
        let mut open = vec![0u8; 36];
        open[0] = 3;
        let data = trace([
            record(1, 33, &clock),
            record(3, 2, &[]),
            record(4, 32, &clock_poll),
            record(5, 32, &mixed_poll),
            record(6, 8, &stdout),
            record(7, 8, &stderr),
            record(8, 9, &pwrite),
            record(9, 26, &open),
        ]);
        let saved = data.clone();
        let result = count(&data, data.len() as u64).unwrap();
        assert_eq!(data, saved);
        assert_eq!((result.calls, result.local, result.remote), (9, 4, 5));
        for name in [
            "poll_oneoff/no-root-fd",
            "fd_write/fd=1",
            "fd_write/fd=2",
            "fd_pwrite/fd=4",
            "path_open/fd=3",
        ] {
            assert_eq!(
                result
                    .operations
                    .get(&format!("wasi_snapshot_preview1/{name}")),
                Some(&1)
            );
        }
    }
    #[test]
    fn full_payload_bounds_sequence_and_unknown_tags_refuse() {
        let mut body = vec![0u8; 12];
        body[0] = 1;
        let valid = trace([record(1, 8, &body)]);
        count(&valid, valid.len() as u64).unwrap();
        let mut trailing = valid.clone();
        trailing.push(0);
        let mut duplicate = valid.clone();
        duplicate.extend_from_slice(&valid[8..]);
        let mut bad_length = vec![0u8; 8];
        bad_length[0] = 1;
        bad_length[4..].fill(255);
        for data in [
            valid[..valid.len() - 1].to_vec(),
            trailing,
            duplicate,
            trace([record(1, 255, &[])]),
            trace([record(1, 8, &bad_length)]),
            trace([record(1, 32, &[255; 4])]),
            br#"{"type":"call","id":1,"op":"sched_yield","input":{}}"#.to_vec(),
        ] {
            assert!(count(&data, data.len() as u64).is_err());
        }
        assert!(count(&valid, valid.len() as u64 - 1).is_err());
    }
    #[test]
    fn clock_fold_prevents_reuse_and_oversize_count() {
        let mut clock = vec![0u8; 28];
        clock[20] = 2;
        assert!(count(&trace([record(1, 33, &clock), record(2, 2, &[])]), 1024).is_err());
        clock[20..28].copy_from_slice(&((1u64 << 28) + 1).to_le_bytes());
        assert!(count(&trace([record(1, 33, &clock)]), 1024).is_err());
    }
}
