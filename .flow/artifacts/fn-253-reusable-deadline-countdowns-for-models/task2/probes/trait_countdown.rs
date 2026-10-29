pub trait Remaining: Copy + Eq {
    fn ticks(self) -> u8;
    fn from_ticks(ticks: u8) -> Self;
}
pub trait Configured: Copy {
    type Remaining: Remaining;
    fn ticks(self) -> u8;
    fn maximum() -> u8;
}
pub fn arm<C: Configured>(configured: Option<C>) -> Option<C::Remaining> {
    configured.map(|ticks| C::Remaining::from_ticks(ticks.ticks()))
}
pub fn tick<R: Remaining>(remaining: Option<R>) -> Option<R> {
    remaining.map(|ticks| R::from_ticks(if ticks.ticks() > 0 { ticks.ticks() - 1 } else { 0 }))
}
pub fn armed<R: Remaining>(remaining: Option<R>) -> bool { remaining.is_some() }
pub fn expired<R: Remaining>(remaining: Option<R>) -> bool {
    remaining.is_some_and(|ticks| ticks.ticks() == 0)
}
pub fn within_holds<C: Configured>(remaining: Option<C::Remaining>, configured: Option<C>) -> bool {
    match (remaining, configured) {
        (None, _) => true,
        (Some(left), Some(limit)) => 0 < left.ticks() && left.ticks() <= limit.ticks()
            && limit.ticks() <= C::maximum(),
        (Some(_), None) => false,
    }
}
pub fn descends_holds<R: Remaining>(after: Option<R>, before: Option<R>) -> bool {
    after == tick(before)
}

pub fn within<C: Configured>(remaining: Option<C::Remaining>, configured: Option<C>) {
    crate::check!(within_holds::<C>(remaining, configured));
}
pub fn descends<R: Remaining>(after: Option<R>, before: Option<R>) {
    crate::check!(descends_holds(after, before));
}
