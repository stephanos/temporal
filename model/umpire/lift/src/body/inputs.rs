use super::{Env, Environment, Result};
use crate::{
    CATALOG_LIMIT, Diagnostic, Type,
    expr::{Expr, ExprKind, Op},
    syntax::{attr, path_name},
    tree::{ActionDef, ActionRef, InputDomain, InputField, Shape},
};
use quote::ToTokens;
use syn::{Expr as RustExpr, Pat};

/// The condition under which an arm's pattern matches one concrete variant, and the payload
/// names it binds.
pub(crate) struct Matched {
    pub(crate) condition: Expr,
    pub(crate) bindings: Vec<(String, Expr)>,
}

impl Environment<'_> {
    /// Admit one payload field and its finite domain, independently of native enumeration.
    pub(crate) fn input_field(
        &self,
        field: &syn::Field,
        index: usize,
        action: &str,
        symbol: String,
    ) -> Result<InputField> {
        for attribute in &field.attrs {
            let name = attribute.path().to_token_stream().to_string();
            if !["doc", "inputs", "domain"].contains(&name.as_str()) {
                return self.err(attribute, format!("unsupported attribute `{name}` on an input"));
            }
        }
        let name = field.ident.as_ref().map_or_else(|| index.to_string(), ToString::to_string);
        let context = format!("{action} field `{name}`");
        let ty = self.ty(&field.ty)?;
        if !matches!(ty, Type::Bool | Type::Int { .. } | Type::Enum { .. }) {
            return self.err(field, format!("{context}: inputs require admitted scalar types"));
        }
        let count = |name| field.attrs.iter().filter(|a| a.path().is_ident(name)).count();
        if count("domain") > 1 || count("inputs") > 1 {
            return self.err(field, format!("{context} declares its domain more than once"));
        }
        let domain = if let Some(domain) = attr(&field.attrs, "domain") {
            if let Some(inputs) = attr(&field.attrs, "inputs") {
                return self
                    .err(inputs, format!("{context} declares both #[inputs] and #[domain]"));
            }
            let function = domain.parse_args::<syn::Ident>().map_err(|_| {
                Diagnostic::at(
                    self.file,
                    domain,
                    format!("{context}: #[domain] names a local function"),
                )
            })?;
            let item = self.domain_functions.get(&function.to_string()).ok_or_else(|| {
                Diagnostic::at(
                    self.file,
                    domain,
                    format!("{context}: #[domain({function})] names no local function"),
                )
            })?;
            self.domain_signature(item, &ty, &context)?;
            InputDomain::World(function.to_string())
        } else if let Some(inputs) = attr(&field.attrs, "inputs") {
            let value = inputs
                .parse_args::<RustExpr>()
                .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
            InputDomain::Values(self.declared_values(inputs, &value, &ty, &context)?)
        } else {
            InputDomain::Values(match &ty {
                Type::Bool => vec![Expr::bool(false), Expr::bool(true)],
                Type::Enum { name, variants } if self.finite.contains(name) => variants
                    .iter()
                    .map(|variant| Expr { kind: ExprKind::Enum(variant.clone()), ty: ty.clone() })
                    .collect(),
                Type::Int { .. } => {
                    return self.err(
                        field,
                        format!("{context} requires #[inputs] with a bounded range or finite list"),
                    );
                }
                _ => {
                    return self
                        .err(field, format!("{context}: its type has no admitted Finite domain"));
                }
            })
        };
        Ok(InputField { name, ty, domain, symbol, location: Diagnostic::at(self.file, field, "") })
    }

    fn declared_values(
        &self,
        attribute: &syn::Attribute,
        value: &RustExpr,
        ty: &Type,
        context: &str,
    ) -> Result<Vec<Expr>> {
        match value {
            RustExpr::Range(range) if matches!(ty, Type::Int { .. }) => {
                let (Some(lo), Some(hi)) = (&range.start, &range.end) else {
                    return self
                        .err(attribute, format!("{context}: #[inputs] ranges need two bounds"));
                };
                let (ExprKind::Int(lo), ExprKind::Int(hi)) =
                    (self.literal(lo, ty)?.kind, self.literal(hi, ty)?.kind)
                else {
                    return self
                        .err(attribute, format!("{context}: range bounds must be integers"));
                };
                let hi =
                    if matches!(range.limits, syn::RangeLimits::HalfOpen(_)) { hi - 1 } else { hi };
                let size = (hi - lo + 1).max(0);
                if size > i128::try_from(CATALOG_LIMIT).unwrap_or(i128::MAX) {
                    return self.err(
                        attribute,
                        format!("{context}: #[inputs] declares {size} values, above the catalog limit {CATALOG_LIMIT}"),
                    );
                }
                Ok((lo..=hi).map(|value| Expr::int(value, ty.clone())).collect())
            }
            RustExpr::Array(array) => {
                array.elems.iter().map(|element| self.literal(element, ty)).collect()
            }
            _ => self.err(
                attribute,
                format!("{context}: #[inputs] requires a bounded integer range or a finite list"),
            ),
        }
    }

    /// A typed literal or literal source constant; other expressions refuse.
    fn literal(&self, node: &RustExpr, ty: &Type) -> Result<Expr> {
        let value = self.expr(node, &Env::new(), &Env::new(), Some(ty), 0)?;
        if !matches!(value.kind, ExprKind::Bool(_) | ExprKind::Int(_) | ExprKind::Enum(_)) {
            return self.err(node, "action input values must be literals");
        }
        Ok(value)
    }

    /// `fn name(world: &World) -> impl Iterator<Item = FieldType>`; its body is never read.
    fn domain_signature(&self, item: &syn::ItemFn, ty: &Type, context: &str) -> Result<()> {
        let signature = &item.sig;
        let message =
            format!("{context}: a #[domain] function is fn(&World) -> impl Iterator<Item = T>");
        let world = match signature.inputs.iter().collect::<Vec<_>>().as_slice() {
            [syn::FnArg::Typed(input)] => {
                matches!(input.ty.as_ref(), syn::Type::Reference(reference)
                if reference.mutability.is_none()
                    && matches!(reference.elem.as_ref(), syn::Type::Path(path) if path.qself.is_none() && path.path.is_ident("World")))
            }
            _ => false,
        };
        if !world
            || signature.asyncness.is_some()
            || matches!(signature.safety, syn::Safety::Unsafe(_))
            || signature.constness.is_some()
            || signature.variadic.is_some()
            || !signature.generics.params.is_empty()
        {
            return self.err(item, message);
        }
        let item_type = match &signature.output {
            syn::ReturnType::Type(_, output) => match output.as_ref() {
                syn::Type::ImplTrait(bounds) => {
                    match bounds.bounds.iter().collect::<Vec<_>>().as_slice() {
                        [syn::TypeParamBound::Trait(bound)] => {
                            bound.path.segments.last().and_then(|segment| {
                                if segment.ident != "Iterator" {
                                    return None;
                                }
                                let syn::PathArguments::AngleBracketed(arguments) =
                                    &segment.arguments
                                else {
                                    return None;
                                };
                                match arguments.args.iter().collect::<Vec<_>>().as_slice() {
                                    [syn::GenericArgument::AssocType(item)]
                                        if item.ident == "Item" =>
                                    {
                                        Some(&item.ty)
                                    }
                                    _ => None,
                                }
                            })
                        }
                        _ => None,
                    }
                }
                _ => None,
            },
            syn::ReturnType::Default => None,
        };
        let Some(item_type) = item_type else {
            return self.err(&signature.output, message);
        };
        if self.ty(item_type)? != *ty {
            return self.err(
                item_type,
                format!("{context}: the #[domain] item type differs from the field"),
            );
        }
        Ok(())
    }

    /// Refuse a statically declared catalog above the limit before any expansion.
    pub(crate) fn static_catalog(
        &self,
        actions: &[ActionDef],
        module: &syn::ItemMod,
    ) -> Result<()> {
        let mut total = Some(0usize);
        for action in actions {
            // World-dependent variants are counted once their selected World is bound, and an
            // empty factor empties a variant even beside an overflowing one.
            let sizes: Vec<_> = action
                .fields
                .iter()
                .map(|field| match &field.domain {
                    InputDomain::Values(values) => values.len(),
                    InputDomain::World(_) => 0,
                })
                .collect();
            let count = if sizes.contains(&0) {
                Some(0)
            } else {
                sizes.iter().try_fold(1usize, |count, size| count.checked_mul(*size))
            };
            total = total.zip(count).and_then(|(total, count)| total.checked_add(count));
        }
        match total {
            None => self.err(module, "action catalog count overflows"),
            Some(count) if count > CATALOG_LIMIT => self.err(
                module,
                format!(
                    "action catalog has {count} concrete actions, above the limit {CATALOG_LIMIT}"
                ),
            ),
            Some(_) => Ok(()),
        }
    }

    /// Whether an arm pattern can match a concrete action of this variant, and under which
    /// payload condition and bindings.
    pub(crate) fn action_pattern(&self, pat: &Pat, action: &ActionDef) -> Result<Option<Matched>> {
        let matched = |condition, bindings| Ok(Some(Matched { condition, bindings }));
        match pat {
            Pat::Wild(_) => matched(Expr::bool(true), Vec::new()),
            Pat::Path(path) => {
                if path_name(&path.path.to_token_stream().to_string()) != action.name {
                    return Ok(None);
                }
                if action.shape != Shape::Unit {
                    return self
                        .err(pat, format!("`{}` carries inputs; match its fields", action.name));
                }
                matched(Expr::bool(true), Vec::new())
            }
            Pat::TupleStruct(pattern) => {
                if path_name(&pattern.path.to_token_stream().to_string()) != action.name {
                    return Ok(None);
                }
                if action.shape != Shape::Tuple {
                    return self.err(pat, format!("`{}` is not a tuple variant", action.name));
                }
                let rest = pattern.elems.iter().position(|element| matches!(element, Pat::Rest(_)));
                let elements: Vec<_> = pattern
                    .elems
                    .iter()
                    .filter(|element| !matches!(element, Pat::Rest(_)))
                    .collect();
                if pattern.elems.len() - elements.len() > 1
                    || (rest.is_none() && elements.len() != action.fields.len())
                    || elements.len() > action.fields.len()
                {
                    return self.err(pat, format!("pattern arity differs from `{}`", action.name));
                }
                let skipped = action.fields.len() - elements.len();
                let mut fields = Vec::new();
                for (index, element) in elements.into_iter().enumerate() {
                    let position = match rest {
                        Some(rest) if index >= rest => index + skipped,
                        _ => index,
                    };
                    fields.push((&action.fields[position], element));
                }
                self.fields_pattern(&fields)
            }
            Pat::Struct(pattern) => {
                if path_name(&pattern.path.to_token_stream().to_string()) != action.name {
                    return Ok(None);
                }
                if action.shape != Shape::Named || pattern.qself.is_some() {
                    return self
                        .err(pat, format!("`{}` is not a named-field variant", action.name));
                }
                let mut fields = Vec::new();
                for field in &pattern.fields {
                    let name = field.member.to_token_stream().to_string();
                    let declared =
                        action.fields.iter().find(|f| f.name == name).ok_or_else(|| {
                            Diagnostic::at(
                                self.file,
                                field,
                                format!("`{}` has no field `{name}`", action.name),
                            )
                        })?;
                    fields.push((declared, field.pat.as_ref()));
                }
                if pattern.rest.is_none() && fields.len() != action.fields.len() {
                    return self.err(pat, format!("pattern omits fields of `{}`", action.name));
                }
                self.fields_pattern(&fields)
            }
            Pat::Or(or) => {
                let mut cases = Vec::new();
                for case in &or.cases {
                    if let Some(case) = self.action_pattern(case, action)? {
                        cases.push(case);
                    }
                }
                if cases.len() > 1 && cases.iter().any(|case| !case.bindings.is_empty()) {
                    return self.err(pat, "or-patterns binding action inputs are unsupported");
                }
                let mut cases = cases.into_iter();
                let Some(first) = cases.next() else { return Ok(None) };
                let condition = cases.fold(first.condition, |out, case| out.or(case.condition));
                matched(condition, first.bindings)
            }
            _ => self.err(pat, "unsupported action match pattern"),
        }
    }

    fn fields_pattern(&self, fields: &[(&InputField, &Pat)]) -> Result<Option<Matched>> {
        let mut condition = Expr::bool(true);
        let mut bindings = Vec::new();
        for (field, pattern) in fields {
            let value = Expr { kind: ExprKind::Input(field.symbol.clone()), ty: field.ty.clone() };
            match pattern {
                Pat::Ident(id)
                    if id.by_ref.is_none()
                        && id.mutability.is_none()
                        && id.subpat.is_none()
                        && !self.constants.contains_key(&id.ident.to_string()) =>
                {
                    bindings.push((id.ident.to_string(), value));
                }
                Pat::Wild(_) => {}
                _ => condition = condition.and(self.value_pattern(pattern, &value)?),
            }
        }
        Ok(Some(Matched { condition, bindings }))
    }

    /// Literal, constant and enum-variant input patterns compare the chosen value.
    fn value_pattern(&self, pattern: &Pat, value: &Expr) -> Result<Expr> {
        match pattern {
            Pat::Wild(_) => Ok(Expr::bool(true)),
            Pat::Or(or) => {
                let mut out = Expr::bool(false);
                for case in &or.cases {
                    out = out.or(self.value_pattern(case, value)?);
                }
                Ok(out)
            }
            Pat::Lit(_) | Pat::Path(_) | Pat::Ident(_) | Pat::Verbatim(_) => {
                let node: RustExpr = syn::parse2(pattern.to_token_stream())
                    .map_err(|_| Diagnostic::at(self.file, pattern, "unsupported input pattern"))?;
                Ok(Expr::binary(Op::Eq, value.clone(), self.literal(&node, &value.ty)?))
            }
            _ => self.err(pattern, "unsupported input pattern"),
        }
    }

    /// One exact concrete action of an assumption list: a unit variant or a constructor with
    /// literal payload values. A bare payload variant does not stand for its inputs.
    pub(crate) fn action_ref(&self, node: &RustExpr, actions: &[ActionDef]) -> Result<ActionRef> {
        let node = match node {
            RustExpr::Paren(inner) => inner.expr.as_ref(),
            node => node,
        };
        let (path, arguments): (&syn::Path, Vec<(Option<String>, &RustExpr)>) = match node {
            RustExpr::Path(path) => (&path.path, Vec::new()),
            RustExpr::Call(call) => match call.func.as_ref() {
                RustExpr::Path(path) => (&path.path, call.args.iter().map(|a| (None, a)).collect()),
                _ => return self.err(node, "unsupported action in an assumption list"),
            },
            RustExpr::Struct(value)
                if value.dot2_token.is_none() && value.rest.is_none() && value.qself.is_none() =>
            {
                (
                    &value.path,
                    value
                        .fields
                        .iter()
                        .map(|f| (Some(f.member.to_token_stream().to_string()), &f.expr))
                        .collect(),
                )
            }
            _ => return self.err(node, "unsupported action in an assumption list"),
        };
        let name = path_name(&path.to_token_stream().to_string());
        let action = actions
            .iter()
            .find(|action| action.name == name)
            .ok_or_else(|| Diagnostic::at(self.file, node, format!("unknown action `{name}`")))?;
        let shape = match node {
            RustExpr::Path(_) => Shape::Unit,
            RustExpr::Call(_) => Shape::Tuple,
            _ => Shape::Named,
        };
        if shape != action.shape {
            return self.err(
                node,
                format!("`{name}` names no concrete action; supply each declared input value"),
            );
        }
        let mut values = Vec::new();
        for (index, field) in action.fields.iter().enumerate() {
            let supplied: Vec<_> = arguments
                .iter()
                .enumerate()
                .filter(|(position, (member, _))| {
                    member.as_ref().map_or(*position == index, |member| *member == field.name)
                })
                .collect();
            let [(_, (_, value))] = supplied.as_slice() else {
                return self.err(
                    node,
                    format!("`{name}` requires exactly one value for `{}`", field.name),
                );
            };
            values.push(self.literal(value, &field.ty)?);
        }
        if arguments.len() != action.fields.len() {
            return self.err(node, format!("`{name}` receives undeclared input values"));
        }
        Ok(ActionRef { name, values })
    }
}
