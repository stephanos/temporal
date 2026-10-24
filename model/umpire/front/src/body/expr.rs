use super::{Env, Environment, Result, tail};
use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind, Op, Unary},
    syntax::path_name,
};
use quote::ToTokens;
use syn::{Expr as RustExpr, Pat};

impl Environment {
    pub(crate) fn require(
        &self,
        node: &impl ToTokens,
        value: Expr,
        expected: Option<&Type>,
    ) -> Result<Expr> {
        if expected.is_some_and(|t| *t != value.ty) {
            return self
                .err(node, format!("type mismatch: expected {expected:?}, got {:?}", value.ty));
        }
        Ok(value)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Expression admission follows the syn expression variants"
    )]
    pub(crate) fn expr(
        &self,
        node: &RustExpr,
        state: &Env,
        locals: &Env,
        expected: Option<&Type>,
        depth: usize,
    ) -> Result<Expr> {
        if depth > 32 {
            return self.err(node, "recursive helper expansion is unsupported");
        }
        if let Some((literal, negative)) = integer_literal(node) {
            let ty = if literal.suffix().is_empty() {
                expected
                    .filter(|ty| matches!(ty, Type::Int { .. }))
                    .cloned()
                    .unwrap_or(Type::Int { min: i128::from(i32::MIN), max: i128::from(i32::MAX) })
            } else {
                self.ty(&syn::parse_str::<syn::Type>(literal.suffix())
                    .map_err(|error| Diagnostic::span(error.span(), error.to_string()))?)
                    .map_err(|error| Diagnostic::at(node, error.message))?
            };
            let mut number = literal
                .base10_parse::<i128>()
                .map_err(|error| Diagnostic::at(node, error.to_string()))?;
            if negative {
                number = -number;
            }
            if let Type::Int { min, max } = ty
                && (number < min || number > max)
            {
                return self.err(node, "integer literal exceeds its declared type");
            }
            return self.require(node, Expr::int(number, ty), expected);
        }
        let value = match node {
            RustExpr::Lit(value) => match &value.lit {
                syn::Lit::Bool(v) => Expr::bool(v.value),
                syn::Lit::Int(v) => {
                    let ty = if v.suffix().is_empty() {
                        expected.filter(|t| matches!(t, Type::Int { .. })).cloned().unwrap_or(
                            Type::Int { min: i128::from(i32::MIN), max: i128::from(i32::MAX) },
                        )
                    } else {
                        self.ty(&syn::parse_str::<syn::Type>(v.suffix())
                            .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?)?
                    };
                    let n =
                        v.base10_parse::<i128>().map_err(|e| Diagnostic::at(v, e.to_string()))?;
                    if let Type::Int { min, max } = ty
                        && (n < min || n > max)
                    {
                        return self.err(node, "integer literal exceeds its declared type");
                    }
                    Expr::int(n, ty)
                }
                syn::Lit::Str(v) => Expr::text(v.value()),
                _ => return self.err(node, "unsupported literal"),
            },
            RustExpr::Paren(v) => return self.expr(&v.expr, state, locals, expected, depth),
            RustExpr::Group(v) => return self.expr(&v.expr, state, locals, expected, depth),
            RustExpr::Path(path) => {
                if let Some(value) =
                    self.optional_constructor(path, None, state, locals, expected, depth)?
                {
                    return self.require(node, value, expected);
                }
                let name = path_name(&path.path.to_token_stream().to_string());
                if locals.world.as_ref() == Some(&name)
                    || locals.states.contains(&name)
                    || locals.outcome.as_ref() == Some(&name)
                {
                    return self.err(node, "complete World, state and borrowed outcome parameters are not scalar expressions");
                }
                if let Some(value) = locals.get(&name).or_else(|| self.constants.get(&name)) {
                    value.clone()
                } else if let Some((owner, variant)) = name.split_once("::") {
                    let owner = if owner == "Self" {
                        match expected {
                            Some(Type::Enum { name, .. }) => name.as_str(),
                            _ => return self.err(node, "Self enum variant has no type context"),
                        }
                    } else {
                        owner
                    };
                    let full = format!("{owner}::{variant}");
                    let ty = self
                        .enums
                        .get(owner)
                        .ok_or_else(|| Diagnostic::at(node, format!("unknown name `{name}`")))?;
                    if let Type::Enum { variants, .. } = ty
                        && !variants.contains(&full)
                    {
                        return self.err(node, format!("unknown enum variant `{full}`"));
                    }
                    Expr { kind: ExprKind::Enum(full), ty: ty.clone() }
                } else {
                    return self.err(node, format!("unresolved name `{name}`"));
                }
            }
            RustExpr::Field(field) => {
                let key = path_name(&node.to_token_stream().to_string());
                if let Some(value) = locals.get(&key) {
                    value.clone()
                } else if matches!(field.base.as_ref(),RustExpr::Path(p)if p.path.is_ident("self"))
                {
                    let member = field.member.to_token_stream().to_string();
                    state.get(&member).cloned().ok_or_else(|| {
                        Diagnostic::at(node, format!("unknown state field `{member}`"))
                    })?
                } else {
                    let base = self.expr(&field.base, state, locals, None, depth)?;
                    let Type::Bounded(descriptor) = &base.ty else {
                        return self.err(node, format!("unresolved field `{key}`"));
                    };
                    let member = match &field.member {
                        syn::Member::Unnamed(index) => {
                            descriptor.field.is_none() && index.index == 0
                        }
                        syn::Member::Named(name) => {
                            descriptor.field.as_deref() == Some(name.to_string().as_str())
                        }
                    };
                    if !member {
                        return self.err(node, "unknown bounded payload field");
                    }
                    Expr { ty: descriptor.backing_type(), kind: ExprKind::Payload(Box::new(base)) }
                }
            }
            RustExpr::Unary(unary) => {
                if matches!(unary.op, syn::UnOp::Deref(_))
                    && matches!(unary.expr.as_ref(), RustExpr::Path(path) if locals.outcome.as_ref().is_some_and(|name| path.path.is_ident(name)))
                {
                    let value = locals
                        .outcome
                        .as_ref()
                        .and_then(|name| locals.get(name))
                        .cloned()
                        .ok_or_else(|| Diagnostic::at(node, "missing borrowed outcome binding"))?;
                    return self.require(node, value, expected);
                }
                let op = super::typer::unary_op(&unary.op)
                    .ok_or_else(|| Diagnostic::at(node, "unsupported unary operator"))?;
                let value = self.expr(
                    &unary.expr,
                    state,
                    locals,
                    if op == Unary::Not { Some(&Type::Bool) } else { expected },
                    depth,
                )?;
                if op == Unary::Neg && !matches!(value.ty, Type::Int { .. }) {
                    return self.err(node, "negation requires an integer");
                }
                Expr { ty: value.ty.clone(), kind: ExprKind::Unary(op, Box::new(value)) }
            }
            RustExpr::Binary(binary) => {
                let op = super::typer::binary_op(&binary.op)
                    .ok_or_else(|| Diagnostic::at(node, "unsupported binary operator"))?;
                if let (Some(left), Some(right)) = (
                    self.whole(&binary.left, state, locals),
                    self.whole(&binary.right, state, locals),
                ) && matches!(op, Op::Eq | Op::Ne)
                {
                    let equal = self.fields.iter().fold(Expr::bool(true), |all, field| {
                        all.and(Expr::binary(
                            Op::Eq,
                            left[&field.name].clone(),
                            right[&field.name].clone(),
                        ))
                    });
                    let value = if op == Op::Eq { equal } else { equal.not() };
                    return self.require(node, value, expected);
                }
                let bool_op = matches!(op, Op::And | Op::Or);
                let arithmetic = matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem);
                let context = if bool_op {
                    Some(&Type::Bool)
                } else if arithmetic {
                    expected
                } else {
                    None
                };
                let (left, right) = if context.is_none()
                    && (integer_literal(&binary.left)
                        .is_some_and(|(literal, _)| literal.suffix().is_empty())
                        || matches!(binary.left.as_ref(), RustExpr::Path(path) if path.path.segments.last().is_some_and(|s| s.ident == "None")))
                    && integer_literal(&binary.right).is_none()
                {
                    let right = self.expr(&binary.right, state, locals, None, depth)?;
                    (self.expr(&binary.left, state, locals, Some(&right.ty), depth)?, right)
                } else {
                    let left = self.expr(&binary.left, state, locals, context, depth)?;
                    let right = self.expr(&binary.right, state, locals, Some(&left.ty), depth)?;
                    (left, right)
                };
                if (arithmetic || matches!(op, Op::Lt | Op::Le | Op::Gt | Op::Ge))
                    && !matches!(left.ty, Type::Int { .. })
                {
                    return self.err(node, "integer operator applied to a non-integer");
                }
                // Rust computes lengths in `usize`, TLC in 32-bit integers: a sum or product can
                // exceed TLC's range where Rust's holds. A difference, quotient or remainder of
                // two in-range lengths stays in range, so every back end reads it alike.
                if matches!(op, Op::Add | Op::Mul) && left.ty == super::vector::index_type() {
                    return self.err(
                        node,
                        "`+` and `*` on a Vector length are refused: Rust computes lengths in \
                         usize and TLC in 32-bit integers, which can disagree; compare the length \
                         instead",
                    );
                }
                Expr::binary(op, left, right)
            }
            RustExpr::If(condition) => {
                let cond = self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                let yes =
                    self.expr(tail(&condition.then_branch)?, state, locals, expected, depth)?;
                let Some((_, no)) = &condition.else_branch else {
                    return self.err(node, "value-producing if requires else");
                };
                let no = self.expr(no, state, locals, Some(&yes.ty), depth)?;
                Expr::select(cond, yes, no)
            }
            RustExpr::Block(block) => {
                return self.expr(tail(&block.block)?, state, locals, expected, depth);
            }
            RustExpr::Tuple(tuple) => {
                if tuple.elems.is_empty() {
                    Expr::unit()
                } else {
                    let values = tuple
                        .elems
                        .iter()
                        .map(|v| self.expr(v, state, locals, None, depth))
                        .collect::<Result<Vec<_>>>()?;
                    Expr {
                        ty: Type::Tuple(values.iter().map(|v| v.ty.clone()).collect()),
                        kind: ExprKind::Tuple(values),
                    }
                }
            }
            RustExpr::Call(call) if self.is_rejected(call) => {
                self.rejected_call(call, state, locals, depth)?
            }
            RustExpr::Struct(value) => {
                let name = value.path.get_ident().map(ToString::to_string);
                let Some(descriptor) = name.as_ref().and_then(|name| self.bounded.get(name)) else {
                    return self.err(node, "unsupported struct construction");
                };
                if locals.contains_key(&descriptor.name)
                    || value.rest.is_some()
                    || value.fields.len() != 1
                {
                    return self
                        .err(node, "bounded named construction requires its one payload field");
                }
                let member = &value.fields[0];
                if !matches!(&member.member, syn::Member::Named(name) if descriptor.field.as_deref() == Some(name.to_string().as_str()))
                {
                    return self.err(node, "unknown bounded payload field");
                }
                let payload = self.expr(
                    &member.expr,
                    state,
                    locals,
                    Some(&descriptor.backing_type()),
                    depth,
                )?;
                Expr {
                    ty: Type::Bounded(descriptor.clone()),
                    kind: ExprKind::Bounded(Box::new(payload)),
                }
            }
            RustExpr::Call(call) => {
                if let RustExpr::Path(path) = call.func.as_ref()
                    && call.args.len() <= 1
                    && let Some(value) = self.optional_constructor(
                        path,
                        call.args.first(),
                        state,
                        locals,
                        expected,
                        depth,
                    )?
                {
                    return self.require(node, value, expected);
                }
                if let Some(value) = self.bounded_call(call, state, locals, depth)? {
                    return self.require(node, value, expected);
                }
                match self.vector_constructor(call, expected)? {
                    Some(value) => value,
                    None => {
                        return self.err(
                            node,
                            format!("unsupported expression `{}`", node.to_token_stream()),
                        );
                    }
                }
            }
            RustExpr::MethodCall(call) => {
                if let Some(value) = self.vector_method(call, state, locals, expected, depth)? {
                    return self.require(node, value, expected);
                }
                if let Some(receiver) = self.whole(&call.receiver, state, locals)
                    && let Some(method) = self.state_method(&call.method.to_string())
                {
                    if method.sig.receiver().is_none_or(|r| r.mutability.is_some()) {
                        return self.err(node, "only pure helpers are admitted in expressions");
                    }
                    return self.expr(
                        tail(&method.block)?,
                        &receiver,
                        &self.helper_env(&method, &call.args, locals)?,
                        expected,
                        depth + 1,
                    );
                }
                let receiver = self.expr(&call.receiver, state, locals, None, depth)?;
                if matches!(receiver.ty, Type::Optional(_)) {
                    let value = self.optional_method(call, receiver, state, locals, depth)?;
                    return self.require(node, value, expected);
                }
                if call.method == "into" && call.args.is_empty() && receiver.ty == Type::Bool {
                    return self.require(node, receiver, expected);
                }
                let Type::Enum { name, .. } = &receiver.ty else {
                    return self.err(node, format!("unsupported method `{}`", call.method));
                };
                let method =
                    self.helpers.get(&format!("{name}::{}", call.method)).ok_or_else(|| {
                        Diagnostic::at(
                            node,
                            format!("unsupported method `{name}::{}`", call.method),
                        )
                    })?;
                if !call.args.is_empty() || method.sig.inputs.len() != 1 {
                    return self.err(node, "only parameterless pure enum helpers are admitted");
                }
                let mut env = locals.clone();
                env.insert("self".into(), receiver);
                self.expr(tail(&method.block)?, state, &env, expected, depth + 1)?
            }
            RustExpr::Macro(mac) if mac.mac.path.is_ident("matches") => {
                return self.matches(mac, state, locals, depth);
            }
            _ => {
                return self
                    .err(node, format!("unsupported expression `{}`", node.to_token_stream()));
            }
        };
        self.require(node, value, expected)
    }
    fn matches(
        &self,
        mac: &syn::ExprMacro,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Expr> {
        struct Args {
            value: RustExpr,
            pat: Pat,
        }
        impl syn::parse::Parse for Args {
            fn parse(input: syn::parse::ParseStream<'_>) -> syn::Result<Self> {
                let value = input.parse()?;
                input.parse::<syn::Token![,]>()?;
                let pat = Pat::parse_multi_with_leading_vert(input)?;
                if !input.is_empty() {
                    input.parse::<syn::Token![,]>()?;
                }
                Ok(Self { value, pat })
            }
        }
        let args: syn::Result<Args> = syn::parse2(mac.mac.tokens.clone());
        let args = args.map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
        let value = self.expr(&args.value, state, locals, None, depth)?;
        if matches!(value.ty, Type::Optional(_)) {
            return self.err(mac, "optional patterns are unsupported; use presence tests");
        }
        self.pattern(&args.pat, &value)
    }
    fn pattern(&self, pat: &Pat, value: &Expr) -> Result<Expr> {
        match pat {
            Pat::Wild(_) => Ok(Expr::bool(true)),
            Pat::Or(or) => {
                let mut out = Expr::bool(false);
                for case in &or.cases {
                    out = out.or(self.pattern(case, value)?);
                }
                Ok(out)
            }
            Pat::Path(path) => {
                let path: RustExpr = syn::parse2(path.to_token_stream())
                    .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
                Ok(Expr::binary(
                    Op::Eq,
                    value.clone(),
                    self.expr(&path, &Env::new(), &Env::new(), Some(&value.ty), 0)?,
                ))
            }
            _ => self.err(pat, "unsupported match pattern"),
        }
    }
}

fn integer_literal(node: &RustExpr) -> Option<(&syn::LitInt, bool)> {
    match node {
        RustExpr::Lit(syn::ExprLit { lit: syn::Lit::Int(literal), .. }) => Some((literal, false)),
        RustExpr::Unary(unary) if matches!(unary.op, syn::UnOp::Neg(_)) => {
            match unary.expr.as_ref() {
                RustExpr::Lit(syn::ExprLit { lit: syn::Lit::Int(literal), .. }) => {
                    Some((literal, true))
                }
                _ => None,
            }
        }
        _ => None,
    }
}
