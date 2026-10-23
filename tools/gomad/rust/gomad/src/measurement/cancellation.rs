use gomad_wasmhost::linked::Cancellation;
use std::cell::RefCell;

thread_local! {
    static CURRENT: RefCell<Option<Cancellation>> = const { RefCell::new(None) };
}

pub(super) fn within<T>(token: &Cancellation, run: impl FnOnce() -> T) -> Result<T, String> {
    struct Reset;
    impl Drop for Reset {
        fn drop(&mut self) {
            CURRENT.with(|current| *current.borrow_mut() = None);
        }
    }
    CURRENT.with(|current| {
        let mut current = current.borrow_mut();
        if current.is_some() {
            return Err("refused: measurement cancellation scope is already active".to_owned());
        }
        *current = Some(token.clone());
        Ok(())
    })?;
    let _reset = Reset;
    Ok(run())
}

pub(super) fn cancelled() -> bool {
    CURRENT.with(|current| {
        current
            .borrow()
            .as_ref()
            .is_some_and(Cancellation::cancelled)
    })
}

pub(super) fn check() -> Result<(), String> {
    if cancelled() {
        Err("measurement cancelled".into())
    } else {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scope_resets_after_return_error_and_unwind() {
        let token = Cancellation::default();
        let result: Result<(), String> = within(&token, || {
            token.cancel();
            check()
        })
        .unwrap();
        assert!(result.unwrap_err().contains("cancelled"));
        assert!(!cancelled());
        let fresh = Cancellation::default();
        let panic = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            let _: Result<(), String> = within(&fresh, || panic!("collector unwound"));
        }));
        assert!(panic.is_err());
        assert!(!cancelled());
        within(&fresh, || check().unwrap()).unwrap();
    }

    #[test]
    fn nesting_cannot_replace_the_active_request_token() {
        let outer = Cancellation::default();
        let inner = Cancellation::default();
        inner.cancel();
        within(&outer, || {
            assert!(within(&inner, || ()).unwrap_err().starts_with("refused:"));
            assert!(!cancelled());
            outer.cancel();
            assert!(cancelled());
        })
        .unwrap();
        assert!(!cancelled());
    }

    #[test]
    fn overlapping_thread_scopes_keep_request_tokens_separate() {
        let barrier = std::sync::Barrier::new(2);
        let cancelled_token = Cancellation::default();
        let ordinary_token = Cancellation::default();
        std::thread::scope(|scope| {
            let cancelled_scope = scope.spawn(|| {
                within(&cancelled_token, || {
                    cancelled_token.cancel();
                    barrier.wait();
                    assert!(cancelled());
                })
                .unwrap();
            });
            within(&ordinary_token, || {
                barrier.wait();
                assert!(!cancelled());
            })
            .unwrap();
            cancelled_scope.join().unwrap();
        });
        assert!(!cancelled());
    }
}
