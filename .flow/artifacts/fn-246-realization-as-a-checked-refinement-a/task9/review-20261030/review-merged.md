All three source reviews returned SHIP. Duplicate protocol-version finding collapsed; raw reviews retained. R1 is partial because frontier/changed past belongs to .10; R5 Model proof belongs to .13/.15 and later integration. Pre-existing unchecked outcome indexing and empty-witness indexing remain outside this change.

## Finding 1

- **Severity**: P3
- **Title**: Use protocol constant in expected rejection
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/hosts/go/runner/modes_test.go:161`
- **R-IDs**: []
- **Problem**: Expected version3 is hardcoded beside ProtocolVersion+1.
- **Suggestion**: Format the reason using the constant.

## Finding 2

- **Severity**: P3
- **Title**: Distinguish mutant event rejection
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/hosts/go/runner/prefix_test.go:65`
- **R-IDs**: []
- **Problem**: Mutated recording controls prove exact replay; semantic coverage comes from positive expected witnesses and consumer control.
- **Suggestion**: Assert rejection occurs at the changed event.

## Finding 3

- **Severity**: P3
- **Title**: Remove vacuous clone immutability assertion
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/protocol/runtime/tests/main/prefix.rs:174`
- **R-IDs**: []
- **Problem**: Reencoding owned clones cannot detect runtime mutation; Go recording byte comparison covers immutability meaningfully.
- **Suggestion**: Remove clone loop and narrow Rust test name.

## Finding 4

- **Severity**: P3
- **Title**: Make admission reason explicit in table
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/hosts/go/runner/admission_test.go:141`
- **R-IDs**: []
- **Problem**: Reason expectation depends on subtest name matching.
- **Suggestion**: Add explicit wantReason field.

Unaddressed R-IDs: []

Classification counts: 4 introduced, 0 pre_existing.

<verdict>SHIP</verdict>
