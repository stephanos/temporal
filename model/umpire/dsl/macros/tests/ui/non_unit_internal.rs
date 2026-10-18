use umpire_dsl::Action;

#[derive(Action)]
enum Heater {
    #[internal]
    Heat(u8),
    #[internal]
    Failover { reason: bool },
}

fn main() {}
