use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Set {
        #[inputs(..=3)]
        value: u8,
    },
}

fn main() {}
