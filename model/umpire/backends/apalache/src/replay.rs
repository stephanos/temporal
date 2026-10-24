//! Typed native confirmation of a selected bounded safety witness.
use std::{
    any::Any,
    collections::{BTreeMap, BTreeSet},
    fmt::{self, Write},
    panic::{AssertUnwindSafe, catch_unwind},
};
use umpire_dsl::{ActionValues, Claim, IntoValue, Machine, Values, Verdict};
use umpire_tla::{Concrete, Literal, Module, Type as SourceType};
use umpire_tla::{
    Refusal,
    values::{Schema, State, TraceState, Type, Value},
};

const ERROR: &str = "__umpire_arithmetic_error";
const INVALID: &str = "__umpire_invalid_step";
const STUTTER: &str = "__umpire_stutter";
const HAS_STEP: &str = "__umpire_has_step";
const ACTION: &str = "__umpire_action";
const INPUT: &str = "__umpire_input";
const CHOICE: &str = "__umpire_choice";
const OUTCOME: &str = "__umpire_outcome";
const FACTS: &str = "__umpire_facts";

/// Retained native inputs for the same World that emitted this module. No graph is consumed.
pub struct Replay<'a, M: Machine> {
    /// Selected native World.
    pub world: &'a M,
    /// Complete starts already enumerated for this World.
    pub starts: &'a [M::State],
    /// Checked concrete catalog retained in its original order.
    pub actions: &'a [M::Action],
    /// Supplemental compiler-checked negative claims, in source order.
    pub negative: &'a [Claim<M>],
}
/// What native evaluation confirmed, independently of wire admission.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Confirmation {
    /// The selected authored claim is false.
    Violation,
    /// Evaluated native arithmetic refused at this trace position; no fallback edge is accepted.
    Arithmetic {
        /// Zero is start/initial-claim evaluation; later indices identify a transition boundary.
        index: usize,
        /// Native arithmetic panic diagnostic.
        reason: String,
    },
}
/// One accepted native step in the valid prefix, with its complete typed event identity.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReplayedStep {
    /// Qualified variant and complete ordered payload.
    pub action: Concrete,
    /// Canonical concrete catalog ordinal emitted by the module.
    pub ordinal: usize,
    /// Enabled native choice.
    pub choice: usize,
    /// Exact native outcome.
    pub outcome: Value,
    /// Ordered native facts.
    pub facts: Vec<Value>,
    /// Complete native successor.
    pub state: State,
}
/// A confirmed result and its deterministic, typed path rendering.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Replayed {
    /// Selected emitted property operator.
    pub property: String,
    /// Requested transition bound, never an unbounded claim.
    pub length: usize,
    /// Exact typed selected World parameter assignment.
    pub world: State,
    /// Native confirmation, with arithmetic kept distinct from an authored violation.
    pub confirmation: Confirmation,
    /// Complete initial state of the valid native prefix; absent for an initial arithmetic refusal.
    pub initial: Option<State>,
    /// Valid native prefix steps; refusal fallback states are excluded.
    pub steps: Vec<ReplayedStep>,
    /// Deterministic rendering; formatting never participates in replay identity.
    pub path: String,
}
impl fmt::Display for Replayed {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "property={} bound={} World={:?}", self.property, self.length, self.world)?;
        writeln!(f, "{:?}", self.confirmation)?;
        f.write_str(&self.path)
    }
}
fn reject(message: impl Into<String>) -> Refusal {
    Refusal(format!("witness-rejected: {}", message.into()))
}

impl<M> Replay<'_, M>
where
    M: Machine + Values,
    M::State: Values,
    M::Action: ActionValues<M>,
    M::Outcome: IntoValue,
    M::Fact: IntoValue,
{
    /// Decode and directly execute every edge, then refute this specific declared property.
    pub fn violation(
        &self,
        module: &Module,
        property: &str,
        length: usize,
        raw: &str,
    ) -> Result<Replayed, Refusal> {
        let meta = &module.metadata;
        let (category, selected) = selected_property(module, property)?;
        let schema = schema(module)?;
        let parameters: BTreeMap<_, _> =
            meta.parameters.iter().map(|p| (p.symbol.clone(), kind(&p.ty))).collect();
        if parameters.len() != meta.parameters.len()
            || meta.parameters.iter().map(|p| &p.field).collect::<BTreeSet<_>>().len()
                != meta.parameters.len()
        {
            return Err(reject("duplicate World parameter"));
        }
        let trace = crate::decode_itf(raw, &schema, &parameters, length)
            .map_err(|e| reject(e.to_string()))?;
        self.binding(module, &trace.parameters, &schema)?;
        self.claims(module)?;
        let initial = &trace.states[0];
        initial_observers(initial, &schema)?;
        let mut path = String::new();
        let evaluated = self.initial(initial, &schema)?;
        let mut current = evaluated.state;
        let mut confirmation = evaluated.confirmation;
        writeln!(&mut path, "0: {:?}", initial.state).map_err(|e| reject(e.to_string()))?;
        let mut last_post = None;
        let mut steps = Vec::new();
        for (offset, pair) in trace.states.windows(2).enumerate() {
            let index = offset + 1;
            let (before, after) = (&pair[0], &pair[1]);
            if boolean(after, STUTTER)? {
                stutter(before, after)?;
                continue;
            }
            if confirmation.is_some() {
                return Err(reject("authored event after a native refusal"));
            }
            let native_before =
                current.as_ref().ok_or_else(|| reject("missing native prefix state"))?;
            let post = (category == Category::Postcondition).then_some(selected.name.as_str());
            let evaluated = self.transition(module, (before, after), index, native_before, post)?;
            let ordinal = integer(after, INPUT)?;
            let choice = integer(after, CHOICE)?;
            last_post = evaluated.post;
            current = evaluated.state;
            steps.extend(evaluated.event);
            writeln!(
                &mut path,
                "{index}: {} input={ordinal} choice={choice} outcome={:?} facts={:?}\n   {:?}",
                text(after, ACTION)?,
                after.observers[OUTCOME],
                after.observers[FACTS],
                after.state
            )
            .map_err(|e| reject(e.to_string()))?;
            confirmation = evaluated.confirmation;
        }
        let final_state = trace.states.last().ok_or_else(|| reject("empty trace"))?;
        let confirmation = match confirmation {
            Some(c @ Confirmation::Arithmetic { .. })
                if category == Category::Arithmetic && boolean(final_state, ERROR)? =>
            {
                c
            }
            Some(_) => return Err(reject("native refusal does not refute selected claim")),
            None => {
                let state = current.as_ref().ok_or_else(|| reject("missing final native state"))?;
                if matches!(category, Category::Arithmetic | Category::ValidStep) {
                    return Err(reject("selected safety monitor has no native refusal"));
                }
                let verdict = if category == Category::Postcondition {
                    last_post.ok_or_else(|| reject("postcondition has no authored transition"))?
                } else {
                    let claims = self.world.invariants();
                    let registrations =
                        if category == Category::Invariant { &claims[..] } else { self.negative };
                    let claim = registrations
                        .iter()
                        .find(|c| c.name == selected.name)
                        .ok_or_else(|| reject("selected property has no native claim"))?;
                    native(|| claim.evaluate(self.world, state).verdict).map_err(reject)?
                };
                if verdict != Verdict::Fails {
                    return Err(reject("specific selected claim is not refuted"));
                }
                Confirmation::Violation
            }
        };
        Ok(Replayed {
            property: property.into(),
            length,
            world: trace.parameters,
            confirmation,
            initial: (!boolean(initial, ERROR)?).then(|| initial.state.clone()),
            steps,
            path,
        })
    }
    fn initial(&self, initial: &TraceState, schema: &Schema) -> Result<NativeStep<M>, Refusal> {
        let current = self
            .starts
            .iter()
            .find(|s| fields(s.values()).is_ok_and(|s| s == initial.state))
            .cloned();
        let mut confirmation = None;
        if boolean(initial, ERROR)? {
            let arithmetic = if let Some(state) = &current {
                self.state_claims(state).err()
            } else {
                let selected = initial
                    .state
                    .iter()
                    .filter_map(|(name, value)| match value {
                        Value::Int(value) => Some((name.as_str(), *value)),
                        _ => None,
                    })
                    .collect::<Vec<_>>();
                let mut matched = None;
                if let Some(failure) = self.world.start_failure(&selected) {
                    let mut expected = defaults(&schema.fields);
                    for (name, value) in fields(failure.prefix)? {
                        if !schema.fields.contains_key(&name) {
                            return Err(reject("native start prefix has an unknown field"));
                        }
                        expected.insert(name, value);
                    }
                    if initial.state == expected {
                        matched = Some(native_panic(failure.panic.as_ref()));
                    }
                }
                matched
            }
            .ok_or_else(|| reject("initial arithmetic flag has no native refusal"))?;
            confirmation =
                Some(Confirmation::Arithmetic { index: 0, reason: arithmetic_reason(arithmetic)? });
        } else {
            let state = current
                .as_ref()
                .ok_or_else(|| reject("initial state is outside retained starts"))?;
            self.state_claims(state).map_err(reject)?;
        }
        Ok(NativeStep { state: current, confirmation, post: None, event: None })
    }

    fn transition(
        &self,
        module: &Module,
        pair: (&TraceState, &TraceState),
        index: usize,
        native_before: &M::State,
        post: Option<&str>,
    ) -> Result<NativeStep<M>, Refusal> {
        let (before, after) = pair;
        if !boolean(after, HAS_STEP)? {
            return Err(reject("event has no has_step history"));
        }
        before_history(before, after)?;
        let (ordinal, action) = self.action(module, after)?;
        let choice = integer(after, CHOICE)?;
        let emitted = native(|| self.world.checked_step(native_before, action.clone()));
        let mut refused = None;
        let mut event = None;
        let mut state = Some(native_before.clone());
        let mut last_post = None;
        match emitted {
            Err(reason) => {
                if !boolean(after, ERROR)? || boolean(after, INVALID)? {
                    return Err(reject("native arithmetic refusal differs from witness"));
                }
                if after.state != before.state
                    || after.observers[OUTCOME]
                        != default_value(&kind(&module.metadata.outcome_type))
                    || after.observers[FACTS] != Value::Seq(vec![])
                {
                    return Err(reject("action arithmetic fallback differs"));
                }
                refused =
                    Some(Confirmation::Arithmetic { index, reason: arithmetic_reason(reason)? });
            }
            Ok(Err(error)) => {
                return Err(reject(format!("native checked_step refused: {error}")));
            }
            Ok(Ok(steps)) => {
                let emitted =
                    steps.get(choice).ok_or_else(|| reject("disabled action or unknown choice"))?;
                if typed_native(
                    emitted.2.values(),
                    module.metadata.fields.iter().map(|f| (f.name.as_str(), &f.ty)),
                )? != after.state
                {
                    return Err(reject("native successor differs"));
                }
                if projected(emitted.0.value())? != after.observers[OUTCOME]
                    || Value::Seq(
                        emitted.1.iter().map(|f| projected(f.value())).collect::<Result<_, _>>()?,
                    ) != after.observers[FACTS]
                {
                    return Err(reject("native outcome/facts differ"));
                }
                if boolean(after, INVALID)? {
                    return Err(reject("successful native checked_step marked invalid"));
                }
                let arithmetic = self
                    .state_claims(&emitted.2)
                    .and_then(|()| self.step_claims(native_before, action, emitted));
                match (boolean(after, ERROR)?, arithmetic) {
                    (true, Err(reason)) => {
                        refused = Some(Confirmation::Arithmetic {
                            index,
                            reason: arithmetic_reason(reason)?,
                        });
                    }
                    (true, Ok(())) => {
                        return Err(reject("arithmetic flag has no native refusal"));
                    }
                    (false, Err(reason)) => {
                        return Err(reject(format!("unreported native arithmetic: {reason}")));
                    }
                    (false, Ok(())) => {}
                }
                if let Some(selected_name) = post {
                    let claims = self.world.postconditions();
                    let claim = claims
                        .iter()
                        .find(|c| c.name == selected_name)
                        .ok_or_else(|| reject("missing native postcondition"))?;
                    last_post = Some(
                        native(|| claim.holds(self.world, native_before, action, emitted))
                            .map_err(reject)?,
                    );
                }
                if refused.is_none() {
                    event = Some(ReplayedStep {
                        action: concrete::<M>(action),
                        ordinal,
                        choice,
                        outcome: projected(emitted.0.value())?,
                        facts: emitted
                            .1
                            .iter()
                            .map(|f| projected(f.value()))
                            .collect::<Result<_, _>>()?,
                        state: after.state.clone(),
                    });
                }
                state = Some(emitted.2.clone());
            }
        }
        Ok(NativeStep { state, confirmation: refused, post: last_post, event })
    }

    fn binding(&self, module: &Module, parameters: &State, schema: &Schema) -> Result<(), Refusal> {
        let meta = &module.metadata;
        let native_world = typed_native(
            self.world.values(),
            meta.parameters.iter().map(|p| (p.field.as_str(), &p.ty)),
        )?;
        if native_world.len() != meta.parameters.len() {
            return Err(reject("native World fields differ"));
        }
        for p in &meta.parameters {
            if native_world.get(&p.field) != parameters.get(&p.symbol)
                || !kind(&p.ty).accepts(&parameters[&p.symbol])
            {
                return Err(reject("crossed selected World binding"));
            }
        }
        let starts = self
            .starts
            .iter()
            .map(|s| typed_native(s.values(), meta.fields.iter().map(|f| (f.name.as_str(), &f.ty))))
            .collect::<Result<Vec<_>, _>>()?;
        for s in &starts {
            validate_fields(s, &schema.fields)?;
        }
        let actions = self.actions.iter().map(concrete::<M>).collect::<Vec<_>>();
        let expected = if let Some(binding) = &meta.binding {
            validate_literals(
                &binding.world,
                meta.parameters.iter().map(|p| (p.field.as_str(), &p.ty)),
            )?;
            for row in &binding.starts {
                validate_literals(row, meta.fields.iter().map(|f| (f.name.as_str(), &f.ty)))?;
            }
            if literals(&binding.world)? != native_world {
                return Err(reject("emitted World binding differs from native World"));
            }
            let mut unique = Vec::new();
            for row in starts {
                if !unique.contains(&row) {
                    unique.push(row);
                }
            }
            if binding.starts.iter().map(|row| literals(row)).collect::<Result<Vec<_>, _>>()?
                != unique
            {
                return Err(reject("retained complete starts differ from emitted binding"));
            }
            binding.actions.clone()
        } else {
            if !meta.parameters.is_empty() {
                return Err(reject("World has no retained emitted binding"));
            }
            meta.actions
                .iter()
                .map(|a| Concrete { action: a.name.clone(), fields: vec![] })
                .collect()
        };
        if actions != expected || (!meta.catalog.is_empty() && meta.catalog != expected) {
            return Err(reject("retained concrete catalog differs from emitted binding"));
        }
        Ok(())
    }
    fn claims(&self, module: &Module) -> Result<(), Refusal> {
        let meta = &module.metadata;
        let invariants = self.world.invariants();
        let posts = self.world.postconditions();
        for (expected, actual) in [
            (
                meta.invariants
                    .iter()
                    .filter(|p| p.operator != "ValidStep")
                    .map(|p| p.name.as_str())
                    .collect::<Vec<_>>(),
                invariants.iter().map(|p| p.name).collect::<Vec<_>>(),
            ),
            (
                meta.negative.iter().map(|p| p.name.as_str()).collect(),
                self.negative.iter().map(|p| p.name).collect(),
            ),
            (
                meta.postconditions.iter().map(|p| p.name.as_str()).collect(),
                posts.iter().map(|p| p.name).collect(),
            ),
        ] {
            if expected != actual {
                return Err(reject("native claim registrations differ"));
            }
        }
        Ok(())
    }
    fn state_claims(&self, state: &M::State) -> Result<(), String> {
        for claim in self.world.invariants().iter().chain(self.negative) {
            native(|| claim.evaluate(self.world, state))?;
        }
        Ok(())
    }
    fn step_claims(
        &self,
        before: &M::State,
        action: &M::Action,
        step: &umpire_dsl::Step<M::State, M::Outcome, M::Fact>,
    ) -> Result<(), String> {
        for claim in self.world.postconditions() {
            native(|| claim.evaluate(self.world, before, action, step))?;
        }
        Ok(())
    }
    fn action<'a>(
        &'a self,
        module: &Module,
        state: &TraceState,
    ) -> Result<(usize, &'a M::Action), Refusal> {
        let ordinal = integer(state, INPUT)?;
        let action = if module.metadata.binding.is_some() {
            self.actions.get(ordinal)
        } else {
            // Legacy unit catalogs emit ordinal zero for every variant.
            if ordinal != 0 {
                return Err(reject("unit action ordinal is not zero"));
            }
            self.actions.iter().find(|a| {
                ActionValues::<M>::action_value(*a).0 == text(state, ACTION).unwrap_or_default()
            })
        }
        .ok_or_else(|| reject("action ordinal outside retained catalog"))?;
        let identity = concrete::<M>(action);
        if let Some(binding) = &module.metadata.binding
            && binding.actions.iter().position(|entry| entry == &identity) != Some(ordinal)
        {
            return Err(reject("input ordinal is not the emitted first concrete identity"));
        }
        if identity.action != text(state, ACTION)? {
            return Err(reject("action variant differs from ordinal"));
        }
        let declaration = module
            .metadata
            .actions
            .iter()
            .find(|a| a.name == identity.action)
            .ok_or_else(|| reject("undeclared action"))?;
        validate_literals(
            &identity.fields,
            declaration.fields.iter().map(|f| (f.name.as_str(), &f.ty)),
        )?;
        let payload = literals(&identity.fields)?;
        validate_fields(
            &payload,
            &declaration.fields.iter().map(|f| (f.name.clone(), kind(&f.ty))).collect(),
        )?;
        Ok((ordinal, action))
    }
}
struct NativeStep<M: Machine> {
    state: Option<M::State>,
    confirmation: Option<Confirmation>,
    post: Option<Verdict>,
    event: Option<ReplayedStep>,
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Category {
    Invariant,
    Postcondition,
    Negative,
    Arithmetic,
    ValidStep,
}
fn selected_property<'a>(
    module: &'a Module,
    operator: &str,
) -> Result<(Category, &'a umpire_tla::Property), Refusal> {
    let meta = &module.metadata;
    let mut matching = [
        (Category::Invariant, &meta.invariants),
        (Category::Postcondition, &meta.postconditions),
        (Category::Negative, &meta.negative),
        (Category::Arithmetic, &meta.arithmetic),
    ]
    .into_iter()
    .flat_map(|(category, properties)| {
        properties.iter().map(move |p| {
            let category = if category == Category::Invariant && p.operator == "ValidStep" {
                Category::ValidStep
            } else {
                category
            };
            (category, p)
        })
    })
    .filter(|(_, p)| p.operator == operator);
    let selected = matching.next().ok_or_else(|| reject("unknown property"))?;
    if matching.next().is_some() {
        return Err(reject("ambiguous property"));
    }
    Ok(selected)
}

fn native<T>(evaluate: impl FnOnce() -> T) -> Result<T, String> {
    catch_unwind(AssertUnwindSafe(evaluate)).map_err(|error| native_panic(error.as_ref()))
}
fn native_panic(error: &(dyn Any + Send)) -> String {
    let message = panic_text(error);
    if [
        "attempt to add with overflow",
        "attempt to subtract with overflow",
        "attempt to multiply with overflow",
        "attempt to divide with overflow",
        "attempt to calculate the remainder with overflow",
        "attempt to negate with overflow",
        "attempt to divide by zero",
        "attempt to calculate the remainder with a divisor of zero",
    ]
    .contains(&message.as_str())
    {
        message
    } else {
        format!("non-arithmetic native panic: {message}")
    }
}
fn arithmetic_reason(reason: String) -> Result<String, Refusal> {
    if reason.starts_with("non-arithmetic native panic:") {
        Err(reject(reason))
    } else {
        Ok(reason)
    }
}
fn panic_text(error: &(dyn Any + Send)) -> String {
    error
        .downcast_ref::<String>()
        .cloned()
        .or_else(|| error.downcast_ref::<&str>().map(|s| (*s).into()))
        .unwrap_or_else(|| "unrecognized panic payload".into())
}
fn kind(ty: &SourceType) -> Type {
    match ty {
        SourceType::Bool => Type::Bool,
        SourceType::Int { min, max } => Type::Int { min: *min, max: *max },
        SourceType::Bounded(b) => Type::Int { min: b.low, max: b.high },
        SourceType::Optional(b) => Type::OptionalInt { low: b.low, high: b.high },
        SourceType::Text => Type::Text,
        SourceType::Enum { variants, .. } => Type::Enum(variants.clone()),
        SourceType::Unit => Type::Tuple(vec![]),
        SourceType::Tuple(types) => Type::Tuple(types.iter().map(kind).collect()),
        SourceType::Seq(inner) => Type::Seq(Box::new(kind(inner))),
    }
}
fn schema(module: &Module) -> Result<Schema, Refusal> {
    let meta = &module.metadata;
    for ty in meta
        .fields
        .iter()
        .map(|f| &f.ty)
        .chain(meta.parameters.iter().map(|p| &p.ty))
        .chain(meta.observer_types.values())
        .chain(meta.actions.iter().flat_map(|a| a.fields.iter().map(|f| &f.ty)))
        .chain(std::iter::once(&meta.outcome_type))
    {
        validate_source_type(ty)?;
    }
    let schema = Schema {
        fields: meta.fields.iter().map(|f| (f.name.clone(), kind(&f.ty))).collect(),
        observers: meta.observer_types.iter().map(|(n, t)| (n.clone(), kind(t))).collect(),
    };
    if schema.fields.len() != meta.fields.len()
        || meta.observer_fields.iter().collect::<BTreeSet<_>>()
            != meta.observer_types.keys().collect()
        || meta.observer_fields.len() != meta.observer_types.len()
    {
        return Err(reject("inconsistent schema declarations"));
    }
    let mut expected = BTreeMap::from([
        (ERROR.into(), Type::Bool),
        (INVALID.into(), Type::Bool),
        (STUTTER.into(), Type::Bool),
        (HAS_STEP.into(), Type::Bool),
        (ACTION.into(), Type::Text),
        (
            INPUT.into(),
            schema.observers.get(INPUT).cloned().ok_or_else(|| reject("missing input observer"))?,
        ),
        (CHOICE.into(), Type::Int { min: 0, max: 0 }),
        (OUTCOME.into(), kind(&meta.outcome_type)),
        (FACTS.into(), Type::Seq(Box::new(Type::Tuple(vec![])))),
    ]);
    for f in &meta.fields {
        expected.insert(format!("__umpire_before_{}", f.name), kind(&f.ty));
    }
    if expected != schema.observers {
        return Err(reject("observer schema differs from safety emitter"));
    }
    schema.validate_assignment()?;
    Ok(schema)
}
fn validate_source_type(ty: &SourceType) -> Result<(), Refusal> {
    match ty {
        SourceType::Bounded(b) if !b.valid() => return Err(reject("invalid bounded descriptor")),
        SourceType::Optional(b) if !b.optional_valid() => {
            return Err(reject("invalid optional descriptor"));
        }
        SourceType::Tuple(types) => {
            for ty in types {
                validate_source_type(ty)?;
            }
        }
        SourceType::Seq(inner) => validate_source_type(inner)?,
        _ => {}
    }
    Ok(())
}
fn typed_literal(ty: &SourceType, value: &Literal) -> bool {
    match (ty, value) {
        (SourceType::Bounded(b), Literal::Bounded { name, raw }) => {
            b.name == *name && b.contains(*raw)
        }
        (SourceType::Optional(b), Literal::Optional(None)) => b.optional_valid(),
        (SourceType::Optional(b), Literal::Optional(Some(value))) => match value.as_ref() {
            Literal::Bounded { name, raw } => b.name == *name && b.encode(*raw).is_some(),
            _ => false,
        },
        (SourceType::Bounded(_) | SourceType::Optional(_), _) => false,
        (SourceType::Seq(inner), Literal::Sequence(values)) => {
            values.iter().all(|v| typed_literal(inner, v))
        }
        (SourceType::Tuple(types), Literal::Sequence(values)) => {
            types.len() == values.len()
                && types.iter().zip(values).all(|(t, v)| typed_literal(t, v))
        }
        (SourceType::Bool, Literal::Bool(_))
        | (SourceType::Int { .. }, Literal::Int(_))
        | (SourceType::Enum { .. }, Literal::Enum(_))
        | (SourceType::Unit, Literal::Sequence(_)) => {
            literal(value).is_ok_and(|value| kind(ty).accepts(&value))
        }
        _ => false,
    }
}
fn validate_literals<'a>(
    values: &[(String, Literal)],
    types: impl Iterator<Item = (&'a str, &'a SourceType)>,
) -> Result<(), Refusal> {
    let types: BTreeMap<_, _> = types.collect();
    if values.len() != types.len()
        || values.iter().map(|(name, _)| name).collect::<BTreeSet<_>>().len() != values.len()
        || values.iter().any(|(name, value)| {
            types.get(name.as_str()).is_none_or(|ty| !typed_literal(ty, value))
        })
    {
        return Err(reject("typed literal fields differ"));
    }
    Ok(())
}
fn typed_native<'a>(
    values: Vec<(&'static str, umpire_dsl::Value)>,
    types: impl Iterator<Item = (&'a str, &'a SourceType)>,
) -> Result<State, Refusal> {
    let types: BTreeMap<_, _> = types.collect();
    if values.len() != types.len() {
        return Err(reject("native field declarations differ"));
    }
    let mut result = State::new();
    for (name, value) in values {
        let value = native_literal(value);
        if types.get(name).is_none_or(|ty| !typed_literal(ty, &value)) {
            return Err(reject(format!("typed native field differs: {name}")));
        }
        if result.insert(name.into(), literal(&value)?).is_some() {
            return Err(reject("duplicate native projected field"));
        }
    }
    Ok(result)
}
fn projected(v: umpire_dsl::Value) -> Result<Value, Refusal> {
    literal(&native_literal(v))
}
fn fields(values: Vec<(&'static str, umpire_dsl::Value)>) -> Result<State, Refusal> {
    let count = values.len();
    let fields: State = values
        .into_iter()
        .map(|(n, v)| Ok((n.into(), projected(v)?)))
        .collect::<Result<_, Refusal>>()?;
    if fields.len() != count {
        return Err(reject("duplicate native projected field"));
    }
    Ok(fields)
}
fn literal(v: &Literal) -> Result<Value, Refusal> {
    Ok(match v {
        Literal::Bool(v) => Value::Bool(*v),
        Literal::Int(v) => Value::Int(*v),
        Literal::Enum(v) => Value::Text(v.clone()),
        Literal::Sequence(v) => Value::Seq(v.iter().map(literal).collect::<Result<_, _>>()?),
        Literal::Bounded { raw, .. } => Value::Int(*raw),
        Literal::Optional(None) => Value::Int(0),
        Literal::Optional(Some(v)) => match v.as_ref() {
            Literal::Bounded { raw, .. } => {
                Value::Int(raw.checked_add(1).ok_or_else(|| reject("optional encoding overflow"))?)
            }
            _ => return Err(reject("optional payload is not bounded")),
        },
    })
}
fn literals(values: &[(String, Literal)]) -> Result<State, Refusal> {
    let fields: State =
        values.iter().map(|(n, v)| Ok((n.clone(), literal(v)?))).collect::<Result<_, Refusal>>()?;
    if fields.len() != values.len() {
        return Err(reject("duplicate binding field"));
    }
    Ok(fields)
}
fn concrete<M: Machine>(a: &M::Action) -> Concrete
where
    M::Action: ActionValues<M>,
{
    let (action, fields) = ActionValues::<M>::action_value(a);
    Concrete {
        action: action.into(),
        fields: fields.into_iter().map(|(n, v)| (n.into(), native_literal(v))).collect(),
    }
}
fn native_literal(v: umpire_dsl::Value) -> Literal {
    match v {
        umpire_dsl::Value::Bool(v) => Literal::Bool(v),
        umpire_dsl::Value::Int(v) => Literal::Int(v),
        umpire_dsl::Value::Bounded { name, raw } => Literal::Bounded { name: name.into(), raw },
        umpire_dsl::Value::Optional(v) => {
            Literal::Optional(v.map(|v| Box::new(native_literal(*v))))
        }
        umpire_dsl::Value::Enum(v) => Literal::Enum(v.into()),
        umpire_dsl::Value::Sequence(v) => {
            Literal::Sequence(v.into_iter().map(native_literal).collect())
        }
    }
}
fn validate_fields(values: &State, types: &BTreeMap<String, Type>) -> Result<(), Refusal> {
    if values.len() != types.len()
        || types.iter().any(|(n, t)| values.get(n).is_none_or(|v| !t.accepts(v)))
    {
        return Err(reject("typed native fields differ"));
    }
    Ok(())
}
fn boolean(state: &TraceState, name: &str) -> Result<bool, Refusal> {
    match state.observers.get(name) {
        Some(Value::Bool(v)) => Ok(*v),
        _ => Err(reject(format!("missing bool observer {name}"))),
    }
}
fn integer(state: &TraceState, name: &str) -> Result<usize, Refusal> {
    match state.observers.get(name) {
        Some(Value::Int(v)) => usize::try_from(*v).map_err(|_| reject("observer ordinal range")),
        _ => Err(reject(format!("missing ordinal observer {name}"))),
    }
}
fn text<'a>(state: &'a TraceState, name: &str) -> Result<&'a str, Refusal> {
    match state.observers.get(name) {
        Some(Value::Text(v)) => Ok(v),
        _ => Err(reject(format!("missing text observer {name}"))),
    }
}
fn default_value(ty: &Type) -> Value {
    match ty {
        Type::Bool => Value::Bool(false),
        Type::Int { min, max } => Value::Int(0.clamp(*min, *max)),
        Type::OptionalInt { .. } => Value::Int(0),
        Type::Text => Value::Text(String::new()),
        Type::Enum(v) => Value::Text(v.first().cloned().unwrap_or_default()),
        Type::Seq(_) | Type::Set(_) => Value::Seq(vec![]),
        Type::Tuple(v) => Value::Seq(v.iter().map(default_value).collect()),
        Type::Record(v) => Value::Record(defaults(v)),
    }
}
fn defaults(types: &BTreeMap<String, Type>) -> State {
    types.iter().map(|(n, t)| (n.clone(), default_value(t))).collect()
}
fn initial_observers(state: &TraceState, schema: &Schema) -> Result<(), Refusal> {
    let mut expected = defaults(&schema.observers);
    expected.insert(STUTTER.into(), Value::Bool(true));
    expected.insert(ERROR.into(), Value::Bool(boolean(state, ERROR)?));
    if state.observers != expected {
        return Err(reject("initial event/history differs from emitted Init"));
    }
    Ok(())
}
fn before_history(before: &TraceState, after: &TraceState) -> Result<(), Refusal> {
    for (name, value) in &before.state {
        if after.observers.get(&format!("__umpire_before_{name}")) != Some(value) {
            return Err(reject("before-state history differs"));
        }
    }
    Ok(())
}
fn stutter(before: &TraceState, after: &TraceState) -> Result<(), Refusal> {
    let mut expected = before.clone();
    expected.observers.insert(STUTTER.into(), Value::Bool(true));
    if expected != *after {
        return Err(reject("tagged stutter changes state or event/history"));
    }
    Ok(())
}

#[cfg(test)]
mod optional_tests {
    use super::*;

    fn ticks() -> SourceType {
        SourceType::Optional(umpire_dsl::__lifted::Bounded {
            name: "Ticks".into(),
            field: None,
            backing: "u8".into(),
            min: 0,
            max: 255,
            low: 1,
            high: 3,
        })
    }

    #[test]
    fn nominal_optional_projection_refuses_wrong_names_and_malformed_payloads() {
        let ty = ticks();
        let present = |name: &str, raw| {
            Literal::Optional(Some(Box::new(Literal::Bounded { name: name.into(), raw })))
        };
        assert!(typed_literal(&ty, &Literal::Optional(None)));
        assert!(typed_literal(&ty, &present("Ticks", 1)));
        for bad in [
            present("Remaining", 1),
            present("Ticks", 0),
            present("Ticks", 4),
            Literal::Int(2),
            Literal::Optional(Some(Box::new(Literal::Int(1)))),
            Literal::Optional(Some(Box::new(Literal::Optional(None)))),
        ] {
            assert!(!typed_literal(&ty, &bad), "accepted {bad:?}");
        }
        for bad in [
            Literal::Optional(Some(Box::new(Literal::Int(1)))),
            Literal::Optional(Some(Box::new(Literal::Optional(None)))),
        ] {
            assert!(literal(&bad).is_err());
        }
        assert!(
            projected(umpire_dsl::Value::Optional(Some(Box::new(umpire_dsl::Value::Optional(
                None
            )))))
            .is_err()
        );
        let SourceType::Optional(mut descriptor) = ty else { unreachable!() };
        descriptor.max = 256;
        assert!(validate_source_type(&SourceType::Optional(descriptor)).is_err());
    }
}
