//! Pure finite countdowns and delegating laws evaluated in a claim frame.
//!
//! Models select a common maximum `N` in `1..=255`. Public fields allow raw values
//! outside the finite domains; enumeration and value functions do not validate or
//! sanitize those values. The laws state only the checks documented below.

use crate::{Finite, check};

struct NominalName {
    bytes: [u8; 37],
    len: usize,
}

impl NominalName {
    const fn new(prefix: &str, bound: u8) -> Self {
        let mut name = Self { bytes: [0; 37], len: 0 };
        let prefix = prefix.as_bytes();
        while name.len < prefix.len() {
            name.bytes[name.len] = prefix[name.len];
            name.len += 1;
        }
        name.bytes[name.len] = b'<';
        name.len += 1;
        if bound >= 100 {
            name.bytes[name.len] = b'0' + bound / 100;
            name.len += 1;
        }
        if bound >= 10 {
            name.bytes[name.len] = b'0' + (bound / 10) % 10;
            name.len += 1;
        }
        name.bytes[name.len] = b'0' + bound % 10;
        name.bytes[name.len + 1] = b'>';
        name.len += 2;
        name
    }

    fn as_str(&'static self) -> &'static str {
        std::str::from_utf8(&self.bytes[..self.len])
            .unwrap_or_else(|error| panic!("invalid nominal type name: {error}"))
    }
}

/// A configured duration with finite domain `1..=N`.
///
/// Use `None` for absence. The public field permits forged zero or out-of-range values.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct Ticks<const N: u8>(pub u8);

impl<const N: u8> crate::IntoValue for Ticks<N> {
    fn value(&self) -> crate::Value {
        crate::Value::Bounded { name: Self::NAME.as_str(), raw: i128::from(self.0) }
    }
}

impl<const N: u8> Ticks<N> {
    const NAME: NominalName = NominalName::new("umpire_dsl::countdown::Ticks", N);

    /// Reports membership in the positive configured domain `1..=N`.
    pub fn contains(value: u8) -> bool {
        N > 0 && value > 0 && value <= N
    }
}

impl<const N: u8> Finite for Ticks<N> {
    fn domain() -> impl Iterator<Item = Self> {
        (1..=N).map(Self)
    }
}

/// An armed countdown with finite domain `0..=N`, distinct from absence.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct Remaining<const N: u8> {
    /// The remaining budget; zero is armed and expired.
    pub ticks: u8,
}

impl<const N: u8> crate::IntoValue for Remaining<N> {
    fn value(&self) -> crate::Value {
        crate::Value::Bounded { name: Self::NAME.as_str(), raw: i128::from(self.ticks) }
    }
}

impl<const N: u8> Remaining<N> {
    const NAME: NominalName = NominalName::new("umpire_dsl::countdown::Remaining", N);

    /// Reports membership in the remaining domain `0..=N`.
    pub fn contains(value: u8) -> bool {
        value <= N
    }
}

impl<const N: u8> Finite for Remaining<N> {
    fn domain() -> impl Iterator<Item = Self> {
        (0..=N).map(|ticks| Self { ticks })
    }
}

/// Copies a present configuration into an armed countdown, preserving absence.
///
/// This does not decide which transition may arm or replenish a countdown.
pub fn arm<const N: u8>(configured: Option<Ticks<N>>) -> Option<Remaining<N>> {
    if configured.is_some() {
        Some(Remaining { ticks: configured.unwrap_or(Ticks(1)).0 })
    } else {
        None
    }
}

/// Subtracts one from a positive countdown, preserving zero and absence.
///
/// This does not clear expiry or prescribe an expiry action.
pub fn tick<const N: u8>(remaining: Option<Remaining<N>>) -> Option<Remaining<N>> {
    if remaining.is_some() && remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0 {
        Some(Remaining { ticks: remaining.unwrap_or(Remaining { ticks: 0 }).ticks - 1 })
    } else {
        remaining
    }
}

/// Reports presence, including zero; this does not promise a positive budget.
pub fn armed<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining.is_some()
}

/// Reports armed zero; absence is not expired.
///
/// This does not impose priority among concurrently expired countdowns.
pub fn expired<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining.is_some() && remaining.unwrap_or(Remaining { ticks: 0 }).ticks == 0
}

/// Checks the bound, every present configuration and every live remaining budget.
///
/// Armed budgets must be positive and within a present configuration, because this
/// law assumes expiry clears in the same step. It permits absent remaining budgets
/// with absent or valid configurations. It does not constrain arming, replenishment,
/// wall time, fairness or expiry priority, and rejects Models retaining armed zero.
pub fn within<const N: u8>(remaining: Option<Remaining<N>>, configured: Option<Ticks<N>>) {
    check!(N > 0);
    check!(
        configured.is_none()
            || (configured.unwrap_or(Ticks(1)).0 > 0 && configured.unwrap_or(Ticks(1)).0 <= N)
    );
    check!(
        remaining.is_none()
            || (configured.is_some()
                && remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                && remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    <= configured.unwrap_or(Ticks(1)).0)
    );
}

/// Checks exactly one saturating tick, preserving absence and armed zero.
///
/// The Model selects the transitions subject to this law. It does not cover retries,
/// reset, terminal clearing, range validity, time, fairness or timeout priority.
pub fn descends<const N: u8>(after: Option<Remaining<N>>, before: Option<Remaining<N>>) {
    check!(after == tick(before));
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Verdict, claim};

    #[test]
    fn nominal_names_preserve_literal_bounds_at_every_decimal_width() {
        fn exported<const N: u8>() {
            use crate::IntoValue;
            assert_eq!(
                Ticks::<N>(1).value(),
                crate::Value::Bounded { name: Ticks::<N>::NAME.as_str(), raw: 1 }
            );
            assert_eq!(Ticks::<N>::NAME.as_str(), format!("umpire_dsl::countdown::Ticks<{N}>"));
            assert_eq!(
                Remaining::<N>::NAME.as_str(),
                format!("umpire_dsl::countdown::Remaining<{N}>")
            );
        }

        for bound in 0..=255 {
            let name = NominalName::new("umpire_dsl::countdown::Remaining", bound);
            assert_eq!(
                std::str::from_utf8(&name.bytes[..name.len]).unwrap_or_else(|e| panic!("{e}")),
                format!("umpire_dsl::countdown::Remaining<{bound}>")
            );
        }
        exported::<1>();
        exported::<9>();
        exported::<10>();
        exported::<99>();
        exported::<100>();
        exported::<255>();
    }

    #[test]
    fn finite_domains_and_membership_cover_the_boundaries() {
        fn domains<const N: u8>() {
            let configured: Vec<_> = Ticks::<N>::domain().map(|value| value.0).collect();
            let remaining: Vec<_> = Remaining::<N>::domain().map(|value| value.ticks).collect();
            assert_eq!(configured, (1..=N).collect::<Vec<_>>());
            assert_eq!(remaining, (0..=N).collect::<Vec<_>>());
            for value in 0..=u8::MAX {
                assert_eq!(Ticks::<N>::contains(value), configured.contains(&value));
                assert_eq!(Remaining::<N>::contains(value), remaining.contains(&value));
            }
        }
        domains::<0>();
        domains::<1>();
        domains::<3>();
        domains::<255>();
        assert_eq!(Ticks::<255>::domain().count(), 255);
        assert_eq!(Remaining::<255>::domain().count(), 256);
    }

    #[test]
    fn arm_preserves_absence_and_copies_even_forged_configuration() {
        for (configured, expected) in [
            (None, None),
            (Some(Ticks::<3>(0)), Some(Remaining { ticks: 0 })),
            (Some(Ticks(1)), Some(Remaining { ticks: 1 })),
            (Some(Ticks(3)), Some(Remaining { ticks: 3 })),
            (Some(Ticks(255)), Some(Remaining { ticks: 255 })),
        ] {
            assert_eq!(arm(configured), expected);
        }
        assert_eq!(arm(Some(Ticks::<255>(255))), Some(Remaining { ticks: 255 }));
        assert_eq!(arm(Some(Ticks::<0>(1))), Some(Remaining { ticks: 1 }));
    }

    #[test]
    fn tick_saturates_without_arming_or_sanitizing() {
        for (remaining, expected) in [
            (None, None),
            (Some(Remaining::<3> { ticks: 0 }), Some(Remaining { ticks: 0 })),
            (Some(Remaining { ticks: 1 }), Some(Remaining { ticks: 0 })),
            (Some(Remaining { ticks: 3 }), Some(Remaining { ticks: 2 })),
            (Some(Remaining { ticks: 255 }), Some(Remaining { ticks: 254 })),
        ] {
            assert_eq!(tick(remaining), expected);
        }
        assert_eq!(tick(Some(Remaining::<255> { ticks: 255 })), Some(Remaining { ticks: 254 }));
        assert_eq!(tick(Some(Remaining::<0> { ticks: 1 })), Some(Remaining { ticks: 0 }));
    }

    #[test]
    fn presence_and_expiry_distinguish_zero_from_absence() {
        assert!(!armed::<3>(None));
        assert!(!expired::<3>(None));
        for (ticks, is_expired) in [(0, true), (1, false), (3, false), (255, false)] {
            let remaining = Some(Remaining::<3> { ticks });
            assert!(armed(remaining));
            assert_eq!(expired(remaining), is_expired);
        }
        assert!(armed(Some(Remaining::<255> { ticks: 255 })));
        assert!(!expired(Some(Remaining::<255> { ticks: 255 })));
        assert!(armed(Some(Remaining::<0> { ticks: 0 })));
        assert!(expired(Some(Remaining::<0> { ticks: 0 })));
    }

    #[test]
    fn within_requires_a_valid_configuration_and_positive_live_budget() {
        for (remaining, configured, expected) in [
            (None, None, Verdict::Holds),
            (None, Some(1), Verdict::Holds),
            (None, Some(3), Verdict::Holds),
            (None, Some(0), Verdict::Fails),
            (None, Some(4), Verdict::Fails),
            (None, Some(255), Verdict::Fails),
            (Some(0), None, Verdict::Fails),
            (Some(1), None, Verdict::Fails),
            (Some(0), Some(3), Verdict::Fails),
            (Some(1), Some(1), Verdict::Holds),
            (Some(2), Some(3), Verdict::Holds),
            (Some(3), Some(3), Verdict::Holds),
            (Some(3), Some(2), Verdict::Fails),
            (Some(1), Some(0), Verdict::Fails),
            (Some(1), Some(4), Verdict::Fails),
            (Some(4), Some(3), Verdict::Fails),
            (Some(255), Some(255), Verdict::Fails),
        ] {
            let result = claim::evaluate(|| {
                within::<3>(remaining.map(|ticks| Remaining { ticks }), configured.map(Ticks));
            });
            assert_eq!(
                result.verdict, expected,
                "remaining={remaining:?}, configured={configured:?}"
            );
            assert_eq!(result.failure.is_some(), expected == Verdict::Fails);
        }
        assert_eq!(
            claim::evaluate(|| within::<1>(Some(Remaining { ticks: 1 }), Some(Ticks(1)))).verdict,
            Verdict::Holds
        );
        assert_eq!(
            claim::evaluate(|| within::<255>(Some(Remaining { ticks: 255 }), Some(Ticks(255))))
                .verdict,
            Verdict::Holds
        );
        assert_eq!(
            claim::evaluate(|| within::<255>(Some(Remaining { ticks: 0 }), Some(Ticks(255))))
                .verdict,
            Verdict::Fails
        );
        for (remaining, configured) in
            [(None, None), (None, Some(1)), (Some(0), Some(0)), (Some(1), Some(1))]
        {
            assert_eq!(
                claim::evaluate(|| within::<0>(
                    remaining.map(|ticks| Remaining { ticks }),
                    configured.map(Ticks)
                ))
                .verdict,
                Verdict::Fails
            );
        }
    }

    #[test]
    fn descends_checks_exact_saturating_descent_without_range_validation() {
        for (before, after, expected) in [
            (None, None, Verdict::Holds),
            (None, Some(0), Verdict::Fails),
            (Some(0), Some(0), Verdict::Holds),
            (Some(0), None, Verdict::Fails),
            (Some(0), Some(1), Verdict::Fails),
            (Some(1), Some(0), Verdict::Holds),
            (Some(1), None, Verdict::Fails),
            (Some(3), Some(2), Verdict::Holds),
            (Some(3), Some(3), Verdict::Fails),
            (Some(3), Some(1), Verdict::Fails),
            (Some(255), Some(254), Verdict::Holds),
            (Some(255), Some(255), Verdict::Fails),
        ] {
            let result = claim::evaluate(|| {
                descends::<3>(
                    after.map(|ticks| Remaining { ticks }),
                    before.map(|ticks| Remaining { ticks }),
                );
            });
            assert_eq!(result.verdict, expected, "before={before:?}, after={after:?}");
            assert_eq!(result.failure.is_some(), expected == Verdict::Fails);
        }
        assert_eq!(
            claim::evaluate(|| descends::<255>(
                Some(Remaining { ticks: 254 }),
                Some(Remaining { ticks: 255 })
            ))
            .verdict,
            Verdict::Holds
        );
        assert_eq!(
            claim::evaluate(|| descends::<0>(
                Some(Remaining { ticks: 0 }),
                Some(Remaining { ticks: 1 })
            ))
            .verdict,
            Verdict::Holds
        );
    }
}
