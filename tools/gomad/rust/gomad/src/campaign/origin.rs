use crate::evidence::{
    canonical,
    record::{Record, Stream},
    store::{Opened, Publication, open},
};
use serde::{Deserialize, Serialize};
use std::{
    ffi::OsStr,
    os::unix::ffi::OsStrExt as _,
    path::Path,
    sync::atomic::{AtomicU64, Ordering},
    time::{SystemTime, UNIX_EPOCH},
};

pub(crate) const FILE: &str = "target/intercepted-origin.json";
pub(crate) const MAXIMUM: u64 = 32 << 20;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Invocation {
    pub schema: String,
    pub wrapper_run_id: String,
    pub invocation_id: String,
    pub original_argv: Vec<String>,
    pub driver_argv: Vec<String>,
    pub compiler_sha256: String,
    pub installation_sha256: String,
    pub recipe_sha256: String,
    pub recipe_file: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Intercepted {
    pub schema: String,
    pub origin: String,
    pub invocation: Invocation,
    pub module_sha256: String,
    pub preparation_sha256: String,
}

pub(crate) fn identity() -> Result<String, String> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_nanos();
    Ok(canonical::digest(
        format!(
            "{}:{nanos}:{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        )
        .as_bytes(),
    ))
}

pub(crate) fn argument(value: &OsStr) -> String {
    value
        .as_bytes()
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect()
}

impl Invocation {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.intercepted-invocation/v1"
            || self.original_argv.is_empty()
            || self.original_argv.len() > 65536
            || self.driver_argv.is_empty()
            || self.driver_argv.len() > 65536
            || [
                &self.wrapper_run_id,
                &self.invocation_id,
                &self.compiler_sha256,
                &self.installation_sha256,
                &self.recipe_sha256,
            ]
            .into_iter()
            .any(|s| !canonical::valid_digest(s))
            || self.original_argv.iter().chain(&self.driver_argv).any(|s| {
                s.len() > 2 << 20
                    || s.len() % 2 != 0
                    || !s
                        .bytes()
                        .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
            })
            || self.recipe_file.is_empty()
            || self.recipe_file.contains('/')
            || self.recipe_file.contains('\\')
            || matches!(self.recipe_file.as_str(), "." | "..")
        {
            return Err("invalid intercepted invocation metadata".into());
        }
        Ok(())
    }
}

pub(crate) fn read_private(path: &Path, expected: &str) -> Result<Vec<u8>, String> {
    if !path.is_absolute() || !canonical::valid_digest(expected) {
        return Err("invalid private invocation reference".into());
    }
    let parent = path.parent().ok_or("invocation parent missing")?;
    let name = path
        .file_name()
        .and_then(OsStr::to_str)
        .ok_or("invalid invocation filename")?;
    open::components(name)?;
    let root = open::pin(parent)?;
    let bytes = open::read(&root, name, MAXIMUM, false)?;
    if canonical::digest(&bytes) != expected {
        return Err("intercepted invocation digest mismatch".into());
    }
    Ok(bytes)
}

impl Intercepted {
    pub(crate) fn validate(&self) -> Result<(), String> {
        self.invocation.validate()?;
        if self.schema != "gomad.intercepted-origin/v1"
            || self.origin != "intercepted-go"
            || !canonical::valid_digest(&self.module_sha256)
            || !canonical::valid_digest(&self.preparation_sha256)
        {
            return Err("invalid intercepted evidence origin".into());
        }
        Ok(())
    }
    pub(crate) fn bind(&self, publication: &mut Publication) -> Result<(), String> {
        self.validate()?;
        if self.module_sha256 != publication.record.identity.target {
            return Err("intercepted origin module mismatch".into());
        }
        let bytes = canonical::encode(self)?;
        publication.record.streams.insert(
            FILE.into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        );
        publication.payloads.insert(FILE.into(), bytes);
        publication
            .payloads
            .insert("record.json".into(), publication.record.bytes()?);
        Ok(())
    }
}

pub(crate) fn authenticate(opened: &Opened) -> Result<Option<Intercepted>, String> {
    let manifest = opened.manifest();
    let bound = manifest.record.streams.get(FILE);
    if manifest.files.contains_key(FILE) != bound.is_some() {
        return Err("intercepted origin is not record-bound".into());
    }
    let Some(stream) = bound else {
        return Ok(None);
    };
    let bytes = opened.payload(FILE, MAXIMUM)?;
    let origin: Intercepted = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    origin.validate()?;
    if canonical::encode(&origin)? != bytes
        || stream.records != 1
        || origin.module_sha256 != manifest.record.identity.target
    {
        return Err("intercepted evidence origin mismatch".into());
    }
    let provenance: serde_json::Value =
        serde_json::from_slice(&opened.payload("target/provenance.json", MAXIMUM)?)
            .map_err(|e| e.to_string())?;
    if provenance["origin"] != serde_json::to_value(&origin).map_err(|e| e.to_string())? {
        return Err("intercepted origin provenance mismatch".into());
    }
    authenticate_provenance(provenance, &origin)?;
    Ok(Some(origin))
}

pub(crate) fn validate_record(origin: &serde_json::Value, record: &Record) -> Result<(), String> {
    if origin == "operator" {
        if record.streams.contains_key(FILE) {
            return Err("operator campaign has intercepted evidence".into());
        }
        return Ok(());
    }
    let origin: Intercepted = serde_json::from_value(origin.clone()).map_err(|e| e.to_string())?;
    origin.validate()?;
    let bytes = canonical::encode(&origin)?;
    if origin.module_sha256 != record.identity.target
        || record.streams.get(FILE)
            != Some(&Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            })
    {
        return Err("campaign origin differs from its evidence".into());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn exact_argument_bytes_and_distinct_identities() {
        assert_eq!(argument(OsStr::from_bytes(b" a'\xff")), "206127ff");
        assert_eq!(argument(OsStr::new("")), "");
        assert_ne!(identity().unwrap(), identity().unwrap());
    }
}

pub(crate) fn from_preparation(
    prepared: &super::prepare::Prepared,
) -> Result<Option<Intercepted>, String> {
    let provenance: serde_json::Value =
        serde_json::from_slice(&prepared.provenance).map_err(|e| e.to_string())?;
    let Some(value) = provenance.get("origin") else {
        return Ok(None);
    };
    let origin: Intercepted = serde_json::from_value(value.clone()).map_err(|e| e.to_string())?;
    authenticate_provenance(provenance, &origin)?;
    if origin.module_sha256 != canonical::digest(&prepared.module) {
        return Err("preparation origin module mismatch".into());
    }
    Ok(Some(origin))
}

fn authenticate_provenance(
    mut provenance: serde_json::Value,
    origin: &Intercepted,
) -> Result<(), String> {
    origin.validate()?;
    provenance
        .as_object_mut()
        .ok_or("invalid intercepted provenance")?
        .remove("origin");
    if origin.preparation_sha256 != canonical::digest(&canonical::encode(&provenance)?) {
        return Err("intercepted preparation digest mismatch".into());
    }
    Ok(())
}

pub(crate) fn intercept(
    prepared: &mut super::prepare::Prepared,
    invocation: Invocation,
) -> Result<Intercepted, String> {
    let origin = Intercepted {
        schema: "gomad.intercepted-origin/v1".into(),
        origin: "intercepted-go".into(),
        invocation,
        module_sha256: canonical::digest(&prepared.module),
        preparation_sha256: canonical::digest(&prepared.provenance),
    };
    origin.validate()?;
    let mut provenance: serde_json::Value =
        serde_json::from_slice(&prepared.provenance).map_err(|e| e.to_string())?;
    provenance
        .as_object_mut()
        .ok_or("invalid preparation provenance")?
        .insert(
            "origin".into(),
            serde_json::to_value(&origin).map_err(|e| e.to_string())?,
        );
    prepared.provenance = canonical::encode(&provenance)?;
    Ok(origin)
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct DriverRegistration {
    pub schema: String,
    pub process_id: u32,
    pub invocation_sha256: String,
    pub compiler_sha256: String,
    pub driver_argv_sha256: String,
    pub module_root: std::path::PathBuf,
}

pub(crate) fn authenticate_driver(
    path: &Path,
    invocation_digest: &str,
    invocation: &Invocation,
    module: &Path,
) -> Result<(), String> {
    let root = open::pin(path.parent().ok_or("invocation owner missing")?)?;
    let bytes = open::read(&root, "driver-registration.json", 16384, false)?;
    let registration: DriverRegistration =
        serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    let parent = rustix::process::getppid().ok_or("intercepted compiler parent missing")?;
    if canonical::encode(&registration)? != bytes
        || registration.schema != "gomad.driver-registration/v1"
        || Some(parent.as_raw_pid()) != i32::try_from(registration.process_id).ok()
        || registration.invocation_sha256 != invocation_digest
        || registration.compiler_sha256 != invocation.compiler_sha256
        || registration.driver_argv_sha256
            != canonical::digest(&canonical::encode(&invocation.driver_argv)?)
        || !std::fs::canonicalize(module)
            .map_err(|e| e.to_string())?
            .starts_with(
                std::fs::canonicalize(&registration.module_root).map_err(|e| e.to_string())?,
            )
    {
        return Err("module is not associated with the registered live pinned Go driver".into());
    }
    open::pin(&registration.module_root)?;
    Ok(())
}
