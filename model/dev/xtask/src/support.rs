//! Deterministic encoding and atomic publication shared by native tools.
use std::{
    fs::{self, File, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};
use serde::Serialize;
use sha2::{Digest, Sha256};

/// Native identity and sidecar version. Old Python records are deliberately rejected.
pub const SCHEMA: u32 = 2;

/// Hash bytes in-process as lowercase SHA-256.
pub fn hash(bytes: &[u8]) -> String {
    let digest = Sha256::digest(bytes);
    let mut output = String::with_capacity(64);
    for byte in digest {
        use std::fmt::Write;
        let _ = write!(output, "{byte:02x}");
    }
    output
}

/// Validate the digest representation used by the native schema.
pub fn valid_hash(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

/// Encode through serde_json's default sorted Map, including nested objects.
pub fn canonical(value: &impl Serialize) -> crate::Result<Vec<u8>> {
    Ok(serde_json::to_vec(&serde_json::to_value(value)?)?)
}

/// Publish complete bytes by sibling rename, cleaning only our own temporary file.
pub fn publish(path: &Path, bytes: &[u8]) -> crate::Result<()> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let parent = path.parent().filter(|p| !p.as_os_str().is_empty()).unwrap_or(Path::new("."));
    fs::create_dir_all(parent)?;
    let mut name = path.file_name().ok_or_else(|| crate::Error::Invalid("publication needs a filename".into()))?.to_os_string();
    name.push(format!(".{}.{}.tmp", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed)));
    let temporary = parent.join(name);
    let mut file = OpenOptions::new().write(true).create_new(true).open(&temporary)?;
    let result = (|| {
        file.write_all(bytes)?;
        file.sync_all()?;
        drop(file);
        fs::rename(&temporary, path)
    })();
    if let Err(error) = result {
        match fs::remove_file(&temporary) {
            Ok(()) => (),
            Err(cleanup) if cleanup.kind() == std::io::ErrorKind::NotFound => (),
            Err(cleanup) => return Err(crate::Error::Invalid(format!("publication failed: {error}; temporary cleanup failed: {cleanup}"))),
        }
        return Err(error.into());
    }
    Ok(())
}

/// Serialize compilers sharing a sidecar so a failed writer cannot remove a peer's record.
/// The persistent lock inode must remain in place across invocations.
pub struct SidecarLease {
    _file: File,
    path: PathBuf,
}

impl SidecarLease {
    /// Acquire the output's lock and reject non-regular or replaced lock files.
    #[cfg(unix)]
    pub fn acquire(path: &Path) -> crate::Result<Self> {
        use std::os::unix::fs::MetadataExt;
        let parent = path.parent().ok_or_else(|| crate::Error::Invalid("sidecar needs a parent".into()))?;
        fs::create_dir_all(parent)?;
        let mut lock = path.as_os_str().to_os_string();
        lock.push(".lock");
        let lock = PathBuf::from(lock);
        let file = OpenOptions::new().read(true).write(true).create(true).truncate(false).open(&lock)?;
        let verify = || -> crate::Result<()> {
            let actual = fs::symlink_metadata(&lock)?;
            let opened = file.metadata()?;
            if !actual.is_file() || actual.dev() != opened.dev() || actual.ino() != opened.ino() {
                return Err(crate::Error::Invalid("sidecar lock is not a stable regular file".into()));
            }
            Ok(())
        };
        verify()?;
        file.lock()?;
        verify()?;
        Ok(Self { _file: file, path: path.to_owned() })
    }

    /// Remove a failed compiler's record while still holding its output lock.
    pub fn remove(&self) -> crate::Result<()> {
        match fs::remove_file(&self.path) {
            Ok(()) => Ok(()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(error) => Err(error.into()),
        }
    }
}
