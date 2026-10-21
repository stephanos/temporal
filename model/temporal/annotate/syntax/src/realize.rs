//! Declarative realization handler expansion over the shared protobuf graph.

use std::collections::{BTreeMap, BTreeSet};

use proc_macro2::{Span, TokenStream};
use quote::{ToTokens, quote};
use syn::{
    Attribute, FnArg, GenericArgument, Ident, ImplItem, ImplItemFn, ItemImpl, LitStr, Path,
    PathArguments, ReturnType, Token, Type,
    ext::IdentExt,
    parse::{Parse, ParseStream},
    spanned::Spanned,
    visit::Visit,
};

use crate::realization::{Classification, ClassifiedField, Schema};

#[derive(Default)]
struct Fields {
    reads: Vec<String>,
    ignores: Vec<(String, String)>,
}

struct Produced {
    ty: Path,
    fields: Fields,
}

struct Declaration {
    target: Path,
    fields: Fields,
    produces: Vec<Produced>,
    project: Option<Path>,
}

fn field_path(input: ParseStream<'_>) -> syn::Result<String> {
    let mut parts = vec![input.call(Ident::parse_any)?.unraw().to_string()];
    while input.peek(Token![.]) {
        input.parse::<Token![.]>()?;
        parts.push(input.call(Ident::parse_any)?.unraw().to_string());
    }
    Ok(parts.join("."))
}

fn fields(input: ParseStream<'_>, kind: &Ident, result: &mut Fields) -> syn::Result<()> {
    let body;
    syn::parenthesized!(body in input);
    while !body.is_empty() {
        let path = field_path(&body)?;
        if kind == "ignores" {
            body.parse::<Token![=]>()?;
            let reason: LitStr = body.parse()?;
            if reason.value().trim().is_empty() {
                return Err(syn::Error::new(reason.span(), "ignore reason must not be empty"));
            }
            result.ignores.push((path, reason.value()));
        } else {
            result.reads.push(path);
        }
        if !body.is_empty() {
            body.parse::<Token![,]>()?;
        }
    }
    Ok(())
}

impl Parse for Declaration {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let mut result = Self {
            target: input.parse()?,
            fields: Fields::default(),
            produces: vec![],
            project: None,
        };
        while !input.is_empty() {
            input.parse::<Token![,]>()?;
            if input.is_empty() {
                break;
            }
            let key: Ident = input.parse()?;
            match key.to_string().as_str() {
                "reads" | "ignores" => fields(input, &key, &mut result.fields)?,
                "project" => {
                    let body;
                    syn::parenthesized!(body in input);
                    if result.project.is_some() {
                        return Err(syn::Error::new(key.span(), "duplicate project mapping"));
                    }
                    result.project = Some(body.parse()?);
                }
                "produces" => {
                    let body;
                    syn::parenthesized!(body in input);
                    let mut produced = Produced { ty: body.parse()?, fields: Fields::default() };
                    while !body.is_empty() {
                        body.parse::<Token![,]>()?;
                        if body.is_empty() {
                            break;
                        }
                        let key: Ident = body.parse()?;
                        if key != "fields" && key != "ignores" {
                            return Err(syn::Error::new(key.span(), "expected fields or ignores"));
                        }
                        fields(&body, &key, &mut produced.fields)?;
                    }
                    result.produces.push(produced);
                }
                _ => {
                    return Err(syn::Error::new(
                        key.span(),
                        "expected reads, ignores, produces, or project",
                    ));
                }
            }
        }
        Ok(result)
    }
}

#[derive(Clone, Copy)]
enum Output {
    Projection,
    Step,
    State,
    Reply,
}

struct Handler {
    function: ImplItemFn,
    declaration: Declaration,
    answer: bool,
    output: Output,
    input: Type,
    fallible: bool,
}

fn type_name(ty: &Type) -> Option<String> {
    let Type::Path(path) = ty else {
        return None;
    };
    path.path.segments.last().map(|segment| segment.ident.to_string())
}

fn type_arguments(ty: &Type) -> Vec<&Type> {
    let Type::Path(path) = ty else {
        return vec![];
    };
    let Some(segment) = path.path.segments.last() else {
        return vec![];
    };
    let PathArguments::AngleBracketed(arguments) = &segment.arguments else {
        return vec![];
    };
    arguments
        .args
        .iter()
        .filter_map(|arg| match arg {
            GenericArgument::Type(ty) => Some(ty),
            _ => None,
        })
        .collect()
}

fn handler(function: &ImplItemFn, attribute: &Attribute, answer: bool) -> syn::Result<Handler> {
    let declaration: Declaration = attribute.parse_args()?;
    if answer && declaration.produces.is_empty() {
        return Err(syn::Error::new(
            declaration.target.span(),
            "answer requires at least one produces request context",
        ));
    }
    let ReturnType::Type(_, result) = &function.sig.output else {
        return Err(syn::Error::new(function.sig.span(), "handler must return Result"));
    };
    let arguments = type_arguments(result);
    if type_name(result).as_deref() != Some("Result") || arguments.len() != 2 {
        return Err(syn::Error::new(result.span(), "handler must return Result<value, Refusal>"));
    }
    let output = match type_name(arguments[0]).as_deref() {
        Some("Reply") if answer => Output::Reply,
        Some("Projection") if !answer => Output::Projection,
        Some("Step") if !answer => Output::Step,
        _ if !answer && declaration.project.is_some() => Output::State,
        _ => {
            return Err(syn::Error::new(
                arguments[0].span(),
                if answer {
                    "answer handler must return Reply"
                } else {
                    "expected Projection, Step, or a state value with project"
                },
            ));
        }
    };
    if answer && declaration.project.is_some() || !answer && !declaration.produces.is_empty() {
        return Err(syn::Error::new(
            attribute.span(),
            "project belongs to observe; produces belongs to answer",
        ));
    }
    let typed: Vec<_> = function
        .sig
        .inputs
        .iter()
        .filter_map(|arg| match arg {
            FnArg::Typed(arg) => Some(arg.ty.as_ref()),
            FnArg::Receiver(_) => None,
        })
        .collect();
    if typed.is_empty() || typed.len() > 2 {
        return Err(syn::Error::new(
            function.sig.span(),
            "handler takes a typed message and optional Model state",
        ));
    }
    let fallible = type_name(typed[0]).as_deref() == Some("Result");
    let input = if fallible {
        let args = type_arguments(typed[0]);
        if answer || args.len() != 2 || type_name(args[1]).as_deref() != Some("i32") {
            return Err(syn::Error::new(
                typed[0].span(),
                "fallible observation requires Result<Response, i32>",
            ));
        }
        args[0].clone()
    } else {
        typed[0].clone()
    };
    Ok(Handler { function: function.clone(), declaration, answer, output, input, fallible })
}

fn message<'a>(schema: &'a Schema, ty: &Type) -> syn::Result<&'a crate::realization::Message> {
    let name = type_name(ty)
        .ok_or_else(|| syn::Error::new(ty.span(), "expected protobuf message type"))?;
    let matches: Vec<_> = schema
        .messages
        .values()
        .filter(|message| {
            message.rust_type.rsplit("::").next().map(str::trim) == Some(name.as_str())
        })
        .collect();
    if let [message] = matches.as_slice() {
        return Ok(message);
    }
    Err(syn::Error::new(ty.span(), "message type is absent or ambiguous in admitted catalog"))
}

fn rpc(target: &Path) -> syn::Result<(String, String)> {
    let names: Vec<_> = target.segments.iter().map(|segment| segment.ident.to_string()).collect();
    if names.len() < 2 {
        return Err(syn::Error::new(target.span(), "observe requires Service::Method"));
    }
    let service = if names.len() == 2 && names[0] == "WorkflowService" {
        "temporal.api.workflowservice.v1.WorkflowService".to_owned()
    } else {
        names[..names.len() - 1].join(".")
    };
    Ok((service, names[names.len() - 1].clone()))
}

fn classify(
    schema: &Schema,
    name: &str,
    fields: &Fields,
    span: Span,
) -> syn::Result<Vec<ClassifiedField>> {
    schema
        .fields(name, &fields.reads, &fields.ignores)
        .map_err(|error| syn::Error::new(span, error))
}

fn metadata(
    kind: &str,
    method: Option<&str>,
    message: &str,
    fields: &[ClassifiedField],
) -> TokenStream {
    let method = if let Some(method) = method { quote!(Some(#method)) } else { quote!(None) };
    let fields = fields.iter().map(|field| {
        let path = &field.path;
        let class = match field.class {
            Classification::Mapped => "mapped",
            Classification::Ignored => "ignored",
            Classification::Unsupported => "unsupported",
        };
        let reason =
            if let Some(reason) = &field.reason { quote!(Some(#reason)) } else { quote!(None) };
        quote!(::umpire_temporal::Field { path: #path, class: #class, reason: #reason })
    });
    quote!(::umpire_temporal::Context { kind: #kind, method: #method, message: #message, fields: &[#(#fields),*] })
}

fn call(handler: &Handler, value: &TokenStream) -> TokenStream {
    let name = &handler.function.sig.ident;
    let receiver =
        if handler.function.sig.inputs.iter().any(|arg| matches!(arg, FnArg::Receiver(_))) {
            quote!(self.#name)
        } else {
            quote!(Self::#name)
        };
    let state =
        (handler.function.sig.inputs.iter().filter(|arg| matches!(arg, FnArg::Typed(_))).count()
            == 2)
            .then(|| quote!(, state));
    quote!(#receiver(#value #state))
}

fn observation(
    handler: &Handler,
    model: &Path,
    schema: &Schema,
    contexts: &mut Vec<TokenStream>,
) -> syn::Result<TokenStream> {
    let target = &handler.declaration.target;
    let (service, method) = rpc(target)?;
    let method_info =
        schema.method(&service, &method).map_err(|error| syn::Error::new(target.span(), error))?;
    let response = message(schema, &handler.input)?;
    if response.name != method_info.output {
        return Err(syn::Error::new(handler.input.span(), "handler input is not the RPC response"));
    }
    let method = format!("/{service}/{method}");
    let fields = classify(schema, &response.name, &handler.declaration.fields, target.span())?;
    contexts.push(metadata("observed response", Some(&method), &response.name, &fields));
    let guards = schema
        .guards(&response.name, &fields, &quote!(__response))
        .map_err(|error| syn::Error::new(target.span(), error))?;
    let input: Type = syn::parse_str(&response.rust_type)?;
    let decode = if handler.fallible {
        quote! {
            let __response: Result<#input, i32> = if let Some(code) = __observed.code()? {
                Err(code)
            } else {
                let __response: #input = __observed.response()
                    .map_err(|error| ::umpire_wasm::protocol::Refusal::malformed(error.message))?;
                #guards
                Ok(__response)
            };
        }
    } else {
        quote!(let __response: #input = __observed.response()?; #guards)
    };
    let called = call(handler, &quote!(__response));
    let projection = match handler.output {
        Output::Projection => quote!(return #called;),
        Output::Step => quote!({
            let __step = #called?;
            let __action: <#model::World as ::umpire_dsl::Machine>::Action = __step.action.into();
            ::umpire_wasm::Projection::Candidates(vec![::umpire_wasm::Candidate {
                path: vec![__action.into()], outcome: __step.expected,
            }])
        }),
        Output::State => {
            let project = handler.declaration.project.as_ref();
            quote!(#project(state, #called?))
        }
        Output::Reply => unreachable!(),
    };
    let result = if matches!(handler.output, Output::Projection) {
        projection
    } else {
        quote!(return Ok(#projection);)
    };
    Ok(quote!(if __observed.method == #method { #decode #result }))
}

fn answer(
    handler: &Handler,
    model: &Path,
    schema: &Schema,
    contexts: &mut Vec<TokenStream>,
) -> syn::Result<TokenStream> {
    let target = &handler.declaration.target;
    let response = message(schema, &handler.input)?;
    let methods = schema.response_methods(&response.name);
    if methods.is_empty() {
        return Err(syn::Error::new(target.span(), "callback task must be an RPC response"));
    }
    let declared_type =
        Type::Path(syn::TypePath { attrs: vec![], qself: None, path: target.clone() });
    if response.name != message(schema, &declared_type)?.name {
        return Err(syn::Error::new(target.span(), "answer task does not match handler input"));
    }
    let fields = classify(schema, &response.name, &handler.declaration.fields, target.span())?;
    contexts.push(metadata(
        "incoming callback",
        if methods.len() == 1 { Some(methods[0].as_str()) } else { None },
        &response.name,
        &fields,
    ));
    let guards = schema
        .guards(&response.name, &fields, &quote!(__task))
        .map_err(|error| syn::Error::new(target.span(), error))?;
    let mut outputs = vec![];
    let mut seen = BTreeSet::new();
    for produced in &handler.declaration.produces {
        let ty =
            Type::Path(syn::TypePath { attrs: vec![], qself: None, path: produced.ty.clone() });
        let request = message(schema, &ty)?;
        let methods = schema.request_methods(&request.name);
        if methods.is_empty() {
            return Err(syn::Error::new(
                produced.ty.span(),
                "produced request must be an RPC input",
            ));
        }
        if !seen.insert(request.name.clone()) {
            return Err(syn::Error::new(produced.ty.span(), "duplicate produced request"));
        }
        let fields = classify(schema, &request.name, &produced.fields, produced.ty.span())?;
        contexts.push(metadata(
            "outgoing request",
            if methods.len() == 1 { Some(methods[0].as_str()) } else { None },
            &request.name,
            &fields,
        ));
        let guards = schema
            .guards(&request.name, &fields, &quote!(__request))
            .map_err(|error| syn::Error::new(produced.ty.span(), error))?;
        let canonical: Type = syn::parse_str(&request.rust_type)?;
        outputs.push(quote! {
            let _: fn(#ty) -> #canonical = |value| value;
            if __reply.request.type_url == <#canonical as ::prost::Name>::type_url() {
                let __request: #canonical = __reply.request.unpack()?;
                #guards
                __recognized = true;
            }
        });
    }
    let validate = (!outputs.is_empty()).then(|| quote! {
        let mut __recognized = false;
        #(#outputs)*
        if !__recognized {
            return Err(::umpire_wasm::protocol::Refusal::unsupported("unsupported reply request"));
        }
    });
    let called = call(handler, &quote!(__task));
    let input: Type = syn::parse_str(&response.rust_type)?;
    Ok(quote! {
        if __task_any.type_url == <#input as ::prost::Name>::type_url() {
            let __task: #input = __task_any.unpack()?;
            #guards
            let __reply = #called?;
            #validate
            let __action: <#model::World as ::umpire_dsl::Machine>::Action = __reply.action.into();
            let __delivery: Vec<::umpire_wasm::Transition<<#model::World as ::umpire_dsl::Machine>::Action>> =
                __reply.delivery.into_iter().map(|action| {
                    let action: <#model::World as ::umpire_dsl::Machine>::Action = action.into();
                    action.into()
                }).collect();
            return Ok(::umpire_wasm::AnswerPlan::new(__delivery, __action,
                ::umpire_wasm::google::protobuf::Any::pack(
                    &::umpire_temporal_api::umpire::temporal::v1::Reply {
                        request: Some(__reply.request),
                    }
                )
            ));
        }
    })
}

fn generated(path: &Path, schema: &Schema, variant: bool) -> bool {
    let mut parts = path.segments.iter().rev();
    let Some(last) = parts.next() else {
        return false;
    };
    let owner = if variant { parts.next().map(|part| &part.ident) } else { Some(&last.ident) };
    owner.is_some_and(|owner| {
        schema.enums.values().any(|enumeration| {
            enumeration.rust_type.rsplit("::").next().map(str::trim)
                == Some(owner.to_string().as_str())
        })
    })
}
fn positions(
    pattern: &syn::Pat,
    schema: &Schema,
    prefix: &[usize],
    result: &mut BTreeSet<Vec<usize>>,
) {
    match pattern {
        syn::Pat::Path(path) if generated(&path.path, schema, true) => {
            result.insert(prefix.to_vec());
        }
        syn::Pat::Tuple(tuple) => {
            for (index, element) in tuple.elems.iter().enumerate() {
                let mut position = prefix.to_vec();
                position.push(index);
                positions(element, schema, &position, result);
            }
        }
        syn::Pat::Or(pattern) => {
            for case in &pattern.cases {
                positions(case, schema, prefix, result);
            }
        }
        syn::Pat::Paren(pattern) => positions(&pattern.pat, schema, prefix, result),
        syn::Pat::Reference(pattern) => positions(&pattern.pat, schema, prefix, result),
        _ => {}
    }
}
fn wildcard(pattern: &syn::Pat, position: &[usize]) -> Option<Span> {
    match pattern {
        syn::Pat::Wild(_) | syn::Pat::Ident(_) => Some(pattern.span()),
        syn::Pat::Or(pattern) => pattern.cases.iter().find_map(|case| wildcard(case, position)),
        syn::Pat::Paren(pattern) => wildcard(&pattern.pat, position),
        syn::Pat::Reference(pattern) => wildcard(&pattern.pat, position),
        syn::Pat::Tuple(pattern) if !position.is_empty() => {
            pattern.elems.iter().nth(position[0]).and_then(|part| wildcard(part, &position[1..]))
        }
        _ => None,
    }
}
fn type_positions(ty: &Type, schema: &Schema) -> BTreeSet<Vec<usize>> {
    match ty {
        Type::Path(path) if generated(&path.path, schema, false) => BTreeSet::from([vec![]]),
        Type::Reference(reference) => type_positions(&reference.elem, schema),
        Type::Paren(ty) => type_positions(&ty.elem, schema),
        Type::Tuple(tuple) => tuple
            .elems
            .iter()
            .enumerate()
            .flat_map(|(index, ty)| {
                type_positions(ty, schema).into_iter().map(move |mut path| {
                    path.insert(0, index);
                    path
                })
            })
            .collect(),
        _ => BTreeSet::new(),
    }
}
struct EnumMatches<'a> {
    schema: &'a Schema,
    error: Option<syn::Error>,
    bindings: BTreeMap<String, BTreeSet<Vec<usize>>>,
}
impl EnumMatches<'_> {
    fn expression_positions(&self, expression: &syn::Expr) -> BTreeSet<Vec<usize>> {
        match expression {
            syn::Expr::Path(path) => self
                .bindings
                .get(&path.path.to_token_stream().to_string())
                .cloned()
                .unwrap_or_default(),
            syn::Expr::Tuple(tuple) => tuple
                .elems
                .iter()
                .enumerate()
                .flat_map(|(index, expr)| {
                    self.expression_positions(expr).into_iter().map(move |mut path| {
                        path.insert(0, index);
                        path
                    })
                })
                .collect(),
            syn::Expr::Call(call) if matches!(call.func.as_ref(), syn::Expr::Path(path) if generated(&path.path, self.schema, true)) => {
                BTreeSet::from([vec![]])
            }
            syn::Expr::Try(value) => self.expression_positions(&value.expr),
            syn::Expr::Paren(value) => self.expression_positions(&value.expr),
            syn::Expr::Reference(value) => self.expression_positions(&value.expr),
            syn::Expr::MethodCall(call)
                if matches!(
                    call.method.to_string().as_str(),
                    "map_err" | "unwrap" | "unwrap_or_else" | "expect"
                ) =>
            {
                self.expression_positions(&call.receiver)
            }
            _ => BTreeSet::new(),
        }
    }
}
impl<'ast> Visit<'ast> for EnumMatches<'_> {
    fn visit_impl_item_fn(&mut self, function: &'ast ImplItemFn) {
        self.bindings.clear();
        for input in &function.sig.inputs {
            if let FnArg::Typed(input) = input
                && let syn::Pat::Ident(binding) = input.pat.as_ref()
            {
                self.bindings
                    .insert(binding.ident.to_string(), type_positions(&input.ty, self.schema));
            }
        }
        self.visit_block(&function.block);
    }

    fn visit_local(&mut self, local: &'ast syn::Local) {
        let binding = match &local.pat {
            syn::Pat::Ident(binding) => Some((
                binding,
                local
                    .init
                    .as_ref()
                    .map(|init| self.expression_positions(&init.expr))
                    .unwrap_or_default(),
            )),
            syn::Pat::Type(typed) => match typed.pat.as_ref() {
                syn::Pat::Ident(binding) => Some((binding, type_positions(&typed.ty, self.schema))),
                _ => None,
            },
            _ => None,
        };
        if let Some((binding, paths)) = binding {
            self.bindings.insert(binding.ident.to_string(), paths);
        }
        syn::visit::visit_local(self, local);
    }

    fn visit_expr_match(&mut self, expression: &'ast syn::ExprMatch) {
        let mut paths = self.expression_positions(&expression.expr);
        for arm in &expression.arms {
            positions(&arm.pat, self.schema, &[], &mut paths);
        }
        for arm in &expression.arms {
            if let Some(span) = paths.iter().find_map(|path| wildcard(&arm.pat, path)) {
                self.error = Some(syn::Error::new(
                    span,
                    "generated enum matches must name every variant; wildcard arm is forbidden",
                ));
            }
        }
        syn::visit::visit_expr_match(self, expression);
    }
}
fn check_enums(item: &ItemImpl, schema: &Schema) -> syn::Result<()> {
    let mut visitor = EnumMatches { schema, error: None, bindings: BTreeMap::new() };
    visitor.visit_item_impl(item);
    visitor.error.map_or(Ok(()), Err)
}

/// Expand one realization impl against the admitted declaration graph.
pub fn expand(model: &Path, mut item: ItemImpl, schema: &Schema) -> syn::Result<TokenStream> {
    if item.trait_.is_some() || !item.generics.params.is_empty() {
        return Err(syn::Error::new(item.span(), "realization requires a concrete inherent impl"));
    }
    let mut handlers = Vec::new();
    let mut seen = BTreeSet::new();
    for entry in &mut item.items {
        let ImplItem::Fn(function) = entry else {
            continue;
        };
        for attribute in &function.attrs {
            if attribute.path().is_ident("propose") {
                return Err(syn::Error::new(attribute.span(), "propose is reserved"));
            }
            let is_answer = attribute.path().is_ident("answer");
            if !is_answer && !attribute.path().is_ident("observe") {
                continue;
            }
            let handler = handler(function, attribute, is_answer)?;
            let key = if is_answer {
                (true, handler.declaration.target.to_token_stream().to_string().replace(' ', ""))
            } else {
                let (service, method) = rpc(&handler.declaration.target)?;
                (false, format!("{service}/{method}"))
            };
            if !seen.insert(key) {
                return Err(syn::Error::new(
                    attribute.span(),
                    if is_answer {
                        "duplicate answer handler"
                    } else {
                        "duplicate observation handler"
                    },
                ));
            }
            handlers.push(handler);
        }
        if function.attrs.iter().any(|attribute| {
            attribute.path().is_ident("answer") || attribute.path().is_ident("observe")
        }) {
            function.attrs.push(syn::parse_quote!(#[allow(clippy::unnecessary_wraps, reason = "Realization handlers share a fallible return contract")]));
        }
        function.attrs.retain(|attribute| {
            !attribute.path().is_ident("answer") && !attribute.path().is_ident("observe")
        });
    }
    check_enums(&item, schema)?;
    let mut task_types = BTreeSet::new();
    for handler in handlers.iter().filter(|handler| handler.answer) {
        let task = message(schema, &handler.input)?;
        if !task_types.insert(task.name.clone()) {
            return Err(syn::Error::new(
                handler.declaration.target.span(),
                "duplicate answer handler",
            ));
        }
    }
    let mut contexts = vec![];
    let mut observations = vec![];
    let mut answers = vec![];
    for handler in &handlers {
        if handler.answer {
            answers.push(answer(handler, model, schema, &mut contexts)?);
        } else {
            observations.push(observation(handler, model, schema, &mut contexts)?);
        }
    }
    let ty = &item.self_ty;
    Ok(quote! {
        #item
        impl #ty {
            /// Declared coverage in each incoming and outgoing protobuf context.
            pub const REALIZATION: &'static [::umpire_temporal::Context] = &[#(#contexts),*];
        }
        impl ::umpire_wasm::Realize<#model::World> for #ty {
            fn observe(&self, observation: &::umpire_wasm::google::protobuf::Any, state: &#model::Model)
                -> Result<::umpire_wasm::Projection<<#model::World as ::umpire_dsl::Machine>::Action>, ::umpire_wasm::protocol::Refusal> {
                let _ = state;
                let __observed = ::umpire_temporal::support::Observed::new(observation)?;
                #(#observations)*
                Err(::umpire_wasm::protocol::Refusal::unsupported("unsupported observation method"))
            }
            fn answer(&self, value: &::umpire_wasm::google::protobuf::Any, state: &#model::Model)
                -> Result<::umpire_wasm::AnswerPlan<<#model::World as ::umpire_dsl::Machine>::Action>, ::umpire_wasm::protocol::Refusal> {
                let _ = state;
                let __callback: ::umpire_temporal_api::umpire::temporal::v1::Callback = value.unpack()?;
                let __task_any = ::umpire_wasm::google::protobuf::Any::required(__callback.task)?;
                if __task_any.type_url.is_empty() {
                    return Err(::umpire_wasm::protocol::Refusal::malformed("Any type URL is missing"));
                }
                #(#answers)*
                Err(::umpire_wasm::protocol::Refusal::unsupported("unsupported Any type URL"))
            }
        }
    })
}
