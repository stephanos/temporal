use crate::volume::{Operation, Storage, StorageModel};
use serde_json::json;

#[test]
fn partial_metadata_scalar_and_escaped_path_bounds_preserve_original_extremes() {
    let hash = format!("sha256:{}", "f".repeat(64));
    let mut operation = Operation {
        id: u64::MAX,
        kind: "namespace".into(),
        dependencies: None,
        inode: u64::MAX,
        offset: u64::MAX,
        bytes: u64::MAX,
        payload_sha256: hash.clone(),
        effect_sha256: hash.clone(),
        path: String::new(),
        destination: String::new(),
    };
    assert!(serde_json::to_vec(&operation).unwrap().len() + 1 <= 361);
    let model: StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1048576}],
        "limits":{"PendingOperations":1000,"Transitions":1000}
    }))
    .unwrap();
    let storage = Storage::new(model, 1 << 20).unwrap();
    let mut snapshots = storage.snapshots();
    let mut entry = snapshots.remove(0).persisted.remove(0);
    entry.path.clear();
    entry.mode = u32::MAX;
    entry.kind = "directory".into();
    entry.mod_time = i64::MIN;
    entry.size = u64::MAX;
    entry.data_sha256 = hash;
    assert!(serde_json::to_vec(&entry).unwrap().len() + 1 <= 195);
    operation.path = "\x01".repeat(4 * 4096);
    operation.destination.clone_from(&operation.path);
    operation.dependencies = Some(vec![u64::MAX; 2 * 4 + 3]);
    assert!(
        serde_json::to_vec(&operation).unwrap().len() + 1
            <= 361 + 2 * 6 * 4 * 4096 + 21 * (2 * 4 + 3)
    );
    entry.path = operation.path;
    assert!(serde_json::to_vec(&entry).unwrap().len() + 1 <= 195 + 6 * 4 * 4096);
}
