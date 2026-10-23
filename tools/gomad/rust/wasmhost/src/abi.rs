use crate::{
    engine::State,
    protocol::{BUFFER_LIMIT, CallbackInput, Outcome},
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use wasmtime::{Caller, Error, Memory, Result, Val};

macro_rules! val {
    (i32, $value:ident) => {
        Val::I32($value)
    };
    (i64, $value:ident) => {
        Val::I64($value)
    };
}
macro_rules! invoke_typed {
    ($caller:ident, $op:expr, [$($param:expr),*], i32) => {{
        let mut result = [Val::I32(0)];
        invoke($caller, $op, &[$($param),*], &mut result)?;
        Ok(result[0].i32().unwrap())
    }};
    ($caller:ident, $op:expr, [$($param:expr),*], ()) => {{
        invoke($caller, $op, &[$($param),*], &mut [])
    }};
}
macro_rules! operations {
    ($($variant:ident: $module:literal, $name:literal, $callback:literal, ($($arg:ident: $ty:ident),*) -> $result:tt;)+) => {
        #[derive(Clone, Copy, Debug, PartialEq, Eq)]
        pub enum Operation { $($variant,)+ }
        impl Operation {
            #[cfg(any(test, feature = "diagnostic-profile"))]
            pub const ALL: &'static [Self] = &[$(Self::$variant,)+];
            #[cfg(feature = "diagnostic-profile")]
            pub const COUNT: usize = Self::ALL.len();
            pub fn resolve(module: &str, name: &str) -> Option<Self> {
                match (module, name) {
                    $(($module, $name) => Some(Self::$variant),)+
                    _ => None,
                }
            }
            pub fn signature(self) -> (&'static [&'static str], &'static [&'static str]) {
                match self {
                    $(Self::$variant => (&[$(stringify!($ty)),*], result_types!($result)),)+
                }
            }
            #[cfg(feature = "diagnostic-profile")]
            pub fn module(self) -> &'static str {
                match self { $(Self::$variant => $module,)+ }
            }
            #[cfg(feature = "diagnostic-profile")]
            pub fn name(self) -> &'static str {
                match self { $(Self::$variant => $name,)+ }
            }
            pub fn callback(self) -> &'static str {
                match self { $(Self::$variant => $callback,)+ }
            }
            pub fn link(self, linker: &mut wasmtime::Linker<State>) -> Result<()> {
                match self {
                    $(Self::$variant => {
                        linker.func_wrap($module, $name, |caller: Caller<'_, State>, $($arg: $ty),*| -> Result<$result> {
                            #[cfg(feature = "diagnostic-profile")]
                            let _boundary = crate::diagnostic::ImportBoundary::new();
                            invoke_typed!(caller, Operation::$variant, [$(val!($ty, $arg)),*], $result)
                        })?;
                    },)+
                }
                Ok(())
            }
        }
    };
}
macro_rules! result_types {
    (i32) => {
        &["i32"]
    };
    (()) => {
        &[]
    };
}
operations! {
    SchedYield: "wasi_snapshot_preview1", "sched_yield", "sched_yield", () -> i32;
    ProcExit: "wasi_snapshot_preview1", "proc_exit", "proc_exit", (p0: i32) -> ();
    FdClose: "wasi_snapshot_preview1", "fd_close", "fd_close", (p0: i32) -> i32;
    FdSync: "wasi_snapshot_preview1", "fd_sync", "fd_sync", (p0: i32) -> i32;
    ArgsGet: "wasi_snapshot_preview1", "args_get", "args_get", (p0: i32, p1: i32) -> i32;
    ArgsSizesGet: "wasi_snapshot_preview1", "args_sizes_get", "args_sizes_get", (p0: i32, p1: i32) -> i32;
    EnvironGet: "wasi_snapshot_preview1", "environ_get", "environ_get", (p0: i32, p1: i32) -> i32;
    EnvironSizesGet: "wasi_snapshot_preview1", "environ_sizes_get", "environ_sizes_get", (p0: i32, p1: i32) -> i32;
    RandomGet: "wasi_snapshot_preview1", "random_get", "random_get", (p0: i32, p1: i32) -> i32;
    FdFilestatGet: "wasi_snapshot_preview1", "fd_filestat_get", "fd_filestat_get", (p0: i32, p1: i32) -> i32;
    FdFdstatGet: "wasi_snapshot_preview1", "fd_fdstat_get", "fd_fdstat_get", (p0: i32, p1: i32) -> i32;
    FdPrestatGet: "wasi_snapshot_preview1", "fd_prestat_get", "fd_prestat_get", (p0: i32, p1: i32) -> i32;
    FdFdstatSetFlags: "wasi_snapshot_preview1", "fd_fdstat_set_flags", "fd_fdstat_set_flags", (p0: i32, p1: i32) -> i32;
    SockShutdown: "wasi_snapshot_preview1", "sock_shutdown", "sock_shutdown", (p0: i32, p1: i32) -> i32;
    ClockTimeGet: "wasi_snapshot_preview1", "clock_time_get", "clock_time_get", (p0: i32, p1: i64, p2: i32) -> i32;
    FdFilestatSetSize: "wasi_snapshot_preview1", "fd_filestat_set_size", "fd_filestat_set_size", (p0: i32, p1: i64) -> i32;
    FdWrite: "wasi_snapshot_preview1", "fd_write", "fd_write", (p0: i32, p1: i32, p2: i32, p3: i32) -> i32;
    FdRead: "wasi_snapshot_preview1", "fd_read", "fd_read", (p0: i32, p1: i32, p2: i32, p3: i32) -> i32;
    PollOneoff: "wasi_snapshot_preview1", "poll_oneoff", "poll_oneoff", (p0: i32, p1: i32, p2: i32, p3: i32) -> i32;
    PathCreateDirectory: "wasi_snapshot_preview1", "path_create_directory", "path_create_directory", (p0: i32, p1: i32, p2: i32) -> i32;
    PathRemoveDirectory: "wasi_snapshot_preview1", "path_remove_directory", "path_remove_directory", (p0: i32, p1: i32, p2: i32) -> i32;
    PathUnlinkFile: "wasi_snapshot_preview1", "path_unlink_file", "path_unlink_file", (p0: i32, p1: i32, p2: i32) -> i32;
    FdPrestatDirName: "wasi_snapshot_preview1", "fd_prestat_dir_name", "fd_prestat_dir_name", (p0: i32, p1: i32, p2: i32) -> i32;
    SockAccept: "wasi_snapshot_preview1", "sock_accept", "sock_accept", (p0: i32, p1: i32, p2: i32) -> i32;
    FdPread: "wasi_snapshot_preview1", "fd_pread", "fd_pread", (p0: i32, p1: i32, p2: i32, p3: i64, p4: i32) -> i32;
    FdPwrite: "wasi_snapshot_preview1", "fd_pwrite", "fd_pwrite", (p0: i32, p1: i32, p2: i32, p3: i64, p4: i32) -> i32;
    FdReaddir: "wasi_snapshot_preview1", "fd_readdir", "fd_readdir", (p0: i32, p1: i32, p2: i32, p3: i64, p4: i32) -> i32;
    FdSeek: "wasi_snapshot_preview1", "fd_seek", "fd_seek", (p0: i32, p1: i64, p2: i32, p3: i32) -> i32;
    PathFilestatGet: "wasi_snapshot_preview1", "path_filestat_get", "path_filestat_get", (p0: i32, p1: i32, p2: i32, p3: i32, p4: i32) -> i32;
    PathSymlink: "wasi_snapshot_preview1", "path_symlink", "path_symlink", (p0: i32, p1: i32, p2: i32, p3: i32, p4: i32) -> i32;
    PathReadlink: "wasi_snapshot_preview1", "path_readlink", "path_readlink", (p0: i32, p1: i32, p2: i32, p3: i32, p4: i32, p5: i32) -> i32;
    PathRename: "wasi_snapshot_preview1", "path_rename", "path_rename", (p0: i32, p1: i32, p2: i32, p3: i32, p4: i32, p5: i32) -> i32;
    PathOpen: "wasi_snapshot_preview1", "path_open", "path_open", (p0: i32, p1: i32, p2: i32, p3: i32, p4: i32, p5: i64, p6: i64, p7: i32, p8: i32) -> i32;
    RuntimeConfig: "gomad_wasm_v1", "config", "runtime_config", (p0: i32, p1: i32) -> i32;
    RuntimeDecision: "gomad_wasm_v1", "decision", "runtime_decision", (p0: i32, p1: i32) -> i32;
    RuntimeObservation: "gomad_wasm_v1", "observation", "runtime_observation", (p0: i32, p1: i32) -> i32;
    RuntimeFinish: "gomad_wasm_v1", "finish", "runtime_finish", (p0: i32, p1: i32) -> i32;
    RuntimeIdle: "gomad_wasm_v1", "idle", "runtime_idle", (p0: i32, p1: i32) -> i32;
    RuntimeNetStream: "gomad_wasm_v2", "net_stream", "runtime_net_stream", (p0: i32, p1: i32) -> i32;
    DebugExchange: "gomad_debug_v1", "exchange", "debug_exchange", (p0: i32, p1: i32) -> i32;
}
impl Operation {
    /// Imports the helper always answers itself whenever it owns the model.
    /// `poll_oneoff` is local only for clock-only subscriptions, or for any
    /// subscriptions when the helper also owns the namespace.
    pub fn is_model_local(self) -> bool {
        matches!(
            self,
            Self::ClockTimeGet
                | Self::SchedYield
                | Self::RandomGet
                | Self::ArgsGet
                | Self::ArgsSizesGet
                | Self::EnvironGet
                | Self::EnvironSizesGet
        ) || self.is_runtime() && self != Self::RuntimeNetStream
    }
    /// Imports answered by the helper-owned namespace, including refused stock sockets.
    pub fn is_namespace(self) -> bool {
        matches!(
            self,
            Self::FdClose
                | Self::FdSync
                | Self::FdFilestatGet
                | Self::FdFdstatGet
                | Self::FdPrestatGet
                | Self::FdFdstatSetFlags
                | Self::FdFilestatSetSize
                | Self::FdWrite
                | Self::FdRead
                | Self::PathCreateDirectory
                | Self::PathRemoveDirectory
                | Self::PathUnlinkFile
                | Self::FdPrestatDirName
                | Self::FdPread
                | Self::FdPwrite
                | Self::FdReaddir
                | Self::FdSeek
                | Self::PathFilestatGet
                | Self::PathSymlink
                | Self::PathReadlink
                | Self::PathRename
                | Self::PathOpen
                | Self::SockAccept
                | Self::SockShutdown
        )
    }
    pub fn is_runtime(self) -> bool {
        matches!(
            self,
            Self::RuntimeConfig
                | Self::RuntimeDecision
                | Self::RuntimeObservation
                | Self::RuntimeFinish
                | Self::RuntimeIdle
                | Self::RuntimeNetStream
        )
    }
}
struct Guest<'a, 'b> {
    caller: &'a mut Caller<'b, State>,
    memory: Memory,
}
#[derive(Debug)]
struct InputCapacity(&'static str);
impl std::fmt::Display for InputCapacity {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.0)
    }
}
impl std::error::Error for InputCapacity {}
fn memory(caller: &mut Caller<'_, State>) -> Result<Memory> {
    if let Some(memory) = caller.data().memory {
        return Ok(memory);
    }
    let memory = caller
        .get_export("memory")
        .and_then(wasmtime::Extern::into_memory)
        .ok_or_else(|| Error::msg("missing guest memory"))?;
    caller.data_mut().memory = Some(memory);
    Ok(memory)
}
impl Guest<'_, '_> {
    /// Decodes `poll_oneoff` subscriptions after validating every guest range.
    fn subscriptions(
        &self,
        input: u32,
        output: u32,
        count: u32,
        events: u32,
    ) -> Result<Vec<crate::transcript::Subscription>> {
        let count = count as usize;
        if count == 0 || count > 4096 {
            return Err(Error::new(InputCapacity("subscription capacity")));
        }
        let bytes = self.bytes(input, count * 48)?;
        self.range(output, count * 32)?;
        self.range(events, 4)?;
        let mut subscriptions = Vec::with_capacity(count);
        for raw in bytes.as_chunks::<48>().0 {
            let kind = raw[8];
            if kind > 2 {
                return Err(Error::msg("invalid subscription type"));
            }
            subscriptions.push(crate::transcript::Subscription {
                userdata: u64::from_le_bytes(raw[..8].try_into().unwrap()),
                kind,
                clock_id: u32::from_le_bytes(raw[16..20].try_into().unwrap()),
                timeout: u64::from_le_bytes(raw[24..32].try_into().unwrap()),
                precision: u64::from_le_bytes(raw[32..40].try_into().unwrap()),
                flags: u16::from_le_bytes(raw[40..42].try_into().unwrap()),
                fd: u32::from_le_bytes(raw[16..20].try_into().unwrap()),
            });
        }
        Ok(subscriptions)
    }
    /// Validates the guest ranges of the derived-input imports. `args_get` and
    /// `environ_get` require their preceding size call.
    fn validate_derived(&self, op: Operation, first: u32, second: u32) -> Result<()> {
        match op {
            Operation::ArgsSizesGet | Operation::EnvironSizesGet => {
                self.range(first, 4)?;
                self.range(second, 4)?;
            }
            Operation::ArgsGet | Operation::EnvironGet => {
                let sizes = if op == Operation::ArgsGet {
                    self.caller.data().argument_sizes
                } else {
                    self.caller.data().environment_sizes
                };
                let (count, length) =
                    sizes.ok_or_else(|| Error::msg("string sizes must precede get"))?;
                self.range(first, count as usize * 4)?;
                self.range(second, length as usize)?;
            }
            Operation::RandomGet => {
                self.range(first, second as usize)?;
            }
            _ => {}
        }
        Ok(())
    }
    fn range(&self, pointer: u32, length: usize) -> Result<std::ops::Range<usize>> {
        if length > BUFFER_LIMIT {
            return Err(Error::new(InputCapacity("ABI buffer capacity")));
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
            return Err(Error::new(InputCapacity("iovec capacity")));
        }
        let descriptors = self.bytes(pointer, count as usize * 8)?;
        let mut total = 0usize;
        let mut result = Vec::new();
        for descriptor in descriptors.as_chunks::<8>().0 {
            let pointer = u32::from_le_bytes(descriptor[..4].try_into().unwrap());
            let length = u32::from_le_bytes(descriptor[4..].try_into().unwrap()) as usize;
            self.range(pointer, length)?;
            total = total
                .checked_add(length)
                .ok_or_else(|| Error::msg("iovec length overflow"))?;
            if total > BUFFER_LIMIT {
                return Err(Error::new(InputCapacity("iovec buffer capacity")));
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
                if width == 1 {
                    u64::from(u8::MAX)
                } else {
                    u64::MAX
                },
            )?,
            width,
        );
    }
    Ok(bytes)
}
#[expect(
    clippy::too_many_lines,
    reason = "Keep ABI admission, dispatch, and reply encoding together so their order remains explicit."
)]
#[expect(
    clippy::cast_possible_truncation,
    reason = "Guest buffers and iovec totals are capped by BUFFER_LIMIT; reply integers and subscription counts are checked before encoding."
)]
pub fn invoke(
    mut caller: Caller<'_, State>,
    op: Operation,
    params: &[Val],
    results: &mut [Val],
) -> Result<()> {
    // The op frame opens before any route so every import is attributed;
    // the debug exchange and the sealed refusal stay out of the v1 buckets.
    #[cfg(feature = "diagnostic-profile")]
    let _abi = crate::diagnostic::AbiSpan::new(op, caller.data().model.is_some());
    if op == Operation::DebugExchange {
        #[cfg(feature = "diagnostic-profile")]
        serve_legacy_excluded(Owner::Inspection);
        return invoke_debug(caller, params, results);
    }
    if caller
        .data()
        .model
        .as_ref()
        .is_some_and(|model| model.inspection.sealed)
    {
        #[cfg(feature = "diagnostic-profile")]
        serve_legacy_excluded(Owner::Inspection);
        return Err(caller.data_mut().fail(
            "debug_error",
            format!("modeled import after inspection seal: {op:?}"),
        ));
    }
    if op.is_runtime() {
        return invoke_runtime(caller, op, params, results);
    }
    if op == Operation::ProcExit {
        #[cfg(feature = "diagnostic-profile")]
        serve(Owner::Lifecycle);
        let mut outcome = Outcome::new("exit", "");
        outcome.exit_code = Some(params[0].i32().unwrap().cast_unsigned());
        caller.data_mut().terminal = Some(outcome);
        return Err(Error::msg("guest proc_exit"));
    }
    let memory = memory(&mut caller)?;
    let mut guest = Guest {
        caller: &mut caller,
        memory,
    };
    let get = |n: usize| params[n].i32().unwrap().cast_unsigned();
    let wide = |n: usize| params[n].i64().unwrap().cast_unsigned();
    let split = guest
        .caller
        .data()
        .model
        .as_ref()
        .is_some_and(crate::model::Model::split);
    // Split ownership keeps guest fd 1/2 writes remote, where broker output
    // boundaries observe them in order.
    let remote_output = split
        && matches!(op, Operation::FdWrite | Operation::FdPwrite)
        && guest
            .caller
            .data()
            .namespace
            .as_ref()
            .is_some_and(|namespace| namespace.standard_output(get(0)));
    if guest.caller.data().namespace.is_some() && op.is_namespace() && !remote_output {
        #[cfg(feature = "diagnostic-profile")]
        serve(Owner::Namespace);
        return invoke_namespace(&mut guest, op, params, results);
    }
    if op == Operation::ClockTimeGet && guest.caller.data().model.is_some() {
        #[cfg(feature = "diagnostic-profile")]
        serve(Owner::Clock);
        if let Err(error) = guest.range(get(2), 8) {
            return Err(guest.caller.data_mut().fail("trap", error.to_string()));
        }
        let timestamp = guest.caller.data_mut().clock_time_get(get(0), wide(1))?;
        guest.put64(get(2), timestamp)?;
        results[0] = Val::I32(0);
        return Ok(());
    }
    if guest.caller.data().model.is_some() {
        if op == Operation::SchedYield {
            #[cfg(feature = "diagnostic-profile")]
            serve(Owner::Scheduler);
            guest.caller.data_mut().yield_call()?;
            results[0] = Val::I32(0);
            return Ok(());
        }
        if op == Operation::PollOneoff {
            // A poll is the clock's until decoding shows a descriptor
            // subscription; one the helper cannot serve goes remote below.
            #[cfg(feature = "diagnostic-profile")]
            serve(Owner::Clock);
            let decoded = guest.subscriptions(get(0), get(1), get(2), get(3));
            let subscriptions = match decoded {
                Ok(subscriptions) => subscriptions,
                Err(error) => {
                    let message = error.to_string();
                    return Err(guest.caller.data_mut().fail(
                        if error.downcast_ref::<InputCapacity>().is_some() {
                            "capacity"
                        } else {
                            "trap"
                        },
                        message,
                    ));
                }
            };
            if subscriptions.iter().all(|sub| sub.kind == 0)
                || guest.caller.data().namespace.is_some()
            {
                #[cfg(feature = "diagnostic-profile")]
                if subscriptions.iter().any(|sub| sub.kind != 0) {
                    serve(Owner::Namespace);
                }
                let reply = guest.caller.data_mut().poll_call(&subscriptions)?;
                results[0] = Val::I32(i32::from(reply.errno));
                if reply.errno == 0 {
                    let mut bytes = vec![0; reply.events.len() * 32];
                    for (index, event) in reply.events.iter().enumerate() {
                        let raw = &mut bytes[index * 32..(index + 1) * 32];
                        put(raw, 0, event.userdata, 8);
                        put(raw, 8, u64::from(event.errno), 2);
                        put(raw, 10, u64::from(event.kind), 1);
                        put(raw, 16, event.nbytes, 8);
                        put(raw, 24, u64::from(event.flags), 2);
                    }
                    guest.write(get(1), &bytes)?;
                    guest.put32(get(3), reply.events.len() as u32)?;
                }
                return Ok(());
            }
        }
    }
    if guest.caller.data().model.is_some() && op.is_model_local() {
        #[cfg(feature = "diagnostic-profile")]
        serve(Owner::Inputs);
        let validated = guest.validate_derived(op, get(0), get(1));
        if let Err(error) = validated {
            let message = error.to_string();
            return Err(guest.caller.data_mut().fail(
                if message.contains("capacity") {
                    "capacity"
                } else {
                    "trap"
                },
                message,
            ));
        }
        match op {
            Operation::ArgsSizesGet | Operation::EnvironSizesGet => {
                let (bytes, offsets) = guest.caller.data_mut().strings(op)?;
                if offsets.len() > BUFFER_LIMIT / 4 {
                    return Err(guest
                        .caller
                        .data_mut()
                        .fail("invalid", "reply count out of range"));
                }
                let sizes = (offsets.len() as u32, bytes.len() as u32);
                guest.put32(get(0), sizes.0)?;
                guest.put32(get(1), sizes.1)?;
                if op == Operation::ArgsSizesGet {
                    guest.caller.data_mut().argument_sizes = Some(sizes);
                } else {
                    guest.caller.data_mut().environment_sizes = Some(sizes);
                }
                results[0] = Val::I32(0);
                return Ok(());
            }
            Operation::ArgsGet | Operation::EnvironGet => {
                let (bytes, offsets) = guest.caller.data_mut().strings(op)?;
                let mut pointers = Vec::with_capacity(offsets.len() * 4);
                for offset in offsets {
                    let pointer = get(1)
                        .checked_add(offset)
                        .ok_or_else(|| Error::msg("string pointer overflow"))?;
                    pointers.extend(pointer.to_le_bytes());
                }
                guest.write(get(0), &pointers)?;
                guest.write(get(1), &bytes)?;
                results[0] = Val::I32(0);
                return Ok(());
            }
            Operation::RandomGet => {
                let bytes = guest.caller.data_mut().random(get(1) as usize)?;
                guest.write(get(0), &bytes)?;
                results[0] = Val::I32(0);
                return Ok(());
            }
            _ => {}
        }
    }
    let mut iovecs = Vec::new();
    let mut write_buffers = None;
    #[cfg(feature = "diagnostic-profile")]
    serve(Owner::Remote);
    #[cfg(feature = "diagnostic-profile")]
    let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
    let input_result = (|| -> Result<Value> {
        let fd = || json!({"fd":get(0)});
        let path = || -> Result<Value> {
            Ok(json!({"fd":get(0),"path_base64":guest.base64(get(1),get(2))?}))
        };
        Ok(match op {
            Operation::ArgsSizesGet
            | Operation::EnvironSizesGet
            | Operation::ArgsGet
            | Operation::EnvironGet => {
                guest.validate_derived(op, get(0), get(1))?;
                json!({})
            }
            Operation::RandomGet => {
                guest.validate_derived(op, get(0), get(1))?;
                json!({"length":get(1)})
            }
            Operation::ClockTimeGet => {
                guest.range(get(2), 8)?;
                json!({"clock_id":get(0),"precision":wide(1)})
            }
            Operation::SchedYield => json!({}),
            Operation::FdWrite | Operation::FdPwrite | Operation::FdRead | Operation::FdPread => {
                iovecs = guest.iovecs(get(1), get(2))?;
                guest.range(
                    get(if matches!(op, Operation::FdPread | Operation::FdPwrite) {
                        4
                    } else {
                        3
                    }),
                    4,
                )?;
                let mut input = Value::Null;
                if op == Operation::FdWrite || op == Operation::FdPwrite {
                    let length = iovecs.iter().map(|&(_, length)| length).sum::<usize>();
                    let buffers = write_buffers
                        .insert(std::mem::take(&mut guest.caller.data_mut().write_scratch));
                    buffers.begin(length)?;
                    for &(pointer, length) in &iovecs {
                        let range = guest.range(pointer, length)?;
                        buffers
                            .bytes
                            .extend_from_slice(&guest.memory.data(&*guest.caller)[range]);
                    }
                    buffers.encode()?;
                    if (get(0) == 1 || get(0) == 2)
                        && (op == Operation::FdWrite || op == Operation::FdPwrite)
                    {
                        let state = guest.caller.data_mut();
                        let total = state
                            .output_bytes
                            .checked_add(buffers.bytes.len() as u64)
                            .ok_or_else(|| Error::msg("output capacity"))?;
                        if total > state.output_limit {
                            return Err(Error::msg("output capacity"));
                        }
                        state.output_bytes = total;
                    }
                } else {
                    input = fd();
                    input["length"] =
                        json!(iovecs.iter().map(|&(_, length)| length).sum::<usize>());
                }
                if op == Operation::FdPread {
                    input["offset"] = json!(wide(3));
                }
                input
            }
            Operation::FdClose | Operation::FdSync => fd(),
            Operation::FdFilestatSetSize => json!({"fd":get(0),"size":wide(1)}),
            Operation::FdFdstatSetFlags => {
                if get(1) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid fd flags"));
                }
                json!({"fd":get(0),"flags":get(1)})
            }
            Operation::FdSeek => {
                guest.range(get(3), 8)?;
                if get(2) > 2 {
                    return Err(Error::msg("invalid whence"));
                }
                json!({"fd":get(0),"offset":params[1].i64().unwrap(),"whence":get(2)})
            }
            Operation::FdFilestatGet => {
                guest.range(get(1), 64)?;
                fd()
            }
            Operation::FdFdstatGet => {
                guest.range(get(1), 24)?;
                fd()
            }
            Operation::FdPrestatGet => {
                guest.range(get(1), 8)?;
                fd()
            }
            Operation::FdPrestatDirName => {
                guest.range(get(1), get(2) as usize)?;
                json!({"fd":get(0),"length":get(2)})
            }
            Operation::FdReaddir => {
                guest.range(get(1), get(2) as usize)?;
                guest.range(get(4), 4)?;
                json!({"fd":get(0),"length":get(2),"cookie":wide(3)})
            }
            Operation::PathCreateDirectory
            | Operation::PathRemoveDirectory
            | Operation::PathUnlinkFile => path()?,
            Operation::PathFilestatGet => {
                guest.range(get(4), 64)?;
                json!({"fd":get(0),"flags":get(1),"path_base64":guest.base64(get(2),get(3))?})
            }
            Operation::PathOpen => {
                guest.range(get(8), 4)?;
                if get(4) > u32::from(u16::MAX) || get(7) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid path flags"));
                }
                json!({"fd":get(0),"dirflags":get(1),"path_base64":guest.base64(get(2),get(3))?,"oflags":get(4),"rights_base":wide(5),"rights_inheriting":wide(6),"fdflags":get(7)})
            }
            Operation::PathReadlink => {
                guest.range(get(3), get(4) as usize)?;
                guest.range(get(5), 4)?;
                let mut input = path()?;
                input["length"] = json!(get(4));
                input
            }
            Operation::PathRename => {
                json!({"fd":get(0),"path_base64":guest.base64(get(1),get(2))?,"new_fd":get(3),"new_path_base64":guest.base64(get(4),get(5))?})
            }
            Operation::PathSymlink => {
                json!({"old_path_base64":guest.base64(get(0),get(1))?,"fd":get(2),"path_base64":guest.base64(get(3),get(4))?})
            }
            Operation::SockAccept => {
                guest.range(get(2), 4)?;
                if get(1) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid socket flags"));
                }
                json!({"fd":get(0),"flags":get(1)})
            }
            Operation::SockShutdown => {
                if get(1) > 3 {
                    return Err(Error::msg("invalid shutdown mode"));
                }
                json!({"fd":get(0),"how":get(1)})
            }
            Operation::PollOneoff => {
                let subscriptions = guest
                    .subscriptions(get(0), get(1), get(2), get(3))?
                    .into_iter()
                    .map(|sub| {
                        if sub.kind == 0 {
                            json!({"userdata":sub.userdata,"type":sub.kind,"clock_id":sub.clock_id,"timeout":sub.timeout,"precision":sub.precision,"flags":sub.flags})
                        } else {
                            json!({"userdata":sub.userdata,"type":sub.kind,"fd":sub.fd})
                        }
                    })
                    .collect::<Vec<_>>();
                json!({"subscriptions":subscriptions})
            }
            _ => return Err(Error::msg("unsupported import")),
        })
    })();
    #[cfg(feature = "diagnostic-profile")]
    drop(encoding);
    let input = match input_result {
        Ok(value) => value,
        Err(error) => {
            if let Some(mut buffers) = write_buffers.take() {
                buffers.clear();
                guest.caller.data_mut().write_scratch = buffers;
            }
            let message = error.to_string();
            let kind = if message.contains("capacity") {
                "capacity"
            } else {
                "trap"
            };
            return Err(guest.caller.data_mut().fail(kind, message));
        }
    };
    let reply = if let Some(buffers) = &write_buffers {
        let data_base64 = &buffers.encoded;
        let callback = if op == Operation::FdPwrite {
            CallbackInput::PWrite {
                fd: get(0),
                data_base64,
                offset: wide(3),
            }
        } else {
            CallbackInput::Write {
                fd: get(0),
                data_base64,
            }
        };
        guest.caller.data_mut().callback_input(callback)
    } else {
        guest.caller.data_mut().callback_typed(op, input)
    };
    if let Some(mut buffers) = write_buffers {
        buffers.clear();
        guest.caller.data_mut().write_scratch = buffers;
    }
    if split {
        let state = guest.caller.data_mut();
        state
            .namespace
            .as_mut()
            .unwrap()
            .adopt_offered_output(state.output_bytes);
    }
    let reply = reply?;
    if reply.errno != 0 {
        results[0] = Val::I32(i32::from(reply.errno));
        return Ok(());
    }
    let output = &reply.output;
    #[cfg(feature = "diagnostic-profile")]
    let _encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
    let written = (|| -> Result<()> {
        match op {
            Operation::ArgsSizesGet | Operation::EnvironSizesGet => {
                fields(output, &["count", "bytes"])?;
                let count = number(output, "count", (BUFFER_LIMIT / 4) as u64)? as u32;
                let bytes = number(output, "bytes", BUFFER_LIMIT as u64)? as u32;
                guest.put32(get(0), count)?;
                guest.put32(get(1), bytes)?;
                if op == Operation::ArgsSizesGet {
                    guest.caller.data_mut().argument_sizes = Some((count, bytes));
                } else {
                    guest.caller.data_mut().environment_sizes = Some((count, bytes));
                }
            }
            Operation::ArgsGet | Operation::EnvironGet => {
                fields(output, &["values"])?;
                let values = output["values"]
                    .as_array()
                    .ok_or_else(|| Error::msg("invalid string list"))?;
                let (count, expected) = if op == Operation::ArgsGet {
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
            Operation::RandomGet => guest.write(get(0), &data(output, get(1) as usize, true)?)?,
            Operation::ClockTimeGet => {
                fields(output, &["timestamp"])?;
                guest.put64(get(2), number(output, "timestamp", u64::MAX)?)?;
            }
            Operation::FdWrite | Operation::FdPwrite => {
                fields(output, &["written"])?;
                let max = iovecs.iter().map(|&(_, length)| length as u64).sum();
                let count = number(output, "written", max)?;
                let count = count as u32;
                guest.put32(get(if op == Operation::FdPwrite { 4 } else { 3 }), count)?;
            }
            Operation::FdRead | Operation::FdPread => {
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
                    get(if op == Operation::FdPread { 4 } else { 3 }),
                    bytes.len() as u32,
                )?;
            }
            Operation::FdSeek => {
                fields(output, &["offset"])?;
                guest.put64(get(3), number(output, "offset", u64::MAX)?)?;
            }
            Operation::FdFilestatGet => guest.write(get(1), &stat(output)?)?,
            Operation::PathFilestatGet => guest.write(get(4), &stat(output)?)?,
            Operation::FdFdstatGet => {
                fields(
                    output,
                    &["filetype", "flags", "rights_base", "rights_inheriting"],
                )?;
                let mut bytes = vec![0; 24];
                for (name, offset, width, max) in [
                    ("filetype", 0, 1, u64::from(u8::MAX)),
                    ("flags", 2, 2, u64::from(u16::MAX)),
                    ("rights_base", 8, 8, u64::MAX),
                    ("rights_inheriting", 16, 8, u64::MAX),
                ] {
                    put(&mut bytes, offset, number(output, name, max)?, width);
                }
                guest.write(get(1), &bytes)?;
            }
            Operation::FdPrestatGet => {
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
            Operation::FdPrestatDirName => {
                guest.write(get(1), &data(output, get(2) as usize, true)?)?;
            }
            Operation::FdReaddir => {
                let bytes = data(output, get(2) as usize, false)?;
                guest.write(get(1), &bytes)?;
                guest.put32(get(4), bytes.len() as u32)?;
            }
            Operation::PathReadlink => {
                let bytes = data(output, get(4) as usize, false)?;
                guest.write(get(3), &bytes)?;
                guest.put32(get(5), bytes.len() as u32)?;
            }
            Operation::PathOpen | Operation::SockAccept => {
                fields(output, &["fd"])?;
                guest.put32(
                    get(if op == Operation::PathOpen { 8 } else { 2 }),
                    number(output, "fd", u64::from(u32::MAX))? as u32,
                )?;
            }
            Operation::PollOneoff => {
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
                        ("errno", 8, 2, u64::from(u16::MAX)),
                        ("type", 10, 1, 2),
                        ("nbytes", 16, 8, u64::MAX),
                        ("flags", 24, 2, u64::from(u16::MAX)),
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

#[expect(
    clippy::too_many_lines,
    reason = "Keep namespace import decoding and matching reply encoding in one ABI dispatch."
)]
#[expect(
    clippy::cast_possible_truncation,
    reason = "Iovec totals and reply bytes are buffer-bounded; flags and whence are range-checked before narrowing."
)]
fn invoke_namespace(
    guest: &mut Guest<'_, '_>,
    op: Operation,
    params: &[Val],
    results: &mut [Val],
) -> Result<()> {
    use crate::namespace::{Operation as NsOperation, Output};
    let get = |n: usize| params[n].i32().unwrap().cast_unsigned();
    let wide = |n: usize| params[n].i64().unwrap().cast_unsigned();
    let mut iovecs = Vec::new();
    let decoded = (|| -> Result<NsOperation> {
        Ok(match op {
            Operation::FdWrite | Operation::FdPwrite | Operation::FdRead | Operation::FdPread => {
                iovecs = guest.iovecs(get(1), get(2))?;
                guest.range(
                    get(if matches!(op, Operation::FdPread | Operation::FdPwrite) {
                        4
                    } else {
                        3
                    }),
                    4,
                )?;
                let offset = if matches!(op, Operation::FdPread | Operation::FdPwrite) {
                    Some(wide(3))
                } else {
                    None
                };
                if matches!(op, Operation::FdWrite | Operation::FdPwrite) {
                    let mut data = Vec::with_capacity(iovecs.iter().map(|(_, n)| *n).sum());
                    for &(pointer, length) in &iovecs {
                        let range = guest.range(pointer, length)?;
                        data.extend_from_slice(&guest.memory.data(&*guest.caller)[range]);
                    }
                    if (get(0) == 1 || get(0) == 2)
                        && guest
                            .caller
                            .data()
                            .model
                            .as_ref()
                            .is_some_and(crate::model::Model::split)
                    {
                        let state = guest.caller.data();
                        if data.len() as u64 > state.output_limit.saturating_sub(state.output_bytes)
                        {
                            return Err(Error::new(InputCapacity("output capacity")));
                        }
                    }
                    NsOperation::Write {
                        fd: get(0),
                        data,
                        offset,
                    }
                } else {
                    NsOperation::Read {
                        fd: get(0),
                        length: iovecs.iter().map(|(_, n)| *n).sum::<usize>() as u32,
                        offset,
                    }
                }
            }
            Operation::FdClose => NsOperation::Close(get(0)),
            Operation::FdSync => NsOperation::Sync(get(0)),
            Operation::FdFilestatSetSize => NsOperation::SetSize {
                fd: get(0),
                size: wide(1),
            },
            Operation::FdFdstatSetFlags => {
                if get(1) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid fd flags"));
                }
                NsOperation::SetFlags {
                    fd: get(0),
                    flags: get(1) as u16,
                }
            }
            Operation::FdSeek => {
                guest.range(get(3), 8)?;
                if get(2) > 2 {
                    return Err(Error::msg("invalid whence"));
                }
                NsOperation::Seek {
                    fd: get(0),
                    offset: params[1].i64().unwrap(),
                    whence: get(2) as u8,
                }
            }
            Operation::FdFilestatGet => {
                guest.range(get(1), 64)?;
                NsOperation::FilestatGet(get(0))
            }
            Operation::FdFdstatGet => {
                guest.range(get(1), 24)?;
                NsOperation::FdstatGet(get(0))
            }
            Operation::FdPrestatGet => {
                guest.range(get(1), 8)?;
                NsOperation::PrestatGet(get(0))
            }
            Operation::FdPrestatDirName => {
                guest.range(get(1), get(2) as usize)?;
                NsOperation::PrestatDirName {
                    fd: get(0),
                    length: get(2),
                }
            }
            Operation::FdReaddir => {
                guest.range(get(1), get(2) as usize)?;
                guest.range(get(4), 4)?;
                NsOperation::Readdir {
                    fd: get(0),
                    length: get(2),
                    cookie: wide(3),
                }
            }
            Operation::PathCreateDirectory
            | Operation::PathRemoveDirectory
            | Operation::PathUnlinkFile => {
                let fd = get(0);
                let path = guest.bytes(get(1), get(2) as usize)?;
                match op {
                    Operation::PathCreateDirectory => NsOperation::CreateDirectory { fd, path },
                    Operation::PathRemoveDirectory => NsOperation::RemoveDirectory { fd, path },
                    _ => NsOperation::UnlinkFile { fd, path },
                }
            }
            Operation::PathFilestatGet => {
                guest.range(get(4), 64)?;
                NsOperation::PathStat {
                    fd: get(0),
                    flags: get(1),
                    path: guest.bytes(get(2), get(3) as usize)?,
                }
            }
            Operation::PathOpen => {
                guest.range(get(8), 4)?;
                if get(4) > u32::from(u16::MAX) || get(7) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid path flags"));
                }
                NsOperation::Open(crate::namespace::Open {
                    fd: get(0),
                    dirflags: get(1),
                    path: guest.bytes(get(2), get(3) as usize)?,
                    oflags: get(4) as u16,
                    base: wide(5),
                    inheriting: wide(6),
                    flags: get(7) as u16,
                })
            }
            Operation::PathReadlink => {
                guest.range(get(3), get(4) as usize)?;
                guest.range(get(5), 4)?;
                NsOperation::Readlink {
                    fd: get(0),
                    path: guest.bytes(get(1), get(2) as usize)?,
                    length: get(4),
                }
            }
            Operation::PathRename => NsOperation::Rename {
                fd: get(0),
                path: guest.bytes(get(1), get(2) as usize)?,
                new_fd: get(3),
                new_path: guest.bytes(get(4), get(5) as usize)?,
            },
            Operation::PathSymlink => NsOperation::Symlink {
                old_path: guest.bytes(get(0), get(1) as usize)?,
                fd: get(2),
                path: guest.bytes(get(3), get(4) as usize)?,
            },
            Operation::SockAccept => {
                guest.range(get(2), 4)?;
                if get(1) > u32::from(u16::MAX) {
                    return Err(Error::msg("invalid socket flags"));
                }
                NsOperation::SockAccept {
                    fd: get(0),
                    flags: get(1) as u16,
                }
            }
            Operation::SockShutdown => {
                if get(1) > 3 {
                    return Err(Error::msg("invalid shutdown mode"));
                }
                NsOperation::SockShutdown {
                    fd: get(0),
                    how: get(1) as u8,
                }
            }
            _ => return Err(Error::msg("unsupported namespace import")),
        })
    })();
    let operation = match decoded {
        Ok(operation) => operation,
        Err(error) => {
            let kind = if error.downcast_ref::<InputCapacity>().is_some() {
                "capacity"
            } else {
                "trap"
            };
            return Err(guest.caller.data_mut().fail(kind, error.to_string()));
        }
    };
    let reply = guest
        .caller
        .data_mut()
        .namespace_call(crate::namespace::call::Call::new(operation))?;
    results[0] = Val::I32(i32::from(reply.errno));
    if reply.errno != 0 {
        return Ok(());
    }
    match reply.output.unwrap() {
        Output::Empty => (),
        Output::Written(count) => {
            guest.put32(get(if op == Operation::FdPwrite { 4 } else { 3 }), count)?;
        }
        Output::Offset(offset) => guest.put64(get(3), offset)?,
        Output::Fd(fd) => guest.put32(get(8), fd)?,
        Output::Prestat(length) => {
            let mut bytes = [0; 8];
            put(&mut bytes, 4, u64::from(length), 4);
            guest.write(get(1), &bytes)?;
        }
        Output::Fdstat(stat) => {
            let mut bytes = [0; 24];
            put(&mut bytes, 0, u64::from(stat.filetype), 1);
            put(&mut bytes, 2, u64::from(stat.flags), 2);
            put(&mut bytes, 8, stat.base, 8);
            put(&mut bytes, 16, stat.inheriting, 8);
            guest.write(get(1), &bytes)?;
        }
        Output::Filestat(stat) => {
            let mut bytes = [0; 64];
            for (offset, width, n) in [
                (0, 8, stat.dev),
                (8, 8, stat.ino),
                (16, 1, u64::from(stat.filetype)),
                (24, 8, stat.nlink),
                (32, 8, stat.size),
                (40, 8, stat.atim),
                (48, 8, stat.mtim),
                (56, 8, stat.ctim),
            ] {
                put(&mut bytes, offset, n, width);
            }
            guest.write(
                get(if op == Operation::PathFilestatGet {
                    4
                } else {
                    1
                }),
                &bytes,
            )?;
        }
        Output::Bytes(bytes) => {
            if matches!(op, Operation::FdRead | Operation::FdPread) {
                let mut copied = 0;
                for (pointer, length) in &iovecs {
                    let count = (*length).min(bytes.len() - copied);
                    guest.write(*pointer, &bytes[copied..copied + count])?;
                    copied += count;
                }
                guest.put32(
                    get(if op == Operation::FdPread { 4 } else { 3 }),
                    bytes.len() as u32,
                )?;
            } else {
                guest.write(get(1), &bytes)?;
                if op == Operation::FdReaddir {
                    guest.put32(get(4), bytes.len() as u32)?;
                }
            }
        }
    }
    Ok(())
}

#[cfg(feature = "diagnostic-profile")]
use crate::diagnostic::{Owner, serve};
#[cfg(feature = "diagnostic-profile")]
fn serve_legacy_excluded(owner: Owner) {
    serve(owner);
    crate::diagnostic::exclude_legacy();
}

fn invoke_debug(mut caller: Caller<'_, State>, params: &[Val], results: &mut [Val]) -> Result<()> {
    let memory = memory(&mut caller)?;
    let mut guest = Guest {
        caller: &mut caller,
        memory,
    };
    let pointer = params[0].i32().unwrap().cast_unsigned();
    let length = params[1].i32().unwrap().cast_unsigned() as usize;
    if !(16..=crate::model::inspection::MAX_FRAME).contains(&length) {
        return Err(guest
            .caller
            .data_mut()
            .fail("debug_error", "debug ABI byte bound"));
    }
    let mut bytes = guest.bytes(pointer, length)?;
    guest.caller.data_mut().debug_exchange(&mut bytes)?;
    guest.write(pointer, &bytes)?;
    results[0] = Val::I32(0);
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "Runtime ABI admission precedes dispatch and reply validation in one ordered boundary."
)]
fn invoke_runtime(
    mut caller: Caller<'_, State>,
    op: Operation,
    params: &[Val],
    results: &mut [Val],
) -> Result<()> {
    let memory = memory(&mut caller)?;
    let mut guest = Guest {
        caller: &mut caller,
        memory,
    };
    let pointer = params[0].i32().unwrap().cast_unsigned();
    let length = params[1].i32().unwrap().cast_unsigned() as usize;
    #[cfg(feature = "diagnostic-profile")]
    serve(
        if matches!(op, Operation::RuntimeIdle | Operation::RuntimeNetStream)
            && (guest.caller.data().broker.is_some() || guest.caller.data().network.is_some())
        {
            if op == Operation::RuntimeIdle {
                Owner::SchedulerHandoff
            } else {
                Owner::Network
            }
        } else if guest
            .caller
            .data()
            .model
            .as_ref()
            .is_some_and(|model| !model.split())
            && op != Operation::RuntimeNetStream
        {
            Owner::Scheduler
        } else {
            Owner::Remote
        },
    );
    #[cfg(feature = "diagnostic-profile")]
    let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
    let input = (|| -> Result<Vec<u8>> {
        let maximum = match op {
            Operation::RuntimeConfig | Operation::RuntimeFinish | Operation::RuntimeIdle => 16,
            Operation::RuntimeNetStream => 64 + 64 * 1024,
            Operation::RuntimeDecision => 32 + 256 * 48 + 96,
            Operation::RuntimeObservation => 192,
            _ => return Err(Error::msg("unsupported runtime import")),
        };
        if length > maximum
            || op == Operation::RuntimeDecision
                && length > 32 + 256 * 32 + 96
                && !(length - 32).is_multiple_of(16)
        {
            return Err(Error::msg("runtime ABI buffer capacity"));
        }
        let valid_length = match op {
            Operation::RuntimeDecision => length >= 96 && (length - 32).is_multiple_of(16),
            Operation::RuntimeObservation => length == 96 || length == 192,
            Operation::RuntimeNetStream => (64..=64 + 64 * 1024).contains(&length),
            _ => length == 16,
        };
        if !valid_length {
            return Err(Error::msg("invalid runtime ABI buffer length"));
        }
        let bytes = guest.bytes(pointer, length)?;
        if op == Operation::RuntimeDecision {
            let count = u32::from_be_bytes(bytes[20..24].try_into().unwrap()) as usize;
            let version = u32::from_be_bytes(bytes[..4].try_into().unwrap());
            let kind = u32::from_be_bytes(bytes[4..8].try_into().unwrap());
            let stride = if version == 2 && kind == 1 { 48 } else { 32 };
            if !(2..=256).contains(&count)
                || (length != 32 + count * stride && length != 32 + count * stride + 96)
            {
                return Err(Error::msg("invalid runtime decision buffer layout"));
            }
        }
        Ok(bytes)
    })();
    #[cfg(feature = "diagnostic-profile")]
    drop(encoding);
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
    if matches!(op, Operation::RuntimeIdle | Operation::RuntimeNetStream)
        && guest
            .caller
            .data()
            .broker
            .as_ref()
            .is_none_or(|member| member.network.is_none())
        && (guest.caller.data().network.is_some()
            || op == Operation::RuntimeNetStream
                && guest
                    .caller
                    .data()
                    .model
                    .as_ref()
                    .is_some_and(|model| model.linked_evidence.is_some()))
    {
        let bytes = guest.caller.data_mut().network_call(op, input)?;
        guest.write(pointer, &bytes)?;
        results[0] = Val::I32(0);
        return Ok(());
    }
    if matches!(op, Operation::RuntimeIdle | Operation::RuntimeNetStream)
        && guest.caller.data().broker.is_some()
    {
        let bytes = guest.caller.data_mut().broker_call(op, input)?;
        guest.write(pointer, &bytes)?;
        results[0] = Val::I32(0);
        return Ok(());
    }
    if guest
        .caller
        .data()
        .model
        .as_ref()
        .is_some_and(|model| !model.split())
        && op != Operation::RuntimeNetStream
    {
        #[cfg(feature = "diagnostic-profile")]
        if op == Operation::RuntimeDecision {
            crate::diagnostic::decision_kind(u32::from_be_bytes(input[4..8].try_into().unwrap()));
        }
        let bytes = guest.caller.data_mut().runtime_typed(op, input)?;
        if bytes.len() != length {
            return Err(guest
                .caller
                .data_mut()
                .fail("invalid", "runtime reply byte length mismatch"));
        }
        guest.write(pointer, &bytes)?;
        results[0] = Val::I32(0);
        return Ok(());
    }
    let reply = guest
        .caller
        .data_mut()
        .callback_typed(op, json!({"data_base64":STANDARD.encode(input)}))?;
    if reply.errno != 0 {
        results[0] = Val::I32(i32::from(reply.errno));
        return Ok(());
    }
    let written = (|| -> Result<()> {
        #[cfg(feature = "diagnostic-profile")]
        let _encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
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
#[cfg(test)]
mod tests {
    use super::Operation;

    #[test]
    fn operation_locality_tables_cover_their_imports() {
        for &op in Operation::ALL {
            let name = op.callback();
            assert_eq!(
                op.is_namespace(),
                name.starts_with("fd_") || name.starts_with("path_") || name.starts_with("sock_"),
                "{name}"
            );
            assert!(!(op.is_namespace() && op.is_model_local()), "{name}");
        }
    }

    #[cfg(feature = "diagnostic-profile")]
    #[test]
    fn operation_profile_keys_are_unique_and_index_their_table() {
        let mut keys = std::collections::BTreeSet::new();
        for (index, &op) in Operation::ALL.iter().enumerate() {
            assert_eq!(op as usize, index);
            assert!(!op.module().is_empty() && !op.name().is_empty());
            assert!(keys.insert(format!("{}.{}", op.module(), op.name())));
        }
        assert_eq!(keys.len(), Operation::COUNT);
    }
}
