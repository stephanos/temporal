//! Semantic field kinds resolved by the Rust type checker.

macro_rules! scalar {
    ($($name:ident),* $(,)?) => {$ (
        #[doc = concat!("The `", stringify!($name), "` field kind.")]
        pub struct $name;
    )*};
}
scalar!(
    Text,
    Identifier,
    Uuid,
    Duration,
    RawDuration,
    Timestamp,
    Counter,
    Flag,
    Number,
    Bytes,
    Secret,
    Ignored
);

macro_rules! parameterized {
    ($($name:ident),*) => {$ (
        #[doc = concat!("The typed `", stringify!($name), "` field kind.")]
        pub struct $name<T>(std::marker::PhantomData<T>);
    )*};
}
parameterized!(Enum, Message, OneOf);

/// A protobuf map whose entry key and value have the declared kinds.
pub struct Map<K, V>(std::marker::PhantomData<(K, V)>);
