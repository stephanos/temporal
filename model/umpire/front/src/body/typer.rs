use super::{Environment, Result};
use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind, Op, Unary},
    syntax::path_name,
};
use quote::ToTokens;

impl Environment {
    pub(crate) fn ty(&self, ty: &syn::Type) -> Result<Type> {
        if let Some(ty) = self.optional_type(ty)? {
            return Ok(ty);
        }
        if let Some(ty) = self.vector_type(ty)? {
            return Ok(ty);
        }
        let name = path_name(&ty.to_token_stream().to_string());
        if name == "bool" {
            return Ok(Type::Bool);
        }
        if let Some((min, max)) = crate::declare::integer(ty) {
            if min < i128::from(i32::MIN) || max > i128::from(i32::MAX) {
                return self.err(
                    ty,
                    format!(
                        "integer type `{name}` is wider than the admitted signed 32-bit integers"
                    ),
                );
            }
            return Ok(Type::Int { min, max });
        }
        if let Some(value) = self.bounded.get(&name)
            && matches!(ty, syn::Type::Path(path) if path.path.is_ident(&name))
        {
            return Ok(Type::Bounded(value.clone()));
        }
        self.enums
            .get(&name)
            .cloned()
            .ok_or_else(|| Diagnostic::at(ty, format!("unsupported type `{name}`")))
    }
    pub(crate) fn default_value(&self, ty: &Type) -> Result<Expr> {
        match ty {
            Type::Bool => Ok(Expr::bool(false)),
            Type::Int { .. } | Type::Optional(_) => Ok(Expr::int(0, ty.clone())),
            Type::Bounded(value) if self.bounded_defaults.contains(&value.name) => Ok(Expr {
                kind: ExprKind::Bounded(Box::new(Expr::int(0, value.backing_type()))),
                ty: ty.clone(),
            }),
            Type::Seq(_) => Ok(Expr { kind: ExprKind::Sequence(Vec::new()), ty: ty.clone() }),
            Type::Enum { name, .. } => {
                let value = self.defaults.get(name).ok_or_else(|| {
                    Diagnostic::span(
                        proc_macro2::Span::call_site(),
                        format!("enum `{name}` has no admitted #[default] variant"),
                    )
                })?;
                Ok(Expr { kind: ExprKind::Enum(value.clone()), ty: ty.clone() })
            }
            _ => self.err(
                &syn::Ident::new("Model", proc_macro2::Span::call_site()),
                "unsupported default value",
            ),
        }
    }
}

pub(crate) fn binary_op(op: &syn::BinOp) -> Option<Op> {
    Some(match op {
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
        _ => return None,
    })
}
pub(crate) fn unary_op(op: &syn::UnOp) -> Option<Unary> {
    match op {
        syn::UnOp::Not(_) => Some(Unary::Not),
        syn::UnOp::Neg(_) => Some(Unary::Neg),
        _ => None,
    }
}
