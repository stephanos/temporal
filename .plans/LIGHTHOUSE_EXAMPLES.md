# Industrial evidence for executable behavior models

These four case studies report concrete results from modeling, model checking, and checking
implementations against specifications. They support investing in a measured Umpire pilot. They
do not establish that Umpire already delivers the same results, or that every behavior is worth
modeling in detail.

The comparisons with Umpire below are our assessment. The reported outcomes belong to the cited
authors and their particular systems, tools, bounds, and workloads.

## 1. Amazon S3 ShardStore: small reference models checking real code

**Reported result.** The SOSP 2021 paper reports that lightweight formal methods prevented
16 issues from reaching production, including crash-consistency and concurrency problems.
The approach combined executable reference models with other checks; the 16 issues are the
result of that combined approach, not a count attributable solely to reference models.

The authors report that reference models averaged approximately 1% of the size of their
corresponding implementations. The broader specification effort increased the codebase by
approximately 14%. These are code-size measurements, not engineering-time estimates.

**Mechanism.** A complicated storage structure can have a much simpler behavioral specification.
For example, a hash table can represent the logical behavior of an LSM tree. Generated operation
sequences exercise the implementation against that reference; failing sequences can be reduced.
Additional techniques check concurrency and crash behavior. The authors also report that
developers without formal-methods specialization extended the specifications.

**Relevance to Umpire.** This is a direct precedent for maintaining a second executable
description when it removes implementation machinery and supports automated checks. For matching,
the analogous opportunity is to describe which tasks must remain recoverable without reproducing
the reader's buffers, trees, and caches.

Sources:

- [Using lightweight formal methods to validate a key-value storage node in Amazon S3, SOSP 2021](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)
- [Author's explanation, including model size and specification overhead](https://www.amazon.science/blog/aws-team-wins-best-paper-award-for-work-on-automated-reasoning)

## 2. MongoDB Realm Sync: generated tests outperform existing coverage

**Reported result.** In the VLDB 2020 study, a bounded TLA+ model generated 4,913 C++ tests for
array-operation merge rules. Those tests covered all 86 branches in the measured portion of the
implementation. The existing handwritten tests covered 21%; AFL fuzzing reached 92% after
approximately eight million executions. Translating the algorithm into TLA+ also exposed an
infinite-recursion defect that the authors confirmed in the C++ implementation.

**Mechanism.** Model checking explored three participants operating on a three-element array,
with each participant performing one operation. The explored behaviors became implementation
tests. This gives a concrete comparison between model-derived cases, handwritten tests, and
fuzzing on the same code.

**Relevance to Umpire.** This closely matches the model-to-case workflow. It suggests a measurable
claim for a pilot: small, systematically explored scenarios can reach behavior that existing tests
miss. The reported coverage applies to selected merge rules, not the entire product; branch
coverage itself is not proof of correctness.

**Important counterexample.** The same study's separate attempt to trace-check MongoDB Server
against an abstract replication specification was impractical. Instrumentation and mismatched
atomicity consumed the budget. The authors' later retrospective reports ten weeks of effort
without successful trace-checking of that specification. This is evidence that the cost of
connecting a model to an implementation must be measured explicitly.

Sources:

- [eXtreme Modelling in Practice, VLDB 2020](https://arxiv.org/html/2006.00915)
- [Author's 2025 retrospective on both the successful and unsuccessful experiments](https://www.mongodb.com/company/blog/engineering/conformance-checking-at-mongodb-testing-our-code-matches-our-tla-specs)

## 3. AWS DynamoDB and S3: finding bugs that survived review and testing

**Reported result.** AWS's 2015 experience report describes a DynamoDB bug whose shortest
counterexample required 35 high-level steps. It had survived extensive design review, code review,
and testing. The model checker subsequently found two additional subtle bugs in other algorithms.
For an S3 algorithm with a known bug, model checking reproduced the bug and found another problem
in the team's proposed fix.

**Mechanism.** Engineers expressed the algorithms in TLA+/PlusCal and explored their behavior
against explicit properties. The model allowed them to examine combinations of failures and
interleavings, and to check proposed changes before implementing them.

**Relevance to Umpire.** The benefit is additional reasoning power over event sequences. Matching
has similar questions involving lost responses, ownership changes, delayed writes, and retries.
A short implementation or a thorough review does not by itself explore those combinations.

**Limit.** These are design-model results. They do not prove that production code implements the
checked design. The report explicitly acknowledges that gap. Umpire's realization and conformance
work must earn that additional claim separately.

Source:

- [How Amazon Web Services Uses Formal Methods, Communications of the ACM, 2015](https://assets.amazon.science/67/f9/92733d574c11ba1a11bd08bfb8ae/how-amazon-web-services-uses-formal-methods.pdf)

## 4. Microsoft CCF: models and implementation traces checked continuously

**Reported result.** The NSDI 2025 paper reports six subtle bugs found in the design and
implementation of the Confidential Consortium Framework before they affected users. CCF is
open source and powers Azure Confidential Ledger. The approach connects TLA+ specifications to
the C++ implementation and integrates checking into CI.

**Mechanism.** The team combined model checking, simulation, trace validation, and conventional
testing. The reported bugs included incorrect election-quorum calculation, unsafe commit
advancement, and premature node retirement. Different methods exposed different problems.

**Relevance to Umpire.** This is a close precedent for using several methods around one behavior
specification and keeping that specification connected to an evolving implementation. The public
repository and issue history also make the work inspectable beyond the paper's aggregate claims.

**Limit.** The authors describe substantial work aligning the consensus model and implementation,
and expensive exhaustive searches. Their results support a combined verification workflow; they
do not show that writing a model alone prevents these bugs or that integration is cheap.

Sources:

- [Smart Casual Verification of the Confidential Consortium Framework, NSDI 2025](https://www.microsoft.com/en-us/research/wp-content/uploads/2024/07/nsdi25spring-final392.pdf)
- [Paper abstract and author list](https://arxiv.org/abs/2406.17455)
- [CCF source repository](https://github.com/microsoft/CCF)
- [Public issue with a model-checker counterexample](https://github.com/microsoft/CCF/issues/3837)

## What we can credibly tell the team

| Claim | Strongest example | Evidence Umpire still needs |
| --- | --- | --- |
| A much smaller executable description can check a complicated implementation. | S3 ShardStore | Model, realization, and maintenance cost for a selected Temporal behavior. |
| Generated cases can reach behavior missed by existing tests. | MongoDB Realm Sync | Incremental coverage and defect detection against the current test suite. |
| Exploring event orderings can reveal bugs that survive review and testing. | AWS DynamoDB and S3 | Counterexamples for independently selected matching defects. |
| Models can remain connected to production implementations through CI. | Microsoft CCF | Reliable conformance checks and affordable updates as Temporal changes. |

The evidence is strongest for finding bugs and exploring behavior. These reports provide less
direct evidence for the broader claim that engineers introduce fewer bugs while authoring models,
or review them faster. Those claims need their own authoring and review experiments.

For a matching pilot, freeze the intended properties, challenge them with historical or
independently seeded defects, and run the resulting checks against real code. Report misses,
false alarms, runtime, and maintenance effort alongside successful detections. That establishes
whether this particular model earns the cost of its second description.
