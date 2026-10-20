//! Literal grammar and statically decidable admission.

use crate::{
    Actions, Binding, Catalog, CustomTag, Enum, Expected, Field, Message, Projection, Rule,
    Service, Tag, Variant,
};
use std::collections::BTreeSet;
use syn::{
    Error, Lit, Token, braced, bracketed,
    ext::IdentExt,
    parenthesized,
    parse::{Parse, ParseStream},
    punctuated::Punctuated,
};

fn word(input: ParseStream<'_>, expected: &str) -> syn::Result<()> {
    let found: syn::Ident = input.parse()?;
    if found != expected {
        return Err(Error::new(found.span(), format!("expected {expected}")));
    }
    Ok(())
}

fn nonempty(text: &syn::LitStr, label: &str) -> syn::Result<()> {
    if text.value().trim().is_empty() {
        return Err(Error::new(text.span(), format!("{label} requires a nonempty reason or name")));
    }
    Ok(())
}

fn unique<'a>(values: impl IntoIterator<Item = &'a syn::Ident>) -> syn::Result<()> {
    let mut seen = BTreeSet::new();
    for value in values {
        if !seen.insert(value.to_string()) {
            return Err(Error::new(value.span(), format!("duplicate declaration {value}")));
        }
    }
    Ok(())
}

impl Parse for Catalog {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let vis = input.parse()?;
        word(input, "catalog")?;
        let name = input.parse()?;
        input.parse::<Token![;]>()?;
        let mut catalog = Self {
            vis,
            name,
            messages: vec![],
            enums: vec![],
            services: vec![],
            actions: vec![],
            tags: vec![],
        };
        while !input.is_empty() {
            let keyword = input.call(syn::Ident::parse_any)?;
            match keyword.to_string().as_str() {
                "message" => catalog.messages.push(input.parse()?),
                "enum" => catalog.enums.push(input.parse()?),
                "service" => catalog.services.push(input.parse()?),
                "actions" => catalog.actions.push(input.parse()?),
                "tag" => catalog.tags.push(input.parse()?),
                _ => {
                    return Err(Error::new(
                        keyword.span(),
                        "expected message, enum, service, actions or tag",
                    ));
                }
            }
        }
        catalog.admit()?;
        Ok(catalog)
    }
}

impl Parse for Message {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let direction: syn::Ident = input.parse()?;
        let request = match direction.to_string().as_str() {
            "request" => true,
            "response" => false,
            _ => return Err(Error::new(direction.span(), "expected request or response")),
        };
        let factory = input.parse()?;
        input.parse::<Token![:]>()?;
        let ty = input.parse()?;
        input.parse::<Token![=]>()?;
        let proto = input.parse()?;
        nonempty(&proto, "message")?;
        let body;
        braced!(body in input);
        let mut fields = vec![];
        let mut rules = vec![];
        while !body.is_empty() {
            let fork = body.fork();
            let name: syn::Ident = fork.parse()?;
            if name == "rules" && fork.peek(syn::token::Brace) {
                word(&body, "rules")?;
                let items;
                braced!(items in body);
                while !items.is_empty() {
                    rules.push(items.parse()?);
                }
            } else {
                fields.push(body.parse()?);
            }
        }
        unique(fields.iter().map(|field: &Field| &field.name))?;
        Ok(Self { request, factory, ty, proto, fields, rules })
    }
}

impl Parse for Field {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let name = input.parse()?;
        input.parse::<Token![:]>()?;
        let kind: syn::Ident = input.parse()?;
        let enumeration = if kind == "Enum" && input.peek(syn::token::Paren) {
            let args;
            parenthesized!(args in input);
            let name = args.parse()?;
            if !args.is_empty() {
                return Err(args.error("expected one enum declaration name"));
            }
            Some(name)
        } else {
            None
        };
        let mut constraints = vec![];
        if input.peek(syn::token::Bracket) {
            let args;
            bracketed!(args in input);
            while !args.is_empty() {
                let key = args.parse()?;
                args.parse::<Token![=]>()?;
                let value = args.parse()?;
                constraints.push((key, value));
                if !args.is_empty() {
                    args.parse::<Token![,]>()?;
                }
            }
        }
        let mut tags = vec![];
        while input.peek(Token![+]) {
            input.parse::<Token![+]>()?;
            tags.push(input.parse()?);
        }
        let mut projection = None;
        let mut variants = vec![];
        let mut expected = None;
        let mut opaque = false;
        while !input.peek(Token![;]) {
            if input.peek(Token![=>]) {
                if expected.is_some() {
                    return Err(input.error("duplicate field expectation"));
                }
                input.parse::<Token![=>]>()?;
                expected = Some(input.parse()?);
            } else {
                let option: syn::Ident = input.parse()?;
                match option.to_string().as_str() {
                    "opaque" if !opaque => opaque = true,
                    "via" if projection.is_none() => {
                        let ty = input.parse()?;
                        let args;
                        parenthesized!(args in input);
                        let get = args.parse()?;
                        args.parse::<Token![,]>()?;
                        let set = args.parse()?;
                        if !args.is_empty() {
                            return Err(args.error("expected getter and setter"));
                        }
                        projection = Some(Projection { ty, get, set });
                    }
                    "variants" if variants.is_empty() => {
                        let args;
                        braced!(args in input);
                        variants = parse_variants(&args)?;
                        if variants.is_empty() {
                            return Err(args.error("oneof variants cannot be empty"));
                        }
                    }
                    _ => {
                        return Err(Error::new(
                            option.span(),
                            "expected via, variants or => expectation",
                        ));
                    }
                }
            }
        }
        input.parse::<Token![;]>()?;
        Ok(Self {
            opaque,
            name,
            kind,
            constraints,
            tags,
            enumeration,
            projection,
            variants,
            expected,
        })
    }
}

impl Parse for Tag {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let name = input.parse()?;
        let argument = if input.peek(syn::token::Paren) {
            let args;
            parenthesized!(args in input);
            let argument = args.parse()?;
            if !args.is_empty() {
                return Err(args.error("tag requires one literal consequence argument"));
            }
            Some(argument)
        } else {
            None
        };
        Ok(Self { name, argument })
    }
}

impl Parse for CustomTag {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let name = input.parse()?;
        let body;
        braced!(body in input);
        word(&body, "kinds")?;
        body.parse::<Token![:]>()?;
        let kinds;
        bracketed!(kinds in body);
        let kinds =
            Punctuated::<syn::Ident, Token![,]>::parse_terminated(&kinds)?.into_iter().collect();
        body.parse::<Token![;]>()?;
        word(&body, "consequence")?;
        body.parse::<Token![:]>()?;
        let consequence = body.parse()?;
        body.parse::<Token![;]>()?;
        if !body.is_empty() {
            return Err(body.error("unexpected custom tag declaration"));
        }
        Ok(Self { name, kinds, consequence })
    }
}

impl Parse for Expected {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let outcome: syn::Ident = input.parse()?;
        let reject = match outcome.to_string().as_str() {
            "Accepts" | "Unobservable" => None,
            "Rejects" => {
                let args;
                parenthesized!(args in input);
                let code = args.parse()?;
                args.parse::<Token![,]>()?;
                let pattern = args.parse()?;
                let details = if args.is_empty() {
                    None
                } else {
                    args.parse::<Token![,]>()?;
                    Some(args.parse()?)
                };
                if !args.is_empty() {
                    return Err(args.error("unexpected Rejects argument"));
                }
                Some((code, pattern, details))
            }
            _ => {
                return Err(Error::new(
                    outcome.span(),
                    "expected Accepts, Unobservable or Rejects",
                ));
            }
        };
        word(input, "covers")?;
        let covers = input.parse()?;
        nonempty(&covers, "expectation covers")?;
        Ok(Self { outcome, reject, covers })
    }
}

impl Parse for Rule {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let operator = input.parse()?;
        let fields;
        parenthesized!(fields in input);
        let fields =
            Punctuated::<syn::Ident, Token![,]>::parse_terminated(&fields)?.into_iter().collect();
        let expected = if input.peek(Token![=>]) {
            input.parse::<Token![=>]>()?;
            Some(input.parse()?)
        } else {
            None
        };
        input.parse::<Token![;]>()?;
        Ok(Self { operator, fields, expected })
    }
}

fn parse_variants(input: ParseStream<'_>) -> syn::Result<Vec<Variant>> {
    let mut variants = vec![];
    while !input.is_empty() {
        let path = input.parse()?;
        input.parse::<Token![=]>()?;
        let proto = input.parse()?;
        nonempty(&proto, "variant")?;
        variants.push(Variant { path, proto });
        if !input.is_empty() {
            input.parse::<Token![,]>()?;
        }
    }
    Ok(variants)
}

impl Parse for Enum {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let factory = input.parse()?;
        input.parse::<Token![:]>()?;
        let ty = input.parse()?;
        input.parse::<Token![=]>()?;
        let proto = input.parse()?;
        nonempty(&proto, "enum")?;
        let body;
        braced!(body in input);
        let variants = parse_variants(&body)?;
        if variants.is_empty() {
            return Err(body.error("enum variants cannot be empty"));
        }
        Ok(Self { factory, ty, proto, variants })
    }
}

impl Parse for Binding {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let kind = input.parse()?;
        let args;
        parenthesized!(args in input);
        let target = args.parse()?;
        nonempty(&target, "binding")?;
        if !args.is_empty() {
            return Err(args.error("binding requires one literal target or reason"));
        }
        Ok(Self { kind, target })
    }
}

impl Parse for Service {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let proto = input.parse()?;
        nonempty(&proto, "service")?;
        let body;
        braced!(body in input);
        let mut methods = vec![];
        while !body.is_empty() {
            let name = body.parse()?;
            body.parse::<Token![:]>()?;
            let binding: Binding = body.parse()?;
            if !["Action", "Observed", "OutOfScope"].contains(&binding.kind.to_string().as_str()) {
                return Err(Error::new(
                    binding.kind.span(),
                    "RPC requires Action, Observed or OutOfScope(reason)",
                ));
            }
            body.parse::<Token![;]>()?;
            methods.push((name, binding));
        }
        unique(methods.iter().map(|(name, _)| name))?;
        Ok(Self { proto, methods })
    }
}

impl Parse for Actions {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let factory = input.parse()?;
        input.parse::<Token![:]>()?;
        let ty = input.parse()?;
        let body;
        braced!(body in input);
        let mut actors = vec![];
        while !body.is_empty() {
            let actor = body.parse()?;
            body.parse::<Token![=>]>()?;
            let leaves;
            braced!(leaves in body);
            let mut bindings = vec![];
            while !leaves.is_empty() {
                let pattern = syn::Pat::parse_single(&leaves)?;
                if !matches!(
                    pattern,
                    syn::Pat::Path(_) | syn::Pat::TupleStruct(_) | syn::Pat::Struct(_)
                ) {
                    return Err(Error::new_spanned(
                        pattern,
                        "action must name a variant; wildcard bindings hide additions",
                    ));
                }
                leaves.parse::<Token![=>]>()?;
                let binding: Binding = leaves.parse()?;
                if !["Rpc", "Observed", "Environmental"]
                    .contains(&binding.kind.to_string().as_str())
                {
                    return Err(Error::new(
                        binding.kind.span(),
                        "action requires Rpc, Observed or Environmental(reason)",
                    ));
                }
                bindings.push((pattern, binding));
                if !leaves.is_empty() {
                    leaves.parse::<Token![,]>()?;
                }
            }
            actors.push((actor, bindings));
            if !body.is_empty() {
                body.parse::<Token![,]>()?;
            }
        }
        Ok(Self { factory, ty, actors })
    }
}

const KINDS: &[&str] = &[
    "Uuid",
    "Identifier",
    "Timestamp",
    "Duration",
    "Counter",
    "Random",
    "Secret",
    "Enum",
    "Text",
    "Bytes",
];

fn builtin(tag: &Tag, kind: &str) -> syn::Result<()> {
    let name = tag.name.to_string();
    let argument_ok = match name.as_str() {
        "Ignored" => {
            matches!(&tag.argument, Some(Lit::Str(reason)) if !reason.value().trim().is_empty())
        }
        "Cached" | "EventuallyConsistent" => {
            matches!(&tag.argument, Some(Lit::Int(value)) if value.base10_parse::<u64>().is_ok_and(|value| value > 0))
        }
        "Monotone" | "Immutable" | "IdempotencyKey" | "Decided" | "Observed" => {
            tag.argument.is_none()
        }
        _ => {
            return Err(Error::new(
                tag.name.span(),
                "custom tag requires a declarative compatible-kind/consequence definition",
            ));
        }
    };
    if !argument_ok {
        return Err(Error::new(
            tag.name.span(),
            "tag requires a valid literal reason or positive consequence",
        ));
    }
    let compatible = match name.as_str() {
        "Monotone" => ["Counter", "Timestamp", "Duration"].contains(&kind),
        "IdempotencyKey" => ["Uuid", "Identifier", "Text", "Bytes"].contains(&kind),
        _ => true,
    };
    if !compatible {
        return Err(Error::new(
            tag.name.span(),
            format!("tag {name} is incompatible with kind {kind}"),
        ));
    }
    Ok(())
}

impl Catalog {
    fn admit(&self) -> syn::Result<()> {
        unique(
            self.messages
                .iter()
                .map(|m| &m.factory)
                .chain(self.enums.iter().map(|e| &e.factory))
                .chain(self.actions.iter().map(|a| &a.factory))
                .chain(self.tags.iter().map(|t| &t.name)),
        )?;
        for tag in &self.tags {
            admit_custom_tag(tag)?;
        }
        for message in &self.messages {
            for field in &message.fields {
                self.admit_field(message, field)?;
            }
            for rule in &message.rules {
                let name = rule.operator.to_string();
                if !["required", "any_of", "one_of", "requires", "excludes"]
                    .contains(&name.as_str())
                    || rule.fields.is_empty()
                    || (["requires", "excludes"].contains(&name.as_str()) && rule.fields.len() != 2)
                {
                    return Err(Error::new(
                        rule.operator.span(),
                        "invalid presence rule or operand count",
                    ));
                }
                unique(rule.fields.iter())?;
                for operand in &rule.fields {
                    let field = message
                        .fields
                        .iter()
                        .find(|field| {
                            if field.variants.is_empty() {
                                field.name == *operand
                            } else {
                                field
                                    .variants
                                    .iter()
                                    .any(|variant| operand.unraw() == variant.proto.value())
                            }
                        })
                        .ok_or_else(|| Error::new(operand.span(), "unknown rule field"))?;
                    if field.tags.iter().any(|tag| tag.name == "Ignored") {
                        return Err(Error::new(
                            operand.span(),
                            "ignored fields cannot appear in rules",
                        ));
                    }
                }
                if message.request && rule.expected.is_none() {
                    return Err(Error::new(
                        rule.operator.span(),
                        "request rule requires an expectation and covers",
                    ));
                }
            }
        }
        for actions in &self.actions {
            for (_, leaves) in &actions.actors {
                for (pattern, binding) in leaves {
                    if binding.kind == "Rpc" {
                        let target = binding.target.value();
                        let classified = self.services.iter().any(|service| {
                            service.methods.iter().any(|(method, disposition)| {
                                format!("{}.{}", service.proto.value(), method) == target
                                    && disposition.kind == "Action"
                                    && disposition.target.value() == action_name(pattern)
                            })
                        });
                        if !classified {
                            return Err(Error::new(
                                binding.target.span(),
                                "action RPC must name a service method classified for this action variant",
                            ));
                        }
                    }
                }
            }
        }
        Ok(())
    }

    fn admit_field(&self, message: &Message, field: &Field) -> syn::Result<()> {
        let kind = field.kind.to_string();
        if !KINDS.contains(&kind.as_str()) {
            return Err(Error::new(
                field.kind.span(),
                "expected exactly one supported semantic kind",
            ));
        }
        if kind == "Enum"
            && !field
                .enumeration
                .as_ref()
                .is_some_and(|factory| self.enums.iter().any(|e| e.factory == *factory))
        {
            return Err(Error::new(
                field.kind.span(),
                "Enum requires a declared exhaustive enum conversion",
            ));
        }
        admit_constraints(field, &kind)?;
        unique(field.tags.iter().map(|tag| &tag.name))?;
        let mut roles = vec![];
        for tag in &field.tags {
            if let Some(custom) = self.tags.iter().find(|custom| custom.name == tag.name) {
                if tag.argument.is_some()
                    || !custom.kinds.iter().any(|candidate| candidate == &field.kind)
                {
                    return Err(Error::new(
                        tag.name.span(),
                        format!("custom tag {} is incompatible with kind {kind}", tag.name),
                    ));
                }
            } else {
                builtin(tag, &kind)?;
            }
            if ["Decided", "Observed", "Ignored"].contains(&tag.name.to_string().as_str()) {
                roles.push(tag.name.to_string());
            }
        }
        if roles.len() > 1
            || (kind == "Random" && roles.iter().any(|r| r == "Decided"))
            || (kind == "Secret" && roles.iter().any(|r| r == "Observed"))
        {
            return Err(Error::new(field.name.span(), "contradictory roles or kind/tag pair"));
        }
        let ignored = roles.iter().any(|r| r == "Ignored");
        if field.opaque
            && (!ignored
                || !["Bytes", "Secret"].contains(&kind.as_str())
                || field.tags.len() != 1
                || !field.constraints.is_empty()
                || field.projection.is_some()
                || field.expected.is_some())
        {
            return Err(Error::new(
                field.name.span(),
                "opaque fields require Bytes or Secret and only Ignored(reason)",
            ));
        }
        if !field.variants.is_empty() && !field.opaque && field.projection.is_none() {
            return Err(Error::new(
                field.name.span(),
                "oneof alternatives require a typed projection or intentional opaque Ignored(reason)",
            ));
        }
        let constrained = field.constraints.iter().any(|(name, literal)| {
            name != "unique" || !matches!(literal, Lit::Bool(value) if !value.value)
        }) || ["Uuid", "Identifier", "Timestamp", "Duration", "Enum"]
            .contains(&kind.as_str());
        if message.request && !ignored && constrained && field.expected.is_none() {
            return Err(Error::new(
                field.name.span(),
                "request constraint requires an expectation and covers",
            ));
        }
        if !constrained && field.expected.is_some() {
            return Err(Error::new(field.name.span(), "expectation names no declared constraint"));
        }
        Ok(())
    }
}

fn action_name(pattern: &syn::Pat) -> String {
    let path = match pattern {
        syn::Pat::Path(pattern) => &pattern.path,
        syn::Pat::TupleStruct(pattern) => &pattern.path,
        syn::Pat::Struct(pattern) => &pattern.path,
        _ => unreachable!("parser admits only named action variants"),
    };
    path.segments
        .iter()
        .map(|segment| segment.ident.unraw().to_string())
        .collect::<Vec<_>>()
        .join("::")
}

fn admit_custom_tag(tag: &CustomTag) -> syn::Result<()> {
    if [
        "Monotone",
        "Immutable",
        "Cached",
        "EventuallyConsistent",
        "IdempotencyKey",
        "Decided",
        "Observed",
        "Ignored",
    ]
    .contains(&tag.name.to_string().as_str())
    {
        return Err(Error::new(tag.name.span(), "custom tag cannot replace a builtin tag"));
    }
    if tag.kinds.is_empty() {
        return Err(Error::new(
            tag.name.span(),
            "custom tag requires compatible kinds and a consequence",
        ));
    }
    unique(tag.kinds.iter())?;
    for kind in &tag.kinds {
        if !KINDS.contains(&kind.to_string().as_str()) {
            return Err(Error::new(kind.span(), "unknown compatible kind"));
        }
        builtin(&tag.consequence, &kind.to_string())?;
    }
    if ["Ignored", "Decided", "Observed"].contains(&tag.consequence.name.to_string().as_str()) {
        return Err(Error::new(tag.name.span(), "custom tags cannot conceal field roles"));
    }
    Ok(())
}

fn admit_constraints(field: &Field, kind: &str) -> syn::Result<()> {
    unique(field.constraints.iter().map(|(name, _)| name))?;
    for (name, literal) in &field.constraints {
        let valid = match name.to_string().as_str() {
            "min_items" | "max_items" => matches!(literal, Lit::Int(_)),
            "unique" => matches!(literal, Lit::Bool(_)),
            "min_len" | "max_len" if ["Text", "Identifier", "Bytes", "Random"].contains(&kind) => {
                matches!(literal, Lit::Int(_))
            }
            "min_bytes" | "max_bytes" if ["Text", "Identifier"].contains(&kind) => {
                matches!(literal, Lit::Int(_))
            }
            "prefix" | "suffix" | "contains" if ["Text", "Identifier"].contains(&kind) => {
                matches!(literal, Lit::Str(_))
            }
            "gt" | "gte" | "lt" | "lte" if ["Counter", "Timestamp", "Duration"].contains(&kind) => {
                matches!(literal, Lit::Int(_))
            }
            _ => false,
        };
        if !valid {
            return Err(Error::new(
                name.span(),
                format!("constraint {name} is incompatible with kind {kind} or requires a literal"),
            ));
        }
    }
    Ok(())
}
