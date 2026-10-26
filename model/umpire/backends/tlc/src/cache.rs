use crate::{Limits, Refusal};
use std::{
    collections::BTreeMap,
    fmt::Write,
    fs,
    os::unix::fs::PermissionsExt,
    path::{Component, Path, PathBuf},
    sync::Mutex,
};
use umpire_backend_process::{Scratch, bounded_text, bytes_sha256, structural_sha256};

pub(super) const VERSION: &str = "tlc-raw-v1-runner1-decoder1-batch1-fp0-serialgc-diskfp";
const MAX_MANIFEST: u64 = 16 * 1024;

#[derive(Clone)]
pub(super) struct Record {
    pub output: String,
    pub dump: String,
}

pub(super) struct Bundle {
    pub key: String,
    pub schema: String,
    pub root: PathBuf,
    pub expected: Vec<String>,
    pub records: Mutex<BTreeMap<String, Record>>,
    pub replay: bool,
    pub limits: Limits,
}

pub(super) fn enabled(quick: Option<&str>, disable: Option<&str>) -> bool {
    disable != Some("1") && quick == Some("1") && !cfg!(feature = "strategy-equivalence")
}

pub(super) fn root() -> Result<PathBuf, Refusal> {
    let root = std::env::var_os("UMPIRE_QUICK_CACHE_ROOT")
        .map(PathBuf::from)
        .ok_or_else(|| Refusal("quick cache requires UMPIRE_QUICK_CACHE_ROOT".into()))?;
    admit_root(&root)
}

fn admit_root(root: &Path) -> Result<PathBuf, Refusal> {
    if !root.is_absolute() {
        return Err(Refusal("quick cache root must be absolute".into()));
    }
    let mut current = PathBuf::new();
    for component in root.components() {
        match component {
            Component::RootDir | Component::Normal(_) => current.push(component),
            _ => return Err(Refusal("invalid quick cache root".into())),
        }
        match fs::symlink_metadata(&current) {
            Ok(meta) if meta.is_dir() && !meta.is_symlink() => {}
            Ok(_) => return Err(Refusal("unsafe quick cache root".into())),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                match fs::create_dir(&current) {
                    Ok(()) => {}
                    Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                        let meta = fs::symlink_metadata(&current)?;
                        if !meta.is_dir() || meta.is_symlink() {
                            return Err(Refusal("unsafe quick cache root".into()));
                        }
                    }
                    Err(error) => return Err(error.into()),
                }
            }
            Err(error) => return Err(error.into()),
        }
    }
    Ok(root.to_path_buf())
}

impl Bundle {
    pub(super) fn load(&mut self) -> Result<(), Refusal> {
        let directory = self.root.join("quick-tlc-v1").join(&self.key);
        regular_directory(&directory)?;
        let manifest = regular_text(&directory.join("manifest"), MAX_MANIFEST)?;
        if !manifest.is_ascii() || !manifest.ends_with('\n') {
            return Err(Refusal("invalid cache manifest encoding".into()));
        }
        let mut lines = manifest.lines();
        if lines.next() != Some(VERSION)
            || lines.next() != Some(self.key.as_str())
            || lines.next() != Some(self.schema.as_str())
            || lines.next() != Some("complete-success")
            || lines.next() != Some(self.expected.len().to_string().as_str())
        {
            return Err(Refusal("cache identity or completeness mismatch".into()));
        }
        let mut records = BTreeMap::new();
        let mut total = 0_u64;
        let maximum = (self.expected.len() as u64)
            .checked_mul(
                self.limits
                    .output_bytes
                    .checked_add(self.limits.dump_bytes)
                    .ok_or_else(|| Refusal("cache byte budget overflow".into()))?,
            )
            .ok_or_else(|| Refusal("cache total budget overflow".into()))?;
        for (index, expected) in self.expected.iter().enumerate() {
            let fields: Vec<_> = lines.next().unwrap_or("").split(' ').collect();
            if fields.len() != 6 || fields[0] != expected || fields[5] != "0" {
                return Err(Refusal("cache record inventory mismatch".into()));
            }
            let length = |text: &str, limit| -> Result<u64, Refusal> {
                let size = text.parse::<u64>().map_err(|_| Refusal("cache length".into()))?;
                if size > limit {
                    return Err(Refusal("cache artifact limit".into()));
                }
                Ok(size)
            };
            let out_size = length(fields[1], self.limits.output_bytes)?;
            let dump_size = length(fields[3], self.limits.dump_bytes)?;
            total = total
                .checked_add(out_size)
                .and_then(|n| n.checked_add(dump_size))
                .ok_or_else(|| Refusal("cache size overflow".into()))?;
            if total > maximum {
                return Err(Refusal("cache total limit".into()));
            }
            let output = regular_text(&directory.join(format!("{index}.out")), out_size)?;
            let dump = regular_text(&directory.join(format!("{index}.dump")), dump_size)?;
            if output.len() as u64 != out_size
                || dump.len() as u64 != dump_size
                || bytes_sha256(output.as_bytes()) != fields[2]
                || bytes_sha256(dump.as_bytes()) != fields[4]
            {
                return Err(Refusal("cache artifact integrity mismatch".into()));
            }
            records.insert(expected.clone(), Record { output, dump });
        }
        if lines.next().is_some()
            || fs::read_dir(&directory)?.count() != 1 + self.expected.len() * 2
        {
            return Err(Refusal("cache extra artifacts or metadata".into()));
        }
        self.records = Mutex::new(records);
        self.replay = true;
        Ok(())
    }

    pub(super) fn publish(&self) -> Result<(), Refusal> {
        let records = self.records.lock().map_err(|_| Refusal("cache poisoned".into()))?;
        if records.len() != self.expected.len() {
            return Ok(());
        }
        let staging = Scratch::new_in(&self.root, "quick-tlc-v1")?;
        let result = (|| {
            let directory = staging.path().join("bundle");
            fs::create_dir(&directory)?;
            fs::set_permissions(&directory, fs::Permissions::from_mode(0o700))?;
            let mut manifest = format!(
                "{VERSION}\n{}\n{}\ncomplete-success\n{}\n",
                self.key,
                self.schema,
                self.expected.len()
            );
            for (index, id) in self.expected.iter().enumerate() {
                let record =
                    records.get(id).ok_or_else(|| Refusal("cache inventory incomplete".into()))?;
                if record.output.len() as u64 > self.limits.output_bytes
                    || record.dump.len() as u64 > self.limits.dump_bytes
                {
                    return Ok(());
                }
                fs::write(directory.join(format!("{index}.out")), &record.output)?;
                fs::write(directory.join(format!("{index}.dump")), &record.dump)?;
                writeln!(
                    manifest,
                    "{id} {} {} {} {} 0",
                    record.output.len(),
                    bytes_sha256(record.output.as_bytes()),
                    record.dump.len(),
                    bytes_sha256(record.dump.as_bytes())
                )
                .map_err(|_| Refusal("cache manifest encoding".into()))?;
            }
            if manifest.len() as u64 > MAX_MANIFEST {
                return Ok(());
            }
            fs::write(directory.join("manifest"), manifest)?;
            let final_path = self.root.join("quick-tlc-v1").join(&self.key);
            match fs::rename(&directory, &final_path) {
                Ok(()) => {}
                Err(error) if final_path.exists() => {
                    // An immutable winner is used only after a subsequent complete load.
                    let _ = error;
                }
                Err(error) => return Err(error.into()),
            }
            Ok(())
        })();
        staging.finish(result)
    }
}

fn regular_directory(path: &Path) -> Result<(), Refusal> {
    let metadata = fs::symlink_metadata(path)?;
    if !metadata.is_dir() || metadata.is_symlink() {
        return Err(Refusal("unsafe cache directory".into()));
    }
    Ok(())
}
fn regular_text(path: &Path, limit: u64) -> Result<String, Refusal> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.is_file() || meta.is_symlink() || meta.len() > limit {
        return Err(Refusal("unsafe or oversized cache artifact".into()));
    }
    bounded_text(path, limit)
}

pub(super) fn record_id(cfg: &str) -> String {
    structural_sha256(&cfg)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn root_admission_rejects_symlinks_and_relative_paths() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let result = (|| {
            assert!(admit_root(Path::new("relative/cache")).is_err());
            let outside = scratch.path().join("outside");
            fs::create_dir(&outside)?;
            let link = scratch.path().join("link");
            std::os::unix::fs::symlink(&outside, &link)?;
            assert!(admit_root(&link.join("child")).is_err());
            assert!(!outside.join("child").exists());
            Ok(())
        })();
        scratch.finish(result)
    }
}
