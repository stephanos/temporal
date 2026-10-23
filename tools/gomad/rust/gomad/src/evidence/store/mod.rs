mod lineage;
pub(crate) mod open;
mod pool;
mod publish;
#[cfg(test)]
mod record_intake_tests;

use super::{canonical, record::Record};
pub use open::Opened;
pub use publish::{Publication, Published, Store};
pub(crate) use publish::{artifact_name, durable_directory, matches_name};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FileEntry {
    pub bytes: u64,
    pub sha256: String,
    pub module: bool,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    pub schema: String,
    pub record: Record,
    pub record_sha256: String,
    pub files: BTreeMap<String, FileEntry>,
}

impl Manifest {
    fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.artifact-manifest/v2" || self.record.hash()? != self.record_sha256
        {
            return Err("manifest record identity mismatch".into());
        }
        let mut modules = 0;
        for (name, entry) in &self.files {
            open::components(name)?;
            if matches!(name.as_str(), "manifest.json" | "storage.json")
                || !canonical::valid_digest(&entry.sha256)
            {
                return Err("invalid manifest payload entry".into());
            }
            if entry.module != (name == "target/module.wasm") {
                return Err("module payload kind mismatch".into());
            }
            if entry.module {
                modules += 1;
                if name != "target/module.wasm" || entry.sha256 != self.record.identity.target {
                    return Err("module payload identity mismatch".into());
                }
            }
        }
        if modules > 1 {
            return Err("multiple module payloads".into());
        }
        if let Some(entry) = self.files.get("record.json")
            && (entry.sha256 != self.record_sha256
                || entry.bytes != self.record.bytes()?.len() as u64)
        {
            return Err("canonical record payload identity mismatch".into());
        }
        for (name, stream) in &self.record.streams {
            let entry = self
                .files
                .get(name)
                .ok_or("missing record stream payload")?;
            if entry.bytes != stream.bytes || entry.sha256 != stream.sha256 {
                return Err("record stream payload identity mismatch".into());
            }
        }
        Ok(())
    }

    fn stored_bytes(&self, manifest_bytes: u64) -> Result<u64, String> {
        self.files.values().try_fold(
            manifest_bytes + lineage::ENVELOPE.len() as u64,
            |total, entry| {
                total
                    .checked_add(entry.bytes)
                    .ok_or_else(|| "retained bytes overflow".into())
            },
        )
    }
}
