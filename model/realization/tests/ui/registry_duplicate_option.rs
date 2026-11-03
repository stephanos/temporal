umpire_temporal::registry! {
    model { specimen }
    actors { enum Client { #[metadata(temporal(event = "event", event = "duplicate", evidence = event))] First } }
    body { #[registry(inputs = ())] impl Binding {} }
}
fn main() {}
