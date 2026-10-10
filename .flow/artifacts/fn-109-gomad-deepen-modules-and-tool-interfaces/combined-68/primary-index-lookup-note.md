# Primary index lookup observation

Primary checkpoint attempts for the next three admitted corrective owners failed with an existing .git/index.lock. The index contains only ten root-owned admission paths: tasks 69/70/71 JSON and Markdown, their three admission artifacts, and the three newly added MILESTONES rows. User amendments remain unstaged. No task completion follows from admission.

Independent read-only research found .git/index and .git/index.lock resolving to device 0:58, inode 418207661, size 7,771,900 and SHA-256 27f275584681a098afa5b718ecb7722d2867f1a362becddbb7877b6caf24a6a6. Both report one link and the same 2026-10-09 19:06:50 -0700 modification time. Directory enumeration lists index but not index.lock; direct lookup/open of the latter succeeds. The mount is host-shared virtiofs.

Visible process inspection and bounded lsof/fuser identify no Git process or holder, but cannot exclude a host process. The observations suggest a lookup/rename-cache anomaly; they do not prove a safe removable lock. Root neither deletes nor renames either path and requests host observations. Isolated task-68 verification remains separate. These observations supply no source-gate or native evidence, and no Git history or user change is reverted.

While preparing this checkpoint, concurrent user edits advanced fn-109 tasks 17/18 and the fn-155 spec pair. Root resnapshotted their current hashes rather than restoring older bytes. Their contents and the user-added fn-155 milestone section remain excluded from root commits.

Subsequent read-only lookup reported index.lock absent without any root unlink, rename or cleanup. The preserved staged admission scope passed whitespace checking and committed as 176a269e651c10f7e88bb8dc11fc03ef78bf2e57. Task68 then integrated as c506713ce063759c5d24129d775d2fefc6314618. No host-side cause or permanent filesystem repair is established. Concurrent fn-110.3 and newly created fn-155 task records also remain user-owned and excluded.
