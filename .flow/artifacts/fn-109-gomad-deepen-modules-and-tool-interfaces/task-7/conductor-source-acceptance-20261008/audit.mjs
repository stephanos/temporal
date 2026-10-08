import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const dir = path.join(repo, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = name => JSON.parse(fs.readFileSync(path.join(dir, name)));
const check = (condition, message) => { if (!condition) throw new Error(message); };
const git = (...args) => {
  const result = spawnSync('git', args, {cwd: repo, maxBuffer: 128 << 20});
  check(result.status === 0, 'git failed: ' + result.stderr);
  return result.stdout;
};
const evidence = read('evidence.json');
const frozen = read(evidence.frozen_candidate.manifest);
check(hash(JSON.stringify(frozen)) === evidence.frozen_candidate.source_tree_sha256, 'frozen manifest identity');
for (const input of frozen) {
  const file = path.join(repo, input.path);
  check(input.sha256 === null ? !fs.existsSync(file) : hash(fs.readFileSync(file)) === input.sha256, 'current source: ' + input.path);
}
const outcomes = new Map();
const exitCounts = {};
const tools = new Map();
const manifests = new Set();
const historicalControlsUnavailable = [];
for (const command of evidence.execution.commands) {
  const receipt = read(command.receipt);
  const bytes = fs.readFileSync(path.join(dir, receipt.log));
  check(hash(bytes) === receipt.log_sha256 && receipt.log_sha256 === command.log_sha256, 'raw log: ' + command.name);
  check(receipt.exit_code === command.exit_code && receipt.command === command.command, 'receipt: ' + command.name);
  check(receipt.source_unchanged && receipt.tools_unchanged && receipt.controls_unchanged !== false, 'frozen execution: ' + command.name);
  check(!command.frozen_candidate || receipt.controls_unchanged === true, 'final control binding: ' + command.name);
  check(receipt.signal === null && receipt.error === null && receipt.ended && receipt.elapsed_seconds >= 0, 'terminal command: ' + command.name);
  const manifest = read(receipt.source_manifest);
  check(hash(JSON.stringify(manifest)) === receipt.source_tree_sha256, 'source manifest: ' + command.name);
  manifests.add(receipt.source_manifest);
  if (receipt.control_manifest) {
    const controls = read(receipt.control_manifest);
    check(hash(JSON.stringify(controls)) === receipt.control_tree_sha256, 'control manifest: ' + command.name);
  } else {
    historicalControlsUnavailable.push(command.name);
  }
  for (const tool of receipt.tools) tools.set(tool.path, tool.sha256);
  const events = bytes.toString().split('\n').flatMap(line => {
    if (!line.startsWith('{')) return [];
    try { return [JSON.parse(line)]; } catch { return []; }
  });
  const named = events.filter(event => event.Test && ['pass', 'fail', 'skip'].includes(event.Action));
  for (const action of ['pass', 'fail', 'skip']) {
    check(named.filter(event => event.Action === action).length === receipt.counts[action], 'raw outcome count: ' + command.name);
    if (command.named_outcome_counts !== null) check(receipt.counts[action] === command.named_outcome_counts[action], 'evidence outcome count: ' + command.name);
  }
  outcomes.set(command.name, {receipt, named});
  exitCounts[receipt.exit_code] = (exitCounts[receipt.exit_code] ?? 0) + 1;
}
for (const [file, digest] of tools) check(hash(fs.readFileSync(file)) === digest, 'current tool: ' + file);
for (const group of evidence.assertion_coverage) {
  const {receipt, named} = outcomes.get(group.command);
  check(receipt.exit_code === 0 && receipt.source_tree_sha256 === evidence.frozen_candidate.source_tree_sha256, 'coverage candidate: ' + group.requirement);
  for (const name of group.named_assertions) check(named.some(event => event.Test === name && event.Action === 'pass'), 'coverage assertion: ' + name);
}
const original = read('caller-controls.json');
for (const input of read('original-caller-source-final.json')) check(hash(fs.readFileSync(path.join(original.original, input.path))) === input.sha256, 'original dependency: ' + input.path);
check(hash(fs.readFileSync(path.join(original.original, 'runner/preparation_source_test.go'))) === hash(fs.readFileSync(path.join(repo, 'tools/gomad3/runner/preparation_source_test.go'))), 'identical actual caller harness');
for (const entry of original.overlayInputs) {
  check(hash(fs.readFileSync(entry.source)) === entry.sha256, 'original overlay source');
  check(hash(fs.readFileSync(entry.overlay)) === entry.overlay_sha256, 'original overlay bytes');
}
const before = fs.readFileSync(path.join(dir, 'original-caller-snapshot.json'));
const after = fs.readFileSync(path.join(dir, 'current-caller-snapshot.json'));
check(before.equals(after) && hash(before) === '069df114861458eab57e5decac9105cafba751f5f2b8c3f97ab40804a9a041a4', 'actual caller equality');
const rows = JSON.parse(before);
check(rows.length === 8 && rows.every(row => row.build_count === 1 && row.prepared_from_validated_caller_plan.Path === '<prepared-path>'), 'actual fresh/cache snapshots');
check(JSON.stringify(read('original-caller-cache-state.json')) === JSON.stringify(read('current-caller-cache-state.json')), 'matched original/current cache states');
const preservation = read('preservation.json');
for (const input of [...preservation.preimages, ...preservation.unchanged_existing_bodies, ...preservation.protected_user_files]) check(hash(fs.readFileSync(path.join(repo, input.path))) === input.sha256, 'preservation: ' + input.path);
for (const input of preservation.unchanged_existing_bodies) check(hash(git('show', evidence.base_commit + ':' + input.path)) === input.sha256, 'admission body: ' + input.path);
for (const input of read('runtime-inputs.json').inputs) check(hash(fs.readFileSync(input.path)) === input.sha256, 'prune runtime input: ' + input.path);
const lint = read('lint-attribution.json');
check(lint.count === 40 && lint.findings.length === 40 && lint.new_task7_findings === 0, 'lint count');
const lintPaths = new Map();
for (const finding of lint.findings) {
  check(finding.whole_file_unchanged && finding.source_sha256 === finding.admission_source_sha256, 'lint unchanged attribution');
  lintPaths.set(finding.path, finding.source_sha256);
  check(fs.readFileSync(path.join(repo, finding.path), 'utf8').split('\n')[finding.line - 1] === finding.line_text, 'lint exact line');
}
for (const [file, digest] of lintPaths) check(hash(git('show', evidence.base_commit + ':' + file)) === digest, 'lint admission source: ' + file);
const inventories = read('static-inventories.json');
for (const inventory of inventories) {
  check(inventory.package_count === 55 && inventory.inventory.Packages.length === 55, 'nonempty static inventory');
  check(inventory.vet.exit_code === 0 && inventory.source_tree_sha256 === evidence.frozen_candidate.source_tree_sha256, 'static source binding');
  check(outcomes.get('final-architecture-host-vet').named.some(event => event.Test === inventory.vet.test && event.Action === 'pass'), 'actual platform vet assertion');
}
check(outcomes.get('qualification-ordinary').receipt.exit_code === 1 && evidence.qualification.remaining_source_gaps.length === 10 && !evidence.qualification.source_gaps_are_native_waived, 'retained qualification red');
check(evidence.terminal.commands_running.length === 0 && evidence.terminal.children_running.length === 0 && evidence.terminal.review_verdict === null, 'worker terminal/no self-verdict');
check(git('diff', '--check').length === 0, 'diff whitespace');
console.log(JSON.stringify({source_paths: frozen.length, source_sha256: evidence.frozen_candidate.source_tree_sha256, raw_receipts: evidence.execution.commands.length, exit_counts: exitCounts, historical_source_manifests: manifests.size, historical_controls_unavailable: historicalControlsUnavailable, current_tools: tools.size, coverage_groups: evidence.assertion_coverage.length, actual_caller_rows: rows.length, original_source_paths: read('original-caller-source-final.json').length, unchanged_lint_findings: lint.findings.length, unchanged_lint_paths: lintPaths.size, static_inventories: inventories.map(i => ({platform: i.platform, packages: i.package_count})), qualification_red_retained: true, independent_review: 'not yet dispatched', task_status: 'in_progress'}));
