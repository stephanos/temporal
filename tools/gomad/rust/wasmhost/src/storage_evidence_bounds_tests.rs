use super::*;
use crate::linked::Sink;
use serde_json::json;

fn storage_descriptor() -> StorageDescriptor {
    let model: crate::volume::StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":16384}],
        "limits":{"PendingOperations":1000,"Transitions":1000}
    }))
    .unwrap();
    serde_json::from_value(
        json!({"model":model,"model_identity":model.identity(),"return_image":true}),
    )
    .unwrap()
}

fn header(version: u32, count: u32) -> Vec<u8> {
    let mut bytes = b"GMSTORE\x01".to_vec();
    bytes.extend(version.to_le_bytes());
    bytes.extend(count.to_le_bytes());
    bytes
}

#[test]
fn partial_header_recognizes_every_original_split_and_rejects_short_or_wrong_headers() {
    let bounds = Bounds::new(16 << 10, 1, 4, &storage_descriptor());
    assert_eq!(
        (bounds.complete, bounds.partial, bounds.image),
        (19_169_536, 21_141_076, 1_064_960)
    );
    let bytes = header(2, 6);
    for length in 0..16 {
        assert_eq!(
            bounds.limit("storage_state", &bytes[..length], &[]),
            Some(19_169_536)
        );
    }
    for split in 0..=16 {
        assert_eq!(
            bounds.limit("storage_state", &bytes[..split], &bytes[split..]),
            Some(21_141_076)
        );
    }
    for mut bytes in [header(1, 5), header(3, 6), header(2, 5), header(2, 6)] {
        if bytes[8..12] == 2_u32.to_le_bytes() && bytes[12..16] == 6_u32.to_le_bytes() {
            bytes[0] ^= 1;
        }
        assert_eq!(bounds.limit("storage_state", &[], &bytes), Some(19_169_536));
    }
}

#[test]
fn storage_budget_counts_every_volume_and_saturates_original_overflow_cases() {
    let mut descriptor = storage_descriptor();
    descriptor.model.volumes.as_mut().unwrap()[0].capacity_bytes = 1 << 20;
    descriptor.model_identity = descriptor.model.identity();
    let base = Bounds::new(8 << 20, 10000, 1000, &descriptor).complete;
    descriptor.crash_limits = Some(crate::volume::CrashEnumerationLimits {
        states: 1,
        operations: 1,
        depth: 1,
        bytes: 1,
        work: 1,
    });
    let one = Bounds::new(8 << 20, 10000, 1000, &descriptor).complete;
    assert!(one > base);
    let context = 18 * ((8 << 20) + (1 << 20)) + 12;
    descriptor.model.volumes.as_mut().unwrap().push(
        serde_json::from_value(json!({"ID":"more","Path":"/other","CapacityBytes":1})).unwrap(),
    );
    assert_eq!(
        Bounds::new(8 << 20, 10000, 1000, &descriptor).complete,
        base + context + 2 * (one - base - context)
    );
    let limits = descriptor.crash_limits.as_mut().unwrap();
    limits.bytes = u64::MAX;
    limits.states = u64::MAX;
    limits.work = u64::MAX;
    assert_eq!(
        Bounds::new(8 << 20, 10000, 1000, &descriptor).complete,
        u64::MAX
    );
    assert_eq!(
        Bounds::new(16 << 10, u64::MAX, 4, &storage_descriptor()).partial,
        u64::MAX
    );
    assert_eq!(
        Bounds::new(16 << 10, 1, u64::MAX, &storage_descriptor()).partial,
        u64::MAX
    );
    assert_eq!(budget_add(7, 0, u64::MAX), 7);
    assert_eq!(budget_add(u64::MAX - 1, 1, 1), u64::MAX);
    assert_eq!(budget_add(u64::MAX - 1, 1, 2), u64::MAX);
}

#[test]
fn native_sink_keeps_independent_storage_caps_and_rejected_append_has_no_effect() {
    for stream in ["storage_state", "storage_image"] {
        let bounds = Bounds::new(16 << 10, 1, 4, &storage_descriptor());
        let limit = bounds.limit(stream, &[], &[]).unwrap();
        let sink = Sink::new(64 << 20).with_storage_bounds(Some(bounds));
        let prefix = vec![0; usize::try_from(limit - 1).unwrap()];
        sink.append(stream, &prefix).unwrap();
        sink.append(stream, &[1]).unwrap();
        assert_eq!(
            sink.append(stream, &[2]).unwrap_err().to_string(),
            "WASI capacity: helper evidence bytes"
        );
        let streams = sink.into_streams().unwrap();
        assert_eq!(streams[stream].len() as u64, limit);
        assert_eq!(&streams[stream][..prefix.len()], prefix);
        assert_eq!(streams[stream].last(), Some(&1));
    }
}

#[test]
fn partial_state_retains_original_complete_and_partial_limits_with_fragmented_header() {
    for partial in [false, true] {
        let bounds = Bounds::new(16 << 10, 1, 4, &storage_descriptor());
        let complete = bounds.complete;
        let maximum = if partial {
            bounds.partial
        } else {
            bounds.complete
        };
        let sink = Sink::new(64 << 20).with_storage_bounds(Some(bounds));
        let header = if partial { header(2, 6) } else { header(1, 5) };
        sink.append("storage_state", &header[..10]).unwrap();
        sink.append("storage_state", &header[10..]).unwrap();
        sink.append(
            "storage_state",
            &vec![0; usize::try_from(complete - 16).unwrap()],
        )
        .unwrap();
        if partial {
            sink.append("storage_state", &[1]).unwrap();
            sink.append(
                "storage_state",
                &vec![0; usize::try_from(maximum - complete - 2).unwrap()],
            )
            .unwrap();
            sink.append("storage_state", &[1]).unwrap();
        }
        assert_eq!(
            sink.append("storage_state", &[2]).unwrap_err().to_string(),
            "WASI capacity: helper evidence bytes"
        );
        let streams = sink.into_streams().unwrap();
        assert_eq!(streams["storage_state"].len() as u64, maximum);
        assert_eq!(&streams["storage_state"][..16], header);
    }
}

#[test]
fn saturated_storage_allowance_keeps_the_existing_aggregate_limit() {
    let mut descriptor = storage_descriptor();
    descriptor.crash_limits = Some(crate::volume::CrashEnumerationLimits {
        states: u64::MAX,
        operations: u64::MAX,
        depth: u64::MAX,
        bytes: u64::MAX,
        work: u64::MAX,
    });
    let bounds = Bounds::new(16 << 10, 1, 4, &descriptor);
    let sink = Sink::new(3).with_storage_bounds(Some(bounds));
    sink.append("storage_state", b"abc").unwrap();
    assert_eq!(
        sink.append("storage_state", b"d").unwrap_err().to_string(),
        "linked evidence capacity"
    );
    assert_eq!(sink.into_streams().unwrap()["storage_state"], b"abc");
}

#[test]
fn native_preparation_retains_guard_before_consuming_model_and_namespace_inputs() {
    use crate::linked::{self, Limits, ModelInput, NamespaceImage, NamespaceLimits, Request};
    use sha2::{Digest, Sha256};
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let request = Request {
        module_sha256:format!("{:x}",Sha256::digest(&module)),module,
        limits:Limits{memory_bytes:65536,fuel:10000,output_bytes:1 << 20,evidence_bytes:64 << 20,wall_timeout:std::time::Duration::from_secs(5)},
        model:Some(ModelInput{
            configuration:serde_json::from_value(json!({
                "seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0;32],
                "working_directory_sha256":format!("{:x}",Sha256::digest(b"/workspace")),
                "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
                "epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":1,"transcript_bytes":1 << 20,"pending_events":4096}
            })).unwrap(),arguments:Vec::new(),environment:vec!["PWD=/workspace".into()],replay_trace:None,
        }),
        namespace:Some(NamespaceImage{
            entries:Vec::new(),writable_directories:vec![b"/workspace".to_vec()],preopens:vec![b"/workspace".to_vec()],working_directory:b"/workspace".to_vec(),stdin:Vec::new(),epoch_nanos:0,
            limits:NamespaceLimits{files:4,descriptors:64,filesystem_bytes:16 << 10,output_bytes:1 << 20},
        }),
        storage:Some(serde_json::from_value(json!({"descriptor":storage_descriptor()})).unwrap()),
    };
    let prepared = linked::prepare(request).unwrap();
    assert!(prepared.request.namespace.is_none());
    assert!(prepared.request.model.is_none());
    let bounds = prepared.storage_bounds.unwrap();
    assert_eq!(
        (bounds.complete, bounds.partial, bounds.image),
        (19_169_536, 21_141_076, 1_064_960)
    );
}

#[test]
fn storage_guard_is_lazy_and_leaves_other_stream_bytes_unchanged() {
    let bounds = Bounds::new(u64::MAX, u64::MAX, u64::MAX, &storage_descriptor());
    let sink = Sink::new(u64::MAX).with_storage_bounds(Some(bounds));
    sink.append("stdout", b"literal\0bytes").unwrap();
    sink.append("storage_image", b"image").unwrap();
    let streams = sink.into_streams().unwrap();
    assert_eq!(streams["stdout"], b"literal\0bytes");
    assert_eq!(streams["storage_image"], b"image");
    assert_eq!(streams.values().map(Vec::len).sum::<usize>(), 18);
}
