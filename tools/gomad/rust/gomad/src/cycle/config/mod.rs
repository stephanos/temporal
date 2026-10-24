#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle endpoint integration belongs to successor batches"
    )
)]
use super::{
    custody,
    model::{Mode, PlanConfig},
    plan,
};
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
    time::{SystemTime, UNIX_EPOCH},
};

#[cfg(test)]
mod tests;

const MAX_CONFIG_BYTES: u64 = 16 << 20;

#[derive(Debug)]
pub(crate) struct Configured {
    original_path: PathBuf,
    original_bytes: Vec<u8>,
    original_sha256: String,
    value: Value,
}

pub(crate) struct TrustedContext<'a> {
    pub repo_root: &'a Path,
    pub now_epoch: f64,
    pub available_cores: usize,
}

impl Configured {
    pub(crate) fn read(path: &Path, requested_sha256: &str) -> Result<Self, String> {
        let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../../..")
            .canonicalize()
            .map_err(|e| format!("candidate checkout: {e}"))?;
        let now_epoch = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|e| e.to_string())?
            .as_secs_f64();
        let available_cores = std::thread::available_parallelism()
            .map_err(|e| e.to_string())?
            .get();
        Self::read_with_context(
            path,
            requested_sha256,
            &TrustedContext {
                repo_root: &repo,
                now_epoch,
                available_cores,
            },
        )
    }

    pub(crate) fn read_with_context(
        path: &Path,
        requested_sha256: &str,
        context: &TrustedContext<'_>,
    ) -> Result<Self, String> {
        if !custody::valid_digest(requested_sha256) {
            return Err("malformed configuration SHA256".into());
        }
        let bytes = custody::read_file(path, MAX_CONFIG_BYTES)?;
        if custody::digest(&bytes) != requested_sha256 {
            return Err("configuration SHA256 mismatch".into());
        }
        let value = custody::strict_json(&bytes)?;
        validate(&value, path, requested_sha256, context)?;
        Ok(Self {
            original_path: path.to_owned(),
            original_bytes: bytes,
            original_sha256: requested_sha256.to_owned(),
            value,
        })
    }

    pub(crate) fn value(&self) -> &Value {
        &self.value
    }

    pub(crate) fn path(&self) -> &Path {
        &self.original_path
    }

    pub(crate) fn bytes(&self) -> &[u8] {
        &self.original_bytes
    }

    pub(crate) fn sha256(&self) -> &str {
        &self.original_sha256
    }

    pub(crate) fn recheck(&self) -> Result<(), String> {
        let bytes = custody::read_file(&self.original_path, MAX_CONFIG_BYTES)
            .map_err(|e| format!("driver config changed: {e}"))?;
        if bytes != self.original_bytes || custody::digest(&bytes) != self.original_sha256 {
            return Err("driver config changed".into());
        }
        Ok(())
    }
}

fn validate(
    config: &Value,
    path: &Path,
    sha256: &str,
    context: &TrustedContext<'_>,
) -> Result<(), String> {
    if config["schema"] != "gomad.rust-qualification-driver/v1" {
        return Err("unknown driver config".into());
    }
    if config
        .as_object()
        .is_some_and(|fields| fields.keys().any(|key| key.starts_with('_')))
    {
        return Err("reserved configuration field".into());
    }
    let expected_repo = context
        .repo_root
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let repo = config["repo"].as_str().ok_or("wrong candidate checkout")?;
    if canonical(Path::new(repo)).as_deref() != Ok(expected_repo.as_path()) {
        return Err("wrong candidate checkout".into());
    }
    validate_stages(config, path, sha256)?;
    let reader = argv(&config["final_reader"]).ok_or("missing actual final readers")?;
    let sections = config["final_sections"]
        .as_object()
        .filter(|entries| !entries.is_empty())
        .ok_or("missing actual final readers")?;
    if config["synthetic"] == true {
        return Err("synthetic fixture cannot qualify".into());
    }
    let identity = config["identity"]
        .as_object()
        .ok_or("frozen host/helper identities required")?;
    if identity.keys().map(String::as_str).collect::<BTreeSet<_>>()
        != BTreeSet::from(["host", "helper"])
    {
        return Err("frozen host/helper identities required".into());
    }
    bind_identity(
        "release",
        &config["identity"],
        "release",
        &config["identity"],
    )?;
    validate_resources(config, context)?;
    bindings(config, &expected_repo)?;
    let files: BTreeMap<String, String> = serde_json::from_value(config["files"].clone())
        .map_err(|e| format!("invalid frozen file inventory: {e}"))?;
    let stat = filesystem_probe()?;
    if !files.contains_key(stat) {
        return Err("unsealed native filesystem probe".into());
    }
    sealed_endpoint(&reader, &files)?;
    for section in sections.values() {
        let arguments = argv(&section["argv"]).ok_or("missing actual final readers")?;
        sealed_endpoint(&arguments, &files)?;
    }
    custody::check_files(&files)
}

fn validate_stages(config: &Value, path: &Path, sha256: &str) -> Result<(), String> {
    let plan = plan::derive(
        &PlanConfig {
            binary: path.to_owned(),
            config_path: path.to_owned(),
            config_sha256: sha256.to_owned(),
        },
        Mode::Parallel,
    )?;
    let stages = config["stages"]
        .as_object()
        .ok_or("missing/unexpected configured stages")?;
    let wanted: BTreeSet<_> = plan
        .stages
        .iter()
        .map(|stage| stage.contract.label.as_str())
        .collect();
    if stages.keys().map(String::as_str).collect::<BTreeSet<_>>() != wanted {
        return Err("missing/unexpected configured stages".into());
    }
    if stages.values().any(|entry| {
        entry["adapter"]
            .as_str()
            .is_none_or(|adapter| adapter.is_empty() || adapter == "pending")
    }) {
        return Err("unbound original stage endpoints cannot start qualification".into());
    }
    Ok(())
}

fn validate_resources(config: &Value, context: &TrustedContext<'_>) -> Result<(), String> {
    let cores = positive(&config["cores"]).ok_or("invalid host resource caps")?;
    positive(&config["mem_gib"]).ok_or("invalid host resource caps")?;
    #[expect(
        clippy::cast_precision_loss,
        reason = "Machine CPU counts are bounded well below f64's exact integer range"
    )]
    let available_cores = context.available_cores as f64;
    if cores > available_cores {
        return Err("core cap exceeds host".into());
    }
    let sent = positive(&config["sent_epoch"]).ok_or("invalid send epoch")?;
    if !context.now_epoch.is_finite() || sent > context.now_epoch {
        return Err("invalid send epoch".into());
    }
    deadline(&config["deadline_s"])?;
    Ok(())
}

fn positive(value: &Value) -> Option<f64> {
    value
        .as_f64()
        .filter(|number| number.is_finite() && *number > 0.0)
}

pub(crate) fn deadline(limit: &Value) -> Result<f64, String> {
    positive(limit)
        .filter(|seconds| *seconds <= 3600.0)
        .ok_or_else(|| "deadline must be positive and at most 3600 seconds".into())
}

pub(crate) fn bind_identity(
    expected_role: &str,
    expected: &Value,
    role: &str,
    observed: &Value,
) -> Result<(), String> {
    if expected_role != "release" || role != "release" {
        return Err("qualification requires admitted release role".into());
    }
    let values = expected.as_object().filter(|values| !values.is_empty());
    if values.is_none() || expected != observed {
        return Err("candidate identity mismatch".into());
    }
    if values.is_none_or(|values| {
        values.values().any(|value| {
            value
                .as_str()
                .is_none_or(|digest| !custody::valid_digest(digest))
        })
    }) {
        return Err("malformed identity".into());
    }
    Ok(())
}

fn argv(value: &Value) -> Option<Vec<&str>> {
    let values = value.as_array().filter(|values| !values.is_empty())?;
    values
        .iter()
        .map(|value| {
            value
                .as_str()
                .filter(|s| !s.is_empty() && !s.contains('\0'))
        })
        .collect()
}

fn sealed_endpoint(argv: &[&str], files: &BTreeMap<String, String>) -> Result<(), String> {
    if !files.contains_key(argv[0])
        || argv
            .get(1)
            .is_some_and(|name| Path::new(name).is_absolute() && !files.contains_key(*name))
    {
        return Err("unsealed actual final endpoint".into());
    }
    Ok(())
}

fn filesystem_probe() -> Result<&'static str, String> {
    ["/usr/bin/stat", "/usr/bin/gnustat"]
        .into_iter()
        .find(|path| {
            fs::symlink_metadata(path).is_ok_and(|metadata| metadata.file_type().is_file())
        })
        .ok_or_else(|| "missing regular native filesystem probe".into())
}

fn canonical(path: &Path) -> Result<PathBuf, String> {
    custody::clean_absolute(path)?;
    let resolved = match fs::symlink_metadata(path) {
        Ok(_) => path.canonicalize().map_err(|e| e.to_string())?,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            let parent = path.parent().ok_or("canonical path has no parent")?;
            canonical(parent)?.join(path.file_name().ok_or("canonical path has no filename")?)
        }
        Err(error) => return Err(error.to_string()),
    };
    if resolved != path {
        return Err("noncanonical path".into());
    }
    Ok(resolved)
}

fn bindings(config: &Value, repo: &Path) -> Result<(), String> {
    let root = config["root"]
        .as_str()
        .ok_or("evidence root must be an absolute canonical path")?;
    let root = canonical(Path::new(root))
        .map_err(|_| "evidence root must be an absolute canonical path")?;
    let scratch = config["scratch"]
        .as_str()
        .ok_or("scratch must be an absolute canonical path")?;
    let scratch =
        canonical(Path::new(scratch)).map_err(|_| "scratch must be an absolute canonical path")?;
    if !scratch.parent().is_some_and(Path::is_dir) {
        return Err("scratch parent must already exist".into());
    }
    if scratch.starts_with(repo) || repo.starts_with(&scratch) {
        return Err("scratch overlaps candidate source".into());
    }
    if scratch.starts_with(&root) || root.starts_with(&scratch) {
        return Err("scratch overlaps evidence root".into());
    }
    let paths = config["temporary_paths"]
        .as_object()
        .filter(|paths| {
            paths.keys().map(String::as_str).collect::<BTreeSet<_>>()
                == BTreeSet::from(["TMPDIR", "GOTMPDIR"])
        })
        .ok_or("explicit TMPDIR and GOTMPDIR scratch bindings required")?;
    if let Some(environment) = config.get("environment")
        && !environment
            .as_object()
            .is_some_and(|values| values.values().all(Value::is_string))
    {
        return Err("invalid configured environment".into());
    }
    let mut observed = BTreeSet::new();
    for (key, name) in paths {
        let name = name
            .as_str()
            .ok_or_else(|| format!("{key} must be a direct canonical scratch child"))?;
        let path = canonical(Path::new(name))
            .map_err(|_| format!("{key} must be a direct canonical scratch child"))?;
        if path.parent() != Some(scratch.as_path()) {
            return Err(format!("{key} must be a direct canonical scratch child"));
        }
        if let Some(value) = config.get("environment").and_then(|env| env.get(key))
            && value.as_str() != Some(name)
        {
            return Err(format!("conflicting temporary environment binding: {key}"));
        }
        observed.insert(path);
    }
    if observed.len() != 2 {
        return Err("temporary paths must be distinct".into());
    }
    Ok(())
}
