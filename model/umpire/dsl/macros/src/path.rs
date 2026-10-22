use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Expr, Ident, Pat, Path, ext::IdentExt};
use umpire_front::path::{Domain, Node, Nodes};

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
