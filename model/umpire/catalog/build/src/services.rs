//! Descriptor-backed service method and annotation dispatch generation.

use std::collections::BTreeSet;

use proc_macro2::TokenStream;
use quote::quote;
use umpire_catalog_syntax::{
    descriptor::{Coverage, Disposition},
    proto::Graph,
    services::Classification,
};

use crate::System;

/// Generate the method inventory and exhaustive action binding macros.
///
/// `graph` names generated types under `system.rust_root`.
pub fn generate(
    graph: &Graph,
    coverage: &Coverage,
    classifications: &[Classification],
    system: &System<'_>,
) -> Result<String, String> {
    let service = graph.service(system.service)?;
    let service_name = system.service;
    let annotated: BTreeSet<_> =
        coverage.messages.iter().map(|message| message.name.trim_start_matches('.')).collect();
    let out_of_scope: BTreeSet<_> = classifications
        .iter()
        .flat_map(|classification| &classification.coverage.methods)
        .filter(|method| matches!(method.disposition, Disposition::OutOfScope(_)))
        .map(|method| method.name.as_str())
        .collect();
    // A mirrored message means some feature validates this RPC, so it must claim it.
    for method in service {
        let name = method.name.as_str();
        if out_of_scope.contains(name)
            && [&method.input, &method.output]
                .into_iter()
                .any(|message| annotated.contains(message.as_str()))
        {
            return Err(format!(
                "annotated RPC is out of scope: {service_name}.{name} has mirrored messages but \
                 no Action or Observed classification"
            ));
        }
    }
    let method_enum = identifier(&system.method_enum())?;
    let method_info = identifier(&format!("{method_enum}Info"))?;
    let methods_const = identifier(&system.methods_const())?;
    let dispatch_macro = identifier(&format!("dispatch_{}_annotations", system.prefix))?;
    let requests = request_dispatch(graph, &annotated, system)?;
    let bindings = bindings(classifications, system)?;
    let macro_root = system.macro_root();
    let mut variants = Vec::new();
    let mut names = Vec::new();
    let mut infos = Vec::new();
    let mut dispatch = Vec::new();
    for method in service {
        let name = method.name.as_str();
        let variant = identifier(name)?;
        let doc = format!(" `{service_name}.{name}`.");
        variants.push(quote! { #[doc = #doc] #variant });
        names.push((variant.clone(), name));
        let request = &graph.message(&method.input)?.rust_type;
        let response = &graph.message(&method.output)?.rust_type;
        infos.push(quote! {
            #method_info { method: #method_enum::#variant, request_type: #request, response_type: #response }
        });
        for (response, ty) in [(false, request), (true, response)] {
            let proto = if response { &method.output } else { &method.input };
            if annotated.contains(proto.as_str()) {
                let path = system.rebase(ty, &macro_root).unwrap_or_else(|| ty.clone());
                let path: TokenStream =
                    path.parse().map_err(|error: proc_macro2::LexError| error.to_string())?;
                dispatch.push(quote! {
                    ($crate::#method_enum::#variant, #response) => { type $ty = #path; $body }
                });
            }
        }
    }
    let name_arms = names.iter().map(|(variant, name)| quote! { Self::#variant => #name });
    let parse_arms = names.iter().map(|(variant, name)| quote! { #name => Some(Self::#variant) });
    let output = quote! {
        /// Every RPC in descriptor source order.
        #[derive(Clone, Copy, Debug, Eq, PartialEq)]
        pub enum #method_enum { #(#variants),* }
        impl #method_enum {
            /// Original protobuf method name.
            pub const fn name(self) -> &'static str { match self { #(#name_arms),* } }
            /// Parse an original protobuf method name.
            pub fn parse(name: &str) -> Option<Self> { match name { #(#parse_arms,)* _ => None } }
        }
        /// One upstream method and its generated Rust message types.
        #[derive(Clone, Copy, Debug, Eq, PartialEq)]
        pub struct #method_info {
            /// The method identity.
            pub method: #method_enum,
            /// Fully qualified Rust request type path.
            pub request_type: &'static str,
            /// Fully qualified Rust response type path.
            pub response_type: &'static str,
        }
        /// Source-ordered request and response types.
        pub const #methods_const: &[#method_info] = &[#(#infos),*];
        #bindings
        #requests
        /// Dispatch only declared annotation message types.
        #[macro_export]
        macro_rules! #dispatch_macro {
            ($method:expr, $response:expr, $ty:ident, $body:block, $fallback:expr) => {
                match ($method, $response) { #(#dispatch,)* _ => $fallback }
            };
        }
    };
    let file = syn::parse2(output).map_err(|error| error.to_string())?;
    Ok(prettyplease::unparse(&file))
}

fn request_dispatch(
    graph: &Graph,
    annotated: &BTreeSet<&str>,
    system: &System<'_>,
) -> Result<TokenStream, String> {
    let findings_macro = identifier(&format!("{}_request_findings", system.prefix))?;
    let request_macro = identifier(&format!("dispatch_{}_request", system.prefix))?;
    let method_enum = identifier(&system.method_enum())?;
    let macro_root = system.macro_root();
    let mut requests = Vec::new();
    let mut findings = Vec::new();
    for method in graph.service(system.service)? {
        let variant = identifier(&method.name)?;
        let request = &graph.message(&method.input)?.rust_type;
        if annotated.contains(method.input.as_str()) {
            let path = system.rebase(request, &macro_root).unwrap_or_else(|| request.clone());
            let path: TokenStream =
                path.parse().map_err(|error: proc_macro2::LexError| error.to_string())?;
            requests.push(quote! {
                $crate::#method_enum::#variant => {
                    let mut message = <#path as ::prost::Message>::decode($bytes)?;
                    $callback!(#variant, message, $($args)*)
                }
            });
            findings.push(quote! {
                $crate::#method_enum::#variant => {
                    let message = <#path as ::prost::Message>::decode($bytes)?;
                    let context = $context;
                    let mut findings = Vec::new();
                    ::umpire_catalog::Inspect::inspect(&message, context, false, &mut findings)?;
                    $supplement!(#variant, message, context, findings);
                    for finding in &mut findings { finding.fields.sort(); }
                    Ok(findings)
                }
            });
        }
    }
    Ok(quote! {
        /// Inspect one decoded request in the catalog owner's context, then apply its supplemental hook.
        #[macro_export]
        macro_rules! #findings_macro {
            ($method:expr, $bytes:expr, $context:expr, $supplement:ident) => {
                match $method { #(#findings,)* _ => Err("request is outside the service catalog".into()) }
            };
        }
        /// Decode a declared request and expand a typed callback in its consumer.
        #[macro_export]
        macro_rules! #request_macro {
            ($method:expr, $bytes:expr, $callback:ident, $($args:tt)*) => {
                match $method { #(#requests,)* _ => Err("request is outside the service catalog".into()) }
            };
        }
    })
}

fn identifier(name: &str) -> Result<syn::Ident, String> {
    syn::parse_str(name).map_err(|error| error.to_string())
}

fn bindings(
    classifications: &[Classification],
    system: &System<'_>,
) -> Result<TokenStream, String> {
    let bind_macro = identifier(&format!("bind_{}_action", system.prefix))?;
    let mut inventory = Vec::new();
    let mut arms = Vec::new();
    let mut actions = BTreeSet::new();
    for classification in classifications {
        for (rpc, role, target) in &classification.bindings {
            inventory.push(quote! { (#rpc, #role, #target) });
        }
        for binding in &classification.actions {
            let pattern = &binding.pattern;
            if !actions.insert(binding.identity.clone()) {
                return Err(format!("duplicate Model action RPC binding: {}", quote! { #pattern }));
            }
            let rpc = format!("{}.{}", system.service, binding.method);
            arms.push(quote! { #pattern => Binding::Rpc(#rpc) });
        }
    }
    Ok(quote! {
        /// Source-reviewed RPC classifications.
        pub const RPC_BINDINGS: &[(&str, &str, &str)] = &[#(#inventory),*];
        /// Bind every Model action to its classified RPC or declared environment.
        #[macro_export]
        macro_rules! #bind_macro {
            ($action:expr, $environmental_pattern:pat => $environmental_expression:expr) => {
                match $action { #(#arms,)* $environmental_pattern => $environmental_expression }
            };
        }
    })
}
