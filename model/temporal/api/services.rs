//! Descriptor-backed workflow method and annotation dispatch generation.

use std::{collections::BTreeSet, fmt::Write};

use prost::Message;
use quote::ToTokens;
use umpire_temporal_annotate_syntax::{descriptor::Coverage, services::Classification};
use umpire_wasm::google::protobuf::FileDescriptorSet;

const SERVICE: &str = "temporal.api.workflowservice.v1.WorkflowService";

pub(crate) fn generate(
    bytes: &[u8],
    coverage: &Coverage,
    classifications: &[Classification],
) -> Result<String, String> {
    let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
    let service = descriptors
        .file
        .iter()
        .find_map(|file| {
            (file.package.as_deref() == Some("temporal.api.workflowservice.v1"))
                .then(|| {
                    file.service
                        .iter()
                        .find(|service| service.name.as_deref() == Some("WorkflowService"))
                })
                .flatten()
        })
        .ok_or("missing WorkflowService descriptor")?;
    let mut output = String::from(
        "/// Every RPC in upstream `WorkflowService` source order.\n\
         #[derive(Clone, Copy, Debug, Eq, PartialEq)]\n\
         pub enum WorkflowServiceMethod {\n",
    );
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed WorkflowService method")?;
        writeln!(output, "/// `{SERVICE}.{name}`.\n{name},").map_err(|error| error.to_string())?;
    }
    output.push_str("}\nimpl WorkflowServiceMethod {\n/// Original protobuf method name.\npub const fn name(self) -> &'static str { match self {\n");
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed WorkflowService method")?;
        writeln!(output, "Self::{name} => {name:?},").map_err(|error| error.to_string())?;
    }
    output.push_str("}}\n/// Parse an original protobuf method name.\npub fn parse(name: &str) -> Option<Self> { match name {\n");
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed WorkflowService method")?;
        writeln!(output, "{name:?} => Some(Self::{name}),").map_err(|error| error.to_string())?;
    }
    output.push_str(
        "_ => None,\n}}\n}\n\
        /// One upstream method and its generated Rust message types.\n\
        #[derive(Clone, Copy, Debug, Eq, PartialEq)]\n\
        pub struct WorkflowServiceMethodInfo {\n\
        /// The method identity.\npub method: WorkflowServiceMethod,\n\
        /// Fully qualified Rust request type path.\npub request_type: &'static str,\n\
        /// Fully qualified Rust response type path.\npub response_type: &'static str,\n}\n\
        /// Source-ordered `WorkflowService` request and response types.\n\
        pub const WORKFLOW_SERVICE_METHODS: &[WorkflowServiceMethodInfo] = &[\n",
    );
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed WorkflowService method")?;
        let request = public_type(method.input_type.as_deref().ok_or("missing request type")?);
        let response = public_type(method.output_type.as_deref().ok_or("missing response type")?);
        writeln!(output, "WorkflowServiceMethodInfo {{ method: WorkflowServiceMethod::{name}, request_type: {request:?}, response_type: {response:?} }},").map_err(|error| error.to_string())?;
    }
    output.push_str("];");
    emit_bindings(&mut output, classifications)?;
    output.push_str("\n/// Dispatch only declared annotation message types.\n#[macro_export]\nmacro_rules! dispatch_workflow_annotations {\n($method:expr, $response:expr, $ty:ident, $body:block, $fallback:expr) => { match ($method, $response) {\n");
    let annotated: BTreeSet<_> =
        coverage.messages.iter().map(|message| message.name.trim_start_matches('.')).collect();
    for method in &service.method {
        let name = method.name.as_deref().ok_or("unnamed WorkflowService method")?;
        for (response, ty) in [(false, &method.input_type), (true, &method.output_type)] {
            let ty = ty.as_deref().ok_or("missing method message type")?;
            if annotated.contains(ty.trim_start_matches('.')) {
                let path = macro_type(ty);
                writeln!(output, "($crate::WorkflowServiceMethod::{name}, {response}) => {{ type $ty = {path}; $body }},").map_err(|error| error.to_string())?;
            }
        }
    }
    output.push_str("_ => $fallback,\n}};\n}\n");
    Ok(output)
}

fn public_type(proto: &str) -> String {
    format!("::umpire_temporal_api::{}", proto.trim_start_matches('.').replace('.', "::"))
}

fn macro_type(proto: &str) -> String {
    format!("$crate::{}", proto.trim_start_matches('.').replace('.', "::"))
}

fn emit_bindings(output: &mut String, classifications: &[Classification]) -> Result<(), String> {
    output.push_str("\n/// Source-reviewed RPC classifications.\npub const RPC_BINDINGS: &[(&str, &str, &str)] = &[\n");
    for classification in classifications {
        for (rpc, role, target) in &classification.bindings {
            writeln!(output, "({rpc:?}, {role:?}, {target:?}),")
                .map_err(|error| error.to_string())?;
        }
    }
    output.push_str("];\n/// Bind every Model action to its classified RPC or declared environment.\n#[macro_export]\nmacro_rules! bind_workflow_action {\n($action:expr, $environmental_pattern:pat => $environmental_expression:expr) => { match $action {\n");
    let mut actions = BTreeSet::new();
    for classification in classifications {
        for binding in &classification.actions {
            let pattern = binding.action.to_token_stream().to_string();
            if !actions.insert(pattern.clone()) {
                return Err(format!("duplicate Model action RPC binding: {pattern}"));
            }
            let rpc = format!("{SERVICE}.{}", binding.method);
            writeln!(output, "{pattern} => Binding::Rpc({rpc:?}),")
                .map_err(|error| error.to_string())?;
        }
    }
    output.push_str("$environmental_pattern => $environmental_expression,\n}};\n}\n");
    Ok(())
}
