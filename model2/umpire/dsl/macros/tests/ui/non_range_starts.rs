use umpire_dsl::State;

#[derive(State)]
struct Kettle {
    #[starts(42)]
    temperature: i32,
    #[starts(0..)]
    unbounded: i32,
}

fn main() {}
