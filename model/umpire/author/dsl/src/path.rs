//! Typed patterns over Model states, actions, inputs and outcomes.
//!
//! The pattern itself is neutral. A `#[progress]` claim reads it over every admitted path,
//! a search reads it as a request for one path.

use std::{fmt, rc::Rc};

/// A finite pattern cannot be expanded under its declared limits.
#[derive(Clone, Debug, PartialEq, Eq, thiserror::Error)]
pub enum PatternError {
    /// The alternative count or expanded sequence length exceeds `usize`.
    #[error("pattern expansion count overflows usize")]
    ExpansionOverflow,
    /// The number of alternatives exceeds the search budget.
    #[error("pattern has {count} alternatives, over budget {budget}")]
    Budget {
        /// Authored alternative count.
        count: usize,
        /// Admitted alternative budget.
        budget: usize,
    },
    /// The expanded concrete node storage exceeds the admission limit.
    #[error("pattern expansion bound is {nodes} nodes, over storage limit {limit}")]
    ExpansionTooLarge {
        /// Conservative bound on concrete nodes across alternatives.
        nodes: usize,
        /// Maximum admitted concrete nodes.
        limit: usize,
    },
    /// An unordered group exceeds its declared element limit.
    #[error("unordered pattern has {elements} elements ({permutations}), over limit {limit}",
        permutations = .count.map_or_else(|| "permutation count overflows usize".to_owned(), |count| format!("{count} permutations")))]
    PermutationLimit {
        /// Number of authored positional elements.
        elements: usize,
        /// Maximum admitted positional elements.
        limit: usize,
        /// Exact permutation count, absent when it overflows.
        count: Option<usize>,
    },
}

/// A lazily indexed finite action domain.
pub struct ActionDomain<A> {
    count: Result<usize, PatternError>,
    get: Rc<dyn Fn(usize) -> A>,
    label: Rc<dyn Fn(&A) -> String>,
}

impl<A> Clone for ActionDomain<A> {
    fn clone(&self) -> Self {
        Self { count: self.count.clone(), get: self.get.clone(), label: self.label.clone() }
    }
}

impl<A: 'static> ActionDomain<A> {
    /// Builds a domain without allocating its values.
    pub fn new(count: Result<usize, PatternError>, get: impl Fn(usize) -> A + 'static) -> Self {
        Self { count, get: Rc::new(get), label: Rc::new(|_| String::new()) }
    }

    /// The checked number of values.
    pub fn count(&self) -> Result<usize, PatternError> {
        self.count.clone()
    }

    /// Retrieves one value in declared order.
    pub fn get(&self, index: usize) -> A {
        (self.get)(index)
    }

    fn described(mut self) -> Self
    where
        A: fmt::Display,
    {
        self.label = Rc::new(ToString::to_string);
        self
    }

    /// Concatenates domains in declaration order.
    pub fn concat(domains: Vec<Self>) -> Self {
        let count = domains.iter().try_fold(0usize, |count, domain| {
            count.checked_add(domain.count()?).ok_or(PatternError::ExpansionOverflow)
        });
        Self::new(count, move |mut index| {
            for domain in &domains {
                let count = domain.count().unwrap_or(0);
                if index < count {
                    return domain.get(index);
                }
                index -= count;
            }
            panic!("action domain index out of bounds")
        })
    }

    /// Converts values while retaining lazy enumeration.
    pub fn map<B: 'static>(self, map: impl Fn(A) -> B + 'static) -> ActionDomain<B> {
        ActionDomain::new(self.count(), move |index| map(self.get(index)))
    }
}

/// A lazy finite domain for an action input.
pub struct InputDomain<T>(ActionDomain<T>);

impl<T> Clone for InputDomain<T> {
    fn clone(&self) -> Self {
        Self(self.0.clone())
    }
}

impl<T: 'static> InputDomain<T> {
    /// Lists exact values in source order, including positional duplicates.
    pub fn values(values: Vec<T>) -> Self
    where
        T: Clone,
    {
        Self(ActionDomain::new(Ok(values.len()), move |index| values[index].clone()))
    }

    /// Retains a finite integer range without allocating its members.
    pub fn range(range: impl InputRange<T>) -> Self {
        Self(range.domain())
    }

    /// Counts a repeatable finite enumeration once and re-enumerates it for each value,
    /// allocating none of its members.
    pub fn enumerated<I: Iterator<Item = T>>(values: impl Fn() -> I + 'static) -> Self {
        let count = values()
            .try_fold(0usize, |count, _| count.checked_add(1))
            .ok_or(PatternError::ExpansionOverflow);
        Self(ActionDomain::new(count, move |index| {
            values().nth(index).unwrap_or_else(|| panic!("action domain index out of bounds"))
        }))
    }

    /// Counts a complete finite stream exactly, retaining at most its first `retain` values.
    /// Admission must refuse any count above `retain` before values are retrieved.
    pub fn iterated(values: impl IntoIterator<Item = T>, retain: usize) -> Self
    where
        T: Clone,
    {
        let mut retained = Vec::new();
        let mut count = Some(0usize);
        for value in values {
            if retained.len() < retain {
                retained.push(value);
            }
            count = count.and_then(|count| count.checked_add(1));
        }
        Self(ActionDomain::new(count.ok_or(PatternError::ExpansionOverflow), move |index| {
            retained[index].clone()
        }))
    }

    /// Converts values while retaining lazy enumeration.
    pub fn map<U: 'static>(self, map: impl Fn(T) -> U + 'static) -> InputDomain<U> {
        InputDomain(self.0.map(map))
    }

    /// The checked number of input values.
    pub fn count(&self) -> Result<usize, PatternError> {
        self.0.count()
    }

    /// Retrieves an input assignment.
    pub fn get(&self, index: usize) -> T {
        self.0.get(index)
    }
}

/// Integer ranges supported by finite action input patterns.
pub trait InputRange<T> {
    /// Converts a bounded range to a lazy domain.
    fn domain(self) -> ActionDomain<T>;
}

macro_rules! integer_domains {
    ($($ty:ty),* $(,)?) => { $(
        impl InputRange<$ty> for std::ops::RangeInclusive<$ty> {
            #[allow(clippy::cast_lossless, clippy::cast_possible_truncation, clippy::cast_possible_wrap, reason = "checked domain counts bound offsets; wrapping offsets preserve signed ranges across zero")]
            fn domain(self) -> ActionDomain<$ty> {
                let (start, end) = self.into_inner();
                let count = if start > end { Ok(0) } else {
                    (end.abs_diff(start) as u128).checked_add(1)
                        .and_then(|n| usize::try_from(n).ok()).ok_or(PatternError::ExpansionOverflow)
                };
                ActionDomain::new(count, move |index| start.wrapping_add(index as $ty))
            }
        }
        impl InputRange<$ty> for std::ops::Range<$ty> {
            #[allow(clippy::cast_possible_truncation, clippy::cast_possible_wrap, reason = "checked domain counts bound offsets; wrapping offsets preserve signed ranges across zero")]
            fn domain(self) -> ActionDomain<$ty> {
                let start = self.start;
                let count = if start >= self.end { Ok(0) } else {
                    usize::try_from(self.end.abs_diff(start)).map_err(|_| PatternError::ExpansionOverflow)
                };
                ActionDomain::new(count, move |index| start.wrapping_add(index as $ty))
            }
        }
    )* };
}

integer_domains!(u8, u16, u32, u64, u128, usize, i8, i16, i32, i64, i128, isize);

/// A borrowed typed outcome classifier.
pub type OutcomePredicate<O> = Rc<dyn Fn(&O) -> bool>;
/// A borrowed typed exclusion for intervening gap edges.
pub type ActionPredicate<A> = Rc<dyn Fn(&A) -> bool>;

/// One concrete node of an enumerated path alternative.
pub enum PathNode<S, A, O> {
    /// Inspect the current state without consuming an edge.
    State(Rc<Waypoint<S>>),
    /// Consume exactly one edge, optionally checking its outcome.
    Action {
        /// Exact typed action value.
        action: Rc<A>,
        /// Optional borrowed outcome judgment.
        outcome: Option<OutcomePredicate<O>>,
        /// Concrete action and requested outcome spelling.
        label: String,
    },
    /// Consume intervening edges, excluding only those matched by `exclude`.
    Gap {
        /// Maximum number of intervening edges.
        gap: Gap,
        /// Actions forbidden only among intervening edges.
        exclude: Option<ActionPredicate<A>>,
    },
}

impl<S, A, O> Clone for PathNode<S, A, O> {
    fn clone(&self) -> Self {
        match self {
            Self::State(state) => Self::State(state.clone()),
            Self::Action { action, outcome, label } => Self::Action {
                action: action.clone(),
                outcome: outcome.clone(),
                label: label.clone(),
            },
            Self::Gap { gap, exclude } => Self::Gap { gap: *gap, exclude: exclude.clone() },
        }
    }
}

/// One authored finite choice, ready for witness search.
pub struct Alternative<S, A, O> {
    /// Concrete state checks, edges and gaps, in source order.
    pub nodes: Vec<PathNode<S, A, O>>,
    /// Concrete choices in a readable stable spelling.
    pub choices: Vec<String>,
}

impl<S, A, O> Clone for Alternative<S, A, O> {
    fn clone(&self) -> Self {
        Self { nodes: self.nodes.clone(), choices: self.choices.clone() }
    }
}

impl<S, A, O> Alternative<S, A, O> {
    fn empty() -> Self {
        Self { nodes: vec![], choices: vec![] }
    }
}

/// A finite typed path expression, expanded only after count admission.
#[must_use]
pub enum Pattern<S, A, O> {
    /// One state check.
    State(Rc<Waypoint<S>>),
    /// A finite action domain with an optional borrowed outcome classifier.
    Action(ActionDomain<A>, Option<OutcomePredicate<O>>, Option<String>),
    /// An intervening edge gap.
    Gap(Gap, Option<ActionPredicate<A>>),
    /// All component patterns in order.
    Sequence(Vec<Self>),
    /// One component pattern in declaration order.
    Either(Vec<Self>),
    /// Fresh component choices for each occurrence and each bounded count.
    Repeat(Box<Self>, usize, usize),
    /// Every positional permutation, preserving duplicate elements.
    Unordered(Vec<Self>),
}

fn product<S, A, O>(
    left: Vec<Alternative<S, A, O>>,
    right: &[Alternative<S, A, O>],
) -> Vec<Alternative<S, A, O>> {
    left.into_iter()
        .flat_map(|prefix| {
            right.iter().map(move |suffix| {
                let mut alternative = prefix.clone();
                alternative.nodes.extend(suffix.nodes.iter().cloned());
                alternative.choices.extend(suffix.choices.iter().cloned());
                alternative
            })
        })
        .collect()
}

impl<S, A, O> Pattern<S, A, O> {
    /// Wraps one state waypoint.
    pub fn state(waypoint: Waypoint<S>) -> Self {
        Self::State(Rc::new(waypoint))
    }
}

impl<S, A: 'static, O> Pattern<S, A, O> {
    /// Builds a typed action pattern.
    pub fn action(
        domain: ActionDomain<A>,
        outcome: Option<OutcomePredicate<O>>,
        label: Option<String>,
    ) -> Self
    where
        A: fmt::Display,
    {
        Self::Action(domain.described(), outcome, label)
    }

    /// Shares this action domain among borrowed outcome alternatives.
    ///
    /// # Panics
    /// Panics when attached to a non-action pattern.
    pub fn with_outcomes(self, outcomes: Vec<(OutcomePredicate<O>, String)>) -> Self {
        match self {
            Self::Action(domain, _, _) => Self::Either(
                outcomes
                    .into_iter()
                    .map(|(outcome, label)| {
                        Self::Action(domain.clone(), Some(outcome), Some(label))
                    })
                    .collect(),
            ),
            _ => panic!("an outcome follows one action"),
        }
    }

    /// Repeats this pattern an exact number of times with fresh choices.
    pub fn repeat_exact(self, count: usize) -> Self {
        Self::Repeat(Box::new(self), count, count)
    }

    fn size(&self, limit: usize) -> Result<(usize, usize), PatternError> {
        let add = |a: usize, b: usize| a.checked_add(b).ok_or(PatternError::ExpansionOverflow);
        let mul = |a: usize, b: usize| a.checked_mul(b).ok_or(PatternError::ExpansionOverflow);
        match self {
            Self::State(_) | Self::Gap(..) => Ok((1, 1)),
            Self::Action(domain, ..) => Ok((domain.count()?, 1)),
            Self::Sequence(parts) | Self::Unordered(parts) => {
                let sizes = parts.iter().map(|part| part.size(limit)).collect::<Vec<_>>();
                if sizes.iter().any(|size| matches!(size, Ok((0, _)))) {
                    return Ok((0, 0));
                }
                let (mut count, mut length) = (1, 0);
                for size in sizes {
                    let (n, len) = size?;
                    count = mul(count, n)?;
                    length = add(length, len)?;
                }
                if matches!(self, Self::Unordered(_)) {
                    let permutations = (1..=parts.len()).try_fold(1usize, mul);
                    if parts.len() > limit {
                        return Err(PatternError::PermutationLimit {
                            elements: parts.len(),
                            limit,
                            count: permutations.ok(),
                        });
                    }
                    count = mul(count, permutations?)?;
                }
                Ok((count, length))
            }
            Self::Either(parts) => {
                let (mut count, mut length) = (0, 0);
                for part in parts {
                    let (n, len) = part.size(limit)?;
                    count = add(count, n)?;
                    length = length.max(len);
                }
                Ok((count, length))
            }
            Self::Repeat(part, min, max) => {
                if min > max {
                    return Ok((0, 0));
                }
                if *max == 0 {
                    return Ok((1, 0));
                }
                let (n, length) = part.size(limit)?;
                if n == 0 {
                    return Ok((usize::from(*min == 0), 0));
                }
                let length = mul(length, *max)?;
                let count = if n == 1 {
                    add(max - min, 1)?
                } else {
                    let mut power = n
                        .checked_pow(
                            u32::try_from(*min).map_err(|_| PatternError::ExpansionOverflow)?,
                        )
                        .ok_or(PatternError::ExpansionOverflow)?;
                    let mut count = power;
                    for _ in *min..*max {
                        power = mul(power, n)?;
                        count = add(count, power)?;
                    }
                    count
                };
                Ok((count, length))
            }
        }
    }

    fn describe(&self) -> String {
        match self {
            Self::State(_) => "state".into(),
            Self::Gap(..) => "..".into(),
            Self::Action(domain, _, outcome) => {
                let mut label = match domain.count() {
                    Ok(0) => "empty action domain".into(),
                    Ok(1) => (domain.label)(&domain.get(0)),
                    Ok(count) => format!("one of {count} actions"),
                    Err(_) => "unexpanded action domain".into(),
                };
                if let Some(outcome) = outcome {
                    label.push_str(" => ");
                    label.push_str(outcome);
                }
                label
            }
            Self::Either(parts) => parts.iter().map(Self::describe).collect::<Vec<_>>().join(" | "),
            Self::Sequence(parts) | Self::Unordered(parts) => {
                parts.iter().map(Self::describe).collect::<Vec<_>>().join(", ")
            }
            Self::Repeat(part, min, max) => format!("[{}; {min}..={max}]", part.describe()),
        }
    }

    fn expand_repeat(part: &Self, min: usize, max: usize) -> Vec<Alternative<S, A, O>> {
        if max == 0 {
            return vec![Alternative {
                nodes: vec![],
                choices: vec![format!("[{}; 0]", part.describe())],
            }];
        }
        let choices = part.expand();
        if choices.len() == 1 {
            return (min..=max)
                .map(|count| {
                    let mut nodes = Vec::new();
                    if !choices[0].nodes.is_empty() {
                        for _ in 0..count {
                            nodes.extend(choices[0].nodes.iter().cloned());
                        }
                    }
                    Alternative { nodes, choices: vec![format!("[{}; {count}]", part.describe())] }
                })
                .collect();
        }
        if choices.is_empty() {
            return if min == 0 {
                vec![Alternative {
                    nodes: vec![],
                    choices: vec![format!("[{}; 0]", part.describe())],
                }]
            } else {
                vec![]
            };
        }
        let mut repeated = vec![Alternative::empty()];
        let mut result = Vec::new();
        for n in 0..=max {
            if n >= min {
                result.extend(repeated.iter().cloned().map(|mut alternative| {
                    alternative.choices.insert(0, format!("[{}; {n}]", part.describe()));
                    alternative
                }));
            }
            if n != max {
                repeated = product(repeated, &choices);
            }
        }
        result
    }

    fn expand(&self) -> Vec<Alternative<S, A, O>> {
        if self.size(usize::MAX).is_ok_and(|size| size.0 == 0) {
            return Vec::new();
        }
        match self {
            Self::State(state) => {
                vec![Alternative { nodes: vec![PathNode::State(state.clone())], choices: vec![] }]
            }
            Self::Gap(gap, exclude) => vec![Alternative {
                nodes: vec![PathNode::Gap { gap: *gap, exclude: exclude.clone() }],
                choices: vec![],
            }],
            Self::Action(domain, outcome, outcome_label) => (0..domain.count().unwrap_or(0))
                .map(|index| {
                    let action = domain.get(index);
                    let mut label = (domain.label)(&action);
                    if let Some(outcome) = outcome_label {
                        label.push_str(" => ");
                        label.push_str(outcome);
                    }
                    Alternative {
                        nodes: vec![PathNode::Action {
                            action: Rc::new(action),
                            outcome: outcome.clone(),
                            label: label.clone(),
                        }],
                        choices: vec![label],
                    }
                })
                .collect(),
            Self::Sequence(parts) => parts
                .iter()
                .fold(vec![Alternative::empty()], |left, part| product(left, &part.expand())),
            Self::Either(parts) => parts.iter().flat_map(Self::expand).collect(),
            Self::Repeat(part, min, max) => Self::expand_repeat(part, *min, *max),
            Self::Unordered(parts) => {
                fn permutations<S, A: 'static, O>(
                    parts: &[Pattern<S, A, O>],
                    order: &mut Vec<usize>,
                    used: &mut [bool],
                    result: &mut Vec<Alternative<S, A, O>>,
                ) {
                    if order.len() == parts.len() {
                        result.extend(
                            order.iter().fold(vec![Alternative::empty()], |left, index| {
                                product(left, &parts[*index].expand())
                            }),
                        );
                    } else {
                        for index in 0..parts.len() {
                            if !used[index] {
                                used[index] = true;
                                order.push(index);
                                permutations(parts, order, used, result);
                                order.pop();
                                used[index] = false;
                            }
                        }
                    }
                }
                let mut result = Vec::new();
                permutations(parts, &mut Vec::new(), &mut vec![false; parts.len()], &mut result);
                result
            }
        }
    }
}

/// Judges one state on a path.
pub enum Waypoint<S> {
    /// A judgment already made, such as a `bool` evaluated inside a Model method.
    Holds(bool),
    /// A predicate over the state, such as a closure in a test.
    Judged(Rc<dyn Fn(&S) -> bool>),
    /// Exactly this state.
    Is(S),
}

impl<S> Waypoint<S> {
    /// Wraps a predicate over the state.
    pub fn judged(predicate: impl Fn(&S) -> bool + 'static) -> Self {
        Self::Judged(Rc::new(predicate))
    }

    /// The path from this waypoint, over `gap`, to `next`.
    pub fn then<A, O>(self, gap: Gap, next: Self) -> Path<S, A, O> {
        Path::pattern(Pattern::Sequence(vec![
            Pattern::state(self),
            Pattern::Gap(gap, None),
            Pattern::state(next),
        ]))
    }

    /// The path from this waypoint, over `gap`, to a predicate; the predicate's parameter type
    /// is inferred from this waypoint.
    pub fn then_judged<A, O>(self, gap: Gap, next: impl Fn(&S) -> bool + 'static) -> Path<S, A, O> {
        self.then(gap, Waypoint::judged(next))
    }

    /// Whether `state` satisfies this waypoint.
    pub fn judge(&self, state: &S) -> bool
    where
        S: PartialEq,
    {
        match self {
            Self::Holds(holds) => *holds,
            Self::Judged(predicate) => predicate(state),
            Self::Is(expected) => expected == state,
        }
    }
}

impl<S: Clone> Clone for Waypoint<S> {
    fn clone(&self) -> Self {
        match self {
            Self::Holds(value) => Self::Holds(*value),
            Self::Judged(predicate) => Self::Judged(predicate.clone()),
            Self::Is(state) => Self::Is(state.clone()),
        }
    }
}

/// How many steps may separate two waypoints.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Gap {
    /// Any number of steps, written `..`.
    Any,
    /// At most this many steps, written `..=n`.
    AtMost(u32),
}

impl Gap {
    /// The step bound, when the gap has one.
    pub const fn bound(self) -> Option<u32> {
        match self {
            Self::Any => None,
            Self::AtMost(bound) => Some(bound),
        }
    }
}

/// Converts a `path!` waypoint; `Kind` only keeps the conversions apart.
pub trait IntoWaypoint<S, Kind> {
    /// The waypoint this value stands for.
    fn into_waypoint(self) -> Waypoint<S>;
}

#[doc(hidden)]
pub mod kind {
    pub struct Holds;
    pub struct Judged;
    pub struct Is;
}

impl<S> IntoWaypoint<S, kind::Holds> for bool {
    fn into_waypoint(self) -> Waypoint<S> {
        Waypoint::Holds(self)
    }
}

impl<S, F: Fn(&S) -> bool + 'static> IntoWaypoint<S, kind::Judged> for F {
    fn into_waypoint(self) -> Waypoint<S> {
        Waypoint::judged(self)
    }
}

impl<S> IntoWaypoint<S, kind::Is> for S {
    fn into_waypoint(self) -> Waypoint<S> {
        Waypoint::Is(self)
    }
}

const CLAIM_REFUSAL: &str = "a progress claim has one trigger and one target; ask a longer progression with find or Run::along";

/// A finite typed path expression, built with `path!`: one pattern and the limit on its
/// unordered groups.
#[must_use]
pub struct Path<S, A = (), O = ()> {
    pattern: Pattern<S, A, O>,
    permutation_limit: usize,
}

impl<S, A, O> Path<S, A, O> {
    /// Builds a path from a typed test-space expression.
    pub fn pattern(pattern: Pattern<S, A, O>) -> Self {
        Self { pattern, permutation_limit: 6 }
    }

    /// Raises or lowers the maximum number of elements in an unordered group.
    pub fn permutation_limit(mut self, limit: usize) -> Self {
        self.permutation_limit = limit;
        self
    }

    /// Appends a gap and the waypoint after it.
    pub fn then(self, gap: Gap, waypoint: Waypoint<S>) -> Self {
        let Self { pattern, permutation_limit } = self;
        let mut parts = match pattern {
            Pattern::Sequence(parts) => parts,
            pattern => vec![pattern],
        };
        parts.push(Pattern::Gap(gap, None));
        parts.push(Pattern::state(waypoint));
        Self { pattern: Pattern::Sequence(parts), permutation_limit }
    }

    /// Appends a gap and a predicate, its parameter type inferred from the path.
    pub fn then_judged(self, gap: Gap, next: impl Fn(&S) -> bool + 'static) -> Self {
        self.then(gap, Waypoint::judged(next))
    }

    /// Splits a claim's path into trigger, gap and target.
    ///
    /// # Panics
    ///
    /// A path other than two state waypoints around one ordinary gap is not a claim;
    /// `#[machine]` refuses a literal `path!` of another shape at compile time and this refuses
    /// one built elsewhere.
    pub fn claim(self) -> (Waypoint<S>, Gap, Waypoint<S>)
    where
        S: Clone,
    {
        let owned = |waypoint: Rc<Waypoint<S>>| {
            Rc::try_unwrap(waypoint).unwrap_or_else(|shared| (*shared).clone())
        };
        let Pattern::Sequence(parts) = self.pattern else { panic!("{CLAIM_REFUSAL}") };
        match <[_; 3]>::try_from(parts) {
            Ok([Pattern::State(trigger), Pattern::Gap(gap, None), Pattern::State(target)]) => {
                (owned(trigger), gap, owned(target))
            }
            _ => panic!("{CLAIM_REFUSAL}"),
        }
    }
}

impl<S, A: 'static, O> Path<S, A, O> {
    /// Counts every authored finite alternative without allocating its expansion.
    pub fn alternative_count(&self) -> Result<usize, PatternError> {
        self.pattern.size(self.permutation_limit).map(|size| size.0)
    }

    /// Expands alternatives after admitting their count and concrete storage.
    /// A conservative storage bound admits at most one million concrete nodes
    /// and alternative records combined,
    /// independently of the alternative budget.
    pub fn alternatives(&self, budget: usize) -> Result<Vec<Alternative<S, A, O>>, PatternError> {
        let limit = 1_000_000;
        let (count, length) = self.pattern.size(self.permutation_limit)?;
        if count > budget {
            return Err(PatternError::Budget { count, budget });
        }
        let nodes = length
            .checked_add(1)
            .and_then(|length| count.checked_mul(length))
            .ok_or(PatternError::ExpansionOverflow)?;
        if nodes > limit {
            return Err(PatternError::ExpansionTooLarge { nodes, limit });
        }
        Ok(self.pattern.expand())
    }
}

/// Constructs a typed path pattern with state checks, actions and optional outcomes.
/// `path![p, .., q]` is a state-only sequence; `..=n` bounds its intervening steps.
/// Tests may add `Actor::Action`, `Actor::Action { field: 0..=3, .. }`, `Actor::*`,
/// `Actor::Action => Outcome::Value`, `(a | b)`, `[a; n]`, `[a; min..=max]`,
/// `unordered![a, b]` and action-excluding gaps `..!(a, b)`.
/// Exact-only partial action fields use `..`; omitted fields then use declared input domains.
/// `#[progress]` admits exactly two state predicates separated by one ordinary gap.
#[macro_export]
macro_rules! path {
    () => { ::core::compile_error!("a path has two or more waypoints: write [p, .., q]") };
    (.. $($rest:tt)*) => { ::core::compile_error!("a path starts with a waypoint, not a gap: write [p, .., q]") };
    (..= $($rest:tt)*) => { ::core::compile_error!("a path starts with a waypoint, not a gap: write [p, .., q]") };
    ($($rest:tt)+) => { $crate::__path_parse!($($rest)+) };
}
