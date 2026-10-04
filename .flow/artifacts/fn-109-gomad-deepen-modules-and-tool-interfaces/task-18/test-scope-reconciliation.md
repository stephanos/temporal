# Filesystem handle test scope clarification

The conductor updated task 18's Description through flowctl while task 17 remained the sole source writer. Task 18 stays todo and no filesystem production or test implementation was admitted.

Files/Touches now permit the nested-module filesystem_handles_ownership_test.go and execution-package test/integration selector required to deliver the existing R12 ownership and actual-process operation checks. The Approach points to task 17's reviewed fixture pattern, distinguishes two filesystem representations from three test backends, requires explicit existing backend outcomes and includes new process cases in Runner's selector. The old oneNodeVolumeSpec helper proves in-process behavior only. Architecture RED must observe old production ownership violations; behavior assertions exercise operation outcomes rather than source text.

No Acceptance criterion, platform/process gate, wire expectation, public contract, single-owner boundary or delivery-order dependency changed. The writing-for-agents skill influenced co-location of the actual-process coverage obligation with its permitted test surfaces, preventing a later worker from omitting required tests to obey an incomplete Touches list. Implement after task 17's integrated source is reviewed, following MILESTONES' source-advancement policy with native acceptance still open.

Task 17's frozen-source review subsequently confirmed that Runner case names
alone are insufficient: the canonical Makefile filter excluded all new network
process cases. Task 18's Touches and Approach now also include Makefile and the
shared gate-selection regression. Apply the final reviewed task-17 correction
to new filesystem cases while preserving the strict-delay exclusion and
separate forward-mode gate. This remains test-delivery clarification, not task
18 implementation admission or native acceptance.
