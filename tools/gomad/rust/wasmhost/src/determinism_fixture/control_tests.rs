use super::super::{Fault, Kind};
use crate::linked_conditions::Conditions;

#[test]
fn explicit_fault_controls_are_scoped_owned_and_never_admit_an_ordinary_role() {
    let owner = crate::model::inspection::Handle {
        node: "a".into(),
        incarnation: 1,
    };
    assert!(
        Conditions::Ordinary
            .enter_with_fault(Some(Kind::Draw), Some(owner.clone()))
            .is_err()
    );
    assert_eq!(Fault::from_control(&owner).bootstrap_seed(7), 7);
    {
        let scope = Conditions::DeterminismFixtureReplay
            .enter_with_fault(Some(Kind::Draw), Some(owner.clone()))
            .unwrap();
        assert_eq!(Fault::from_control(&owner).bootstrap_seed(7), 7 | (1 << 63));
        let other = crate::model::inspection::Handle {
            node: "b".into(),
            incarnation: 1,
        };
        assert_eq!(Fault::from_control(&other).bootstrap_seed(7), 7);
        let session = scope.session();
        std::thread::scope(|threads| {
            threads
                .spawn(|| {
                    let _attached = Conditions::DeterminismFixtureReplay
                        .enter_shared(&session)
                        .unwrap();
                    assert_eq!(Fault::from_control(&owner).bootstrap_seed(7), 7 | (1 << 63));
                })
                .join()
                .unwrap();
        });
    }
    assert_eq!(Fault::from_control(&owner).bootstrap_seed(7), 7);
    let _plain = Conditions::DeterminismFixtureReplay.enter().unwrap();
    assert_eq!(Fault::from_control(&owner).bootstrap_seed(7), 7);
}

#[test]
fn output_clock_uses_only_its_execution_and_exact_logical_output_owner() {
    let owner = crate::model::inspection::Handle {
        node: "a".into(),
        incarnation: 2,
    };
    for committed in [false, true, false] {
        let _scope = Conditions::DeterminismFixtureReplay
            .enter_with_fault(Some(Kind::OutputClock), Some(owner.clone()))
            .unwrap();
        let mut fault = Fault::from_control(&owner);
        assert_eq!(fault.clock_shift(8), 0);
        let context = |instance: &str, incarnation| crate::event_order::Context {
            instance: instance.into(),
            incarnation,
            trace: 0,
            decision: 0,
            time_nanos: 1,
        };
        {
            let _other = crate::event_order::enter(context("a", 1));
            super::super::output_committed();
        }
        assert_eq!(fault.clock_shift(8), 0);
        if committed {
            let _selected = crate::event_order::enter(context("a", 2));
            super::super::output_committed();
        }
        assert_eq!(fault.clock_shift(7), 0);
        assert_eq!(fault.clock_shift(8), u64::from(committed));
        assert_eq!(fault.clock_shift(9), 0);
    }
}

#[test]
fn every_private_fixture_control_requires_opt_in_and_is_cleared_before_plain_execution() {
    let owner = crate::model::inspection::Handle {
        node: "fixture".into(),
        incarnation: 1,
    };
    for kind in [
        Kind::None,
        Kind::Draw,
        Kind::Clock,
        Kind::Handoff,
        Kind::OutputClock,
    ] {
        assert!(
            Conditions::Ordinary
                .enter_with_fault(Some(kind), Some(owner.clone()))
                .is_err()
        );
        assert!(super::current().is_none());
        {
            let _scope = Conditions::DeterminismFixtureReplay
                .enter_with_fault(Some(kind), Some(owner.clone()))
                .unwrap();
            assert!(super::current().unwrap().kind == kind);
            let mut fault = Fault::from_control(&owner);
            let seed = match kind {
                Kind::Draw => 7 | (1 << 63),
                Kind::OutputClock => 7 | (1 << 62),
                _ => 7,
            };
            assert_eq!(fault.bootstrap_seed(7), seed);
            if kind == Kind::OutputClock {
                let _context = crate::event_order::enter(crate::event_order::Context {
                    instance: owner.node.clone(),
                    incarnation: owner.incarnation,
                    trace: 0,
                    decision: 0,
                    time_nanos: 1,
                });
                super::super::output_committed();
            }
            assert_eq!(fault.clock_shift(7), 0);
            assert_eq!(
                fault.clock_shift(8),
                u64::from(matches!(kind, Kind::Clock | Kind::OutputClock))
            );
            assert_eq!(fault.clock_shift(9), 0);
            let mut handoff = Fault::from_control(&owner);
            assert_eq!(
                handoff.handoff(2, [true, true]),
                (kind == Kind::Handoff).then_some(1)
            );
        }
        assert!(super::current().is_none());
        let _plain = Conditions::DeterminismFixtureReplay.enter().unwrap();
        assert!(super::current().is_none());
        let mut fault = Fault::from_control(&owner);
        assert_eq!(fault.bootstrap_seed(7), 7);
        assert_eq!(fault.clock_shift(8), 0);
        assert_eq!(fault.handoff(2, [true, true]), None);
    }
}
