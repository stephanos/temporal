use rustix::fs::{self, AtFlags, FileType, FlockOperation, Mode, OFlags};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, VecDeque};
use std::ffi::{OsStr, OsString};
use std::fs::File;
use std::io::{Read, Write};
use std::os::unix::ffi::OsStringExt;
use std::path::{Component, Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use crate::fork_order::Locked;

const LOCK_TIMEOUT: Duration = Duration::from_secs(60);
static SEQUENCE: AtomicU64 = AtomicU64::new(0);

pub struct RetainedRun {
    store: File,
    root: File,
    lease: Option<Locked>,
    path: PathBuf,
    name: String,
}

pub struct ChildLease {
    requested_root: PathBuf,
    _lease: Locked,
}
impl ChildLease {
    #[must_use]
    pub fn requested_root(&self) -> &Path {
        &self.requested_root
    }
    /// # Errors
    /// Refuses moved or malformed parent stages before a child can execute within them.
    pub fn for_path(path: &Path) -> Result<Option<Self>, String> {
        let Some(parent) = path.ancestors().find(|path| managed_run(path)) else {
            return Ok(None);
        };
        let (canonical, root) = retention_base(parent, false)?;
        validate_pin(&root, &canonical)?;
        let lease = Locked::new(match create(&root, ".children") {
            Ok(file) => file,
            Err(_) => open(&root, OsStr::new(".children"), false)?,
        });
        fs::flock(&*lease, FlockOperation::NonBlockingLockShared)
            .map_err(|error| error.to_string())?;
        let current = directory(&canonical, false)?;
        if !same(&root, &current)? {
            return Err("retained parent stage was collected before child lease".into());
        }
        let requested_root = canonical
            .parent()
            .and_then(Path::parent)
            .and_then(Path::parent)
            .ok_or("managed child stage lost requested root")?
            .to_owned();
        Ok(Some(Self {
            requested_root,
            _lease: lease,
        }))
    }
}

impl RetainedRun {
    /// # Errors
    /// Refuses invalid scratch ownership or active inputs and returns filesystem or cancellation errors.
    pub fn begin(
        requested_root: &Path,
        producer: &str,
        inputs: &BTreeMap<String, String>,
        keep: bool,
        cancelled: &dyn Fn() -> bool,
    ) -> Result<Self, String> {
        check_cancelled(cancelled)?;
        validate_absolute(requested_root)?;
        if inside_store(requested_root) || !single_name(producer) {
            return Err("refused: invalid measurement output root or producer".into());
        }
        let (base_path, base) = retention_base(requested_root, true)?;
        if inside_store(&base_path) {
            return Err("refused: measurement output root is inside a retained store".into());
        }
        let measure = child_directory(&base, OsStr::new(".gomad-measure"), true)?;
        let store = child_directory(&measure, OsStr::new(producer), true)?;
        let lock = store_lock(&store, cancelled)?;
        let name = format!("run-{}", unique());
        let stage = format!(".stage-{name}");
        fs::mkdirat(&store, stage.as_str(), Mode::from_raw_mode(0o700))
            .map_err(|error| error.to_string())?;
        let setup = (|| {
            let root = child_directory(&store, OsStr::new(&stage), false)?;
            let lease = Locked::new(create(&root, ".lease")?);
            fs::flock(&*lease, FlockOperation::NonBlockingLockExclusive)
                .map_err(|error| error.to_string())?;
            let path = base_path.join(".gomad-measure").join(producer).join(&name);
            atomic_write(&root, ".pin", path.as_os_str().as_encoded_bytes())?;
            if keep {
                atomic_write(&root, ".keep", b"")?;
            }
            rename_exclusive(&store, &stage, &name)?;
            store.sync_all().map_err(|error| error.to_string())?;
            Ok(Self {
                store: store.try_clone().map_err(|error| error.to_string())?,
                root,
                lease: Some(lease),
                path,
                name,
            })
        })();
        drop(lock);
        let run = match setup {
            Ok(run) => run,
            Err(error) => {
                let _ = remove_tree(&store, OsStr::new(&stage));
                return Err(error);
            }
        };
        let pinned = run
            .pin_inputs(inputs)
            .and_then(|()| check_cancelled(cancelled));
        if let Err(error) = pinned {
            let _ = run.finish(false, &|| false);
            return Err(format!("refused: retain measurement inputs: {error}"));
        }
        Ok(run)
    }

    #[must_use]
    pub fn path(&self) -> &Path {
        &self.path
    }

    /// # Errors
    /// Returns errors publishing retained evidence, checking cancellation or releasing input pins.
    pub fn finish(mut self, success: bool, cancelled: &dyn Fn() -> bool) -> Result<(), String> {
        let published = if success {
            self.publish(cancelled)
        } else {
            Ok(())
        };
        self.lease.take();
        // Collection is best effort after the result is final, including cancellation.
        let _ = reclaim_cancellable(
            self.path
                .parent()
                .ok_or("measurement store has no parent")?,
            cancelled,
        );
        published
    }

    fn pin_inputs(&self, inputs: &BTreeMap<String, String>) -> Result<(), String> {
        for value in inputs.values() {
            let value = Path::new(value);
            if !value.is_absolute() {
                continue;
            }
            let Some(path) = value.ancestors().find(|path| managed_run(path)) else {
                continue;
            };
            let (canonical_path, root) = retention_base(path, false)?;
            let path = canonical_path.as_path();
            let lease = Locked::new(open(&root, OsStr::new(".lease"), false)?);
            fs::flock(&*lease, FlockOperation::NonBlockingLockShared)
                .map_err(|error| error.to_string())?;
            validate_pin(&root, path)?;
            let current = directory(path, false)?;
            if !same(&root, &current)? {
                return Err("retained measurement input was collected".into());
            }
            let pins = child_directory(&self.root, OsStr::new(".pins"), true)?;
            let digest = format!("{:x}", Sha256::digest(path.as_os_str().as_encoded_bytes()));
            let source = open(&root, OsStr::new(".pin"), false)?;
            match fs::linkat(&root, ".pin", &pins, digest.as_str(), AtFlags::empty()) {
                Ok(()) => {}
                Err(rustix::io::Errno::EXIST) => {
                    let source = open(&root, OsStr::new(".pin"), false)?;
                    let target = open(&pins, OsStr::new(&digest), false)?;
                    if !same(&source, &target)? {
                        return Err("retention input pin identity changed".into());
                    }
                }
                Err(error) => return Err(error.to_string()),
            }
            let linked = open(&pins, OsStr::new(&digest), false)?;
            if !same(&source, &linked)? {
                return Err("retention input pin identity changed".into());
            }
            pins.sync_all().map_err(|error| error.to_string())?;
            self.root.sync_all().map_err(|error| error.to_string())?;
        }
        Ok(())
    }

    fn publish(&self, cancelled: &dyn Fn() -> bool) -> Result<(), String> {
        check_cancelled(cancelled)?;
        let current = directory(&self.path, false)?;
        if !same(&current, &self.root)? {
            return Err("measurement run path changed".into());
        }
        sync_tree(&self.root, cancelled)?;
        atomic_write(&self.root, ".complete", b"")?;
        let result = (|| {
            let _lock = store_lock(&self.store, cancelled)?;
            check_cancelled(cancelled)?;
            latest(&self.store)?;
            let mut encoded = serde_json::to_vec(&self.name).map_err(|error| error.to_string())?;
            encoded.push(b'\n');
            atomic_write(&self.store, "latest.json", &encoded)
        })();
        if result.is_err() {
            fs::unlinkat(&self.root, ".complete", AtFlags::empty())
                .map_err(|error| error.to_string())?;
            self.root.sync_all().map_err(|error| error.to_string())?;
        }
        result
    }
}

fn unique() -> String {
    format!(
        "{}-{}-{}",
        std::process::id(),
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos(),
        SEQUENCE.fetch_add(1, Ordering::Relaxed)
    )
}

fn single_name(name: &str) -> bool {
    !name.is_empty() && name != "." && name != ".." && !name.contains(['/', '\\', '\0'])
}

fn validate_absolute(path: &Path) -> Result<(), String> {
    if !path.is_absolute()
        || path == Path::new("/")
        || path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
        || path.as_os_str() != path.components().collect::<PathBuf>().as_os_str()
    {
        return Err("refused: measurement root must be an absolute clean non-root path".into());
    }
    Ok(())
}

fn inside_store(path: &Path) -> bool {
    if path
        .components()
        .any(|part| part.as_os_str() == ".gomad-measure")
    {
        return true;
    }
    path.ancestors()
        .find_map(|ancestor| std::fs::canonicalize(ancestor).ok())
        .is_some_and(|canonical| {
            canonical
                .components()
                .any(|part| part.as_os_str() == ".gomad-measure")
        })
}

fn retention_base(path: &Path, create_missing: bool) -> Result<(PathBuf, File), String> {
    use std::os::unix::fs::MetadataExt;
    validate_absolute(path)?;
    if create_missing && !path.try_exists().map_err(|error| error.to_string())? {
        std::fs::create_dir_all(path).map_err(|error| error.to_string())?;
    }
    let before = std::fs::symlink_metadata(path).map_err(|error| error.to_string())?;
    if !before.is_dir() {
        return Err("refused: measurement root is not a real directory".into());
    }
    let canonical = std::fs::canonicalize(path).map_err(|error| error.to_string())?;
    let root = directory(&canonical, false)?;
    let opened = root.metadata().map_err(|error| error.to_string())?;
    if before.dev() != opened.dev() || before.ino() != opened.ino() {
        return Err("measurement output root changed".into());
    }
    Ok((canonical, root))
}

fn directory(path: &Path, create_missing: bool) -> Result<File, String> {
    validate_absolute(path)?;
    let mut root = File::open("/").map_err(|error| error.to_string())?;
    for part in path.components() {
        if let Component::Normal(part) = part {
            root = child_directory(&root, part, create_missing)?;
        }
    }
    Ok(root)
}

fn child_directory(root: &File, name: &OsStr, create_missing: bool) -> Result<File, String> {
    if create_missing {
        match fs::mkdirat(root, name, Mode::from_raw_mode(0o700)) {
            Ok(()) => root.sync_all().map_err(|error| error.to_string())?,
            Err(rustix::io::Errno::EXIST) => {}
            Err(error) => return Err(error.to_string()),
        }
    }
    open(root, name, true)
}

fn open(root: &File, name: &OsStr, directory: bool) -> Result<File, String> {
    let before =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    let expected = if directory {
        FileType::Directory
    } else {
        FileType::RegularFile
    };
    if FileType::from_raw_mode(before.st_mode) != expected {
        return Err("refused: retention entry is not a real file or directory".into());
    }
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
    if (name == OsStr::new("latest.json")
        && (before.st_nlink != 1 || opened.st_nlink != 1 || after.st_nlink != 1))
        || before.st_ino != opened.st_ino
        || before.st_dev != opened.st_dev
        || after.st_ino != opened.st_ino
        || after.st_dev != opened.st_dev
    {
        return Err("retention entry changed while opening".into());
    }
    Ok(file)
}

fn same(a: &File, b: &File) -> Result<bool, String> {
    let a = fs::fstat(a).map_err(|error| error.to_string())?;
    let b = fs::fstat(b).map_err(|error| error.to_string())?;
    Ok(a.st_ino == b.st_ino && a.st_dev == b.st_dev)
}

fn create(root: &File, name: &str) -> Result<File, String> {
    fs::openat(
        root,
        name,
        OFlags::RDWR | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
        Mode::from_raw_mode(0o600),
    )
    .map(File::from)
    .map_err(|error| error.to_string())
}

fn atomic_write(root: &File, name: &str, data: &[u8]) -> Result<(), String> {
    let stage = format!(".stage-{}", unique());
    let result = (|| {
        let mut file = create(root, &stage)?;
        file.write_all(data).map_err(|error| error.to_string())?;
        file.sync_all().map_err(|error| error.to_string())?;
        fs::renameat(root, stage.as_str(), root, name).map_err(|error| error.to_string())?;
        root.sync_all().map_err(|error| error.to_string())
    })();
    match fs::unlinkat(root, stage.as_str(), AtFlags::empty()) {
        Ok(()) | Err(rustix::io::Errno::NOENT) => {}
        Err(error) => return Err(error.to_string()),
    }
    result
}

fn rename_exclusive(root: &File, from: &str, to: &str) -> Result<(), String> {
    fs::renameat_with(root, from, root, to, fs::RenameFlags::NOREPLACE)
        .map_err(|error| error.to_string())
}

fn store_lock(root: &File, cancelled: &dyn Fn() -> bool) -> Result<Locked, String> {
    let file = Locked::new(match create(root, ".lock") {
        Ok(file) => file,
        Err(_) => open(root, OsStr::new(".lock"), false)?,
    });
    let deadline = Instant::now() + LOCK_TIMEOUT;
    loop {
        check_cancelled(cancelled)?;
        match fs::flock(&*file, FlockOperation::NonBlockingLockExclusive) {
            Ok(()) => {
                let current = open(root, OsStr::new(".lock"), false)?;
                if !same(&file, &current)? {
                    return Err("measurement store lock changed".into());
                }
                return Ok(file);
            }
            Err(rustix::io::Errno::WOULDBLOCK) => {}
            Err(error) => return Err(error.to_string()),
        }
        if Instant::now() >= deadline {
            return Err("measurement store lock timed out after 60 seconds".into());
        }
        std::thread::sleep(Duration::from_millis(1));
    }
}

fn check_cancelled(cancelled: &dyn Fn() -> bool) -> Result<(), String> {
    if cancelled() {
        Err("measurement cancelled".into())
    } else {
        Ok(())
    }
}

fn entries(root: &File) -> Result<Vec<OsString>, String> {
    fs::Dir::read_from(root)
        .map_err(|error| error.to_string())?
        .filter_map(|entry| match entry {
            Ok(entry)
                if entry.file_name().to_bytes() == b"."
                    || entry.file_name().to_bytes() == b".." =>
            {
                None
            }
            Ok(entry) => Some(Ok(OsString::from_vec(
                entry.file_name().to_bytes().to_vec(),
            ))),
            Err(error) => Some(Err(error.to_string())),
        })
        .collect()
}

fn read_pin(root: &File, name: &OsStr) -> Result<PathBuf, String> {
    let file = open(root, name, false)?;
    let mut bytes = Vec::new();
    file.take(8193)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() > 8192 {
        return Err("invalid retention pin size".into());
    }
    Ok(PathBuf::from(OsString::from_vec(bytes)))
}

fn validate_pin(root: &File, path: &Path) -> Result<(), String> {
    if read_pin(root, OsStr::new(".pin"))? != path {
        return Err("retention run identity changed".into());
    }
    Ok(())
}

fn managed_run(path: &Path) -> bool {
    path.file_name()
        .and_then(OsStr::to_str)
        .is_some_and(|name| name.starts_with("run-"))
        && path
            .parent()
            .and_then(Path::parent)
            .and_then(Path::file_name)
            == Some(OsStr::new(".gomad-measure"))
}

fn latest(root: &File) -> Result<Option<String>, String> {
    if !exists(root, "latest.json")? {
        return Ok(None);
    }
    let file = open(root, OsStr::new("latest.json"), false)?;
    let mut bytes = Vec::new();
    file.take(1025)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    let name: String = serde_json::from_slice(&bytes)
        .map_err(|error| format!("invalid latest measurement pointer: {error}"))?;
    if bytes.len() > 1024 || !single_name(&name) || !name.starts_with("run-") {
        return Err("invalid latest measurement pointer".into());
    }
    Ok(Some(name))
}

fn exists(root: &File, name: &str) -> Result<bool, String> {
    match fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW) {
        Ok(_) => Ok(true),
        Err(rustix::io::Errno::NOENT) => Ok(false),
        Err(error) => Err(error.to_string()),
    }
}

fn sync_tree(root: &File, cancelled: &dyn Fn() -> bool) -> Result<(), String> {
    for name in entries(root)? {
        check_cancelled(cancelled)?;
        let info = fs::statat(root, &name, AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        match FileType::from_raw_mode(info.st_mode) {
            FileType::Symlink => {}
            FileType::Directory => sync_tree(&open(root, &name, true)?, cancelled)?,
            FileType::RegularFile => open(root, &name, false)?
                .sync_all()
                .map_err(|error| error.to_string())?,
            _ => return Err("nonregular measurement output".into()),
        }
    }
    check_cancelled(cancelled)?;
    root.sync_all().map_err(|error| error.to_string())
}

fn remove_tree(root: &File, name: &OsStr) -> Result<(), String> {
    let info =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    let directory = FileType::from_raw_mode(info.st_mode) == FileType::Directory;
    if directory {
        let child = open(root, name, true)?;
        for entry in entries(&child)? {
            remove_tree(&child, &entry)?;
        }
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

fn dependencies(root: &File) -> Result<Vec<PathBuf>, String> {
    if !exists(root, ".pins")? {
        return Ok(Vec::new());
    }
    let pins = child_directory(root, OsStr::new(".pins"), false)?;
    let mut paths = Vec::new();
    for name in entries(&pins)? {
        let path = read_pin(&pins, &name)?;
        validate_absolute(&path)?;
        if !managed_run(&path) {
            return Err("invalid retained measurement input path".into());
        }
        paths.push(path);
    }
    Ok(paths)
}

#[cfg(test)]
fn reclaim(path: &Path) -> Result<(), String> {
    reclaim_cancellable(path, &|| false)
}

fn reclaim_cancellable(path: &Path, cancelled: &dyn Fn() -> bool) -> Result<(), String> {
    let deadline = Instant::now() + LOCK_TIMEOUT;
    let mut pending = VecDeque::from([path.to_path_buf()]);
    while let Some(path) = pending.pop_front() {
        check_cancelled(cancelled)?;
        let store = directory(&path, false)?;
        let lock = store_lock(&store, &|| cancelled() || Instant::now() >= deadline)?;
        let (trash, released) = collect(&store, &path)?;
        store.sync_all().map_err(|error| error.to_string())?;
        drop(lock);
        for name in trash {
            check_cancelled(cancelled)?;
            remove_tree(&store, OsStr::new(&name))?;
        }
        store.sync_all().map_err(|error| error.to_string())?;
        pending.extend(released);
    }
    Ok(())
}

fn collect(store: &File, path: &Path) -> Result<(Vec<String>, Vec<PathBuf>), String> {
    let latest = latest(store)?;
    let mut ordered = Vec::new();
    for name in entries(store)? {
        let info = fs::statat(store, &name, AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        ordered.push((name, info));
    }
    ordered.sort_by(|(a, ai), (b, bi)| {
        (bi.st_mtime, bi.st_mtime_nsec)
            .cmp(&(ai.st_mtime, ai.st_mtime_nsec))
            .then_with(|| a.cmp(b))
    });
    let mut trash = Vec::new();
    let mut released = Vec::new();
    let mut failed_kept = false;
    for (name, info) in ordered {
        let Some(name) = name.to_str() else {
            continue;
        };
        let is_directory = FileType::from_raw_mode(info.st_mode) == FileType::Directory;
        if is_directory && (name.starts_with(".stage-run-") || name.starts_with(".trash-")) {
            // Staging creation is protected by the store lock; a stage here is abandoned.
            let abandoned = child_directory(store, OsStr::new(name), false)?;
            let inputs = dependencies(&abandoned)?;
            released.push(path.to_path_buf());
            released.extend(
                inputs
                    .into_iter()
                    .filter_map(|input| input.parent().map(Path::to_path_buf)),
            );
            let claimed = format!(".trash-{}", unique());
            rename_exclusive(store, name, &claimed)?;
            trash.push(claimed);
            continue;
        }
        if !is_directory && name.starts_with(".stage-") {
            fs::unlinkat(store, name, AtFlags::empty()).map_err(|error| error.to_string())?;
            continue;
        }
        if !is_directory || !name.starts_with("run-") || latest.as_deref() == Some(name) {
            continue;
        }
        let root = child_directory(store, OsStr::new(name), false)?;
        if validate_pin(&root, &path.join(name)).is_err() {
            continue;
        }
        let lease = Locked::new(open(&root, OsStr::new(".lease"), false)?);
        match fs::flock(&*lease, FlockOperation::NonBlockingLockExclusive) {
            Ok(()) => {}
            Err(rustix::io::Errno::WOULDBLOCK) => continue,
            Err(error) => return Err(error.to_string()),
        }
        let children = if exists(&root, ".children")? {
            let children = Locked::new(open(&root, OsStr::new(".children"), false)?);
            match fs::flock(&*children, FlockOperation::NonBlockingLockExclusive) {
                Ok(()) => Some(children),
                Err(rustix::io::Errno::WOULDBLOCK) => continue,
                Err(error) => return Err(error.to_string()),
            }
        } else {
            None
        };
        if exists(&root, ".keep")? {
            continue;
        }
        let pin = open(&root, OsStr::new(".pin"), false)?;
        if fs::fstat(&pin).map_err(|error| error.to_string())?.st_nlink != 1 {
            continue;
        }
        if !exists(&root, ".complete")? && !failed_kept {
            failed_kept = true;
            continue;
        }
        let inputs = dependencies(&root)?;
        let claimed = format!(".trash-{}", unique());
        rename_exclusive(store, name, &claimed)?;
        drop(children);
        trash.push(claimed);
        released.push(path.to_path_buf());
        released.extend(
            inputs
                .into_iter()
                .filter_map(|input| input.parent().map(Path::to_path_buf)),
        );
    }
    Ok((trash, released))
}

#[cfg(test)]
#[path = "retention_tests.rs"]
mod tests;

#[cfg(test)]
#[path = "retention_process_tests.rs"]
mod process_tests;
