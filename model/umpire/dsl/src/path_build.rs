//! Trait-directed construction preserving state and actor inference in path macros.

use std::cell::RefCell;

use crate::{
    ActionDomain, Actor, Gap, IntoWaypoint, Machine, Path, Pattern, Registered, RegisteredActor,
    Waypoint,
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

#[must_use]
pub struct StateBuilder<S> {
    first: Option<Waypoint<S>>,
    path: Option<Path<S>>,
    gap: Gap,
}
impl<S> StateBuilder<S> {
    fn new(first: Waypoint<S>) -> Self {
        Self { first: Some(first), path: None, gap: Gap::AtMost(0) }
    }
    pub fn gap(mut self, gap: Gap) -> Self {
        self.gap = gap;
        self
    }
    fn append(mut self, waypoint: Waypoint<S>) -> Self {
        self.path = Some(if let Some(path) = self.path {
            path.then(self.gap, waypoint)
        } else {
            self.first
                .take()
                .unwrap_or_else(|| panic!("missing path start"))
                .then(self.gap, waypoint)
        });
        self.gap = Gap::AtMost(0);
        self
    }
    pub fn then_judged(self, predicate: impl Fn(&S) -> bool + 'static) -> Self {
        self.append(Waypoint::judged(predicate))
    }
    pub fn finish(self) -> Path<S> {
        self.path.unwrap_or_else(|| panic!("a path has two or more waypoints"))
    }
    pub fn structure(
        self,
        pattern: Pattern<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
    ) -> WorldBuilder<S>
    where
        S: Registered,
        <S::World as Machine>::Action: 'static,
    {
        let gap = self.gap;
        let mut builder = self.world();
        builder.parts.push(Pattern::Gap(gap, None));
        builder.parts.push(pattern);
        builder
    }
    pub fn structure_with(
        self,
        make: impl FnOnce(PatternContext<S>) -> WorldPattern<S>,
    ) -> WorldBuilder<S>
    where
        S: Registered,
        <S::World as Machine>::Action: 'static,
    {
        self.structure(make(PatternContext(std::marker::PhantomData)))
    }
    fn world(self) -> WorldBuilder<S>
    where
        S: Registered,
        <S::World as Machine>::Action: 'static,
    {
        let parts = if let Some(path) = self.path {
            vec![path.into_world::<S::World>().into_pattern()]
        } else {
            vec![Pattern::state(self.first.unwrap_or_else(|| panic!("missing path start")))]
        };
        WorldBuilder { parts, gap: self.gap, pending: false }
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
pub struct StateFactory<S>(Waypoint<S>);
impl<S> StateFactory<S> {
    pub fn judged(predicate: impl Fn(&S) -> bool + 'static) -> Self {
        Self(Waypoint::judged(predicate))
    }
    pub fn build<B: Finish>(self, rest: impl FnOnce(StateBuilder<S>) -> B) -> B::Output {
        rest(StateBuilder::new(self.0)).complete()
    }
}
pub struct ActorFactory<T>(T);
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
impl<T: Actor + 'static> ActorFactory<T> {
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
    type Factory = StateFactory<S>;
    fn factory(self) -> Self::Factory {
        StateFactory(self.take().into_waypoint())
    }
}
pub trait ActorFactoryValue {
    type Factory;
    fn factory(self) -> Self::Factory;
}
impl<T: Actor> ActorFactoryValue for &Probe<T> {
    type Factory = ActorFactory<T>;
    fn factory(self) -> Self::Factory {
        ActorFactory(self.take())
    }
}
pub trait Finish {
    type Output;
    fn complete(self) -> Self::Output;
}
impl<S> Finish for StateBuilder<S> {
    type Output = Path<S>;
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
impl<S: Registered, T: Actor + 'static> PatternValue<S, ActionKind> for &Probe<T>
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
impl<S, T, K> AppendPath<StateBuilder<S>, K> for &&Probe<T>
where
    T: IntoWaypoint<S, K>,
{
    type Builder = StateBuilder<S>;
    fn append(self, builder: StateBuilder<S>) -> Self::Builder {
        builder.append(self.take().into_waypoint())
    }
}
impl<S: Registered, T: Actor + 'static> AppendPath<StateBuilder<S>, ActionKind> for &Probe<T>
where
    <S::World as Machine>::Action: From<T> + 'static,
{
    type Builder = WorldBuilder<S>;
    fn append(self, builder: StateBuilder<S>) -> Self::Builder {
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
impl<S: Registered, T: Actor + 'static> AppendPath<WorldBuilder<S>, ActionKind> for &Probe<T>
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
