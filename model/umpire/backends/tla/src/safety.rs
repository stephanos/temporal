mod value;
use crate::{
    Diagnostic, Lifted, Module, Property, Type,
    emit::{conjoin, default, disjoin, line, text},
    names::operator,
    snowcat::ty,
};
use std::collections::BTreeMap;
use umpire_front::tree::{Branch, Start};
use value::Values;
const ERROR: &str = "__umpire_arithmetic_error";
const HAS_STEP: &str = "__umpire_has_step";
const STUTTER: &str = "__umpire_stutter";

fn definition(out: &mut String, name: &str, kind: &str, body: impl AsRef<str>) {
    line(out, format!("\\* @type: {kind};\n{name} == {}", body.as_ref()));
}
fn property(name: String, operator: String, cfg: &str) -> Property {
    Property { name, cfg: format!("{cfg}INVARIANT {operator}\n"), operator }
}

#[expect(
    clippy::too_many_lines,
    reason = "Safety definitions follow their source dependency order"
)]
pub(super) fn emit(model: &Lifted) -> Result<Module, Diagnostic> {
    validate(model)?;
    let mut module = crate::emit::emit(model)?;
    let mut out = format!("---- MODULE {} ----\nEXTENDS Integers, Sequences\n", model.name);
    definition(
        &mut out,
        "__umpire_SafeDiv(__umpire_a, __umpire_b)",
        "(Int, Int) => Int",
        r"IF __umpire_b = 0 THEN 0 ELSE
        LET (* @type: Int; *) __umpire_abs_a == IF __umpire_a < 0 THEN -__umpire_a ELSE __umpire_a
            (* @type: Int; *) __umpire_abs_b == IF __umpire_b < 0 THEN -__umpire_b ELSE __umpire_b
        IN IF (__umpire_a < 0) = (__umpire_b < 0) THEN __umpire_abs_a \div __umpire_abs_b ELSE -(__umpire_abs_a \div __umpire_abs_b)",
    );
    for (symbol, field) in model.names.world.iter().zip(&model.world_fields) {
        line(&mut out, format!("CONSTANT \\* @type: {};\n{symbol}", ty(&field.ty)));
    }
    for (name, value) in &model.constants {
        let mut values = Values::new(&model.names.shared);
        let v = values.emit(value);
        definition(
            &mut out,
            model.names.constants.get(name).unwrap_or(name),
            &ty(&value.ty),
            values.scope(v.value),
        );
    }
    let mut observers = BTreeMap::from([
        (ERROR.into(), Type::Bool),
        (HAS_STEP.into(), Type::Bool),
        (STUTTER.into(), Type::Bool),
        ("__umpire_invalid_step".into(), Type::Bool),
        ("__umpire_action".into(), Type::Text),
        (
            "__umpire_input".into(),
            Type::Int {
                min: 0,
                max: model.catalog.as_ref().map_or(0, |c| c.len().saturating_sub(1) as i128),
            },
        ),
        ("__umpire_choice".into(), Type::Int { min: 0, max: 0 }),
        ("__umpire_outcome".into(), model.outcome_type.clone()),
        ("__umpire_facts".into(), Type::Seq(Box::new(model.fact_type.clone()))),
    ]);
    for field in &model.fields {
        observers.insert(format!("__umpire_before_{}", field.name), field.ty.clone());
    }
    for field in &model.fields {
        line(&mut out, format!("VARIABLE \\* @type: {};\n{}", ty(&field.ty), field.name));
    }
    for (name, kind) in &observers {
        line(&mut out, format!("VARIABLE \\* @type: {};\n{name}", ty(kind)));
    }
    let variables: Vec<_> = model
        .fields
        .iter()
        .map(|f| (f.name.clone(), f.ty.clone()))
        .chain(observers.iter().map(|(n, t)| (n.clone(), t.clone())))
        .collect();
    definition(
        &mut out,
        "vars",
        &crate::snowcat::tuple(variables.iter().map(|(_, t)| ty(t))),
        format!("<<{}>>", variables.iter().map(|(n, _)| n.as_str()).collect::<Vec<_>>().join(", ")),
    );
    let mut values = Values::new(&model.names.shared);
    let (starts, starts_valid) = initial_states(model, &mut values)?;
    let mut initial = Vec::new();
    for field in &model.fields {
        initial.push(format!("__umpire_before_{} = {}", field.name, default(&field.ty)));
    }
    initial.extend([
        format!("{HAS_STEP} = FALSE"),
        format!("{STUTTER} = TRUE"),
        "__umpire_invalid_step = FALSE".into(),
        "__umpire_action = \"\"".into(),
        "__umpire_input = 0".into(),
        "__umpire_choice = 0".into(),
        format!("__umpire_outcome = {}", default(&model.outcome_type)),
        "__umpire_facts = <<>>".into(),
    ]);
    let cfg =
        module.cfg.lines().filter(|l| l.starts_with("CONSTANT ")).collect::<Vec<_>>().join("\n");
    let cfg = format!("INIT Init\nNEXT Next\n{cfg}\n");
    let initial_claims = claim_valid(model, false, false);
    initial.extend([starts, format!("{ERROR} = ~({starts_valid} /\\ ({initial_claims}))")]);
    definition(&mut out, "Init", "Bool", values.scope(conjoin(initial)));
    let successor_claims = claim_valid(model, true, true);
    for (index, action) in model.actions.iter().enumerate() {
        let entries: Vec<_> = model
            .catalog
            .as_deref()
            .unwrap_or_default()
            .iter()
            .filter(|e| e.action == index)
            .collect();
        let ordinal = if action.fields.is_empty() {
            entries.first().map_or("0".into(), |e| e.ordinal.to_string())
        } else {
            format!("{}_Inputs[{}][1]", operator(&action.name), model.names.index)
        };
        let body = disjoin(
            action
                .branches
                .iter()
                .map(|branch| step(model, &action.name, branch, &ordinal, &successor_claims))
                .collect(),
        );
        let name = operator(&action.name);
        if action.fields.is_empty() {
            definition(&mut out, &name, "Bool", body);
        } else {
            let rows: Vec<_> = entries
                .iter()
                .map(|e| {
                    let mut row = vec![e.ordinal.to_string()];
                    row.extend(e.values.iter().map(crate::emit::rendered));
                    format!("<<{}>>", row.join(", "))
                })
                .collect();
            let row = crate::snowcat::tuple(
                std::iter::once("Int".into()).chain(action.fields.iter().map(|f| ty(&f.ty))),
            );
            definition(
                &mut out,
                &format!("{name}_Inputs"),
                &format!("Seq({row})"),
                format!("<<{}>>", rows.join(", ")),
            );
            let mut body = body;
            for (pos, (symbol, field)) in
                model.names.inputs[index].iter().zip(&action.fields).enumerate().rev()
            {
                body = format!(
                    "LET \\* @type: {};\n{symbol} == {name}_Inputs[{}][{}] IN {body}",
                    ty(&field.ty),
                    model.names.index,
                    pos + 2
                );
            }
            definition(&mut out, &format!("{name}_At({})", model.names.index), "Int => Bool", body);
            definition(
                &mut out,
                &name,
                "Bool",
                format!(
                    "\\E {} \\in DOMAIN {name}_Inputs : {name}_At({})",
                    model.names.index, model.names.index
                ),
            );
        }
    }
    definition(
        &mut out,
        "ModelNext",
        "Bool",
        disjoin(model.actions.iter().map(|a| operator(&a.name)).collect()),
    );
    let preserved = variables
        .iter()
        .filter(|(n, _)| n != STUTTER)
        .map(|(n, _)| n.as_str())
        .collect::<Vec<_>>()
        .join(", ");
    definition(
        &mut out,
        "Stutter",
        "Bool",
        format!("{STUTTER}' = TRUE /\\ UNCHANGED <<{preserved}>>"),
    );
    definition(&mut out, "Next", "Bool", "ModelNext \\/ Stutter");
    module.metadata.invariants.clear();
    module.metadata.postconditions.clear();
    module.metadata.negative.clear();
    module.metadata.progress.clear();
    for (claims, negative) in [(&model.invariants, false), (&model.negative, true)] {
        for claim in claims {
            let name =
                format!("{}_{}", if negative { "Negative" } else { "Invariant" }, claim.name);
            let mut values = Values::new(&model.names.shared);
            let v = values.emit(&claim.expr);
            definition(&mut out, &name, "Bool", values.scope(format!("{ERROR} \\/ {}", v.value)));
            let p = property(claim.name.clone(), name, &cfg);
            if negative {
                module.metadata.negative.push(p);
            } else {
                module.metadata.invariants.push(p);
            }
        }
    }
    for claim in &model.postconditions {
        let mut values = Values::new(&model.names.shared);
        values.post = true;
        let v = values.emit(&claim.expr);
        let name = format!("Post_{}", claim.name);
        let applies = claim.after.as_ref().map_or_else(
            || HAS_STEP.into(),
            |a| format!("{HAS_STEP} /\\ __umpire_action = {}", text(a)),
        );
        definition(
            &mut out,
            &name,
            "Bool",
            values.scope(format!("{ERROR} \\/ ~({applies}) \\/ {}", v.value)),
        );
        module.metadata.postconditions.push(property(claim.name.clone(), name, &cfg));
    }
    definition(&mut out, "ArithmeticSafe", "Bool", format!("~{ERROR}"));
    definition(&mut out, "ValidStep", "Bool", "~__umpire_invalid_step");
    module.metadata.arithmetic = vec![property("arithmetic".into(), "ArithmeticSafe".into(), &cfg)];
    module.metadata.invariants.push(property("valid_step".into(), "ValidStep".into(), &cfg));
    line(&mut out, "====");
    module.metadata.observer_fields = observers.keys().cloned().collect();
    module.metadata.observer_types = observers;
    module.cfg = cfg;
    for p in module
        .metadata
        .invariants
        .iter()
        .chain(&module.metadata.postconditions)
        .chain(&module.metadata.arithmetic)
    {
        line(&mut module.cfg, format!("INVARIANT {}", p.operator));
    }
    module.tla = out;
    Ok(module)
}

fn claim_valid(model: &Lifted, successor: bool, post: bool) -> String {
    let mut values = Values::new(&model.names.safety_claims);
    values.successor = successor;
    let mut terms: Vec<_> = model
        .invariants
        .iter()
        .chain(&model.negative)
        .map(|c| values.emit(&c.expr).valid)
        .collect();
    if post {
        values.post = true;
        // A separate frame keeps structurally equal pre/post expressions in their own environment.
        let mut posts = Values::new(&model.names.safety_posts);
        posts.successor = successor;
        posts.post = true;
        for c in &model.postconditions {
            let v = posts.emit(&c.expr);
            terms.push(c.after.as_ref().map_or(v.valid.clone(), |a| {
                format!("IF __umpire_action' = {} THEN {} ELSE TRUE", text(a), v.valid)
            }));
        }
        let body = conjoin(terms);
        return values.scope(posts.scope(body));
    }
    values.scope(conjoin(terms))
}
fn step(model: &Lifted, action: &str, branch: &Branch, ordinal: &str, claims: &str) -> String {
    let mut values = Values::new(&model.names.shared);
    for (name, e) in &branch.definitions {
        let v = values.emit(e);
        values.locals.insert(name.clone(), v);
    }
    let guard = values.emit(&branch.guard);
    let mut valid = vec![guard.valid.clone()];
    valid.extend(branch.checks.iter().map(|e| values.emit(e).valid));
    let updates: Vec<_> =
        model.fields.iter().map(|f| branch.updates.get(&f.name).map(|e| values.emit(e))).collect();
    valid.extend(updates.iter().flatten().map(|v| v.valid.clone()));
    let outcome = values.emit(&branch.outcome);
    valid.push(outcome.valid.clone());
    let facts = if model.fact_type == Type::Unit {
        None
    } else {
        let facts = values.emit(&branch.facts);
        valid.push(facts.valid.clone());
        Some(facts)
    };
    let ok = values.condition(conjoin(valid));
    let changed = disjoin(
        model
            .fields
            .iter()
            .zip(&updates)
            .filter_map(|(f, v)| v.as_ref().map(|v| format!("{} # {}", v.value, f.name)))
            .chain(
                (model.outcome_type != Type::Unit)
                    .then(|| format!("{} # {}", outcome.value, text("Outcome::Accepted"))),
            )
            .collect(),
    );
    let enabled = if branch.preserve_self_loop { "TRUE".into() } else { changed };
    let mut terms = vec![
        format!("~{ERROR} /\\ ~__umpire_invalid_step"),
        format!("~({}) \\/ ({})", guard.valid, guard.value),
        format!("~({ok}) \\/ ({enabled})"),
    ];
    for (f, v) in model.fields.iter().zip(updates) {
        terms.push(format!(
            "{}' = IF {ok} THEN {} ELSE {}",
            f.name,
            v.map_or(f.name.clone(), |v| v.value),
            f.name
        ));
        terms.push(format!("__umpire_before_{}' = {}", f.name, f.name));
    }
    terms.extend([
        format!("{HAS_STEP}' = TRUE"),
        format!("{STUTTER}' = FALSE"),
        format!("__umpire_action' = {}", text(action)),
        format!("__umpire_input' = {ordinal}"),
        "__umpire_choice' = 0".into(),
        format!(
            "__umpire_outcome' = IF {ok} THEN {} ELSE {}",
            outcome.value,
            default(&model.outcome_type)
        ),
        facts.map_or_else(
            || "__umpire_facts' = <<>>".into(),
            |facts| format!("__umpire_facts' = IF {ok} THEN {} ELSE <<>>", facts.value),
        ),
        format!("{ERROR}' = (~({ok}) \\/ ~({claims}))"),
    ]);
    let unchanged = conjoin(model.fields.iter().map(|f| format!("{}' = {}", f.name, f.name)));
    terms.push(if model.outcome_type == Type::Unit {
        "__umpire_invalid_step' = FALSE".into()
    } else {
        format!(
            "__umpire_invalid_step' = ({ok} /\\ __umpire_outcome' # {} /\\ ~({unchanged}))",
            text("Outcome::Accepted")
        )
    });
    values.scope(conjoin(terms))
}

fn validate(model: &Lifted) -> Result<(), Diagnostic> {
    let mut generated = vec![
        ERROR.to_owned(),
        HAS_STEP.to_owned(),
        STUTTER.to_owned(),
        "__umpire_invalid_step".into(),
        "ArithmeticSafe".into(),
        "ValidStep".into(),
        "Stutter".into(),
        "__umpire_SafeDiv".into(),
        "__umpire_abs_a".into(),
        "__umpire_abs_b".into(),
    ];
    generated.extend(model.fields.iter().map(|f| format!("__umpire_before_{}", f.name)));
    for f in &model.fields {
        if generated.contains(&f.name) {
            return Err(Diagnostic::of(
                &model.file,
                &f.location,
                format!("reserved safety name `{}`", f.name),
            ));
        }
    }
    for name in model
        .constants
        .keys()
        .map(|n| model.names.constants.get(n).unwrap_or(n).clone())
        .chain(model.actions.iter().map(|a| operator(&a.name)))
    {
        if generated.contains(&name) {
            return Err(Diagnostic::of(
                &model.file,
                &model.location,
                format!("declaration collides with generated safety name `{name}`"),
            ));
        }
    }
    Ok(())
}

fn initial_states<'a>(
    model: &'a Lifted,
    values: &mut Values<'a>,
) -> Result<(String, String), Diagnostic> {
    if let Some(starts) = &model.complete_starts {
        let states = if starts.is_empty() {
            conjoin(
                std::iter::once("FALSE".into())
                    .chain(model.fields.iter().map(|f| format!("{} = {}", f.name, default(&f.ty)))),
            )
        } else {
            disjoin(
                starts
                    .iter()
                    .map(|state| {
                        conjoin(
                            model
                                .fields
                                .iter()
                                .zip(state)
                                .map(|(f, e)| format!("{} = {}", f.name, values.emit(e).value)),
                        )
                    })
                    .collect(),
            )
        };
        return Ok((states, "TRUE".into()));
    }
    let mut terms = Vec::new();
    let mut valid = "TRUE".to_owned();
    for field in &model.fields {
        let (current_valid, domain) = match &field.start {
            Start::Computed => {
                return Err(Diagnostic::of(
                    &model.file,
                    &field.location,
                    "computed starts require a complete-start binding",
                ));
            }
            Start::Value(e) => {
                let v = values.emit(e);
                (v.valid, format!("{} = {}", field.name, v.value))
            }
            Start::Range(lo, hi) | Start::ExclusiveRange(lo, hi) => {
                let lo = values.emit(lo);
                let hi = values.emit(hi);
                (
                    conjoin([lo.valid, hi.valid]),
                    format!(
                        "{} \\in ({}..({}{}))",
                        field.name,
                        lo.value,
                        hi.value,
                        if matches!(field.start, Start::ExclusiveRange(..)) { " - 1" } else { "" }
                    ),
                )
            }
        };
        valid = values.condition(conjoin([valid, current_valid]));
        terms.push(format!(
            "IF {valid} THEN ({domain}) ELSE {} = {}",
            field.name,
            default(&field.ty)
        ));
    }
    Ok((conjoin(terms), valid))
}
