//! One selected World shared by the native checker and, as values only, the lifter.

use std::path::Path;

use umpire_checker::Checker;
use umpire_dsl::{__lifted::Lifts, ActionValues, CATALOG_LIMIT, Claim, Machine, Value, Values};
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
        let (graph, files) = self.native()?;
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

fn fields(values: Vec<(&'static str, Value)>) -> Vec<(String, Literal)> {
    values.into_iter().map(|(name, value)| (name.into(), literal(value))).collect()
}

fn literal(value: Value) -> Literal {
    match value {
        Value::Bool(value) => Literal::Bool(value),
        Value::Int(value) => Literal::Int(value),
        Value::Enum(value) => Literal::Enum(value.into()),
        Value::Sequence(values) => Literal::Sequence(values.into_iter().map(literal).collect()),
    }
}
