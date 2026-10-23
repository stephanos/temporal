use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::BTreeMap,
    fs,
    io::{Read, Write},
    os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt},
    path::{Path, PathBuf},
};

pub const SCHEMA: &str = "gomad.rust-wasm-installation/v1";
const MAXIMUM: u64 = 512 << 20;
fn failure(error: impl std::fmt::Display) -> String {
    format!("runner_failure: {error}")
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Admission {
    pub compiler_inputs: BTreeMap<PathBuf, String>,
    pub inputs_sha256: String,
    pub guest_configuration: Value,
    pub configuration_sha256: String,
    pub host_executable_sha256: String,
    pub explicit_helper: Option<Tool>,
    pub environment: BTreeMap<String, String>,
    pub arguments: Vec<String>,
    pub compile_cache_root: Option<PathBuf>,
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Tool {
    pub path: PathBuf,
    pub sha256: String,
}
#[derive(Serialize)]
pub struct Description {
    pub path: PathBuf,
    pub source: String,
    pub sha256: String,
    pub manifest: Installation,
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Installation {
    pub schema: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub admission: Option<Admission>,
    pub compiler_path: PathBuf,
    pub compiler_sha256: String,
    pub go_version: String,
    pub runtime_root: PathBuf,
    #[serde(default)]
    pub overlay_module_root: Option<PathBuf>,
    pub cache_root: PathBuf,
    pub profile: String,
    pub fuel: u64,
    pub memory_bytes: u64,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub runtime_implementation_sha256: Option<String>,
    pub model: Value,
    pub namespace: linked::NamespaceImage,
    pub storage: Option<linked::StorageInput>,
    #[serde(default)]
    pub network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
    #[serde(default)]
    pub build_tags: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub excluded_build_tags: Vec<String>,
}

pub(crate) fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let before = fs::symlink_metadata(path).map_err(failure)?;
    if !before.is_file() || before.len() > limit {
        return Err(failure("unsafe or oversized installation input"));
    }
    let mut file = fs::OpenOptions::new()
        .read(true)
        .custom_flags(
            (rustix::fs::OFlags::NOFOLLOW | rustix::fs::OFlags::NONBLOCK)
                .bits()
                .cast_signed(),
        )
        .open(path)
        .map_err(failure)?;
    let opened = file.metadata().map_err(failure)?;
    if !before.is_file()
        || before.dev() != opened.dev()
        || before.ino() != opened.ino()
        || opened.len() > limit
    {
        return Err(failure("unsafe or oversized installation input"));
    }
    let mut bytes = Vec::new();
    std::io::Read::by_ref(&mut file)
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(failure)?;
    let after = file.metadata().map_err(failure)?;
    if bytes.len() as u64 != opened.len()
        || after.len() != opened.len()
        || after.mtime() != opened.mtime()
        || after.mtime_nsec() != opened.mtime_nsec()
        || after.ctime() != opened.ctime()
        || after.ctime_nsec() != opened.ctime_nsec()
    {
        return Err(failure("installation input changed while reading"));
    }
    Ok(bytes)
}

pub(crate) fn executable(path: &Path, expected: &str) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(failure)?;
    if !path.is_absolute()
        || !meta.is_file()
        || meta.permissions().mode() & 0o111 == 0
        || !canonical::valid_digest(expected)
        || canonical::digest(&read(path, MAXIMUM)?) != expected
    {
        return Err(failure("pinned compiler identity mismatch"));
    }
    Ok(())
}

pub(crate) fn selected(explicit: Option<&Path>) -> Result<(PathBuf, &'static str), String> {
    let (path, source) = if let Some(path) = explicit.filter(|path| !path.as_os_str().is_empty()) {
        (path.to_path_buf(), "explicit")
    } else if let Some(path) = std::env::var_os("GOMAD_WASM_INSTALLATION").filter(|s| !s.is_empty())
    {
        (PathBuf::from(path), "environment")
    } else {
        (
            std::env::current_exe()
                .map_err(failure)?
                .with_file_name("gomad-wasm.json"),
            "executable-sibling",
        )
    };
    Ok((std::path::absolute(path).map_err(failure)?, source))
}

/// # Errors
/// Refuses a selected incompatible manifest without trying another installation.
pub fn resolve(explicit: Option<&Path>) -> Result<Description, String> {
    let (path, source) = selected(explicit)?;
    let raw = read(&path, 16 << 20)?;
    let header: Value = crate::qualify::spec::strict(&raw).map_err(failure)?;
    if header["schema"] != SCHEMA {
        return Err(failure("unsupported Rust installation format"));
    }
    let manifest: Installation = crate::qualify::spec::strict(&raw).map_err(failure)?;
    validate(&manifest)?;
    Ok(Description {
        path,
        source: source.into(),
        sha256: canonical::digest(&raw),
        manifest,
    })
}

pub(crate) fn validate(installed: &Installation) -> Result<(), String> {
    super::tags::validate(&installed.build_tags, &installed.excluded_build_tags)?;
    if installed.schema != SCHEMA
        || installed.helper_sha256 != linked::IMPLEMENTATION_SHA256
        || installed.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || !installed.cache_root.is_absolute()
        || !installed.compiler_path.is_absolute()
        || !canonical::valid_digest(&installed.compiler_sha256)
        || installed.fuel == 0
        || installed.memory_bytes == 0
        || !matches!(installed.go_version.as_str(), "go1.27.0" | "go1.27.1")
    {
        return Err(failure(
            "unsupported Rust installation or pinned implementation mismatch",
        ));
    }
    match installed.profile.as_str() {
        "gomad.wasi-cooperative/v1"
            if installed.go_version == "go1.27.1"
                && installed.runtime_root.is_absolute()
                && installed.runtime_implementation_sha256.as_deref()
                    == Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")) => {}
        "gomad.wasi-stock-exploratory/v1"
            if installed.runtime_implementation_sha256.is_none()
                && installed.runtime_root.as_os_str().is_empty() => {}
        _ => return Err(failure("unsupported runtime profile or implementation")),
    }
    if let Some(admission) = &installed.admission {
        let current = std::env::current_exe().map_err(failure)?;
        if canonical::digest(&read(&current, MAXIMUM)?) != admission.host_executable_sha256 {
            return Err(failure("installed host executable identity mismatch"));
        }
        let derived =
            super::config::convert(admission.guest_configuration.clone(), &installed.profile)?;
        if derived.model != installed.model
            || canonical::encode(&derived.namespace)? != canonical::encode(&installed.namespace)?
            || canonical::encode(&derived.storage)? != canonical::encode(&installed.storage)?
            || canonical::encode(&derived.network)? != canonical::encode(&installed.network)?
            || derived.environment != admission.environment
            || derived.arguments != admission.arguments
        {
            return Err(failure("installation configuration preimage mismatch"));
        }
        if admission.compiler_inputs.is_empty()
            || admission
                .compiler_inputs
                .iter()
                .any(|(path, digest)| !path.is_absolute() || !canonical::valid_digest(digest))
            || canonical::digest(&canonical::encode(&admission.compiler_inputs)?)
                != admission.inputs_sha256
            || canonical::digest(&canonical::encode(&admission.guest_configuration)?)
                != admission.configuration_sha256
            || !canonical::valid_digest(&admission.host_executable_sha256)
            || admission.explicit_helper.as_ref().is_some_and(|tool| {
                !tool.path.is_absolute() || tool.sha256 != admission.host_executable_sha256
            })
            || admission
                .compile_cache_root
                .as_ref()
                .is_some_and(|path| !path.is_absolute())
        {
            return Err(failure("installation admission identity mismatch"));
        }
    }
    Ok(())
}

pub(crate) fn admit_build(installed: &Installation) -> Result<(), String> {
    validate(installed)?;
    executable(&installed.compiler_path, &installed.compiler_sha256)?;
    if let Some(admission) = &installed.admission {
        for (path, expected) in &admission.compiler_inputs {
            if canonical::digest(&read(path, MAXIMUM)?) != *expected {
                return Err(failure(format!(
                    "pinned build input changed: {}",
                    path.display()
                )));
            }
        }
    }
    Ok(())
}

pub(crate) fn write(path: &Path, installed: &Installation) -> Result<(), String> {
    validate(installed)?;
    let bytes = canonical::encode(installed)?;
    if bytes.len() > 16 << 20 {
        return Err(failure("installation manifest exceeds size bound"));
    }
    let parent = path
        .parent()
        .filter(|path| !path.as_os_str().is_empty())
        .unwrap_or(Path::new("."));
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| failure("invalid installation output path"))?;
    crate::evidence::store::open::components(name)?;
    let before = fs::symlink_metadata(parent).map_err(failure)?;
    if !before.is_dir() {
        return Err(failure("installation output parent is not a directory"));
    }
    let directory = fs::OpenOptions::new()
        .read(true)
        .custom_flags(
            (rustix::fs::OFlags::DIRECTORY | rustix::fs::OFlags::NOFOLLOW)
                .bits()
                .cast_signed(),
        )
        .open(parent)
        .map_err(failure)?;
    let opened = directory.metadata().map_err(failure)?;
    if before.dev() != opened.dev() || before.ino() != opened.ino() {
        return Err(failure("installation output parent changed"));
    }
    let mut file = fs::File::from(
        rustix::fs::openat(
            &directory,
            name,
            rustix::fs::OFlags::WRONLY
                | rustix::fs::OFlags::CREATE
                | rustix::fs::OFlags::EXCL
                | rustix::fs::OFlags::NOFOLLOW
                | rustix::fs::OFlags::CLOEXEC,
            rustix::fs::Mode::from_raw_mode(0o600),
        )
        .map_err(failure)?,
    );
    file.write_all(&bytes).map_err(failure)?;
    file.sync_all().map_err(failure)?;
    directory.sync_all().map_err(failure)?;
    let after = fs::symlink_metadata(parent).map_err(failure)?;
    if after.dev() != opened.dev() || after.ino() != opened.ino() {
        return Err(failure("installation output parent changed"));
    }
    Ok(())
}
