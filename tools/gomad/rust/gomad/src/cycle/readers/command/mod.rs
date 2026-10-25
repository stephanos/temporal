#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle reader routing belongs to B15")
)]

mod identity;
mod preparation;
mod result;
#[cfg(test)]
mod tests;

use crate::{
    cycle::{
        config::Configured,
        custody::{self, digest},
    },
    evidence::{
        canonical::encode as canonical,
        store::{self, Opened, open},
    },
};
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::File,
    os::unix::fs::MetadataExt,
    path::{Path, PathBuf},
};

fn require(condition: bool, reason: &str) -> Result<(), String> {
    if condition {
        Ok(())
    } else {
        Err(reason.into())
    }
}
fn text<'a>(value: &'a Value, field: &str) -> Result<&'a str, String> {
    value
        .get(field)
        .and_then(Value::as_str)
        .ok_or_else(|| format!("missing text: {field}"))
}
fn unsigned(value: &Value, field: &str) -> Result<u64, String> {
    value
        .get(field)
        .and_then(Value::as_u64)
        .ok_or_else(|| format!("missing unsigned integer: {field}"))
}
fn argument<'a>(argv: &'a [String], flag: &str) -> Result<&'a str, String> {
    let indices = argv
        .iter()
        .enumerate()
        .filter(|(_, v)| v.as_str() == flag)
        .map(|(i, _)| i)
        .collect::<Vec<_>>();
    require(indices.len() == 1, "missing/duplicate command argument")?;
    argv.get(indices[0] + 1)
        .map(String::as_str)
        .ok_or_else(|| "missing command argument value".into())
}

pub(crate) struct RetainedCommand {
    pub row: Value,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}

pub(crate) struct RetainedArtifact {
    pub manifest: store::Manifest,
    pub payloads: BTreeMap<String, Vec<u8>>,
}

pub(super) struct PrivateInputAuthority {
    pub original: PathBuf,
    pub sha256: String,
    pub detached: bool,
}

pub(crate) struct RetainedSession {
    config: Value,
    root: PathBuf,
    pinned: File,
    preparations: BTreeMap<PathBuf, preparation::PreparedObservation>,
    pending_preparations: Vec<preparation::Pending>,
    build_roles: BTreeMap<PathBuf, Value>,
    private_inputs: BTreeMap<String, PrivateInputAuthority>,
    private_input_roots: BTreeMap<PathBuf, PathBuf>,
}
impl RetainedSession {
    pub(crate) fn new(config: &Configured) -> Result<Self, String> {
        config.recheck()?;
        Self::from_value(config.value().clone())
    }
    fn from_value(config: Value) -> Result<Self, String> {
        let root = PathBuf::from(text(&config, "root")?);
        custody::clean_absolute(&root)?;
        require(
            root.canonicalize().map_err(|e| e.to_string())? == root,
            "attempt root aliases another directory",
        )?;
        let pinned = open::pin(&root)?;
        Ok(Self {
            config,
            root,
            pinned,
            preparations: BTreeMap::new(),
            pending_preparations: Vec::new(),
            build_roles: BTreeMap::new(),
            private_inputs: BTreeMap::new(),
            private_input_roots: BTreeMap::new(),
        })
    }
    fn check_root(&self) -> Result<(), String> {
        let named = open::pin(&self.root)?;
        let old = self.pinned.metadata().map_err(|e| e.to_string())?;
        let new = named.metadata().map_err(|e| e.to_string())?;
        require(
            (old.dev(), old.ino()) == (new.dev(), new.ino())
                && self.root.canonicalize().map_err(|e| e.to_string())? == self.root,
            "attempt root identity changed",
        )
    }
    pub(crate) fn contained(&self, path: &Path) -> Result<(), String> {
        custody::clean_absolute(path)?;
        require(
            path.starts_with(&self.root),
            "receiving observation outside frozen attempt",
        )?;
        self.check_root()
    }
    fn relative<'a>(&self, path: &'a Path) -> Result<&'a str, String> {
        self.contained(path)?;
        let relative = path
            .strip_prefix(&self.root)
            .map_err(|e| e.to_string())?
            .to_str()
            .ok_or("non-UTF8 receiving path")?;
        open::components(relative)?;
        Ok(relative)
    }
    pub(crate) fn read(&self, path: &Path, maximum: u64) -> Result<Vec<u8>, String> {
        let name = self.relative(path)?;
        let bytes = open::read(&self.pinned, name, maximum, false)?;
        self.check_root()?;
        Ok(bytes)
    }
    fn json(&self, path: &Path, maximum: u64) -> Result<Value, String> {
        custody::strict_json(&self.read(path, maximum)?)
    }
    fn directory(&self, path: &Path) -> Result<File, String> {
        self.contained(path)?;
        let mut directory = self.pinned.try_clone().map_err(|e| e.to_string())?;
        if path != self.root {
            for part in open::components(self.relative(path)?)? {
                directory = open::child(&directory, part, true, false)?;
            }
        }
        self.check_root()?;
        Ok(directory)
    }
    pub(crate) fn write(&self, path: &Path, expected: &[u8]) -> Result<(), String> {
        require(
            self.read(path, expected.len() as u64)? == expected,
            "retained recomputation differs",
        )
    }
    #[expect(
        clippy::too_many_lines,
        reason = "One command transaction binds independently opened process records and raw streams"
    )]
    pub(crate) fn command(
        &mut self,
        root: &Path,
        name: &str,
        expected: &crate::cycle::process::Command,
        preparation_required: bool,
    ) -> Result<RetainedCommand, String> {
        let argv = &expected.argv;
        self.contained(root)?;
        require(
            !name.is_empty() && !name.contains('/') && name != "." && name != "..",
            "unsafe command name",
        )?;
        let row = self.json(&root.join(format!("{name}.command.json")), 16 << 20)?;
        let process = self.json(&root.join(format!("{name}.process.json")), 16 << 20)?;
        require(
            row["argv"] == serde_json::to_value(argv).map_err(|e| e.to_string())?
                && row["cwd"] == serde_json::to_value(&expected.cwd).map_err(|e| e.to_string())?,
            "retained command argv/cwd mismatch",
        )?;
        let pid = row["pid"].as_u64().filter(|v| *v > 0);
        let exit = row["exit"].as_i64().and_then(|v| i32::try_from(v).ok());
        require(
            pid.is_some()
                && row["observation"] == "completed"
                && exit.is_some_and(|v| expected.accepted.contains(&v)),
            "unexecuted/failed retained completed command",
        )?;
        let start = unsigned(&row, "start_ns")?;
        let end = unsigned(&row, "end_ns")?;
        let epoch = self.config["sent_epoch"]
            .as_f64()
            .filter(|v| v.is_finite() && *v >= 0.0)
            .ok_or("invalid frozen sent epoch")?;
        #[expect(
            clippy::cast_precision_loss,
            reason = "Epoch configuration is fractional seconds, matching the configured contract"
        )]
        let valid_interval = end >= start && start as f64 >= epoch * 1_000_000_000.0;
        require(
            valid_interval,
            "retained command outside frozen send interval",
        )?;
        require(
            process["active"] == false && process["stage_pid"].as_u64().is_some_and(|v| v > 0),
            "incomplete process registration",
        )?;
        require(process == row, "retained process/command mismatch")?;
        require(
            row["resources"]["pid"] == row["pid"]
                && row["resources"]["waited_pid"] == row["pid"]
                && row["resources"]["exit_code"] == row["exit"]
                && row["resources"]["reaps"] == 1
                && row["resources"]["wait4_calls"] == 1,
            "retained PID/result differs from reaped process resources",
        )?;
        require(
            row["cleanup_complete"] == true && row["failure"].is_null(),
            "incomplete command cleanup",
        )?;
        require(
            u128::from(end - start) <= expected.timeout.as_nanos(),
            "original command bound exceeded",
        )?;
        require(
            row["accepted_exits"]
                == serde_json::to_value(&expected.accepted).map_err(|e| e.to_string())?
                && row["private_context"]
                    == serde_json::to_value(&expected.private_context)
                        .map_err(|e| e.to_string())?,
            "retained command declared contract changed",
        )?;
        let executable = argv.first().ok_or("empty command argv")?;
        let role = self.build_roles.get(Path::new(executable));
        if let Some(role) = role {
            let inputs =
                serde_json::from_value(role["inputs"].clone()).map_err(|e| e.to_string())?;
            custody::check_files(&inputs)?;
        }
        let expected = self.config["files"]
            .get(executable)
            .or_else(|| role.and_then(|role| role.get("binary_sha256")))
            .and_then(Value::as_str)
            .ok_or("retained executable is not sealed")?;
        require(
            digest(&custody::read_file(Path::new(executable), 512 << 20)?) == expected,
            "retained executable changed from frozen role",
        )?;
        let stdout_path = root.join(format!("{name}.stdout"));
        let stderr_path = root.join(format!("{name}.stderr"));
        let stdout = self.read(&stdout_path, 256 << 20)?;
        let stderr = self.read(&stderr_path, 256 << 20)?;
        require(
            row["stdout"] == stdout_path.to_string_lossy().as_ref()
                && row["stderr"] == stderr_path.to_string_lossy().as_ref()
                && row["stdout_sha256"] == digest(&stdout)
                && row["stderr_sha256"] == digest(&stderr)
                && row["command"] == row["argv"]
                && row["argv_sha256"] == digest(&canonical(&row["argv"])?),
            "raw command streams/argv changed",
        )?;
        for ancestor in root.ancestors().take_while(|p| p.starts_with(&self.root)) {
            let directory = self.directory(ancestor)?;
            if open::entries(&directory)?
                .iter()
                .any(|v| v == "producer-root.json")
            {
                let owner = self.json(&ancestor.join("producer-root.json"), 1 << 20)?;
                require(
                    process["stage_pid"] == owner["pid"] && start >= unsigned(&owner, "start_ns")?,
                    "nested command not owned by actual producer root/process",
                )?;
                break;
            }
        }
        require(
            !preparation_required
                || row
                    .get("private_observations")
                    .is_some_and(|v| !v.is_null()),
            "native command lacks actual preparation capture",
        )?;
        self.capture_preparations(&row)?;
        if argv.iter().any(|v| v == "report") && argv.iter().any(|v| v == "--result") {
            self.result(
                Path::new(argument(argv, "--result")?),
                &custody::strict_json(&stdout)?,
            )?;
        }
        if argv.iter().any(|v| v == "inspect") && argv.iter().any(|v| v == "--json") {
            let reported = custody::strict_json(&stdout)?;
            let artifact =
                self.artifact(Path::new(argv.last().ok_or("missing artifact argument")?))?;
            let manifest = serde_json::to_value(&artifact.manifest).map_err(|e| e.to_string())?;
            require(
                ["record", "files"]
                    .iter()
                    .all(|field| reported["artifact"][field] == manifest[field]),
                "retained public inspect differs from raw manifest",
            )?;
        }
        Ok(RetainedCommand {
            row,
            stdout,
            stderr,
        })
    }
    pub(crate) fn with_artifact<T>(
        &self,
        path: &Path,
        read: impl FnOnce(&Opened) -> Result<T, String>,
    ) -> Result<T, String> {
        let parent = self.directory(path.parent().ok_or("artifact parent missing")?)?;
        let pool = if open::entries(&parent)?
            .iter()
            .any(|name| name == ".store.json")
        {
            require(
                open::read(&parent, ".store.json", 4096, false)? == open::LAYOUT,
                "unsupported artifact store layout",
            )?;
            Some(open::child(&parent, ".targets", true, false)?)
        } else {
            None
        };
        let directory = self.directory(path)?;
        let before = directory.metadata().map_err(|e| e.to_string())?;
        let opened = Opened::from_root_with_pool(directory, pool.as_ref())?;
        let value = read(&opened)?;
        opened.authenticate_all()?;
        let after = self
            .directory(path)?
            .metadata()
            .map_err(|e| e.to_string())?;
        require(
            (before.dev(), before.ino()) == (after.dev(), after.ino()),
            "artifact directory identity changed during receiving",
        )?;
        self.check_root()?;
        Ok(value)
    }
    pub(crate) fn artifact(&self, path: &Path) -> Result<RetainedArtifact, String> {
        self.with_artifact(path, |opened| {
            let manifest = opened.manifest().clone();
            require(
                manifest.files.values().filter(|e| e.module).count() == 1,
                "exactly one authenticated native module required",
            )?;
            let mut payloads = BTreeMap::new();
            let mut total = 0_u64;
            for (name, entry) in &manifest.files {
                let limit = match name.as_str() {
                    "target/module.wasm" => 512 << 20,
                    "target/provenance.json" | "evidence/replay.json" | "record.json" => 16 << 20,
                    _ if matches!(name.rsplit('/').next(), Some("choice" | "transcript"))
                        || name.starts_with("output/")
                        || name.contains("/output/") =>
                    {
                        1 << 30
                    }
                    _ => 256 << 20,
                };
                require(
                    entry.bytes <= limit,
                    "artifact payload exceeds current global bound",
                )?;
                total = total
                    .checked_add(entry.bytes)
                    .ok_or("artifact aggregate byte overflow")?;
                if matches!(
                    name.as_str(),
                    "target/module.wasm" | "target/provenance.json" | "evidence/replay.json"
                ) {
                    payloads.insert(name.clone(), opened.payload(name, limit)?);
                }
            }
            Ok(RetainedArtifact { manifest, payloads })
        })
    }
    pub(crate) fn authenticate_build(
        &mut self,
        path: &Path,
        role: &str,
        runner: &mut crate::cycle::process::Runner,
        cancelled: &std::sync::atomic::AtomicBool,
        context: &str,
    ) -> Result<Value, String> {
        self.contained(path)?;
        let record = crate::cycle::build::read_record(
            path,
            Some(&self.config),
            Some(role),
            runner,
            cancelled,
            context,
        )?;
        let binary = PathBuf::from(text(&record, "binary")?);
        self.contained(&binary)?;
        if let Some(previous) = self.build_roles.get(&binary) {
            require(previous == &record, "authenticated build role changed")?;
        }
        self.build_roles.insert(binary, record.clone());
        Ok(record)
    }
}

#[cfg(test)]
pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    tests::run_cases(owner)
}

pub(crate) fn observation(raw: &[u8], prefix: &str) -> Result<Value, String> {
    let text = std::str::from_utf8(raw).map_err(|e| e.to_string())?;
    let mut values = text.lines().filter_map(|line| line.strip_prefix(prefix));
    let value = values.next().ok_or("missing raw observation")?;
    require(
        !value.is_empty() && value.len() <= 1 << 20 && values.next().is_none(),
        "duplicate/empty/oversized raw observation",
    )?;
    custody::strict_json(value.as_bytes())
}
