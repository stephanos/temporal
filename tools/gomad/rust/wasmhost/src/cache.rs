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

const MAGIC: &[u8; 8] = b"GOMADC03";
const HEADER: usize = 24;
const MAX_ARTIFACT: usize = 1024 * 1024 * 1024;
const MAX_MANIFEST: usize = 16 * 1024;

// Every cache failure is a host failure: the cache is a trusted same-UID host
// store outside guest semantics, so it never yields a guest or module verdict.
fn host(message: impl std::fmt::Display) -> Outcome {
    Outcome::new("infrastructure", format!("compile cache: {message}"))
}
fn corrupt(entry: &std::path::Path, message: impl std::fmt::Display) -> Outcome {
    Outcome::new(
        "infrastructure",
        format!(
            "compile cache: corrupt entry {}: {message}",
            entry.display()
        ),
    )
}
/// Running out of space or quota while publishing only loses the cache entry.
fn storage_full(error: &std::io::Error) -> bool {
    matches!(
        error.kind(),
        std::io::ErrorKind::StorageFull | std::io::ErrorKind::QuotaExceeded
    )
}
/// A collision-proof staging name: process ID, a per-process counter and
/// fresh random bytes; the caller creates it with `O_EXCL`.
fn staging_name(name: &str) -> String {
    use std::sync::atomic::{AtomicU64, Ordering};
    static COUNTER: AtomicU64 = AtomicU64::new(0);
    let mut random = [0u8; 8];
    if File::open("/dev/urandom")
        .and_then(|mut file| file.read_exact(&mut random))
        .is_err()
    {
        #[expect(
            clippy::cast_possible_truncation,
            reason = "Only the low 64 timestamp bits seed the fallback staging suffix."
        )]
        let nanos = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_or(0, |elapsed| elapsed.as_nanos() as u64);
        random = nanos.to_le_bytes();
    }
    format!(
        "{name}.{}.{}.{:016x}.tmp",
        std::process::id(),
        COUNTER.fetch_add(1, Ordering::Relaxed),
        u64::from_le_bytes(random)
    )
}
enum Publication {
    Io(std::io::Error),
    Outcome(Outcome),
}
impl From<Outcome> for Publication {
    fn from(outcome: Outcome) -> Self {
        Self::Outcome(outcome)
    }
}
#[expect(
    clippy::needless_pass_by_value,
    reason = "This adapter consumes errors passed by Result::map_err."
)]
fn io(error: std::io::Error) -> Outcome {
    Outcome::new("infrastructure", format!("compile cache: {error}"))
}
fn capacity() -> Outcome {
    Outcome::new("capacity", "compile cache artifact capacity")
}

#[cfg(test)]
thread_local! {
    /// Test hook: fails staged publication writes with this error kind.
    pub(crate) static PUBLICATION_FAULT: std::cell::Cell<Option<std::io::ErrorKind>> =
        const { std::cell::Cell::new(None) };
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
    #[expect(
        clippy::large_stack_arrays,
        reason = "The executable hash uses one fixed, bounded 64 KiB streaming buffer."
    )]
    let mut buffer = [0u8; 65536];
    loop {
        let size = helper.read(&mut buffer).map_err(io)?;
        if size == 0 {
            break;
        }
        hash.update(&buffer[..size]);
    }
    if !same_file(&before, &helper.metadata().map_err(io)?) {
        return Err(host("helper executable changed during identity read"));
    }
    let mut compatibility = Compatibility(Sha256::new());
    engine
        .precompile_compatibility_hash()
        .hash(&mut compatibility);
    serde_json::to_vec(&serde_json::json!({
        "version":3, "module_sha256":module_sha256, "helper_sha256":format!("{:x}", hash.finalize()),
        "engine":"wasmtime-47.0.3", "configuration":crate::engine::CONFIGURATION_REVISION,
        "host_os":std::env::consts::OS, "host_arch":std::env::consts::ARCH,
        "engine_compatibility_sha256":format!("{:x}", compatibility.0.finalize())
    })).map_err(host)
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
    use super::{File, Outcome, corrupt, host, io};
    use rustix::{
        fs::{AtFlags, Mode, OFlags, open, openat, renameat, unlinkat},
        process::geteuid,
    };
    use std::os::unix::fs::{DirBuilderExt, MetadataExt};

    pub struct Directory(File);
    impl Directory {
        pub fn open(path: &str) -> Result<Self, Outcome> {
            match std::fs::symlink_metadata(path) {
                Ok(metadata) if metadata.file_type().is_symlink() => {
                    return Err(host("symlink directory"));
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
            let file = File::from(
                open(
                    path,
                    OFlags::RDONLY | OFlags::NOFOLLOW | OFlags::DIRECTORY | OFlags::CLOEXEC,
                    Mode::empty(),
                )
                .map_err(std::io::Error::from)
                .map_err(host)?,
            );
            let metadata = file.metadata().map_err(io)?;
            if !metadata.is_dir()
                || metadata.uid() != geteuid().as_raw()
                || metadata.mode() & 0o7777 != 0o700
            {
                return Err(host("directory must be owned and mode 0700"));
            }
            Ok(Self(file))
        }
        pub fn open_entry(&self, name: &str, create: bool) -> Result<File, std::io::Error> {
            let mut flags = OFlags::RDWR | OFlags::NOFOLLOW | OFlags::CLOEXEC | OFlags::NONBLOCK;
            if create {
                flags |= OFlags::CREATE | OFlags::EXCL;
            }
            openat(&self.0, name, flags, Mode::RUSR | Mode::WUSR)
                .map(File::from)
                .map_err(std::io::Error::from)
        }
        pub fn publish(&self, staging: &str, target: &str) -> std::io::Result<()> {
            renameat(&self.0, staging, &self.0, target).map_err(std::io::Error::from)?;
            self.0.sync_all()
        }
        pub fn remove(&self, name: &str) {
            let _ = unlinkat(&self.0, name, AtFlags::empty());
        }
    }
    pub fn validate(file: &File, entry: &std::path::Path) -> Result<std::fs::Metadata, Outcome> {
        let metadata = file.metadata().map_err(io)?;
        if !metadata.is_file()
            || metadata.uid() != geteuid().as_raw()
            || metadata.nlink() != 1
            || metadata.mode() & 0o7777 != 0o600
        {
            return Err(corrupt(
                entry,
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
#[expect(
    clippy::too_many_lines,
    reason = "The ordered cache transaction keeps trust checks and publication cleanup together."
)]
pub fn module(
    engine: &Engine,
    load: impl FnOnce() -> Result<Vec<u8>, Outcome>,
    directory: &str,
    module_sha256: &str,
    metrics: &mut Metrics,
) -> Result<Module, Outcome> {
    let phase = Instant::now();
    let prepared = (|| {
        let manifest = identity(engine, module_sha256)?;
        let name = format!("{:x}.gomad-cache-v3", Sha256::digest(&manifest));
        let root = std::path::Path::new(directory).to_owned();
        let entry = root.join(&name);
        let directory = private::Directory::open(directory)?;
        let lock_name = format!("{name}.lock");
        let lock = match directory.open_entry(&lock_name, true) {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                directory.open_entry(&lock_name, false).map_err(host)?
            }
            Err(error) => return Err(host(error)),
        };
        let lock_path = root.join(&lock_name);
        private::validate(&lock, &lock_path)?;
        lock.lock().map_err(io)?;
        private::validate(&lock, &lock_path)?;
        let loaded = match directory.open_entry(&name, false) {
            Ok(mut file) => {
                #[cfg(feature = "diagnostic-profile")]
                let diagnostic_phase =
                    crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::CacheRead);
                let before = private::validate(&file, &entry)?;
                let size = usize::try_from(before.len())
                    .map_err(|_| corrupt(&entry, "entry length does not fit host"))?;
                if size > HEADER + MAX_MANIFEST + MAX_ARTIFACT {
                    return Err(corrupt(&entry, "entry length exceeds limit"));
                }
                let mut envelope = Vec::new();
                envelope.try_reserve_exact(size).map_err(|_| capacity())?;
                envelope.resize(size, 0);
                file.read_exact(&mut envelope)
                    .map_err(|error| corrupt(&entry, error))?;
                let mut trailing = [0u8; 1];
                if file.read(&mut trailing).map_err(io)? != 0
                    || !same_file(&before, &file.metadata().map_err(io)?)
                {
                    return Err(corrupt(&entry, "entry changed during read"));
                }
                #[cfg(feature = "diagnostic-profile")]
                drop(diagnostic_phase);
                #[cfg(feature = "diagnostic-profile")]
                let diagnostic_phase =
                    crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::EnvelopeVerify);
                if envelope.len() < HEADER || &envelope[..8] != MAGIC {
                    return Err(corrupt(&entry, "envelope magic"));
                }
                let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
                let artifact_len = u64::from_le_bytes(envelope[12..20].try_into().unwrap());
                if manifest_len > MAX_MANIFEST || artifact_len > MAX_ARTIFACT as u64 {
                    return Err(corrupt(&entry, "envelope length exceeds limit"));
                }
                #[expect(
                    clippy::cast_possible_truncation,
                    reason = "This implementation is restricted to 64-bit hosts; artifact_len is also bounded by MAX_ARTIFACT above."
                )]
                if HEADER + manifest_len + artifact_len as usize != size
                    || envelope[HEADER..HEADER + manifest_len] != manifest
                {
                    return Err(corrupt(&entry, "envelope identity or length"));
                }
                let artifact = &envelope[HEADER + manifest_len..];
                if crc32fast::hash(artifact)
                    != u32::from_le_bytes(envelope[20..24].try_into().unwrap())
                {
                    return Err(corrupt(&entry, "artifact checksum"));
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
                            corrupt(&entry, error)
                        }
                    })?,
                )
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
            Err(error) => return Err(corrupt(&entry, error)),
        };
        Ok((root, directory, lock, name, manifest, loaded))
    })();
    metrics.cache_load_nanos = metrics::elapsed(phase);
    let (root, directory, _lock, name, manifest, loaded) = prepared?;
    if let Some(module) = loaded {
        metrics.cache_hit = true;
        return Ok(module);
    }
    let module = compile(engine, &load()?, metrics)?;
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
        // O_EXCL with a fresh name: a stale staging file left by a dead
        // process (including one whose PID was reused) is never fatal.
        let mut attempts = 0;
        let (staging, mut file) = loop {
            let staging = staging_name(&name);
            match directory.open_entry(&staging, true) {
                Ok(file) => break (staging, file),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists && attempts < 8 => {
                    attempts += 1;
                }
                // No space for the entry: the module is compiled, so run uncached.
                Err(error) if storage_full(&error) => return Ok(module),
                Err(error) => return Err(io(error)),
            }
        };
        let manifest_len = u32::try_from(manifest.len()).map_err(|_| capacity())?;
        let result = (|| {
            private::validate(&file, &root.join(&staging))?;
            #[cfg(test)]
            if let Some(kind) = PUBLICATION_FAULT.with(std::cell::Cell::get) {
                return Err(Publication::Io(std::io::Error::from(kind)));
            }
            file.write_all(MAGIC)
                .and_then(|()| file.write_all(&manifest_len.to_le_bytes()))
                .and_then(|()| file.write_all(&(artifact.len() as u64).to_le_bytes()))
                .and_then(|()| file.write_all(&crc32fast::hash(&artifact).to_le_bytes()))
                .and_then(|()| file.write_all(&manifest))
                .and_then(|()| file.write_all(&artifact))
                .and_then(|()| file.sync_all())
                .and_then(|()| directory.publish(&staging, &name))
                .map_err(Publication::Io)
        })();
        if let Err(failure) = result {
            directory.remove(&staging);
            match failure {
                Publication::Io(error) if storage_full(&error) => (),
                Publication::Io(error) => return Err(io(error)),
                Publication::Outcome(outcome) => return Err(outcome),
            }
        }
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
pub fn module(
    _: &Engine,
    _: impl FnOnce() -> Result<Vec<u8>, Outcome>,
    _: &str,
    _: &str,
    _: &mut Metrics,
) -> Result<Module, Outcome> {
    Err(Outcome::new(
        "unsupported",
        "private compile cache requires Linux or macOS",
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    #[ignore = "manual host throughput measurement"]
    fn crc_throughput() {
        let bytes = vec![0xa5; 16 * 1024 * 1024];
        let expected = crc32fast::hash(&bytes);
        let started = Instant::now();
        for _ in 0..64 {
            assert_eq!(crc32fast::hash(std::hint::black_box(&bytes)), expected);
        }
        let seconds = started.elapsed().as_secs_f64();
        let total = bytes.len() as u64 * 64;
        #[expect(
            clippy::cast_precision_loss,
            reason = "Throughput is an approximate diagnostic rate, not a protocol value."
        )]
        let gb_per_second = total as f64 / seconds / 1e9;
        println!(
            "CRC-32: host={}/{} bytes={} seconds={seconds:.6} GB/s={gb_per_second:.3}",
            std::env::consts::OS,
            std::env::consts::ARCH,
            total,
        );
        #[cfg(all(target_os = "linux", target_arch = "aarch64"))]
        assert!(gb_per_second >= 2.0);
    }
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
                    || Ok(bytes.to_vec()),
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
                    || panic!("cache hit read module bytes"),
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
            .find(|p| p.extension().is_some_and(|e| e == "gomad-cache-v3"))
            .unwrap();
        let mut envelope = std::fs::read(&entry).unwrap();
        envelope[HEADER - 1] ^= 1;
        std::fs::write(&entry, envelope).unwrap();
        {
            let _session = crate::diagnostic::Session::start();
            let outcome = module(
                &engine,
                || panic!("corrupt entry read module bytes"),
                directory.to_str().unwrap(),
                &digest,
                &mut metrics,
            )
            .err()
            .unwrap();
            assert_eq!(outcome.termination, "infrastructure");
            assert_eq!(
                outcome.message,
                format!(
                    "compile cache: corrupt entry {}: artifact checksum",
                    entry.display()
                )
            );
            let report = crate::diagnostic::snapshot();
            assert_eq!(report["phases"]["cache_read"]["count"], 1);
            assert_eq!(report["phases"]["envelope_verify"]["count"], 1);
            assert_eq!(report["phases"]["deserialize"]["count"], 0);
        }
        std::fs::remove_dir_all(directory).unwrap();
    }
    #[cfg(all(
        target_pointer_width = "64",
        any(target_arch = "x86_64", target_arch = "aarch64"),
        any(target_os = "linux", target_os = "macos")
    ))]
    mod host_failures {
        use super::*;
        use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt};
        const WAT: &[u8] = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))";
        struct Fixture {
            directory: std::path::PathBuf,
            engine: Engine,
            digest: String,
        }
        impl Fixture {
            fn new(label: &str) -> Self {
                let directory = std::env::temp_dir().join(format!(
                    "gomad-cache-{label}-{}-{:?}",
                    std::process::id(),
                    std::thread::current().id()
                ));
                let _ = std::fs::remove_dir_all(&directory);
                std::fs::DirBuilder::new()
                    .mode(0o700)
                    .create(&directory)
                    .unwrap();
                Self {
                    directory,
                    engine: Engine::default(),
                    digest: format!("{:x}", Sha256::digest(WAT)),
                }
            }
            fn run(&self) -> (Result<Module, Outcome>, Metrics) {
                let mut metrics = Metrics::default();
                let result = module(
                    &self.engine,
                    || Ok(WAT.to_vec()),
                    self.directory.to_str().unwrap(),
                    &self.digest,
                    &mut metrics,
                );
                (result, metrics)
            }
            fn entry(&self) -> std::path::PathBuf {
                let manifest = identity(&self.engine, &self.digest)
                    .unwrap_or_else(|outcome| panic!("{}", outcome.message));
                self.directory
                    .join(format!("{:x}.gomad-cache-v3", Sha256::digest(&manifest)))
            }
        }
        impl Drop for Fixture {
            fn drop(&mut self) {
                let _ = std::fs::set_permissions(
                    &self.directory,
                    std::fs::Permissions::from_mode(0o700),
                );
                let _ = std::fs::remove_dir_all(&self.directory);
            }
        }

        #[test]
        fn corrupt_entry_fails_closed_as_infrastructure_naming_entry() {
            let fixture = Fixture::new("corrupt");
            assert!(fixture.run().0.is_ok());
            let entry = fixture.entry();
            let mut envelope = std::fs::read(&entry).unwrap();
            let last = envelope.len() - 1;
            envelope[last] ^= 1;
            std::fs::write(&entry, &envelope).unwrap();
            let outcome = fixture.run().0.err().unwrap();
            assert_eq!(outcome.termination, "infrastructure");
            assert!(
                outcome.message.contains(entry.to_str().unwrap())
                    && outcome.message.contains("artifact checksum"),
                "{}",
                outcome.message
            );
            // Fail closed: the corrupt entry is neither replaced nor removed.
            assert_eq!(std::fs::read(&entry).unwrap(), envelope);
        }

        #[test]
        fn oversized_entry_lengths_fail_closed_without_loading_or_mutation() {
            use std::io::{Seek, SeekFrom};
            for case in ["file", "manifest", "artifact", "overflow"] {
                let fixture = Fixture::new(case);
                assert!(fixture.run().0.is_ok());
                let entry = fixture.entry();
                let mut envelope = std::fs::read(&entry).unwrap();
                match case {
                    "file" => {
                        let mut file = std::fs::OpenOptions::new()
                            .write(true)
                            .open(&entry)
                            .unwrap();
                        file.set_len((HEADER + MAX_MANIFEST + MAX_ARTIFACT + 1) as u64)
                            .unwrap();
                        file.seek(SeekFrom::End(-1)).unwrap();
                        file.write_all(b"!").unwrap();
                    }
                    "manifest" => {
                        envelope[8..12].copy_from_slice(
                            &u32::try_from(MAX_MANIFEST + 1).unwrap().to_le_bytes(),
                        );
                        std::fs::write(&entry, &envelope).unwrap();
                    }
                    "artifact" => {
                        envelope[12..20]
                            .copy_from_slice(&((MAX_ARTIFACT + 1) as u64).to_le_bytes());
                        std::fs::write(&entry, &envelope).unwrap();
                    }
                    "overflow" => {
                        envelope[8..12].copy_from_slice(&u32::MAX.to_le_bytes());
                        envelope[12..20].copy_from_slice(&u64::MAX.to_le_bytes());
                        std::fs::write(&entry, &envelope).unwrap();
                    }
                    _ => unreachable!(),
                }
                let before = std::fs::metadata(&entry).unwrap();
                let mut metrics = Metrics::default();
                let outcome = module(
                    &fixture.engine,
                    || panic!("corrupt {case} entry loaded source bytes"),
                    fixture.directory.to_str().unwrap(),
                    &fixture.digest,
                    &mut metrics,
                )
                .err()
                .unwrap();
                assert_eq!(outcome.termination, "infrastructure", "{}", outcome.message);
                assert!(
                    outcome
                        .message
                        .contains(&format!("corrupt entry {}", entry.display())),
                    "{}",
                    outcome.message
                );
                assert_eq!(metrics.compile_nanos, 0);
                assert_eq!(metrics.cache_write_nanos, 0);
                assert!(!metrics.cache_hit);
                let mut file = File::open(&entry).unwrap();
                assert!(same_file(&before, &file.metadata().unwrap()));
                let mut retained = vec![0; envelope.len()];
                file.read_exact(&mut retained).unwrap();
                assert_eq!(retained, envelope);
                if case == "file" {
                    file.seek(SeekFrom::End(-1)).unwrap();
                    let mut tail = [0];
                    file.read_exact(&mut tail).unwrap();
                    assert_eq!(tail, *b"!");
                }
            }
        }

        #[test]
        fn v2_envelope_is_ignored_and_preserved() {
            let fixture = Fixture::new("v2");
            assert!(fixture.run().0.is_ok());
            let entry = fixture.entry();
            let envelope = std::fs::read(&entry).unwrap();
            let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
            let artifact = &envelope[HEADER + manifest_len..];
            assert_eq!(&envelope[..8], MAGIC);
            assert_eq!(
                u32::from_le_bytes(envelope[20..24].try_into().unwrap()),
                crc32fast::hash(artifact)
            );
            let mut manifest: serde_json::Value =
                serde_json::from_slice(&envelope[HEADER..HEADER + manifest_len]).unwrap();
            assert_eq!(manifest["version"], 3);
            manifest["version"] = serde_json::json!(2);
            let manifest = serde_json::to_vec(&manifest).unwrap();
            let old = fixture
                .directory
                .join(format!("{:x}.gomad-cache-v2", Sha256::digest(&manifest)));
            let mut previous = b"GOMADC02".to_vec();
            previous.extend_from_slice(&u32::try_from(manifest.len()).unwrap().to_le_bytes());
            previous.extend_from_slice(&(artifact.len() as u64).to_le_bytes());
            previous.extend_from_slice(&Sha256::digest(artifact));
            previous.extend_from_slice(&manifest);
            previous.extend_from_slice(artifact);
            std::fs::write(&old, &previous).unwrap();
            std::fs::remove_file(entry).unwrap();
            let (result, metrics) = fixture.run();
            assert!(result.is_ok());
            assert!(!metrics.cache_hit);
            assert_eq!(std::fs::read(old).unwrap(), previous);
            assert!(fixture.run().1.cache_hit);
        }

        #[test]
        fn bad_directory_mode_is_infrastructure() {
            let fixture = Fixture::new("mode");
            std::fs::set_permissions(&fixture.directory, std::fs::Permissions::from_mode(0o755))
                .unwrap();
            let outcome = fixture.run().0.err().unwrap();
            assert_eq!(outcome.termination, "infrastructure");
            assert!(outcome.message.contains("0700"), "{}", outcome.message);
        }

        #[test]
        fn stale_staging_file_from_reused_pid_is_not_fatal() {
            let fixture = Fixture::new("stale");
            let entry = fixture.entry();
            let stale = format!("{}.{}.tmp", entry.to_str().unwrap(), std::process::id());
            std::fs::OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .open(&stale)
                .unwrap()
                .write_all(b"stale")
                .unwrap();
            let (result, metrics) = fixture.run();
            assert!(result.is_ok(), "{}", result.err().unwrap().message);
            assert!(!metrics.cache_hit);
            assert!(entry.exists());
            let (result, metrics) = fixture.run();
            assert!(result.is_ok() && metrics.cache_hit);
        }

        #[test]
        fn storage_full_during_publication_runs_uncached() {
            for kind in [
                std::io::ErrorKind::StorageFull,
                std::io::ErrorKind::QuotaExceeded,
            ] {
                let fixture = Fixture::new("enospc");
                PUBLICATION_FAULT.with(|fault| fault.set(Some(kind)));
                let (result, metrics) = fixture.run();
                PUBLICATION_FAULT.with(|fault| fault.set(None));
                assert!(result.is_ok(), "{}", result.err().unwrap().message);
                assert!(!metrics.cache_hit);
                assert!(!fixture.entry().exists());
                let leftovers: Vec<_> = std::fs::read_dir(&fixture.directory)
                    .unwrap()
                    .map(|e| e.unwrap().file_name().into_string().unwrap())
                    .filter(|name| {
                        std::path::Path::new(name)
                            .extension()
                            .is_some_and(|extension| extension == "tmp")
                    })
                    .collect();
                assert!(leftovers.is_empty(), "{leftovers:?}");
                // A later run with space publishes normally.
                assert!(fixture.run().0.is_ok());
                assert!(fixture.run().1.cache_hit);
            }
            let fixture = Fixture::new("eio");
            PUBLICATION_FAULT.with(|fault| fault.set(Some(std::io::ErrorKind::Other)));
            let (result, _) = fixture.run();
            PUBLICATION_FAULT.with(|fault| fault.set(None));
            assert_eq!(result.err().unwrap().termination, "infrastructure");
        }
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
        config.native_unwind_info(false);
        let no_unwind = Engine::new(&config).unwrap();
        assert_ne!(
            identity(&different, &"a".repeat(64)),
            identity(&no_unwind, &"a".repeat(64))
        );
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
