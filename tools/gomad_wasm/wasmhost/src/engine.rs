use crate::{
    abi,
    protocol::{self, Authorization, Execute, Outcome, Reply},
};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::HashSet;
use std::io::{BufRead, BufReader, Read, Write};
use wasmtime::{
    CodeBuilder, Config, Engine, Error, ExternType, Linker, Module, ResourceLimiter, Result, Store,
    Trap, ValType,
};

pub struct Limits {
    pub max: usize,
    pub peak: usize,
    pub failure: Option<&'static str>,
}
impl ResourceLimiter for Limits {
    fn memory_growing(&mut self, _: usize, desired: usize, maximum: Option<usize>) -> Result<bool> {
        if desired > self.max || maximum.is_some_and(|max| desired > max) {
            self.failure = Some("capacity");
            return Err(Error::msg("guest linear memory capacity"));
        }
        self.peak = self.peak.max(desired);
        Ok(true)
    }
    fn memory_grow_failed(&mut self, error: Error) -> Result<()> {
        self.failure = Some("infrastructure");
        Err(error)
    }
    fn table_growing(&mut self, _: usize, desired: usize, maximum: Option<usize>) -> Result<bool> {
        if desired > 1_000_000 || maximum.is_some_and(|max| desired > max) {
            self.failure = Some("capacity");
            return Err(Error::msg("guest table capacity"));
        }
        Ok(true)
    }
    fn table_grow_failed(&mut self, error: Error) -> Result<()> {
        self.failure = Some("infrastructure");
        Err(error)
    }
    fn instances(&self) -> usize {
        1
    }
    fn memories(&self) -> usize {
        1
    }
    fn tables(&self) -> usize {
        8
    }
}
pub struct State {
    pub input: Box<dyn BufRead + Send>,
    pub output: Box<dyn Write + Send>,
    pub limits: Limits,
    pub calls: u64,
    pub output_limit: u64,
    pub output_bytes: u64,
    pub terminal: Option<Outcome>,
    pub argument_sizes: Option<(u32, u32)>,
    pub environment_sizes: Option<(u32, u32)>,
}
impl State {
    pub fn fail(&mut self, kind: &'static str, message: impl Into<String>) -> Error {
        let message = message.into();
        self.terminal = Some(Outcome::new(kind, message.clone()));
        Error::msg(message)
    }
    pub fn callback(&mut self, op: &str, input: Value) -> Result<Reply> {
        if self.calls >= 1_000_000 {
            return Err(self.fail("capacity", "callback capacity"));
        }
        self.calls += 1;
        let result = (|| {
            protocol::send_frame(
                &mut self.output,
                &json!({"type":"call","id":self.calls,"op":op,"input":input}),
            )?;
            let reply: Reply = serde_json::from_slice(&protocol::read_frame(&mut self.input)?)
                .map_err(|e| Error::msg(e.to_string()))?;
            if reply.kind != "reply" || reply.id != self.calls || !reply.output.is_object() {
                return Err(Error::msg("invalid callback reply type, ID or output"));
            }
            Ok(reply)
        })();
        match result {
            Ok(reply) if reply.unsupported => {
                Err(self.fail("unsupported", format!("{op}: {}", reply.message)))
            }
            Ok(reply) => Ok(reply),
            Err(error) => Err(self.fail(protocol::classification(&error), error.to_string())),
        }
    }
}
pub fn execute() -> Result<()> {
    let mut input: Box<dyn BufRead + Send> = Box::new(BufReader::new(std::io::stdin()));
    let mut output: Box<dyn Write + Send> = Box::new(std::io::stdout());
    let parsed: Result<Execute> = (|| {
        let request: Execute = serde_json::from_slice(&protocol::read_frame(&mut input)?)
            .map_err(|e| Error::msg(e.to_string()))?;
        request.validate()?;
        Ok(request)
    })();
    let request = match parsed {
        Ok(request) => request,
        Err(error) => {
            return protocol::send_frame(
                &mut output,
                &Outcome::new(protocol::classification(&error), error.to_string()),
            );
        }
    };
    let prepared = prepare(&request);
    let (engine, module, started) = match prepared {
        Ok(value) => value,
        Err(outcome) => return protocol::send_frame(&mut output, &outcome),
    };
    if let Err(error) = protocol::send_frame(&mut output, &started) {
        if protocol::classification(&error) == "capacity" {
            return protocol::send_frame(&mut output, &Outcome::new("capacity", error.to_string()));
        }
        return Err(error);
    }
    let authorized = (|| -> Result<()> {
        let authorization: Authorization =
            serde_json::from_slice(&protocol::read_frame(&mut input)?)
                .map_err(|e| Error::msg(e.to_string()))?;
        if authorization.kind != "authorize"
            || authorization.module_sha256 != request.module_sha256
            || authorization.configuration != "cranelift-fuel-nan-canonical-v1"
        {
            return Err(Error::msg("execution authorization identity mismatch"));
        }
        Ok(())
    })();
    if let Err(error) = authorized {
        return protocol::send_frame(
            &mut output,
            &Outcome::new(protocol::classification(&error), error.to_string()),
        );
    }
    let mut store = Store::new(
        &engine,
        State {
            input,
            output,
            limits: Limits {
                max: request.memory_bytes as usize,
                peak: 0,
                failure: None,
            },
            calls: 0,
            output_limit: request.output_bytes,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
        },
    );
    store.limiter(|state| &mut state.limits);
    store.set_fuel(request.fuel)?;
    let mut linker = Linker::new(&engine);
    let mut defined = HashSet::new();
    for import in module.imports() {
        if !defined.insert((import.module().to_owned(), import.name().to_owned())) {
            continue;
        }
        let name = import.name().to_owned();
        let import_module = import.module().to_owned();
        let ExternType::Func(ty) = import.ty() else {
            unreachable!()
        };
        linker.func_new(
            import.module(),
            import.name(),
            ty,
            move |caller, params, results| {
                abi::invoke(caller, &import_module, &name, params, results)
            },
        )?;
    }
    let executed = (|| {
        let instance = linker.instantiate(&mut store, &module)?;
        instance
            .get_typed_func::<(), ()>(&mut store, "_start")?
            .call(&mut store, ())
    })();
    let mut outcome = store
        .data_mut()
        .terminal
        .take()
        .unwrap_or_else(|| match executed {
            Ok(()) => {
                let mut result = Outcome::new("exit", "");
                result.exit_code = Some(0);
                result
            }
            Err(error) => execution_failure(error, store.data().limits.failure),
        });
    outcome.fuel_remaining = store.get_fuel()?;
    outcome.peak_memory_bytes = store.data().limits.peak;
    outcome.output_bytes = store.data().output_bytes;
    protocol::send_frame(&mut store.data_mut().output, &outcome)
}
fn prepare(request: &Execute) -> std::result::Result<(Engine, Module, Value), Outcome> {
    let file = std::fs::File::open(&request.module_path)
        .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
    let mut bytes = Vec::new();
    file.take(512 * 1024 * 1024 + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
    if bytes.len() > 512 * 1024 * 1024 {
        return Err(Outcome::new("capacity", "module exceeds 512 MiB"));
    }
    if format!("{:x}", Sha256::digest(&bytes)) != request.module_sha256 {
        return Err(Outcome::new("invalid", "module SHA-256 mismatch"));
    }
    let mut config = Config::new();
    config
        .consume_fuel(true)
        .cranelift_nan_canonicalization(true)
        .wasm_relaxed_simd(false)
        .wasm_memory64(false)
        .wasm_multi_memory(false);
    let engine = Engine::new(&config)
        .map_err(|error| Outcome::new("infrastructure", format!("{error:#}")))?;
    let mut builder = CodeBuilder::new(&engine);
    builder
        .wasm_binary_or_text(&bytes, None)
        .map_err(|error| Outcome::new("invalid", format!("{error:#}")))?;
    let module = builder.compile_module().map_err(compilation_failure)?;
    let resources = module.resources_required();
    if resources.num_tables > 8
        || resources
            .max_initial_table_size
            .is_some_and(|size| size > 1_000_000)
    {
        return Err(Outcome::new(
            "capacity",
            "initial guest tables exceed limit",
        ));
    }
    let mut imports = Vec::new();
    for import in module.imports() {
        let ExternType::Func(ty) = import.ty() else {
            return Err(Outcome::new("unsupported", "non-function import"));
        };
        let params: Vec<_> = ty.params().map(type_name).collect();
        let results: Vec<_> = ty.results().map(type_name).collect();
        let Some((expected_params, expected_results)) =
            abi::signature(import.module(), import.name())
        else {
            return Err(Outcome::new(
                "unsupported",
                format!("unknown import {}::{}", import.module(), import.name()),
            ));
        };
        if params != expected_params || results != expected_results {
            return Err(Outcome::new(
                "unsupported",
                format!(
                    "invalid import signature {}::{}",
                    import.module(),
                    import.name()
                ),
            ));
        }
        imports.push(json!({"module":import.module(),"name":import.name(),"params":params,"results":results}));
    }
    let mut exports = Vec::new();
    let mut pages = None;
    let mut entry = false;
    for export in module.exports() {
        let kind = match export.ty() {
            ExternType::Memory(ty) => {
                if export.name() != "memory" || ty.is_64() || ty.is_shared() || pages.is_some() {
                    return Err(Outcome::new(
                        "unsupported",
                        "expected one unshared memory32 export",
                    ));
                }
                pages = Some(ty.minimum());
                "memory"
            }
            ExternType::Func(ty) => {
                if export.name() == "_start" {
                    entry = ty.params().len() == 0 && ty.results().len() == 0;
                }
                "func"
            }
            ExternType::Table(_) => "table",
            ExternType::Global(_) => "global",
            _ => "other",
        };
        exports.push(json!({"name":export.name(),"kind":kind}));
    }
    let Some(pages) = pages else {
        return Err(Outcome::new("unsupported", "missing exported memory"));
    };
    if !entry {
        return Err(Outcome::new("unsupported", "missing _start: () -> ()"));
    }
    if pages > request.memory_bytes / 65536 {
        return Err(Outcome::new(
            "capacity",
            "initial guest memory exceeds limit",
        ));
    }
    let started = json!({"schema":protocol::SCHEMA,"type":"started","engine":{"name":"wasmtime","version":"47.0.3","configuration":"cranelift-fuel-nan-canonical-v1","host_os":std::env::consts::OS,"host_arch":std::env::consts::ARCH},"module_sha256":request.module_sha256,"imports":imports,"exports":exports,"initial_memory_pages":pages});
    Ok((engine, module, started))
}
fn type_name(ty: ValType) -> &'static str {
    match ty {
        ValType::I32 => "i32",
        ValType::I64 => "i64",
        ValType::F32 => "f32",
        ValType::F64 => "f64",
        ValType::V128 => "v128",
        _ => "ref",
    }
}

fn execution_failure(error: Error, limiter_failure: Option<&'static str>) -> Outcome {
    let classification = limiter_failure.unwrap_or_else(|| match error.downcast_ref::<Trap>() {
        Some(Trap::OutOfFuel) => "capacity",
        Some(_) => "trap",
        None => "infrastructure",
    });
    Outcome::new(classification, format!("{error:#}"))
}

fn compilation_failure(error: Error) -> Outcome {
    let cause = error.root_cause().to_string();
    let message = error
        .downcast_ref::<wasmtime::wasmparser::BinaryReaderError>()
        .map(|error| error.message())
        .or_else(|| {
            // Function translation wraps parser errors in an unexported WasmError.
            let (offset, message) = cause
                .strip_prefix("Invalid input WebAssembly code at offset ")?
                .split_once(": ")?;
            if offset.is_empty() || !offset.bytes().all(|byte| byte.is_ascii_digit()) {
                return None;
            }
            Some(message)
        });
    let classification = match message {
        Some(
            "memory64 must be enabled for 64-bit memories"
            | "memory64 must be enabled for 64-bit tables"
            | "threads must be enabled for shared memories"
            | "threads support is not enabled"
            | "multiple memories"
            | "relaxed SIMD support is not enabled",
        ) => "unsupported",
        Some(_) => "invalid",
        None => "infrastructure",
    };
    Outcome::new(classification, format!("{error:#}"))
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    #[test]
    fn untyped_execution_io_failure_is_infrastructure() {
        let error = std::fs::File::open("/missing-gomad3-wasmhost-backing").unwrap_err();
        assert_eq!(
            execution_failure(Error::new(error), None).termination,
            "infrastructure"
        );
    }
    #[test]
    fn broken_callback_transport_is_infrastructure() {
        let (output, peer) = std::os::unix::net::UnixStream::pair().unwrap();
        drop(peer);
        let mut state = State {
            input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
            output: Box::new(output),
            limits: Limits {
                max: 65536,
                peak: 0,
                failure: None,
            },
            calls: 0,
            output_limit: 1024,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
        };
        assert!(state.callback("sched_yield", json!({})).is_err());
        assert_eq!(state.terminal.unwrap().termination, "infrastructure");
    }
}
