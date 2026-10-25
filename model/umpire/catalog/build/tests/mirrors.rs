//! Semantic claims are checked independently of descriptor-owned structural fields.
#![forbid(unsafe_code)]

use prost::Message;
use umpire_catalog_build::mirrors;
use umpire_catalog_syntax::{catalogs, descriptor, proto::Graph};
use umpire_runtime::prost_types::{
    DescriptorProto, FieldDescriptorProto, FileDescriptorProto, FileDescriptorSet,
    OneofDescriptorProto, field_descriptor_proto::Type,
};

fn graph(extra_variant: bool) -> Result<Graph, String> {
    let mut fields = vec![
        FieldDescriptorProto {
            name: Some("name".into()),
            number: Some(1),
            r#type: Some(Type::String as i32),
            ..Default::default()
        },
        FieldDescriptorProto {
            name: Some("opaque".into()),
            number: Some(2),
            r#type: Some(Type::Bytes as i32),
            ..Default::default()
        },
        FieldDescriptorProto {
            name: Some("first".into()),
            number: Some(3),
            r#type: Some(Type::String as i32),
            oneof_index: Some(0),
            ..Default::default()
        },
    ];
    if extra_variant {
        fields.push(FieldDescriptorProto {
            name: Some("added".into()),
            number: Some(4),
            r#type: Some(Type::String as i32),
            oneof_index: Some(0),
            ..Default::default()
        });
    }
    Graph::new(
        &FileDescriptorSet {
            file: vec![FileDescriptorProto {
                package: Some("example.v1".into()),
                message_type: vec![DescriptorProto {
                    name: Some("Input".into()),
                    field: fields,
                    oneof_decl: vec![OneofDescriptorProto {
                        name: Some("choice".into()),
                        ..Default::default()
                    }],
                    ..Default::default()
                }],
                ..Default::default()
            }],
        }
        .encode_to_vec(),
        "::device_api",
    )
}

#[test]
fn descriptors_own_structure_and_every_unclaimed_field_has_a_reason()
-> Result<(), Box<dyn std::error::Error>> {
    let graph = graph(true)?;
    let source = r#"
        #[derive(Catalog)]
        #[mirror(request = "example.v1.Input")]
        pub struct Input { #[rule(max_bytes = 8, rejects = "name")] name: Text }
    "#;
    let expanded = mirrors::expand(source, &graph)?;
    let parsed = catalogs(&expanded)?;
    let coverage = parsed[0].coverage();
    descriptor::admit(&graph, &coverage)?;
    assert_eq!(coverage.messages[0].fields, ["name", "opaque", "first", "added"]);
    assert_eq!(coverage.messages[0].opaque_fields, ["opaque", "first", "added"]);
    assert!(expanded.contains("Unclaimed descriptor field example.v1.Input.opaque"));
    assert!(expanded.contains("Unclaimed descriptor field example.v1.Input.choice"));
    assert!(expanded.contains("max_bytes = 8"));
    assert_eq!(mirrors::expand(source, &graph)?, expanded);
    Ok(())
}

#[test]
fn claimed_variants_and_invalid_claims_fail_admission() -> Result<(), Box<dyn std::error::Error>> {
    let source = r#"
        #[derive(Catalog)]
        #[mirror(response = "example.v1.Input")]
        pub struct Input {
            name: Text,
            #[mirror(oneof(First = "first"))]
            choice: OneOf<wire::Choice>,
        }
    "#;
    let before = graph(false)?;
    let expanded = mirrors::expand(source, &before)?;
    descriptor::admit(&before, &catalogs(&expanded)?[0].coverage())?;
    assert!(mirrors::expand(source, &graph(true)?).is_err());
    for invalid in [
        source.replace("example.v1.Input", "example.v1.WrongIdentity"),
        source.replace("name: Text", "stale: Text"),
        source.replace("name: Text", "name: Text, name: Text"),
        source.replace("name: Text", "#[rule(max_bytes = 8)] name: Ignored"),
    ] {
        let result = mirrors::expand(&invalid, &before)
            .map_err(|error| error.to_string())
            .and_then(|source| catalogs(&source).map_err(|error| error.to_string()))
            .and_then(|catalogs| descriptor::admit(&before, &catalogs[0].coverage()));
        assert!(result.is_err(), "invalid claim admitted: {invalid}");
    }
    Ok(())
}
