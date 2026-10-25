//! Typed action builders preserve declaration generics and payload field names.

#![forbid(unsafe_code)]

use umpire_dsl::{Action, InputDomain};

#[derive(Action)]
enum User<const N: usize> {
    Poll,
}

#[derive(Action)]
enum Worker {
    Complete {
        #[inputs([false, true])]
        finish: bool,
    },
}

#[test]
fn const_generic_actor_builders_enumerate_the_declared_variant() {
    let domain = User::<3>::__umpire_path_Poll().finish();
    assert_eq!(domain.count(), Ok(1));
    assert_eq!(domain.get(0), User::<3>::Poll);
    assert_eq!(User::<3>::__umpire_path_actions().count(), Ok(1));
}

#[test]
fn finish_payload_field_can_be_selected_or_left_at_its_declared_domain() {
    let selected = Worker::__umpire_path_Complete()
        .__umpire_input_finish(Some(InputDomain::values(vec![true])))
        .finish();
    assert_eq!(selected.count(), Ok(1));
    assert_eq!(selected.get(0), Worker::Complete { finish: true });

    let wildcard = Worker::__umpire_path_Complete().__umpire_input_finish(None).finish();
    assert_eq!(wildcard.count(), Ok(2));
    assert_eq!(wildcard.get(0), Worker::Complete { finish: false });
    assert_eq!(wildcard.get(1), Worker::Complete { finish: true });
}
