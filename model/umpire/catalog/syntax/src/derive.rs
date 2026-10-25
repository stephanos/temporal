//! The shared attribute grammar for compiler expansion and descriptor admission.

use darling::{FromMeta, ast::NestedMeta, util::Flag};
use heck::ToUpperCamelCase;
use proc_macro2::Span;
use quote::format_ident;
use syn::{
    Attribute, DeriveInput, Expr, Ident, Lit, LitStr, Path, Token, Type, ext::IdentExt,
    parse::ParseStream, punctuated::Punctuated,
};

use crate::{Catalog, Expected, Field, Message, Projection, Rule, Tag};

/// A checked mirror struct, parsed identically by the derive and build script.
#[derive(Debug)]
pub struct DeriveCatalog {
    /// Caller-owned inspection and limit context.
    pub context: Path,
    /// Author's mirror type.
    pub name: Ident,
    /// Concrete protobuf message type.
    pub ty: Path,
    /// Fully qualified protobuf message name.
    pub proto: LitStr,
    /// Request expectation policy.
    pub request: bool,
    /// Supplemental intents do not own exhaustive coverage or inspection.
    pub supplemental: bool,
    /// Message description from Rust doc comments.
    pub description: String,
    /// Supplemental mirror used only when inspecting a response.
    pub response_checks: Option<Path>,
    /// Field declarations in source order.
    pub fields: Vec<DeriveField>,
    /// Presence rules.
    pub rules: Vec<DeriveRule>,
}

/// One field and its attribute intents.
#[derive(Debug, Clone)]
pub struct DeriveField {
    /// Actual protobuf field name.
    pub name: Ident,
    /// Resolved marker type as authored.
    pub ty: Type,
    /// Field documentation.
    pub description: String,
    /// Attribute declarations.
    pub attributes: ApiAttributes,
}

/// Presence rule with an expectation shared by its attribute line.
#[derive(Debug)]
pub struct DeriveRule {
    /// Presence operator.
    pub operator: Ident,
    /// Referenced protobuf source names.
    pub fields: Vec<Ident>,
    /// Expected server behavior.
    pub expected: Option<ApiExpectation>,
}

/// Expected behavior in the attribute surface.
#[derive(Debug, Clone)]
pub enum ApiExpectation {
    /// Named status and optional message regex.
    Rejects(Ident, Option<LitStr>),
    /// Any non-OK status.
    AnyStatus,
    /// Server accepts the invalid value.
    Accepts,
    /// Cannot be observed with a request probe.
    Unobservable,
}

/// Parsed helper attributes; shared with descriptor and compiler consumers.
#[derive(Debug, Clone, Default)]
pub struct ApiAttributes {
    /// Caller-owned context type.
    pub context: Option<Path>,
    /// Root of generated protobuf modules.
    pub api: Option<Path>,
    /// Typed supplemental response mirror.
    pub response_checks: Option<Path>,
    /// Message direction and protobuf name.
    pub direction: Option<(bool, LitStr)>,
    /// Explicit prost type override.
    pub proto_type: Option<Path>,
    /// Excludes a second mirror from exhaustive ownership.
    pub supplemental: bool,
    /// Constraint expressions, including dynamic config constants.
    pub constraints: Vec<(Ident, Expr)>,
    /// Builtin annotation consequences.
    pub tags: Vec<Tag>,
    /// Server expectation.
    pub expected: Option<ApiExpectation>,
    /// Presence rules declared on this attribute line.
    pub rules: Vec<(Ident, Vec<Ident>)>,
    /// Module with typed view/set functions.
    pub via: Option<Path>,
    /// Exhaustive oneof alternatives and protobuf names.
    pub variants: Vec<(Ident, LitStr)>,
    /// Supplemental kind intents on this field.
    pub also: Vec<ApiAttributes>,
    /// Supplemental kind name.
    pub also_kind: Option<Ident>,
    options: Vec<NestedMeta>,
    expected_span: Option<Span>,
}

#[derive(Debug, FromMeta)]
struct ApiOptions {
    context: Option<Path>,
    api: Option<Path>,
    response_checks: Option<Path>,
    request: Option<LitStr>,
    response: Option<LitStr>,
    #[darling(rename = "type")]
    proto_type: Option<Path>,
    supplemental: Flag,
    via: Option<Path>,
}

impl ApiAttributes {
    fn apply_options(&mut self) -> syn::Result<()> {
        let options = ApiOptions::from_list(&self.options)?;
        self.context = options.context;
        self.api = options.api;
        self.response_checks = options.response_checks;
        self.proto_type = options.proto_type;
        self.supplemental = options.supplemental.is_present();
        self.via = options.via;
        self.direction = match (options.request, options.response) {
            (Some(request), Some(response)) => {
                let span = if request.span().start() > response.span().start() {
                    request.span()
                } else {
                    response.span()
                };
                return Err(syn::Error::new(span, "duplicate message direction"));
            }
            (Some(name), None) => Some((true, name)),
            (None, Some(name)) => Some((false, name)),
            (None, None) => None,
        };
        Ok(())
    }
}

fn set_expectation(
    slot: &mut Option<ApiExpectation>,
    value: ApiExpectation,
    span: Span,
) -> syn::Result<()> {
    if slot.is_some() {
        return Err(syn::Error::new(span, "duplicate expectation"));
    }
    *slot = Some(value);
    Ok(())
}

fn names(input: ParseStream<'_>) -> syn::Result<Vec<Ident>> {
    let body;
    syn::parenthesized!(body in input);
    Ok(Punctuated::<Ident, Token![,]>::parse_terminated(&body)?.into_iter().collect())
}

impl syn::parse::Parse for ApiAttributes {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let mut result = Self::default();
        while !input.is_empty() {
            let key = input.call(Ident::parse_any)?;
            let span = key.span();
            match key.to_string().as_str() {
                "request" | "response" => {
                    let eq_token = input.parse::<Token![=]>()?;
                    let value: LitStr = input.parse()?;
                    result.options.push(NestedMeta::Meta(syn::Meta::NameValue(
                        syn::MetaNameValue {
                            path: key.into(),
                            eq_token,
                            value: syn::parse_quote!(#value),
                        },
                    )));
                }
                "context" | "api" | "type" | "via" | "response_checks" => {
                    let eq_token = input.parse::<Token![=]>()?;
                    let value: Path = input.parse()?;
                    result.options.push(NestedMeta::Meta(syn::Meta::NameValue(
                        syn::MetaNameValue {
                            path: key.into(),
                            eq_token,
                            value: Expr::Path(syn::ExprPath {
                                attrs: Vec::new(),
                                qself: None,
                                path: value,
                            }),
                        },
                    )));
                }
                "supplemental" => {
                    result.options.push(NestedMeta::Meta(syn::Meta::Path(key.into())));
                }
                "rejects" => {
                    let expected = parse_rejection(input, span)?;
                    set_expectation(&mut result.expected, expected, span)?;
                    result.expected_span = Some(span);
                }
                "accepts" | "unobservable" => {
                    let expected = if key == "accepts" {
                        ApiExpectation::Accepts
                    } else {
                        ApiExpectation::Unobservable
                    };
                    set_expectation(&mut result.expected, expected, span)?;
                    result.expected_span = Some(span);
                }
                "required" | "requires" | "excludes" | "any_of" | "one_of" => {
                    result.rules.push((key, names(input)?));
                }
                "oneof" => {
                    result.variants.extend(oneof_variants(input)?);
                }
                "also" => {
                    let attributes = parse_also(input)?;
                    result.also.push(attributes);
                }
                "ignored" => {
                    input.parse::<Token![=]>()?;
                    result.tags.push(Tag {
                        name: Ident::new("Ignored", span),
                        argument: Some(Lit::Str(input.parse()?)),
                    });
                }
                "decided" | "observed" | "idempotency_key" | "monotone" | "immutable" => {
                    let name = match key.to_string().as_str() {
                        "decided" => "Decided",
                        "observed" => "Observed",
                        "idempotency_key" => "IdempotencyKey",
                        "monotone" => "Monotone",
                        _ => "Immutable",
                    };
                    result.tags.push(Tag { name: Ident::new(name, span), argument: None });
                }
                "cached" | "eventually_consistent" => {
                    input.parse::<Token![=]>()?;
                    result.tags.push(Tag {
                        name: Ident::new(
                            if key == "cached" { "Cached" } else { "EventuallyConsistent" },
                            span,
                        ),
                        argument: Some(input.parse()?),
                    });
                }
                "min_len" | "max_len" | "min_bytes" | "max_bytes" | "prefix" | "suffix"
                | "contains" | "gt" | "gte" | "lt" | "lte" | "min_items" | "max_items"
                | "unique" | "defined_only" => {
                    input.parse::<Token![=]>()?;
                    result.constraints.push((key, input.parse()?));
                }
                _ => return Err(syn::Error::new(span, "unknown catalog attribute")),
            }
            if !input.is_empty() {
                input.parse::<Token![,]>()?;
            }
        }
        result.apply_options()?;
        Ok(result)
    }
}

fn oneof_variants(input: ParseStream<'_>) -> syn::Result<Vec<(Ident, LitStr)>> {
    let body;
    syn::parenthesized!(body in input);
    let mut variants = Vec::new();
    while !body.is_empty() {
        let variant = body.parse()?;
        body.parse::<Token![=]>()?;
        variants.push((variant, body.parse()?));
        if !body.is_empty() {
            body.parse::<Token![,]>()?;
        }
    }
    Ok(variants)
}

fn attributes(attrs: &[Attribute]) -> syn::Result<Vec<ApiAttributes>> {
    attrs
        .iter()
        .filter(|attr| attr.path().is_ident("rule") || attr.path().is_ident("mirror"))
        .map(|attr| {
            let parsed: ApiAttributes = attr.parse_args()?;
            let mapping = parsed.context.is_some()
                || parsed.api.is_some()
                || parsed.direction.is_some()
                || parsed.proto_type.is_some()
                || parsed.supplemental
                || parsed.response_checks.is_some()
                || parsed.via.is_some()
                || !parsed.tags.is_empty()
                || !parsed.variants.is_empty()
                || !parsed.also.is_empty();
            let rule = parsed.expected.is_some()
                || !parsed.constraints.is_empty()
                || !parsed.rules.is_empty();
            if attr.path().is_ident("rule") && mapping {
                return Err(syn::Error::new_spanned(
                    attr,
                    "mapping attributes belong in #[mirror]",
                ));
            }
            if attr.path().is_ident("mirror") && rule {
                return Err(syn::Error::new_spanned(
                    attr,
                    "validation attributes belong in #[rule]",
                ));
            }
            Ok(parsed)
        })
        .collect()
}

fn merge(mut lines: Vec<ApiAttributes>) -> syn::Result<ApiAttributes> {
    let mut result = ApiAttributes::default();
    for mut line in lines.drain(..) {
        result.options.append(&mut line.options);
        let expected_span = line.expected_span.unwrap_or_else(Span::call_site);
        if let Some(value) = line.expected.take() {
            set_expectation(&mut result.expected, value, expected_span)?;
            result.expected_span = Some(expected_span);
        }
        result.constraints.extend(line.constraints);
        result.tags.extend(line.tags);
        result.rules.extend(line.rules);
        result.variants.extend(line.variants);
        result.also.extend(line.also);
    }
    result.apply_options()?;
    Ok(result)
}

fn documentation(attrs: &[Attribute]) -> String {
    attrs
        .iter()
        .filter_map(|attr| {
            if !attr.path().is_ident("doc") {
                return None;
            }
            if let syn::Meta::NameValue(meta) = &attr.meta
                && let Expr::Lit(value) = &meta.value
                && let Lit::Str(text) = &value.lit
            {
                return Some(text.value().trim().to_owned());
            }
            None
        })
        .collect::<Vec<_>>()
        .join("\n")
}

/// Last path segment of a concrete marker type.
pub fn marker(ty: &Type) -> syn::Result<&syn::PathSegment> {
    if let Type::Path(ty) = ty {
        return ty
            .path
            .segments
            .last()
            .ok_or_else(|| syn::Error::new_spanned(ty, "expected a kind type"));
    }
    Err(syn::Error::new_spanned(ty, "expected a kind type"))
}

impl DeriveCatalog {
    /// Parses and admits a mirror using the shared catalog admission rules.
    pub fn parse(input: &DeriveInput) -> syn::Result<Self> {
        Self::parse_with(input, syn::parse_quote!(crate::CatalogContext), &syn::parse_quote!(crate))
    }

    /// Parse with consumer defaults; explicit mirror parameters take precedence.
    pub fn parse_with(input: &DeriveInput, context: Path, api: &Path) -> syn::Result<Self> {
        if !input.generics.params.is_empty() {
            return Err(syn::Error::new_spanned(
                &input.generics,
                "catalog mirrors cannot be generic",
            ));
        }
        let item_lines = attributes(&input.attrs)?;
        let mut rules = Vec::new();
        for line in &item_lines {
            for (operator, fields) in &line.rules {
                rules.push(DeriveRule {
                    operator: operator.clone(),
                    fields: fields.clone(),
                    expected: line.expected.clone(),
                });
            }
        }
        let item = merge(
            item_lines
                .into_iter()
                .map(|mut line| {
                    if !line.rules.is_empty() {
                        line.expected = None;
                        line.rules.clear();
                    }
                    line
                })
                .collect(),
        )?;
        if !item.constraints.is_empty()
            || !item.tags.is_empty()
            || item.via.is_some()
            || !item.variants.is_empty()
            || !item.also.is_empty()
            || item.expected.is_some()
        {
            return Err(syn::Error::new_spanned(
                input,
                "field attributes cannot be used on the catalog item",
            ));
        }
        let (request, proto) = item.direction.clone().ok_or_else(|| {
            syn::Error::new_spanned(
                &input.ident,
                "catalog requires #[mirror(request = ...)] or #[mirror(response = ...)]",
            )
        })?;
        if proto.value().trim().is_empty() {
            return Err(syn::Error::new(proto.span(), "message requires a nonempty name"));
        }
        let ty = item
            .proto_type
            .clone()
            .map_or_else(|| inferred_type(&proto, item.api.as_ref().unwrap_or(api)), Ok)?;
        let syn::Data::Struct(data) = &input.data else {
            return Err(syn::Error::new_spanned(input, "Catalog requires a mirror struct"));
        };
        let syn::Fields::Named(fields) = &data.fields else {
            return Err(syn::Error::new_spanned(&data.fields, "Catalog requires named fields"));
        };
        let parsed = fields
            .named
            .iter()
            .map(|field| parse_field(field, request))
            .collect::<syn::Result<_>>()?;
        let result = Self {
            context: item.context.clone().unwrap_or(context),
            name: input.ident.clone(),
            ty,
            proto,
            request,
            supplemental: item.supplemental,
            description: documentation(&input.attrs),
            response_checks: item.response_checks.clone(),
            fields: parsed,
            rules,
        };
        result.admit()?;
        Ok(result)
    }

    fn admit(&self) -> syn::Result<()> {
        self.admission_catalog()?.admit()?;
        for field in &self.fields {
            for additional in &field.attributes.also {
                let extra = Self::supplemental_field(field, additional)?;
                Self {
                    context: self.context.clone(),
                    name: self.name.clone(),
                    ty: self.ty.clone(),
                    proto: self.proto.clone(),
                    request: self.request,
                    supplemental: true,
                    description: String::new(),
                    response_checks: None,
                    fields: vec![extra],
                    rules: Vec::new(),
                }
                .admission_catalog()?
                .admit()?;
            }
        }
        Ok(())
    }

    pub(crate) fn supplemental_field(
        field: &DeriveField,
        additional: &ApiAttributes,
    ) -> syn::Result<DeriveField> {
        let kind = additional
            .also_kind
            .as_ref()
            .ok_or_else(|| syn::Error::new(field.name.span(), "missing supplemental kind"))?;
        let mut attributes = additional.clone();
        if attributes.tags.is_empty() {
            attributes.tags.clone_from(&field.attributes.tags);
        }
        Ok(DeriveField {
            name: field.name.clone(),
            ty: syn::parse_quote!(::umpire_catalog::kinds::#kind),
            description: field.description.clone(),
            attributes,
        })
    }

    fn admission_fields(&self) -> syn::Result<(Vec<Field>, Vec<crate::Enum>)> {
        let mut fields = Vec::new();
        let mut enums = Vec::new();
        for field in &self.fields {
            let attrs = &field.attributes;
            let actual = marker(semantic_type(&field.ty)?)?.ident.to_string();
            let kind = match actual.as_str() {
                "Ignored" | "Message" | "OneOf" | "Map" | "Vec" => "Bytes",
                "RawDuration" => "Duration",
                other => other,
            };
            let kind = Ident::new(kind, field.name.span());
            let enumeration = if actual == "Enum" {
                let factory = format_ident!("{}_enum", field.name.unraw());
                enums.push(crate::Enum {
                    factory: factory.clone(),
                    ty: syn::parse_quote!(UnusedEnum),
                    proto: self.proto.clone(),
                    variants: Vec::new(),
                });
                Some(factory)
            } else {
                None
            };
            let constraints: Vec<_> = attrs
                .constraints
                .iter()
                .map(|(name, value)| {
                    let literal = if let Expr::Lit(value) = value {
                        value.lit.clone()
                    } else {
                        match name.to_string().as_str() {
                            "prefix" | "suffix" | "contains" => syn::parse_quote!(""),
                            "unique" => syn::parse_quote!(true),
                            _ => syn::parse_quote!(0),
                        }
                    };
                    (name.clone(), literal)
                })
                .collect();
            let tags = attrs
                .tags
                .iter()
                .map(|tag| Tag { name: tag.name.clone(), argument: tag.argument.clone() })
                .collect();
            let variants = attrs
                .variants
                .iter()
                .map(|(name, proto)| crate::Variant {
                    path: syn::parse_quote!(UnusedOneof::#name),
                    proto: proto.clone(),
                })
                .collect();
            let ignored = attrs.tags.iter().any(|tag| tag.name == "Ignored");
            if actual == "Ignored" && !ignored {
                return Err(syn::Error::new(
                    field.name.span(),
                    "Ignored requires a nonempty explicit ignored reason",
                ));
            }
            let mut admitted = Field {
                opaque: ignored,
                name: field.name.clone(),
                kind,
                constraints,
                tags,
                enumeration,
                projection: if actual == "OneOf" && !ignored {
                    Some(Projection {
                        ty: syn::parse_quote!(Vec<u8>),
                        get: syn::parse_quote!(|_| Vec::new()),
                        set: syn::parse_quote!(|_, _| ()),
                    })
                } else {
                    None
                },
                variants,
                expected: admission_expected(attrs.expected.as_ref())?,
            };
            if !attrs.rules.is_empty() && !crate::parse::field_is_constrained(&admitted) {
                admitted.expected = None;
            }
            fields.push(admitted);
        }
        Ok((fields, enums))
    }

    /// Projects the mirror to the shared admission and coverage representation.
    pub fn admission_catalog(&self) -> syn::Result<Catalog> {
        let (fields, enums) = self.admission_fields()?;
        let rules = self
            .rules
            .iter()
            .map(|rule| {
                Ok(Rule {
                    operator: rule.operator.clone(),
                    fields: rule.fields.clone(),
                    expected: admission_expected(rule.expected.as_ref())?,
                })
            })
            .collect::<syn::Result<_>>()?;
        let mut messages = vec![Message {
            request: self.request,
            factory: self.name.clone(),
            ty: self.ty.clone(),
            proto: self.proto.clone(),
            fields,
            rules,
        }];
        for field in &self.fields {
            if marker(&field.ty)?.ident == "Map" {
                let entry = map_entry_proto(&self.proto, &field.name);
                messages.push(Message {
                    request: false,
                    factory: format_ident!("{}_entry", field.name.unraw()),
                    ty: syn::parse_quote!(::umpire_catalog::views::MetadataEntry),
                    proto: LitStr::new(&entry, field.name.span()),
                    fields: ["key", "value"]
                        .map(|name| Field {
                            opaque: false,
                            name: Ident::new(name, field.name.span()),
                            kind: Ident::new("Bytes", field.name.span()),
                            constraints: Vec::new(),
                            tags: Vec::new(),
                            enumeration: None,
                            projection: None,
                            variants: Vec::new(),
                            expected: None,
                        })
                        .into_iter()
                        .collect(),
                    rules: Vec::new(),
                });
            }
        }
        Ok(Catalog {
            vis: syn::Visibility::Inherited,
            name: self.name.clone(),
            messages,
            enums,
            services: Vec::new(),
            actions: Vec::new(),
            tags: Vec::new(),
        })
    }
}

fn admission_expected(expected: Option<&ApiExpectation>) -> syn::Result<Option<Expected>> {
    expected
        .map(|expectation| {
            let outcome = match expectation {
                ApiExpectation::Accepts => "Accepts",
                ApiExpectation::Unobservable => "Unobservable",
                _ => "Rejects",
            };
            Ok(Expected {
                outcome: Ident::new(outcome, Span::call_site()),
                reject: if outcome == "Rejects" {
                    Some((syn::parse_quote!(3), syn::parse_quote!(""), None))
                } else {
                    None
                },
            })
        })
        .transpose()
}

fn inferred_type(proto: &LitStr, api: &Path) -> syn::Result<Path> {
    if crate::descriptor::well_known_primitive(&proto.value()).is_some() {
        return Err(syn::Error::new(
            proto.span(),
            "primitive well-known types have no named Rust fields to mirror",
        ));
    }
    let name = proto.value();
    let mut parts: Vec<_> = name.split('.').collect();
    let message = parts
        .pop()
        .ok_or_else(|| syn::Error::new(proto.span(), "missing protobuf message name"))?;
    let message = crate::naming::upper_camel(message);
    let package =
        parts.iter().map(|part| crate::naming::snake(part)).collect::<Vec<_>>().join("::");
    let path = if package == "google::protobuf" {
        format!("::umpire_runtime::prost_types::{message}")
    } else {
        {
            let root = quote::quote!(#api).to_string();
            format!("{root}::{package}::{message}")
        }
    };
    syn::parse_str(&path).map_err(|_| syn::Error::new(proto.span(), "invalid protobuf name"))
}

/// Stable snake-case name used for catalog factories.
pub fn factory_name(name: &Ident) -> Ident {
    format_ident!("{}", crate::naming::snake(&name.unraw().to_string()))
}

pub(crate) fn map_entry_proto(parent: &LitStr, field: &Ident) -> String {
    let camel = field.unraw().to_string().to_upper_camel_case();
    format!("{}.{camel}Entry", parent.value())
}

pub(crate) fn secret_map(ty: &Type) -> syn::Result<bool> {
    let segment = marker(ty)?;
    if segment.ident == "Map"
        && let syn::PathArguments::AngleBracketed(arguments) = &segment.arguments
        && let Some(syn::GenericArgument::Type(value)) = arguments.args.last()
    {
        return Ok(marker(value)?.ident == "Secret");
    }
    Ok(false)
}

pub(crate) fn semantic_type(ty: &Type) -> syn::Result<&Type> {
    let segment = marker(ty)?;
    if segment.ident == "Vec"
        && let syn::PathArguments::AngleBracketed(arguments) = &segment.arguments
        && let Some(syn::GenericArgument::Type(inner)) = arguments.args.first()
    {
        return semantic_type(inner);
    }
    Ok(ty)
}

fn parse_rejection(input: ParseStream<'_>, span: Span) -> syn::Result<ApiExpectation> {
    let code = Ident::new("InvalidArgument", span);
    let expected = if input.peek(Token![=]) {
        input.parse::<Token![=]>()?;
        if input.peek(LitStr) {
            ApiExpectation::Rejects(code, Some(input.parse()?))
        } else {
            let value: Ident = input.parse()?;
            if value != "any_status" {
                return Err(syn::Error::new(value.span(), "expected a regex or any_status"));
            }
            ApiExpectation::AnyStatus
        }
    } else if input.peek(syn::token::Paren) {
        let body;
        syn::parenthesized!(body in input);
        let code = body.parse()?;
        let pattern = if body.is_empty() {
            None
        } else {
            body.parse::<Token![,]>()?;
            Some(body.parse()?)
        };
        if !body.is_empty() {
            return Err(body.error("expected status and optional regex"));
        }
        ApiExpectation::Rejects(code, pattern)
    } else {
        ApiExpectation::Rejects(code, None)
    };
    Ok(expected)
}

fn parse_also(input: ParseStream<'_>) -> syn::Result<ApiAttributes> {
    let body;
    syn::parenthesized!(body in input);
    let kind: Ident = body.parse()?;
    let kind = format_ident!(
        "{}",
        match kind.to_string().as_str() {
            "uuid" => "Uuid",
            "identifier" => "Identifier",
            "text" => "Text",
            "bytes" => "Bytes",
            "duration" => "Duration",
            "counter" => "Counter",
            _ => return Err(syn::Error::new(kind.span(), "unsupported supplemental kind")),
        }
    );
    let mut attributes: ApiAttributes = if body.is_empty() {
        ApiAttributes::default()
    } else {
        body.parse::<Token![,]>()?;
        body.parse()?
    };
    if attributes.context.is_some() || attributes.api.is_some() {
        return Err(syn::Error::new(
            kind.span(),
            "context and API roots belong on the catalog item",
        ));
    }
    attributes.also_kind = Some(kind);
    Ok(attributes)
}

fn parse_field(field: &syn::Field, request: bool) -> syn::Result<DeriveField> {
    let name = field
        .ident
        .clone()
        .ok_or_else(|| syn::Error::new_spanned(field, "expected named field"))?;
    let field_lines = attributes(&field.attrs)?;
    if field_lines.iter().any(|line| !line.rules.is_empty()) {
        return Err(syn::Error::new_spanned(field, "presence rules belong on the catalog item"));
    }
    let mut attrs = merge(field_lines)?;
    if attrs.context.is_some()
        || attrs.api.is_some()
        || attrs.direction.is_some()
        || attrs.proto_type.is_some()
        || attrs.supplemental
        || attrs.response_checks.is_some()
    {
        return Err(syn::Error::new_spanned(field, "message attributes cannot be used on fields"));
    }
    let kind = marker(&field.ty)?.ident.to_string();
    validate_kind(&field.ty, &kind)?;
    if kind == "OneOf" && attrs.variants.is_empty() {
        return Err(syn::Error::new(
            name.span(),
            "OneOf requires exhaustive #[mirror(oneof(...))] alternatives",
        ));
    }
    if kind != "OneOf" && !attrs.variants.is_empty() {
        return Err(syn::Error::new(name.span(), "oneof alternatives require a OneOf kind"));
    }
    let has_role = attrs
        .tags
        .iter()
        .any(|tag| ["Decided", "Observed", "Ignored"].contains(&tag.name.to_string().as_str()));
    if !has_role && kind != "Ignored" {
        attrs.tags.insert(
            0,
            Tag {
                name: Ident::new(
                    if request
                        || marker(semantic_type(&field.ty)?)?.ident == "Secret"
                        || secret_map(&field.ty)?
                    {
                        "Decided"
                    } else {
                        "Observed"
                    },
                    name.span(),
                ),
                argument: None,
            },
        );
    }
    Ok(DeriveField {
        name,
        ty: field.ty.clone(),
        description: documentation(&field.attrs),
        attributes: attrs,
    })
}

fn validate_kind(ty: &Type, kind: &str) -> syn::Result<()> {
    if ![
        "Text",
        "Identifier",
        "Uuid",
        "Duration",
        "RawDuration",
        "Timestamp",
        "Counter",
        "Flag",
        "Number",
        "Bytes",
        "Secret",
        "Enum",
        "Message",
        "OneOf",
        "Map",
        "Ignored",
        "Vec",
    ]
    .contains(&kind)
    {
        return Err(syn::Error::new_spanned(ty, "expected exactly one supported semantic kind"));
    }
    Ok(())
}

#[cfg(test)]
mod attribute_controls {
    use super::*;

    #[test]
    fn scalar_options_preserve_generic_type_paths() -> syn::Result<()> {
        for key in ["context", "api", "type", "via", "response_checks"] {
            let parsed: ApiAttributes = syn::parse_str(&format!("{key} = crate::Context<u32>"))?;
            let path = match key {
                "context" => parsed.context,
                "api" => parsed.api,
                "type" => parsed.proto_type,
                "via" => parsed.via,
                _ => parsed.response_checks,
            }
            .ok_or_else(|| syn::Error::new(Span::call_site(), "missing scalar path"))?;
            assert_eq!(
                quote::quote!(#path).to_string(),
                quote::quote!(crate::Context<u32>).to_string()
            );
        }
        Ok(())
    }

    #[test]
    fn scalar_options_refuse_duplicates_at_the_second_occurrence()
    -> Result<(), Box<dyn std::error::Error>> {
        for source in [
            "supplemental, supplemental",
            "context = crate::First, context = crate::Second",
            "request = \"first.Message\", response = \"second.Message\"",
        ] {
            let error = syn::parse_str::<ApiAttributes>(source).err().ok_or(source)?;
            assert!(error.span().start().column > source.find(',').ok_or("comma")?, "{error}");
        }
        Ok(())
    }

    #[test]
    fn scalar_errors_accumulate_across_attribute_lines() -> Result<(), Box<dyn std::error::Error>> {
        let input: DeriveInput = syn::parse_str(
            "#[mirror(context = First, api = first)]\n#[mirror(context = Second, api = second)] struct Mirror {}",
        )?;
        let error = merge(attributes(&input.attrs)?).err().ok_or("duplicate options")?;
        let errors: Vec<_> = error.into_iter().collect();
        assert_eq!(errors.len(), 2);
        assert!(errors.iter().all(|error| error.span().start().line == 2));
        let input: DeriveInput =
            syn::parse_str("#[rule(accepts)]\n#[rule(unobservable)] struct Mirror {}")?;
        let error = merge(attributes(&input.attrs)?).err().ok_or("duplicate expectation")?;
        assert_eq!(error.span().start().line, 2);
        Ok(())
    }

    #[test]
    fn manual_families_keep_source_order_and_strict_shapes() -> syn::Result<()> {
        let parsed: ApiAttributes = syn::parse_str(
            "observed, min_len = 1, decided, max_len = 9, also(uuid, observed), oneof(Second = \"second\", First = \"first\"), rejects(InvalidArgument, \"bad\")",
        )?;
        assert_eq!(
            parsed.tags.iter().map(|tag| tag.name.to_string()).collect::<Vec<_>>(),
            ["Observed", "Decided"]
        );
        assert_eq!(
            parsed.constraints.iter().map(|(name, _)| name.to_string()).collect::<Vec<_>>(),
            ["min_len", "max_len"]
        );
        assert_eq!(
            parsed.variants.iter().map(|(name, _)| name.to_string()).collect::<Vec<_>>(),
            ["Second", "First"]
        );
        assert_eq!(
            parsed.also[0].also_kind.as_ref().map(ToString::to_string).as_deref(),
            Some("Uuid")
        );
        assert!(matches!(parsed.expected, Some(ApiExpectation::Rejects(_, Some(_)))));
        for source in [
            "context = \"crate::Context\"",
            "request = Message",
            "supplemental = false",
            "also(uuid = true)",
            "oneof(A)",
            "rejects(Code, \"bad\", extra)",
            "unknown = true",
        ] {
            assert!(syn::parse_str::<ApiAttributes>(source).is_err(), "{source}");
        }
        Ok(())
    }

    #[test]
    fn names_follow_prost_case_conversion() -> syn::Result<()> {
        for (source, expected) in [
            ("HTTPInput", "http_input"),
            ("foo__Bar", "foo_bar"),
            ("_Foo_", "foo"),
            ("URL42Value", "url42_value"),
        ] {
            assert_eq!(factory_name(&syn::parse_str::<Ident>(source)?).to_string(), expected);
        }
        Ok(())
    }
}
