//! Every descriptor case is now admitted by claims or an independently sourced semantic rule.

use umpire_catalog::AcceptedGap;

pub(super) fn unannotated() -> Vec<(String, AcceptedGap)> {
    Vec::new()
}
