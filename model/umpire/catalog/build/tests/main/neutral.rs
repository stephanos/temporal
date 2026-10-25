//! A second service exercises the shared generation contract.
#![forbid(unsafe_code)]

use prost::Message;
use umpire_catalog_build::{System, services, wire_schema};
use umpire_catalog_syntax::{descriptor::Coverage, proto::Graph, services::Classification};
use umpire_runtime::prost_types::{
    DescriptorProto, FileDescriptorProto, FileDescriptorSet, MethodDescriptorProto,
    ServiceDescriptorProto,
};

#[path = "../../../syntax/tests/support/mod.rs"]
mod support;

#[test]
fn second_service_uses_configured_identity_and_roots() -> Result<(), String> {
    let bytes = fixture().encode_to_vec();
    let config = System { rust_root: "::device_api::generated", ..service_config() };
    let source = generate_services(&bytes, &Coverage::default(), &[], &config)?;
    assert!(source.contains("pub enum DeviceMethod"));
    assert!(source.contains("pub fn parse_rpc("));
    assert!(source.contains("pub struct DeviceMethodInfo"));
    assert!(source.contains("pub const DEVICE_METHODS"));
    assert!(source.contains("macro_rules! annotate_device"));
    assert!(source.contains("macro_rules! device_request_findings"));
    assert!(source.contains("macro_rules! dispatch_device_request"));
    assert!(source.contains("macro_rules! bind_device_action"));
    assert!(source.contains("\"::device_api::generated::example::v1::Input\""));
    assert_eq!(config.local_root(), "crate::generated");
    assert_eq!(config.macro_root(), "$crate::generated");
    let source = wire_schema::generate(&Graph::new(&bytes, &config.local_root())?, &config)?;
    assert!(source.contains("pub fn device_request_fields(method: DeviceMethod)"));
    assert!(source.contains("pub fn mutate_device_request("));
    assert!(source.contains("crate::generated::example::v1::Input"));
    assert_eq!(
        config.rebase("::device_api::generated::example::v1::Input", "$crate"),
        Some("$crate::example::v1::Input".into())
    );
    assert_eq!(config.rebase("::device_api::generated", "$crate"), Some("$crate".into()));
    assert_eq!(config.rebase("::device_api::generatedx::Input", "$crate"), None);
    let local = System { rust_root: "crate", ..service_config() };
    assert_eq!((local.local_root(), local.macro_root()), (String::new(), "$crate".into()));
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

fn service_config() -> System<'static> {
    System {
        service: "example.v1.Device",
        prefix: "device",
        rust_root: "::device_api",
        request_roots: &["Set"],
        catalog_root: std::path::PathBuf::new(),
    }
}

fn generate_services(
    bytes: &[u8],
    coverage: &Coverage,
    classifications: &[Classification],
    system: &System<'_>,
) -> Result<String, String> {
    services::generate(&Graph::new(bytes, system.rust_root)?, coverage, classifications, system)
}

#[test]
fn omitted_field_enum_and_method_are_rejected() -> Result<(), String> {
    use umpire_catalog_syntax::descriptor::{
        self, Disposition, EnumCoverage, MessageCoverage, MethodCoverage, ServiceCoverage,
    };
    use umpire_runtime::prost_types::{
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
    support::assert_additions_refused(
        &bytes,
        &coverage,
        "example.v1.Input",
        "example.v1.State",
        "example.v1.Device",
    )
    .map_err(|error| error.to_string())?;
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
    let classifications = classifications(source, config.service, &config.method_enum())?;
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
        let generated = generate_services(
            &descriptors.encode_to_vec(),
            &Coverage::default(),
            &classifications,
            &config,
        )?;
        let extra = if missing == Some("AddedAction") { ", AddedAction" } else { "" };
        let control = format!(
            "{generated}\nenum Device {{ Set{extra} }} enum ModelAction {{ Device(Device), Clock }} enum Binding {{ Rpc(&'static str), Environment }} fn bind(action: ModelAction) -> Binding {{ bind_device_action!(action, ModelAction::Clock => Binding::Environment) }} fn classify(method: DeviceMethod) {{ match method {{ DeviceMethod::Set => () }} }} fn main() {{}}"
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
fn an_annotated_rpc_must_be_claimed() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_catalog_syntax::{descriptor::MessageCoverage, services::classifications};
    let config = service_config();
    let coverage = Coverage {
        messages: vec![MessageCoverage {
            name: "example.v1.Input".into(),
            fields: vec![],
            opaque_fields: vec![],
        }],
        ..Coverage::default()
    };
    let bytes = fixture().encode_to_vec();
    let classify = |role: &str| {
        classifications(
            &format!(
                "fn classify(method: DeviceMethod) -> RpcRole {{ match method {{ DeviceMethod::Set => {role} }}}}"
            ),
            config.service,
            &config.method_enum(),
        )
    };
    generate_services(&bytes, &coverage, &classify("RpcRole::Observed(\"set\")")?, &config)?;
    let out_of_scope = classify("RpcRole::OutOfScope(\"device settings\")")?;
    assert_eq!(
        generate_services(&bytes, &coverage, &out_of_scope, &config).err().as_deref(),
        Some(
            "annotated RPC is out of scope: example.v1.Device.Set has mirrored messages but no \
             Action or Observed classification"
        )
    );
    generate_services(&bytes, &Coverage::default(), &out_of_scope, &config)?;
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
    let (coverage, classifications) = coverage::check(
        &root,
        &Graph::new(&fs::read(&descriptor)?, "")?,
        "example.v1.Device",
        "DeviceMethod",
    )?;
    assert_eq!(coverage.messages[0].name, "example.v1.Input");
    assert_eq!(classifications[0].coverage.name, "example.v1.Device");
    let out = root.join("out");
    fs::create_dir(&out)?;
    System { catalog_root: root.clone(), ..service_config() }.generate(&descriptor, &out)?;
    assert!(
        fs::read_to_string(out.join("services.rs"))?
            .contains("Binding::Rpc(\"example.v1.Device.Set\")")
    );
    assert!(
        fs::read_to_string(out.join("wire_schema.rs"))?.contains("pub fn device_request_fields")
    );
    fs::remove_dir_all(root)?;
    Ok(())
}

#[test]
fn coverage_build_resolves_feature_claims_against_one_service_table()
-> Result<(), Box<dyn std::error::Error>> {
    use std::fs;
    use umpire_catalog_build::coverage;
    let root = std::env::temp_dir().join(format!("catalog-build-claims-{}", std::process::id()));
    let (first, second) = (root.join("first"), root.join("second"));
    fs::create_dir_all(&first)?;
    fs::create_dir_all(&second)?;
    fs::write(
        root.join("service.rs"),
        "fn family(method: DeviceMethod) -> &'static str { match method { DeviceMethod::Set => \"device settings\" }}",
    )?;
    let claim = "fn classify(method: DeviceMethod) -> RpcRole<ModelAction> { match method { DeviceMethod::Set => RpcRole::Action(ModelAction::Device(Device::Set)), _ => RpcRole::OutOfScope(service::family(method)) }}";
    fs::write(first.join("api.rs"), claim)?;
    let descriptor = root.join("selected.binpb");
    fs::write(&descriptor, fixture().encode_to_vec())?;
    let result = (|| -> Result<(), Box<dyn std::error::Error>> {
        let (_, classifications) = coverage::check(
            &root,
            &Graph::new(&fs::read(&descriptor)?, "")?,
            "example.v1.Device",
            "DeviceMethod",
        )?;
        assert_eq!(classifications[0].bindings[0].1, "Action");
        fs::write(second.join("api.rs"), claim)?;
        let error = coverage::check(
            &root,
            &Graph::new(&fs::read(&descriptor)?, "")?,
            "example.v1.Device",
            "DeviceMethod",
        )
        .err()
        .ok_or("a duplicate feature claim passed the build")?;
        assert_eq!(error.to_string(), "duplicate feature claim: example.v1.Device.Set");
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn payload_classification_binds_every_value_and_rejects_duplicate_variants()
-> Result<(), Box<dyn std::error::Error>> {
    use std::{fs, process::Command};
    use umpire_catalog_syntax::{descriptor::Disposition, services::classifications};
    let config = service_config();
    let source = "fn classify(method: DeviceMethod) -> RpcRole { match method {
        DeviceMethod::Set => RpcRole::Action(ModelAction::Device(Device::Set)),
        DeviceMethod::Poll => RpcRole::Action(ModelAction::Device(Device::Poll(1))),
        DeviceMethod::Fail => RpcRole::Action(ModelAction::Device(Device::Failed { retryable: false })),
    }}";
    let parsed = classifications(source, config.service, &config.method_enum())?;
    assert_eq!(
        quote::ToTokens::to_token_stream(&parsed[0].actions[1].action).to_string().replace(' ', ""),
        "ModelAction::Device(Device::Poll(1))"
    );
    assert_eq!(
        quote::ToTokens::to_token_stream(&parsed[0].actions[2].action).to_string().replace(' ', ""),
        "ModelAction::Device(Device::Failed{retryable:false})"
    );

    assert!(
        matches!(&parsed[0].coverage.methods[1].disposition, Disposition::Action(name) if name == "Device::Poll")
    );
    assert!(
        matches!(&parsed[0].coverage.methods[2].disposition, Disposition::Action(name) if name == "Device::Failed")
    );
    let mut descriptors = fixture();
    for name in ["Poll", "Fail"] {
        let mut method = descriptors.file[0].service[0].method[0].clone();
        method.name = Some(name.into());
        descriptors.file[0].service[0].method.push(method);
    }
    let bytes = descriptors.encode_to_vec();
    let generated = generate_services(&bytes, &Coverage::default(), &parsed, &config)?;
    let directory =
        std::env::temp_dir().join(format!("catalog-payload-controls-{}", std::process::id()));
    fs::create_dir_all(&directory)?;
    let input = directory.join("control.rs");
    fs::write(
        &input,
        format!(
            r#"{generated}
        enum Device {{ Set, Poll(u8), Failed {{ retryable: bool }} }}
        enum ModelAction {{ Device(Device), Clock }}
        #[derive(Debug, PartialEq)] enum Binding {{ Rpc(&'static str), Environment }}
        fn bind(action: ModelAction) -> Binding {{ bind_device_action!(action, ModelAction::Clock => Binding::Environment) }}
        fn main() {{
            assert_eq!(bind(ModelAction::Device(Device::Set)), Binding::Rpc("example.v1.Device.Set"));
            for attempt in [1, 3] {{ assert_eq!(bind(ModelAction::Device(Device::Poll(attempt))), Binding::Rpc("example.v1.Device.Poll")); }}
            for retryable in [false, true] {{ assert_eq!(bind(ModelAction::Device(Device::Failed {{ retryable }})), Binding::Rpc("example.v1.Device.Fail")); }}
        }}"#
        ),
    )?;
    let binary = directory.join("control");
    let compiled = Command::new("rustc")
        .args(["--edition=2024", "-Awarnings"])
        .arg(&input)
        .arg("-o")
        .arg(&binary)
        .output()?;
    assert!(compiled.status.success(), "{}", String::from_utf8_lossy(&compiled.stderr));
    assert!(Command::new(&binary).status()?.success());
    for (first, second) in [
        ("Device::Poll(1)", "Device::Poll(3)"),
        ("Device::Failed { retryable: false }", "Device::Failed { retryable: true }"),
    ] {
        let duplicate = format!(
            "fn classify(method: DeviceMethod) -> RpcRole {{ match method {{
            DeviceMethod::Set => RpcRole::Action(ModelAction::Device({first})),
            DeviceMethod::Poll => RpcRole::Action(ModelAction::Device({second})),
        }} }}"
        );
        let parsed = classifications(&duplicate, config.service, &config.method_enum())?;
        let error = generate_services(&bytes, &Coverage::default(), &parsed, &config)
            .err()
            .ok_or("duplicate variant passed")?;
        assert!(error.contains("duplicate Model action RPC binding"), "{error}");
    }
    fs::remove_dir_all(directory)?;
    Ok(())
}
