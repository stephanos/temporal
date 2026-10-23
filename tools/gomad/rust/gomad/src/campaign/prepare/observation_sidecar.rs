use super::Prepared;
use crate::evidence::{
    canonical,
    store::{durable_directory, open as store_open},
};
use gomad_wasmhost::linked;
use rustix::fs::{self, AtFlags, FileType, Mode, OFlags};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::File,
    io::Write,
    path::Path,
    sync::atomic::{AtomicU64, Ordering},
};

const DIRECTORY: &str = "preparation-observations";
const SCHEMA: &str = "gomad.native-preparation-observation-receipt/v1";
const RECEIPT_LIMIT: u64 = 16 << 20;
const BLOB_LIMIT: u64 = 128 << 20;
static SEQUENCE: AtomicU64 = AtomicU64::new(0);

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Blob {
    bytes: u64,
    sha256: String,
}

#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Receipt {
    schema: String,
    owner_pid: u32,
    host_sha256: String,
    helper_sha256: String,
    prepared_record_sha256: String,
    module_sha256: String,
    provenance_sha256: String,
    build_key: String,
    compiler_sha256: String,
    sources_sha256: String,
    files: BTreeMap<String, Blob>,
}

fn commitment(prepared: &Prepared, record: &str) -> Result<Receipt, String> {
    if !canonical::valid_digest(record) || !canonical::valid_digest(&prepared.toolchain) {
        return Err("invalid preparation observation record or build key".into());
    }
    if prepared.provenance.len() as u64 > RECEIPT_LIMIT {
        return Err("preparation observation provenance exceeds bound".into());
    }
    let provenance: serde_json::Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    if canonical::encode(&provenance)? != prepared.provenance {
        return Err("preparation observation provenance is not canonical".into());
    }
    let inputs = provenance
        .get("inputs")
        .filter(|inputs| inputs.is_object())
        .ok_or("missing preparation observation inputs")?;
    if provenance
        .get("build_key")
        .and_then(serde_json::Value::as_str)
        != Some(prepared.toolchain.as_str())
        || canonical::digest(&canonical::encode(inputs)?) != prepared.toolchain
    {
        return Err("preparation observation build key preimage changed".into());
    }
    let identity = |name: &str| -> Result<String, String> {
        let value = provenance
            .get("inputs")
            .and_then(|inputs| inputs.get(name))
            .ok_or("missing preparation observation inventory")?;
        let inventory: BTreeMap<String, String> =
            serde_json::from_value(value.clone()).map_err(|error| error.to_string())?;
        if inventory.is_empty()
            || inventory.iter().any(|(path, digest)| {
                !Path::new(path).is_absolute() || !canonical::valid_digest(digest)
            })
        {
            return Err("invalid preparation observation inventory".into());
        }
        Ok(canonical::digest(&canonical::encode(value)?))
    };
    Ok(Receipt {
        schema: SCHEMA.into(),
        owner_pid: std::process::id(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        prepared_record_sha256: record.into(),
        module_sha256: canonical::digest(&prepared.module),
        provenance_sha256: canonical::digest(&prepared.provenance),
        build_key: prepared.toolchain.clone(),
        compiler_sha256: identity("compiler")?,
        sources_sha256: identity("sources")?,
        files: BTreeMap::new(),
    })
}

fn valid_name(name: &str) -> Result<(), String> {
    store_open::components(name)?;
    if name == "receipt.json" || name.starts_with("receipt.json/") {
        return Err("preparation observation payload conflicts with receipt".into());
    }
    Ok(())
}

fn add_size(total: u64, bytes: u64) -> Result<u64, String> {
    if bytes > BLOB_LIMIT {
        return Err("preparation observation blob exceeds bound".into());
    }
    total
        .checked_add(bytes)
        .ok_or_else(|| "preparation observation size overflow".into())
}

pub(super) fn publish(
    campaign: &Path,
    prepared: &Prepared,
    prepared_record_sha256: &str,
) -> Result<(), String> {
    if prepared.auxiliary.is_empty() {
        return Ok(());
    }
    if prepared.auxiliary.len() > 32 {
        return Err("preparation observation file table exceeds bound".into());
    }
    let mut receipt = commitment(prepared, prepared_record_sha256)?;
    let mut total = 0;
    for (name, bytes) in &prepared.auxiliary {
        valid_name(name)?;
        total = add_size(total, bytes.len() as u64)?;
        receipt.files.insert(
            name.clone(),
            Blob {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(bytes),
            },
        );
    }
    let encoded = canonical::encode(&receipt)?;
    if encoded.len() as u64 > RECEIPT_LIMIT {
        return Err("preparation observation receipt exceeds bound".into());
    }
    let campaign = durable_directory(campaign, |_| Ok(()))?;
    let name = format!(
        ".preparation-observations-{}-{}",
        std::process::id(),
        SEQUENCE.fetch_add(1, Ordering::Relaxed)
    );
    fs::mkdirat(&campaign, name.as_str(), Mode::from_raw_mode(0o700))
        .map_err(|error| error.to_string())?;
    let result = (|| {
        let stage = store_open::child(&campaign, &name, true, false)?;
        for (name, bytes) in &prepared.auxiliary {
            write(&stage, name, bytes)?;
        }
        write(&stage, "receipt.json", &encoded)?;
        let retained = read(&stage, prepared, prepared_record_sha256)?;
        if retained != prepared.auxiliary {
            return Err("staged preparation observation set changed".into());
        }
        sync_tree(&stage)?;
        fs::renameat_with(
            &campaign,
            name.as_str(),
            &campaign,
            DIRECTORY,
            fs::RenameFlags::NOREPLACE,
        )
        .map_err(|error| format!("publish preparation observations (published=false): {error}"))?;
        campaign
            .sync_all()
            .map_err(|error| format!("publish preparation observations (published=true): {error}"))
    })();
    let cleanup = match fs::statat(&campaign, name.as_str(), AtFlags::SYMLINK_NOFOLLOW) {
        Err(rustix::io::Errno::NOENT) => Ok(()),
        Err(error) => Err(error.to_string()),
        Ok(_) => remove_tree(&campaign, &name),
    };
    match (result, cleanup) {
        (result, Ok(())) => result,
        (Ok(()), Err(cleanup)) => Err(format!("cleanup preparation observations: {cleanup}")),
        (Err(primary), Err(cleanup)) => Err(format!(
            "{primary}; cleanup preparation observations: {cleanup}"
        )),
    }
}

pub(super) fn open(
    campaign: &Path,
    prepared: &Prepared,
    prepared_record_sha256: &str,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let campaign = store_open::pin(campaign)?;
    match fs::statat(&campaign, DIRECTORY, AtFlags::SYMLINK_NOFOLLOW) {
        Err(rustix::io::Errno::NOENT) => return Ok(BTreeMap::new()),
        Err(error) => return Err(error.to_string()),
        Ok(_) => (),
    }
    read(
        &store_open::child(&campaign, DIRECTORY, true, false)?,
        prepared,
        prepared_record_sha256,
    )
}

fn read(
    directory: &File,
    prepared: &Prepared,
    record: &str,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let bytes = store_open::read(directory, "receipt.json", RECEIPT_LIMIT, false)?;
    let receipt: Receipt = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if canonical::encode(&receipt)? != bytes
        || receipt.owner_pid == 0
        || receipt.files.is_empty()
        || receipt.files.len() > 32
    {
        return Err("noncanonical or empty preparation observation receipt".into());
    }
    let mut expected = commitment(prepared, record)?;
    expected.owner_pid = receipt.owner_pid;
    expected.files.clone_from(&receipt.files);
    if expected != receipt {
        return Err("preparation observation commitment changed".into());
    }
    let mut seen = BTreeSet::new();
    walk(directory, "", &receipt.files, &mut seen)?;
    if seen.len() != receipt.files.len() {
        return Err("preparation observation payload missing".into());
    }
    let mut total = 0;
    let mut payloads = BTreeMap::new();
    for (name, blob) in receipt.files {
        valid_name(&name)?;
        total = add_size(total, blob.bytes)?;
        if !canonical::valid_digest(&blob.sha256) {
            return Err("invalid preparation observation digest".into());
        }
        let bytes = store_open::read(directory, &name, blob.bytes, false)?;
        if bytes.len() as u64 != blob.bytes || canonical::digest(&bytes) != blob.sha256 {
            return Err("preparation observation payload changed".into());
        }
        payloads.insert(name, bytes);
    }
    Ok(payloads)
}

fn walk(
    directory: &File,
    prefix: &str,
    files: &BTreeMap<String, Blob>,
    seen: &mut BTreeSet<String>,
) -> Result<(), String> {
    for name in store_open::entries(directory)? {
        if prefix.is_empty() && name == "receipt.json" {
            continue;
        }
        let relative = if prefix.is_empty() {
            name.clone()
        } else {
            format!("{prefix}/{name}")
        };
        if files.contains_key(&relative) {
            store_open::child(directory, &name, false, false)?;
            seen.insert(relative);
        } else if files
            .keys()
            .any(|path| path.starts_with(&format!("{relative}/")))
        {
            walk(
                &store_open::child(directory, &name, true, false)?,
                &relative,
                files,
                seen,
            )?;
        } else {
            return Err("unlisted preparation observation entry".into());
        }
    }
    Ok(())
}

fn write(root: &File, name: &str, bytes: &[u8]) -> Result<(), String> {
    let parts = store_open::components(name)?;
    let mut directory = root.try_clone().map_err(|error| error.to_string())?;
    for part in &parts[..parts.len() - 1] {
        match fs::mkdirat(&directory, *part, Mode::from_raw_mode(0o700)) {
            Ok(()) | Err(rustix::io::Errno::EXIST) => (),
            Err(error) => return Err(error.to_string()),
        }
        directory = store_open::child(&directory, part, true, false)?;
    }
    let mut file = File::from(
        fs::openat(
            &directory,
            parts[parts.len() - 1],
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())
}

fn sync_tree(directory: &File) -> Result<(), String> {
    for name in store_open::entries(directory)? {
        let stat = fs::statat(directory, name.as_str(), AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if FileType::from_raw_mode(stat.st_mode) == FileType::Directory {
            sync_tree(&store_open::child(directory, &name, true, false)?)?;
        }
    }
    directory.sync_all().map_err(|error| error.to_string())
}

fn remove_tree(parent: &File, name: &str) -> Result<(), String> {
    let directory = store_open::child(parent, name, true, false)?;
    for name in store_open::entries(&directory)? {
        let stat = fs::statat(&directory, name.as_str(), AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if FileType::from_raw_mode(stat.st_mode) == FileType::Directory {
            remove_tree(&directory, &name)?;
        } else {
            store_open::child(&directory, &name, false, false)?;
            fs::unlinkat(&directory, name.as_str(), AtFlags::empty())
                .map_err(|error| error.to_string())?;
        }
    }
    fs::unlinkat(parent, name, AtFlags::REMOVEDIR).map_err(|error| error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        evidence::{
            record::Limits,
            search_context::tests::{publication, store},
            store::Opened,
        },
        replay::preflight::Configuration,
    };
    use serde_json::json;
    use std::{
        fs as hostfs,
        os::unix::fs::{DirBuilderExt, PermissionsExt},
        path::PathBuf,
    };

    struct Campaign(PathBuf);
    impl Campaign {
        fn new() -> Self {
            let root = std::env::temp_dir().join(format!(
                "gomad-observation-sidecar-{}-{}",
                std::process::id(),
                SEQUENCE.fetch_add(1, Ordering::Relaxed)
            ));
            hostfs::DirBuilder::new().mode(0o700).create(&root).unwrap();
            Self(root)
        }
    }
    impl Drop for Campaign {
        fn drop(&mut self) {
            hostfs::remove_dir_all(&self.0).unwrap();
        }
    }
    fn prepared(observation: u64) -> Prepared {
        let inputs = json!({"compiler":{"/compiler":"1".repeat(64)}, "sources":{"/source.go":"2".repeat(64)}});
        let toolchain = canonical::digest(&canonical::encode(&inputs).unwrap());
        let provenance =
            canonical::encode(&json!({"inputs":inputs,"build_key":toolchain})).unwrap();
        // These observation values are fixtures, not claims about a compiler execution.
        let raw = canonical::encode(&json!({"owner_pid":std::process::id(),"observation_id":observation,"timestamp_fixture":observation})).unwrap();
        Prepared {
            auxiliary: BTreeMap::from([(
                "target/preparation-inputs/admission/metadata.json".into(),
                raw,
            )]),
            module: publication(0).payloads["target/module.wasm"].clone(),
            configuration: Configuration {
                engine: "fixture".into(),
                profile: "gomad.wasi-cooperative/v1".into(),
                limits: Limits {
                    fuel: 1,
                    memory_bytes: 65536,
                    wall_millis: 1,
                    output_bytes: 4096,
                    evidence_bytes: 4096,
                },
                wall_nanos: None,
                arguments: Vec::new(),
                environment: BTreeMap::new(),
                derive_seed_environment: false,
                network: None,
                runtime_sha256: None,
                model: None,
                namespace: None,
                storage: None,
            },
            toolchain,
            choice_identity: None,
            scenario: None,
            provenance,
            network: None,
            soft_assertion_report_only: false,
        }
    }
    fn receipt_path(campaign: &Campaign) -> PathBuf {
        campaign.0.join(DIRECTORY).join("receipt.json")
    }

    #[test]
    fn same_actual_store_record_remains_idempotent_with_distinct_campaign_observations() {
        let store = store();
        let first_campaign = Campaign::new();
        let second_campaign = Campaign::new();
        let first_prepared = prepared(101);
        let second_prepared = prepared(202);
        let first_publication = publication(0);
        let record = first_publication.record.hash().unwrap();
        let canonical_payloads = first_publication.payloads.clone();
        let first = store.publish(first_publication).unwrap();
        publish(&first_campaign.0, &first_prepared, &record).unwrap();
        let second_publication = publication(0);
        assert_eq!(canonical_payloads, second_publication.payloads);
        assert_eq!(record, second_publication.record.hash().unwrap());
        let second = store.publish(second_publication).unwrap();
        publish(&second_campaign.0, &second_prepared, &record).unwrap();
        assert_eq!(first.path, second.path);
        let reopened = Opened::open(&second.path).unwrap();
        assert_eq!(reopened.manifest().record.hash().unwrap(), record);
        assert!(
            reopened
                .manifest()
                .files
                .keys()
                .all(|name| !name.starts_with("target/preparation-inputs/"))
        );
        for (name, bytes) in canonical_payloads {
            assert_eq!(reopened.payload(&name, bytes.len() as u64).unwrap(), bytes);
        }
        let first_raw = open(&first_campaign.0, &first_prepared, &record).unwrap();
        let second_raw = open(&second_campaign.0, &second_prepared, &record).unwrap();
        assert_eq!(first_raw, first_prepared.auxiliary);
        assert_eq!(second_raw, second_prepared.auxiliary);
        assert_ne!(first_raw, second_raw);
        for campaign in [&first_campaign, &second_campaign] {
            let bytes = hostfs::read(receipt_path(campaign)).unwrap();
            let receipt: Receipt = serde_json::from_slice(&bytes).unwrap();
            assert_eq!(receipt.owner_pid, std::process::id());
            assert_eq!(receipt.prepared_record_sha256, record);
            assert_eq!(
                hostfs::metadata(receipt_path(campaign))
                    .unwrap()
                    .permissions()
                    .mode()
                    & 0o777,
                0o600
            );
        }
        drop(reopened);
        hostfs::remove_dir_all(&store.root).unwrap();
    }

    #[test]
    fn absent_legacy_is_optional_but_present_sidecar_fails_closed() {
        let record = "3".repeat(64);
        for fault in [
            "tamper",
            "missing",
            "unknown",
            "duplicate",
            "extra",
            "symlink",
            "owner",
            "preimage",
            "binding",
            "size",
            "receipt-missing",
        ] {
            let campaign = Campaign::new();
            let mut prepared = prepared(101);
            assert!(open(&campaign.0, &prepared, &record).unwrap().is_empty());
            publish(&campaign.0, &prepared, &record).unwrap();
            let path = receipt_path(&campaign);
            let raw = hostfs::read(&path).unwrap();
            let mut receipt: serde_json::Value = serde_json::from_slice(&raw).unwrap();
            let payload = campaign
                .0
                .join(DIRECTORY)
                .join(prepared.auxiliary.keys().next().unwrap());
            match fault {
                "tamper" => hostfs::write(&payload, b"changed").unwrap(),
                "missing" => hostfs::remove_file(&payload).unwrap(),
                "unknown" => {
                    receipt["extra"] = json!(true);
                    hostfs::write(&path, canonical::encode(&receipt).unwrap()).unwrap();
                }
                "duplicate" => {
                    let mut duplicate = b"{\"owner_pid\":1,".to_vec();
                    duplicate.extend_from_slice(&raw[1..]);
                    hostfs::write(&path, duplicate).unwrap();
                }
                "extra" => {
                    hostfs::write(campaign.0.join(DIRECTORY).join("unlisted"), b"extra").unwrap()
                }
                "symlink" => {
                    hostfs::remove_file(&payload).unwrap();
                    std::os::unix::fs::symlink(&path, &payload).unwrap();
                }
                "owner" => {
                    receipt["owner_pid"] = json!(0);
                    hostfs::write(&path, canonical::encode(&receipt).unwrap()).unwrap();
                }
                "preimage" => {
                    prepared.provenance = canonical::encode(
                        &json!({"inputs":{"sources":{}},"build_key":prepared.toolchain}),
                    )
                    .unwrap()
                }
                "binding" => {
                    receipt["module_sha256"] = json!("4".repeat(64));
                    hostfs::write(&path, canonical::encode(&receipt).unwrap()).unwrap();
                }
                "size" => {
                    receipt["files"][prepared.auxiliary.keys().next().unwrap()]["bytes"] =
                        json!(BLOB_LIMIT + 1);
                    hostfs::write(&path, canonical::encode(&receipt).unwrap()).unwrap();
                }
                "receipt-missing" => hostfs::remove_file(&path).unwrap(),
                _ => unreachable!(),
            }
            assert!(open(&campaign.0, &prepared, &record).is_err(), "{fault}");
        }
    }

    #[test]
    fn immutable_publication_does_not_clobber_and_cleans_private_stage() {
        let campaign = Campaign::new();
        let record = "3".repeat(64);
        let first = prepared(101);
        publish(&campaign.0, &first, &record).unwrap();
        let receipt = hostfs::read(receipt_path(&campaign)).unwrap();
        assert!(
            publish(&campaign.0, &prepared(202), &record)
                .unwrap_err()
                .contains("published=false")
        );
        assert_eq!(hostfs::read(receipt_path(&campaign)).unwrap(), receipt);
        assert_eq!(open(&campaign.0, &first, &record).unwrap(), first.auxiliary);
        assert_eq!(hostfs::read_dir(&campaign.0).unwrap().count(), 1);
    }

    #[test]
    fn malformed_preimage_refused_empty_legacy_skipped_and_per_blob_bounds_preserved() {
        let campaign = Campaign::new();
        let mut invalid = prepared(101);
        invalid.provenance = b"{}".to_vec();
        assert!(publish(&campaign.0, &invalid, &"3".repeat(64)).is_err());
        assert_eq!(hostfs::read_dir(&campaign.0).unwrap().count(), 0);
        invalid.auxiliary.clear();
        publish(&campaign.0, &invalid, &"3".repeat(64)).unwrap();
        assert_eq!(hostfs::read_dir(&campaign.0).unwrap().count(), 0);
        assert_eq!(add_size(BLOB_LIMIT, BLOB_LIMIT).unwrap(), 2 * BLOB_LIMIT);
        assert!(add_size(0, BLOB_LIMIT + 1).is_err());
        assert!(add_size(u64::MAX, 1).is_err());
    }
}
