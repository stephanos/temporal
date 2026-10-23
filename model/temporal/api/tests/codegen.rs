//! Real API generation must not depend on descriptor-set file order.

use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
    time::{SystemTime, UNIX_EPOCH},
};

use prost::Message;
use umpire_runtime::google::protobuf::FileDescriptorSet;

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
