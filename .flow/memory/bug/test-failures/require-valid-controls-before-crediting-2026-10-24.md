---
title: Require valid controls before crediting qualification catcher proof
date: "2026-10-24"
track: bug
category: test-failures
module: .flow/artifacts/fn242-obligation-inventory.json
tags: [qualification, proof, inventory]
problem_type: test-failure
symptoms: Incidental exceptions and incomplete source-site enumeration earned false catching credit
root_cause: Exception-only attribution omitted valid controls and flattened source branch semantics
resolution_type: documentation
last_updated: "2026-10-24"
---

## Problem
An exception-only trace credited a sealing check when incomplete configuration merely raised KeyError. The inventory also omitted delegated receipts, boolean validators and a generator.throw refusal, and flattened else branches with incorrect polarity.

## What Didn't Work
A passing assertRaises test and a traceback through a validator do not establish that the intended defect caused refusal. Source assertions without execution are not catching proof.

## Solution
The fn-242 inventory now binds each kept check to an explicit intended refusal and an exception-free valid control traversing the same predicate in the same passing test. The audit retained192 credits and withdrew516. Exact source expressions and structured true/false ancestor paths replace flattened conditions. The73-module site sweep and ten transition probes cover the corrected classes. See .flow/artifacts/fn242-obligation-inventory.json (proof_control_audit, source_bindings, review_corrections) and its companion Markdown; the review receipt is /tmp/impl-review-receipt-0b8490237aa9-fn-242-gomad-shrink-the-qualification.1.json.

## Prevention
Require a valid control, targeted mutation, exact source binding and numeric execution result before crediting catching proof. Inventory callbacks, boolean predicates and delegated validation; preserve branch grouping and polarity structurally. One authoritative validator and fixture may cover several sites at the same immutable input and trust boundary; independent reopen boundaries still require validation.

## Update 2026-10-24

## Problem
An exception-only trace credited a sealing check when incomplete configuration merely raised KeyError. The inventory also omitted delegated receipts, boolean validators and a generator.throw refusal, and flattened else branches with incorrect polarity.

## What Didn't Work
A passing assertRaises test and a traceback through a validator do not establish that the intended defect caused refusal. Source assertions without execution are not catching proof.

## Solution
The fn-242 inventory now binds each kept check to an explicit intended refusal and an exception-free valid control traversing the same predicate in the same passing test. The audit retained 192 credits and withdrew 516. Exact source expressions and structured true/false ancestor paths replace flattened conditions. The 73-module site sweep and ten transition probes cover the corrected classes. See .flow/artifacts/fn242-obligation-inventory.json (proof_control_audit, source_bindings, review_corrections) and its companion Markdown; the review receipt is /tmp/impl-review-receipt-0b8490237aa9-fn-242-gomad-shrink-the-qualification.1.json.

## Prevention
Require a valid control, targeted mutation, exact source binding and numeric execution result before crediting catching proof. Inventory callbacks, boolean predicates and delegated validation; preserve branch grouping and polarity structurally. One authoritative validator and fixture may cover several sites at the same immutable input and trust boundary; independent reopen boundaries still require validation.
