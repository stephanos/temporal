#[automatically_derived]
impl<T> ::core::clone::Clone for World<T>
where
    T: ::core::clone::Clone,
    u8: ::core::clone::Clone,
{
    fn clone(&self) -> Self {
        Self {
            initial: ::core::clone::Clone::clone(&self.initial),
            limit: ::core::clone::Clone::clone(&self.limit),
        }
    }
}
#[automatically_derived]
impl<T> ::core::cmp::PartialEq for World<T>
where
    T: ::core::cmp::PartialEq,
    u8: ::core::cmp::PartialEq,
{
    fn eq(&self, other: &Self) -> bool {
        true && self.initial == other.initial && self.limit == other.limit
    }
}
#[automatically_derived]
impl<T> ::core::cmp::Eq for World<T>
where
    T: ::core::cmp::Eq,
    u8: ::core::cmp::Eq,
{}
#[automatically_derived]
impl<T> ::core::hash::Hash for World<T>
where
    T: ::core::hash::Hash,
    u8: ::core::hash::Hash,
{
    fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, state: &mut __UmpireHasher) {
        ::core::hash::Hash::hash(&self.initial, state);
        ::core::hash::Hash::hash(&self.limit, state);
    }
}
#[automatically_derived]
impl<T> ::core::fmt::Debug for World<T>
where
    T: ::core::fmt::Debug,
    u8: ::core::fmt::Debug,
{
    fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
        out.debug_struct("World")
            .field("initial", &self.initial)
            .field("limit", &self.limit)
            .finish()
    }
}
#[automatically_derived]
impl<T> ::core::fmt::Display for World<T>
where
    T: ::core::fmt::Debug,
    u8: ::core::fmt::Debug,
{
    fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
        ::core::fmt::Debug::fmt(self, out)
    }
}
impl<T> World<T> {
    /// Constructs the fixed parameters in field declaration order.
    pub fn new(initial: T, limit: u8) -> Self {
        Self { initial, limit }
    }
}
