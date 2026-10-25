//! Breadth-first exploration, safety claims and fair progress over finite machines.

#![forbid(unsafe_code)]

mod explore;
#[cfg(not(target_arch = "wasm32"))]
mod interpret;
mod progress;
mod refusal;
mod report;
mod rng;
mod visited;

pub use explore::{Checker, Counts, Result, Transition, panic_message};
#[cfg(not(target_arch = "wasm32"))]
pub use interpret::Interpreted;
pub use refusal::Refusal;
pub use report::{ClaimResult, Path, PathStep, ProgressVerdict, Report};
pub use rng::SplitMix64;
pub use visited::{FullStates, StateId, Visited};

#[cfg(test)]
mod helper_tests {
    use super::{SplitMix64, panic_message};

    #[test]
    fn panic_payloads_preserve_strings_and_share_one_fallback() {
        assert_eq!(panic_message(&"borrowed"), "borrowed");
        assert_eq!(panic_message(&String::from("owned")), "owned");
        assert_eq!(panic_message(&42_u8), "non-string panic payload");
    }

    #[test]
    fn seeded_stream_reproduces_and_indices_stay_in_domain() {
        for seed in [0, 1, u64::MAX] {
            let mut first = SplitMix64::new(seed);
            let mut second = SplitMix64::new(seed);
            for len in [1, 2, 3, 255, usize::MAX] {
                assert_eq!(first.next_u64(), second.next_u64());
                let index = first.index(len);
                assert_eq!(index, second.index(len));
                assert!(index < len);
            }
        }
    }

    #[test]
    #[should_panic(expected = "empty index domain")]
    fn seeded_stream_refuses_empty_domain() {
        SplitMix64::new(0).index(0);
    }
}
