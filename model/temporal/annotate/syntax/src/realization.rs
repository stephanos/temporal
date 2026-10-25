//! Descriptor-backed field admission shared by realization expansion and coverage.

use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};

use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use umpire_catalog_syntax::proto::clean;
pub use umpire_catalog_syntax::proto::{Field, Graph, Kind, Message, Method, Scalar};

use crate::Catalog;

/// The protobuf graph admitted by the shared catalog declarations.
#[derive(Debug, Default)]
pub struct Schema {
    graph: Graph,
    admitted: BTreeSet<String>,
    opaque: BTreeSet<(String, String)>,
    /// Admitted generated enums, keyed by protobuf identity.
    pub enums: BTreeMap<String, Enumeration>,
}

/// An admitted generated enum.
#[derive(Debug)]
pub struct Enumeration {
    /// Fully qualified generated Rust type.
    pub rust_type: String,
    /// Fully qualified Rust variant paths from the shared declaration inventory.
    pub variants: Vec<String>,
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
    /// Build from descriptors generated under `rust_root` and the parsed catalog declarations.
    pub fn new(bytes: &[u8], rust_root: &str, catalogs: &[Catalog]) -> Result<Self, String> {
        let mut coverage = crate::descriptor::Coverage::default();
        for catalog in catalogs {
            let projected = catalog.coverage();
            coverage.messages.extend(projected.messages);
            coverage.enums.extend(projected.enums);
        }
        let graph = Graph::new(bytes, rust_root)
            .map_err(|error| format!("invalid protobuf descriptor set: {error}"))?;
        crate::descriptor::admit(&graph, &coverage)?;
        let mut result = Self { graph, ..Self::default() };
        for catalog in catalogs {
            for message in &catalog.messages {
                let name = clean(&message.proto.value());
                let node =
                    result.graph.message(&name).map_err(|_| format!("missing message: {name}"))?;
                for field in &message.fields {
                    if field.opaque {
                        if !field.tags.iter().any(|tag| {
                            tag.name == "Ignored" && matches!(tag.argument, Some(syn::Lit::Str(_)))
                        }) {
                            return Err(format!("missing opaque reason: {name}.{}", field.name));
                        }
                        let names = if field.variants.is_empty() {
                            vec![field.name.to_string().trim_start_matches("r#").to_owned()]
                        } else {
                            field.variants.iter().map(|variant| variant.proto.value()).collect()
                        };
                        for field in names {
                            if node.fields.iter().any(|actual| actual.name == field) {
                                result.opaque.insert((name.clone(), field));
                            }
                        }
                    }
                }
                result.admitted.insert(name);
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
    pub fn load(
        root: &Path,
        descriptor: &Path,
        rust_root: &str,
    ) -> Result<(Self, Vec<PathBuf>), String> {
        let bytes =
            fs::read(descriptor).map_err(|error| format!("{}: {error}", descriptor.display()))?;
        let graph = Graph::new(&bytes, rust_root)?;
        let mut inputs = catalog_inputs(root)?.files;
        let mut catalogs = Vec::new();
        for path in &inputs {
            let mut source =
                fs::read_to_string(path).map_err(|error| format!("{}: {error}", path.display()))?;
            if path.file_name().is_some_and(|name| name == "claims.rs") {
                source = umpire_catalog_build::mirrors::expand(&source, &graph)
                    .map_err(|error| format!("{}: {error}", path.display()))?;
            }
            catalogs.extend(
                crate::catalogs(&source).map_err(|error| format!("{}: {error}", path.display()))?,
            );
        }
        inputs.push(descriptor.to_owned());
        Ok((Self::new(&bytes, rust_root, &catalogs)?, inputs))
    }

    /// The fully qualified service whose last segment is `name`; see [`Graph::service_named`].
    pub fn service_named(&self, name: &str) -> Result<Option<&str>, String> {
        self.graph.service_named(name)
    }

    /// Resolve the actual protobuf input/output edges of a service method.
    pub fn method(&self, service: &str, method: &str) -> Result<&Method, String> {
        self.graph.method(service, method)
    }

    /// Canonical RPC paths whose request has this protobuf identity.
    pub fn request_methods(&self, name: &str) -> Vec<String> {
        self.graph.request_methods(name)
    }

    /// Canonical RPC paths whose response has this protobuf identity.
    pub fn response_methods(&self, name: &str) -> Vec<String> {
        self.graph.response_methods(name)
    }

    /// Messages admitted by the shared catalog declarations.
    pub fn messages(&self) -> impl Iterator<Item = &Message> {
        self.admitted.iter().filter_map(|name| self.graph.messages.get(name))
    }

    /// Resolve a message admitted by the shared catalog declarations.
    pub fn message(&self, name: &str) -> Result<&Message, String> {
        self.admitted
            .get(&clean(name))
            .and_then(|name| self.graph.messages.get(name))
            .ok_or_else(|| format!("unadmitted protobuf message: {name}"))
    }

    /// Generated Rust type of a descriptor enum.
    pub fn enum_type(&self, name: &str) -> Option<&str> {
        self.graph.enums.get(&clean(name)).map(|enumeration| enumeration.rust_type.as_str())
    }

    fn opaque(&self, message: &str, field: &str) -> bool {
        self.opaque.contains(&(clean(message), field.to_owned()))
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
            if self.opaque(name, head) {
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
            if umpire_catalog_syntax::descriptor::well_known_primitive(&message.name).is_some() {
                quote!(*(#value))
            } else if message.map_entry {
                quote!(*(#value).#member)
            } else {
                quote!((#value).#member)
            };
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

fn qualified(parent: &str, name: &str) -> String {
    if parent.is_empty() { name.to_owned() } else { format!("{parent}.{name}") }
}

fn overlaps(left: &str, right: &str) -> bool {
    left == right
        || left.starts_with(&format!("{right}."))
        || right.starts_with(&format!("{left}."))
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message as _;
    use umpire_runtime::prost_types::{
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
        Schema::new(&bytes, "::api", &catalogs)
    }

    #[test]
    fn graph_retains_decoded_optional_presence_and_opaque_catalog_reason() -> Result<(), String> {
        let schema = fixture()?;
        let response = schema.message("test.Response")?;
        assert!(response.fields[0].optional);
        assert_eq!(response.rust_type, "::api::test::Response");
        assert!(schema.opaque("test.Info", "external"));
        assert!(!schema.opaque("test.Info", "status"));
        assert_eq!(
            schema.resolve("test.Response", "info.external.value").err().as_deref(),
            Some("opaque field stops nested path: test.Info.external.value")
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
