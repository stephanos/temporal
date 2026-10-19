use super::{Env, Parser, Result, has, path_name, tail};
use crate::{
    Diagnostic, Type,
    tree::{ActionDef, AssumptionDef, ExprKind, ProgressDef},
};
use quote::ToTokens;
use std::collections::BTreeSet;
use syn::Expr as RustExpr;

impl Parser<'_> {
    pub(super) fn action_name(&self, node: &RustExpr, actions: &[ActionDef]) -> Result<String> {
        let name = path_name(&node.to_token_stream().to_string());
        if !actions.iter().any(|a| a.name == name) {
            return self.err(node, format!("unknown action `{name}`"));
        }
        Ok(name)
    }
    pub(super) fn assumption(
        &self,
        node: &RustExpr,
        actions: &[ActionDef],
        depth: usize,
    ) -> Result<AssumptionDef> {
        if depth > 32 {
            return self.err(node, "recursive assumption helper");
        }
        match node {
            RustExpr::Call(call) => {
                let name = path_name(&call.func.to_token_stream().to_string());
                if let Some(method) =
                    name.strip_prefix("Self::").and_then(|name| self.methods.get(name))
                {
                    if !has(&method.attrs, "assumption") || !call.args.is_empty() {
                        return self.err(node, "unsupported assumption call");
                    }
                    return self.assumption(tail(self.file, &method.block)?, actions, depth + 1);
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
                let mut value = self.assumption(&call.receiver, actions, depth + 1)?;
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
    fn action_list(&self, node: &RustExpr, actions: &[ActionDef]) -> Result<Vec<String>> {
        let node = if let RustExpr::Reference(v) = node { v.expr.as_ref() } else { node };
        let RustExpr::Array(array) = node else {
            return self.err(node, "assumption requires an explicit action array");
        };
        let mut seen = BTreeSet::new();
        let mut out = Vec::new();
        for value in &array.elems {
            let name = self.action_name(value, actions)?;
            if seen.insert(name.clone()) {
                out.push(name);
            }
        }
        Ok(out)
    }
    pub(super) fn progress(
        &self,
        method: &syn::ImplItemFn,
        actions: &[ActionDef],
    ) -> Result<ProgressDef> {
        let mut node = tail(self.file, &method.block)?;
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
                        &Env::new(),
                        Some(&Type::Bool),
                        0,
                    )?);
                }
                "assuming" => assumption = self.assumption(&call.args[0], actions, 0)?,
                "within" => {
                    let value = self.expr(&call.args[0], &Env::new(), &Env::new(), None, 0)?;
                    let ExprKind::Int(n) = value.kind else {
                        return self.err(node, "progress bound must be a positive literal");
                    };
                    if n <= 0 || n >= i128::from(i32::MAX) {
                        return self.err(node, "progress bound must be in 1..2147483647 for TLC's capped transition observer");
                    }
                    bound = Some(u64::try_from(n).map_err(|_| {
                        Diagnostic::at(self.file, node, "progress bound exceeds u64")
                    })?);
                }
                _ => return self.err(node, "unsupported progress builder"),
            }
            node = &call.receiver;
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
                &Env::new(),
                Some(&Type::Bool),
                0,
            )?,
            target: target.ok_or_else(|| {
                Diagnostic::at(self.file, node, "progress requires eventually(target)")
            })?,
            assumption,
            bound,
        })
    }
}
