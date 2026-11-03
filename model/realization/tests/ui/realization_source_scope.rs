use umpire_temporal::{activity, realization};

struct Binding;

macro_rules! hidden_scope {
    () => {
        #[realization(activity)]
        impl Binding {}
    };
}

hidden_scope!();

fn main() {
    let _ = activity::World::new(3, None, None, None, None, false);
}
