//! Generate raw descriptor inventories and typed protobuf mutation setters.

use std::collections::BTreeMap;

use heck::ToSnakeCase;
use proc_macro2::TokenStream;
use quote::{format_ident, quote};

use prost::Message;
use umpire_runtime::google::protobuf::{
    DescriptorProto, FieldDescriptorProto, FileDescriptorSet,
    field_descriptor_proto::{Label, Type},
};

/// Request generation identities and explicitly admitted roots.
pub struct Config<'a> {
    /// Fully qualified protobuf service name.
    pub service: &'a str,
    /// Rust module containing generated packages; empty for the current module.
    pub rust_type_root: &'a str,
    /// Generated method enum name.
    pub method_enum: &'a str,
    /// Public request field inventory function name.
    pub fields_function: &'a str,
    /// Public request mutation function name.
    pub mutate_function: &'a str,
    /// Source method names whose request closures are admitted for mutation.
    pub request_roots: &'a [&'a str],
}
const MAX_DEPTH: usize = 4;

struct Node<'a> {
    descriptor: &'a DescriptorProto,
    rust: String,
}

#[derive(Clone)]
struct Step {
    parent: String,
    field: FieldDescriptorProto,
    element: bool,
}

struct Field {
    path: String,
    declaration: String,
    kind: TokenStream,
    optional: bool,
    refusal: Option<String>,
    steps: Vec<Step>,
}

struct Schema<'a> {
    nodes: BTreeMap<String, Node<'a>>,
    enums: BTreeMap<String, Vec<i32>>,
}

/// Generate descriptor field inventories and typed request mutation setters.
pub fn generate(bytes: &[u8], config: &Config<'_>) -> Result<String, String> {
    let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
    let schema = Schema::new(&descriptors, config.rust_type_root);
    let (package, name) =
        config.service.rsplit_once('.').ok_or("service must be fully qualified")?;
    let service = descriptors
        .file
        .iter()
        .filter(|file| file.package.as_deref() == Some(package))
        .flat_map(|file| &file.service)
        .find(|service| service.name.as_deref() == Some(name))
        .ok_or_else(|| format!("missing service descriptor: {}", config.service))?;
    let method_enum: syn::Ident =
        syn::parse_str(config.method_enum).map_err(|error| error.to_string())?;
    let fields_function: syn::Ident =
        syn::parse_str(config.fields_function).map_err(|error| error.to_string())?;
    let mutate_function: syn::Ident =
        syn::parse_str(config.mutate_function).map_err(|error| error.to_string())?;
    let mut roots = Vec::new();
    for method in config.request_roots {
        let proto = service
            .method
            .iter()
            .find(|rpc| rpc.name.as_deref() == Some(method))
            .and_then(|rpc| rpc.input_type.as_deref())
            .ok_or_else(|| format!("missing request descriptor: {}.{method}", config.service))?
            .trim_start_matches('.')
            .to_owned();
        let mut fields = Vec::new();
        schema.walk(&proto, &proto, &[], &[], &mut fields)?;
        roots.push((*method, proto, fields));
    }
    let mut inventories = Vec::new();
    let mut mutations = Vec::new();
    let mut setters = Vec::new();
    for (method, proto, fields) in &roots {
        let variant: syn::Ident = syn::parse_str(method).map_err(|error| error.to_string())?;
        let rust: syn::Type =
            syn::parse_str(&schema.node(proto)?.rust).map_err(|error| error.to_string())?;
        let factory = format_ident!("mutate_{}", method.to_snake_case());
        let inventory = fields.iter().map(|field| {
            let path = &field.path;
            let declaration = &field.declaration;
            let kind = &field.kind;
            let optional = field.optional;
            let refusal = field.refusal.as_ref().map_or_else(|| quote! { None }, |reason| quote! { Some(#reason.into()) });
            quote! { WireField { path: #path.into(), declaration: #declaration.into(), kind: #kind, optional: #optional, refusal: #refusal } }
        });
        inventories.push(quote! { #method_enum::#variant => vec![#(#inventory),*] });
        mutations.push(quote! {
            #method_enum::#variant => {
                let mut message = <#rust as ::prost::Message>::decode(bytes).map_err(|_| GenerationError::InvalidEncoding)?;
                #factory(&mut message, path, mutation)?;
                Ok(::prost::Message::encode_to_vec(&message))
            }
        });
        let arms = fields
            .iter()
            .filter(|field| field.refusal.is_none())
            .map(|field| {
                let path = &field.path;
                let setter = schema.setter(field)?;
                Ok(quote! { #path => { let node0 = message; #setter Ok(()) } })
            })
            .collect::<Result<Vec<_>, String>>()?;
        setters.push(quote! {
            fn #factory(message: &mut #rust, path: &str, mutation: &WireMutation) -> Result<(), GenerationError> {
                match path { #(#arms,)* _ => Err(GenerationError::UnsupportedType) }
            }
        });
    }
    if roots.len() < service.method.len() {
        inventories.push(quote! {
            _ => vec![WireField { path: method.name().into(), declaration: method.name().into(), kind: WireKind::Message, optional: false, refusal: Some("request root is outside the live probe surface".into()) }]
        });
        mutations.push(quote! { _ => Err(GenerationError::UnsupportedType) });
    }
    let collection = roots.iter().any(|(_, _, fields)| fields.iter().any(|field| {
        field.steps.last().is_some_and(|step| step.field.label == Some(Label::Repeated as i32))
    })).then(|| quote! {
        fn wire_collection_len(mutation: &WireMutation) -> Result<usize, GenerationError> {
            let WireMutation::LimitPlusOne { limit } = mutation else { return Err(GenerationError::InapplicableMutation) };
            let length = limit.checked_add(1).ok_or(GenerationError::SizeLimit)?;
            if length > 4096 { return Err(GenerationError::SizeLimit) }
            usize::try_from(length).map_err(|_| GenerationError::SizeLimit)
        }
    });
    let output = quote! {
        use ::umpire_catalog::{WireField, WireKind, WireMutation, WireValue, GenerationError};
        /// Raw request fields, including descriptor fields ignored by semantic catalogs.
        pub fn #fields_function(method: #method_enum) -> Vec<WireField> { match method { #(#inventories),* } }
        /// Apply one typed descriptor mutation to a decoded request.
        pub fn #mutate_function(method: #method_enum, bytes: &[u8], path: &str, mutation: &WireMutation) -> Result<Vec<u8>, GenerationError> {
            match method { #(#mutations),* }
        }
        #(#setters)*
        #collection
    };
    let file = syn::parse2(output).map_err(|error| error.to_string())?;
    Ok(prettyplease::unparse(&file))
}

impl<'a> Schema<'a> {
    fn new(descriptors: &'a FileDescriptorSet, rust_type_root: &str) -> Self {
        let mut schema = Self { nodes: BTreeMap::new(), enums: BTreeMap::new() };
        for file in &descriptors.file {
            let package = file.package.as_deref().unwrap_or_default();
            let rust = if package == "google.protobuf" {
                "::umpire_runtime::google::protobuf".into()
            } else if rust_type_root.is_empty() {
                prost_build::Module::from_protobuf_package_name(package).to_string()
            } else {
                format!(
                    "{rust_type_root}::{}",
                    prost_build::Module::from_protobuf_package_name(package)
                )
            };
            schema.collect(package, &rust, &file.message_type);
            for enumeration in &file.enum_type {
                schema.enums.insert(
                    format!("{package}.{}", enumeration.name.as_deref().unwrap_or_default()),
                    enumeration.value.iter().filter_map(|value| value.number).collect(),
                );
            }
        }
        schema
    }

    fn collect(&mut self, proto: &str, rust: &str, messages: &'a [DescriptorProto]) {
        for descriptor in messages {
            let name = descriptor.name.as_deref().unwrap_or_default();
            let proto = format!("{proto}.{name}");
            self.nodes.insert(
                proto.clone(),
                Node { descriptor, rust: format!("{rust}::{}", super::services::type_name(name)) },
            );
            for enumeration in &descriptor.enum_type {
                self.enums.insert(
                    format!("{proto}.{}", enumeration.name.as_deref().unwrap_or_default()),
                    enumeration.value.iter().filter_map(|value| value.number).collect(),
                );
            }
            self.collect(
                &proto,
                &format!("{rust}::{}", prost_build::Module::from_protobuf_package_name(name)),
                &descriptor.nested_type,
            );
        }
    }

    fn node(&self, name: &str) -> Result<&Node<'a>, String> {
        self.nodes
            .get(name.trim_start_matches('.'))
            .ok_or_else(|| format!("missing descriptor message: {name}"))
    }

    fn walk(
        &self,
        proto: &str,
        path: &str,
        steps: &[Step],
        parents: &[String],
        fields: &mut Vec<Field>,
    ) -> Result<(), String> {
        let node = self.node(proto)?;
        let refusal = if parents.iter().any(|parent| parent == proto) {
            Some("recursive descriptor edge; deeper fields were not expanded")
        } else if parents.len() >= MAX_DEPTH {
            Some("descriptor depth limit reached; deeper fields were not expanded")
        } else {
            None
        };
        let mut parents = parents.to_vec();
        parents.push(proto.into());
        for field in &node.descriptor.field {
            let name = field.name.as_deref().ok_or("unnamed descriptor field")?;
            let path = format!("{path}.{name}");
            let mut steps = steps.to_vec();
            steps.push(Step { parent: proto.into(), field: field.clone(), element: false });
            fields.push(Field {
                path: path.clone(),
                declaration: format!("{proto}.{name}"),
                kind: self.kind(field)?,
                optional: self.optional(proto, field)?,
                refusal: refusal.map(str::to_owned),
                steps: steps.clone(),
            });
            if refusal.is_some() {
                continue;
            }
            if field.r#type == Some(Type::Message as i32) {
                let child = field
                    .type_name
                    .as_deref()
                    .ok_or("message field without target")?
                    .trim_start_matches('.');
                self.walk(child, &path, &steps, &parents, fields)?;
            } else if field.label == Some(Label::Repeated as i32) {
                let mut scalar = field.clone();
                scalar.label = Some(Label::Optional as i32);
                steps.push(Step { parent: proto.into(), field: scalar.clone(), element: true });
                fields.push(Field {
                    path: format!("{path}.value"),
                    declaration: format!("{proto}.{name}"),
                    kind: self.kind(&scalar)?,
                    optional: false,
                    refusal: None,
                    steps,
                });
            }
        }
        Ok(())
    }

    fn optional(&self, parent: &str, field: &FieldDescriptorProto) -> Result<bool, String> {
        Ok(!self.is_map_entry(parent)?
            && field.label != Some(Label::Repeated as i32)
            && (field.r#type == Some(Type::Message as i32)
                || field.oneof_index.is_some()
                || field.proto3_optional == Some(true)))
    }

    fn is_map_entry(&self, name: &str) -> Result<bool, String> {
        Ok(self
            .node(name)?
            .descriptor
            .options
            .as_ref()
            .is_some_and(|options| options.map_entry == Some(true)))
    }

    fn map(&self, field: &FieldDescriptorProto) -> Result<bool, String> {
        if field.r#type == Some(Type::Message as i32) {
            self.is_map_entry(field.type_name.as_deref().ok_or("missing message target")?)
        } else {
            Ok(false)
        }
    }

    fn kind(&self, field: &FieldDescriptorProto) -> Result<TokenStream, String> {
        if self.map(field)? {
            return Ok(quote! { WireKind::Map });
        }
        if field.label == Some(Label::Repeated as i32) {
            return Ok(quote! { WireKind::Repeated });
        }
        let kind = match Type::try_from(field.r#type.unwrap_or_default())
            .map_err(|_| "invalid field type")?
        {
            Type::String => quote! { WireKind::String },
            Type::Bytes => quote! { WireKind::Bytes },
            Type::Bool => quote! { WireKind::Bool },
            Type::Double | Type::Float => quote! { WireKind::Float },
            Type::Uint32 | Type::Uint64 | Type::Fixed32 | Type::Fixed64 => {
                quote! { WireKind::Unsigned }
            }
            Type::Int32
            | Type::Int64
            | Type::Sint32
            | Type::Sint64
            | Type::Sfixed32
            | Type::Sfixed64 => quote! { WireKind::Signed },
            Type::Message => quote! { WireKind::Message },
            Type::Enum => {
                let name = field
                    .type_name
                    .as_deref()
                    .ok_or("missing enum target")?
                    .trim_start_matches('.');
                let values = self
                    .enums
                    .get(name)
                    .ok_or_else(|| format!("missing enum descriptor: {name}"))?;
                quote! { WireKind::Enum { defined: vec![#(#values),*] } }
            }
            Type::Group => return Err("protobuf groups are unsupported".into()),
        };
        Ok(kind)
    }

    fn member(&self, step: &Step) -> Result<syn::Member, String> {
        if self.is_map_entry(&step.parent)? {
            let index = usize::from(step.field.number != Some(1));
            return Ok(syn::Member::Unnamed(syn::Index::from(index)));
        }
        let name = prost_build::Module::from_protobuf_package_name(
            step.field.name.as_deref().ok_or("unnamed field")?,
        )
        .to_string();
        syn::parse_str(&name).map(syn::Member::Named).map_err(|error| error.to_string())
    }

    fn oneof(&self, step: &Step) -> Result<Option<(syn::Ident, syn::Path, syn::Ident)>, String> {
        if step.field.proto3_optional == Some(true) {
            return Ok(None);
        }
        let Some(index) = step.field.oneof_index else {
            return Ok(None);
        };
        let node = self.node(&step.parent)?;
        let name = node
            .descriptor
            .oneof_decl
            .get(usize::try_from(index).map_err(|_| "invalid oneof index")?)
            .and_then(|oneof| oneof.name.as_deref())
            .ok_or("missing oneof name")?;
        let (module, _) = node.rust.rsplit_once("::").ok_or("invalid message Rust path")?;
        let owner = prost_build::Module::from_protobuf_package_name(
            node.descriptor.name.as_deref().ok_or("unnamed message")?,
        );
        let member = prost_build::Module::from_protobuf_package_name(name).to_string();
        let ty = format!("{module}::{owner}::{}", super::services::type_name(name));
        let variant =
            super::services::type_name(step.field.name.as_deref().ok_or("unnamed oneof field")?);
        Ok(Some((
            syn::parse_str(&member).map_err(|error| error.to_string())?,
            syn::parse_str(&ty).map_err(|error| error.to_string())?,
            syn::parse_str(&variant).map_err(|error| error.to_string())?,
        )))
    }

    fn setter(&self, field: &Field) -> Result<TokenStream, String> {
        let mut cleanups = Vec::new();
        let mut output = TokenStream::new();
        for (index, step) in field.steps.iter().enumerate() {
            let parent = format_ident!("node{index}");
            let child = format_ident!("node{}", index + 1);
            if index + 1 == field.steps.len() {
                output.extend(self.leaf(step, &parent, &field.kind)?);
                break;
            }
            if let Some((member, ty, variant)) = self.oneof(step)? {
                output.extend(absent_guard(
                    &quote! { !matches!(#parent.#member, Some(#ty::#variant(_))) },
                    &cleanups,
                ));
                output.extend(quote! {
                    if !matches!(#parent.#member, Some(#ty::#variant(_))) {
                        #parent.#member = Some(#ty::#variant(Default::default()));
                    }
                    let Some(#ty::#variant(#child)) = #parent.#member.as_mut() else { return Err(GenerationError::UnsupportedType) };
                });
            } else {
                let member = self.member(step)?;
                if self.map(&step.field)? {
                    output.extend(absent_guard(&quote! { #parent.#member.is_empty() }, &cleanups));
                    let key = format_ident!("key{index}");
                    let entry = format_ident!("entry{index}");
                    output.extend(quote! {
                        let #key = #parent.#member.keys().next().cloned().unwrap_or_default();
                        let mut #entry = (#key.clone(), #parent.#member.remove(&#key).unwrap_or_default());
                        let #child = &mut #entry;
                    });
                    cleanups.push(quote! { #parent.#member.insert(#entry.0, #entry.1); });
                } else if step.field.label == Some(Label::Repeated as i32) {
                    output.extend(absent_guard(&quote! { #parent.#member.is_empty() }, &cleanups));
                    output.extend(quote! {
                        if #parent.#member.is_empty() { #parent.#member.push(Default::default()); }
                        let #child = #parent.#member.first_mut().ok_or(GenerationError::UnsupportedType)?;
                    });
                } else if self.optional(&step.parent, &step.field)? {
                    output.extend(absent_guard(&quote! { #parent.#member.is_none() }, &cleanups));
                    output.extend(quote! { let #child = #parent.#member.get_or_insert_with(Default::default); });
                } else {
                    output.extend(quote! { let #child = &mut #parent.#member; });
                }
            }
        }
        output.extend(cleanups.into_iter().rev().flatten());
        Ok(output)
    }

    fn leaf(
        &self,
        step: &Step,
        parent: &syn::Ident,
        kind: &TokenStream,
    ) -> Result<TokenStream, String> {
        if step.element {
            return Ok(quote! { WireValue::mutate(#parent, &#kind, mutation)?; });
        }
        if let Some((member, ty, variant)) = self.oneof(step)? {
            let change = if step.field.r#type == Some(Type::Message as i32) {
                quote! {
                    if !matches!(mutation, WireMutation::Empty) { return Err(GenerationError::InapplicableMutation) }
                    #parent.#member = Some(#ty::#variant(Default::default()));
                }
            } else {
                quote! {
                    if !matches!(#parent.#member, Some(#ty::#variant(_))) { #parent.#member = Some(#ty::#variant(Default::default())); }
                    let Some(#ty::#variant(value)) = #parent.#member.as_mut() else { return Err(GenerationError::UnsupportedType) };
                    WireValue::mutate(value, &#kind, mutation)?;
                }
            };
            return Ok(quote! {
                if matches!(mutation, WireMutation::Absent) {
                    if matches!(#parent.#member, Some(#ty::#variant(_))) { #parent.#member = None; }
                } else { #change }
            });
        }
        let member = self.member(step)?;
        let target = quote! { #parent.#member };
        if self.map(&step.field)? {
            return self.map_leaf(&step.field, &target);
        }
        Ok(if step.field.label == Some(Label::Repeated as i32) {
            quote! {
                match mutation {
                    WireMutation::Absent | WireMutation::Empty => #target.clear(),
                    WireMutation::LimitPlusOne { .. } => {
                        let length = wire_collection_len(mutation)?;
                        #target.clear(); #target.resize_with(length, Default::default);
                    }
                    _ => return Err(GenerationError::InapplicableMutation),
                }
            }
        } else if step.field.r#type == Some(Type::Message as i32) {
            if self.optional(&step.parent, &step.field)? {
                quote! {
                    match mutation {
                        WireMutation::Absent => #target = None,
                        WireMutation::Empty => #target = Some(Default::default()),
                        _ => return Err(GenerationError::InapplicableMutation),
                    }
                }
            } else {
                quote! {
                    match mutation {
                        WireMutation::Absent | WireMutation::Empty => #target = Default::default(),
                        _ => return Err(GenerationError::InapplicableMutation),
                    }
                }
            }
        } else if self.optional(&step.parent, &step.field)? {
            quote! {
                if matches!(mutation, WireMutation::Absent) { #target = None; }
                else { WireValue::mutate(#target.get_or_insert_with(Default::default), &#kind, mutation)?; }
            }
        } else {
            quote! { WireValue::mutate(&mut #target, &#kind, mutation)?; }
        })
    }

    fn map_leaf(
        &self,
        field: &FieldDescriptorProto,
        target: &TokenStream,
    ) -> Result<TokenStream, String> {
        let entry = self.node(field.type_name.as_deref().ok_or("missing map entry")?)?;
        let key = entry
            .descriptor
            .field
            .iter()
            .find(|field| field.number == Some(1))
            .ok_or("missing map key")?;
        let (key, guard) = match Type::try_from(key.r#type.unwrap_or_default())
            .map_err(|_| "invalid map key")?
        {
            Type::String => (quote! { index.to_string() }, quote! {}),
            Type::Bool => (
                quote! { index != 0 },
                quote! { if length > 2 { return Err(GenerationError::SizeLimit) } },
            ),
            _ => (quote! { index.try_into().map_err(|_| GenerationError::SizeLimit)? }, quote! {}),
        };
        Ok(quote! {
            match mutation {
                WireMutation::Absent | WireMutation::Empty => #target.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    #guard
                    #target.clear();
                    for index in 0..length { #target.insert(#key, Default::default()); }
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
        })
    }
}

fn absent_guard(condition: &TokenStream, cleanups: &[TokenStream]) -> TokenStream {
    let cleanups = cleanups.iter().rev();
    quote! {
        if matches!(mutation, WireMutation::Absent) && (#condition) {
            #(#cleanups)*
            return Ok(());
        }
    }
}
