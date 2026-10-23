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
                env.bind_state(&id.ident.to_string(), &self.fields, &self.field_env(false));
            }
        }
        if let Some(input) = parameters.positional.get(1) {
            let unit = self.outcome_type == Type::Unit;
            if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if reference.mutability.is_none() && match reference.elem.as_ref() {
                syn::Type::Tuple(tuple) => unit && tuple.elems.is_empty(),
                syn::Type::Path(path) => !unit && self.dsl_path(&path.path).as_deref() == Some("Outcome"),
                _ => false,
            }) {
                return self.err(
                    &input.ty,
                    if unit {
                        "source postconditions currently admit only the immutable unit outcome &()"
                    } else {
                        "a rejecting Model's postcondition outcome must be the immutable &Outcome"
                    },
                );
            }
            if let Pat::Ident(id) = input.pat.as_ref() {
                let name = id.ident.to_string();
                env.insert(
                    name.clone(),
                    Expr { kind: ExprKind::Outcome, ty: self.outcome_type.clone() },
                );
                env.outcome = Some(name);
            }
        }
        self.claim(&method.block, &self.field_env(true), &env, 0)
    }
    /// A hook's negative claims, each with the location of its name.
    pub(crate) fn supplemental(
        &self,
        mac: &syn::Macro,
        initial: &Env,
    ) -> Result<Vec<(ClaimDef, Diagnostic)>> {
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
                        env.bind_state(&id.ident.to_string(), &self.fields, initial);
                    }
                }
                negative.push((
                    ClaimDef {
                        name: name.value(),
                        expr: self.expr(&closure.body, initial, &env, Some(&Type::Bool), 0)?,
                    },
                    Diagnostic::at(name, ""),
                ));
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
                Stmt::Macro(syn::StmtMacro { mac, .. })
                | Stmt::Expr(RustExpr::Macro(syn::ExprMacro { mac, .. }), _)
                    if mac.path.is_ident("check") =>
                {
                    self.check(mac, state, locals, depth)?
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
                Stmt::Expr(RustExpr::Call(call), Some(_)) if self.law_name(call).is_some() => {
                    self.law(call, state, locals, depth)?
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
    /// One `check!`, recording its tokens so a failure can report them as `check!` does.
    fn check(&self, mac: &syn::Macro, state: &Env, locals: &Env, depth: usize) -> Result<Expr> {
        let expr: RustExpr = syn::parse2(mac.tokens.clone())
            .map_err(|e| Diagnostic::span(e.span(), e.to_string()))?;
        let value = self.expr(&expr, state, locals, Some(&Type::Bool), depth)?;
        let mut checks = self.checks.borrow_mut();
        checks.push(mac.tokens.clone());
        Ok(Expr { kind: ExprKind::Check(checks.len() - 1, Box::new(value)), ty: Type::Bool })
    }
}

impl Environment {
    /// The name of the `umpire_dsl::closable` function a claim statement calls, if it calls one.
    fn law_name(&self, call: &syn::ExprCall) -> Option<String> {
        let RustExpr::Path(path) = call.func.as_ref() else { return None };
        self.dsl_path(&path.path)?.strip_prefix("closable::").map(str::to_owned)
    }
    /// A closure law's checks, read from its library source with its arguments bound: whole
    /// states, the judged edge's outcome, and typed values.
    fn law(&self, call: &syn::ExprCall, state: &Env, locals: &Env, depth: usize) -> Result<Expr> {
        let name = self.law_name(call).unwrap_or_default();
        let law = crate::library::law(&name).ok_or_else(|| {
            Diagnostic::at(call, format!("unknown `umpire_dsl::closable` law `{name}`"))
        })?;
        let name = format!("umpire_dsl::closable::{}", law.sig.ident);
        if law.sig.inputs.len() != call.args.len() {
            return self.err(call, format!("`{name}` takes {} arguments", law.sig.inputs.len()));
        }
        let generic: Vec<_> = law.sig.generics.type_params().map(|param| &param.ident).collect();
        let mut env = Env::new();
        for (input, argument) in law.sig.inputs.iter().zip(&call.args) {
            let syn::FnArg::Typed(input) = input else {
                return self.err(call, format!("`{name}` is not a free function"));
            };
            let Pat::Ident(parameter) = input.pat.as_ref() else {
                return self.err(call, format!("`{name}` has a pattern parameter"));
            };
            let parameter = parameter.ident.to_string();
            let borrowed = match input.ty.as_ref() {
                syn::Type::Reference(reference) if reference.mutability.is_none() => {
                    match reference.elem.as_ref() {
                        syn::Type::Path(path) => path.path.get_ident(),
                        _ => None,
                    }
                }
                _ => None,
            };
            if let Some(ty) = borrowed.filter(|ty| generic.contains(ty)) {
                let fields = self.whole(argument, state, locals).ok_or_else(|| {
                    Diagnostic::at(argument, format!("`{name}` reads a whole state `{ty}` here"))
                })?;
                env.bind_state(&parameter, &self.fields, &fields);
            } else if borrowed.is_some_and(|ty| ty == "Outcome") {
                let outcome = match argument {
                    RustExpr::Path(path) => locals
                        .outcome
                        .as_ref()
                        .filter(|outcome| path.path.is_ident(outcome))
                        .and_then(|outcome| locals.get(outcome)),
                    _ => None,
                }
                .ok_or_else(|| {
                    Diagnostic::at(
                        argument,
                        format!("`{name}` reads the judged edge's outcome here"),
                    )
                })?;
                env.insert(parameter.clone(), outcome.clone());
                env.outcome = Some(parameter);
            } else {
                let ty = self.ty(&input.ty).map_err(|refusal| {
                    Diagnostic::at(argument, format!("`{name}`: {}", refusal.message))
                })?;
                let value = self.expr(argument, state, locals, Some(&ty), depth)?;
                env.insert(parameter, value);
            }
        }
        // A refusal inside the library body locates at the call that brought it in.
        self.claim(&law.block, &Env::new(), &env, depth + 1)
            .map_err(|refusal| Diagnostic::at(call, format!("in `{name}`: {}", refusal.message)))
    }
}
