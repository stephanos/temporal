use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Set {
        #[inputs(0..=1)]
        #[inputs([2, 3])]
        value: u8,
    },
}

fn main() {}
