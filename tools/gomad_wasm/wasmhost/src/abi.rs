use crate::{
    engine::State,
    protocol::{BUFFER_LIMIT, Outcome},
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use wasmtime::{Caller, Error, Memory, Result, Val};

pub fn signature(module: &str, name: &str) -> Option<(Vec<&'static str>, Vec<&'static str>)> {
    if module == "gomad_wasm_v1" {
        return match name {
            "config" | "decision" | "observation" | "finish" | "idle" => {
                Some((vec!["i32", "i32"], vec!["i32"]))
            }
            _ => None,
        };
    }
    if module != "wasi_snapshot_preview1" {
        return None;
    }
    let params: &[&str] = match name {
        "sched_yield" => &[],
        "proc_exit" | "fd_close" | "fd_sync" => &["i32"],
        "args_get"
        | "args_sizes_get"
        | "environ_get"
        | "environ_sizes_get"
        | "random_get"
        | "fd_filestat_get"
        | "fd_fdstat_get"
        | "fd_prestat_get"
        | "fd_fdstat_set_flags"
        | "sock_shutdown" => &["i32", "i32"],
        "clock_time_get" => &["i32", "i64", "i32"],
        "fd_filestat_set_size" => &["i32", "i64"],
        "fd_write"
        | "fd_read"
        | "poll_oneoff"
        | "path_create_directory"
        | "path_remove_directory"
        | "path_unlink_file" => match name {
            "path_create_directory" | "path_remove_directory" | "path_unlink_file" => {
                &["i32", "i32", "i32"]
            }
            _ => &["i32", "i32", "i32", "i32"],
        },
        "fd_pread" | "fd_pwrite" | "fd_readdir" => &["i32", "i32", "i32", "i64", "i32"],
        "fd_seek" => &["i32", "i64", "i32", "i32"],
        "fd_prestat_dir_name" | "sock_accept" => &["i32", "i32", "i32"],
        "path_filestat_get" => &["i32", "i32", "i32", "i32", "i32"],
        "path_readlink" | "path_rename" => &["i32", "i32", "i32", "i32", "i32", "i32"],
        "path_symlink" => &["i32", "i32", "i32", "i32", "i32"],
        "path_open" => &[
            "i32", "i32", "i32", "i32", "i32", "i64", "i64", "i32", "i32",
        ],
        _ => return None,
    };
    Some((
        params.to_vec(),
        if name == "proc_exit" {
            vec![]
        } else {
            vec!["i32"]
        },
    ))
}
struct Guest<'a, 'b> {
    caller: &'a mut Caller<'b, State>,
    memory: Memory,
}
impl Guest<'_, '_> {
    fn range(&self, pointer: u32, length: usize) -> Result<std::ops::Range<usize>> {
        if length > BUFFER_LIMIT {
            return Err(Error::msg("ABI buffer capacity"));
        }
        let start = pointer as usize;
        let end = start
            .checked_add(length)
            .ok_or_else(|| Error::msg("guest range overflow"))?;
        if end > self.memory.data_size(&*self.caller) {
            return Err(Error::msg("guest memory range out of bounds"));
        }
        Ok(start..end)
    }
    fn bytes(&self, pointer: u32, length: usize) -> Result<Vec<u8>> {
        let range = self.range(pointer, length)?;
        Ok(self.memory.data(&*self.caller)[range].to_vec())
    }
    fn base64(&self, pointer: u32, length: u32) -> Result<String> {
        Ok(STANDARD.encode(self.bytes(pointer, length as usize)?))
    }
    fn write(&mut self, pointer: u32, bytes: &[u8]) -> Result<()> {
        let range = self.range(pointer, bytes.len())?;
        self.memory.data_mut(&mut *self.caller)[range].copy_from_slice(bytes);
        Ok(())
    }
    fn put32(&mut self, pointer: u32, value: u32) -> Result<()> {
        self.write(pointer, &value.to_le_bytes())
    }
    fn put64(&mut self, pointer: u32, value: u64) -> Result<()> {
        self.write(pointer, &value.to_le_bytes())
    }
    fn iovecs(&self, pointer: u32, count: u32) -> Result<Vec<(u32, usize)>> {
        if count > 4096 {
            return Err(Error::msg("iovec capacity"));
        }
        let descriptors = self.bytes(pointer, count as usize * 8)?;
        let mut total = 0usize;
        let mut result = Vec::new();
        for descriptor in descriptors.chunks_exact(8) {
            let pointer = u32::from_le_bytes(descriptor[..4].try_into().unwrap());
            let length = u32::from_le_bytes(descriptor[4..].try_into().unwrap()) as usize;
            self.range(pointer, length)?;
            total = total
                .checked_add(length)
                .ok_or_else(|| Error::msg("iovec length overflow"))?;
            if total > BUFFER_LIMIT {
                return Err(Error::msg("iovec buffer capacity"));
            }
            result.push((pointer, length));
        }
        Ok(result)
    }
}
fn fields(value: &Value, expected: &[&str]) -> Result<()> {
    let object = value
        .as_object()
        .ok_or_else(|| Error::msg("reply output must be object"))?;
    if object.len() != expected.len() || expected.iter().any(|name| !object.contains_key(*name)) {
        return Err(Error::msg("unexpected reply output fields"));
    }
    Ok(())
}
fn number(value: &Value, name: &str, max: u64) -> Result<u64> {
    let result = value[name]
        .as_u64()
        .ok_or_else(|| Error::msg(format!("invalid unsigned reply field {name}")))?;
    if result > max {
        return Err(Error::msg(format!("reply {name} out of range")));
    }
    Ok(result)
}
fn data(value: &Value, max: usize, exact: bool) -> Result<Vec<u8>> {
    fields(value, &["data_base64"])?;
    let encoded = value["data_base64"]
        .as_str()
        .ok_or_else(|| Error::msg("invalid reply byte string"))?;
    if encoded.len() > BUFFER_LIMIT.div_ceil(3) * 4 {
        return Err(Error::msg("reply buffer capacity"));
    }
    let decoded = STANDARD
        .decode(encoded)
        .map_err(|e| Error::msg(e.to_string()))?;
    if decoded.len() > max || (exact && decoded.len() != max) {
        return Err(Error::msg("reply byte length mismatch"));
    }
    Ok(decoded)
}
fn put(bytes: &mut [u8], offset: usize, value: u64, width: usize) {
    bytes[offset..offset + width].copy_from_slice(&value.to_le_bytes()[..width]);
}
fn stat(value: &Value) -> Result<Vec<u8>> {
    fields(
        value,
        &[
            "dev", "ino", "filetype", "nlink", "size", "atim", "mtim", "ctim",
        ],
    )?;
    let mut bytes = vec![0u8; 64];
    for (name, offset, width) in [
        ("dev", 0, 8),
        ("ino", 8, 8),
        ("filetype", 16, 1),
        ("nlink", 24, 8),
        ("size", 32, 8),
        ("atim", 40, 8),
        ("mtim", 48, 8),
        ("ctim", 56, 8),
    ] {
        put(
            &mut bytes,
            offset,
            number(
                value,
                name,
                if width == 1 { u8::MAX as u64 } else { u64::MAX },
            )?,
            width,
        );
    }
    Ok(bytes)
}
pub fn invoke(
    mut caller: Caller<'_, State>,
    module: &str,
    op: &str,
    params: &[Val],
    results: &mut [Val],
) -> Result<()> {
    if module == "gomad_wasm_v1" {
        return invoke_runtime(caller, op, params, results);
    }
    if op == "proc_exit" {
        let mut outcome = Outcome::new("exit", "");
        outcome.exit_code = Some(params[0].i32().unwrap() as u32);
        caller.data_mut().terminal = Some(outcome);
        return Err(Error::msg("guest proc_exit"));
    }
    let memory = caller
        .get_export("memory")
        .and_then(|e| e.into_memory())
        .ok_or_else(|| Error::msg("missing guest memory"))?;
    let mut guest = Guest {
        caller: &mut caller,
        memory,
    };
    let get = |n: usize| params[n].i32().unwrap() as u32;
    let wide = |n: usize| params[n].i64().unwrap() as u64;
    let mut iovecs = Vec::new();
    let input_result = (|| -> Result<Value> {
        let fd = || json!({"fd":get(0)});
        let path = || -> Result<Value> {
            Ok(json!({"fd":get(0),"path_base64":guest.base64(get(1),get(2))?}))
        };
        Ok(match op {
            "args_sizes_get" | "environ_sizes_get" => {
                guest.range(get(0), 4)?;
                guest.range(get(1), 4)?;
                json!({})
            }
            "args_get" | "environ_get" => {
                let sizes = if op == "args_get" {
                    guest.caller.data().argument_sizes
                } else {
                    guest.caller.data().environment_sizes
                };
                let (count, length) =
                    sizes.ok_or_else(|| Error::msg("string sizes must precede get"))?;
                guest.range(get(0), count as usize * 4)?;
                guest.range(get(1), length as usize)?;
                json!({})
            }
            "random_get" => {
                guest.range(get(0), get(1) as usize)?;
                json!({"length":get(1)})
            }
            "clock_time_get" => {
                guest.range(get(2), 8)?;
                json!({"clock_id":get(0),"precision":wide(1)})
            }
            "sched_yield" => json!({}),
            "fd_write" | "fd_pwrite" | "fd_read" | "fd_pread" => {
                iovecs = guest.iovecs(get(1), get(2))?;
                guest.range(
                    get(if op.ends_with("pread") || op.ends_with("pwrite") {
                        4
                    } else {
                        3
                    }),
                    4,
                )?;
                let mut input = fd();
                if op == "fd_write" || op == "fd_pwrite" {
                    let mut bytes = Vec::new();
                    for &(pointer, length) in &iovecs {
                        bytes.extend(guest.bytes(pointer, length)?);
                    }
                    input["data_base64"] = json!(STANDARD.encode(&bytes));
                    if (get(0) == 1 || get(0) == 2) && (op == "fd_write" || op == "fd_pwrite") {
                        let state = guest.caller.data_mut();
                        let total = state
                            .output_bytes
                            .checked_add(bytes.len() as u64)
                            .ok_or_else(|| Error::msg("output capacity"))?;
                        if total > state.output_limit {
                            return Err(Error::msg("output capacity"));
                        }
                        state.output_bytes = total;
                    }
                } else {
                    input["length"] =
                        json!(iovecs.iter().map(|&(_, length)| length).sum::<usize>());
                }
                if op == "fd_pread" || op == "fd_pwrite" {
                    input["offset"] = json!(wide(3));
                }
                input
            }
            "fd_close" | "fd_sync" => fd(),
            "fd_filestat_set_size" => json!({"fd":get(0),"size":wide(1)}),
            "fd_fdstat_set_flags" => {
                if get(1) > u16::MAX as u32 {
                    return Err(Error::msg("invalid fd flags"));
                }
                json!({"fd":get(0),"flags":get(1)})
            }
            "fd_seek" => {
                guest.range(get(3), 8)?;
                if get(2) > 2 {
                    return Err(Error::msg("invalid whence"));
                }
                json!({"fd":get(0),"offset":params[1].i64().unwrap(),"whence":get(2)})
            }
            "fd_filestat_get" => {
                guest.range(get(1), 64)?;
                fd()
            }
            "fd_fdstat_get" => {
                guest.range(get(1), 24)?;
                fd()
            }
            "fd_prestat_get" => {
                guest.range(get(1), 8)?;
                fd()
            }
            "fd_prestat_dir_name" => {
                guest.range(get(1), get(2) as usize)?;
                json!({"fd":get(0),"length":get(2)})
            }
            "fd_readdir" => {
                guest.range(get(1), get(2) as usize)?;
                guest.range(get(4), 4)?;
                json!({"fd":get(0),"length":get(2),"cookie":wide(3)})
            }
            "path_create_directory" | "path_remove_directory" | "path_unlink_file" => path()?,
            "path_filestat_get" => {
                guest.range(get(4), 64)?;
                json!({"fd":get(0),"flags":get(1),"path_base64":guest.base64(get(2),get(3))?})
            }
            "path_open" => {
                guest.range(get(8), 4)?;
                if get(4) > u16::MAX as u32 || get(7) > u16::MAX as u32 {
                    return Err(Error::msg("invalid path flags"));
                }
                json!({"fd":get(0),"dirflags":get(1),"path_base64":guest.base64(get(2),get(3))?,"oflags":get(4),"rights_base":wide(5),"rights_inheriting":wide(6),"fdflags":get(7)})
            }
            "path_readlink" => {
                guest.range(get(3), get(4) as usize)?;
                guest.range(get(5), 4)?;
                let mut input = path()?;
                input["length"] = json!(get(4));
                input
            }
            "path_rename" => {
                json!({"fd":get(0),"path_base64":guest.base64(get(1),get(2))?,"new_fd":get(3),"new_path_base64":guest.base64(get(4),get(5))?})
            }
            "path_symlink" => {
                json!({"old_path_base64":guest.base64(get(0),get(1))?,"fd":get(2),"path_base64":guest.base64(get(3),get(4))?})
            }
            "sock_accept" => {
                guest.range(get(2), 4)?;
                if get(1) > u16::MAX as u32 {
                    return Err(Error::msg("invalid socket flags"));
                }
                json!({"fd":get(0),"flags":get(1)})
            }
            "sock_shutdown" => {
                if get(1) > 3 {
                    return Err(Error::msg("invalid shutdown mode"));
                }
                json!({"fd":get(0),"how":get(1)})
            }
            "poll_oneoff" => {
                let count = get(2) as usize;
                if count == 0 || count > 4096 {
                    return Err(Error::msg("subscription capacity"));
                }
                let bytes = guest.bytes(get(0), count * 48)?;
                guest.range(get(1), count * 32)?;
                guest.range(get(3), 4)?;
                let mut subscriptions = Vec::new();
                for raw in bytes.chunks_exact(48) {
                    let userdata = u64::from_le_bytes(raw[..8].try_into().unwrap());
                    let kind = raw[8];
                    let sub = match kind {
                        0 => {
                            json!({"userdata":userdata,"type":kind,"clock_id":u32::from_le_bytes(raw[16..20].try_into().unwrap()),"timeout":u64::from_le_bytes(raw[24..32].try_into().unwrap()),"precision":u64::from_le_bytes(raw[32..40].try_into().unwrap()),"flags":u16::from_le_bytes(raw[40..42].try_into().unwrap())})
                        }
                        1 | 2 => {
                            json!({"userdata":userdata,"type":kind,"fd":u32::from_le_bytes(raw[16..20].try_into().unwrap())})
                        }
                        _ => return Err(Error::msg("invalid subscription type")),
                    };
                    subscriptions.push(sub);
                }
                json!({"subscriptions":subscriptions})
            }
            _ => return Err(Error::msg("unsupported import")),
        })
    })();
    let input = match input_result {
        Ok(value) => value,
        Err(error) => {
            let message = error.to_string();
            let kind = if message.contains("capacity") {
                "capacity"
            } else {
                "trap"
            };
            return Err(guest.caller.data_mut().fail(kind, message));
        }
    };
    let reply = guest.caller.data_mut().callback(op, input)?;
    if reply.errno != 0 {
        results[0] = Val::I32(reply.errno as i32);
        return Ok(());
    }
    let output = &reply.output;
    let written = (|| -> Result<()> {
        match op {
            "args_sizes_get" | "environ_sizes_get" => {
                fields(output, &["count", "bytes"])?;
                let count = number(output, "count", (BUFFER_LIMIT / 4) as u64)? as u32;
                let bytes = number(output, "bytes", BUFFER_LIMIT as u64)? as u32;
                guest.put32(get(0), count)?;
                guest.put32(get(1), bytes)?;
                if op == "args_sizes_get" {
                    guest.caller.data_mut().argument_sizes = Some((count, bytes));
                } else {
                    guest.caller.data_mut().environment_sizes = Some((count, bytes));
                }
            }
            "args_get" | "environ_get" => {
                fields(output, &["values"])?;
                let values = output["values"]
                    .as_array()
                    .ok_or_else(|| Error::msg("invalid string list"))?;
                let (count, expected) = if op == "args_get" {
                    guest.caller.data().argument_sizes.unwrap()
                } else {
                    guest.caller.data().environment_sizes.unwrap()
                };
                if values.len() != count as usize {
                    return Err(Error::msg("string count mismatch"));
                }
                let mut bytes = Vec::new();
                let mut pointers = Vec::new();
                for value in values {
                    let decoded = data(&json!({"data_base64":value}), BUFFER_LIMIT, false)?;
                    if decoded.contains(&0) {
                        return Err(Error::msg("NUL in string"));
                    }
                    let pointer = get(1)
                        .checked_add(bytes.len() as u32)
                        .ok_or_else(|| Error::msg("string pointer overflow"))?;
                    pointers.extend_from_slice(&pointer.to_le_bytes());
                    bytes.extend(decoded);
                    bytes.push(0);
                    if bytes.len() > expected as usize {
                        return Err(Error::msg("string byte length mismatch"));
                    }
                }
                if bytes.len() != expected as usize {
                    return Err(Error::msg("string byte length mismatch"));
                }
                guest.write(get(0), &pointers)?;
                guest.write(get(1), &bytes)?;
            }
            "random_get" => guest.write(get(0), &data(output, get(1) as usize, true)?)?,
            "clock_time_get" => {
                fields(output, &["timestamp"])?;
                guest.put64(get(2), number(output, "timestamp", u64::MAX)?)?;
            }
            "fd_write" | "fd_pwrite" => {
                fields(output, &["written"])?;
                let max = iovecs.iter().map(|&(_, length)| length as u64).sum();
                let count = number(output, "written", max)? as u32;
                guest.put32(get(if op == "fd_pwrite" { 4 } else { 3 }), count)?;
            }
            "fd_read" | "fd_pread" => {
                let bytes = data(
                    output,
                    iovecs.iter().map(|&(_, length)| length).sum(),
                    false,
                )?;
                let mut copied = 0;
                for &(pointer, length) in &iovecs {
                    let count = length.min(bytes.len() - copied);
                    guest.write(pointer, &bytes[copied..copied + count])?;
                    copied += count;
                }
                guest.put32(
                    get(if op == "fd_pread" { 4 } else { 3 }),
                    bytes.len() as u32,
                )?;
            }
            "fd_seek" => {
                fields(output, &["offset"])?;
                guest.put64(get(3), number(output, "offset", u64::MAX)?)?;
            }
            "fd_filestat_get" => guest.write(get(1), &stat(output)?)?,
            "path_filestat_get" => guest.write(get(4), &stat(output)?)?,
            "fd_fdstat_get" => {
                fields(
                    output,
                    &["filetype", "flags", "rights_base", "rights_inheriting"],
                )?;
                let mut bytes = vec![0; 24];
                for (name, offset, width, max) in [
                    ("filetype", 0, 1, u8::MAX as u64),
                    ("flags", 2, 2, u16::MAX as u64),
                    ("rights_base", 8, 8, u64::MAX),
                    ("rights_inheriting", 16, 8, u64::MAX),
                ] {
                    put(&mut bytes, offset, number(output, name, max)?, width);
                }
                guest.write(get(1), &bytes)?;
            }
            "fd_prestat_get" => {
                fields(output, &["name_length"])?;
                let mut bytes = vec![0; 8];
                put(
                    &mut bytes,
                    4,
                    number(output, "name_length", BUFFER_LIMIT as u64)?,
                    4,
                );
                guest.write(get(1), &bytes)?;
            }
            "fd_prestat_dir_name" => guest.write(get(1), &data(output, get(2) as usize, true)?)?,
            "fd_readdir" => {
                let bytes = data(output, get(2) as usize, false)?;
                guest.write(get(1), &bytes)?;
                guest.put32(get(4), bytes.len() as u32)?;
            }
            "path_readlink" => {
                let bytes = data(output, get(4) as usize, false)?;
                guest.write(get(3), &bytes)?;
                guest.put32(get(5), bytes.len() as u32)?;
            }
            "path_open" | "sock_accept" => {
                fields(output, &["fd"])?;
                guest.put32(
                    get(if op == "path_open" { 8 } else { 2 }),
                    number(output, "fd", u32::MAX as u64)? as u32,
                )?;
            }
            "poll_oneoff" => {
                fields(output, &["events"])?;
                let events = output["events"]
                    .as_array()
                    .ok_or_else(|| Error::msg("invalid event array"))?;
                if events.len() > get(2) as usize {
                    return Err(Error::msg("event count exceeds subscriptions"));
                }
                let mut bytes = vec![0; events.len() * 32];
                for (index, event) in events.iter().enumerate() {
                    fields(event, &["userdata", "errno", "type", "nbytes", "flags"])?;
                    let raw = &mut bytes[index * 32..(index + 1) * 32];
                    for (name, offset, width, max) in [
                        ("userdata", 0, 8, u64::MAX),
                        ("errno", 8, 2, u16::MAX as u64),
                        ("type", 10, 1, 2),
                        ("nbytes", 16, 8, u64::MAX),
                        ("flags", 24, 2, u16::MAX as u64),
                    ] {
                        put(raw, offset, number(event, name, max)?, width);
                    }
                }
                guest.write(get(1), &bytes)?;
                guest.put32(get(3), events.len() as u32)?;
            }
            _ => fields(output, &[])?,
        }
        Ok(())
    })();
    if let Err(error) = written {
        return Err(guest.caller.data_mut().fail("invalid", error.to_string()));
    }
    results[0] = Val::I32(0);
    Ok(())
}

fn invoke_runtime(
    mut caller: Caller<'_, State>,
    op: &str,
    params: &[Val],
    results: &mut [Val],
) -> Result<()> {
    let memory = caller
        .get_export("memory")
        .and_then(|export| export.into_memory())
        .ok_or_else(|| Error::msg("missing guest memory"))?;
    let mut guest = Guest {
        caller: &mut caller,
        memory,
    };
    let pointer = params[0].i32().unwrap() as u32;
    let length = params[1].i32().unwrap() as u32 as usize;
    let input = (|| -> Result<Value> {
        let maximum = match op {
            "config" | "finish" | "idle" => 16,
            "decision" => 32 + 256 * 32 + 96,
            "observation" => 192,
            _ => return Err(Error::msg("unsupported runtime import")),
        };
        if length > maximum {
            return Err(Error::msg("runtime ABI buffer capacity"));
        }
        let valid_length = match op {
            "decision" => length >= 96 && (length - 32).is_multiple_of(32),
            "observation" => length == 96 || length == 192,
            _ => length == 16,
        };
        if !valid_length {
            return Err(Error::msg("invalid runtime ABI buffer length"));
        }
        let bytes = guest.bytes(pointer, length)?;
        if op == "decision" {
            let count = u32::from_be_bytes(bytes[20..24].try_into().unwrap()) as usize;
            if !(2..=256).contains(&count)
                || (length != 32 + count * 32 && length != 32 + count * 32 + 96)
            {
                return Err(Error::msg("invalid runtime decision buffer layout"));
            }
        }
        Ok(json!({"data_base64":STANDARD.encode(bytes)}))
    })();
    let input = match input {
        Ok(input) => input,
        Err(error) => {
            let message = error.to_string();
            let kind = if message.contains("capacity") {
                "capacity"
            } else {
                "trap"
            };
            return Err(guest.caller.data_mut().fail(kind, message));
        }
    };
    let reply = guest
        .caller
        .data_mut()
        .callback(&format!("runtime_{op}"), input)?;
    if reply.errno != 0 {
        results[0] = Val::I32(reply.errno as i32);
        return Ok(());
    }
    let written = (|| -> Result<()> {
        fields(&reply.output, &["data_base64"])?;
        let encoded = reply.output["data_base64"]
            .as_str()
            .ok_or_else(|| Error::msg("invalid runtime reply byte string"))?;
        if encoded.len() > length.div_ceil(3) * 4 {
            return Err(Error::msg("runtime reply buffer length mismatch"));
        }
        guest.write(pointer, &data(&reply.output, length, true)?)
    })();
    if let Err(error) = written {
        return Err(guest.caller.data_mut().fail("invalid", error.to_string()));
    }
    results[0] = Val::I32(0);
    Ok(())
}
