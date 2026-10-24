//! One selected World shared by the native checker and, as values only, the lifter.

use std::path::Path;

use umpire_checker::Checker;
use umpire_dsl::{
    __lifted::Lifts, ActionValues, Assumption, CATALOG_LIMIT, Claim, Machine, Value, Values,
};
use umpire_tla::{Binding, Concrete, Literal, Module};

/// A declared World whose derives export its own, its states' and its actions' values, and
/// whose `#[machine]` embeds the admitted Model.
pub trait Declared: Machine<State: Values, Action: ActionValues<Self>> + Values + Lifts {}

impl<M: Machine<State: Values, Action: ActionValues<M>> + Values + Lifts> Declared for M {}

/// One selected World, its starts enumerated once and the values of its checked catalog, with a
/// fixture's state budget and optional negative claim.
pub struct Bound<M: Machine> {
    world: M,
    starts: Vec<M::State>,
    binding: Result<Binding, String>,
    budget: usize,
    negative: Option<Claim<M>>,
}

/// Selects `world` for native checking, source binding and TLC replay alike.
pub fn bound<M: Declared>(world: M, budget: usize, negative: Option<Claim<M>>) -> Bound<M> {
    let starts = world.starts();
    let binding = world
        .checked_catalog(CATALOG_LIMIT)
        .map(|catalog| Binding {
            world: fields(world.values()),
            starts: starts.iter().map(|state| fields(state.values())).collect(),
            actions: catalog.actions().iter().map(concrete::<M>).collect(),
        })
        .map_err(|error| error.to_string());
    Bound { world, starts, binding, budget, negative }
}

impl<M: Declared> Bound<M> {
    /// The Model this World's `#[machine]` embeds, read for `source`, bound to this World and
    /// emitted.
    ///
    /// # Errors
    /// Returns the catalog refusal or the lifter's diagnostic.
    pub fn lift(&self, source: impl AsRef<Path>) -> Result<Module, String> {
        let binding = self.binding.as_ref().map_err(Clone::clone)?;
        umpire_tla::Lift::embedded(Some(&M::__umpire_lifted()), source)
            .and_then(|lifted| lifted.bind(binding))
            .and_then(|lifted| umpire_tla::Tla::emit(&lifted))
            .map_err(|diagnostic| diagnostic.to_string())
    }

    /// Source text, such as a mutant of this World's Model, read as the `#[machine]` macro
    /// would, bound to this World and emitted.
    ///
    /// # Errors
    /// Returns the catalog refusal or the lifter's diagnostic.
    pub fn lift_source(&self, file: &str, source: &str) -> Result<Module, String> {
        let binding = self.binding.as_ref().map_err(Clone::clone)?;
        umpire_tla::Lift::parse(file, source)
            .and_then(|lifted| lifted.bind(binding))
            .and_then(|lifted| umpire_tla::Tla::emit(&lifted))
            .map_err(|diagnostic| diagnostic.to_string())
    }

    /// The native checker's files for this World.
    ///
    /// # Errors
    /// Returns a checker refusal or a rendering failure.
    pub fn run(mut self) -> crate::RunResult {
        self.native().map(|(_, files)| files)
    }

    /// Checks the folder's `model.rs`, lifted for this World, with TLC against the native
    /// exploration of the same World and starts.
    ///
    /// # Errors
    /// Returns any lift, checker or backend failure, or a disagreement.
    pub fn check(self, folder: &Path) -> crate::RunResult {
        let module = self.lift(folder.join("model.rs"))?;
        self.check_module(folder, &module)
    }

    /// Checks an emitted module, such as a mutated one, with TLC against the native
    /// exploration of this World and starts.
    ///
    /// # Errors
    /// Returns any checker or backend failure, or a disagreement.
    pub fn check_module(mut self, folder: &Path, module: &Module) -> crate::RunResult {
        let expected = self.binding.as_ref().map_err(Clone::clone)?;
        let mut starts = Vec::new();
        for state in &expected.starts {
            if !starts.contains(state) {
                starts.push(state.clone());
            }
        }
        let inputs =
            M::__umpire_lifted().model.as_ref().is_ok_and(umpire_dsl::__lifted::Lifted::has_inputs)
                || expected.actions.iter().any(|action| !action.fields.is_empty());
        if module.metadata.catalog != if inputs { expected.actions.clone() } else { Vec::new() }
            || module.metadata.binding.as_ref().is_none_or(|actual| {
                actual.world != expected.world
                    || actual.starts != starts
                    || actual.actions != expected.actions
            })
        {
            return Err("emitted module differs from the selected World binding".into());
        }
        let (graph, files) = self.native()?;
        self.check_progress_metadata(&graph, module, inputs)?;
        crate::backend::check_exploration(
            &self.world,
            &self.starts,
            &graph,
            module,
            folder,
            &files,
            Some(&concrete::<M>),
        )
    }

    fn check_progress_metadata(
        &self,
        graph: &umpire_checker::Result<M>,
        module: &Module,
        inputs: bool,
    ) -> Result<(), String> {
        let claims = self.world.progress();
        if claims.len() != module.metadata.progress.len() {
            return Err("emitted progress registrations differ from the compiled Model".into());
        }
        let catalog =
            self.world.checked_catalog(CATALOG_LIMIT).map_err(|error| error.to_string())?;
        for (claim, metadata) in claims.iter().zip(&module.metadata.progress) {
            if claim.name != metadata.name {
                return Err("emitted progress registration differs from the compiled Model".into());
            }
            if let Some(evaluate) = claim.evaluator {
                for (_, state) in graph.states() {
                    let evaluated = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                        evaluate(&self.world, state)
                    }))
                    .map_err(|_| format!("progress `{}` metadata callback panicked", claim.name))?;
                    scope_metadata::<M>(
                        claim.assumption_override.as_ref().unwrap_or(&evaluated.assumption),
                        claim.bound_override.or(evaluated.bound),
                        catalog.actions(),
                        inputs,
                        metadata,
                    )?;
                }
            } else {
                scope_metadata::<M>(
                    claim.assumption_override.as_ref().unwrap_or(&claim.scope),
                    claim.bound_override.or(claim.bound),
                    catalog.actions(),
                    inputs,
                    metadata,
                )?;
            }
        }
        Ok(())
    }

    fn native(
        &mut self,
    ) -> Result<(umpire_checker::Result<M>, crate::Files), Box<dyn std::error::Error>> {
        let graph = Checker::explore_within(
            &self.world,
            self.starts.clone(),
            self.budget,
            Checker::CATALOG_LIMIT,
        )?;
        let mut files = crate::render(&graph)?;
        files.insert("rows.txt".into(), crate::render_rows(&graph)?);
        if let Some(claim) = self.negative.take() {
            let report =
                Checker::refuted_from(&self.world, self.starts.clone(), claim)?.verify_all()?;
            files.insert("negative_report.txt".into(), report.to_string());
            crate::add_paths(&mut files, report);
        }
        Ok((graph, files))
    }
}

fn concrete<M: Declared>(action: &M::Action) -> Concrete {
    let (action, values) = ActionValues::<M>::action_value(action);
    Concrete { action: action.into(), fields: fields(values) }
}

fn scope_metadata<M: Declared>(
    scope: &Assumption<M::Action>,
    bound: Option<u32>,
    actions: &[M::Action],
    inputs: bool,
    metadata: &umpire_tla::Progress,
) -> Result<(), String> {
    let mut allowed = Vec::new();
    let mut fair = Vec::new();
    let mut allowed_inputs = Vec::new();
    let mut fair_inputs = Vec::new();
    for (ordinal, action) in actions.iter().enumerate() {
        if !scope.allows(action) {
            continue;
        }
        let name = ActionValues::<M>::action_value(action).0.to_owned();
        if !allowed.contains(&name) {
            allowed.push(name);
        }
        if inputs && !actions[..ordinal].contains(action) {
            allowed_inputs.push(ordinal);
            if scope.is_fair(action) {
                fair_inputs.push(ordinal);
            }
        }
    }
    for action in &scope.fair {
        let name = ActionValues::<M>::action_value(action).0.to_owned();
        if !fair.contains(&name) {
            fair.push(name);
        }
    }
    if metadata.allowed != allowed
        || metadata.fair != fair
        || metadata.allowed_inputs != allowed_inputs
        || metadata.fair_inputs != fair_inputs
        || metadata.bound != bound.map(u64::from)
    {
        return Err(format!(
            "emitted progress `{}` metadata differs from the compiled scope",
            metadata.name
        ));
    }
    Ok(())
}

fn fields(values: Vec<(&'static str, Value)>) -> Vec<(String, Literal)> {
    values.into_iter().map(|(name, value)| (name.into(), literal(value))).collect()
}

fn literal(value: Value) -> Literal {
    match value {
        Value::Bool(value) => Literal::Bool(value),
        Value::Int(value) => Literal::Int(value),
        Value::Enum(value) => Literal::Enum(value.into()),
        Value::Bounded { name, raw } => Literal::Bounded { name: name.into(), raw },
        Value::Optional(value) => Literal::Optional(value.map(|value| Box::new(literal(*value)))),
        Value::Sequence(values) => Literal::Sequence(values.into_iter().map(literal).collect()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nominal_exports_preserve_identity_and_optional_structure() {
        for (value, expected) in [
            (
                Value::Bounded { name: "Ticks", raw: 1 },
                Literal::Bounded { name: "Ticks".into(), raw: 1 },
            ),
            (Value::Optional(None), Literal::Optional(None)),
            (
                Value::Optional(Some(Box::new(Value::Bounded { name: "Remaining", raw: 0 }))),
                Literal::Optional(Some(Box::new(Literal::Bounded {
                    name: "Remaining".into(),
                    raw: 0,
                }))),
            ),
        ] {
            assert_eq!(literal(value), expected);
        }
    }
}
