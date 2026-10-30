use umpire_dsl::Action;

#[derive(Action)]
enum User {
    #[domain(sizes)]
    Set(bool),
}

fn main() {}
