import {readFileSync, writeFileSync, readdirSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = dirname(fileURLToPath(import.meta.url));
const local = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008';
const sha = value => createHash('sha256').update(value).digest('hex');
const read = path => readFileSync(path);
const json = name => JSON.parse(read(out + '/' + name));
const git = args => {
  const result = spawnSync('git', args, {cwd: root, maxBuffer: 32 << 20});
  assert.equal(result.status, 0);
  return result.stdout;
};
const binding = json('source-binding.json');
const names = ['baseline-validate', 'portable-source-controls', 'task-source-lint', 'configured-vet', 'task-owned-format', 'generated-source-validation', 'source-binding-encoded-raw', 'source-binding-final'];
const commands = names.map(name => {
  const receipt = json(name + '.json');
  assert.equal(sha(read(out + '/' + name + '.stdout')), receipt.stdout_sha256);
  assert.equal(sha(read(out + '/' + name + '.stderr')), receipt.stderr_sha256);
  assert.deepEqual(receipt.source_changes, []);
  return {receipt: local + '/' + name + '.json', receipt_sha256: sha(read(out + '/' + name + '.json')), ...receipt};
});
const passed = ['baseline-validate', 'portable-source-controls', 'configured-vet', 'generated-source-validation', 'source-binding-final'];
for (const name of passed) assert.equal(json(name + '.json').exit, 0, name);
assert.deepEqual(json('portable-source-controls.json').test_counts, {pass: 21, fail: 0, skip: 0});
assert.equal(json('task-source-lint.json').exit, 2);
const lintOutput = read(out + '/task-source-lint.stdout').toString();
const issues = [...lintOutput.matchAll(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)].map(match => ({path: match[1], line: Number(match[2]), column: Number(match[3]), message: match[4], linter: match[5]}));
assert.equal(issues.length, 2);
assert(issues.every(issue => issue.path === 'tools/gomad3/runner/internal/execution/descriptor_dup_linux_test.go' && issue.linter === 'errcheck'));
const taskPaths = [
  'tools/gomad3/toolchain/draw_inventory_test.go',
  'tools/gomad3/toolchain/runtime/go1.27.1.patch',
  'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go',
  'tools/gomad3/runner/internal/execution/draw_check_toolchain_test.go',
  'tools/gomad3/choice/internal/wire/wire_generated.go',
  'tools/gomad3/target/internal/livecap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go',
  'tools/gomad3/Makefile', 'Makefile', 'tools/gomad3/runner/testdata/diagnostic-identity-choices.json',
];
const isTaskPath = path => taskPaths.includes(path) || /^tools\/gomad3\/internal\/gomadtool\/conformance\/runtime[^/]*\.go$/.test(path) || path.startsWith('tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/');
const ownIssues = issues.filter(issue => isTaskPath(issue.path));
assert.equal(ownIssues.length, 0);
const outside = issues[0].path;
const outsideData = read(root + '/' + outside);
assert(outsideData.equals(git(['show', binding.base_commit + ':' + outside])));
assert(outsideData.equals(git(['show', '1deced3efa4e7000163cb269e70e35f0c6b7dbd7:' + outside])));
const historicalLint = '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/source-acceptance-20261007/raw/original-base-fast-lint.stdout';
for (const issue of issues) assert(read(root + '/' + historicalLint).toString().includes(issue.path + ':' + issue.line + ':' + issue.column + ': ' + issue.message));
const formatOutput = read(out + '/task-owned-format.stdout').toString().trim();
assert.equal(formatOutput, 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go');
const runtime = read(root + '/' + formatOutput);
const formatted = spawnSync('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt', [], {input: runtime});
assert.equal(formatted.status, 0);
const onlyExpectedFormat = runtime.toString().replace('const gomadChoiceMaximumAlternatives = 256\n// The response buffer', 'const gomadChoiceMaximumAlternatives = 256\n\n// The response buffer');
assert.equal(formatted.stdout.toString(), onlyExpectedFormat);
assert(runtime.equals(git(['show', 'a936b597b4c62fa50f11a6c16c91111cd52b1ec3:' + formatOutput])));
const priorPreservation = JSON.parse(read(root + '/' + binding.preservation.path));
const overlayPreservation = priorPreservation.files.find(file => file.path === formatOutput);
assert.equal(sha(formatted.stdout), overlayPreservation.alpha_gofmt_sha256);
const standards = {scope: 'Actual configured lint-code and errortype over task five host packages, original task source base d5330bb779c55f1b9af57c5845ec195275df0276; task-file attribution only', lint_exit: 2, issue_count: issues.length, issues, task_owned_path_predicate: {exact: taskPaths, runtime_glob: 'tools/gomad3/internal/gomadtool/conformance/runtime*.go', fixture_prefix: 'tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/'}, task_owned_findings: ownIssues, inherited_non_task_file: {path: outside, sha256: sha(outsideData), introduced_commit: '70bb38e5ddec2d271c12f0e8c2489855f08eb0ef', historical_receipt: historicalLint, historical_sha256: sha(read(root + '/' + historicalLint))}, configuration_sha256: sha(read(root + '/.github/.golangci.yml')), lint_tools: Object.fromEntries(['golangci-lint-v2.13.0', 'errortype'].map(name => [name, sha(read('/tmp/fn109-lint-tools.ZdNe1t50/' + name))])), configured_errortype_exit: 0, aggregate_green: false, formatting: {observation: 'one inherited missing blank line before response-buffer comment at runtime/gomad.go:108; all other selected files clean', actual_format_clean: false, task_introduced_format_changes: 0, source_sha256: sha(runtime), formatted_sha256: sha(formatted.stdout), same_raw_source_as_compact: true, matches_retained_normalized_preservation_hash: true}};
writeFileSync(out + '/standards-attribution.json', JSON.stringify(standards, null, 2) + '\n');
const archive = read(root + '/' + binding.archive.path);
assert.equal(sha(archive), binding.archive.sha256);
for (const entry of binding.input_closure) assert.equal(sha(read(root + '/' + entry.path)), entry.current_sha256);
for (const entry of binding.user_files) assert.equal(sha(read(root + '/' + entry.path)), entry.sha256);
assert.equal(git(['rev-parse', 'HEAD']).toString().trim(), binding.head);
const conductorReceipts = ['conductor-portable-source-controls', 'conductor-generated-validation'].map(name => {
  const receipt = json(name + '.json');
  assert.equal(receipt.exit, 0);
  assert.equal(sha(read(out + '/' + name + '.stdout')), receipt.stdout_sha256);
  assert.equal(sha(read(out + '/' + name + '.stderr')), receipt.stderr_sha256);
  return {path: local + '/' + name + '.json', sha256: sha(read(out + '/' + name + '.json')), exit: receipt.exit, test_counts: receipt.test_counts};
});
const evidence = {base_commit: binding.base_commit, commits: [], tests: commands.map(command => command.argv.join(' ') + (command.exit === 0 ? '' : ' [exit ' + command.exit + '; see retained receipt]')), prs: [], task: 'fn-112-gomad-determinism-assurance-and-test.5', worker_status: 'in_progress; conductor owns review, commit and completion', current_source_head: binding.head, review_source_base: '1b0bc277589d141aca8b534b03135ab3e57fc050', original_task_source_base: 'd5330bb779c55f1b9af57c5845ec195275df0276', baseline: 'applicable validate green; source unchanged; configured lint red for two inherited non-task findings; aggregate native gates transferred and not run', source_binding: {path: local + '/source-binding.json', sha256: sha(read(out + '/source-binding.json'))}, standards: {path: local + '/standards-attribution.json', sha256: sha(read(out + '/standards-attribution.json'))}, commands, retained_proof: {exact_current_bindings: 87, reused_input_closure_files: 1218, excluded_changed_documents: 5, original_raw_files_verified: 72, original_raw_bytes_verified: 521317, positive_inventory: binding.positive_inventory, canonical: binding.canonical, identity: {path: binding.identity.receipt, sha256: binding.identity.sha256, unchanged_current_source_bindings: binding.identity.bindings.length, pointers: 7, bytes: 13271}, preservation: binding.preservation}, runtime_boundary: {host_guard_pre_mutation_source_checks: binding.guard_order.length, collector: binding.collector_blocker, numeric_fault: 'retained target draw/localization control', host_fault: 'host:N checks per-M gomadHostDraw before seeded mutations', host_timed_fault: 'host-timed:N arms next bracketed host draw; retained exit125 fixture'}, native_qualification: {darwin_arm64: 'unverified; deferred fn-149.1/.2/.4', linux_amd64: 'unverified; deferred fn-128.1/.4/.7', linux_arm64: 'unsupported; no native proof, build-key or host spoof', native_quick_commands: 'make -C tools/gomad3 test/test-toolchain/test-runtime and actual runtime/core/smoke/replay/full controls transferred; no receipt skip/pass claim'}, product_edits: [], user_files: binding.user_files, commands_running: []};
evidence.conductor_receipts = conductorReceipts;
evidence.calibration_observations = {path: local + '/calibration-observations.json', sha256: sha(read(out + '/calibration-observations.json'))};
writeFileSync(out + '/evidence.json', JSON.stringify(evidence, null, 2) + '\n');
const manifest = readdirSync(out).filter(name => !name.endsWith('.md') && name !== 'bundle-manifest.json').sort().map(name => ({path: local + '/' + name, bytes: read(out + '/' + name).length, sha256: sha(read(out + '/' + name))}));
writeFileSync(out + '/bundle-manifest.json', JSON.stringify(manifest, null, 2) + '\n');
console.log(JSON.stringify({commands: commands.length, portable_pass: 21, portable_fail: 0, portable_skip: 0, task_owned_lint_findings: 0, inherited_non_task_lint_findings: 2, source_format_clean: false, product_changes: 0, frozen_inputs: binding.input_closure.length, evidence: local + '/evidence.json'}));
