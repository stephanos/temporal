//! Located refusal controls for declarative realization handlers.

use umpire_temporal_annotate_syntax::{
    realization::Schema,
    realize::{expand, expand_in_source},
};

#[test]
fn handler_admission_rejects_ambiguous_or_unclassified_handlers()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema::new(&[], "", &[])?;
    for (body, expected) in [
        (
            "#[observe(WorkflowService::Describe)] fn first(response: Response) -> Result<Projection<Action>, Refusal> { todo!() } #[observe(WorkflowService::Describe)] fn second(response: Response) -> Result<Projection<Action>, Refusal> { todo!() }",
            "duplicate observation handler",
        ),
        (
            "#[observe(WorkflowService::Describe)] fn first(response: Response) -> Result<usize, Refusal> { todo!() }",
            "expected Projection, Step, or a state value with project",
        ),
        ("#[propose] fn first() {} #[propose] fn second() {}", "duplicate proposal handler"),
        ("#[propose(unknown = method)] fn first() {}", "Unknown field: `unknown`"),
        (
            "#[answer(Task, ignores(input = \" \"))] fn first(task: Task) -> Result<Reply<Action>, Refusal> { todo!() }",
            "ignore reason must not be empty",
        ),
        (
            "#[answer(Task)] fn first(task: Task) -> Result<Reply<Action>, Refusal> { todo!() }",
            "answer requires at least one produces request context",
        ),
    ] {
        let item = syn::parse_str(&format!("impl Realization {{ {body} }}"))?;
        let error = expand(&syn::parse_str("activity")?, item, &schema)
            .err()
            .ok_or("expected handler admission refusal")?;
        assert!(error.to_string().contains(expected), "{error}");
    }
    Ok(())
}

#[test]
fn generated_enum_wildcards_are_refused_without_rejecting_byte_policy_fallbacks()
-> Result<(), Box<dyn std::error::Error>> {
    let mut schema = Schema::default();
    schema.enums.insert(
        "test.Status".into(),
        umpire_temporal_annotate_syntax::realization::Enumeration {
            rust_type: "generated :: Status".into(),
            variants: vec!["generated :: Status :: Ready".into()],
        },
    );
    for body in [
        "fn decode(value: generated::Status) { match value { generated::Status::Ready => (), _ => () } }",
        "fn decode(value: generated::Status) { match value { _ => () } }",
        "fn decode(value: generated::Status) { match &value { _ => () } }",
        "fn decode(value: (generated::Status, generated::Status)) { match value { (generated::Status::Ready, generated::Status::Ready) => (), (generated::Status::Ready, _) => () } }",
    ] {
        let item = syn::parse_str(&format!("impl Realization {{ {body} }}"))?;
        let error = expand(&syn::parse_str("activity")?, item, &schema)
            .err()
            .ok_or("expected generated enum wildcard refusal")?;
        assert!(error.to_string().contains("wildcard"), "{error}");
    }
    let bytes = syn::parse_str(
        "impl Realization { fn policy(value: &[u8]) { match value { b\"Complete\" => (), _ => () } } }",
    )?;
    assert!(expand(&syn::parse_str("activity")?, bytes, &schema).is_ok());
    Ok(())
}

#[test]
fn enum_aliases_follow_module_and_block_scopes_without_classifying_model_enums()
-> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(ALIAS_CASES)
}

#[test]
fn wrapper_alias_payloads_do_not_erase_enum_identity() -> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(&[
        (
            "type Wrapped=Option<generated::Status>; impl Binding { fn f(value:Wrapped) { match value { Some(_)=>(), None=>() } } }",
            0,
            true,
        ),
        (
            "type Wrapped<T,E>=Result<E,T>; impl Binding { fn f(value:Wrapped<i32,generated::Status>) { match value { Ok(_)=>(), Err(_)=>() } } }",
            0,
            true,
        ),
        (
            "type Wrapped=Option<generated::Status>; impl Binding { fn f(value:Wrapped) { match value { Some(generated::Status::Ready)=>(), None=>() } } }",
            0,
            false,
        ),
        (
            "impl Binding { fn f(value:Result<generated::Status,i32>) { match value { Ok(generated::Status::Ready)=>(), Err(_)=>() } } }",
            0,
            false,
        ),
    ])
}

#[test]
fn dereferenced_enum_subjects_preserve_coverage() -> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(&[
        (
            "impl Binding { fn f(value:&Option<generated::Status>) { match value { Some(_)=>(), None=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:&Option<generated::Status>) { match value { &Some(_)=>(), &None=>() } } }",
            0,
            true,
        ),
        (
            "enum Local { Ready, Other } impl Binding { fn f(value:Local) { let borrowed=&value; match *borrowed { _=>() } } }",
            0,
            false,
        ),
        (
            "struct Wrapper; impl std::ops::Deref for Wrapper { type Target=generated::Status; fn deref(&self)->&Self::Target { todo!() } } impl Binding { fn f(value:Wrapper) { match *value { _=>() } } }",
            1,
            true,
        ),
        ("impl Binding { fn f(value:&generated::Status) { match *value { _=>() } } }", 0, true),
        (
            "enum Local { Ready, Other } impl Binding { fn f(value:&Local) { match *value { _=>() } } }",
            0,
            false,
        ),
        (
            "impl Binding { fn f(value:&generated::Status) { match *value { generated::Status::Ready=>() } } }",
            0,
            false,
        ),
    ])
}

#[test]
fn deferred_and_reassigned_bindings_require_proven_coverage()
-> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(&[
        (
            "impl Binding { fn f() { let value:Option<_>=Some(generated::Status::Ready); match value { Some(_)=>(), None=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f() { let value; value=generated::Status::Ready; match value { _=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f() { let mut value:Option<_>=None; value=Some(generated::Status::Ready); match value { Some(_)=>(), None=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(flag:bool) { let mut value:Option<_>=None; if flag { value=Some(generated::Status::Ready); } match value { Some(_)=>(), None=>() } } }",
            0,
            true,
        ),
        ("impl Binding { fn f() { let value; value=0; match value { _=>() } } }", 0, false),
        ("impl Binding { fn f() { let mut value=0; value=1; match value { _=>() } } }", 0, false),
    ])
}

#[test]
fn method_and_cast_results_require_proven_coverage() -> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(&[
        (
            "impl Binding { fn f(value:Result<(generated::Status,u8),()>)->Result<(),()> { match value? { (_,_)=>() }; Ok(()) } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:Result<(),i32>) { let mapped=value.map_err(|_|generated::Status::Ready); match mapped { Ok(())=>(), Err(_)=>() } } }",
            0,
            true,
        ),
        (
            "struct Local; impl Local { fn unwrap(self)->generated::Status { todo!() } } impl Binding { fn f(value:Local) { match value.unwrap() { _=>() } } }",
            1,
            true,
        ),
        (
            "impl Binding { fn f(value:Result<(generated::Status,u8),()>) { match value.unwrap() { (_,_)=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:&generated::Status) { match value as &generated::Status { _=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:generated::Status) { match value as i32 { _=>() } } }",
            0,
            false,
        ),
    ])
}

#[test]
fn rest_patterns_cover_actual_payload_positions() -> Result<(), Box<dyn std::error::Error>> {
    check_alias_cases(&[
        (
            "impl Binding { fn f(value:(generated::Status,u8,u8,generated::Status)) { match value { (generated::Status::Ready,..,generated::Status::Ready)=>() } } }",
            0,
            false,
        ),
        (
            "impl Binding { fn f(value:Option<generated::Status>) { match value { Some(..)=>(), None=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:(generated::Status,u8)) { match value { (..)=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:(u8,generated::Status,u8,u8,u8)) { match value { (_,..,_)=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:(u8,u8,u8,u8,generated::Status)) { match value { (..,0,_)=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:Result<(generated::Status,u8),()>) { match value.unwrap() { (..)=>() } } }",
            0,
            true,
        ),
        (
            "impl Binding { fn f(value:(generated::Status,u8,u8,u8)) { match value { (generated::Status::Ready,..)=>() } } }",
            0,
            false,
        ),
        (
            "impl Binding { fn f(value:(u8,u8,u8,generated::Status)) { match value { (..,generated::Status::Ready)=>() } } }",
            0,
            false,
        ),
        (
            "impl Binding { fn f(value:Result<(u8,u8),i32>) { match value { Ok((..))=>(), Err(_)=>() } } }",
            0,
            false,
        ),
    ])
}

fn check_alias_cases(cases: &[(&str, usize, bool)]) -> Result<(), Box<dyn std::error::Error>> {
    fn implementations(items: &[syn::Item], result: &mut Vec<syn::ItemImpl>) {
        for item in items {
            match item {
                syn::Item::Impl(item) => result.push(item.clone()),
                syn::Item::Mod(module) if module.content.is_some() => {
                    if let Some((_, items)) = &module.content {
                        implementations(items, result);
                    }
                }
                _ => {}
            }
        }
    }
    let mut schema = Schema::default();
    schema.enums.insert(
        "test.Status".into(),
        umpire_temporal_annotate_syntax::realization::Enumeration {
            rust_type: "generated::Status".into(),
            variants: vec!["generated::Status::Ready".into()],
        },
    );
    let mut failures = vec![];
    for &(source, index, refused) in cases {
        let file = syn::parse_file(source)?;
        let mut items = vec![];
        implementations(&file.items, &mut items);
        let item = items.into_iter().nth(index).ok_or("missing test impl")?;
        let location = item.impl_token.span.start();
        let expanded =
            expand_in_source(&syn::parse_str("activity")?, item, &schema, source, location);
        match expanded {
            Err(error) if refused && error.to_string().contains("wildcard") => {}
            Err(error) => failures.push(format!("unexpected refusal: {source}: {error}")),
            Ok(_) if refused => failures.push(format!("expected alias wildcard refusal: {source}")),
            Ok(_) => {}
        }
    }
    if !failures.is_empty() {
        return Err(failures.join("\n").into());
    }
    Ok(())
}

const ALIAS_CASES: &[(&str, usize, bool)] = &[
    (
        "enum Local { Ready, Other } impl Binding { fn f() { fn inner(s:Local) { match s { _=>() } } let inner=|s:Local| match s { _=>() }; } }",
        0,
        false,
    ),
    (
        "use generated::Status as S; impl Binding { fn f() { let s=0; fn inner(s:S) { match s { _=>() } } } }",
        0,
        true,
    ),
    (
        "use generated::Status as S; impl Binding { fn f() { let s=0; let inner=|s:S| match s { _=>() }; } }",
        0,
        true,
    ),
    (
        "use generated::Status as S; impl Binding { fn f(value:S) { let s=0; let (s,)= (value,); match s { _=>() } } }",
        0,
        true,
    ),
    (
        "use generated::Status as S; impl Binding { fn f(value:Option<S>) { let s=0; match value { Some(s)=>match s { _=>() }, None=>() } } }",
        0,
        true,
    ),
    ("impl Binding { fn f() { match sample() { Some(_)=>(), None=>() } } }", 0, true),
    (
        "impl Binding { fn f(value:Result<generated::Status,i32>) { match value { Ok(generated::Status::Ready)=>(), Ok(_)=>(), Err(_)=>() } } }",
        0,
        true,
    ),
    (
        "impl Binding { fn f(value:Option<generated::Status>) { match value { Some(generated::Status::Ready)=>(), None=>() } } }",
        0,
        false,
    ),
    (
        "impl Binding { fn f(value:Result<generated::Status,i32>) { match value { Ok(generated::Status::Ready)=>(), Err(_)=>() } } }",
        0,
        false,
    ),
    (
        "use generated::Status::{Ready as Active}; impl Binding { fn f() { match sample() { Active=>(), _=>() } } }",
        0,
        true,
    ),
    (
        "use generated::Status::*; impl Binding { fn f() { match sample() { Ready=>(), _=>() } } }",
        0,
        true,
    ),
    (
        "use generated::Status::{Ready as Active}; impl Binding { fn f() { match sample() { Active=>() } } }",
        0,
        false,
    ),
    ("impl Binding { fn f() { match external_status() { _=>() } } }", 0, true),
    (
        "use generated::Status::*; impl Binding { fn f(value:Option<generated::Status>) { match value { Some(_)=>(), None=>() } } }",
        0,
        true,
    ),
    (
        "use generated::Status::*; impl Binding { fn f(value:Result<(),i32>) { match value { Ok(())=>(), Err(_)=>() } } }",
        0,
        false,
    ),
    (
        "use generated::Status::*; impl Binding { fn f(value:generated::Status) { match value { other=>() } } }",
        0,
        true,
    ),
    (
        "mod aliases; use aliases::*; impl Binding { fn f(value:String) { match value { String::Running=>(), _=>() } } }",
        0,
        true,
    ),
    (
        "type S=(generated::Status,); impl Binding { fn f(value:S) { match value { _=>() } } }",
        0,
        true,
    ),
    (
        "#[cfg(any())] type S=generated::Status; #[cfg(not(any()))] enum S { Ready, Other } impl Binding { fn f(value:S) { match value { S::Ready=>(), _=>() } } }",
        0,
        true,
    ),
    (
        "impl Binding { fn policy(value:Result<(),i32>) { match value { Ok(())=>(), Err(_)=>() } } }",
        0,
        false,
    ),
    (
        "use generated::{Status as S}; impl Binding { fn f(value:S) { match value { S::Ready => (), _ => () } } }",
        0,
        true,
    ),
    (
        "use generated as e; type S=e::Status; impl Binding { fn f(value:S) { match value { _ => () } } }",
        0,
        true,
    ),
    (
        "type S2=S; type S=generated::Status; impl Binding { fn f(value:S2) { match value { _ => () } } }",
        0,
        true,
    ),
    (
        "impl Binding { fn f() { use generated::Status as S; let value:S=todo!(); match value { S::Ready => (), _ => () } } }",
        0,
        true,
    ),
    (
        "mod parent { use generated::Status as S; mod child { use super::S as C; impl Binding { fn f(value:C) { match value { _ => () } } } } }",
        0,
        true,
    ),
    (
        "use generated::Status as S; impl Binding { fn f(value:S) { { enum S { Ready, Other } let value=S::Ready; match value { S::Ready=>(), _=>() } } match value { S::Ready=>(), _=>() } } }",
        0,
        true,
    ),
    (
        "use generated::Status as S; impl Binding { fn f() { enum S { Ready, Other } let value=S::Ready; match value { S::Ready=>(), _=>() } } }",
        0,
        false,
    ),
    (
        "use generated::Status as S; impl Binding { fn f<S>(value:S) { match value { _=>() } } }",
        0,
        false,
    ),
    (
        "enum Status { Ready, Other } impl Binding { fn f(value:Status) { match value { Status::Ready=>(), _=>() } } }",
        0,
        false,
    ),
    (
        "mod domain { pub enum Status { Ready, Other } } use domain::Status as S; impl Binding { fn f(value:S) { match value { S::Ready=>(), _=>() } } }",
        0,
        false,
    ),
    (
        "mod first { use generated::Status as S; impl Binding { fn f(value:S) { match value { _=>() } } } } mod second { enum S { Ready, Other } impl Binding { fn f(value:S) { match value { S::Ready=>(), _=>() } } } }",
        1,
        false,
    ),
    (
        "use generated::Status as S; impl Binding { fn policy(value:Vec<u8>) { match value.as_slice() { b\"Complete\"=>drop(value), _=>drop(value) } } }",
        0,
        false,
    ),
];

#[test]
fn proposal_dispatch_delegates_the_handler_and_optional_hooks()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema::new(&[], "", &[])?;
    for (attribute, hooks) in [
        ("#[propose]", false),
        ("#[propose(matches = correlated, continue_polling = polling)]", true),
    ] {
        let item = syn::parse_str(&format!(
            "impl Realization {{ {attribute} fn selected(&self, state: &activity::Model, seed: u64) -> Result<Proposal<Action>, Refusal> {{ todo!() }} }}"
        ))?;
        let expanded = expand(&syn::parse_str("activity")?, item, &schema)?.to_string();
        assert!(expanded.contains("self . selected (state , seed)"));
        assert_eq!(expanded.contains("self . correlated"), hooks);
        assert_eq!(expanded.contains("self . polling"), hooks);
    }
    Ok(())
}

#[test]
fn observation_only_realizations_inherit_the_refusing_answer_default()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema::new(&[], "", &[])?;
    let expanded =
        expand(&syn::parse_str("activity")?, syn::parse_str("impl Realization {}")?, &schema)?;
    assert!(!expanded.to_string().contains("fn answer"));
    Ok(())
}
