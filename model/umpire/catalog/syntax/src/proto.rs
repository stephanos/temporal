//! The protobuf descriptor graph and its prost-generated Rust names.

use std::collections::{BTreeMap, BTreeSet};

use prost::Message as _;
use umpire_runtime::prost_types::{
    DescriptorProto, EnumDescriptorProto, FileDescriptorSet,
    field_descriptor_proto::{Label, Type},
};

use crate::{descriptor::well_known_primitive, naming};

/// Messages, enums and RPC edges of a descriptor set, named as prost generates them.
#[derive(Debug, Default)]
pub struct Graph {
    /// Messages keyed by protobuf identity, nested messages included.
    pub messages: BTreeMap<String, Message>,
    /// Enums keyed by protobuf identity, nested enums included.
    pub enums: BTreeMap<String, Enum>,
    /// Methods in descriptor source order, keyed by fully qualified service.
    pub services: BTreeMap<String, Vec<Method>>,
}

/// A protobuf message and its generated Rust representation.
#[derive(Clone, Debug)]
pub struct Message {
    /// Fully qualified protobuf identity.
    pub name: String,
    /// Fully qualified generated Rust type.
    pub rust_type: String,
    /// Whether this is a synthetic map-entry message.
    pub map_entry: bool,
    /// Fields in descriptor source order.
    pub fields: Vec<Field>,
}

/// A protobuf enum and its generated Rust representation.
#[derive(Clone, Debug)]
pub struct Enum {
    /// Fully qualified generated Rust type.
    pub rust_type: String,
    /// Defined values in descriptor order.
    pub values: Vec<EnumValue>,
}

/// One defined protobuf enum value.
#[derive(Clone, Debug)]
pub struct EnumValue {
    /// Original protobuf value name.
    pub name: String,
    /// Wire number.
    pub number: i32,
}

/// A generated protobuf field, retaining actual decoding presence.
#[derive(Clone, Debug)]
pub struct Field {
    /// Original protobuf field name.
    pub name: String,
    /// Protobuf field number.
    pub number: i32,
    /// Generated Rust member: an identifier, or `0`/`1` inside a map entry tuple.
    pub member: String,
    /// Value type before optional/repeated wrapping.
    pub kind: Kind,
    /// Repeated list or map.
    pub repeated: bool,
    /// A map, whose target is a synthetic map-entry message.
    pub map: bool,
    /// Protobuf presence represented by a Rust Option: messages, oneof members and proto3
    /// `optional` scalars. Proto2 optional scalars, which prost also wraps, are unsupported.
    pub optional: bool,
    /// A real oneof alternative, excluding synthetic proto3 optional groups.
    pub oneof: Option<Oneof>,
}

/// A protobuf field's value type.
#[derive(Clone, Debug)]
pub enum Kind {
    /// Scalar value.
    Scalar(Scalar),
    /// Nested message identity.
    Message {
        /// Fully qualified protobuf message identity.
        target: String,
    },
    /// Generated enum identity; decoded fields retain raw i32 values.
    Enum {
        /// Fully qualified protobuf enum identity.
        target: String,
    },
}

/// Scalar categories with identical decoded default-presence rules.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Scalar {
    /// UTF-8 text.
    String,
    /// Opaque bytes.
    Bytes,
    /// Boolean value.
    Bool,
    /// Floating point value.
    Float,
    /// Signed integer value.
    Signed,
    /// Unsigned integer value.
    Unsigned,
}

/// A generated oneof member and one of its tuple variants.
#[derive(Clone, Debug)]
pub struct Oneof {
    /// Rust Option member on the enclosing message.
    pub member: String,
    /// Fully qualified tuple variant path.
    pub variant: String,
}

/// An RPC's protobuf request and response edges.
#[derive(Clone, Debug)]
pub struct Method {
    /// Original protobuf method name.
    pub name: String,
    /// Fully qualified input message identity.
    pub input: String,
    /// Fully qualified output message identity.
    pub output: String,
}

impl Graph {
    /// Decode descriptors whose generated packages live under `rust_root` (empty: the current
    /// module); well-known types resolve to `prost_types` and its primitive carriers.
    pub fn new(bytes: &[u8], rust_root: &str) -> Result<Self, String> {
        let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
        let mut graph = Self::default();
        for file in &descriptors.file {
            let package = file.package.as_deref().unwrap_or_default();
            let modules = package.split('.').map(naming::snake).collect::<Vec<_>>().join("::");
            let module = if package == "google.protobuf" {
                "::umpire_runtime::prost_types".to_owned()
            } else if rust_root.is_empty() {
                modules
            } else {
                format!("{rust_root}::{modules}")
            };
            graph.collect(package, &module, &file.message_type)?;
            graph.collect_enums(package, &module, &file.enum_type)?;
            for service in &file.service {
                let service_name =
                    qualified(package, service.name.as_deref().ok_or("unnamed service")?);
                let methods = graph.services.entry(service_name.clone()).or_default();
                for method in &service.method {
                    let name = method.name.clone().ok_or("unnamed method")?;
                    let edge = |edge: &Option<String>, side: &str| {
                        edge.as_deref()
                            .map(clean)
                            .ok_or_else(|| format!("missing method {side}: {service_name}.{name}"))
                    };
                    methods.push(Method {
                        input: edge(&method.input_type, "input")?,
                        output: edge(&method.output_type, "output")?,
                        name,
                    });
                }
            }
        }
        let entries: BTreeSet<String> = graph
            .messages
            .values()
            .filter(|message| message.map_entry)
            .map(|message| message.name.clone())
            .collect();
        for field in graph.messages.values_mut().flat_map(|message| &mut message.fields) {
            field.map = matches!(&field.kind, Kind::Message { target } if entries.contains(target));
        }
        Ok(graph)
    }

    /// Resolve a message by protobuf identity, with or without a leading dot.
    pub fn message(&self, name: &str) -> Result<&Message, String> {
        self.messages.get(&clean(name)).ok_or_else(|| format!("missing descriptor message: {name}"))
    }

    /// A service's methods in descriptor source order.
    pub fn service(&self, service: &str) -> Result<&[Method], String> {
        self.services
            .get(&clean(service))
            .map(Vec::as_slice)
            .ok_or_else(|| format!("missing service descriptor: {service}"))
    }

    /// The fully qualified service whose last segment is `name`; `None` when no service has
    /// it, and an error when several packages do.
    pub fn service_named(&self, name: &str) -> Result<Option<&str>, String> {
        let matches: Vec<_> = self
            .services
            .keys()
            .filter(|service| service.rsplit('.').next() == Some(name))
            .map(String::as_str)
            .collect();
        match matches.as_slice() {
            [] => Ok(None),
            [service] => Ok(Some(service)),
            _ => Err(format!("ambiguous service short name {name}: {}", matches.join(", "))),
        }
    }

    /// Resolve the actual protobuf input/output edges of a service method.
    pub fn method(&self, service: &str, method: &str) -> Result<&Method, String> {
        self.services
            .get(&clean(service))
            .and_then(|methods| methods.iter().find(|edge| edge.name == method))
            .ok_or_else(|| format!("unknown RPC: {service}::{method}"))
    }

    /// Canonical RPC paths whose request has this protobuf identity.
    pub fn request_methods(&self, name: &str) -> Vec<String> {
        self.paths(|edge| edge.input == clean(name))
    }

    /// Canonical RPC paths whose response has this protobuf identity.
    pub fn response_methods(&self, name: &str) -> Vec<String> {
        self.paths(|edge| edge.output == clean(name))
    }

    fn paths(&self, matches: impl Fn(&Method) -> bool) -> Vec<String> {
        self.services
            .iter()
            .flat_map(|(service, methods)| {
                methods
                    .iter()
                    .filter(|edge| matches(edge))
                    .map(move |edge| format!("/{service}/{}", edge.name))
            })
            .collect()
    }

    fn collect(
        &mut self,
        package: &str,
        module: &str,
        descriptors: &[DescriptorProto],
    ) -> Result<(), String> {
        for descriptor in descriptors {
            let name = descriptor.name.as_deref().ok_or("unnamed message")?;
            let proto = qualified(package, name);
            let rust = well_known_primitive(&proto)
                .map_or_else(|| format!("{module}::{}", naming::upper_camel(name)), str::to_owned);
            let nested = format!("{module}::{}", naming::snake(name));
            let map_entry =
                descriptor.options.as_ref().is_some_and(|options| options.map_entry == Some(true));
            let mut fields = Vec::new();
            for field in &descriptor.field {
                let field_name = field.name.clone().ok_or("unnamed field")?;
                let kind = match Type::try_from(field.r#type.unwrap_or_default())
                    .map_err(|error| format!("invalid field type: {proto}.{field_name}: {error}"))?
                {
                    Type::String => Kind::Scalar(Scalar::String),
                    Type::Bytes => Kind::Scalar(Scalar::Bytes),
                    Type::Bool => Kind::Scalar(Scalar::Bool),
                    Type::Double | Type::Float => Kind::Scalar(Scalar::Float),
                    Type::Uint32 | Type::Uint64 | Type::Fixed32 | Type::Fixed64 => {
                        Kind::Scalar(Scalar::Unsigned)
                    }
                    Type::Int32
                    | Type::Int64
                    | Type::Sint32
                    | Type::Sint64
                    | Type::Sfixed32
                    | Type::Sfixed64 => Kind::Scalar(Scalar::Signed),
                    Type::Message => Kind::Message {
                        target: clean(field.type_name.as_deref().ok_or("missing message type")?),
                    },
                    Type::Enum => Kind::Enum {
                        target: clean(field.type_name.as_deref().ok_or("missing enum type")?),
                    },
                    Type::Group => return Err("protobuf groups are unsupported".into()),
                };
                let oneof = match field.oneof_index {
                    Some(index) if field.proto3_optional != Some(true) => {
                        let index = usize::try_from(index).map_err(|error| error.to_string())?;
                        let oneof = descriptor
                            .oneof_decl
                            .get(index)
                            .and_then(|oneof| oneof.name.as_deref())
                            .ok_or("missing oneof declaration")?;
                        Some(Oneof {
                            member: naming::snake(oneof),
                            variant: format!(
                                "{nested}::{}::{}",
                                naming::upper_camel(oneof),
                                naming::upper_camel(&field_name)
                            ),
                        })
                    }
                    _ => None,
                };
                let repeated = field.label == Some(Label::Repeated as i32);
                fields.push(Field {
                    member: if map_entry {
                        if field.number == Some(1) { "0" } else { "1" }.into()
                    } else {
                        naming::snake(&field_name)
                    },
                    name: field_name,
                    number: field.number.unwrap_or_default(),
                    optional: !map_entry
                        && !repeated
                        && (matches!(kind, Kind::Message { .. })
                            || field.oneof_index.is_some()
                            || field.proto3_optional == Some(true)),
                    kind,
                    repeated,
                    map: false,
                    oneof,
                });
            }
            self.collect_enums(&proto, &nested, &descriptor.enum_type)?;
            self.messages.insert(
                proto.clone(),
                Message { name: proto.clone(), rust_type: rust, map_entry, fields },
            );
            self.collect(&proto, &nested, &descriptor.nested_type)?;
        }
        Ok(())
    }

    fn collect_enums(
        &mut self,
        parent: &str,
        module: &str,
        descriptors: &[EnumDescriptorProto],
    ) -> Result<(), String> {
        for enumeration in descriptors {
            let name = enumeration.name.as_deref().ok_or("unnamed enum")?;
            self.enums.insert(
                qualified(parent, name),
                Enum {
                    rust_type: format!("{module}::{}", naming::upper_camel(name)),
                    values: enumeration
                        .value
                        .iter()
                        .map(|value| {
                            let value_name = value.name.clone().ok_or("unnamed enum value")?;
                            let number = value.number.ok_or_else(|| {
                                format!("enum value without number: {parent}.{name}.{value_name}")
                            })?;
                            Ok(EnumValue { name: value_name, number })
                        })
                        .collect::<Result<_, String>>()?,
                },
            );
        }
        Ok(())
    }
}

/// A protobuf identity without its leading dot.
pub fn clean(name: &str) -> String {
    name.trim_start_matches('.').to_owned()
}

fn qualified(parent: &str, name: &str) -> String {
    if parent.is_empty() { name.to_owned() } else { format!("{parent}.{name}") }
}

#[cfg(test)]
mod tests {
    use super::*;
    use umpire_runtime::prost_types::{
        EnumValueDescriptorProto, FieldDescriptorProto, FileDescriptorProto, MessageOptions,
        MethodDescriptorProto, OneofDescriptorProto, ServiceDescriptorProto,
    };

    fn field(name: &str, number: i32, r#type: Type) -> FieldDescriptorProto {
        FieldDescriptorProto {
            name: Some(name.into()),
            number: Some(number),
            label: Some(Label::Optional as i32),
            r#type: Some(r#type as i32),
            ..Default::default()
        }
    }

    fn bytes() -> Vec<u8> {
        set().encode_to_vec()
    }

    fn set() -> FileDescriptorSet {
        let message = |name: &str, target: &str| FieldDescriptorProto {
            type_name: Some(target.into()),
            ..field(name, 0, Type::Message)
        };
        FileDescriptorSet {
            file: vec![FileDescriptorProto {
                package: Some("self.v1".into()),
                service: vec![ServiceDescriptorProto {
                    name: Some("Service".into()),
                    method: vec![MethodDescriptorProto {
                        name: Some("Call".into()),
                        input_type: Some(".self.v1.Self_".into()),
                        output_type: Some(".self.v1._42Foo".into()),
                        ..Default::default()
                    }],
                    ..Default::default()
                }],
                message_type: vec![
                    DescriptorProto {
                        name: Some("Self_".into()),
                        field: vec![
                            field("self_", 1, Type::String),
                            FieldDescriptorProto {
                                oneof_index: Some(0),
                                ..message("first_choice", ".self.v1._42Foo")
                            },
                            FieldDescriptorProto {
                                oneof_index: Some(1),
                                proto3_optional: Some(true),
                                ..field("maybe", 3, Type::Int32)
                            },
                            FieldDescriptorProto {
                                label: Some(Label::Repeated as i32),
                                ..message("labels", ".self.v1.Self_.LabelsEntry")
                            },
                            message("wrapped", ".google.protobuf.StringValue"),
                            field("type", 6, Type::Bool),
                        ],
                        nested_type: vec![DescriptorProto {
                            name: Some("LabelsEntry".into()),
                            field: vec![
                                field("key", 1, Type::String),
                                field("value", 2, Type::Int64),
                            ],
                            options: Some(MessageOptions {
                                map_entry: Some(true),
                                ..Default::default()
                            }),
                            ..Default::default()
                        }],
                        enum_type: vec![EnumDescriptorProto {
                            name: Some("state".into()),
                            value: vec![EnumValueDescriptorProto {
                                name: Some("STATE_READY".into()),
                                number: Some(3),
                                ..Default::default()
                            }],
                            ..Default::default()
                        }],
                        oneof_decl: vec![
                            OneofDescriptorProto {
                                name: Some("choice".into()),
                                ..Default::default()
                            },
                            OneofDescriptorProto {
                                name: Some("_maybe".into()),
                                ..Default::default()
                            },
                        ],
                        ..Default::default()
                    },
                    DescriptorProto { name: Some("_42Foo".into()), ..Default::default() },
                ],
                ..Default::default()
            }],
        }
    }

    #[test]
    fn prost_names_escape_keywords_and_numeric_prefixes_under_the_rust_root() -> Result<(), String>
    {
        let graph = Graph::new(&bytes(), "::api")?;
        assert_eq!(graph.message("self.v1.Self_")?.rust_type, "::api::self_::v1::Self_");
        assert_eq!(graph.message(".self.v1._42Foo")?.rust_type, "::api::self_::v1::_42foo");
        assert_eq!(graph.message("self.v1.Self_")?.fields[0].member, "self_");
        let state = graph.enums.get("self.v1.Self_.state").ok_or("missing nested enum")?;
        assert_eq!(
            (state.rust_type.as_str(), state.values[0].name.as_str(), state.values[0].number),
            ("::api::self_::v1::self_::State", "STATE_READY", 3)
        );
        let local = Graph::new(&bytes(), "")?;
        assert_eq!(
            local.message("self.v1.Self_.LabelsEntry")?.rust_type,
            "self_::v1::self_::LabelsEntry"
        );
        Ok(())
    }

    #[test]
    fn fields_retain_decoded_presence_oneofs_and_maps() -> Result<(), String> {
        let graph = Graph::new(&bytes(), "")?;
        let fields = &graph.message("self.v1.Self_")?.fields;
        let [text, choice, maybe, labels, wrapped, keyword] = fields.as_slice() else {
            return Err("unexpected field count".into());
        };
        assert!(!text.optional && text.oneof.is_none());
        let oneof = choice.oneof.as_ref().ok_or("missing oneof")?;
        assert_eq!(oneof.member, "choice");
        assert_eq!(oneof.variant, "self_::v1::self_::Choice::FirstChoice");
        assert!(choice.optional);
        assert!(maybe.optional && maybe.oneof.is_none());
        assert!(labels.repeated && labels.map && !labels.optional);
        assert!(wrapped.optional && !wrapped.map);
        assert_eq!((keyword.member.as_str(), keyword.optional), ("r#type", false));
        let entry = graph.message("self.v1.Self_.LabelsEntry")?;
        assert!(entry.map_entry);
        assert_eq!(
            entry
                .fields
                .iter()
                .map(|field| (field.member.as_str(), field.optional))
                .collect::<Vec<_>>(),
            [("0", false), ("1", false)]
        );
        assert_eq!(
            graph.message("google.protobuf.StringValue").err(),
            Some("missing descriptor message: google.protobuf.StringValue".into())
        );
        Ok(())
    }

    #[test]
    fn rpc_edges_resolve_by_service_and_short_name() -> Result<(), String> {
        let graph = Graph::new(&bytes(), "")?;
        assert_eq!(graph.service_named("Service")?, Some("self.v1.Service"));
        assert_eq!(graph.service_named("Missing")?, None);
        assert_eq!(graph.method("self.v1.Service", "Call")?.output, "self.v1._42Foo");
        assert_eq!(graph.request_methods(".self.v1.Self_"), ["/self.v1.Service/Call"]);
        assert_eq!(graph.response_methods("self.v1._42Foo"), ["/self.v1.Service/Call"]);
        assert_eq!(graph.request_methods("self.v1._42Foo"), Vec::<String>::new());
        assert!(graph.method("self.v1.Service", "Missing").is_err());
        assert!(graph.service("self.v1.Missing").is_err());
        Ok(())
    }

    #[test]
    fn a_short_service_name_shared_by_two_packages_is_ambiguous() -> Result<(), String> {
        let mut descriptors = set();
        let mut other = descriptors.file[0].clone();
        other.package = Some("other.v1".into());
        other.message_type.clear();
        other.service[0].method.clear();
        descriptors.file.push(other);
        let graph = Graph::new(&descriptors.encode_to_vec(), "")?;
        assert_eq!(
            graph.service_named("Service"),
            Err("ambiguous service short name Service: other.v1.Service, self.v1.Service".into())
        );
        assert_eq!(graph.service("self.v1.Service")?.len(), 1);
        Ok(())
    }

    #[test]
    fn service_methods_keep_descriptor_order() -> Result<(), String> {
        let mut descriptors = set();
        let methods = &mut descriptors.file[0].service[0].method;
        let mut first = methods[0].clone();
        first.name = Some("Zeta".into());
        methods.insert(0, first);
        let graph = Graph::new(&descriptors.encode_to_vec(), "")?;
        let names: Vec<_> =
            graph.service("self.v1.Service")?.iter().map(|method| method.name.as_str()).collect();
        assert_eq!(names, ["Zeta", "Call"]);
        assert_eq!(
            graph.request_methods("self.v1.Self_"),
            ["/self.v1.Service/Zeta", "/self.v1.Service/Call"]
        );
        Ok(())
    }

    #[test]
    fn malformed_descriptors_are_refused_anywhere_in_the_set() {
        let cases: [(&str, fn(&mut FileDescriptorSet)); 12] = [
            ("unnamed service", |set| set.file[0].service[0].name = None),
            ("unnamed method", |set| set.file[0].service[0].method[0].name = None),
            ("unnamed message", |set| set.file[0].message_type[1].name = None),
            ("unnamed field", |set| set.file[0].message_type[0].field[0].name = None),
            ("unnamed enum", |set| set.file[0].message_type[0].enum_type[0].name = None),
            ("protobuf groups are unsupported", |set| {
                set.file[0].message_type[0].field[0].r#type = Some(Type::Group as i32);
            }),
            ("missing oneof declaration", |set| set.file[0].message_type[0].oneof_decl.clear()),
            ("missing method input: self.v1.Service.Call", |set| {
                set.file[0].service[0].method[0].input_type = None;
            }),
            ("missing method output: self.v1.Service.Call", |set| {
                set.file[0].service[0].method[0].output_type = None;
            }),
            ("invalid field type: self.v1.Self_.self_: unknown enumeration value 0", |set| {
                set.file[0].message_type[0].field[0].r#type = None;
            }),
            ("unnamed enum value", |set| {
                set.file[0].message_type[0].enum_type[0].value[0].name = None;
            }),
            ("enum value without number: self.v1.Self_.state.STATE_READY", |set| {
                set.file[0].message_type[0].enum_type[0].value[0].number = None;
            }),
        ];
        for (expected, corrupt) in cases {
            let mut descriptors = set();
            corrupt(&mut descriptors);
            assert_eq!(
                Graph::new(&descriptors.encode_to_vec(), "").err().as_deref(),
                Some(expected)
            );
        }
    }
}
