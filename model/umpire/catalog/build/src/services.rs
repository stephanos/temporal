//! Descriptor-backed service method and annotation dispatch generation.

use std::collections::BTreeSet;

use proc_macro2::TokenStream;
use prost::Message;
use quote::quote;
use umpire_catalog_syntax::{
    descriptor::{Coverage, Disposition},
    services::Classification,
};
use umpire_runtime::google::protobuf::FileDescriptorSet;

/// Generated service identities and Rust surface names.
pub struct Config<'a> {
    /// Fully qualified protobuf service name.
    pub service: &'a str,
    /// Absolute Rust crate or module path for public type inventory.
    pub rust_type_root: &'a str,
    /// Rust root used inside exported dispatch macros.
    pub macro_type_root: &'a str,
    /// Generated method enum name.
    pub method_enum: &'a str,
    /// Generated method information struct name.
    pub method_info: &'a str,
    /// Generated ordered method inventory constant.
    pub methods_const: &'a str,
    /// Exported annotation dispatch macro name.
    pub dispatch_macro: &'a str,
    /// Exported action binding macro name.
    pub bind_macro: &'a str,
}

/// Generate the method inventory and exhaustive action binding macros.
pub fn generate(
    bytes: &[u8],
    coverage: &Coverage,
    classifications: &[Classification],
    config: &Config<'_>,
) -> Result<String, String> {
    let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
    let (package, name) =
        config.service.rsplit_once('.').ok_or("service must be fully qualified")?;
    let service = descriptors
        .file
        .iter()
        .filter(|file| file.package.as_deref() == Some(package))
        .flat_map(|file| &file.service)
        .find(|service| service.name.as_deref() == Some(name))
        .ok_or_else(|| format!("missing service descriptor: {}", config.service))?;
    let service_name = config.service;
    let annotated: BTreeSet<_> =
        coverage.messages.iter().map(|message| message.name.trim_start_matches('.')).collect();
    let out_of_scope: BTreeSet<_> = classifications
        .iter()
        .flat_map(|classification| &classification.coverage.methods)
        .filter(|method| matches!(method.disposition, Disposition::OutOfScope(_)))
        .map(|method| method.name.as_str())
        .collect();
    // A mirrored message means some feature validates this RPC, so it must claim it.
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        if out_of_scope.contains(name)
            && [&method.input_type, &method.output_type]
                .into_iter()
                .flatten()
                .any(|message| annotated.contains(message.trim_start_matches('.')))
        {
            return Err(format!(
                "annotated RPC is out of scope: {service_name}.{name} has mirrored messages but \
                 no Action or Observed classification"
            ));
        }
    }
    let method_enum = identifier(config.method_enum)?;
    let method_info = identifier(config.method_info)?;
    let methods_const = identifier(config.methods_const)?;
    let dispatch_macro = identifier(config.dispatch_macro)?;
    let bindings = bindings(classifications, config)?;
    let mut variants = Vec::new();
    let mut names = Vec::new();
    let mut infos = Vec::new();
    let mut dispatch = Vec::new();
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        let variant = identifier(name)?;
        let doc = format!("`{service_name}.{name}`.");
        variants.push(quote! { #[doc = #doc] #variant });
        names.push((variant.clone(), name));
        let request = public_type(
            method.input_type.as_deref().ok_or("missing request type")?,
            config.rust_type_root,
        );
        let response = public_type(
            method.output_type.as_deref().ok_or("missing response type")?,
            config.rust_type_root,
        );
        infos.push(quote! {
            #method_info { method: #method_enum::#variant, request_type: #request, response_type: #response }
        });
        for (response, ty) in [(false, &method.input_type), (true, &method.output_type)] {
            let ty = ty.as_deref().ok_or("missing method message type")?;
            if annotated.contains(ty.trim_start_matches('.')) {
                let path: TokenStream = public_type(ty, config.macro_type_root)
                    .parse()
                    .map_err(|error: proc_macro2::LexError| error.to_string())?;
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

fn identifier(name: &str) -> Result<syn::Ident, String> {
    syn::parse_str(name).map_err(|error| error.to_string())
}

fn public_type(proto: &str, root: &str) -> String {
    let proto = proto.trim_start_matches('.');
    let (package, name) = proto.rsplit_once('.').unwrap_or(("", proto));
    let package = prost_build::Module::from_protobuf_package_name(package);
    let name = type_name(name);
    if package.is_empty() {
        format!("{root}::{name}")
    } else {
        format!("{root}::{package}::{name}")
    }
}

pub(super) fn type_name(proto: &str) -> String {
    umpire_catalog_syntax::naming::upper_camel(proto)
}

fn bindings(
    classifications: &[Classification],
    config: &Config<'_>,
) -> Result<TokenStream, String> {
    let bind_macro = identifier(config.bind_macro)?;
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
            let rpc = format!("{}.{}", config.service, binding.method);
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
