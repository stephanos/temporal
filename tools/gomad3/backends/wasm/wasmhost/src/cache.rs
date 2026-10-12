use crate::{
    engine::compile,
    metrics::{self, Metrics},
    protocol::Outcome,
};
use sha2::{Digest, Sha256};
use std::{
    fs::File,
    hash::{Hash, Hasher},
    io::{Read, Write},
    time::Instant,
};
use wasmtime::{Engine, Module};

const MAGIC: &[u8; 8] = b"GOMADC02";
const HEADER: usize = 52;
const MAX_ARTIFACT: usize = 1024 * 1024 * 1024;
const MAX_MANIFEST: usize = 16 * 1024;

fn invalid(message: impl std::fmt::Display) -> Outcome {
    Outcome::new("invalid", format!("compile cache: {message}"))
}
fn io(error: std::io::Error) -> Outcome {
    Outcome::new("infrastructure", format!("compile cache: {error}"))
}
fn capacity() -> Outcome {
    Outcome::new("capacity", "compile cache artifact capacity")
}

struct Compatibility(Sha256);
impl Hasher for Compatibility {
    fn write(&mut self, bytes: &[u8]) {
        self.0.update(bytes);
    }
    fn finish(&self) -> u64 {
        u64::from_le_bytes(self.0.clone().finalize()[..8].try_into().unwrap())
    }
}
fn identity(engine: &Engine, module_sha256: &str) -> Result<Vec<u8>, Outcome> {
    #[cfg(feature = "diagnostic-profile")]
    let _phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::CacheIdentity);
    #[cfg(target_os = "linux")]
    let mut helper = File::open("/proc/self/exe").map_err(io)?;
    #[cfg(target_os = "macos")]
    let mut helper = {
        use std::os::unix::fs::OpenOptionsExt;
        std::fs::OpenOptions::new()
            .read(true)
            .custom_flags(0x100)
            .open(std::env::current_exe().map_err(io)?)
            .map_err(io)?
    };
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    let mut helper = File::open(std::env::current_exe().map_err(io)?).map_err(io)?;
    let before = helper.metadata().map_err(io)?;
    let mut hash = Sha256::new();
    let mut buffer = [0u8; 65536];
    loop {
        let size = helper.read(&mut buffer).map_err(io)?;
        if size == 0 {
            break;
        }
        hash.update(&buffer[..size]);
    }
    if !same_file(&before, &helper.metadata().map_err(io)?) {
        return Err(invalid("helper executable changed during identity read"));
    }
    let mut compatibility = Compatibility(Sha256::new());
    engine
        .precompile_compatibility_hash()
        .hash(&mut compatibility);
    serde_json::to_vec(&serde_json::json!({
        "version":2, "module_sha256":module_sha256, "helper_sha256":format!("{:x}", hash.finalize()),
        "engine":"wasmtime-47.0.3", "configuration":"cranelift-fuel-nan-canonical-v1",
        "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
        "engine_compatibility_sha256":format!("{:x}", compatibility.0.finalize())
    })).map_err(invalid)
}

#[cfg(unix)]
fn same_file(a: &std::fs::Metadata, b: &std::fs::Metadata) -> bool {
    use std::os::unix::fs::MetadataExt;
    a.dev() == b.dev()
        && a.ino() == b.ino()
        && a.len() == b.len()
        && a.mode() == b.mode()
        && a.uid() == b.uid()
        && a.nlink() == b.nlink()
        && a.mtime() == b.mtime()
        && a.mtime_nsec() == b.mtime_nsec()
        && a.ctime() == b.ctime()
        && a.ctime_nsec() == b.ctime_nsec()
}
#[cfg(not(unix))]
fn same_file(a: &std::fs::Metadata, b: &std::fs::Metadata) -> bool {
    a.len() == b.len() && a.modified().ok() == b.modified().ok()
}

#[cfg(all(
    target_pointer_width = "64",
    any(target_arch = "x86_64", target_arch = "aarch64"),
    any(target_os = "linux", target_os = "macos")
))]
mod private {
    use super::*;
    use std::{
        ffi::CString,
        os::{
            fd::{AsRawFd, FromRawFd},
            unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt},
        },
    };
    unsafe extern "C" {
        fn geteuid() -> u32;
        fn openat(dir: i32, path: *const std::ffi::c_char, flags: i32, mode: u32) -> i32;
        fn renameat(
            from_dir: i32,
            from: *const std::ffi::c_char,
            to_dir: i32,
            to: *const std::ffi::c_char,
        ) -> i32;
        fn unlinkat(dir: i32, path: *const std::ffi::c_char, flags: i32) -> i32;
    }
    #[cfg(all(target_os = "linux", target_arch = "x86_64"))]
    const NOFOLLOW: i32 = 0x20000;
    #[cfg(all(target_os = "linux", target_arch = "aarch64"))]
    const NOFOLLOW: i32 = 0x8000;
    #[cfg(target_os = "macos")]
    const NOFOLLOW: i32 = 0x100;
    #[cfg(all(target_os = "linux", target_arch = "x86_64"))]
    const DIRECTORY: i32 = 0x10000;
    #[cfg(all(target_os = "linux", target_arch = "aarch64"))]
    const DIRECTORY: i32 = 0x4000;
    #[cfg(target_os = "macos")]
    const DIRECTORY: i32 = 0x100000;
    #[cfg(target_os = "linux")]
    const CREATE_EXCLUSIVE: i32 = 0x40 | 0x80;
    #[cfg(target_os = "macos")]
    const CREATE_EXCLUSIVE: i32 = 0x200 | 0x800;
    #[cfg(target_os = "linux")]
    const CLOEXEC: i32 = 0x80000;
    #[cfg(target_os = "macos")]
    const CLOEXEC: i32 = 0x1000000;
    #[cfg(target_os = "linux")]
    const NONBLOCK: i32 = 0x800;
    #[cfg(target_os = "macos")]
    const NONBLOCK: i32 = 4;

    pub struct Directory(File);
    impl Directory {
        pub fn open(path: &str) -> Result<Self, Outcome> {
            match std::fs::symlink_metadata(path) {
                Ok(metadata) if metadata.file_type().is_symlink() => {
                    return Err(invalid("symlink directory"));
                }
                Ok(_) => (),
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                    std::fs::DirBuilder::new()
                        .recursive(true)
                        .mode(0o700)
                        .create(path)
                        .map_err(io)?;
                }
                Err(error) => return Err(io(error)),
            }
            let file = std::fs::OpenOptions::new()
                .read(true)
                .custom_flags(NOFOLLOW | DIRECTORY)
                .open(path)
                .map_err(invalid)?;
            let metadata = file.metadata().map_err(io)?;
            if !metadata.is_dir()
                || metadata.uid() != unsafe { geteuid() }
                || metadata.mode() & 0o7777 != 0o700
            {
                return Err(invalid("directory must be owned and mode 0700"));
            }
            Ok(Self(file))
        }
        pub fn open_entry(&self, name: &str, create: bool) -> Result<File, std::io::Error> {
            let name = CString::new(name).unwrap();
            let flags =
                NOFOLLOW | CLOEXEC | NONBLOCK | if create { 2 | CREATE_EXCLUSIVE } else { 2 };
            let fd = unsafe { openat(self.0.as_raw_fd(), name.as_ptr(), flags, 0o600) };
            if fd < 0 {
                return Err(std::io::Error::last_os_error());
            }
            Ok(unsafe { File::from_raw_fd(fd) })
        }
        pub fn publish(&self, staging: &str, target: &str) -> Result<(), Outcome> {
            let from = CString::new(staging).unwrap();
            let to = CString::new(target).unwrap();
            if unsafe {
                renameat(
                    self.0.as_raw_fd(),
                    from.as_ptr(),
                    self.0.as_raw_fd(),
                    to.as_ptr(),
                )
            } != 0
            {
                return Err(io(std::io::Error::last_os_error()));
            }
            self.0.sync_all().map_err(io)
        }
        pub fn remove(&self, name: &str) {
            let name = CString::new(name).unwrap();
            unsafe {
                unlinkat(self.0.as_raw_fd(), name.as_ptr(), 0);
            }
        }
    }
    pub fn validate(file: &File) -> Result<std::fs::Metadata, Outcome> {
        let metadata = file.metadata().map_err(io)?;
        if !metadata.is_file()
            || metadata.uid() != unsafe { geteuid() }
            || metadata.nlink() != 1
            || metadata.mode() & 0o7777 != 0o600
        {
            return Err(invalid(
                "entry must be owned, single-link, regular and mode 0600",
            ));
        }
        Ok(metadata)
    }
}

#[cfg(all(
    target_pointer_width = "64",
    any(target_arch = "x86_64", target_arch = "aarch64"),
    any(target_os = "linux", target_os = "macos")
))]
pub fn module(
    engine: &Engine,
    bytes: &[u8],
    directory: &str,
    module_sha256: &str,
    metrics: &mut Metrics,
) -> Result<Module, Outcome> {
    let phase = Instant::now();
    let prepared = (|| {
        let manifest = identity(engine, module_sha256)?;
        let name = format!("{:x}.gomad-cache-v2", Sha256::digest(&manifest));
        let directory = private::Directory::open(directory)?;
        let lock_name = format!("{name}.lock");
        let lock = match directory.open_entry(&lock_name, true) {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                directory.open_entry(&lock_name, false).map_err(invalid)?
            }
            Err(error) => return Err(invalid(error)),
        };
        private::validate(&lock)?;
        lock.lock().map_err(io)?;
        private::validate(&lock)?;
        let loaded = match directory.open_entry(&name, false) {
            Ok(mut file) => {
                #[cfg(feature = "diagnostic-profile")]
                let diagnostic_phase =
                    crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::CacheRead);
                let before = private::validate(&file)?;
                let size = usize::try_from(before.len()).map_err(|_| capacity())?;
                if size > HEADER + MAX_MANIFEST + MAX_ARTIFACT {
                    return Err(capacity());
                }
                let mut envelope = Vec::new();
                envelope.try_reserve_exact(size).map_err(|_| capacity())?;
                envelope.resize(size, 0);
                file.read_exact(&mut envelope).map_err(invalid)?;
                let mut trailing = [0u8; 1];
                if file.read(&mut trailing).map_err(io)? != 0
                    || !same_file(&before, &file.metadata().map_err(io)?)
                {
                    return Err(invalid("entry changed during read"));
                }
                #[cfg(feature = "diagnostic-profile")]
                drop(diagnostic_phase);
                #[cfg(feature = "diagnostic-profile")]
                let diagnostic_phase =
                    crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::EnvelopeVerify);
                if envelope.len() < HEADER || &envelope[..8] != MAGIC {
                    return Err(invalid("envelope magic"));
                }
                let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
                let artifact_len = u64::from_le_bytes(envelope[12..20].try_into().unwrap());
                if manifest_len > MAX_MANIFEST || artifact_len > MAX_ARTIFACT as u64 {
                    return Err(capacity());
                }
                if HEADER + manifest_len + artifact_len as usize != size
                    || envelope[HEADER..HEADER + manifest_len] != manifest
                {
                    return Err(invalid("envelope identity or length"));
                }
                let artifact = &envelope[HEADER + manifest_len..];
                if Sha256::digest(artifact)[..] != envelope[20..52] {
                    return Err(invalid("artifact checksum"));
                }
                #[cfg(feature = "diagnostic-profile")]
                drop(diagnostic_phase);
                #[cfg(feature = "diagnostic-profile")]
                let _diagnostic_phase =
                    crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Deserialize);
                // SAFETY: owned private host cache is outside guest access. Identity,
                // file trust and checksum were validated; no file-backed mapping is used.
                Some(
                    unsafe { Module::deserialize(engine, artifact) }.map_err(|error| {
                        if error.is::<wasmtime::OutOfMemory>() {
                            capacity()
                        } else {
                            invalid(error)
                        }
                    })?,
                )
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
            Err(error) => return Err(invalid(error)),
        };
        Ok((directory, lock, name, manifest, loaded))
    })();
    metrics.cache_load_nanos = metrics::elapsed(phase);
    let (directory, _lock, name, manifest, loaded) = prepared?;
    if let Some(module) = loaded {
        metrics.cache_hit = true;
        return Ok(module);
    }
    let module = compile(engine, bytes, metrics)?;
    #[cfg(feature = "diagnostic-profile")]
    let _diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::CacheWrite);
    let phase = Instant::now();
    let published = (|| {
        let artifact = module
            .serialize()
            .map_err(|error| Outcome::new("infrastructure", format!("{error:#}")))?;
        if artifact.len() > MAX_ARTIFACT || manifest.len() > MAX_MANIFEST {
            return Err(capacity());
        }
        let staging = format!("{name}.{}.tmp", std::process::id());
        let mut created = false;
        let result = (|| {
            let mut file = directory.open_entry(&staging, true).map_err(invalid)?;
            created = true;
            private::validate(&file)?;
            file.write_all(MAGIC)
                .and_then(|_| file.write_all(&(manifest.len() as u32).to_le_bytes()))
                .and_then(|_| file.write_all(&(artifact.len() as u64).to_le_bytes()))
                .and_then(|_| file.write_all(&Sha256::digest(&artifact)))
                .and_then(|_| file.write_all(&manifest))
                .and_then(|_| file.write_all(&artifact))
                .and_then(|_| file.sync_all())
                .map_err(io)?;
            directory.publish(&staging, &name)
        })();
        if result.is_err() && created {
            directory.remove(&staging);
        }
        result?;
        Ok(module)
    })();
    metrics.cache_write_nanos = metrics::elapsed(phase);
    published
}

#[cfg(not(all(
    target_pointer_width = "64",
    any(target_arch = "x86_64", target_arch = "aarch64"),
    any(target_os = "linux", target_os = "macos")
)))]
pub fn module(_: &Engine, _: &[u8], _: &str, _: &str, _: &mut Metrics) -> Result<Module, Outcome> {
    Err(Outcome::new(
        "unsupported",
        "private compile cache requires Linux or macOS",
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[cfg(all(
        feature = "diagnostic-profile",
        target_pointer_width = "64",
        any(target_arch = "x86_64", target_arch = "aarch64"),
        any(target_os = "linux", target_os = "macos")
    ))]
    #[test]
    fn diagnostic_cache_phases_separate_read_verify_and_deserialize_failures() {
        use std::os::unix::fs::DirBuilderExt;
        let directory =
            std::env::temp_dir().join(format!("gomad-profile-cache-{}", std::process::id()));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&directory)
            .unwrap();
        let engine = Engine::default();
        let bytes = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))";
        let digest = format!("{:x}", Sha256::digest(bytes));
        let mut metrics = Metrics::default();
        {
            let _session = crate::diagnostic::Session::start();
            assert!(
                module(
                    &engine,
                    bytes,
                    directory.to_str().unwrap(),
                    &digest,
                    &mut metrics
                )
                .is_ok()
            );
            let report = crate::diagnostic::snapshot();
            assert_eq!(report["phases"]["cache_identity"]["count"], 1);
            assert_eq!(report["phases"]["compile"]["count"], 1);
            assert_eq!(report["phases"]["cache_write"]["count"], 1);
            assert_eq!(report["phases"]["deserialize"]["count"], 0);
        }
        {
            let _session = crate::diagnostic::Session::start();
            assert!(
                module(
                    &engine,
                    bytes,
                    directory.to_str().unwrap(),
                    &digest,
                    &mut metrics
                )
                .is_ok()
            );
            assert!(metrics.cache_hit);
            let report = crate::diagnostic::snapshot();
            for phase in [
                "cache_identity",
                "cache_read",
                "envelope_verify",
                "deserialize",
            ] {
                assert_eq!(report["phases"][phase]["count"], 1);
                assert!(report["phases"][phase]["wall_nanos"].as_u64().unwrap() > 0);
            }
            assert_eq!(report["phases"]["compile"]["count"], 0);
            assert_eq!(report["phases"]["cache_write"]["count"], 0);
        }
        let entry = std::fs::read_dir(&directory)
            .unwrap()
            .map(|e| e.unwrap().path())
            .find(|p| p.extension().is_some_and(|e| e == "gomad-cache-v2"))
            .unwrap();
        let mut envelope = std::fs::read(&entry).unwrap();
        envelope[HEADER - 1] ^= 1;
        std::fs::write(&entry, envelope).unwrap();
        {
            let _session = crate::diagnostic::Session::start();
            let outcome = module(
                &engine,
                bytes,
                directory.to_str().unwrap(),
                &digest,
                &mut metrics,
            )
            .err()
            .unwrap();
            assert_eq!(outcome.termination, "invalid");
            assert_eq!(outcome.message, "compile cache: artifact checksum");
            let report = crate::diagnostic::snapshot();
            assert_eq!(report["phases"]["cache_read"]["count"], 1);
            assert_eq!(report["phases"]["envelope_verify"]["count"], 1);
            assert_eq!(report["phases"]["deserialize"]["count"], 0);
        }
        std::fs::remove_dir_all(directory).unwrap();
    }
    #[test]
    fn cache_identity_binds_helper_source_platform_and_engine_settings() {
        let engine = Engine::default();
        let identity = |engine: &Engine, module: &str| {
            super::identity(engine, module).unwrap_or_else(|outcome| panic!("{}", outcome.message))
        };
        let identity_a = identity(&engine, &"a".repeat(64));
        assert_eq!(identity_a, identity(&engine, &"a".repeat(64)));
        assert_ne!(identity_a, identity(&engine, &"b".repeat(64)));
        let mut config = wasmtime::Config::new();
        config.consume_fuel(true);
        let different = Engine::new(&config).unwrap();
        assert_ne!(identity_a, identity(&different, &"a".repeat(64)));
        let manifest: serde_json::Value = serde_json::from_slice(&identity_a).unwrap();
        assert_eq!(manifest["host_arch"], std::env::consts::ARCH);
        assert_eq!(manifest["host_os"], std::env::consts::OS);
        assert_eq!(manifest["helper_sha256"].as_str().unwrap().len(), 64);
        assert_eq!(
            manifest["engine_compatibility_sha256"]
                .as_str()
                .unwrap()
                .len(),
            64
        );
    }
}
