use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, fs, path::Path};

pub fn sources(root: &Path, directory: &Path, files: &mut BTreeMap<String, Vec<u8>>) {
    for entry in fs::read_dir(directory).expect("read helper source directory") {
        let path = entry.expect("read helper source entry").path();
        if path.is_dir() {
            sources(root, &path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.insert(
                path.strip_prefix(root)
                    .expect("contained helper source")
                    .to_string_lossy()
                    .replace('\\', "/"),
                fs::read(&path).expect("read helper source"),
            );
        }
    }
}

pub const REQUIRED_OWNERS: &[&str] = &[
    "src/lib.rs",
    "src/linked_cluster.rs",
    "src/linked.rs",
    "src/linked_storage.rs",
    "src/storage_transport.rs",
    "src/volume/mod.rs",
    "src/namespace/image.rs",
    "src/namespace/call.rs",
    "src/metrics.rs",
    "src/volume/image.rs",
    "src/cluster/network/evidence.rs",
    "src/linked_inspection.rs",
    "src/linked_storage_inspection.rs",
    "src/model/inspection.rs",
    "src/protocol.rs",
    "src/model.rs",
];

pub fn implementation(
    files: &BTreeMap<String, Vec<u8>>,
    bindings: &BTreeMap<String, String>,
    compiler: &[u8],
) -> Result<String, String> {
    for name in REQUIRED_OWNERS {
        if !files.contains_key(*name) {
            return Err(format!("missing implementation owner {name}"));
        }
    }
    let mut hash = Sha256::new();
    hash.update(b"gomad.linked-implementation/v1\0");
    for (name, value) in bindings {
        hash.update(name.as_bytes());
        hash.update([0]);
        hash.update(value.as_bytes());
        hash.update([0]);
    }
    hash.update(compiler);
    for (name, bytes) in files {
        hash.update((name.len() as u64).to_be_bytes());
        hash.update(name.as_bytes());
        hash.update((bytes.len() as u64).to_be_bytes());
        hash.update(bytes);
    }
    Ok(format!("{:x}", hash.finalize()))
}
