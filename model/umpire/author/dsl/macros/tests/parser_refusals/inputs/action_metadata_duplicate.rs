use umpire_dsl::Action;

#[derive(Action)]
enum User {
    #[metadata(example(first))]
    #[metadata(example(second))]
    Toggle,
}

fn main() {}
