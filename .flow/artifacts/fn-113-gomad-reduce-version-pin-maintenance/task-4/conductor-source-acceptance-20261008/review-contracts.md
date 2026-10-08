I’ll read the task spec first, then check the diff and acceptance evidence for contract inconsistencies.
The code changes update the generated guide; the other source changes align docs with existing refresh behavior. The evidence distinguishes source checks from deferred native qualification and records the failed broad command and focused rerun. I’ll check those claims against the implementation and raw receipts.
The refresh implementation matches the new wording: it evaluates mapped directories first, then merges a saved report’s invalidations. The architecture correction also matches the dossier’s required core-corpus check. The sandbox blocked the retained audit script’s child-process call, so I’m verifying its hashes and test-event counts with read-only checks.
No surviving findings. The revised documentation matches refresh behavior and the dossier’s required core-corpus check.

Verified the generated-output check, source/artifact hashes, scratch publication, and recorded test counts. The broad authoring command remains explicitly RED; its focused rerun is reported separately. Sandbox restrictions prevented rerunning the complete audit script.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Retained first baseline and task-1 inventory |
| R2 | met | Predecessor pin-impact acceptance |
| R3 | met | Predecessor source acceptance; approved scratch publication preserved |
| R4 | met | Predecessor refresh acceptance; docs match live discovery and saved-report merge |
| R5 | met | Updated procedure, generated guide, and matched measurement distinguishing observed and normalized counts |
| R6 | deferred | Native qualification transferred to fn-149/fn-128; retained source validation and static checks verified |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
