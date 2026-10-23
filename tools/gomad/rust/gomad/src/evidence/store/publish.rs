use super::{FileEntry, Manifest, Opened, canonical, lineage, open, pool};
use crate::evidence::record::Record;
use rustix::fs::{self, AtFlags, FlockOperation, Mode, OFlags};
use sha2::{Digest, Sha256};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::{DirBuilder, File},
    io::{Read, Write},
    os::unix::fs::DirBuilderExt,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

static STAGING_SEQUENCE: AtomicU64 = AtomicU64::new(0);

pub struct Store {
    pub root: PathBuf,
    pub maximum_bytes: u64,
    pub target_pool: Option<PathBuf>,
    #[cfg(feature = "determinism-fixture")]
    pub fault: Option<String>,
}

pub struct Publication {
    pub record: Record,
    pub payloads: BTreeMap<String, Vec<u8>>,
}

pub struct Published {
    pub path: PathBuf,
    pub record_sha256: String,
    pub stored_bytes: u64,
}

pub(crate) fn durable_directory(
    path: &Path,
    mut checkpoint: impl FnMut(&str) -> Result<(), String>,
) -> Result<File, String> {
    let mut missing = Vec::new();
    let mut current = path;
    while !current.exists() {
        missing.push(current.to_path_buf());
        current = current.parent().ok_or("store has no existing parent")?;
    }
    for directory in missing.iter().rev() {
        match DirBuilder::new().mode(0o700).create(directory) {
            Ok(()) => checkpoint("directory_created")?,
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(error) => return Err(error.to_string()),
        }
    }
    let root = open::pin(path)?;
    let mut directory = root.try_clone().map_err(|error| error.to_string())?;
    // Existing entries may belong to an interrupted creator. Pin and sync every
    // ancestor, rather than relying on who happened to observe a missing path.
    loop {
        directory.sync_all().map_err(|error| error.to_string())?;
        let parent = File::from(
            fs::openat(
                &directory,
                "..",
                OFlags::RDONLY | OFlags::DIRECTORY | OFlags::CLOEXEC | OFlags::NOFOLLOW,
                Mode::empty(),
            )
            .map_err(|error| error.to_string())?,
        );
        checkpoint("before_parent_sync")?;
        parent.sync_all().map_err(|error| error.to_string())?;
        checkpoint("parent_durable")?;
        let current = fs::fstat(&directory).map_err(|error| error.to_string())?;
        let above = fs::fstat(&parent).map_err(|error| error.to_string())?;
        if current.st_dev == above.st_dev && current.st_ino == above.st_ino {
            break;
        }
        directory = parent;
    }
    Ok(root)
}

pub(crate) fn lock(root: &File) -> Result<File, String> {
    let lock = File::from(
        fs::openat(
            root,
            ".lock",
            OFlags::CREATE | OFlags::RDWR | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    let opened = fs::fstat(&lock).map_err(|error| error.to_string())?;
    let checked = open::child(root, ".lock", false, false)?;
    if !open::same(
        &opened,
        &fs::fstat(&checked).map_err(|error| error.to_string())?,
    ) {
        return Err("store lock substituted".into());
    }
    fs::flock(&lock, FlockOperation::LockExclusive).map_err(|error| error.to_string())?;
    if !open::same(
        &opened,
        &fs::statat(root, ".lock", AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?,
    ) {
        return Err("store lock substituted while waiting".into());
    }
    root.sync_all().map_err(|error| error.to_string())?;
    Ok(lock)
}

fn ordered_roots<'a>(store: &'a File, pools: &[&'a File]) -> Result<Vec<&'a File>, String> {
    let identity = |file: &File| {
        fs::fstat(file)
            .map(|stat| (stat.st_dev, stat.st_ino))
            .map_err(|error| error.to_string())
    };
    let store_identity = identity(store)?;
    let mut roots = vec![(store_identity, store)];
    for pool in pools {
        let pool_identity = identity(pool)?;
        if pool_identity == store_identity {
            return Err("target pool must differ from artifact store".into());
        }
        if !roots.iter().any(|(found, _)| *found == pool_identity) {
            roots.push((pool_identity, *pool));
        }
    }
    roots.sort_unstable_by_key(|(identity, _)| *identity);
    Ok(roots.into_iter().map(|(_, file)| file).collect())
}

pub(super) type ByteWriter<'a> = dyn FnMut(&mut File, &[u8]) -> std::io::Result<()> + 'a;

fn write_with(
    root: &File,
    name: &str,
    bytes: &[u8],
    writer: &mut ByteWriter<'_>,
) -> Result<(), String> {
    let mut file = File::from(
        fs::openat(
            root,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    writer(&mut file, bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())
}

fn destination(root: &File, name: &str) -> Result<(File, String), String> {
    let parts = open::components(name)?;
    let mut directory = root.try_clone().map_err(|error| error.to_string())?;
    for part in &parts[..parts.len() - 1] {
        match fs::mkdirat(&directory, *part, Mode::from_raw_mode(0o700)) {
            Ok(()) => directory.sync_all().map_err(|error| error.to_string())?,
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
        directory = open::child(&directory, part, true, false)?;
    }
    Ok((directory, parts[parts.len() - 1].into()))
}

fn sync_tree(root: &File) -> Result<(), String> {
    for name in open::entries(root)? {
        let stat = fs::statat(root, name.as_str(), AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if fs::FileType::from_raw_mode(stat.st_mode) == fs::FileType::Directory {
            sync_tree(&open::child(root, &name, true, false)?)?;
        }
    }
    root.sync_all().map_err(|error| error.to_string())
}

fn remove_stage(root: &File, name: &str) -> Result<(), String> {
    let directory = open::child(root, name, true, false)?;
    for child in open::entries(&directory)? {
        let stat = fs::statat(&directory, child.as_str(), AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if fs::FileType::from_raw_mode(stat.st_mode) == fs::FileType::Directory {
            remove_stage(&directory, &child)?;
        } else {
            let file = open::exclusive(&directory, &child, true)?;
            unlink_checked(&directory, &child, &file, false)?;
        }
    }
    fs::unlinkat(root, name, AtFlags::REMOVEDIR).map_err(|error| error.to_string())
}

fn rename_exclusive(
    source_root: &File,
    source: &str,
    target_root: &File,
    target: &str,
) -> Result<(), rustix::io::Errno> {
    #[cfg(any(target_os = "linux", target_os = "macos"))]
    {
        fs::renameat_with(
            source_root,
            source,
            target_root,
            target,
            fs::RenameFlags::NOREPLACE,
        )
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        let _ = (source_root, source, target_root, target);
        Err(rustix::io::Errno::NOSYS)
    }
}

fn staging(root: &File) -> Result<(String, File), String> {
    loop {
        let sequence = STAGING_SEQUENCE.fetch_add(1, Ordering::Relaxed);
        let name = format!(".publish-{}-{sequence}", std::process::id());
        match fs::mkdirat(root, name.as_str(), Mode::from_raw_mode(0o700)) {
            Ok(()) => return Ok((name.clone(), open::child(root, &name, true, false)?)),
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
    }
}

pub(crate) fn artifact_name(name: &str) -> bool {
    if canonical::valid_digest(name) {
        return true;
    }
    let Some(execution) = name.strip_prefix("execution-") else {
        return false;
    };
    execution.split_once("-context-").map_or_else(
        || canonical::valid_digest(execution),
        |(record, context)| canonical::valid_digest(record) && canonical::valid_digest(context),
    )
}

fn context_name(manifest: &Manifest) -> Option<String> {
    manifest
        .files
        .get(crate::evidence::search_context::FILE)
        .map(|entry| {
            format!(
                "execution-{}-context-{}",
                manifest.record_sha256, entry.sha256
            )
        })
}

pub(crate) fn matches_name(name: &str, manifest: &Manifest) -> Result<bool, String> {
    if let Some(qualified) = context_name(manifest) {
        return Ok(name == qualified);
    }
    Ok(name == manifest.record.failure_signature()?
        || name == format!("execution-{}", manifest.record_sha256))
}

fn unlink_checked(root: &File, name: &str, file: &File, directory: bool) -> Result<(), String> {
    let opened = fs::fstat(file).map_err(|error| error.to_string())?;
    let named =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    if !open::same(&opened, &named) {
        return Err("retiring artifact entry substituted".into());
    }
    fs::unlinkat(
        root,
        name,
        if directory {
            AtFlags::REMOVEDIR
        } else {
            AtFlags::empty()
        },
    )
    .map_err(|error| error.to_string())
}

fn retire_file(root: &File, name: &str, entry: &FileEntry) -> Result<(), String> {
    let mut file = open::exclusive(root, name, entry.module)?;
    let before = fs::fstat(&file).map_err(|error| error.to_string())?;
    if u64::try_from(before.st_size) != Ok(entry.bytes) {
        return Err("retiring artifact payload size mismatch".into());
    }
    let mut hash = Sha256::new();
    let mut buffer = vec![0_u8; 64 << 10];
    let mut count = 0_u64;
    loop {
        let size = file.read(&mut buffer).map_err(|error| error.to_string())?;
        if size == 0 {
            break;
        }
        count = count
            .checked_add(size as u64)
            .ok_or("retiring artifact payload size overflow")?;
        if count > entry.bytes {
            return Err("retiring artifact payload grew while reading".into());
        }
        hash.update(&buffer[..size]);
    }
    if count != entry.bytes
        || format!("{:x}", hash.finalize()) != entry.sha256
        || !open::same(
            &before,
            &fs::fstat(&file).map_err(|error| error.to_string())?,
        )
    {
        return Err("retiring artifact payload authentication failed".into());
    }
    unlink_checked(root, name, &file, false)
}

fn retire_payloads(root: &File, prefix: &str, manifest: &Manifest) -> Result<(), String> {
    for name in open::entries(root)? {
        if prefix.is_empty() && matches!(name.as_str(), "manifest.json" | "storage.json") {
            continue;
        }
        let path = if prefix.is_empty() {
            name.clone()
        } else {
            format!("{prefix}/{name}")
        };
        if let Some(entry) = manifest.files.get(&path) {
            retire_file(root, &name, entry)?;
        } else {
            let nested = format!("{path}/");
            if !manifest.files.keys().any(|path| path.starts_with(&nested)) {
                return Err("retiring artifact contains an unlisted payload".into());
            }
            let directory = open::child(root, &name, true, false)?;
            retire_payloads(&directory, &path, manifest)?;
            unlink_checked(root, &name, &directory, true)?;
        }
    }
    root.sync_all().map_err(|error| error.to_string())
}

fn retire_pending(root: &File, name: &str) -> Result<(), String> {
    let original = name
        .strip_prefix(".retire-")
        .filter(|name| artifact_name(name))
        .ok_or("invalid artifact retirement name")?;
    let directory = open::child(root, name, true, false)?;
    let bytes = match open::read(&directory, "manifest.json", 16 << 20, false) {
        Ok(bytes) => bytes,
        Err(error)
            if error == rustix::io::Errno::NOENT.to_string()
                && open::entries(&directory)?.is_empty() =>
        {
            unlink_checked(root, name, &directory, true)?;
            return root.sync_all().map_err(|error| error.to_string());
        }
        Err(error) => return Err(error),
    };
    let manifest: Manifest = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    manifest.validate()?;
    if canonical::encode(&manifest)? != bytes || !matches_name(original, &manifest)? {
        return Err("retiring artifact manifest identity mismatch".into());
    }
    match open::read(&directory, "storage.json", 4096, false) {
        Ok(bytes) => lineage::validate(&bytes)?,
        Err(error) if error == rustix::io::Errno::NOENT.to_string() => {}
        Err(error) => return Err(error),
    }
    retire_payloads(&directory, "", &manifest)?;
    match open::child(&directory, "storage.json", false, false) {
        Ok(_) => retire_file(
            &directory,
            "storage.json",
            &FileEntry {
                bytes: lineage::ENVELOPE.len() as u64,
                sha256: canonical::digest(lineage::ENVELOPE),
                module: false,
            },
        )?,
        Err(error) if error == rustix::io::Errno::NOENT.to_string() => {}
        Err(error) => return Err(error),
    }
    directory.sync_all().map_err(|error| error.to_string())?;
    retire_file(
        &directory,
        "manifest.json",
        &FileEntry {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            module: false,
        },
    )?;
    directory.sync_all().map_err(|error| error.to_string())?;
    unlink_checked(root, name, &directory, true)?;
    root.sync_all().map_err(|error| error.to_string())
}

impl Store {
    #[cfg_attr(
        not(feature = "determinism-fixture"),
        expect(
            clippy::unused_self,
            clippy::unnecessary_wraps,
            reason = "Immutable-file faults are absent from ordinary builds"
        )
    )]
    fn immutable_checkpoint(&self, module: bool, point: &str) -> Result<(), String> {
        #[cfg(feature = "determinism-fixture")]
        self.checkpoint(&format!(
            "{}_{point}",
            if module { "pool" } else { "layout" }
        ))?;
        let _ = (module, point);
        Ok(())
    }

    pub(super) fn immutable_file_with(
        &self,
        root: &File,
        name: &str,
        bytes: &[u8],
        module: bool,
        writer: &mut ByteWriter<'_>,
    ) -> Result<(), String> {
        match open::child(root, name, false, module) {
            Ok(mut file) => {
                self.reuse_immutable(&mut file, bytes, module)?;
                return root.sync_all().map_err(|error| error.to_string());
            }
            Err(error) if error == rustix::io::Errno::NOENT.to_string() => {}
            Err(error) => return Err(error),
        }
        let (stage_name, stage) = staging(root)?;
        let mut file = File::from(
            fs::openat(
                &stage,
                "payload",
                OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
                Mode::from_raw_mode(0o600),
            )
            .map_err(|error| error.to_string())?,
        );
        if module {
            fs::flock(&file, FlockOperation::LockExclusive).map_err(|error| error.to_string())?;
        }
        self.immutable_checkpoint(module, "file_created")?;
        let (first, rest) = bytes.split_at(bytes.len() / 2);
        writer(&mut file, first).map_err(|error| error.to_string())?;
        self.immutable_checkpoint(module, "file_partial")?;
        writer(&mut file, rest).map_err(|error| error.to_string())?;
        self.immutable_checkpoint(module, "before_file_sync")?;
        self.immutable_checkpoint(module, "file_sync_error")?;
        file.sync_all().map_err(|error| error.to_string())?;
        self.immutable_checkpoint(module, "file_durable")?;
        stage.sync_all().map_err(|error| error.to_string())?;
        match rename_exclusive(&stage, "payload", root, name) {
            Ok(()) => {
                self.immutable_checkpoint(module, "file_published")?;
                drop(file);
            }
            Err(error) if error == rustix::io::Errno::EXIST => {
                drop(file);
                let mut existing = open::child(root, name, false, module)?;
                self.reuse_immutable(&mut existing, bytes, module)?;
            }
            Err(error) => return Err(format!("exclusive immutable publication refused: {error}")),
        }
        remove_stage(root, &stage_name)?;
        root.sync_all().map_err(|error| error.to_string())
    }

    fn reuse_immutable(&self, file: &mut File, bytes: &[u8], module: bool) -> Result<(), String> {
        let bound = if module { bytes.len() as u64 } else { 4096 };
        if open::read_handle(file, bound, || Ok(()))? != bytes {
            return Err(if module {
                "target pool identity mismatch".into()
            } else {
                "unsupported artifact store layout".into()
            });
        }
        self.immutable_checkpoint(module, "before_file_sync")?;
        self.immutable_checkpoint(module, "file_sync_error")?;
        file.sync_all().map_err(|error| error.to_string())?;
        self.immutable_checkpoint(module, "file_durable")
    }

    pub(crate) fn retire(root: &File, live: &BTreeSet<String>) -> Result<(), String> {
        let pool = open::child(root, ".targets", true, false)?;
        let _locks = ordered_roots(root, &[&pool])?
            .into_iter()
            .map(lock)
            .collect::<Result<Vec<_>, _>>()?;
        if open::read(root, ".store.json", 4096, false)? != open::LAYOUT {
            return Err("unsupported artifact store layout".into());
        }
        if live.iter().any(|name| !artifact_name(name)) {
            return Err("invalid live artifact reference".into());
        }
        for name in open::entries(root)? {
            if name.starts_with(".retire-") {
                retire_pending(root, &name)?;
            }
        }
        for name in open::entries(root)? {
            if matches!(name.as_str(), ".store.json" | ".lock" | ".targets") {
                continue;
            }
            if name.starts_with(".publish-") {
                open::child(root, &name, true, false)?;
                continue;
            }
            if !artifact_name(&name) {
                return Err("unexpected artifact store entry during retirement".into());
            }
            let directory = open::child(root, &name, true, false)?;
            let before = fs::fstat(&directory).map_err(|error| error.to_string())?;
            let opened = Opened::from_root_with_pool(directory, Some(&pool))?;
            opened.authenticate_all()?;
            if !matches_name(&name, opened.manifest())? {
                return Err("retiring artifact name does not match its record".into());
            }
            if live.contains(&name) {
                continue;
            }
            let after = fs::statat(root, name.as_str(), AtFlags::SYMLINK_NOFOLLOW)
                .map_err(|error| error.to_string())?;
            if !open::same(&before, &after) {
                return Err("retiring artifact directory substituted".into());
            }
            let retired = format!(".retire-{name}");
            rename_exclusive(root, &name, root, &retired).map_err(|error| error.to_string())?;
            root.sync_all().map_err(|error| error.to_string())?;
            retire_pending(root, &retired)?;
        }
        root.sync_all().map_err(|error| error.to_string())
    }

    #[cfg_attr(
        not(feature = "determinism-fixture"),
        expect(
            clippy::unused_self,
            clippy::unnecessary_wraps,
            reason = "The same publication boundaries are fallible only in fixture builds"
        )
    )]
    fn checkpoint(&self, point: &str) -> Result<(), String> {
        #[cfg(feature = "determinism-fixture")]
        if self.fault.as_deref() == Some(&format!("pause:{point}")) {
            std::fs::write(self.root.with_extension("paused"), b"paused")
                .map_err(|error| error.to_string())?;
            let deadline = std::time::Instant::now() + std::time::Duration::from_secs(10);
            while !self.root.with_extension("release").exists() {
                if std::time::Instant::now() >= deadline {
                    return Err("fixture pause expired".into());
                }
                std::thread::sleep(std::time::Duration::from_millis(1));
            }
        }
        #[cfg(feature = "determinism-fixture")]
        if self.fault.as_deref() == Some(&format!("audit:{point}")) {
            std::fs::write(self.root.with_extension("parent-synced"), b"synced")
                .map_err(|error| error.to_string())?;
        }
        #[cfg(feature = "determinism-fixture")]
        if self
            .fault
            .as_deref()
            .and_then(|fault| fault.strip_prefix("crash:"))
            == Some(point)
        {
            std::process::exit(86);
        }
        #[cfg(feature = "determinism-fixture")]
        if self.fault.as_deref() == Some(point) {
            return Err(format!("fixture publication interruption at {point}"));
        }
        let _ = point;
        Ok(())
    }

    #[cfg_attr(
        not(feature = "determinism-fixture"),
        expect(
            clippy::unused_self,
            reason = "Fixture-only I/O faults use the same production sync boundary"
        )
    )]
    fn sync_store(&self, root: &File) -> Result<(), String> {
        #[cfg(feature = "determinism-fixture")]
        if self.fault.as_deref() == Some("store_sync_error") {
            return Err(format!("sync artifact store: {}", rustix::io::Errno::IO));
        }
        root.sync_all().map_err(|error| error.to_string())
    }

    #[cfg_attr(
        not(feature = "determinism-fixture"),
        expect(
            clippy::unused_self,
            reason = "Fixture-only unsupported-rename faults use the production error path"
        )
    )]
    fn rename(&self, root: &File, source: &str, target: &str) -> Result<(), rustix::io::Errno> {
        #[cfg(feature = "determinism-fixture")]
        if self.fault.as_deref() == Some("rename_unsupported") {
            return Err(rustix::io::Errno::NOSYS);
        }
        rename_exclusive(root, source, root, target)
    }

    /// # Errors
    /// Refuses invalid evidence, unsafe stores, exhausted budgets, collisions and durability failures.
    pub fn publish(&self, publication: Publication) -> Result<Published, String> {
        self.publish_with_writer(publication, &mut |file, bytes| file.write_all(bytes))
    }

    #[expect(
        clippy::too_many_lines,
        reason = "Publication keeps lock, staging, authentication, exclusive rename and durability boundaries in order."
    )]
    fn publish_with_writer(
        &self,
        publication: Publication,
        writer: &mut ByteWriter<'_>,
    ) -> Result<Published, String> {
        #[cfg(feature = "determinism-fixture")]
        if let Some(fault) = &self.fault {
            let point = fault
                .strip_prefix("crash:")
                .or_else(|| fault.strip_prefix("pause:"))
                .or_else(|| fault.strip_prefix("audit:"))
                .unwrap_or(fault);
            if ![
                "stage_created",
                "payloads_durable",
                "envelope_durable",
                "manifest_durable",
                "staging_durable",
                "before_rename",
                "after_rename",
                "store_durable",
                "before_reuse_sync",
                "after_reuse_sync",
                "store_sync_error",
                "rename_unsupported",
                "directory_created",
                "before_parent_sync",
                "parent_durable",
                "pool_file_created",
                "pool_file_partial",
                "pool_before_file_sync",
                "pool_file_sync_error",
                "pool_file_durable",
                "pool_file_published",
                "layout_file_created",
                "layout_file_partial",
                "layout_before_file_sync",
                "layout_file_sync_error",
                "layout_file_durable",
                "layout_file_published",
            ]
            .contains(&point)
            {
                return Err("unknown publication fault".into());
            }
        }
        let root = durable_directory(&self.root, |point| self.checkpoint(point))?;
        let shared_pool = self
            .target_pool
            .as_deref()
            .map(|path| durable_directory(path, |point| self.checkpoint(point)))
            .transpose()?;
        match fs::mkdirat(&root, ".targets", Mode::from_raw_mode(0o700)) {
            Ok(()) => root.sync_all().map_err(|error| error.to_string())?,
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
        let pool_root = open::child(&root, ".targets", true, false)?;
        let mut pools = vec![&pool_root];
        if let Some(shared) = &shared_pool {
            pools.push(shared);
        }
        let _locks = ordered_roots(&root, &pools)?
            .into_iter()
            .map(lock)
            .collect::<Result<Vec<_>, _>>()?;
        self.immutable_file_with(&root, ".store.json", open::LAYOUT, false, writer)?;
        publication.record.validate()?;
        if publication.payloads.is_empty() {
            return Err("artifact requires payloads".into());
        }
        let mut files = BTreeMap::new();
        for (name, bytes) in &publication.payloads {
            open::components(name)?;
            if matches!(name.as_str(), "manifest.json" | "storage.json") {
                return Err("reserved payload name".into());
            }
            files.insert(
                name.clone(),
                FileEntry {
                    bytes: bytes.len() as u64,
                    sha256: canonical::digest(bytes),
                    module: name == "target/module.wasm",
                },
            );
        }
        let manifest = Manifest {
            schema: "gomad.artifact-manifest/v2".into(),
            record_sha256: publication.record.hash()?,
            record: publication.record,
            files,
        };
        manifest.validate()?;
        let manifest_bytes = canonical::encode(&manifest)?;
        if manifest_bytes.len() > 16 << 20 {
            return Err("manifest exceeds bound".into());
        }
        let stored_bytes = manifest.stored_bytes(manifest_bytes.len() as u64)?;
        let names = context_name(&manifest).map_or_else(
            || {
                Ok(vec![
                    manifest.record.failure_signature()?,
                    format!("execution-{}", manifest.record_sha256),
                ])
            },
            |name| Ok::<_, String>(vec![name]),
        )?;
        for name in &names {
            match open::child(&root, name, true, false) {
                Ok(directory) => {
                    let existing = Opened::from_root_with_pool(directory, Some(&pool_root))?;
                    existing.authenticate_all()?;
                    if existing.manifest() == &manifest {
                        self.checkpoint("before_reuse_sync")?;
                        self.sync_store(&root)?;
                        self.checkpoint("after_reuse_sync")?;
                        return Ok(Published {
                            path: self.root.join(name),
                            record_sha256: manifest.record_sha256,
                            stored_bytes,
                        });
                    }
                }
                Err(error) if error == rustix::io::Errno::NOENT.to_string() => break,
                Err(error) => return Err(error),
            }
        }
        let mut retained = 0_u64;
        for name in open::entries(&root)? {
            if name.starts_with('.') {
                continue;
            }
            let existing = Opened::from_root_with_pool(
                open::child(&root, &name, true, false)?,
                Some(&pool_root),
            )?;
            existing.authenticate_all()?;
            if !matches_name(&name, existing.manifest())? {
                return Err("artifact store name does not match its manifest".into());
            }
            retained = retained
                .checked_add(existing.stored_bytes())
                .ok_or("retained bytes overflow")?;
        }
        let required = retained
            .checked_add(stored_bytes)
            .ok_or("retained bytes overflow")?;
        if self.maximum_bytes != 0 && required > self.maximum_bytes {
            return Err(format!(
                "artifact budget exceeded: requires {required}, maximum {}",
                self.maximum_bytes
            ));
        }
        let (stage_name, stage) = staging(&root)?;
        self.checkpoint("stage_created")?;
        for (name, bytes) in &publication.payloads {
            let (directory, basename) = destination(&stage, name)?;
            if name == "target/module.wasm"
                && let Some(pool) = &shared_pool
            {
                let local = fs::fstat(&pool_root).map_err(|error| error.to_string())?;
                let shared = fs::fstat(pool).map_err(|error| error.to_string())?;
                if local.st_dev != shared.st_dev || local.st_ino != shared.st_ino {
                    pool::place_with(
                        self,
                        pool,
                        &pool_root,
                        &canonical::digest(bytes),
                        bytes,
                        writer,
                    )?;
                }
                pool::place_with(self, &pool_root, &directory, &basename, bytes, writer)?;
            } else {
                write_with(&directory, &basename, bytes, writer)?;
            }
        }
        sync_tree(&stage)?;
        self.checkpoint("payloads_durable")?;
        write_with(&stage, "storage.json", lineage::ENVELOPE, writer)?;
        self.checkpoint("envelope_durable")?;
        write_with(&stage, "manifest.json", &manifest_bytes, writer)?;
        self.checkpoint("manifest_durable")?;
        sync_tree(&stage)?;
        self.checkpoint("staging_durable")?;
        Opened::from_root_with_pool(
            stage.try_clone().map_err(|error| error.to_string())?,
            Some(&pool_root),
        )?;
        for name in &names {
            self.checkpoint("before_rename")?;
            match self.rename(&root, &stage_name, name) {
                Ok(()) => {
                    self.checkpoint("after_rename")?;
                    self.sync_store(&root)?;
                    self.checkpoint("store_durable")?;
                    return Ok(Published {
                        path: self.root.join(name),
                        record_sha256: manifest.record_sha256,
                        stored_bytes,
                    });
                }
                Err(error)
                    if error == rustix::io::Errno::EXIST
                        || error == rustix::io::Errno::NOTEMPTY =>
                {
                    let existing = Opened::from_root_with_pool(
                        open::child(&root, name, true, false)?,
                        Some(&pool_root),
                    )?;
                    existing.authenticate_all()?;
                    if existing.manifest() == &manifest {
                        remove_stage(&root, &stage_name)?;
                        self.sync_store(&root)?;
                        return Ok(Published {
                            path: self.root.join(name),
                            record_sha256: manifest.record_sha256,
                            stored_bytes,
                        });
                    }
                }
                Err(error) => {
                    return Err(format!("exclusive artifact publication refused: {error}"));
                }
            }
        }
        Err("artifact identity collision".into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{sync::mpsc, thread, time::Duration};

    #[test]
    fn context_qualified_names_require_two_exact_digests() {
        let record = "1".repeat(64);
        let context = "2".repeat(64);
        assert!(super::artifact_name(&format!(
            "execution-{record}-context-{context}"
        )));
        for name in [
            format!("execution-{record}-context-unknown"),
            format!("execution-{record}-other-{context}"),
        ] {
            assert!(!super::artifact_name(&name));
        }
    }

    struct PoolFixture {
        root: PathBuf,
        store: Store,
        shared: File,
        first: File,
        second: File,
    }

    impl PoolFixture {
        fn new() -> Self {
            let root = std::env::temp_dir().join(format!(
                "gomad-pool-guard-{}",
                crate::campaign::origin::identity().unwrap(),
            ));
            let shared = durable_directory(&root.join("shared"), |_| Ok(())).unwrap();
            let first = durable_directory(&root.join("first"), |_| Ok(())).unwrap();
            let second = durable_directory(&root.join("second"), |_| Ok(())).unwrap();
            let store = Store {
                root: root.clone(),
                maximum_bytes: 64 << 20,
                target_pool: Some(root.join("shared")),
                #[cfg(feature = "determinism-fixture")]
                fault: None,
            };
            pool::place(&store, &shared, &first, "module.wasm", b"\0asm\x01\0\0\0").unwrap();
            Self {
                root,
                store,
                shared,
                first,
                second,
            }
        }
    }

    #[test]
    fn module_read_guard_is_shared_across_aliases_and_retained_by_clone() {
        let fixture = PoolFixture::new();
        let bytes = b"\0asm\x01\0\0\0";
        let digest = canonical::digest(bytes);
        // Releases go through the fork-ordered path so a concurrently forked
        // test child cannot still share the guard when the contender probes.
        let reader = crate::fork_order::Locked::new(
            open::child(&fixture.first, "module.wasm", false, true).unwrap(),
        );
        let cloned = crate::fork_order::Locked::new(reader.try_clone().unwrap());
        let contender = File::from(
            fs::openat(
                &fixture.shared,
                digest.as_str(),
                OFlags::RDONLY | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK,
                Mode::empty(),
            )
            .unwrap(),
        );
        assert_eq!(
            fs::flock(&contender, FlockOperation::NonBlockingLockExclusive),
            Err(rustix::io::Errno::WOULDBLOCK)
        );
        drop(reader);
        assert_eq!(
            fs::flock(&contender, FlockOperation::NonBlockingLockExclusive),
            Err(rustix::io::Errno::WOULDBLOCK)
        );
        drop(cloned);
        fs::flock(&contender, FlockOperation::NonBlockingLockExclusive).unwrap();
        drop(contender);
        std::fs::remove_dir_all(fixture.root).unwrap();
    }

    fn writer_waits_for_reader(operation: &'static str) {
        let fixture = PoolFixture::new();
        let bytes = b"\0asm\x01\0\0\0";
        if operation == "retire" {
            pool::place(
                &fixture.store,
                &fixture.shared,
                &fixture.second,
                "module.wasm",
                bytes,
            )
            .unwrap();
        } else if operation == "cleanup" {
            let stage = durable_directory(&fixture.root.join("second/stage"), |_| Ok(())).unwrap();
            pool::place(
                &fixture.store,
                &fixture.shared,
                &stage,
                "module.wasm",
                bytes,
            )
            .unwrap();
        }
        let mut reader = open::child(&fixture.first, "module.wasm", false, true).unwrap();
        let before = fs::fstat(&reader).unwrap();
        let shared = fixture.shared.try_clone().unwrap();
        let second = fixture.second.try_clone().unwrap();
        let writer_root = fixture.root.clone();
        let (ready, waiting) = mpsc::channel();
        let (done, completion) = mpsc::channel();
        let writer = thread::spawn(move || {
            let result = (|| {
                let _locks = ordered_roots(&second, &[&shared])?
                    .into_iter()
                    .map(lock)
                    .collect::<Result<Vec<_>, _>>()?;
                let digest = canonical::digest(bytes);
                let probe = File::from(
                    fs::openat(
                        &shared,
                        digest.as_str(),
                        OFlags::RDONLY | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK,
                        Mode::empty(),
                    )
                    .map_err(|error| error.to_string())?,
                );
                if fs::flock(&probe, FlockOperation::NonBlockingLockExclusive)
                    != Err(rustix::io::Errno::WOULDBLOCK)
                {
                    return Err("reader did not retain the shared module guard".into());
                }
                drop(probe);
                ready.send(()).unwrap();
                if operation == "retire" {
                    retire_file(
                        &second,
                        "module.wasm",
                        &FileEntry {
                            bytes: bytes.len() as u64,
                            sha256: digest,
                            module: true,
                        },
                    )
                } else if operation == "cleanup" {
                    remove_stage(&second, "stage")
                } else {
                    let store = Store {
                        root: writer_root,
                        maximum_bytes: 64 << 20,
                        target_pool: None,
                        #[cfg(feature = "determinism-fixture")]
                        fault: None,
                    };
                    pool::place(&store, &shared, &second, "module.wasm", bytes)
                }
            })();
            done.send(result).unwrap();
        });
        let observed = open::read_handle(&mut reader, bytes.len() as u64, || {
            waiting
                .recv_timeout(Duration::from_secs(2))
                .map_err(|error| error.to_string())?;
            if completion.recv_timeout(Duration::from_millis(50))
                != Err(mpsc::RecvTimeoutError::Timeout)
            {
                return Err("publisher modified the inode while the read guard was held".into());
            }
            Ok(())
        })
        .unwrap();
        assert_eq!(observed, bytes);
        assert!(open::same(&before, &fs::fstat(&reader).unwrap()));
        drop(reader);
        completion
            .recv_timeout(Duration::from_secs(2))
            .unwrap()
            .unwrap();
        writer.join().unwrap();
        std::fs::remove_dir_all(fixture.root).unwrap();
    }

    #[test]
    fn module_reader_serializes_cross_store_link_retirement_and_staging_cleanup() {
        for operation in ["link", "retire", "cleanup"] {
            writer_waits_for_reader(operation);
        }
    }

    #[test]
    fn target_link_collision_releases_exclusive_guard_before_reuse_read() {
        let (done, completion) = mpsc::channel();
        let worker = thread::spawn(move || {
            let fixture = PoolFixture::new();
            let bytes = b"\0asm\x01\0\0\0";
            pool::place(
                &fixture.store,
                &fixture.shared,
                &fixture.second,
                "module.wasm",
                bytes,
            )
            .unwrap();
            pool::place(
                &fixture.store,
                &fixture.shared,
                &fixture.second,
                "module.wasm",
                bytes,
            )
            .unwrap();
            assert_eq!(
                open::read(&fixture.second, "module.wasm", bytes.len() as u64, true).unwrap(),
                bytes
            );
            std::fs::remove_dir_all(fixture.root).unwrap();
            done.send(()).unwrap();
        });
        completion
            .recv_timeout(Duration::from_secs(2))
            .expect("same-inode collision self-deadlocked");
        worker.join().unwrap();
    }

    // Global descriptor-identity lock order prevents cross-assigned store/pool deadlock; CLI runs cannot force the acquisition interleaving.
    #[test]
    fn lock_order_is_independent_of_store_and_pool_roles() {
        let current = std::fs::File::open(".").unwrap();
        let parent = std::fs::File::open("..").unwrap();
        let identities = |roots: Vec<&std::fs::File>| {
            roots
                .into_iter()
                .map(|file| {
                    let stat = rustix::fs::fstat(file).unwrap();
                    (stat.st_dev, stat.st_ino)
                })
                .collect::<Vec<_>>()
        };
        assert_eq!(
            identities(super::ordered_roots(&current, &[&parent]).unwrap()),
            identities(super::ordered_roots(&parent, &[&current]).unwrap())
        );
        assert!(super::ordered_roots(&current, &[&current]).is_err());
    }
}

#[cfg(all(test, feature = "determinism-fixture"))]
#[path = "write_failure_tests.rs"]
mod write_failure_tests;
