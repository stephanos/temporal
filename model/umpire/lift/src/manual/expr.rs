use super::{Manual, path_name, refusal, single_statement, tail};
use crate::tree::{Expr, ExprKind, Op, Unary};
use crate::{Diagnostic, Type};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{Expr as RustExpr, spanned::Spanned};

pub(super) fn scalar_type(ty: &syn::Type) -> Option<Type> {
    let syn::Type::Path(path) = ty else {
        return None;
    };
    if path.qself.is_some() || path.path.leading_colon.is_some() || path.path.segments.len() != 1 {
        return None;
    }
    Some(match path.path.segments.last()?.ident.to_string().as_str() {
        "bool" => Type::Bool,
        "u8" => Type::Int { min: 0, max: 255 },
        "i32" => Type::Int { min: i128::from(i32::MIN), max: i128::from(i32::MAX) },
        _ => return None,
    })
}

pub(super) fn result_type(file: &str, ty: &syn::Type) -> Result<Type, Diagnostic> {
    match ty {
        syn::Type::Tuple(tuple) if tuple.elems.is_empty() => Ok(Type::Unit),
        syn::Type::Path(path) if path.path.is_ident("bool") => Ok(Type::Bool),
        syn::Type::Reference(reference) if matches!(reference.elem.as_ref(), syn::Type::Path(path) if path.path.is_ident("str")) => {
            Ok(Type::Text)
        }
        _ => Err(refusal(file, ty, "unsupported manual outcome or fact type")),
    }
}

pub(super) fn integer(value: i128) -> Expr {
    Expr::int(value, Type::Int { min: i128::from(i32::MIN), max: i128::from(i32::MAX) })
}

pub(super) fn constant_bool(expression: &Expr) -> Option<bool> {
    match &expression.kind {
        ExprKind::Bool(value) => Some(*value),
        ExprKind::Unary(Unary::Not, value) => Some(!constant_bool(value)?),
        ExprKind::Binary(Op::Or, left, right) => {
            Some(constant_bool(left)? || constant_bool(right)?)
        }
        ExprKind::Binary(Op::And, left, right) => {
            Some(constant_bool(left)? && constant_bool(right)?)
        }
        ExprKind::Binary(Op::Eq, left, right)
            if matches!(left.kind, ExprKind::Text(_))
                && matches!(right.kind, ExprKind::Text(_)) =>
        {
            Some(left == right)
        }
        _ => None,
    }
}

impl Manual<'_> {
    pub(super) fn pattern(&self, pattern: &syn::Pat, value: Expr) -> Result<Expr, Diagnostic> {
        match pattern {
            syn::Pat::Wild(_) => Ok(Expr::bool(true)),
            syn::Pat::Or(pattern) => {
                pattern.cases.iter().try_fold(Expr::bool(false), |guard, pattern| {
                    Ok(guard.or(self.pattern(pattern, value.clone())?))
                })
            }
            syn::Pat::Lit(pattern) => {
                let expression = RustExpr::Lit(pattern.clone());
                Ok(Expr::binary(Op::Eq, value, self.expr(&expression, &BTreeMap::new())?))
            }
            syn::Pat::Path(pattern) => Ok(Expr::binary(
                Op::Eq,
                value,
                Expr::text(pattern.path.to_token_stream().to_string().replace(' ', "")),
            )),
            _ => Err(refusal(self.file, pattern, "unsupported state match pattern")),
        }
    }

    #[allow(clippy::too_many_lines)]
    pub(super) fn expr(
        &self,
        expression: &RustExpr,
        aliases: &BTreeMap<String, String>,
    ) -> Result<Expr, Diagnostic> {
        match expression {
            RustExpr::Paren(expression) => self.expr(&expression.expr, aliases),
            RustExpr::Reference(expression) => self.expr(&expression.expr, aliases),
            RustExpr::Block(expression) => {
                single_statement(self.file, &expression.block)?;
                self.expr(
                    tail(&expression.block)
                        .ok_or_else(|| refusal(self.file, expression, "empty predicate"))?,
                    aliases,
                )
            }
            RustExpr::MethodCall(expression)
                if expression.method == "into" || expression.method == "clone" =>
            {
                self.expr(&expression.receiver, aliases)
            }
            RustExpr::Lit(expression) => match &expression.lit {
                syn::Lit::Bool(value) => Ok(Expr::bool(value.value)),
                syn::Lit::Int(value) => value
                    .base10_parse::<i128>()
                    .and_then(|value| {
                        if value <= i128::from(i32::MAX) {
                            Ok(integer(value))
                        } else {
                            Err(syn::Error::new(
                                expression.span(),
                                "integer exceeds the admitted signed 32-bit TLC range",
                            ))
                        }
                    })
                    .map_err(|error| refusal(self.file, expression, &error.to_string())),
                syn::Lit::Str(value) => Ok(Expr::text(value.value())),
                _ => Err(refusal(self.file, expression, "unsupported literal")),
            },
            RustExpr::Path(expression) => {
                let name = expression.path.to_token_stream().to_string().replace(' ', "");
                if name == "Verdict::Holds" {
                    return Ok(Expr::bool(true));
                }
                if name == "Verdict::Fails" {
                    return Ok(Expr::bool(false));
                }
                if let Some(value) = self.constants.get(&name) {
                    return Ok(value.clone());
                }
                if let Some(alias) = aliases.get(&name) {
                    if alias.contains("::") {
                        return Ok(Expr::text(alias));
                    }
                    if alias == "state" || alias == "after" {
                        let fields = self
                            .fields
                            .iter()
                            .map(|field| Expr::var(&field.name, field.ty.clone(), alias == "after"))
                            .collect::<Vec<_>>();
                        return Ok(Expr {
                            ty: Type::Tuple(fields.iter().map(|field| field.ty.clone()).collect()),
                            kind: ExprKind::Tuple(fields),
                        });
                    }
                    return Ok(match alias.as_str() {
                        "outcome" => Expr::var("__umpire_outcome", Type::Bool, true),
                        "facts" => {
                            Expr::var("__umpire_facts", Type::Seq(Box::new(Type::Text)), true)
                        }
                        _ => Expr::var(alias, Type::Text, false),
                    });
                }
                if self.variants.contains(&name) {
                    Ok(Expr::text(name))
                } else {
                    Err(refusal(
                        self.file,
                        expression,
                        &format!("unresolved expression path {name}"),
                    ))
                }
            }
            RustExpr::Field(expression) => {
                let RustExpr::Path(base) = expression.base.as_ref() else {
                    return Err(refusal(
                        self.file,
                        expression,
                        "nested field access is unsupported",
                    ));
                };
                let base = base.path.to_token_stream().to_string();
                let role = aliases.get(&base).map_or(base.as_str(), String::as_str);
                if role == "world" || base == "self" {
                    return Ok(Expr::text(self.mode.clone().ok_or_else(|| {
                        refusal(self.file, expression, "unbound world parameter")
                    })?));
                }
                if role != "state" && role != "after" && role != "before" {
                    return Err(refusal(
                        self.file,
                        expression,
                        "field base must be the declared state callback parameter",
                    ));
                }
                let name = match &expression.member {
                    syn::Member::Named(name) => name.to_string(),
                    syn::Member::Unnamed(index) => format!("field_{}", index.index),
                };
                let field = self
                    .fields
                    .iter()
                    .find(|field| field.name == name)
                    .ok_or_else(|| refusal(self.file, expression, "unknown state field"))?;
                Ok(Expr::var(name, field.ty.clone(), role == "after"))
            }
            RustExpr::Unary(expression) => {
                if matches!(expression.op, syn::UnOp::Neg(_))
                    && let RustExpr::Lit(literal) = expression.expr.as_ref()
                    && let syn::Lit::Int(value) = &literal.lit
                {
                    let value = value
                        .base10_parse::<i128>()
                        .map_err(|error| refusal(self.file, expression, &error.to_string()))?;
                    if value <= -i128::from(i32::MIN) {
                        return Ok(integer(-value));
                    }
                    return Err(refusal(
                        self.file,
                        expression,
                        "negative integer exceeds the admitted signed 32-bit TLC range",
                    ));
                }
                let value = self.expr(&expression.expr, aliases)?;
                if matches!(expression.op, syn::UnOp::Not(_)) && value.ty != Type::Bool
                    || matches!(expression.op, syn::UnOp::Neg(_))
                        && !matches!(value.ty, Type::Int { .. })
                {
                    return Err(refusal(self.file, expression, "unary operand has the wrong type"));
                }
                Ok(match expression.op {
                    syn::UnOp::Deref(_) => value,
                    syn::UnOp::Not(_) => value.not(),
                    syn::UnOp::Neg(_) => Expr {
                        ty: value.ty.clone(),
                        kind: ExprKind::Unary(Unary::Neg, Box::new(value)),
                    },
                    _ => return Err(refusal(self.file, expression, "unsupported unary operator")),
                })
            }
            RustExpr::Binary(expression) => {
                let op = match expression.op {
                    syn::BinOp::Add(_) => Op::Add,
                    syn::BinOp::Sub(_) => Op::Sub,
                    syn::BinOp::Mul(_) => Op::Mul,
                    syn::BinOp::Div(_) => Op::Div,
                    syn::BinOp::Rem(_) => Op::Rem,
                    syn::BinOp::And(_) => Op::And,
                    syn::BinOp::Or(_) => Op::Or,
                    syn::BinOp::Eq(_) => Op::Eq,
                    syn::BinOp::Ne(_) => Op::Ne,
                    syn::BinOp::Lt(_) => Op::Lt,
                    syn::BinOp::Le(_) => Op::Le,
                    syn::BinOp::Gt(_) => Op::Gt,
                    syn::BinOp::Ge(_) => Op::Ge,
                    _ => return Err(refusal(self.file, expression, "unsupported binary operator")),
                };
                let mut left = self.expr(&expression.left, aliases)?;
                let mut right = self.expr(&expression.right, aliases)?;
                if matches!((&left.ty, &right.ty), (Type::Int { .. }, Type::Int { .. })) {
                    if unsuffixed_literal(&expression.left) {
                        left.ty = right.ty.clone();
                    }
                    if unsuffixed_literal(&expression.right) {
                        right.ty = left.ty.clone();
                    }
                    if left.ty != right.ty {
                        return Err(refusal(
                            self.file,
                            expression,
                            "integer operands require one resolved Rust scalar type",
                        ));
                    }
                }
                let valid = match op {
                    Op::Add
                    | Op::Sub
                    | Op::Mul
                    | Op::Div
                    | Op::Rem
                    | Op::Lt
                    | Op::Le
                    | Op::Gt
                    | Op::Ge => {
                        matches!((&left.ty, &right.ty), (Type::Int { .. }, Type::Int { .. }))
                    }
                    Op::And | Op::Or => left.ty == Type::Bool && right.ty == Type::Bool,
                    Op::Eq | Op::Ne => left.ty == right.ty || same_scalar_type(&left.ty, &right.ty),
                };
                if !valid {
                    return Err(refusal(
                        self.file,
                        expression,
                        "binary operands have incompatible types",
                    ));
                }
                if let (ExprKind::Text(left), ExprKind::Text(right)) = (&left.kind, &right.kind)
                    && matches!(op, Op::Eq | Op::Ne)
                {
                    return Ok(Expr::bool((left == right) == (op == Op::Eq)));
                }
                Ok(Expr::binary(op, left, right))
            }
            RustExpr::Tuple(expression) if expression.elems.is_empty() => Ok(Expr::unit()),
            RustExpr::Array(expression) => {
                let values = expression
                    .elems
                    .iter()
                    .map(|item| self.expr(item, aliases))
                    .collect::<Result<Vec<_>, _>>()?;
                Ok(Expr {
                    ty: Type::Seq(Box::new(
                        values.first().map_or(Type::Unit, |item| item.ty.clone()),
                    )),
                    kind: ExprKind::Tuple(values),
                })
            }
            RustExpr::Match(expression) => {
                let selected = self.select(&RustExpr::Match(expression.clone()))?.clone();
                self.expr(&selected, aliases)
            }
            RustExpr::Macro(expression) if expression.mac.path.is_ident("matches") => {
                use syn::parse::Parser;
                let parser = |input: syn::parse::ParseStream<'_>| {
                    let value: RustExpr = input.parse()?;
                    input.parse::<syn::Token![,]>()?;
                    let pattern = syn::Pat::parse_multi(input)?;
                    Ok((value, pattern))
                };
                let (value, pattern) = parser
                    .parse2(expression.mac.tokens.clone())
                    .map_err(|error| refusal(self.file, expression, &error.to_string()))?;
                let value = self.expr(&value, aliases)?;
                let patterns = pattern_names(&pattern);
                if patterns.is_empty() {
                    return Err(refusal(self.file, &pattern, "unsupported matches pattern"));
                }
                if patterns.iter().any(|pattern| pattern == "_") {
                    return Ok(Expr::bool(true));
                }
                Ok(patterns.iter().fold(Expr::bool(false), |result, name| {
                    result.or(Expr::binary(Op::Eq, value.clone(), Expr::text(name)))
                }))
            }
            expression => Err(refusal(
                self.file,
                expression,
                &format!("unsupported expression {}", expression.to_token_stream()),
            )),
        }
    }

    pub(super) fn state(
        &self,
        expression: &RustExpr,
        aliases: &BTreeMap<String, String>,
    ) -> Result<BTreeMap<String, Expr>, Diagnostic> {
        let values: BTreeMap<String, Expr> = match expression {
            RustExpr::Call(expression)
                if expression.args.len() == self.fields.len()
                    && path_name(&expression.func).as_deref() == Some(self.state_name.as_str()) =>
            {
                self.fields
                    .iter()
                    .zip(&expression.args)
                    .map(|(field, expression)| {
                        Ok((field.name.clone(), self.expr(expression, aliases)?))
                    })
                    .collect()
            }
            RustExpr::Struct(expression)
                if expression.path.is_ident(&self.state_name) && expression.rest.is_none() =>
            {
                expression
                    .fields
                    .iter()
                    .map(|field| {
                        Ok((
                            field.member.to_token_stream().to_string(),
                            self.expr(&field.expr, aliases)?,
                        ))
                    })
                    .collect()
            }
            RustExpr::MethodCall(expression)
                if expression.method == "clone"
                    && expression.args.is_empty()
                    && path_name(&expression.receiver).as_deref() == Some("state") =>
            {
                Ok(self
                    .fields
                    .iter()
                    .map(|field| {
                        (field.name.clone(), Expr::var(&field.name, field.ty.clone(), false))
                    })
                    .collect())
            }
            expression => Err(refusal(self.file, expression, "unsupported state constructor")),
        }?;
        if values.len() != self.fields.len()
            || self.fields.iter().any(|field| {
                values.get(&field.name).is_none_or(|value| !same_scalar_type(&field.ty, &value.ty))
            })
        {
            return Err(refusal(
                self.file,
                expression,
                "state constructor fields do not match the declared state type",
            ));
        }
        Ok(values)
    }
}

pub(super) fn same_scalar_type(left: &Type, right: &Type) -> bool {
    matches!((left, right), (Type::Bool, Type::Bool) | (Type::Int { .. }, Type::Int { .. }))
}

fn unsuffixed_literal(expression: &RustExpr) -> bool {
    match expression {
        RustExpr::Paren(expression) => unsuffixed_literal(&expression.expr),
        RustExpr::Unary(expression) if matches!(expression.op, syn::UnOp::Neg(_)) => {
            unsuffixed_literal(&expression.expr)
        }
        RustExpr::Lit(expression) => {
            matches!(&expression.lit, syn::Lit::Int(value) if value.suffix().is_empty())
        }
        _ => false,
    }
}

pub(super) fn pattern_names(pattern: &syn::Pat) -> Vec<String> {
    match pattern {
        syn::Pat::Guard(pattern) => pattern_names(&pattern.pat),
        syn::Pat::Or(pattern) => pattern.cases.iter().flat_map(pattern_names).collect(),
        syn::Pat::Path(pattern) => {
            vec![pattern.path.to_token_stream().to_string().replace(' ', "")]
        }
        syn::Pat::Wild(_) => vec!["_".into()],
        _ => Vec::new(),
    }
}

pub(super) fn constant_int(expression: &Expr) -> Option<i128> {
    match &expression.kind {
        ExprKind::Int(value) => Some(*value),
        ExprKind::Unary(Unary::Neg, value) => constant_int(value)?.checked_neg(),
        _ => None,
    }
}
