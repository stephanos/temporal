use prost::Message;
use umpire_catalog_syntax::descriptor::{self, Coverage};
use umpire_runtime::prost_types::{
    EnumValueDescriptorProto, FieldDescriptorProto, FileDescriptorSet,
};

pub(crate) fn assert_additions_refused(
    bytes: &[u8],
    coverage: &Coverage,
    message: &str,
    enumeration: &str,
    service: &str,
) -> Result<(), Box<dyn std::error::Error>> {
    let original = FileDescriptorSet::decode(bytes)?;
    descriptor::check(bytes, coverage)?;
    for (kind, owner, added) in [
        ("field", message, "unreviewed_field"),
        ("enum", enumeration, "UNREVIEWED_VARIANT"),
        ("method", service, "UnreviewedMethod"),
    ] {
        let mut descriptor = original.clone();
        let mut mutated = false;
        for file in &mut descriptor.file {
            let package = file.package.as_deref().unwrap_or_default();
            let name = |name: Option<&str>| format!("{package}.{}", name.unwrap_or_default());
            match kind {
                "field" => {
                    if let Some(message) =
                        file.message_type.iter_mut().find(|m| name(m.name.as_deref()) == owner)
                    {
                        message.field.push(FieldDescriptorProto {
                            name: Some(added.into()),
                            number: Some(999_999),
                            r#type: Some(9),
                            ..Default::default()
                        });
                        mutated = true;
                    }
                }
                "enum" => {
                    if let Some(enumeration) =
                        file.enum_type.iter_mut().find(|e| name(e.name.as_deref()) == owner)
                    {
                        enumeration.value.push(EnumValueDescriptorProto {
                            name: Some(added.into()),
                            number: Some(999_999),
                            ..Default::default()
                        });
                        mutated = true;
                    }
                }
                _ => {
                    if let Some(service) =
                        file.service.iter_mut().find(|s| name(s.name.as_deref()) == owner)
                    {
                        let mut method =
                            service.method.first().cloned().ok_or("service has no method")?;
                        method.name = Some(added.into());
                        service.method.push(method);
                        mutated = true;
                    }
                }
            }
            if mutated {
                break;
            }
        }
        assert!(mutated, "missing {kind} descriptor: {owner}");
        let error = descriptor::check(&descriptor.encode_to_vec(), coverage)
            .err()
            .ok_or("descriptor addition passed coverage")?;
        assert!(error.contains(added), "{kind} in {owner}: {error}");
    }
    Ok(())
}
