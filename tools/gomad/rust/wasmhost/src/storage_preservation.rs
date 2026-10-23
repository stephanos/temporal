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

#[test]
fn refused_large_history_retains_original_hundred_thousand_write_prefix() {
    use crate::namespace::{Image, Limits, Namespace, Open, Operation as N, Output, RIGHTS_ALL};
    use crate::storage_transport::{StorageDescriptor, evidence};
    let model: StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":65536}],
        "limits":{"PendingOperations":200000,"Transitions":200000}
    }))
    .unwrap();
    let descriptor: StorageDescriptor = serde_json::from_value(json!({
        "model":model,"model_identity":model.identity(),"return_image":true
    }))
    .unwrap();
    let mut namespace = Namespace::new(Image {
        entries: Vec::new(),
        writable_directories: vec![b"/tmp".to_vec(), b"/workspace".to_vec()],
        preopens: vec![b"/tmp".to_vec(), b"/workspace".to_vec()],
        working_directory: b"/workspace".to_vec(),
        stdin: Vec::new(),
        epoch_nanos: 946_684_800_000_000_000,
        limits: Limits {
            files: 1000,
            descriptors: 64,
            filesystem_bytes: 64 << 10,
            output_bytes: 1 << 20,
        },
    })
    .unwrap();
    descriptor.attach(&mut namespace, None, 128 << 20).unwrap();
    let reply = namespace
        .execute(
            N::Open(Open {
                fd: 4,
                path: b"file".to_vec(),
                dirflags: 0,
                oflags: 1,
                base: RIGHTS_ALL,
                inheriting: 0,
                flags: 0,
            }),
            1,
        )
        .unwrap();
    assert_eq!(reply.errno, 0);
    let Some(Output::Fd(fd)) = reply.output else {
        panic!("missing storage file descriptor")
    };
    for _ in 0..100_000 {
        let reply = namespace
            .execute(
                N::Write {
                    fd,
                    data: b"x".to_vec(),
                    offset: Some(0),
                },
                1,
            )
            .unwrap();
        assert_eq!((reply.errno, reply.output), (0, Some(Output::Written(1))));
    }
    let storage = namespace.storage().unwrap();
    let operations = storage.operations();
    assert_eq!(
        operations["data"]
            .iter()
            .filter(|operation| operation.kind == "write")
            .count(),
        100_000
    );
    let encoded = serde_json::to_vec(&operations).unwrap();
    assert!(encoded.len() > 256 + 18 * ((64 << 10) + (1 << 20)));
    let snapshots = serde_json::to_vec(&storage.snapshots()).unwrap();
    let log = storage.log.clone();
    let collected = evidence(&mut namespace, &descriptor);
    assert_eq!(
        collected.failure,
        Some(("capacity", "WASI capacity: storage-image-bytes".into()))
    );
    assert!(collected.image.is_none());
    assert_eq!(collected.log, log);
    assert_eq!(&collected.state[..8], b"GMSTORE\x01");
    assert_eq!(
        u32::from_le_bytes(collected.state[8..12].try_into().unwrap()),
        2
    );
    assert_eq!(
        u32::from_le_bytes(collected.state[12..16].try_into().unwrap()),
        6
    );
    let first_length = usize::try_from(u64::from_le_bytes(
        collected.state[20..28].try_into().unwrap(),
    ))
    .unwrap();
    assert_eq!(&collected.state[28..28 + first_length], snapshots);
    let second = 28 + first_length;
    assert_eq!(
        u32::from_le_bytes(collected.state[second..second + 4].try_into().unwrap()),
        2
    );
    let second_length = usize::try_from(u64::from_le_bytes(
        collected.state[second + 4..second + 12].try_into().unwrap(),
    ))
    .unwrap();
    assert_eq!(
        &collected.state[second + 12..second + 12 + second_length],
        encoded
    );
}

#[test]
fn maximum_enumeration_limits_preserve_actual_native_empty_volume_evidence() {
    use crate::namespace::{Image, Limits, Namespace};
    use crate::storage_transport::{StorageDescriptor, evidence};
    let model: StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":256 << 10}],
        "limits":{"PendingOperations":1000,"Transitions":1000}
    }))
    .unwrap();
    let mut descriptor: StorageDescriptor = serde_json::from_value(json!({
        "model":model,"model_identity":model.identity(),"return_image":true,
        "crash_limits":{"States":512,"Operations":32,"Depth":32,"Bytes":64 << 20,"Work":1 << 20}
    }))
    .unwrap();
    let image = Image {
        entries: Vec::new(),
        writable_directories: vec![b"/workspace".to_vec()],
        preopens: vec![b"/workspace".to_vec()],
        working_directory: b"/workspace".to_vec(),
        stdin: Vec::new(),
        epoch_nanos: 946_684_800_000_000_000,
        limits: Limits {
            files: 1000,
            descriptors: 64,
            filesystem_bytes: 256 << 10,
            output_bytes: 1 << 20,
        },
    };
    let mut ordinary = Namespace::new(image.clone()).unwrap();
    descriptor.attach(&mut ordinary, None, 1 << 20).unwrap();
    let ordinary = evidence(&mut ordinary, &descriptor);
    assert!(ordinary.failure.is_none());
    descriptor.crash_limits = Some(crate::volume::CrashEnumerationLimits {
        states: u64::MAX,
        operations: u64::MAX,
        depth: u64::MAX,
        bytes: u64::MAX,
        work: u64::MAX,
    });
    let mut maximum = Namespace::new(image).unwrap();
    descriptor.attach(&mut maximum, None, 1 << 20).unwrap();
    let maximum = evidence(&mut maximum, &descriptor);
    assert!(maximum.failure.is_none());
    assert_eq!(maximum.log, ordinary.log);
    assert_eq!(maximum.state, ordinary.state);
    assert_eq!(maximum.image, ordinary.image);
    assert!(maximum.image.is_some());
}
