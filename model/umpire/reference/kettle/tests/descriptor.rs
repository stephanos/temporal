//! Additions to the real Kettle descriptor must refuse naming their missing declaration.
use prost::Message;
use prost_types::{
    EnumValueDescriptorProto, FieldDescriptorProto, FileDescriptorSet, MethodDescriptorProto,
};
use umpire_catalog_syntax::{
    catalogs,
    descriptor::{self, Coverage},
    services::classifications,
};

#[test]
fn descriptor_additions_name_the_unreviewed_field_enum_and_method()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../api.rs");
    let mut coverage = Coverage::default();
    for catalog in catalogs(source)? {
        let c = catalog.coverage();
        coverage.messages.extend(c.messages);
        coverage.enums.extend(c.enums);
    }
    for classification in
        classifications(source, "umpire.reference.kettle.v1.KettleService", "KettleServiceMethod")?
    {
        coverage.services.push(classification.coverage);
    }
    let original = FileDescriptorSet::decode(
        include_bytes!(concat!(env!("OUT_DIR"), "/kettle.binpb")).as_slice(),
    )?;
    descriptor::check(&original.encode_to_vec(), &coverage)?;
    for (kind, needle) in [
        ("field", "unreviewed_field"),
        ("enum", "UNREVIEWED_VARIANT"),
        ("method", "UnreviewedMethod"),
    ] {
        let mut changed = original.clone();
        let file = changed
            .file
            .iter_mut()
            .find(|f| f.name.as_deref() == Some("kettle/v1/kettle.proto"))
            .ok_or("missing descriptor")?;
        match kind {
            "field" => file
                .message_type
                .iter_mut()
                .find(|m| m.name.as_deref() == Some("FillRequest"))
                .ok_or("missing FillRequest")?
                .field
                .push(FieldDescriptorProto {
                    name: Some(needle.into()),
                    number: Some(99),
                    r#type: Some(9),
                    ..Default::default()
                }),
            "enum" => file.enum_type[0].value.push(EnumValueDescriptorProto {
                name: Some(needle.into()),
                number: Some(99),
                ..Default::default()
            }),
            "method" => file.service[0].method.push(MethodDescriptorProto {
                name: Some(needle.into()),
                input_type: Some(".umpire.reference.kettle.v1.FillRequest".into()),
                output_type: Some(".umpire.reference.kettle.v1.FillResponse".into()),
                ..Default::default()
            }),
            _ => unreachable!(),
        }
        let error = descriptor::check(&changed.encode_to_vec(), &coverage)
            .err()
            .ok_or("descriptor addition passed coverage")?;
        assert!(error.contains(needle), "{kind}: {error}");
    }
    Ok(())
}
