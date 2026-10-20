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
    Ok(quote!(#result.finish()))
}

fn outcome_alternatives(outcomes: &[Pat]) -> Vec<TokenStream> {
    outcomes
        .iter()
        .map(|outcome| {
            quote!((
                ::std::rc::Rc::new(|__outcome: &_| matches!(__outcome, #outcome))
                    as ::umpire_dsl::OutcomePredicate<_>,
                stringify!(#outcome).into()
            ))
        })
        .collect()
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
            quote!(::umpire_dsl::Pattern::action(#domain.map(::core::convert::Into::into), None, None))
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
            let alternatives = outcome_alternatives(outcomes);
            quote!((#base).with_outcomes(vec![#(#alternatives),*]))
        }
    })
}

fn piece(node: &Node, built: &TokenStream, actor_only: bool) -> syn::Result<TokenStream> {
    Ok(match node {
        Node::Value(value) if !actor_only && state_closure(value).is_some() => {
            let mut context = StateContext::default();
            state_context(node, &mut context);
            if context.needs_owner {
                quote!((#built).resolve().judged(#value))
            } else if context.authored {
                quote!({ let __built = #built; __built.join(::umpire_dsl::path_build::StateFactory::judged(#value).piece()) })
            } else {
                quote!((#built).judged(#value))
            }
        }
        Node::Value(value) => {
            let factory = if actor_only {
                quote!(::umpire_dsl::path_build::actor_factory(#value))
            } else {
                quote!({
                    use ::umpire_dsl::path_build::{Factory as _, ActorFactoryValue as _};
                    let __value = ::umpire_dsl::path_build::Probe::new(#value);
                    (&__value).factory()
                })
            };
            quote!({ let __built = #built; __built.join((#factory).piece()) })
        }
        Node::Structured(path, fields) => {
            let domain = domain(path, fields)?;
            quote!({ let __built = #built; __built.join(::umpire_dsl::path_build::DomainFactory::new(#domain).piece()) })
        }
        Node::Wildcard(actor) => {
            quote!({ let __built = #built; __built.join(::umpire_dsl::path_build::DomainFactory::new(#actor::__umpire_path_actions()).piece()) })
        }
        Node::Either(nodes) | Node::Unordered(nodes) => {
            let mut child = quote!(__child);
            for node in nodes {
                child = piece(node, &child, false)?;
            }
            let wrap = if matches!(node, Node::Either(_)) {
                quote!(::umpire_dsl::Pattern::Either)
            } else {
                quote!(::umpire_dsl::Pattern::Unordered)
            };
            quote!((#built).group(|__child| (#child).wrap(#wrap)))
        }
        Node::Repeat(node, min, max) => {
            let child = piece(node, &quote!(__child), false)?;
            let wrapped = max.as_ref().map_or_else(
                || quote!(__node.repeat_exact(#min)),
                |max| quote!(::umpire_dsl::Pattern::Repeat(Box::new(__node), #min, #max)),
            );
            quote!((#built).group(|__child| (#child).wrap(|mut __parts| {
                let __node = __parts.pop().expect("repeat has one child");
                #wrapped
            })))
        }
        Node::Outcome(node, outcomes) => {
            let child = piece(node, &quote!(__child), true)?;
            let alternatives = outcome_alternatives(outcomes);
            quote!((#built).group(|__child| (#child).wrap(|mut __parts| {
                __parts.pop().expect("outcome has one action").with_outcomes(vec![#(#alternatives),*])
            })))
        }
        Node::Gap(..) => {
            let node = pattern(node)?;
            quote!((#built).push(#node))
        }
    })
}

#[derive(Default)]
struct StateContext {
    explicit: Option<syn::Type>,
    needs_owner: bool,
    authored: bool,
}

fn state_closure(value: &Expr) -> Option<&syn::ExprClosure> {
    match value {
        Expr::Closure(closure) => Some(closure),
        Expr::Paren(value) => state_closure(&value.expr),
        _ => None,
    }
}

fn rooted_at(value: &Expr, parameter: &Ident) -> bool {
    match value {
        Expr::Path(value) => {
            value.path.get_ident().is_some_and(|ident| ident.unraw() == parameter.unraw())
        }
        Expr::Field(value) => rooted_at(&value.base, parameter),
        Expr::MethodCall(value) => rooted_at(&value.receiver, parameter),
        Expr::Paren(value) => rooted_at(&value.expr, parameter),
        Expr::Reference(value) => rooted_at(&value.expr, parameter),
        Expr::Unary(value) => rooted_at(&value.expr, parameter),
        _ => false,
    }
}

struct StateMembers<'a> {
    parameter: &'a Ident,
    visible: bool,
    found: bool,
    independent: bool,
}

fn tokens_use_parameter(tokens: TokenStream, parameter: &Ident) -> bool {
    tokens.into_iter().any(|token| match token {
        proc_macro2::TokenTree::Ident(ident) => ident.unraw() == parameter.unraw(),
        proc_macro2::TokenTree::Group(group) => tokens_use_parameter(group.stream(), parameter),
        _ => false,
    })
}

fn is_parameter(value: &Expr, parameter: &Ident) -> bool {
    match value {
        Expr::Path(value) => {
            value.path.get_ident().is_some_and(|ident| ident.unraw() == parameter.unraw())
        }
        Expr::Paren(value) => is_parameter(&value.expr, parameter),
        Expr::Reference(value) => is_parameter(&value.expr, parameter),
        _ => false,
    }
}

fn binds_parameter(pattern: &Pat, parameter: &Ident) -> bool {
    struct Binding<'a> {
        parameter: &'a Ident,
        found: bool,
    }
    impl<'ast> syn::visit::Visit<'ast> for Binding<'_> {
        fn visit_pat_ident(&mut self, pattern: &'ast syn::PatIdent) {
            self.found |= pattern.ident.unraw() == self.parameter.unraw();
            syn::visit::visit_pat_ident(self, pattern);
        }
        fn visit_expr(&mut self, _: &'ast Expr) {}
    }
    let mut binding = Binding { parameter, found: false };
    syn::visit::Visit::visit_pat(&mut binding, pattern);
    binding.found
}

impl<'ast> syn::visit::Visit<'ast> for StateMembers<'_> {
    fn visit_expr_field(&mut self, field: &'ast syn::ExprField) {
        self.found |= self.visible && rooted_at(&field.base, self.parameter);
        syn::visit::visit_expr_field(self, field);
    }
    fn visit_expr_method_call(&mut self, call: &'ast syn::ExprMethodCall) {
        self.found |= self.visible && rooted_at(&call.receiver, self.parameter);
        self.independent |= self.visible
            && !rooted_at(&call.receiver, self.parameter)
            && call.args.iter().any(|argument| is_parameter(argument, self.parameter));
        syn::visit::visit_expr_method_call(self, call);
    }
    fn visit_expr_call(&mut self, call: &'ast syn::ExprCall) {
        self.independent |=
            self.visible && call.args.iter().any(|argument| is_parameter(argument, self.parameter));
        syn::visit::visit_expr_call(self, call);
    }
    fn visit_macro(&mut self, invocation: &'ast syn::Macro) {
        self.independent |=
            self.visible && tokens_use_parameter(invocation.tokens.clone(), self.parameter);
    }
    fn visit_block(&mut self, block: &'ast syn::Block) {
        let visible = self.visible;
        syn::visit::visit_block(self, block);
        self.visible = visible;
    }
    fn visit_local(&mut self, local: &'ast syn::Local) {
        if let Some(initializer) = &local.init {
            self.visit_expr(&initializer.expr);
            if let Some((_, otherwise)) = &initializer.diverge {
                self.visit_expr(otherwise);
            }
        }
        self.visible &= !binds_parameter(&local.pat, self.parameter);
    }
    fn visit_expr_closure(&mut self, closure: &'ast syn::ExprClosure) {
        let visible = self.visible;
        self.visible &= !closure.inputs.iter().any(|input| binds_parameter(input, self.parameter));
        self.visit_expr(&closure.body);
        self.visible = visible;
    }
    fn visit_expr_for_loop(&mut self, expression: &'ast syn::ExprForLoop) {
        let visible = self.visible;
        self.visit_expr(&expression.expr);
        self.visible &= !binds_parameter(&expression.pat, self.parameter);
        self.visit_block(&expression.body);
        self.visible = visible;
    }
    fn visit_arm(&mut self, arm: &'ast syn::Arm) {
        let visible = self.visible;
        self.visible &= !binds_parameter(&arm.pat, self.parameter);
        self.visit_pat(&arm.pat);
        self.visit_expr(&arm.body);
        self.visible = visible;
    }
    fn visit_expr_let(&mut self, expression: &'ast syn::ExprLet) {
        self.visit_expr(&expression.expr);
        self.visible &= !binds_parameter(&expression.pat, self.parameter);
    }
    fn visit_expr_if(&mut self, expression: &'ast syn::ExprIf) {
        let visible = self.visible;
        self.visit_expr(&expression.cond);
        self.visit_block(&expression.then_branch);
        self.visible = visible;
        if let Some((_, otherwise)) = &expression.else_branch {
            self.visit_expr(otherwise);
        }
    }
    fn visit_expr_while(&mut self, expression: &'ast syn::ExprWhile) {
        let visible = self.visible;
        self.visit_expr(&expression.cond);
        self.visit_block(&expression.body);
        self.visible = visible;
    }
    fn visit_item(&mut self, _: &'ast syn::Item) {}
}

fn state_context(node: &Node, context: &mut StateContext) {
    match node {
        Node::Value(value) => {
            if let Some(closure) = state_closure(value) {
                match closure.inputs.first() {
                    Some(Pat::Type(input)) => {
                        context.authored = true;
                        if let syn::Type::Reference(reference) = input.ty.as_ref() {
                            context.explicit.get_or_insert_with(|| *reference.elem.clone());
                        }
                    }
                    Some(Pat::Ident(input)) if input.subpat.is_none() => {
                        let mut members = StateMembers {
                            parameter: &input.ident,
                            visible: true,
                            found: false,
                            independent: false,
                        };
                        syn::visit::Visit::visit_expr(&mut members, &closure.body);
                        context.needs_owner |= members.found && !members.independent;
                        context.authored |= members.independent;
                    }
                    Some(Pat::Wild(_)) => {}
                    _ => context.authored = true,
                }
            }
        }
        Node::Either(nodes) | Node::Unordered(nodes) => {
            for node in nodes {
                state_context(node, context);
            }
        }
        Node::Repeat(node, ..) => state_context(node, context),
        _ => {}
    }
}

fn leading_pattern(node: &Node, context: &StateContext) -> syn::Result<TokenStream> {
    if let Some(state) = &context.explicit {
        let pattern = pattern(node)?;
        return Ok(
            quote!(::umpire_dsl::path_build::WorldBuilder::new(::umpire_dsl::path_build::pattern_with::<#state>(|__pattern| #pattern))),
        );
    }
    let pattern = pattern(node)?;
    Ok(
        quote!(::umpire_dsl::path_build::WorldBuilder::new(::umpire_dsl::path_build::pattern_with(|__pattern| #pattern))),
    )
}

pub(crate) fn expand(tokens: TokenStream) -> syn::Result<TokenStream> {
    let Nodes(nodes) = syn::parse2(tokens)?;
    let mut context = StateContext::default();
    for node in &nodes {
        state_context(node, &mut context);
    }
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
    if !matches!(&nodes[0], Node::Value(_)) && context.explicit.is_none() && context.needs_owner {
        let mut built = quote!(::umpire_dsl::path_build::PatternPiece::empty());
        for node in &nodes {
            built = piece(node, &built, false)?;
        }
        return Ok(quote!((#built).finish()));
    }
    let first = if matches!(&nodes[0], Node::Value(_)) {
        quote!(__path)
    } else {
        leading_pattern(&nodes[0], &context)?
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
