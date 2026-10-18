use umpire_dsl::Action;

#[derive(Action)]
enum Heater {
    #[internal]
    Heat,
}

fn main() {}
