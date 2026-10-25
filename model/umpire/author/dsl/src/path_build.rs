//! Trait-directed construction preserving state and actor inference in path macros.

use std::{cell::RefCell, rc::Rc};

use crate::{
    ActionDomain, Actor, ActorValue, Associated as Registered, Gap, IntoWaypoint, Machine, Path,
    Pattern, RegisteredActor, Waypoint,
};

pub struct Probe<T>(RefCell<Option<T>>);
impl<T> Probe<T> {
    pub fn new(value: T) -> Self {
        Self(RefCell::new(Some(value)))
    }
    fn take(&self) -> T {
        self.0.borrow_mut().take().unwrap_or_else(|| panic!("path value consumed twice"))
    }
}

/// A path of state waypoints so far; its action and outcome types are fixed by its use, or by
/// the first structured node that needs the world's.
#[must_use]
pub struct StateBuilder<S, A = (), O = ()> {
    parts: Vec<Pattern<S, A, O>>,
    gap: Gap,
}
impl<S, A, O> StateBuilder<S, A, O> {
    fn new(first: Waypoint<S>) -> Self {
        Self { parts: vec![Pattern::State(Rc::new(first))], gap: Gap::AtMost(0) }
    }
    pub fn gap(mut self, gap: Gap) -> Self {
        self.gap = gap;
        self
    }
    fn append(mut self, waypoint: Waypoint<S>) -> Self {
        self.parts.push(Pattern::Gap(self.gap, None));
        self.parts.push(Pattern::State(Rc::new(waypoint)));
        self.gap = Gap::AtMost(0);
        self
    }
    pub fn then_judged(self, predicate: impl Fn(&S) -> bool + 'static) -> Self {
        self.append(Waypoint::judged(predicate))
    }
    pub fn finish(self) -> Path<S, A, O> {
        assert!(self.parts.len() > 1, "a path has two or more waypoints");
        Path::pattern(Pattern::Sequence(self.parts))
    }
}
impl<S: Registered> StateBuilder<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>
where
    <S::World as Machine>::Action: 'static,
{
    pub fn structure(
        self,
        pattern: Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
    ) -> WorldBuilder<S> {
        let gap = self.gap;
        let mut builder = self.world();
        builder.parts.push(Pattern::Gap(gap, None));
        builder.parts.push(pattern);
        builder
    }
    pub fn structure_with(
        self,
        make: impl FnOnce(PatternContext<S>) -> WorldPattern<S>,
    ) -> WorldBuilder<S> {
        self.structure(make(PatternContext(std::marker::PhantomData)))
    }
    fn world(self) -> WorldBuilder<S> {
        WorldBuilder { parts: self.parts, gap: self.gap, pending: false }
    }
}

#[must_use]
pub struct WorldBuilder<S: Registered> {
    parts: Vec<Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>>,
    gap: Gap,
    pending: bool,
}
impl<S: Registered> WorldBuilder<S>
where
    <S::World as Machine>::Action: 'static,
{
    pub fn new(
        pattern: Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
    ) -> Self {
        Self { parts: vec![pattern], gap: Gap::AtMost(0), pending: false }
    }
    pub fn gap(mut self, gap: Gap) -> Self {
        self.gap = gap;
        self.pending = true;
        self
    }
    pub fn structure(
        mut self,
        pattern: Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
    ) -> Self {
        if self.pending {
            self.parts.push(Pattern::Gap(self.gap, None));
            self.pending = false;
        }
        self.parts.push(pattern);
        self
    }
    pub fn structure_with(self, make: impl FnOnce(PatternContext<S>) -> WorldPattern<S>) -> Self {
        self.structure(make(PatternContext(std::marker::PhantomData)))
    }
    pub fn then_judged(self, predicate: impl Fn(&S) -> bool + 'static) -> Self {
        self.structure(Pattern::state(Waypoint::judged(predicate)))
    }
    pub fn finish(self) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        Path::pattern(Pattern::Sequence(self.parts))
    }
}

pub struct ActionKind;
pub fn outcome_action<S: Registered, T: Actor + 'static>(value: T) -> WorldPattern<S>
where
    <S::World as Machine>::Action: From<T> + 'static,
{
    Pattern::action(ActionDomain::new(Ok(1), move |_| value.clone().into()), None, None)
}
pub struct StateFactory<S, K = crate::waypoint_kind::Judged>(
    Waypoint<S>,
    std::marker::PhantomData<K>,
);
impl<S> StateFactory<S> {
    pub fn judged(predicate: impl Fn(&S) -> bool + 'static) -> Self {
        Self(Waypoint::judged(predicate), std::marker::PhantomData)
    }
}
impl<S, K> StateFactory<S, K> {
    pub fn build<A, O, B: Finish>(
        self,
        rest: impl FnOnce(StateBuilder<S, A, O>) -> B,
    ) -> B::Output {
        rest(StateBuilder::new(self.0)).complete()
    }
    pub fn piece(self) -> PatternPiece<S, K::Hint>
    where
        S: Registered,
        K: WaypointHint,
        <S::World as Machine>::Action: 'static,
    {
        PatternPiece::one(Pattern::state(self.0))
    }
}
pub struct ActorFactory<T>(T);
pub fn actor_factory<T: ActorValue>(value: T) -> ActorFactory<T> {
    ActorFactory(value)
}

/// A World-dependent actor's or variant's domain requested without its World.
pub struct Unbound<A>(std::marker::PhantomData<A>);
impl<A> Unbound<A> {
    #[allow(clippy::new_without_default)]
    pub fn new() -> Self {
        Self(std::marker::PhantomData)
    }
    pub fn map<B, F: BindWorld<A, B>>(self, map: F) -> ActionDomain<B> {
        map.bind()
    }
}

/// Never implemented: refuses a World-dependent wildcard that names no World.
#[diagnostic::on_unimplemented(
    message = "`{A}` has World-dependent inputs and no static action domain",
    label = "this pattern names no World",
    note = "build the domain from the selected World, such as `Pattern::action({A}::actions_in(&world).map(Into::into), None, None)`"
)]
pub trait BindWorld<A, B> {
    fn bind(self) -> ActionDomain<B>;
}

/// A static action domain; a World-dependent domain requested without its World has none.
#[diagnostic::on_unimplemented(
    message = "`{Self}` names a World-dependent domain without its World",
    label = "this pattern names no World",
    note = "build the domain from the selected World, such as `Pattern::action(Actor::actions_in(&world).map(Into::into), None, None)`"
)]
pub trait StaticDomain {
    type Action;
    fn domain(self) -> ActionDomain<Self::Action>;
}
impl<T> StaticDomain for ActionDomain<T> {
    type Action = T;
    fn domain(self) -> ActionDomain<T> {
        self
    }
}

pub struct DomainFactory<T>(ActionDomain<T>);
impl<T: ActorValue + 'static> DomainFactory<T> {
    pub fn new(domain: impl StaticDomain<Action = T>) -> Self {
        Self(domain.domain())
    }
    pub fn piece<S: Registered>(self) -> PatternPiece<S, OwnerHint<T>>
    where
        <S::World as Machine>::Action: From<T> + 'static,
    {
        PatternPiece::one(Pattern::action(self.0.map(Into::into), None, None))
    }
}
pub type WorldPath<S> = Path<
    S,
    <<S as Registered>::World as Machine>::Action,
    <<S as Registered>::World as Machine>::Outcome,
>;
pub type WorldPattern<S> = Pattern<
    S,
    <<S as Registered>::World as Machine>::Action,
    <<S as Registered>::World as Machine>::Outcome,
>;

pub struct Unknown;
pub struct Known;
pub struct OwnerHint<T>(std::marker::PhantomData<T>);
pub trait WaypointHint {
    type Hint;
}
impl WaypointHint for crate::waypoint_kind::Holds {
    type Hint = Unknown;
}
impl WaypointHint for crate::waypoint_kind::Is {
    type Hint = Known;
}
impl WaypointHint for crate::waypoint_kind::Judged {
    type Hint = Known;
}
pub trait MergeHint<Rhs> {
    type Hint;
}
impl<Rhs> MergeHint<Rhs> for Known {
    type Hint = Known;
}
impl MergeHint<Known> for Unknown {
    type Hint = Known;
}
impl<T> MergeHint<Known> for OwnerHint<T> {
    type Hint = Known;
}
impl MergeHint<Unknown> for Unknown {
    type Hint = Unknown;
}
impl<T> MergeHint<OwnerHint<T>> for Unknown {
    type Hint = OwnerHint<T>;
}
impl<T> MergeHint<Unknown> for OwnerHint<T> {
    type Hint = OwnerHint<T>;
}
impl<T, U> MergeHint<OwnerHint<U>> for OwnerHint<T> {
    type Hint = OwnerHint<T>;
}
pub trait ResolveHint<S> {}
impl<S> ResolveHint<S> for Known {}
impl<S: Registered, T: RegisteredActor<S>> ResolveHint<S> for OwnerHint<T> {}

#[must_use]
pub struct PatternPiece<S: Registered, H> {
    parts: Vec<WorldPattern<S>>,
    hint: std::marker::PhantomData<H>,
}
impl<S: Registered> PatternPiece<S, Unknown> {
    pub fn empty() -> Self {
        Self { parts: Vec::new(), hint: std::marker::PhantomData }
    }
}
impl<S: Registered, H> PatternPiece<S, H> {
    fn one(pattern: WorldPattern<S>) -> Self {
        Self { parts: vec![pattern], hint: std::marker::PhantomData }
    }
    pub fn join<J>(mut self, other: PatternPiece<S, J>) -> PatternPiece<S, H::Hint>
    where
        H: MergeHint<J>,
    {
        self.parts.extend(other.parts);
        PatternPiece { parts: self.parts, hint: std::marker::PhantomData }
    }
    pub fn push(mut self, pattern: WorldPattern<S>) -> Self {
        self.parts.push(pattern);
        self
    }
    pub fn group<J>(
        mut self,
        make: impl FnOnce(PatternPiece<S, H>) -> PatternPiece<S, J>,
    ) -> PatternPiece<S, J> {
        let child = make(PatternPiece { parts: Vec::new(), hint: std::marker::PhantomData });
        self.parts.extend(child.parts);
        PatternPiece { parts: self.parts, hint: std::marker::PhantomData }
    }
    pub fn wrap(self, wrap: impl FnOnce(Vec<WorldPattern<S>>) -> WorldPattern<S>) -> Self {
        Self::one(wrap(self.parts))
    }
    pub fn resolve(self) -> PatternPiece<S, Known>
    where
        H: ResolveHint<S>,
    {
        PatternPiece { parts: self.parts, hint: std::marker::PhantomData }
    }
    pub fn finish(self) -> WorldPath<S> {
        Path::pattern(Pattern::Sequence(self.parts))
    }
    pub fn judged(self, predicate: impl Fn(&S) -> bool + 'static) -> Self
    where
        <S::World as Machine>::Action: 'static,
    {
        self.push(Pattern::state(Waypoint::judged(predicate)))
    }
}
pub struct PatternContext<S>(std::marker::PhantomData<S>);
impl<S: Registered> PatternContext<S>
where
    <S::World as Machine>::Action: 'static,
{
    pub fn judged(&self, predicate: impl Fn(&S) -> bool + 'static) -> WorldPattern<S> {
        Pattern::state(Waypoint::judged(predicate))
    }
}
pub fn pattern_with<S: Registered>(
    make: impl FnOnce(PatternContext<S>) -> WorldPattern<S>,
) -> WorldPattern<S> {
    make(PatternContext(std::marker::PhantomData))
}
impl<T: ActorValue + 'static> ActorFactory<T> {
    pub fn piece<S: Registered>(self) -> PatternPiece<S, OwnerHint<T>>
    where
        <S::World as Machine>::Action: From<T> + 'static,
    {
        DomainFactory::new(ActionDomain::new(Ok(1), move |_| self.0.clone())).piece()
    }
    pub fn build<S: Registered>(
        self,
        rest: impl FnOnce(WorldBuilder<S>) -> WorldBuilder<S>,
    ) -> WorldPath<S>
    where
        T: RegisteredActor<S>,
        <S::World as Machine>::Action: From<T> + 'static,
    {
        rest(WorldBuilder::new(Pattern::action(
            ActionDomain::new(Ok(1), move |_| self.0.clone().into()),
            None,
            None,
        )))
        .finish()
    }
}
pub trait Factory<S, Kind> {
    type Factory;
    fn factory(self) -> Self::Factory;
}
impl<S, T, K> Factory<S, K> for &&Probe<T>
where
    T: IntoWaypoint<S, K>,
{
    type Factory = StateFactory<S, K>;
    fn factory(self) -> Self::Factory {
        StateFactory(self.take().into_waypoint(), std::marker::PhantomData)
    }
}
pub trait ActorFactoryValue {
    type Factory;
    fn factory(self) -> Self::Factory;
}
impl<T: ActorValue> ActorFactoryValue for &Probe<T> {
    type Factory = ActorFactory<T>;
    fn factory(self) -> Self::Factory {
        ActorFactory(self.take())
    }
}
pub trait Finish {
    type Output;
    fn complete(self) -> Self::Output;
}
impl<S, A, O> Finish for StateBuilder<S, A, O> {
    type Output = Path<S, A, O>;
    fn complete(self) -> Self::Output {
        self.finish()
    }
}
impl<S: Registered> Finish for WorldBuilder<S>
where
    <S::World as Machine>::Action: 'static,
{
    type Output = Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>;
    fn complete(self) -> Self::Output {
        self.finish()
    }
}
pub trait PatternValue<S: Registered, Kind> {
    fn node(self) -> Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>;
}
impl<S: Registered, T, K> PatternValue<S, K> for &&Probe<T>
where
    T: IntoWaypoint<S, K>,
    <S::World as Machine>::Action: 'static,
{
    fn node(self) -> Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        Pattern::state(self.take().into_waypoint())
    }
}
impl<S: Registered, T: ActorValue + 'static> PatternValue<S, ActionKind> for &Probe<T>
where
    <S::World as Machine>::Action: From<T> + 'static,
{
    fn node(self) -> Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        let value = self.take();
        Pattern::action(ActionDomain::new(Ok(1), move |_| value.clone().into()), None, None)
    }
}

pub trait AppendPath<B, Kind> {
    type Builder;
    fn append(self, builder: B) -> Self::Builder;
}
impl<S, A, O, T, K> AppendPath<StateBuilder<S, A, O>, K> for &&Probe<T>
where
    T: IntoWaypoint<S, K>,
{
    type Builder = StateBuilder<S, A, O>;
    fn append(self, builder: StateBuilder<S, A, O>) -> Self::Builder {
        builder.append(self.take().into_waypoint())
    }
}
impl<S: Registered, T: ActorValue + 'static>
    AppendPath<
        StateBuilder<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
        ActionKind,
    > for &Probe<T>
where
    <S::World as Machine>::Action: From<T> + 'static,
{
    type Builder = WorldBuilder<S>;
    fn append(
        self,
        builder: StateBuilder<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
    ) -> Self::Builder {
        let value = self.take();
        let gap = builder.gap;
        builder.world().gap(gap).structure(Pattern::action(
            ActionDomain::new(Ok(1), move |_| value.clone().into()),
            None,
            None,
        ))
    }
}
impl<S: Registered, T, K> AppendPath<WorldBuilder<S>, K> for &&Probe<T>
where
    T: IntoWaypoint<S, K>,
    <S::World as Machine>::Action: 'static,
{
    type Builder = WorldBuilder<S>;
    fn append(self, builder: WorldBuilder<S>) -> Self::Builder {
        builder.structure(Pattern::state(self.take().into_waypoint()))
    }
}
impl<S: Registered, T: ActorValue + 'static> AppendPath<WorldBuilder<S>, ActionKind> for &Probe<T>
where
    <S::World as Machine>::Action: From<T> + 'static,
{
    type Builder = WorldBuilder<S>;
    fn append(self, builder: WorldBuilder<S>) -> Self::Builder {
        let value = self.take();
        builder.structure(Pattern::action(
            ActionDomain::new(Ok(1), move |_| value.clone().into()),
            None,
            None,
        ))
    }
}

/// An input-bearing actor's per-variant lazy domains for catalog admission, built from the
/// selected World. Static actors ignore it; World-dependent actors implement it for one World.
pub trait ActorCatalog<W>: Sized {
    fn catalog(world: &W, retain: usize) -> Vec<(String, ActionDomain<Self>)>;
    /// The unchecked catalog in declaration order, for `Machine::actions`.
    fn actions(world: &W) -> Vec<Self>;
}
