# Scoped source verification admission

The final authoring command remains RED: 666 pass, one portable failure and one
native-profile skip. At frozen source `7550610348c8f837d0874639ccf8173509b9b2a0437fc7a477897c8e6f8068cb`,
`TestRunCompatibilityPackRefreshResolvesTwoModulesAndKeepsPartialApproval`
failed with ENOSPC opening the child Go cache under `/home/agent/.cache/go-build`.
The worker reported the raw failure; root independently inspected
`target/internal/build/context.go:88`: `Environment` removes `GOCACHE` but retains
`XDG_CACHE_HOME`. Root's actual `df` showed zero root-overlay space and 67 GiB
available on the assigned workspace filesystem.

Admit only this exact unchanged test with command-local `XDG_CACHE_HOME` set to
`/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/.source-target-cache`.
Retain the actual child-effective cache query, command environment, tool/source
identities and outcomes. This is a controlled rerun after a known cause and
changed relevant cache input, not an unchanged retry. Ordinary workspace temp
placement and all production source/test assertions remain unchanged. The broad
command, native execution and global cache remain unclaimed; no cache sweep,
temporary-filesystem relocation or additional failure-site admission follows.

The earlier baseline report-directory Sync permission failure remains recorded
and unexplained. Its isolated original test passed before edits, and the final
whole authoring package passed in the mixed command; neither proves a repair
or changes the raw baseline verdict.
