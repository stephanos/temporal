//! A second service exercises the shared generation contract.
use prost::Message;
use umpire_catalog_build::{services, wire_schema};
use umpire_catalog_syntax::descriptor::Coverage;
use umpire_wasm::google::protobuf::{
    DescriptorProto, FileDescriptorProto, FileDescriptorSet, MethodDescriptorProto,
    ServiceDescriptorProto,
};

#[test]
fn second_service_uses_configured_identity_and_roots() -> Result<(), String> {
    let bytes = fixture().encode_to_vec();
    let config = service_config();
    let source = services::generate(&bytes, &Coverage::default(), &[], &config)?;
    assert!(source.contains("pub enum DeviceMethod"));
    assert!(source.contains("::device_api::example::v1::Input"));
    let config = wire_schema::Config {
        service: config.service,
        rust_type_root: "crate::generated",
        method_enum: config.method_enum,
        fields_function: "device_fields",
        mutate_function: "mutate_device",
        request_roots: &["Set"],
    };
    let source = wire_schema::generate(&bytes, &config)?;
    assert!(source.contains("pub fn device_fields(method: DeviceMethod)"));
    assert!(source.contains("crate::generated::example::v1::Input"));
    Ok(())
}

fn fixture() -> FileDescriptorSet {
    FileDescriptorSet {
        file: vec![FileDescriptorProto {
            package: Some("example.v1".into()),
            service: vec![ServiceDescriptorProto {
                name: Some("Device".into()),
                method: vec![MethodDescriptorProto {
                    name: Some("Set".into()),
                    input_type: Some(".example.v1.Input".into()),
                    output_type: Some(".example.v1.Output".into()),
                    ..Default::default()
                }],
                ..Default::default()
            }],
            message_type: vec![
                DescriptorProto { name: Some("Input".into()), ..Default::default() },
                DescriptorProto { name: Some("Output".into()), ..Default::default() },
            ],
            ..Default::default()
        }],
    }
}

fn service_config() -> services::Config<'static> {
    services::Config {
        service: "example.v1.Device",
        rust_type_root: "::device_api",
        macro_type_root: "$crate",
        method_enum: "DeviceMethod",
        method_info: "DeviceMethodInfo",
        methods_const: "DEVICE_METHODS",
        dispatch_macro: "dispatch_device",
        bind_macro: "bind_device",
    }
}

#[test]
fn omitted_field_enum_and_method_are_rejected() -> Result<(), String> {
    use umpire_catalog_syntax::descriptor::{
        self, Disposition, EnumCoverage, MessageCoverage, MethodCoverage, ServiceCoverage,
    };
    use umpire_wasm::google::protobuf::{
        EnumDescriptorProto, EnumValueDescriptorProto, FieldDescriptorProto,
        field_descriptor_proto::{Label, Type},
    };
    let mut descriptors = fixture();
    descriptors.file[0].message_type[0].field.push(FieldDescriptorProto {
        name: Some("state".into()),
        number: Some(1),
        label: Some(Label::Optional as i32),
        r#type: Some(Type::Enum as i32),
        type_name: Some(".example.v1.State".into()),
        ..Default::default()
    });
    descriptors.file[0].enum_type.push(EnumDescriptorProto {
        name: Some("State".into()),
        value: vec![EnumValueDescriptorProto {
            name: Some("OFF".into()),
            number: Some(0),
            ..Default::default()
        }],
        ..Default::default()
    });
    let coverage = Coverage {
        messages: vec![MessageCoverage {
            name: "example.v1.Input".into(),
            fields: vec!["state".into()],
            opaque_fields: vec![],
        }],
        enums: vec![EnumCoverage { name: "example.v1.State".into(), values: vec!["OFF".into()] }],
        services: vec![ServiceCoverage {
            name: "example.v1.Device".into(),
            methods: vec![MethodCoverage {
                name: "Set".into(),
                disposition: Disposition::Observed,
            }],
        }],
    };
    let bytes = descriptors.encode_to_vec();
    descriptor::check(&bytes, &coverage)?;
    for (kind, expected) in [("field", "state"), ("enum", "OFF"), ("method", "Set")] {
        let mut omitted = coverage.clone();
        match kind {
            "field" => omitted.messages[0].fields.clear(),
            "enum" => omitted.enums[0].values.clear(),
            _ => omitted.services[0].methods.clear(),
        }
        let error = descriptor::check(&bytes, &omitted)
            .err()
            .ok_or("omitted descriptor obligation passed")?;
        assert!(error.contains(expected), "{kind}: {error}");
    }
    Ok(())
}

#[test]
fn added_method_and_action_break_exhaustive_generated_matches()
-> Result<(), Box<dyn std::error::Error>> {
    use std::{fs, process::Command};
    use umpire_catalog_syntax::services::classifications;
    let config = service_config();
    let source = "fn classify(method: DeviceMethod) -> RpcRole { match method { DeviceMethod::Set => RpcRole::Action(ModelAction::Device(Device::Set)) }}";
    let classifications = classifications(source, config.service, config.method_enum)?;
    let mut descriptors = fixture();
    let directory =
        std::env::temp_dir().join(format!("catalog-build-controls-{}", std::process::id()));
    fs::create_dir_all(&directory)?;
    for (added, missing) in
        [(false, None), (false, Some("AddedAction")), (true, Some("AddedMethod"))]
    {
        if added {
            let mut method = descriptors.file[0].service[0].method[0].clone();
            method.name = Some("AddedMethod".into());
            descriptors.file[0].service[0].method.push(method);
        }
        let generated = services::generate(
            &descriptors.encode_to_vec(),
            &Coverage::default(),
            &classifications,
            &config,
        )?;
        let extra = if missing == Some("AddedAction") { ", AddedAction" } else { "" };
        let control = format!(
            "{generated}\nenum Device {{ Set{extra} }} enum ModelAction {{ Device(Device), Clock }} enum Binding {{ Rpc(&'static str), Environment }} fn bind(action: ModelAction) -> Binding {{ bind_device!(action, ModelAction::Clock => Binding::Environment) }} fn classify(method: DeviceMethod) {{ match method {{ DeviceMethod::Set => () }} }} fn main() {{}}"
        );
        let input = directory.join("control.rs");
        fs::write(&input, control)?;
        let result = Command::new("rustc")
            .args([
                "--edition=2024",
                "--crate-name=catalog_control",
                "--emit=metadata",
                "-Awarnings",
            ])
            .arg(&input)
            .arg("--out-dir")
            .arg(&directory)
            .output()?;
        let error = String::from_utf8(result.stderr)?;
        if let Some(missing) = missing {
            assert!(
                !result.status.success() && error.contains("E0004") && error.contains(missing),
                "{error}"
            );
        } else {
            assert!(result.status.success(), "{error}");
        }
    }
    fs::remove_dir_all(directory)?;
    Ok(())
}

#[test]
fn coverage_build_reads_the_explicit_catalog_root_and_descriptor_path()
-> Result<(), Box<dyn std::error::Error>> {
    use std::fs;
    use umpire_catalog_build::coverage;
    let root = std::env::temp_dir().join(format!("catalog-build-inputs-{}", std::process::id()));
    let nested = root.join("nested");
    fs::create_dir_all(&nested)?;
    fs::write(root.join("ignored.rs"), "invalid Rust ignored by catalog filename policy")?;
    fs::write(
        nested.join("api.rs"),
        "#[derive(Catalog)] #[mirror(request = \"example.v1.Input\")] struct Input {} fn classify(method: DeviceMethod) -> RpcRole { match method { DeviceMethod::Set => RpcRole::Action(ModelAction::Device(Device::Set)) }}",
    )?;
    let descriptor = root.join("selected.binpb");
    fs::write(&descriptor, fixture().encode_to_vec())?;
    let inputs = coverage::inputs(&root)?;
    assert_eq!(inputs.files, [nested.join("api.rs")]);
    assert_eq!(inputs.directories, [root.clone(), nested]);
    let (coverage, classifications) =
        coverage::check(&root, &descriptor, "example.v1.Device", "DeviceMethod")?;
    assert_eq!(coverage.messages[0].name, "example.v1.Input");
    assert_eq!(classifications[0].coverage.name, "example.v1.Device");
    fs::remove_dir_all(root)?;
    Ok(())
}
