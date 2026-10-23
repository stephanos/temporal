//! Descriptor coverage controls using the real upstream protobuf closure.

use std::{fs, path::Path};

use prost::Message;
use umpire_runtime::google::protobuf::{
    DescriptorProto, EnumValueDescriptorProto, FieldDescriptorProto, FileDescriptorSet,
    MethodDescriptorProto,
};
use umpire_temporal_annotate_syntax::descriptor::{
    Coverage, Disposition, EnumCoverage, MessageCoverage, MethodCoverage, ServiceCoverage, check,
};

#[test]
fn executable_generated_api_declaration_matches_real_descriptors()
-> Result<(), Box<dyn std::error::Error>> {
    let declarations = umpire_temporal_annotate_syntax::catalogs(include_str!(
        "../../tests/ui-pass/generated_api.rs"
    ))?;
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
    check(bytes, &declarations[0].coverage())?;
    Ok(())
}

fn fixture() -> Result<(FileDescriptorSet, Coverage), Box<dyn std::error::Error>> {
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../proto/api.binpb");
    let upstream = FileDescriptorSet::decode(fs::read(path)?.as_slice())?;
    let linked = FileDescriptorSet::decode(
        include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb")).as_slice(),
    )?;
    for original in &upstream.file {
        let retained = linked.file.iter().find(|file| file.name == original.name);
        assert_eq!(
            retained,
            Some(original),
            "linker must preserve upstream files, including any upstream service definitions"
        );
    }
    let mut coverage = Coverage::default();
    for file in &linked.file {
        let package = file.package.clone().unwrap_or_default();
        if coverage.messages.is_empty()
            && let Some(message) =
                file.message_type.iter().find(|message| !message.field.is_empty())
        {
            coverage.messages.push(message_coverage(
                format!("{package}.{}", message.name.as_deref().unwrap_or_default()),
                message,
            ));
        }
        if coverage.enums.is_empty()
            && let Some(enumeration) = file.enum_type.first()
        {
            coverage.enums.push(EnumCoverage {
                name: format!("{package}.{}", enumeration.name.as_deref().unwrap_or_default()),
                values: enumeration.value.iter().filter_map(|value| value.name.clone()).collect(),
            });
        }
        if coverage.services.is_empty()
            && let Some(service) = file.service.iter().find(|service| !service.method.is_empty())
        {
            coverage.services.push(ServiceCoverage {
                name: format!("{package}.{}", service.name.as_deref().unwrap_or_default()),
                methods: service
                    .method
                    .iter()
                    .map(|method| MethodCoverage {
                        name: method.name.clone().unwrap_or_default(),
                        disposition: Disposition::Observed,
                    })
                    .collect(),
            });
        }
    }
    assert_eq!(coverage.messages.len(), 1, "real descriptor has a message");
    assert_eq!(coverage.enums.len(), 1, "real descriptor has an enum");
    assert_eq!(coverage.services.len(), 1, "real descriptor has a service");
    check(&linked.encode_to_vec(), &coverage)?;
    Ok((linked, coverage))
}

fn message_coverage(name: String, descriptor: &DescriptorProto) -> MessageCoverage {
    MessageCoverage {
        name,
        fields: descriptor.field.iter().filter_map(|field| field.name.clone()).collect(),
        opaque_fields: descriptor
            .field
            .iter()
            .filter(|field| field.type_name.is_some())
            .filter_map(|field| field.name.clone())
            .collect(),
    }
}

#[test]
fn declaration_order_names_the_first_out_of_place_symbol() -> Result<(), Box<dyn std::error::Error>>
{
    let (descriptor, coverage) = fixture()?;
    for method in [false, true] {
        let mut reordered = coverage.clone();
        let (owner, first) = if method {
            let service = &mut reordered.services[0];
            let first = service.methods[0].name.clone();
            service.methods.swap(0, 1);
            (service.name.clone(), first)
        } else {
            let message = &mut reordered.messages[0];
            let first = message.fields[0].clone();
            message.fields.swap(0, 1);
            (message.name.clone(), first)
        };
        let error = check(&descriptor.encode_to_vec(), &reordered)
            .err()
            .ok_or("source declaration order must be admitted explicitly")?;
        assert!(error.contains("out-of-order"), "{error}");
        assert!(error.contains(&format!("{owner}.{first}")), "{error}");
    }
    Ok(())
}

#[test]
fn nested_message_requires_coverage_and_refuses_added_fields()
-> Result<(), Box<dyn std::error::Error>> {
    let (mut descriptor, _) = fixture()?;
    let messages: Vec<_> = descriptor
        .file
        .iter()
        .flat_map(|file| {
            file.message_type.iter().map(move |message| {
                (
                    format!(
                        "{}.{}",
                        file.package.as_deref().unwrap_or_default(),
                        message.name.as_deref().unwrap_or_default()
                    ),
                    message,
                )
            })
        })
        .collect();
    let (parent_name, parent, field_name, target_name, target) = messages
        .iter()
        .find_map(|(owner, parent)| {
            parent.field.iter().find_map(|field| {
                let target_name = field.type_name.as_deref()?.trim_start_matches('.');
                let (_, target) =
                    messages.iter().find(|(name, _)| name == target_name && name != owner)?;
                Some((owner.clone(), *parent, field.name.clone()?, target_name.to_owned(), *target))
            })
        })
        .ok_or("real descriptors must contain a message referencing another message")?;
    let mut parent = message_coverage(parent_name, parent);
    parent.opaque_fields.retain(|field| field != &field_name);
    let mut coverage = Coverage { messages: vec![parent], ..Default::default() };
    let error = check(&descriptor.encode_to_vec(), &coverage)
        .err()
        .ok_or("missing nested coverage must refuse")?;
    assert!(error.contains(&target_name), "{error}");
    coverage.messages.push(message_coverage(target_name.clone(), target));
    check(&descriptor.encode_to_vec(), &coverage)?;
    for file in &mut descriptor.file {
        let package = file.package.as_deref().unwrap_or_default();
        if let Some(target) = file.message_type.iter_mut().find(|message| {
            format!("{package}.{}", message.name.as_deref().unwrap_or_default()) == target_name
        }) {
            target.field.push(FieldDescriptorProto {
                name: Some("added_nested_field".into()),
                number: Some(999_999),
                ..Default::default()
            });
            break;
        }
    }
    let error = check(&descriptor.encode_to_vec(), &coverage)
        .err()
        .ok_or("added nested field must refuse")?;
    assert!(error.contains("added_nested_field"), "{error}");
    Ok(())
}

#[test]
fn real_descriptor_additions_are_independently_refused() -> Result<(), Box<dyn std::error::Error>> {
    let (original, coverage) = fixture()?;
    for kind in ["field", "enum", "rpc"] {
        let mut descriptor = original.clone();
        let added = "ADDED_UNANNOTATED_SYMBOL";
        let mut mutated = false;
        for file in &mut descriptor.file {
            let package = file.package.as_deref().unwrap_or_default();
            let message = file.message_type.iter_mut().find(|message| {
                format!("{package}.{}", message.name.as_deref().unwrap_or_default())
                    == coverage.messages[0].name
            });
            let enumeration = file.enum_type.iter_mut().find(|enumeration| {
                format!("{package}.{}", enumeration.name.as_deref().unwrap_or_default())
                    == coverage.enums[0].name
            });
            let service = file.service.iter_mut().find(|service| {
                format!("{package}.{}", service.name.as_deref().unwrap_or_default())
                    == coverage.services[0].name
            });
            match (kind, message, enumeration, service) {
                ("field", Some(message), _, _) => {
                    message.field.push(FieldDescriptorProto {
                        name: Some(added.into()),
                        number: Some(999_999),
                        ..Default::default()
                    });
                    mutated = true;
                }
                ("enum", _, Some(enumeration), _) => {
                    enumeration.value.push(EnumValueDescriptorProto {
                        name: Some(added.into()),
                        number: Some(999_999),
                        ..Default::default()
                    });
                    mutated = true;
                }
                ("rpc", _, _, Some(service)) => {
                    service.method.push(MethodDescriptorProto {
                        name: Some(added.into()),
                        ..Default::default()
                    });
                    mutated = true;
                }
                _ => {}
            }
            if mutated {
                break;
            }
        }
        assert!(mutated, "real {kind} descriptor was mutated");
        let error = check(&descriptor.encode_to_vec(), &coverage)
            .err()
            .ok_or("added symbol must refuse")?;
        assert!(error.contains(added), "{kind}: {error}");
    }
    Ok(())
}

#[test]
fn stale_duplicates_and_empty_dispositions_are_refused() -> Result<(), Box<dyn std::error::Error>> {
    let (descriptor, original) = fixture()?;
    let bytes = descriptor.encode_to_vec();
    for case in [
        "duplicate-message",
        "duplicate-field",
        "duplicate-opaque-field",
        "duplicate-enum",
        "duplicate-value",
        "duplicate-service",
        "duplicate-method",
        "stale-message",
        "stale-enum",
        "stale-service",
        "stale-field",
        "stale-opaque-field",
        "stale-value",
        "stale-method",
        "blank-action",
        "blank-reason",
    ] {
        let mut coverage = original.clone();
        match case {
            "duplicate-message" => coverage.messages.push(coverage.messages[0].clone()),
            "duplicate-field" => {
                let field = coverage.messages[0].fields[0].clone();
                coverage.messages[0].fields.push(field);
            }
            "duplicate-opaque-field" => {
                let field = coverage.messages[0].fields[0].clone();
                coverage.messages[0].opaque_fields = vec![field.clone(), field];
            }
            "duplicate-enum" => coverage.enums.push(coverage.enums[0].clone()),
            "duplicate-value" => {
                let value = coverage.enums[0].values[0].clone();
                coverage.enums[0].values.push(value);
            }
            "duplicate-service" => coverage.services.push(coverage.services[0].clone()),
            "duplicate-method" => {
                let method = coverage.services[0].methods[0].clone();
                coverage.services[0].methods.push(method);
            }
            "stale-message" => coverage.messages[0].name = "missing.Message".into(),
            "stale-enum" => coverage.enums[0].name = "missing.Enum".into(),
            "stale-service" => coverage.services[0].name = "missing.Service".into(),
            "stale-field" => coverage.messages[0].fields.push("missing_field".into()),
            "stale-opaque-field" => coverage.messages[0].opaque_fields.push("missing_field".into()),
            "stale-value" => coverage.enums[0].values.push("MISSING_VALUE".into()),
            "stale-method" => coverage.services[0].methods.push(MethodCoverage {
                name: "MissingMethod".into(),
                disposition: Disposition::Observed,
            }),
            "blank-action" => {
                coverage.services[0].methods[0].disposition = Disposition::Action("  ".into());
            }
            "blank-reason" => {
                coverage.services[0].methods[0].disposition = Disposition::OutOfScope("  ".into());
            }
            _ => unreachable!(),
        }
        let error = check(&bytes, &coverage).err().ok_or(case)?;
        assert!(
            error.contains("duplicate") || error.contains("stale") || error.contains("missing"),
            "{case}: {error}"
        );
    }
    assert!(check(b"invalid", &original).is_err());
    Ok(())
}
