use super::{Completed, Registration};
use rustix::fs::{self, AtFlags, Mode, OFlags};
use std::{
    collections::BTreeMap,
    fs::File,
    io::{Read, Write},
    os::unix::fs::MetadataExt,
    path::{Path, PathBuf},
    sync::Mutex,
};

pub(super) struct Registry {
    pub path: PathBuf,
    directory: File,
    pids: Mutex<BTreeMap<u32, (u64, u64)>>,
    pub degraded: std::sync::atomic::AtomicBool,
    pub intervals: Mutex<BTreeMap<u32, super::OwnedInterval>>,
}
impl Registry {
    pub fn create(path: &Path) -> Result<Self, String> {
        use std::os::unix::fs::DirBuilderExt;
        if !path.is_absolute()
            || path.components().any(|part| {
                matches!(
                    part,
                    std::path::Component::ParentDir | std::path::Component::CurDir
                )
            })
        {
            return Err("registry path must be absolute and clean".into());
        }
        let parent = path.parent().ok_or("registry parent missing")?;
        if parent.canonicalize().map_err(|e| e.to_string())? != parent {
            return Err("registry parent is not canonical".into());
        }
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(path)
            .map_err(|e| format!("fresh process registry required: {e}"))?;
        let directory = File::from(
            fs::open(
                path,
                OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                Mode::empty(),
            )
            .map_err(|e| e.to_string())?,
        );
        let registry = Self {
            path: path.to_owned(),
            directory,
            pids: Mutex::new(BTreeMap::new()),
            intervals: Mutex::new(BTreeMap::new()),
            degraded: std::sync::atomic::AtomicBool::new(false),
        };
        registry.check()?;
        Ok(registry)
    }
    pub fn check(&self) -> Result<(), String> {
        let actual = self.directory.metadata().map_err(|e| e.to_string())?;
        let named = std::fs::symlink_metadata(&self.path).map_err(|e| e.to_string())?;
        if !named.is_dir()
            || actual.dev() != named.dev()
            || actual.ino() != named.ino()
            || actual.mode() & 0o777 != 0o700
            || actual.uid() != rustix::process::getuid().as_raw()
            || self.path.canonicalize().map_err(|e| e.to_string())? != self.path
        {
            return Err("private process registry identity changed".into());
        }
        Ok(())
    }
    pub fn create_file(&self, name: &str) -> Result<File, String> {
        self.check()?;
        fs::openat(
            &self.directory,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        )
        .map(File::from)
        .map_err(|e| e.to_string())
    }
    pub fn register(&self, registration: &Registration) -> Result<(), String> {
        self.check()?;
        {
            let mut pids = self.pids.lock().map_err(|_| "PID registry poisoned")?;
            if registration.pid == 0 || pids.contains_key(&registration.pid) {
                return Err("duplicate/invalid process PID registration".into());
            }
            pids.insert(registration.pid, (0, 0));
        }
        let result = self
            .create_file(&format!("{}.process.json", registration.name))
            .and_then(|mut file| {
                serde_json::to_writer(&mut file, registration).map_err(|e| e.to_string())?;
                file.write_all(b"\n").map_err(|e| e.to_string())?;
                file.sync_all().map_err(|e| e.to_string())?;
                let metadata = file.metadata().map_err(|e| e.to_string())?;
                self.pids
                    .lock()
                    .map_err(|_| "PID registry poisoned")?
                    .insert(registration.pid, (metadata.dev(), metadata.ino()));
                self.directory.sync_all().map_err(|e| e.to_string())
            });
        if result.is_ok()
            && let Some(interval) = &registration.interval
        {
            self.intervals
                .lock()
                .map_err(|_| "interval registry poisoned")?
                .insert(registration.pid, interval.clone());
        }
        result
    }
    pub fn verify(&self, registration: &Registration) -> Result<(), String> {
        self.check()?;
        let file = File::from(
            fs::openat(
                &self.directory,
                format!("{}.process.json", registration.name),
                OFlags::RDONLY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                Mode::empty(),
            )
            .map_err(|e| e.to_string())?,
        );
        let metadata = file.metadata().map_err(|e| e.to_string())?;
        let identity = self
            .pids
            .lock()
            .map_err(|_| "PID registry poisoned")?
            .get(&registration.pid)
            .copied()
            .ok_or("unregistered process PID")?;
        if (metadata.dev(), metadata.ino()) != identity
            || metadata.mode() & 0o777 != 0o600
            || metadata.nlink() != 1
        {
            return Err("process registration identity changed".into());
        }
        let mut bytes = Vec::new();
        file.take(1024 * 1024 + 1)
            .read_to_end(&mut bytes)
            .map_err(|e| e.to_string())?;
        let mut expected = serde_json::to_vec(registration).map_err(|e| e.to_string())?;
        expected.push(b'\n');
        if bytes != expected {
            return Err("process registration changed before admission".into());
        }
        Ok(())
    }
    pub fn complete(
        &self,
        registration: &Registration,
        completed: &Completed,
    ) -> Result<(), String> {
        let mut value = serde_json::to_value(registration).map_err(|e| e.to_string())?;
        value["active"] = serde_json::Value::Bool(!completed.cleanup_complete);
        value["completed"] = serde_json::to_value(completed).map_err(|e| e.to_string())?;
        let temporary = format!("{}.process.terminal", registration.name);
        let mut file = self.create_file(&temporary)?;
        serde_json::to_writer(&mut file, &value).map_err(|e| e.to_string())?;
        file.sync_all().map_err(|e| e.to_string())?;
        fs::renameat(
            &self.directory,
            temporary.as_str(),
            &self.directory,
            format!("{}.process.json", registration.name),
        )
        .map_err(|e| e.to_string())?;
        self.directory.sync_all().map_err(|e| e.to_string())?;
        if completed.cleanup_complete {
            self.intervals
                .lock()
                .map_err(|_| "interval registry poisoned")?
                .remove(&registration.pid);
        }
        Ok(())
    }
    pub fn exists(&self, name: &str) -> Result<bool, String> {
        self.check()?;
        match fs::statat(
            &self.directory,
            format!("{name}.process.json"),
            AtFlags::SYMLINK_NOFOLLOW,
        ) {
            Ok(_) => Ok(true),
            Err(rustix::io::Errno::NOENT) => Ok(false),
            Err(e) => Err(e.to_string()),
        }
    }
}
