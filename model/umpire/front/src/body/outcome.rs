//! Rejecting outcomes: `reject!` in handler bodies, and the stock `Outcome::Rejected(..)` value.
use super::{Env, Environment, Result};
use crate::{
    Diagnostic, Locate, REJECTIONS, Type,
    expr::{Expr, ExprKind, Op},
};
use syn::Expr as RustExpr;

/// What one handler step has decided so far: each `reject!` reached, in source order, with the
/// branch conditions that reach it, and when the running function has not yet returned. A
/// `reject!` in an `#[effect]` helper returns from that helper only, so each call starts its own
/// `running` from its caller's.
///
/// The first reached rejection is the outcome. A `reject!` reached after a return is always
/// preceded by the one that returned, so its condition need not repeat `running`, and the
/// outcome stays linear in the number of rejections.
#[derive(Clone)]
pub(super) struct Flow {
    rejections: Vec<(Expr, Expr)>,
    unit: bool,
    pub(super) running: Expr,
}
impl Flow {
    pub(super) fn new(outcome: &Type) -> Self {
        Self { rejections: Vec::new(), unit: *outcome == Type::Unit, running: Expr::bool(true) }
    }
    /// The step's outcome: the first reached rejection, otherwise accepted.
    pub(super) fn outcome(&self) -> Expr {
        let mut reached: Vec<&(Expr, Expr)> = Vec::new();
        for entry in &self.rejections {
            if !reached.iter().any(|(condition, _)| *condition == entry.0) {
                reached.push(entry);
            }
        }
        let otherwise = if self.unit { Expr::unit() } else { accepted() };
        reached.into_iter().rev().fold(otherwise, |otherwise, (condition, rejection)| {
            choose(condition, rejection.clone(), otherwise)
        })
    }
    /// Whether every path has returned, so later statements cannot run.
    pub(super) fn returned(&self) -> bool {
        self.running.kind == ExprKind::Bool(false)
    }
    /// Whether some path may have returned, so a later effect applies only where it runs.
    pub(super) fn diverged(&self) -> bool {
        self.running.kind != ExprKind::Bool(true)
    }
    /// The flow after `if condition`, from each side's flow; both sides started from `self`.
    pub(super) fn join(&mut self, condition: &Expr, yes: Self, no: Self) {
        let start = self.rejections.len();
        let reached = |side: Self, condition: Expr| {
            side.rejections.into_iter().skip(start).map(move |(reach, rejection)| {
                let reach = if reach.kind == ExprKind::Bool(true) {
                    condition.clone()
                } else {
                    condition.clone().and(reach)
                };
                (reach, rejection)
            })
        };
        self.running = choose(condition, yes.running.clone(), no.running.clone());
        self.rejections.extend(reached(yes, condition.clone()));
        self.rejections.extend(reached(no, condition.clone().not()));
    }
    /// `value` where the step still runs, `otherwise` where it has returned.
    pub(super) fn guard(&self, value: Expr, otherwise: Expr) -> Expr {
        if self.diverged() && value != otherwise {
            Expr::select(self.running.clone(), value, otherwise)
        } else {
            value
        }
    }
}
fn choose(condition: &Expr, yes: Expr, no: Expr) -> Expr {
    match (&yes.kind, &no.kind) {
        _ if yes == no || condition.kind == ExprKind::Bool(true) => yes,
        _ if condition.kind == ExprKind::Bool(false) => no,
        (ExprKind::Bool(true), ExprKind::Bool(false)) => condition.clone(),
        (ExprKind::Bool(false), ExprKind::Bool(true)) => condition.clone().not(),
        _ => Expr::select(condition.clone(), yes, no),
    }
}
fn accepted() -> Expr {
    Expr { kind: ExprKind::Enum("Outcome::Accepted".into()), ty: Type::outcome() }
}
/// Whether an outcome is one fixed rejection.
pub(crate) fn rejects(outcome: &Expr) -> bool {
    matches!(&outcome.kind, ExprKind::Enum(value) if value.starts_with("Outcome::Rejected("))
}

impl Environment {
    pub(super) fn is_reject(mac: &syn::Macro) -> bool {
        mac.path.is_ident("reject")
    }
    /// `reject!(rejection)`: record the first rejection, then return from the running function.
    pub(super) fn reject(
        &self,
        mac: &syn::Macro,
        state: &Env,
        locals: &Env,
        flow: &mut Flow,
        depth: usize,
    ) -> Result<()> {
        let arguments = mac
            .parse_body_with(
                syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
            )
            .map_err(Diagnostic::syn)?;
        let [rejection] = arguments.iter().collect::<Vec<_>>()[..] else {
            return self.err(mac, "reject! takes one rejection");
        };
        let value =
            rejected(&self.expr(rejection, state, locals, Some(&Type::rejection()), depth)?);
        flow.rejections.push((Expr::bool(true), value));
        flow.running = Expr::bool(false);
        Ok(())
    }
    pub(super) fn is_rejected(&self, call: &syn::ExprCall) -> bool {
        matches!(call.func.as_ref(), RustExpr::Path(path) if path.path.segments.len() == 2
            && path.path.segments[0].ident == "Outcome" && path.path.segments[1].ident == "Rejected")
            && self.enums.get("Outcome") == Some(&Type::outcome())
    }
    /// `Outcome::Rejected(rejection)`.
    pub(super) fn rejected_call(
        &self,
        call: &syn::ExprCall,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Expr> {
        let [rejection] = call.args.iter().collect::<Vec<_>>()[..] else {
            return self.err(call, "Outcome::Rejected takes one rejection");
        };
        Ok(rejected(&self.expr(rejection, state, locals, Some(&Type::rejection()), depth)?))
    }
}

/// The stock outcome rejecting with a typed rejection value.
fn rejected(rejection: &Expr) -> Expr {
    let outcome = |variant: &str| Expr {
        kind: ExprKind::Enum(format!("Outcome::Rejected({variant})")),
        ty: Type::outcome(),
    };
    if let ExprKind::Enum(value) = &rejection.kind {
        return outcome(value.trim_start_matches("Rejection::"));
    }
    let (last, others) = REJECTIONS.split_last().unwrap_or((&"", &[]));
    others.iter().rev().fold(outcome(last), |otherwise, variant| {
        let rejection = Expr::binary(
            Op::Eq,
            rejection.clone(),
            Expr { kind: ExprKind::Enum(format!("Rejection::{variant}")), ty: Type::rejection() },
        );
        Expr::select(rejection, outcome(variant), otherwise)
    })
}
