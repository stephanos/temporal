use super::{Gate, STAGES, boundary};
use crate::{Error, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    env, fs,
    io::{self, Write},
    os::unix::ffi::{OsStrExt, OsStringExt},
    path::{Path, PathBuf},
    process::Command,
    time::Instant,
};
use umpire_guest_driver::context::ToolRoot;

#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Input {
    path_hex: String,
    sha256: String,
    executable: bool,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Provenance {
    version: u32,
    source: PathBuf,
    root: PathBuf,
    target: PathBuf,
    inputs: Vec<Input>,
    excluded: Vec<String>,
}
fn hex(bytes: &[u8]) -> String {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    bytes
        .iter()
        .flat_map(|byte| {
            [char::from(DIGITS[usize::from(byte >> 4)]), char::from(DIGITS[usize::from(byte & 15)])]
        })
        .collect()
}
fn digest(bytes: &[u8]) -> String {
    hex(&Sha256::digest(bytes))
}
fn inventory(root: &Path, target: &Path) -> Result<Vec<Input>> {
    use std::os::unix::fs::PermissionsExt;
    boundary::files(root, target)?
        .into_iter()
        .map(|path| {
            Ok(Input {
                path_hex: hex(path
                    .strip_prefix(root)
                    .map_err(|e| Error::Invalid(e.to_string()))?
                    .as_os_str()
                    .as_bytes()),
                sha256: digest(&fs::read(&path)?),
                executable: fs::metadata(path)?.permissions().mode() & 0o111 != 0,
            })
        })
        .collect()
}
fn path(input: &Input) -> Result<PathBuf> {
    let bytes = (0..input.path_hex.len())
        .step_by(2)
        .map(|index| {
            input
                .path_hex
                .get(index..index + 2)
                .and_then(|part| u8::from_str_radix(part, 16).ok())
                .ok_or_else(|| Error::Invalid("invalid standalone input path".into()))
        })
        .collect::<Result<Vec<_>>>()?;
    let path = PathBuf::from(std::ffi::OsString::from_vec(bytes));
    if path.as_os_str().is_empty()
        || path.components().any(|p| !matches!(p, std::path::Component::Normal(_)))
    {
        return Err(Error::Invalid("escaping standalone input path".into()));
    }
    Ok(path)
}
fn local_configuration(root: &Path) -> Result<()> {
    ToolRoot::admit(root)?;
    for name in [
        "Cargo.lock",
        "mise.lock",
        ".cargo/config.toml",
        ".config/nextest.toml",
        "dev/guest-driver/Cargo.toml",
    ] {
        let file = root.join(name);
        if !file.is_file() || !fs::canonicalize(file)?.starts_with(root) {
            return Err(Error::Invalid(format!(
                "missing or escaping standalone configuration: {name}"
            )));
        }
    }
    Ok(())
}
fn snapshot(source: &Path, source_target: &Path, root: &Path, target: &Path) -> Result<Provenance> {
    let inputs = inventory(source, source_target)?;
    fs::create_dir(root)?;
    for input in &inputs {
        let relative = path(input)?;
        let original = source.join(&relative);
        let destination = root.join(&relative);
        fs::create_dir_all(
            destination.parent().ok_or_else(|| Error::Invalid("input has no parent".into()))?,
        )?;
        let bytes = fs::read(&original)?;
        if digest(&bytes) != input.sha256
            || fs::symlink_metadata(&original)?.file_type().is_symlink()
        {
            return Err(Error::Invalid("standalone source changed while copying".into()));
        }
        fs::write(&destination, bytes)?;
        fs::set_permissions(&destination, fs::metadata(original)?.permissions())?;
    }
    if inventory(source, source_target)? != inputs || inventory(root, target)? != inputs {
        return Err(Error::Invalid("standalone source changed while copying".into()));
    }
    Ok(Provenance {
        version: 1,
        source: source.to_owned(),
        root: root.to_owned(),
        target: target.to_owned(),
        inputs,
        excluded: [".git", "target", "build", ".bin"].map(String::from).to_vec(),
    })
}
pub(super) fn admit(manifest: &Path, gate: &Gate) -> Result<PathBuf> {
    let bytes =
        fs::read(manifest).map_err(|e| Error::Invalid(format!("standalone provenance: {e}")))?;
    let provenance: Provenance = serde_json::from_slice(&bytes)?;
    if provenance.version != 1
        || provenance.root != gate.root
        || provenance.target != gate.target
        || provenance.source == provenance.root
        || provenance.root.starts_with(&provenance.source)
        || env::var_os("UMPIRE_STANDALONE_CHILD").as_deref() != Some(std::ffi::OsStr::new("1"))
        || !env::current_exe()?.starts_with(&gate.target)
        || !gate.target.join("debug/umpire-guest-driver").is_file()
    {
        return Err(Error::Invalid("standalone provenance/context mismatch".into()));
    }
    for name in [".git", ".bin", "target"] {
        if gate.root.join(name).exists() {
            return Err(Error::Invalid(format!("unexpected standalone copied cache: {name}")));
        }
    }
    if inventory(&gate.root, &gate.target)? != provenance.inputs {
        return Err(Error::Invalid("standalone provenance input hashes changed".into()));
    }
    local_configuration(&gate.root)?;
    let records = manifest.with_file_name("stages.jsonl");
    if records.exists() {
        return Err(Error::Invalid("standalone recursion or reused stage receipt".into()));
    }
    Ok(records)
}

fn clean_command(gate: &Gate, root: &Path, target: &Path, program: &str) -> Result<Command> {
    let mut command = Command::new(program);
    command.env_clear();
    for name in [
        "HOME",
        "USER",
        "LOGNAME",
        "LANG",
        "LC_ALL",
        "TZ",
        "RUSTUP_HOME",
        "JAVA_HOME",
        "HTTP_PROXY",
        "HTTPS_PROXY",
        "ALL_PROXY",
        "NO_PROXY",
        "http_proxy",
        "https_proxy",
        "all_proxy",
        "no_proxy",
        "SSL_CERT_FILE",
        "SSL_CERT_DIR",
        "CARGO_BUILD_JOBS",
        "RUST_TEST_THREADS",
        "UMPIRE_APALACHE",
    ] {
        if let Some(value) = env::var_os(name) {
            command.env(name, value);
        }
    }
    let paths = env::split_paths(&env::var_os("PATH").unwrap_or_default())
        .filter(|path| {
            path.is_absolute()
                && !gate
                    .root
                    .ancestors()
                    .filter(|ancestor| ancestor.join(".git").exists())
                    .any(|ancestor| path.starts_with(ancestor))
        })
        .collect::<Vec<_>>();
    if let Ok(path) = env::join_paths(paths) {
        command.env("PATH", path);
    }
    let trusted = env::join_paths([root.to_owned(), target.join("tmp")]).map_err(|error| {
        Error::Invalid(format!("invalid standalone trusted config roots: {error}"))
    })?;
    command
        .env("MISE_TRUSTED_CONFIG_PATHS", trusted)
        .current_dir(root)
        .env("TMPDIR", target.join("tmp"))
        .env("CARGO_HOME", &gate.home)
        .env("CARGO_TARGET_DIR", target)
        .env("CARGO_INCREMENTAL", "0")
        .env("GOCACHE", target.join("go-build"))
        .env("GOENV", "off")
        .env("GOWORK", "off")
        .env("GOTOOLCHAIN", "local")
        .env("UMPIRE_TOOL_ROOT", root)
        .env("UMPIRE_BACKENDS", "require")
        .env("UMPIRE_QUICK_CACHE", "0")
        .env("UMPIRE_CACHE_DISABLE", "1")
        .env("UMPIRE_STANDALONE_CHILD", "1")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    crate::test_run::configure(
        &mut command,
        true,
        env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
    );
    Ok(command)
}
fn refusal(result: Result<()>, expected: &str) -> Result<()> {
    match result {
        Err(error) if error.to_string().contains(expected) => Ok(()),
        other => Err(Error::Invalid(format!(
            "standalone negative control expected {expected}: {other:?}"
        ))),
    }
}
fn controls(root: &Path, target: &Path, outside: &Path) -> Result<Vec<String>> {
    let policy: toml::Value =
        toml::from_str(&fs::read_to_string(root.join("dev/gate/policy/framework-boundary.txt"))?)?;
    let forbidden = policy["forbidden"]
        .as_array()
        .and_then(|v| v.first())
        .and_then(toml::Value::as_str)
        .ok_or_else(|| Error::Invalid("missing boundary control vocabulary".into()))?;
    let planted = root.join("standalone-control.rs");
    fs::write(&planted, format!("fn {forbidden}_control() {{}}\n"))?;
    let result = boundary::check(root, target);
    fs::remove_file(planted)?;
    refusal(result, "framework forbidden name")?;
    fs::create_dir_all(outside.join("src"))?;
    fs::write(outside.join("src/lib.rs"), "pub fn outside_control() {}\n")?;
    fs::write(
        outside.join("Cargo.toml"),
        format!("[package]\nname='{forbidden}-control'\nversion='0.0.0'\n"),
    )?;
    let manifest = root.join("dev/xtask/Cargo.toml");
    let original = fs::read(&manifest)?;
    for foreign in [outside.to_string_lossy().into_owned(), "../../../..".into()] {
        let mut bytes = original.clone();
        bytes.extend(format!("\n[target.'cfg(any())'.dependencies]\nstandalone-outside = {{ path = {foreign:?} }}\n").as_bytes());
        fs::write(&manifest, bytes)?;
        let result = boundary::check(root, target);
        fs::write(&manifest, &original)?;
        refusal(result, "framework dependency containment: path escapes framework root")?;
    }
    fs::write(
        root.join("dev/xtask/src/standalone-control.rs"),
        "fn probe() { let _ = std::path::Path::new(env!(\"CARGO_MANIFEST_DIR\")).join(\"../../../..\"); }\n",
    )?;
    let result = boundary::check(root, target);
    fs::remove_file(root.join("dev/xtask/src/standalone-control.rs"))?;
    refusal(result, "framework source-path containment: path escapes framework root")?;
    let graph = serde_json::json!({"packages": [{"source": null, "manifest_path": outside.join("Cargo.toml")}]});
    refusal(
        boundary::metadata(root, &serde_json::to_vec(&graph)?),
        "framework resolved dependency containment",
    )?;
    let config = root.join("mise.toml");
    let bytes = fs::read(&config)?;
    fs::remove_file(&config)?;
    let result = ToolRoot::admit(root).map(|_| ());
    fs::write(&config, bytes)?;
    refusal(result, "missing or escaping framework configuration: mise.toml")?;
    Ok([
        "forbidden-name",
        "existing-outside-dependency",
        "root-escape-dependency",
        "manifest-root-source-escape",
        "resolved-outside-dependency",
        "missing-local-configuration",
    ]
    .map(String::from)
    .to_vec())
}
pub(super) fn record(path: &Path, name: &str, code: Option<i32>, seconds: f64) -> Result<()> {
    let mut file = fs::OpenOptions::new().create(true).append(true).open(path)?;
    serde_json::to_writer(
        &mut file,
        &serde_json::json!({"stage": name, "exit": code, "seconds": seconds}),
    )?;
    writeln!(file)?;
    Ok(())
}
fn tool_versions(gate: &Gate, root: &Path, target: &Path, manifest: &Path) -> Result<()> {
    let mut tools = serde_json::Map::new();
    let mut selected = vec!["rustc", "cargo", "go", "java", "protoc"];
    if root.join(".config/nextest.toml").exists() {
        selected.push("cargo-nextest");
    }
    for tool in selected {
        let mut command = clean_command(gate, root, target, "mise")?;
        command.args([
            "exec",
            "--",
            tool,
            if tool == "java" {
                "-version"
            } else if tool == "go" {
                "version"
            } else {
                "--version"
            },
        ]);
        let output = command.output()?;
        let mut which = clean_command(gate, root, target, "mise")?;
        if matches!(tool, "rustc" | "cargo") {
            which.args(["exec", "--", "rustc", "--print", "sysroot"]);
        } else {
            which.args(["which", tool]);
        }
        let selected = which.output()?;
        if !selected.status.success() {
            return Err(Error::Invalid(format!("standalone tool path admission failed: {tool}")));
        }
        let selected_path = PathBuf::from(std::ffi::OsString::from_vec(
            selected.stdout.strip_suffix(b"\n").unwrap_or(&selected.stdout).to_vec(),
        ));
        let selected_path = if matches!(tool, "rustc" | "cargo") {
            selected_path.join("bin").join(tool)
        } else {
            selected_path
        };
        let selected_path = fs::canonicalize(selected_path)?;
        tools.insert(tool.into(), serde_json::json!({"exit": output.status.code(), "stdout": String::from_utf8_lossy(&output.stdout), "stderr": String::from_utf8_lossy(&output.stderr), "executable": selected_path, "executable_sha256": digest(&fs::read(&selected_path)?)}));
        fs::write(manifest.with_file_name("tools.json"), serde_json::to_vec_pretty(&tools)?)?;
        if !output.status.success() {
            return Err(Error::Invalid(format!("standalone pinned tool admission failed: {tool}")));
        }
    }
    Ok(())
}
pub(super) fn run(gate: &Gate) -> Result<i32> {
    if env::var_os("UMPIRE_STANDALONE_CHILD").is_some() {
        return Err(Error::Invalid("standalone recursion refused".into()));
    }
    let started = Instant::now();
    let output = match env::var_os("UMPIRE_STANDALONE_OUTPUT") {
        Some(path) => {
            let path = PathBuf::from(path);
            if !path.is_absolute() {
                return Err(Error::Invalid("standalone output must be absolute".into()));
            }
            fs::create_dir_all(&path)?;
            tempfile::Builder::new().prefix("proof-").tempdir_in(path)?.keep()
        }
        None => tempfile::Builder::new().prefix("umpire-standalone-").tempdir()?.keep(),
    };
    let output = fs::canonicalize(output)?;
    if output.starts_with(&gate.root)
        || gate
            .root
            .ancestors()
            .any(|ancestor| ancestor.join(".git").exists() && output.starts_with(ancestor))
    {
        return Err(Error::Invalid("standalone copy must be outside repository".into()));
    }
    writeln!(io::stdout(), "standalone proof: {} (tier=validate, TLC=require)", output.display())?;
    let root = output.join("framework");
    let target = output.join("target");
    let provenance = snapshot(&gate.root, &gate.target, &root, &target)?;
    let manifest = output.join("inputs.json");
    fs::write(&manifest, serde_json::to_vec_pretty(&provenance)?)?;
    local_configuration(&root)?;
    fs::create_dir_all(target.join("tmp"))?;
    let controls = controls(&root, &target, &output.join("foreign"))?;
    fs::write(output.join("controls.json"), serde_json::to_vec_pretty(&controls)?)?;
    if inventory(&root, &target)? != provenance.inputs {
        return Err(Error::Invalid("standalone controls changed source inputs".into()));
    }
    tool_versions(gate, &root, &target, &manifest)?;
    for (name, program, args) in [
        (
            "standalone-local-tools",
            "mise",
            vec![
                "exec",
                "--",
                "cargo",
                "build",
                "--locked",
                "-p",
                "xtask",
                "-p",
                "umpire-guest-driver",
                "--bins",
            ],
        ),
        ("standalone-tlc-tool", "/bin/bash", vec!["scripts/fetch-tlc.sh"]),
    ] {
        let mut command = clean_command(gate, &root, &target, program)?;
        command.args(args);
        let setup_started = Instant::now();
        let code = gate.execute(name, command)?;
        record(
            &output.join("setup.jsonl"),
            name,
            Some(code),
            setup_started.elapsed().as_secs_f64(),
        )?;
        if code != 0 {
            return Ok(code);
        }
    }
    let mut command = clean_command(gate, &root, &target, "mise")?;
    command.args(["exec", "--", "cargo", "xtask", "gate", "--standalone-child"]).arg(&manifest);
    let code = gate.execute("standalone-child", command)?;
    finish(&output, &target, &manifest, code, started.elapsed().as_secs_f64())
}
fn artifact(path: &Path) -> serde_json::Value {
    let files = path.is_dir().then(|| inventory(path, &path.join("target")).ok()).flatten();
    let bytes = if path.is_dir() {
        files.as_ref().and_then(|files| serde_json::to_vec(files).ok())
    } else {
        fs::read(path).ok()
    };
    serde_json::json!({"path": path, "sha256": bytes.map(|bytes| digest(&bytes)), "files": files})
}
fn finish(output: &Path, target: &Path, manifest: &Path, code: i32, seconds: f64) -> Result<i32> {
    let records = fs::read_to_string(output.join("stages.jsonl")).unwrap_or_default();
    let stages = records
        .lines()
        .map(serde_json::from_str::<serde_json::Value>)
        .collect::<std::result::Result<Vec<_>, _>>()?;
    let expected = STAGES
        .iter()
        .filter(|(name, _)| *name != "standalone")
        .map(|(name, _)| *name)
        .collect::<Vec<_>>();
    let actual = stages.iter().filter_map(|value| value["stage"].as_str()).collect::<Vec<_>>();
    let stages_complete = actual == expected && stages.iter().all(|value| value["exit"] == 0);
    let mut binaries = serde_json::Map::new();
    for name in [
        "debug/xtask",
        "debug/umpire-guest-driver",
        "xtask-bootstrap/debug/xtask",
        "xtask-bootstrap/debug/umpire-guest-driver",
        "guest/panic-control-reports",
        "guest/current-contract-reports",
    ] {
        let binary = target.join(name);
        binaries.insert(
            name.into(),
            serde_json::json!({"path": binary, "sha256": fs::read(&binary).ok().map(|bytes| digest(&bytes))}),
        );
    }
    let provenance: Provenance = serde_json::from_slice(&fs::read(manifest)?)?;
    let source_unchanged = inventory(&provenance.root, target)? == provenance.inputs;
    let mut artifacts = serde_json::Map::new();
    for (base, names) in [
        (
            target,
            &[
                "wasm32-wasip1/guest/umpire_stock_guest.wasm",
                "wasm32-wasip1/guest/umpire_kettle_guest.wasm",
                "wasm32-wasip1/guest/stateful_control.wasm",
                "wasm32-wasip1/guest/caught_panic_control.wasm",
                "wasm32-wasip1/guest/overflow_control.wasm",
                "panic-native",
                "current-contract-native",
            ][..],
        ),
        (
            provenance.root.as_path(),
            &[
                "build/tlc/tla2tools-1.7.4.jar",
                ".bin/protoc-gen-go-v1.36.6",
                ".bin/protoc-gen-go-grpc-v1.3.0",
            ][..],
        ),
    ] {
        for name in names {
            let path = base.join(name);
            artifacts.insert((*name).into(), artifact(&path));
        }
    }
    let complete = stages_complete
        && source_unchanged
        && binaries.values().chain(artifacts.values()).all(|value| !value["sha256"].is_null());
    fs::write(
        output.join("result.json"),
        serde_json::to_vec_pretty(&serde_json::json!({
            "version": 1, "tier": "validate", "backend": "require", "apalache": env::var("UMPIRE_APALACHE").unwrap_or_else(|_| "skip (default)".into()),
            "profile": "dev", "nextest_profile": "required", "compile_jobs": env::var("CARGO_BUILD_JOBS").ok(), "test_threads": crate::test_run::test_threads(env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require")), "skipped_child_stages": ["standalone"], "seconds": seconds, "child_exit": code, "complete": complete,
            "manifest_sha256": digest(&fs::read(manifest)?), "stages_sha256": digest(records.as_bytes()), "binaries": binaries,
            "expected_stages": expected, "actual_stages": actual, "source_unchanged": source_unchanged, "artifacts": artifacts,
        }))?,
    )?;
    if code != 0 {
        return Ok(code);
    }
    if !complete {
        return Err(Error::Invalid("standalone incomplete or repeated framework stages".into()));
    }
    Ok(0)
}

#[cfg(test)]
#[allow(clippy::unwrap_used, clippy::expect_used)]
mod tests {
    use super::*;
    #[test]
    fn isolated_child_uses_the_reported_required_profile_and_thread_cap() {
        let directory = tempfile::tempdir().unwrap();
        let source = directory.path().join("source");
        fs::create_dir(&source).unwrap();
        let root = directory.path().join("copy");
        let target = directory.path().join("target");
        let provenance = snapshot(&source, &source.join("target"), &root, &target).unwrap();
        let manifest = directory.path().join("inputs.json");
        fs::write(&manifest, serde_json::to_vec(&provenance).unwrap()).unwrap();
        let gate = Gate {
            root: source,
            target: directory.path().join("native"),
            home: directory.path().join("cargo"),
            required: true,
            timed: false,
            child_records: None,
        };
        let mut command = clean_command(&gate, &root, &target, "/bin/sh").unwrap();
        let output = command.args(["-c", "printf '%s:%s:%s:%s' \"$NEXTEST_PROFILE\" \"$NEXTEST_TEST_THREADS\" \"$UMPIRE_QUICK_CACHE\" \"$UMPIRE_CACHE_DISABLE\""]).output().unwrap();
        assert!(output.status.success());
        assert_eq!(finish(directory.path(), &target, &manifest, 37, 0.0).unwrap(), 37);
        let receipt: serde_json::Value =
            serde_json::from_slice(&fs::read(directory.path().join("result.json")).unwrap())
                .unwrap();
        assert_eq!(receipt["nextest_profile"], "required");
        let threads = if env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require") {
            1
        } else {
            4
        };
        assert_eq!(receipt["test_threads"], threads);
        assert_eq!(receipt["child_exit"], 37);
        assert_eq!(receipt["complete"], false);
        assert_eq!(String::from_utf8(output.stdout).unwrap(), format!("required:{threads}:0:1"));
    }
    #[test]
    fn copied_config_trust_is_limited_to_copy_and_owned_scratch() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("copy");
        let target = directory.path().join("target");
        fs::create_dir(&root).unwrap();
        fs::create_dir_all(target.join("tmp")).unwrap();
        let gate = Gate {
            root: directory.path().join("source"),
            target: directory.path().join("native"),
            home: directory.path().join("cargo"),
            required: true,
            timed: false,
            child_records: None,
        };
        let mut command = clean_command(&gate, &root, &target, "mise").unwrap();
        let output = command.args(["settings", "get", "trusted_config_paths"]).output().unwrap();
        assert!(output.status.success(), "{output:?}");
        let trusted: Vec<PathBuf> = serde_json::from_slice(&output.stdout).unwrap();
        assert_eq!(trusted, [root, target.join("tmp")]);
    }
    #[test]
    fn report_directory_artifact_hashes_every_generated_file() {
        let directory = tempfile::tempdir().unwrap();
        fs::write(directory.path().join("request.pb"), b"request").unwrap();
        fs::write(directory.path().join("response.pb"), b"response").unwrap();
        let before = artifact(directory.path());
        assert!(before["sha256"].is_string());
        assert_eq!(before["files"].as_array().unwrap().len(), 2);
        fs::write(directory.path().join("response.pb"), b"changed response").unwrap();
        assert_ne!(before["sha256"], artifact(directory.path())["sha256"]);
        assert!(artifact(&directory.path().join("missing"))["sha256"].is_null());
    }
    #[test]
    fn copy_uses_current_bytes_and_excludes_only_generated_roots() {
        let directory = tempfile::tempdir().unwrap();
        let source = directory.path().join("source");
        fs::create_dir(&source).unwrap();
        fs::write(source.join("current"), b"uncommitted candidate").unwrap();
        fs::create_dir(source.join("target")).unwrap();
        fs::write(source.join("target/cache"), b"must not copy").unwrap();
        fs::create_dir(source.join(".cargo")).unwrap();
        fs::write(source.join(".cargo/config.toml"), b"local config").unwrap();
        let copy = directory.path().join("copy");
        let target = directory.path().join("output");
        let proof = snapshot(&source, &source.join("target"), &copy, &target).unwrap();
        assert_eq!(proof.inputs.len(), 2);
        assert_eq!(fs::read(copy.join("current")).unwrap(), b"uncommitted candidate");
        assert!(!copy.join("target").exists());
        assert_eq!(proof.inputs, inventory(&copy, &target).unwrap());
        fs::write(copy.join("current"), b"changed").unwrap();
        assert_ne!(proof.inputs, inventory(&copy, &target).unwrap());
    }
    #[test]
    fn copy_refuses_existing_and_dangling_source_symlinks() {
        let directory = tempfile::tempdir().unwrap();
        let source = directory.path().join("source");
        fs::create_dir(&source).unwrap();
        for destination in [directory.path().to_owned(), directory.path().join("missing")] {
            std::os::unix::fs::symlink(destination, source.join("escape")).unwrap();
            let result = snapshot(
                &source,
                &source.join("target"),
                &directory.path().join("copy"),
                &directory.path().join("output"),
            );
            assert!(result.unwrap_err().to_string().contains("framework source symlink refused"));
            fs::remove_file(source.join("escape")).unwrap();
        }
    }
}
