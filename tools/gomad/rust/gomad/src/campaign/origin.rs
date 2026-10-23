mod cache;

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
    #[serde(rename = "invocation_id")]
    pub id: String,
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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub launch: Option<Launch>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Launch {
    pub invocation_id: String,
    pub argv: Vec<String>,
    pub process_id: u32,
    pub parent_process_id: Option<u32>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub preparation_receipt_sha256: Option<String>,
}

impl Launch {
    fn current(preparation_receipt_sha256: Option<String>) -> Result<Self, String> {
        Ok(Self {
            invocation_id: identity()?,
            argv: std::env::args_os().map(|value| argument(&value)).collect(),
            process_id: std::process::id(),
            parent_process_id: rustix::process::getppid()
                .and_then(|pid| u32::try_from(pid.as_raw_pid()).ok()),
            preparation_receipt_sha256,
        })
    }

    fn validate(&self, driver: &Invocation) -> Result<(), String> {
        if !canonical::valid_digest(&self.invocation_id)
            || self.invocation_id == driver.id
            || self.invocation_id == driver.wrapper_run_id
            || self.process_id == 0
            || self.parent_process_id == Some(0)
            || self.argv.is_empty()
            || self.argv.len() > 65536
            || self.argv.iter().any(|value| {
                value.len() > 2 << 20
                    || value.len() % 2 != 0
                    || !value
                        .bytes()
                        .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
            })
            || self
                .preparation_receipt_sha256
                .as_ref()
                .is_some_and(|value| !canonical::valid_digest(value))
        {
            return Err("invalid intercepted launch association".into());
        }
        Ok(())
    }
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
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut encoded = String::with_capacity(value.as_bytes().len() * 2);
    for byte in value.as_bytes() {
        encoded.push(char::from(HEX[usize::from(byte >> 4)]));
        encoded.push(char::from(HEX[usize::from(byte & 15)]));
    }
    encoded
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
                &self.id,
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
        if let Some(launch) = &self.launch {
            launch.validate(&self.invocation)?;
        }
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
        launch: Some(Launch::current(None)?),
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

pub(crate) fn renew_launch(
    prepared: &mut super::prepare::Prepared,
    receipt_sha256: String,
) -> Result<(), String> {
    let mut origin =
        from_preparation(prepared)?.ok_or("standalone preparation has no intercepted origin")?;
    origin.launch = Some(Launch::current(Some(receipt_sha256))?);
    origin.validate()?;
    let mut provenance: serde_json::Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    provenance["origin"] = serde_json::to_value(origin).map_err(|error| error.to_string())?;
    prepared.provenance = canonical::encode(&provenance)?;
    Ok(())
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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub module_cache_root: Option<std::path::PathBuf>,
}

pub(crate) fn authenticate_driver(
    path: &Path,
    invocation_digest: &str,
    invocation: &Invocation,
    module: &Path,
    module_bytes: &[u8],
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
    {
        return Err("module is not associated with the registered live pinned Go driver".into());
    }
    open::pin(&registration.module_root)?;
    if std::fs::canonicalize(module)
        .map_err(|error| error.to_string())?
        .starts_with(
            std::fs::canonicalize(&registration.module_root).map_err(|error| error.to_string())?,
        )
    {
        return Ok(());
    }
    let cache = registration
        .module_cache_root
        .as_deref()
        .ok_or("module is not associated with the registered live pinned Go driver")?;
    cache::authenticate(cache, module, module_bytes, invocation)
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

#[cfg(test)]
#[path = "origin_cache_tests.rs"]
mod cache_tests;
