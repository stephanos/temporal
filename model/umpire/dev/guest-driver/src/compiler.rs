//! Portable compiler identities and direct compiler execution.
use crate::{
    Error, Result,
    support::{self, SCHEMA},
};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    ffi::{OsStr, OsString},
    fs,
    os::unix::{
        ffi::{OsStrExt, OsStringExt},
        process::ExitStatusExt,
    },
    path::{Path, PathBuf},
    process::Command,
};

use crate::context::{Environment, Remaps as Roots, value as environment};
type Options = BTreeMap<&'static str, Vec<Vec<u8>>>;

/// A native compiler record; all transitive host macro variants travel with it.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Sidecar {
    /// Native identity schema version.
    pub schema: u32,
    /// Portable compiler identity.
    pub identity: String,
    /// Portable macro identity to Cargo host metadata identity.
    #[serde(default)]
    pub host_variants: BTreeMap<String, String>,
}

/// Prepared argv and optional publication, without changing process environment.
#[derive(Debug)]
pub struct Prepared {
    /// Compiler executable followed by its arguments.
    pub command: Vec<OsString>,
    /// Record published before starting the compiler.
    pub sidecar: Option<(PathBuf, Sidecar)>,
}

fn invalid(message: impl Into<String>) -> Error {
    Error::Invalid(message.into())
}

fn options(arguments: &[OsString]) -> Result<Options> {
    let mut parsed = BTreeMap::new();
    let mut index = 1;
    while index < arguments.len() {
        let argument = arguments[index].as_bytes();
        for name in [
            "--crate-name",
            "--crate-type",
            "--edition",
            "--out-dir",
            "--target",
            "--emit",
            "--cfg",
            "--extern",
            "-C",
            "-Z",
        ] {
            let prefix = name.as_bytes();
            let value = if argument == prefix {
                index += 1;
                Some(
                    arguments
                        .get(index)
                        .ok_or_else(|| invalid(format!("missing value for {name}")))?
                        .as_bytes(),
                )
            } else if let Some(value) = argument.strip_prefix(prefix) {
                if name == "-C" || name == "-Z" { Some(value) } else { value.strip_prefix(b"=") }
            } else {
                None
            };
            if let Some(value) = value {
                if value.is_empty() {
                    return Err(invalid(format!("empty value for {name}")));
                }
                parsed.entry(name).or_insert_with(Vec::new).push(value.to_vec());
                break;
            }
        }
        index += 1;
    }
    Ok(parsed)
}

fn values<'a>(parsed: &'a Options, name: &str) -> &'a [Vec<u8>] {
    parsed.get(name).map_or(&[], Vec::as_slice)
}

fn last<'a>(parsed: &'a Options, name: &str) -> Result<&'a [u8]> {
    values(parsed, name).last().map(Vec::as_slice).ok_or_else(|| invalid(format!("missing {name}")))
}

fn identity_file(artifact: &Path) -> Result<PathBuf> {
    let stem = artifact
        .file_stem()
        .ok_or_else(|| invalid("dependency artifact needs a filename"))?
        .as_bytes();
    let mut name = stem.strip_prefix(b"lib").unwrap_or(stem).to_vec();
    name.extend(b".umpire-id");
    Ok(artifact.with_file_name(OsString::from_vec(name)))
}

#[derive(Deserialize)]
struct Library {
    #[serde(default)]
    package: Vec<Package>,
}
#[derive(Deserialize)]
struct Package {
    name: String,
    version: String,
}

fn standard(
    target: bool,
    parsed: &Options,
    manifest: &[u8],
    package: &str,
    env: &Environment,
) -> Result<bool> {
    if !target || !values(parsed, "-Z").iter().any(|v| v == b"force-unstable-if-unmarked") {
        return Ok(false);
    }
    if manifest.starts_with(b"/rust/sysroot/") {
        return Ok(true);
    }
    if !manifest.starts_with(b"/cargo/registry/") {
        return Ok(false);
    }
    let lock = Path::new(environment(env, "UMPIRE_GUEST_SYSROOT")?)
        .join("lib/rustlib/src/rust/library/Cargo.lock");
    let library: Library = toml::from_str(&fs::read_to_string(lock)?)?;
    Ok(library.package.iter().any(|p| format!("{} {}", p.name, p.version) == package))
}

struct Codegen {
    portable: Vec<Vec<u8>>,
    metadata: Vec<Vec<u8>>,
    extra: Vec<u8>,
}

fn codegen(parsed: &Options, target: bool, roots: &Roots) -> Result<Codegen> {
    let mut codegen = Vec::new();
    let mut metadata = Vec::new();
    let mut extra = Vec::new();
    for value in values(parsed, "-C") {
        let key = value.split(|b| *b == b'=').next().unwrap_or_default();
        match key {
            b"metadata" => metadata.push(value.clone()),
            b"extra-filename" => {
                extra = value.strip_prefix(b"extra-filename=").unwrap_or_default().to_vec();
            }
            b"incremental" => (),
            _ if target
                || [
                    b"opt-level".as_slice(),
                    b"debug-assertions",
                    b"overflow-checks",
                    b"panic",
                    b"codegen-units",
                    b"lto",
                ]
                .contains(&key) =>
            {
                codegen.push(roots.token(value));
            }
            _ => (),
        }
    }
    if extra.contains(&b'/') {
        return Err(invalid("invalid extra-filename"));
    }
    Ok(Codegen { portable: codegen, metadata, extra })
}

struct Dependencies {
    edges: BTreeSet<(Vec<u8>, String)>,
    host_variants: BTreeMap<String, String>,
}

fn dependencies(parsed: &Options, target: bool) -> Result<Dependencies> {
    let mut dependencies = BTreeSet::new();
    let mut host_variants = BTreeMap::new();
    if target {
        for external in values(parsed, "--extern") {
            let split = external
                .iter()
                .position(|b| *b == b'=')
                .ok_or_else(|| invalid("guest dependency has no artifact path"))?;
            if split == 0 || split + 1 == external.len() {
                return Err(invalid("guest dependency has no artifact path"));
            }
            let artifact = PathBuf::from(OsString::from_vec(external[split + 1..].to_vec()));
            let record: Sidecar = serde_json::from_slice(&fs::read(identity_file(&artifact)?)?)?;
            if record.schema != SCHEMA
                || !support::valid_hash(&record.identity)
                || record
                    .host_variants
                    .iter()
                    .any(|(k, v)| !support::valid_hash(k) || !support::valid_hash(v))
            {
                return Err(invalid("invalid guest dependency identity"));
            }
            dependencies.insert((external[..split].to_vec(), record.identity));
            for (key, value) in record.host_variants {
                if host_variants.get(&key).is_some_and(|prior| prior != &value) {
                    return Err(invalid("conflicting native macro variants in guest dependency"));
                }
                host_variants.insert(key, value);
            }
        }
    }
    Ok(Dependencies { edges: dependencies, host_variants })
}

fn lock_closure(
    target: bool,
    parsed: &Options,
    manifest: &[u8],
    package: &str,
    env: &Environment,
) -> Result<Option<String>> {
    if standard(target, parsed, manifest, package, env)? {
        Ok(None)
    } else {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct LockMap {
            schema: u32,
            closures: BTreeMap<String, Option<String>>,
        }
        let map: LockMap =
            serde_json::from_slice(&fs::read(environment(env, "UMPIRE_GUEST_LOCK")?)?)?;
        if map.schema != SCHEMA {
            return Err(invalid("invalid lock map schema"));
        }
        Ok(Some(
            map.closures
                .get(package)
                .and_then(Option::as_ref)
                .filter(|v| support::valid_hash(v))
                .cloned()
                .ok_or_else(|| invalid("no unambiguous lockfile closure for guest package"))?,
        ))
    }
}

fn target_command(arguments: &[OsString], env: &Environment, identity: &str) -> Vec<OsString> {
    let mut command = vec![arguments[0].clone()];
    let mut index = 1;
    while index < arguments.len() {
        let argument = arguments[index].as_bytes();
        if argument == b"-C"
            && arguments.get(index + 1).is_some_and(|v| v.as_bytes().starts_with(b"metadata="))
        {
            index += 2;
            continue;
        }
        if argument.starts_with(b"-Cmetadata=") {
            index += 1;
            continue;
        }
        command.push(arguments[index].clone());
        index += 1;
    }
    command.extend([OsString::from("-C"), OsString::from(format!("metadata={identity}"))]);
    if let Some(out) = env.get(OsStr::new("OUT_DIR")) {
        let mut remap = b"--remap-path-prefix=".to_vec();
        remap.extend(out.as_bytes());
        remap.extend(format!("=/generated/{identity}").as_bytes());
        command.push(OsString::from_vec(remap));
    }
    command
}

/// Normalize guest inputs. Identity-bearing OS values use byte arrays in native JSON.
/// Probes and ordinary host compilations need no guest environment; host macro argv is untouched.
pub fn prepare(arguments: Vec<OsString>, env: &Environment) -> Result<Prepared> {
    if arguments.is_empty() {
        return Err(invalid("missing compiler executable"));
    }
    let parsed = options(&arguments)?;
    let types: BTreeSet<Vec<u8>> = values(&parsed, "--crate-type")
        .iter()
        .flat_map(|v| v.split(|b| *b == b',').map(<[u8]>::to_vec))
        .collect();
    let target = values(&parsed, "--target").last().is_some_and(|v| v == b"wasm32-wasip1");
    if !parsed.contains_key("--emit") || (!target && !types.contains(b"proc-macro".as_slice())) {
        return Ok(Prepared { command: arguments, sidecar: None });
    }
    let roots = Roots::new(env)?;
    let manifest = Path::new(environment(env, "CARGO_MANIFEST_DIR")?);
    let normalized_manifest = roots.path(manifest)?;
    let name = last(&parsed, "--crate-name")?;
    if name.contains(&b'/') || name == b"." || name == b".." {
        return Err(invalid("invalid crate name"));
    }
    let source = arguments
        .iter()
        .skip(1)
        .find(|a| a.as_bytes().ends_with(b".rs"))
        .ok_or_else(|| invalid("missing Rust source"))?;
    let source = roots.path(&manifest.join(source))?;
    let codegen = codegen(&parsed, target, &roots)?;
    let cfg: BTreeSet<_> = values(&parsed, "--cfg")
        .iter()
        .filter(|v| target || v.starts_with(b"feature="))
        .map(|v| roots.token(v))
        .collect();
    let Dependencies { edges: dependencies, mut host_variants } = dependencies(&parsed, target)?;
    let package_name = environment(env, "CARGO_PKG_NAME")?
        .to_str()
        .ok_or_else(|| invalid("package name must be UTF-8"))?;
    let version = environment(env, "CARGO_PKG_VERSION")?
        .to_str()
        .ok_or_else(|| invalid("package version must be UTF-8"))?;
    let package = format!("{package_name} {version}");
    let lock = lock_closure(target, &parsed, &normalized_manifest, &package, env)?;
    let key = serde_json::json!({
        "schema": SCHEMA, "compiler": roots.token(environment(env, "UMPIRE_GUEST_COMPILER")?.as_bytes()),
        "lock": lock, "package": [package_name.as_bytes(), version.as_bytes(), &normalized_manifest],
        "name": name, "source": source, "types": types,
        "target": if target { "wasm32-wasip1" } else { "proc-macro-host" },
        "edition": values(&parsed, "--edition").last().map_or(b"2015".as_slice(), Vec::as_slice),
        "cfg": cfg, "codegen": codegen.portable, "unstable": if target { values(&parsed, "-Z").iter().map(|v| roots.token(v)).collect::<Vec<_>>() } else { Vec::new() },
        "dependencies": dependencies,
    });
    let identity = support::hash(&support::canonical(&key)?);
    let mut filename = name.to_vec();
    filename.extend(&codegen.extra);
    filename.extend(b".umpire-id");
    let path = PathBuf::from(OsString::from_vec(last(&parsed, "--out-dir")?.to_vec()))
        .join(OsString::from_vec(filename));
    let command = if target {
        target_command(&arguments, env, &identity)
    } else {
        host_variants
            .insert(identity.clone(), support::hash(&support::canonical(&codegen.metadata)?));
        arguments
    };
    Ok(Prepared {
        command,
        sidecar: Some((path, Sidecar { schema: SCHEMA, identity, host_variants })),
    })
}

/// Publish before compilation for Cargo pipelining and retain the lease until child exit.
/// Child argv, environment, loader variables and inherited jobserver handles pass directly.
pub fn run(arguments: Vec<OsString>) -> Result<i32> {
    let env: Environment = std::env::vars_os().collect();
    let prepared = prepare(arguments, &env)?;
    let lease = if let Some((path, record)) = &prepared.sidecar {
        let lease = support::SidecarLease::acquire(path)?;
        support::publish(path, &support::canonical(record)?)?;
        Some(lease)
    } else {
        None
    };
    let mut command = if let Some(previous) =
        env.get(OsStr::new("UMPIRE_GUEST_PREVIOUS_WRAPPER")).filter(|v| !v.is_empty())
    {
        let mut command = Command::new(previous);
        command.args(&prepared.command);
        command
    } else {
        let mut command = Command::new(&prepared.command[0]);
        command.args(&prepared.command[1..]);
        command
    };
    let status = command.status();
    match status {
        Ok(status) if status.success() => Ok(0),
        outcome => {
            if let Some(lease) = lease {
                lease.remove().map_err(|cleanup| {
                    let failure = match &outcome {
                        Ok(status) => format!("compiler exited with {status}"),
                        Err(error) => format!("compiler launch failed: {error}"),
                    };
                    invalid(format!("{failure}; sidecar cleanup failed: {cleanup}"))
                })?;
            }
            match outcome {
                Ok(status) => {
                    Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
                }
                Err(error) => Err(error.into()),
            }
        }
    }
}
