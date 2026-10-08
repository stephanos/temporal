import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';

const out = dirname(fileURLToPath(import.meta.url));
const rel = '.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-32/source-acceptance-20261008';
const task5 = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008';
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = path => fs.readFileSync(path);
const json = path => JSON.parse(read(path));
const reference = path => ({path, sha256: sha(read(path))});
const labels = ['baseline-portable', 'source-binding-check', 'durable-source-binding-check', 'task-source-lint', 'generated-source-validation', 'whole-file-lint', 'task-owned-format', 'final-diff-check'];
const observations = labels.map(label => {
  const item = json(out + '/' + label + '.json');
  assert.equal(sha(read(out + '/' + label + '.stdout')), item.stdout_sha256);
  assert.equal(sha(read(out + '/' + label + '.stderr')), item.stderr_sha256);
  assert.deepEqual(item.source_changes, []);
  assert.equal(item.sources_before_sha256, item.sources_after_sha256);
  assert.equal(item.exit, label === 'whole-file-lint' ? 1 : 0);
  return {...reference(rel + '/' + label + '.json'), exit: item.exit, argv: item.argv, test_counts: item.test_counts};
});
const portable = json(out + '/baseline-portable.json');
assert.deepEqual(portable.test_counts, {pass: 6, fail: 0, skip: 0});
assert.equal(read(out + '/task-owned-format.stdout').length, 0);
const lint = JSON.parse(read(out + '/whole-file-lint.stdout').toString().split('\n')[0]);
assert.equal(lint.Issues.length, 16);
const taskOwned = lint.Issues.filter(issue => issue.Pos.Filename === 'tools/gomad3/toolchain/clock_inventory_test.go');
assert.deepEqual(taskOwned, []);
const binding = json(out + '/source-binding.json');
const standards = {
  task_path: 'tools/gomad3/toolchain/clock_inventory_test.go', task_owned_issues: taskOwned,
  configured_changed_source_command: reference(rel + '/task-source-lint.json'), configured_changed_source_exit: 0,
  complete_file_coverage_command: reference(rel + '/whole-file-lint.json'), unfiltered_package_exit: 1,
  unfiltered_package_issues: lint.Issues.map(issue => ({linter: issue.FromLinter, file: issue.Pos.Filename, line: issue.Pos.Line, message: issue.Text})),
  broader_prior_red: reference(task5 + '/task-source-lint.json'),
  broader_prior_attribution: reference(task5 + '/standards-attribution.json'),
  inherited_overlay_format: reference(task5 + '/task-owned-format.json'),
  limits: 'Only D27 changed-source lint and the complete clock-inventory file pass. No unfiltered package, global lint or full-format pass is claimed. Unrelated 16 toolchain findings, prior 2 descriptor-close findings and inherited runtime-overlay format remain with their source owners; no new exception or suppression.',
  tool_hashes: Object.fromEntries(['/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype', '.github/.golangci.yml'].map(path => [path, sha(read(path))]))
};
fs.writeFileSync(out + '/standards-attribution.json', JSON.stringify(standards, null, 2) + '\n');
const head = execFileSync('git', ['rev-parse', 'HEAD']).toString().trim();
assert.equal(head, binding.base_commit);
for (const item of binding.user_files) assert.equal(sha(read(item.path)), item.sha256);
const evidence = {
  base_commit: binding.base_commit, commits: [], implementation_commits: [binding.implementation],
  tests: observations.map(item => item.argv.join(' ')), prs: [],
  baseline: 'green; 6 portable top-level tests, zero failure/skip. Historical unsupported native baseline remains historical and native gates are transferred.',
  observations, source_binding: reference(rel + '/source-binding.json'), standards_attribution: reference(rel + '/standards-attribution.json'),
  policy_decision: reference(rel + '/policy-decision.md'), policy_raw: binding.policy_provenance,
  source_review_required: {current_full_files: ['tools/gomad3/README.md#Contract', 'tools/gomad3/toolchain/clock_inventory_test.go'], original_base: binding.original_source_base, implementation: binding.implementation, fresh_review_owner: 'conductor', verdict: null},
  native_qualification: binding.native_qualification, product_edits: [], user_files: binding.user_files,
  positive_inventory_reuse: binding.positive_inventory, commands_running: [],
  test_count_scope: 'Portable JSON stream supplies 6 observed top-level passes. Non-JSON commands report exit codes; their test_counts zero means no parsed JSON test verdicts, not empty gate selection.',
  stage: 'impl-review skipped(policy: conductor owns fresh source review and completion)'
};
fs.writeFileSync(out + '/evidence.json', JSON.stringify(evidence, null, 2) + '\n');
console.log(JSON.stringify({observations: observations.length, portable_pass: 6, fail: 0, skip: 0, task_owned_lint: 0, unfiltered_package_issues: 16, source_changes: 0, policy_raw_original_bytes: binding.policy_provenance.reduce((n, item) => n + item.decoded_bytes, 0), native_qualification: false}));
