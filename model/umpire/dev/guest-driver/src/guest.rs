//! Native guest orchestration, prerequisite admission and artifact publication.
use crate::{
    Error, Result,
    context::{Context, ToolRoot, caller_path},
    support,
};
use serde::Deserialize;
use std::{
    env,
    ffi::{OsStr, OsString},
    fs,
    io::Write,
    os::unix::{
        fs::{MetadataExt, OpenOptionsExt},
        process::ExitStatusExt,
    },
    path::{Path, PathBuf},
    process::{Command, Stdio},
};

fn invalid(message: impl Into<String>) -> Error {
    Error::Invalid(message.into())
}

fn probe(workspace: &Path, args: &[&str]) -> Result<Vec<u8>> {
    let output = Command::new("mise")
        .args(args)
        .current_dir(workspace)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .output()?;
    if !output.status.success() {
        return Err(invalid(format!(
            "mise {args:?} failed ({}): {}",
            output.status,
            String::from_utf8_lossy(&output.stderr)
        )));
    }
    Ok(output.stdout)
}

fn text(bytes: Vec<u8>) -> Result<String> {
    let value = String::from_utf8(bytes).map_err(|_| invalid("tool output must be UTF-8"))?;
    if value.trim().is_empty() {
        return Err(invalid("empty tool output"));
    }
    Ok(value.trim().to_owned())
}

#[derive(Deserialize)]
struct Pins {
    tools: Tools,
}
#[derive(Deserialize)]
struct Tools {
    rust: String,
    go: String,
    protoc: String,
}
#[derive(Deserialize)]
struct ToolchainFile {
    toolchain: Toolchain,
}
#[derive(Deserialize)]
struct Toolchain {
    channel: String,
}
#[derive(Deserialize)]
struct ToolInstallation {
    version: String,
    install_path: PathBuf,
    installed: bool,
    active: bool,
}

fn executable(path: &Path) -> bool {
    use std::os::unix::fs::PermissionsExt;
    fs::metadata(path).is_ok_and(|m| m.is_file() && m.permissions().mode() & 0o111 != 0)
}

/// Admit installed pinned tools and Rust source/target readiness on every freshness call.
fn prerequisites(workspace: &Path) -> Result<()> {
    let pins: Pins = toml::from_str(&fs::read_to_string(workspace.join("mise.toml"))?)?;
    let rust: ToolchainFile =
        toml::from_str(&fs::read_to_string(workspace.join("rust-toolchain.toml"))?)?;
    if pins.tools.rust != rust.toolchain.channel {
        return Err(invalid("Rust toolchain pins disagree"));
    }
    for (tool, pin, commands) in [
        ("rust", pins.tools.rust.as_str(), &["rustc", "cargo"][..]),
        ("go", pins.tools.go.as_str(), &["go"][..]),
        ("protoc", pins.tools.protoc.as_str(), &["protoc"][..]),
    ] {
        let entries: Vec<ToolInstallation> =
            serde_json::from_slice(&probe(workspace, &["ls", "--installed", "--json", tool])?)?;
        let mut matched = entries.iter().filter(|e| e.version == pin && e.installed && e.active);
        let entry =
            matched.next().ok_or_else(|| invalid(format!("missing pinned {tool} {pin}")))?;
        if matched.next().is_some() {
            return Err(invalid(format!("ambiguous installed {tool}")));
        }
        for command in commands {
            let compiler =
                [entry.install_path.join(command), entry.install_path.join("bin").join(command)]
                    .into_iter()
                    .find(|p| executable(p))
                    .ok_or_else(|| invalid(format!("missing executable {command}")))?;
            if *command == "rustc" {
                let rustup = compiler.with_file_name("rustup");
                if let Ok(up) = fs::metadata(&rustup) {
                    let cc = fs::metadata(&compiler)?;
                    if up.dev() == cc.dev() && up.ino() == cc.ino() {
                        let output = Command::new(rustup)
                            .args(["toolchain", "list"])
                            .env("MISE_AUTO_INSTALL", "false")
                            .env("MISE_EXEC_AUTO_INSTALL", "false")
                            .output()?;
                        if !output.status.success() {
                            return Err(invalid(format!(
                                "rustup probe failed: {}",
                                String::from_utf8_lossy(&output.stderr)
                            )));
                        }
                        let list = text(output.stdout)?;
                        if !list
                            .lines()
                            .filter_map(|l| l.split_whitespace().next())
                            .any(|s| s.starts_with(&format!("{pin}-")))
                        {
                            return Err(invalid(format!("missing pinned Rust {pin}")));
                        }
                    }
                }
            }
        }
    }
    let sysroot =
        PathBuf::from(text(probe(workspace, &["exec", "--", "rustc", "--print", "sysroot"])?)?);
    if !sysroot.join("lib/rustlib/src/rust/library/Cargo.toml").is_file() {
        return Err(invalid("missing rust-src"));
    }
    let libraries = sysroot.join("lib/rustlib/wasm32-wasip1/lib");
    let ready = fs::read_dir(libraries).is_ok_and(|entries| {
        entries.filter_map(std::result::Result::ok).any(|e| {
            e.file_type().is_ok_and(|t| t.is_file())
                && e.file_name().to_str().is_some_and(|n| {
                    n.starts_with("libstd-") && Path::new(n).extension() == Some(OsStr::new("rlib"))
                })
        })
    });
    if !ready {
        return Err(invalid("missing wasm32-wasip1 target"));
    }
    Ok(())
}

fn launcher(path: &Path, binary: &[u8]) -> Result<()> {
    use std::fs::OpenOptions;
    let parent = path.parent().ok_or_else(|| invalid("launcher needs parent"))?;
    fs::create_dir_all(parent)?;
    let temporary = parent.join(format!("launcher.{}.tmp", std::process::id()));
    let mut file = OpenOptions::new().write(true).create_new(true).mode(0o755).open(&temporary)?;
    let result = (|| -> std::io::Result<()> {
        file.write_all(binary)?;
        file.sync_all()?;
        drop(file);
        fs::rename(&temporary, path)
    })();
    if let Err(error) = result {
        fs::remove_file(&temporary)
            .map_err(|cleanup| invalid(format!("launcher: {error}; cleanup: {cleanup}")))?;
        return Err(error.into());
    }
    Ok(())
}

#[derive(Deserialize)]
struct Artifact {
    reason: String,
    target: Option<Target>,
    filenames: Option<Vec<PathBuf>>,
}
#[derive(Deserialize)]
struct Target {
    crate_types: Vec<String>,
}

/// Select exactly one cdylib wasm from Cargo's newline-delimited JSON stream.
fn artifact(bytes: &[u8]) -> Result<PathBuf> {
    let stream = std::str::from_utf8(bytes).map_err(|_| invalid("Cargo JSON must be UTF-8"))?;
    let mut candidates = Vec::new();
    for line in stream.lines() {
        let message: Artifact = serde_json::from_str(line)?;
        if message.reason == "compiler-artifact" {
            let target = message.target.ok_or_else(|| invalid("artifact missing target"))?;
            let files = message.filenames.ok_or_else(|| invalid("artifact missing filenames"))?;
            if target.crate_types.iter().any(|v| v == "cdylib") {
                candidates.extend(
                    files.into_iter().filter(|p| p.extension() == Some(OsStr::new("wasm"))),
                );
            }
        }
    }
    if candidates.len() != 1 {
        return Err(invalid(format!(
            "expected one guest wasm artifact, found {}",
            candidates.len()
        )));
    }
    Ok(candidates.remove(0))
}

/// Build or compare one guest. Returns 2 for stale output, 1 for native errors, child status for compiler failure.
pub fn run(arguments: &[OsString], freshness: bool) -> Result<i32> {
    if arguments.len() < 2
        || arguments.len() > 3
        || (arguments.len() == 3 && arguments[2] != "--check")
        || (freshness && arguments.len() != 2)
    {
        return Err(invalid(
            "usage: xtask guest <crate> <output> [--check]; guest-freshness <crate> <output>",
        ));
    }
    let tools = ToolRoot::local()?;
    let workspace = env::current_dir()?.canonicalize()?;
    let output = caller_path(PathBuf::from(&arguments[1]), &workspace);
    if freshness {
        prerequisites(tools.path())?;
    }
    let sysroot = text(probe(tools.path(), &["exec", "--", "rustc", "--print", "sysroot"])?)?;
    let compiler = text(probe(tools.path(), &["exec", "--", "rustc", "-V"])?)?;
    let cargo_home = env::var_os("CARGO_HOME")
        .map(PathBuf::from)
        .or_else(|| env::var_os("HOME").map(|p| PathBuf::from(p).join(".cargo")))
        .ok_or_else(|| invalid("missing CARGO_HOME/HOME"))?;
    let cargo_home = caller_path(cargo_home, &workspace);
    let target = caller_path(
        env::var_os("CARGO_TARGET_DIR").map_or_else(|| workspace.join("target"), PathBuf::from),
        &workspace,
    );
    fs::create_dir_all(&target)?;
    let target = target.canonicalize()?;
    let binary = fs::read(env::current_exe()?)?;
    let key = cache_key(&binary, &fs::read(workspace.join("Cargo.lock"))?, &compiler)?;
    let cache = target.join("guest-cache").join(&key);
    let wrapper = target.join("guest-wrappers").join(&key).join("guest-rustc");
    launcher(&wrapper, &binary)?;
    let metadata = metadata(tools.path(), &workspace)?;
    let lock = crate::lock::publish(&metadata, &workspace.join("Cargo.lock"), &cache)?;
    let previous =
        env::var_os("RUSTC_WRAPPER").filter(|p| p != wrapper.as_os_str()).unwrap_or_default();
    let context = Context {
        tool: tools.path().to_owned(),
        workspace: workspace.clone(),
        sysroot: sysroot.into(),
        cargo_home,
        target: cache.clone(),
        lock,
        compiler: compiler.into(),
        previous,
    };
    let flags = guest_flags(&context)?;
    let mut command = Command::new("mise");
    context.encode(&mut command);
    let built = command
        .args([
            OsStr::new("exec"),
            OsStr::new("--"),
            OsStr::new("cargo"),
            OsStr::new("build"),
            OsStr::new("-p"),
            &arguments[0],
        ])
        .args([
            "--lib",
            "--profile",
            "guest",
            "--target",
            "wasm32-wasip1",
            "-Zbuild-std=std,panic_unwind",
            "--locked",
            "--message-format=json-render-diagnostics",
        ])
        .arg("--manifest-path")
        .arg(workspace.join("Cargo.toml"))
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("RUSTC_WRAPPER", wrapper)
        .env("CARGO_TARGET_DIR", &cache)
        .current_dir(tools.path())
        .env("RUSTC_BOOTSTRAP", "1")
        .env("CARGO_ENCODED_RUSTFLAGS", flags)
        .stderr(Stdio::inherit())
        .output()?;
    if !built.status.success() {
        return Ok(built.status.code().unwrap_or_else(|| 128 + built.status.signal().unwrap_or(0)));
    }
    publish_artifact(&built.stdout, &output, freshness || arguments.len() == 3)
}

fn publish_artifact(stream: &[u8], output: &Path, check: bool) -> Result<i32> {
    let bytes = fs::read(artifact(stream)?)?;
    if check {
        let existing = match fs::read(output) {
            Ok(v) => Some(v),
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => None,
            Err(e) => return Err(e.into()),
        };
        if existing.as_deref() != Some(bytes.as_slice()) {
            let offset = existing
                .as_deref()
                .unwrap_or_default()
                .iter()
                .zip(&bytes)
                .position(|(a, b)| a != b)
                .unwrap_or_else(|| existing.as_ref().map_or(0, Vec::len).min(bytes.len()));
            let _ = writeln!(
                std::io::stderr(),
                "stale guest {}: existing={}, built={} bytes; first differing offset={offset}",
                output.display(),
                existing
                    .as_ref()
                    .map_or_else(|| "missing".into(), |v| format!("{} bytes", v.len())),
                bytes.len()
            );
            return Ok(2);
        }
    } else {
        support::publish(output, &bytes)?;
    }
    Ok(0)
}

fn guest_flags(context: &Context) -> Result<OsString> {
    let mut flags = env::var_os("CARGO_ENCODED_RUSTFLAGS").unwrap_or_default();
    if flags.is_empty()
        && let Some(value) = env::var_os("RUSTFLAGS")
    {
        let value = value.to_str().ok_or_else(|| {
            invalid("RUSTFLAGS must be UTF-8; use CARGO_ENCODED_RUSTFLAGS for raw argv")
        })?;
        flags = value.split_whitespace().collect::<Vec<_>>().join("\x1f").into();
    }
    let mut append = |value: OsString| {
        if !flags.is_empty() {
            flags.push("\x1f");
        }
        flags.push(value);
    };
    for flag in context.remaps()?.flags() {
        append(flag);
    }
    for flag in [
        "-Cpanic=unwind",
        "-Ctarget-feature=+exception-handling",
        "-Cllvm-args=-wasm-use-legacy-eh=false",
    ] {
        append(flag.into());
    }
    Ok(flags)
}

pub(crate) fn cache_key(driver: &[u8], lock: &[u8], compiler: &str) -> Result<String> {
    Ok(support::hash(&support::canonical(&(support::hash(driver), support::hash(lock), compiler))?))
}

fn metadata(tools: &Path, workspace: &Path) -> Result<Vec<u8>> {
    let metadata = Command::new("mise")
        .args([
            "exec",
            "--",
            "cargo",
            "metadata",
            "--format-version",
            "1",
            "--locked",
            "--manifest-path",
        ])
        .arg(workspace.join("Cargo.toml"))
        .current_dir(tools)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .output()?;
    if !metadata.status.success() {
        return Err(invalid(format!(
            "Cargo metadata failed ({}): {}",
            metadata.status,
            String::from_utf8_lossy(&metadata.stderr)
        )));
    }
    Ok(metadata.stdout)
}
