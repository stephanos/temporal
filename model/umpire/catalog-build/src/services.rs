//! Descriptor-backed service method and annotation dispatch generation.

use std::{collections::BTreeSet, fmt::Write};

use prost::Message;
use quote::ToTokens;
use umpire_catalog_syntax::{descriptor::Coverage, services::Classification};
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
    let method_enum = config.method_enum;
    let method_info = config.method_info;
    let mut output = template(
        config,
        "/// Every RPC in descriptor source order.\n\
         #[derive(Clone, Copy, Debug, Eq, PartialEq)]\n\
         pub enum CatalogServiceMethod {\n",
    );
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        writeln!(output, "/// `{service_name}.{name}`.\n{name},")
            .map_err(|error| error.to_string())?;
    }
    output.push_str(&template(config,"}\nimpl CatalogServiceMethod {\n/// Original protobuf method name.\npub const fn name(self) -> &'static str { match self {\n"));
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        writeln!(output, "Self::{name} => {name:?},").map_err(|error| error.to_string())?;
    }
    output.push_str("}}\n/// Parse an original protobuf method name.\npub fn parse(name: &str) -> Option<Self> { match name {\n");
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        writeln!(output, "{name:?} => Some(Self::{name}),").map_err(|error| error.to_string())?;
    }
    output.push_str(&template(
        config,
        "_ => None,\n}}\n}\n\
        /// One upstream method and its generated Rust message types.\n\
        #[derive(Clone, Copy, Debug, Eq, PartialEq)]\n\
        pub struct CatalogServiceMethodInfo {\n\
        /// The method identity.\npub method: CatalogServiceMethod,\n\
        /// Fully qualified Rust request type path.\npub request_type: &'static str,\n\
        /// Fully qualified Rust response type path.\npub response_type: &'static str,\n}\n\
        /// Source-ordered request and response types.\n\
        pub const CATALOG_SERVICE_METHODS: &[CatalogServiceMethodInfo] = &[\n",
    ));
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        let request = public_type(
            method.input_type.as_deref().ok_or("missing request type")?,
            config.rust_type_root,
        );
        let response = public_type(
            method.output_type.as_deref().ok_or("missing response type")?,
            config.rust_type_root,
        );
        writeln!(output, "{method_info} {{ method: {method_enum}::{name}, request_type: {request:?}, response_type: {response:?} }},").map_err(|error| error.to_string())?;
    }
    output.push_str("];");
    emit_bindings(&mut output, classifications, config)?;
    output.push_str(&template(config,"\n/// Dispatch only declared annotation message types.\n#[macro_export]\nmacro_rules! dispatch_catalog_annotations {\n($method:expr, $response:expr, $ty:ident, $body:block, $fallback:expr) => { match ($method, $response) {\n"));
    let annotated: BTreeSet<_> =
        coverage.messages.iter().map(|message| message.name.trim_start_matches('.')).collect();
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed service method")?;
        for (response, ty) in [(false, &method.input_type), (true, &method.output_type)] {
            let ty = ty.as_deref().ok_or("missing method message type")?;
            if annotated.contains(ty.trim_start_matches('.')) {
                let path = public_type(ty, config.macro_type_root);
                writeln!(
                    output,
                    "($crate::{method_enum}::{name}, {response}) => {{ type $ty = {path}; $body }},"
                )
                .map_err(|error| error.to_string())?;
            }
        }
    }
    output.push_str("_ => $fallback,\n}};\n}\n");
    Ok(output)
}

fn public_type(proto: &str, root: &str) -> String {
    format!("{root}::{}", proto.trim_start_matches('.').replace('.', "::"))
}

fn emit_bindings(
    output: &mut String,
    classifications: &[Classification],
    config: &Config<'_>,
) -> Result<(), String> {
    output.push_str("\n/// Source-reviewed RPC classifications.\npub const RPC_BINDINGS: &[(&str, &str, &str)] = &[\n");
    for classification in classifications {
        for (rpc, role, target) in &classification.bindings {
            writeln!(output, "({rpc:?}, {role:?}, {target:?}),")
                .map_err(|error| error.to_string())?;
        }
    }
    output.push_str(&template(config,"];\n/// Bind every Model action to its classified RPC or declared environment.\n#[macro_export]\nmacro_rules! bind_catalog_action {\n($action:expr, $environmental_pattern:pat => $environmental_expression:expr) => { match $action {\n"));
    let mut actions = BTreeSet::new();
    for classification in classifications {
        for binding in &classification.actions {
            let pattern = binding.pattern.to_token_stream().to_string();
            if !actions.insert(binding.identity.clone()) {
                return Err(format!("duplicate Model action RPC binding: {pattern}"));
            }
            let rpc = format!("{}.{}", config.service, binding.method);
            writeln!(output, "{pattern} => Binding::Rpc({rpc:?}),")
                .map_err(|error| error.to_string())?;
        }
    }
    output.push_str("$environmental_pattern => $environmental_expression,\n}};\n}\n");
    Ok(())
}

fn template(config: &Config<'_>, source: &str) -> String {
    source
        .replace("CatalogServiceMethodInfo", config.method_info)
        .replace("CatalogServiceMethod", config.method_enum)
        .replace("CATALOG_SERVICE_METHODS", config.methods_const)
        .replace("dispatch_catalog_annotations", config.dispatch_macro)
        .replace("bind_catalog_action", config.bind_macro)
}
