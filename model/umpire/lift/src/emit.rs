use crate::{
    Action, Diagnostic, Field, Metadata, Module, Progress, Property, Type,
    tree::{Expr, ExprKind, Lifted, Op, Start, Unary},
};
use std::collections::BTreeMap;
use std::fmt::Write;

const OBSERVERS: [&str; 7] = [
    "__umpire_edge_tick",
    "__umpire_pending",
    "__umpire_steps",
    "__umpire_action",
    "__umpire_choice",
    "__umpire_outcome",
    "__umpire_facts",
];

#[expect(clippy::too_many_lines, reason = "Definitions are emitted in module dependency order")]
pub(crate) fn emit(model: &Lifted) -> Result<Module, Diagnostic> {
    for field in &model.fields {
        if OBSERVERS.contains(&field.name.as_str()) {
            return Err(Diagnostic::span(
                &model.file,
                proc_macro2::Span::call_site(),
                format!("reserved observer name `{}`", field.name),
            ));
        }
    }
    let mut out = String::new();
    line(&mut out, format!("---- MODULE {} ----", model.name));
    line(&mut out, "EXTENDS Integers, TLC, Sequences");
    line(
        &mut out,
        r#"__umpire_Div(__umpire_a, __umpire_b) ==
    IF Assert(__umpire_b # 0, "Rust integer division by zero") /\ Assert(~(__umpire_a = (-2147483647 - 1) /\ __umpire_b = -1), "Rust integer division overflow")
    THEN IF __umpire_b = (-2147483647 - 1)
         THEN IF __umpire_a = (-2147483647 - 1) THEN 1 ELSE 0
         ELSE IF __umpire_a < 0
              THEN LET __umpire_d == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
                       __umpire_n == -(__umpire_a + 1)
                       __umpire_q == __umpire_n \div __umpire_d
                       __umpire_r == __umpire_n - __umpire_q * __umpire_d
                       __umpire_negative == -__umpire_q - (IF __umpire_r = __umpire_d - 1 THEN 1 ELSE 0)
                   IN IF __umpire_b < 0 THEN -__umpire_negative ELSE __umpire_negative
              ELSE IF __umpire_b < 0 THEN -(__umpire_a \div (-__umpire_b)) ELSE __umpire_a \div __umpire_b
    ELSE 0"#,
    );
    line(&mut out, "");
    for (name, value) in &model.constants {
        line(&mut out, format!("{name} == {}", expr(value)));
    }
    let fields: Vec<_> = model.fields.iter().map(|f| f.name.as_str()).collect();
    let mut variables = fields.clone();
    variables.extend(OBSERVERS);
    line(&mut out, format!("VARIABLES {}", variables.join(", ")));
    line(&mut out, format!("vars == <<{}>>", variables.join(", ")));
    line(&mut out, format!("ModelVars == <<{}>>", fields.join(", ")));
    line(&mut out, "");
    let mut initial = Vec::new();
    if let Some(starts) = &model.starts {
        initial.push(disjoin(
            starts
                .iter()
                .map(|start| {
                    conjoin(
                        model
                            .fields
                            .iter()
                            .map(|f| format!("{} = {}", f.name, expr(&start[&f.name]))),
                    )
                })
                .collect(),
        ));
    } else {
        for field in &model.fields {
            initial.push(match &field.start {
                Start::Value(value) => format!("{} = {}", field.name, expr(value)),
                Start::Range(lo, hi) => format!("{} \\in ({}..{})", field.name, expr(lo), expr(hi)),
                Start::ExclusiveRange(lo, hi) => format!(
                    "(IF {} < {} THEN {} \\in ({}..({} - 1)) ELSE FALSE)",
                    expr(lo),
                    expr(hi),
                    field.name,
                    expr(lo),
                    expr(hi)
                ),
            });
        }
    }
    let outcome_type = model.outcome_type.clone();
    let fact_type = model.fact_type.clone();
    initial.extend([
        "__umpire_edge_tick = FALSE".into(),
        "__umpire_pending = 0".into(),
        "__umpire_steps = 0".into(),
        "__umpire_action = \"\"".into(),
        "__umpire_choice = 0".into(),
        format!("__umpire_outcome = {}", default(&outcome_type)),
        "__umpire_facts = <<>>".into(),
    ]);
    line(&mut out, format!("Init == {}", conjoin(initial)));
    line(&mut out, "");
    for action in &model.actions {
        let branches = action
            .branches
            .iter()
            .map(|branch| {
                let mut terms = vec![expr(&branch.guard)];
                for check in &branch.checks {
                    let value = expr(check);
                    terms.push(format!("({value}) = ({value})"));
                }
                if !branch.preserve_self_loop {
                    terms.push(disjoin(
                        model
                            .fields
                            .iter()
                            .map(|f| {
                                format!(
                                    "{} # {}",
                                    expr(branch.updates.get(&f.name).unwrap_or(&Expr::var(
                                        &f.name,
                                        f.ty.clone(),
                                        false
                                    ))),
                                    f.name
                                )
                            })
                            .collect(),
                    ));
                }
                for field in &model.fields {
                    terms.push(format!(
                        "{}' = {}",
                        field.name,
                        branch.updates.get(&field.name).map_or_else(|| field.name.clone(), expr)
                    ));
                }
                terms.extend([
                    format!("__umpire_action' = {}", text(&action.name)),
                    format!("__umpire_choice' = {}", branch.choice),
                    format!("__umpire_outcome' = {}", expr(&branch.outcome)),
                    format!(
                        "__umpire_facts' = <<{}>>",
                        branch.facts.iter().map(expr).collect::<Vec<_>>().join(", ")
                    ),
                ]);
                conjoin(terms)
            })
            .collect();
        line(&mut out, format!("{} == {}", action.operator, disjoin(branches)));
    }
    line(
        &mut out,
        format!(
            "ModelNext == {}",
            disjoin(model.actions.iter().map(|a| a.operator.clone()).collect())
        ),
    );
    line(&mut out, "ModelEdge == __umpire_edge_tick' # __umpire_edge_tick");
    line(
        &mut out,
        "Edge == ModelNext /\\ (__umpire_edge_tick' = ~__umpire_edge_tick) /\\ UNCHANGED <<__umpire_pending, __umpire_steps>>",
    );
    line(&mut out, "Complete == ~ENABLED ModelNext /\\ UNCHANGED vars");
    line(&mut out, "Next == Edge \\/ Complete");
    line(&mut out, "Spec == Init /\\ [][Next]_vars");
    line(&mut out, "");
    let mut cfg = "SPECIFICATION Spec\n".to_owned();
    let invariants = model
        .invariants
        .iter()
        .map(|claim| {
            let operator = format!("Invariant_{}", claim.name);
            line(&mut out, format!("{operator} == {}", expr(&claim.expr)));
            line(&mut cfg, format!("INVARIANT {operator}"));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\nINVARIANT {operator}\n"),
                operator,
            }
        })
        .collect();
    let postconditions = model
        .postconditions
        .iter()
        .map(|claim| {
            let operator = format!("Post_{}", claim.name);
            let applies = claim.after.as_ref().map_or_else(
                || "ModelEdge".into(),
                |action| format!("ModelEdge /\\ (__umpire_action' = {})", text(action)),
            );
            line(
                &mut out,
                format!("{operator} == [][~({applies}) \\/ ({})]_vars", expr(&claim.expr)),
            );
            line(&mut cfg, format!("PROPERTY {operator}"));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\nPROPERTY {operator}\n"),
                operator,
            }
        })
        .collect();
    let negative = model
        .negative
        .iter()
        .map(|claim| {
            let operator = format!("Negative_{}", claim.name);
            line(&mut out, format!("{operator} == {}", expr(&claim.expr)));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\nINVARIANT {operator}\n"),
                operator,
            }
        })
        .collect();
    for assumption in &model.assumptions {
        if model.progress.iter().any(|claim| claim.assumption.same_scope(assumption)) {
            let allowed = model.actions.iter().filter(|action| {
                assumption.only.as_ref().is_none_or(|only| only.contains(&action.name))
                    && !assumption.without.contains(&action.name)
            });
            line(
                &mut out,
                format!(
                    "Assumption_{} == {}",
                    assumption.name,
                    disjoin(allowed.map(|action| action.operator.clone()).collect())
                ),
            );
        }
    }
    let mut progress = Vec::new();
    for claim in &model.progress {
        let name = &claim.name;
        let allowed: Vec<_> = model
            .actions
            .iter()
            .filter(|a| {
                claim.assumption.only.as_ref().is_none_or(|only| only.contains(&a.name))
                    && !claim.assumption.without.contains(&a.name)
            })
            .collect();
        let scoped = format!("Scoped_{name}");
        let scope = model
            .assumptions
            .iter()
            .find(|assumption| assumption.same_scope(&claim.assumption))
            .map_or_else(
                || disjoin(allowed.iter().map(|a| a.operator.clone()).collect()),
                |assumption| format!("Assumption_{}", assumption.name),
            );
        line(&mut out, format!("{scoped} == {scope}"));
        line(
            &mut out,
            format!(
                "Activate_{name} == (__umpire_pending = 0) /\\ ({}) /\\ ~({}) /\\ (__umpire_pending' = 1) /\\ (__umpire_steps' = 0) /\\ UNCHANGED <<ModelVars, __umpire_edge_tick, __umpire_action, __umpire_choice, __umpire_outcome, __umpire_facts>>",
                expr(&claim.trigger),
                expr(&claim.target)
            ),
        );
        let count = claim.bound.map_or_else(
            || "0".into(),
            |bound| {
                format!(
                    "(IF __umpire_steps < {} THEN __umpire_steps + 1 ELSE __umpire_steps)",
                    bound + 1
                )
            },
        );
        let target = prime(&claim.target);
        line(
            &mut out,
            format!(
                "Monitor_{name} == (__umpire_edge_tick' = ~__umpire_edge_tick) /\\ (__umpire_pending' = (IF {} THEN 2 ELSE 1)) /\\ (__umpire_steps' = {count})",
                expr(&target)
            ),
        );
        line(
            &mut out,
            format!("ActiveEdge_{name} == (__umpire_pending = 1) /\\ {scoped} /\\ Monitor_{name}"),
        );
        line(&mut out, format!("PrefixEdge_{name} == (__umpire_pending = 0) /\\ Edge"));
        line(
            &mut out,
            format!(
                "Complete_{name} == ((__umpire_pending = 2) \\/ ((__umpire_pending = 1) /\\ ~ENABLED {scoped}) \\/ ((__umpire_pending = 0) /\\ ~ENABLED ModelNext)) /\\ UNCHANGED vars"
            ),
        );
        line(
            &mut out,
            format!(
                "ProgressNext_{name} == PrefixEdge_{name} \\/ Activate_{name} \\/ ActiveEdge_{name} \\/ Complete_{name}"
            ),
        );
        let mut spec = format!("Init /\\ [][ProgressNext_{name}]_vars");
        if claim.bound.is_none() {
            for action in &allowed {
                if claim.assumption.fair.contains(&action.name) {
                    let fair = format!("Fair_{name}_{}", action.operator);
                    line(
                        &mut out,
                        format!(
                            "{fair} == (__umpire_pending = 1) /\\ {} /\\ Monitor_{name}",
                            action.operator
                        ),
                    );
                    write!(spec, " /\\ WF_vars({fair})").ok();
                }
            }
        }
        line(&mut out, format!("ProgressSpec_{name} == {spec}"));
        line(
            &mut out,
            format!(
                "Maximal_{name} == (([]<>__umpire_edge_tick) /\\ ([]<>(~__umpire_edge_tick))) \\/ (<>[]((__umpire_pending = 1) /\\ ~ENABLED {scoped}))"
            ),
        );
        let operator = format!("Progress_{name}");
        let mut property =
            format!("(Maximal_{name} => ((__umpire_pending = 1) ~> (__umpire_pending = 2)))");
        if let Some(bound) = claim.bound {
            write!(property, " /\\ [](__umpire_steps <= {bound})").ok();
        }
        line(&mut out, format!("{operator} == {property}"));
        progress.push(Progress {
            name: name.clone(),
            cfg: format!("SPECIFICATION ProgressSpec_{name}\nPROPERTY {operator}\n"),
            operator,
            allowed: allowed.iter().map(|a| a.name.clone()).collect(),
            fair: claim.assumption.fair.clone(),
            bound: claim.bound,
        });
    }
    line(&mut out, "====");
    let max_steps = model
        .progress
        .iter()
        .filter_map(|p| p.bound)
        .max()
        .map_or(0, |bound| i128::from(bound) + 1);
    let max_choice =
        model.actions.iter().flat_map(|a| &a.branches).map(|b| b.choice).max().unwrap_or(0) as i128;
    let observer_types = BTreeMap::from([
        ("__umpire_edge_tick".into(), Type::Bool),
        ("__umpire_pending".into(), Type::Int { min: 0, max: 2 }),
        ("__umpire_steps".into(), Type::Int { min: 0, max: max_steps }),
        ("__umpire_action".into(), Type::Text),
        ("__umpire_choice".into(), Type::Int { min: 0, max: max_choice }),
        ("__umpire_outcome".into(), outcome_type.clone()),
        ("__umpire_facts".into(), Type::Seq(Box::new(fact_type.clone()))),
    ]);
    Ok(Module {
        name: model.name.clone(),
        tla: out,
        cfg,
        metadata: Metadata {
            state_name: model.state_name.clone(),
            state_tuple: model.state_tuple,
            world_path: model.world_path.clone(),
            inputs: model.inputs.clone(),
            fields: model
                .fields
                .iter()
                .map(|f| Field { name: f.name.clone(), ty: f.ty.clone() })
                .collect(),
            actions: model
                .actions
                .iter()
                .map(|a| Action {
                    name: a.name.clone(),
                    operator: a.operator.clone(),
                    internal: a.internal,
                })
                .collect(),
            invariants,
            postconditions,
            progress,
            negative,
            observer_fields: OBSERVERS.iter().map(|s| (*s).into()).collect(),
            observer_types,
            outcome_type,
            fact_type,
        },
    })
}
fn line(out: &mut String, value: impl AsRef<str>) {
    out.push_str(value.as_ref());
    out.push('\n');
}
fn conjoin(terms: impl IntoIterator<Item = String>) -> String {
    let terms: Vec<_> = terms.into_iter().map(|s| format!("({s})")).collect();
    if terms.is_empty() { "TRUE".into() } else { terms.join(" /\\ ") }
}
fn disjoin(terms: Vec<String>) -> String {
    if terms.is_empty() {
        "FALSE".into()
    } else {
        terms.into_iter().map(|s| format!("({s})")).collect::<Vec<_>>().join(" \\/ ")
    }
}
fn text(value: &str) -> String {
    format!(
        "\"{}\"",
        value
            .replace('\\', "\\\\")
            .replace('"', "\\\"")
            .replace('\n', "\\n")
            .replace('\r', "\\r")
            .replace('\t', "\\t")
    )
}
fn default(ty: &Type) -> String {
    match ty {
        Type::Bool => "FALSE".into(),
        Type::Int { .. } => "0".into(),
        Type::Enum { variants, .. } => text(&variants[0]),
        Type::Text => "\"\"".into(),
        Type::Unit | Type::Seq(_) => "<<>>".into(),
        Type::Tuple(types) => {
            format!("<<{}>>", types.iter().map(default).collect::<Vec<_>>().join(", "))
        }
    }
}
fn prime(value: &Expr) -> Expr {
    let kind = match &value.kind {
        ExprKind::Var(name, _) => ExprKind::Var(name.clone(), true),
        ExprKind::Unary(op, value) => ExprKind::Unary(*op, Box::new(prime(value))),
        ExprKind::Binary(op, left, right) => {
            ExprKind::Binary(*op, Box::new(prime(left)), Box::new(prime(right)))
        }
        ExprKind::If(cond, yes, no) => {
            ExprKind::If(Box::new(prime(cond)), Box::new(prime(yes)), Box::new(prime(no)))
        }
        other => other.clone(),
    };
    Expr { kind, ty: value.ty.clone() }
}
fn checked(raw: String, ty: &Type) -> String {
    if let Type::Int { min, max } = ty {
        format!(
            "(IF Assert(({raw}) \\in ({}..{}), \"checked Rust integer overflow\") THEN ({raw}) ELSE ({raw}))",
            integer(*min),
            integer(*max)
        )
    } else {
        raw
    }
}
fn quotient(left: &str, right: &str) -> String {
    format!("__umpire_Div(({left}), ({right}))")
}
fn expr(value: &Expr) -> String {
    match &value.kind {
        ExprKind::Bool(v) => {
            if *v {
                "TRUE".into()
            } else {
                "FALSE".into()
            }
        }
        ExprKind::Int(v) => integer(*v),
        ExprKind::Text(v) | ExprKind::Enum(v) => text(v),
        ExprKind::Unit => "<<>>".into(),
        ExprKind::Var(name, primed) => format!("{name}{}", if *primed { "'" } else { "" }),
        ExprKind::Tuple(values) => {
            format!("<<{}>>", values.iter().map(expr).collect::<Vec<_>>().join(", "))
        }
        ExprKind::Unary(op, inner) => {
            let inner = expr(inner);
            match op {
                Unary::Not => format!("~({inner})"),
                Unary::Neg => checked(format!("-({inner})"), &value.ty),
            }
        }
        ExprKind::If(condition, yes, no) => {
            format!("(IF {} THEN {} ELSE {})", expr(condition), expr(yes), expr(no))
        }
        ExprKind::Binary(op, left, right) => {
            let left = expr(left);
            let right = expr(right);
            let symbol = match op {
                Op::Add => "+",
                Op::Sub => "-",
                Op::Mul => "*",
                Op::And => "/\\",
                Op::Or => "\\/",
                Op::Eq => "=",
                Op::Ne => "#",
                Op::Lt => "<",
                Op::Le => "<=",
                Op::Gt => ">",
                Op::Ge => ">=",
                Op::Div | Op::Rem => "",
            };
            let raw = match op {
                Op::Div => quotient(&left, &right),
                Op::Rem => format!("({left}) - ({}) * ({right})", quotient(&left, &right)),
                _ => format!("({left}) {symbol} ({right})"),
            };
            if matches!(op, Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem) {
                checked(raw, &value.ty)
            } else {
                format!("({raw})")
            }
        }
    }
}

fn integer(value: i128) -> String {
    if value == i128::from(i32::MIN) { "(-2147483647 - 1)".into() } else { value.to_string() }
}
