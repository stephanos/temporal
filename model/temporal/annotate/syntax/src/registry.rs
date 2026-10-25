//! Descriptor-admitted action inventory for shared realization consumers.

use std::collections::{BTreeMap, BTreeSet};

use proc_macro2::{LineColumn, Span, TokenStream};
use quote::{ToTokens, format_ident, quote, quote_spanned};
use syn::{
    Ident, ItemEnum, ItemImpl, LitStr, Path, Token, Type,
    ext::IdentExt,
    parse::{Parse, ParseStream},
    spanned::Spanned,
};

use crate::realization::Schema;

struct Input {
    model: Path,
    actors: Vec<ItemEnum>,
    item: ItemImpl,
}

fn section(input: ParseStream<'_>, expected: &str) -> syn::Result<TokenStream> {
    let name: Ident = input.parse()?;
    if name != expected {
        return Err(syn::Error::new(name.span(), format!("expected {expected}")));
    }
    let body;
    syn::braced!(body in input);
    body.parse()
}

impl Parse for Input {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let model = syn::parse2(section(input, "model")?)?;
        let actors = syn::parse2::<syn::File>(section(input, "actors")?)?
            .items
            .into_iter()
            .map(|item| match item {
                syn::Item::Enum(item) => Ok(item),
                _ => Err(syn::Error::new_spanned(item, "expected actual actor enum inventory")),
            })
            .collect::<syn::Result<_>>()?;
        let item = syn::parse2(section(input, "body")?)?;
        Ok(Self { model, actors, item })
    }
}

#[derive(Default)]
struct Options {
    values: BTreeMap<String, (Ident, TokenStream)>,
    reads: BTreeMap<String, Vec<String>>,
}

impl Parse for Options {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let mut result = Self::default();
        while !input.is_empty() {
            let key: Ident = input.parse()?;
            let name = key.to_string();
            if result.values.contains_key(&name) || result.reads.contains_key(&name) {
                return Err(syn::Error::new(key.span(), "duplicate temporal action option"));
            }
            if name == "request_reads" || name == "response_reads" {
                let body;
                syn::parenthesized!(body in input);
                let mut paths = Vec::new();
                while !body.is_empty() {
                    let mut path = body.call(Ident::parse_any)?.unraw().to_string();
                    while body.peek(Token![.]) {
                        body.parse::<Token![.]>()?;
                        path.push('.');
                        path.push_str(&body.call(Ident::parse_any)?.unraw().to_string());
                    }
                    paths.push(path);
                    if !body.is_empty() {
                        body.parse::<Token![,]>()?;
                    }
                }
                result.reads.insert(name, paths);
            } else {
                input.parse::<Token![=]>()?;
                let value = if ["inputs", "request", "response", "receipt", "target"]
                    .contains(&name.as_str())
                {
                    input.parse::<Type>()?.into_token_stream()
                } else {
                    input.parse::<syn::Expr>()?.into_token_stream()
                };
                result.values.insert(name, (key, value));
            }
            if !input.is_empty() {
                input.parse::<Token![,]>()?;
            }
        }
        Ok(result)
    }
}

impl Options {
    fn admit(&self, allowed: &[&str], span: Span) -> syn::Result<()> {
        for (key, _) in self.values.values() {
            if !allowed.contains(&key.to_string().as_str()) {
                return Err(syn::Error::new(key.span(), "unknown temporal action option"));
            }
        }
        for key in self.reads.keys() {
            if !allowed.contains(&key.as_str()) {
                return Err(syn::Error::new(span, "field reads belong to RPC action declarations"));
            }
        }
        Ok(())
    }

    fn required<T: Parse>(&self, name: &str, span: Span) -> syn::Result<T> {
        let (_, value) = self.values.get(name).ok_or_else(|| {
            syn::Error::new(span, format!("missing temporal declaration option: {name}"))
        })?;
        syn::parse2(value.clone())
    }
}

struct Row {
    actor: Ident,
    variant: syn::Variant,
    options: Options,
    event: Option<LitStr>,
}

fn rows(actors: &[ItemEnum]) -> syn::Result<Vec<Row>> {
    let mut result = Vec::new();
    let mut seen = BTreeSet::new();
    for actor in actors {
        if !actor.generics.params.is_empty() {
            return Err(syn::Error::new(
                actor.generics.span(),
                "registry requires concrete actors",
            ));
        }
        for variant in &actor.variants {
            if !seen.insert((actor.ident.to_string(), variant.ident.to_string())) {
                return Err(syn::Error::new(variant.span(), "duplicate action in actor inventory"));
            }
            let mut declaration = None;
            for attribute in variant.attrs.iter().filter(|attr| attr.path().is_ident("metadata")) {
                let namespaces = attribute.parse_args_with(
                    syn::punctuated::Punctuated::<syn::Meta, Token![,]>::parse_terminated,
                )?;
                for namespace in namespaces {
                    if namespace.path().is_ident("temporal") {
                        if declaration.is_some() {
                            return Err(syn::Error::new(
                                namespace.span(),
                                "duplicate temporal action declaration",
                            ));
                        }
                        let syn::Meta::List(list) = namespace else {
                            return Err(syn::Error::new(
                                namespace.span(),
                                "temporal declaration requires options",
                            ));
                        };
                        declaration = Some(syn::parse2::<Options>(list.tokens)?);
                    }
                }
            }
            let options = declaration.ok_or_else(|| {
                syn::Error::new(
                    variant.ident.span(),
                    "action variant requires one temporal declaration",
                )
            })?;
            let event = if options.values.contains_key("event") {
                if options.values.keys().any(|key| {
                    ["rpc", "direction", "request", "response", "fill"].contains(&key.as_str())
                }) {
                    return Err(syn::Error::new(
                        variant.ident.span(),
                        "event declaration cannot contain RPC options",
                    ));
                }
                options.admit(&["event", "evidence"], variant.span())?;
                let event: LitStr = options.required("event", variant.span())?;
                if event.value().trim().is_empty() {
                    return Err(syn::Error::new(event.span(), "event reason must not be empty"));
                }
                Some(event)
            } else {
                options.admit(
                    &[
                        "rpc",
                        "direction",
                        "request",
                        "response",
                        "fill",
                        "evidence",
                        "request_reads",
                        "response_reads",
                    ],
                    variant.span(),
                )?;
                None
            };
            result.push(Row {
                actor: actor.ident.clone(),
                variant: variant.clone(),
                options,
                event,
            });
        }
    }
    Ok(result)
}

struct Rpc {
    method: String,
    request: Type,
    response: Type,
    declared_request: Type,
    declared_response: Type,
    request_guards: TokenStream,
    response_guards: TokenStream,
}

fn rpc(options: &Options, schema: &Schema, span: Span) -> syn::Result<Rpc> {
    let path: Path = options.required("rpc", span)?;
    let names: Vec<_> = path.segments.iter().map(|segment| segment.ident.to_string()).collect();
    if names.len() < 2 {
        return Err(syn::Error::new(path.span(), "RPC declaration requires Service::Method"));
    }
    let service = if names.len() == 2 {
        schema
            .service_named(&names[0])
            .map_err(|error| syn::Error::new(path.span(), error))?
            .map_or_else(|| names[0].clone(), str::to_owned)
    } else {
        names[..names.len() - 1].join(".")
    };
    let method = names.last().ok_or_else(|| syn::Error::new(span, "missing RPC method"))?;
    let node =
        schema.method(&service, method).map_err(|error| syn::Error::new(path.span(), error))?;
    let request =
        schema.message(&node.input).map_err(|error| syn::Error::new(path.span(), error))?;
    let response =
        schema.message(&node.output).map_err(|error| syn::Error::new(path.span(), error))?;
    let guards = |name: &str, key: &str, value: TokenStream| -> syn::Result<TokenStream> {
        let fields = schema
            .fields(name, options.reads.get(key).map_or(&[], Vec::as_slice), &[])
            .map_err(|error| syn::Error::new(path.span(), error))?;
        schema.guards(name, &fields, &value).map_err(|error| syn::Error::new(path.span(), error))
    };
    Ok(Rpc {
        method: format!("/{service}/{method}"),
        request: syn::parse_str(&request.rust_type)?,
        response: syn::parse_str(&response.rust_type)?,
        declared_request: options.required("request", span)?,
        declared_response: options.required("response", span)?,
        request_guards: guards(&request.name, "request_reads", quote!(__request))?,
        response_guards: guards(&response.name, "response_reads", quote!(__response))?,
    })
}

impl Rpc {
    fn witnesses(&self) -> TokenStream {
        let Self { request, response, declared_request, declared_response, .. } = self;
        let request_witness = quote_spanned!(Span::call_site().located_at(declared_request.span())=>
            let _: fn(#declared_request) -> #request = |value| value;
        );
        let response_witness = quote_spanned!(Span::call_site().located_at(declared_response.span())=>
            let _: fn(#declared_response) -> #response = |value| value;
        );
        quote!(#request_witness #response_witness)
    }

    fn configuration(&self, runtime: &Path) -> TokenStream {
        let method = &self.method;
        quote!(#runtime::registry::Rpc { method: #method, types: ::core::marker::PhantomData })
    }
}

fn take(item: &mut ItemImpl, name: &str) -> syn::Result<Option<Options>> {
    let mut result = None;
    for attribute in &item.attrs {
        if attribute.path().is_ident(name) {
            if result.is_some() {
                return Err(syn::Error::new(
                    attribute.span(),
                    format!("duplicate {name} declaration"),
                ));
            }
            result = Some(attribute.parse_args()?);
        }
    }
    item.attrs.retain(|attribute| !attribute.path().is_ident(name));
    Ok(result)
}

fn pattern(model: &Path, row: &Row) -> TokenStream {
    let actor = &row.actor;
    let variant = &row.variant.ident;
    let payload = match &row.variant.fields {
        syn::Fields::Unit => quote!(),
        syn::Fields::Unnamed(fields) => {
            let wildcards = fields.unnamed.iter().map(|_| quote!(_));
            quote!((#(#wildcards),*))
        }
        syn::Fields::Named(fields) => {
            let names = fields.named.iter().filter_map(|field| field.ident.as_ref());
            quote!({#(#names: _),*})
        }
    };
    quote!(#model::#actor::#variant #payload)
}

fn action_dispatch(
    model: &Path,
    actors: &[ItemEnum],
    rows: &[Row],
    bodies: &[TokenStream],
) -> TokenStream {
    if actors.is_empty() {
        return quote!(match *action {});
    }
    let arms = actors.iter().map(|actor| {
        let name = &actor.ident;
        let variants =
            rows.iter().zip(bodies).filter(|(row, _)| row.actor == *name).map(|(row, body)| {
                let pattern = pattern(model, row);
                quote!(__actor @ #pattern => { #body })
            });
        let matched = if actor.variants.is_empty() {
            quote!(match *__actor {})
        } else {
            quote!(match __actor { #(#variants),* })
        };
        quote!(#model::Action::#name(__actor) => #matched)
    });
    quote!(match action { #(#arms),* })
}

/// The authored binding location retained through the generic inventory callback.
pub fn source_span(input: TokenStream) -> syn::Result<Span> {
    Ok(syn::parse2::<Input>(input)?.item.impl_token.span)
}

/// Expand the complete generic inventory into one typed Temporal registry.
pub fn expand(input: TokenStream, runtime: &Path, schema: &Schema) -> syn::Result<TokenStream> {
    expand_scoped(syn::parse2(input)?, runtime, schema, None)
}

/// Expand while preserving enum identities from the authored Rust scope.
pub fn expand_in_source(
    input: TokenStream,
    runtime: &Path,
    schema: &Schema,
    source: &str,
    location: LineColumn,
) -> syn::Result<TokenStream> {
    expand_scoped(syn::parse2(input)?, runtime, schema, Some((source, location)))
}

fn expand_scoped(
    mut input: Input,
    runtime: &Path,
    schema: &Schema,
    source: Option<(&str, LineColumn)>,
) -> syn::Result<TokenStream> {
    let model = &input.model;
    let item = &mut input.item;
    if item.trait_.is_some() || !item.generics.params.is_empty() {
        return Err(syn::Error::new(item.span(), "registry requires a concrete inherent impl"));
    }
    crate::realize::check_registry_enums(item, schema, model, source)?;
    let options = take(item, "registry")?
        .ok_or_else(|| syn::Error::new(item.span(), "registry requires typed inputs"))?;
    options.admit(&["inputs"], item.span())?;
    let inputs: Type = options.required("inputs", item.span())?;
    let entity = take(item, "entity")?;
    let read = take(item, "read")?;
    let configuration =
        configurations(item, model, runtime, schema, entity.as_ref(), read.as_ref())?;
    let rows = rows(&input.actors)?;
    hook_contracts(item, &rows, entity.as_ref(), read.as_ref());
    let ty = &item.self_ty;
    let Type::Path(binding) = ty.as_ref() else {
        return Err(syn::Error::new(ty.span(), "registry binding requires a named type"));
    };
    let name = &binding
        .path
        .segments
        .last()
        .ok_or_else(|| syn::Error::new(ty.span(), "missing binding type"))?
        .ident;
    let request_sum = format_ident!("{name}Request");
    let visibility =
        &input.actors.first().map_or(syn::Visibility::Inherited, |actor| actor.vis.clone());
    let Dispatches { request_variants, pack_arms, fills, classifications, admission, evidence } =
        dispatches(&rows, schema, runtime, &request_sum)?;
    let pack = if request_variants.is_empty() {
        quote!(match *self {})
    } else {
        quote!(match self { #(#pack_arms),* })
    };
    let classify = action_dispatch(model, &input.actors, &rows, &classifications);
    let fill = action_dispatch(model, &input.actors, &rows, &fills);
    let check = action_dispatch(model, &input.actors, &rows, &evidence);
    Ok(quote! {
        #item
        #[doc = "Typed requests constructed by this binding's admitted registry."]
        #visibility enum #request_sum { #(#request_variants),* }
        impl ::umpire_runtime::capability::Request for #request_sum {
            fn pack_request(&self) -> ::umpire_runtime::prost_types::Any { #pack }
        }
        impl ::umpire_runtime::capability::Fill<#model::Action> for #ty {
            type World = #model::World;
            type Inputs = #inputs;
            type Request = #request_sum;
            #[allow(deprecated, reason = "descriptor admission validates deprecated wire fields")]
            fn fill(&self, action: &#model::Action, world: &Self::World, inputs: &Self::Inputs, identity: &Self::Id) -> Result<Self::Request, ::umpire_runtime::protocol::Refusal> {
                let _ = (world, inputs, identity);
                #fill
            }
        }
        impl #runtime::registry::Registry<#model::Action> for #ty {
            fn row(&self, action: &#model::Action) -> #runtime::registry::ActionRow { #classify }
            #[allow(deprecated, reason = "descriptor admission validates deprecated wire fields")]
            fn check_registered(&self, observation: &::umpire_runtime::prost_types::Any, path: &[&::umpire_runtime::Evidence<#model::World>]) -> Result<::umpire_runtime::EvidenceMatch, ::umpire_runtime::protocol::Refusal> {
                let __observed = #runtime::support::Observed::new(observation)?;
                #(#admission)*
                let Some(__edge) = path.last() else { return Ok(::umpire_runtime::EvidenceMatch::Reject); };
                let action = &__edge.action;
                #check
            }
        }
        #configuration
    })
}

fn hook_contracts(
    item: &mut ItemImpl,
    rows: &[Row],
    entity: Option<&Options>,
    read: Option<&Options>,
) {
    let hook_names: BTreeSet<_> = rows
        .iter()
        .map(|row| &row.options)
        .chain(entity)
        .chain(read)
        .flat_map(|options| options.values.iter())
        .filter(|(key, _)| {
            [
                "fill",
                "evidence",
                "admit",
                "make_receipt",
                "correlate",
                "project",
                "finality",
                "continue_polling",
            ]
            .contains(&key.as_str())
        })
        .filter_map(|(_, (_, tokens))| syn::parse2::<Ident>(tokens.clone()).ok())
        .map(|name| name.to_string())
        .collect();
    let named = |options: &Options, key: &str| {
        options
            .values
            .get(key)
            .and_then(|(_, tokens)| syn::parse2::<Ident>(tokens.clone()).ok())
            .map(|name| name.to_string())
    };
    let rpc_evidence: BTreeSet<_> = rows
        .iter()
        .filter(|row| row.event.is_none())
        .filter_map(|row| named(&row.options, "evidence"))
        .collect();
    let read_policy: BTreeSet<_> = ["finality", "continue_polling"]
        .into_iter()
        .filter_map(|key| read.and_then(|options| named(options, key)))
        .collect();
    for member in &mut item.items {
        let syn::ImplItem::Fn(function) = member else {
            continue;
        };
        let name = function.sig.ident.to_string();
        if !hook_names.contains(&name) {
            continue;
        }
        function.attrs.push(syn::parse_quote!(#[allow(clippy::unnecessary_wraps, clippy::unused_self, reason = "registry hooks share typed binding contracts")]));
        if rpc_evidence.contains(&name) {
            function.attrs.push(syn::parse_quote!(#[allow(clippy::needless_pass_by_value, reason = "RPC evidence hooks receive owned canonical messages")]));
        }
        if read_policy.contains(&name) {
            function.attrs.push(syn::parse_quote!(#[allow(clippy::trivially_copy_pass_by_ref, reason = "read policy hooks share the typed projection by reference")]));
        }
    }
}

struct Dispatches {
    request_variants: Vec<TokenStream>,
    pack_arms: Vec<TokenStream>,
    fills: Vec<TokenStream>,
    classifications: Vec<TokenStream>,
    admission: Vec<TokenStream>,
    evidence: Vec<TokenStream>,
}

fn dispatches(
    rows: &[Row],
    schema: &Schema,
    runtime: &Path,
    request_sum: &Ident,
) -> syn::Result<Dispatches> {
    let mut request_variants = Vec::new();
    let mut pack_arms = Vec::new();
    let mut fills = Vec::new();
    let mut classifications = Vec::new();
    let mut admission = Vec::new();
    let mut methods = BTreeSet::new();
    let mut evidence = Vec::new();
    for (index, row) in rows.iter().enumerate() {
        let actor = row.actor.to_string();
        let variant = row.variant.ident.to_string();
        let span = row.variant.span();
        let check: Ident = row.options.required("evidence", span)?;
        let kind = if let Some(reason) = &row.event {
            fills.push(quote!(Err(::umpire_runtime::protocol::Refusal::unsupported(
                "environmental action has no RPC request"
            ))));
            evidence.push(quote_spanned!(Span::call_site().located_at(check.span())=> self.#check(&__observed, path)));
            quote!(#runtime::registry::ActionKind::Event { reason: #reason })
        } else {
            let rpc = rpc(&row.options, schema, span)?;
            let direction: Ident = row.options.required("direction", span)?;
            let direction = match direction.to_string().as_str() {
                "client" => quote!(#runtime::registry::Direction::Client),
                "worker" => quote!(#runtime::registry::Direction::Worker),
                _ => {
                    return Err(syn::Error::new(
                        direction.span(),
                        "direction must be client or worker",
                    ));
                }
            };
            let fill: Ident = row.options.required("fill", span)?;
            let sum_variant = format_ident!("Action{index}");
            let request_doc = format!("Canonical descriptor request for {actor}::{variant}.");
            let Rpc { method, request, response, request_guards, response_guards, .. } = &rpc;
            let witnesses = rpc.witnesses();
            if methods.insert(method.clone()) {
                admission.push(quote! {
                    if __observed.method == #method {
                        let _: #request = __observed.request()?.ok_or_else(|| ::umpire_runtime::protocol::Refusal::malformed("registered RPC evidence requires its actual request"))?;
                        if __observed.code()?.is_none() {
                            let _: #response = __observed.response()?;
                        }
                    }
                });
            }
            request_variants.push(quote!(#[doc = #request_doc] #sum_variant(#request)));
            pack_arms.push(quote!(Self::#sum_variant(value) => <::umpire_runtime::prost_types::Any as ::umpire_runtime::AnyMessage>::pack(value)));
            let filled = quote_spanned!(Span::call_site().located_at(fill.span())=> self.#fill(__actor, world, inputs, identity)?);
            let checked = quote_spanned!(Span::call_site().located_at(check.span())=> self.#check(__request, __response, path));
            fills.push(quote! {
                #witnesses
                let __request: #request = #filled;
                #request_guards
                Ok(#request_sum::#sum_variant(__request))
            });
            evidence.push(quote! {
                #witnesses
                if __observed.method != #method { return Ok(::umpire_runtime::EvidenceMatch::Reject); }
                let __request: #request = __observed.request()?.ok_or_else(|| ::umpire_runtime::protocol::Refusal::malformed("registered RPC evidence requires its actual request"))?;
                #request_guards
                let __response: Result<#response, i32> = if let Some(code) = __observed.code()? { Err(code) } else {
                    let __response: #response = __observed.response()?;
                    #response_guards
                    Ok(__response)
                };
                #checked
            });
            quote!(#runtime::registry::ActionKind::Rpc { method: #method, direction: #direction })
        };
        classifications.push(
            quote!(#runtime::registry::ActionRow { actor: #actor, variant: #variant, kind: #kind }),
        );
    }
    Ok(Dispatches { request_variants, pack_arms, fills, classifications, admission, evidence })
}

fn hook(name: &Ident) -> TokenStream {
    quote_spanned!(Span::call_site().located_at(name.span())=> Self::#name)
}

fn configurations(
    item: &ItemImpl,
    model: &Path,
    runtime: &Path,
    schema: &Schema,
    entity: Option<&Options>,
    read: Option<&Options>,
) -> syn::Result<TokenStream> {
    let Some(entity) = entity else {
        if read.is_some() {
            return Err(syn::Error::new(
                item.span(),
                "read declaration requires entity configuration",
            ));
        }
        return Ok(TokenStream::new());
    };
    let span = item.span();
    entity.admit(
        &["rpc", "request", "response", "receipt", "admit", "make_receipt", "correlate"],
        span,
    )?;
    let ty = &item.self_ty;
    let receipt: Type = entity.required("receipt", span)?;
    let admit: Ident = entity.required("admit", span)?;
    let make_receipt: Ident = entity.required("make_receipt", span)?;
    let correlate: Ident = entity.required("correlate", span)?;
    let admit = hook(&admit);
    let make_receipt = hook(&make_receipt);
    let correlate = hook(&correlate);
    let entity_rpc = rpc(entity, schema, span)?;
    let request = &entity_rpc.request;
    let response = &entity_rpc.response;
    let witnesses = entity_rpc.witnesses();
    let value = entity_rpc.configuration(runtime);
    let read = read.map(|options| -> syn::Result<_> {
        options.admit(&["rpc", "request", "response", "target", "fill", "correlate", "project", "finality", "continue_polling"], span)?;
        let rpc = rpc(options, schema, span)?;
        let request = &rpc.request;
        let response = &rpc.response;
        let target: Type = options.required("target", span)?;
        let fill: Ident = options.required("fill", span)?;
        let correlate: Ident = options.required("correlate", span)?;
        let project: Ident = options.required("project", span)?;
        let finality: Ident = options.required("finality", span)?;
        let polling: Ident = options.required("continue_polling", span)?;
        let fill = hook(&fill);
        let correlate = hook(&correlate);
        let project = hook(&project);
        let finality = hook(&finality);
        let polling = hook(&polling);
        let witnesses = rpc.witnesses();
        let value = rpc.configuration(runtime);
        Ok(quote! {
            pub fn read() -> #runtime::registry::Read<Self, #model::World, #receipt, #request, #response, #target> {
                #witnesses
                #runtime::registry::Read { rpc: #value, fill: #fill, correlate: #correlate, project: #project, finality: #finality, continue_polling: #polling }
            }
        })
    }).transpose()?;
    Ok(quote! {
        impl #ty {
            pub fn entity() -> #runtime::registry::Entity<Self, #receipt, #request, #response> {
                #witnesses
                #runtime::registry::Entity { bootstrap: #value, admit: #admit, receipt: #make_receipt, correlate: #correlate }
            }
            #read
        }
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn missing_action_metadata_is_located_on_the_actual_variant() -> syn::Result<()> {
        let input: Input = syn::parse_str(
            "model { sample } actors { enum Client { Missing } } body { #[registry(inputs = ())] impl Binding {} }",
        )?;
        let error = rows(&input.actors)
            .err()
            .ok_or_else(|| syn::Error::new(Span::call_site(), "missing metadata was admitted"))?;
        assert_eq!(error.to_string(), "action variant requires one temporal declaration");
        assert_eq!(error.span().start().column, 40);
        Ok(())
    }

    #[test]
    fn invalid_variant_options_are_refused_at_the_declared_option() -> syn::Result<()> {
        for (metadata, expected) in [
            (
                "#[metadata(temporal(event = \"elapsed\", evidence = matches, unknown = value))]",
                "unknown temporal action option",
            ),
            (
                "#[metadata(temporal(event = \"elapsed\", event = \"again\", evidence = matches))]",
                "duplicate temporal action option",
            ),
            (
                "#[metadata(temporal(event = \"elapsed\", evidence = matches, rpc = Service::Call))]",
                "event declaration cannot contain RPC options",
            ),
            (
                "#[metadata(temporal(event = \"elapsed\", evidence = matches))] #[metadata(temporal(event = \"again\", evidence = matches))]",
                "duplicate temporal action declaration",
            ),
        ] {
            let source = format!(
                "model {{ sample }} actors {{ enum Clock {{ {metadata} Tick }} }} body {{ #[registry(inputs = ())] impl Binding {{}} }}"
            );
            let input: Input = syn::parse_str(&source)?;
            let error = rows(&input.actors)
                .err()
                .ok_or_else(|| syn::Error::new(Span::call_site(), "invalid option was admitted"))?;
            assert!(error.to_string().contains(expected), "{error}");
            assert!(error.span().start().column >= source.find("#[metadata").unwrap_or(0));
        }
        Ok(())
    }

    #[test]
    fn entity_and_read_configuration_refuses_missing_conflicting_and_unknown_options()
    -> syn::Result<()> {
        let schema = Schema::default();
        let runtime = syn::parse_str("temporal")?;
        for (attributes, expected) in [
            ("#[registry(inputs = (), inputs = ())]", "duplicate temporal action option"),
            ("#[registry(inputs = ())] #[entity()] #[entity()]", "duplicate entity declaration"),
            ("#[registry(inputs = ())] #[read()] #[read()]", "duplicate read declaration"),
            (
                "#[registry(inputs = ())] #[read()]",
                "read declaration requires entity configuration",
            ),
            (
                "#[registry(inputs = ())] #[entity(unknown = hook)]",
                "unknown temporal action option",
            ),
            (
                "#[registry(inputs = ())] #[entity()]",
                "missing temporal declaration option: receipt",
            ),
        ] {
            let source =
                format!("model {{ sample }} actors {{}} body {{ {attributes} impl Binding {{}} }}");
            let error = expand(source.parse()?, &runtime, &schema)
                .err()
                .ok_or_else(|| syn::Error::new(Span::call_site(), "invalid config admitted"))?;
            assert!(error.to_string().contains(expected), "{error}");
        }
        Ok(())
    }

    #[test]
    fn metadata_callback_scope_preserves_generated_enum_alias_coverage() -> syn::Result<()> {
        let source = "mod binding { use api::Status as Status; metadata! { model { sample } actors {} body { #[registry(inputs = ())] impl Binding { fn status(value: Status) { match value { Status::Ready => (), _ => () } } } } } }";
        let file = syn::parse_file(source)?;
        let syn::Item::Mod(module) = &file.items[0] else {
            return Err(syn::Error::new(Span::call_site(), "missing module"));
        };
        let (_, items) = module
            .content
            .as_ref()
            .ok_or_else(|| syn::Error::new(Span::call_site(), "missing module body"))?;
        let syn::Item::Macro(mac) = &items[1] else {
            return Err(syn::Error::new(Span::call_site(), "missing callback"));
        };
        let input = mac.mac.tokens.clone();
        let location = source_span(input.clone())?.start();
        let mut schema = Schema::default();
        schema.enums.insert(
            "test.Status".into(),
            crate::realization::Enumeration {
                rust_type: "api::Status".into(),
                variants: vec!["api::Status::Ready".into()],
            },
        );
        let error =
            expand_in_source(input, &syn::parse_str("temporal")?, &schema, source, location)
                .err()
                .ok_or_else(|| {
                    syn::Error::new(Span::call_site(), "generated enum wildcard was admitted")
                })?;
        assert!(
            error.to_string().contains("generated enum matches must name every variant"),
            "{error}"
        );
        assert_eq!(
            error.span().start().column,
            source
                .find("_ =>")
                .ok_or_else(|| syn::Error::new(Span::call_site(), "missing wildcard"))?
        );
        Ok(())
    }

    #[test]
    fn local_action_payload_bindings_are_not_external_variants_from_a_glob() -> syn::Result<()> {
        let source = "use foreign::*; metadata! { model { sample } actors {} body { #[registry(inputs = ())] impl Binding { fn fill(action: &sample::Worker) { let flag = match action { sample::Worker::Notify { flag } | sample::Worker::RetryNotify(flag) => flag }; } } } }";
        let file = syn::parse_file(source)?;
        let syn::Item::Macro(mac) = &file.items[1] else {
            return Err(syn::Error::new(Span::call_site(), "missing callback"));
        };
        let input = mac.mac.tokens.clone();
        let location = source_span(input.clone())?.start();
        expand_in_source(
            input,
            &syn::parse_str("temporal")?,
            &Schema::default(),
            source,
            location,
        )?;
        Ok(())
    }

    #[test]
    fn rpc_type_witnesses_are_located_at_the_authored_type_options() -> syn::Result<()> {
        let source = "request = DeclaredRequest, response = DeclaredResponse";
        let options: Options = syn::parse_str(source)?;
        let rpc = Rpc {
            method: "/test.Service/Call".into(),
            request: syn::parse_str("CanonicalRequest")?,
            response: syn::parse_str("CanonicalResponse")?,
            declared_request: options.required("request", Span::call_site())?,
            declared_response: options.required("response", Span::call_site())?,
            request_guards: TokenStream::new(),
            response_guards: TokenStream::new(),
        };
        let witnesses = rpc.witnesses();
        let block: syn::Block = syn::parse2(quote!({ #witnesses }))?;
        assert_eq!(block.stmts.len(), 2);
        for (statement, name) in block.stmts.iter().zip(["DeclaredRequest", "DeclaredResponse"]) {
            let syn::Stmt::Local(local) = statement else {
                return Err(syn::Error::new(statement.span(), "missing witness"));
            };
            let value = local
                .init
                .as_ref()
                .ok_or_else(|| syn::Error::new(local.span(), "missing coercion"))?;
            let syn::Expr::Closure(closure) = value.expr.as_ref() else {
                return Err(syn::Error::new(value.expr.span(), "missing coercion closure"));
            };
            assert_eq!(
                closure.body.span().start().column,
                source
                    .find(name)
                    .ok_or_else(|| syn::Error::new(Span::call_site(), "missing source type"))?
            );
        }
        Ok(())
    }
}
