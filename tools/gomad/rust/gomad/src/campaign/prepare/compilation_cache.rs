use crate::evidence::{
    canonical,
    store::{durable_directory, open},
};
use gomad_wasmhost::linked;
use rustix::fs::{self, AtFlags, Mode, OFlags};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::BTreeMap,
    fs::File,
    io::{Read, Write},
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
};

const MODULE_LIMIT: u64 = 512 << 20;
const RECEIPT_LIMIT: u64 = 16 << 20;
const AUXILIARY_LIMIT: u64 = 128 << 20;
const SCHEMA: &str = "gomad.compilation-cache/v1";
static SEQUENCE: AtomicU64 = AtomicU64::new(0);

#[derive(Debug, PartialEq, Eq)]
pub(super) struct Cached {
    pub module: Vec<u8>,
    pub auxiliary: BTreeMap<String, Vec<u8>>,
}

#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Blob {
    sha256: String,
    bytes: u64,
}

impl Blob {
    fn of(bytes: &[u8]) -> Self {
        Self {
            sha256: canonical::digest(bytes),
            bytes: bytes.len() as u64,
        }
    }
}

#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Receipt {
    schema: String,
    version: u32,
    key: String,
    inputs: Value,
    module: Blob,
    metadata: linked::ModuleMetadata,
    auxiliary: BTreeMap<String, Blob>,
}

pub(super) fn key(inputs: &Value) -> Result<String, String> {
    Ok(canonical::digest(&canonical::encode(inputs)?))
}

fn check_key(build_key: &str, inputs: &Value) -> Result<(), String> {
    if !canonical::valid_digest(build_key) || key(inputs)? != build_key {
        return Err("WASM compilation cache key mismatch".into());
    }
    Ok(())
}

fn directory(root: &Path, build_key: &str, create: bool) -> Result<Option<File>, String> {
    if !root.is_absolute() {
        return Err("absolute WASM compilation cache root is required".into());
    }
    let mut directory = if create && !root.exists() {
        durable_directory(root, |_| Ok(()))?
    } else {
        match fs::statat(fs::CWD, root, AtFlags::SYMLINK_NOFOLLOW) {
            Err(rustix::io::Errno::NOENT) => return Ok(None),
            Err(error) => return Err(error.to_string()),
            Ok(before) => {
                if rustix::fs::FileType::from_raw_mode(before.st_mode)
                    != rustix::fs::FileType::Directory
                    || before.st_uid != rustix::process::geteuid().as_raw()
                    || before.st_mode & 0o022 != 0
                {
                    return Err("compilation cache parent type, ownership or mode violation".into());
                }
                let opened = File::from(
                    fs::open(
                        root,
                        OFlags::RDONLY | OFlags::DIRECTORY | OFlags::CLOEXEC | OFlags::NOFOLLOW,
                        Mode::empty(),
                    )
                    .map_err(|error| error.to_string())?,
                );
                let after = fs::fstat(&opened).map_err(|error| error.to_string())?;
                if before.st_dev != after.st_dev
                    || before.st_ino != after.st_ino
                    || before.st_mode != after.st_mode
                    || before.st_uid != after.st_uid
                {
                    return Err("compilation cache parent changed while opening".into());
                }
                opened
            }
        }
    };
    for part in ["compilation-v1", build_key] {
        if create {
            match fs::mkdirat(&directory, part, Mode::from_raw_mode(0o700)) {
                Ok(()) => directory.sync_all().map_err(|error| error.to_string())?,
                Err(rustix::io::Errno::EXIST) => (),
                Err(error) => return Err(error.to_string()),
            }
        } else {
            match fs::statat(&directory, part, AtFlags::SYMLINK_NOFOLLOW) {
                Err(rustix::io::Errno::NOENT) => return Ok(None),
                Err(error) => return Err(error.to_string()),
                Ok(_) => (),
            }
        }
        directory = open::child(&directory, part, true, false)?;
    }
    Ok(Some(directory))
}

pub(super) fn load(root: &Path, build_key: &str, inputs: &Value) -> Result<Option<Cached>, String> {
    check_key(build_key, inputs)?;
    let Some(directory) = directory(root, build_key, false)? else {
        return Ok(None);
    };
    match fs::statat(&directory, "receipt.json", AtFlags::SYMLINK_NOFOLLOW) {
        Err(rustix::io::Errno::NOENT) => return Ok(None),
        Err(error) => return Err(error.to_string()),
        Ok(_) => (),
    }
    // A missing marker is interrupted publication; a present corrupt marker fails closed.
    let receipt: Receipt = serde_json::from_slice(&open::read(
        &directory,
        "receipt.json",
        RECEIPT_LIMIT,
        false,
    )?)
    .map_err(|error| error.to_string())?;
    if receipt.schema != SCHEMA
        || receipt.version != 1
        || receipt.key != build_key
        || receipt.inputs != *inputs
    {
        return Err("WASM compilation cache identity changed".into());
    }
    if receipt.auxiliary.len() > 32 {
        return Err("compilation auxiliary file table exceeds bound".into());
    }
    let module = read_module(&directory, "module.wasm")?;
    if receipt.module != Blob::of(&module)
        || receipt.metadata
            != linked::module_metadata(&module).map_err(|error| error.to_string())?
    {
        return Err("cached WASM import, memory or module identity changed".into());
    }
    let mut auxiliary = BTreeMap::new();
    let mut total = 0_u64;
    for (name, blob) in receipt.auxiliary {
        open::components(&name)?;
        if !canonical::valid_digest(&blob.sha256) {
            return Err("invalid compilation auxiliary digest".into());
        }
        total = auxiliary_size(total, blob.bytes)?;
        let bytes = open::read(
            &directory,
            &format!("aux-{}", blob.sha256),
            blob.bytes,
            false,
        )?;
        if Blob::of(&bytes) != blob {
            return Err("compilation auxiliary identity changed".into());
        }
        auxiliary.insert(name, bytes);
    }
    Ok(Some(Cached { module, auxiliary }))
}

pub(super) fn build_directory(root: &Path, build_key: &str) -> Result<PathBuf, String> {
    if !canonical::valid_digest(build_key) {
        return Err("invalid compilation cache key".into());
    }
    let directory =
        directory(root, build_key, true)?.ok_or("compilation cache directory missing")?;
    match fs::mkdirat(&directory, "objects", Mode::from_raw_mode(0o700)) {
        Ok(()) => directory.sync_all().map_err(|error| error.to_string())?,
        Err(rustix::io::Errno::EXIST) => (),
        Err(error) => return Err(error.to_string()),
    }
    open::child(&directory, "objects", true, false)?;
    Ok(root.join("compilation-v1").join(build_key))
}

pub(super) fn publish_with_mode(
    root: &Path,
    build_key: &str,
    inputs: &Value,
    module: &[u8],
    module_mode: u32,
    auxiliary: BTreeMap<String, Vec<u8>>,
    mut checkpoint: impl FnMut() -> Result<(), String>,
) -> Result<Cached, String> {
    check_key(build_key, inputs)?;
    let directory =
        directory(root, build_key, true)?.ok_or("compilation cache directory missing")?;
    loop {
        checkpoint()?;
        match fs::flock(&directory, fs::FlockOperation::NonBlockingLockExclusive) {
            Ok(()) => break,
            Err(rustix::io::Errno::WOULDBLOCK) => std::thread::yield_now(),
            Err(error) => return Err(error.to_string()),
        }
    }
    if let Some(cached) = load(root, build_key, inputs)? {
        if cached.module != module {
            return Err("compiled WASM module differs from existing compilation identity".into());
        }
        return Ok(cached);
    }
    publish_with(
        root,
        build_key,
        inputs,
        module,
        module_mode,
        auxiliary,
        &mut checkpoint,
        |_, _, _| Ok(()),
    )
}

#[expect(
    clippy::too_many_arguments,
    clippy::too_many_lines,
    reason = "Publication keeps the ordered receipt-last fault boundaries in one transaction"
)]
fn publish_with(
    root: &Path,
    build_key: &str,
    inputs: &Value,
    module: &[u8],
    module_mode: u32,
    auxiliary: BTreeMap<String, Vec<u8>>,
    checkpoint: &mut impl FnMut() -> Result<(), String>,
    mut fault: impl FnMut(&str, &File, &str) -> Result<(), String>,
) -> Result<Cached, String> {
    check_key(build_key, inputs)?;
    if module_mode & !0o777 != 0 {
        return Err("invalid compiled module mode".into());
    }
    checkpoint()?;
    if module.len() as u64 > MODULE_LIMIT {
        return Err("compiled WASM module exceeds bound".into());
    }
    if auxiliary.len() > 32 {
        return Err("compilation auxiliary file table exceeds bound".into());
    }
    let metadata = linked::module_metadata(module).map_err(|error| error.to_string())?;
    let mut total = 0_u64;
    for (name, bytes) in &auxiliary {
        open::components(name)?;
        total = auxiliary_size(total, bytes.len() as u64)?;
    }
    let receipt = Receipt {
        schema: SCHEMA.into(),
        version: 1,
        key: build_key.into(),
        inputs: inputs.clone(),
        module: Blob::of(module),
        metadata,
        auxiliary: auxiliary
            .iter()
            .map(|(name, bytes)| (name.clone(), Blob::of(bytes)))
            .collect(),
    };
    let encoded = canonical::encode(&receipt)?;
    if encoded.len() as u64 > RECEIPT_LIMIT {
        return Err("compilation receipt exceeds bound".into());
    }
    let directory =
        directory(root, build_key, true)?.ok_or("compilation cache directory missing")?;
    let mut stages = Vec::new();
    let result = (|| {
        let module_stage = stage(&directory, "module", module, module_mode, &mut stages)?;
        fault("module-stage", &directory, &module_stage)?;
        let receipt_stage = stage(&directory, "receipt", &encoded, 0o600, &mut stages)?;
        fault("receipt-stage", &directory, &receipt_stage)?;
        let mut auxiliary_stages = BTreeMap::new();
        for bytes in auxiliary.values() {
            let destination = format!("aux-{}", canonical::digest(bytes));
            if let std::collections::btree_map::Entry::Vacant(e) =
                auxiliary_stages.entry(destination)
            {
                let raw_name = stage(&directory, "aux", bytes, 0o600, &mut stages)?;
                fault("aux-stage", &directory, &raw_name)?;
                e.insert(raw_name);
            }
        }
        if read_module(&directory, &module_stage)? != module {
            return Err("staged WASM module identity changed".into());
        }
        let validated_receipt: Receipt = serde_json::from_slice(&open::read(
            &directory,
            &receipt_stage,
            RECEIPT_LIMIT,
            false,
        )?)
        .map_err(|error| error.to_string())?;
        if validated_receipt != receipt {
            return Err("staged WASM receipt identity changed".into());
        }
        for (destination, staged) in &auxiliary_stages {
            let bytes = open::read(&directory, staged, AUXILIARY_LIMIT, false)?;
            if destination != &format!("aux-{}", canonical::digest(&bytes)) {
                return Err("staged compilation auxiliary identity changed".into());
            }
        }
        fault("validated", &directory, &receipt_stage)?;
        checkpoint()?;
        for (destination, staged) in &auxiliary_stages {
            rename(&directory, staged, destination)?;
        }
        rename(&directory, &module_stage, "module.wasm")?;
        directory.sync_all().map_err(|error| {
            format!("WASM cache module published=true; receipt published=false: {error}")
        })?;
        fault("module-published", &directory, &receipt_stage).map_err(|error| {
            format!("WASM cache module published=true; receipt published=false: {error}")
        })?;
        // The cancellation boundary precedes all renames, so cancellation never fabricates a marker.
        rename(&directory, &receipt_stage, "receipt.json").map_err(|error| {
            format!("WASM cache module published=true; receipt published=false: {error}")
        })?;
        directory.sync_all().map_err(|error| {
            format!("WASM cache module published=true; receipt published=true: {error}")
        })?;
        Ok(Cached {
            module: module.to_vec(),
            auxiliary,
        })
    })();
    let mut cleanup = Vec::new();
    for name in stages.iter().rev() {
        if let Err(error) = fs::unlinkat(&directory, name.as_str(), AtFlags::empty())
            && error != rustix::io::Errno::NOENT
        {
            cleanup.push(format!("cleanup compilation stage {name}: {error}"));
        }
    }
    if cleanup.is_empty() {
        return result;
    }
    let cleanup = cleanup.join("; ");
    match result {
        Ok(_) => Err(cleanup),
        Err(primary) => Err(format!("{primary}; {cleanup}")),
    }
}

fn auxiliary_size(total: u64, bytes: u64) -> Result<u64, String> {
    if bytes > AUXILIARY_LIMIT {
        return Err("compilation auxiliary exceeds bound".into());
    }
    total
        .checked_add(bytes)
        .ok_or_else(|| "compilation auxiliary size overflow".into())
}

fn stage(
    directory: &File,
    prefix: &str,
    bytes: &[u8],
    mode: u32,
    stages: &mut Vec<String>,
) -> Result<String, String> {
    let name = format!(
        ".{prefix}-{}-{}",
        std::process::id(),
        SEQUENCE.fetch_add(1, Ordering::Relaxed)
    );
    let mut file = File::from(
        fs::openat(
            directory,
            name.as_str(),
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
            Mode::from_raw_mode(mode),
        )
        .map_err(|error| error.to_string())?,
    );
    stages.push(name.clone());
    fs::fchmod(&file, Mode::from_raw_mode(mode)).map_err(|error| error.to_string())?;
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())?;
    Ok(name)
}

fn rename(directory: &File, source: &str, destination: &str) -> Result<(), String> {
    if let Ok(stat) = fs::statat(directory, destination, AtFlags::SYMLINK_NOFOLLOW)
        && rustix::fs::FileType::from_raw_mode(stat.st_mode) == rustix::fs::FileType::Symlink
    {
        return Err("compilation destination is a symbolic link".into());
    }
    fs::renameat(directory, source, directory, destination).map_err(|error| error.to_string())
}

fn read_module(directory: &File, name: &str) -> Result<Vec<u8>, String> {
    let before = fs::statat(directory, name, AtFlags::SYMLINK_NOFOLLOW)
        .map_err(|error| error.to_string())?;
    if rustix::fs::FileType::from_raw_mode(before.st_mode) != rustix::fs::FileType::RegularFile
        || before.st_mode & 0o7000 != 0
        || before.st_nlink != 1
        || before.st_uid != rustix::process::geteuid().as_raw()
        || !u64::try_from(before.st_size).is_ok_and(|size| size <= MODULE_LIMIT)
    {
        return Err("invalid compilation module type, ownership, links, mode or size".into());
    }
    let mut file = File::from(
        fs::openat(
            directory,
            name,
            OFlags::RDONLY | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK,
            Mode::empty(),
        )
        .map_err(|error| error.to_string())?,
    );
    if !open::same(
        &before,
        &fs::fstat(&file).map_err(|error| error.to_string())?,
    ) {
        return Err("compilation module changed while opening".into());
    }
    let mut bytes = Vec::new();
    (&mut file)
        .take(MODULE_LIMIT + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > MODULE_LIMIT
        || !open::same(
            &before,
            &fs::fstat(&file).map_err(|error| error.to_string())?,
        )
        || !open::same(
            &before,
            &fs::statat(directory, name, AtFlags::SYMLINK_NOFOLLOW)
                .map_err(|error| error.to_string())?,
        )
    {
        return Err("compilation module changed while reading".into());
    }
    Ok(bytes)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use std::{
        fs as hostfs,
        os::unix::fs::{DirBuilderExt, PermissionsExt},
    };

    struct Scratch(PathBuf);
    impl Scratch {
        fn new() -> Self {
            let path = std::env::temp_dir().join(format!(
                "gomad-compilation-cache-{}-{}",
                std::process::id(),
                SEQUENCE.fetch_add(1, Ordering::Relaxed)
            ));
            hostfs::DirBuilder::new().mode(0o700).create(&path).unwrap();
            Self(path)
        }
    }
    impl Drop for Scratch {
        fn drop(&mut self) {
            hostfs::remove_dir_all(&self.0).unwrap();
        }
    }
    fn module() -> Vec<u8> {
        b"\0asm\x01\0\0\0\x05\x03\x01\x00\x01".to_vec()
    }
    fn auxiliary() -> BTreeMap<String, Vec<u8>> {
        BTreeMap::from([(
            "target/preparation-inputs/metadata.json".into(),
            b"raw command observation\0\xff".to_vec(),
        )])
    }
    fn replace(directory: &File, name: &str, bytes: &[u8]) {
        let mut file = File::from(
            fs::openat(
                directory,
                name,
                OFlags::WRONLY | OFlags::TRUNC,
                Mode::empty(),
            )
            .unwrap(),
        );
        file.write_all(bytes).unwrap();
    }
    fn fill(root: &Path, inputs: &Value) -> Cached {
        publish_with_mode(
            root,
            &key(inputs).unwrap(),
            inputs,
            &module(),
            0o755,
            auxiliary(),
            || Ok(()),
        )
        .unwrap()
    }

    #[test]
    fn auxiliary_bounds_apply_per_output_and_total_arithmetic_is_checked() {
        let first = auxiliary_size(0, AUXILIARY_LIMIT).unwrap();
        assert_eq!(first, AUXILIARY_LIMIT);
        assert_eq!(
            auxiliary_size(first, AUXILIARY_LIMIT).unwrap(),
            2 * AUXILIARY_LIMIT
        );
        assert_eq!(auxiliary_size(u64::MAX, 0).unwrap(), u64::MAX);
        assert!(
            auxiliary_size(0, AUXILIARY_LIMIT + 1)
                .unwrap_err()
                .contains("exceeds bound")
        );
        assert!(
            auxiliary_size(u64::MAX, 1)
                .unwrap_err()
                .contains("overflow")
        );
    }

    #[test]
    fn roundtrip_authenticates_module_metadata_and_all_raw_observations() {
        let root = Scratch::new();
        let inputs = json!({"source":"./package", "flags":["-trimpath"]});
        let build_key = key(&inputs).unwrap();
        assert_eq!(load(&root.0, &build_key, &inputs).unwrap(), None);
        let built = fill(&root.0, &inputs);
        assert_eq!(load(&root.0, &build_key, &inputs).unwrap(), Some(built));
        let entry = build_directory(&root.0, &build_key).unwrap();
        assert!(entry.join("objects").is_dir());
        for (name, mode) in [("module.wasm", 0o755), ("receipt.json", 0o600)] {
            assert_eq!(
                hostfs::metadata(entry.join(name))
                    .unwrap()
                    .permissions()
                    .mode()
                    & 0o777,
                mode
            );
        }
        assert_ne!(
            build_key,
            key(&json!({"source":"./other", "flags":["-trimpath"]})).unwrap()
        );
        hostfs::remove_file(entry.join("receipt.json")).unwrap();
        assert_eq!(load(&root.0, &build_key, &inputs).unwrap(), None);
    }

    #[test]
    fn existing_complete_entry_keeps_original_observations_and_refuses_different_module() {
        let root = Scratch::new();
        let inputs = json!({"source":"fixture"});
        let original = fill(&root.0, &inputs);
        let changed = BTreeMap::from([(
            "target/preparation-inputs/compile/metadata.json".into(),
            b"different owner".to_vec(),
        )]);
        let retained = publish_with_mode(
            &root.0,
            &key(&inputs).unwrap(),
            &inputs,
            &module(),
            0o755,
            changed.clone(),
            || Ok(()),
        )
        .unwrap();
        assert_eq!(retained, original);
        let mut different = module();
        different.extend([0, 2, 1, b'x']);
        assert!(
            publish_with_mode(
                &root.0,
                &key(&inputs).unwrap(),
                &inputs,
                &different,
                0o755,
                changed,
                || Ok(())
            )
            .is_err()
        );
        assert_eq!(
            load(&root.0, &key(&inputs).unwrap(), &inputs).unwrap(),
            Some(original)
        );
    }

    #[test]
    #[expect(
        clippy::needless_collect,
        reason = "All publishers must spawn before joining because they synchronize on a barrier"
    )]
    fn concurrent_publishers_reuse_one_complete_original_proof_and_waiting_cancellation_is_bounded()
    {
        let root = Scratch::new();
        let inputs = json!({"source":"concurrent"});
        let barrier = std::sync::Arc::new(std::sync::Barrier::new(4));
        let results = std::thread::scope(|scope| {
            let workers: Vec<_> = (0..4)
                .map(|owner| {
                    let barrier = barrier.clone();
                    let root = &root.0;
                    let inputs = &inputs;
                    scope.spawn(move || {
                        barrier.wait();
                        publish_with_mode(
                            root,
                            &key(inputs).unwrap(),
                            inputs,
                            &module(),
                            0o755,
                            BTreeMap::from([(
                                "target/preparation-inputs/compile/metadata.json".into(),
                                vec![owner],
                            )]),
                            || Ok(()),
                        )
                        .unwrap()
                    })
                })
                .collect();
            workers
                .into_iter()
                .map(|worker| worker.join().unwrap())
                .collect::<Vec<_>>()
        });
        assert!(results.iter().all(|value| *value == results[0]));
        let directory = directory(&root.0, &key(&inputs).unwrap(), false)
            .unwrap()
            .unwrap();
        fs::flock(&directory, fs::FlockOperation::LockExclusive).unwrap();
        let mut checks = 0;
        let error = publish_with_mode(
            &root.0,
            &key(&inputs).unwrap(),
            &inputs,
            &module(),
            0o755,
            auxiliary(),
            || {
                checks += 1;
                if checks == 2 {
                    Err("canceled waiting publisher".into())
                } else {
                    Ok(())
                }
            },
        )
        .unwrap_err();
        assert_eq!(error, "canceled waiting publisher");
        assert_eq!(
            load(&root.0, &key(&inputs).unwrap(), &inputs).unwrap(),
            Some(results.into_iter().next().unwrap())
        );
    }

    #[test]
    fn corruption_is_an_error_and_does_not_rebuild() {
        for corruption in [
            "module",
            "receipt",
            "schema",
            "version",
            "key",
            "inputs",
            "size",
            "imports",
            "memory",
            "unknown",
            "auxiliary",
            "symlink",
            "oversize",
        ] {
            let root = Scratch::new();
            let inputs = json!({"source":"fixture"});
            let build_key = key(&inputs).unwrap();
            fill(&root.0, &inputs);
            let entry = root.0.join("compilation-v1").join(&build_key);
            let receipt_path = entry.join("receipt.json");
            let mut receipt: Value =
                serde_json::from_slice(&hostfs::read(&receipt_path).unwrap()).unwrap();
            match corruption {
                "module" => hostfs::write(entry.join("module.wasm"), b"corrupt").unwrap(),
                "receipt" => hostfs::write(&receipt_path, b"corrupt").unwrap(),
                "auxiliary" => hostfs::write(
                    entry.join(format!(
                        "aux-{}",
                        canonical::digest(auxiliary().values().next().unwrap())
                    )),
                    b"corrupt",
                )
                .unwrap(),
                "symlink" => {
                    hostfs::remove_file(entry.join("module.wasm")).unwrap();
                    std::os::unix::fs::symlink(&receipt_path, entry.join("module.wasm")).unwrap();
                }
                "oversize" => {
                    hostfs::OpenOptions::new()
                        .write(true)
                        .open(&receipt_path)
                        .unwrap()
                        .set_len(RECEIPT_LIMIT + 1)
                        .unwrap();
                }
                other => {
                    match other {
                        "schema" => receipt["schema"] = json!("other"),
                        "version" => receipt["version"] = json!(2),
                        "key" => receipt["key"] = json!("other"),
                        "inputs" => receipt["inputs"] = json!({}),
                        "size" => receipt["module"]["bytes"] = json!(999),
                        "imports" => {
                            receipt["metadata"]["imports"] =
                                json!([{"module":"x","name":"y","params":[],"results":[]}]);
                        }
                        "memory" => receipt["metadata"]["initial_memory_pages"] = json!(2),
                        "unknown" => receipt["unknown"] = json!(true),
                        _ => unreachable!(),
                    }
                    hostfs::write(&receipt_path, serde_json::to_vec(&receipt).unwrap()).unwrap();
                }
            }
            assert!(load(&root.0, &build_key, &inputs).is_err(), "{corruption}");
        }
    }

    #[test]
    fn publication_fault_matrix_preserves_destinations_and_reports_cleanup_and_partial_commit() {
        for fault in [
            "module-stage",
            "receipt-stage",
            "module-validation",
            "receipt-validation",
            "cancel",
            "cleanup",
            "receipt-rename",
            "success",
        ] {
            let root = Scratch::new();
            let inputs = json!({"source":"fixture"});
            let build_key = key(&inputs).unwrap();
            let entry = build_directory(&root.0, &build_key).unwrap();
            hostfs::write(entry.join("module.wasm"), b"prior module").unwrap();
            hostfs::write(entry.join("receipt.json"), b"prior receipt").unwrap();
            let mut calls = 0;
            let mut checkpoint = || {
                calls += 1;
                if fault == "cancel" && calls == 2 {
                    Err("original cancellation identity".into())
                } else {
                    Ok(())
                }
            };
            let result = publish_with(
                &root.0,
                &build_key,
                &inputs,
                &module(),
                0o755,
                auxiliary(),
                &mut checkpoint,
                |event, directory, name| {
                    if event == "module-stage" {
                        assert_eq!(entry.join(name).parent().unwrap(), entry);
                        if fault == "module-validation" {
                            replace(directory, name, b"changed exact stage");
                        }
                        if fault == "cleanup" {
                            hostfs::remove_file(entry.join(name)).unwrap();
                            hostfs::create_dir(entry.join(name)).unwrap();
                            hostfs::write(entry.join(name).join("child"), b"blocks removal")
                                .unwrap();
                        }
                    }
                    if event == "receipt-stage" && fault == "receipt-validation" {
                        replace(directory, name, b"changed exact stage");
                    }
                    if event == "module-published" && fault == "receipt-rename" {
                        hostfs::remove_file(entry.join("receipt.json")).unwrap();
                        hostfs::create_dir(entry.join("receipt.json")).unwrap();
                    }
                    if (event == "module-stage" && fault == "module-stage")
                        || (event == "receipt-stage"
                            && matches!(fault, "receipt-stage" | "cleanup"))
                    {
                        return Err("cache fill failure".into());
                    }
                    Ok(())
                },
            );
            if fault == "success" {
                assert_eq!(result.unwrap().module, module());
                assert!(load(&root.0, &build_key, &inputs).unwrap().is_some());
            } else {
                let error = result.unwrap_err();
                if fault == "cancel" {
                    assert_eq!(error, "original cancellation identity");
                }
                if fault == "cleanup" {
                    assert!(
                        error.contains("cache fill failure")
                            && error.contains("cleanup compilation stage")
                    );
                }
                if fault == "receipt-rename" {
                    assert!(error.contains("module published=true; receipt published=false"));
                    assert_eq!(hostfs::read(entry.join("module.wasm")).unwrap(), module());
                } else {
                    assert_eq!(
                        hostfs::read(entry.join("module.wasm")).unwrap(),
                        b"prior module"
                    );
                    assert_eq!(
                        hostfs::read(entry.join("receipt.json")).unwrap(),
                        b"prior receipt"
                    );
                }
            }
            if fault != "cleanup" {
                assert!(
                    hostfs::read_dir(&entry).unwrap().all(|item| !item
                        .unwrap()
                        .file_name()
                        .to_string_lossy()
                        .starts_with('.'))
                );
            }
        }
    }

    #[test]
    fn canceled_new_publication_has_no_receipt_and_auxiliary_stages_are_validated() {
        for corruption in [false, true] {
            let root = Scratch::new();
            let inputs = json!({"source":"fixture"});
            let build_key = key(&inputs).unwrap();
            let mut calls = 0;
            let mut checkpoint = || {
                calls += 1;
                if calls == 2 {
                    Err("cancelled".into())
                } else {
                    Ok(())
                }
            };
            let result = publish_with(
                &root.0,
                &build_key,
                &inputs,
                &module(),
                0o600,
                auxiliary(),
                &mut checkpoint,
                |event, directory, name| {
                    if corruption && event == "aux-stage" {
                        replace(directory, name, b"changed raw observation");
                    }
                    Ok(())
                },
            );
            let error = result.unwrap_err();
            assert!(if corruption {
                error.contains("auxiliary identity changed")
            } else {
                error == "cancelled"
            });
            assert!(
                !root
                    .0
                    .join("compilation-v1")
                    .join(&build_key)
                    .join("receipt.json")
                    .exists()
            );
        }
    }
}
