use crate::{
    evidence::canonical,
    install::{
        config,
        manifest::{self, Admission, Installation, Tool},
        toolchain,
    },
};
use gomad_wasmhost::linked;
use std::{
    collections::{BTreeMap, BTreeSet},
    io::Write,
    path::{Path, PathBuf},
};

fn error(status: i32, message: impl Into<String>) -> super::Error {
    super::Error {
        status,
        message: message.into(),
    }
}

fn parse(arguments: &[String]) -> Result<BTreeMap<String, String>, super::Error> {
    let mut flags = BTreeMap::<String, String>::new();
    let allowed = [
        "compiler",
        "helper",
        "runtime-root",
        "cache",
        "compile-cache",
        "output",
        "profile",
        "go-version",
        "config",
        "temporal-checkout",
        "temporal-package",
        "memory-bytes",
        "fuel",
    ]
    .into_iter()
    .collect::<BTreeSet<_>>();
    let mut index = 0;
    while index < arguments.len() {
        let argument = &arguments[index];
        if argument == "--" && index + 1 == arguments.len() {
            break;
        }
        let name = argument
            .strip_prefix("--")
            .or_else(|| argument.strip_prefix('-'))
            .filter(|name| !name.is_empty())
            .ok_or("install requires --output FILE and named tool/cache paths")?;
        let (name, value) = name
            .split_once('=')
            .map_or((name, None), |(name, value)| (name, Some(value)));
        if !allowed.contains(name) {
            return Err(format!("flag provided but not defined: -{name}").into());
        }
        let value = if let Some(value) = value {
            value
        } else {
            index += 1;
            arguments
                .get(index)
                .ok_or_else(|| format!("flag needs an argument: -{name}"))?
        };
        flags.insert(name.into(), value.into());
        index += 1;
    }
    Ok(flags)
}

fn validate_temporal(flags: &BTreeMap<String, String>, temporal: bool) -> Result<(), super::Error> {
    let value = |name: &str| flags.get(name).map_or("", String::as_str);
    if temporal {
        if let Some((key, _)) =
            std::env::vars_os().find(|(key, _)| key.to_string_lossy().starts_with("GOMAD_"))
        {
            return Err(format!(
                "unset {}; Temporal setup accepts explicit inputs only",
                key.to_string_lossy()
            )
            .into());
        }
        for name in [
            "temporal-checkout",
            "temporal-package",
            "compiler",
            "cache",
            "runtime-root",
        ] {
            if value(name).is_empty() {
                return Err(format!("temporal setup requires --{name}").into());
            }
        }
        for name in ["config", "profile"] {
            if flags.contains_key(name) {
                return Err(format!("temporal setup derives --{name}").into());
            }
        }
    }
    Ok(())
}

fn helper(supplied: &str) -> Result<(String, Option<Tool>), super::Error> {
    let current = std::env::current_exe().map_err(|err| error(2, err.to_string()))?;
    let executable_digest = canonical::digest(
        &manifest::read(&current, 512 << 20).map_err(|message| error(2, message))?,
    );
    let helper = if supplied.is_empty() {
        None
    } else {
        let path = std::path::absolute(supplied).map_err(|err| error(2, err.to_string()))?;
        manifest::executable(&path, &executable_digest).map_err(|_| {
            error(
                2,
                "pinned helper identity does not match the linked Rust implementation",
            )
        })?;
        Some(Tool {
            path,
            sha256: executable_digest.clone(),
        })
    };
    Ok((executable_digest, helper))
}

fn overlay_root(cooperative: bool) -> Result<Option<PathBuf>, super::Error> {
    if cooperative {
        std::fs::canonicalize(Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."))
            .map(Some)
            .map_err(|err| error(2, err.to_string()))
    } else {
        Ok(None)
    }
}

fn numeric(
    flags: &BTreeMap<String, String>,
    name: &str,
    default: u64,
) -> Result<u64, super::Error> {
    let Some(value) = flags.get(name) else {
        return Ok(default);
    };
    let invalid = || error(2, format!("invalid value for --{name}"));
    let (digits, radix, prefix) = match value.as_bytes() {
        [b'0', b'x' | b'X', ..] => (&value[2..], 16, true),
        [b'0', b'b' | b'B', ..] => (&value[2..], 2, true),
        [b'0', b'o' | b'O', ..] => (&value[2..], 8, true),
        [b'0', _, ..] => (value.as_str(), 8, false),
        _ => (value.as_str(), 10, false),
    };
    if digits.is_empty() {
        return Err(invalid());
    }
    let mut cleaned = String::new();
    let bytes = digits.as_bytes();
    for (index, byte) in bytes.iter().copied().enumerate() {
        if byte == b'_' {
            let preceding =
                index > 0 && (bytes[index - 1] as char).is_digit(radix) || index == 0 && prefix;
            let following = bytes
                .get(index + 1)
                .is_some_and(|byte| (*byte as char).is_digit(radix));
            if !preceding || !following {
                return Err(invalid());
            }
        } else if (byte as char).is_digit(radix) {
            cleaned.push(byte as char);
        } else {
            return Err(invalid());
        }
    }
    u64::from_str_radix(&cleaned, radix).map_err(|_| invalid())
}

/// # Errors
/// Refuses invalid installation inputs and failed exclusive publication.
#[expect(
    clippy::too_many_lines,
    reason = "Installation admission keeps validated policy and pinned tools in one ordered publication transaction"
)]
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let flags = parse(arguments)?;
    let value = |name: &str| flags.get(name).map_or("", String::as_str);
    if value("output").is_empty() {
        return Err("install requires --output FILE and named tool/cache paths".into());
    }
    let temporal = !value("temporal-checkout").is_empty() || !value("temporal-package").is_empty();
    validate_temporal(&flags, temporal)?;
    for name in ["compiler", "cache"] {
        if value(name).is_empty() {
            return Err("compiler and cache paths are required".into());
        }
    }
    let profile = if temporal {
        "gomad.wasi-cooperative/v1"
    } else {
        flags
            .get("profile")
            .map_or("gomad.wasi-stock-exploratory/v1", String::as_str)
    };
    let memory = numeric(
        &flags,
        "memory-bytes",
        if temporal { 4 << 30 } else { 256 << 20 },
    )?;
    let fuel = numeric(
        &flags,
        "fuel",
        if temporal { 1 << 50 } else { 4_000_000_000 },
    )?;
    let config = config::build(
        profile,
        (!value("config").is_empty()).then(|| Path::new(value("config"))),
        temporal.then(|| {
            (
                Path::new(value("temporal-checkout")),
                value("temporal-package"),
            )
        }),
    )
    .map_err(|message| error(2, message))?;
    let absolute =
        |name: &str| std::path::absolute(value(name)).map_err(|err| error(2, err.to_string()));
    let compiler = absolute("compiler")?;
    let compiler_digest = canonical::digest(
        &manifest::read(&compiler, 512 << 20).map_err(|message| error(2, message))?,
    );
    manifest::executable(&compiler, &compiler_digest).map_err(|message| error(2, message))?;
    let (executable_digest, helper) = helper(value("helper"))?;
    let cooperative = profile == "gomad.wasi-cooperative/v1";
    let overlay_module_root = overlay_root(cooperative)?;
    let mut installed = Installation {
        schema: manifest::SCHEMA.into(),
        admission: None,
        compiler_path: compiler,
        compiler_sha256: compiler_digest,
        go_version: flags
            .get("go-version")
            .cloned()
            .unwrap_or_else(|| "go1.27.1".into()),
        runtime_root: if value("runtime-root").is_empty() {
            PathBuf::new()
        } else {
            absolute("runtime-root")?
        },
        overlay_module_root,
        cache_root: absolute("cache")?,
        profile: profile.into(),
        fuel,
        memory_bytes: memory,
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: cooperative
            .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        model: config.model,
        namespace: config.namespace,
        storage: config.storage,
        network: config.network,
        build_tags: config.build_tags,
        excluded_build_tags: Vec::new(),
    };
    manifest::validate(&installed).map_err(|message| error(2, message))?;
    let compiler_inputs = toolchain::pin(&installed).map_err(|message| error(2, message))?;
    installed.admission = Some(Admission {
        inputs_sha256: canonical::digest(&canonical::encode(&compiler_inputs)?),
        compiler_inputs,
        configuration_sha256: canonical::digest(&canonical::encode(&config.original)?),
        guest_configuration: config.original,
        host_executable_sha256: executable_digest,
        explicit_helper: helper,
        environment: config.environment,
        arguments: config.arguments,
        compile_cache_root: if value("compile-cache").is_empty() {
            None
        } else {
            Some(absolute("compile-cache")?)
        },
    });
    manifest::write(Path::new(value("output")), &installed).map_err(|message| error(3, message))?;
    writeln!(
        output,
        "gomad: installed WASM manifest {} profile={profile}",
        value("output")
    )
    .map_err(|err| error(3, err.to_string()))?;
    Ok(0)
}
