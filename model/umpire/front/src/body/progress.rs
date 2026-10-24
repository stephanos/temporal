use super::{Env, Environment, Result, tail};
use crate::syntax::{has, path_name};
use crate::{
    Diagnostic, Locate, Type,
    expr::{ExprKind, Op},
    tree::{ActionDef, ActionRef, AssumptionDef, ConditionalAssumption, ProgressDef},
};
use quote::ToTokens;
use std::collections::BTreeSet;
use syn::{Expr as RustExpr, RangeLimits, Token, parse::Parser as _, punctuated::Punctuated};

impl Environment {
    pub(crate) fn assumption_body(
        &self,
        method: &syn::ImplItemFn,
        actions: &[ActionDef],
    ) -> Result<AssumptionDef> {
        self.assumption(tail(&method.block)?, actions, &self.method_env(method)?, 0)
    }
    pub(super) fn assumption(
        &self,
        node: &RustExpr,
        actions: &[ActionDef],
        locals: &Env,
        depth: usize,
    ) -> Result<AssumptionDef> {
        if depth > 32 {
            return self.err(node, "recursive assumption helper");
        }
        match node {
            RustExpr::Paren(value) => self.assumption(&value.expr, actions, locals, depth + 1),
            RustExpr::Block(value) => {
                self.assumption(tail(&value.block)?, actions, locals, depth + 1)
            }
            RustExpr::If(value) => {
                self.assumption_guard(&value.cond, locals, depth)?;
                let condition =
                    self.expr(&value.cond, &Env::new(), locals, Some(&Type::Bool), depth)?;
                let yes = self.assumption(tail(&value.then_branch)?, actions, locals, depth + 1)?;
                let Some((_, no)) = &value.else_branch else {
                    return self.err(node, "conditional assumption requires an exhaustive else");
                };
                let no = self.assumption(no, actions, locals, depth + 1)?;
                Ok(AssumptionDef {
                    conditional: Some(Box::new(ConditionalAssumption {
                        condition,
                        yes,
                        no,
                        location: Diagnostic::at(node, "conditional assumption"),
                    })),
                    ..AssumptionDef::default()
                })
            }
            RustExpr::Call(call) => {
                let name = path_name(&call.func.to_token_stream().to_string());
                if let Some(method) =
                    name.strip_prefix("Self::").and_then(|name| self.methods.get(name))
                {
                    if !has(&method.attrs, "assumption") && !has(&method.attrs, "scope") {
                        return self.err(node, "unsupported assumption call");
                    }
                    return self.assumption(
                        tail(&method.block)?,
                        actions,
                        &self.helper_env(method, &call.args, locals)?,
                        depth + 1,
                    );
                }
                let mut value = AssumptionDef::default();
                let kind = name.rsplit("::").next().unwrap_or(&name);
                if !["only", "without", "fair"].contains(&kind) || call.args.len() != 1 {
                    return self.err(node, format!("unsupported assumption `{name}`"));
                }
                let values = self.action_list(&call.args[0], actions)?;
                match kind {
                    "only" => value.only = Some(values),
                    "without" => value.without = values,
                    "fair" => value.fair = values,
                    _ => {}
                }
                Ok(value)
            }
            RustExpr::MethodCall(call) => {
                let mut value = self.assumption(&call.receiver, actions, locals, depth + 1)?;
                if value.conditional.is_some() {
                    return self.err(node, "assumption builders must be inside conditional leaves");
                }
                if call.args.len() != 1 {
                    return self.err(node, "assumption method requires an action list");
                }
                let values = self.action_list(&call.args[0], actions)?;
                match call.method.to_string().as_str() {
                    "only" => value.only = Some(values),
                    "without" => value.without.extend(values),
                    "fair" => value.fair.extend(values),
                    _ => return self.err(node, "unsupported assumption builder"),
                }
                Ok(value)
            }
            _ => self.err(node, "unsupported named assumption expression"),
        }
    }
    fn assumption_guard(&self, node: &RustExpr, locals: &Env, depth: usize) -> Result<()> {
        if depth > 32 {
            return self.err(node, "assumption guard exceeds the helper-depth limit");
        }
        match node {
            RustExpr::Paren(value) => return self.assumption_guard(&value.expr, locals, depth + 1),
            RustExpr::Group(value) => return self.assumption_guard(&value.expr, locals, depth + 1),
            RustExpr::Lit(value) if matches!(value.lit, syn::Lit::Bool(_) | syn::Lit::Int(_)) => {}
            RustExpr::Path(path) => {
                let name = path_name(&path.path.to_token_stream().to_string());
                if !self.constants.get(&name).is_some_and(|value| {
                    matches!(
                        (&value.kind, &value.ty),
                        (ExprKind::Bool(_), Type::Bool) | (ExprKind::Int(_), Type::Int { .. })
                    )
                }) {
                    return self
                        .err(node, "assumption guards require primitive literal source constants");
                }
            }
            RustExpr::Field(_) => {
                let name = path_name(&node.to_token_stream().to_string());
                if !locals.get(&name).is_some_and(|value| {
                    matches!(value.kind, ExprKind::World(_))
                        && matches!(value.ty, Type::Bool | Type::Int { .. })
                }) {
                    return self
                        .err(node, "assumption guards require primitive selected World fields");
                }
            }
            RustExpr::Unary(value) if matches!(value.op, syn::UnOp::Not(_)) => {
                self.assumption_guard(&value.expr, locals, depth + 1)?;
            }
            RustExpr::Unary(value)
                if matches!(value.op, syn::UnOp::Neg(_))
                    && matches!(value.expr.as_ref(), RustExpr::Lit(value) if matches!(value.lit, syn::Lit::Int(_))) =>
                {}
            RustExpr::Binary(value) => {
                let Some(op) = super::typer::binary_op(&value.op) else {
                    return self.err(node, "unsupported assumption guard operator");
                };
                if !matches!(
                    op,
                    Op::And | Op::Or | Op::Eq | Op::Ne | Op::Lt | Op::Le | Op::Gt | Op::Ge
                ) {
                    return self.err(node, "assumption guards do not admit arithmetic");
                }
                self.assumption_guard(&value.left, locals, depth + 1)?;
                self.assumption_guard(&value.right, locals, depth + 1)?;
                let typed = self.expr(node, &Env::new(), locals, Some(&Type::Bool), depth)?;
                if let ExprKind::Binary(_, left, right) = &typed.kind
                    && matches!(left.ty, Type::Int { .. })
                    && !matches!(left.kind, ExprKind::Int(_))
                    && !matches!(right.kind, ExprKind::Int(_))
                {
                    return self
                        .err(node, "integer assumption comparisons require a primitive literal");
                }
            }
            _ => return self.err(node, "unsupported World-only assumption guard"),
        }
        let _ = self.expr(node, &Env::new(), locals, None, depth)?;
        Ok(())
    }
    fn action_list(&self, node: &RustExpr, actions: &[ActionDef]) -> Result<Vec<ActionRef>> {
        let node = if let RustExpr::Reference(v) = node { v.expr.as_ref() } else { node };
        let RustExpr::Array(array) = node else {
            return self.err(node, "assumption requires an explicit action array");
        };
        let mut out = Vec::new();
        for value in &array.elems {
            let action = self.action_ref(value, actions)?;
            if !out.contains(&action) {
                out.push(action);
            }
        }
        Ok(out)
    }
    pub(crate) fn progress(
        &self,
        method: &syn::ImplItemFn,
        actions: &[ActionDef],
    ) -> Result<ProgressDef> {
        let locals = self.method_env(method)?;
        let mut node = tail(&method.block)?;
        let mut target = None;
        let mut assumption = AssumptionDef::default();
        let mut bound = None;
        let mut builders = BTreeSet::new();
        while let RustExpr::MethodCall(call) = node {
            if !builders.insert(call.method.to_string()) {
                return self.err(
                    node,
                    format!(
                        "repeated progress builder `{}` is outside the admitted subset",
                        call.method
                    ),
                );
            }
            if call.args.len() != 1 {
                return self.err(node, "progress builder requires one argument");
            }
            match call.method.to_string().as_str() {
                "eventually" => {
                    target = Some(self.expr(
                        &call.args[0],
                        &self.field_env(false),
                        &locals,
                        Some(&Type::Bool),
                        0,
                    )?);
                }
                "assuming" => assumption = self.assumption(&call.args[0], actions, &locals, 0)?,
                "within" => bound = Some(self.progress_bound(node, &call.args[0])?),
                _ => return self.err(node, "unsupported progress builder"),
            }
            node = &call.receiver;
        }
        if let RustExpr::Macro(call) = node
            && call.mac.path.is_ident("path")
        {
            return self.path_claim(method, call, assumption, &locals);
        }
        let RustExpr::Call(call) = node else {
            return self.err(node, "progress requires whenever");
        };
        if path_name(&call.func.to_token_stream().to_string()) != "whenever" || call.args.len() != 1
        {
            return self.err(node, "progress requires whenever(trigger)");
        }
        Ok(ProgressDef {
            name: method.sig.ident.to_string(),
            trigger: self.expr(
                &call.args[0],
                &self.field_env(false),
                &locals,
                Some(&Type::Bool),
                0,
            )?,
            target: target
                .ok_or_else(|| Diagnostic::at(node, "progress requires eventually(target)"))?,
            assumption,
            bound,
        })
    }
    fn progress_bound(&self, node: &impl ToTokens, value: &RustExpr) -> Result<u64> {
        let value = self.expr(value, &Env::new(), &Env::new(), None, 0)?;
        let ExprKind::Int(n) = value.kind else {
            return self.err(node, "progress bound must be a positive literal");
        };
        if n <= 0 || n >= i128::from(i32::MAX) {
            return self.err(node, "progress bound must be in 1..2147483647");
        }
        u64::try_from(n).map_err(|_| Diagnostic::at(node, "progress bound exceeds u64"))
    }
    /// `path![trigger, .., target]` or `path![trigger, ..=n, target]`.
    fn path_claim(
        &self,
        method: &syn::ImplItemFn,
        call: &syn::ExprMacro,
        assumption: AssumptionDef,
        locals: &Env,
    ) -> Result<ProgressDef> {
        let parts = Punctuated::<RustExpr, Token![,]>::parse_terminated
            .parse2(call.mac.tokens.clone())
            .map_err(|error| Diagnostic::at(call, error.to_string()))?;
        // `declare::machine` admitted the claim as a trigger, a `..` or `..=n` gap and a target.
        let [trigger, RustExpr::Range(gap), target] = parts.iter().collect::<Vec<_>>()[..] else {
            unreachable!("declare::machine validates path! claims")
        };
        let bound = match (&gap.start, &gap.limits, &gap.end) {
            (None, RangeLimits::Closed(_), Some(bound)) => Some(self.progress_bound(call, bound)?),
            _ => None,
        };
        let predicate =
            |node| self.expr(node, &self.field_env(false), locals, Some(&Type::Bool), 0);
        Ok(ProgressDef {
            name: method.sig.ident.to_string(),
            trigger: predicate(trigger)?,
            target: predicate(target)?,
            assumption,
            bound,
        })
    }
}
