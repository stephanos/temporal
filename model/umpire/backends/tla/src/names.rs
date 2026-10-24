use crate::{Diagnostic, Lifted};
use std::collections::{BTreeMap, BTreeSet};
use umpire_front::{
    Source,
    expr::{Expr, ExprKind},
};

/// TLA+ keywords, standard-module operators and fixed emitted names that no generated or
/// authored symbol may take.
const RESERVED_NAMES: &[&str] = &[
    "vars",
    "ModelVars",
    "Init",
    "ModelNext",
    "ModelEdge",
    "Edge",
    "Complete",
    "Next",
    "Spec",
    "__umpire_Div",
    "__umpire_a",
    "__umpire_b",
    "__umpire_d",
    "__umpire_n",
    "__umpire_q",
    "__umpire_r",
    "__umpire_negative",
    "__umpire_edge_tick",
    "__umpire_pending",
    "__umpire_steps",
    "__umpire_action",
    "__umpire_choice",
    "__umpire_outcome",
    "__umpire_facts",
    "__umpire_input",
    "TRUE",
    "FALSE",
    "BOOLEAN",
    "Int",
    "Nat",
    "Assert",
    "Seq",
    "Len",
    "Append",
    "Head",
    "Tail",
    "SubSeq",
    "SelectSeq",
    "DOMAIN",
    "IF",
    "THEN",
    "ELSE",
    "CASE",
    "OTHER",
    "LET",
    "IN",
    "UNCHANGED",
    "ENABLED",
    "SUBSET",
    "UNION",
    "CHOOSE",
    "ASSUME",
    "THEOREM",
    "VARIABLE",
    "VARIABLES",
    "CONSTANT",
    "CONSTANTS",
    "MODULE",
    "INSTANCE",
    "LOCAL",
    "EXTENDS",
];

/// The TLA+ names emission gives a Model's World fields, action inputs, catalog position and
/// module constants.
#[derive(Clone, Debug)]
pub(crate) struct Names {
    /// One per World field.
    pub(crate) world: Vec<String>,
    /// One per input field of each action; chosen values are LET-bound inside one variant's
    /// relation, so each variant allocates its names apart from the others.
    pub(crate) inputs: Vec<Vec<String>>,
    /// A catalog position inside input relations.
    pub(crate) index: String,
    /// The prefix of a step operator's numbered shared definitions.
    pub(crate) shared: String,
    pub(crate) safety_claims: String,
    pub(crate) safety_posts: String,
    /// Module constants under a declared World.
    pub(crate) constants: BTreeMap<String, String>,
}

/// An action's TLA+ relation operator.
pub(crate) fn operator(action: &str) -> String {
    action.replace("::", "_")
}

/// Allocate the generated names, never like an authored or reserved name, and rename the
/// World and input references of the admitted tree to them.
pub(crate) fn assign(source: &Source, model: &mut umpire_front::Lifted) -> Names {
    let taken = &occupied(source);
    let mut generated = BTreeSet::new();
    let world: Vec<_> = model
        .world_fields
        .iter()
        .map(|field| allocate(taken, &format!("__umpire_world_{}", field.name), &mut generated))
        .collect();
    let inputs: Vec<Vec<_>> = model
        .actions
        .iter()
        .map(|action| {
            let mut scope = generated.clone();
            action
                .fields
                .iter()
                .map(|field| allocate(taken, &format!("__umpire_input_{}", field.name), &mut scope))
                .collect()
        })
        .collect();
    let index = allocate(taken, "__umpire_i", &mut generated);
    let shared = numbered(taken, "__umpire_s", &mut generated);
    let safety_claims = numbered(taken, "__umpire_c", &mut generated);
    let safety_posts = numbered(taken, "__umpire_p", &mut generated);
    let local = numbered(taken, "__umpire_l", &mut generated);
    let constants = if model.declared_world {
        model
            .constants
            .keys()
            .map(|name| {
                (name.clone(), allocate(taken, &format!("__umpire_source_{name}"), &mut generated))
            })
            .collect()
    } else {
        BTreeMap::new()
    };
    let worlds: BTreeMap<_, _> =
        model.world_fields.iter().map(|field| field.name.clone()).zip(world.clone()).collect();
    let rename_world = |expr: &mut Expr| {
        rename(expr, &|kind| match kind {
            ExprKind::World(name) => worlds.get(name.as_str()).cloned().map(ExprKind::World),
            _ => None,
        });
    };
    for value in model.constants.values_mut() {
        rename_world(value);
    }
    for field in &mut model.fields {
        match &mut field.start {
            umpire_front::tree::Start::Computed => {}
            umpire_front::tree::Start::Value(value) => rename_world(value),
            umpire_front::tree::Start::Range(lo, hi)
            | umpire_front::tree::Start::ExclusiveRange(lo, hi) => {
                rename_world(lo);
                rename_world(hi);
            }
        }
    }
    for (action, symbols) in model.actions.iter_mut().zip(&inputs) {
        let chosen: BTreeMap<_, _> =
            action.fields.iter().map(|field| field.name.clone()).zip(symbols.clone()).collect();
        let mut next_local = 0;
        for branch in &mut action.branches {
            let locals = local_names(branch, &local, &mut next_local);
            let rename_branch = |expr: &mut Expr| {
                rename(expr, &|kind| match kind {
                    ExprKind::World(name) => {
                        worlds.get(name.as_str()).cloned().map(ExprKind::World)
                    }
                    ExprKind::Input(name) => {
                        chosen.get(name.as_str()).cloned().map(ExprKind::Input)
                    }
                    ExprKind::Local(name) => locals.get(name).cloned().map(ExprKind::Local),
                    _ => None,
                });
            };
            for (_, value) in &mut branch.definitions {
                rename_branch(value);
            }
            rename_branch(&mut branch.guard);
            branch.updates.values_mut().for_each(rename_branch);
            branch.checks.iter_mut().for_each(rename_branch);
            rename_branch(&mut branch.outcome);
            rename_branch(&mut branch.facts);
        }
    }
    for claim in model.invariants.iter_mut().chain(&mut model.negative) {
        rename_world(&mut claim.expr);
    }
    for claim in &mut model.postconditions {
        rename_world(&mut claim.expr);
    }
    for claim in &mut model.progress {
        rename_world(&mut claim.trigger);
        rename_world(&mut claim.target);
    }
    Names { world, inputs, index, shared, safety_claims, safety_posts, constants }
}

fn local_names(
    branch: &mut umpire_front::tree::Branch,
    prefix: &str,
    next: &mut usize,
) -> BTreeMap<String, String> {
    branch
        .definitions
        .iter_mut()
        .map(|(name, _)| {
            *next += 1;
            let generated = format!("{prefix}{next}");
            (std::mem::replace(name, generated.clone()), generated)
        })
        .collect()
}

fn rename(expr: &mut Expr, renamed: &dyn Fn(&ExprKind) -> Option<ExprKind>) {
    if let Some(kind) = renamed(&expr.kind) {
        expr.kind = kind;
        return;
    }
    match &mut expr.kind {
        ExprKind::Unary(_, value) | ExprKind::Length(value) | ExprKind::Check(_, value) => {
            rename(value, renamed);
        }
        ExprKind::Binary(_, left, right) | ExprKind::Append(left, right) => {
            rename(left, renamed);
            rename(right, renamed);
        }
        ExprKind::If(first, second, third) | ExprKind::GetOr(first, second, third) => {
            rename(first, renamed);
            rename(second, renamed);
            rename(third, renamed);
        }
        ExprKind::Tuple(values) | ExprKind::Sequence(values) | ExprKind::Checks(values) => {
            for value in values {
                rename(value, renamed);
            }
        }
        ExprKind::Bool(_)
        | ExprKind::Int(_)
        | ExprKind::Text(_)
        | ExprKind::Unit
        | ExprKind::Var(..)
        | ExprKind::World(_)
        | ExprKind::Input(_)
        | ExprKind::Outcome
        | ExprKind::Local(_)
        | ExprKind::Enum(_) => {}
    }
}

/// The reserved names and every name the source spells or a back end derives from it.
fn occupied(source: &Source) -> BTreeSet<String> {
    let mut occupied: BTreeSet<_> = RESERVED_NAMES.iter().map(|name| (*name).to_owned()).collect();
    occupied.extend(source.idents.iter().cloned());
    occupied.extend(source.variants.iter().map(|(name, variant)| format!("{name}_{variant}")));
    for method in &source.methods {
        for prefix in [
            "Invariant",
            "Post",
            "Negative",
            "Assumption",
            "Scoped",
            "Activate",
            "Monitor",
            "ActiveEdge",
            "PrefixEdge",
            "Complete",
            "ProgressNext",
            "ProgressSpec",
            "Maximal",
            "Progress",
        ] {
            occupied.insert(format!("{prefix}_{method}"));
        }
    }
    occupied
}

fn allocate(occupied: &BTreeSet<String>, base: &str, generated: &mut BTreeSet<String>) -> String {
    let mut name = base.to_owned();
    let mut suffix = 0;
    while occupied.contains(&name) || generated.contains(&name) {
        suffix += 1;
        name = format!("{base}_{suffix}");
    }
    generated.insert(name.clone());
    name
}

pub(crate) fn validate(file: &str, source: &Source, model: &Lifted) -> Result<(), Diagnostic> {
    for field in &model.fields {
        if crate::emit::OBSERVERS.contains(&field.name.as_str()) || field.name == crate::emit::INPUT
        {
            return Err(Diagnostic::of(
                file,
                &field.location,
                format!("reserved observer name `{}`", field.name),
            ));
        }
    }
    if let Some(control) = &source.control {
        return Err(Diagnostic::of(file, control, control.message.clone()));
    }
    let mut names = Registry {
        file,
        source,
        names: RESERVED_NAMES.iter().map(|name| (*name).into()).collect(),
        prefixes: Vec::new(),
    };
    // Input symbols are scoped to one variant's relation and may repeat across variants.
    let mut symbols = BTreeSet::new();
    for (action, inputs) in model.actions.iter().zip(&model.names.inputs) {
        let variant = action.name.rsplit("::").next().unwrap_or(&action.name);
        let operator = operator(&action.name);
        names.register(&operator, variant)?;
        if !action.fields.is_empty() {
            for suffix in ["Inputs", "At"] {
                names.register(&format!("{operator}_{suffix}"), variant)?;
            }
            for (field, symbol) in action.fields.iter().zip(inputs) {
                if !symbols.contains(symbol) {
                    names.register(symbol, &field.name)?;
                    symbols.insert(symbol.clone());
                }
            }
        }
    }
    if model.has_inputs() {
        names.register(&model.names.index, "__umpire_i")?;
    }
    for claim in &model.invariants {
        names.register(&format!("Invariant_{}", claim.name), &claim.name)?;
    }
    for claim in &model.postconditions {
        names.register(&format!("Post_{}", claim.name), &claim.name)?;
    }
    names.register_negative(model)?;
    for assumption in &model.assumptions {
        if model.progress.iter().any(|claim| claim.assumption.same_scope(assumption)) {
            names.register(&format!("Assumption_{}", assumption.name), &assumption.name)?;
        }
    }
    for claim in &model.progress {
        for prefix in [
            "Scoped",
            "Activate",
            "Monitor",
            "ActiveEdge",
            "PrefixEdge",
            "Complete",
            "ProgressNext",
            "ProgressSpec",
            "Maximal",
            "Progress",
        ] {
            names.register(&format!("{prefix}_{}", claim.name), &claim.name)?;
        }
        for action in &model.actions {
            if !claim.assumption.fair.iter().any(|fair| fair.name == action.name) {
                continue;
            }
            let fair = format!("Fair_{}_{}", claim.name, operator(&action.name));
            if action.fields.is_empty() {
                names.register(&fair, &claim.name)?;
            } else {
                // Exact payload obligations append a catalog position once the World is bound.
                names.register_prefix(&format!("{fair}_"), &claim.name)?;
            }
        }
    }
    for field in &model.fields {
        names.register(&field.name, &field.name)?;
    }
    for name in model.constants.keys() {
        names.register(model.names.constants.get(name).unwrap_or(name), name)?;
    }
    for (field, symbol) in model.world_fields.iter().zip(&model.names.world) {
        names.register(symbol, &field.name)?;
    }
    Ok(())
}

/// The generated names taken so far, and the prefixes that catalog positions extend.
struct Registry<'a> {
    file: &'a str,
    source: &'a Source,
    names: BTreeSet<String>,
    prefixes: Vec<String>,
}
impl Registry<'_> {
    fn register(&mut self, name: &str, source: &str) -> Result<(), Diagnostic> {
        if self.fresh(name) {
            return Ok(());
        }
        Err(self.collision(source, &format!("generated TLA+ name `{name}`")))
    }

    /// The negative claims; a hook claim is written in its hook's file, not in the module.
    fn register_negative(&mut self, model: &Lifted) -> Result<(), Diagnostic> {
        let (file, locations) = &model.claims;
        for (index, claim) in model.negative.iter().enumerate() {
            let name = format!("Negative_{}", claim.name);
            match locations.get(index) {
                Some(location) if !self.fresh(&name) => {
                    return Err(Diagnostic::of(
                        file,
                        location,
                        format!("authored declaration collides with generated TLA+ name `{name}`"),
                    ));
                }
                Some(_) => {}
                None => self.register(&name, &claim.name)?,
            }
        }
        Ok(())
    }

    /// Take `name` if no generated name or prefix has it and it is a plain TLA+ identifier.
    fn fresh(&mut self, name: &str) -> bool {
        !self.prefixes.iter().any(|prefix| positioned(name, prefix))
            && self.names.insert(name.into())
            && !name.starts_with("r#")
            && name.chars().all(|c| c.is_ascii_alphanumeric() || c == '_')
    }

    /// A prefix of generated `<prefix><position>` names, which no authored identifier and no
    /// other generated name may share.
    fn register_prefix(&mut self, prefix: &str, source: &str) -> Result<(), Diagnostic> {
        if self.source.idents.iter().any(|ident| ident.starts_with(prefix))
            || self.names.iter().any(|name| positioned(name, prefix))
        {
            return Err(self.collision(source, &format!("generated TLA+ names `{prefix}*`")));
        }
        self.prefixes.push(prefix.into());
        Ok(())
    }

    fn collision(&self, source: &str, generated: &str) -> Diagnostic {
        let (line, column) = self.source.first.get(source).copied().unwrap_or((1, 1));
        Diagnostic {
            file: self.file.into(),
            line,
            column,
            message: format!("authored declaration collides with {generated}"),
        }
    }
}

/// Whether `name` is `prefix` followed by a catalog position.
fn positioned(name: &str, prefix: &str) -> bool {
    name.strip_prefix(prefix).is_some_and(|position| {
        !position.is_empty() && position.bytes().all(|b| b.is_ascii_digit())
    })
}

/// A prefix that no authored, reserved or generated name extends with digits alone.
fn numbered(occupied: &BTreeSet<String>, base: &str, generated: &mut BTreeSet<String>) -> String {
    let extends = |prefix: &str, name: &String| {
        name.strip_prefix(prefix)
            .is_some_and(|rest| !rest.is_empty() && rest.bytes().all(|byte| byte.is_ascii_digit()))
    };
    let mut prefix = base.to_owned();
    let mut suffix = 0;
    while occupied.iter().chain(generated.iter()).any(|name| extends(&prefix, name)) {
        suffix += 1;
        prefix = format!("{base}_{suffix}_");
    }
    generated.insert(prefix.clone());
    prefix
}
