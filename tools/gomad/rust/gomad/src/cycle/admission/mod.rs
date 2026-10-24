#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle CLI integration belongs to B15")
)]
use std::collections::BTreeMap;
use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::{Path, PathBuf};

const GIB: u64 = 1 << 30;

#[derive(Clone, Debug)]
pub(crate) struct Bindings {
    pub repo: PathBuf,
    pub evidence: PathBuf,
    pub scratch: PathBuf,
    pub temporary_paths: BTreeMap<String, PathBuf>,
    pub environment: BTreeMap<String, String>,
    pub config_sha256: String,
    pub host: PathBuf,
    pub overlay: PathBuf,
}

#[derive(Clone, Debug)]
pub(crate) struct Filesystem {
    pub kind: String,
    pub available_bytes: u64,
}

pub(crate) trait Probe {
    fn filesystem(&self, path: &Path) -> Result<Filesystem, String>;
}

pub(crate) struct KernelProbe;

impl Probe for KernelProbe {
    fn filesystem(&self, path: &Path) -> Result<Filesystem, String> {
        let filesystem = rustix::fs::statfs(path).map_err(|e| format!("filesystem probe: {e}"))?;
        let space = rustix::fs::statvfs(path).map_err(|e| format!("space probe: {e}"))?;
        let available_bytes = space
            .f_bavail
            .checked_mul(space.f_frsize)
            .ok_or("filesystem byte conversion overflow")?;
        Ok(Filesystem {
            kind: if filesystem.f_type == 0x794c_7630 {
                "overlayfs".into()
            } else if filesystem.f_type == 0x6573_5546 {
                "fuse".into()
            } else {
                format!("fs-{:x}", filesystem.f_type)
            },
            available_bytes,
        })
    }
}

fn require(condition: bool, message: &str) -> Result<(), String> {
    if condition {
        Ok(())
    } else {
        Err(message.into())
    }
}

fn canonical(path: &Path) -> bool {
    if !path.is_absolute() {
        return false;
    }
    if let Ok(resolved) = fs::canonicalize(path) {
        return resolved.as_os_str() == path.as_os_str();
    }
    let Some(parent) = path.parent() else {
        return false;
    };
    let Some(name) = path.file_name() else {
        return false;
    };
    canonical(parent)
        && parent.join(name).as_os_str() == path.as_os_str()
        && !matches!(name.to_str(), Some("." | ".."))
}

fn overlap(a: &Path, b: &Path) -> bool {
    a.starts_with(b) || b.starts_with(a)
}

impl Bindings {
    fn validate(&self) -> Result<(), String> {
        require(
            canonical(&self.repo) && self.repo.is_dir(),
            "candidate root must be canonical",
        )?;
        require(
            canonical(&self.evidence),
            "evidence root must be an absolute canonical path",
        )?;
        require(
            canonical(&self.scratch),
            "scratch must be an absolute canonical path",
        )?;
        require(
            self.scratch.parent().is_some_and(Path::is_dir),
            "scratch parent must already exist",
        )?;
        require(
            !overlap(&self.scratch, &self.repo),
            "scratch overlaps candidate source",
        )?;
        require(
            !overlap(&self.scratch, &self.evidence),
            "scratch overlaps evidence root",
        )?;
        require(
            self.temporary_paths.len() == 2
                && self.temporary_paths.contains_key("TMPDIR")
                && self.temporary_paths.contains_key("GOTMPDIR"),
            "explicit TMPDIR and GOTMPDIR scratch bindings required",
        )?;
        for (key, path) in &self.temporary_paths {
            require(
                canonical(path) && path.parent() == Some(self.scratch.as_path()),
                "temporary path must be a direct canonical scratch child",
            )?;
            require(
                self.environment
                    .get(key)
                    .is_none_or(|value| Path::new(value).as_os_str() == path.as_os_str()),
                "conflicting temporary environment binding",
            )?;
        }
        require(
            self.temporary_paths["TMPDIR"] != self.temporary_paths["GOTMPDIR"],
            "temporary paths must be distinct",
        )?;
        for path in [&self.host, &self.overlay] {
            require(
                canonical(path) && path.is_dir(),
                "serving path must be an existing canonical directory",
            )?;
        }
        require(self.evidence.is_dir(), "evidence root must exist")
    }
}

#[derive(Debug)]
pub(crate) struct Admission {
    bindings: Bindings,
    config_sha256: String,
    directories: Vec<(PathBuf, fs::File)>,
}

impl Admission {
    pub(crate) fn evidence(&self) -> &Path {
        &self.bindings.evidence
    }

    pub(crate) fn prepare(bindings: Bindings, probe: &impl Probe) -> Result<Self, String> {
        bindings.validate()?;
        require(
            !bindings.scratch.try_exists().map_err(|e| e.to_string())?,
            "scratch must be fresh",
        )?;
        let mut admission = Self {
            config_sha256: bindings.config_sha256.clone(),
            bindings,
            directories: Vec::new(),
        };
        admission.floors(probe, false)?;
        fs::create_dir(&admission.bindings.scratch).map_err(|e| format!("create scratch: {e}"))?;
        for path in admission.bindings.temporary_paths.values() {
            fs::create_dir(path).map_err(|e| format!("create temporary path: {e}"))?;
        }
        let b = &admission.bindings;
        for path in [
            b.repo.as_path(),
            b.evidence.as_path(),
            b.scratch.as_path(),
            b.host.as_path(),
            b.overlay.as_path(),
            b.scratch.parent().ok_or("scratch parent missing")?,
        ]
        .into_iter()
        .chain(b.temporary_paths.values().map(PathBuf::as_path))
        {
            let file = fs::File::open(path).map_err(|e| format!("pin admitted directory: {e}"))?;
            admission.directories.push((path.to_path_buf(), file));
        }
        admission.identities()?;
        Ok(admission)
    }

    pub(crate) fn recheck(&self, probe: &impl Probe) -> Result<(), String> {
        self.recheck_config(&self.bindings.config_sha256, probe)
    }

    pub(crate) fn recheck_config(
        &self,
        config_sha256: &str,
        probe: &impl Probe,
    ) -> Result<(), String> {
        self.bindings.validate()?;
        require(
            self.config_sha256 == config_sha256,
            "missing authenticated initial host admission",
        )?;
        require(
            self.bindings.scratch.is_dir()
                && self.bindings.temporary_paths.values().all(|p| p.is_dir()),
            "admitted temporary paths disappeared",
        )?;
        self.identities()?;
        self.floors(probe, true)?;
        self.identities()
    }

    fn identities(&self) -> Result<(), String> {
        for (path, file) in &self.directories {
            let pinned = file.metadata().map_err(|e| e.to_string())?;
            let current = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
            require(
                current.is_dir() && current.dev() == pinned.dev() && current.ino() == pinned.ino(),
                "admitted directory identity changed",
            )?;
        }
        Ok(())
    }

    fn floors(&self, probe: &impl Probe, reservation: bool) -> Result<(), String> {
        let b = &self.bindings;
        let inner = b.scratch.parent().ok_or("scratch parent missing")?;
        for (path, floor, scratch) in [
            (b.evidence.as_path(), 12 * GIB, false),
            (inner, 12 * GIB, true),
            (
                b.host.as_path(),
                if reservation { 45 * GIB } else { 12 * GIB },
                false,
            ),
            (
                b.overlay.as_path(),
                if reservation { 20 * GIB } else { 12 * GIB },
                false,
            ),
        ] {
            let fact = probe.filesystem(path)?;
            require(
                !fact.kind.is_empty()
                    && fact
                        .kind
                        .bytes()
                        .all(|b| b.is_ascii_alphanumeric() || b"._-".contains(&b)),
                "unrecognized filesystem result",
            )?;
            require(
                !scratch || (!fact.kind.contains("fuse") && fact.kind != "virtiofs"),
                "scratch filesystem refused",
            )?;
            require(
                !reservation || !scratch || fact.kind == "overlayfs",
                "actual overlay scratch is required",
            )?;
            require(
                fact.available_bytes >= floor,
                &format!(
                    "filesystem free byte floor: {} requires {floor}",
                    path.display()
                ),
            )?;
        }
        Ok(())
    }
}

pub(crate) fn unsigned_decimal(value: &str) -> Result<u64, String> {
    require(
        !value.is_empty() && value.len() <= 20 && value.bytes().all(|b| b.is_ascii_digit()),
        "bounded unsigned decimal memory value required",
    )?;
    value
        .parse()
        .map_err(|_| "bounded unsigned decimal memory value required".into())
}

pub(crate) fn admit_memory(
    threshold: &str,
    platform: &str,
    input: impl Read,
) -> Result<u64, String> {
    let threshold = unsigned_decimal(threshold)?;
    require(
        threshold > 0,
        "positive decimal memory admission threshold required",
    )?;
    require(
        platform == "linux",
        "explicit memory admission requires Linux MemAvailable",
    )?;
    let mut raw = String::new();
    input
        .take(8193)
        .read_to_string(&mut raw)
        .map_err(|e| format!("MemAvailable observation: {e}"))?;
    require(raw.len() <= 8192, "MemAvailable observation exceeds 8KiB")?;
    let available = super::scheduler::parse_mem_available_bytes(&raw)?;
    require(
        available >= threshold,
        "insufficient available memory before helper dispatch",
    )?;
    Ok(available)
}

#[cfg(test)]
mod tests;
