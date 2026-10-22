//! The `path!` pattern language as `path!` queries and progress claims both read it.
use proc_macro2::TokenStream;
use quote::quote;
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

/// One step pattern of a path.
pub enum Node {
    /// A state predicate, a state value or a concrete action.
    Value(Expr),
    /// An action variant whose input fields are matched by domains.
    Structured(Path, Vec<(Ident, Domain)>),
    /// Every action of one actor: `Actor::*`.
    Wildcard(Path),
    /// An action and the outcomes it must produce.
    Outcome(Box<Self>, Vec<Pat>),
    /// Alternatives: `(a | b)`.
    Either(Vec<Self>),
    /// A repetition with a count or an inclusive count range.
    Repeat(Box<Self>, Expr, Option<Box<Expr>>),
    /// Patterns in any order: `unordered![a, b]`.
    Unordered(Vec<Self>),
    /// A gap with an optional bound and excluded actions.
    Gap(Option<Expr>, Vec<Expr>),
}

/// The domain of one action input field in a structured pattern.
pub enum Domain {
    /// One value.
    Exact(Expr),
    /// A finite range.
    Range(Expr),
    /// Alternatives: `(a | b)`.
    Set(Vec<Expr>),
    /// Any value: `_`.
    Any,
}

/// The comma-separated patterns of one `path!`.
pub struct Nodes(pub Vec<Node>);
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

/// Admits a `path!` progress claim: a trigger, one gap and a target, all state predicates.
pub fn validate_claim(call: &syn::Macro, actors: &[Ident]) -> syn::Result<()> {
    let refuse = |message: String| Err(syn::Error::new_spanned(call, message));
    let nodes = match syn::parse2::<Nodes>(call.tokens.clone()) {
        Ok(Nodes(nodes)) => nodes,
        Err(error) => return refuse(error.to_string()),
    };
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
        return refuse("a progress claim is two state predicates".into());
    }
    if let [Node::Value(_), Node::Value(gap @ Expr::Range(_)), Node::Value(_)] = nodes.as_slice() {
        return Err(syn::Error::new_spanned(gap, "a path gap is .. or ..=n"));
    }
    if nodes.iter().filter(|node| !matches!(node, Node::Gap(..))).count() > 2 {
        return refuse(
            "a progress claim has one trigger and one target; ask a longer progression with find or Run::along".into(),
        );
    }
    if !matches!(nodes.as_slice(), [Node::Value(_), Node::Gap(_, _), Node::Value(_)]) {
        return refuse("a progress claim needs two state predicates separated by a gap".into());
    }
    Ok(())
}
