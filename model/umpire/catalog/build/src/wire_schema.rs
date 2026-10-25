//! Generate raw descriptor inventories and typed protobuf mutation setters.

use crate::System;
use heck::ToSnakeCase;
use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use umpire_catalog_syntax::{
    descriptor::well_known_primitive,
    proto::{self, Graph, Kind, Scalar},
};

const MAX_DEPTH: usize = 4;

#[derive(Clone)]
struct Step {
    parent: String,
    field: proto::Field,
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

/// Generate descriptor field inventories and typed request mutation setters for
/// `system.request_roots`; `graph` names generated types under [`System::local_root`].
pub fn generate(graph: &Graph, system: &System<'_>) -> Result<String, String> {
    let service = graph.service(system.service)?;
    let method_enum: syn::Ident =
        syn::parse_str(&system.method_enum()).map_err(|error| error.to_string())?;
    let fields_function = format_ident!("{}_request_fields", system.prefix);
    let mutate_function = format_ident!("mutate_{}_request", system.prefix);
    let mut roots = Vec::new();
    for method in system.request_roots {
        let proto = graph
            .method(system.service, method)
            .map_err(|_| format!("missing request descriptor: {}.{method}", system.service))?
            .input
            .clone();
        let mut fields = Vec::new();
        walk(graph, &proto, &proto, &[], &[], &mut fields)?;
        roots.push((*method, proto, fields));
    }
    let mut inventories = Vec::new();
    let mut mutations = Vec::new();
    let mut setters = Vec::new();
    for (method, proto, fields) in &roots {
        let variant: syn::Ident = syn::parse_str(method).map_err(|error| error.to_string())?;
        let rust: syn::Type =
            syn::parse_str(&graph.message(proto)?.rust_type).map_err(|error| error.to_string())?;
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
                let setter = setter(graph, field)?;
                Ok(quote! { #path => { let node0 = message; #setter Ok(()) } })
            })
            .collect::<Result<Vec<_>, String>>()?;
        setters.push(quote! {
            fn #factory(message: &mut #rust, path: &str, mutation: &WireMutation) -> Result<(), GenerationError> {
                match path { #(#arms,)* _ => Err(GenerationError::UnsupportedType) }
            }
        });
    }
    if roots.len() < service.len() {
        inventories.push(quote! {
            _ => vec![WireField { path: method.name().into(), declaration: method.name().into(), kind: WireKind::Message, optional: false, refusal: Some("request root is outside the live probe surface".into()) }]
        });
        mutations.push(quote! { _ => Err(GenerationError::UnsupportedType) });
    }
    let collection = roots.iter().any(|(_, _, fields)| fields.iter().any(|field| {
        field.steps.last().is_some_and(|step| step.field.repeated)
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

fn walk(
    graph: &Graph,
    proto: &str,
    path: &str,
    steps: &[Step],
    parents: &[String],
    fields: &mut Vec<Field>,
) -> Result<(), String> {
    let message = graph.message(proto)?;
    let refusal = if parents.iter().any(|parent| parent == proto) {
        Some("recursive descriptor edge; deeper fields were not expanded")
    } else if parents.len() >= MAX_DEPTH {
        Some("descriptor depth limit reached; deeper fields were not expanded")
    } else {
        None
    };
    let mut parents = parents.to_vec();
    parents.push(proto.into());
    for field in &message.fields {
        let name = &field.name;
        let path = format!("{path}.{name}");
        let mut steps = steps.to_vec();
        steps.push(Step { parent: proto.into(), field: field.clone(), element: false });
        fields.push(Field {
            path: path.clone(),
            declaration: format!("{proto}.{name}"),
            kind: kind(graph, field, false)?,
            optional: field.optional,
            refusal: refusal.map(str::to_owned),
            steps: steps.clone(),
        });
        if refusal.is_some() {
            continue;
        }
        if let Kind::Message { target } = &field.kind {
            walk(graph, target, &path, &steps, &parents, fields)?;
        } else if field.repeated {
            steps.push(Step { parent: proto.into(), field: field.clone(), element: true });
            fields.push(Field {
                path: format!("{path}.value"),
                declaration: format!("{proto}.{name}"),
                kind: kind(graph, field, true)?,
                optional: false,
                refusal: None,
                steps,
            });
        }
    }
    Ok(())
}

fn kind(graph: &Graph, field: &proto::Field, element: bool) -> Result<TokenStream, String> {
    if !element && field.map {
        return Ok(quote! { WireKind::Map });
    }
    if !element && field.repeated {
        return Ok(quote! { WireKind::Repeated });
    }
    Ok(match &field.kind {
        Kind::Scalar(Scalar::String) => quote! { WireKind::String },
        Kind::Scalar(Scalar::Bytes) => quote! { WireKind::Bytes },
        Kind::Scalar(Scalar::Bool) => quote! { WireKind::Bool },
        Kind::Scalar(Scalar::Float) => quote! { WireKind::Float },
        Kind::Scalar(Scalar::Unsigned) => quote! { WireKind::Unsigned },
        Kind::Scalar(Scalar::Signed) => quote! { WireKind::Signed },
        Kind::Message { .. } => quote! { WireKind::Message },
        Kind::Enum { target } => {
            let values = graph
                .enums
                .get(target)
                .ok_or_else(|| format!("missing enum descriptor: {target}"))?
                .values
                .iter()
                .map(|value| value.number);
            quote! { WireKind::Enum { defined: vec![#(#values),*] } }
        }
    })
}

fn member(step: &Step) -> Result<syn::Member, String> {
    syn::parse_str(&step.field.member).map_err(|error| error.to_string())
}

fn oneof(step: &Step) -> Result<Option<(syn::Member, syn::Path)>, String> {
    step.field
        .oneof
        .as_ref()
        .map(|oneof| {
            Ok((
                syn::parse_str(&oneof.member).map_err(|error| error.to_string())?,
                syn::parse_str(&oneof.variant).map_err(|error| error.to_string())?,
            ))
        })
        .transpose()
}

fn setter(graph: &Graph, field: &Field) -> Result<TokenStream, String> {
    let mut cleanups = Vec::new();
    let mut output = TokenStream::new();
    for (index, step) in field.steps.iter().enumerate() {
        let parent = format_ident!("node{index}");
        let child = format_ident!("node{}", index + 1);
        if index + 1 == field.steps.len() {
            output.extend(leaf(graph, step, &parent, &field.kind)?);
            break;
        }
        if let Some((member, variant)) = oneof(step)? {
            output.extend(absent_guard(
                &quote! { !matches!(#parent.#member, Some(#variant(_))) },
                &cleanups,
            ));
            output.extend(quote! {
                if !matches!(#parent.#member, Some(#variant(_))) {
                    #parent.#member = Some(#variant(Default::default()));
                }
                let Some(#variant(#child)) = #parent.#member.as_mut() else { return Err(GenerationError::UnsupportedType) };
            });
        } else {
            let member = member(step)?;
            if step.field.map {
                output.extend(absent_guard(&quote! { #parent.#member.is_empty() }, &cleanups));
                let key = format_ident!("key{index}");
                let entry = format_ident!("entry{index}");
                output.extend(quote! {
                    let #key = #parent.#member.keys().next().cloned().unwrap_or_default();
                    let mut #entry = (#key.clone(), #parent.#member.remove(&#key).unwrap_or_default());
                    let #child = &mut #entry;
                });
                cleanups.push(quote! { #parent.#member.insert(#entry.0, #entry.1); });
            } else if step.field.repeated {
                output.extend(absent_guard(&quote! { #parent.#member.is_empty() }, &cleanups));
                output.extend(quote! {
                    if #parent.#member.is_empty() { #parent.#member.push(Default::default()); }
                    let #child = #parent.#member.first_mut().ok_or(GenerationError::UnsupportedType)?;
                });
            } else if step.field.optional {
                output.extend(absent_guard(&quote! { #parent.#member.is_none() }, &cleanups));
                output.extend(
                    quote! { let #child = #parent.#member.get_or_insert_with(Default::default); },
                );
            } else {
                output.extend(quote! { let #child = &mut #parent.#member; });
            }
        }
    }
    output.extend(cleanups.into_iter().rev().flatten());
    Ok(output)
}

fn leaf(
    graph: &Graph,
    step: &Step,
    parent: &syn::Ident,
    kind: &TokenStream,
) -> Result<TokenStream, String> {
    if step.element {
        return Ok(quote! { WireValue::mutate(#parent, &#kind, mutation)?; });
    }
    let message = matches!(step.field.kind, Kind::Message { .. });
    if let Some((member, variant)) = oneof(step)? {
        let change = if message {
            quote! {
                if !matches!(mutation, WireMutation::Empty) { return Err(GenerationError::InapplicableMutation) }
                #parent.#member = Some(#variant(Default::default()));
            }
        } else {
            quote! {
                if !matches!(#parent.#member, Some(#variant(_))) { #parent.#member = Some(#variant(Default::default())); }
                let Some(#variant(value)) = #parent.#member.as_mut() else { return Err(GenerationError::UnsupportedType) };
                WireValue::mutate(value, &#kind, mutation)?;
            }
        };
        return Ok(quote! {
            if matches!(mutation, WireMutation::Absent) {
                if matches!(#parent.#member, Some(#variant(_))) { #parent.#member = None; }
            } else { #change }
        });
    }
    let member = member(step)?;
    let target = if well_known_primitive(&step.parent).is_some() {
        quote! { *#parent }
    } else {
        quote! { #parent.#member }
    };
    if step.field.map {
        return map_leaf(graph, &step.field, &target);
    }
    Ok(if step.field.repeated {
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
    } else if message {
        if step.field.optional {
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
    } else if step.field.optional {
        quote! {
            if matches!(mutation, WireMutation::Absent) { #target = None; }
            else { WireValue::mutate(#target.get_or_insert_with(Default::default), &#kind, mutation)?; }
        }
    } else {
        quote! { WireValue::mutate(&mut #target, &#kind, mutation)?; }
    })
}

fn map_leaf(
    graph: &Graph,
    field: &proto::Field,
    target: &TokenStream,
) -> Result<TokenStream, String> {
    let Kind::Message { target: entry } = &field.kind else {
        return Err("missing map entry".into());
    };
    let key = graph
        .message(entry)?
        .fields
        .iter()
        .find(|field| field.number == 1)
        .ok_or("missing map key")?;
    let (key, guard) = match key.kind {
        Kind::Scalar(Scalar::String) => (quote! { index.to_string() }, quote! {}),
        Kind::Scalar(Scalar::Bool) => (
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

fn absent_guard(condition: &TokenStream, cleanups: &[TokenStream]) -> TokenStream {
    let cleanups = cleanups.iter().rev();
    quote! {
        if matches!(mutation, WireMutation::Absent) && (#condition) {
            #(#cleanups)*
            return Ok(());
        }
    }
}
