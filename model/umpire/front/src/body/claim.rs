use super::{Env, Environment, Result};
use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind},
    tree::ClaimDef,
};
use syn::{Expr as RustExpr, Pat, Stmt};

impl Environment {
    pub(crate) fn invariant(&self, method: &syn::ImplItemFn, state: &Env) -> Result<Expr> {
        self.claim(&method.block, state, &self.method_env(method)?, 0)
    }
    pub(crate) fn postcondition(&self, method: &syn::ImplItemFn) -> Result<Expr> {
        let parameters = self.parameters(method)?;
        let mut env = self.method_env(method)?;
        if let Some(input) = parameters.positional.first() {
            if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if reference.mutability.is_none() && matches!(reference.elem.as_ref(), syn::Type::Path(path) if path.path.is_ident("Self") || path.path.is_ident(&self.state)))
            {
                return self.err(
                    &input.ty,
                    format!(
                        "the before-state parameter must be an immutable &Self or &{}",
                        self.state
                    ),
                );
            }
            if let Pat::Ident(id) = input.pat.as_ref() {
                env.state = Some(id.ident.to_string());
                for field in &self.fields {
                    env.insert(
                        format!("{}.{}", id.ident, field.name),
                        Expr::var(&field.name, field.ty.clone(), false),
                    );
                }
            }
        }
        if let Some(input) = parameters.positional.get(1) {
            if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if reference.mutability.is_none() && matches!(reference.elem.as_ref(), syn::Type::Tuple(tuple) if tuple.elems.is_empty()))
            {
                return self.err(
                    &input.ty,
                    "source postconditions currently admit only the immutable unit outcome &()",
                );
            }
            if let Pat::Ident(id) = input.pat.as_ref() {
                let name = id.ident.to_string();
                env.insert(name.clone(), Expr { kind: ExprKind::Outcome, ty: Type::Unit });
                env.outcome = Some(name);
            }
        }
        self.claim(&method.block, &self.field_env(true), &env, 0)
    }
    pub(crate) fn supplemental(&self, mac: &syn::Macro, initial: &Env) -> Result<Vec<ClaimDef>> {
        let mut negative = Vec::new();
        let args = mac
            .parse_body_with(
                syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
            )
            .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
        for expr in args.iter().skip(2) {
            if let RustExpr::Call(call) = expr
                && call.args.len() == 2
            {
                let Some(RustExpr::Lit(name)) = call.args.first() else { continue };
                let syn::Lit::Str(name) = &name.lit else { continue };
                let Some(RustExpr::Closure(closure)) = call.args.last() else {
                    continue;
                };
                let mut env = Env::new();
                if self.declared_world
                    && let Some(pattern) = closure.inputs.first()
                {
                    self.bind_world(pattern, &mut env)?;
                }
                if let Some(pat) = closure.inputs.iter().nth(1) {
                    let pat = if let Pat::Type(p) = pat { p.pat.as_ref() } else { pat };
                    if let Pat::Ident(id) = pat {
                        env.state = Some(id.ident.to_string());
                        for f in &self.fields {
                            env.insert(
                                format!("{}.{}", id.ident, f.name),
                                Expr::var(&f.name, f.ty.clone(), false),
                            );
                        }
                    }
                }
                negative.push(ClaimDef {
                    name: name.value(),
                    expr: self.expr(&closure.body, initial, &env, Some(&Type::Bool), 0)?,
                });
            }
        }
        Ok(negative)
    }
    pub(crate) fn claim(
        &self,
        block: &syn::Block,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Expr> {
        let mut checks = Vec::new();
        for statement in &block.stmts {
            let value = match statement {
                Stmt::Macro(mac) if mac.mac.path.is_ident("check") => {
                    let expr: RustExpr = syn::parse2(mac.mac.tokens.clone())
                        .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
                    self.expr(&expr, state, locals, Some(&Type::Bool), depth)?
                }
                Stmt::Expr(RustExpr::Macro(mac), _) if mac.mac.path.is_ident("check") => {
                    let expr: RustExpr = syn::parse2(mac.mac.tokens.clone())
                        .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
                    self.expr(&expr, state, locals, Some(&Type::Bool), depth)?
                }
                Stmt::Expr(RustExpr::If(condition), _) => {
                    let cond =
                        self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                    let yes = cond.clone().implies(self.claim(
                        &condition.then_branch,
                        state,
                        locals,
                        depth,
                    )?);
                    if let Some((_, other)) = &condition.else_branch {
                        let RustExpr::Block(other) = other.as_ref() else {
                            return self.err(other, "claim else-if is unsupported");
                        };
                        yes.and(cond.not().implies(self.claim(
                            &other.block,
                            state,
                            locals,
                            depth,
                        )?))
                    } else {
                        yes
                    }
                }
                _ => {
                    return self
                        .err(statement, "claim body supports check! and conditional checks only");
                }
            };
            checks.push(value);
        }
        Ok(Expr { kind: ExprKind::Checks(checks), ty: Type::Bool })
    }
}
