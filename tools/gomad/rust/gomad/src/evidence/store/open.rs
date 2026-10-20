use super::{FileEntry, Manifest, canonical, lineage};
use rustix::fs::{self, AtFlags, FileType, Mode, OFlags, Stat};
#[cfg(feature = "determinism-fixture")]
use std::os::unix::fs::OpenOptionsExt as _;
use std::{
    collections::BTreeSet,
    fs::File,
    io::Read,
    path::{Component, Path},
};

pub(crate) fn components(name: &str) -> Result<Vec<&str>, String> {
    if name.is_empty() || name.contains('\\') || name.contains('\0') {
        return Err("unsafe payload path".into());
    }
    let parts: Vec<_> = name.split('/').collect();
    if parts
        .iter()
        .any(|part| part.is_empty() || matches!(*part, "." | ".."))
        || Path::new(name)
            .components()
            .any(|part| !matches!(part, Component::Normal(_)))
    {
        return Err("unsafe payload path".into());
    }
    Ok(parts)
}

pub(crate) fn same(before: &Stat, after: &Stat) -> bool {
    before.st_dev == after.st_dev
        && before.st_ino == after.st_ino
        && before.st_mode == after.st_mode
        && before.st_nlink == after.st_nlink
        && before.st_uid == after.st_uid
        && before.st_size == after.st_size
        && before.st_mtime == after.st_mtime
        && before.st_mtime_nsec == after.st_mtime_nsec
        && before.st_ctime == after.st_ctime
        && before.st_ctime_nsec == after.st_ctime_nsec
}

fn same_opened(before: &Stat, after: &Stat, directory: bool) -> bool {
    if directory {
        // Concurrent publishers may change directory contents, never its identity or authority.
        before.st_dev == after.st_dev
            && before.st_ino == after.st_ino
            && before.st_mode == after.st_mode
            && before.st_uid == after.st_uid
    } else {
        same(before, after)
    }
}

fn validate(stat: &Stat, directory: bool, module: bool) -> Result<(), String> {
    let expected = if directory {
        FileType::Directory
    } else {
        FileType::RegularFile
    };
    let mode = if directory { 0o700 } else { 0o600 };
    if FileType::from_raw_mode(stat.st_mode) != expected
        || stat.st_mode & 0o7777 != mode
        || stat.st_uid != rustix::process::geteuid().as_raw()
    {
        return Err("artifact entry type, ownership or private mode violation".into());
    }
    if !directory && (stat.st_nlink == 0 || (!module && stat.st_nlink != 1)) {
        return Err("artifact forbidden hard link".into());
    }
    Ok(())
}

pub(crate) fn child(
    root: &File,
    name: &str,
    directory: bool,
    module: bool,
) -> Result<File, String> {
    let before =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    validate(&before, directory, module)?;
    child_checked(root, name, directory, &before)
}

fn child_checked(root: &File, name: &str, directory: bool, before: &Stat) -> Result<File, String> {
    let flags = OFlags::RDONLY
        | OFlags::CLOEXEC
        | OFlags::NOFOLLOW
        | OFlags::NONBLOCK
        | if directory {
            OFlags::DIRECTORY
        } else {
            OFlags::empty()
        };
    let file = File::from(
        fs::openat(root, name, flags, Mode::empty()).map_err(|error| error.to_string())?,
    );
    let opened = fs::fstat(&file).map_err(|error| error.to_string())?;
    let after =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    if !same_opened(before, &opened, directory) || !same_opened(&opened, &after, directory) {
        return Err("artifact entry changed while opening".into());
    }
    Ok(file)
}

pub(crate) fn pin(path: &Path) -> Result<File, String> {
    let before =
        fs::statat(fs::CWD, path, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    validate(&before, true, false)?;
    pin_checked(path, &before)
}

fn pin_checked(path: &Path, before: &Stat) -> Result<File, String> {
    let root = File::from(
        fs::open(
            path,
            OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::empty(),
        )
        .map_err(|error| error.to_string())?,
    );
    let after =
        fs::statat(fs::CWD, path, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    let opened = fs::fstat(&root).map_err(|error| error.to_string())?;
    if !same_opened(before, &opened, true) || !same_opened(&opened, &after, true) {
        return Err("artifact root changed while opening".into());
    }
    Ok(root)
}

pub(crate) fn read(root: &File, name: &str, bound: u64, module: bool) -> Result<Vec<u8>, String> {
    read_authorized(root, name, bound, module, None)
}

fn read_authorized(
    root: &File,
    name: &str,
    bound: u64,
    module: bool,
    proof: Option<(&FileEntry, Option<&File>)>,
) -> Result<Vec<u8>, String> {
    validate(
        &fs::fstat(root).map_err(|error| error.to_string())?,
        true,
        false,
    )?;
    let parts = components(name)?;
    let mut directory = root.try_clone().map_err(|error| error.to_string())?;
    for part in &parts[..parts.len() - 1] {
        directory = child(&directory, part, true, false)?;
    }
    let mut file = child(&directory, parts[parts.len() - 1], false, module)?;
    let before = fs::fstat(&file).map_err(|error| error.to_string())?;
    if let Some((entry, pool)) = proof
        && entry.module
    {
        module_link(&before, pool, entry)?;
    }
    read_handle(&mut file, bound, || Ok(()))
}

fn read_handle(
    file: &mut File,
    bound: u64,
    completed: impl FnOnce() -> Result<(), String>,
) -> Result<Vec<u8>, String> {
    let before = fs::fstat(&*file).map_err(|error| error.to_string())?;
    if !u64::try_from(before.st_size).is_ok_and(|size| size <= bound) {
        return Err("artifact file exceeds bound".into());
    }
    let capacity =
        usize::try_from(before.st_size).map_err(|_| "artifact file exceeds host capacity")?;
    let mut bytes = Vec::new();
    bytes
        .try_reserve_exact(capacity)
        .map_err(|error| error.to_string())?;
    (&mut *file)
        .take(bound.checked_add(1).ok_or("artifact read bound overflow")?)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    completed()?;
    let after = fs::fstat(&*file).map_err(|error| error.to_string())?;
    if bytes.len() != capacity || !same(&before, &after) {
        return Err("artifact file changed while reading".into());
    }
    Ok(bytes)
}

fn module_link(module: &Stat, pool: Option<&File>, entry: &FileEntry) -> Result<(), String> {
    if module.st_nlink == 1 {
        return Ok(());
    }
    let pool = pool.ok_or("module hard link has no authenticated target pool")?;
    validate(
        &fs::fstat(pool).map_err(|error| error.to_string())?,
        true,
        false,
    )?;
    let mut file = child(pool, &entry.sha256, false, true)?;
    let before = fs::fstat(&file).map_err(|error| error.to_string())?;
    if before.st_dev != module.st_dev || before.st_ino != module.st_ino {
        return Err("module target pool inode mismatch".into());
    }
    let mut bytes = Vec::new();
    (&mut file)
        .take(
            entry
                .bytes
                .checked_add(1)
                .ok_or("module pool size overflow")?,
        )
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 != entry.bytes
        || canonical::digest(&bytes) != entry.sha256
        || !same(
            &before,
            &fs::fstat(&file).map_err(|error| error.to_string())?,
        )
    {
        return Err("module target pool identity mismatch".into());
    }
    Ok(())
}

pub(crate) const LAYOUT: &[u8] =
    b"{\"schema\":\"gomad.artifact-store/v2\",\"target_pool\":\".targets\"}";

fn store_pool(path: &Path, artifact: &File) -> Result<Option<File>, String> {
    let Some(parent) = path.parent() else {
        return Ok(None);
    };
    let Ok(store) = pin(parent) else {
        return Ok(None);
    };
    let layout = match read(&store, ".store.json", 4096, false) {
        Ok(layout) => layout,
        Err(error) if error == rustix::io::Errno::NOENT.to_string() => return Ok(None),
        Err(error) => return Err(error),
    };
    if layout != LAYOUT {
        return Err("unsupported artifact store layout".into());
    }
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("invalid artifact directory name")?;
    let from_store = child(&store, name, true, false)?;
    let pinned = fs::fstat(artifact).map_err(|error| error.to_string())?;
    let contained = fs::fstat(&from_store).map_err(|error| error.to_string())?;
    if pinned.st_dev != contained.st_dev || pinned.st_ino != contained.st_ino {
        return Err("artifact store context substituted".into());
    }
    Ok(Some(child(&store, ".targets", true, false)?))
}

pub(crate) fn entries(root: &File) -> Result<Vec<String>, String> {
    let directory = fs::Dir::read_from(root).map_err(|error| error.to_string())?;
    let mut names = Vec::new();
    for item in directory {
        let item = item.map_err(|error| error.to_string())?;
        let name = item
            .file_name()
            .to_str()
            .map_err(|_| "artifact path is not UTF-8")?;
        if !matches!(name, "." | "..") {
            names.push(name.into());
        }
    }
    names.sort();
    Ok(names)
}

fn walk(
    root: &File,
    prefix: &str,
    manifest: &Manifest,
    seen: &mut BTreeSet<String>,
    pool: Option<&File>,
) -> Result<(), String> {
    for name in entries(root)? {
        let relative = if prefix.is_empty() {
            name.clone()
        } else {
            format!("{prefix}/{name}")
        };
        if prefix.is_empty() && matches!(name.as_str(), "manifest.json" | "storage.json") {
            continue;
        }
        let stat = fs::statat(root, name.as_str(), AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if FileType::from_raw_mode(stat.st_mode) == FileType::Directory {
            if !manifest
                .files
                .keys()
                .any(|path| path.starts_with(&format!("{relative}/")))
            {
                return Err("unlisted artifact directory".into());
            }
            walk(
                &child(root, &name, true, false)?,
                &relative,
                manifest,
                seen,
                pool,
            )?;
        } else {
            let expected = manifest
                .files
                .get(&relative)
                .ok_or("unlisted artifact payload")?;
            let file = child(root, &name, false, expected.module)?;
            let opened = fs::fstat(&file).map_err(|error| error.to_string())?;
            if expected.module {
                module_link(&opened, pool, expected)?;
            }
            if u64::try_from(opened.st_size) != Ok(expected.bytes) {
                return Err("artifact payload size mismatch".into());
            }
            seen.insert(relative);
        }
    }
    Ok(())
}

pub struct Opened {
    root: File,
    pool: Option<File>,
    manifest: Manifest,
    stored_bytes: u64,
}

impl Opened {
    /// # Errors
    /// Refuses unsafe roots, unsupported lineage and unauthenticated manifests or entries.
    pub fn open(path: &Path) -> Result<Self, String> {
        let root = pin(path)?;
        let pool = store_pool(path, &root)?;
        Self::from_root_with_pool(root, pool.as_ref())
    }

    /// # Errors
    /// Refuses unsafe pool roots or module links without an authenticated pool inode match.
    pub fn open_with_pool(path: &Path, pool: &Path) -> Result<Self, String> {
        Self::from_root_with_pool(pin(path)?, Some(&pin(pool)?))
    }

    #[cfg(feature = "determinism-fixture")]
    /// # Errors
    /// Deliberately substitutes the root between stat and open; safe pinning must refuse it.
    pub fn fixture_root_substitution(path: &Path) -> Result<Self, String> {
        let before = fs::statat(fs::CWD, path, AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        validate(&before, true, false)?;
        let moved = path.with_extension("substituted");
        std::fs::rename(path, &moved).map_err(|error| error.to_string())?;
        std::os::unix::fs::DirBuilderExt::mode(&mut std::fs::DirBuilder::new(), 0o700)
            .create(path)
            .map_err(|error| error.to_string())?;
        Self::from_root_with_pool(pin_checked(path, &before)?, None)
    }

    #[cfg(feature = "determinism-fixture")]
    /// # Errors
    /// Mutates a payload between the production read and final metadata check.
    pub fn fixture_payload_mutation(path: &Path, substitute: bool) -> Result<(), String> {
        let root = pin(path)?;
        let output = child(&root, "output", true, false)?;
        let before = fs::statat(&output, "stdout", AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        let payload = path.join("output/stdout");
        if substitute {
            let bytes = std::fs::read(&payload).map_err(|error| error.to_string())?;
            std::fs::rename(&payload, payload.with_extension("old"))
                .map_err(|error| error.to_string())?;
            std::fs::OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .open(&payload)
                .and_then(|mut file| std::io::Write::write_all(&mut file, &bytes))
                .map_err(|error| error.to_string())?;
            child_checked(&output, "stdout", false, &before)?;
        } else {
            let mut file = child(&output, "stdout", false, false)?;
            read_handle(&mut file, 4096, || {
                std::fs::write(&payload, b"changed while authenticated handle is reading")
                    .map_err(|error| error.to_string())
            })?;
        }
        Err("mutation unexpectedly accepted".into())
    }

    pub(crate) fn from_root_with_pool(root: File, pool: Option<&File>) -> Result<Self, String> {
        let pool = pool
            .map(File::try_clone)
            .transpose()
            .map_err(|error| error.to_string())?;
        let envelope = read(&root, "storage.json", 4096, false)
            .map_err(|error| format!("unsupported artifact storage format: {error}"))?;
        lineage::validate(&envelope)?;
        let bytes = read(&root, "manifest.json", 16 << 20, false)?;
        let manifest: Manifest = serde_json::from_slice(&bytes)
            .map_err(|error| format!("decode artifact manifest: {error}"))?;
        manifest.validate()?;
        if canonical::encode(&manifest)? != bytes {
            return Err("manifest is not canonical".into());
        }
        let mut seen = BTreeSet::new();
        walk(&root, "", &manifest, &mut seen, pool.as_ref())?;
        if seen.len() != manifest.files.len() {
            return Err("artifact missing listed payload".into());
        }
        let stored_bytes = manifest.stored_bytes(bytes.len() as u64)?;
        Ok(Self {
            root,
            pool,
            manifest,
            stored_bytes,
        })
    }

    #[must_use]
    pub fn manifest(&self) -> &Manifest {
        &self.manifest
    }
    #[must_use]
    pub fn stored_bytes(&self) -> u64 {
        self.stored_bytes
    }

    /// # Errors
    /// Refuses unlisted, oversized, unsafe, mutated or digest-mismatched payloads.
    pub fn payload(&self, name: &str, bound: u64) -> Result<Vec<u8>, String> {
        let entry = self
            .manifest
            .files
            .get(name)
            .ok_or("unlisted artifact payload")?;
        if entry.bytes > bound {
            return Err("artifact payload exceeds caller bound".into());
        }
        let bytes = read_authorized(
            &self.root,
            name,
            entry.bytes,
            entry.module,
            Some((entry, self.pool.as_ref())),
        )?;
        if bytes.len() as u64 != entry.bytes || canonical::digest(&bytes) != entry.sha256 {
            return Err("artifact payload identity mismatch".into());
        }
        Ok(bytes)
    }

    /// # Errors
    /// Returns the first payload authentication failure.
    pub fn authenticate_all(&self) -> Result<(), String> {
        for (name, entry) in &self.manifest.files {
            self.payload(name, entry.bytes)?;
        }
        Ok(())
    }
}
