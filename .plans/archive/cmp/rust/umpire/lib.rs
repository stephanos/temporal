//! # umpire: the model layer surface the Rust samples author against
//!
//! Crate `umpire`. Types and signatures for Step, Machine, finite state enumeration, the refinement
//! check, Property, Scenario, Limits, Query, Set and Composition. The DSL entry points (`entity!`,
//! `action!`, `observation!`, `machine!`, `compose!`, `property!`, `scenario!`, `limits!`, `query!`,
//! `set!`, `#[derive(Finite)]`) are proc-macros and by Rust's rules live in their own crate,
//! `umpire-macros` (`macros.rs` beside this file); they are re-exported here so a Model imports one
//! crate.
//!
//! Bodies are sketched. Signatures are the surface a Model binds to, so they are real.
//!
//! ## Where the checks run
//!
//! - Exhaustive `match` in a step function, the type of every step function against its action's
//!   declared inputs, and every name a `machine!`/`scenario!`/`property!` block references: rustc,
//!   at compile time, pinned to the token.
//! - The shape of a block (a timer with no step, an `unobservable:` that is not a timer, a
//!   `set!` whose purpose and fields disagree): the proc-macro, at compile time, pinned to the
//!   token via `syn::Error::new_spanned`.
//! - `Finite::CARDINALITY`: an associated `const`, so a state count can be pinned with
//!   `const _: () = assert!(..)`.
//! - The state table, reachability, the refinement rows, search and verification: `cargo test`.
//!   Rust `const fn` cannot allocate a `Vec`, and a `Machine` boxes its step functions, so a table
//!   is built once at first use (`LazyLock`) and pinned by `#[test]` functions.

#![forbid(unsafe_code)]

use std::any::{Any, TypeId};
use std::collections::{BTreeMap, BTreeSet};
use std::fmt::Debug;
use std::hash::Hash;
use std::sync::{LazyLock, OnceLock};

pub use umpire_macros::{
    action, compose, entity, limits, machine, observation, property, query, scenario, set, Finite,
};

// ---------------------------------------------------------------------------------------------
// Finite domains
// ---------------------------------------------------------------------------------------------

/// A type with finitely many values, listed in a canonical order. Every state type, input domain,
/// outcome and fact is `Finite`; `#[derive(Finite)]` writes the impl for enums (payload variants
/// contribute one member per assignment of their `Finite` fields, in field order) and for structs
/// (the product of their fields, first field slowest).
///
/// `CARDINALITY` is an associated `const`, which is what lets a state count be pinned at compile
/// time. `all()` is what the table enumerator walks.
pub trait Finite: Sized + Clone + Eq + Hash + Debug + Send + Sync + 'static {
    const CARDINALITY: usize;

    /// Every member, in canonical order. `all().len() == CARDINALITY`.
    fn all() -> Vec<Self>;

    /// The member a state field begins at: the first declared variant, or zero. `machine!` uses it
    /// for the fields a `starts:` line does not name (`S { phase, ..S::first() }`).
    fn first() -> Self {
        Self::all().swap_remove(0)
    }

    /// The row-key segment: the variant name in lowerCamel, payload fields joined with `-`
    /// (`handlerError-true`), struct fields joined with `-` (`scheduled-0-unset-unset-unset`).
    fn key(&self) -> String;
}

impl Finite for bool {
    const CARDINALITY: usize = 2;
    fn all() -> Vec<Self> {
        vec![false, true]
    }
    fn key(&self) -> String {
        self.to_string()
    }
}

impl Finite for () {
    const CARDINALITY: usize = 1;
    fn all() -> Vec<Self> {
        vec![()]
    }
    fn key(&self) -> String {
        String::new()
    }
}

/// Tuples of `Finite` are `Finite`: the input of an action with several fields.
macro_rules! finite_tuple {
    ($($t:ident),+) => {
        impl<$($t: Finite),+> Finite for ($($t,)+) {
            const CARDINALITY: usize = 1 $(* $t::CARDINALITY)+;
            fn all() -> Vec<Self> {
                // cartesian product, first field slowest; elided
                todo!("cartesian product of {}", stringify!($($t),+))
            }
            fn key(&self) -> String {
                todo!()
            }
        }
    };
}
finite_tuple!(A);
finite_tuple!(A, B);
finite_tuple!(A, B, C);
finite_tuple!(A, B, C, D);

/// The Lean `Fin (n + 1)`: an integer in `0..=MAX` with a saturating successor. `attempts` in both
/// protocol machines is a `Bounded<ATTEMPT_BOUND>`.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct Bounded<const MAX: u8>(u8);

impl<const MAX: u8> Bounded<MAX> {
    pub const ZERO: Self = Self(0);
    pub const LAST: Self = Self(MAX);

    pub const fn new(n: u8) -> Option<Self> {
        if n <= MAX { Some(Self(n)) } else { None }
    }

    /// The successor that stays inside the bound: `LAST.saturating_succ() == LAST`.
    pub const fn saturating_succ(self) -> Self {
        if self.0 >= MAX { self } else { Self(self.0 + 1) }
    }

    pub const fn get(self) -> u8 {
        self.0
    }
}

impl<const MAX: u8> Finite for Bounded<MAX> {
    const CARDINALITY: usize = MAX as usize + 1;
    fn all() -> Vec<Self> {
        (0..=MAX).map(Self).collect()
    }
    fn key(&self) -> String {
        self.0.to_string()
    }
}

// ---------------------------------------------------------------------------------------------
// Vocabulary: parties, entities, actions, observations
// ---------------------------------------------------------------------------------------------

/// Who performs an action. `System` is reserved for the timers a machine owns.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub enum Party {
    Caller,
    Handler,
    Worker,
    Network,
    Operator,
    System,
}

/// What a machine is about. `entity!` expands to a `static` of this type.
#[derive(Debug)]
pub struct Entity {
    pub name: &'static str,
    /// Which recorded field names an instance.
    pub key: Option<&'static str>,
    /// Links to other entities, by role.
    pub refer: &'static [(&'static str, &'static Entity)],
}

/// How an action relates to an entity.
#[derive(Clone, Copy, Debug)]
pub enum Binding {
    Creates(&'static Entity),
    On(&'static Entity),
}

/// A declared action. `action!` expands to a unit struct named exactly as the action (Diesel's
/// `table!` names its column types the same way, so `#[allow(non_camel_case_types)]` is the
/// precedent) and an impl of this trait, plus the inherent `class(..)`, `any()` and `bind(..)`
/// functions whose arities follow the declared `input:` fields. A timer named under a machine's
/// `timers:` gets the same expansion with `PARTY = Party::System` and `Input = ()`.
pub trait Action: 'static {
    const NAME: &'static str;
    const PARTY: Party;
    /// The finite input, as a tuple of the declared fields; `()` for none. One member is one class.
    type Input: Finite;
    fn binding() -> Option<Binding>;
    /// The protobuf message name, when the action carries one.
    const SCHEMA: Option<&'static str>;
    /// Concrete realization values for input classes.
    fn examples() -> Vec<(Self::Input, &'static str)> {
        Vec::new()
    }
}

/// One action with one assignment of its inputs: the unit of a Scenario line, a `when:` clause and
/// a transition row. Keyed like the Lean catalog: `handlerReply-handlerError-true`.
#[derive(Clone, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct ActionClass {
    pub action: &'static str,
    pub party: Party,
    pub inputs: Vec<String>,
}

impl ActionClass {
    pub fn key(&self) -> String {
        std::iter::once(self.action.to_string()).chain(self.inputs.iter().cloned()).collect::<Vec<_>>().join("-")
    }
}

/// What a `when:` clause names: one class, or every class of an action.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub enum ActionPattern {
    Class(ActionClass),
    Any(&'static str),
}

impl ActionPattern {
    pub fn matches(&self, class: &ActionClass) -> bool {
        match self {
            ActionPattern::Class(c) => c == class,
            ActionPattern::Any(name) => *name == class.action,
        }
    }
}

impl From<ActionClass> for ActionPattern {
    fn from(class: ActionClass) -> Self {
        ActionPattern::Class(class)
    }
}

/// A derived read used as evidence with no history event. `observation!` expands to a `static`.
#[derive(Debug)]
pub struct Observation {
    pub name: &'static str,
    pub on: &'static Entity,
    pub read: &'static str,
}

// ---------------------------------------------------------------------------------------------
// Steps and machines
// ---------------------------------------------------------------------------------------------

/// `{ outcome, state, facts }`: what one enabled action does. A step function returns an empty
/// `Vec` when its action is not enabled, and more than one `Step` when the action is
/// nondeterministic.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct Step<S, O, F> {
    pub outcome: O,
    pub state: S,
    pub facts: Vec<F>,
}

/// A step function with its action class already applied: what `Action::bind` produces for each
/// member of the action's input domain. The table enumerator runs it on every state.
pub struct BoundStep<S, O, F> {
    pub class: ActionClass,
    pub run: Box<dyn Fn(&S) -> Vec<Step<S, O, F>> + Send + Sync>,
}

/// One row of the finite state table.
#[derive(Clone, PartialEq, Eq, Debug)]
pub struct Row<S, O, F> {
    pub from: S,
    pub class: ActionClass,
    pub step: Step<S, O, F>,
}

impl<S: Finite, O: Finite, F: Finite> Row<S, O, F> {
    /// `scheduled-0-unset-unset-unset-handlerReply-async`
    pub fn key(&self) -> String {
        format!("{}-{}", self.from.key(), self.class.key())
    }
}

/// The finite state table: every state of `S` and every row a bound step produces from it.
pub struct Table<S, O, F> {
    pub states: Vec<S>,
    pub transitions: Vec<Row<S, O, F>>,
}

/// A fact-to-evidence line: which recorded event or observation confirms a fact.
pub struct Evidence<F> {
    pub fact: fn(&F) -> bool,
    pub recorded: &'static str,
}

/// A behavioral machine over state `S`, outcome `O` and facts `F`. `machine!` expands to a
/// `static NAME: LazyLock<Machine<S, O, F>>`; the table is computed on first use and cached.
pub struct Machine<S: Finite, O: Finite, F: Finite> {
    pub name: &'static str,
    pub entity: &'static Entity,
    pub starts: Vec<S>,
    pub ends: fn(&S) -> bool,
    pub timers: Vec<&'static str>,
    pub unobservable: Vec<&'static str>,
    pub evidence: Vec<Evidence<F>>,
    pub steps: Vec<BoundStep<S, O, F>>,
    pub refines: Option<Box<dyn Refines<S>>>,
    table: OnceLock<Table<S, O, F>>,
}

impl<S: Finite, O: Finite, F: Finite> Machine<S, O, F> {
    /// The finite state table, built once: `S::all()` crossed with every bound step.
    pub fn table(&self) -> &Table<S, O, F> {
        self.table.get_or_init(|| {
            let states = S::all();
            let transitions = states
                .iter()
                .flat_map(|from| {
                    self.steps.iter().flat_map(move |bound| {
                        (bound.run)(from).into_iter().map(move |step| Row {
                            from: from.clone(),
                            class: bound.class.clone(),
                            step,
                        })
                    })
                })
                .collect();
            Table { states, transitions }
        })
    }

    /// The states `ends` accepts, in canonical order.
    pub fn ends(&self) -> Vec<S> {
        S::all().into_iter().filter(|s| (self.ends)(s)).collect()
    }

    /// Every action class the machine steps on, in catalog (sorted key) order.
    pub fn action_keys(&self) -> BTreeSet<String> {
        self.steps.iter().map(|b| b.class.key()).collect()
    }

    /// The states a run from `starts` can reach.
    pub fn reachable(&self) -> Vec<S> {
        todo!("BFS over table().transitions from starts")
    }

    /// A reachable, non-end state with no enabled step, if any.
    pub fn stuck(&self) -> Option<S> {
        todo!()
    }

    /// The refinement rows, when `refines:` is declared. Each protocol row maps to the product
    /// step it is (`Some(key)`) or to a stutter (`None`).
    pub fn refinement(&self) -> Option<RefinementRows> {
        self.refines.as_ref().map(|r| r.check(self.rows_erased()))
    }

    fn rows_erased(&self) -> Vec<ErasedRow<S>> {
        todo!("project table().transitions to (from, class, outcome key, to)")
    }

    /// `machine! { name from: other restrict: [..] }`: the same machine with only the named
    /// actions. The composition's view of the worker is built this way.
    pub fn restrict(&'static self, actions: &[&'static str]) -> Machine<S, O, F> {
        todo!("copy with steps filtered to {actions:?}")
    }
}

// ---------------------------------------------------------------------------------------------
// Refinement
// ---------------------------------------------------------------------------------------------

/// A protocol row with its states kept and the rest reduced to keys: what the refinement check
/// reads without knowing the product's types.
pub struct ErasedRow<S> {
    pub from: S,
    pub to: S,
    pub class: ActionClass,
    pub outcome: String,
}

/// `refines: product map: f`. Object-safe so a `Machine<S, O, F>` can hold one without naming the
/// product's outcome and fact types.
pub trait Refines<S>: Send + Sync {
    fn target_name(&self) -> &'static str;
    fn target_state_type(&self) -> TypeId;
    /// Walk every protocol row through the map: a row whose mapped states are equal is a stutter;
    /// a row for which the product has some transition from `map s` to `map s'`, under any action
    /// class, is that transition (the first in table order when several fit, so a protocol timer
    /// row maps to the product's `timeout`); anything else is rejected, naming the row.
    fn check(&self, rows: Vec<ErasedRow<S>>) -> RefinementRows;
    /// The mapped state, erased, so a product Property can be read on a protocol step.
    fn map_erased(&self, state: &S) -> Box<dyn Any + Send + Sync>;
}

pub struct Refinement<S, P: Finite, PO: Finite, PF: Finite> {
    pub target: &'static LazyLock<Machine<P, PO, PF>>,
    pub map: fn(&S) -> P,
}

impl<S: Finite, P: Finite, PO: Finite, PF: Finite> Refines<S> for Refinement<S, P, PO, PF> {
    fn target_name(&self) -> &'static str {
        self.target.name
    }
    fn target_state_type(&self) -> TypeId {
        TypeId::of::<P>()
    }
    fn check(&self, rows: Vec<ErasedRow<S>>) -> RefinementRows {
        // Mapped states only: (from, to) -> the first product class in table order between them.
        let mut product: BTreeMap<(String, String), String> = BTreeMap::new();
        for r in &self.target.table().transitions {
            product.entry((r.from.key(), r.step.state.key())).or_insert_with(|| r.class.key());
        }
        let mut out = RefinementRows { rows: Vec::new(), rejected: None };
        for row in rows {
            let (before, after) = ((self.map)(&row.from), (self.map)(&row.to));
            let mapped = match product.get(&(before.key(), after.key())) {
                _ if before == after => None,
                Some(step) => Some(step.clone()),
                None => {
                    out.rejected.get_or_insert(RefinementError {
                        row: format!("{}-{}", row.from.key(), row.class.key()),
                        reason: format!(
                            "maps {} -> {} on {} and {} has no transition between them and it is not a stutter",
                            before.key(), after.key(), row.class.key(), self.target.name
                        ),
                    });
                    continue;
                }
            };
            out.rows.push((format!("{}-{}", row.from.key(), row.class.key()), mapped));
        }
        out
    }
    fn map_erased(&self, state: &S) -> Box<dyn Any + Send + Sync> {
        Box::new((self.map)(state))
    }
}

#[derive(Clone, PartialEq, Eq, Debug)]
pub struct RefinementError {
    pub row: String,
    pub reason: String,
}

/// `rows`: protocol row key -> `Some(product step key)` or `None` for a stutter.
#[derive(Clone, PartialEq, Eq, Debug)]
pub struct RefinementRows {
    pub rows: Vec<(String, Option<String>)>,
    pub rejected: Option<RefinementError>,
}

impl RefinementRows {
    pub fn lookup(&self, key: &str) -> Option<Option<&str>> {
        self.rows.iter().find(|(k, _)| k == key).map(|(_, v)| v.as_deref())
    }
}

// ---------------------------------------------------------------------------------------------
// Properties, Scenarios, Limits, Queries
// ---------------------------------------------------------------------------------------------

/// A same-step claim (`when:` + `holds: |step|`) or a transition claim (`holds: |before, after|`).
pub enum Claim<S, O, F> {
    SameStep { when: ActionPattern, holds: fn(&Step<S, O, F>) -> bool },
    Transition { holds: fn(&Step<S, O, F>, &Step<S, O, F>) -> bool },
}

pub struct Property<S: Finite, O: Finite, F: Finite> {
    pub name: &'static str,
    pub machine: &'static LazyLock<Machine<S, O, F>>,
    pub claim: Claim<S, O, F>,
}

/// A Property read on a machine: its own, or one that machine refines, through the `map:`. The
/// blanket impl decides at admission time: same types read directly; otherwise the scenario's
/// machine must declare `refines:` with a target of the Property's state type, or the Query is
/// rejected naming both machines. (Lean elaborates this; here it is a test-time admission error.)
pub trait PropertyOn<S: Finite, O: Finite, F: Finite>: Send + Sync {
    fn id(&self) -> &'static str;
    fn admit(&self, on: &Machine<S, O, F>) -> Result<(), Admission>;
    fn when(&self) -> Option<&ActionPattern>;
    fn holds_step(&self, on: &Machine<S, O, F>, step: &Step<S, O, F>) -> bool;
    fn holds_transition(&self, on: &Machine<S, O, F>, before: &Step<S, O, F>, after: &Step<S, O, F>) -> bool;
}

impl<P: Finite, PO: Finite, PF: Finite, S: Finite, O: Finite, F: Finite> PropertyOn<S, O, F>
    for Property<P, PO, PF>
{
    fn id(&self) -> &'static str {
        self.name
    }
    fn admit(&self, on: &Machine<S, O, F>) -> Result<(), Admission> {
        if TypeId::of::<P>() == TypeId::of::<S>() {
            return Ok(());
        }
        match &on.refines {
            Some(r) if r.target_state_type() == TypeId::of::<P>() => {
                // A same-step claim on the refined machine must name an action the refining one has.
                if let Claim::SameStep { when, .. } = &self.claim {
                    let name = match when { ActionPattern::Any(n) => *n, ActionPattern::Class(c) => c.action };
                    if !on.steps.iter().any(|b| b.class.action == name) {
                        return Err(Admission::UnknownAction {
                            property: self.name, action: name, machine: on.name,
                        });
                    }
                }
                Ok(())
            }
            _ => Err(Admission::NotReadable { property: self.name, machine: on.name }),
        }
    }
    fn when(&self) -> Option<&ActionPattern> {
        match &self.claim { Claim::SameStep { when, .. } => Some(when), Claim::Transition { .. } => None }
    }
    fn holds_step(&self, on: &Machine<S, O, F>, step: &Step<S, O, F>) -> bool {
        let _ = (on, step);
        todo!("downcast directly, or map state through on.refines and downcast to P")
    }
    fn holds_transition(&self, on: &Machine<S, O, F>, before: &Step<S, O, F>, after: &Step<S, O, F>) -> bool {
        let _ = (on, before, after);
        todo!()
    }
}

/// `model`, `starts` (a state by its phase), `actions` (classed, in order).
pub struct Scenario<S: Finite, O: Finite, F: Finite> {
    pub name: &'static str,
    pub model: &'static LazyLock<Machine<S, O, F>>,
    pub starts: S,
    pub actions: Vec<ActionClass>,
}

impl<S: Finite, O: Finite, F: Finite> Scenario<S, O, F> {
    /// The classed action keys in order: what the Lean pins read as `names.occurrences`.
    pub fn action_keys(&self) -> Vec<String> {
        self.actions.iter().map(ActionClass::key).collect()
    }
}

/// `steps`, `actions`, `search`: what bounds a search. `limits!` expands to a `static`, and the
/// struct is `const`-constructible so the expansion is a plain literal.
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub struct Limits {
    pub name: &'static str,
    pub steps: u32,
    pub actions: u32,
    pub search: u32,
}

pub enum QueryKind {
    /// Search for a path of the Scenario on which the same-step claim holds; realized by a set.
    Find,
    /// Check the claim on every trace of the Scenario within the limits; never realized.
    Verify,
}

pub struct Query<S: Finite, O: Finite, F: Finite> {
    pub name: &'static str,
    pub kind: QueryKind,
    pub property: &'static dyn PropertyOn<S, O, F>,
    pub scenario: &'static LazyLock<Scenario<S, O, F>>,
    pub limits: &'static Limits,
}

/// Why a Query is not admitted. In Lean this is an elaboration error at the `query` block; here it
/// is the `Err` of `Query::run`, so a `#[test]` pins it.
#[derive(Clone, PartialEq, Eq, Debug)]
pub enum Admission {
    /// The Property's machine is neither the Scenario's nor one it refines.
    NotReadable { property: &'static str, machine: &'static str },
    /// A product Property names an action the protocol machine has no step for.
    UnknownAction { property: &'static str, action: &'static str, machine: &'static str },
    /// `find:` on a transition claim, or `verify:` of a claim the Scenario never triggers.
    Shape(String),
}

/// What a search says.
#[derive(Clone, PartialEq, Eq, Debug)]
pub enum QueryOutcome {
    Found { witness: Vec<String> },
    NotFound,
    VerifiedWithinLimits { paths: usize },
    Violated { trace: Vec<String> },
    Exhausted,
}

impl QueryOutcome {
    /// The Lean `run.result.outcome.name`: `found`, `verified-within-limits`, ...
    pub fn name(&self) -> &'static str {
        match self {
            QueryOutcome::Found { .. } => "found",
            QueryOutcome::NotFound => "not-found",
            QueryOutcome::VerifiedWithinLimits { .. } => "verified-within-limits",
            QueryOutcome::Violated { .. } => "violated",
            QueryOutcome::Exhausted => "exhausted",
        }
    }
}

pub struct Checked {
    pub property: &'static str,
    pub outcome: QueryOutcome,
}

impl<S: Finite, O: Finite, F: Finite> Query<S, O, F> {
    /// Admit, then search the Scenario's traces within the limits. A `find` stops at the first
    /// path whose triggering step satisfies the claim; a `verify` checks every trace.
    pub fn run(&self) -> Result<Checked, Admission> {
        self.property.admit(&self.scenario.model)?;
        match (&self.kind, self.property.when()) {
            (QueryKind::Find, None) => {
                return Err(Admission::Shape(format!("{}: find of a transition claim", self.name)))
            }
            _ => {}
        }
        todo!("bounded search over scenario.model.table() following scenario.actions")
    }
}

// ---------------------------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------------------------

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Purpose {
    Functional,
    Canary,
    Exploratory,
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Bind {
    Driven,
    Observed,
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Repeat {
    /// Run once per value of the HSM/CHASM implementation switch.
    Implementation,
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Cover {
    Rows,
    Results,
    ClassMembers,
}

/// A Query of any machine, for a Set's list.
pub trait AnyQuery: Send + Sync {
    fn name(&self) -> &'static str;
    fn run_erased(&self) -> Result<Checked, Admission>;
}

impl<S: Finite, O: Finite, F: Finite> AnyQuery for Query<S, O, F> {
    fn name(&self) -> &'static str {
        self.name
    }
    fn run_erased(&self) -> Result<Checked, Admission> {
        self.run()
    }
}

/// A Machine of any type, for an exploratory Set's `machine:` line.
pub trait AnyMachine: Send + Sync {
    fn name(&self) -> &'static str;
    fn row_count(&self) -> usize;
}

impl<S: Finite, O: Finite, F: Finite> AnyMachine for Machine<S, O, F> {
    fn name(&self) -> &'static str {
        self.name
    }
    fn row_count(&self) -> usize {
        self.table().transitions.len()
    }
}

/// `purpose`, `bind`, optional `repeat`, and either `queries` or (exploratory) `machine`, `cover`,
/// `budget`. `set!` refuses at expansion a shape whose purpose and fields disagree.
pub struct Set {
    pub name: &'static str,
    pub purpose: Purpose,
    pub bind: Vec<(Party, Bind)>,
    pub repeat: Option<Repeat>,
    pub queries: Vec<&'static dyn AnyQuery>,
    pub machine: Option<&'static dyn AnyMachine>,
    pub cover: Vec<Cover>,
    pub budget: Option<&'static Limits>,
}

// ---------------------------------------------------------------------------------------------
// Composition
// ---------------------------------------------------------------------------------------------

/// A member of a composition, projected out of and back into the composite state.
pub struct Member<St, S: Finite, O: Finite, F: Finite> {
    pub role: &'static str,
    pub machine: &'static LazyLock<Machine<S, O, F>>,
    pub get: fn(&St) -> &S,
    pub set: fn(St, S) -> St,
}

/// `sync: name: a.x || b.y`: two member actions fire as one composite action named `name`.
#[derive(Clone, PartialEq, Eq, Debug)]
pub struct Sync {
    pub name: &'static str,
    pub left: (&'static str, &'static str),
    pub right: (&'static str, &'static str),
}

/// The outcome or fact of a composite step: which member produced it.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub enum Either<L, R> {
    Left(L),
    Right(R),
}

impl<L: Finite, R: Finite> Finite for Either<L, R> {
    const CARDINALITY: usize = L::CARDINALITY + R::CARDINALITY;
    fn all() -> Vec<Self> {
        L::all().into_iter().map(Either::Left).chain(R::all().into_iter().map(Either::Right)).collect()
    }
    fn key(&self) -> String {
        match self { Either::Left(l) => l.key(), Either::Right(r) => r.key() }
    }
}

/// The product of two machines of different entities. `compose!` expands to a
/// `LazyLock<Machine<St, Either<O1, O2>, Either<F1, F2>>>` built by [`Compose::machine`], so a
/// composition is a Machine to every Property, Scenario and Query.
pub struct Compose<St: Finite, S1: Finite, O1: Finite, F1: Finite, S2: Finite, O2: Finite, F2: Finite> {
    pub name: &'static str,
    pub entities: Vec<&'static Entity>,
    pub left: Member<St, S1, O1, F1>,
    pub right: Member<St, S2, O2, F2>,
    pub sync: Vec<Sync>,
    pub starts: Vec<St>,
    pub ends: fn(&St) -> bool,
}

impl<St: Finite, S1: Finite, O1: Finite, F1: Finite, S2: Finite, O2: Finite, F2: Finite>
    Compose<St, S1, O1, F1, S2, O2, F2>
{
    /// Every unsynchronized member action keeps its own row, prefixed by its role
    /// (`operation_backoff`); a synchronized pair has a row only where both members have one, under
    /// the sync's name.
    pub fn machine(self) -> Machine<St, Either<O1, O2>, Either<F1, F2>> {
        todo!("interleave left and right steps; join sync pairs; facts = left ++ right")
    }
}

// ---------------------------------------------------------------------------------------------
// Convenience for the macro expansions
// ---------------------------------------------------------------------------------------------

/// What `machine!` expands to: the builder the generated code calls, so the expansion is short and
/// the type errors land on the author's tokens rather than inside a struct literal.
pub struct MachineBuilder<S: Finite, O: Finite, F: Finite>(Machine<S, O, F>);

impl<S: Finite, O: Finite, F: Finite> MachineBuilder<S, O, F> {
    pub fn new(name: &'static str, entity: &'static Entity) -> Self {
        Self(Machine {
            name, entity, starts: Vec::new(), ends: |_| false, timers: Vec::new(),
            unobservable: Vec::new(), evidence: Vec::new(), steps: Vec::new(), refines: None,
            table: OnceLock::new(),
        })
    }
    pub fn starts(mut self, starts: Vec<S>) -> Self { self.0.starts = starts; self }
    pub fn ends(mut self, ends: fn(&S) -> bool) -> Self { self.0.ends = ends; self }
    pub fn timers(mut self, timers: Vec<&'static str>) -> Self { self.0.timers = timers; self }
    pub fn unobservable(mut self, u: Vec<&'static str>) -> Self { self.0.unobservable = u; self }
    pub fn evidence(mut self, fact: fn(&F) -> bool, recorded: &'static str) -> Self {
        self.0.evidence.push(Evidence { fact, recorded }); self
    }
    pub fn steps(mut self, bound: Vec<BoundStep<S, O, F>>) -> Self { self.0.steps.extend(bound); self }
    pub fn refines<P: Finite, PO: Finite, PF: Finite>(
        mut self, target: &'static LazyLock<Machine<P, PO, PF>>, map: fn(&S) -> P,
    ) -> Self {
        self.0.refines = Some(Box::new(Refinement { target, map })); self
    }
    pub fn build(self) -> Machine<S, O, F> { self.0 }
}

// ---------------------------------------------------------------------------------------------
// What the macros lean on
// ---------------------------------------------------------------------------------------------

/// A state with a `phase` field. `#[derive(Finite)]` on a struct that has one also derives this,
/// which is what lets `starts: [Unscheduled]` and `ends: [Succeeded, ..]` name a state by its
/// phase: `<<S as Phased>::Phase>::Unscheduled` is a real path, so a misspelled phase is rustc's
/// E0599 at the token.
pub trait Phased: Finite {
    type Phase: Finite;
    fn phase(&self) -> Self::Phase;
    /// The state at `phase` with every other field at its first value.
    fn at(phase: Self::Phase) -> Self;
}

/// The type-namespace twin of a `machine!`, `compose!` or `scenario!` static. Rust keeps types and
/// values in separate namespaces, so `machine! { nexusProtocol .. }` emits both
/// `pub static nexusProtocol: LazyLock<Machine<..>>` and `pub enum nexusProtocol {}` with this
/// impl; `property!` and `query!` then spell their static's type as
/// `Property<<nexusProtocol as Typed>::S, ..>` without knowing the state type themselves.
pub trait Typed {
    type S: Finite;
    type O: Finite;
    type F: Finite;
}

impl ActionClass {
    /// A member's action inside a composition: keyed `operation_schedule-unset-expires-unset`.
    pub fn member(role: &'static str, class: ActionClass) -> ActionClass {
        ActionClass { action: Box::leak(format!("{role}_{}", class.action).into_boxed_str()), ..class }
    }
}
