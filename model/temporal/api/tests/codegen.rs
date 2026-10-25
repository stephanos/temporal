//! Real API generation must not depend on descriptor-set file order.

use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
    time::{SystemTime, UNIX_EPOCH},
};

use prost::Message;
use prost_types::FileDescriptorSet;

#[path = "../codegen.rs"]
mod codegen;

#[test]
fn descriptor_file_permutations_generate_identical_rust() -> Result<(), Box<dyn std::error::Error>>
{
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
    let upstream = manifest.join("../../../proto/api.binpb");
    let mut files = FileDescriptorSet::decode(fs::read(upstream)?.as_slice())?;
    files.file.sort_by(|a, b| a.name.cmp(&b.name));
    let scratch = Path::new(env!("OUT_DIR")).join(format!(
        "codegen-order-{}",
        SystemTime::now().duration_since(UNIX_EPOCH)?.as_nanos()
    ));
    let output = scratch.join("generated");
    fs::create_dir_all(&output)?;
    let input = scratch.join("api.binpb");
    let mut results = Vec::new();
    for _ in 0..2 {
        fs::write(&input, files.encode_to_vec())?;
        let mut config = prost_build::Config::new();
        config.out_dir(&output);
        codegen::generate(
            &mut config,
            &input,
            &manifest.join("../proto/umpire/temporal/v1/temporal.proto"),
            &manifest.join("../proto"),
        )?;
        let mut generated = BTreeMap::<PathBuf, Vec<u8>>::new();
        for entry in fs::read_dir(&output)? {
            let entry = entry?;
            generated.insert(entry.file_name().into(), fs::read(entry.path())?);
            fs::remove_file(entry.path())?;
        }
        assert!(generated.contains_key(Path::new("temporal.api.sdk.v1.rs")));
        assert!(generated.contains_key(Path::new("umpire.temporal.v1.rs")));
        results.push(generated);
        files.file.reverse();
    }
    assert_eq!(results[0].keys().collect::<Vec<_>>(), results[1].keys().collect::<Vec<_>>());
    for (name, bytes) in &results[0] {
        assert!(bytes == &results[1][name], "descriptor file order changed {}", name.display());
    }
    fs::remove_dir_all(scratch)?;
    Ok(())
}

#[test]
#[allow(deprecated)]
fn empty_oneof_uses_unit_and_preserves_presence() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::workflow::v1::{
        PendingActivityInfo, pending_activity_info::AssignedBuildId,
    };
    let present = PendingActivityInfo {
        assigned_build_id: Some(AssignedBuildId::UseWorkflowBuildId(())),
        ..Default::default()
    };
    assert_eq!(present.encode_to_vec(), [0x6a, 0]);
    assert_eq!(PendingActivityInfo::decode([0x6a, 0].as_slice())?, present);
    assert_ne!(present, PendingActivityInfo::decode([].as_slice())?);
    Ok(())
}

#[test]
fn double_wrapper_uses_scalar_and_preserves_presence() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::taskqueue::v1::TaskQueueMetadata;
    let absent = TaskQueueMetadata::default();
    let zero = TaskQueueMetadata { max_tasks_per_second: Some(0.0) };
    assert_eq!(zero.encode_to_vec(), [0x0a, 0]);
    assert_ne!(absent.encode_to_vec(), zero.encode_to_vec());
    for value in [zero, TaskQueueMetadata { max_tasks_per_second: Some(12.5) }] {
        assert_eq!(TaskQueueMetadata::decode(value.encode_to_vec().as_slice())?, value);
    }
    Ok(())
}
