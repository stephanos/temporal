use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{
    Expr, Ident, Pat, Path, Token, braced, bracketed,
    ext::IdentExt,
    parenthesized,
    parse::discouraged::Speculative,
    parse::{Parse, ParseStream, Parser},
    punctuated::Punctuated,
};

fn split_pipes(tokens: TokenStream) -> Vec<TokenStream> {
    let mut parts = vec![TokenStream::new()];
    for token in tokens {
        if matches!(&token, proc_macro2::TokenTree::Punct(p) if p.as_char() == '|') {
            parts.push(TokenStream::new());
        } else {
            parts.last_mut().unwrap_or_else(|| unreachable!()).extend([token]);
        }
    }
    parts
}

enum Node {
    Value(Expr),
    Structured(Path, Vec<(Ident, Domain)>),
    Wildcard(Path),
    Outcome(Box<Self>, Vec<Pat>),
    Either(Vec<Self>),
    Repeat(Box<Self>, Expr, Option<Box<Expr>>),
    Unordered(Vec<Self>),
    Gap(Option<Expr>, Vec<Expr>),
}

enum Domain {
    Exact(Expr),
    Range(Expr),
    Set(Vec<Expr>),
    Any,
}

struct Nodes(Vec<Node>);
impl Parse for Nodes {
    fn parse(input: ParseStream<'_>) -> syn::Result<Self> {
        let mut nodes = Vec::new();
        while !input.is_empty() {
            nodes.push(parse_node(input)?);
            if !input.is_empty() {
                input.parse::<Token![,]>()?;
            }
        }
        Ok(Self(nodes))
    }
}

fn parse_node(input: ParseStream<'_>) -> syn::Result<Node> {
    let node =
        if input.peek(Token![..]) {
            input.parse::<Token![..]>()?;
            if input.peek(Token![!]) {
                input.parse::<Token![!]>()?;
                let inner;
                parenthesized!(inner in input);
                let excluded = Punctuated::<Expr, Token![,]>::parse_terminated(&inner)?;
                if excluded.is_empty() {
                    return Err(inner
                        .error("an exclusion gap names at least one action: ..!(Actor::Action)"));
                }
                Node::Gap(None, excluded.into_iter().collect())
            } else if input.peek(Token![=]) {
                input.parse::<Token![=]>()?;
                Node::Gap(Some(input.parse()?), vec![])
            } else {
                Node::Gap(None, vec![])
            }
        } else if input.peek(syn::token::Bracket) {
            let inner;
            bracketed!(inner in input);
            let body = parse_node(&inner)?;
            inner
                .parse::<Token![;]>()
                .map_err(|_| inner.error("a repetition is [pattern; n] or [pattern; a..=b]"))?;
            let count: Expr = inner.parse()?;
            let (min, max) = if let Expr::Range(range) = count {
                let Some(min) = range.start else {
                    return Err(inner.error("a repetition needs finite lower and upper bounds"));
                };
                let Some(max) = range.end else {
                    return Err(inner.error("a repetition needs finite lower and upper bounds"));
                };
                if !matches!(range.limits, syn::RangeLimits::Closed(_)) {
                    return Err(inner
                        .error("a bounded repetition uses an inclusive range: [pattern; a..=b]"));
                }
                (*min, Some(max))
            } else {
                (count, None)
            };
            Node::Repeat(Box::new(body), min, max)
        } else if input.peek(syn::token::Paren) {
            let inner;
            parenthesized!(inner in input);
            let tokens: TokenStream = inner.parse()?;
            let is_closure = tokens.clone().into_iter().next().is_some_and(
                |token| matches!(token, proc_macro2::TokenTree::Punct(p) if p.as_char() == '|'),
            ) || tokens.to_string().starts_with("move |");
            let parts = split_pipes(tokens.clone());
            if parts.len() > 1 && !is_closure {
                Node::Either(
                    parts
                        .into_iter()
                        .map(|tokens| parse_node.parse2(tokens))
                        .collect::<syn::Result<_>>()?,
                )
            } else {
                Node::Value(syn::parse2(quote!((#tokens)))?)
            }
        } else if input.peek(Ident) && input.fork().parse::<Ident>()? == "unordered" {
            input.parse::<Ident>()?;
            input.parse::<Token![!]>()?;
            let inner;
            bracketed!(inner in input);
            let nodes = Nodes::parse(&inner)?.0;
            if nodes.is_empty() || nodes.iter().any(|node| matches!(node, Node::Gap(..))) {
                return Err(inner.error("unordered![a, b, c] needs one or more non-gap patterns"));
            }
            Node::Unordered(nodes)
        } else {
            parse_value(input)?
        };
    if input.peek(Token![=>]) {
        input.parse::<Token![=>]>()?;
        let outcomes = parse_outcomes(input)?;
        if matches!(node, Node::Gap(..) | Node::Repeat(..) | Node::Either(..) | Node::Unordered(..))
        {
            return Err(
                input.error("an outcome follows one action: Actor::Action => Outcome::Value")
            );
        }
        Ok(Node::Outcome(Box::new(node), outcomes))
    } else {
        Ok(node)
    }
}

fn parse_outcomes(input: ParseStream<'_>) -> syn::Result<Vec<Pat>> {
    if !input.peek(syn::token::Paren) {
        return Ok(vec![Pat::parse_single(input)?]);
    }
    let inner;
    parenthesized!(inner in input);
    if inner.is_empty() {
        return Ok(vec![Pat::parse_single.parse2(quote!(()))?]);
    }
    Ok(Punctuated::<Pat, Token![|]>::parse_separated_nonempty_with(&inner, Pat::parse_single)?
        .into_iter()
        .collect())
}

fn parse_value(input: ParseStream<'_>) -> syn::Result<Node> {
    let fork = input.fork();
    if let Ok(path) = fork.call(path_prefix) {
        if fork.peek(Token![::]) {
            fork.parse::<Token![::]>()?;
            if fork.peek(Token![*]) {
                input.call(path_prefix)?;
                input.parse::<Token![::]>()?;
                input.parse::<Token![*]>()?;
                return Ok(Node::Wildcard(path));
            }
        }
        if fork.peek(syn::token::Brace) && path.segments.len() >= 2 {
            let ordinary = input.fork();
            if let Ok(Expr::Struct(value)) = ordinary.parse::<Expr>()
                && (value.dot2_token.is_none() || value.rest.is_some())
                && !value.fields.iter().any(|field| match &field.expr {
                    Expr::Infer(_) | Expr::Range(_) => true,
                    Expr::Paren(value) => matches!(value.expr.as_ref(), Expr::Binary(binary)
                        if matches!(binary.op, syn::BinOp::BitOr(_))),
                    _ => false,
                })
            {
                input.advance_to(&ordinary);
                return Ok(Node::Value(Expr::Struct(value)));
            }
            let path: Path = input.call(Path::parse_mod_style)?;
            let inner;
            braced!(inner in input);
            let mut fields: Vec<(Ident, Domain)> = Vec::new();
            let mut patterned = false;
            while !inner.is_empty() {
                if inner.peek(Token![..]) {
                    patterned = true;
                    inner.parse::<Token![..]>()?;
                    if !inner.is_empty() {
                        inner.parse::<Token![,]>()?;
                    }
                    break;
                }
                let field: Ident = inner.parse()?;
                if fields.iter().any(|(prior, _)| prior.unraw() == field.unraw()) {
                    return Err(syn::Error::new_spanned(
                        &field,
                        format!("duplicate field `{}` in path pattern", field.unraw()),
                    ));
                }
                inner.parse::<Token![:]>()?;
                let domain = if inner.peek(Token![_]) {
                    inner.parse::<Token![_]>()?;
                    Domain::Any
                } else if inner.peek(syn::token::Paren) {
                    let values;
                    parenthesized!(values in inner);
                    let tokens: TokenStream = values.parse()?;
                    Domain::Set(
                        split_pipes(tokens)
                            .into_iter()
                            .map(syn::parse2)
                            .collect::<syn::Result<_>>()?,
                    )
                } else {
                    let value: Expr = inner.parse()?;
                    if let Expr::Range(range) = &value {
                        if range.start.is_none() || range.end.is_none() {
                            return Err(syn::Error::new_spanned(
                                value,
                                "an action input needs a finite range",
                            ));
                        }
                        Domain::Range(value)
                    } else {
                        Domain::Exact(value)
                    }
                };
                fields.push((field, domain));
                if !inner.is_empty() {
                    inner.parse::<Token![,]>()?;
                }
            }
            if !inner.is_empty() {
                return Err(inner.error(".. must end the action input fields"));
            }
            patterned |= fields.iter().any(|(_, domain)| !matches!(domain, Domain::Exact(_)));
            if !patterned {
                let values = fields.iter().filter_map(|(field, domain)| {
                    if let Domain::Exact(value) = domain {
                        Some(quote!(#field: #value))
                    } else {
                        None
                    }
                });
                return Ok(Node::Value(syn::parse2(quote!(#path { #(#values),* }))?));
            }
            return Ok(Node::Structured(path, fields));
        }
    }
    Ok(Node::Value(input.parse()?))
}

fn path_prefix(input: ParseStream<'_>) -> syn::Result<Path> {
    let leading_colon = input.parse::<Option<Token![::]>>()?;
    let mut segments = Punctuated::new();
    segments.push_value(syn::PathSegment::from(input.parse::<Ident>()?));
    while input.peek(Token![::]) {
        let fork = input.fork();
        fork.parse::<Token![::]>()?;
        if !fork.peek(Ident) {
            break;
        }
        segments.push_punct(input.parse::<Token![::]>()?);
        segments.push_value(syn::PathSegment::from(input.parse::<Ident>()?));
    }
    Ok(Path { leading_colon, segments })
}

fn domain(path: &Path, fields: &[(Ident, Domain)]) -> syn::Result<TokenStream> {
    let variant = &path
        .segments
        .last()
        .ok_or_else(|| syn::Error::new_spanned(path, "an action names Actor::Action"))?
        .ident;
    let segments = path.segments.iter().take(path.segments.len() - 1).collect::<Vec<_>>();
    let leading = &path.leading_colon;
    let actor = quote!(#leading #(#segments)::*);
    let constructor = format_ident!("__umpire_path_{}", variant);
    let mut result = quote!(#actor::#constructor());
    for (field, domain) in fields {
        let input = match domain {
            Domain::Any => quote!(None),
            Domain::Exact(value) => quote!(Some(::umpire_dsl::InputDomain::values(vec![#value]))),
            Domain::Range(value) => quote!(Some(::umpire_dsl::InputDomain::range(#value))),
            Domain::Set(values) => {
                quote!(Some(::umpire_dsl::InputDomain::values(vec![#(#values),*])))
            }
        };
        let setter = format_ident!("__umpire_input_{}", field.unraw());
        result = quote!(#result.#setter(#input));
    }
    Ok(quote!(#result.finish().map(::core::convert::Into::into)))
}

fn pattern(node: &Node) -> syn::Result<TokenStream> {
    Ok(match node {
        Node::Value(Expr::Closure(closure)) => {
            quote!(__pattern.judged(#closure))
        }
        Node::Value(value) => {
            quote!({ use ::umpire_dsl::path_build::PatternValue as _; let __value = ::umpire_dsl::path_build::Probe::new(#value); (&__value).node() })
        }
        Node::Structured(path, fields) => {
            let domain = domain(path, fields)?;
            quote!(::umpire_dsl::Pattern::action(#domain, None, None))
        }
        Node::Wildcard(actor) => {
            quote!(::umpire_dsl::Pattern::action(#actor::__umpire_path_actions().map(::core::convert::Into::into), None, None))
        }
        Node::Either(nodes) => {
            let nodes = nodes.iter().map(pattern).collect::<syn::Result<Vec<_>>>()?;
            quote!(::umpire_dsl::Pattern::Either(vec![#(#nodes),*]))
        }
        Node::Unordered(nodes) => {
            let nodes = nodes.iter().map(pattern).collect::<syn::Result<Vec<_>>>()?;
            quote!(::umpire_dsl::Pattern::Unordered(vec![#(#nodes),*]))
        }
        Node::Repeat(node, min, max) => {
            let node = pattern(node)?;
            max.as_ref().map_or_else(
                || quote!((#node).repeat_exact(#min)),
                |max| quote!(::umpire_dsl::Pattern::Repeat(Box::new(#node), #min, #max)),
            )
        }
        Node::Gap(bound, excluded) => {
            let gap = bound.as_ref().map_or(
                quote!(::umpire_dsl::Gap::Any),
                |bound| quote!(::umpire_dsl::Gap::AtMost(#bound)),
            );
            let exclude = if excluded.is_empty() {
                quote!(None)
            } else {
                quote!(Some(::std::rc::Rc::new(move |__action| false #(|| __action == &(#excluded).into())*)))
            };
            quote!(::umpire_dsl::Pattern::Gap(#gap, #exclude))
        }
        Node::Outcome(node, outcomes) => {
            let base = if let Node::Value(value) = node.as_ref() {
                quote!(::umpire_dsl::path_build::outcome_action(#value))
            } else {
                pattern(node)?
            };
            let alternatives = outcomes.iter().map(|outcome| {
                quote!((
                    ::std::rc::Rc::new(|__outcome: &_| matches!(__outcome, #outcome))
                        as ::umpire_dsl::OutcomePredicate<_>,
                    stringify!(#outcome).into()
                ))
            });
            quote!((#base).with_outcomes(vec![#(#alternatives),*]))
        }
    })
}

pub(crate) fn expand(tokens: TokenStream) -> syn::Result<TokenStream> {
    let Nodes(nodes) = syn::parse2(tokens)?;
    if nodes.is_empty() {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a path has two or more waypoints: write [p, .., q]",
        ));
    }
    if matches!(nodes.first(), Some(Node::Gap(..))) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a path starts with a waypoint, not a gap: write [p, .., q]",
        ));
    }
    if matches!(nodes.last(), Some(Node::Gap(..))) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a path ends with a waypoint, not a gap: write [p, .., q]",
        ));
    }
    if nodes.windows(2).any(|pair| pair.iter().all(|node| matches!(node, Node::Gap(..)))) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "write one gap between two waypoints: [p, .., q]",
        ));
    }
    if nodes
        .windows(2)
        .any(|pair| pair.iter().all(|node| matches!(node, Node::Value(Expr::Lit(_)))))
    {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "write [p, ..=1, q] for at most one step, or [p, .., q]",
        ));
    }
    if nodes.len() == 1 && matches!(&nodes[0], Node::Value(_)) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a path has two or more waypoints: write [p, .., q]",
        ));
    }
    let first = match &nodes[0] {
        Node::Value(_) => quote!(__path),
        node => {
            let pattern = pattern(node)?;
            quote!(::umpire_dsl::path_build::WorldBuilder::new(::umpire_dsl::path_build::pattern_with(|__pattern| #pattern)))
        }
    };
    let mut built = if matches!(&nodes[0], Node::Value(_)) { quote!(__path) } else { first };
    for node in nodes.iter().skip(1) {
        let next = match node {
            Node::Gap(bound, excluded) if excluded.is_empty() => {
                let gap = bound.as_ref().map_or(
                    quote!(::umpire_dsl::Gap::Any),
                    |bound| quote!(::umpire_dsl::Gap::AtMost(#bound)),
                );
                quote!((#built).gap(#gap))
            }
            Node::Value(Expr::Closure(closure)) => quote!((#built).then_judged(#closure)),
            Node::Value(value) => {
                quote!({ let __built = #built; let __value = ::umpire_dsl::path_build::Probe::new(#value); (&__value).append(__built) })
            }
            node => {
                let pattern = pattern(node)?;
                quote!((#built).structure_with(|__pattern| #pattern))
            }
        };
        built = next;
    }
    match &nodes[0] {
        Node::Value(Expr::Closure(closure)) => Ok(quote!({
            use ::umpire_dsl::path_build::AppendPath as _;
            ::umpire_dsl::path_build::StateFactory::judged(#closure).build(|__path| #built)
        })),
        Node::Value(value) => Ok(
            quote!({ use ::umpire_dsl::path_build::{Factory as _, ActorFactoryValue as _, AppendPath as _}; let __first = ::umpire_dsl::path_build::Probe::new(#value); let __factory = (&__first).factory(); __factory.build(|__path| #built) }),
        ),
        _ => Ok(quote!({ use ::umpire_dsl::path_build::AppendPath as _; (#built).finish() })),
    }
}

pub(crate) fn validate_claim(tokens: TokenStream, actors: &[Ident]) -> syn::Result<()> {
    let Nodes(nodes) = syn::parse2(tokens)?;
    let actor_path =
        |path: &Path| path.segments.iter().any(|segment| actors.contains(&segment.ident));
    if nodes.iter().any(|node| match node {
        Node::Value(Expr::Path(value)) => actor_path(&value.path),
        Node::Value(Expr::Struct(value)) => actor_path(&value.path),
        Node::Value(Expr::Call(value)) => {
            matches!(value.func.as_ref(), Expr::Path(path) if actor_path(&path.path))
        }
        Node::Value(_) => false,
        Node::Gap(_, excluded) => !excluded.is_empty(),
        _ => true,
    }) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a progress claim is two state predicates",
        ));
    }
    if nodes.iter().filter(|node| !matches!(node, Node::Gap(..))).count() > 2 {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a progress claim has one trigger and one target; ask a longer progression with find or Run::along",
        ));
    }
    if !matches!(nodes.as_slice(), [Node::Value(_), Node::Gap(_, _), Node::Value(_)]) {
        return Err(syn::Error::new(
            proc_macro2::Span::call_site(),
            "a progress claim needs two state predicates separated by a gap",
        ));
    }
    Ok(())
}
