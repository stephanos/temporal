//! Descriptor-backed field admission shared by realization expansion and coverage.

use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
};

use proc_macro2::TokenStream;
use prost::Message as _;
use quote::{format_ident, quote};
use umpire_runtime::google::protobuf::{
    DescriptorProto, FileDescriptorSet,
    field_descriptor_proto::{Label, Type},
};

use crate::Catalog;

/// The protobuf graph admitted by the shared catalog declarations.
#[derive(Debug, Default)]
pub struct Schema {
    /// Admitted messages, keyed by protobuf identity.
    pub messages: BTreeMap<String, Message>,
    /// Admitted generated enums, keyed by protobuf identity.
    pub enums: BTreeMap<String, Enumeration>,
    methods: BTreeMap<(String, String), Method>,
}

/// A protobuf message and its generated Rust representation.
#[derive(Debug)]
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

/// A generated protobuf field, retaining actual decoding presence.
#[derive(Debug)]
pub struct Field {
    /// Original protobuf field name.
    pub name: String,
    /// Generated Rust member, including raw-identifier escaping.
    pub member: String,
    /// Value type before optional/repeated wrapping.
    pub kind: Kind,
    /// Repeated list or map.
    pub repeated: bool,
    /// Protobuf presence represented by a Rust Option.
    pub optional: bool,
    /// A real oneof alternative, excluding synthetic proto3 optional groups.
    pub oneof: Option<Oneof>,
    /// Catalog reason for stopping traversal at an opaque parent.
    pub opaque_reason: Option<String>,
}

/// A protobuf field's value type.
#[derive(Debug)]
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
#[derive(Debug)]
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
#[derive(Debug)]
pub struct Oneof {
    /// Rust Option member on the enclosing message.
    pub member: String,
    /// Fully qualified tuple variant path.
    pub variant: String,
}

/// An admitted generated enum.
#[derive(Debug)]
pub struct Enumeration {
    /// Fully qualified generated Rust type.
    pub rust_type: String,
    /// Fully qualified Rust variant paths from the shared declaration inventory.
    pub variants: Vec<String>,
}

/// An RPC's protobuf request and response edges.
#[derive(Clone, Debug)]
pub struct Method {
    /// Fully qualified input message identity.
    pub input: String,
    /// Fully qualified output message identity.
    pub output: String,
}

/// The declared treatment of a field in one realization context.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Classification {
    /// Read or produced by the handler in this context.
    Mapped,
    /// Explicitly ignored with a reason.
    Ignored,
    /// Presence refuses the context.
    Unsupported,
}

/// One classified raw protobuf field path.
#[derive(Debug)]
pub struct ClassifiedField {
    /// Dot-separated raw protobuf field path.
    pub path: String,
    /// Handler's declaration or the unsupported default.
    pub class: Classification,
    /// Authored explanation for an ignore.
    pub reason: Option<String>,
}

/// Catalog sources and their directory freshness inputs.
pub use umpire_catalog_build::coverage::Inputs as CatalogInputs;

/// Discover declarations using the framework-owned catalog filename policy.
pub fn catalog_inputs(root: &Path) -> Result<CatalogInputs, String> {
    umpire_catalog_build::coverage::inputs(root).map_err(|error| error.to_string())
}

impl Schema {
    /// Build from descriptors and the existing parsed annotation declarations.
    pub fn new(bytes: &[u8], catalogs: &[Catalog]) -> Result<Self, String> {
        let mut coverage = crate::descriptor::Coverage::default();
        for catalog in catalogs {
            let projected = catalog.coverage();
            coverage.messages.extend(projected.messages);
            coverage.enums.extend(projected.enums);
        }
        crate::descriptor::check(bytes, &coverage)?;
        let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
        let mut all = BTreeMap::new();
        let mut result = Self::default();
        for file in &descriptors.file {
            let package = file.package.as_deref().unwrap_or_default();
            let module = if package == "google.protobuf" {
                "::umpire_runtime::google::protobuf".to_owned()
            } else {
                format!("::umpire_temporal_api::{}", package.replace('.', "::"))
            };
            collect(package, &module, &file.message_type, &mut all)?;
            for service in &file.service {
                let service_name = qualified(package, service.name.as_deref().unwrap_or_default());
                for method in &service.method {
                    result.methods.insert(
                        (service_name.clone(), method.name.clone().ok_or("unnamed method")?),
                        Method {
                            input: clean(
                                method.input_type.as_deref().ok_or("missing method input")?,
                            ),
                            output: clean(
                                method.output_type.as_deref().ok_or("missing method output")?,
                            ),
                        },
                    );
                }
            }
        }
        for catalog in catalogs {
            for message in &catalog.messages {
                let name = clean(&message.proto.value());
                let mut node =
                    all.remove(&name).ok_or_else(|| format!("missing message: {name}"))?;
                for field in &message.fields {
                    if field.opaque {
                        let reason = field
                            .tags
                            .iter()
                            .find(|tag| tag.name == "Ignored")
                            .and_then(|tag| match &tag.argument {
                                Some(syn::Lit::Str(reason)) => Some(reason.value()),
                                _ => None,
                            })
                            .ok_or_else(|| {
                                format!("missing opaque reason: {name}.{}", field.name)
                            })?;
                        let names = if field.variants.is_empty() {
                            vec![field.name.to_string().trim_start_matches("r#").to_owned()]
                        } else {
                            field.variants.iter().map(|variant| variant.proto.value()).collect()
                        };
                        for name in names {
                            if let Some(actual) =
                                node.fields.iter_mut().find(|actual| actual.name == name)
                            {
                                actual.opaque_reason = Some(reason.clone());
                            }
                        }
                    }
                }
                result.messages.insert(name, node);
            }
            for enumeration in &catalog.enums {
                use quote::ToTokens;
                result.enums.insert(
                    clean(&enumeration.proto.value()),
                    Enumeration {
                        rust_type: enumeration.ty.to_token_stream().to_string(),
                        variants: enumeration
                            .variants
                            .iter()
                            .map(|variant| variant.path.to_token_stream().to_string())
                            .collect(),
                    },
                );
            }
        }
        Ok(result)
    }

    /// Load precisely the annotation inputs used by API generation, returning tracked paths.
    pub fn load(root: &Path, descriptor: &Path) -> Result<(Self, Vec<PathBuf>), String> {
        let mut inputs = catalog_inputs(root)?.files;
        let mut catalogs = Vec::new();
        for path in &inputs {
            let source =
                fs::read_to_string(path).map_err(|error| format!("{}: {error}", path.display()))?;
            catalogs.extend(
                crate::catalogs(&source).map_err(|error| format!("{}: {error}", path.display()))?,
            );
        }
        let bytes =
            fs::read(descriptor).map_err(|error| format!("{}: {error}", descriptor.display()))?;
        inputs.push(descriptor.to_owned());
        Ok((Self::new(&bytes, &catalogs)?, inputs))
    }

    /// Resolve the actual protobuf input/output edges of a service method.
    pub fn method(&self, service: &str, method: &str) -> Result<&Method, String> {
        self.methods
            .get(&(clean(service), method.to_owned()))
            .ok_or_else(|| format!("unknown RPC: {service}::{method}"))
    }

    /// Canonical RPC paths whose request has this protobuf identity.
    pub fn request_methods(&self, name: &str) -> Vec<String> {
        self.methods
            .iter()
            .filter(|(_, edge)| edge.input == clean(name))
            .map(|((service, method), _)| format!("/{service}/{method}"))
            .collect()
    }

    /// Canonical RPC paths whose response has this protobuf identity.
    pub fn response_methods(&self, name: &str) -> Vec<String> {
        self.methods
            .iter()
            .filter(|(_, edge)| edge.output == clean(name))
            .map(|((service, method), _)| format!("/{service}/{method}"))
            .collect()
    }

    /// Resolve a message admitted by the shared catalog declarations.
    pub fn message(&self, name: &str) -> Result<&Message, String> {
        self.messages
            .get(&clean(name))
            .ok_or_else(|| format!("unadmitted protobuf message: {name}"))
    }

    /// Validate declarations and classify every field boundary in source order.
    pub fn fields(
        &self,
        name: &str,
        reads: &[String],
        ignores: &[(String, String)],
    ) -> Result<Vec<ClassifiedField>, String> {
        let mut declarations: BTreeMap<String, (Classification, Option<String>)> = BTreeMap::new();
        for (path, class, reason) in
            reads.iter().map(|path| (path, Classification::Mapped, None)).chain(
                ignores.iter().map(|(path, reason)| (path, Classification::Ignored, Some(reason))),
            )
        {
            self.resolve(name, path)?;
            if reason.is_some_and(|reason| reason.trim().is_empty()) {
                return Err(format!("ignore requires a nonempty reason: {path}"));
            }
            for previous in declarations.keys() {
                if overlaps(previous, path) {
                    return Err(format!("overlapping field paths: {previous} and {path}"));
                }
            }
            declarations.insert(path.clone(), (class, reason.cloned()));
        }
        let mut result = Vec::new();
        self.classify(name, "", &declarations, &mut result)?;
        Ok(result)
    }

    fn classify(
        &self,
        name: &str,
        parent: &str,
        declarations: &BTreeMap<String, (Classification, Option<String>)>,
        result: &mut Vec<ClassifiedField>,
    ) -> Result<(), String> {
        for field in &self.message(name)?.fields {
            let path = qualified(parent, &field.name);
            if let Some((class, reason)) = declarations.get(&path) {
                result.push(ClassifiedField { path, class: *class, reason: reason.clone() });
            } else if declarations.keys().any(|declared| declared.starts_with(&format!("{path}.")))
            {
                let Kind::Message { target } = &field.kind else {
                    return Err(format!("field has no nested paths: {path}"));
                };
                self.classify(target, &path, declarations, result)?;
            } else {
                result.push(ClassifiedField {
                    path,
                    class: Classification::Unsupported,
                    reason: None,
                });
            }
        }
        Ok(())
    }

    fn resolve(&self, name: &str, path: &str) -> Result<(), String> {
        let (head, tail) =
            path.split_once('.').map_or((path, None), |(head, tail)| (head, Some(tail)));
        let field = self
            .message(name)?
            .fields
            .iter()
            .find(|field| field.name == head)
            .ok_or_else(|| format!("unknown field path: {name}.{path}"))?;
        if let Some(tail) = tail {
            if field.opaque_reason.is_some() {
                return Err(format!("opaque field stops nested path: {name}.{path}"));
            }
            let Kind::Message { target } = &field.kind else {
                return Err(format!("field has no nested paths: {name}.{path}"));
            };
            self.resolve(target, tail)?;
        }
        Ok(())
    }

    /// Emit decoded-field presence checks, independently of annotation metadata presence.
    pub fn guards(
        &self,
        name: &str,
        fields: &[ClassifiedField],
        value: &TokenStream,
    ) -> Result<TokenStream, String> {
        let mut checks = TokenStream::new();
        for field in fields.iter().filter(|field| field.class == Classification::Unsupported) {
            let present = self.presence(name, &field.path, value, 0)?;
            let message = format!("unsupported realization field: {name}.{}", field.path);
            checks.extend(quote! { if #present { return Err(::umpire_runtime::protocol::Refusal::unsupported(#message)); } });
        }
        Ok(checks)
    }

    fn presence(
        &self,
        name: &str,
        path: &str,
        value: &TokenStream,
        depth: usize,
    ) -> Result<TokenStream, String> {
        let (head, tail) =
            path.split_once('.').map_or((path, None), |(head, tail)| (head, Some(tail)));
        let message = self.message(name)?;
        let field = message
            .fields
            .iter()
            .find(|field| field.name == head)
            .ok_or_else(|| format!("unknown field path: {name}.{path}"))?;
        let binding = format_ident!("__realization_field_{depth}");
        if let Some(oneof) = &field.oneof {
            let member: syn::Member =
                syn::parse_str(&oneof.member).map_err(|error| error.to_string())?;
            let variant: syn::Path =
                syn::parse_str(&oneof.variant).map_err(|error| error.to_string())?;
            let nested = if let Some(tail) = tail {
                let Kind::Message { target } = &field.kind else {
                    return Err("nested scalar path".into());
                };
                self.presence(target, tail, &quote!(#binding), depth + 1)?
            } else {
                quote!(true)
            };
            return Ok(
                quote!(match &(#value).#member { Some(#variant(#binding)) => #nested, _ => false }),
            );
        }
        let member: syn::Member =
            syn::parse_str(&field.member).map_err(|error| error.to_string())?;
        let access =
            if message.map_entry { quote!(*(#value).#member) } else { quote!((#value).#member) };
        if let Some(tail) = tail {
            let Kind::Message { target } = &field.kind else {
                return Err("nested scalar path".into());
            };
            let nested = self.presence(target, tail, &quote!(#binding), depth + 1)?;
            if field.repeated {
                return Ok(quote!((#access).iter().any(|#binding| #nested)));
            }
            if !field.optional {
                return self.presence(target, tail, &quote!(&(#access)), depth + 1);
            }
            return Ok(quote!((#access).as_ref().is_some_and(|#binding| #nested)));
        }
        if field.repeated {
            return Ok(quote!(!(#access).is_empty()));
        }
        if field.optional {
            return Ok(quote!((#access).is_some()));
        }
        Ok(match field.kind {
            Kind::Scalar(Scalar::String | Scalar::Bytes) => quote!(!(#access).is_empty()),
            Kind::Scalar(Scalar::Bool) => access,
            Kind::Scalar(Scalar::Float) => quote!((#access) != 0.0),
            Kind::Scalar(Scalar::Signed | Scalar::Unsigned) | Kind::Enum { .. } => {
                quote!((#access) != 0)
            }
            Kind::Message { .. } => quote!(true),
        })
    }
}

fn clean(name: &str) -> String {
    name.trim_start_matches('.').to_owned()
}

fn qualified(parent: &str, name: &str) -> String {
    if parent.is_empty() { name.to_owned() } else { format!("{parent}.{name}") }
}

fn overlaps(left: &str, right: &str) -> bool {
    left == right
        || left.starts_with(&format!("{right}."))
        || right.starts_with(&format!("{left}."))
}

fn collect(
    package: &str,
    module: &str,
    descriptors: &[DescriptorProto],
    result: &mut BTreeMap<String, Message>,
) -> Result<(), String> {
    for descriptor in descriptors {
        let name = descriptor.name.as_deref().ok_or("unnamed message")?;
        let proto = qualified(package, name);
        let rust = format!("{module}::{name}");
        let map_entry =
            descriptor.options.as_ref().is_some_and(|options| options.map_entry == Some(true));
        let mut fields = Vec::new();
        for field in &descriptor.field {
            let name = field.name.clone().ok_or("unnamed field")?;
            let kind = match Type::try_from(field.r#type.unwrap_or_default())
                .map_err(|error| error.to_string())?
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
            let oneof = if field.proto3_optional == Some(true) {
                None
            } else {
                field
                    .oneof_index
                    .map(|index| {
                        let index = usize::try_from(index).map_err(|error| error.to_string())?;
                        let oneof = descriptor
                            .oneof_decl
                            .get(index)
                            .and_then(|oneof| oneof.name.as_deref())
                            .ok_or("missing oneof declaration")?;
                        Ok::<_, String>(Oneof {
                            member: format!("r#{}", snake(oneof)),
                            variant: format!(
                                "{module}::{}::{}::{}",
                                snake(descriptor.name.as_deref().ok_or("unnamed message")?),
                                camel(oneof),
                                camel(&name)
                            ),
                        })
                    })
                    .transpose()?
            };
            let repeated = field.label == Some(Label::Repeated as i32);
            fields.push(Field {
                member: if map_entry {
                    if field.number == Some(1) { "0" } else { "1" }.into()
                } else {
                    format!("r#{}", snake(&name))
                },
                name,
                kind,
                repeated,
                optional: !map_entry
                    && !repeated
                    && (field.r#type == Some(Type::Message as i32)
                        || field.oneof_index.is_some()
                        || field.proto3_optional == Some(true)),
                oneof,
                opaque_reason: None,
            });
        }
        result.insert(
            proto.clone(),
            Message { name: proto.clone(), rust_type: rust, map_entry, fields },
        );
        collect(&proto, &format!("{module}::{}", snake(name)), &descriptor.nested_type, result)?;
    }
    Ok(())
}

fn snake(name: &str) -> String {
    let chars: Vec<_> = name.chars().collect();
    let mut result = String::new();
    for (index, ch) in chars.iter().enumerate() {
        if ch.is_uppercase()
            && index > 0
            && (chars[index - 1].is_lowercase()
                || chars[index - 1].is_ascii_digit()
                || chars.get(index + 1).is_some_and(|next| next.is_lowercase()))
        {
            result.push('_');
        }
        result.extend(ch.to_lowercase());
    }
    result
}

fn camel(name: &str) -> String {
    snake(name)
        .split('_')
        .map(|part| {
            let mut chars = part.chars();
            chars.next().map_or_else(String::new, |head| head.to_uppercase().chain(chars).collect())
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use umpire_runtime::google::protobuf::{
        DescriptorProto, FieldDescriptorProto, FileDescriptorProto, FileDescriptorSet,
        MethodDescriptorProto, ServiceDescriptorProto,
        field_descriptor_proto::{Label, Type},
    };

    fn fixture() -> Result<Schema, String> {
        let catalogs = crate::catalogs(
            r#"
            #[derive(Catalog)] #[mirror(response = "test.Response")]
            struct Response { #[mirror(decided)] info: Message<Info>, extra: Text }
            #[derive(Catalog)] #[mirror(response = "test.Info")]
            struct Info {
                status: Counter,
                #[mirror(ignored = "external storage is opaque")]
                external: Ignored,
            }
        "#,
        )
        .map_err(|error| error.to_string())?;
        let scalar = |name: &str, number| FieldDescriptorProto {
            name: Some(name.into()),
            number: Some(number),
            label: Some(Label::Optional as i32),
            r#type: Some(Type::String as i32),
            ..Default::default()
        };
        let bytes = FileDescriptorSet {
            file: vec![FileDescriptorProto {
                package: Some("test".into()),
                service: vec![ServiceDescriptorProto {
                    name: Some("Service".into()),
                    method: vec![MethodDescriptorProto {
                        name: Some("Call".into()),
                        input_type: Some(".test.Response".into()),
                        output_type: Some(".test.Info".into()),
                        ..Default::default()
                    }],
                    ..Default::default()
                }],
                message_type: vec![
                    DescriptorProto {
                        name: Some("Response".into()),
                        field: vec![
                            FieldDescriptorProto {
                                name: Some("info".into()),
                                number: Some(1),
                                label: Some(Label::Optional as i32),
                                r#type: Some(Type::Message as i32),
                                type_name: Some(".test.Info".into()),
                                ..Default::default()
                            },
                            scalar("extra", 2),
                        ],
                        ..Default::default()
                    },
                    DescriptorProto {
                        name: Some("Info".into()),
                        field: vec![
                            FieldDescriptorProto {
                                r#type: Some(Type::Int32 as i32),
                                ..scalar("status", 1)
                            },
                            scalar("external", 2),
                        ],
                        ..Default::default()
                    },
                ],
                ..Default::default()
            }],
        }
        .encode_to_vec();
        Schema::new(&bytes, &catalogs)
    }

    #[test]
    fn graph_retains_decoded_optional_presence_and_opaque_catalog_reason() -> Result<(), String> {
        let schema = fixture()?;
        let response = schema.messages.get("test.Response").ok_or("missing response graph")?;
        assert!(response.fields[0].optional);
        assert_eq!(response.fields[0].member, "r#info");
        assert_eq!(
            schema.messages["test.Info"].fields[1].opaque_reason.as_deref(),
            Some("external storage is opaque")
        );
        Ok(())
    }

    #[test]
    fn contexts_follow_rpc_input_output_edges_instead_of_message_admission() -> Result<(), String> {
        let schema = fixture()?;
        assert_eq!(schema.request_methods("test.Response"), ["/test.Service/Call"]);
        assert_eq!(schema.response_methods("test.Info"), ["/test.Service/Call"]);
        assert_eq!(schema.request_methods("test.Info"), Vec::<String>::new());
        assert_eq!(schema.response_methods("test.Response"), Vec::<String>::new());
        assert_eq!(schema.request_methods("test.Missing"), Vec::<String>::new());
        Ok(())
    }

    #[test]
    fn catalog_discovery_tracks_empty_directories_and_only_admitted_filenames()
    -> Result<(), Box<dyn std::error::Error>> {
        let root =
            std::env::temp_dir().join(format!("umpire-realization-inputs-{}", std::process::id()));
        fs::create_dir(&root)?;
        let result = (|| -> Result<(), Box<dyn std::error::Error>> {
            fs::create_dir(root.join("empty"))?;
            fs::create_dir(root.join("nested"))?;
            fs::write(root.join("api.rs"), "catalog source")?;
            fs::write(root.join("noise.rs"), "not a catalog")?;
            fs::write(root.join("nested/enum_inventory.rs"), "enum source")?;
            let actual = catalog_inputs(&root)?;
            if actual.files != [root.join("api.rs"), root.join("nested/enum_inventory.rs")]
                || actual.directories != [root.clone(), root.join("empty"), root.join("nested")]
            {
                return Err(format!(
                    "catalog discovery omitted sources or directory freshness inputs: {actual:?}"
                )
                .into());
            }
            Ok(())
        })();
        fs::remove_dir_all(root)?;
        result
    }
}
