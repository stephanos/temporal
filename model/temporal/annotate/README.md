# Exhaustive annotation catalogs

`annotate!` declares a module containing executable `Catalog<T>` factories, enum
conversions and Model action bindings. The API build script parses the same
declarations to check protobuf coverage before generating Rust API types.

```rust,ignore
annotate! {
    pub catalog api;
    message request start: StartRequest = "example.StartRequest" {
        request_id: Uuid => Rejects(3, "request_id") covers "validateStart";
        namespace: Identifier + Decided => Rejects(3, "namespace") covers "validateStart";
        rules {
            required(namespace) => Rejects(3, "namespace") covers "validateStart";
        }
    }
    enum status: Status = "example.Status" {
        Status::Unspecified = "STATUS_UNSPECIFIED",
        Status::Running = "STATUS_RUNNING",
    }
    service "example.Service" {
        Start: Action("Client::Start");
        Describe: Observed("describe");
        Delete: OutOfScope("This Model does not delete executions");
    }
    actions bind: Action {
        Action::Client => {
            Client::Start => Rpc("example.Service.Start"),
        },
        Action::Timer => {
            Timer::Expire => Environmental("Elapsed server deadline"),
        },
    }
}
```

Factories return `Result<Catalog<T>, AdmissionErrors>`. Enum conversions take an
`i32`, call the concrete enum's `TryFrom<i32>` and refuse unknown integers with
`UnknownEnum`. Their known variants and both levels of action matches have no
fallback. Field patterns omit `..`; rustc checks every declared field and kind.

Kinds use named literal constraints in brackets: `Text[min_len = 1, prefix = "a"]`,
`Counter[gte = 0]`, `Bytes[max_len = 32]`, and repeated `min_items`, `max_items`,
`unique`. `Enum(status)` uses the discriminants of the named exhaustive converter
and requires defined values. Request constraints and presence rules require an
expectation (`Accepts`, `Unobservable`, or `Rejects(code, pattern, optional_details)`)
and nonempty `covers` provenance.

Builtin tags follow `+`, such as `+ Immutable`, `+ Cached(30)` (seconds), and
`+ EventuallyConsistent(3)` (polls). A custom tag is a generated type implementing
the generic `Tag` trait; its kinds and consequence are checked from literals:

```rust,ignore
tag Visibility {
    kinds: [Timestamp];
    consequence: EventuallyConsistent(3);
}
```

Arbitrary dynamic `Tag` implementations remain usable through the generic
runtime builder. The macro admits custom tags only through this declaration.

Wrapper fields can use a typed projection:

```rust,ignore
timestamp: Timestamp via umpire_annotations::Timestamp (
    |m: &Response| project_time(&m.timestamp),
    |m: &mut Response, time: umpire_annotations::Timestamp| m.timestamp = wrap_time(time)
);
```

An intentionally ignored bool, map or message uses the explicit opaque adapter,
retaining its real Rust type and reason. Its kind is unconstrained `Bytes` or
`Secret`; secret redaction still clears it. Ordinary ignored fields retain kind
witnesses. Rules cannot depend on ignored fields.

```rust,ignore
opaque: Bytes + Ignored("Payload not judged by this Model") opaque;
choice: Secret + Ignored("Credentials") opaque variants {
    Choice::Password = "password",
    Choice::Token = "token",
};
```

Oneof alternatives produce separate source-field metadata and typed payload
projections; an exhaustive borrowed variant match catches additions. The
descriptor check additionally checks all source alternatives. Nested message
and enum references require their own coverage declarations unless the owning
field is explicitly opaque.

The API build tracks feature `api.rs` files and `annotate/catalogs/*.rs` recursively,
including directories and `proto/api.binpb`. The upstream image currently has no
services. The existing `getproto --linked-api` facility adds actual linked
WorkflowService descriptors to an `OUT_DIR` image for checking; API generation
continues to use the upstream image. All upstream-owned files survive linking
unchanged. The bootstrap control classifies every WorkflowService RPC as outside
its behavioral scope. Task .3 replaces or moves these classifications into the
feature catalog; it must not add a second classification of the same service.

The normal facade has no parser, descriptor or generated API dependency. The
parser/checker lives only in host proc-macro and build dependencies.
