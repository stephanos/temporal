#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Ticks<const N: u8>(pub u8);
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Remaining<const N: u8>(pub u8);

pub fn arm<const N: u8>(configured: Option<Ticks<N>>) -> Option<Remaining<N>> {
    configured.map(|ticks| Remaining(ticks.0))
}
pub fn tick<const N: u8>(remaining: Option<Remaining<N>>) -> Option<Remaining<N>> {
    remaining.map(|ticks| Remaining(if ticks.0 > 0 { ticks.0 - 1 } else { 0 }))
}
pub fn armed<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining.is_some()
}
pub fn expired<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining == Some(Remaining(0))
}
pub fn within_holds<const N: u8>(remaining: Option<Remaining<N>>, configured: Option<Ticks<N>>) -> bool {
    match (remaining, configured) {
        (None, _) => true,
        (Some(left), Some(limit)) => 0 < left.0 && left.0 <= limit.0 && limit.0 <= N,
        (Some(_), None) => false,
    }
}
pub fn descends_holds<const N: u8>(after: Option<Remaining<N>>, before: Option<Remaining<N>>) -> bool {
    after == tick(before)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn absent_zero_and_positive_are_distinct() {
        assert_eq!(tick::<3>(None), None);
        assert_eq!(tick(Some(Remaining::<3>(0))), Some(Remaining(0)));
        assert!(!expired::<3>(None));
        assert!(expired(Some(Remaining::<3>(0))));
        for n in 1..=3 {
            assert_eq!(arm(Some(Ticks::<3>(n))), Some(Remaining(n)));
            assert_eq!(tick(Some(Remaining::<3>(n))), Some(Remaining(n - 1)));
        }
        assert!(!within_holds(Some(Remaining::<3>(0)), Some(Ticks(1))));
        assert!(!within_holds(Some(Remaining::<3>(1)), None));
        assert!(!within_holds(Some(Remaining::<3>(1)), Some(Ticks(4))));
    }
}
