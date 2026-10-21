//! Retained qualification `Result` and `PublicationReceipt` in the Rust lineage.
//! Both are canonical JSON, content addressed and published without replace.

use super::{
    cost::{self, Budget, Cost, Observations},
    report::{self, Report},
    usage::Usage,
};
use crate::evidence::{
    canonical,
    store::{durable_directory, open},
};
use rustix::fs::{self as fsx, Mode, OFlags};
use serde::{Deserialize, Serialize};
use std::{
    fs::File,
    io::Write as _,
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

pub(crate) const MANIFEST_SCHEMA: &str = "gomad.rust-qualification-manifest/v1";
pub(crate) const RESULT_SCHEMA: &str = "gomad.rust-qualification-result/v1";
pub(crate) const RECEIPT_SCHEMA: &str = "gomad.rust-qualification-receipt/v1";
const DIRECTORY: [&str; 2] = ["rust-qualifications", "v1"];
const MAXIMUM_RESULT_BYTES: u64 = 64 << 20;
const MAXIMUM_RECEIPT_BYTES: u64 = 8 << 10;

/// A frozen comparison cell, not an assertion that it passed. Every identity
/// is a Rust-lineage digest observed during freeze.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Manifest {
    pub schema: String,
    pub provisional: bool,
    pub integrated_source_manifest_path: Option<String>,
    pub id: String,
    pub workload: String,
    pub profile: String,
    pub cell: Option<String>,
    pub integrated_source_sha256: String,
    pub workload_source_sha256: String,
    pub runtime_sha256: String,
    pub compiler_sha256: String,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub model_sha256: String,
    pub engine_sha256: String,
    pub host_config_sha256: String,
    pub module_sha256: String,
    pub toolchain_sha256: String,
    pub host_os: String,
    pub host_arch: String,
    pub seed: u64,
    pub repeat: u64,
    pub fuel: u64,
    pub memory_bytes: u64,
    pub unsupported: Vec<String>,
}

impl Manifest {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if self
            .cell
            .as_deref()
            .is_some_and(|cell| !matches!(cell, "success" | "canary"))
            || !self.provisional && self.cell.is_none()
        {
            return Err("final qualification requires explicit success/canary cell".into());
        }
        if !self.provisional
            && !self
                .integrated_source_manifest_path
                .as_deref()
                .is_some_and(|path| Path::new(path).is_absolute())
        {
            return Err(
                "final WASM qualification requires conductor integrated-source manifest reference"
                    .into(),
            );
        }
        if self.schema != MANIFEST_SCHEMA
            || self.id.is_empty()
            || self.host_os.is_empty()
            || self.host_arch.is_empty()
            || !(2..=report::MAXIMUM_REPEAT).contains(&self.repeat)
            || self.fuel == 0
            || self.memory_bytes == 0
        {
            return Err("frozen WASM identity, limits and 2..32 repeats are required".into());
        }
        for digest in [
            &self.integrated_source_sha256,
            &self.workload_source_sha256,
            &self.runtime_sha256,
            &self.compiler_sha256,
            &self.helper_sha256,
            &self.host_sha256,
            &self.model_sha256,
            &self.engine_sha256,
            &self.host_config_sha256,
            &self.module_sha256,
            &self.toolchain_sha256,
        ] {
            if !canonical::valid_digest(digest) {
                return Err("unbound WASM qualification identity".into());
            }
        }
        if !matches!(
            self.workload.as_str(),
            "fixture" | "sqlite" | "onebox" | "multi-instance"
        ) {
            return Err("unsupported WASM qualification workload".into());
        }
        if !matches!(self.profile.as_str(), "controlled" | "cooperative") {
            return Err(format!(
                "unsupported WASM profile {:?}: no admitted backend owner",
                self.profile
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct QualificationResult {
    pub schema: String,
    pub manifest: Manifest,
    pub report: Report,
    pub observations: Observations,
    pub cost: Option<Cost>,
    pub cost_failure: Option<String>,
    pub budget: Option<Budget>,
    pub budget_failure: Option<String>,
    pub integrated_source_manifest: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Receipt {
    pub schema: String,
    pub result_sha256: String,
    pub operator_process_id: u64,
    pub linked_after: Option<Usage>,
}

/// The conductor's sha256sum manifest: lowercase hex, two spaces, absolute
/// path. With `check_files` every listed file is re-read and authenticated.
pub(crate) fn validate_source_manifest(encoded: &[u8], check_files: bool) -> Result<(), String> {
    if encoded.is_empty() || encoded.len() > 16 << 20 {
        return Err("bounded nonempty integrated source manifest required".into());
    }
    let text = std::str::from_utf8(encoded)
        .map_err(|_| "malformed integrated source manifest line".to_owned())?;
    let lines = text
        .strip_suffix('\n')
        .unwrap_or(text)
        .split('\n')
        .collect::<Vec<_>>();
    if lines.len() > 20_000 {
        return Err("integrated source manifest entry limit".into());
    }
    let mut seen = std::collections::BTreeSet::new();
    let mut total = 0_u64;
    for line in lines {
        if line.len() < 67 || !line.is_char_boundary(64) || &line[64..66] != "  " {
            return Err("malformed integrated source manifest line".into());
        }
        let (digest, path) = (&line[..64], &line[66..]);
        if !canonical::valid_digest(digest) {
            return Err("malformed integrated source manifest digest".into());
        }
        let parsed = Path::new(path);
        if !parsed.is_absolute()
            || path.contains(['\0', '\r'])
            || parsed.components().any(|part| {
                matches!(
                    part,
                    std::path::Component::CurDir | std::path::Component::ParentDir
                )
            })
            || path.ends_with('/')
            || path.contains("//")
            || !seen.insert(path)
        {
            return Err("duplicate or malformed integrated source manifest path".into());
        }
        if check_files {
            let data = super::spec::read_bounded(parsed, 128 << 20)?;
            total += data.len() as u64;
            if total > 512 << 20 {
                return Err("integrated source inventory byte limit".into());
            }
            if canonical::digest(&data) != digest {
                return Err("integrated source manifest file identity changed".into());
            }
        }
    }
    Ok(())
}

/// Installation-free validation: recomputes report verdicts, costs and budget
/// from retained raw observations and refuses any claim that differs.
pub(crate) fn validate(result: &QualificationResult) -> Result<(), String> {
    if result.schema != RESULT_SCHEMA {
        return Err("unsupported qualification result schema".into());
    }
    let manifest = &result.manifest;
    manifest.validate()?;
    if !manifest.provisional && result.observations.operator_process_id == 0 {
        return Err("final result requires actual operator process identity".into());
    }
    match &result.integrated_source_manifest {
        Some(encoded) => {
            if canonical::digest(encoded.as_bytes()) != manifest.integrated_source_sha256 {
                return Err("retained integrated source manifest identity changed".into());
            }
            validate_source_manifest(encoded.as_bytes(), false)?;
        }
        None if !manifest.provisional => {
            return Err("final result omitted integrated source manifest".into());
        }
        None => {}
    }
    report::validate(&result.report)?;
    if result.report.seed != manifest.seed || result.report.repeat != manifest.repeat {
        return Err("retained qualification report differs from frozen sample selection".into());
    }
    for execution in &result.report.executions {
        let identity = &execution.record.identity;
        if identity.target != manifest.module_sha256
            || identity.toolchain != manifest.toolchain_sha256
            || identity.configuration != manifest.host_config_sha256
            || execution.record.limits.fuel != manifest.fuel
            || execution.record.limits.memory_bytes != manifest.memory_bytes
        {
            return Err("qualification execution identity differs from the frozen manifest".into());
        }
    }
    let (cost, cost_failure, budget, budget_failure) =
        cost::evaluate(&result.observations, &result.report);
    if cost != result.cost || cost_failure != result.cost_failure {
        return Err(if result.cost.is_some() && cost.is_none() {
            "unmeasured WASM costs were claimed complete".into()
        } else {
            "retained WASM cost differs from raw measured samples".into()
        });
    }
    if budget != result.budget || budget_failure != result.budget_failure {
        return Err(if result.budget.is_some() && budget.is_none() {
            "unmeasured budget was claimed complete".into()
        } else {
            "retained budget differs from raw measurements".into()
        });
    }
    Ok(())
}

pub(crate) fn encode(result: &QualificationResult) -> Result<Vec<u8>, String> {
    let mut encoded = canonical::encode(result)?;
    encoded.push(b'\n');
    Ok(encoded)
}

fn directory(root: &Path) -> Result<(PathBuf, File), String> {
    if !root.is_absolute() {
        return Err("absolute WASM qualification result root required".into());
    }
    let path = DIRECTORY
        .iter()
        .fold(root.to_path_buf(), |path, part| path.join(part));
    let pinned = durable_directory(&path, |_| Ok(()))?;
    Ok((path, pinned))
}

static STAGING: AtomicU64 = AtomicU64::new(0);

/// Stages, syncs and exclusively renames immutable bytes; a collision is
/// reused only when the existing file authenticates to the same bytes.
fn publish_bytes(root: &File, name: &str, bytes: &[u8], limit: u64) -> Result<(), String> {
    if bytes.is_empty() || bytes.len() as u64 > limit {
        return Err("WASM qualification result byte limit".into());
    }
    let stage = format!(
        ".result-{}-{}.partial",
        std::process::id(),
        STAGING.fetch_add(1, Ordering::Relaxed)
    );
    let mut file = File::from(
        fsx::openat(
            root,
            stage.as_str(),
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    let staged = (|| {
        file.write_all(bytes).map_err(|error| error.to_string())?;
        file.sync_all().map_err(|error| error.to_string())?;
        rename_exclusive(root, &stage, name)
    })();
    let outcome = match staged {
        Ok(()) => Ok(()),
        Err(error) if error == rustix::io::Errno::EXIST.to_string() => {
            if open::read(root, name, limit, false)? == bytes {
                Ok(())
            } else {
                Err("immutable WASM qualification result collision".into())
            }
        }
        Err(error) => Err(error),
    };
    match fsx::unlinkat(root, stage.as_str(), fsx::AtFlags::empty()) {
        Ok(()) | Err(rustix::io::Errno::NOENT) => {}
        Err(error) => return Err(error.to_string()),
    }
    outcome?;
    root.sync_all().map_err(|error| error.to_string())
}

fn rename_exclusive(root: &File, source: &str, target: &str) -> Result<(), String> {
    #[cfg(any(target_os = "linux", target_os = "macos"))]
    {
        fsx::renameat_with(root, source, root, target, fsx::RenameFlags::NOREPLACE)
            .map_err(|error| error.to_string())
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        let _ = (root, source, target);
        Err("exclusive qualification publication is unsupported on this platform".into())
    }
}

fn name(bytes: &[u8]) -> String {
    format!("result-{}.json", canonical::digest(bytes))
}

/// Validates, encodes and publishes the Result; returns its absolute path.
pub(crate) fn publish(root: &Path, result: &QualificationResult) -> Result<PathBuf, String> {
    validate(result)?;
    let encoded = encode(result)?;
    let (path, pinned) = directory(root)?;
    let name = name(&encoded);
    publish_bytes(&pinned, &name, &encoded, MAXIMUM_RESULT_BYTES)?;
    Ok(path.join(name))
}

fn result_identity(path: &Path) -> Result<String, String> {
    path.file_name()
        .and_then(|name| name.to_str())
        .and_then(|name| name.strip_prefix("result-"))
        .and_then(|name| name.strip_suffix(".json"))
        .filter(|digest| canonical::valid_digest(digest))
        .map(str::to_owned)
        .ok_or_else(|| "publication requires content-addressed Result path".into())
}

fn pinned_parent(path: &Path) -> Result<(File, String), String> {
    let parent = path
        .parent()
        .ok_or("qualification result path has no directory")?;
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("qualification result path is not UTF-8")?;
    Ok((open::pin(parent)?, name.to_owned()))
}

/// Publishes the receipt measured after the sealed Result publication.
pub(crate) fn publish_receipt(path: &Path, receipt: &Receipt) -> Result<(), String> {
    let (pinned, name) = pinned_parent(path)?;
    let mut encoded = canonical::encode(receipt)?;
    encoded.push(b'\n');
    publish_bytes(
        &pinned,
        &format!("{name}.publication.json"),
        &encoded,
        MAXIMUM_RECEIPT_BYTES,
    )
}

/// Opens a retained Result without installation, source or compiler: bounded
/// private regular file, content address, strict decode and recomputation.
pub(crate) fn open(path: &Path) -> Result<QualificationResult, String> {
    let digest = result_identity(path)?;
    let (pinned, name) = pinned_parent(path)?;
    let encoded = open::read(&pinned, &name, MAXIMUM_RESULT_BYTES, false)?;
    if canonical::digest(&encoded) != digest {
        return Err("WASM qualification result content identity changed".into());
    }
    let result: QualificationResult = super::spec::strict(&encoded)?;
    if encode(&result)? != encoded {
        return Err("WASM qualification result is not canonical".into());
    }
    validate(&result)?;
    Ok(result)
}

/// Opens and binds the receipt to its already opened Result; returns the
/// linked self CPU measured after the Result boundary.
pub(crate) fn open_receipt(
    path: &Path,
    result: &QualificationResult,
) -> Result<(Receipt, u64), String> {
    let digest = result_identity(path)?;
    let (pinned, name) = pinned_parent(path)?;
    let encoded = open::read(
        &pinned,
        &format!("{name}.publication.json"),
        MAXIMUM_RECEIPT_BYTES,
        false,
    )?;
    let receipt: Receipt = super::spec::strict(&encoded)?;
    let mut canonical_bytes = canonical::encode(&receipt)?;
    canonical_bytes.push(b'\n');
    if canonical_bytes != encoded {
        return Err("qualification publication receipt is not canonical".into());
    }
    if canonical::digest(&encode(result)?) != digest {
        return Err("publication receipt Result content binding changed".into());
    }
    let tail = validate_receipt(result, &receipt, &digest)?;
    Ok((receipt, tail))
}

pub(crate) fn validate_receipt(
    result: &QualificationResult,
    receipt: &Receipt,
    digest: &str,
) -> Result<u64, String> {
    if receipt.schema != RECEIPT_SCHEMA
        || receipt.result_sha256 != digest
        || receipt.operator_process_id == 0
        || receipt.operator_process_id != result.observations.operator_process_id
    {
        return Err("missing or mismatched publication self observation".into());
    }
    let after = receipt
        .linked_after
        .ok_or("missing or mismatched publication self observation")?;
    cost::terminal(&result.observations, after)
}
