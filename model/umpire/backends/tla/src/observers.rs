//! Private trace field names shared by emission, decoding and replay.
use crate::{
    Refusal,
    values::{TraceState, Value},
};

macro_rules! names {
    ($($name:ident = $wire:literal),* $(,)?) => {$(
        #[doc = concat!("The `", $wire, "` trace observer.")]
        pub const $name: &str = $wire;
    )*};
}
names! {
    ACTION = "__umpire_action", INPUT = "__umpire_input", CHOICE = "__umpire_choice",
    OUTCOME = "__umpire_outcome", FACTS = "__umpire_facts", EDGE_TICK = "__umpire_edge_tick",
    PENDING = "__umpire_pending", STEPS = "__umpire_steps", ERROR = "__umpire_arithmetic_error",
    INVALID = "__umpire_invalid_step", STUTTER = "__umpire_stutter", HAS_STEP = "__umpire_has_step",
}
/// The retained predecessor field used by safety traces.
#[must_use]
pub fn before(field: &str) -> String {
    format!("__umpire_before_{field}")
}
/// TLC's fixed observer fields, in emission order.
pub const TLC: [&str; 7] = [EDGE_TICK, PENDING, STEPS, ACTION, CHOICE, OUTCOME, FACTS];

/// Decoded event identity and typed result, without backend-specific history.
pub struct EdgeObservers<'a> {
    /// Qualified action variant.
    pub action: &'a str,
    /// Concrete input ordinal; absent on TLC's unit-action encoding.
    pub input: Option<usize>,
    /// Enabled choice ordinal.
    pub choice: usize,
    /// Exact outcome value.
    pub outcome: &'a Value,
    /// Exact ordered fact vector.
    pub facts: &'a [Value],
}
impl<'a> EdgeObservers<'a> {
    /// Admit the edge fields, refusing absent or mistyped values by field name.
    ///
    /// # Errors
    /// Returns the malformed observer's name.
    pub fn decode(state: &'a TraceState) -> Result<Self, Refusal> {
        let missing = |name| Refusal(format!("missing or invalid edge observer {name}"));
        let Some(Value::Text(action)) = state.observers.get(ACTION) else {
            return Err(missing(ACTION));
        };
        let ordinal = |name| match state.observers.get(name) {
            Some(Value::Int(value)) => usize::try_from(*value).map_err(|_| missing(name)),
            _ => Err(missing(name)),
        };
        let input = state.observers.contains_key(INPUT).then(|| ordinal(INPUT)).transpose()?;
        let choice = ordinal(CHOICE)?;
        let outcome = state.observers.get(OUTCOME).ok_or_else(|| missing(OUTCOME))?;
        let Some(Value::Seq(facts)) = state.observers.get(FACTS) else {
            return Err(missing(FACTS));
        };
        Ok(Self { action, input, choice, outcome, facts })
    }
}
