use super::{
    config,
    manifest::{self, Admission, Installation},
    toolchain,
};
use crate::{
    config::repo::{Derivation, Repo},
    evidence::canonical,
};
use gomad_wasmhost::linked;
use rustix::fs::{FlockOperation, OFlags};
use std::{
    collections::BTreeMap,
    fs::{self, File},
    os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

#[path = "derive_cache_publish.rs"]
mod publication;

const PROFILE: &str = "gomad.wasi-cooperative/v1";

#[cfg(test)]
#[path = "derive_cache_tests.rs"]
mod tests;

struct Candidate {
    installed: Installation,
    key: String,
}

fn runtime_inputs(root: &Path) -> Result<BTreeMap<PathBuf, String>, String> {
    let mut pending = vec![root.to_path_buf()];
    let mut inputs = BTreeMap::new();
    let mut entries = 0_u64;
    let mut total = 0_u64;
    while let Some(path) = pending.pop() {
        entries += 1;
        if entries > 100_000 {
            return Err("runtime inventory exceeds entry capacity".into());
        }
        let metadata = fs::symlink_metadata(&path).map_err(|e| e.to_string())?;
        let relative = path
            .strip_prefix(root)
            .map_err(|e| e.to_string())?
            .to_path_buf();
        if metadata.is_dir() {
            inputs.insert(relative, "directory".into());
            for child in fs::read_dir(&path).map_err(|e| e.to_string())? {
                pending.push(child.map_err(|e| e.to_string())?.path());
                if pending.len() as u64 + entries > 100_000 {
                    return Err("runtime inventory exceeds entry capacity".into());
                }
            }
        } else if metadata.is_file() {
            let raw = manifest::read(&path, 512 << 20)?;
            total += raw.len() as u64;
            if total > 512 << 20 {
                return Err("runtime inventory exceeds byte capacity".into());
            }
            inputs.insert(relative, canonical::digest(&raw));
        } else {
            return Err("runtime inventory refuses symlinks and special files".into());
        }
    }
    Ok(inputs)
}

fn candidate(
    compiler: &Path,
    repo: &Repo,
    package: &str,
    root: &Path,
    pin: &impl Fn(&Installation) -> Result<BTreeMap<PathBuf, String>, String>,
) -> Result<Candidate, String> {
    let package = crate::config::repo::relative(package)?;
    crate::config::repo::contained(&repo.module_root, &package)?;
    let config = config::build(
        PROFILE,
        None,
        (repo.derive == Derivation::Temporal)
            .then_some((repo.module_root.as_path(), package.as_str())),
    )?;
    let configuration_sha256 = canonical::digest(&canonical::encode(&config.original)?);
    let compiler = std::path::absolute(compiler).map_err(|e| e.to_string())?;
    let compiler_sha256 = canonical::digest(&manifest::read(&compiler, 512 << 20)?);
    manifest::executable(&compiler, &compiler_sha256)?;
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    let executable_sha256 = canonical::digest(&manifest::read(&executable, 512 << 20)?);
    let runtime = runtime_inputs(&repo.runtime_root)?;
    let overlay_module_root = fs::canonicalize(Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."))
        .map_err(|e| e.to_string())?;
    let temporal = repo.derive == Derivation::Temporal;
    let mut installed = Installation {
        schema: manifest::SCHEMA.into(),
        admission: None,
        compiler_path: compiler,
        compiler_sha256,
        go_version: "go1.27.1".into(),
        runtime_root: repo.runtime_root.clone(),
        overlay_module_root: Some(overlay_module_root),
        cache_root: root.join("preparation"),
        profile: PROFILE.into(),
        fuel: if temporal { 1 << 50 } else { 4_000_000_000 },
        memory_bytes: if temporal { 4 << 30 } else { 256 << 20 },
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        model: config.model,
        namespace: config.namespace,
        storage: config.storage,
        network: config.network,
        build_tags: config.build_tags,
        excluded_build_tags: Vec::new(),
    };
    manifest::validate(&installed)?;
    let compiler_inputs = pin(&installed)?;
    let inputs_sha256 = canonical::digest(&canonical::encode(&compiler_inputs)?);
    let key = canonical::digest(&canonical::encode(&serde_json::json!({
        "schema": "gomad.derived-installation-key/v1",
        "compiler": installed.compiler_path, "compiler_sha256": installed.compiler_sha256,
        "runtime_root": repo.runtime_root, "runtime_inputs": runtime,
        "module_root": repo.module_root, "derivation": repo.derive, "package": package,
        "configuration_sha256": configuration_sha256, "executable_sha256": executable_sha256,
        "host_sha256": installed.host_sha256, "helper_sha256": installed.helper_sha256,
        "runtime_implementation_sha256": installed.runtime_implementation_sha256,
        "overlay_module_root": installed.overlay_module_root, "inputs_sha256": inputs_sha256,
    }))?);
    let admission = Admission {
        compiler_inputs,
        inputs_sha256,
        guest_configuration: config.original,
        configuration_sha256,
        host_executable_sha256: executable_sha256,
        explicit_helper: None,
        environment: config.environment,
        arguments: config.arguments,
        compile_cache_root: None,
    };
    installed.admission = Some(admission);
    Ok(Candidate { installed, key })
}

fn same_directory(path: &Path, file: &File) -> Result<(), String> {
    let named = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    let opened = file.metadata().map_err(|e| e.to_string())?;
    if !named.is_dir()
        || named.dev() != opened.dev()
        || named.ino() != opened.ino()
        || named.uid() != rustix::process::geteuid().as_raw()
        || named.permissions().mode() & 0o777 != 0o700
    {
        return Err("derived installation cache is not a private unchanged directory".into());
    }
    Ok(())
}

fn directory(cache_root: &Path) -> Result<(PathBuf, File), String> {
    if !cache_root.is_absolute() {
        return Err("derived installation cache root must be absolute".into());
    }
    fs::create_dir_all(cache_root).map_err(|e| e.to_string())?;
    let root = fs::canonicalize(cache_root)
        .map_err(|e| e.to_string())?
        .join("derived-installations");
    match fs::DirBuilder::new().mode(0o700).create(&root) {
        Ok(()) => {}
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(error) => return Err(error.to_string()),
    }
    let file = fs::OpenOptions::new()
        .read(true)
        .custom_flags(
            (OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC)
                .bits()
                .cast_signed(),
        )
        .open(&root)
        .map_err(|e| e.to_string())?;
    same_directory(&root, &file)?;
    Ok((root, file))
}

fn before_deadline(deadline: Instant) -> Result<(), String> {
    if Instant::now() >= deadline {
        return Err("derived installation deadline exceeded".into());
    }
    Ok(())
}

fn lock(directory: &File, key: &str, deadline: Instant) -> Result<File, String> {
    let file = File::from(
        rustix::fs::openat(
            directory,
            format!("{key}.lock"),
            OFlags::RDWR | OFlags::CREATE | OFlags::NOFOLLOW | OFlags::CLOEXEC | OFlags::NONBLOCK,
            rustix::fs::Mode::from_raw_mode(0o600),
        )
        .map_err(|e| e.to_string())?,
    );
    loop {
        before_deadline(deadline)?;
        match rustix::fs::flock(&file, FlockOperation::NonBlockingLockExclusive) {
            Ok(()) => break,
            Err(rustix::io::Errno::WOULDBLOCK) => {
                std::thread::sleep(
                    Duration::from_millis(1)
                        .min(deadline.saturating_duration_since(Instant::now())),
                );
            }
            Err(error) => return Err(error.to_string()),
        }
    }
    let metadata = file.metadata().map_err(|e| e.to_string())?;
    let named = rustix::fs::statat(
        directory,
        format!("{key}.lock"),
        rustix::fs::AtFlags::SYMLINK_NOFOLLOW,
    )
    .map_err(|e| e.to_string())?;
    if !metadata.is_file()
        || metadata.uid() != rustix::process::geteuid().as_raw()
        || metadata.permissions().mode() & 0o777 != 0o600
        || metadata.nlink() != 1
        || metadata.dev() != named.st_dev
        || metadata.ino() != named.st_ino
    {
        return Err("unsafe derived installation cache lock".into());
    }
    Ok(file)
}

fn admit(path: &Path, candidate: &Candidate) -> Result<(), String> {
    let selected = manifest::resolve(Some(path))?.manifest;
    manifest::admit_build(&selected)?;
    if canonical::encode(&selected)? != canonical::encode(&candidate.installed)? {
        return Err("cached installation does not match current derivation and complete toolchain inventory".into());
    }
    Ok(())
}

pub(crate) fn derive(
    compiler: &Path,
    repo: &Repo,
    concrete_package: &str,
    cache_root: &Path,
    deadline: Instant,
) -> Result<PathBuf, String> {
    derive_with_pin(
        compiler,
        repo,
        concrete_package,
        cache_root,
        deadline,
        &|installed| toolchain::pin_before(installed, deadline),
    )
}

fn derive_with_pin(
    compiler: &Path,
    repo: &Repo,
    concrete_package: &str,
    cache_root: &Path,
    deadline: Instant,
    pin: &impl Fn(&Installation) -> Result<BTreeMap<PathBuf, String>, String>,
) -> Result<PathBuf, String> {
    before_deadline(deadline)?;
    let (root, directory) = directory(cache_root)?;
    // Schema capture and identity derivation must precede lookup, including cache hits.
    let candidate = candidate(compiler, repo, concrete_package, &root, pin)?;
    before_deadline(deadline)?;
    let _lock = lock(&directory, &candidate.key, deadline)?;
    before_deadline(deadline)?;
    same_directory(&root, &directory)?;
    if candidate.key != self::candidate(compiler, repo, concrete_package, &root, pin)?.key {
        return Err("installation inputs changed while awaiting cache admission".into());
    }
    let path = root.join(format!("{}.json", candidate.key));
    before_deadline(deadline)?;
    publication::discard_stage(&directory, &format!(".{}.staged", candidate.key))?;
    match fs::symlink_metadata(&path) {
        Ok(_) => admit(&path, &candidate)?,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            same_directory(&root, &directory)?;
            publication::publish(&root, &directory, &path, &candidate)?;
        }
        Err(error) => return Err(error.to_string()),
    }
    same_directory(&root, &directory)?;
    before_deadline(deadline)?;
    if path.file_stem().and_then(|value| value.to_str())
        != Some(
            self::candidate(compiler, repo, concrete_package, &root, pin)?
                .key
                .as_str(),
        )
    {
        return Err("installation inputs changed during cache admission".into());
    }
    before_deadline(deadline)?;
    Ok(path)
}
