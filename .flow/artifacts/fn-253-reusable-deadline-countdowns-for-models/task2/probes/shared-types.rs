use crate::Finite;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct Ticks<const N: u8>(pub u8);
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct Remaining<const N: u8> { pub ticks: u8 }

impl<const N: u8> Ticks<N> {
    pub fn contains(value: u8) -> bool { N > 0 && value > 0 && value <= N }
}
impl<const N: u8> Remaining<N> {
    pub fn contains(value: u8) -> bool { value <= N }
}

impl<const N: u8> Finite for Ticks<N> {
    fn domain() -> impl Iterator<Item = Self> { (1..=N).map(Self) }
}
impl<const N: u8> Finite for Remaining<N> {
    fn domain() -> impl Iterator<Item = Self> { (0..=N).map(|ticks| Self { ticks }) }
}
