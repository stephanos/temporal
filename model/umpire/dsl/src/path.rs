//! Patterns over paths through a Model: waypoints in order, separated by gaps.
//!
//! The pattern itself is neutral. A `#[progress]` claim reads it over every admitted path,
//! a search reads it as a request for one path.

/// Judges one state on a path.
pub enum Waypoint<S> {
    /// A judgment already made, such as a `bool` evaluated inside a Model method.
    Holds(bool),
    /// A predicate over the state, such as a closure in a test.
    Judged(Box<dyn Fn(&S) -> bool>),
    /// Exactly this state.
    Is(S),
}

impl<S> Waypoint<S> {
    /// Wraps a predicate over the state.
    pub fn judged(predicate: impl Fn(&S) -> bool + 'static) -> Self {
        Self::Judged(Box::new(predicate))
    }

    /// The path from this waypoint, over `gap`, to `next`.
    pub const fn then(self, gap: Gap, next: Self) -> Path<S> {
        Path { first: self, leg: (gap, next), more: Vec::new() }
    }

    /// The path from this waypoint, over `gap`, to a predicate; the predicate's parameter type
    /// is inferred from this waypoint.
    pub fn then_judged(self, gap: Gap, next: impl Fn(&S) -> bool + 'static) -> Path<S> {
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

/// Two or more waypoints in order, each pair separated by a gap; built with `path!`.
///
/// The first leg is stored inline so a two-waypoint claim evaluated at every state allocates
/// nothing beyond its closures.
#[must_use]
pub struct Path<S> {
    first: Waypoint<S>,
    leg: (Gap, Waypoint<S>),
    more: Vec<(Gap, Waypoint<S>)>,
}

impl<S> Path<S> {
    /// Appends a gap and the waypoint after it.
    pub fn then(mut self, gap: Gap, waypoint: Waypoint<S>) -> Self {
        self.more.push((gap, waypoint));
        self
    }

    /// Appends a gap and a predicate, its parameter type inferred from the path.
    pub fn then_judged(self, gap: Gap, next: impl Fn(&S) -> bool + 'static) -> Self {
        self.then(gap, Waypoint::judged(next))
    }

    /// The first waypoint.
    pub const fn first(&self) -> &Waypoint<S> {
        &self.first
    }

    /// Each later waypoint with the gap before it, in order.
    pub fn legs(&self) -> impl Iterator<Item = (Gap, &Waypoint<S>)> {
        std::iter::once(&self.leg).chain(&self.more).map(|(gap, waypoint)| (*gap, waypoint))
    }

    /// The number of waypoints, at least two.
    pub fn len(&self) -> usize {
        2 + self.more.len()
    }

    /// Always false: a path has two or more waypoints.
    pub const fn is_empty(&self) -> bool {
        false
    }

    /// Splits a claim's path into trigger, gap and target.
    ///
    /// # Panics
    ///
    /// A path with more than two waypoints is not a claim; `#[machine]` refuses a literal
    /// `path!` of that shape at compile time and this refuses one built elsewhere.
    pub fn claim(self) -> (Waypoint<S>, Gap, Waypoint<S>) {
        assert!(self.more.is_empty(), "{CLAIM_REFUSAL}");
        (self.first, self.leg.0, self.leg.1)
    }
}

/// A pattern over paths: `path![p, .., q]` is "p, then any number of steps, then q";
/// `path![p, ..=n, q]` allows at most `n` steps between them. Waypoints are `bool`s, closures
/// `|state| ..` or states; two or more, with a gap between each pair.
#[macro_export]
macro_rules! path {
    () => { ::core::compile_error!("a path has two or more waypoints: write [p, .., q]") };
    (.. $($rest:tt)*) => { ::core::compile_error!("a path starts with a waypoint, not a gap: write [p, .., q]") };
    (..= $($rest:tt)*) => { ::core::compile_error!("a path starts with a waypoint, not a gap: write [p, .., q]") };
    ($($rest:tt)+) => { $crate::__path!(@waypoint [] $($rest)+) };
}

#[doc(hidden)]
#[macro_export]
macro_rules! __path {
    // A waypoint is expected after `[]` (nothing yet) or `[{built} (gap)]`. A closure is passed
    // to `then_judged` directly so its parameter type is inferred from the waypoints before it.
    (@waypoint [$($acc:tt)*] .. $($rest:tt)*) => {
        ::core::compile_error!("write one gap between two waypoints: [p, .., q]")
    };
    (@waypoint [$($acc:tt)*] ..= $($rest:tt)*) => {
        ::core::compile_error!("write one gap between two waypoints: [p, .., q]")
    };
    (@waypoint [$($acc:tt)*]) => {
        ::core::compile_error!("a path ends with a waypoint, not a gap: write [p, .., q]")
    };
    // `move` is matched as the keyword: an identifier would read `a | b | c` as a closure.
    (@waypoint [$($acc:tt)*] move | $arg:ident $(: $ty:ty)? | $body:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@judged [$($acc)*] (move | $arg $(: $ty)? | $body) $($($rest)*)?)
    };
    (@waypoint [$($acc:tt)*] move | $arg:pat_param | $body:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@judged [$($acc)*] (move | $arg | $body) $($($rest)*)?)
    };
    (@waypoint [$($acc:tt)*] | $arg:ident $(: $ty:ty)? | $body:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@judged [$($acc)*] (| $arg $(: $ty)? | $body) $($($rest)*)?)
    };
    (@waypoint [$($acc:tt)*] | $arg:pat_param | $body:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@judged [$($acc)*] (| $arg | $body) $($($rest)*)?)
    };
    (@waypoint [] $waypoint:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@gap [first { $crate::IntoWaypoint::into_waypoint($waypoint) }] $($($rest)*)?)
    };
    (@waypoint [$kind:ident {$($built:tt)*} $gap:tt] $waypoint:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@gap [path {
            ($($built)*).then($gap, $crate::IntoWaypoint::into_waypoint($waypoint))
        }] $($($rest)*)?)
    };
    (@judged [] $closure:tt $($rest:tt)*) => {
        $crate::__path!(@gap [first { $crate::Waypoint::judged $closure }] $($rest)*)
    };
    (@judged [$kind:ident {$($built:tt)*} $gap:tt] ($($closure:tt)*) $($rest:tt)*) => {
        $crate::__path!(@gap [path { ($($built)*).then_judged($gap, $($closure)*) }] $($rest)*)
    };
    // A gap or the end is expected.
    (@gap [first $first:tt]) => {
        ::core::compile_error!("a path has two or more waypoints: write [p, .., q]")
    };
    (@gap [path {$($path:tt)*}]) => { $($path)* };
    (@gap [$kind:ident $built:tt] .. $(, $($rest:tt)*)?) => {
        $crate::__path!(@waypoint [$kind $built ($crate::Gap::Any)] $($($rest)*)?)
    };
    (@gap [$kind:ident $built:tt] ..= $bound:expr $(, $($rest:tt)*)?) => {
        $crate::__path!(@waypoint [$kind $built ($crate::Gap::AtMost($bound))] $($($rest)*)?)
    };
    (@gap [$($acc:tt)*] $($rest:tt)*) => {
        ::core::compile_error!("write [p, ..=1, q] for at most one step, or [p, .., q]")
    };
}
