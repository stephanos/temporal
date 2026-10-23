use super::manifest::{self, Installation};
use crate::{
    campaign::prepare::{inventory, process},
    evidence::canonical,
};
use serde::Deserialize;
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Overlay {
    pub path: PathBuf,
    pub identity: String,
    pub implementation_sha256: String,
}

pub(crate) fn overlay(
    installed: &Installation,
    root: &Path,
    environment: &BTreeMap<String, String>,
    scratch: &Path,
    deadline: Instant,
) -> Result<Overlay, String> {
    let module_root = installed
        .overlay_module_root
        .as_deref()
        .or_else(|| {
            installed
                .runtime_root
                .parent()
                .and_then(Path::parent)
                .and_then(Path::parent)
        })
        .ok_or("runner_failure: overlay module owner is required")?;
    if !module_root.is_absolute() {
        return Err("runner_failure: absolute overlay module owner is required".into());
    }
    let mut host_environment = environment.clone();
    for name in ["GOOS", "GOARCH", "GOEXPERIMENT"] {
        host_environment.remove(name);
    }
    host_environment.insert("GOROOT".into(), root.to_string_lossy().into_owned());
    let result = process::command(
        &installed.compiler_path,
        &[
            "run".into(),
            "-mod=readonly".into(),
            "./internal/backends/wasm/toolchain/cmd/overlay".into(),
            root.to_string_lossy().into_owned(),
            installed.runtime_root.to_string_lossy().into_owned(),
            installed
                .cache_root
                .join("runtime-overlay")
                .to_string_lossy()
                .into_owned(),
        ],
        module_root,
        &host_environment,
        scratch,
        4 << 20,
        deadline,
    )?;
    let generated: Overlay = serde_json::from_slice(&result).map_err(|e| e.to_string())?;
    if !generated.path.is_absolute()
        || !canonical::valid_digest(&generated.identity)
        || generated.implementation_sha256 != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
    {
        return Err(
            "runner_failure: invalid overlay identity or runtime implementation changed".into(),
        );
    }
    Ok(generated)
}

pub(super) fn tree(path: &Path, inputs: &mut BTreeMap<PathBuf, String>) -> Result<(), String> {
    let metadata = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if metadata.is_dir() {
        for entry in fs::read_dir(path).map_err(|e| e.to_string())? {
            tree(&entry.map_err(|e| e.to_string())?.path(), inputs)?;
        }
    } else if metadata.is_file() {
        inputs.insert(
            path.into(),
            canonical::digest(&manifest::read(path, 512 << 20)?),
        );
    } else {
        return Err("runner_failure: runtime input is not a regular file or directory".into());
    }
    Ok(())
}

fn generator_inputs(
    installed: &Installation,
    root: &Path,
    environment: &BTreeMap<String, String>,
    scratch: &Path,
    deadline: Instant,
) -> Result<BTreeMap<PathBuf, String>, String> {
    let mut host_environment = environment.clone();
    for name in ["GOOS", "GOARCH", "GOEXPERIMENT"] {
        host_environment.remove(name);
    }
    host_environment.insert("GOROOT".into(), root.to_string_lossy().into_owned());
    let module = installed
        .overlay_module_root
        .as_deref()
        .ok_or("runner_failure: overlay module owner missing")?;
    let listed = process::command(
        &installed.compiler_path,
        &[
            "list".into(),
            "-mod=readonly".into(),
            "-deps".into(),
            "-json".into(),
            "./internal/backends/wasm/toolchain/cmd/overlay".into(),
        ],
        module,
        &host_environment,
        scratch,
        128 << 20,
        deadline,
    )?;
    inventory::sources(&listed, &root.join("pkg/include"), &BTreeMap::new())
}

pub(crate) fn pin(installed: &Installation) -> Result<BTreeMap<PathBuf, String>, String> {
    pin_before(installed, Instant::now() + Duration::from_secs(600))
}

#[expect(
    clippy::too_many_lines,
    reason = "Toolchain pinning verifies the complete compiler and generator inventory before and after overlay generation"
)]
pub(crate) fn pin_before(
    installed: &Installation,
    deadline: Instant,
) -> Result<BTreeMap<PathBuf, String>, String> {
    if Instant::now() >= deadline {
        return Err("runner_failure: toolchain pin deadline exceeded".into());
    }
    manifest::executable(&installed.compiler_path, &installed.compiler_sha256)?;
    let temporary = process::Temporary::new(&installed.cache_root)?;
    let directory = std::env::current_dir().map_err(|e| e.to_string())?;
    let environment = process::environment()?;
    let version = process::command(
        &installed.compiler_path,
        &["version".into()],
        &directory,
        &environment,
        &temporary.0,
        64 << 10,
        deadline,
    )?;
    if !String::from_utf8_lossy(&version)
        .starts_with(&format!("go version {} ", installed.go_version))
    {
        return Err("runner_failure: compiler version does not match installation".into());
    }
    let (mut inputs, root) = inventory::compiler(
        &installed.compiler_path,
        &directory,
        &environment,
        &temporary.0,
        deadline,
    )?;
    inputs.insert(
        installed.compiler_path.clone(),
        installed.compiler_sha256.clone(),
    );
    if installed.profile == "gomad.wasi-cooperative/v1" {
        let cache = process::command(
            &installed.compiler_path,
            &["env".into(), "GOMODCACHE".into()],
            &directory,
            &environment,
            &temporary.0,
            64 << 10,
            deadline,
        )?;
        let cache = PathBuf::from(String::from_utf8_lossy(&cache).trim());
        if root.starts_with(fs::canonicalize(&cache).unwrap_or(cache)) {
            return Err("runner_failure: cooperative compiler must be outside module cache".into());
        }
        let module = installed
            .overlay_module_root
            .as_deref()
            .ok_or("runner_failure: overlay module owner missing")?;
        for name in ["go.mod", "go.sum"] {
            let path = module.join(name);
            inputs.insert(
                path.clone(),
                canonical::digest(&manifest::read(&path, 512 << 20)?),
            );
        }
        tree(&installed.runtime_root, &mut inputs)?;
        for name in [
            "internal/backends/wasm/toolchain",
            "internal/model/choice/schema",
        ] {
            tree(&module.join(name), &mut inputs)?;
        }
        for name in [
            "internal/model/choice/trace.go",
            "internal/model/choice/tape.go",
        ] {
            let path = module.join(name);
            inputs.insert(
                path.clone(),
                canonical::digest(&manifest::read(&path, 512 << 20)?),
            );
        }
        inputs.extend(generator_inputs(
            installed,
            &root,
            &environment,
            &temporary.0,
            deadline,
        )?);
        inventory::verify(&inputs)?;
        let generated = overlay(installed, &root, &environment, &temporary.0, deadline)?;
        let raw = manifest::read(&generated.path, 16 << 20)?;
        let value: serde_json::Value = serde_json::from_slice(&raw).map_err(|e| e.to_string())?;
        let replacements: BTreeMap<PathBuf, PathBuf> = serde_json::from_value(
            value
                .get("Replace")
                .cloned()
                .ok_or("runner_failure: overlay replacements missing")?,
        )
        .map_err(|e| e.to_string())?;
        for (original, replacement) in replacements {
            for path in [original, replacement] {
                if path.is_file() {
                    inputs.insert(
                        path.clone(),
                        canonical::digest(&manifest::read(&path, 512 << 20)?),
                    );
                }
            }
        }
        inputs.insert(generated.path, canonical::digest(&raw));
    }
    inventory::verify(&inputs)?;
    manifest::executable(&installed.compiler_path, &installed.compiler_sha256)?;
    if Instant::now() >= deadline {
        return Err("runner_failure: toolchain pin deadline exceeded".into());
    }
    Ok(inputs)
}
