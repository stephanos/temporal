use crate::interleaved_claims::CounterWorld;
use umpire_dsl::Claim;

umpire_testing::fixture!(CounterWorld, 2, Claim::<CounterWorld>::new("supplemental", |_, _| true.into()));
