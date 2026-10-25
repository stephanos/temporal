//! Typed native confirmation of a selected bounded safety witness.
use std::{
    any::Any,
    collections::{BTreeMap, BTreeSet},
    fmt::{self, Write},
    panic::{AssertUnwindSafe, catch_unwind},
};
use umpire_dsl::{__lifted::ConcreteOf, ActionValues, Claim, IntoValue, Machine, Values, Verdict};
use umpire_tla::{
    Concrete, Literal, Module, Refusal, Type as SourceType,
    values::{Schema, State, TraceState, Value, accepts},
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
        let schema = module.safety_schema().map_err(|Refusal(error)| reject(error))?;
        unit_facts(module)?;
        let parameters: BTreeMap<_, _> =
            meta.parameters.iter().map(|p| (p.symbol.clone(), p.ty.clone())).collect();
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
        let evaluated = self.initial(module, initial, &schema)?;
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
    fn initial(
        &self,
        module: &Module,
        initial: &TraceState,
        schema: &Schema,
    ) -> Result<NativeStep<M>, Refusal> {
        let types = || module.metadata.fields.iter().map(|f| (f.name.as_str(), &f.ty));
        let current = self
            .starts
            .iter()
            .find(|s| typed_native(s.values(), types()).is_ok_and(|s| s == initial.state))
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
                    expected.extend(typed_fields(failure.prefix, &types().collect())?);
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
                    || after.observers[OUTCOME] != default_value(&module.metadata.outcome_type)
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
                let outcome = admitted(emitted.0.value(), &module.metadata.outcome_type)?;
                let facts = emitted
                    .1
                    .iter()
                    .map(|f| admitted(f.value(), fact_type(module)?))
                    .collect::<Result<Vec<_>, _>>()?;
                if outcome != after.observers[OUTCOME]
                    || Value::Seq(facts.clone()) != after.observers[FACTS]
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
                        action: Concrete::of::<M>(action),
                        ordinal,
                        choice,
                        outcome,
                        facts,
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
                || !accepts(&p.ty, &parameters[&p.symbol])
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
        let actions = self.actions.iter().map(Concrete::of::<M>).collect::<Vec<_>>();
        let expected = if let Some(binding) = &meta.binding {
            if literals(&binding.world, meta.parameters.iter().map(|p| (p.field.as_str(), &p.ty)))?
                != native_world
            {
                return Err(reject("emitted World binding differs from native World"));
            }
            let mut unique = Vec::new();
            for row in starts {
                if !unique.contains(&row) {
                    unique.push(row);
                }
            }
            if binding
                .starts
                .iter()
                .map(|row| literals(row, meta.fields.iter().map(|f| (f.name.as_str(), &f.ty))))
                .collect::<Result<Vec<_>, _>>()?
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
        let identity = Concrete::of::<M>(action);
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
        let payload = literals(
            &identity.fields,
            declaration.fields.iter().map(|f| (f.name.as_str(), &f.ty)),
        )?;
        validate_fields(
            &payload,
            &declaration.fields.iter().map(|f| (f.name.clone(), f.ty.clone())).collect(),
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
    let message = umpire_checker::panic_message(error);
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
    .contains(&message)
    {
        message.into()
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
/// Refuses a Model whose edges carry facts: their replay has no Apalache control yet.
fn unit_facts(module: &Module) -> Result<(), Refusal> {
    let fact = fact_type(module)?;
    if *fact != SourceType::Unit {
        return Err(reject(format!(
            "Apalache replay does not yet confirm edge facts; the Model declares facts of {fact:?}"
        )));
    }
    Ok(())
}
/// An exported native value as the wire value of its admitted type.
fn admitted(value: umpire_dsl::Value, ty: &SourceType) -> Result<Value, Refusal> {
    let admitted = Literal::from(value).admit(ty).map_err(|error| reject(error.to_string()))?;
    Ok(Value::admitted(&admitted, ty))
}
/// The element type of the facts observer.
fn fact_type(module: &Module) -> Result<&SourceType, Refusal> {
    match module.metadata.observer_types.get(FACTS) {
        Some(SourceType::Seq(inner)) => Ok(inner),
        _ => Err(reject("missing facts observer")),
    }
}
/// Emitted literal fields, exactly the declared ones, as wire values of their admitted types.
fn literals<'a>(
    values: &[(String, Literal)],
    types: impl Iterator<Item = (&'a str, &'a SourceType)>,
) -> Result<State, Refusal> {
    let types: BTreeMap<_, _> = types.collect();
    let mut fields = State::new();
    for (name, value) in values {
        let ty = types
            .get(name.as_str())
            .ok_or_else(|| reject(format!("undeclared binding field: {name}")))?;
        let admitted =
            value.admit(ty).map_err(|error| reject(format!("binding field {name}: {error}")))?;
        if fields.insert(name.clone(), Value::admitted(&admitted, ty)).is_some() {
            return Err(reject(format!("duplicate binding field: {name}")));
        }
    }
    if let Some(name) = types.keys().find(|name| !fields.contains_key(**name)) {
        return Err(reject(format!("missing binding field: {name}")));
    }
    Ok(fields)
}
/// Exactly the declared native fields, as wire values of their admitted types.
fn typed_native<'a>(
    values: Vec<(&'static str, umpire_dsl::Value)>,
    types: impl Iterator<Item = (&'a str, &'a SourceType)>,
) -> Result<State, Refusal> {
    let types: BTreeMap<_, _> = types.collect();
    if values.len() != types.len() {
        return Err(reject("native field declarations differ"));
    }
    typed_fields(values, &types)
}
/// Native fields, each declared, as wire values of their admitted types.
fn typed_fields(
    values: Vec<(&'static str, umpire_dsl::Value)>,
    types: &BTreeMap<&str, &SourceType>,
) -> Result<State, Refusal> {
    let mut result = State::new();
    for (name, value) in values {
        let ty =
            types.get(name).ok_or_else(|| reject(format!("undeclared native field: {name}")))?;
        let admitted = Literal::from(value)
            .admit(ty)
            .map_err(|error| reject(format!("typed native field differs: {name}: {error}")))?;
        if result.insert(name.into(), Value::admitted(&admitted, ty)).is_some() {
            return Err(reject("duplicate native projected field"));
        }
    }
    Ok(result)
}
fn validate_fields(values: &State, types: &BTreeMap<String, SourceType>) -> Result<(), Refusal> {
    if values.len() != types.len()
        || types.iter().any(|(n, t)| values.get(n).is_none_or(|v| !accepts(t, v)))
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
fn default_value(ty: &SourceType) -> Value {
    match ty {
        SourceType::Bool => Value::Bool(false),
        SourceType::Int { min, max } => Value::Int(0.clamp(*min, *max)),
        SourceType::Bounded(b) => Value::Int(0.clamp(b.low, b.high)),
        SourceType::Optional(_) => Value::Int(0),
        SourceType::Text => Value::Text(String::new()),
        SourceType::Enum { variants, .. } => {
            Value::Text(variants.first().cloned().unwrap_or_default())
        }
        SourceType::Seq(_) | SourceType::Unit => Value::Seq(vec![]),
        SourceType::Tuple(v) => Value::Seq(v.iter().map(default_value).collect()),
    }
}
fn defaults(types: &BTreeMap<String, SourceType>) -> State {
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
mod fact_tests {
    use super::{SourceType, unit_facts};

    #[test]
    fn models_with_edge_facts_are_refused_until_their_replay_is_controlled()
    -> Result<(), Box<dyn std::error::Error>> {
        let source = r"
            use umpire_dsl::machine;
            #[machine] mod plain {
                use umpire_dsl::{State, Action};
                #[derive(State)] struct Model { done: bool }
                #[derive(Action)] enum User { Finish }
                impl Model {
                    #[on(User)] fn user(&mut self, action: User) {
                        match action { User::Finish => self.done = true }
                    }
                }
            }
        ";
        let mut module = umpire_tla::Tla::safety(&umpire_tla::Lift::parse("plain.rs", source)?)?;
        unit_facts(&module)?;
        module
            .metadata
            .observer_types
            .insert("__umpire_facts".into(), SourceType::Seq(Box::new(SourceType::Bool)));
        let error = unit_facts(&module).err().ok_or("non-unit facts were admitted")?;
        assert!(error.0.contains("does not yet confirm edge facts"), "{error}");
        module.metadata.observer_types.remove("__umpire_facts");
        assert!(unit_facts(&module).is_err());
        Ok(())
    }
}
