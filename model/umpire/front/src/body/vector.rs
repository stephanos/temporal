use super::{Env, Environment, Result};
use crate::{
    Locate, Type,
    expr::{Expr, ExprKind, Op},
};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::Expr as RustExpr;

/// `usize` lengths and indices, spelled as the non-negative 32-bit integers TLC computes in.
pub(super) fn index_type() -> Type {
    Type::Int { min: 0, max: i128::from(i32::MAX) }
}

impl Environment {
    pub(super) fn vector_type(&self, ty: &syn::Type) -> Result<Option<Type>> {
        let syn::Type::Path(path) = ty else { return Ok(None) };
        let names: Vec<_> = path.path.segments.iter().map(|part| part.ident.to_string()).collect();
        if names != ["Vector"] && names != ["imbl", "Vector"] {
            return Ok(None);
        }
        if names == ["Vector"] && !self.vector_imported {
            return self.err(ty, "Vector requires the explicit canonical imbl::Vector import");
        }
        let vector = path
            .path
            .segments
            .last()
            .ok_or_else(|| crate::Diagnostic::at(ty, "missing Vector type"))?;
        let syn::PathArguments::AngleBracketed(arguments) = &vector.arguments else {
            return self.err(ty, "Vector requires one declared element type");
        };
        let [syn::GenericArgument::Type(element)] =
            arguments.args.iter().collect::<Vec<_>>().as_slice()
        else {
            return self.err(ty, "Vector requires one declared element type");
        };
        let element = self.ty(element)?;
        if !matches!(element, Type::Bool | Type::Int { .. } | Type::Enum { .. }) {
            return self.err(
                ty,
                "Vector elements must be scalar bool, checked integer, or fieldless enum values",
            );
        }
        Ok(Some(Type::Seq(Box::new(element))))
    }

    pub(super) fn vector_constructor(
        &self,
        call: &syn::ExprCall,
        expected: Option<&Type>,
    ) -> Result<Option<Expr>> {
        let RustExpr::Path(path) = call.func.as_ref() else { return Ok(None) };
        let names: Vec<_> = path.path.segments.iter().map(|part| part.ident.to_string()).collect();
        if !matches!(names.as_slice(),[owner,method]if owner=="Vector"&&["new","default"].contains(&method.as_str()))
            && !matches!(names.as_slice(),[root,owner,method]if root=="imbl"&&owner=="Vector"&&["new","default"].contains(&method.as_str()))
        {
            return Ok(None);
        }
        if names[0] == "Vector" && !self.vector_imported {
            return self.err(
                call,
                "Vector constructor requires the explicit canonical imbl::Vector import",
            );
        }
        if !call.args.is_empty() {
            return self.err(call, "Vector new/default constructors take no arguments");
        }
        let Some(ty @ Type::Seq(_)) = expected else {
            return self.err(call, "Vector constructor requires a declared target element type");
        };
        let mut owner = path.path.clone();
        owner.segments.pop();
        owner.segments.pop_punct();
        if owner
            .segments
            .last()
            .is_some_and(|part| !matches!(part.arguments, syn::PathArguments::None))
        {
            let declared: syn::Type = syn::parse2(owner.to_token_stream())
                .map_err(|error| crate::Diagnostic::span(error.span(), error.to_string()))?;
            if self.ty(&declared)? != *ty {
                return self.err(call, "Vector constructor element type differs from its target");
            }
        }
        Ok(Some(Expr { kind: ExprKind::Sequence(Vec::new()), ty: ty.clone() }))
    }

    pub(super) fn vector_method(
        &self,
        call: &syn::ExprMethodCall,
        state: &Env,
        locals: &Env,
        expected: Option<&Type>,
        depth: usize,
    ) -> Result<Option<Expr>> {
        if call.method == "unwrap" {
            return self.err(
                call,
                "unsupported unwrap; Vector get requires explicit copied().unwrap_or(default)",
            );
        }
        if call.method == "unwrap_or"
            && let RustExpr::MethodCall(copied) = call.receiver.as_ref()
            && copied.method == "copied"
            && copied.args.is_empty()
            && let RustExpr::MethodCall(get) = copied.receiver.as_ref()
            && get.method == "get"
            && get.args.len() == 1
            && call.args.len() == 1
        {
            let sequence = self.expr(&get.receiver, state, locals, None, depth)?;
            let Type::Seq(element) = &sequence.ty else {
                return self.err(get, "get receiver must be an admitted Vector");
            };
            let index = self.vector_index(&get.args[0], state, locals, depth)?;
            let fallback = self.expr(&call.args[0], state, locals, Some(element), depth)?;
            return Ok(Some(self.require(
                call,
                Expr {
                    ty: (**element).clone(),
                    kind: ExprKind::GetOr(
                        Box::new(sequence.clone()),
                        Box::new(index),
                        Box::new(fallback),
                    ),
                },
                expected,
            )?));
        }
        if ["is_some", "is_none"].iter().any(|method| call.method == method)
            && call.args.is_empty()
            && let RustExpr::MethodCall(get) = call.receiver.as_ref()
            && get.method == "get"
            && get.args.len() == 1
        {
            let sequence = self.expr(&get.receiver, state, locals, None, depth)?;
            if !matches!(sequence.ty, Type::Seq(_)) {
                return self.err(get, "get receiver must be an admitted Vector");
            }
            let index = self.vector_index(&get.args[0], state, locals, depth)?;
            let length = Expr { kind: ExprKind::Length(Box::new(sequence)), ty: index_type() };
            let present = Expr::binary(Op::Lt, index, length);
            return Ok(Some(if call.method == "is_none" { present.not() } else { present }));
        }
        if ["len", "is_empty"].iter().any(|method| call.method == method) && call.args.is_empty() {
            let sequence = self.expr(&call.receiver, state, locals, None, depth)?;
            if !matches!(sequence.ty, Type::Seq(_)) {
                return Ok(None);
            }
            let length = Expr { kind: ExprKind::Length(Box::new(sequence)), ty: index_type() };
            return Ok(Some(if call.method == "is_empty" {
                Expr::binary(Op::Eq, length, Expr::int(0, index_type()))
            } else {
                length
            }));
        }
        Ok(None)
    }

    fn vector_index(
        &self,
        node: &RustExpr,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Expr> {
        if let RustExpr::Lit(syn::ExprLit { lit: syn::Lit::Int(literal), .. }) = node
            && literal.suffix() == "usize"
        {
            let index = literal
                .base10_parse::<i128>()
                .map_err(|error| crate::Diagnostic::at(node, error.to_string()))?;
            if index > i128::from(i32::MAX) {
                return self.err(node, "Vector index exceeds the admitted signed 32-bit integers");
            }
            return Ok(Expr::int(index, index_type()));
        }
        self.expr(node, state, locals, Some(&index_type()), depth)
    }

    #[expect(clippy::too_many_arguments, reason = "SSA lowering threads its frame by reference")]
    pub(super) fn vector_push(
        &self,
        call: &syn::ExprMethodCall,
        state: &mut Env,
        locals: &Env,
        checks: &mut Vec<Expr>,
        flow: &super::outcome::Flow,
        depth: usize,
    ) -> Result<()> {
        if call.args.len() != 1 {
            return self.err(call, "Vector push_back requires one element");
        }
        let field = self.assigned_field(&call.receiver)?;
        let sequence = state
            .get(&field)
            .ok_or_else(|| crate::Diagnostic::at(call, "unknown Vector field"))?
            .clone();
        let Type::Seq(element) = &sequence.ty else {
            return self.err(call, "push_back requires an admitted Vector field");
        };
        let value = self.expr(&call.args[0], state, locals, Some(element), depth)?;
        checks.push(flow.guard(value.clone(), super::effects::default_expr(&value.ty)));
        let pushed = Expr {
            ty: sequence.ty.clone(),
            kind: ExprKind::Append(Box::new(sequence.clone()), Box::new(value)),
        };
        state.insert(field, flow.guard(pushed, sequence));
        Ok(())
    }

    pub(super) fn bounded_vector_growth(
        &self,
        node: &RustExpr,
        state: &Env,
        guard: &Expr,
        definitions: &[(String, Expr)],
    ) -> Result<()> {
        let values: BTreeMap<_, _> =
            definitions.iter().map(|(name, value)| (name.as_str(), value)).collect();
        for (field, value) in state.iter().filter(|(_, value)| matches!(value.ty, Type::Seq(_))) {
            let mut bounds = BTreeMap::new();
            for (name, definition) in definitions {
                let bound = bound(definition, field, &bounds);
                bounds.insert(name.as_str(), bound);
            }
            if !bounded(
                value,
                upper_bound(guard, field, &bounds),
                field,
                &values,
                &bounds,
                &mut BTreeMap::new(),
            ) {
                return self.err(
                    node,
                    format!(
                        "Vector push_back on `{field}` requires an authored bounded length guard"
                    ),
                );
            }
        }
        Ok(())
    }
}

fn bounded(
    value: &Expr,
    limited: bool,
    field: &str,
    definitions: &BTreeMap<&str, &Expr>,
    bounds: &BTreeMap<&str, Bound>,
    memo: &mut BTreeMap<(String, bool), bool>,
) -> bool {
    match &value.kind {
        ExprKind::Local(name) => {
            let key = (name.clone(), limited);
            if let Some(result) = memo.get(&key) {
                return *result;
            }
            let result = definitions
                .get(name.as_str())
                .is_some_and(|value| bounded(value, limited, field, definitions, bounds, memo));
            memo.insert(key, result);
            result
        }
        ExprKind::Append(sequence, _) => {
            limited && bounded(sequence, limited, field, definitions, bounds, memo)
        }
        ExprKind::If(condition, yes, no) => {
            bounded(
                yes,
                limited || upper_bound(condition, field, bounds),
                field,
                definitions,
                bounds,
                memo,
            ) && bounded(
                no,
                limited || upper_bound(&condition.clone().not(), field, bounds),
                field,
                definitions,
                bounds,
                memo,
            )
        }
        _ => true,
    }
}
#[derive(Clone, Copy, PartialEq, Eq)]
enum Bound {
    Unknown,
    Sequence,
    Length,
    Finite,
    Upper,
}

fn upper_bound(guard: &Expr, field: &str, bounds: &BTreeMap<&str, Bound>) -> bool {
    bound(guard, field, bounds) == Bound::Upper
}

fn bound(value: &Expr, field: &str, bounds: &BTreeMap<&str, Bound>) -> Bound {
    match &value.kind {
        ExprKind::Bool(false) => Bound::Upper,
        ExprKind::Local(name) => bounds.get(name.as_str()).copied().unwrap_or(Bound::Unknown),
        ExprKind::Var(name, false) if name == field => Bound::Sequence,
        ExprKind::Length(sequence) if bound(sequence, field, bounds) == Bound::Sequence => {
            Bound::Length
        }
        ExprKind::Int(value) if *value >= 0 && *value < i128::from(i32::MAX) => Bound::Finite,
        ExprKind::If(_, yes, no) => {
            let yes = bound(yes, field, bounds);
            if yes != Bound::Unknown && bound(no, field, bounds) == yes {
                yes
            } else {
                Bound::Unknown
            }
        }
        ExprKind::Binary(op, left, right) => {
            let (left, right) = (bound(left, field, bounds), bound(right, field, bounds));
            let upper = match op {
                Op::And => left == Bound::Upper || right == Bound::Upper,
                Op::Or => left == Bound::Upper && right == Bound::Upper,
                Op::Lt | Op::Le | Op::Eq if left == Bound::Length && right == Bound::Finite => true,
                Op::Gt | Op::Ge | Op::Eq if left == Bound::Finite && right == Bound::Length => true,
                _ => false,
            };
            if upper { Bound::Upper } else { Bound::Unknown }
        }
        _ => Bound::Unknown,
    }
}
