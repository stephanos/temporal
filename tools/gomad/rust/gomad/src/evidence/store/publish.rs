use super::{FileEntry, Manifest, Opened, canonical, lineage, open, pool};
use crate::evidence::record::Record;
use rustix::fs::{self, AtFlags, FlockOperation, Mode, OFlags};
use std::{
    collections::BTreeMap,
    fs::{DirBuilder, File},
    io::Write,
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

pub(crate) fn durable_directory(path: &Path) -> Result<File, String> {
    let mut missing = Vec::new();
    let mut current = path;
    while !current.exists() {
        missing.push(current.to_path_buf());
        current = current.parent().ok_or("store has no existing parent")?;
    }
    for directory in missing.iter().rev() {
        match DirBuilder::new().mode(0o700).create(directory) {
            Ok(()) => {}
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(error) => return Err(error.to_string()),
        }
        open::pin(directory)?
            .sync_all()
            .map_err(|error| error.to_string())?;
        let parent = File::open(directory.parent().ok_or("store directory has no parent")?)
            .map_err(|error| error.to_string())?;
        parent.sync_all().map_err(|error| error.to_string())?;
    }
    open::pin(path)
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

pub(crate) fn write(root: &File, name: &str, bytes: &[u8]) -> Result<(), String> {
    let mut file = File::from(
        fs::openat(
            root,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(bytes).map_err(|error| error.to_string())?;
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
            fs::unlinkat(&directory, child.as_str(), AtFlags::empty())
                .map_err(|error| error.to_string())?;
        }
    }
    fs::unlinkat(root, name, AtFlags::REMOVEDIR).map_err(|error| error.to_string())
}

fn rename_exclusive(root: &File, source: &str, target: &str) -> Result<(), rustix::io::Errno> {
    #[cfg(any(target_os = "linux", target_os = "macos"))]
    {
        fs::renameat_with(root, source, root, target, fs::RenameFlags::NOREPLACE)
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        let _ = (root, source, target);
        Err(rustix::io::Errno::NOSYS)
    }
}

impl Store {
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
        rename_exclusive(root, source, target)
    }

    /// # Errors
    /// Refuses invalid evidence, unsafe stores, exhausted budgets, collisions and durability failures.
    #[expect(
        clippy::too_many_lines,
        reason = "Publication keeps lock, staging, authentication, exclusive rename and durability boundaries in order."
    )]
    pub fn publish(&self, publication: Publication) -> Result<Published, String> {
        #[cfg(feature = "determinism-fixture")]
        if let Some(fault) = &self.fault {
            let point = fault.strip_prefix("crash:").unwrap_or(fault);
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
            ]
            .contains(&point)
            {
                return Err("unknown publication fault".into());
            }
        }
        let root = durable_directory(&self.root)?;
        let shared_pool = self
            .target_pool
            .as_deref()
            .map(durable_directory)
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
        match open::read(&root, ".store.json", 4096, false) {
            Ok(layout) if layout == open::LAYOUT => {}
            Ok(_) => return Err("unsupported artifact store layout".into()),
            Err(error) if error == rustix::io::Errno::NOENT.to_string() => {
                write(&root, ".store.json", open::LAYOUT)?;
                root.sync_all().map_err(|error| error.to_string())?;
            }
            Err(error) => return Err(error),
        }
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
        let first_name = manifest.record.failure_signature()?;
        let fallback_name = format!("execution-{}", manifest.record_sha256);
        for name in [&first_name, &fallback_name] {
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
        let stage_name = loop {
            let sequence = STAGING_SEQUENCE.fetch_add(1, Ordering::Relaxed);
            let name = format!(".publish-{}-{sequence}", std::process::id());
            match fs::mkdirat(&root, name.as_str(), Mode::from_raw_mode(0o700)) {
                Ok(()) => break name,
                Err(error) if error == rustix::io::Errno::EXIST => {}
                Err(error) => return Err(error.to_string()),
            }
        };
        let stage = open::child(&root, &stage_name, true, false)?;
        self.checkpoint("stage_created")?;
        for (name, bytes) in &publication.payloads {
            let (directory, basename) = destination(&stage, name)?;
            if name == "target/module.wasm"
                && let Some(pool) = &shared_pool
            {
                let local = fs::fstat(&pool_root).map_err(|error| error.to_string())?;
                let shared = fs::fstat(pool).map_err(|error| error.to_string())?;
                if local.st_dev != shared.st_dev || local.st_ino != shared.st_ino {
                    pool::place(pool, &pool_root, &canonical::digest(bytes), bytes)?;
                }
                pool::place(&pool_root, &directory, &basename, bytes)?;
            } else {
                write(&directory, &basename, bytes)?;
            }
        }
        sync_tree(&stage)?;
        self.checkpoint("payloads_durable")?;
        write(&stage, "storage.json", lineage::ENVELOPE)?;
        self.checkpoint("envelope_durable")?;
        write(&stage, "manifest.json", &manifest_bytes)?;
        self.checkpoint("manifest_durable")?;
        sync_tree(&stage)?;
        self.checkpoint("staging_durable")?;
        for name in [&first_name, &fallback_name] {
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
