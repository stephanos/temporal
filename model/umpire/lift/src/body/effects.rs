use super::{Env, Environment, Result};
use crate::{
    Diagnostic, Type,
    expr::{Expr, ExprKind, Op},
    syntax::has,
    tree::{ActionDef, Branch},
};
use quote::ToTokens;
use syn::{BinOp, Expr as RustExpr, Pat, Stmt};

impl Environment<'_> {
    pub(crate) fn handler<'a>(&self, method: &'a syn::ImplItemFn) -> Result<&'a syn::ExprMatch> {
        let parameters = self.parameters(method)?;
        if parameters.positional.len() != 1 {
            return self.err(method, "a handler requires its actor and an optional trailing World");
        }
        let [Stmt::Expr(RustExpr::Match(matched), _)] = method.block.stmts.as_slice() else {
            return self.err(method, "action handler must match its action parameter");
        };
        let parameter = parameters
            .positional
            .first()
            .and_then(|input| match input.pat.as_ref() {
                Pat::Ident(id) => Some(id.ident.to_string()),
                _ => None,
            })
            .ok_or_else(|| Diagnostic::at(self.file, method, "missing action parameter"))?;
        if !matches!(matched.expr.as_ref(),RustExpr::Path(p)if p.path.is_ident(&parameter)) {
            return self.err(&matched.expr, "handler must match the declared action parameter");
        }
        Ok(matched)
    }
    pub(crate) fn branches(
        &self,
        matched: &syn::ExprMatch,
        action: &ActionDef,
        env: &Env,
    ) -> Result<Vec<Branch>> {
        let mut branches = Vec::new();
        let mut previous = Expr::bool(false);
        for arm in &matched.arms {
            let (pattern, guard) = if let Pat::Guard(pattern) = &arm.pat {
                (&*pattern.pat, Some(&*pattern.guard))
            } else {
                (&arm.pat, None)
            };
            let Some(matched) = self.action_pattern(pattern, action)? else {
                continue;
            };
            // Payload names shadow the World parameter and source constants lexically, as in Rust.
            let mut locals = env.clone();
            for (name, value) in matched.bindings {
                locals.insert(name, value);
            }
            let guard = if let Some(guard) = guard {
                matched.condition.and(self.expr(
                    guard,
                    &self.field_env(false),
                    &locals,
                    Some(&Type::Bool),
                    0,
                )?)
            } else {
                matched.condition
            };
            let effective = previous.clone().not().and(guard.clone());
            previous = previous.or(guard);
            let mut state = self.field_env(false);
            let mut checks = Vec::new();
            let (body, stays) = without_stay(&arm.body);
            self.effects(&body, &mut state, &mut locals, &mut checks, 0)?;
            self.bounded_vector_growth(&body, &state, &effective)?;
            branches.push(Branch {
                guard: effective,
                updates: state.values,
                checks,
                outcome: Expr::unit(),
                preserve_self_loop: stays,
            });
        }
        Ok(branches)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "SSA lowering follows the admitted statement variants"
    )]
    fn effects(
        &self,
        node: &RustExpr,
        state: &mut Env,
        locals: &mut Env,
        checks: &mut Vec<Expr>,
        depth: usize,
    ) -> Result<()> {
        if depth > 32 {
            return self.err(node, "recursive effects are unsupported");
        }
        match node {
            RustExpr::Block(block) => {
                let mut scoped = locals.clone();
                for statement in &block.block.stmts {
                    match statement {
                        Stmt::Local(local) => {
                            let Pat::Ident(id) = &local.pat else {
                                return self.err(local, "only simple local bindings are admitted");
                            };
                            let Some(init) = &local.init else {
                                return self.err(local, "uninitialized local is unsupported");
                            };
                            if init.diverge.is_some() {
                                return self.err(local, "let-else is unsupported");
                            }
                            let value = self.expr(&init.expr, state, &scoped, None, depth)?;
                            checks.push(value.clone());
                            scoped.insert(id.ident.to_string(), value);
                        }
                        Stmt::Expr(expr, _) => {
                            self.effects(expr, state, &mut scoped, checks, depth)?;
                        }
                        _ => return self.err(statement, "unsupported effect statement"),
                    }
                }
            }
            RustExpr::Assign(assign) => {
                let field = self.assigned_field(&assign.left)?;
                let ty = &self
                    .fields
                    .iter()
                    .find(|f| f.name == field)
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown assignment field"))?
                    .ty;
                let value = self.expr(&assign.right, state, locals, Some(ty), depth)?;
                checks.push(value.clone());
                state.insert(field, value);
            }
            RustExpr::Binary(binary) => {
                let op = match binary.op {
                    BinOp::AddAssign(_) => Op::Add,
                    BinOp::SubAssign(_) => Op::Sub,
                    BinOp::MulAssign(_) => Op::Mul,
                    _ => return self.err(node, "unsupported effect binary operator"),
                };
                let field = self.assigned_field(&binary.left)?;
                let left = state
                    .get(&field)
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown assignment field"))?
                    .clone();
                if !matches!(left.ty, Type::Int { .. }) {
                    return self.err(node, "compound assignment requires an integer");
                }
                let right = self.expr(&binary.right, state, locals, Some(&left.ty), depth)?;
                let value = Expr::binary(op, left, right);
                checks.push(value.clone());
                state.insert(field, value);
            }
            RustExpr::If(condition) => {
                let cond = self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                let mut yes = state.clone();
                let mut no = state.clone();
                let mut yc = Vec::new();
                let mut nc = Vec::new();
                let yes_expr = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: condition.then_branch.clone(),
                });
                self.effects(&yes_expr, &mut yes, &mut locals.clone(), &mut yc, depth)?;
                if let Some((_, other)) = &condition.else_branch {
                    self.effects(other, &mut no, &mut locals.clone(), &mut nc, depth)?;
                }
                for (name, value) in state.iter_mut() {
                    *value = Expr::select(cond.clone(), yes[name].clone(), no[name].clone());
                }
                for value in yc {
                    checks.push(Expr::select(cond.clone(), value.clone(), default_expr(&value.ty)));
                }
                for value in nc {
                    checks.push(Expr::select(cond.clone(), default_expr(&value.ty), value));
                }
            }
            RustExpr::MethodCall(call) if call.method == "push_back" => {
                self.vector_push(call, state, locals, checks, depth)?;
            }
            RustExpr::MethodCall(call) if matches!(call.receiver.as_ref(),RustExpr::Path(p)if p.path.is_ident("self")) =>
            {
                let method = self
                    .methods
                    .get(&call.method.to_string())
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown effect helper"))?;
                if !has(&method.attrs, "effect") || method.sig.receiver().is_none() {
                    return self.err(
                        node,
                        "only #[effect] calls with an optional trailing World are admitted",
                    );
                }
                let body = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: method.block.clone(),
                });
                self.effects(
                    &body,
                    state,
                    &mut self.helper_env(method, &call.args, locals)?,
                    checks,
                    depth + 1,
                )?;
            }
            _ => return self.err(node, format!("unsupported effect `{}`", node.to_token_stream())),
        }
        Ok(())
    }
    pub(super) fn assigned_field(&self, node: &RustExpr) -> Result<String> {
        if let RustExpr::Field(field) = node
            && matches!(field.base.as_ref(),RustExpr::Path(p)if p.path.is_ident("self"))
        {
            return Ok(field.member.to_token_stream().to_string());
        }
        self.err(node, "assignment must target a declared self field")
    }
}
fn is_stay(mac: &syn::Macro) -> bool {
    mac.path.is_ident("stay") && mac.tokens.is_empty()
}
/// Strips `stay!()` from an arm's top level; one nested in a condition stays and is refused.
fn without_stay(body: &RustExpr) -> (RustExpr, bool) {
    match body {
        RustExpr::Macro(mac) if is_stay(&mac.mac) => (syn::parse_quote!({}), true),
        RustExpr::Block(block) => {
            let mut block = block.clone();
            let before = block.block.stmts.len();
            block.block.stmts.retain(|statement| match statement {
                Stmt::Macro(mac) => !is_stay(&mac.mac),
                Stmt::Expr(RustExpr::Macro(mac), _) => !is_stay(&mac.mac),
                _ => true,
            });
            let stays = block.block.stmts.len() != before;
            (RustExpr::Block(block), stays)
        }
        _ => (body.clone(), false),
    }
}
fn default_expr(ty: &Type) -> Expr {
    match ty {
        Type::Bool => Expr::bool(false),
        Type::Int { .. } => Expr::int(0, ty.clone()),
        Type::Seq(_) => Expr { kind: ExprKind::Sequence(Vec::new()), ty: ty.clone() },
        Type::Enum { variants, .. } => {
            Expr { kind: ExprKind::Enum(variants[0].clone()), ty: ty.clone() }
        }
        _ => Expr::unit(),
    }
}
