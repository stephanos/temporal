//! Declarative realization handler expansion over the shared protobuf graph.

use std::collections::{BTreeMap, BTreeSet};

use proc_macro2::{LineColumn, Span, TokenStream};
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

mod patterns;
mod source;
use patterns::{match_positions, positions, wildcard};
use source::{EnumKind, Scope};

#[derive(Clone, PartialEq, Eq, PartialOrd, Ord)]
enum Position {
    Reference,
    Tuple(usize, usize),
    Variant(String, usize, usize),
}

type Positions = BTreeMap<Vec<Position>, EnumKind>;

fn position(kind: EnumKind) -> Positions {
    if kind == EnumKind::Other { Positions::new() } else { Positions::from([(vec![], kind)]) }
}

fn referenced(mut paths: Positions) -> Positions {
    if paths.is_empty() {
        paths.insert(vec![], EnumKind::Other);
    }
    paths
        .into_iter()
        .map(|(mut path, kind)| {
            path.insert(0, Position::Reference);
            (path, kind)
        })
        .collect()
}

fn dereferenced(paths: Positions) -> Positions {
    if paths.is_empty() || paths.keys().any(|path| path.first() != Some(&Position::Reference)) {
        return position(EnumKind::Unknown);
    }
    paths.into_iter().map(|(path, kind)| (path[1..].to_vec(), kind)).collect()
}

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
                    .map_err(|error| ::umpire_runtime::protocol::Refusal::malformed(error.message))?;
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
            ::umpire_runtime::Projection::Candidates(vec![::umpire_runtime::Candidate {
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
            return Err(::umpire_runtime::protocol::Refusal::unsupported("unsupported reply request"));
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
            let __delivery: Vec<::umpire_runtime::Transition<<#model::World as ::umpire_dsl::Machine>::Action>> =
                __reply.delivery.into_iter().map(|action| {
                    let action: <#model::World as ::umpire_dsl::Machine>::Action = action.into();
                    action.into()
                }).collect();
            return Ok(::umpire_runtime::AnswerPlan::new(__delivery, __action,
                ::umpire_runtime::google::protobuf::Any::pack(
                    &::umpire_runtime::realization::Reply {
                        request: Some(__reply.request),
                    }
                )
            ));
        }
    })
}

fn type_positions(ty: &Type, schema: &Schema, scope: &Scope) -> Positions {
    match ty {
        Type::Path(path)
            if scope.framework(&path.path, "Option", schema)
                || scope.framework(&path.path, "Result", schema) =>
        {
            let names = if scope.framework(&path.path, "Option", schema) {
                &["Some"][..]
            } else {
                &["Ok", "Err"][..]
            };
            type_arguments(ty)
                .into_iter()
                .zip(names)
                .flat_map(|(ty, name)| {
                    type_positions(ty, schema, scope).into_iter().map(move |(mut path, kind)| {
                        path.insert(0, Position::Variant((*name).into(), 0, 1));
                        (path, kind)
                    })
                })
                .collect()
        }
        Type::Path(path) => position(scope.kind(&path.path, schema, false)),
        Type::Reference(reference) => referenced(type_positions(&reference.elem, schema, scope)),
        Type::Paren(ty) => type_positions(&ty.elem, schema, scope),
        Type::Tuple(tuple) => tuple
            .elems
            .iter()
            .enumerate()
            .flat_map(|(index, ty)| {
                type_positions(ty, schema, scope).into_iter().map(move |(mut path, kind)| {
                    path.insert(0, Position::Tuple(index, tuple.elems.len()));
                    (path, kind)
                })
            })
            .collect(),
        _ => position(EnumKind::Unknown),
    }
}
struct EnumMatches<'a> {
    schema: &'a Schema,
    scope: Scope,
    error: Option<syn::Error>,
    bindings: BTreeMap<String, Positions>,
    assigned: BTreeSet<String>,
}
impl EnumMatches<'_> {
    fn function(&mut self, signature: &syn::Signature, block: &syn::Block) {
        let scope = self.scope.clone();
        let bindings = self.bindings.clone();
        let assigned = std::mem::take(&mut self.assigned);
        self.scope.generics(&signature.generics);
        self.bindings.clear();
        for input in &signature.inputs {
            if let FnArg::Typed(input) = input
                && let syn::Pat::Ident(binding) = input.pat.as_ref()
            {
                self.bindings.insert(
                    binding.ident.to_string(),
                    type_positions(&input.ty, self.schema, &self.scope),
                );
            }
        }
        self.visit_block(block);
        self.scope = scope;
        self.bindings = bindings;
        self.assigned = assigned;
    }

    fn expression_positions(&self, expression: &syn::Expr) -> Positions {
        match expression {
            syn::Expr::Path(path)
                if self.assigned.contains(&path.path.to_token_stream().to_string()) =>
            {
                position(EnumKind::Unknown)
            }
            syn::Expr::Path(path) => self
                .bindings
                .get(&path.path.to_token_stream().to_string())
                .cloned()
                .unwrap_or_else(|| position(self.scope.kind(&path.path, self.schema, true))),
            syn::Expr::Tuple(tuple) => tuple
                .elems
                .iter()
                .enumerate()
                .flat_map(|(index, expr)| {
                    self.expression_positions(expr).into_iter().map(move |(mut path, kind)| {
                        path.insert(0, Position::Tuple(index, tuple.elems.len()));
                        (path, kind)
                    })
                })
                .collect(),
            syn::Expr::Call(call) if matches!(call.func.as_ref(), syn::Expr::Path(path) if self.scope.kind(&path.path, self.schema, true) == EnumKind::Generated) => {
                position(EnumKind::Generated)
            }
            syn::Expr::Call(call) if matches!(call.func.as_ref(), syn::Expr::Path(path) if ["Some", "Ok", "Err"].iter().any(|name| self.scope.framework(&path.path, name, self.schema))) =>
            {
                let syn::Expr::Path(path) = call.func.as_ref() else {
                    return position(EnumKind::Unknown);
                };
                call.args
                    .iter()
                    .enumerate()
                    .flat_map(|(index, value)| {
                        self.expression_positions(value).into_iter().map(
                            move |(mut position, kind)| {
                                position.insert(
                                    0,
                                    Position::Variant(
                                        self.scope.constructor(&path.path),
                                        index,
                                        call.args.len(),
                                    ),
                                );
                                (position, kind)
                            },
                        )
                    })
                    .collect()
            }
            syn::Expr::Paren(value) => self.expression_positions(&value.expr),
            syn::Expr::Reference(value) => referenced(self.expression_positions(&value.expr)),
            syn::Expr::Unary(value) if matches!(value.op, syn::UnOp::Deref(_)) => {
                dereferenced(self.expression_positions(&value.expr))
            }
            syn::Expr::Cast(value) => type_positions(&value.ty, self.schema, &self.scope),
            syn::Expr::Lit(_) => Positions::new(),
            _ => position(EnumKind::Unknown),
        }
    }
}
impl<'ast> Visit<'ast> for EnumMatches<'_> {
    fn visit_impl_item_fn(&mut self, function: &'ast ImplItemFn) {
        self.function(&function.sig, &function.block);
    }

    fn visit_item_fn(&mut self, function: &'ast syn::ItemFn) {
        self.function(&function.sig, &function.block);
    }

    fn visit_pat_ident(&mut self, pattern: &'ast syn::PatIdent) {
        if self.scope.kind(&Path::from(pattern.ident.clone()), self.schema, true)
            != EnumKind::Generated
        {
            self.bindings.insert(pattern.ident.to_string(), position(EnumKind::Unknown));
        }
        syn::visit::visit_pat_ident(self, pattern);
    }

    fn visit_pat_type(&mut self, pattern: &'ast syn::PatType) {
        syn::visit::visit_pat_type(self, pattern);
        if let syn::Pat::Ident(binding) = pattern.pat.as_ref() {
            self.bindings.insert(
                binding.ident.to_string(),
                type_positions(&pattern.ty, self.schema, &self.scope),
            );
        }
    }

    fn visit_arm(&mut self, arm: &'ast syn::Arm) {
        let bindings = self.bindings.clone();
        syn::visit::visit_arm(self, arm);
        self.bindings = bindings;
    }

    fn visit_expr_closure(&mut self, expression: &'ast syn::ExprClosure) {
        let bindings = self.bindings.clone();
        syn::visit::visit_expr_closure(self, expression);
        self.bindings = bindings;
    }

    fn visit_expr_if(&mut self, expression: &'ast syn::ExprIf) {
        let bindings = self.bindings.clone();
        syn::visit::visit_expr_if(self, expression);
        self.bindings = bindings;
    }

    fn visit_expr_while(&mut self, expression: &'ast syn::ExprWhile) {
        let bindings = self.bindings.clone();
        syn::visit::visit_expr_while(self, expression);
        self.bindings = bindings;
    }

    fn visit_expr_for_loop(&mut self, expression: &'ast syn::ExprForLoop) {
        let bindings = self.bindings.clone();
        syn::visit::visit_expr_for_loop(self, expression);
        self.bindings = bindings;
    }

    fn visit_block(&mut self, block: &'ast syn::Block) {
        let scope = self.scope.clone();
        let bindings = self.bindings.clone();
        self.scope.block(block);
        syn::visit::visit_block(self, block);
        self.scope = scope;
        self.bindings = bindings;
    }

    fn visit_local(&mut self, local: &'ast syn::Local) {
        let binding = match &local.pat {
            syn::Pat::Ident(binding) => Some((
                binding,
                local.init.as_ref().map_or_else(
                    || position(EnumKind::Unknown),
                    |init| self.expression_positions(&init.expr),
                ),
            )),
            syn::Pat::Type(typed) => match typed.pat.as_ref() {
                syn::Pat::Ident(binding) => {
                    Some((binding, type_positions(&typed.ty, self.schema, &self.scope)))
                }
                _ => None,
            },
            _ => None,
        };
        if let Some(init) = &local.init {
            self.visit_expr(&init.expr);
            if let Some((_, diverge)) = &init.diverge {
                self.visit_expr(diverge);
            }
        }
        self.visit_pat(&local.pat);
        if let Some((binding, paths)) = binding {
            self.bindings.insert(binding.ident.to_string(), paths);
        }
    }

    fn visit_expr_assign(&mut self, expression: &'ast syn::ExprAssign) {
        syn::visit::visit_expr_assign(self, expression);
        if let syn::Expr::Path(path) = expression.left.as_ref() {
            let name = path.path.to_token_stream().to_string();
            let positions = self.expression_positions(&expression.right);
            if positions.values().any(|kind| *kind != EnumKind::Other) {
                self.assigned.insert(name.clone());
            }
            self.bindings.insert(name, positions);
        }
    }

    fn visit_expr_match(&mut self, expression: &'ast syn::ExprMatch) {
        let mut paths = match_positions(self.expression_positions(&expression.expr));
        for arm in &expression.arms {
            positions(&arm.pat, self.schema, &self.scope, &[], &mut paths);
        }
        for arm in &expression.arms {
            if let Some((span, kind)) = paths
                .iter()
                .filter(|(_, kind)| **kind != EnumKind::Other)
                .find_map(|(path, kind)| {
                    wildcard(&arm.pat, path, &self.scope, self.schema).map(|span| (span, *kind))
                })
            {
                if kind == EnumKind::Unknown
                    && expression.arms.iter().any(|arm| matches!(arm.pat, syn::Pat::Lit(_)))
                {
                    continue;
                }
                self.error = Some(syn::Error::new(
                    span,
                    if kind == EnumKind::Generated {
                        "generated enum matches must name every variant; wildcard arm is forbidden"
                    } else {
                        "cannot resolve external enum wildcard coverage; use a canonical enum path or explicit variants"
                    },
                ));
            }
        }
        syn::visit::visit_expr_match(self, expression);
    }
}
fn check_enums(
    item: &ItemImpl,
    schema: &Schema,
    mut scope: Scope,
    model: &Path,
) -> syn::Result<()> {
    scope.model(model);
    let mut visitor = EnumMatches {
        schema,
        scope,
        error: None,
        bindings: BTreeMap::new(),
        assigned: BTreeSet::new(),
    };
    visitor.visit_item_impl(item);
    visitor.error.map_or(Ok(()), Err)
}

/// Expand one realization impl against the admitted declaration graph.
pub fn expand(model: &Path, item: ItemImpl, schema: &Schema) -> syn::Result<TokenStream> {
    expand_scoped(model, item, schema, Scope::default())
}

/// Expand using the enclosing source module's ordinary Rust type aliases and imports.
pub fn expand_in_source(
    model: &Path,
    item: ItemImpl,
    schema: &Schema,
    source: &str,
    location: LineColumn,
) -> syn::Result<TokenStream> {
    let scope = Scope::in_source(source, location)
        .map_err(|error| syn::Error::new(item.impl_token.span, error.to_string()))?;
    expand_scoped(model, item, schema, scope)
}

struct ProposalHandler {
    name: Ident,
    matches: Option<Ident>,
    polling: Option<Ident>,
}

fn proposal_handler(name: &Ident, attribute: &Attribute) -> syn::Result<ProposalHandler> {
    let mut handler = ProposalHandler { name: name.clone(), matches: None, polling: None };
    if matches!(attribute.meta, syn::Meta::Path(_)) {
        return Ok(handler);
    }
    let options = attribute.parse_args_with(
        syn::punctuated::Punctuated::<syn::MetaNameValue, Token![,]>::parse_terminated,
    )?;
    for option in options {
        let slot = if option.path.is_ident("matches") {
            &mut handler.matches
        } else if option.path.is_ident("continue_polling") {
            &mut handler.polling
        } else {
            return Err(syn::Error::new(option.path.span(), "unknown proposal hook"));
        };
        let syn::Expr::Path(value) = option.value else {
            return Err(syn::Error::new(
                option.value.span(),
                "proposal hook requires a method name",
            ));
        };
        let name = value
            .path
            .get_ident()
            .ok_or_else(|| syn::Error::new(value.span(), "proposal hook requires a method name"))?;
        if slot.replace(name.clone()).is_some() {
            return Err(syn::Error::new(option.path.span(), "duplicate proposal hook"));
        }
    }
    Ok(handler)
}

fn proposal_dispatch(model: &Path, handler: &ProposalHandler) -> TokenStream {
    let name = &handler.name;
    let matches = handler.matches.as_ref().map(|name| quote! {
        fn matches(&self, request: &::umpire_runtime::google::protobuf::Any, observation: &::umpire_runtime::google::protobuf::Any)
            -> Result<(), ::umpire_runtime::protocol::Refusal> { self.#name(request, observation) }
    });
    let polling = handler.polling.as_ref().map(|name| quote! {
        fn continue_polling(&self, observation: &::umpire_runtime::google::protobuf::Any, state: &#model::Model, action: Option<&<#model::World as ::umpire_dsl::Machine>::Action>)
            -> Result<bool, ::umpire_runtime::protocol::Refusal> { self.#name(observation, state, action) }
    });
    quote! {
        fn propose(&self, state: &#model::Model, seed: u64)
            -> Result<::umpire_runtime::Proposal<<#model::World as ::umpire_dsl::Machine>::Action>, ::umpire_runtime::protocol::Refusal> { self.#name(state, seed) }
        #matches
        #polling
    }
}

fn admit_handlers(item: &mut ItemImpl) -> syn::Result<(Vec<Handler>, Option<ProposalHandler>)> {
    let mut handlers = Vec::new();
    let mut proposal = None;
    let mut seen = BTreeSet::new();
    for entry in &mut item.items {
        let ImplItem::Fn(function) = entry else {
            continue;
        };
        for attribute in &function.attrs {
            if attribute.path().is_ident("propose") {
                if proposal.is_some() {
                    return Err(syn::Error::new(attribute.span(), "duplicate proposal handler"));
                }
                proposal = Some(proposal_handler(&function.sig.ident, attribute)?);
                continue;
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
            !attribute.path().is_ident("answer")
                && !attribute.path().is_ident("observe")
                && !attribute.path().is_ident("propose")
        });
    }
    Ok((handlers, proposal))
}

fn expand_scoped(
    model: &Path,
    mut item: ItemImpl,
    schema: &Schema,
    scope: Scope,
) -> syn::Result<TokenStream> {
    if item.trait_.is_some() || !item.generics.params.is_empty() {
        return Err(syn::Error::new(item.span(), "realization requires a concrete inherent impl"));
    }
    let (handlers, proposal) = admit_handlers(&mut item)?;
    check_enums(&item, schema, scope, model)?;
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
    let proposal = proposal.as_ref().map(|name| proposal_dispatch(model, name));
    let answer_dispatch = (!answers.is_empty()).then(|| quote! {
            fn answer(&self, value: &::umpire_runtime::google::protobuf::Any, state: &#model::Model)
                -> Result<::umpire_runtime::AnswerPlan<<#model::World as ::umpire_dsl::Machine>::Action>, ::umpire_runtime::protocol::Refusal> {
                let _ = state;
                let __callback: ::umpire_runtime::realization::Callback = value.unpack()?;
                let __task_any = ::umpire_runtime::google::protobuf::Any::required(__callback.task)?;
                if __task_any.type_url.is_empty() {
                    return Err(::umpire_runtime::protocol::Refusal::malformed("Any type URL is missing"));
                }
                #(#answers)*
                Err(::umpire_runtime::protocol::Refusal::unsupported("unsupported Any type URL"))
            }
    });
    let ty = &item.self_ty;
    Ok(quote! {
        #item
        impl #ty {
            /// Declared coverage in each incoming and outgoing protobuf context.
            pub const REALIZATION: &'static [::umpire_temporal::Context] = &[#(#contexts),*];
        }
        impl ::umpire_runtime::Realize<#model::World> for #ty {
            #proposal
            fn observe(&self, observation: &::umpire_runtime::google::protobuf::Any, state: &#model::Model)
                -> Result<::umpire_runtime::Projection<<#model::World as ::umpire_dsl::Machine>::Action>, ::umpire_runtime::protocol::Refusal> {
                let _ = state;
                let __observed = ::umpire_temporal::support::Observed::new(observation)?;
                #(#observations)*
                Err(::umpire_runtime::protocol::Refusal::unsupported("unsupported observation method"))
            }
            #answer_dispatch
        }
    })
}
