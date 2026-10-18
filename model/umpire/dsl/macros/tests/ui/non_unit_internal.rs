use umpire_dsl::Action;

#[derive(Action)]
#[internal]
enum Heater {
    Heat(u8),
    Failover { reason: bool },
}

fn main() {}
