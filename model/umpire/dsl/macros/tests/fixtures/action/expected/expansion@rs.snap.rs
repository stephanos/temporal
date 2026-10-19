#[automatically_derived]
impl ::core::clone::Clone for Heater {
    fn clone(&self) -> Self {
        match *self {
            Self::Heat => Self::Heat,
            Self::Failover => Self::Failover,
        }
    }
}
#[automatically_derived]
impl ::core::cmp::PartialEq for Heater {
    fn eq(&self, other: &Self) -> bool {
        ::core::mem::discriminant(self) == ::core::mem::discriminant(other)
    }
}
#[automatically_derived]
impl ::core::cmp::Eq for Heater {}
#[automatically_derived]
impl ::core::hash::Hash for Heater {
    fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, state: &mut __UmpireHasher) {
        ::core::hash::Hash::hash(&::core::mem::discriminant(self), state);
    }
}
#[automatically_derived]
impl ::core::fmt::Debug for Heater {
    fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
        match *self {
            Self::Heat => out.write_str("Heat"),
            Self::Failover => out.write_str("Failover"),
        }
    }
}
#[automatically_derived]
impl ::core::fmt::Display for Heater {
    fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
        match *self {
            Self::Heat => out.write_str("Heater::Heat"),
            Self::Failover => out.write_str("Heater::Failover"),
        }
    }
}
impl Heater {
    /// Enumerates actions in their declaration order.
    pub fn actions() -> ::std::vec::Vec<Self> {
        ::std::vec![Self::Heat, Self::Failover]
    }
    /// Identifies environment-independent actions; this does not imply fairness.
    pub fn internal(&self) -> bool {
        true
    }
}
