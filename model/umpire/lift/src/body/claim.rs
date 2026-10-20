use super::{Env, Environment, Result};
use crate::{
    Diagnostic, Type,
    expr::{Expr, ExprKind},
    tree::ClaimDef,
};
use syn::{Expr as RustExpr, Pat, Stmt};

impl Environment<'_> {
    pub(crate) fn postcondition(&self, method: &syn::ImplItemFn) -> Result<Expr> {
        let mut env = Env::new();
        for input in &method.sig.inputs {
            if let syn::FnArg::Typed(input) = input
                && let Pat::Ident(id) = input.pat.as_ref()
            {
                for field in &self.fields {
                    env.insert(
                        format!("{}.{}", id.ident, field.name),
                        Expr::var(&field.name, field.ty.clone(), false),
                    );
                }
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
            .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
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
                if let Some(pat) = closure.inputs.iter().nth(1) {
                    let pat = if let Pat::Type(p) = pat { p.pat.as_ref() } else { pat };
                    if let Pat::Ident(id) = pat {
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
                        .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
                    self.expr(&expr, state, locals, Some(&Type::Bool), depth)?
                }
                Stmt::Expr(RustExpr::Macro(mac), _) if mac.mac.path.is_ident("check") => {
                    let expr: RustExpr = syn::parse2(mac.mac.tokens.clone())
                        .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
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
