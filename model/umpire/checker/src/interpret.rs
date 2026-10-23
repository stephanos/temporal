//! A `#[machine]` Model executed by evaluating the admitted tree its macro embeds, rather than
//! its compiled handler and claim bodies. States stay the compiled Model's typed states, read
//! and written through the accessors the macro generates; the catalog, claim order and computed
//! or World-bound starts stay the compiled Model's.

use std::{
    collections::{HashMap, HashSet},
    sync::{Mutex, OnceLock},
};

use rustc_hash::FxHashMap;
use umpire_dsl::{
    __lifted::{
        Callbacks, Evaluates, Interprets, Type,
        expr::{Expr, ExprKind, Op, Unary},
        tree::{ActionRef, AssumptionDef, Branch, Lifted, Start},
        value::Value,
    },
    ActionCatalog, ActionValues, Assumption, CatalogError, Claim, Machine, OutcomeClass, Progress,
    ProgressEvaluation, StepClaim, StepError, Steps, Value as Exported, Values, Verdict, claim,
};

/// A Model whose steps, claims, assumptions and declared starts are evaluated from the admitted
/// tree its `#[machine]` embeds.
pub struct Interpreted<M: Machine> {
    world: M,
    program: Program,
    /// Each catalog action's tree action and input values.
    bindings: FxHashMap<M::Action, (usize, Vec<Value>)>,
    progress: Vec<Evaluated<M::Action>>,
    assumptions: Vec<(&'static str, Assumption<M::Action>)>,
    after: Vec<Option<M::Action>>,
    /// The compiled Model's one callback per claim, which judge through this World.
    callbacks: Callbacks<Self, M>,
    negative: Option<Claim<M>>,
}

struct Evaluated<A> {
    trigger: Node,
    target: Node,
    assumption: Assumption<A>,
    bound: Option<u32>,
}

/// The admitted tree with every name resolved to a slot.
struct Program {
    fields: usize,
    rejecting: bool,
    /// Field start domains, or `None` when the compiled Model enumerates its starts.
    starts: Option<Vec<Domain>>,
    actions: Vec<Action>,
    invariants: Vec<(&'static str, Node)>,
    postconditions: Vec<(&'static str, Node)>,
    progress: Vec<&'static str>,
}

enum Domain {
    Value(Node),
    Range(Node, Node, bool),
}

struct Action {
    internal: bool,
    branches: Vec<Arm>,
}

struct Arm {
    definitions: Vec<Node>,
    guard: Node,
    updates: Vec<(usize, Node)>,
    checks: Vec<Node>,
    outcome: Node,
    preserve_self_loop: bool,
}

/// An inclusive integer range, the type of a checked arithmetic result.
type Range = (i128, i128);

enum Node {
    Const(Value),
    /// A field of the state, or of the before-state a postcondition judges.
    Field(usize),
    /// A field of the after-state a postcondition judges.
    Next(usize),
    Input(usize),
    Outcome,
    Local(usize),
    Not(Box<Node>),
    Neg(Box<Node>, Range),
    Binary(Op, Box<Node>, Box<Node>, Range),
    If(Box<Node>, Box<Node>, Box<Node>),
    Tuple(Vec<Node>),
    Seq(Vec<Node>),
    Length(Box<Node>),
    Append(Box<Node>, Box<Node>),
    GetOr(Box<Node>, Box<Node>, Box<Node>),
    Checks(Vec<Node>),
    Check(&'static str, Box<Node>),
}

impl<M> Interpreted<M>
where
    M: Interprets + Values,
    M::Action: ActionValues<M>,
{
    /// Interprets `world`'s embedded Model.
    ///
    /// # Errors
    /// Refuses a World whose `#[machine]` module the front end refused, or a tree this
    /// interpreter cannot resolve against the World's values and catalog.
    pub fn new(world: M) -> Result<Self, String> {
        let embedded = M::__umpire_lifted();
        let model = embedded.model.map_err(|refusal| {
            format!(
                "the Model is not admitted: {}:{}: {}",
                refusal.line, refusal.column, refusal.message
            )
        })?;
        Self::from_tree(world, &model)
    }

    /// Interprets `model` in place of the tree `world` embeds, such as a seeded mutant of it.
    ///
    /// # Errors
    /// Refuses a tree this interpreter cannot resolve against the World's values and catalog.
    #[doc(hidden)]
    #[expect(
        clippy::too_many_lines,
        reason = "resolves the catalog, claims and assumptions in turn"
    )]
    pub fn from_tree(world: M, model: &Lifted) -> Result<Self, String> {
        let checks = M::__umpire_checks();
        let world_values = world_values(model, &world.values())?;
        let resolver = Resolver { model, world: &world_values, checks, inputs: &[], locals: &[] };
        let program = resolver.program()?;
        let names: HashMap<&str, usize> = model
            .actions
            .iter()
            .enumerate()
            .map(|(index, action)| (action.name.as_str(), index))
            .collect();
        let mut bindings = FxHashMap::default();
        let mut actions = HashMap::new();
        // An oversized catalog is refused when the checker admits it, before any step.
        if let Ok(catalog) = world.checked_catalog(umpire_dsl::CATALOG_LIMIT) {
            for action in catalog.into_actions() {
                let (name, values) = ActionValues::<M>::action_value(&action);
                let index = *names
                    .get(name)
                    .ok_or_else(|| format!("catalog action {name} is not in the admitted Model"))?;
                let fields = &model.actions[index].fields;
                if fields.len() != values.len() {
                    return Err(format!("catalog action {name} has {} inputs", values.len()));
                }
                let values = fields
                    .iter()
                    .zip(&values)
                    .map(|(field, (_, value))| import(&field.ty, value))
                    .collect::<Result<Vec<_>, _>>()?;
                actions.entry((index, values.clone())).or_insert_with(|| action.clone());
                bindings.insert(action, (index, values));
            }
        }
        let action = |reference: &ActionRef| -> Result<M::Action, String> {
            let index = *names
                .get(reference.name.as_str())
                .ok_or_else(|| format!("unknown action {}", reference.name))?;
            let values = reference
                .values
                .iter()
                .map(|value| resolver.constant(value))
                .collect::<Result<Vec<_>, _>>()?;
            actions
                .get(&(index, values))
                .cloned()
                .ok_or_else(|| format!("action {} is outside the catalog", reference.name))
        };
        let assumption = |def: &AssumptionDef| -> Result<Assumption<M::Action>, String> {
            let list = |references: &[ActionRef]| -> Result<Vec<M::Action>, String> {
                let mut actions = Vec::new();
                for reference in references {
                    let action = action(reference)?;
                    if !actions.contains(&action) {
                        actions.push(action);
                    }
                }
                Ok(actions)
            };
            Ok(Assumption {
                only: def.only.as_deref().map(list).transpose()?,
                excluded: list(&def.without)?,
                fair: list(&def.fair)?,
            })
        };
        let progress = model
            .progress
            .iter()
            .map(|claim| {
                Ok(Evaluated {
                    trigger: resolver.node(&claim.trigger)?,
                    target: resolver.node(&claim.target)?,
                    assumption: assumption(&claim.assumption)?,
                    bound: claim
                        .bound
                        .map(|bound| {
                            u32::try_from(bound).map_err(|_| "progress bound overflows".to_owned())
                        })
                        .transpose()?,
                })
            })
            .collect::<Result<Vec<_>, String>>()?;
        let assumptions = model
            .assumptions
            .iter()
            .map(|def| Ok((intern(&def.name), assumption(def)?)))
            .collect::<Result<Vec<_>, String>>()?;
        let after = model
            .postconditions
            .iter()
            .map(|claim| {
                claim
                    .after
                    .as_ref()
                    .map(|name| action(&ActionRef { name: name.clone(), values: Vec::new() }))
                    .transpose()
            })
            .collect::<Result<Vec<_>, String>>()?;
        let callbacks = registered::<M, Self>(model)?;
        Ok(Self {
            world,
            program,
            bindings,
            progress,
            assumptions,
            after,
            callbacks,
            negative: None,
        })
    }

    /// Adds a fixture's negative claim, which the compiled World judges on interpreted states.
    #[must_use]
    pub fn with_negative(mut self, claim: Claim<M>) -> Self {
        self.negative = Some(claim);
        self
    }

    /// The negative claim given to [`Interpreted::with_negative`], judged on this Model's states.
    pub fn negative(&self) -> Option<Claim<Self>> {
        self.negative.as_ref().map(|claim| {
            Claim::new(claim.name, |world: &Self, state| {
                world
                    .negative
                    .as_ref()
                    .map_or(Verdict::Holds, |claim| claim.evaluate(&world.world, state).verdict)
            })
        })
    }

    fn binding(&self, action: &M::Action) -> &(usize, Vec<Value>) {
        self.bindings
            .get(action)
            .unwrap_or_else(|| panic!("action {action} is outside the interpreted catalog"))
    }

    fn judge(
        &self,
        node: &Node,
        before: &M::State,
        after: Option<&M::State>,
        outcome: Option<&Value>,
    ) -> bool {
        let mut frame = Frame::new(
            M::__umpire_get,
            Some(before),
            after,
            &[],
            outcome,
            &[],
            self.program.fields,
        );
        frame.eval(node).truth()
    }
}

impl<M> Machine for Interpreted<M>
where
    M: Interprets + Values,
    M::Action: ActionValues<M>,
{
    type State = M::State;
    type Action = M::Action;
    type Outcome = M::Outcome;
    type Fact = M::Fact;

    fn starts(&self) -> Vec<Self::State> {
        let Some(domains) = &self.program.starts else { return self.world.starts() };
        let mut starts = Vec::new();
        let mut evaluated = vec![None; domains.len()];
        let mut selected = Vec::with_capacity(domains.len());
        product(domains, &mut evaluated, &mut selected, &mut |values| {
            starts.push(M::__umpire_state(values));
        });
        starts
    }

    fn actions(&self) -> Vec<Self::Action> {
        self.world.actions()
    }

    fn checked_catalog(&self, limit: usize) -> Result<ActionCatalog<Self::Action>, CatalogError> {
        self.world.checked_catalog(limit)
    }

    fn outcome_catalog(&self) -> Option<Vec<OutcomeClass<Self::Outcome>>> {
        self.world.outcome_catalog()
    }

    fn fault_catalog(&self) -> Option<Vec<Self::Action>> {
        self.world.fault_catalog()
    }

    fn internal(&self, action: &Self::Action) -> bool {
        self.program.actions[self.binding(action).0].internal
    }

    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact> {
        match self.checked_step(state, action) {
            Ok(steps) => steps,
            Err(error) => panic!("unchecked Model step: {error}"),
        }
    }

    fn checked_step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Result<Steps<Self::State, Self::Outcome, Self::Fact>, StepError> {
        let (index, inputs) = self.binding(&action);
        let mut steps = Steps::new();
        // One frame per step, so each field is read once however many arms are tried.
        let mut frame =
            Frame::new(M::__umpire_get, Some(state), None, inputs, None, &[], self.program.fields);
        for arm in &self.program.actions[*index].branches {
            frame.definitions = &arm.definitions;
            frame.locals.clear();
            frame.locals.resize(arm.definitions.len(), None);
            if !frame.eval(&arm.guard).truth() {
                continue;
            }
            for check in &arm.checks {
                frame.eval(check);
            }
            let outcome = frame.eval(&arm.outcome);
            let mut changes = Vec::new();
            for (slot, update) in &arm.updates {
                let value = frame.eval(update);
                if value != frame.field(*slot) {
                    changes.push((*slot, value));
                }
            }
            // The stock outcome's first variant is `Outcome::Accepted`.
            let rejected = self.program.rejecting && outcome != Value::Enum(0);
            if rejected && !changes.is_empty() {
                return Err(StepError::RejectedAfterMutation);
            }
            if !changes.is_empty() || rejected || arm.preserve_self_loop {
                let mut next = state.clone();
                for (slot, value) in &changes {
                    M::__umpire_set(&mut next, *slot, value);
                }
                steps.push((M::__umpire_outcome(&outcome), Vec::new(), next));
            }
            break;
        }
        Ok(steps)
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        self.program
            .invariants
            .iter()
            .zip(&self.callbacks.invariants)
            .map(|((name, _), (checks, verdict))| Claim::with_checks(name, *checks, *verdict))
            .collect()
    }

    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        self.program
            .postconditions
            .iter()
            .zip(&self.callbacks.postconditions)
            .zip(&self.after)
            .map(|(((name, _), checks), after)| match after {
                Some(action) => StepClaim::after_checks(name, action.clone(), *checks),
                None => StepClaim::with_checks(name, *checks),
            })
            .collect()
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        self.program
            .progress
            .iter()
            .zip(&self.callbacks.progress)
            .map(|(name, evaluator)| Progress::evaluated(name, *evaluator))
            .collect()
    }

    fn assumptions(&self) -> Vec<(&'static str, Assumption<Self::Action>)> {
        self.assumptions.clone()
    }

    fn claim_order(&self) -> Vec<&'static str> {
        self.world.claim_order()
    }
}

impl<M> Evaluates<M> for Interpreted<M>
where
    M: Interprets + Values,
    M::Action: ActionValues<M>,
{
    fn __umpire_invariant(&self, index: usize, state: &M::State) {
        self.judge(&self.program.invariants[index].1, state, None, None);
    }

    fn __umpire_postcondition(
        &self,
        index: usize,
        before: &M::State,
        after: &M::State,
        outcome: &M::Outcome,
    ) {
        let outcome = M::__umpire_outcome_value(outcome);
        self.judge(&self.program.postconditions[index].1, before, Some(after), Some(&outcome));
    }

    fn __umpire_progress(&self, index: usize, state: &M::State) -> ProgressEvaluation<M::Action> {
        let claim = &self.progress[index];
        ProgressEvaluation {
            trigger: self.judge(&claim.trigger, state, None, None).into(),
            target: self.judge(&claim.target, state, None, None).into(),
            assumption: claim.assumption.clone(),
            bound: claim.bound,
        }
    }
}

/// The compiled Model's claim callbacks, which must be exactly one per claim of the tree.
fn registered<M: Interprets, W: Evaluates<M>>(model: &Lifted) -> Result<Callbacks<W, M>, String> {
    let callbacks = M::__umpire_callbacks::<W>();
    for (kind, tree, compiled) in [
        ("invariants", model.invariants.len(), callbacks.invariants.len()),
        ("postconditions", model.postconditions.len(), callbacks.postconditions.len()),
        ("progress claims", model.progress.len(), callbacks.progress.len()),
    ] {
        if tree != compiled {
            return Err(format!(
                "the tree has {tree} {kind}; the compiled Model registers {compiled}"
            ));
        }
    }
    Ok(callbacks)
}

/// Field-order starts, each field's domain evaluated only once an earlier field has a value,
/// as the compiled nested loops do, so an empty earlier domain never evaluates a later one.
fn product(
    domains: &[Domain],
    evaluated: &mut [Option<Vec<Value>>],
    selected: &mut Vec<Value>,
    visit: &mut dyn FnMut(&[Value]),
) {
    let depth = selected.len();
    let Some(domain) = domains.get(depth) else {
        visit(selected);
        return;
    };
    let values = evaluated[depth].get_or_insert_with(|| domain.values()).clone();
    for value in values {
        selected.push(value);
        product(domains, evaluated, selected, visit);
        selected.pop();
    }
}

impl Domain {
    fn values(&self) -> Vec<Value> {
        let mut frame = Frame::<()>::new(
            |(), _| unreachable!("a start domain reads no state"),
            None,
            None,
            &[],
            None,
            &[],
            0,
        );
        match self {
            Self::Value(value) => vec![frame.eval(value)],
            Self::Range(low, high, inclusive) => {
                let (low, high) = (frame.eval(low).integer(), frame.eval(high).integer());
                let high = if *inclusive { high } else { high - 1 };
                (low..=high).map(Value::Int).collect()
            }
        }
    }
}

/// Claim names and `check!` texts live as long as the process, as compiled ones do.
fn intern(name: &str) -> &'static str {
    static NAMES: OnceLock<Mutex<HashSet<&'static str>>> = OnceLock::new();
    let mut names =
        NAMES.get_or_init(Mutex::default).lock().unwrap_or_else(std::sync::PoisonError::into_inner);
    if let Some(interned) = names.get(name) {
        return interned;
    }
    let interned: &'static str = Box::leak(name.to_owned().into_boxed_str());
    names.insert(interned);
    interned
}

fn world_values(
    model: &Lifted,
    values: &[(&'static str, Exported)],
) -> Result<HashMap<String, Value>, String> {
    let mut world = HashMap::new();
    for field in &model.world_fields {
        let (_, value) = values
            .iter()
            .find(|(name, _)| *name == field.name)
            .ok_or_else(|| format!("the World exports no field {}", field.name))?;
        world.insert(field.name.clone(), import(&field.ty, value)?);
    }
    Ok(world)
}

/// An exported Model value as the interpreter's value of its admitted type.
fn import(ty: &Type, value: &Exported) -> Result<Value, String> {
    Ok(match (ty, value) {
        (Type::Bool, Exported::Bool(value)) => Value::Bool(*value),
        (Type::Int { .. }, Exported::Int(value)) => Value::Int(*value),
        (Type::Enum { variants, .. }, Exported::Enum(name)) => Value::Enum(
            variants
                .iter()
                .position(|variant| variant == name)
                .ok_or_else(|| format!("{name} is not a variant of {ty:?}"))?,
        ),
        (Type::Unit, Exported::Sequence(values)) if values.is_empty() => Value::Unit,
        (Type::Seq(inner), Exported::Sequence(values)) => {
            Value::Seq(values.iter().map(|value| import(inner, value)).collect::<Result<_, _>>()?)
        }
        _ => return Err(format!("{value:?} is not a value of {ty:?}")),
    })
}

struct Resolver<'a> {
    model: &'a Lifted,
    world: &'a HashMap<String, Value>,
    checks: &'static [&'static str],
    inputs: &'a [String],
    locals: &'a [String],
}

impl Resolver<'_> {
    fn program(&self) -> Result<Program, String> {
        let model = self.model;
        let starts = if model.declared_world
            || model.fields.iter().any(|field| matches!(field.start, Start::Computed))
        {
            None
        } else {
            Some(
                model
                    .fields
                    .iter()
                    .map(|field| {
                        Ok(match &field.start {
                            Start::Value(value) => Domain::Value(self.node(value)?),
                            Start::Range(low, high) => {
                                Domain::Range(self.node(low)?, self.node(high)?, true)
                            }
                            Start::ExclusiveRange(low, high) => {
                                Domain::Range(self.node(low)?, self.node(high)?, false)
                            }
                            Start::Computed => unreachable!("computed starts stay compiled"),
                        })
                    })
                    .collect::<Result<_, String>>()?,
            )
        };
        let actions = model
            .actions
            .iter()
            .map(|action| {
                let inputs: Vec<String> =
                    action.fields.iter().map(|field| field.name.clone()).collect();
                let resolver = Resolver { inputs: &inputs, ..*self };
                Ok(Action {
                    internal: action.internal,
                    branches: action
                        .branches
                        .iter()
                        .map(|branch| resolver.arm(branch))
                        .collect::<Result<_, String>>()?,
                })
            })
            .collect::<Result<_, String>>()?;
        let claims = |claims: &mut dyn Iterator<Item = (&String, &Expr)>| {
            claims
                .map(|(name, expr)| Ok((intern(name), self.node(expr)?)))
                .collect::<Result<Vec<_>, String>>()
        };
        Ok(Program {
            fields: model.fields.len(),
            rejecting: model.outcome_type != Type::Unit,
            starts,
            actions,
            invariants: claims(
                &mut model.invariants.iter().map(|claim| (&claim.name, &claim.expr)),
            )?,
            postconditions: claims(
                &mut model.postconditions.iter().map(|claim| (&claim.name, &claim.expr)),
            )?,
            progress: model.progress.iter().map(|claim| intern(&claim.name)).collect(),
        })
    }

    fn arm(&self, branch: &Branch) -> Result<Arm, String> {
        let names: Vec<String> = branch.definitions.iter().map(|(name, _)| name.clone()).collect();
        let resolver = Resolver { locals: &names, ..*self };
        let mut updates = Vec::new();
        for (slot, field) in self.model.fields.iter().enumerate() {
            let Some(update) = branch.updates.get(&field.name) else { continue };
            if update.kind != ExprKind::Var(field.name.clone(), false) {
                updates.push((slot, resolver.node(update)?));
            }
        }
        Ok(Arm {
            definitions: branch
                .definitions
                .iter()
                .map(|(_, value)| resolver.node(value))
                .collect::<Result<_, String>>()?,
            guard: resolver.node(&branch.guard)?,
            updates,
            checks: branch
                .checks
                .iter()
                .map(|check| resolver.node(check))
                .collect::<Result<_, String>>()?,
            outcome: resolver.node(&branch.outcome)?,
            preserve_self_loop: branch.preserve_self_loop,
        })
    }

    /// A value the tree spells without reading a state, such as an assumption's action input.
    fn constant(&self, expr: &Expr) -> Result<Value, String> {
        let node = self.node(expr)?;
        let mut frame = Frame::<()>::new(
            |(), _| unreachable!("a constant reads no state"),
            None,
            None,
            &[],
            None,
            &[],
            0,
        );
        Ok(frame.eval(&node))
    }

    fn node(&self, expr: &Expr) -> Result<Node, String> {
        let boxed = |expr: &Expr| self.node(expr).map(Box::new);
        let list = |values: &[Expr]| {
            values.iter().map(|value| self.node(value)).collect::<Result<Vec<_>, _>>()
        };
        Ok(match &expr.kind {
            ExprKind::Bool(value) => Node::Const(Value::Bool(*value)),
            ExprKind::Int(value) => Node::Const(Value::Int(*value)),
            ExprKind::Text(value) => Node::Const(Value::Text(value.clone())),
            ExprKind::Unit => Node::Const(Value::Unit),
            ExprKind::Enum(name) => match &expr.ty {
                Type::Enum { variants, .. } => Node::Const(Value::Enum(
                    variants
                        .iter()
                        .position(|variant| variant == name)
                        .ok_or_else(|| format!("{name} is not a variant of its type"))?,
                )),
                _ => return Err(format!("enum value {name} has a non-enum type")),
            },
            ExprKind::Var(name, primed) => {
                let slot = self
                    .model
                    .fields
                    .iter()
                    .position(|field| field.name == *name)
                    .ok_or_else(|| format!("unknown state field {name}"))?;
                if *primed { Node::Next(slot) } else { Node::Field(slot) }
            }
            ExprKind::World(name) => Node::Const(
                self.world
                    .get(name)
                    .cloned()
                    .ok_or_else(|| format!("unknown World field {name}"))?,
            ),
            ExprKind::Input(name) => Node::Input(
                self.inputs
                    .iter()
                    .position(|input| input == name)
                    .ok_or_else(|| format!("unknown input {name}"))?,
            ),
            ExprKind::Outcome => Node::Outcome,
            ExprKind::Local(name) => Node::Local(
                self.locals
                    .iter()
                    .position(|local| local == name)
                    .ok_or_else(|| format!("unknown local {name}"))?,
            ),
            ExprKind::Unary(Unary::Not, value) => Node::Not(boxed(value)?),
            ExprKind::Unary(Unary::Neg, value) => Node::Neg(boxed(value)?, range(&expr.ty)),
            ExprKind::Binary(op, left, right) => {
                Node::Binary(*op, boxed(left)?, boxed(right)?, range(&expr.ty))
            }
            ExprKind::If(condition, yes, no) => {
                Node::If(boxed(condition)?, boxed(yes)?, boxed(no)?)
            }
            ExprKind::Tuple(values) => Node::Tuple(list(values)?),
            ExprKind::Sequence(values) => Node::Seq(list(values)?),
            ExprKind::Length(value) => Node::Length(boxed(value)?),
            ExprKind::Append(sequence, value) => Node::Append(boxed(sequence)?, boxed(value)?),
            ExprKind::GetOr(sequence, index, fallback) => {
                Node::GetOr(boxed(sequence)?, boxed(index)?, boxed(fallback)?)
            }
            ExprKind::Checks(values) => Node::Checks(list(values)?),
            ExprKind::Check(index, value) => Node::Check(
                self.checks
                    .get(*index)
                    .copied()
                    .ok_or_else(|| format!("no check! text at {index}"))?,
                boxed(value)?,
            ),
        })
    }
}

fn range(ty: &Type) -> Range {
    match ty {
        // The front spells `usize` (lengths, indices) as this range, which no admitted integer
        // type has; Rust computes it in `usize`.
        Type::Int { min: 0, max } if *max == i128::from(i32::MAX) => {
            (0, i128::try_from(usize::MAX).unwrap_or(i128::MAX))
        }
        Type::Int { min, max } => (*min, *max),
        _ => (i128::MIN, i128::MAX),
    }
}

/// What one evaluation reads: a state or a judged edge, the action's inputs, and the branch's
/// definitions, each read at most once.
struct Frame<'a, S> {
    get: fn(&S, usize) -> Value,
    before: Option<&'a S>,
    after: Option<&'a S>,
    inputs: &'a [Value],
    outcome: Option<&'a Value>,
    definitions: &'a [Node],
    fields: Vec<Option<Value>>,
    next: Vec<Option<Value>>,
    locals: Vec<Option<Value>>,
}

impl<'a, S> Frame<'a, S> {
    #[expect(clippy::too_many_arguments, reason = "a frame names everything one evaluation reads")]
    fn new(
        get: fn(&S, usize) -> Value,
        before: Option<&'a S>,
        after: Option<&'a S>,
        inputs: &'a [Value],
        outcome: Option<&'a Value>,
        definitions: &'a [Node],
        fields: usize,
    ) -> Self {
        Self {
            get,
            before,
            after,
            inputs,
            outcome,
            definitions,
            fields: vec![None; fields],
            next: if after.is_some() { vec![None; fields] } else { Vec::new() },
            locals: vec![None; definitions.len()],
        }
    }

    fn field(&mut self, slot: usize) -> Value {
        let state = self.before.unwrap_or_else(|| panic!("a constant reads no state"));
        let get = self.get;
        self.fields[slot].get_or_insert_with(|| get(state, slot)).clone()
    }

    fn eval(&mut self, node: &Node) -> Value {
        match node {
            Node::Const(value) => value.clone(),
            Node::Field(slot) => self.field(*slot),
            Node::Next(slot) => {
                let state = self
                    .after
                    .unwrap_or_else(|| panic!("only a postcondition reads an after-state"));
                let get = self.get;
                self.next[*slot].get_or_insert_with(|| get(state, *slot)).clone()
            }
            Node::Input(index) => self.inputs[*index].clone(),
            Node::Outcome => self
                .outcome
                .cloned()
                .unwrap_or_else(|| panic!("only a postcondition reads an outcome")),
            Node::Local(index) => {
                if let Some(value) = &self.locals[*index] {
                    return value.clone();
                }
                let definitions = self.definitions;
                let value = self.eval(&definitions[*index]);
                self.locals[*index] = Some(value.clone());
                value
            }
            Node::Not(value) => Value::Bool(!self.eval(value).truth()),
            Node::Neg(value, (min, max)) => {
                let value = self.eval(value).integer();
                Value::Int(
                    Some(-value)
                        .filter(|negated| (*min..=*max).contains(negated))
                        .unwrap_or_else(|| panic!("attempt to negate with overflow")),
                )
            }
            Node::Binary(Op::And, left, right, _) => {
                Value::Bool(self.eval(left).truth() && self.eval(right).truth())
            }
            Node::Binary(Op::Or, left, right, _) => {
                Value::Bool(self.eval(left).truth() || self.eval(right).truth())
            }
            Node::Binary(op, left, right, range) => {
                let (left, right) = (self.eval(left), self.eval(right));
                match op {
                    Op::Eq => Value::Bool(left == right),
                    Op::Ne => Value::Bool(left != right),
                    Op::Lt => Value::Bool(left.integer() < right.integer()),
                    Op::Le => Value::Bool(left.integer() <= right.integer()),
                    Op::Gt => Value::Bool(left.integer() > right.integer()),
                    Op::Ge => Value::Bool(left.integer() >= right.integer()),
                    Op::Add | Op::Sub | Op::Mul | Op::Div | Op::Rem => {
                        Value::Int(arithmetic(*op, left.integer(), right.integer(), *range))
                    }
                    Op::And | Op::Or => unreachable!("short-circuit operators evaluate above"),
                }
            }
            Node::If(condition, yes, no) => {
                if self.eval(condition).truth() {
                    self.eval(yes)
                } else {
                    self.eval(no)
                }
            }
            Node::Tuple(values) => {
                Value::Tuple(values.iter().map(|value| self.eval(value)).collect())
            }
            Node::Seq(values) => Value::Seq(values.iter().map(|value| self.eval(value)).collect()),
            Node::Length(sequence) => Value::Int(
                i128::try_from(self.eval(sequence).elements().len()).unwrap_or(i128::MAX),
            ),
            Node::Append(sequence, value) => {
                let mut elements = self.eval(sequence).into_elements();
                elements.push(self.eval(value));
                Value::Seq(elements)
            }
            Node::GetOr(sequence, index, fallback) => {
                // `get(index).copied().unwrap_or(fallback)` evaluates its arguments in order.
                let elements = self.eval(sequence).into_elements();
                let index = self.eval(index).integer();
                let fallback = self.eval(fallback);
                usize::try_from(index)
                    .ok()
                    .and_then(|index| elements.into_iter().nth(index))
                    .unwrap_or(fallback)
            }
            Node::Checks(checks) => {
                // Every statement of a claim body runs; the claim holds when all of them do.
                let mut holds = true;
                for check in checks {
                    holds &= self.eval(check).truth();
                }
                Value::Bool(holds)
            }
            Node::Check(text, value) => {
                let holds = self.eval(value).truth();
                if !holds {
                    claim::fail(text);
                }
                Value::Bool(holds)
            }
        }
    }
}

/// Checked integer arithmetic within the result type, failing as compiled Rust does.
fn arithmetic(op: Op, left: i128, right: i128, (min, max): Range) -> i128 {
    let (result, verb) = match op {
        Op::Add => (left.checked_add(right), "add"),
        Op::Sub => (left.checked_sub(right), "subtract"),
        Op::Mul => (left.checked_mul(right), "multiply"),
        Op::Div => {
            assert!(right != 0, "attempt to divide by zero");
            (left.checked_div(right), "divide")
        }
        Op::Rem => {
            assert!(right != 0, "attempt to calculate the remainder with a divisor of zero");
            assert!(
                !(left == min && right == -1 && min < 0),
                "attempt to calculate the remainder with overflow"
            );
            (left.checked_rem(right), "calculate the remainder")
        }
        _ => unreachable!("not an arithmetic operator"),
    };
    match result.filter(|result| (min..=max).contains(result)) {
        Some(result) => result,
        None => match verb {
            "add" => panic!("attempt to add with overflow"),
            "subtract" => panic!("attempt to subtract with overflow"),
            "multiply" => panic!("attempt to multiply with overflow"),
            "divide" => panic!("attempt to divide with overflow"),
            _ => panic!("attempt to calculate the remainder with overflow"),
        },
    }
}

/// Reading interpreted values by their admitted type.
trait Typed {
    fn truth(&self) -> bool;
    fn integer(&self) -> i128;
    fn elements(&self) -> &[Value];
    fn into_elements(self) -> Vec<Value>;
}

impl Typed for Value {
    fn truth(&self) -> bool {
        match self {
            Self::Bool(value) => *value,
            _ => panic!("an interpreted condition is not a boolean: {self:?}"),
        }
    }
    fn integer(&self) -> i128 {
        match self {
            Self::Int(value) => *value,
            _ => panic!("an interpreted operand is not an integer: {self:?}"),
        }
    }
    fn elements(&self) -> &[Value] {
        match self {
            Self::Seq(values) => values,
            _ => panic!("an interpreted operand is not a sequence: {self:?}"),
        }
    }
    fn into_elements(self) -> Vec<Value> {
        match self {
            Self::Seq(values) => values,
            _ => panic!("an interpreted operand is not a sequence: {self:?}"),
        }
    }
}
