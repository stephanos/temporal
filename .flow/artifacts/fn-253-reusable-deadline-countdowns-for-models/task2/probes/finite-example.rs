use umpire_dsl::Finite;
pub use umpire_dsl::check;
#[path = "shared-types.rs"]
mod shared;
#[path = "trait_countdown.rs"]
mod countdown;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Finite)]
struct Ticks(#[inputs(1..=3)] u8);
#[derive(Clone, Copy, Debug, PartialEq, Eq, Finite)]
struct Remaining { #[inputs(0..=3)] ticks: u8 }
impl countdown::Configured for Ticks {
    type Remaining = Remaining;
    fn ticks(self) -> u8 { self.0 }
    fn maximum() -> u8 { 3 }
}
impl countdown::Remaining for Remaining {
    fn ticks(self) -> u8 { self.ticks }
    fn from_ticks(ticks: u8) -> Self { Self { ticks } }
}
fn main() {
    assert_eq!(shared::Ticks::<3>::domain().map(|v| v.0).collect::<Vec<_>>(), [1, 2, 3]);
    assert_eq!(shared::Remaining::<3>::domain().map(|v| v.ticks).collect::<Vec<_>>(), [0, 1, 2, 3]);
    assert_eq!(shared::Ticks::<0>::domain().count(), 0);
    assert_eq!(shared::Remaining::<0>::domain().count(), 1);
    assert_eq!(shared::Remaining::<255>::domain().count(), 256);
    assert_eq!(Ticks::domain().count(), 3);
    assert_eq!(Remaining::domain().count(), 4);
    assert_eq!(countdown::arm::<Ticks>(None), None);
    assert!(!countdown::armed::<Remaining>(None));
    assert!(!countdown::expired::<Remaining>(None));
    assert!(countdown::within_holds::<Ticks>(None, None));
    assert!(countdown::descends_holds::<Remaining>(None, None));
    for remaining in 0..=4 {
        let left = Some(Remaining { ticks: remaining });
        assert_eq!(countdown::tick(left), Some(Remaining { ticks: remaining.saturating_sub(1) }));
        assert_eq!(countdown::expired(left), remaining == 0);
        assert!(countdown::armed(left));
        assert!(!countdown::within_holds::<Ticks>(left, None));
        for configured in 0..=4 {
            let ticks = Some(Ticks(configured));
            assert_eq!(countdown::arm(ticks), Some(Remaining { ticks: configured }));
            assert_eq!(countdown::within_holds::<Ticks>(left, ticks),
                remaining > 0 && remaining <= configured && configured <= 3);
            assert!(countdown::descends_holds(countdown::tick(left), left));
        }
    }
    use umpire_dsl::{countdown_proposed as candidate, countdown_types as types, claim, Verdict};
    assert_eq!(candidate::arm::<3>(None), None);
    assert_eq!(candidate::tick::<3>(None), None);
    assert!(!candidate::expired::<3>(None));
    assert!(!candidate::armed::<3>(None));
    assert_eq!(claim::evaluate(|| candidate::within::<0>(None, None)).verdict, Verdict::Fails);
    assert_eq!(claim::evaluate(|| candidate::within::<3>(None, None)).verdict, Verdict::Holds);
    assert_eq!(claim::evaluate(|| candidate::descends::<3>(None, None)).verdict, Verdict::Holds);
    assert_eq!(claim::evaluate(|| candidate::descends(
        Some(types::Remaining::<3> { ticks: 0 }), None)).verdict, Verdict::Fails);
    for remaining in 0..=4u8 {
        let before = Some(types::Remaining::<3> { ticks: remaining });
        let after = candidate::tick(before);
        assert_eq!(claim::evaluate(|| candidate::within(before, None)).verdict, Verdict::Fails);
        assert_eq!(after, Some(types::Remaining { ticks: remaining.saturating_sub(1) }));
        assert_eq!(candidate::expired(before), remaining == 0);
        assert!(candidate::armed(before));
        assert_eq!(claim::evaluate(|| candidate::descends(after, before)).verdict, Verdict::Holds);
        assert_eq!(claim::evaluate(|| candidate::descends(None, before)).verdict, Verdict::Fails);
        for configured in 0..=4u8 {
            let ticks = Some(types::Ticks::<3>(configured));
            assert_eq!(candidate::arm(ticks), Some(types::Remaining { ticks: configured }));
            let expected = if configured > 0 && configured <= 3 && remaining > 0 && remaining <= configured {
                Verdict::Holds
            } else { Verdict::Fails };
            assert_eq!(claim::evaluate(|| candidate::within(before, ticks)).verdict, expected);
            assert_eq!(claim::evaluate(|| candidate::within(None, ticks)).verdict,
                if configured > 0 && configured <= 3 { Verdict::Holds } else { Verdict::Fails });
        }
    }
    assert_eq!(candidate::tick(Some(types::Remaining::<255> { ticks: 255 })),
        Some(types::Remaining { ticks: 254 }));
    assert_eq!(candidate::arm(Some(types::Ticks::<255>(255))),
        Some(types::Remaining { ticks: 255 }));
    println!("finite domains and generic/trait native boundary controls passed");
}
