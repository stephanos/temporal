use crate::observers::TLC as OBSERVERS;
use crate::observers::{ACTION, CHOICE, EDGE_TICK, FACTS, INPUT, OUTCOME, PENDING, STEPS};
use crate::{
    Action, Diagnostic, Field, Lifted, Metadata, Module, Progress, Property, Type,
    bind::Entry,
    lower::{
        arithmetic, conjoin, default, disjoin, division, integer, leaf, signed_overflow, symbol,
        text,
    },
    names::operator,
};
use std::collections::BTreeMap;
use std::fmt::Write;
use umpire_front::{
    expr::{Expr, ExprKind, Unary},
    tree::{ActionDef, ActionRef, AssumptionDef, Branch, FieldDef, Start},
};

#[expect(clippy::too_many_lines, reason = "Definitions are emitted in module dependency order")]
pub(crate) fn emit(model: &Lifted) -> Result<Module, Diagnostic> {
    if model.requires_binding() && (model.world_values.is_none() || model.complete_starts.is_none())
    {
        return Err(Diagnostic::of(
            &model.file,
            &model.location,
            "declared World requires a selected World and complete-start binding",
        ));
    }
    if model
        .assumptions
        .iter()
        .chain(model.progress.iter().map(|claim| &claim.assumption))
        .any(|assumption| !assumption.is_resolved())
    {
        return Err(Diagnostic::of(
            &model.file,
            &model.location,
            "unresolved conditional assumption requires a selected World binding",
        ));
    }
    // Only Models with action inputs observe the concrete catalog ordinal of each edge.
    let inputs = model.has_inputs();
    let mut observers = OBSERVERS.to_vec();
    if inputs {
        observers.push(INPUT);
    }
    let observer_types = observer_types(model);
    let mut out = String::new();
    line(&mut out, format!("---- MODULE {} ----", model.name));
    line(&mut out, "EXTENDS Integers, TLC, Sequences");
    for (name, field) in model.names.world.iter().zip(&model.world_fields) {
        line(&mut out, format!("CONSTANT \\* @type: {};\n{name}", crate::snowcat::ty(&field.ty)));
    }
    let mut constants_cfg = String::new();
    if let Some(values) = &model.world_values {
        for (symbol, value) in model.names.world.iter().zip(values) {
            line(&mut constants_cfg, format!("CONSTANT {symbol} = {}", expr(value)));
        }
    }
    typed(
        &mut out,
        "(Int, Int) => Int",
        r#"__umpire_Div(__umpire_a, __umpire_b) ==
    IF Assert(__umpire_b # 0, "Rust integer division by zero") /\ Assert(~(__umpire_a = (-2147483647 - 1) /\ __umpire_b = -1), "Rust integer division overflow")
    THEN IF __umpire_b = (-2147483647 - 1)
         THEN IF __umpire_a = (-2147483647 - 1) THEN 1 ELSE 0
         ELSE IF __umpire_a < 0
              THEN LET (* @type: Int; *) __umpire_d == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
                       (* @type: Int; *) __umpire_n == -(__umpire_a + 1)
                       (* @type: Int; *) __umpire_q == __umpire_n \div __umpire_d
                       (* @type: Int; *) __umpire_r == __umpire_n - __umpire_q * __umpire_d
                       (* @type: Int; *) __umpire_negative == -__umpire_q - (IF __umpire_r = __umpire_d - 1 THEN 1 ELSE 0)
                   IN IF __umpire_b < 0 THEN -__umpire_negative ELSE __umpire_negative
              ELSE IF __umpire_b < 0 THEN -(__umpire_a \div (-__umpire_b)) ELSE __umpire_a \div __umpire_b
    ELSE 0"#,
    );
    line(&mut out, "");
    for (name, value) in &model.constants {
        let name = model.names.constants.get(name).unwrap_or(name);
        typed(&mut out, &crate::snowcat::ty(&value.ty), format!("{name} == {}", expr(value)));
    }
    let fields: Vec<_> = model.fields.iter().map(|f| f.name.as_str()).collect();
    let mut variables = fields.clone();
    variables.extend(&observers);
    for field in &model.fields {
        line(
            &mut out,
            format!("VARIABLE \\* @type: {};\n{}", crate::snowcat::ty(&field.ty), field.name),
        );
    }
    for name in &observers {
        line(
            &mut out,
            format!("VARIABLE \\* @type: {};\n{name}", crate::snowcat::ty(&observer_types[*name])),
        );
    }
    let field_types: Vec<_> = model.fields.iter().map(|f| crate::snowcat::ty(&f.ty)).collect();
    let variable_types = field_types
        .iter()
        .cloned()
        .chain(observers.iter().map(|name| crate::snowcat::ty(&observer_types[*name])));
    typed(
        &mut out,
        &crate::snowcat::tuple(variable_types),
        format!("vars == <<{}>>", variables.join(", ")),
    );
    typed(
        &mut out,
        &crate::snowcat::tuple(field_types),
        format!("ModelVars == <<{}>>", fields.join(", ")),
    );
    line(&mut out, "");
    let mut initial = Vec::new();
    if let Some(starts) = &model.complete_starts {
        initial.push(disjoin(starts.iter().map(|state| {
            conjoin(
                model
                    .fields
                    .iter()
                    .zip(state)
                    .map(|(field, value)| format!("{} = {}", field.name, expr(value))),
            )
        })));
    } else {
        for field in &model.fields {
            initial.push(match &field.start {
                Start::Computed => {
                    return Err(Diagnostic::of(
                        &model.file,
                        &field.location,
                        "computed starts require a complete-start binding",
                    ));
                }
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
    initial.extend([
        format!("{EDGE_TICK} = FALSE"),
        format!("{PENDING} = 0"),
        format!("{STEPS} = 0"),
        format!("{ACTION} = \"\""),
        format!("{CHOICE} = 0"),
        format!("{OUTCOME} = {}", default(&outcome_type)),
        format!("{FACTS} = <<>>"),
    ]);
    if inputs {
        initial.push(format!("{INPUT} = 0"));
    }
    predicate(&mut out, format!("Init == {}", conjoin(initial)));
    line(&mut out, "");
    let catalog = model.catalog.as_deref().unwrap_or_default();
    for (index, action) in model.actions.iter().enumerate() {
        let entries: Vec<_> = catalog.iter().filter(|entry| entry.action == index).collect();
        let identity = if action.fields.is_empty() {
            entries.first().map(|entry| entry.ordinal.to_string())
        } else {
            Some(format!("{}_Inputs[{}][1]", operator(&action.name), model.names.index))
        };
        let identity = identity.as_deref();
        let branches = disjoin(action.branches.iter().map(|branch| {
            let mut shared = Shared::new(&model.names.shared);
            step(model, &action.name, branch, identity, &mut |value| {
                shared.count(value);
                String::new()
            });
            let terms = conjoin(step(model, &action.name, branch, identity, &mut |value| {
                shared.emit(value)
            }));
            let mut definitions = String::new();
            for (name, value) in &branch.definitions {
                let _ = write!(
                    definitions,
                    "LET (* @type: {}; *) {name} == {} IN ",
                    crate::snowcat::ty(&value.ty),
                    expr(value)
                );
            }
            format!("{definitions}{}{terms}", shared.definitions.concat())
        }));
        if action.fields.is_empty() {
            predicate(&mut out, format!("{} == {branches}", operator(&action.name)));
        } else {
            input_relation(&mut out, model, index, &entries, branches);
        }
    }
    predicate(
        &mut out,
        format!("ModelNext == {}", disjoin(model.actions.iter().map(|a| operator(&a.name)))),
    );
    predicate(&mut out, format!("ModelEdge == {EDGE_TICK}' # {EDGE_TICK}"));
    predicate(
        &mut out,
        format!(
            "Edge == ModelNext /\\ ({EDGE_TICK}' = ~{EDGE_TICK}) /\\ UNCHANGED <<{PENDING}, {STEPS}>>"
        ),
    );
    predicate(&mut out, "Complete == ~ENABLED ModelNext /\\ UNCHANGED vars");
    predicate(&mut out, "Next == Edge \\/ Complete");
    predicate(&mut out, "Spec == Init /\\ [][Next]_vars");
    line(&mut out, "");
    let mut cfg = format!("SPECIFICATION Spec\n{constants_cfg}");
    let invariants = model
        .invariants
        .iter()
        .map(|claim| {
            let operator = format!("Invariant_{}", claim.name);
            predicate(&mut out, format!("{operator} == {}", expr(&claim.expr)));
            line(&mut cfg, format!("INVARIANT {operator}"));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\n{constants_cfg}INVARIANT {operator}\n"),
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
                |action| format!("ModelEdge /\\ ({ACTION}' = {})", text(action)),
            );
            predicate(
                &mut out,
                format!("{operator} == [][~({applies}) \\/ ({})]_vars", expr(&claim.expr)),
            );
            line(&mut cfg, format!("PROPERTY {operator}"));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\n{constants_cfg}PROPERTY {operator}\n"),
                operator,
            }
        })
        .collect();
    let negative = model
        .negative
        .iter()
        .map(|claim| {
            let operator = format!("Negative_{}", claim.name);
            predicate(&mut out, format!("{operator} == {}", expr(&claim.expr)));
            Property {
                name: claim.name.clone(),
                cfg: format!("SPECIFICATION Spec\n{constants_cfg}INVARIANT {operator}\n"),
                operator,
            }
        })
        .collect();
    for assumption in &model.assumptions {
        if model.progress.iter().any(|claim| claim.assumption.same_scope(assumption)) {
            predicate(
                &mut out,
                format!(
                    "Assumption_{} == {}",
                    assumption.name,
                    disjoin(scoped_relations(model, assumption))
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
            .enumerate()
            .filter(|(index, action)| {
                positions(model, *index, action)
                    .iter()
                    .any(|(_, reference, _)| claim.assumption.allows(reference))
            })
            .map(|(_, action)| action)
            .collect();
        let scoped = format!("Scoped_{name}");
        let scope = model
            .assumptions
            .iter()
            .find(|assumption| assumption.same_scope(&claim.assumption))
            .map_or_else(
                || disjoin(scoped_relations(model, &claim.assumption)),
                |assumption| format!("Assumption_{}", assumption.name),
            );
        predicate(&mut out, format!("{scoped} == {scope}"));
        predicate(
            &mut out,
            format!(
                "Activate_{name} == ({PENDING} = 0) /\\ ({}) /\\ ~({}) /\\ ({PENDING}' = 1) /\\ ({STEPS}' = 0) /\\ UNCHANGED <<ModelVars, {EDGE_TICK}, {ACTION}, {CHOICE}, {OUTCOME}, {FACTS}{}>>",
                expr(&claim.trigger),
                expr(&claim.target),
                if inputs { format!(", {INPUT}") } else { String::new() }
            ),
        );
        let count = claim.bound.map_or_else(
            || "0".into(),
            |bound| format!("(IF {STEPS} < {} THEN {STEPS} + 1 ELSE {STEPS})", bound + 1),
        );
        let target = prime(&claim.target);
        predicate(
            &mut out,
            format!(
                "Monitor_{name} == ({EDGE_TICK}' = ~{EDGE_TICK}) /\\ ({PENDING}' = (IF {} THEN 2 ELSE 1)) /\\ ({STEPS}' = {count})",
                expr(&target)
            ),
        );
        predicate(
            &mut out,
            format!("ActiveEdge_{name} == ({PENDING} = 1) /\\ {scoped} /\\ Monitor_{name}"),
        );
        predicate(&mut out, format!("PrefixEdge_{name} == ({PENDING} = 0) /\\ Edge"));
        predicate(
            &mut out,
            format!(
                "Complete_{name} == (({PENDING} = 2) \\/ (({PENDING} = 1) /\\ ~ENABLED {scoped}) \\/ (({PENDING} = 0) /\\ ~ENABLED ModelNext)) /\\ UNCHANGED vars"
            ),
        );
        predicate(
            &mut out,
            format!(
                "ProgressNext_{name} == PrefixEdge_{name} \\/ Activate_{name} \\/ ActiveEdge_{name} \\/ Complete_{name}"
            ),
        );
        let mut spec = format!("Init /\\ [][ProgressNext_{name}]_vars");
        if claim.bound.is_none() {
            for (fair, relation) in fair_relations(model, name, &claim.assumption) {
                predicate(
                    &mut out,
                    format!("{fair} == ({PENDING} = 1) /\\ {relation} /\\ Monitor_{name}"),
                );
                write!(spec, " /\\ WF_vars({fair})").ok();
            }
        }
        predicate(&mut out, format!("ProgressSpec_{name} == {spec}"));
        predicate(
            &mut out,
            format!(
                "Maximal_{name} == (([]<>{EDGE_TICK}) /\\ ([]<>(~{EDGE_TICK}))) \\/ (<>[](({PENDING} = 1) /\\ ~ENABLED {scoped}))"
            ),
        );
        let operator = format!("Progress_{name}");
        let mut property = format!("(Maximal_{name} => (({PENDING} = 1) ~> ({PENDING} = 2)))");
        if let Some(bound) = claim.bound {
            write!(property, " /\\ []({STEPS} <= {bound})").ok();
        }
        predicate(&mut out, format!("{operator} == {property}"));
        progress.push(Progress {
            name: name.clone(),
            cfg: format!("SPECIFICATION ProgressSpec_{name}\n{constants_cfg}PROPERTY {operator}\n"),
            operator,
            allowed: allowed.iter().map(|a| a.name.clone()).collect(),
            fair: claim.assumption.fair.iter().fold(Vec::new(), |mut names, action| {
                if !names.contains(&action.name) {
                    names.push(action.name.clone());
                }
                names
            }),
            allowed_inputs: ordinals(model, inputs, |reference| claim.assumption.allows(reference)),
            fair_inputs: ordinals(model, inputs, |reference| {
                claim.assumption.allows(reference) && claim.assumption.is_fair(reference)
            }),
            bound: claim.bound,
        });
    }
    line(&mut out, "====");
    Ok(Module {
        name: model.name.clone(),
        tla: out,
        cfg,
        metadata: Metadata {
            state_name: model.state_name.clone(),
            world_path: model.world_path.clone(),
            parameters: model
                .world_fields
                .iter()
                .zip(&model.names.world)
                .map(|(field, symbol)| crate::Parameter {
                    field: field.name.clone(),
                    symbol: symbol.clone(),
                    ty: field.ty.clone(),
                })
                .collect(),
            binding: model.binding.clone(),
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
                    operator: operator(&a.name),
                    internal: a.internal,
                    fields: a
                        .fields
                        .iter()
                        .map(|field| Field { name: field.name.clone(), ty: field.ty.clone() })
                        .collect(),
                })
                .collect(),
            catalog: if inputs { model.concrete() } else { Vec::new() },
            invariants,
            postconditions,
            arithmetic: Vec::new(),
            progress,
            negative,
            observer_fields: observers.iter().map(|s| (*s).into()).collect(),
            observer_types,
            outcome_type,
        },
    })
}
/// One branch's conjuncts, with each of its expressions as `value` emits it.
fn step<'e>(
    model: &Lifted,
    action: &str,
    branch: &'e Branch,
    identity: Option<&str>,
    value: &mut dyn FnMut(&'e Expr) -> String,
) -> Vec<String> {
    let mut terms = vec![value(&branch.guard)];
    for check in &branch.checks {
        let check = value(check);
        terms.push(format!("({check}) = ({check})"));
    }
    let updated = |f: &FieldDef| {
        branch
            .updates
            .get(&f.name)
            .filter(|value| value.kind != ExprKind::Var(f.name.clone(), false))
    };
    let may_reject = !matches!(&branch.outcome.kind, ExprKind::Unit)
        && branch.outcome.kind != ExprKind::Enum(ACCEPTED.into());
    let changed: Vec<_> =
        model.fields.iter().filter_map(|f| updated(f).map(|update| (f, update))).collect();
    if may_reject && !changed.is_empty() {
        // A rejecting step that changed its state is a Model error, as natively.
        let outcome = value(&branch.outcome);
        let unchanged = conjoin(
            changed
                .iter()
                .map(|(f, update)| format!("{} = {}", value(update), f.name))
                .collect::<Vec<_>>(),
        );
        terms.push(format!(
            "Assert(({outcome} = {}) \\/ ({unchanged}), \"rejecting step changed state\")",
            text(ACCEPTED)
        ));
    }
    if !branch.preserve_self_loop {
        let mut steps: Vec<_> = model
            .fields
            .iter()
            .map(|f| {
                let next = branch.updates.get(&f.name).map_or_else(|| f.name.clone(), &mut *value);
                format!("{next} # {}", f.name)
            })
            .collect();
        // A rejection is emitted even when it leaves the state unchanged.
        if may_reject {
            steps.push(format!("{} # {}", value(&branch.outcome), text(ACCEPTED)));
        }
        terms.push(disjoin(steps));
    }
    for field in &model.fields {
        let next = branch.updates.get(&field.name).map_or_else(|| field.name.clone(), &mut *value);
        terms.push(format!("{}' = {next}", field.name));
    }
    terms.extend([
        format!("{ACTION}' = {}", text(action)),
        format!("{CHOICE}' = 0"),
        format!("{OUTCOME}' = {}", value(&branch.outcome)),
        format!("{FACTS}' = {}", value(&branch.facts)),
    ]);
    if model.has_inputs()
        && let Some(identity) = identity
    {
        terms.push(format!("{INPUT}' = {identity}"));
    }
    terms
}
/// The stock accepted outcome, as the front end spells it.
const ACCEPTED: &str = "Outcome::Accepted";

/// A payload variant's concrete inputs as `<<ordinal, values..>>` rows in catalog order, its
/// relation at one catalog position, and the finite existential choice over every position.
fn input_relation(
    out: &mut String,
    model: &Lifted,
    action: usize,
    entries: &[&Entry],
    branches: String,
) {
    let symbols = &model.names.inputs[action];
    let operator = operator(&model.actions[action].name);
    let index = &model.names.index;
    let rows = entries
        .iter()
        .map(|entry| {
            let mut row = vec![entry.ordinal.to_string()];
            row.extend(entry.values.iter().map(expr));
            format!("<<{}>>", row.join(", "))
        })
        .collect::<Vec<_>>();
    let row_type = crate::snowcat::tuple(
        std::iter::once("Int".into())
            .chain(model.actions[action].fields.iter().map(|f| crate::snowcat::ty(&f.ty))),
    );
    typed(
        out,
        &format!("Seq({row_type})"),
        format!("{operator}_Inputs == <<{}>>", rows.join(", ")),
    );
    let mut body = branches;
    for (position, symbol) in symbols.iter().enumerate().rev() {
        body = format!(
            "LET (* @type: {}; *) {symbol} == {operator}_Inputs[{index}][{}] IN {body}",
            crate::snowcat::ty(&model.actions[action].fields[position].ty),
            position + 2
        );
    }
    typed(out, "Int => Bool", format!("{operator}_At({index}) == {body}"));
    predicate(
        out,
        format!("{operator} == \\E {index} \\in DOMAIN {operator}_Inputs : {operator}_At({index})"),
    );
}

/// Each concrete catalog position of a variant: its one-based position, exact identity and
/// canonical ordinal. A unit variant outside an input catalog is its own sole position.
fn positions(model: &Lifted, index: usize, action: &ActionDef) -> Vec<(usize, ActionRef, usize)> {
    let reference =
        |values: &[Expr]| ActionRef { name: action.name.clone(), values: values.to_vec() };
    match &model.catalog {
        Some(catalog) if model.has_inputs() => catalog
            .iter()
            .filter(|entry| entry.action == index)
            .enumerate()
            .map(|(position, entry)| (position + 1, reference(&entry.values), entry.ordinal))
            .collect(),
        _ => vec![(1, reference(&[]), 0)],
    }
}

/// The admitted relation of each variant: the whole existential relation when every
/// position is admitted, otherwise the choice restricted to the admitted exact positions.
fn scoped_relations(model: &Lifted, assumption: &AssumptionDef) -> Vec<String> {
    let mut relations = Vec::new();
    for (index, action) in model.actions.iter().enumerate() {
        let all = positions(model, index, action);
        let admitted: Vec<_> =
            all.iter().filter(|(_, reference, _)| assumption.allows(reference)).collect();
        if admitted.len() == all.len() {
            relations.push(operator(&action.name));
        } else if !admitted.is_empty() {
            let index = &model.names.index;
            let selected =
                admitted.iter().map(|(position, _, _)| position.to_string()).collect::<Vec<_>>();
            relations.push(format!(
                "\\E {index} \\in {{{}}} : {}_At({index})",
                selected.join(", "),
                operator(&action.name)
            ));
        }
    }
    relations
}

/// Weak fairness obligations: one per explicitly fair unit action, and one per exact fair
/// payload position, never one over a variant's existential union. Positional duplicates
/// share their canonical ordinal and so one obligation.
fn fair_relations(
    model: &Lifted,
    claim: &str,
    assumption: &AssumptionDef,
) -> Vec<(String, String)> {
    let mut relations = Vec::new();
    for (index, action) in model.actions.iter().enumerate() {
        let mut seen = Vec::new();
        for (position, reference, ordinal) in positions(model, index, action) {
            if !assumption.allows(&reference) || !assumption.is_fair(&reference) {
                continue;
            }
            let operator = operator(&action.name);
            if action.fields.is_empty() {
                relations.push((format!("Fair_{claim}_{operator}"), operator));
            } else if !seen.contains(&ordinal) {
                seen.push(ordinal);
                relations.push((
                    format!("Fair_{claim}_{operator}_{position}"),
                    format!("{operator}_At({position})"),
                ));
            }
        }
    }
    relations
}

/// Canonical catalog ordinals selected by a predicate, for typed progress metadata.
fn ordinals(model: &Lifted, inputs: bool, select: impl Fn(&ActionRef) -> bool) -> Vec<usize> {
    if !inputs {
        return Vec::new();
    }
    let mut out = Vec::new();
    for (index, action) in model.actions.iter().enumerate() {
        for (_, reference, ordinal) in positions(model, index, action) {
            if select(&reference) && !out.contains(&ordinal) {
                out.push(ordinal);
            }
        }
    }
    out
}

/// A typed expression as emitted TLA+, for diagnostics.
pub(crate) fn rendered(value: &Expr) -> String {
    expr(value)
}

pub(super) fn line(out: &mut String, value: impl AsRef<str>) {
    out.push_str(value.as_ref());
    out.push('\n');
}
fn prime(value: &Expr) -> Expr {
    let mut value = value.clone();
    prime_in_place(&mut value);
    value
}
fn prime_in_place(value: &mut Expr) {
    if let ExprKind::Var(_, primed) = &mut value.kind {
        *primed = true;
    }
    for child in value.kind.children_mut() {
        prime_in_place(child);
    }
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
/// `__umpire_Div` over atomic operands, asserting Rust's signed overflow case.
fn quotient(left: &str, right: &str, ty: &Type) -> String {
    let quotient = format!("__umpire_Div({left}, {right})");
    match signed_overflow(left, right, ty) {
        Some(overflow) => format!(
            "(IF Assert(~({overflow}), \"Rust signed division/remainder overflow\") THEN {quotient} ELSE 0)"
        ),
        None => quotient,
    }
}
fn expr(value: &Expr) -> String {
    render(value, &mut |child| expr(child))
}
/// One node as TLA+, with each child as `child` emits it, once per textual occurrence.
fn render<'e>(value: &'e Expr, child: &mut dyn FnMut(&'e Expr) -> String) -> String {
    let mut list =
        |values: &'e [Expr]| values.iter().map(&mut *child).collect::<Vec<_>>().join(", ");
    match &value.kind {
        kind @ (ExprKind::Bool(_)
        | ExprKind::Int(_)
        | ExprKind::Text(_)
        | ExprKind::Enum(_)
        | ExprKind::Unit
        | ExprKind::World(_)
        | ExprKind::Input(_)) => leaf(kind),
        ExprKind::Var(name, primed) => format!("{name}{}", if *primed { "'" } else { "" }),
        ExprKind::Local(name) => name.clone(),
        ExprKind::Outcome => format!("{OUTCOME}'"),
        ExprKind::Tuple(values) | ExprKind::Sequence(values) => format!("<<{}>>", list(values)),
        ExprKind::Length(sequence) => format!("Len({})", child(sequence)),
        ExprKind::Bounded(inner) | ExprKind::Payload(inner) => child(inner),
        ExprKind::Some(inner) => checked(
            format!("({}) + 1", child(inner)),
            &Type::Int { min: 1, max: i128::from(i32::MAX) },
        ),
        ExprKind::OptionalOr(option, fallback) => {
            let option = child(option);
            let fallback = child(fallback);
            format!(
                "(IF ({fallback}) = ({fallback}) THEN (IF ({option}) # 0 THEN ({option}) - 1 ELSE ({fallback})) ELSE ({fallback}))"
            )
        }
        ExprKind::Check(_, value) => child(value),
        ExprKind::Checks(checks) => {
            format!("(<<{}>> = <<{}>>)", list(checks), vec!["TRUE"; checks.len()].join(", "))
        }
        ExprKind::Append(sequence, value) => {
            format!("Append({}, {})", child(sequence), child(value))
        }
        ExprKind::GetOr(sequence, index, fallback) => {
            let sequence = child(sequence);
            let index = child(index);
            let fallback = child(fallback);
            format!(
                "(IF ({fallback}) = ({fallback}) THEN (IF ({index}) < Len({sequence}) THEN ({sequence})[({index}) + 1] ELSE ({fallback})) ELSE ({fallback}))"
            )
        }
        ExprKind::Unary(op, inner) => {
            let inner = child(inner);
            match op {
                Unary::Not => format!("~({inner})"),
                Unary::Neg => checked(format!("-({inner})"), &value.ty),
            }
        }
        ExprKind::If(condition, yes, no) => {
            format!("(IF {} THEN {} ELSE {})", child(condition), child(yes), child(no))
        }
        ExprKind::Binary(op, left, right) => {
            let left = format!("({})", child(left));
            let right = format!("({})", child(right));
            let raw = match symbol(*op) {
                Some(symbol) => format!("{left} {symbol} {right}"),
                None => division(*op, &left, &right, &quotient(&left, &right, &value.ty)),
            };
            if arithmetic(*op) { checked(raw, &value.ty) } else { format!("({raw})") }
        }
    }
}

/// Large repeated subexpressions, emitted once within their branch's local definitions.
struct Shared<'a> {
    nodes: std::collections::HashMap<&'a Expr, Node>,
    definitions: Vec<String>,
    prefix: &'a str,
}
#[derive(Default)]
struct Node {
    /// Textual occurrences: under each distinct parent, and as a root.
    uses: usize,
    /// Inlined length, saturating.
    length: usize,
    name: Option<String>,
}
const SHARED: usize = 400;
impl<'a> Shared<'a> {
    fn new(prefix: &'a str) -> Self {
        Self { nodes: std::collections::HashMap::new(), definitions: Vec::new(), prefix }
    }
    /// Count one textual occurrence of `value`, and its children's the first time.
    fn count(&mut self, value: &'a Expr) {
        let node = self.nodes.entry(value).or_default();
        node.uses += 1;
        if node.uses > 1 {
            return;
        }
        let mut children = Vec::new();
        let mut length = render(value, &mut |child| {
            children.push(child);
            String::new()
        })
        .len();
        for child in children {
            self.count(child);
            length = length.saturating_add(self.nodes[child].length);
        }
        if let Some(node) = self.nodes.get_mut(value) {
            node.length = length;
        }
    }
    fn emit(&mut self, value: &'a Expr) -> String {
        let shared =
            self.nodes.get(value).is_some_and(|node| node.uses > 1 && node.length > SHARED);
        if !shared {
            return render(value, &mut |child| self.emit(child));
        }
        if let Some(name) = self.nodes.get(value).and_then(|node| node.name.clone()) {
            return name;
        }
        let body = render(value, &mut |child| self.emit(child));
        let name = format!("{}{}", self.prefix, self.definitions.len() + 1);
        self.definitions.push(format!(
            "LET (* @type: {}; *) {name} == {body} IN ",
            crate::snowcat::ty(&value.ty)
        ));
        if let Some(node) = self.nodes.get_mut(value) {
            node.name = Some(name.clone());
        }
        name
    }
}

fn observer_types(model: &Lifted) -> BTreeMap<String, Type> {
    let max_steps = model
        .progress
        .iter()
        .filter_map(|p| p.bound)
        .max()
        .map_or(0, |bound| i128::from(bound) + 1);
    let mut observer_types = BTreeMap::from([
        (EDGE_TICK.into(), Type::Bool),
        (PENDING.into(), Type::Int { min: 0, max: 2 }),
        (STEPS.into(), Type::Int { min: 0, max: max_steps }),
        (ACTION.into(), Type::Text),
        (CHOICE.into(), Type::Int { min: 0, max: 0 }),
        (OUTCOME.into(), model.outcome_type.clone()),
        (FACTS.into(), Type::Seq(Box::new(model.fact_type.clone()))),
    ]);
    if model.has_inputs() {
        let max = i128::try_from(model.catalog.as_ref().map_or(0, Vec::len).saturating_sub(1))
            .unwrap_or(i128::MAX);
        observer_types.insert(INPUT.into(), Type::Int { min: 0, max });
    }
    observer_types
}

fn typed(out: &mut String, kind: &str, value: impl AsRef<str>) {
    line(out, format!("\\* @type: {kind};"));
    line(out, value);
}
fn predicate(out: &mut String, value: impl AsRef<str>) {
    typed(out, "Bool", value);
}

#[cfg(test)]
#[path = "emit_optional_tests.rs"]
mod optional_tests;
