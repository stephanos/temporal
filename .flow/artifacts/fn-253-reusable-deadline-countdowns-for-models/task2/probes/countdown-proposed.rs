//! Proposed .3 surface, separate from the comparative trait adapter.
use crate::{check, countdown_types::{Remaining, Ticks}};

pub fn arm<const N: u8>(configured: Option<Ticks<N>>) -> Option<Remaining<N>> {
    if configured.is_some() {
        Some(Remaining { ticks: configured.unwrap_or(Ticks(1)).0 })
    } else { None }
}
pub fn tick<const N: u8>(remaining: Option<Remaining<N>>) -> Option<Remaining<N>> {
    if remaining.is_some() && remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0 {
        Some(Remaining { ticks: remaining.unwrap_or(Remaining { ticks: 0 }).ticks - 1 })
    } else { remaining }
}
pub fn armed<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining.is_some()
}
pub fn expired<const N: u8>(remaining: Option<Remaining<N>>) -> bool {
    remaining.is_some() && remaining.unwrap_or(Remaining { ticks: 0 }).ticks == 0
}
pub fn within<const N: u8>(remaining: Option<Remaining<N>>, configured: Option<Ticks<N>>) {
    check!(N > 0);
    check!(configured.is_none()
        || (configured.unwrap_or(Ticks(1)).0 > 0 && configured.unwrap_or(Ticks(1)).0 <= N));
    check!(remaining.is_none()
        || (configured.is_some() && remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            && remaining.unwrap_or(Remaining { ticks: 0 }).ticks <= configured.unwrap_or(Ticks(1)).0));
}
pub fn descends<const N: u8>(after: Option<Remaining<N>>, before: Option<Remaining<N>>) {
    check!(after == tick(before));
}
