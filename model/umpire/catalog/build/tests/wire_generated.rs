//! Generated wire setters compile against prost-shaped message and collection carriers.
use prost::Message;
use umpire_catalog_build::wire_schema;
use umpire_runtime::google::protobuf::{
    DescriptorProto, FieldDescriptorProto, FileDescriptorProto, FileDescriptorSet, MessageOptions,
    MethodDescriptorProto, OneofDescriptorProto, ServiceDescriptorProto,
    field_descriptor_proto::{Label, Type},
};

#[test]
fn generated_wire_setters_compile_and_preserve_carrier_mutations()
-> Result<(), Box<dyn std::error::Error>> {
    let mut children = field("children", 1, Type::Message, Some(".example.v1.HTTPChild"));
    children.label = Some(Label::Repeated as i32);
    let mut labels = field("labels", 2, Type::Message, Some(".example.v1.HTTPInput.LabelsEntry"));
    labels.label = Some(Label::Repeated as i32);
    let mut choice_child = field("choice_child", 3, Type::Message, Some(".example.v1.HTTPChild"));
    choice_child.oneof_index = Some(0);
    let mut choice_text = field("choice_text", 4, Type::String, None);
    choice_text.oneof_index = Some(0);
    let mut numbers = field("numbers", 6, Type::Int32, None);
    numbers.label = Some(Label::Repeated as i32);
    let descriptors = FileDescriptorSet {
        file: vec![FileDescriptorProto {
            package: Some("example.v1".into()),
            service: vec![ServiceDescriptorProto {
                name: Some("Device".into()),
                method: vec![MethodDescriptorProto {
                    name: Some("Set".into()),
                    input_type: Some(".example.v1.HTTPInput".into()),
                    output_type: Some(".example.v1.HTTPInput".into()),
                    ..Default::default()
                }],
                ..Default::default()
            }],
            message_type: vec![
                DescriptorProto {
                    name: Some("HTTPInput".into()),
                    field: vec![
                        children,
                        labels,
                        choice_child,
                        choice_text,
                        field("nested", 5, Type::Message, Some(".example.v1.HTTPChild")),
                        numbers,
                    ],
                    oneof_decl: vec![OneofDescriptorProto {
                        name: Some("__HTTP__choice__".into()),
                        ..Default::default()
                    }],
                    nested_type: vec![DescriptorProto {
                        name: Some("LabelsEntry".into()),
                        field: vec![
                            field("key", 1, Type::String, None),
                            field("value", 2, Type::Message, Some(".example.v1.HTTPChild")),
                        ],
                        options: Some(MessageOptions {
                            map_entry: Some(true),
                            ..Default::default()
                        }),
                        ..Default::default()
                    }],
                    ..Default::default()
                },
                DescriptorProto {
                    name: Some("HTTPChild".into()),
                    field: vec![field("HTTP__value", 1, Type::String, None)],
                    ..Default::default()
                },
            ],
            ..Default::default()
        }],
    };
    let generated = wire_schema::generate(
        &descriptors.encode_to_vec(),
        &wire_schema::Config {
            service: "example.v1.Device",
            rust_type_root: "crate",
            method_enum: "DeviceMethod",
            fields_function: "device_fields",
            mutate_function: "mutate_device",
            request_roots: &["Set"],
        },
    )?;
    let directory =
        std::env::temp_dir().join(format!("catalog-wire-controls-{}", std::process::id()));
    std::fs::create_dir_all(&directory)?;
    let input = directory.join("control.rs");
    std::fs::write(
        &input,
        format!(
            "{CARRIERS}\nmod generated {{ use crate::DeviceMethod;\n{generated}\n{ASSERTIONS}\n}}\nfn main() {{ generated::check(); }}"
        ),
    )?;
    let binary = directory.join("control");
    let compiled = std::process::Command::new("rustc")
        .args(["--edition=2024", "-Awarnings"])
        .arg(&input)
        .arg("-o")
        .arg(&binary)
        .output()?;
    assert!(compiled.status.success(), "{}", String::from_utf8_lossy(&compiled.stderr));
    assert!(std::process::Command::new(&binary).status()?.success());
    std::fs::remove_dir_all(directory)?;
    Ok(())
}

fn field(name: &str, number: i32, ty: Type, target: Option<&str>) -> FieldDescriptorProto {
    FieldDescriptorProto {
        name: Some(name.into()),
        number: Some(number),
        r#type: Some(ty as i32),
        label: Some(Label::Optional as i32),
        type_name: target.map(str::to_owned),
        ..Default::default()
    }
}

const CARRIERS: &str = r#"
extern crate self as umpire_catalog;
extern crate self as prost;
#[derive(Debug)] pub enum GenerationError { InvalidEncoding, UnsupportedType, InapplicableMutation, SizeLimit }
pub enum WireKind { String, Message, Repeated, Map, Signed }
pub enum WireMutation { Absent, Empty, LimitPlusOne { limit: u64 }, Change }
pub struct WireField { pub path: String, pub declaration: String, pub kind: WireKind, pub optional: bool, pub refusal: Option<String> }
pub trait WireValue { fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError>; }
impl WireValue for String { fn mutate(&mut self, _: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> { *self = if matches!(mutation, WireMutation::Absent | WireMutation::Empty) { String::new() } else { "changed".into() }; Ok(()) } }
impl WireValue for i32 { fn mutate(&mut self, _: &WireKind, _: &WireMutation) -> Result<(), GenerationError> { *self = 17; Ok(()) } }
pub trait Message: Sized { fn decode(bytes: &[u8]) -> Result<Self, ()>; fn encode_to_vec(&self) -> Vec<u8>; }
pub enum DeviceMethod { Set }
impl DeviceMethod { fn name(self) -> &'static str { "Set" } }
pub mod example { pub mod v1 {
    #[derive(Default)] pub struct HttpInput {
        pub children: Vec<HttpChild>, pub labels: std::collections::BTreeMap<String, HttpChild>,
        pub http_choice: Option<http_input::HttpChoice>, pub nested: Option<HttpChild>, pub numbers: Vec<i32>,
    }
    #[derive(Default)] pub struct HttpChild { pub http_value: String }
    pub mod http_input { pub enum HttpChoice { ChoiceChild(super::HttpChild), ChoiceText(String) } }
    impl crate::Message for HttpInput { fn decode(_: &[u8]) -> Result<Self, ()> { Ok(Self::default()) } fn encode_to_vec(&self) -> Vec<u8> { vec![] } }
} }
"#;

const ASSERTIONS: &str = r#"
pub fn check() {
    use crate::example::v1::{HttpInput, http_input::HttpChoice};
    let mut message = HttpInput::default();
    let prefix = "example.v1.HTTPInput";
    let mutate = |message: &mut HttpInput, field: &str, mutation: &WireMutation| mutate_set(message, &format!("{prefix}.{field}"), mutation);
    for field in ["nested.HTTP__value", "children.HTTP__value", "labels.value.HTTP__value", "choice_child.HTTP__value"] {
        mutate(&mut message, field, &WireMutation::Absent).unwrap();
    }
    assert!(message.nested.is_none() && message.children.is_empty() && message.labels.is_empty() && message.http_choice.is_none());
    mutate(&mut message, "nested.HTTP__value", &WireMutation::Change).unwrap();
    assert_eq!(message.nested.as_ref().unwrap().http_value, "changed");
    mutate(&mut message, "children.HTTP__value", &WireMutation::Change).unwrap();
    assert_eq!(message.children[0].http_value, "changed");
    mutate(&mut message, "labels.value.HTTP__value", &WireMutation::Change).unwrap();
    assert_eq!(message.labels[""].http_value, "changed");
    mutate(&mut message, "labels.value.HTTP__value", &WireMutation::Absent).unwrap();
    assert_eq!(message.labels[""].http_value, "");
    mutate(&mut message, "choice_child.HTTP__value", &WireMutation::Change).unwrap();
    assert!(matches!(&message.http_choice, Some(HttpChoice::ChoiceChild(child)) if child.http_value == "changed"));
    mutate(&mut message, "choice_text", &WireMutation::Change).unwrap();
    assert!(matches!(&message.http_choice, Some(HttpChoice::ChoiceText(text)) if text == "changed"));
    mutate(&mut message, "choice_child", &WireMutation::Absent).unwrap();
    assert!(matches!(&message.http_choice, Some(HttpChoice::ChoiceText(_))));
    mutate(&mut message, "choice_text", &WireMutation::Absent).unwrap();
    assert!(message.http_choice.is_none());
    mutate(&mut message, "numbers.value", &WireMutation::Change).unwrap();
    assert_eq!(message.numbers, vec![17]);
    mutate(&mut message, "numbers", &WireMutation::LimitPlusOne { limit: 2 }).unwrap();
    assert_eq!(message.numbers, vec![0; 3]);
    assert!(matches!(mutate(&mut message, "numbers", &WireMutation::LimitPlusOne { limit: u64::MAX }), Err(GenerationError::SizeLimit)));
    assert!(matches!(mutate(&mut message, "numbers", &WireMutation::Change), Err(GenerationError::InapplicableMutation)));
    assert!(matches!(mutate(&mut message, "missing", &WireMutation::Change), Err(GenerationError::UnsupportedType)));
    assert!(!device_fields(DeviceMethod::Set).is_empty());
}
"#;
