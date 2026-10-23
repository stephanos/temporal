//! Private raw compiler observations; never part of the execution identity.
use super::{canonical, process};
use crate::evidence::store::open;
use rustix::fs::{self as descriptor, AtFlags, Mode, OFlags, RenameFlags};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs::{self, File},
    io::Write,
    os::unix::fs::DirBuilderExt as _,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

const PREFIX: &str = "target/preparation-inputs";

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Blob {
    path: PathBuf,
    bytes: u64,
    sha256: String,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Command {
    name: String,
    argv: Vec<String>,
    cwd: PathBuf,
    environment: BTreeMap<String, String>,
    owner_pid: u32,
    pid: u32,
    start_ns: u64,
    end_ns: u64,
    exit: i32,
    output_limit: u64,
    stdout: String,
    stderr: String,
}

#[derive(Clone, Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Observations {
    commands: Vec<Command>,
    blobs: BTreeMap<String, Blob>,
    snapshots: BTreeMap<String, String>,
}

fn blob(cache: &Path, raw: &[u8]) -> Result<Blob, String> {
    let directory = cache.join("preparation-observations-v1");
    match fs::DirBuilder::new().mode(0o700).create(&directory) {
        Ok(()) => (),
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(error) => return Err(error.to_string()),
    }
    let root = open::pin(&directory)?;
    let hash = canonical::digest(raw);
    if descriptor::statat(&root, hash.as_str(), AtFlags::SYMLINK_NOFOLLOW).is_ok() {
        if open::read(&root, &hash, raw.len() as u64, false)? != raw {
            return Err("corrupt private preparation observation blob".into());
        }
    } else {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let stage = format!(
            ".observation-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        );
        let operation = || -> Result<(), String> {
            let mut file = File::from(
                descriptor::openat(
                    &root,
                    stage.as_str(),
                    OFlags::WRONLY
                        | OFlags::CREATE
                        | OFlags::EXCL
                        | OFlags::NOFOLLOW
                        | OFlags::CLOEXEC,
                    Mode::from_raw_mode(0o600),
                )
                .map_err(|e| e.to_string())?,
            );
            file.write_all(raw)
                .and_then(|()| file.sync_all())
                .map_err(|e| e.to_string())?;
            match descriptor::renameat_with(
                &root,
                stage.as_str(),
                &root,
                hash.as_str(),
                RenameFlags::NOREPLACE,
            ) {
                Ok(()) => root.sync_all().map_err(|e| e.to_string())?,
                Err(error) if error == rustix::io::Errno::EXIST => {
                    if open::read(&root, &hash, raw.len() as u64, false)? != raw {
                        return Err("competing corrupt preparation observation blob".into());
                    }
                }
                Err(error) => return Err(error.to_string()),
            }
            Ok(())
        };
        let result = operation();
        let cleanup = descriptor::unlinkat(&root, stage.as_str(), AtFlags::empty());
        match (result, cleanup) {
            (Err(primary), Err(cleanup)) if cleanup != rustix::io::Errno::NOENT => {
                return Err(format!(
                    "{primary}; preparation observation cleanup: {cleanup}"
                ));
            }
            (Err(primary), _) => return Err(primary),
            (Ok(()), Err(cleanup)) if cleanup != rustix::io::Errno::NOENT => {
                return Err(format!("preparation observation cleanup: {cleanup}"));
            }
            _ => (),
        }
    }
    Ok(Blob {
        path: directory.join(&hash),
        bytes: raw.len() as u64,
        sha256: hash,
    })
}

impl Observations {
    pub(crate) fn record(
        &mut self,
        name: &str,
        command: process::ObservedCommand,
        cache: &Path,
    ) -> Result<Vec<u8>, String> {
        if self.commands.iter().any(|row| row.name == name) {
            return Err("duplicate private compiler observation".into());
        }
        let stdout = blob(cache, &command.stdout)?;
        let stderr = blob(cache, &command.stderr)?;
        let row = Command {
            name: name.into(),
            argv: command.argv,
            cwd: command.cwd,
            environment: command.environment,
            owner_pid: command.owner_pid,
            pid: command.pid,
            start_ns: command.start_ns,
            end_ns: command.end_ns,
            exit: command.exit,
            output_limit: command.output_limit,
            stdout: stdout.sha256.clone(),
            stderr: stderr.sha256.clone(),
        };
        self.blobs.insert(stdout.sha256.clone(), stdout);
        self.blobs.insert(stderr.sha256.clone(), stderr);
        self.commands.push(row);
        Ok(command.stdout)
    }

    pub(crate) fn snapshot(&mut self, name: &str, raw: &[u8], cache: &Path) -> Result<(), String> {
        if self.snapshots.contains_key(name) {
            return Err("duplicate private preparation snapshot".into());
        }
        let value = blob(cache, raw)?;
        self.snapshots.insert(name.into(), value.sha256.clone());
        self.blobs.insert(value.sha256.clone(), value);
        Ok(())
    }

    pub(crate) fn payloads(&self, phase: &str) -> Result<BTreeMap<String, Vec<u8>>, String> {
        if !matches!(phase, "compile" | "admission") {
            return Err("invalid preparation observation phase".into());
        }
        let mut payloads = BTreeMap::new();
        let mut blobs = BTreeMap::new();
        for (hash, blob) in &self.blobs {
            if hash != &blob.sha256 || !canonical::valid_digest(hash) {
                return Err("invalid observation blob identity".into());
            }
            let parent = blob
                .path
                .parent()
                .ok_or("preparation blob parent missing")?;
            if blob.path.file_name().and_then(|v| v.to_str()) != Some(hash) {
                return Err("preparation blob filename mismatch".into());
            }
            let raw = open::read(&open::pin(parent)?, hash, blob.bytes, false)?;
            if raw.len() as u64 != blob.bytes || canonical::digest(&raw) != *hash {
                return Err("changed preparation observation blob".into());
            }
            let path = format!("{PREFIX}/{phase}/{hash}.bin");
            blobs.insert(
                hash,
                serde_json::json!({"path":path,"bytes":blob.bytes,"sha256":hash}),
            );
            payloads.insert(path, raw);
        }
        let metadata = serde_json::json!({"schema":"gomad.native-preparation-observations/v1","phase":phase,"commands":self.commands,"snapshots":self.snapshots,"blobs":blobs});
        payloads.insert(
            format!("{PREFIX}/{phase}/metadata.json"),
            canonical::encode(&metadata)?,
        );
        Ok(payloads)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    struct Root(PathBuf);
    impl Root {
        fn new() -> Self {
            static NEXT: AtomicU64 = AtomicU64::new(0);
            let path = std::env::temp_dir().join(format!(
                "gomad-raw-preparation-{}-{}",
                std::process::id(),
                NEXT.fetch_add(1, Ordering::Relaxed)
            ));
            fs::DirBuilder::new().mode(0o700).create(&path).unwrap();
            Self(path)
        }
    }
    impl Drop for Root {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }
    fn command() -> process::ObservedCommand {
        process::ObservedCommand {
            argv: vec![
                "/compiler/go".into(),
                "list".into(),
                "-deps".into(),
                "-json".into(),
                "./probe".into(),
            ],
            cwd: "/frozen/source".into(),
            environment: BTreeMap::from([("GOOS".into(), "wasip1".into())]),
            owner_pid: 17,
            pid: 18,
            start_ns: 1,
            end_ns: 2,
            exit: 0,
            output_limit: 128 << 20,
            stdout: b"{\"Dir\":\"/frozen/source\"}".to_vec(),
            stderr: vec![0, 255],
        }
    }
    #[test]
    fn receiving_raw_observations_preserve_bytes_and_small_private_recipe_references() {
        let root = Root::new();
        let mut observed = Observations::default();
        let raw = observed
            .record("selected-source-closure", command(), &root.0)
            .unwrap();
        assert_eq!(raw, command().stdout);
        observed
            .snapshot("compiler-base-paths", b"[]", &root.0)
            .unwrap();
        let recipe = canonical::encode(&observed).unwrap();
        let restored: Observations = serde_json::from_slice(&recipe).unwrap();
        let payloads = restored.payloads("compile").unwrap();
        assert!(payloads.values().any(|raw| raw == &vec![0, 255]));
        let metadata: serde_json::Value =
            serde_json::from_slice(&payloads[&format!("{PREFIX}/compile/metadata.json")]).unwrap();
        assert_eq!(metadata["commands"][0]["pid"], 18);
        assert_eq!(metadata["commands"][0]["owner_pid"], 17);
        assert_eq!(metadata["commands"][0]["argv"][1], "list");
        assert!(restored.payloads("unexpected").is_err());
        assert!(
            observed
                .record("selected-source-closure", command(), &root.0)
                .is_err()
        );
    }
    #[test]
    fn receiving_raw_observation_reopen_refuses_tamper_symlink_and_unknown_fields() {
        let root = Root::new();
        let mut observed = Observations::default();
        observed
            .record("selected-source-closure", command(), &root.0)
            .unwrap();
        let blob = observed.blobs.values().next().unwrap().clone();
        fs::write(&blob.path, b"changed").unwrap();
        assert!(observed.payloads("compile").is_err());
        fs::remove_file(&blob.path).unwrap();
        std::os::unix::fs::symlink("/missing", &blob.path).unwrap();
        assert!(observed.payloads("compile").is_err());
        let mut value = serde_json::to_value(&observed).unwrap();
        value["unknown"] = serde_json::json!(true);
        assert!(serde_json::from_value::<Observations>(value).is_err());
        assert!(
            fs::read_dir(root.0.join("preparation-observations-v1"))
                .unwrap()
                .all(|entry| !entry
                    .unwrap()
                    .file_name()
                    .to_string_lossy()
                    .starts_with('.'))
        );
    }
}
